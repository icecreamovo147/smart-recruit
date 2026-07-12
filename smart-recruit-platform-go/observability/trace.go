package observability

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type TraceConfig struct {
	ServiceName    string
	ServiceVersion string
	Env            string
	SampleRatio    float64
}

type TraceRuntime struct {
	Provider *sdktrace.TracerProvider
}

func NewTraceRuntime(ctx context.Context, cfg TraceConfig) (*TraceRuntime, error) {
	if strings.TrimSpace(cfg.ServiceName) == "" {
		return nil, errors.New("trace service name is required")
	}
	ratio := cfg.SampleRatio
	if ratio <= 0 || ratio > 1 {
		ratio = 1
	}
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(strings.TrimSpace(cfg.ServiceName)),
			semconv.ServiceVersion(strings.TrimSpace(cfg.ServiceVersion)),
			attribute.String("deployment.environment.name", strings.TrimSpace(cfg.Env)),
		),
	)
	if err != nil {
		return nil, err
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(ratio)),
	)
	otel.SetTracerProvider(provider)
	return &TraceRuntime{Provider: provider}, nil
}

func (runtime *TraceRuntime) Shutdown(ctx context.Context) error {
	if runtime == nil || runtime.Provider == nil {
		return nil
	}
	return runtime.Provider.Shutdown(ctx)
}
