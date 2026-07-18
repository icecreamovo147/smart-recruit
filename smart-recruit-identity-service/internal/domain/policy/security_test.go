package policy

import (
	"errors"
	"testing"

	"smart-recruit-identity-service/internal/domain/model"
)

func TestValidatePasswordCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "upper lower digit", password: "Password1", wantErr: nil},
		{name: "lower digit special", password: "password1!", wantErr: nil},
		{name: "too short", password: "Aa1!", wantErr: ErrPasswordTooShort},
		{name: "weak two classes", password: "password1", wantErr: ErrPasswordWeak},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidatePassword() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegistrationPlanForPreservesInviteSemantics(t *testing.T) {
	candidate, err := RegistrationPlanFor(model.LegacyRoleCandidate, "")
	if err != nil {
		t.Fatalf("candidate plan returned error: %v", err)
	}
	if candidate.AccountType != model.AccountTypeCandidate || candidate.RoleKey != model.RoleCandidate || candidate.RequiresInvite {
		t.Fatalf("unexpected candidate plan: %#v", candidate)
	}

	_, err = RegistrationPlanFor(model.LegacyRoleHR, "")
	if !errors.Is(err, ErrStaffInviteRequired) {
		t.Fatalf("staff without invite error = %v, want %v", err, ErrStaffInviteRequired)
	}

	staff, err := RegistrationPlanFor(model.LegacyRoleHR, "INVITE")
	if err != nil {
		t.Fatalf("staff plan returned error: %v", err)
	}
	if staff.AccountType != model.AccountTypeStaff || staff.RoleKey != model.RoleRecruiter || staff.ScopeKey != model.ScopeOwnJobs || !staff.RequiresInvite {
		t.Fatalf("unexpected staff plan: %#v", staff)
	}
}

func TestCheckSystemAdminRevocationGuards(t *testing.T) {
	admin := &model.Principal{UserID: 7, Roles: []string{model.RoleSystemAdmin}}
	err := CheckSystemAdminRevocation(7, 7, model.RoleSystemAdmin, admin, 2)
	if !errors.Is(err, ErrSelfSystemAdminRevoke) {
		t.Fatalf("self revoke error = %v, want %v", err, ErrSelfSystemAdminRevoke)
	}

	err = CheckSystemAdminRevocation(9, 7, model.RoleSystemAdmin, admin, 1)
	if !errors.Is(err, ErrLastSystemAdmin) {
		t.Fatalf("last admin error = %v, want %v", err, ErrLastSystemAdmin)
	}

	err = CheckSystemAdminRevocation(7, 7, model.RoleRecruiter, admin, 1)
	if err != nil {
		t.Fatalf("non-system role revoke returned error: %v", err)
	}
}
