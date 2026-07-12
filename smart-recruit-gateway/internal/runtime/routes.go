package runtime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const LogicService = "logic"

var GatewayRouteServices = []string{
	"identity",
	"recruitment",
	"interview",
	"offer",
	"notification",
	"ai-agent",
	"analytics",
}

type TargetResolver interface {
	ResolveTarget(ctx context.Context, serviceName string) (string, error)
}

type StaticTargetResolver map[string]string

func (resolver StaticTargetResolver) ResolveTarget(_ context.Context, serviceName string) (string, error) {
	target := strings.TrimSpace(resolver[normalizeRouteName(serviceName)])
	if target == "" {
		return "", fmt.Errorf("%s static target is required", serviceName)
	}
	return target, nil
}

type RouteEntry struct {
	Service string
	Mode    string
	Target  string
}

type RouteTable struct {
	entries map[string]RouteEntry
}

type routeConfigFile struct {
	RouteModes map[string]string `yaml:"routeModes"`
}

func ParseRouteTable(data []byte) (RouteTable, error) {
	var file routeConfigFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return RouteTable{}, fmt.Errorf("parse gateway route config: %w", err)
	}
	return NewRouteTable(file.RouteModes)
}

func NewRouteTable(routeModes map[string]string) (RouteTable, error) {
	entries := make(map[string]RouteEntry, len(GatewayRouteServices))
	for _, service := range GatewayRouteServices {
		mode := LogicService
		for key, value := range routeModes {
			if normalizeRouteName(key) == service {
				mode = strings.TrimSpace(value)
				break
			}
		}
		mode = normalizeRouteName(mode)
		if mode == "" {
			mode = LogicService
		}
		if mode != LogicService && mode != service {
			return RouteTable{}, fmt.Errorf("%s route mode must be %s or %s", service, LogicService, service)
		}
		entries[service] = RouteEntry{Service: service, Mode: mode}
	}
	return RouteTable{entries: entries}, nil
}

func (table RouteTable) Entry(service string) (RouteEntry, bool) {
	entry, ok := table.entries[normalizeRouteName(service)]
	return entry, ok
}

func (table RouteTable) Entries() []RouteEntry {
	entries := make([]RouteEntry, 0, len(GatewayRouteServices))
	for _, service := range GatewayRouteServices {
		entry, ok := table.entries[service]
		if ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (table RouteTable) ResolveTargets(ctx context.Context, logicTarget string, resolver TargetResolver) (RouteTable, error) {
	logicTarget = strings.TrimSpace(logicTarget)
	if logicTarget == "" {
		return RouteTable{}, errors.New("logic target is required")
	}
	resolved := make(map[string]RouteEntry, len(table.entries))
	for _, entry := range table.Entries() {
		target := logicTarget
		if entry.Mode != LogicService {
			if resolver == nil {
				return RouteTable{}, fmt.Errorf("%s route mode requires discovery resolver", entry.Service)
			}
			discovered, err := resolver.ResolveTarget(ctx, entry.Service)
			if err != nil {
				return RouteTable{}, fmt.Errorf("resolve %s target: %w", entry.Service, err)
			}
			target = discovered
		}
		if strings.TrimSpace(target) == "" {
			return RouteTable{}, fmt.Errorf("%s target is empty", entry.Service)
		}
		entry.Target = target
		resolved[entry.Service] = entry
	}
	return RouteTable{entries: resolved}, nil
}

func (table RouteTable) ReadyTargets(logicTarget string) []RouteEntry {
	logicTarget = strings.TrimSpace(logicTarget)
	targets := []RouteEntry{{Service: LogicService, Mode: LogicService, Target: logicTarget}}
	for _, entry := range table.Entries() {
		if entry.Mode != LogicService {
			targets = append(targets, entry)
		}
	}
	sort.SliceStable(targets, func(i, j int) bool {
		return targets[i].Service < targets[j].Service
	})
	return targets
}

func normalizeRouteName(value string) string {
	normalized := strings.TrimSpace(value)
	normalized = strings.ReplaceAll(normalized, "_", "-")
	switch strings.ToLower(normalized) {
	case "aiagent", "ai-agent":
		return "ai-agent"
	default:
		return strings.ToLower(normalized)
	}
}
