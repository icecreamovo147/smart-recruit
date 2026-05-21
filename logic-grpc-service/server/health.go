package server

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"logic-grpc-service/mq"
)

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
	notServing := &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_NOT_SERVING}
	serving := &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}

	// MySQL is a hard dependency
	if s.db == nil {
		return notServing, nil
	}
	if err := s.db.PingContext(ctx); err != nil {
		return notServing, nil
	}

	// Redis is a hard dependency when configured
	if s.redisClient != nil {
		if err := s.redisClient.Ping(ctx).Err(); err != nil {
			return notServing, nil
		}
	}

	// RabbitMQ is a soft dependency — services degrade without it
	// (outbox accumulates, consumers pause), so we don't fail readiness.
	// We only check: if mqConn is present and closed, report NOT_SERVING.
	if s.mqConn != nil && s.mqConn.IsClosed() {
		return notServing, nil
	}

	return serving, nil
}
