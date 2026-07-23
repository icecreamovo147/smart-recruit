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

func TestRouteTableDefaultsEveryServiceToDirectMode(t *testing.T) {
	table, err := NewRouteTable(nil)
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	for _, service := range GatewayRouteServices {
		entry, ok := table.Entry(service)
		if !ok {
			t.Fatalf("missing route entry for %s", service)
		}
		if entry.Mode != service {
			t.Fatalf("%s mode = %q, want %q", service, entry.Mode, service)
		}
	}
}

func TestParseRouteTableResolvesAllDirectServiceTargets(t *testing.T) {
	table, err := ParseRouteTable([]byte(`
routeModes:
  identity: identity
  recruitment: recruitment
  interview: interview
  offer: offer
  notification: notification
  aiAgent: ai-agent
  analytics: analytics
  billing: billing
`))
	if err != nil {
		t.Fatalf("ParseRouteTable returned error: %v", err)
	}
	resolved, err := table.ResolveTargets(context.Background(), "recruitment:50062", fakeTargetResolver{
		"identity":     "identity:50061",
		"recruitment":  "recruitment:50062",
		"interview":    "interview:50063",
		"offer":        "offer:50064",
		"notification": "notification:50065",
		"ai-agent":     "ai-agent:50066",
		"analytics":    "analytics:50067",
		"billing":      "billing:50069",
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
		if entry.Target == "" {
			t.Fatalf("%s target was not discovered", service)
		}
	}
}

func TestReadyTargetsCoversOnlyDirectTargetsByDefault(t *testing.T) {
	table, err := NewRouteTable(map[string]string{
		"identity":  "identity",
		"analytics": "analytics",
		"billing":   "billing",
	})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	resolved, err := table.ResolveTargets(context.Background(), "recruitment:50062", fakeTargetResolver{
		"identity":     "identity:50061",
		"recruitment":  "recruitment:50062",
		"interview":    "interview:50063",
		"offer":        "offer:50064",
		"notification": "notification:50065",
		"ai-agent":     "ai-agent:50066",
		"analytics":    "analytics:50067",
		"billing":      "billing:50069",
	})
	if err != nil {
		t.Fatalf("ResolveTargets returned error: %v", err)
	}
	targets := resolved.ReadyTargets("recruitment:50062")
	seen := map[string]string{}
	for _, target := range targets {
		seen[target.Service] = target.Target
	}
	for _, service := range GatewayRouteServices {
		if seen[service] == "" {
			t.Fatalf("missing readiness target for %s", service)
		}
	}
	if len(seen) != len(GatewayRouteServices) {
		t.Fatalf("ready target count = %d, want %d", len(seen), len(GatewayRouteServices))
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
	if _, err := table.ResolveTargets(context.Background(), "recruitment:50062", nil); err == nil {
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
