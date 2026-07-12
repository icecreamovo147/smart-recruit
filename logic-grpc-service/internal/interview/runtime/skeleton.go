package runtime

import (
	"fmt"

	"logic-grpc-service/internal/platform/servicebinary"
)

const (
	ServiceName = "interview-service"
	CutoverMode = "none"
)

type Descriptor struct {
	Unit           servicebinary.Unit
	CutoverMode    string
	TrafficEnabled bool
	StartupMode    string
	ExtractedAPIs  []string
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
		ExtractedAPIs: []string{
			"InterviewService.ScheduleInterview",
			"InterviewService.UpdateInterview",
			"InterviewService.CancelInterview",
			"InterviewService.GetInterview",
			"InterviewService.ListInterviewers",
			"InterviewService.ListApplicationInterviews",
			"InterviewService.ListMyInterviews",
			"InterviewService.ListCandidateInterviews",
			"InterviewService.SubmitFeedback",
			"InterviewService.GetFeedback",
			"InterviewService.BatchCancelInterviews",
		},
		Notes: []string{
			"default execution does not bind a network listener",
			"InterviewService is registered only through explicit runtime registration",
			"does not start interview lifecycle workers or consumers",
			"does not mutate interview schedules, feedback, notifications, or application lifecycle state by default",
			"gateway traffic is not routed to interview-service in this TASK",
			"ready for later controlled interview gateway cutover with rollback evidence",
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
