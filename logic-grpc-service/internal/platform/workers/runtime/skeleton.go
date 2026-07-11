package runtime

import (
	"fmt"

	"logic-grpc-service/internal/platform/servicebinary"
)

const (
	ServiceName = "worker-services"
	CutoverMode = "none"
)

type Workload struct {
	Name        string
	Owner       string
	Queue       string
	Description string
}

type Descriptor struct {
	Unit           servicebinary.Unit
	CutoverMode    string
	TrafficEnabled bool
	StartupMode    string
	Workloads      []Workload
	Notes          []string
}

func NewDescriptor() (Descriptor, error) {
	unit, ok := servicebinary.Find(ServiceName)
	if !ok {
		return Descriptor{}, fmt.Errorf("service binary unit %q is not registered", ServiceName)
	}
	descriptor := Descriptor{
		Unit:           unit,
		CutoverMode:    CutoverMode,
		TrafficEnabled: false,
		StartupMode:    "descriptor-only-no-consumer-start",
		Workloads: []Workload{
			{Name: "outbox-dispatcher", Owner: "platform-workers", Queue: "event_outbox", Description: "publishes durable domain and notification events"},
			{Name: "notification-consumer", Owner: "notification", Queue: "RABBITMQ_NOTIFICATION_QUEUE", Description: "creates notification records from events"},
			{Name: "email-consumer", Owner: "notification", Queue: "RABBITMQ_EMAIL_QUEUE", Description: "dispatches email outbox work"},
			{Name: "resume-parse-consumer", Owner: "recruitment", Queue: "RABBITMQ_RESUME_PARSE_QUEUE", Description: "parses uploaded resumes"},
			{Name: "embedding-consumer", Owner: "ai-agent", Queue: "RABBITMQ_EMBEDDING_QUEUE", Description: "runs embedding workloads"},
			{Name: "agent-run-consumer", Owner: "ai-agent", Queue: "RABBITMQ_AGENT_RUN_QUEUE", Description: "executes durable AI agent runs"},
			{Name: "analytics-projection-consumer", Owner: "analytics", Queue: "domain-event projections", Description: "projects domain events into Analytics read models"},
		},
		Notes: []string{
			"default execution does not bind a network listener",
			"default execution does not start queue consumers or workers",
			"logic-grpc-service --worker-only remains the active worker deployment",
			"workloads are independently named for later per-worker scaling and cutover",
			"cutover requires scoped readiness, metrics, queue ownership, and rollback evidence",
		},
	}
	if err := Validate(descriptor); err != nil {
		return Descriptor{}, err
	}
	return descriptor, nil
}

func Validate(descriptor Descriptor) error {
	if descriptor.Unit.Name != ServiceName {
		return fmt.Errorf("descriptor unit name = %q, want %q", descriptor.Unit.Name, ServiceName)
	}
	if descriptor.Unit.Role != servicebinary.RoleWorker {
		return fmt.Errorf("descriptor unit role = %q, want %q", descriptor.Unit.Role, servicebinary.RoleWorker)
	}
	if descriptor.TrafficEnabled {
		return fmt.Errorf("%s skeleton must not enable production traffic", ServiceName)
	}
	if descriptor.CutoverMode != CutoverMode {
		return fmt.Errorf("cutover mode = %q, want %q", descriptor.CutoverMode, CutoverMode)
	}
	if descriptor.StartupMode == "" {
		return fmt.Errorf("startup mode is required")
	}
	if len(descriptor.Workloads) == 0 {
		return fmt.Errorf("worker workloads are required")
	}
	seen := map[string]struct{}{}
	for _, workload := range descriptor.Workloads {
		if workload.Name == "" || workload.Owner == "" || workload.Queue == "" {
			return fmt.Errorf("worker workload has empty required fields: %+v", workload)
		}
		if _, exists := seen[workload.Name]; exists {
			return fmt.Errorf("duplicate worker workload %q", workload.Name)
		}
		seen[workload.Name] = struct{}{}
	}
	if len(descriptor.Notes) == 0 {
		return fmt.Errorf("notes are required")
	}
	return nil
}
