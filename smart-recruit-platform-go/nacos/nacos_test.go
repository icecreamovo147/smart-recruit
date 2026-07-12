package nacos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAddressesDefaultsSchemeAndPort(t *testing.T) {
	addresses, err := ParseAddresses("127.0.0.1,https://nacos.example.com:9848/base")
	if err != nil {
		t.Fatalf("ParseAddresses returned error: %v", err)
	}
	if got := addresses[0].String(); got != "http://127.0.0.1:8848" {
		t.Fatalf("unexpected default address: %s", got)
	}
	if got := addresses[1].String(); got != "https://nacos.example.com:9848/base" {
		t.Fatalf("unexpected explicit address: %s", got)
	}
}

func TestConfigProviderFailFastOutsideLocalFallback(t *testing.T) {
	_, err := NewConfigProvider(ConfigOptions{
		Env:           "prod",
		AllowFallback: true,
		StaticFallback: map[string]string{
			"gateway.yaml": "routeMode: logic",
		},
	})
	if err == nil {
		t.Fatal("expected fail-fast without NACOS_ADDR outside local")
	}
}

func TestStaticConfigProviderFallback(t *testing.T) {
	provider, err := NewConfigProvider(ConfigOptions{
		Env:           "local",
		AllowFallback: true,
		StaticFallback: map[string]string{
			"gateway.yaml": "routeMode: logic",
		},
	})
	if err != nil {
		t.Fatalf("NewConfigProvider returned error: %v", err)
	}
	value, err := provider.Load(context.Background(), "gateway.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if value != "routeMode: logic" {
		t.Fatalf("unexpected fallback value: %q", value)
	}
}

func TestHTTPConfigProviderLoadsFromNacosAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nacos/v1/cs/configs" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("dataId") != "gateway.yaml" {
			t.Fatalf("unexpected dataId: %s", r.URL.Query().Get("dataId"))
		}
		_, _ = w.Write([]byte("routeMode: logic"))
	}))
	defer server.Close()

	provider, err := NewConfigProvider(ConfigOptions{Addresses: server.URL})
	if err != nil {
		t.Fatalf("NewConfigProvider returned error: %v", err)
	}
	value, err := provider.Load(context.Background(), "gateway.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if value != "routeMode: logic" {
		t.Fatalf("unexpected config value: %q", value)
	}
}

func TestStaticDiscoveryFallback(t *testing.T) {
	discovery, err := NewDiscovery(DiscoveryOptions{
		Env:           "local",
		AllowFallback: true,
		StaticFallback: map[string][]Instance{
			"identity": {{ServiceName: "identity", IP: "127.0.0.1", Port: 50061, Healthy: true}},
		},
	})
	if err != nil {
		t.Fatalf("NewDiscovery returned error: %v", err)
	}
	instances, err := discovery.Resolve(context.Background(), "identity")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 50061 {
		t.Fatalf("unexpected instances: %#v", instances)
	}
}

func TestHTTPDiscoveryResolvesHealthyInstances(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nacos/v1/ns/instance/list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"hosts":[{"ip":"127.0.0.1","port":50061,"healthy":true},{"ip":"127.0.0.2","port":50062,"healthy":false}]}`))
	}))
	defer server.Close()

	discovery, err := NewDiscovery(DiscoveryOptions{Addresses: server.URL})
	if err != nil {
		t.Fatalf("NewDiscovery returned error: %v", err)
	}
	instances, err := discovery.Resolve(context.Background(), "identity")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(instances) != 1 || instances[0].IP != "127.0.0.1" {
		t.Fatalf("unexpected instances: %#v", instances)
	}
}
