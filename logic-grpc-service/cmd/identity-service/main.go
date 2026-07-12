package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"logic-grpc-service/config"
	identityruntime "logic-grpc-service/internal/identity/runtime"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
	"logic-grpc-service/server"
	"logic-grpc-service/service"
)

func main() {
	describe := flag.Bool("describe", false, "print the Identity service skeleton descriptor")
	check := flag.Bool("check", false, "validate the Identity service skeleton and exit")
	serve := flag.Bool("serve", false, "start the Identity gRPC API runtime")
	addr := flag.String("addr", ":50051", "Identity gRPC listen address when --serve is set")
	flag.Parse()

	descriptor, err := identityruntime.NewDescriptor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "identity-service skeleton invalid: %v\n", err)
		os.Exit(1)
	}

	if *check {
		return
	}
	if *describe {
		printDescriptor(descriptor)
		return
	}
	if *serve {
		if err := serveIdentity(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "identity-service failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Fprintln(os.Stderr, "identity-service skeleton is compile-safe but intentionally unrouted.")
	fmt.Fprintln(os.Stderr, "Run with --describe for details, --check for validation, or --serve for explicit API runtime startup.")
	os.Exit(2)
}

func printDescriptor(descriptor identityruntime.Descriptor) {
	fmt.Printf("name: %s\n", descriptor.Unit.Name)
	fmt.Printf("role: %s\n", descriptor.Unit.Role)
	fmt.Printf("command: %s\n", descriptor.Unit.Command)
	fmt.Printf("image: %s\n", descriptor.Unit.Image)
	fmt.Printf("config_prefix: %s\n", descriptor.Unit.ConfigPrefix)
	fmt.Printf("health: %s\n", descriptor.Unit.Health)
	fmt.Printf("cutover_mode: %s\n", descriptor.CutoverMode)
	fmt.Printf("traffic_enabled: %t\n", descriptor.TrafficEnabled)
	fmt.Printf("startup_mode: %s\n", descriptor.StartupMode)
	fmt.Printf("extracted_apis: %s\n", strings.Join(descriptor.ExtractedAPIs, ", "))
	fmt.Printf("notes: %s\n", strings.Join(descriptor.Notes, "; "))
}

func serveIdentity(addr string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := server.ValidateInternalToken(); err != nil {
		return fmt.Errorf("gRPC internal token validation: %w", err)
	}
	if err := logger.Init(cfg.Logging); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	log := logger.L()

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

	userRepo := repository.NewUserRepo(db)
	tokenRepo := repository.NewRefreshTokenRepo(db)
	authzRepo := repository.NewAuthzRepo(db)
	inviteCodeRepo := repository.NewInviteCodeRepo(db)
	usageLogRepo := repository.NewUsageLogRepo(db)
	analyticsRepo := repository.NewAnalyticsRepo(db)
	serviceAuth := service.NewServiceAuthorizer(authzRepo, nil)

	authSvc := service.NewAuthService(userRepo, tokenRepo, authzRepo, inviteCodeRepo, cfg.JWT.Secret)
	adminSvc := service.NewAdminService(inviteCodeRepo, usageLogRepo, userRepo, authzRepo, redisClient, serviceAuth)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo, authzRepo, serviceAuth)
	runtime, err := identityruntime.New(identityruntime.Deps{
		Auth:  authSvc,
		Admin: adminSvc,
		Audit: analyticsSvc,
	})
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(1000),
		grpc.ChainUnaryInterceptor(
			server.UnaryAuthInterceptor(),
			logger.UnaryServerInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			server.StreamAuthInterceptor(),
			logger.StreamServerInterceptor(),
		),
	)
	if err := runtime.RegisterGRPC(grpcServer); err != nil {
		return err
	}
	healthpb.RegisterHealthServer(grpcServer, server.NewHealthServer(sqlDB, redisClient, nil))

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
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
	}()

	log.Info("identity grpc server listening",
		zap.String("addr", addr),
		zap.Bool("gateway_traffic_enabled", runtime.Descriptor.TrafficEnabled),
		zap.Strings("extracted_apis", runtime.Descriptor.ExtractedAPIs),
	)
	if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("grpc serve: %w", err)
	}
	return nil
}

func redisOptions(cfg config.Config) *redis.Options {
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
