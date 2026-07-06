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
	t.Setenv("AGENT_FEATURE_PLANNER", "false")
	t.Setenv("AGENT_FEATURE_CANDIDATE_MATCH_SEMANTIC", "false")
	t.Setenv("AGENT_FEATURE_CANDIDATE_MATCH_SHADOW", "true")
	t.Setenv("AGENT_FEATURE_SEMANTIC_RETRIEVAL", "false")
	t.Setenv("AGENT_RESUME_PARSE_TIMEOUT", "7s")
	t.Setenv("AGENT_CANDIDATE_MATCH_TIMEOUT", "8s")
	t.Setenv("AGENT_SEMANTIC_RETRIEVAL_TIMEOUT", "9s")

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
	if cfg.Agent.Features.Planner == nil || *cfg.Agent.Features.Planner {
		t.Fatalf("expected planner feature override false, got %v", cfg.Agent.Features.Planner)
	}
	if cfg.Agent.Features.CandidateMatchSemantic == nil || *cfg.Agent.Features.CandidateMatchSemantic {
		t.Fatalf("expected candidate match semantic feature override false, got %v", cfg.Agent.Features.CandidateMatchSemantic)
	}
	if cfg.Agent.Features.CandidateMatchShadow == nil || !*cfg.Agent.Features.CandidateMatchShadow {
		t.Fatalf("expected candidate match shadow feature override true, got %v", cfg.Agent.Features.CandidateMatchShadow)
	}
	if cfg.Agent.Features.SemanticRetrieval == nil || *cfg.Agent.Features.SemanticRetrieval {
		t.Fatalf("expected semantic retrieval feature override false, got %v", cfg.Agent.Features.SemanticRetrieval)
	}
	if cfg.Agent.Features.ResumeParseTimeout.Duration != 7*time.Second ||
		cfg.Agent.Features.CandidateMatchTimeout.Duration != 8*time.Second ||
		cfg.Agent.Features.SemanticRetrievalTimeout.Duration != 9*time.Second {
		t.Fatalf("unexpected agent feature timeouts: parse=%s match=%s semantic=%s",
			cfg.Agent.Features.ResumeParseTimeout.Duration,
			cfg.Agent.Features.CandidateMatchTimeout.Duration,
			cfg.Agent.Features.SemanticRetrievalTimeout.Duration)
	}
}

func TestOSSProviderEnvOverride(t *testing.T) {
	t.Setenv("OSS_PROVIDER", "aliyun_oss")
	t.Setenv("OSS_ENDPOINT", "oss-cn-shanghai.aliyuncs.com")

	var cfg Config
	applyEnvOverrides(&cfg)

	if cfg.OSS.Provider != "aliyun_oss" {
		t.Fatalf("expected oss provider aliyun_oss, got %q", cfg.OSS.Provider)
	}
	if cfg.OSS.Endpoint != "oss-cn-shanghai.aliyuncs.com" {
		t.Fatalf("expected oss endpoint oss-cn-shanghai.aliyuncs.com, got %q", cfg.OSS.Endpoint)
	}
}
