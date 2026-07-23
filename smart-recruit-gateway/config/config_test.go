package config

import (
	"strings"
	"testing"
)

func testJWTSecret() string {
	return strings.Repeat("j", 32)
}

func TestLoadAuthCookieSecure(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
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

func TestLoadBillingReturnURLDefaultsMatchFrontendRoutes(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("BILLING_HR_RETURN_URL", "")
	t.Setenv("BILLING_CANDIDATE_RETURN_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BillingHRReturnURL != "http://localhost:5173/hr/billing" {
		t.Fatalf("BillingHRReturnURL = %q", cfg.BillingHRReturnURL)
	}
	if cfg.BillingCandidateReturnURL != "http://localhost:5174/billing" {
		t.Fatalf("BillingCandidateReturnURL = %q", cfg.BillingCandidateReturnURL)
	}
}

// TASK-FU-009: verify gateway ranking env loading stays aligned with backend scoring knobs.
func TestLoadRankingConfig(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
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

func TestLoadNotificationRouteDefaultsToService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.NotificationRouteMode != "notification" {
		t.Fatalf("NotificationRouteMode = %q, want notification", cfg.NotificationRouteMode)
	}
	if cfg.NotificationGRPCAddr != "127.0.0.1:50065" {
		t.Fatalf("NotificationGRPCAddr = %q, want 127.0.0.1:50065", cfg.NotificationGRPCAddr)
	}
}

func TestLoadAIAgentRouteDefaultsToService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AIAgentRouteMode != "ai-agent" {
		t.Fatalf("AIAgentRouteMode = %q, want ai-agent", cfg.AIAgentRouteMode)
	}
	if cfg.AIAgentGRPCAddr != "127.0.0.1:50066" {
		t.Fatalf("AIAgentGRPCAddr = %q, want 127.0.0.1:50066", cfg.AIAgentGRPCAddr)
	}
}

func TestLoadIdentityRouteDefaultsToService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IdentityRouteMode != "identity" {
		t.Fatalf("IdentityRouteMode = %q, want identity", cfg.IdentityRouteMode)
	}
	if cfg.IdentityGRPCAddr != "127.0.0.1:50061" {
		t.Fatalf("IdentityGRPCAddr = %q, want 127.0.0.1:50061", cfg.IdentityGRPCAddr)
	}
}

func TestLoadRecruitmentRouteDefaultsToService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RecruitmentRouteMode != "recruitment" {
		t.Fatalf("RecruitmentRouteMode = %q, want recruitment", cfg.RecruitmentRouteMode)
	}
	if cfg.RecruitmentGRPCAddr != "127.0.0.1:50062" {
		t.Fatalf("RecruitmentGRPCAddr = %q, want 127.0.0.1:50062", cfg.RecruitmentGRPCAddr)
	}
}

func TestLoadInterviewRouteDefaultsToService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.InterviewRouteMode != "interview" {
		t.Fatalf("InterviewRouteMode = %q, want interview", cfg.InterviewRouteMode)
	}
	if cfg.InterviewGRPCAddr != "127.0.0.1:50063" {
		t.Fatalf("InterviewGRPCAddr = %q, want 127.0.0.1:50063", cfg.InterviewGRPCAddr)
	}
}

func TestLoadOfferRouteDefaultsToService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OfferRouteMode != "offer" {
		t.Fatalf("OfferRouteMode = %q, want offer", cfg.OfferRouteMode)
	}
	if cfg.OfferGRPCAddr != "127.0.0.1:50064" {
		t.Fatalf("OfferGRPCAddr = %q, want 127.0.0.1:50064", cfg.OfferGRPCAddr)
	}
}

func TestLoadRecruitmentRouteToExtractedService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("RECRUITMENT_ROUTE_MODE", "recruitment")
	t.Setenv("RECRUITMENT_GRPC_ADDR", "dns:///recruitment-service:50051")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.RecruitmentRouteMode != "recruitment" {
		t.Fatalf("RecruitmentRouteMode = %q, want recruitment", cfg.RecruitmentRouteMode)
	}
	if cfg.RecruitmentGRPCAddr != "dns:///recruitment-service:50051" {
		t.Fatalf("RecruitmentGRPCAddr = %q", cfg.RecruitmentGRPCAddr)
	}
}

func TestLoadInterviewRouteToExtractedService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("INTERVIEW_ROUTE_MODE", "interview")
	t.Setenv("INTERVIEW_GRPC_ADDR", "dns:///interview-service:50051")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.InterviewRouteMode != "interview" {
		t.Fatalf("InterviewRouteMode = %q, want interview", cfg.InterviewRouteMode)
	}
	if cfg.InterviewGRPCAddr != "dns:///interview-service:50051" {
		t.Fatalf("InterviewGRPCAddr = %q", cfg.InterviewGRPCAddr)
	}
}

func TestLoadOfferRouteToExtractedService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("OFFER_ROUTE_MODE", "offer")
	t.Setenv("OFFER_GRPC_ADDR", "dns:///offer-service:50051")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OfferRouteMode != "offer" {
		t.Fatalf("OfferRouteMode = %q, want offer", cfg.OfferRouteMode)
	}
	if cfg.OfferGRPCAddr != "dns:///offer-service:50051" {
		t.Fatalf("OfferGRPCAddr = %q", cfg.OfferGRPCAddr)
	}
}

func TestLoadRejectsInvalidRecruitmentRouteMode(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("RECRUITMENT_ROUTE_MODE", "invalid")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "RECRUITMENT_ROUTE_MODE") {
		t.Fatalf("expected recruitment route mode error, got %v", err)
	}
}

func TestLoadRejectsInvalidInterviewRouteMode(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("INTERVIEW_ROUTE_MODE", "invalid")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "INTERVIEW_ROUTE_MODE") {
		t.Fatalf("expected interview route mode error, got %v", err)
	}
}

func TestLoadRejectsInvalidOfferRouteMode(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("OFFER_ROUTE_MODE", "invalid")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "OFFER_ROUTE_MODE") {
		t.Fatalf("expected offer route mode error, got %v", err)
	}
}

func TestLoadIdentityRouteToExtractedService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
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

func TestLoadRejectsInvalidIdentityRouteMode(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("IDENTITY_ROUTE_MODE", "invalid")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "IDENTITY_ROUTE_MODE") {
		t.Fatalf("expected identity route mode error, got %v", err)
	}
}

func TestLoadAIAgentRouteToExtractedService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
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

func TestLoadRejectsInvalidAIAgentRouteMode(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("AI_AGENT_ROUTE_MODE", "invalid")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "AI_AGENT_ROUTE_MODE") {
		t.Fatalf("expected ai agent route mode error, got %v", err)
	}
}

func TestLoadNotificationRouteToExtractedService(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
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

func TestLoadRejectsInvalidNotificationRouteMode(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("NOTIFICATION_ROUTE_MODE", "invalid")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "NOTIFICATION_ROUTE_MODE") {
		t.Fatalf("expected notification route mode error, got %v", err)
	}
}

func TestLoadRankingConfigDefaultsEmpty(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
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
	t.Setenv("JWT_SECRET", testJWTSecret())
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
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "GRPC_INTERNAL_TOKEN") {
		t.Fatalf("expected GRPC_INTERNAL_TOKEN error, got %v", err)
	}
}

func TestLoadRejectsPlaceholderGRPCInternalToken(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", "CHANGE_ME_INTERNAL_TOKEN_32_CHARS_LONG")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("expected placeholder token error, got %v", err)
	}
}

func TestLoadRequiresGRPCCAFileWhenInternalTLSRequired(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("GRPC_INTERNAL_TLS", "required")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "GRPC_TLS_CA_FILE") {
		t.Fatalf("expected GRPC_TLS_CA_FILE error, got %v", err)
	}
}

func TestLoadInternalTLSConfig(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "false")
	t.Setenv("JWT_SECRET", testJWTSecret())
	t.Setenv("GRPC_INTERNAL_TOKEN", strings.Repeat("t", 32))
	t.Setenv("GRPC_INTERNAL_TLS", "required")
	t.Setenv("GRPC_TLS_CA_FILE", "/etc/recruitment/tls/ca.crt")
	t.Setenv("GRPC_TLS_SERVER_NAME", "smart-recruit-identity-service.recruitment.svc.cluster.local")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.GRPCInternalTLS != "required" {
		t.Fatalf("GRPCInternalTLS = %q, want required", cfg.GRPCInternalTLS)
	}
	if cfg.GRPCTLSCAFile != "/etc/recruitment/tls/ca.crt" {
		t.Fatalf("GRPCTLSCAFile = %q", cfg.GRPCTLSCAFile)
	}
	if cfg.GRPCTLSServerName != "smart-recruit-identity-service.recruitment.svc.cluster.local" {
		t.Fatalf("GRPCTLSServerName = %q", cfg.GRPCTLSServerName)
	}
}

func TestLoadAllowsInsecureDevConfig(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEV_CONFIG", "true")

	if _, err := Load(); err != nil {
		t.Fatalf("Load with dev bypass: %v", err)
	}
}
