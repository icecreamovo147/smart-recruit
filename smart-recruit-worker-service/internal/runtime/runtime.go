package runtime

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

const ServiceName = "worker-service"

var DefaultWorkloads = []Workload{
	{Name: "outbox-dispatcher", Queue: "event_outbox", Idempotency: "outbox claim/mark-published status transition"},
	{Name: "notification-consumer", Queue: "RABBITMQ_NOTIFICATION_QUEUE", Idempotency: "notification inbox business key"},
	{Name: "email-consumer", Queue: "RABBITMQ_EMAIL_QUEUE", Idempotency: "email inbox and email log uniqueness"},
	{Name: "resume-parse-consumer", Queue: "RABBITMQ_RESUME_PARSE_QUEUE", Idempotency: "resume parse run and resume idempotency"},
	{Name: "embedding-consumer", Queue: "RABBITMQ_EMBEDDING_QUEUE", Idempotency: "embedding content hash and model key"},
	{Name: "agent-run-consumer", Queue: "RABBITMQ_AGENT_RUN_QUEUE", Idempotency: "agent run status transition guard"},
	{Name: "analytics-projection-consumer", Queue: "domain-event projections", Idempotency: "analytics projection event id ledger"},
}

type Workload struct {
	Name        string
	Queue       string
	Idempotency string
}

type WorkloadConfig struct {
	Enabled []string
}

func ParseWorkloadConfig(enabledList string, disabledList string) (WorkloadConfig, error) {
	known := knownWorkloadSet()
	enabled := parseList(enabledList)
	if len(enabled) == 0 {
		enabled = defaultEnabledWorkloads()
	}
	disabled := parseSet(disabledList)
	result := make([]string, 0, len(enabled))
	seen := map[string]struct{}{}
	for _, name := range enabled {
		if _, ok := known[name]; !ok {
			return WorkloadConfig{}, fmt.Errorf("unknown worker workload %q", name)
		}
		if _, off := disabled[name]; off {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	sort.Strings(result)
	return WorkloadConfig{Enabled: result}, nil
}

func (cfg WorkloadConfig) EnabledSet() map[string]struct{} {
	set := make(map[string]struct{}, len(cfg.Enabled))
	for _, name := range cfg.Enabled {
		set[name] = struct{}{}
	}
	return set
}

type Starter interface {
	Start(context.Context) error
}

type StarterFunc func(context.Context) error

func (fn StarterFunc) Start(ctx context.Context) error {
	return fn(ctx)
}

type DependencyStatus struct {
	RabbitMQ bool
	MySQL    bool
}

type Deps struct {
	Config   WorkloadConfig
	Starters map[string]Starter
	Status   func(context.Context) DependencyStatus
}

type Runtime struct {
	workloads []Workload
	profiles  map[string]WorkloadProfile
	enabled   map[string]struct{}
	starters  map[string]Starter
	status    func(context.Context) DependencyStatus
	mu        sync.Mutex
	started   []string
	cancel    context.CancelFunc
	stopped   bool
}

func New(deps Deps) (*Runtime, error) {
	if err := ValidateWorkloadProfiles(); err != nil {
		return nil, err
	}
	enabled := deps.Config.EnabledSet()
	if len(enabled) == 0 {
		return nil, fmt.Errorf("at least one worker workload must be enabled")
	}
	if deps.Status == nil {
		return nil, fmt.Errorf("worker dependency status checker is required")
	}
	known := knownWorkloadSet()
	for name := range enabled {
		workload, ok := known[name]
		if !ok {
			return nil, fmt.Errorf("unknown worker workload %q", name)
		}
		profile, ok := ProfileByName(name)
		if !ok {
			return nil, fmt.Errorf("worker workload %q profile is required", name)
		}
		if strings.TrimSpace(workload.Idempotency) == "" {
			return nil, fmt.Errorf("worker workload %q lacks idempotency policy", name)
		}
		if strings.TrimSpace(profile.Contract.OwnerContext) == "" || strings.TrimSpace(profile.Contract.Writes) == "" {
			return nil, fmt.Errorf("worker workload %q owner contract is incomplete", name)
		}
		if deps.Starters[name] == nil {
			return nil, fmt.Errorf("worker workload %q starter is required", name)
		}
	}
	return &Runtime{
		workloads: append([]Workload(nil), DefaultWorkloads...),
		profiles:  workloadProfileSet(),
		enabled:   enabled,
		starters:  deps.Starters,
		status:    deps.Status,
	}, nil
}

func (r *Runtime) Start(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("worker runtime is required")
	}
	if err := r.Ready(ctx); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	if r.cancel != nil {
		r.mu.Unlock()
		return fmt.Errorf("worker runtime already started")
	}
	r.cancel = cancel
	r.stopped = false
	r.mu.Unlock()
	for _, workload := range r.workloads {
		if _, ok := r.enabled[workload.Name]; !ok {
			continue
		}
		profile, ok := r.profiles[workload.Name]
		if !ok {
			return fmt.Errorf("worker workload %q profile is required", workload.Name)
		}
		if strings.TrimSpace(profile.Contract.OwnerContext) == "" {
			return fmt.Errorf("worker workload %q owner context is required", workload.Name)
		}
		starter := r.starters[workload.Name]
		if starter == nil {
			return fmt.Errorf("worker workload %q starter is required", workload.Name)
		}
		if err := starter.Start(runCtx); err != nil {
			return fmt.Errorf("start worker workload %s: %w", workload.Name, err)
		}
		r.mu.Lock()
		r.started = append(r.started, workload.Name)
		r.mu.Unlock()
	}
	return nil
}

type Stopper interface {
	Stop(context.Context) error
}

func (r *Runtime) Stop(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("worker runtime is required")
	}
	r.mu.Lock()
	cancel := r.cancel
	if r.stopped {
		r.mu.Unlock()
		return nil
	}
	r.stopped = true
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	for i := len(r.started) - 1; i >= 0; i-- {
		starter := r.starters[r.started[i]]
		stopper, ok := starter.(Stopper)
		if !ok {
			continue
		}
		if err := stopper.Stop(ctx); err != nil {
			return fmt.Errorf("stop worker workload %s: %w", r.started[i], err)
		}
	}
	return nil
}

func (r *Runtime) Ready(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("worker runtime is required")
	}
	status := r.status(ctx)
	if len(r.enabled) > 0 && !status.RabbitMQ {
		return fmt.Errorf("worker rabbitmq dependency is not ready")
	}
	if !status.MySQL {
		return fmt.Errorf("worker mysql dependency is not ready")
	}
	return nil
}

func (r *Runtime) StartedWorkloads() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	started := append([]string(nil), r.started...)
	sort.Strings(started)
	return started
}

func (r *Runtime) EnabledWorkloads() []Workload {
	workloads := make([]Workload, 0, len(r.enabled))
	for _, workload := range r.workloads {
		if _, ok := r.enabled[workload.Name]; ok {
			workloads = append(workloads, workload)
		}
	}
	return workloads
}

func (r *Runtime) EnabledProfiles() []WorkloadProfile {
	profiles := make([]WorkloadProfile, 0, len(r.enabled))
	for _, workload := range r.workloads {
		if _, ok := r.enabled[workload.Name]; !ok {
			continue
		}
		if profile, ok := r.profiles[workload.Name]; ok {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

func knownWorkloadSet() map[string]Workload {
	set := make(map[string]Workload, len(DefaultWorkloads))
	for _, workload := range DefaultWorkloads {
		set[workload.Name] = workload
	}
	return set
}

func workloadProfileSet() map[string]WorkloadProfile {
	set := make(map[string]WorkloadProfile, len(DefaultWorkloadProfiles))
	for _, profile := range DefaultWorkloadProfiles {
		set[profile.Name] = profile
	}
	return set
}

func defaultEnabledWorkloads() []string {
	result := make([]string, 0, len(DefaultWorkloads))
	for _, workload := range DefaultWorkloads {
		profile, ok := ProfileByName(workload.Name)
		if !ok || !profile.Toggle.DefaultOn {
			continue
		}
		result = append(result, workload.Name)
	}
	return result
}

func parseSet(value string) map[string]struct{} {
	items := parseList(value)
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}

func parseList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t' || r == ' '
	})
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
