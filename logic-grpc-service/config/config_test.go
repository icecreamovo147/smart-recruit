package config

import (
	"strings"
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
	t.Setenv("RABBITMQ_AGENT_RUN_QUEUE", "recruitment.agent.run.execute.test")
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
	if cfg.RabbitMQ.AgentRunQueue != "recruitment.agent.run.execute.test" {
		t.Fatalf("unexpected rabbitmq agent run queue: %q", cfg.RabbitMQ.AgentRunQueue)
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

func TestEmbeddingConfigDefaults(t *testing.T) {
	var cfg Config
	if cfg.Embedding.RequestTimeout.Duration != 0 {
		t.Fatal("expected zero value before Load")
	}
}

func TestEmbeddingConfigDefaultValues(t *testing.T) {
	var cfg Config
	applyEnvOverrides(&cfg)
	defaultBool(&cfg.Embedding.Enabled, true)
	defaultBool(&cfg.Embedding.FallbackToRuleRetrieval, true)
	if cfg.Embedding.MaxConcurrency <= 0 {
		cfg.Embedding.MaxConcurrency = 8
	}
	if cfg.Embedding.RequestTimeout.Duration <= 0 {
		cfg.Embedding.RequestTimeout.Duration = 30 * time.Second
	}
	if cfg.Embedding.SlowRequestThreshold.Duration <= 0 {
		cfg.Embedding.SlowRequestThreshold.Duration = 2 * time.Second
	}

	if cfg.Embedding.Enabled == nil || !*cfg.Embedding.Enabled {
		t.Fatal("expected enabled=true by default")
	}
	if cfg.Embedding.FallbackToRuleRetrieval == nil || !*cfg.Embedding.FallbackToRuleRetrieval {
		t.Fatal("expected fallback_to_rule_retrieval=true by default")
	}
	if cfg.Embedding.MaxConcurrency != 8 {
		t.Fatalf("expected max_concurrency=8, got %d", cfg.Embedding.MaxConcurrency)
	}
	if cfg.Embedding.RequestTimeout.Duration != 30*time.Second {
		t.Fatalf("expected request_timeout=30s, got %s", cfg.Embedding.RequestTimeout.Duration)
	}
	if cfg.Embedding.SlowRequestThreshold.Duration != 2*time.Second {
		t.Fatalf("expected slow_request_threshold=2s, got %s", cfg.Embedding.SlowRequestThreshold.Duration)
	}
}

func TestEmbeddingConfigEnvOverrides(t *testing.T) {
	t.Setenv("EMBEDDING_ENABLED", "false")
	t.Setenv("EMBEDDING_DEFAULT_MODEL_ID", "42")
	t.Setenv("EMBEDDING_FALLBACK_TO_RULE_RETRIEVAL", "false")
	t.Setenv("EMBEDDING_REQUEST_TIMEOUT", "15s")
	t.Setenv("EMBEDDING_MAX_CONCURRENCY", "16")
	t.Setenv("EMBEDDING_SLOW_REQUEST_THRESHOLD", "3s")

	var cfg Config
	applyEnvOverrides(&cfg)

	if cfg.Embedding.Enabled == nil || *cfg.Embedding.Enabled {
		t.Fatal("expected enabled=false from env")
	}
	if cfg.Embedding.DefaultModelID != 42 {
		t.Fatalf("expected default_model_id=42, got %d", cfg.Embedding.DefaultModelID)
	}
	if cfg.Embedding.FallbackToRuleRetrieval == nil || *cfg.Embedding.FallbackToRuleRetrieval {
		t.Fatal("expected fallback_to_rule_retrieval=false from env")
	}
	if cfg.Embedding.RequestTimeout.Duration != 15*time.Second {
		t.Fatalf("expected request_timeout=15s, got %s", cfg.Embedding.RequestTimeout.Duration)
	}
	if cfg.Embedding.MaxConcurrency != 16 {
		t.Fatalf("expected max_concurrency=16, got %d", cfg.Embedding.MaxConcurrency)
	}
	if cfg.Embedding.SlowRequestThreshold.Duration != 3*time.Second {
		t.Fatalf("expected slow_request_threshold=3s, got %s", cfg.Embedding.SlowRequestThreshold.Duration)
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

func TestValidateProductionSecretsPassesWithExternalSecrets(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("GRPC_INTERNAL_AUTH", "required")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("ENCRYPTION_KEY", strings.Repeat("a", 64))

	cfg := productionReadyConfig()
	if err := validateProductionSecrets(cfg); err != nil {
		t.Fatalf("validateProductionSecrets: %v", err)
	}
}

func TestValidateProductionSecretsRequiresInternalAuth(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("GRPC_INTERNAL_AUTH", "optional")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("ENCRYPTION_KEY", strings.Repeat("a", 64))

	err := validateProductionSecrets(productionReadyConfig())
	if err == nil || !strings.Contains(err.Error(), "GRPC_INTERNAL_AUTH") {
		t.Fatalf("expected GRPC_INTERNAL_AUTH error, got %v", err)
	}
}

func TestValidateProductionSecretsRejectsRabbitMQDefaultCredentials(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("GRPC_INTERNAL_AUTH", "required")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("ENCRYPTION_KEY", strings.Repeat("a", 64))

	cfg := productionReadyConfig()
	cfg.RabbitMQ.URL = "amqp://guest:guest@rabbitmq:5672/"
	err := validateProductionSecrets(cfg)
	if err == nil || !strings.Contains(err.Error(), "RABBITMQ_URL") {
		t.Fatalf("expected RABBITMQ_URL default credential error, got %v", err)
	}
}

func TestValidateJWTSecretRejectsLongPlaceholder(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	secret := "CHANGE_ME_JWT_SECRET_AT_LEAST_32_CHARS"
	err := validateJWTSecret(secret)
	if err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("expected placeholder JWT error, got %v", err)
	}
}

func TestInsecureDevConfigBypassesProductionSecretChecks(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "true")

	if err := validateProductionSecrets(Config{}); err != nil {
		t.Fatalf("validateProductionSecrets with dev bypass: %v", err)
	}
	if err := validateJWTSecret("CHANGE_ME"); err != nil {
		t.Fatalf("validateJWTSecret with dev bypass: %v", err)
	}
}

func productionReadyConfig() Config {
	var cfg Config
	cfg.MySQL.DSN = "smart_recruit:strong-password@tcp(mysql:3306)/recruitment?charset=utf8mb4&parseTime=True&loc=Local"
	cfg.RabbitMQ.URL = "amqp://recruitment:strong-password@rabbitmq:5672/"
	cfg.OSS.AccessKeyID = "AKIDEXTERNAL123456"
	cfg.OSS.AccessKeySecret = "external-oss-secret-value"
	cfg.OSS.BucketName = "recruitment-prod"
	return cfg
}
