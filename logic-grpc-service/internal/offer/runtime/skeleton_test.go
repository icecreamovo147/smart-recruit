package runtime

import (
	"strings"
	"testing"

	"logic-grpc-service/internal/platform/servicebinary"
)

func TestNewDescriptorIsUnroutedOfferService(t *testing.T) {
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
		t.Fatal("offer skeleton must not enable traffic")
	}
	if descriptor.CutoverMode != "none" {
		t.Fatalf("cutover mode = %q, want none", descriptor.CutoverMode)
	}
	if len(descriptor.ExtractedAPIs) == 0 {
		t.Fatal("offer descriptor should list extracted APIs")
	}
}

func TestDescriptorDocumentsNoRuntimeSideEffects(t *testing.T) {
	descriptor, err := NewDescriptor()
	if err != nil {
		t.Fatalf("NewDescriptor() error = %v", err)
	}
	notes := strings.Join(descriptor.Notes, "\n")
	for _, required := range []string{
		"default execution does not bind a network listener",
		"registered only through explicit runtime registration",
		"does not start offer lifecycle workers or consumers",
		"gateway traffic is not routed to offer-service",
	} {
		if !strings.Contains(notes, required) {
			t.Fatalf("descriptor notes missing %q: %s", required, notes)
		}
	}
}

func TestDescriptorListsOfferExtractedAPIs(t *testing.T) {
	descriptor, err := NewDescriptor()
	if err != nil {
		t.Fatalf("NewDescriptor() error = %v", err)
	}
	apis := strings.Join(descriptor.ExtractedAPIs, "\n")
	for _, required := range []string{
		"OfferService.CreateOffer",
		"OfferService.SendOffer",
		"OfferService.AcceptOffer",
		"OfferService.ListOfferEvents",
	} {
		if !strings.Contains(apis, required) {
			t.Fatalf("descriptor extracted APIs missing %q: %s", required, apis)
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
		t.Fatal("Validate accepted traffic-enabled offer skeleton")
	}
}
