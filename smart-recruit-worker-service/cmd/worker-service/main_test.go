package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	platformconfig "smart-recruit-platform-go/config"
	logicconfig "smart-recruit-platform-go/serviceconfig"
	workerruntime "smart-recruit-worker-service/internal/runtime"
)

func TestInstanceFromAddrUsesWorkerDiscoveryName(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50068", platformconfig.Bootstrap{
		ServiceName:    "worker-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	if instance.ServiceName != "worker" {
		t.Fatalf("ServiceName = %q, want worker", instance.ServiceName)
	}
	if instance.Port != 50068 {
		t.Fatalf("Port = %d, want 50068", instance.Port)
	}
	if instance.Metadata["service"] != "worker-service" {
		t.Fatalf("metadata service = %q", instance.Metadata["service"])
	}
}

func TestSetupNacosSupportsLocalStaticFallback(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50068", platformconfig.Bootstrap{
		ServiceName:    "worker-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	discovery, err := setupNacos(context.Background(), platformconfig.Bootstrap{
		ServiceName:    "worker-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
		NacosGroup:     "DEFAULT_GROUP",
		StaticFallback: true,
	}, instance)
	if err != nil {
		t.Fatalf("setupNacos returned error: %v", err)
	}
	instances, err := discovery.Resolve(context.Background(), "worker")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 50068 {
		t.Fatalf("unexpected instances: %#v", instances)
	}
}

func TestMQConfigMapsAllWorkerQueues(t *testing.T) {
	var cfg logicconfig.Config
	cfg.RabbitMQ.URL = "amqp://example"
	cfg.RabbitMQ.Exchange = "events"
	cfg.RabbitMQ.DLXExchange = "events.dlx"
	cfg.RabbitMQ.RetryExchange = "events.retry"
	cfg.RabbitMQ.NotificationQueue = "notification.create"
	cfg.RabbitMQ.ResumeParseQueue = "resume.parse"
	cfg.RabbitMQ.EmailQueue = "email.send"
	cfg.RabbitMQ.EmbeddingQueue = "embedding.upsert"
	cfg.RabbitMQ.AgentRunQueue = "agent.run"
	cfg.RabbitMQ.PrefetchCount = 7
	cfg.RabbitMQ.MaxRetries = 3

	mapped := mqConfig(cfg)
	if mapped.URL != "amqp://example" || mapped.Exchange != "events" {
		t.Fatalf("unexpected mq target mapping: %#v", mapped)
	}
	if mapped.NotificationQueue != "notification.create" ||
		mapped.ResumeParseQueue != "resume.parse" ||
		mapped.EmailQueue != "email.send" ||
		mapped.EmbeddingQueue != "embedding.upsert" ||
		mapped.AgentRunQueue != "agent.run" {
		t.Fatalf("unexpected worker queue mapping: %#v", mapped)
	}
	if mapped.PrefetchCount != 7 || mapped.MaxRetries != 3 {
		t.Fatalf("unexpected retry settings: %#v", mapped)
	}
}

func TestHealthMuxReportsReadiness(t *testing.T) {
	cfg, err := workerruntime.ParseWorkloadConfig("outbox-dispatcher", "")
	if err != nil {
		t.Fatalf("ParseWorkloadConfig returned error: %v", err)
	}
	runtime, err := workerruntime.New(workerruntime.Deps{
		Config:   cfg,
		Starters: controlledStarters(cfg.Enabled, nil),
		Status: func(context.Context) workerruntime.DependencyStatus {
			return workerruntime.DependencyStatus{RabbitMQ: true, MySQL: true}
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	healthMux(runtime).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz status = %d, body=%s", rec.Code, rec.Body.String())
	}
}
