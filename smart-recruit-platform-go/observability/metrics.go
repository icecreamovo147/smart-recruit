package observability

import (
	"errors"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var allowedHTTPLabels = []string{"service", "route", "method", "status"}

func NewMetricsRegistry(serviceName string) (*prometheus.Registry, error) {
	if strings.TrimSpace(serviceName) == "" {
		return nil, errors.New("metrics service name is required")
	}
	registry := prometheus.NewRegistry()
	buildInfo := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "smart_recruit_service_info",
		Help: "Smart Recruit service build information.",
	}, []string{"service"})
	buildInfo.WithLabelValues(strings.TrimSpace(serviceName)).Set(1)
	if err := registry.Register(buildInfo); err != nil {
		return nil, err
	}
	return registry, nil
}

func NewHTTPRequestsCounter(registry *prometheus.Registry) (*prometheus.CounterVec, error) {
	if registry == nil {
		return nil, errors.New("prometheus registry is required")
	}
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "smart_recruit_http_requests_total",
		Help: "Total HTTP requests by low-cardinality route, method, and status.",
	}, allowedHTTPLabels)
	if err := registry.Register(counter); err != nil {
		return nil, err
	}
	return counter, nil
}
