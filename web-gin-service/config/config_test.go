package config

import "testing"

func TestLoadAuthCookieSecure(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
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
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
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

func TestLoadRankingConfigDefaultsEmpty(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
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
	t.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long")
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
