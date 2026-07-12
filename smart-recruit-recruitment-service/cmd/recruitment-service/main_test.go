package main

import (
	"context"
	"testing"

	"smart-recruit-platform-go/config"
)

func TestRecruitmentNacosStaticFallback(t *testing.T) {
	bootstrap := config.Bootstrap{ServiceName: "recruitment-service", ServiceEnv: "local", ServiceVersion: "test", StaticFallback: true}
	instance, err := instanceFromAddr("127.0.0.1:50062", bootstrap)
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	if instance.ServiceName != "recruitment" {
		t.Fatalf("ServiceName = %q, want recruitment", instance.ServiceName)
	}
	discovery, err := setupNacos(context.Background(), bootstrap, instance)
	if err != nil {
		t.Fatalf("setupNacos returned error: %v", err)
	}
	instances, err := discovery.Resolve(context.Background(), "recruitment")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 50062 {
		t.Fatalf("unexpected instances: %#v", instances)
	}
}
