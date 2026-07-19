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

	platformconfig "smart-recruit-platform-go/config"
	"smart-recruit-platform-go/nacos"
	platformobs "smart-recruit-platform-go/observability"

	appservice "smart-recruit-identity-service/internal/application/service"
	identitycache "smart-recruit-identity-service/internal/infrastructure/cache"
	identityclient "smart-recruit-identity-service/internal/infrastructure/client"
	identitypersistence "smart-recruit-identity-service/internal/infrastructure/persistence"
	identitygrpc "smart-recruit-identity-service/internal/interfaces/grpc"
	identityruntime "smart-recruit-identity-service/internal/runtime"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/logger"
	logicobservability "smart-recruit-platform-go/observability"
	"smart-recruit-platform-go/server"
	logicconfig "smart-recruit-platform-go/serviceconfig"
	"smart-recruit-proto/recruitment/pb"
)

const nacosServiceName = "identity"

func main() {
	check := flag.Bool("check", false, "validate the Identity service runtime wiring and exit")
	serve := flag.Bool("serve", false, "start the Identity gRPC API runtime")
	addr := flag.String("addr", envOrDefault("GRPC_ADDR", ":50061"), "Identity gRPC listen address")
	flag.Parse()

	if *check {
		if err := checkRuntime(); err != nil {
			fmt.Fprintf(os.Stderr, "identity-service check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "identity-service runtime check passed")
		return
	}
	if *serve {
		if err := serveIdentity(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "identity-service failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Fprintln(os.Stderr, "identity-service requires --check or --serve")
	os.Exit(2)
}

func checkRuntime() error {
	runtime, err := identityruntime.New(identityruntime.Deps{
		Auth:   noopAuthAPI{},
		Admin:  noopAdminAPI{},
		Audit:  noopAuditAPI{},
		Tenant: noopTenantAPI{},
	})
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	defer server.Stop()
	return runtime.RegisterGRPC(server)
}

func serveIdentity(addr string) error {
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
	logicobservability.DefaultMetrics = logicobservability.NewRegistry(identityruntime.ServiceName)

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

	repos := identitypersistence.NewRepositories(db)
	passwords := identityclient.PasswordService{}
	tokenFactory := identityclient.TokenGenerator{}
	actorVerifier := identityclient.ActorVerifier{}
	adminAuthorizer := identityclient.NewAdminAuthorizer(repos.Authz, repos.Tenants)
	tokenVersionCache := identitycache.NewTokenVersionCache(redisClient)
	authSvc, err := appservice.NewAuthService(appservice.AuthDeps{
		Users:         repos.Users,
		Tokens:        repos.Tokens,
		Authz:         repos.Authz,
		Tenants:       repos.Tenants,
		Invites:       repos.Invites,
		Audit:         repos.Audit,
		Passwords:     passwords,
		TokenFactory:  tokenFactory,
		ActorVerifier: actorVerifier,
	})
	if err != nil {
		return fmt.Errorf("build identity auth service: %w", err)
	}
	adminSvc, err := appservice.NewAdminService(appservice.AdminDeps{
		Users:      repos.Users,
		Authz:      repos.Authz,
		Audit:      repos.Audit,
		Passwords:  passwords,
		Authorizer: adminAuthorizer,
		TokenCache: tokenVersionCache,
		Tenants:    repos.Tenants,
	})
	if err != nil {
		return fmt.Errorf("build identity admin service: %w", err)
	}
	tenantSvc := appservice.NewTenantService(repos.Tenants, repos.Authz)
	localIdentity := identitygrpc.NewServer(authSvc, adminSvc, tenantSvc)
	runtime, err := identityruntime.New(identityruntime.Deps{
		Auth:   localIdentity,
		Admin:  localIdentity,
		Audit:  localIdentity,
		Tenant: localIdentity,
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
		zap.String("addr", listener.Addr().String()),
		zap.String("nacos_service", instance.ServiceName),
		zap.String("env", bootstrap.ServiceEnv),
	)
	if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return fmt.Errorf("grpc serve: %w", err)
	}
	return nil
}

func loadBootstrap(addr string) (platformconfig.Bootstrap, error) {
	return platformconfig.LoadWithLookup(func(key string) string {
		switch key {
		case "SERVICE_NAME":
			return envOrDefault(key, identityruntime.ServiceName)
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
			"identity-service.yaml": "service:\n  name: identity-service\n",
		},
		AllowFallback: bootstrap.StaticFallback,
		Username:      bootstrap.NacosUsername,
		Password:      bootstrap.NacosPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("init nacos config: %w", err)
	}
	if _, err := configProvider.Load(ctx, "identity-service.yaml"); err != nil {
		return nil, fmt.Errorf("load identity nacos config: %w", err)
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
		return nil, fmt.Errorf("register identity in nacos: %w", err)
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

func ensureLogicConfigPath() error {
	if os.Getenv("CONFIG_PATH") != "" {
		return nil
	}
	for _, candidate := range []string{
		filepath.Join("smart-recruit-commons", "config", "config.yaml"),
		filepath.Join("..", "smart-recruit-commons", "config", "config.yaml"),
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

type noopAuthAPI struct{}

func (noopAuthAPI) Register(context.Context, *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{Code: errs.OK}, nil
}

func (noopAuthAPI) Login(context.Context, *pb.LoginRequest) (*pb.LoginResponse, error) {
	return &pb.LoginResponse{Code: errs.OK}, nil
}

func (noopAuthAPI) RefreshToken(context.Context, *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return &pb.RefreshTokenResponse{Code: errs.OK}, nil
}

func (noopAuthAPI) SwitchTenant(context.Context, *pb.SwitchTenantRequest) (*pb.LoginResponse, error) {
	return &pb.LoginResponse{Code: errs.OK}, nil
}

func (noopAuthAPI) RevokeRefreshToken(context.Context, *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (noopAuthAPI) RecordAuthDecision(context.Context, *pb.AuthAuditRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (noopAuthAPI) GetPrincipal(context.Context, *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	return &pb.GetPrincipalResponse{Code: errs.OK}, nil
}

func (noopAuthAPI) AuthorizeInternal(context.Context, *pb.AuthorizeInternalRequest) (*pb.AuthorizeInternalResponse, error) {
	return &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}, nil
}

func (noopAuthAPI) UpdateEmail(context.Context, *pb.UpdateEmailRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

type noopAdminAPI struct{}

func (noopAdminAPI) ListRoles(context.Context, *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	return &pb.ListRolesResponse{Code: errs.OK}, nil
}

func (noopAdminAPI) ListPermissions(context.Context, *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	return &pb.ListPermissionsResponse{Code: errs.OK}, nil
}

func (noopAdminAPI) GetUserRoles(context.Context, *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return &pb.GetUserRolesResponse{Code: errs.OK}, nil
}

func (noopAdminAPI) AssignUserRole(context.Context, *pb.AssignUserRoleRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (noopAdminAPI) RevokeUserRole(context.Context, *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (noopAdminAPI) AssignDataScope(context.Context, *pb.AssignDataScopeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (noopAdminAPI) RevokeDataScope(context.Context, *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (noopAdminAPI) ListStaffUsers(context.Context, *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error) {
	return &pb.ListStaffUsersResponse{Code: errs.OK}, nil
}

func (noopAdminAPI) CreateStaffUser(context.Context, *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error) {
	return &pb.CreateStaffUserResponse{Code: errs.OK}, nil
}

type noopAuditAPI struct{}

func (noopAuditAPI) QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	return &pb.QueryAuthAuditLogsResponse{Code: errs.OK}, nil
}

type noopTenantAPI struct{}

func (noopTenantAPI) CreateTenant(context.Context, *pb.CreateTenantRequest) (*pb.TenantResponse, error) {
	return &pb.TenantResponse{Code: errs.OK}, nil
}
func (noopTenantAPI) ListTenants(context.Context, *pb.ListTenantsRequest) (*pb.ListTenantsResponse, error) {
	return &pb.ListTenantsResponse{Code: errs.OK}, nil
}
func (noopTenantAPI) UpdateTenantStatus(context.Context, *pb.UpdateTenantStatusRequest) (*pb.TenantResponse, error) {
	return &pb.TenantResponse{Code: errs.OK}, nil
}
func (noopTenantAPI) ListTenantMemberships(context.Context, *pb.ListTenantMembershipsRequest) (*pb.ListTenantMembershipsResponse, error) {
	return &pb.ListTenantMembershipsResponse{Code: errs.OK}, nil
}
