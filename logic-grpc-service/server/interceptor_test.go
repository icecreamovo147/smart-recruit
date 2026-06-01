package server

import "testing"

func TestAuthModeDefaultsToOptional(t *testing.T) {
	t.Setenv("GRPC_INTERNAL_AUTH", "")

	if got := authMode(); got != "optional" {
		t.Fatalf("authMode() = %q, want optional", got)
	}
}

func TestAuthModeRequired(t *testing.T) {
	t.Setenv("GRPC_INTERNAL_AUTH", "required")

	if got := authMode(); got != "required" {
		t.Fatalf("authMode() = %q, want required", got)
	}
}

func TestValidateInternalTokenRequiresToken(t *testing.T) {
	t.Setenv("GRPC_INTERNAL_AUTH", "required")
	t.Setenv("GRPC_INTERNAL_TOKEN", "")

	if err := ValidateInternalToken(); err == nil {
		t.Fatal("ValidateInternalToken() error = nil, want error")
	}
}

func TestValidateInternalTokenOptionalAllowsEmptyToken(t *testing.T) {
	t.Setenv("GRPC_INTERNAL_AUTH", "optional")
	t.Setenv("GRPC_INTERNAL_TOKEN", "")

	if err := ValidateInternalToken(); err != nil {
		t.Fatalf("ValidateInternalToken() error = %v, want nil", err)
	}
}
