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
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"smart-recruit-domain-go/oss"
	"smart-recruit-domain-go/repository"
	"smart-recruit-domain-go/service"
	"smart-recruit-platform-go/config"
	"smart-recruit-platform-go/nacos"
	logicobservability "smart-recruit-platform-go/observability"
	platformobs "smart-recruit-platform-go/observability"
	"smart-recruit-platform-go/server"
	logicconfig "smart-recruit-platform-go/serviceconfig"
	"smart-recruit-proto/recruitment/pb"
	recruitmentruntime "smart-recruit-recruitment-service/internal/runtime"
)

const nacosServiceName = "recruitment"

func main() {
	check := flag.Bool("check", false, "validate Recruitment service runtime wiring and exit")
	serve := flag.Bool("serve", false, "start Recruitment gRPC runtime")
	addr := flag.String("addr", envOrDefault("GRPC_ADDR", ":50062"), "Recruitment gRPC listen address")
	flag.Parse()
	if *check {
		if err := checkRuntime(); err != nil {
			fmt.Fprintf(os.Stderr, "recruitment-service check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "recruitment-service runtime check passed")
		return
	}
	if *serve {
		if err := serveRecruitment(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "recruitment-service failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	fmt.Fprintln(os.Stderr, "recruitment-service requires --check or --serve")
	os.Exit(2)
}

func checkRuntime() error {
	runtime, err := recruitmentruntime.New(recruitmentruntime.Deps{
		Job:           noopJobAPI{},
		JobTaxonomy:   noopJobTaxonomyAPI{},
		TaxonomyAdmin: noopTaxonomyAdminAPI{},
		Admin:         noopRecruitmentAdminAPI{},
		UsageStats:    noopUsageStatsAPI{},
		Candidate:     noopCandidateAPI{},
		Application:   noopApplicationAPI{},
		Collaboration: noopCollaborationService{},
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

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	var redisClient *redis.Client
	if cfg.Redis.Addr != "" {
		redisClient = redis.NewClient(redisOptions(cfg))
		defer redisClient.Close()
	}
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

	jobRepo := repository.NewJobRepo(db)
	userRepo := repository.NewUserRepo(db)
	profileRepo := repository.NewProfileRepo(db)
	resumeRepo := repository.NewResumeRepo(db)
	applicationRepo := repository.NewApplicationRepo(db)
	interviewRepo := repository.NewInterviewRepo(db)
	offerRepo := repository.NewOfferRepo(db)
	notificationRepo := repository.NewNotificationRepo(db)
	authzRepo := repository.NewAuthzRepo(db)
	collaborationRepo := repository.NewCollaborationRepo(db)
	scopeEval := service.NewScopeEvaluator(authzRepo)
	serviceAuth := service.NewServiceAuthorizer(authzRepo, scopeEval)
	outboxPublisher := service.NewOutboxPublisher(repository.NewOutboxRepo(db), nil)
	taxonomySvc := service.NewJobTaxonomyService(repository.NewDepartmentRepo(db), repository.NewJobLocationRepo(db), jobRepo, repository.NewDepartmentLocationRepo(db))
	jobSvc := service.NewJobService(jobRepo, nil, authzRepo, taxonomySvc, scopeEval)
	candidateSvc := service.NewCandidateService(profileRepo, resumeRepo, ossClient, outboxPublisher, repository.NewUsageLogRepo(db), serviceAuth)
	applicationSvc := service.NewApplicationService(authzRepo, applicationRepo, profileRepo, resumeRepo, jobRepo, interviewRepo, notificationRepo, outboxPublisher, ossClient, nil, scopeEval)
	adminSvc := service.NewAdminService(repository.NewInviteCodeRepo(db), repository.NewUsageLogRepo(db), userRepo, authzRepo, redisClient, serviceAuth)
	usageStatsSvc := service.NewUsageStatsService(repository.NewUsageStatsRepo(db), serviceAuth)
	collaborationSvc := service.NewCollaborationService(
		authzRepo,
		collaborationRepo,
		applicationRepo,
		profileRepo,
		jobRepo,
		userRepo,
		interviewRepo,
		offerRepo,
		resumeRepo,
		ossClient,
		serviceAuth,
		scopeEval,
	)
	runtime, err := recruitmentruntime.New(recruitmentruntime.Deps{
		Job:           jobSvc,
		JobTaxonomy:   taxonomySvc,
		TaxonomyAdmin: taxonomySvc,
		Admin:         adminSvc,
		UsageStats:    usageStatsSvc,
		Candidate:     candidateSvc,
		Application:   applicationSvc,
		Collaboration: collaborationSvc,
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
	fmt.Fprintf(os.Stdout, "recruitment grpc server listening addr=%s service=%s\n", listener.Addr().String(), instance.ServiceName)
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
	for _, candidate := range []string{filepath.Join("smart-recruit-domain-go", "config", "config.yaml"), filepath.Join("..", "smart-recruit-domain-go", "config", "config.yaml")} {
		if _, err := os.Stat(candidate); err == nil {
			return os.Setenv("CONFIG_PATH", candidate)
		}
	}
	return nil
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

type noopCollaborationService struct {
	pb.UnimplementedCollaborationServiceServer
}
