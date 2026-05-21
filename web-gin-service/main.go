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
	logger.Set(logger.New("info"))
	log := logger.L()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config validation failed", zap.Error(err))
	}

	clients, err := rpc.NewClients(cfg.GRPCAddr)
	if err != nil {
		log.Fatal("connect logic grpc service failed", zap.String("addr", cfg.GRPCAddr), zap.Error(err))
	}
	defer clients.Close()

	rdb := redisclient.New(cfg.Redis)
	if err := redisclient.Ping(context.Background(), rdb); err != nil {
		log.Fatal("connect redis failed", zap.String("addr", cfg.Redis.Addr), zap.Error(err))
	}
	defer rdb.Close()

	r := router.Setup(cfg, clients, rdb)
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
		log.Fatal("http shutdown failed", zap.Error(err))
	}
	log.Info("web gin service stopped")
}
