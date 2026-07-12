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

	analyticsruntime "smart-recruit-analytics-service/internal/runtime"
	"smart-recruit-domain-go/repository"
	"smart-recruit-domain-go/service"
	platformconfig "smart-recruit-platform-go/config"
	"smart-recruit-platform-go/logger"
	"smart-recruit-platform-go/nacos"
	logicobservability "smart-recruit-platform-go/observability"
	platformobs "smart-recruit-platform-go/observability"
	"smart-recruit-platform-go/server"
	logicconfig "smart-recruit-platform-go/serviceconfig"
	"smart-recruit-proto/recruitment/pb"
)

const nacosServiceName = "analytics"

func main() {
	check := flag.Bool("check", false, "validate Analytics service runtime wiring and exit")
	serve := flag.Bool("serve", false, "start Analytics gRPC runtime")
	addr := flag.String("addr", envOrDefault("GRPC_ADDR", ":50067"), "Analytics gRPC listen address")
	flag.Parse()

	if *check {
		if err := checkRuntime(); err != nil {
			fmt.Fprintf(os.Stderr, "analytics-service check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "analytics-service runtime check passed")
		return
	}
	if *serve {
		if err := serveAnalytics(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "analytics-service failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	fmt.Fprintln(os.Stderr, "analytics-service requires --check or --serve")
	os.Exit(2)
}

func checkRuntime() error {
	runtime, err := analyticsruntime.New(analyticsruntime.Deps{
		Reporting: noopReportingAPI{},
	})
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	defer server.Stop()
	return runtime.RegisterGRPC(server)
}

func serveAnalytics(addr string) error {
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
	logicobservability.DefaultMetrics = logicobservability.NewRegistry(analyticsruntime.ServiceName)

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

	reporting := buildReportingService(db)
	runtime, err := analyticsruntime.New(analyticsruntime.Deps{
		Reporting: reporting,
	})
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
	healthpb.RegisterHealthServer(grpcServer, server.NewHealthServer(sqlDB, redisClient, nil))
	go stopOnSignal(grpcServer)

	log.Info("analytics grpc server listening",
		zap.String("addr", listener.Addr().String()),
		zap.String("nacos_service", instance.ServiceName),
		zap.String("env", bootstrap.ServiceEnv),
		zap.String("projection_mode", runtime.Projection.ProjectionMode),
	)
	if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("grpc serve: %w", err)
	}
	return nil
}

func buildReportingService(db *gorm.DB) analyticsruntime.ReportingAPI {
	authzRepo := repository.NewAuthzRepo(db)
	return service.NewAnalyticsService(
		repository.NewAnalyticsRepo(db),
		authzRepo,
		service.NewServiceAuthorizer(authzRepo, nil),
	)
}

func loadBootstrap(addr string) (platformconfig.Bootstrap, error) {
	return platformconfig.LoadWithLookup(func(key string) string {
		switch key {
		case "SERVICE_NAME":
			return envOrDefault(key, analyticsruntime.ServiceName)
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
			"analytics-service.yaml": "service:\n  name: analytics-service\nprojection:\n  mode: domain-event-projection-read-models\n",
		},
		AllowFallback: bootstrap.StaticFallback,
		Username:      bootstrap.NacosUsername,
		Password:      bootstrap.NacosPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("init nacos config: %w", err)
	}
	if _, err := configProvider.Load(ctx, "analytics-service.yaml"); err != nil {
		return nil, fmt.Errorf("load analytics nacos config: %w", err)
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
		return nil, fmt.Errorf("register analytics in nacos: %w", err)
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
		filepath.Join("smart-recruit-domain-go", "config", "config.yaml"),
		filepath.Join("..", "smart-recruit-domain-go", "config", "config.yaml"),
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

type noopReportingAPI struct{}

func (noopReportingAPI) GetDashboardReport(context.Context, *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error) {
	return &pb.GetDashboardReportResponse{}, nil
}

func (noopReportingAPI) GetFunnelReport(context.Context, *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error) {
	return &pb.GetFunnelReportResponse{}, nil
}

func (noopReportingAPI) GetTimeInStageReport(context.Context, *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error) {
	return &pb.GetTimeInStageReportResponse{}, nil
}

func (noopReportingAPI) GetInterviewOfferMetrics(context.Context, *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error) {
	return &pb.GetInterviewOfferMetricsResponse{}, nil
}
