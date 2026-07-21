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
)

func main() {
	businessclock.Configure()
	logger.Set(logger.NewConsole())
	log := logger.L()

	if _, err := runtime.PlatformBootstrap(); err != nil {
		log.Fatal("platform bootstrap failed", zap.Error(err))
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config validation failed", zap.Error(err))
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
		log.Fatal("gateway route mode validation failed", zap.Error(err))
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
		log.Fatal("gateway route target validation failed", zap.Error(err))
	}
	log.Info("gateway route targets validated", zap.Int("ready_targets", len(resolvedRoutes.ReadyTargets(cfg.GRPCAddr))))

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
		log.Fatal("connect grpc service failed", zap.String("addr", cfg.GRPCAddr), zap.Error(err))
	}
	defer clients.Close()

	rdb := redisclient.New(cfg.Redis)
	if err := redisclient.Ping(context.Background(), rdb); err != nil {
		log.Warn("connect redis failed, rate limiting will fall back to in-process memory", zap.String("addr", cfg.Redis.Addr), zap.Error(err))
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
		log.Info("smart recruit gateway starting", zap.String("port", cfg.HTTPPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("serve http failed", zap.Error(err))
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Info("received signal, shutting down", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Warn("http shutdown failed", zap.Error(err))
	}
	if limiters != nil {
		limiters.Close()
	}
	if rdb != nil {
		_ = rdb.Close()
	}
	log.Info("smart recruit gateway stopped")
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
