package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	appmemory "smart-recruit-ai-agent-service/internal/application/memory"
	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	aiagentpersistence "smart-recruit-ai-agent-service/internal/infrastructure/persistence"
	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
	aiagentruntime "smart-recruit-ai-agent-service/internal/runtime"
	"smart-recruit-commons/mq"
	"smart-recruit-commons/pkg/crypto"
	"smart-recruit-platform-go/businessclock"
	platformconfig "smart-recruit-platform-go/config"
	"smart-recruit-platform-go/i18n"
	"smart-recruit-platform-go/logger"
	"smart-recruit-platform-go/mysqltime"
	"smart-recruit-platform-go/nacos"
	logicobservability "smart-recruit-platform-go/observability"
	platformobs "smart-recruit-platform-go/observability"
	"smart-recruit-platform-go/server"
	logicconfig "smart-recruit-platform-go/serviceconfig"
	"smart-recruit-platform-go/tenantgorm"
	"smart-recruit-proto/recruitment/pb"
)

const nacosServiceName = "ai-agent"

var (
	aiAgentTenantOwnedTables = []string{"candidate_match_evaluations", "candidate_match_evidence", "jobs", "applications", "application_status_transitions", "interview_schedules", "interview_feedback", "offers", "offer_events"}
	aiAgentMixedScopeTables  = []string{"ai_chat_sessions", "ai_chat_history", "ai_session_summaries", "ai_tool_traces", "agent_runs", "agent_run_events", "agent_run_steps", "ai_memories", "ai_embeddings", "third_party_usage_logs", "ai_usage_auth_contexts", "mcp_tool_logs"}
)

func main() {
	if err := i18n.ConfigureFromEnv(); err != nil {
		logger.L().Error("log.service.config_failed", zap.String("cause", err.Error()))
		os.Exit(2)
	}
	businessclock.Configure()
	check := flag.Bool("check", false, "validate AI Agent service runtime wiring and exit")
	serve := flag.Bool("serve", false, "start AI Agent gRPC runtime")
	addr := flag.String("addr", envOrDefault("GRPC_ADDR", ":50066"), "AI Agent gRPC listen address")
	flag.Parse()

	if *check {
		if err := checkRuntime(); err != nil {
			logger.L().Error("log.service.check_failed", zap.String("service", "ai-agent-service"), zap.String("cause", err.Error()))
			os.Exit(1)
		}
		logger.L().Info("log.service.check_passed", zap.String("service", "ai-agent-service"))
		return
	}
	if *serve {
		if err := serveAIAgent(*addr); err != nil {
			logger.L().Error("log.service.serve_failed", zap.String("service", "ai-agent-service"), zap.String("cause", err.Error()))
			os.Exit(1)
		}
		return
	}
	logger.L().Error("log.service.arguments_required", zap.String("service", "ai-agent-service"))
	os.Exit(2)
}

func checkRuntime() error {
	runtime, err := aiagentruntime.New(aiagentruntime.Deps{
		AI:                     noopAIService{},
		LlmConfig:              noopLlmConfigService{},
		Prompt:                 noopPromptService{},
		AgentConfig:            noopAgentConfigService{},
		MCP:                    noopMCPService{},
		AgentSkill:             noopAgentSkillService{},
		RecruitingIntelligence: noopRecruitingIntelligenceService{},
		EmbeddingConfig:        noopEmbeddingConfigService{},
		PlatformAIControlPlane: noopPlatformAIControlPlaneService{},
	})
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	defer server.Stop()
	return runtime.RegisterGRPC(server)
}

func serveAIAgent(addr string) error {
	if err := ensureLogicConfigPath(); err != nil {
		return err
	}
	bootstrap, err := loadBootstrap(addr)
	if err != nil {
		return fmt.Errorf("load platform bootstrap: %w", err)
	}
	traceRuntime, err := platformobs.NewTraceRuntime(context.Background(), platformobs.TraceConfig{
		ServiceName:    bootstrap.ServiceName,
		ServiceVersion: bootstrap.ServiceVersion,
		Env:            bootstrap.ServiceEnv,
	})
	if err != nil {
		return fmt.Errorf("init trace runtime: %w", err)
	}
	defer func() { _ = traceRuntime.Shutdown(context.Background()) }()

	cfg, err := logicconfig.Load()
	if err != nil {
		return fmt.Errorf("load service config: %w", err)
	}
	if err := server.ValidateInternalToken(); err != nil {
		return fmt.Errorf("gRPC internal token validation: %w", err)
	}
	if err := logger.Init(cfg.Logging); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	log := logger.L()
	logicobservability.DefaultMetrics = logicobservability.NewRegistry(aiagentruntime.ServiceName)

	dsn, err := mysqltime.NormalizeDSN(cfg.MySQL.DSN)
	if err != nil {
		return err
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		TranslateError: true,
		Logger:         logger.NewGormLogger(&cfg.Logging.Gorm),
	})
	if err != nil {
		return fmt.Errorf("connect mysql: %w", err)
	}
	if err := db.Use(tenantgorm.NewWithMixed(aiAgentTenantOwnedTables, aiAgentMixedScopeTables)); err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql db: %w", err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.MySQL.ConnMaxLifetime.Duration)
	sqlDB.SetConnMaxIdleTime(cfg.MySQL.ConnMaxIdleTime.Duration)
	if err := mysqltime.ValidateSession(context.Background(), sqlDB); err != nil {
		return err
	}

	var redisClient *redis.Client
	if cfg.Redis.Addr != "" {
		redisClient = redis.NewClient(redisOptions(cfg))
		defer redisClient.Close()
	}
	mqConn, err := mq.New(mqConfig(cfg))
	if err != nil {
		return fmt.Errorf("connect rabbitmq: %w", err)
	}
	defer mqConn.Close()
	identityConn, err := dialInternalGRPC(envOrDefault("IDENTITY_GRPC_ADDR", "127.0.0.1:50061"))
	if err != nil {
		return fmt.Errorf("dial identity grpc: %w", err)
	}
	defer identityConn.Close()
	recruitmentConn, err := dialInternalGRPC(envOrDefault("RECRUITMENT_GRPC_ADDR", "127.0.0.1:50062"))
	if err != nil {
		return fmt.Errorf("dial recruitment grpc: %w", err)
	}
	defer recruitmentConn.Close()
	billingConn, err := dialInternalGRPC(envOrDefault("BILLING_GRPC_ADDR", "127.0.0.1:50069"))
	if err != nil {
		return fmt.Errorf("dial billing grpc: %w", err)
	}
	defer billingConn.Close()

	metricsServer, err := server.StartMetricsServer(cfg.Observability.MetricsAddr)
	if err != nil {
		return fmt.Errorf("start metrics server: %w", err)
	}
	defer server.ShutdownMetricsServer(context.Background(), metricsServer)

	nativeStore := aiagentpersistence.NewNativeStore(db)
	memoryRepo := aiagentpersistence.NewMemoryRepositoryAdapter(nativeStore)
	embeddingService := embeddinginfra.NewEmbeddingService(nativeStore, nil)
	memoryCfg := appmemory.ConfigFromService(cfg)
	memoryService := appmemory.NewService(memoryRepo, appmemory.NewExtractor(nil), embeddingService, memoryCfg)
	catalogSyncCtx, cancelCatalogSync := context.WithTimeout(context.Background(), 15*time.Second)
	if err := nativeStore.SyncBundledLlmModelCatalog(catalogSyncCtx); err != nil {
		cancelCatalogSync()
		return fmt.Errorf("sync bundled llm model catalog: %w", err)
	}
	cancelCatalogSync()
	if encKey, encKeyErr := crypto.LoadEncryptionKey(); encKeyErr != nil {
		log.Warn("log.ai.encryption_unavailable", zap.String("cause", encKeyErr.Error()))
	} else {
		nativeStore.SetEncryptionKey(encKey)
	}
	nativeStore.SetRuntimeLLMConfig(aiagentpersistence.RuntimeLLMConfig{
		APIKey:                  cfg.AI.APIKey,
		Model:                   cfg.AI.Model,
		BaseURL:                 cfg.AI.BaseURL,
		ProviderType:            "openai_compatible",
		Timeout:                 cfg.AI.Timeout.Duration,
		TotalTimeout:            cfg.AI.TotalTimeout.Duration,
		ToolMaxRounds:           cfg.AI.ToolMaxRounds,
		ToolTotalTimeout:        cfg.AI.ToolTotalTimeout.Duration,
		MaxConcurrency:          cfg.AI.MaxConcurrency,
		CircuitFailureThreshold: cfg.AI.CircuitFailureThreshold,
		CircuitOpenTimeout:      cfg.AI.CircuitOpenTimeout.Duration,
		HalfOpenMaxRequests:     cfg.AI.CircuitHalfOpenMaxRequests,
		RetryMaxAttempts:        cfg.AI.RetryMaxAttempts,
		RetryBaseDelay:          cfg.AI.RetryBaseDelay.Duration,
		SlowResponseThreshold:   cfg.AI.SlowResponseThreshold.Duration,
	})
	billingClient := pb.NewBillingServiceClient(billingConn)
	runtime, err := aiagentruntime.New(aiagentgrpc.NewNativeRuntimeDeps(aiagentgrpc.RuntimeDeps{
		Store:            nativeStore,
		Provider:         nativeStore,
		MemoryService:    memoryService,
		EmbeddingService: embeddingService,
		PlatformAI:       aiagentpersistence.NewPlatformAIControlPlaneServer(nativeStore),
		RecruitingPolicy: recruitingRuntimePolicy(cfg),
		EmbeddingWorker:  true,
		AgentRunWorker:   true,
		RuntimeName:      cfg.AI.AgentRuntime,
		Auth:             pb.NewAuthServiceClient(identityConn),
		Billing:          billingClient,
		BillingRequired:  strings.EqualFold(envOrDefault("AI_BILLING_MODE", "shadow"), "enforce"),
		AgentRunTimeout:  cfg.AI.TotalTimeout.Duration,
		Applications:     pb.NewApplicationOwnerServiceClient(recruitmentConn),
		AppList:          pb.NewApplicationServiceClient(recruitmentConn),
		Jobs:             pb.NewJobServiceClient(recruitmentConn),
	}))
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	instance, err := instanceFromAddr(listener.Addr().String(), bootstrap)
	if err != nil {
		_ = listener.Close()
		return err
	}
	discovery, err := setupNacos(context.Background(), bootstrap, instance)
	if err != nil {
		_ = listener.Close()
		return err
	}
	if discovery != nil {
		defer func() { _ = discovery.Deregister(context.Background(), instance) }()
	}

	options := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(1000),
		grpc.ChainUnaryInterceptor(server.UnaryAuthInterceptor(), logger.UnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(server.StreamAuthInterceptor(), logger.StreamServerInterceptor()),
	}
	if tlsOption, enabled, err := server.TransportSecurityOption(cfg.GRPC.TLSCertFile, cfg.GRPC.TLSKeyFile); err != nil {
		_ = listener.Close()
		return err
	} else if enabled {
		options = append(options, tlsOption)
	}
	grpcServer := grpc.NewServer(options...)
	if err := runtime.RegisterGRPC(grpcServer); err != nil {
		_ = listener.Close()
		return err
	}
	healthpb.RegisterHealthServer(grpcServer, server.NewHealthServer(sqlDB, redisClient, mqConn))
	outboxCtx, stopOutbox := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stopOutbox()
	go nativeStore.RunBillingSettlementOutbox(outboxCtx, billingClient)
	go runMemoryCleanupLoop(outboxCtx, log, memoryService, memoryCfg)
	go stopOnSignal(grpcServer)

	log.Info("log.service.listening",
		zap.String("addr", listener.Addr().String()),
		zap.String("nacos_service", instance.ServiceName),
		zap.String("env", bootstrap.ServiceEnv),
	)
	if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("grpc serve: %w", err)
	}
	return nil
}

func recruitingRuntimePolicy(cfg logicconfig.Config) recruitingruntime.RuntimePolicy {
	return recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{
		StructuredResumeParse:  boolSetting(cfg.Agent.Features.StructuredResumeParse, true),
		CandidateMatch:         boolSetting(cfg.Agent.Features.CandidateMatch, true),
		CandidateMatchSemantic: boolSetting(cfg.Agent.Features.CandidateMatchSemantic, true),
		CandidateMatchShadow:   boolSetting(cfg.Agent.Features.CandidateMatchShadow, false),
		Fallbacks:              boolSetting(cfg.Agent.Features.Fallbacks, true),
		ResumeParseTimeout:     cfg.Agent.Features.ResumeParseTimeout.Duration,
		CandidateMatchTimeout:  cfg.Agent.Features.CandidateMatchTimeout.Duration,
	})
}

func boolSetting(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func dialInternalGRPC(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(internalClientUnaryInterceptor()),
	)
}

func internalClientUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if incoming, ok := metadata.FromIncomingContext(ctx); ok {
			ctx = metadata.NewOutgoingContext(ctx, incoming.Copy())
		}
		if token := os.Getenv("GRPC_INTERNAL_TOKEN"); token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "x-internal-token", token)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func loadBootstrap(addr string) (platformconfig.Bootstrap, error) {
	return platformconfig.LoadWithLookup(func(key string) string {
		switch key {
		case "SERVICE_NAME":
			return envOrDefault(key, aiagentruntime.ServiceName)
		case "SERVICE_ENV":
			return envOrDefault(key, "local")
		case "SERVICE_VERSION":
			return envOrDefault(key, "dev")
		case "GRPC_ADDR":
			return envOrDefault(key, addr)
		default:
			return os.Getenv(key)
		}
	})
}

func setupNacos(ctx context.Context, bootstrap platformconfig.Bootstrap, instance nacos.Instance) (nacos.Discovery, error) {
	if strings.TrimSpace(bootstrap.NacosAddr) == "" && !bootstrap.StaticFallback {
		return nil, nil
	}
	configProvider, err := nacos.NewConfigProvider(nacos.ConfigOptions{
		Addresses: bootstrap.NacosAddr,
		Namespace: bootstrap.NacosNamespace,
		Group:     bootstrap.NacosGroup,
		Env:       bootstrap.ServiceEnv,
		StaticFallback: map[string]string{
			"ai-agent-service.yaml": "service:\n  name: ai-agent-service\n",
		},
		AllowFallback: bootstrap.StaticFallback,
		Username:      bootstrap.NacosUsername,
		Password:      bootstrap.NacosPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("init nacos config: %w", err)
	}
	if _, err := configProvider.Load(ctx, "ai-agent-service.yaml"); err != nil {
		return nil, fmt.Errorf("load ai-agent nacos config: %w", err)
	}
	discovery, err := nacos.NewDiscovery(nacos.DiscoveryOptions{
		Addresses: bootstrap.NacosAddr,
		Namespace: bootstrap.NacosNamespace,
		Group:     bootstrap.NacosGroup,
		Env:       bootstrap.ServiceEnv,
		StaticFallback: map[string][]nacos.Instance{
			instance.ServiceName: []nacos.Instance{instance},
		},
		AllowFallback: bootstrap.StaticFallback,
		Username:      bootstrap.NacosUsername,
		Password:      bootstrap.NacosPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("init nacos discovery: %w", err)
	}
	if err := discovery.Register(ctx, instance); err != nil {
		return nil, fmt.Errorf("register ai-agent in nacos: %w", err)
	}
	return discovery, nil
}

func instanceFromAddr(addr string, bootstrap platformconfig.Bootstrap) (nacos.Instance, error) {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return nacos.Instance{}, fmt.Errorf("parse grpc addr %q: %w", addr, err)
	}
	if host == "" || host == "::" {
		host = "127.0.0.1"
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return nacos.Instance{}, fmt.Errorf("parse grpc port %q: %w", portText, err)
	}
	instance := nacos.Instance{ServiceName: nacosServiceName, IP: host, Port: port, Healthy: true, Metadata: map[string]string{"service": bootstrap.ServiceName, "env": bootstrap.ServiceEnv, "version": bootstrap.ServiceVersion}}
	return instance, instance.Validate()
}

func stopOnSignal(grpcServer *grpc.Server) {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		grpcServer.Stop()
	}
}

func ensureLogicConfigPath() error {
	if os.Getenv("CONFIG_PATH") != "" {
		return nil
	}
	for _, candidate := range []string{filepath.Join("smart-recruit-commons", "config", "config.yaml"), filepath.Join("..", "smart-recruit-commons", "config", "config.yaml")} {
		if _, err := os.Stat(candidate); err == nil {
			return os.Setenv("CONFIG_PATH", candidate)
		}
	}
	return nil
}

func redisOptions(cfg logicconfig.Config) *redis.Options {
	return &redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB, PoolSize: cfg.Redis.PoolSize, MinIdleConns: cfg.Redis.MinIdleConns, DialTimeout: cfg.Redis.DialTimeout.Duration, ReadTimeout: cfg.Redis.ReadTimeout.Duration, WriteTimeout: cfg.Redis.WriteTimeout.Duration}
}

func mqConfig(cfg logicconfig.Config) mq.Config {
	return mq.Config{URL: cfg.RabbitMQ.URL, Exchange: cfg.RabbitMQ.Exchange, DLXExchange: cfg.RabbitMQ.DLXExchange, RetryExchange: cfg.RabbitMQ.RetryExchange, NotificationQueue: cfg.RabbitMQ.NotificationQueue, ResumeParseQueue: cfg.RabbitMQ.ResumeParseQueue, EmailQueue: cfg.RabbitMQ.EmailQueue, EmbeddingQueue: cfg.RabbitMQ.EmbeddingQueue, AgentRunQueue: cfg.RabbitMQ.AgentRunQueue, PrefetchCount: cfg.RabbitMQ.PrefetchCount, MaxRetries: cfg.RabbitMQ.MaxRetries, RetryDelay: cfg.RabbitMQ.RetryDelay.Duration}
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func runMemoryCleanupLoop(ctx context.Context, log *zap.Logger, memoryService *appmemory.Service, cfg appmemory.Config) {
	if memoryService == nil || !memoryService.Enabled() {
		return
	}
	interval := cfg.CleanupInterval
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	timeout := cfg.CleanupTimeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	retention := cfg.RevokedRetention
	if retention <= 0 {
		retention = 30 * 24 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	runCleanup := func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Warn("log.ai.memory_cleanup_panic", zap.Any("cause", recovered))
			}
		}()
		cleanupCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		result, err := memoryService.ExpireAndCleanup(cleanupCtx, retention)
		if err != nil {
			log.Warn("log.ai.memory_cleanup_failed", zap.String("cause", err.Error()))
			return
		}
		if result.ExpiredArchived > 0 || result.RevokedPurged > 0 || result.EmbeddingsInvalidated > 0 {
			log.Info("log.ai.memory_cleanup_completed",
				zap.Int64("expired_archived", result.ExpiredArchived),
				zap.Int64("revoked_purged", result.RevokedPurged),
				zap.Int64("embeddings_invalidated", result.EmbeddingsInvalidated),
			)
		}
	}
	runCleanup()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCleanup()
		}
	}
}

type unavailableEmbeddingConfigService struct {
	pb.UnimplementedEmbeddingConfigServiceServer
}
type unavailableLlmConfigService struct {
	pb.UnimplementedLlmConfigServiceServer
}
type noopAIService struct {
	pb.UnimplementedAIServiceServer
}
type noopLlmConfigService struct {
	pb.UnimplementedLlmConfigServiceServer
}
type noopPromptService struct {
	pb.UnimplementedPromptServiceServer
}
type noopAgentConfigService struct {
	pb.UnimplementedAgentConfigServiceServer
}
type noopMCPService struct {
	pb.UnimplementedMCPServiceServer
}
type noopAgentSkillService struct {
	pb.UnimplementedAgentSkillServiceServer
}
type noopRecruitingIntelligenceService struct {
	pb.UnimplementedRecruitingIntelligenceServiceServer
}
type noopEmbeddingConfigService struct {
	pb.UnimplementedEmbeddingConfigServiceServer
}
type noopPlatformAIControlPlaneService struct {
	pb.UnimplementedPlatformAIControlPlaneServiceServer
}
