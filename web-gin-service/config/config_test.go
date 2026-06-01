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
