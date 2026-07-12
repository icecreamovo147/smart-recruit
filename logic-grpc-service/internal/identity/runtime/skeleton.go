package runtime

import (
	"fmt"

	"logic-grpc-service/internal/platform/servicebinary"
)

const (
	ServiceName = "identity-service"
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
		StartupMode:    "explicit-serve-only",
		ExtractedAPIs: []string{
			"AuthService.Register",
			"AuthService.Login",
			"AuthService.RefreshToken",
			"AuthService.RevokeRefreshToken",
			"AuthService.RecordAuthDecision",
			"AuthService.GetPrincipal",
			"AuthService.UpdateEmail",
			"AdminService.ListRoles",
			"AdminService.ListPermissions",
			"AdminService.GetUserRoles",
			"AdminService.AssignUserRole",
			"AdminService.RevokeUserRole",
			"AdminService.AssignDataScope",
			"AdminService.RevokeDataScope",
			"AdminService.ListStaffUsers",
			"AdminService.CreateStaffUser",
			"AdminService.QueryAuthAuditLogs",
		},
		Notes: []string{
			"default execution does not bind a network listener",
			"auth, principal, RBAC, scope, staff identity, and security-audit APIs are registered only in explicit serve mode",
			"gateway traffic is not routed to identity-service in this TASK",
			"non-Identity AdminService methods remain on the monolith gateway target",
			"ready for later controlled identity gateway cutover with rollback evidence",
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
