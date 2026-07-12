package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.uber.org/zap"
)

func TestNewLoggerAndRedactSecrets(t *testing.T) {
	logger, err := NewLogger(LoggerConfig{ServiceName: "gateway", Env: "local", Level: "debug"})
	if err != nil {
		t.Fatalf("NewLogger returned error: %v", err)
	}
	defer func() { _ = logger.Sync() }()
	logger.Info("logger initialized", zap.String("component", "test"))

	redacted := RedactSecrets(map[string]string{
		"grpc_internal_token": "secret",
		"route_mode":          "logic",
	})
	if redacted["grpc_internal_token"] != "[REDACTED]" {
		t.Fatalf("token was not redacted: %#v", redacted)
	}
	if redacted["route_mode"] != "logic" {
		t.Fatalf("non-secret field changed: %#v", redacted)
	}
}

func TestMetricsRegistryUsesLowCardinalityLabels(t *testing.T) {
	registry, err := NewMetricsRegistry("gateway")
	if err != nil {
		t.Fatalf("NewMetricsRegistry returned error: %v", err)
	}
	counter, err := NewHTTPRequestsCounter(registry)
	if err != nil {
		t.Fatalf("NewHTTPRequestsCounter returned error: %v", err)
	}
	counter.WithLabelValues("gateway", "/api/v1/jobs", "GET", "200").Inc()
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather returned error: %v", err)
	}
	if !hasMetric(metrics, "smart_recruit_http_requests_total") {
		t.Fatalf("expected http request metric, got %#v", metrics)
	}
	if strings.Join(allowedHTTPLabels, ",") != "service,route,method,status" {
		t.Fatalf("unexpected labels: %v", allowedHTTPLabels)
	}
}

func TestTraceRuntimeShutdown(t *testing.T) {
	runtime, err := NewTraceRuntime(context.Background(), TraceConfig{
		ServiceName:    "identity",
		ServiceVersion: "dev",
		Env:            "local",
		SampleRatio:    0.5,
	})
	if err != nil {
		t.Fatalf("NewTraceRuntime returned error: %v", err)
	}
	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestHealthHandlers(t *testing.T) {
	registry := NewHealthRegistry()
	registry.Register("mysql", func(context.Context) error { return nil })
	registry.Register("rabbitmq", func(context.Context) error { return errors.New("unavailable") })

	live := httptest.NewRecorder()
	registry.LiveHandler().ServeHTTP(live, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if live.Code != http.StatusOK {
		t.Fatalf("unexpected live status: %d", live.Code)
	}

	ready := httptest.NewRecorder()
	registry.ReadyHandler().ServeHTTP(ready, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if ready.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected ready status: %d", ready.Code)
	}
	if !strings.Contains(ready.Body.String(), "rabbitmq") {
		t.Fatalf("expected failing dependency in response: %s", ready.Body.String())
	}
}

func hasMetric(metrics []*dto.MetricFamily, name string) bool {
	for _, metric := range metrics {
		if metric.GetName() == name {
			return true
		}
	}
	return false
}

var _ prometheus.Collector
