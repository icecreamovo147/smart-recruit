package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"logic-grpc-service/mq"
	"logic-grpc-service/pkg/logger"
)

const (
	ReadinessModeService = "service"
	ReadinessModeWorker  = "worker"

	DependencyStatusOK       = "ok"
	DependencyStatusSkipped  = "skipped"
	DependencyStatusDegraded = "degraded"
	DependencyStatusFailed   = "failed"
)

type sqlPinger interface {
	PingContext(context.Context) error
}

type redisPinger interface {
	Ping(context.Context) *redis.StatusCmd
}

type mqState interface {
	IsClosed() bool
	IsReady() bool
}

type DependencyReport struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Severity string `json:"severity"`
	Message  string `json:"message,omitempty"`
}

type ReadinessReport struct {
	Status       string             `json:"status"`
	Mode         string             `json:"mode"`
	Dependencies []DependencyReport `json:"dependencies"`
}

type ReadinessOptions struct {
	Mode  string
	DB    sqlPinger
	Redis redisPinger
	MQ    mqState
}

type HealthServer struct {
	healthpb.UnimplementedHealthServer
	db          *sql.DB
	redisClient *redis.Client
	mqConn      *mq.Conn
}

func NewHealthServer(db *sql.DB, redisClient *redis.Client, mqConn *mq.Conn) *HealthServer {
	return &HealthServer{db: db, redisClient: redisClient, mqConn: mqConn}
}

func (s *HealthServer) Check(ctx context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	report := EvaluateReadiness(ctx, ReadinessOptions{
		Mode:  ReadinessModeService,
		DB:    s.db,
		Redis: s.redisClient,
		MQ:    s.mqConn,
	})
	if report.Status == "not_ready" {
		return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_NOT_SERVING}, nil
	}
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func EvaluateReadiness(ctx context.Context, options ReadinessOptions) ReadinessReport {
	mode := options.Mode
	if mode == "" {
		mode = ReadinessModeService
	}
	report := ReadinessReport{
		Status:       "ready",
		Mode:         mode,
		Dependencies: []DependencyReport{},
	}
	report.add(checkMySQL(ctx, options.DB))
	report.add(checkRedis(ctx, options.Redis))
	report.add(checkRabbitMQ(options.MQ, mode))
	return report
}

func (r *ReadinessReport) add(dep DependencyReport) {
	r.Dependencies = append(r.Dependencies, dep)
	if dep.Status == DependencyStatusFailed && dep.Severity == "hard" {
		r.Status = "not_ready"
		return
	}
	if dep.Status == DependencyStatusDegraded && r.Status == "ready" {
		r.Status = "degraded"
	}
}

func checkMySQL(ctx context.Context, db sqlPinger) DependencyReport {
	if db == nil {
		return DependencyReport{Name: "mysql", Status: DependencyStatusFailed, Severity: "hard", Message: "not configured"}
	}
	if err := db.PingContext(ctx); err != nil {
		return DependencyReport{Name: "mysql", Status: DependencyStatusFailed, Severity: "hard", Message: err.Error()}
	}
	return DependencyReport{Name: "mysql", Status: DependencyStatusOK, Severity: "hard"}
}

func checkRedis(ctx context.Context, rdb redisPinger) DependencyReport {
	if rdb == nil {
		return DependencyReport{Name: "redis", Status: DependencyStatusSkipped, Severity: "optional", Message: "not configured"}
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		return DependencyReport{Name: "redis", Status: DependencyStatusFailed, Severity: "hard", Message: err.Error()}
	}
	return DependencyReport{Name: "redis", Status: DependencyStatusOK, Severity: "hard"}
}

func checkRabbitMQ(mqConn mqState, mode string) DependencyReport {
	severity := "soft"
	if mode == ReadinessModeWorker {
		severity = "hard"
	}
	if mqConn == nil {
		status := DependencyStatusDegraded
		if severity == "hard" {
			status = DependencyStatusFailed
		}
		return DependencyReport{Name: "rabbitmq", Status: status, Severity: severity, Message: "not connected"}
	}
	if mqConn.IsClosed() {
		status := DependencyStatusDegraded
		if severity == "hard" {
			status = DependencyStatusFailed
		}
		return DependencyReport{Name: "rabbitmq", Status: status, Severity: severity, Message: "connection closed"}
	}
	if !mqConn.IsReady() {
		status := DependencyStatusDegraded
		if severity == "hard" {
			status = DependencyStatusFailed
		}
		return DependencyReport{Name: "rabbitmq", Status: status, Severity: severity, Message: "reconnecting"}
	}
	return DependencyReport{Name: "rabbitmq", Status: DependencyStatusOK, Severity: severity}
}

func StartWorkerHealthServer(addr string, options ReadinessOptions) (*http.Server, error) {
	if addr == "" {
		return nil, nil
	}
	options.Mode = ReadinessModeWorker
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, _ *http.Request) {
		writeHealthJSON(w, http.StatusOK, map[string]string{"status": "ok", "mode": ReadinessModeWorker})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		report := EvaluateReadiness(ctx, options)
		statusCode := http.StatusOK
		if report.Status == "not_ready" {
			statusCode = http.StatusServiceUnavailable
		}
		writeHealthJSON(w, statusCode, report)
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
		logger.L().Info("worker health server listening", zap.String("addr", addr))
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.L().Error("worker health server stopped unexpectedly", zap.Error(err))
		}
	}()
	return srv, nil
}

func ShutdownHealthServer(ctx context.Context, srv *http.Server) {
	if srv == nil {
		return
	}
	if err := srv.Shutdown(ctx); err != nil {
		logger.L().Warn("health server shutdown failed", zap.Error(err))
	}
}

func writeHealthJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
