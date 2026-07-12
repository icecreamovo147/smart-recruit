package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadWithLookupDefaultsAndValidates(t *testing.T) {
	cfg, err := LoadWithLookup(func(key string) string {
		values := map[string]string{
			"SERVICE_NAME":        "identity",
			"SERVICE_VERSION":     "1.2.3",
			"REQUEST_TIMEOUT":     "1500ms",
			"STATIC_FALLBACK":     "true",
			"GRPC_INTERNAL_TOKEN": "local-token",
		}
		return values[key]
	})
	if err != nil {
		t.Fatalf("LoadWithLookup returned error: %v", err)
	}
	if cfg.ServiceEnv != "local" {
		t.Fatalf("expected default env local, got %q", cfg.ServiceEnv)
	}
	if cfg.GRPCAddr != ":0" {
		t.Fatalf("expected default grpc addr :0, got %q", cfg.GRPCAddr)
	}
	if cfg.RequestTimeout != 1500*time.Millisecond {
		t.Fatalf("unexpected timeout: %s", cfg.RequestTimeout)
	}
	if !cfg.StaticFallback {
		t.Fatal("expected static fallback to parse true")
	}
}

func TestLoadWithLookupReportsMissingServiceName(t *testing.T) {
	_, err := LoadWithLookup(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected missing service name error")
	}
	if !strings.Contains(err.Error(), "SERVICE_NAME") {
		t.Fatalf("expected SERVICE_NAME in error, got %v", err)
	}
}

func TestValidateRejectsInvalidTimeout(t *testing.T) {
	err := (Bootstrap{
		ServiceName:    "gateway",
		ServiceEnv:     "local",
		ServiceVersion: "dev",
		GRPCAddr:       ":50051",
		RequestTimeout: 0,
	}).Validate()
	if err == nil {
		t.Fatal("expected invalid timeout error")
	}
}
