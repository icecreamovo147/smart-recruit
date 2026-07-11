package runtime

import (
	"strings"
	"testing"

	"logic-grpc-service/internal/platform/servicebinary"
)

func TestNewDescriptorIsUnroutedWorkerServices(t *testing.T) {
	descriptor, err := NewDescriptor()
	if err != nil {
		t.Fatalf("NewDescriptor() error = %v", err)
	}
	if descriptor.Unit.Name != ServiceName {
		t.Fatalf("unit name = %q, want %q", descriptor.Unit.Name, ServiceName)
	}
	if descriptor.Unit.Role != servicebinary.RoleWorker {
		t.Fatalf("unit role = %q, want %q", descriptor.Unit.Role, servicebinary.RoleWorker)
	}
	if descriptor.TrafficEnabled {
		t.Fatal("worker-services skeleton must not enable traffic")
	}
	if descriptor.CutoverMode != "none" {
		t.Fatalf("cutover mode = %q, want none", descriptor.CutoverMode)
	}
}

func TestDescriptorListsExpectedWorkerWorkloads(t *testing.T) {
	descriptor, err := NewDescriptor()
	if err != nil {
		t.Fatalf("NewDescriptor() error = %v", err)
	}
	workloads := map[string]bool{}
	for _, workload := range descriptor.Workloads {
		workloads[workload.Name] = true
	}
	for _, required := range []string{
		"outbox-dispatcher",
		"notification-consumer",
		"email-consumer",
		"resume-parse-consumer",
		"embedding-consumer",
		"agent-run-consumer",
		"analytics-projection-consumer",
	} {
		if !workloads[required] {
			t.Fatalf("descriptor workloads missing %q: %+v", required, descriptor.Workloads)
		}
	}
}

func TestDescriptorDocumentsNoConsumerStartup(t *testing.T) {
	descriptor, err := NewDescriptor()
	if err != nil {
		t.Fatalf("NewDescriptor() error = %v", err)
	}
	notes := strings.Join(descriptor.Notes, "\n")
	for _, required := range []string{
		"does not start queue consumers or workers",
		"logic-grpc-service --worker-only remains the active worker deployment",
		"per-worker scaling and cutover",
	} {
		if !strings.Contains(notes, required) {
			t.Fatalf("descriptor notes missing %q: %s", required, notes)
		}
	}
}

func TestValidateRejectsDuplicateWorkloads(t *testing.T) {
	descriptor, err := NewDescriptor()
	if err != nil {
		t.Fatalf("NewDescriptor() error = %v", err)
	}
	descriptor.Workloads = append(descriptor.Workloads, descriptor.Workloads[0])
	if err := Validate(descriptor); err == nil {
		t.Fatal("Validate accepted duplicate worker workloads")
	}
}
