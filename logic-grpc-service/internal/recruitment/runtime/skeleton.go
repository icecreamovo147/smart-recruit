package runtime

import (
	"fmt"

	"logic-grpc-service/internal/platform/servicebinary"
)

const (
	ServiceName = "recruitment-service"
	CutoverMode = "none"
)

type Descriptor struct {
	Unit           servicebinary.Unit
	CutoverMode    string
	TrafficEnabled bool
	StartupMode    string
	OwnedAPIs      []string
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
		StartupMode:    "skeleton-describe-only",
		OwnedAPIs: []string{
			"JobService.CreateJob",
			"JobService.UpdateJob",
			"JobService.OfflineJob",
			"JobService.OnlineJob",
			"JobService.ListHRJobs",
			"JobService.ListPublicJobs",
			"JobService.GetJobDetail",
			"CandidateService.GetProfile",
			"CandidateService.UpdateProfile",
			"CandidateService.GetResume",
			"CandidateService.PresignResumeUpload",
			"CandidateService.ConfirmResumeUpload",
			"ApplicationService.ApplyJob",
			"ApplicationService.ListMyApplications",
			"ApplicationService.ListJobApplications",
			"ApplicationService.UpdateApplicationStatus",
			"ApplicationService.ListApplicationStatusTransitions",
		},
		Notes: []string{
			"does not bind a network listener",
			"does not register generated gRPC services",
			"does not start recruitment lifecycle workers or consumers",
			"does not mutate jobs, candidates, resumes, applications, statuses, notifications, or analytics projections",
			"does not receive gateway traffic",
			"ready for later recruitment API extraction and controlled gateway cutover tasks",
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
	if len(descriptor.OwnedAPIs) == 0 {
		return fmt.Errorf("owned APIs are required")
	}
	if len(descriptor.Notes) == 0 {
		return fmt.Errorf("notes are required")
	}
	return nil
}
