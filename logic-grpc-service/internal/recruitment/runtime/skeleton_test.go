package runtime

import (
	"strings"
	"testing"

	"logic-grpc-service/internal/platform/servicebinary"
)

func TestNewDescriptorIsUnroutedRecruitmentService(t *testing.T) {
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
		t.Fatal("recruitment skeleton must not enable traffic")
	}
	if descriptor.CutoverMode != "none" {
		t.Fatalf("cutover mode = %q, want none", descriptor.CutoverMode)
	}
	if len(descriptor.OwnedAPIs) == 0 {
		t.Fatal("recruitment descriptor should list owned APIs")
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
		"does not register generated gRPC services",
		"does not start recruitment lifecycle workers or consumers",
		"does not mutate jobs, candidates, resumes, applications, statuses, notifications, or analytics projections",
		"does not receive gateway traffic",
	} {
		if !strings.Contains(notes, required) {
			t.Fatalf("descriptor notes missing %q: %s", required, notes)
		}
	}
}

func TestDescriptorListsRecruitmentOwnedAPIs(t *testing.T) {
	descriptor, err := NewDescriptor()
	if err != nil {
		t.Fatalf("NewDescriptor() error = %v", err)
	}
	apis := strings.Join(descriptor.OwnedAPIs, "\n")
	for _, required := range []string{
		"JobService.CreateJob",
		"CandidateService.GetProfile",
		"ApplicationService.ApplyJob",
		"ApplicationService.UpdateApplicationStatus",
	} {
		if !strings.Contains(apis, required) {
			t.Fatalf("descriptor owned APIs missing %q: %s", required, apis)
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
		t.Fatal("Validate accepted traffic-enabled recruitment skeleton")
	}
}
