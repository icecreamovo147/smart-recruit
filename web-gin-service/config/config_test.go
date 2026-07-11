package config

import (
	"strings"
	"testing"
)

func TestLoadAuthCookieSecure(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("AUTH_COOKIE_SECURE", "true")
	t.Setenv("CANDIDATE_AUTH_COOKIE_NAME", "candidate_session")
	t.Setenv("HR_AUTH_COOKIE_NAME", "hr_session")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.AuthCookieSecure {
		t.Fatalf("expected AuthCookieSecure=true")
	}
	if cfg.CandidateCookie != "candidate_session" {
		t.Fatalf("expected candidate cookie override, got %q", cfg.CandidateCookie)
	}
	if cfg.HRCookie != "hr_session" {
		t.Fatalf("expected HR cookie override, got %q", cfg.HRCookie)
	}
}

// TASK-FU-009：验证 web-gin 同步 logic-grpc 的 ranking 段 env 加载。
func TestLoadRankingConfig(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("RANKING_WEIGHT_VECTOR", "0.7")
	t.Setenv("RANKING_BUSINESS_BOOST_MAX", "1.3")
	t.Setenv("RANKING_RELEVANCE_GATE", "0.2")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Ranking.WeightVector != 0.7 {
		t.Fatalf("WeightVector = %v, want 0.7", cfg.Ranking.WeightVector)
	}
	if cfg.Ranking.BusinessBoostMax != 1.3 {
		t.Fatalf("BusinessBoostMax = %v, want 1.3", cfg.Ranking.BusinessBoostMax)
	}
	if cfg.Ranking.RelevanceGate != 0.2 {
		t.Fatalf("RelevanceGate = %v, want 0.2", cfg.Ranking.RelevanceGate)
	}
	// 未设 env 的字段保持 0
	if cfg.Ranking.WeightLexical != 0 {
		t.Fatalf("WeightLexical should be 0 (unset), got %v", cfg.Ranking.WeightLexical)
	}
}

func TestLoadNotificationRouteDefaultsToLogic(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.NotificationRouteMode != "logic" {
		t.Fatalf("NotificationRouteMode = %q, want logic", cfg.NotificationRouteMode)
	}
	if cfg.NotificationGRPCAddr != "" {
		t.Fatalf("NotificationGRPCAddr = %q, want empty", cfg.NotificationGRPCAddr)
	}
}

func TestLoadAIAgentRouteDefaultsToLogic(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AIAgentRouteMode != "logic" {
		t.Fatalf("AIAgentRouteMode = %q, want logic", cfg.AIAgentRouteMode)
	}
	if cfg.AIAgentGRPCAddr != "" {
		t.Fatalf("AIAgentGRPCAddr = %q, want empty", cfg.AIAgentGRPCAddr)
	}
}

func TestLoadIdentityRouteDefaultsToLogic(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IdentityRouteMode != "logic" {
		t.Fatalf("IdentityRouteMode = %q, want logic", cfg.IdentityRouteMode)
	}
	if cfg.IdentityGRPCAddr != "" {
		t.Fatalf("IdentityGRPCAddr = %q, want empty", cfg.IdentityGRPCAddr)
	}
}

func TestLoadIdentityRouteToExtractedService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("IDENTITY_ROUTE_MODE", "identity")
	t.Setenv("IDENTITY_GRPC_ADDR", "dns:///identity-service:50051")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IdentityRouteMode != "identity" {
		t.Fatalf("IdentityRouteMode = %q, want identity", cfg.IdentityRouteMode)
	}
	if cfg.IdentityGRPCAddr != "dns:///identity-service:50051" {
		t.Fatalf("IdentityGRPCAddr = %q", cfg.IdentityGRPCAddr)
	}
}

func TestLoadIdentityRouteRequiresAddress(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("IDENTITY_ROUTE_MODE", "identity")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "IDENTITY_GRPC_ADDR") {
		t.Fatalf("expected identity addr error, got %v", err)
	}
}

func TestLoadRejectsInvalidIdentityRouteMode(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("IDENTITY_ROUTE_MODE", "invalid")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "IDENTITY_ROUTE_MODE") {
		t.Fatalf("expected identity route mode error, got %v", err)
	}
}

func TestLoadAIAgentRouteToExtractedService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("AI_AGENT_ROUTE_MODE", "ai-agent")
	t.Setenv("AI_AGENT_GRPC_ADDR", "dns:///ai-agent-service:50051")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AIAgentRouteMode != "ai-agent" {
		t.Fatalf("AIAgentRouteMode = %q, want ai-agent", cfg.AIAgentRouteMode)
	}
	if cfg.AIAgentGRPCAddr != "dns:///ai-agent-service:50051" {
		t.Fatalf("AIAgentGRPCAddr = %q", cfg.AIAgentGRPCAddr)
	}
}

func TestLoadAIAgentRouteRequiresAddress(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("AI_AGENT_ROUTE_MODE", "ai-agent")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "AI_AGENT_GRPC_ADDR") {
		t.Fatalf("expected ai agent addr error, got %v", err)
	}
}

func TestLoadRejectsInvalidAIAgentRouteMode(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("AI_AGENT_ROUTE_MODE", "invalid")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "AI_AGENT_ROUTE_MODE") {
		t.Fatalf("expected ai agent route mode error, got %v", err)
	}
}

func TestLoadNotificationRouteToExtractedService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("NOTIFICATION_ROUTE_MODE", "notification")
	t.Setenv("NOTIFICATION_GRPC_ADDR", "dns:///notification-service:50051")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.NotificationRouteMode != "notification" {
		t.Fatalf("NotificationRouteMode = %q, want notification", cfg.NotificationRouteMode)
	}
	if cfg.NotificationGRPCAddr != "dns:///notification-service:50051" {
		t.Fatalf("NotificationGRPCAddr = %q", cfg.NotificationGRPCAddr)
	}
}

func TestLoadNotificationRouteRequiresAddress(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("NOTIFICATION_ROUTE_MODE", "notification")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "NOTIFICATION_GRPC_ADDR") {
		t.Fatalf("expected notification addr error, got %v", err)
	}
}

func TestLoadRejectsInvalidNotificationRouteMode(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("NOTIFICATION_ROUTE_MODE", "invalid")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "NOTIFICATION_ROUTE_MODE") {
		t.Fatalf("expected notification route mode error, got %v", err)
	}
}

func TestLoadRankingConfigDefaultsEmpty(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	// 不设任何 RANKING_* env
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Ranking.WeightVector != 0 {
		t.Fatalf("WeightVector should be 0 (unset), got %v", cfg.Ranking.WeightVector)
	}
	if cfg.Ranking.BusinessBoostMax != 0 {
		t.Fatalf("BusinessBoostMax should be 0 (unset), got %v", cfg.Ranking.BusinessBoostMax)
	}
}

func TestLoadRankingConfigInvalidFloat(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("RANKING_WEIGHT_VECTOR", "not-a-number")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// 非法 env 值不修改 target，保持 0
	if cfg.Ranking.WeightVector != 0 {
		t.Fatalf("invalid env should keep default 0, got %v", cfg.Ranking.WeightVector)
	}
}

func TestLoadRequiresGRPCInternalTokenInProduction(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "GRPC_INTERNAL_TOKEN") {
		t.Fatalf("expected GRPC_INTERNAL_TOKEN error, got %v", err)
	}
}

func TestLoadRejectsPlaceholderGRPCInternalToken(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
	t.Setenv("GRPC_INTERNAL_TOKEN", "CHANGE_ME_INTERNAL_TOKEN_32_CHARS_LONG")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("expected placeholder token error, got %v", err)
	}
}

func TestLoadAllowsInsecureDevConfig(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "true")

	if _, err := Load(); err != nil {
		t.Fatalf("Load with dev bypass: %v", err)
	}
}
