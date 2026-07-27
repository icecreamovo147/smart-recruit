package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"smart-recruit-gateway/config"
	_ "smart-recruit-gateway/docs"
	"smart-recruit-gateway/internal/runtime"
	"smart-recruit-gateway/pkg/logger"
	"smart-recruit-gateway/pkg/redisclient"
	"smart-recruit-gateway/router"
	"smart-recruit-gateway/rpc"
	"smart-recruit-platform-go/businessclock"
	"smart-recruit-platform-go/i18n"
)

func main() {
	businessclock.Configure()
	logger.Set(logger.NewConsole())
	log := logger.L()
	if err := i18n.ConfigureFromEnv(); err != nil {
		log.Fatal("log.gateway.config_failed", zap.String("cause", err.Error()))
	}

	if _, err := runtime.PlatformBootstrap(); err != nil {
		log.Fatal("log.gateway.bootstrap_failed", zap.String("cause", err.Error()))
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("log.gateway.config_failed", zap.String("cause", err.Error()))
	}

	routeTable, err := runtime.NewRouteTable(map[string]string{
		"identity":     cfg.IdentityRouteMode,
		"recruitment":  cfg.RecruitmentRouteMode,
		"interview":    cfg.InterviewRouteMode,
		"offer":        cfg.OfferRouteMode,
		"notification": cfg.NotificationRouteMode,
		"ai-agent":     cfg.AIAgentRouteMode,
		"analytics":    envOrDefault("ANALYTICS_ROUTE_MODE", "analytics"),
		"billing":      "billing",
	})
	if err != nil {
		log.Fatal("log.gateway.route_validation_failed", zap.String("cause", err.Error()))
	}
	resolvedRoutes, err := routeTable.ResolveTargets(context.Background(), cfg.GRPCAddr, runtime.StaticTargetResolver{
		"identity":     cfg.IdentityGRPCAddr,
		"recruitment":  cfg.RecruitmentGRPCAddr,
		"interview":    cfg.InterviewGRPCAddr,
		"offer":        cfg.OfferGRPCAddr,
		"notification": cfg.NotificationGRPCAddr,
		"ai-agent":     cfg.AIAgentGRPCAddr,
		"analytics":    envOrDefault("ANALYTICS_GRPC_ADDR", "127.0.0.1:50067"),
		"billing":      envOrDefault("BILLING_GRPC_ADDR", "127.0.0.1:50069"),
	})
	if err != nil {
		log.Fatal("log.gateway.route_validation_failed", zap.String("cause", err.Error()))
	}
	log.Info("log.gateway.route_targets_validated", zap.Int("ready_targets", len(resolvedRoutes.ReadyTargets(cfg.GRPCAddr))))

	clients, err := rpc.NewClientsWithOptions(cfg.GRPCAddr, rpc.ClientOptions{
		NotificationAddr:      cfg.NotificationGRPCAddr,
		NotificationRouteMode: cfg.NotificationRouteMode,
		AIAgentAddr:           cfg.AIAgentGRPCAddr,
		AIAgentRouteMode:      cfg.AIAgentRouteMode,
		IdentityAddr:          cfg.IdentityGRPCAddr,
		IdentityRouteMode:     cfg.IdentityRouteMode,
		RecruitmentAddr:       cfg.RecruitmentGRPCAddr,
		RecruitmentRouteMode:  cfg.RecruitmentRouteMode,
		InterviewAddr:         cfg.InterviewGRPCAddr,
		InterviewRouteMode:    cfg.InterviewRouteMode,
		OfferAddr:             cfg.OfferGRPCAddr,
		OfferRouteMode:        cfg.OfferRouteMode,
		AnalyticsAddr:         envOrDefault("ANALYTICS_GRPC_ADDR", "127.0.0.1:50067"),
		AnalyticsRouteMode:    envOrDefault("ANALYTICS_ROUTE_MODE", "analytics"),
		BillingAddr:           envOrDefault("BILLING_GRPC_ADDR", "127.0.0.1:50069"),
		GRPCInternalTLS:       cfg.GRPCInternalTLS,
		GRPCTLSCAFile:         cfg.GRPCTLSCAFile,
		GRPCTLSServerName:     cfg.GRPCTLSServerName,
	})
	if err != nil {
		log.Fatal("log.gateway.grpc_connect_failed", zap.String("addr", cfg.GRPCAddr), zap.String("cause", err.Error()))
	}
	defer clients.Close()

	rdb := redisclient.New(cfg.Redis)
	if err := redisclient.Ping(context.Background(), rdb); err != nil {
		log.Warn("log.gateway.redis_fallback", zap.String("addr", cfg.Redis.Addr), zap.String("cause", err.Error()))
	}

	r, limiters := router.Setup(cfg, clients, rdb)
	httpServer := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Info("log.gateway.starting", zap.String("port", cfg.HTTPPort), zap.String("locale", string(cfg.AppLocale)))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("log.gateway.http_serve_failed", zap.String("cause", err.Error()))
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Info("log.gateway.shutdown_received", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Warn("log.gateway.shutdown_failed", zap.String("cause", err.Error()))
	}
	if limiters != nil {
		limiters.Close()
	}
	if rdb != nil {
		_ = rdb.Close()
	}
	log.Info("log.gateway.stopped")
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
