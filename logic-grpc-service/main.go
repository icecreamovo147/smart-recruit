package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
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

	"logic-grpc-service/ai"
	"logic-grpc-service/config"
	"logic-grpc-service/mq"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
	"logic-grpc-service/server"
	"logic-grpc-service/service"
)

func main() {
	workerOnly := flag.Bool("worker-only", false, "run background workers without serving gRPC")
	flag.Parse()

	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("load config: %v\n", err)
		panic("load config: " + err.Error())
	}

	if err := server.ValidateInternalToken(); err != nil {
		fmt.Printf("gRPC internal token validation failed: %v\n", err)
		panic("gRPC internal token validation: " + err.Error())
	}

	logger.Set(logger.New("info"))
	log := logger.L()
	log.Info("starting logic-grpc-service")

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatal("connect mysql failed", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.MySQL.ConnMaxLifetime.Duration)
	sqlDB.SetConnMaxIdleTime(cfg.MySQL.ConnMaxIdleTime.Duration)
	log.Info("mysql connected", zap.Int("max_open", cfg.MySQL.MaxOpenConns), zap.Int("max_idle", cfg.MySQL.MaxIdleConns))

	userRepo := repository.NewUserRepo(db)
	tokenRepo := repository.NewRefreshTokenRepo(db)
	jobRepo := repository.NewJobRepo(db)
	profileRepo := repository.NewProfileRepo(db)
	resumeRepo := repository.NewResumeRepo(db)
	applicationRepo := repository.NewApplicationRepo(db)
	interviewRepo := repository.NewInterviewRepo(db)
	offerRepo := repository.NewOfferRepo(db)
	chatRepo := repository.NewChatRepo(db)
	summaryRepo := repository.NewSessionSummaryRepo(db)
	toolTraceRepo := repository.NewToolTraceRepo(db)
	memoryRepo := repository.NewMemoryRepo(db)
	notificationRepo := repository.NewNotificationRepo(db)
	outboxRepo := repository.NewOutboxRepo(db)
	inviteCodeRepo := repository.NewInviteCodeRepo(db)
	departmentRepo := repository.NewDepartmentRepo(db)
	locationRepo := repository.NewJobLocationRepo(db)
	deptLocationRepo := repository.NewDepartmentLocationRepo(db)
	usageLogRepo := repository.NewUsageLogRepo(db)
	authzRepo := repository.NewAuthzRepo(db)

	mqConn, err := mq.New(mq.Config{
		URL:               cfg.RabbitMQ.URL,
		Exchange:          cfg.RabbitMQ.Exchange,
		DLXExchange:       cfg.RabbitMQ.DLXExchange,
		RetryExchange:     cfg.RabbitMQ.RetryExchange,
		NotificationQueue: cfg.RabbitMQ.NotificationQueue,
		ResumeParseQueue:  cfg.RabbitMQ.ResumeParseQueue,
		PrefetchCount:     cfg.RabbitMQ.PrefetchCount,
		MaxRetries:        cfg.RabbitMQ.MaxRetries,
		RetryDelay:        cfg.RabbitMQ.RetryDelay.Duration,
	})
	if err != nil {
		log.Warn("rabbitmq not available, background workers will reconnect", zap.Error(err))
	} else {
		log.Info("rabbitmq connected", zap.String("url", cfg.RabbitMQ.URL))
	}
	defer mqConn.Close()

	ossClient, err := oss.NewStorage(oss.Config{
		Provider:        cfg.OSS.Provider,
		Endpoint:        cfg.OSS.Endpoint,
		AccessKeyID:     cfg.OSS.AccessKeyID,
		AccessKeySecret: cfg.OSS.AccessKeySecret,
		BucketName:      cfg.OSS.BucketName,
		PublicBaseURL:   cfg.OSS.PublicBaseURL,
	})
	if err != nil {
		log.Fatal("init object storage failed", zap.Error(err))
	}
	ossProvider := strings.TrimSpace(cfg.OSS.Provider)
	if ossProvider == "" {
		ossProvider = oss.ProviderTencentCOS
	}
	log.Info("object storage initialized", zap.String("provider", ossProvider))

	var healthRedis *redis.Client
	var presignCache *oss.PresignCache
	if cfg.Redis.Addr != "" {
		healthRedis = redis.NewClient(redisOptions(cfg))
		presignCache = oss.NewPresignCacheWithOptions(redisOptions(cfg))
		ossClient.SetPresignCache(presignCache)
		log.Info("redis presign cache enabled", zap.String("addr", cfg.Redis.Addr))
	}

	var notifCache *cache.NotificationCache
	var jobCache *cache.JobCache
	if cfg.Redis.Addr != "" {
		notifCache = cache.NewNotificationCacheWithOptions(redisOptions(cfg))
		log.Info("redis notification cache enabled", zap.String("addr", cfg.Redis.Addr))
		jobCache = cache.NewJobCacheWithOptions(redisOptions(cfg))
		log.Info("redis job cache enabled", zap.String("addr", cfg.Redis.Addr))
	}

	aiClient, err := ai.NewClient(ctx, cfg.AI.APIKey, cfg.AI.Model, cfg.AI.BaseURL, ai.Options{
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
	if err != nil {
		log.Fatal("init ai client failed", zap.Error(err))
	}
	log.Info("ai client initialized", zap.String("model", cfg.AI.Model))

	services := service.NewServices(
		healthRedis,
		userRepo, tokenRepo,
		jobRepo, profileRepo, resumeRepo, applicationRepo, interviewRepo, offerRepo, chatRepo,
		summaryRepo, toolTraceRepo, memoryRepo, notificationRepo, outboxRepo, inviteCodeRepo,
		departmentRepo, locationRepo, deptLocationRepo,
		usageLogRepo, authzRepo,
		notifCache, jobCache,
		ossClient, aiClient, mqConn, cfg, cfg.JWT.Secret,
	)

	// Bootstrap initial admin: promote user specified by INITIAL_ADMIN_USERNAME
	// to recruiting_admin + recruiter via the RBAC system, with legacy role=3
	// for backward compatibility.
	if adminName := os.Getenv("INITIAL_ADMIN_USERNAME"); adminName != "" {
		adminUser, err := userRepo.GetByUsername(ctx, adminName)
		if err != nil {
			log.Warn("bootstrap admin: lookup failed", zap.String("username", adminName), zap.Error(err))
		} else if adminUser == nil {
			log.Warn("bootstrap admin: user not found", zap.String("username", adminName))
		} else {
			hasAdmin, err := authzRepo.HasActiveAdminRole(ctx, uint64(adminUser.ID))
				if err != nil {
					log.Warn("bootstrap admin: RBAC role check failed", zap.Error(err))
				}
				needsPromotion := !hasAdmin || adminUser.AccountType != "staff"
				if needsPromotion {
				if err := userRepo.UpdateRole(ctx, adminUser.ID, 3); err != nil {
					log.Warn("bootstrap admin: update legacy role failed", zap.Error(err))
				}
				if err := userRepo.UpdateAccountType(ctx, adminUser.ID, "staff"); err != nil {
					log.Warn("bootstrap admin: update account_type failed", zap.Error(err))
				}
				if err := authzRepo.MigrateLegacyUserRoles(ctx, uint64(adminUser.ID), 3); err != nil {
					log.Warn("bootstrap admin: RBAC assignment failed", zap.Error(err))
				}
				log.Info("bootstrap admin: promoted to recruiting_admin+recruiter",
					zap.String("username", adminName), zap.Int64("user_id", adminUser.ID))
			}
		}
	}

	// Migrate all legacy users (those without user_roles records) to RBAC.
	// Must run before gRPC server starts so existing users have valid roles.
	if migrated, err := authzRepo.MigrateAllLegacyUsers(ctx); err != nil {
		log.Warn("legacy user migration completed with errors", zap.Error(err))
	} else {
		log.Info("legacy user migration completed", zap.Int64("migrated", migrated))
	}

	// Start background workers
	bgCtx, cancelBg := context.WithCancel(ctx)
	defer cancelBg()
	workersEnabled := *workerOnly || !envBool("DISABLE_BACKGROUND_WORKERS")
	if workersEnabled {
		services.OutboxPublisher.Start(bgCtx)
		if err := services.NotificationConsumer.Start(bgCtx, mqConn); err != nil {
			log.Warn("notification consumer start failed", zap.Error(err))
		}
		if err := services.ResumeParseConsumer.Start(bgCtx, mqConn); err != nil {
			log.Warn("resume parse consumer start failed", zap.Error(err))
		}
		go mqConn.KeepAlive(bgCtx, cfg.RabbitMQ.ReconnectInterval.Duration)
	} else {
		log.Info("background workers disabled")
	}

	if *workerOnly {
		log.Info("logic worker started")
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Info("received signal, shutting down worker", zap.String("signal", sig.String()))
		cancelBg()
		mqConn.Close()
		if notifCache != nil {
			_ = notifCache.Close()
		}
		if jobCache != nil {
			_ = jobCache.Close()
		}
		if presignCache != nil {
			_ = presignCache.Close()
		}
		if healthRedis != nil {
			_ = healthRedis.Close()
		}
		_ = sqlDB.Close()
		log.Info("logic worker stopped")
		return
	}

	addr := fmt.Sprintf(":%d", cfg.GRPC.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("listen failed", zap.String("addr", addr), zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.MaxConcurrentStreams(1000),
		grpc.UnaryInterceptor(server.UnaryAuthInterceptor()),
		grpc.StreamInterceptor(server.StreamAuthInterceptor()),
	)
	recruitmentServer := server.New(services)
	pb.RegisterAuthServiceServer(grpcServer, recruitmentServer)
	pb.RegisterJobServiceServer(grpcServer, recruitmentServer)
	pb.RegisterCandidateServiceServer(grpcServer, recruitmentServer)
	pb.RegisterApplicationServiceServer(grpcServer, recruitmentServer)
	pb.RegisterAIServiceServer(grpcServer, recruitmentServer)
	pb.RegisterNotificationServiceServer(grpcServer, recruitmentServer)
	pb.RegisterInterviewServiceServer(grpcServer, recruitmentServer)
	pb.RegisterOfferServiceServer(grpcServer, recruitmentServer)
	pb.RegisterAdminServiceServer(grpcServer, recruitmentServer)
	healthpb.RegisterHealthServer(grpcServer, server.NewHealthServer(sqlDB, healthRedis, mqConn))

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Info("received signal, shutting down", zap.String("signal", sig.String()))

		// Stop background workers first
		cancelBg()
		log.Info("background workers stopped")

		done := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			log.Warn("grpc graceful stop timeout, forcing stop")
			grpcServer.Stop()
		}
		mqConn.Close()
		if notifCache != nil {
			_ = notifCache.Close()
		}
		if jobCache != nil {
			_ = jobCache.Close()
		}
		if presignCache != nil {
			_ = presignCache.Close()
		}
		if healthRedis != nil {
			_ = healthRedis.Close()
		}
		_ = sqlDB.Close()
	}()

	log.Info("grpc server listening", zap.String("addr", addr))
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal("grpc serve failed", zap.Error(err))
	}
	log.Info("server stopped")
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

func envBool(key string) bool {
	value := os.Getenv(key)
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}
