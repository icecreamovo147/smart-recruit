package config

import (
	"testing"
	"time"
)

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv("MYSQL_DSN", "root:pass@tcp(mysql:3306)/recruitment")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("GRPC_PORT", "60051")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("AI_API_KEY", "sk-test")
	t.Setenv("AI_MODEL", "qwen-plus")
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	t.Setenv("RABBITMQ_RETRY_DELAY", "9s")

	var cfg Config
	applyEnvOverrides(&cfg)

	if cfg.MySQL.DSN != "root:pass@tcp(mysql:3306)/recruitment" {
		t.Fatalf("unexpected mysql dsn: %q", cfg.MySQL.DSN)
	}
	if cfg.JWT.Secret != "test-secret" || cfg.GRPC.Port != 60051 {
		t.Fatalf("unexpected jwt/grpc override: secret=%q port=%d", cfg.JWT.Secret, cfg.GRPC.Port)
	}
	if cfg.Redis.Addr != "redis:6379" {
		t.Fatalf("unexpected redis addr: %q", cfg.Redis.Addr)
	}
	if cfg.AI.APIKey != "sk-test" || cfg.AI.Model != "qwen-plus" {
		t.Fatalf("unexpected ai override: key=%q model=%q", cfg.AI.APIKey, cfg.AI.Model)
	}
	if cfg.RabbitMQ.URL != "amqp://guest:guest@rabbitmq:5672/" {
		t.Fatalf("unexpected rabbitmq url: %q", cfg.RabbitMQ.URL)
	}
	if cfg.RabbitMQ.RetryDelay.Duration != 9*time.Second {
		t.Fatalf("unexpected rabbitmq retry delay: %s", cfg.RabbitMQ.RetryDelay.Duration)
	}
}
