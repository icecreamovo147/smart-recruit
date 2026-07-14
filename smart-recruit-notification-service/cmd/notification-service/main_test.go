package main

import (
	"context"
	"testing"

	platformconfig "smart-recruit-platform-go/config"
	logicconfig "smart-recruit-platform-go/serviceconfig"
)

func TestInstanceFromAddrUsesNotificationDiscoveryName(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50065", platformconfig.Bootstrap{
		ServiceName:    "notification-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	if instance.ServiceName != "notification" {
		t.Fatalf("ServiceName = %q, want notification", instance.ServiceName)
	}
	if instance.Port != 50065 {
		t.Fatalf("Port = %d, want 50065", instance.Port)
	}
	if instance.Metadata["service"] != "notification-service" {
		t.Fatalf("metadata service = %q", instance.Metadata["service"])
	}
}

func TestSetupNacosSupportsLocalStaticFallback(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50065", platformconfig.Bootstrap{
		ServiceName:    "notification-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	discovery, err := setupNacos(context.Background(), platformconfig.Bootstrap{
		ServiceName:    "notification-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
		NacosGroup:     "DEFAULT_GROUP",
		StaticFallback: true,
	}, instance)
	if err != nil {
		t.Fatalf("setupNacos returned error: %v", err)
	}
	instances, err := discovery.Resolve(context.Background(), "notification")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 50065 {
		t.Fatalf("unexpected instances: %#v", instances)
	}
}

func TestMQConfigMapsNotificationOutboxInboxAndEmailQueues(t *testing.T) {
	var cfg logicconfig.Config
	cfg.RabbitMQ.URL = "amqp://example"
	cfg.RabbitMQ.Exchange = "events"
	cfg.RabbitMQ.DLXExchange = "events.dlx"
	cfg.RabbitMQ.RetryExchange = "events.retry"
	cfg.RabbitMQ.NotificationQueue = "notification.create"
	cfg.RabbitMQ.EmailQueue = "email.send"
	cfg.RabbitMQ.PrefetchCount = 7
	cfg.RabbitMQ.MaxRetries = 3

	mapped := mqConfig(cfg)
	if mapped.URL != "amqp://example" || mapped.Exchange != "events" {
		t.Fatalf("unexpected mq target mapping: %#v", mapped)
	}
	if mapped.NotificationQueue != "notification.create" {
		t.Fatalf("NotificationQueue = %q", mapped.NotificationQueue)
	}
	if mapped.EmailQueue != "email.send" {
		t.Fatalf("EmailQueue = %q", mapped.EmailQueue)
	}
	if mapped.PrefetchCount != 7 || mapped.MaxRetries != 3 {
		t.Fatalf("unexpected retry settings: %#v", mapped)
	}
}

func TestNotificationOutboxDispatcherEnabledDefaultsOff(t *testing.T) {
	t.Setenv("NOTIFICATION_OUTBOX_DISPATCHER_ENABLED", "")
	if notificationOutboxDispatcherEnabled() {
		t.Fatal("notification outbox dispatcher should be disabled by default")
	}

	t.Setenv("NOTIFICATION_OUTBOX_DISPATCHER_ENABLED", "true")
	if !notificationOutboxDispatcherEnabled() {
		t.Fatal("notification outbox dispatcher should be enabled when explicitly true")
	}

	t.Setenv("NOTIFICATION_OUTBOX_DISPATCHER_ENABLED", "false")
	if notificationOutboxDispatcherEnabled() {
		t.Fatal("notification outbox dispatcher should stay disabled unless explicitly true")
	}
}
