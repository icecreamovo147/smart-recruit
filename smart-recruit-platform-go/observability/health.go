package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"
)

type CheckFunc func(context.Context) error

type HealthRegistry struct {
	mu     sync.RWMutex
	checks map[string]CheckFunc
}

func NewHealthRegistry() *HealthRegistry {
	return &HealthRegistry{checks: map[string]CheckFunc{}}
}

func (registry *HealthRegistry) Register(name string, check CheckFunc) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.checks == nil {
		registry.checks = map[string]CheckFunc{}
	}
	registry.checks[name] = check
}

func (registry *HealthRegistry) Ready(ctx context.Context) map[string]string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	result := make(map[string]string, len(registry.checks))
	names := make([]string, 0, len(registry.checks))
	for name := range registry.checks {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := registry.checks[name](ctx); err != nil {
			result[name] = err.Error()
			continue
		}
		result[name] = "ok"
	}
	return result
}

func (registry *HealthRegistry) LiveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
}

func (registry *HealthRegistry) ReadyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := registry.Ready(r.Context())
		status := http.StatusOK
		for _, value := range result {
			if value != "ok" {
				status = http.StatusServiceUnavailable
				break
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(result)
	})
}
