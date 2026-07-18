package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPPort              string
	GRPCAddr              string
	NotificationGRPCAddr  string
	NotificationRouteMode string
	AIAgentGRPCAddr       string
	AIAgentRouteMode      string
	IdentityGRPCAddr      string
	IdentityRouteMode     string
	RecruitmentGRPCAddr   string
	RecruitmentRouteMode  string
	InterviewGRPCAddr     string
	InterviewRouteMode    string
	OfferGRPCAddr         string
	OfferRouteMode        string
	GRPCInternalTLS       string
	GRPCTLSCAFile         string
	GRPCTLSServerName     string
	JWTSecret             string
	AuthCookieName        string
	CandidateCookie       string
	HRCookie              string
	InterviewerCookie     string
	AuthCookieSecure      bool
	ShutdownTimeout       time.Duration
	Redis                 RedisConfig
	RateLimit             RateLimitConfig
	// Ranking keeps the gateway-side env contract for skill and memory scoring knobs.
	Ranking Ranking
}

// Ranking 描述 Skill / Memory 召回排序的可调权重与阈值。
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
	notificationRouteMode := env("NOTIFICATION_ROUTE_MODE", "notification")
	notificationGRPCAddr := env("NOTIFICATION_GRPC_ADDR", "127.0.0.1:50065")
	if err := validateNotificationRoute(notificationRouteMode, notificationGRPCAddr); err != nil {
		return Config{}, err
	}
	aiAgentRouteMode := env("AI_AGENT_ROUTE_MODE", "ai-agent")
	aiAgentGRPCAddr := env("AI_AGENT_GRPC_ADDR", "127.0.0.1:50066")
	if err := validateAIAgentRoute(aiAgentRouteMode, aiAgentGRPCAddr); err != nil {
		return Config{}, err
	}
	identityRouteMode := env("IDENTITY_ROUTE_MODE", "identity")
	identityGRPCAddr := env("IDENTITY_GRPC_ADDR", "127.0.0.1:50061")
	if err := validateIdentityRoute(identityRouteMode, identityGRPCAddr); err != nil {
		return Config{}, err
	}
	recruitmentRouteMode := env("RECRUITMENT_ROUTE_MODE", "recruitment")
	recruitmentGRPCAddr := env("RECRUITMENT_GRPC_ADDR", "127.0.0.1:50062")
	if err := validateRecruitmentRoute(recruitmentRouteMode, recruitmentGRPCAddr); err != nil {
		return Config{}, err
	}
	interviewRouteMode := env("INTERVIEW_ROUTE_MODE", "interview")
	interviewGRPCAddr := env("INTERVIEW_GRPC_ADDR", "127.0.0.1:50063")
	if err := validateInterviewRoute(interviewRouteMode, interviewGRPCAddr); err != nil {
		return Config{}, err
	}
	offerRouteMode := env("OFFER_ROUTE_MODE", "offer")
	offerGRPCAddr := env("OFFER_GRPC_ADDR", "127.0.0.1:50064")
	if err := validateOfferRoute(offerRouteMode, offerGRPCAddr); err != nil {
		return Config{}, err
	}
	grpcInternalTLS := env("GRPC_INTERNAL_TLS", "optional")
	grpcTLSCAFile := env("GRPC_TLS_CA_FILE", "")
	if err := validateInternalTLSConfig(grpcInternalTLS, grpcTLSCAFile); err != nil {
		return Config{}, err
	}
	return Config{
		HTTPPort:              env("HTTP_PORT", "8080"),
		GRPCAddr:              env("GRPC_ADDR", "127.0.0.1:50062"),
		NotificationGRPCAddr:  notificationGRPCAddr,
		NotificationRouteMode: notificationRouteMode,
		AIAgentGRPCAddr:       aiAgentGRPCAddr,
		AIAgentRouteMode:      aiAgentRouteMode,
		IdentityGRPCAddr:      identityGRPCAddr,
		IdentityRouteMode:     identityRouteMode,
		RecruitmentGRPCAddr:   recruitmentGRPCAddr,
		RecruitmentRouteMode:  recruitmentRouteMode,
		InterviewGRPCAddr:     interviewGRPCAddr,
		InterviewRouteMode:    interviewRouteMode,
		OfferGRPCAddr:         offerGRPCAddr,
		OfferRouteMode:        offerRouteMode,
		GRPCInternalTLS:       grpcInternalTLS,
		GRPCTLSCAFile:         grpcTLSCAFile,
		GRPCTLSServerName:     env("GRPC_TLS_SERVER_NAME", ""),
		JWTSecret:             secret,
		AuthCookieName:        env("AUTH_COOKIE_NAME", "recruitment_token"),
		CandidateCookie:       env("CANDIDATE_AUTH_COOKIE_NAME", "recruitment_candidate_token"),
		HRCookie:              env("HR_AUTH_COOKIE_NAME", "recruitment_hr_token"),
		InterviewerCookie:     env("INTERVIEWER_AUTH_COOKIE_NAME", "recruitment_interviewer_token"),
		AuthCookieSecure:      envBool("AUTH_COOKIE_SECURE", false),
		ShutdownTimeout:       envDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
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

func validateOfferRoute(mode, addr string) error {
	switch mode {
	case "offer":
		if strings.TrimSpace(addr) == "" {
			return fmt.Errorf("OFFER_GRPC_ADDR is required when OFFER_ROUTE_MODE=offer")
		}
		return nil
	default:
		return fmt.Errorf("OFFER_ROUTE_MODE must be offer")
	}
}

func validateInterviewRoute(mode, addr string) error {
	switch mode {
	case "interview":
		if strings.TrimSpace(addr) == "" {
			return fmt.Errorf("INTERVIEW_GRPC_ADDR is required when INTERVIEW_ROUTE_MODE=interview")
		}
		return nil
	default:
		return fmt.Errorf("INTERVIEW_ROUTE_MODE must be interview")
	}
}

func validateRecruitmentRoute(mode, addr string) error {
	switch mode {
	case "recruitment":
		if strings.TrimSpace(addr) == "" {
			return fmt.Errorf("RECRUITMENT_GRPC_ADDR is required when RECRUITMENT_ROUTE_MODE=recruitment")
		}
		return nil
	default:
		return fmt.Errorf("RECRUITMENT_ROUTE_MODE must be recruitment")
	}
}

func validateIdentityRoute(mode, addr string) error {
	switch mode {
	case "identity":
		if strings.TrimSpace(addr) == "" {
			return fmt.Errorf("IDENTITY_GRPC_ADDR is required when IDENTITY_ROUTE_MODE=identity")
		}
		return nil
	default:
		return fmt.Errorf("IDENTITY_ROUTE_MODE must be identity")
	}
}

func validateAIAgentRoute(mode, addr string) error {
	switch mode {
	case "ai-agent":
		if strings.TrimSpace(addr) == "" {
			return fmt.Errorf("AI_AGENT_GRPC_ADDR is required when AI_AGENT_ROUTE_MODE=ai-agent")
		}
		return nil
	default:
		return fmt.Errorf("AI_AGENT_ROUTE_MODE must be ai-agent")
	}
}

func validateNotificationRoute(mode, addr string) error {
	switch mode {
	case "notification":
		if strings.TrimSpace(addr) == "" {
			return fmt.Errorf("NOTIFICATION_GRPC_ADDR is required when NOTIFICATION_ROUTE_MODE=notification")
		}
		return nil
	default:
		return fmt.Errorf("NOTIFICATION_ROUTE_MODE must be notification")
	}
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

// envFloat64 parses a float64 env value and returns 0 when the value is absent or invalid.
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

func validateInternalTLSConfig(mode, caFile string) error {
	mode = strings.TrimSpace(strings.ToLower(mode))
	switch mode {
	case "", "optional":
		return nil
	case "required":
		if strings.TrimSpace(caFile) == "" {
			return fmt.Errorf("GRPC_TLS_CA_FILE is empty while GRPC_INTERNAL_TLS=required")
		}
		return nil
	default:
		return fmt.Errorf("GRPC_INTERNAL_TLS must be optional or required")
	}
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
