package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"web-gin-service/config"
	_ "web-gin-service/docs"
	"web-gin-service/pkg/logger"
	"web-gin-service/pkg/redisclient"
	"web-gin-service/router"
	"web-gin-service/rpc"
)

// @title           智能招聘系统 API
// @version         1.0
// @description     智能招聘平台后端接口文档，包含候选人端和 HR 管理端接口。
func main() {
	logger.Set(logger.NewConsole())
	log := logger.L()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config validation failed", zap.Error(err))
	}

	// TASK-FU-009：与 logic-grpc-service 同步加载 ranking 段。
	// web-gin 当前不直接使用 cfg.Ranking；保留是为未来扩展做准备。
	// 输出结构化日志便于运维 / 排查时确认两端配置一致。
	log.Info("[ranking] config loaded (mirror of logic-grpc-service/config.Ranking)",
		zap.Float64("weight_vector", cfg.Ranking.WeightVector),
		zap.Float64("weight_lexical", cfg.Ranking.WeightLexical),
		zap.Float64("weight_metadata", cfg.Ranking.WeightMetadata),
		zap.Float64("business_boost_max", cfg.Ranking.BusinessBoostMax),
		zap.Float64("priority_norm", cfg.Ranking.PriorityNorm),
		zap.Float64("boost_alpha", cfg.Ranking.BoostAlpha),
		zap.Float64("boost_beta", cfg.Ranking.BoostBeta),
		zap.Float64("boost_gamma", cfg.Ranking.BoostGamma),
		zap.Float64("relevance_gate", cfg.Ranking.RelevanceGate),
		zap.Float64("gap_high", cfg.Ranking.GapHigh),
		zap.Float64("gap_medium", cfg.Ranking.GapMedium),
	)

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
	})
	if err != nil {
		log.Fatal("connect logic grpc service failed", zap.String("addr", cfg.GRPCAddr), zap.Error(err))
	}
	defer clients.Close()
	log.Info("grpc clients initialized",
		zap.String("logic_addr", cfg.GRPCAddr),
		zap.String("notification_route_mode", clients.NotificationRouteMode),
		zap.String("notification_target_addr", clients.NotificationTargetAddr),
		zap.String("ai_agent_route_mode", clients.AIAgentRouteMode),
		zap.String("ai_agent_target_addr", clients.AIAgentTargetAddr),
		zap.String("identity_route_mode", clients.IdentityRouteMode),
		zap.String("identity_target_addr", clients.IdentityTargetAddr),
		zap.String("recruitment_route_mode", clients.RecruitmentRouteMode),
		zap.String("recruitment_target_addr", clients.RecruitmentTargetAddr),
		zap.String("interview_route_mode", clients.InterviewRouteMode),
		zap.String("interview_target_addr", clients.InterviewTargetAddr),
		zap.String("offer_route_mode", clients.OfferRouteMode),
		zap.String("offer_target_addr", clients.OfferTargetAddr),
	)

	rdb := redisclient.New(cfg.Redis)
	if err := redisclient.Ping(context.Background(), rdb); err != nil {
		log.Warn("connect redis failed, rate limiting will fall back to in-process memory", zap.String("addr", cfg.Redis.Addr), zap.Error(err))
	} else {
		log.Info("redis connected", zap.String("addr", cfg.Redis.Addr))
	}

	r, limiters := router.Setup(cfg, clients, rdb)
	httpServer := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // SSE 长连接禁用写超时，由应用层心跳管理
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	go func() {
		log.Info("web gin service starting", zap.String("port", cfg.HTTPPort))
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
	// Close in-memory rate limiter goroutines.
	if limiters != nil {
		limiters.Close()
	}
	// Close Redis client if connected.
	if rdb != nil {
		rdb.Close()
	}
	log.Info("web gin service stopped")
}
