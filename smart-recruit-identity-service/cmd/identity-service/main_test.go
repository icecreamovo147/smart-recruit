package main

import (
	"context"
	"testing"

	platformconfig "smart-recruit-platform-go/config"
)

func TestInstanceFromAddrUsesIdentityDiscoveryName(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50061", platformconfig.Bootstrap{
		ServiceName:    "identity-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	if instance.ServiceName != "identity" {
		t.Fatalf("ServiceName = %q, want identity", instance.ServiceName)
	}
	if instance.Port != 50061 {
		t.Fatalf("Port = %d, want 50061", instance.Port)
	}
	if instance.Metadata["service"] != "identity-service" {
		t.Fatalf("metadata service = %q", instance.Metadata["service"])
	}
}

func TestSetupNacosSupportsLocalStaticFallback(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50061", platformconfig.Bootstrap{
		ServiceName:    "identity-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	discovery, err := setupNacos(context.Background(), platformconfig.Bootstrap{
		ServiceName:    "identity-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
		NacosGroup:     "DEFAULT_GROUP",
		StaticFallback: true,
	}, instance)
	if err != nil {
		t.Fatalf("setupNacos returned error: %v", err)
	}
	instances, err := discovery.Resolve(context.Background(), "identity")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 50061 {
		t.Fatalf("unexpected instances: %#v", instances)
	}
}
