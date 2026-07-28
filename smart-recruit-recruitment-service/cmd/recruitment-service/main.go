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
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"smart-recruit-commons/oss"
	"smart-recruit-platform-go/businessclock"
	"smart-recruit-platform-go/config"
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
	recruitmentpersistence "smart-recruit-recruitment-service/internal/infrastructure/persistence"
	recruitmentruntime "smart-recruit-recruitment-service/internal/runtime"
)

const nacosServiceName = "recruitment"

func main() {
	if err := i18n.ConfigureFromEnv(); err != nil {
		logger.L().Error("log.service.config_failed", zap.String("cause", err.Error()))
		os.Exit(2)
	}
	businessclock.Configure()
	check := flag.Bool("check", false, "validate Recruitment service runtime wiring and exit")
	serve := flag.Bool("serve", false, "start Recruitment gRPC runtime")
	addr := flag.String("addr", envOrDefault("GRPC_ADDR", ":50062"), "Recruitment gRPC listen address")
	flag.Parse()
	if *check {
		if err := checkRuntime(); err != nil {
			logger.L().Error("log.service.check_failed", zap.String("service", "recruitment-service"), zap.String("cause", err.Error()))
			os.Exit(1)
		}
		logger.L().Info("log.service.check_passed", zap.String("service", "recruitment-service"))
		return
	}
	if *serve {
		if err := serveRecruitment(*addr); err != nil {
			logger.L().Error("log.service.serve_failed", zap.String("service", "recruitment-service"), zap.String("cause", err.Error()))
			os.Exit(1)
		}
		return
	}
	logger.L().Error("log.service.arguments_required", zap.String("service", "recruitment-service"))
	os.Exit(2)
}

func checkRuntime() error {
	runtime, err := recruitmentruntime.New(recruitmentruntime.Deps{
		Job:                      noopJobAPI{},
		JobTaxonomy:              noopJobTaxonomyAPI{},
		TaxonomyAdmin:            noopTaxonomyAdminAPI{},
		Admin:                    noopRecruitmentAdminAPI{},
		UsageStats:               noopUsageStatsAPI{},
		Candidate:                noopCandidateAPI{},
		Application:              noopApplicationAPI{},
		ApplicationOwnerContract: noopApplicationOwnerContractAPI{},
		Collaboration:            noopCollaborationService{},
	})
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	defer server.Stop()
	return runtime.RegisterGRPC(server)
}

func serveRecruitment(addr string) error {
	if err := ensureLogicConfigPath(); err != nil {
		return err
	}
	bootstrap, err := loadBootstrap(addr)
	if err != nil {
		return err
	}
	traceRuntime, err := platformobs.NewTraceRuntime(context.Background(), platformobs.TraceConfig{
		ServiceName:    bootstrap.ServiceName,
		ServiceVersion: bootstrap.ServiceVersion,
		Env:            bootstrap.ServiceEnv,
	})
	if err != nil {
		return err
	}
	defer func() { _ = traceRuntime.Shutdown(context.Background()) }()

	cfg, err := logicconfig.Load()
	if err != nil {
		return err
	}
	if err := server.ValidateInternalToken(); err != nil {
		return err
	}
	if err := serviceLoggerInit(cfg); err != nil {
		return err
	}
	logicobservability.DefaultMetrics = logicobservability.NewRegistry(recruitmentruntime.ServiceName)

	dsn, err := mysqltime.NormalizeDSN(cfg.MySQL.DSN)
	if err != nil {
		return err
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	if err := db.Use(tenantgorm.NewWithMixed(
		[]string{"jobs", "applications", "application_status_transitions", "invite_codes", "departments", "job_locations", "department_locations", "candidate_notes", "candidate_tags", "candidate_tag_assignments", "follow_up_tasks", "interview_schedules", "interview_feedback", "offers", "offer_events"},
		[]string{"event_outbox", "third_party_usage_logs", "ai_usage_events"},
	)); err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	if err := mysqltime.ValidateSession(context.Background(), sqlDB); err != nil {
		return err
	}
	redisClient, err := newRecruitmentRedisClient(cfg)
	if err != nil {
		return err
	}
	defer redisClient.Close()
	metricsServer, err := server.StartMetricsServer(cfg.Observability.MetricsAddr)
	if err != nil {
		return err
	}
	defer server.ShutdownMetricsServer(context.Background(), metricsServer)

	ossClient, err := oss.NewStorage(oss.Config{
		Provider:        cfg.OSS.Provider,
		Endpoint:        cfg.OSS.Endpoint,
		AccessKeyID:     cfg.OSS.AccessKeyID,
		AccessKeySecret: cfg.OSS.AccessKeySecret,
		BucketName:      cfg.OSS.BucketName,
		PublicBaseURL:   cfg.OSS.PublicBaseURL,
	})
	if err != nil {
		return err
	}
	presignCache, err := attachOSSPresignCache(ossClient, cfg)
	if err != nil {
		return err
	}
	defer presignCache.Close()

	nativeBundle, err := recruitmentpersistence.NewNativeBundle(recruitmentpersistence.NativeOptions{
		DB:    db,
		OSS:   ossClient,
		Redis: redisClient,
	})
	if err != nil {
		return err
	}
	runtime, err := recruitmentruntime.New(recruitmentruntime.Deps{
		Job:                      nativeBundle.Job,
		JobTaxonomy:              nativeBundle.JobTaxonomy,
		TaxonomyAdmin:            nativeBundle.TaxonomyAdmin,
		Admin:                    nativeBundle.Admin,
		UsageStats:               nativeBundle.UsageStats,
		Candidate:                nativeBundle.Candidate,
		Application:              nativeBundle.Application,
		ApplicationOwnerContract: nativeBundle.ApplicationOwnerContract,
		Collaboration:            nativeBundle.Collaboration,
	})
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
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
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(server.UnaryAuthInterceptor()),
		grpc.ChainStreamInterceptor(server.StreamAuthInterceptor()),
	)
	if err := runtime.RegisterGRPC(grpcServer); err != nil {
		_ = listener.Close()
		return err
	}
	healthpb.RegisterHealthServer(grpcServer, server.NewHealthServer(sqlDB, redisClient, nil))
	go stopOnSignal(grpcServer)
	logger.L().Info("log.service.listening",
		zap.String("service", instance.ServiceName),
		zap.String("addr", listener.Addr().String()),
	)
	if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}
	return nil
}

func serviceLoggerInit(logicconfig.Config) error { return nil }

func loadBootstrap(addr string) (config.Bootstrap, error) {
	return config.LoadWithLookup(func(key string) string {
		switch key {
		case "SERVICE_NAME":
			return envOrDefault(key, recruitmentruntime.ServiceName)
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

func setupNacos(ctx context.Context, bootstrap config.Bootstrap, instance nacos.Instance) (nacos.Discovery, error) {
	if strings.TrimSpace(bootstrap.NacosAddr) == "" && !bootstrap.StaticFallback {
		return nil, nil
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
		return nil, err
	}
	return discovery, discovery.Register(ctx, instance)
}

func instanceFromAddr(addr string, bootstrap config.Bootstrap) (nacos.Instance, error) {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return nacos.Instance{}, err
	}
	if host == "" || host == "::" {
		host = "127.0.0.1"
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return nacos.Instance{}, err
	}
	instance := nacos.Instance{ServiceName: nacosServiceName, IP: host, Port: port, Healthy: true, Metadata: map[string]string{"service": bootstrap.ServiceName, "env": bootstrap.ServiceEnv}}
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

func newRecruitmentRedisClient(cfg logicconfig.Config) (*redis.Client, error) {
	if strings.TrimSpace(cfg.Redis.Addr) == "" {
		return nil, fmt.Errorf("recruitment redis addr is required for resume presign cache")
	}
	rdb := redis.NewClient(redisOptions(cfg))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("connect redis for resume presign cache: %w", err)
	}
	return rdb, nil
}

func attachOSSPresignCache(storage oss.Storage, cfg logicconfig.Config) (*oss.PresignCache, error) {
	if storage == nil {
		return nil, fmt.Errorf("recruitment oss storage is required for resume presign cache")
	}
	if strings.TrimSpace(cfg.Redis.Addr) == "" {
		return nil, fmt.Errorf("recruitment redis addr is required for resume presign cache")
	}
	cache := oss.NewPresignCacheWithOptions(redisOptions(cfg))
	storage.SetPresignCache(cache)
	return cache, nil
}

func redisOptions(cfg logicconfig.Config) *redis.Options {
	return &redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB, PoolSize: cfg.Redis.PoolSize}
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

type noopJobAPI struct{ recruitmentruntime.JobAPI }
type noopJobTaxonomyAPI struct {
	recruitmentruntime.JobTaxonomyAPI
}
type noopTaxonomyAdminAPI struct {
	recruitmentruntime.TaxonomyAdminAPI
}
type noopRecruitmentAdminAPI struct {
	recruitmentruntime.RecruitmentAdminAPI
}
type noopUsageStatsAPI struct {
	recruitmentruntime.UsageStatsAPI
}
type noopCandidateAPI struct {
	recruitmentruntime.CandidateAPI
}
type noopApplicationAPI struct {
	recruitmentruntime.ApplicationAPI
}
type noopApplicationOwnerContractAPI struct {
	recruitmentruntime.ApplicationOwnerContractAPI
}

type noopCollaborationService struct {
	pb.UnimplementedCollaborationServiceServer
}
