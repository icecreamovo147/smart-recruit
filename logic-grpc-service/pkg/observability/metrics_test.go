package observability

import (
	"strings"
	"testing"
	"time"
)

func TestRegistryPrometheusIncludesRPCMetrics(t *testing.T) {
	registry := NewRegistry("test")
	registry.ObserveRPC("/recruitment.JobService/ListPublicJobs", "OK", 20*time.Millisecond)
	registry.RecordRPCPanic("/recruitment.JobService/ListPublicJobs")
	registry.RecordAuthRejection("/recruitment.JobService/CreateJob", "missing_or_invalid_token")

	output := registry.Prometheus()
	for _, want := range []string{
		"# TYPE smart_recruit_grpc_server_requests_total counter",
		`smart_recruit_grpc_server_requests_total{service="logic-grpc-service",grpc_method="/recruitment.JobService/ListPublicJobs",grpc_code="OK"} 1`,
		`smart_recruit_grpc_server_request_duration_seconds_bucket{service="logic-grpc-service",grpc_method="/recruitment.JobService/ListPublicJobs",grpc_code="OK",le="+Inf"} 1`,
		`smart_recruit_grpc_server_panics_total{service="logic-grpc-service",grpc_method="/recruitment.JobService/ListPublicJobs"} 1`,
		`smart_recruit_grpc_internal_auth_rejections_total{service="logic-grpc-service",grpc_method="/recruitment.JobService/CreateJob",reason="missing_or_invalid_token"} 1`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("Prometheus output missing %q:\n%s", want, output)
		}
	}
}
