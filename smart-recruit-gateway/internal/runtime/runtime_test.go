package runtime

import (
	"context"
	"testing"
)

func TestGatewayUsesPlatformBootstrap(t *testing.T) {
	cfg, err := PlatformBootstrap()
	if err != nil {
		t.Fatalf("PlatformBootstrap returned error: %v", err)
	}
	if cfg.ServiceName != ServiceName {
		t.Fatalf("unexpected service name: %s", cfg.ServiceName)
	}
}

func TestGatewayReferencesUnifiedProtoContracts(t *testing.T) {
	names := ProtoServiceNames()
	if len(names) < 4 {
		t.Fatalf("expected proto service names, got %#v", names)
	}
	if names[0] != "recruitment.AuthService" {
		t.Fatalf("unexpected first proto service: %s", names[0])
	}
}

func TestGatewayTraceRuntimeInitializes(t *testing.T) {
	runtime, err := NewTraceRuntime(context.Background())
	if err != nil {
		t.Fatalf("NewTraceRuntime returned error: %v", err)
	}
	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}
