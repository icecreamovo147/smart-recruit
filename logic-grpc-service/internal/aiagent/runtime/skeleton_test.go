package runtime

import (
	"strings"
	"testing"

	"logic-grpc-service/internal/platform/servicebinary"
)

func TestNewDescriptorIsUnroutedAIAgentService(t *testing.T) {
	descriptor, err := NewDescriptor()
	if err != nil {
		t.Fatalf("NewDescriptor() error = %v", err)
	}
	if descriptor.Unit.Name != ServiceName {
		t.Fatalf("unit name = %q, want %q", descriptor.Unit.Name, ServiceName)
	}
	if descriptor.Unit.Role != servicebinary.RoleService {
		t.Fatalf("unit role = %q, want %q", descriptor.Unit.Role, servicebinary.RoleService)
	}
	if descriptor.TrafficEnabled {
		t.Fatal("AI Agent skeleton must not enable traffic")
	}
	if descriptor.CutoverMode != "none" {
		t.Fatalf("cutover mode = %q, want none", descriptor.CutoverMode)
	}
}

func TestDescriptorDocumentsNoRuntimeSideEffects(t *testing.T) {
	descriptor, err := NewDescriptor()
	if err != nil {
		t.Fatalf("NewDescriptor() error = %v", err)
	}
	notes := strings.Join(descriptor.Notes, "\n")
	for _, required := range []string{
		"does not bind a network listener",
		"does not start AI chat, agent-run, embedding, MCP, or memory runtime workers",
		"does not receive gateway traffic",
	} {
		if !strings.Contains(notes, required) {
			t.Fatalf("descriptor notes missing %q: %s", required, notes)
		}
	}
}

func TestValidateRejectsTrafficEnabledSkeleton(t *testing.T) {
	descriptor, err := NewDescriptor()
	if err != nil {
		t.Fatalf("NewDescriptor() error = %v", err)
	}
	descriptor.TrafficEnabled = true
	if err := Validate(descriptor); err == nil {
		t.Fatal("Validate accepted traffic-enabled AI Agent skeleton")
	}
}
