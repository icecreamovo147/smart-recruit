package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPPort          string
	GRPCAddr          string
	JWTSecret         string
	AuthCookieName    string
	CandidateCookie   string
	HRCookie          string
	InterviewerCookie string
	AuthCookieSecure  bool
	ShutdownTimeout   time.Duration
	Redis             RedisConfig
	RateLimit         RateLimitConfig
	// TASK-FU-009：与 logic-grpc-service 同步的 ranking 权重 / 阈值。
	// web-gin 当前不直接使用这些值；保留是为未来扩展做准备，
	// 并保证两端的 env 变量读取行为一致。
	Ranking Ranking
}

// Ranking 描述 Skill / Memory 召回排序的可调权重与阈值。
// TASK-FU-009 引入：与 logic-grpc-service/config.Ranking 同形；
// web-gin 不直接 import logic-grpc-service，故在本地复制 struct 定义。
// 字段语义与 SDD §3.3 完全对齐；默认值与 logic-grpc-service 一致。
type Ranking struct {
	WeightVector     float64
	WeightLexical    float64
	WeightMetadata   float64
	BusinessBoostMax float64
	PriorityNorm     float64
	BoostAlpha       float64
	BoostBeta        float64
	BoostGamma       float64
	RelevanceGate    float64
	GapHigh          float64
	GapMedium        float64
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
	AuthRPS                  int
	AuthBurst                int
	AIRPS                    int
	AIBurst                  int
	GeneralRPS               int
	GeneralBurst             int
	AIQuotaCandidateDaily    int
	AIQuotaHRDaily           int
	ResumePresignHourlyLimit int
	ResumePresignDailyLimit  int
	ResumeConfirmHourlyLimit int
	ResumeConfirmDailyLimit  int
}

func Load() (Config, error) {
	secret := jwtSecret()
	if err := validateJWTSecret(secret); err != nil {
		return Config{}, err
	}
	if err := validateInternalAuthConfig(); err != nil {
		return Config{}, err
	}
	return Config{
		HTTPPort:          env("HTTP_PORT", "8080"),
		GRPCAddr:          env("GRPC_ADDR", "127.0.0.1:50051"),
		JWTSecret:         secret,
		AuthCookieName:    env("AUTH_COOKIE_NAME", "recruitment_token"),
		CandidateCookie:   env("CANDIDATE_AUTH_COOKIE_NAME", "recruitment_candidate_token"),
		HRCookie:          env("HR_AUTH_COOKIE_NAME", "recruitment_hr_token"),
		InterviewerCookie: env("INTERVIEWER_AUTH_COOKIE_NAME", "recruitment_interviewer_token"),
		AuthCookieSecure:  envBool("AUTH_COOKIE_SECURE", false),
		ShutdownTimeout:   envDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
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
			AuthRPS:                  envInt("RATE_LIMIT_AUTH_RPS", 20),
			AuthBurst:                envInt("RATE_LIMIT_AUTH_BURST", 40),
			AIRPS:                    envInt("RATE_LIMIT_AI_RPS", 1),
			AIBurst:                  envInt("RATE_LIMIT_AI_BURST", 3),
			GeneralRPS:               envInt("RATE_LIMIT_GENERAL_RPS", 100),
			GeneralBurst:             envInt("RATE_LIMIT_GENERAL_BURST", 200),
			AIQuotaCandidateDaily:    envInt("AI_QUOTA_CANDIDATE_DAILY", 20),
			AIQuotaHRDaily:           envInt("AI_QUOTA_HR_DAILY", 100),
			ResumePresignHourlyLimit: envInt("RESUME_PRESIGN_HOURLY_LIMIT", 5),
			ResumePresignDailyLimit:  envInt("RESUME_PRESIGN_DAILY_LIMIT", 20),
			ResumeConfirmHourlyLimit: envInt("RESUME_CONFIRM_HOURLY_LIMIT", 5),
			ResumeConfirmDailyLimit:  envInt("RESUME_CONFIRM_DAILY_LIMIT", 20),
		},
		// TASK-FU-009：与 logic-grpc-service 同步的 ranking 段。
		// web-gin 不直接 import logic-grpc；本地复制 struct 定义 + 独立 env 解析。
		Ranking: Ranking{
			WeightVector:     envFloat64("RANKING_WEIGHT_VECTOR"),
			WeightLexical:    envFloat64("RANKING_WEIGHT_LEXICAL"),
			WeightMetadata:   envFloat64("RANKING_WEIGHT_METADATA"),
			BusinessBoostMax: envFloat64("RANKING_BUSINESS_BOOST_MAX"),
			PriorityNorm:     envFloat64("RANKING_PRIORITY_NORM"),
			BoostAlpha:       envFloat64("RANKING_BOOST_ALPHA"),
			BoostBeta:        envFloat64("RANKING_BOOST_BETA"),
			BoostGamma:       envFloat64("RANKING_BOOST_GAMMA"),
			RelevanceGate:    envFloat64("RANKING_RELEVANCE_GATE"),
			GapHigh:          envFloat64("RANKING_GAP_HIGH"),
			GapMedium:        envFloat64("RANKING_GAP_MEDIUM"),
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

func envBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
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

// envFloat64 TASK-FU-009 引入：解析 float64 env 值，失败时返回 0。
// 与 logic-grpc-service/config.setFloat64 行为一致。
func envFloat64(key string) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// validateJWTSecret rejects weak JWT secrets in non-dev environments.
// Set ALLOW_INSECURE_DEV_CONFIG=true to bypass this check (local dev only).
func validateJWTSecret(secret string) error {
	if allowInsecureDevConfig() {
		return nil
	}
	if secret == "" {
		return fmt.Errorf("JWT_SECRET is empty: production requires a strong secret (>= 32 chars). Set ALLOW_INSECURE_DEV_CONFIG=true only for local development")
	}
	if isPlaceholderSecret(secret) {
		return fmt.Errorf("JWT_SECRET is still the default placeholder: production requires a strong secret (>= 32 chars). Set ALLOW_INSECURE_DEV_CONFIG=true only for local development")
	}
	if len(secret) < 16 {
		return fmt.Errorf("JWT_SECRET is too short (%d chars): production requires at least 16 chars, 32 recommended. Set ALLOW_INSECURE_DEV_CONFIG=true only for local development", len(secret))
	}
	return nil
}

func validateInternalAuthConfig() error {
	if allowInsecureDevConfig() {
		return nil
	}
	token := strings.TrimSpace(os.Getenv("GRPC_INTERNAL_TOKEN"))
	if token == "" {
		return fmt.Errorf("GRPC_INTERNAL_TOKEN is empty: production gateway must send a shared internal token. Set ALLOW_INSECURE_DEV_CONFIG=true only for local development")
	}
	if isPlaceholderSecret(token) {
		return fmt.Errorf("GRPC_INTERNAL_TOKEN is still a placeholder: production gateway must send a real shared internal token")
	}
	if len(token) < 16 {
		return fmt.Errorf("GRPC_INTERNAL_TOKEN is too short: production requires at least 16 chars, 32 recommended. Set ALLOW_INSECURE_DEV_CONFIG=true only for local development")
	}
	return nil
}

func jwtSecret() string {
	value := os.Getenv("JWT_SECRET")
	if value != "" {
		return value
	}
	if os.Getenv("ALLOW_INSECURE_DEV_CONFIG") == "true" {
		return ""
	}
	return ""
}

func allowInsecureDevConfig() bool {
	return os.Getenv("ALLOW_INSECURE_DEV_CONFIG") == "true"
}

func isPlaceholderSecret(value string) bool {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return false
	}
	upper := strings.ToUpper(normalized)
	lower := strings.ToLower(normalized)
	placeholderFragments := []string{
		"CHANGE_ME",
		"PLEASE_CHANGE_ME",
		"your_password",
		"your-secret",
		"your-access-key",
		"your-api-key",
		"sk-your",
		"bucket-name",
		"dev-placeholder",
		"dev-shared-token",
		"dev-jwt-secret",
	}
	for _, fragment := range placeholderFragments {
		if strings.Contains(upper, strings.ToUpper(fragment)) || strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}
