package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"smart-recruit-platform-go/nacos"
)

func TestGatewayNacosRuntimeDiscoverySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/nacos/v1/cs/configs":
			_, _ = w.Write([]byte("routeModes:\n  identity: identity\n"))
		case "/nacos/v1/ns/instance/list":
			_, _ = w.Write([]byte(`{"hosts":[{"ip":"127.0.0.1","port":50061,"healthy":true}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	runtime, err := NewNacosRuntime(NacosOptions{Env: "prod", Addresses: server.URL})
	if err != nil {
		t.Fatalf("NewNacosRuntime returned error: %v", err)
	}
	config, err := runtime.LoadRequiredConfig(context.Background(), "gateway.yaml")
	if err != nil {
		t.Fatalf("LoadRequiredConfig returned error: %v", err)
	}
	if !strings.Contains(config, "identity") {
		t.Fatalf("unexpected config: %s", config)
	}
	target, err := runtime.ResolveTarget(context.Background(), "identity")
	if err != nil {
		t.Fatalf("ResolveTarget returned error: %v", err)
	}
	if target != "127.0.0.1:50061" {
		t.Fatalf("unexpected target: %s", target)
	}
}

func TestGatewayNacosRuntimeLocalFallback(t *testing.T) {
	runtime, err := NewNacosRuntime(NacosOptions{
		Env:              "local",
		AllowLocalStatic: true,
		StaticConfigs: map[string]string{
			"gateway.yaml": "routeModes:\n  identity: identity\n",
		},
		StaticTargets: map[string][]nacos.Instance{
			"identity": {{ServiceName: "identity", IP: "127.0.0.1", Port: 50051, Healthy: true}},
		},
	})
	if err != nil {
		t.Fatalf("NewNacosRuntime returned error: %v", err)
	}
	target, err := runtime.ResolveTarget(context.Background(), "identity")
	if err != nil {
		t.Fatalf("ResolveTarget returned error: %v", err)
	}
	if target != "127.0.0.1:50051" {
		t.Fatalf("unexpected fallback target: %s", target)
	}
}

func TestGatewayNacosRuntimeMissingConfigFails(t *testing.T) {
	_, err := NewNacosRuntime(NacosOptions{Env: "prod", AllowLocalStatic: true})
	if err == nil {
		t.Fatal("expected missing NACOS_ADDR error outside local")
	}
}
