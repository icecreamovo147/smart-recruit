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

	logicconfig "logic-grpc-service/config"
	"logic-grpc-service/email"
	"logic-grpc-service/mq"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/pkg/logger"
	logicobservability "logic-grpc-service/pkg/observability"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
	"logic-grpc-service/server"
	"logic-grpc-service/service"
	notificationruntime "smart-recruit-notification-service/internal/runtime"
	platformconfig "smart-recruit-platform-go/config"
	"smart-recruit-platform-go/nacos"
	platformobs "smart-recruit-platform-go/observability"
)

const nacosServiceName = "notification"

func main() {
	check := flag.Bool("check", false, "validate Notification service runtime wiring and exit")
	serve := flag.Bool("serve", false, "start Notification gRPC runtime")
	addr := flag.String("addr", envOrDefault("GRPC_ADDR", ":50065"), "Notification gRPC listen address")
	flag.Parse()

	if *check {
		if err := checkRuntime(); err != nil {
			fmt.Fprintf(os.Stderr, "notification-service check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "notification-service runtime check passed")
		return
	}
	if *serve {
		if err := serveNotification(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "notification-service failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	fmt.Fprintln(os.Stderr, "notification-service requires --check or --serve")
	os.Exit(2)
}

func checkRuntime() error {
	runtime, err := notificationruntime.New(notificationruntime.Deps{Notification: noopNotificationAPI{}})
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	defer server.Stop()
	return runtime.RegisterGRPC(server)
}

func serveNotification(addr string) error {
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
		return fmt.Errorf("load logic config: %w", err)
	}
	if err := server.ValidateInternalToken(); err != nil {
		return fmt.Errorf("gRPC internal token validation: %w", err)
	}
	if err := logger.Init(cfg.Logging); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	log := logger.L()
	logicobservability.DefaultMetrics = logicobservability.NewRegistry(notificationruntime.ServiceName)

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		TranslateError: true,
		Logger:         logger.NewGormLogger(&cfg.Logging.Gorm),
	})
	if err != nil {
		return fmt.Errorf("connect mysql: %w", err)
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

	var redisClient *redis.Client
	if cfg.Redis.Addr != "" {
		redisClient = redis.NewClient(redisOptions(cfg))
		defer redisClient.Close()
	}

	metricsServer, err := server.StartMetricsServer(cfg.Observability.MetricsAddr)
	if err != nil {
		return fmt.Errorf("start metrics server: %w", err)
	}
	defer server.ShutdownMetricsServer(context.Background(), metricsServer)

	notificationRuntime, mqConn, cleanup, err := buildNotificationRuntime(cfg, db, redisClient)
	if err != nil {
		return err
	}
	defer cleanup()
	runtime, err := notificationruntime.New(notificationruntime.Deps{
		Notification: notificationRuntime.Notification,
		Runtime:      notificationRuntime,
	})
	if err != nil {
		return err
	}
	if runtimeErrors := notificationRuntime.Start(context.Background(), mqConn); len(runtimeErrors) > 0 {
		return fmt.Errorf("start notification runtime: %s: %w", runtimeErrors[0].Component, runtimeErrors[0].Err)
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
		grpc.ChainUnaryInterceptor(
			server.UnaryAuthInterceptor(),
			logger.UnaryServerInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			server.StreamAuthInterceptor(),
			logger.StreamServerInterceptor(),
		),
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
	go stopOnSignal(grpcServer)

	log.Info("notification grpc server listening",
		zap.String("addr", listener.Addr().String()),
		zap.String("nacos_service", instance.ServiceName),
		zap.String("env", bootstrap.ServiceEnv),
	)
	if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("grpc serve: %w", err)
	}
	return nil
}

func buildNotificationRuntime(cfg logicconfig.Config, db *gorm.DB, redisClient *redis.Client) (*service.NotificationRuntime, *mq.Conn, func(), error) {
	authzRepo := repository.NewAuthzRepo(db)
	var notificationCache *cache.NotificationCache
	if redisClient != nil {
		notificationCache = cache.NewNotificationCacheWithOptions(redisOptions(cfg))
	}
	renderer, err := email.NewRenderer(cfg.FrontendBaseURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("init email renderer: %w", err)
	}
	sender, err := email.NewSender(email.SMTPConfig{
		Host:        cfg.SMTP.Host,
		Port:        cfg.SMTP.Port,
		Username:    cfg.SMTP.Username,
		Password:    string(cfg.SMTP.Password),
		FromAddress: cfg.SMTP.FromAddress,
		FromName:    cfg.SMTP.FromName,
		TLS:         cfg.SMTP.TLS,
		Required:    cfg.SMTP.Required,
	})
	if err != nil {
		if notificationCache != nil {
			_ = notificationCache.Close()
		}
		return nil, nil, nil, fmt.Errorf("init email sender: %w", err)
	}
	mqConn, err := mq.New(mqConfig(cfg))
	if err != nil {
		_ = sender.Close()
		if notificationCache != nil {
			_ = notificationCache.Close()
		}
		return nil, nil, nil, fmt.Errorf("connect rabbitmq: %w", err)
	}
	runtime := service.NewNotificationRuntime(service.NotificationRuntimeDeps{
		Users:         repository.NewUserRepo(db),
		Notifications: repository.NewNotificationRepo(db),
		Outbox:        repository.NewOutboxRepo(db),
		Inbox:         repository.NewInboxRepo(db),
		EmailLog:      repository.NewEmailLogRepo(db),
		Cache:         notificationCache,
		MQ:            mqConn,
		Authz:         service.NewServiceAuthorizer(authzRepo, nil),
		EmailRenderer: renderer,
		EmailSender:   sender,
	})
	cleanup := func() {
		mqConn.Close()
		_ = sender.Close()
		if notificationCache != nil {
			_ = notificationCache.Close()
		}
	}
	return runtime, mqConn, cleanup, nil
}

func mqConfig(cfg logicconfig.Config) mq.Config {
	return mq.Config{
		URL:               cfg.RabbitMQ.URL,
		Exchange:          cfg.RabbitMQ.Exchange,
		DLXExchange:       cfg.RabbitMQ.DLXExchange,
		RetryExchange:     cfg.RabbitMQ.RetryExchange,
		NotificationQueue: cfg.RabbitMQ.NotificationQueue,
		ResumeParseQueue:  cfg.RabbitMQ.ResumeParseQueue,
		EmailQueue:        cfg.RabbitMQ.EmailQueue,
		EmbeddingQueue:    cfg.RabbitMQ.EmbeddingQueue,
		AgentRunQueue:     cfg.RabbitMQ.AgentRunQueue,
		PrefetchCount:     cfg.RabbitMQ.PrefetchCount,
		MaxRetries:        cfg.RabbitMQ.MaxRetries,
		RetryDelay:        cfg.RabbitMQ.RetryDelay.Duration,
	}
}

func loadBootstrap(addr string) (platformconfig.Bootstrap, error) {
	return platformconfig.LoadWithLookup(func(key string) string {
		switch key {
		case "SERVICE_NAME":
			return envOrDefault(key, notificationruntime.ServiceName)
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
			"notification-service.yaml": "service:\n  name: notification-service\n",
		},
		AllowFallback: bootstrap.StaticFallback,
		Username:      bootstrap.NacosUsername,
		Password:      bootstrap.NacosPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("init nacos config: %w", err)
	}
	if _, err := configProvider.Load(ctx, "notification-service.yaml"); err != nil {
		return nil, fmt.Errorf("load notification nacos config: %w", err)
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
		return nil, fmt.Errorf("register notification in nacos: %w", err)
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
	instance := nacos.Instance{
		ServiceName: nacosServiceName,
		IP:          host,
		Port:        port,
		Healthy:     true,
		Metadata: map[string]string{
			"service": bootstrap.ServiceName,
			"env":     bootstrap.ServiceEnv,
			"version": bootstrap.ServiceVersion,
		},
	}
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
	for _, candidate := range []string{
		filepath.Join("logic-grpc-service", "config", "config.yaml"),
		filepath.Join("..", "logic-grpc-service", "config", "config.yaml"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return os.Setenv("CONFIG_PATH", candidate)
		}
	}
	return nil
}

func redisOptions(cfg logicconfig.Config) *redis.Options {
	return &redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout.Duration,
		ReadTimeout:  cfg.Redis.ReadTimeout.Duration,
		WriteTimeout: cfg.Redis.WriteTimeout.Duration,
	}
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

type noopNotificationAPI struct {
	notificationruntime.NotificationAPI
}

var _ notificationruntime.NotificationAPI = noopNotificationAPI{}

func (noopNotificationAPI) ListNotifications(context.Context, *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	return &pb.ListNotificationsResponse{}, nil
}
