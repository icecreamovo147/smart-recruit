package observability

import (
	"strings"
	"testing"
	"time"
)

func TestRegistryPrometheusIncludesHTTPMetrics(t *testing.T) {
	registry := NewRegistry("test")
	registry.ObserveHTTPRequest("GET", "/readyz", 200, 15*time.Millisecond)
	registry.ObserveGRPCClient("/grpc.health.v1.Health/Check", "OK", 5*time.Millisecond)
	registry.RecordHTTPPanic("GET", "/panic")

	output := registry.Prometheus()
	for _, want := range []string{
		"# TYPE smart_recruit_http_requests_total counter",
		`smart_recruit_http_requests_total{service="web-gin-service",method="GET",route="/readyz",status="200"} 1`,
		`smart_recruit_http_request_duration_seconds_bucket{service="web-gin-service",method="GET",route="/readyz",status="200",le="+Inf"} 1`,
		`smart_recruit_http_panics_total{service="web-gin-service",method="GET",route="/panic"} 1`,
		`smart_recruit_grpc_client_requests_total{service="web-gin-service",grpc_method="/grpc.health.v1.Health/Check",grpc_code="OK"} 1`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("Prometheus output missing %q:\n%s", want, output)
		}
	}
}
