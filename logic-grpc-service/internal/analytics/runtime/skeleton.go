package runtime

import (
	"fmt"

	"logic-grpc-service/internal/platform/servicebinary"
)

const (
	ServiceName = "analytics-service"
	CutoverMode = "none"
)

type Descriptor struct {
	Unit           servicebinary.Unit
	CutoverMode    string
	TrafficEnabled bool
	StartupMode    string
	ExtractedAPIs  []string
	ProjectionMode string
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
		StartupMode:    "explicit-runtime-registration-only",
		ProjectionMode: "domain-event-projection-read-models",
		ExtractedAPIs: []string{
			"AdminService.GetDashboardReport",
			"AdminService.GetFunnelReport",
			"AdminService.GetTimeInStageReport",
			"AdminService.GetInterviewOfferMetrics",
		},
		Notes: []string{
			"default execution does not bind a network listener",
			"Analytics reporting AdminService subset is registered only through explicit runtime registration",
			"uses Analytics projection/read-model ports and domain-event projection ingestion",
			"does not introduce transactional service-read adapters",
			"does not route gateway traffic to analytics-service in this TASK",
			"QueryAuthAuditLogs remains Identity-owned and is not part of this Analytics runtime",
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
	if descriptor.Unit.Role != servicebinary.RoleService {
		return fmt.Errorf("descriptor unit role = %q, want %q", descriptor.Unit.Role, servicebinary.RoleService)
	}
	if descriptor.TrafficEnabled {
		return fmt.Errorf("%s skeleton must not enable production traffic", ServiceName)
	}
	if descriptor.CutoverMode != CutoverMode {
		return fmt.Errorf("cutover mode = %q, want %q", descriptor.CutoverMode, CutoverMode)
	}
	if descriptor.ProjectionMode != "domain-event-projection-read-models" {
		return fmt.Errorf("projection mode = %q, want domain-event-projection-read-models", descriptor.ProjectionMode)
	}
	if descriptor.StartupMode == "" {
		return fmt.Errorf("startup mode is required")
	}
	if len(descriptor.ExtractedAPIs) == 0 {
		return fmt.Errorf("extracted APIs are required")
	}
	if len(descriptor.Notes) == 0 {
		return fmt.Errorf("notes are required")
	}
	return nil
}
