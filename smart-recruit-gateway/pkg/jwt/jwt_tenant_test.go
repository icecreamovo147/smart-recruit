package jwt

import (
	"testing"
	"time"
)

func TestGenerateTenantFullRoundTripsTenantBinding(t *testing.T) {
	token, err := GenerateTenantFull(
		"tenant-test-secret", 42, "staff-user", 2, "staff",
		[]string{"recruiter"}, []string{"job.read"}, 7,
		11, 19, "staff", time.Minute,
	)
	if err != nil {
		t.Fatalf("GenerateTenantFull: %v", err)
	}
	claims, err := Parse(token, &Claims{}, []byte("tenant-test-secret"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.TenantID != 11 || claims.MembershipID != 19 || claims.ClientApp != "staff" {
		t.Fatalf("tenant binding = (%d, %d, %q)", claims.TenantID, claims.MembershipID, claims.ClientApp)
	}
	if claims.UserID != 42 || claims.TokenVersion != 7 {
		t.Fatalf("identity binding = (%d, %d)", claims.UserID, claims.TokenVersion)
	}
}

func TestGenerateFullKeepsGlobalSessionUnscoped(t *testing.T) {
	token, err := GenerateFull("global-test-secret", 8, "candidate", 1, "candidate", []string{"candidate"}, nil, 1, time.Minute)
	if err != nil {
		t.Fatalf("GenerateFull: %v", err)
	}
	claims, err := Parse(token, &Claims{}, []byte("global-test-secret"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.TenantID != 0 || claims.MembershipID != 0 || claims.ClientApp != "" {
		t.Fatalf("global token unexpectedly tenant-bound: %#v", claims)
	}
}
