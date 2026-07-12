package runtime

import (
	"context"
	"errors"
	"testing"
)

type fakeTargetResolver map[string]string

func (resolver fakeTargetResolver) ResolveTarget(_ context.Context, serviceName string) (string, error) {
	target, ok := resolver[serviceName]
	if !ok {
		return "", errors.New("missing target")
	}
	return target, nil
}

func TestRouteTableDefaultsEveryServiceToLogic(t *testing.T) {
	table, err := NewRouteTable(nil)
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	for _, service := range GatewayRouteServices {
		entry, ok := table.Entry(service)
		if !ok {
			t.Fatalf("missing route entry for %s", service)
		}
		if entry.Mode != LogicService {
			t.Fatalf("%s mode = %q, want logic", service, entry.Mode)
		}
	}
	targets, err := table.ResolveTargets(context.Background(), "logic:50051", nil)
	if err != nil {
		t.Fatalf("ResolveTargets returned error: %v", err)
	}
	for _, entry := range targets.Entries() {
		if entry.Target != "logic:50051" {
			t.Fatalf("%s target = %q, want logic:50051", entry.Service, entry.Target)
		}
	}
}

func TestParseRouteTableSupportsAllServiceModesAndRollback(t *testing.T) {
	table, err := ParseRouteTable([]byte(`
routeModes:
  identity: identity
  recruitment: recruitment
  interview: interview
  offer: offer
  notification: notification
  aiAgent: ai-agent
  analytics: analytics
`))
	if err != nil {
		t.Fatalf("ParseRouteTable returned error: %v", err)
	}
	resolved, err := table.ResolveTargets(context.Background(), "logic:50051", fakeTargetResolver{
		"identity":     "identity:50061",
		"recruitment":  "recruitment:50062",
		"interview":    "interview:50063",
		"offer":        "offer:50064",
		"notification": "notification:50065",
		"ai-agent":     "ai-agent:50066",
		"analytics":    "analytics:50067",
	})
	if err != nil {
		t.Fatalf("ResolveTargets returned error: %v", err)
	}
	for _, service := range GatewayRouteServices {
		entry, ok := resolved.Entry(service)
		if !ok {
			t.Fatalf("missing resolved entry for %s", service)
		}
		if entry.Mode != service {
			t.Fatalf("%s mode = %q, want %q", service, entry.Mode, service)
		}
		if entry.Target == "logic:50051" || entry.Target == "" {
			t.Fatalf("%s target was not discovered: %q", service, entry.Target)
		}
	}

	rollback, err := NewRouteTable(map[string]string{
		"identity":     "logic",
		"recruitment":  "logic",
		"interview":    "logic",
		"offer":        "logic",
		"notification": "logic",
		"aiAgent":      "logic",
		"analytics":    "logic",
	})
	if err != nil {
		t.Fatalf("rollback NewRouteTable returned error: %v", err)
	}
	targets, err := rollback.ResolveTargets(context.Background(), "logic:50051", nil)
	if err != nil {
		t.Fatalf("rollback ResolveTargets returned error: %v", err)
	}
	for _, entry := range targets.Entries() {
		if entry.Target != "logic:50051" {
			t.Fatalf("%s rollback target = %q, want logic", entry.Service, entry.Target)
		}
	}
}

func TestReadyTargetsCoverEnabledTargets(t *testing.T) {
	table, err := NewRouteTable(map[string]string{
		"identity":  "identity",
		"analytics": "analytics",
	})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	resolved, err := table.ResolveTargets(context.Background(), "logic:50051", fakeTargetResolver{
		"identity":  "identity:50061",
		"analytics": "analytics:50067",
	})
	if err != nil {
		t.Fatalf("ResolveTargets returned error: %v", err)
	}
	targets := resolved.ReadyTargets("logic:50051")
	seen := map[string]string{}
	for _, target := range targets {
		seen[target.Service] = target.Target
	}
	if seen[LogicService] != "logic:50051" {
		t.Fatalf("logic ready target = %q", seen[LogicService])
	}
	if seen["identity"] != "identity:50061" {
		t.Fatalf("identity ready target = %q", seen["identity"])
	}
	if seen["analytics"] != "analytics:50067" {
		t.Fatalf("analytics ready target = %q", seen["analytics"])
	}
	if _, ok := seen["offer"]; ok {
		t.Fatal("logic-mode offer should not add a separate ready target")
	}
}

func TestRouteTableRejectsInvalidModeAndMissingResolver(t *testing.T) {
	if _, err := NewRouteTable(map[string]string{"identity": "recruitment"}); err == nil {
		t.Fatal("expected invalid route mode error")
	}
	table, err := NewRouteTable(map[string]string{"identity": "identity"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	if _, err := table.ResolveTargets(context.Background(), "logic:50051", nil); err == nil {
		t.Fatal("expected missing resolver error")
	}
}

func TestStaticTargetResolverRequiresTarget(t *testing.T) {
	resolver := StaticTargetResolver{"ai-agent": "ai-agent:50066"}
	target, err := resolver.ResolveTarget(context.Background(), "aiAgent")
	if err != nil {
		t.Fatalf("ResolveTarget returned error: %v", err)
	}
	if target != "ai-agent:50066" {
		t.Fatalf("target = %q, want ai-agent:50066", target)
	}
	if _, err := resolver.ResolveTarget(context.Background(), "analytics"); err == nil {
		t.Fatal("expected missing analytics target error")
	}
}
