package server

import (
	"context"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/pkg/observability"
)

func StartMetricsServer(addr string) (*http.Server, error) {
	if addr == "" {
		return nil, nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(observability.DefaultMetrics.Prometheus()))
	})
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() {
		logger.L().Info("metrics server listening", zap.String("addr", addr))
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.L().Error("metrics server stopped unexpectedly", zap.Error(err))
		}
	}()
	return srv, nil
}

func ShutdownMetricsServer(ctx context.Context, srv *http.Server) {
	if srv == nil {
		return
	}
	if err := srv.Shutdown(ctx); err != nil {
		logger.L().Warn("metrics server shutdown failed", zap.Error(err))
	}
}
