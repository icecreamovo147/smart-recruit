package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTPPort        string
	GRPCAddr        string
	JWTSecret       string
	ShutdownTimeout time.Duration
	Redis           RedisConfig
	RateLimit       RateLimitConfig
}

type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type RateLimitConfig struct {
	AuthRPS      int
	AuthBurst    int
	AIRPS        int
	AIBurst      int
	GeneralRPS   int
	GeneralBurst int
}

func Load() (Config, error) {
	secret := jwtSecret()
	if err := validateJWTSecret(secret); err != nil {
		return Config{}, err
	}
	return Config{
		HTTPPort:        env("HTTP_PORT", "8080"),
		GRPCAddr:        env("GRPC_ADDR", "127.0.0.1:50051"),
		JWTSecret:       secret,
		ShutdownTimeout: envDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		Redis: RedisConfig{
			Addr:         env("REDIS_ADDR", "127.0.0.1:6379"),
			Password:     env("REDIS_PASSWORD", ""),
			DB:           envInt("REDIS_DB", 0),
			PoolSize:     envInt("REDIS_POOL_SIZE", 20),
			MinIdleConns: envInt("REDIS_MIN_IDLE_CONNS", 5),
			DialTimeout:  envDuration("REDIS_DIAL_TIMEOUT", 2*time.Second),
			ReadTimeout:  envDuration("REDIS_READ_TIMEOUT", 0), // PubSub 长连接不能有读超时
			WriteTimeout: envDuration("REDIS_WRITE_TIMEOUT", time.Second),
		},
		RateLimit: RateLimitConfig{
			AuthRPS:      envInt("RATE_LIMIT_AUTH_RPS", 20),
			AuthBurst:    envInt("RATE_LIMIT_AUTH_BURST", 40),
			AIRPS:        envInt("RATE_LIMIT_AI_RPS", 10),
			AIBurst:      envInt("RATE_LIMIT_AI_BURST", 20),
			GeneralRPS:   envInt("RATE_LIMIT_GENERAL_RPS", 100),
			GeneralBurst: envInt("RATE_LIMIT_GENERAL_BURST", 200),
		},
	}, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		parsed, err := time.ParseDuration(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
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

func jwtSecret() string {
	if value := os.Getenv("JWT_SECRET"); value != "" {
		return value
	}
	path := os.Getenv("LOGIC_CONFIG_PATH")
	if path == "" {
		path = "../logic-grpc-service/config/config.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var cfg struct {
		JWT struct {
			Secret string `yaml:"secret"`
		} `yaml:"jwt"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil || cfg.JWT.Secret == "" {
		return ""
	}
	return cfg.JWT.Secret
}
