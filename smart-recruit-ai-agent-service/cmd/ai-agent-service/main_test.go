package main

import (
	"context"
	"testing"

	logicconfig "logic-grpc-service/config"
	platformconfig "smart-recruit-platform-go/config"
)

func TestInstanceFromAddrUsesAIAgentDiscoveryName(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50066", platformconfig.Bootstrap{
		ServiceName:    "ai-agent-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	if instance.ServiceName != "ai-agent" {
		t.Fatalf("ServiceName = %q, want ai-agent", instance.ServiceName)
	}
	if instance.Port != 50066 {
		t.Fatalf("Port = %d, want 50066", instance.Port)
	}
	if instance.Metadata["service"] != "ai-agent-service" {
		t.Fatalf("metadata service = %q", instance.Metadata["service"])
	}
}

func TestSetupNacosSupportsLocalStaticFallback(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50066", platformconfig.Bootstrap{
		ServiceName:    "ai-agent-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	discovery, err := setupNacos(context.Background(), platformconfig.Bootstrap{
		ServiceName:    "ai-agent-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
		NacosGroup:     "DEFAULT_GROUP",
		StaticFallback: true,
	}, instance)
	if err != nil {
		t.Fatalf("setupNacos returned error: %v", err)
	}
	instances, err := discovery.Resolve(context.Background(), "ai-agent")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 50066 {
		t.Fatalf("unexpected instances: %#v", instances)
	}
}

func TestMQConfigMapsAIAgentWorkerQueues(t *testing.T) {
	var cfg logicconfig.Config
	cfg.RabbitMQ.URL = "amqp://example"
	cfg.RabbitMQ.Exchange = "events"
	cfg.RabbitMQ.EmbeddingQueue = "embedding.upsert"
	cfg.RabbitMQ.AgentRunQueue = "agent.run"
	cfg.RabbitMQ.PrefetchCount = 8
	cfg.RabbitMQ.MaxRetries = 4

	mapped := mqConfig(cfg)
	if mapped.URL != "amqp://example" || mapped.Exchange != "events" {
		t.Fatalf("unexpected mq target mapping: %#v", mapped)
	}
	if mapped.EmbeddingQueue != "embedding.upsert" {
		t.Fatalf("EmbeddingQueue = %q", mapped.EmbeddingQueue)
	}
	if mapped.AgentRunQueue != "agent.run" {
		t.Fatalf("AgentRunQueue = %q", mapped.AgentRunQueue)
	}
	if mapped.PrefetchCount != 8 || mapped.MaxRetries != 4 {
		t.Fatalf("unexpected retry settings: %#v", mapped)
	}
}
