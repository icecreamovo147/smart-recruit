package domain

import (
	"reflect"
	"testing"

	"logic-grpc-service/pkg/authz"
)

func TestIdentityAuthzKeysMirrorCurrentCatalog(t *testing.T) {
	t.Parallel()

	if string(RoleCandidate) != authz.RoleCandidate {
		t.Fatalf("candidate role drifted: %q", RoleCandidate)
	}
	if string(RoleSystemAdmin) != authz.RoleSystemAdmin {
		t.Fatalf("system admin role drifted: %q", RoleSystemAdmin)
	}
	if string(ScopeOwnJobs) != authz.ScopeOwnJobs {
		t.Fatalf("own_jobs scope drifted: %q", ScopeOwnJobs)
	}
	if string(PermAdminRoleManage) != authz.PermAdminRoleManage {
		t.Fatalf("admin role permission drifted: %q", PermAdminRoleManage)
	}
	if string(PermAuditSecurityRead) != authz.PermAuditSecurityRead {
		t.Fatalf("security audit permission drifted: %q", PermAuditSecurityRead)
	}
}

func TestIdentityRoleGroupsMirrorCurrentCatalog(t *testing.T) {
	t.Parallel()

	toStrings := func(keys []RoleKey) []string {
		out := make([]string, 0, len(keys))
		for _, key := range keys {
			out = append(out, string(key))
		}
		return out
	}

	if got, want := toStrings(StaffRoles()), authz.StaffRoles(); !reflect.DeepEqual(got, want) {
		t.Fatalf("staff roles drifted: got %v want %v", got, want)
	}
	if got, want := toStrings(AdminRoles()), authz.AdminRoles(); !reflect.DeepEqual(got, want) {
		t.Fatalf("admin roles drifted: got %v want %v", got, want)
	}
}
