package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"smart-recruit-commons/mq"
	"smart-recruit-commons/oss"
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
	workeroutbox "smart-recruit-worker-service/internal/outbox"
	workerresumeparse "smart-recruit-worker-service/internal/resumeparse"
	workerruntime "smart-recruit-worker-service/internal/runtime"
)

const nacosServiceName = "worker"

func main() {
	if err := i18n.ConfigureFromEnv(); err != nil {
		logger.L().Error("log.service.config_failed", zap.String("cause", err.Error()))
		os.Exit(2)
	}
	businessclock.Configure()
	check := flag.Bool("check", false, "validate Worker service runtime wiring and exit")
	serve := flag.Bool("serve", false, "start Worker runtime")
	healthAddr := flag.String("health-addr", envOrDefault("WORKER_HEALTH_ADDR", ":50068"), "Worker health/readiness listen address")
	flag.Parse()

	if *check {
		if err := checkRuntime(); err != nil {
			logger.L().Error("log.service.check_failed", zap.String("service", "worker-service"), zap.String("cause", err.Error()))
			os.Exit(1)
		}
		logger.L().Info("log.service.check_passed", zap.String("service", "worker-service"))
		return
	}
	if *serve {
		if err := serveWorker(*healthAddr); err != nil {
			logger.L().Error("log.service.serve_failed", zap.String("service", "worker-service"), zap.String("cause", err.Error()))
			os.Exit(1)
		}
		return
	}
	logger.L().Error("log.service.arguments_required", zap.String("service", "worker-service"))
	os.Exit(2)
}

func checkRuntime() error {
	cfg, err := workerruntime.ParseWorkloadConfig("", "")
	if err != nil {
		return err
	}
	runtime, err := workerruntime.New(workerruntime.Deps{
		Config: cfg,
		Starters: controlledStarters(cfg.Enabled, starterDeps{
			OutboxStore:     checkOutboxStore{},
			OutboxPublisher: checkOutboxPublisher{},
		}),
		Status: func(context.Context) workerruntime.DependencyStatus {
			return workerruntime.DependencyStatus{RabbitMQ: true, MySQL: true}
		},
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := runtime.Start(ctx); err != nil {
		return err
	}
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	return runtime.Stop(stopCtx)
}

func serveWorker(healthAddr string) error {
	if err := ensureLogicConfigPath(); err != nil {
		return err
	}
	bootstrap, err := loadBootstrap(healthAddr)
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
	if err := logger.Init(cfg.Logging); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	log := logger.L()
	logicobservability.DefaultMetrics = logicobservability.NewRegistry(workerruntime.ServiceName)

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

	mqConn, err := mq.New(mqConfig(cfg))
	if err != nil {
		return fmt.Errorf("connect rabbitmq: %w", err)
	}
	defer mqConn.Close()

	workloadCfg, err := workerruntime.ParseWorkloadConfig(os.Getenv("WORKER_WORKLOADS"), os.Getenv("WORKER_DISABLED_WORKLOADS"))
	if err != nil {
		return err
	}
	runtime, err := workerruntime.New(workerruntime.Deps{
		Config: workloadCfg,
		Starters: controlledStarters(workloadCfg.Enabled, starterDeps{
			DB:  db,
			MQ:  mqConn,
			Log: log,
			OSS: oss.Config{
				Provider:        cfg.OSS.Provider,
				Endpoint:        cfg.OSS.Endpoint,
				AccessKeyID:     cfg.OSS.AccessKeyID,
				AccessKeySecret: cfg.OSS.AccessKeySecret,
				BucketName:      cfg.OSS.BucketName,
				PublicBaseURL:   cfg.OSS.PublicBaseURL,
			},
		}),
		Status: dependencyStatus(sqlDB, mqConn),
	})
	if err != nil {
		return err
	}

	metricsServer, err := server.StartMetricsServer(cfg.Observability.MetricsAddr)
	if err != nil {
		return fmt.Errorf("start metrics server: %w", err)
	}
	defer server.ShutdownMetricsServer(context.Background(), metricsServer)

	listener, err := net.Listen("tcp", healthAddr)
	if err != nil {
		return fmt.Errorf("listen health %s: %w", healthAddr, err)
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

	ctx, cancel := signalContext()
	defer cancel()
	if err := runtime.Start(ctx); err != nil {
		_ = listener.Close()
		return err
	}

	healthServer := &http.Server{Handler: healthMux(runtime)}
	go func() {
		if err := healthServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("log.worker.health_failed", zap.String("cause", err.Error()))
			cancel()
		}
	}()

	log.Info("log.service.started",
		zap.String("health_addr", listener.Addr().String()),
		zap.String("nacos_service", instance.ServiceName),
		zap.Strings("workloads", runtime.StartedWorkloads()),
	)
	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	return healthServer.Shutdown(shutdownCtx)
}

type starterDeps struct {
	DB              *gorm.DB
	MQ              *mq.Conn
	Log             *zap.Logger
	OSS             oss.Config
	OutboxStore     workeroutbox.Store
	OutboxPublisher workeroutbox.Publisher
}

func controlledStarters(names []string, deps starterDeps) map[string]workerruntime.Starter {
	starters := make(map[string]workerruntime.Starter, len(names))
	for _, name := range names {
		workloadName := name
		switch workloadName {
		case "outbox-dispatcher":
			starters[workloadName] = outboxDispatcherStarter(deps)
		case "resume-parse-consumer":
			starters[workloadName] = resumeParseConsumerStarter(deps)
		default:
			starters[workloadName] = unsupportedWorkloadStarter(workloadName)
		}
	}
	return starters
}

func outboxDispatcherStarter(deps starterDeps) workerruntime.Starter {
	store := deps.OutboxStore
	if store == nil && deps.DB != nil {
		store = workeroutbox.NewGormStore(deps.DB)
	}
	publisher := deps.OutboxPublisher
	if publisher == nil && deps.MQ != nil {
		publisher = workeroutbox.NewMQPublisher(deps.MQ)
	}
	return workeroutbox.NewDispatcher(store, publisher, workeroutbox.Options{
		Logger: deps.Log,
	})
}

func resumeParseConsumerStarter(deps starterDeps) workerruntime.Starter {
	return workerruntime.StarterFunc(func(ctx context.Context) error {
		if deps.DB == nil {
			return fmt.Errorf("resume-parse-consumer requires mysql")
		}
		if deps.MQ == nil {
			return fmt.Errorf("resume-parse-consumer requires rabbitmq")
		}
		storage, err := oss.NewStorage(deps.OSS)
		if err != nil {
			return fmt.Errorf("resume-parse-consumer oss: %w", err)
		}
		consumer := workerresumeparse.NewConsumer(
			deps.MQ,
			workerresumeparse.NewStore(deps.DB),
			storage,
			workerresumeparse.Options{Logger: deps.Log},
		)
		return consumer.Start(ctx)
	})
}

func unsupportedWorkloadStarter(name string) workerruntime.Starter {
	return workerruntime.StarterFunc(func(context.Context) error {
		switch name {
		case "notification-consumer", "email-consumer":
			return fmt.Errorf("worker workload %q is not started by worker-service; notification-service owns this consumer to avoid duplicate consumption", name)
		case "embedding-consumer", "agent-run-consumer":
			return fmt.Errorf("worker workload %q is not implemented in worker-service yet; keep it disabled until its real consumer is wired", name)
		default:
			return fmt.Errorf("worker workload %q is not implemented in worker-service yet; keep it disabled", name)
		}
	})
}

type checkOutboxStore struct{}

func (checkOutboxStore) ClaimPending(context.Context, int, string, time.Duration) ([]workeroutbox.Event, error) {
	return nil, nil
}

func (checkOutboxStore) MarkPublished(context.Context, uint64) error { return nil }

func (checkOutboxStore) MarkRetryableFailure(context.Context, uint64, string, time.Time) error {
	return nil
}

func (checkOutboxStore) MarkDead(context.Context, uint64, string) error { return nil }

type checkOutboxPublisher struct{}

func (checkOutboxPublisher) Publish(context.Context, string, []byte) error { return nil }

func dependencyStatus(sqlDB *sql.DB, mqConn *mq.Conn) func(context.Context) workerruntime.DependencyStatus {
	return func(ctx context.Context) workerruntime.DependencyStatus {
		status := workerruntime.DependencyStatus{RabbitMQ: mqConn != nil}
		if sqlDB != nil && sqlDB.PingContext(ctx) == nil {
			status.MySQL = true
		}
		return status
	}
}

func healthMux(runtime *workerruntime.Runtime) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := runtime.Ready(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	return mux
}

func loadBootstrap(healthAddr string) (platformconfig.Bootstrap, error) {
	return platformconfig.LoadWithLookup(func(key string) string {
		switch key {
		case "SERVICE_NAME":
			return envOrDefault(key, workerruntime.ServiceName)
		case "SERVICE_ENV":
			return envOrDefault(key, "local")
		case "SERVICE_VERSION":
			return envOrDefault(key, "dev")
		case "GRPC_ADDR":
			return envOrDefault(key, healthAddr)
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
			"worker-service.yaml": "service:\n  name: worker-service\nworker:\n  workloads: all\n",
		},
		AllowFallback: bootstrap.StaticFallback,
		Username:      bootstrap.NacosUsername,
		Password:      bootstrap.NacosPassword,
	})
	if err != nil {
		return nil, fmt.Errorf("init nacos config: %w", err)
	}
	if _, err := configProvider.Load(ctx, "worker-service.yaml"); err != nil {
		return nil, fmt.Errorf("load worker nacos config: %w", err)
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
		return nil, fmt.Errorf("register worker in nacos: %w", err)
	}
	return discovery, nil
}

func instanceFromAddr(addr string, bootstrap platformconfig.Bootstrap) (nacos.Instance, error) {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return nacos.Instance{}, fmt.Errorf("parse worker addr %q: %w", addr, err)
	}
	if host == "" || host == "::" {
		host = "127.0.0.1"
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return nacos.Instance{}, fmt.Errorf("parse worker port %q: %w", portText, err)
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

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-shutdown:
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(shutdown)
	}()
	return ctx, cancel
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

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
