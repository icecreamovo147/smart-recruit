package main

import (
	"context"
	"testing"

	platformconfig "smart-recruit-platform-go/config"
)

func TestInstanceFromAddrUsesInterviewDiscoveryName(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50063", platformconfig.Bootstrap{
		ServiceName:    "interview-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	if instance.ServiceName != "interview" {
		t.Fatalf("ServiceName = %q, want interview", instance.ServiceName)
	}
	if instance.Port != 50063 {
		t.Fatalf("Port = %d, want 50063", instance.Port)
	}
	if instance.Metadata["service"] != "interview-service" {
		t.Fatalf("metadata service = %q", instance.Metadata["service"])
	}
}

func TestSetupNacosSupportsLocalStaticFallback(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50063", platformconfig.Bootstrap{
		ServiceName:    "interview-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	discovery, err := setupNacos(context.Background(), platformconfig.Bootstrap{
		ServiceName:    "interview-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
		NacosGroup:     "DEFAULT_GROUP",
		StaticFallback: true,
	}, instance)
	if err != nil {
		t.Fatalf("setupNacos returned error: %v", err)
	}
	instances, err := discovery.Resolve(context.Background(), "interview")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 50063 {
		t.Fatalf("unexpected instances: %#v", instances)
	}
}
