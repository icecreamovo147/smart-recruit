package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MySQL struct {
		DSN             string   `yaml:"dsn"`
		MaxOpenConns    int      `yaml:"max_open_conns"`
		MaxIdleConns    int      `yaml:"max_idle_conns"`
		ConnMaxLifetime Duration `yaml:"conn_max_lifetime"`
		ConnMaxIdleTime Duration `yaml:"conn_max_idle_time"`
	} `yaml:"mysql"`
	OSS struct {
		Provider        string `yaml:"provider"`
		Endpoint        string `yaml:"endpoint"`
		AccessKeyID     string `yaml:"access_key_id"`
		AccessKeySecret string `yaml:"access_key_secret"`
		BucketName      string `yaml:"bucket_name"`
		PublicBaseURL   string `yaml:"public_base_url"`
	} `yaml:"oss"`
	AI struct {
		APIKey                     string   `yaml:"api_key"`
		Model                      string   `yaml:"model"`
		BaseURL                    string   `yaml:"base_url"`
		AgentRuntime               string   `yaml:"agent_runtime"`
		Timeout                    Duration `yaml:"timeout"`
		TotalTimeout               Duration `yaml:"total_timeout"`
		ToolMaxRounds              int      `yaml:"tool_max_rounds"`
		ToolTotalTimeout           Duration `yaml:"tool_total_timeout"`
		MaxConcurrency             int      `yaml:"max_concurrency"`
		CircuitFailureThreshold    int      `yaml:"circuit_failure_threshold"`
		CircuitOpenTimeout         Duration `yaml:"circuit_open_timeout"`
		CircuitHalfOpenMaxRequests int      `yaml:"circuit_half_open_max_requests"`
		RetryMaxAttempts           int      `yaml:"retry_max_attempts"`
		RetryBaseDelay             Duration `yaml:"retry_base_delay"`
		SlowResponseThreshold      Duration `yaml:"slow_response_threshold"`
	} `yaml:"ai"`
	JWT struct {
		Secret string `yaml:"secret"`
	} `yaml:"jwt"`
	GRPC struct {
		Port int `yaml:"port"`
	} `yaml:"grpc"`
	Redis struct {
		Addr         string   `yaml:"addr"`
		Password     string   `yaml:"password"`
		DB           int      `yaml:"db"`
		PoolSize     int      `yaml:"pool_size"`
		MinIdleConns int      `yaml:"min_idle_conns"`
		DialTimeout  Duration `yaml:"dial_timeout"`
		ReadTimeout  Duration `yaml:"read_timeout"`
		WriteTimeout Duration `yaml:"write_timeout"`
	} `yaml:"redis"`
	Agent struct {
		RecentMessageLimit     int `yaml:"recent_message_limit"`
		SummaryTriggerMessages int `yaml:"summary_trigger_messages"`
		MaxMemoryChars         int `yaml:"max_memory_chars"`
		MaxPromptChars         int `yaml:"max_prompt_chars"`
		MaxMemories            int `yaml:"max_memories"`
	} `yaml:"agent"`
	RabbitMQ struct {
		URL               string   `yaml:"url"`
		Exchange          string   `yaml:"exchange"`
		DLXExchange       string   `yaml:"dlx_exchange"`
		RetryExchange     string   `yaml:"retry_exchange"`
		NotificationQueue string   `yaml:"notification_queue"`
		ResumeParseQueue  string   `yaml:"resume_parse_queue"`
		PrefetchCount     int      `yaml:"prefetch_count"`
		MaxRetries        int      `yaml:"max_retries"`
		RetryDelay        Duration `yaml:"retry_delay"`
		ReconnectInterval Duration `yaml:"reconnect_interval"`
	} `yaml:"rabbitmq"`
}

func Load() (Config, error) {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config/config.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		// In production (ALLOW_INSECURE_DEV_CONFIG not set), do NOT fall back to
		// config.example.yaml which contains localhost/weak credentials.
		if os.Getenv("ALLOW_INSECURE_DEV_CONFIG") == "true" {
			data, err = os.ReadFile("config/config.example.yaml")
			if err != nil {
				return Config{}, err
			}
		} else {
			return Config{}, fmt.Errorf("config file not found at %q and no fallback is allowed in production. Set ALLOW_INSECURE_DEV_CONFIG=true for local development", path)
		}
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	applyEnvOverrides(&cfg)
	if cfg.GRPC.Port == 0 {
		cfg.GRPC.Port = 50051
	}
	if cfg.JWT.Secret == "" {
		cfg.JWT.Secret = os.Getenv("JWT_SECRET")
	}
	if err := validateJWTSecret(cfg.JWT.Secret); err != nil {
		return Config{}, err
	}
	if cfg.MySQL.MaxOpenConns <= 0 {
		cfg.MySQL.MaxOpenConns = 50
	}
	if cfg.MySQL.MaxIdleConns <= 0 {
		cfg.MySQL.MaxIdleConns = 10
	}
	if cfg.MySQL.ConnMaxLifetime.Duration <= 0 {
		cfg.MySQL.ConnMaxLifetime.Duration = time.Hour
	}
	if cfg.MySQL.ConnMaxIdleTime.Duration <= 0 {
		cfg.MySQL.ConnMaxIdleTime.Duration = 10 * time.Minute
	}
	if cfg.Redis.PoolSize <= 0 {
		cfg.Redis.PoolSize = 20
	}
	if cfg.Redis.MinIdleConns <= 0 {
		cfg.Redis.MinIdleConns = 5
	}
	if cfg.Redis.DialTimeout.Duration <= 0 {
		cfg.Redis.DialTimeout.Duration = 2 * time.Second
	}
	if cfg.Redis.ReadTimeout.Duration <= 0 {
		cfg.Redis.ReadTimeout.Duration = time.Second
	}
	if cfg.Redis.WriteTimeout.Duration <= 0 {
		cfg.Redis.WriteTimeout.Duration = time.Second
	}
	if cfg.AI.Timeout.Duration <= 0 {
		cfg.AI.Timeout.Duration = 90 * time.Second
	}
	if cfg.AI.TotalTimeout.Duration <= 0 {
		cfg.AI.TotalTimeout.Duration = 120 * time.Second
	}
	if cfg.AI.ToolMaxRounds <= 0 {
		cfg.AI.ToolMaxRounds = 5
	}
	if cfg.AI.ToolTotalTimeout.Duration <= 0 {
		cfg.AI.ToolTotalTimeout.Duration = 30 * time.Second
	}
	if cfg.AI.MaxConcurrency <= 0 {
		cfg.AI.MaxConcurrency = 10
	}
	if cfg.AI.CircuitFailureThreshold <= 0 {
		cfg.AI.CircuitFailureThreshold = 5
	}
	if cfg.AI.CircuitOpenTimeout.Duration <= 0 {
		cfg.AI.CircuitOpenTimeout.Duration = 30 * time.Second
	}
	if cfg.AI.CircuitHalfOpenMaxRequests <= 0 {
		cfg.AI.CircuitHalfOpenMaxRequests = 2
	}
	if cfg.AI.RetryMaxAttempts < 0 {
		cfg.AI.RetryMaxAttempts = 0
	}
	if cfg.AI.RetryBaseDelay.Duration <= 0 {
		cfg.AI.RetryBaseDelay.Duration = 500 * time.Millisecond
	}
	if cfg.AI.SlowResponseThreshold.Duration <= 0 {
		cfg.AI.SlowResponseThreshold.Duration = 5 * time.Second
	}
	if cfg.AI.AgentRuntime == "" {
		cfg.AI.AgentRuntime = "adk"
	}
	if cfg.Agent.RecentMessageLimit <= 0 {
		cfg.Agent.RecentMessageLimit = 20
	}
	if cfg.Agent.SummaryTriggerMessages <= 0 {
		cfg.Agent.SummaryTriggerMessages = 30
	}
	if cfg.Agent.MaxMemoryChars <= 0 {
		cfg.Agent.MaxMemoryChars = 1500
	}
	if cfg.Agent.MaxPromptChars <= 0 {
		cfg.Agent.MaxPromptChars = 20000
	}
	if cfg.Agent.MaxMemories <= 0 {
		cfg.Agent.MaxMemories = 10
	}
	if cfg.RabbitMQ.URL == "" {
		cfg.RabbitMQ.URL = "amqp://guest:guest@127.0.0.1:5672/"
	}
	if cfg.RabbitMQ.Exchange == "" {
		cfg.RabbitMQ.Exchange = "recruitment.events"
	}
	if cfg.RabbitMQ.DLXExchange == "" {
		cfg.RabbitMQ.DLXExchange = "recruitment.events.dlx"
	}
	if cfg.RabbitMQ.RetryExchange == "" {
		cfg.RabbitMQ.RetryExchange = "recruitment.events.retry"
	}
	if cfg.RabbitMQ.NotificationQueue == "" {
		cfg.RabbitMQ.NotificationQueue = "recruitment.notification.create"
	}
	if cfg.RabbitMQ.ResumeParseQueue == "" {
		cfg.RabbitMQ.ResumeParseQueue = "recruitment.resume.parse"
	}
	if cfg.RabbitMQ.PrefetchCount <= 0 {
		cfg.RabbitMQ.PrefetchCount = 10
	}
	if cfg.RabbitMQ.MaxRetries <= 0 {
		cfg.RabbitMQ.MaxRetries = 5
	}
	if cfg.RabbitMQ.RetryDelay.Duration <= 0 {
		cfg.RabbitMQ.RetryDelay.Duration = 5 * time.Second
	}
	if cfg.RabbitMQ.ReconnectInterval.Duration <= 0 {
		cfg.RabbitMQ.ReconnectInterval.Duration = 3 * time.Second
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	setString(&cfg.MySQL.DSN, "MYSQL_DSN")
	setInt(&cfg.MySQL.MaxOpenConns, "MYSQL_MAX_OPEN_CONNS")
	setInt(&cfg.MySQL.MaxIdleConns, "MYSQL_MAX_IDLE_CONNS")
	setDuration(&cfg.MySQL.ConnMaxLifetime, "MYSQL_CONN_MAX_LIFETIME")
	setDuration(&cfg.MySQL.ConnMaxIdleTime, "MYSQL_CONN_MAX_IDLE_TIME")

	setString(&cfg.OSS.Provider, "OSS_PROVIDER")
	setString(&cfg.OSS.Endpoint, "OSS_ENDPOINT")
	setString(&cfg.OSS.AccessKeyID, "OSS_ACCESS_KEY_ID")
	setString(&cfg.OSS.AccessKeySecret, "OSS_ACCESS_KEY_SECRET")
	setString(&cfg.OSS.BucketName, "OSS_BUCKET_NAME")
	setString(&cfg.OSS.PublicBaseURL, "OSS_PUBLIC_BASE_URL")

	setString(&cfg.AI.APIKey, "AI_API_KEY")
	setString(&cfg.AI.Model, "AI_MODEL")
	setString(&cfg.AI.BaseURL, "AI_BASE_URL")
	setString(&cfg.AI.AgentRuntime, "AI_AGENT_RUNTIME")
	setDuration(&cfg.AI.Timeout, "AI_TIMEOUT")
	setDuration(&cfg.AI.TotalTimeout, "AI_TOTAL_TIMEOUT")
	setInt(&cfg.AI.ToolMaxRounds, "AI_TOOL_MAX_ROUNDS")
	setDuration(&cfg.AI.ToolTotalTimeout, "AI_TOOL_TOTAL_TIMEOUT")
	setInt(&cfg.AI.MaxConcurrency, "AI_MAX_CONCURRENCY")
	setInt(&cfg.AI.CircuitFailureThreshold, "AI_CIRCUIT_FAILURE_THRESHOLD")
	setDuration(&cfg.AI.CircuitOpenTimeout, "AI_CIRCUIT_OPEN_TIMEOUT")
	setInt(&cfg.AI.CircuitHalfOpenMaxRequests, "AI_CIRCUIT_HALF_OPEN_MAX_REQUESTS")
	setInt(&cfg.AI.RetryMaxAttempts, "AI_RETRY_MAX_ATTEMPTS")
	setDuration(&cfg.AI.RetryBaseDelay, "AI_RETRY_BASE_DELAY")
	setDuration(&cfg.AI.SlowResponseThreshold, "AI_SLOW_RESPONSE_THRESHOLD")

	setString(&cfg.JWT.Secret, "JWT_SECRET")
	setInt(&cfg.GRPC.Port, "GRPC_PORT")

	setString(&cfg.Redis.Addr, "REDIS_ADDR")
	setString(&cfg.Redis.Password, "REDIS_PASSWORD")
	setInt(&cfg.Redis.DB, "REDIS_DB")
	setInt(&cfg.Redis.PoolSize, "REDIS_POOL_SIZE")
	setInt(&cfg.Redis.MinIdleConns, "REDIS_MIN_IDLE_CONNS")
	setDuration(&cfg.Redis.DialTimeout, "REDIS_DIAL_TIMEOUT")
	setDuration(&cfg.Redis.ReadTimeout, "REDIS_READ_TIMEOUT")
	setDuration(&cfg.Redis.WriteTimeout, "REDIS_WRITE_TIMEOUT")

	setInt(&cfg.Agent.RecentMessageLimit, "AGENT_RECENT_MESSAGE_LIMIT")
	setInt(&cfg.Agent.SummaryTriggerMessages, "AGENT_SUMMARY_TRIGGER_MESSAGES")
	setInt(&cfg.Agent.MaxMemoryChars, "AGENT_MAX_MEMORY_CHARS")
	setInt(&cfg.Agent.MaxPromptChars, "AGENT_MAX_PROMPT_CHARS")
	setInt(&cfg.Agent.MaxMemories, "AGENT_MAX_MEMORIES")

	setString(&cfg.RabbitMQ.URL, "RABBITMQ_URL")
	setString(&cfg.RabbitMQ.Exchange, "RABBITMQ_EXCHANGE")
	setString(&cfg.RabbitMQ.DLXExchange, "RABBITMQ_DLX_EXCHANGE")
	setString(&cfg.RabbitMQ.RetryExchange, "RABBITMQ_RETRY_EXCHANGE")
	setString(&cfg.RabbitMQ.NotificationQueue, "RABBITMQ_NOTIFICATION_QUEUE")
	setString(&cfg.RabbitMQ.ResumeParseQueue, "RABBITMQ_RESUME_PARSE_QUEUE")
	setInt(&cfg.RabbitMQ.PrefetchCount, "RABBITMQ_PREFETCH_COUNT")
	setInt(&cfg.RabbitMQ.MaxRetries, "RABBITMQ_MAX_RETRIES")
	setDuration(&cfg.RabbitMQ.RetryDelay, "RABBITMQ_RETRY_DELAY")
	setDuration(&cfg.RabbitMQ.ReconnectInterval, "RABBITMQ_RECONNECT_INTERVAL")
}

func setString(target *string, key string) {
	if value := os.Getenv(key); value != "" {
		*target = value
	}
}

func setInt(target *int, key string) {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			*target = parsed
		}
	}
}

func setDuration(target *Duration, key string) {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			target.Duration = parsed
		}
	}
}

// validateJWTSecret rejects weak JWT secrets in non-dev environments.
// Set ALLOW_INSECURE_DEV_CONFIG=true to bypass this check (local dev only).
func validateJWTSecret(secret string) error {
	if os.Getenv("ALLOW_INSECURE_DEV_CONFIG") == "true" {
		return nil
	}
	if secret == "" {
		return fmt.Errorf("JWT_SECRET is empty: production requires a strong secret (>= 32 chars). Set ALLOW_INSECURE_DEV_CONFIG=true only for local development")
	}
	if secret == "please-change-me" || secret == "CHANGE_ME" {
		return fmt.Errorf("JWT_SECRET is still the default placeholder: production requires a strong secret (>= 32 chars). Set ALLOW_INSECURE_DEV_CONFIG=true only for local development")
	}
	if len(secret) < 16 {
		return fmt.Errorf("JWT_SECRET is too short (%d chars): production requires at least 16 chars, 32 recommended. Set ALLOW_INSECURE_DEV_CONFIG=true only for local development", len(secret))
	}
	return nil
}
