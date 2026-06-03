// Package authz defines role, permission, and scope constants for the RBAC system.
// These are stable string keys used for explicit authorization — never numeric ordering.
package authz

// ── Role keys ──────────────────────────────────────────────────────────

const (
	RoleCandidate      = "candidate"
	RoleRecruiter      = "recruiter"
	RoleRecruitingAdmin = "recruiting_admin"
	RoleSystemAdmin    = "system_admin"
	RoleInterviewer    = "interviewer"
)

// StaffRoles returns all staff roles (non-candidate).
func StaffRoles() []string {
	return []string{RoleRecruiter, RoleRecruitingAdmin, RoleSystemAdmin, RoleInterviewer}
}

// AdminRoles returns roles considered administrative.
func AdminRoles() []string {
	return []string{RoleRecruitingAdmin, RoleSystemAdmin}
}

// RoleDisplayNames maps role keys to human-readable Chinese names.
var RoleDisplayNames = map[string]string{
	RoleCandidate:       "求职者",
	RoleRecruiter:       "招聘专员",
	RoleRecruitingAdmin: "招聘管理员",
	RoleSystemAdmin:     "系统管理员",
	RoleInterviewer:     "面试官",
}

// ── Deprecated numeric role constants (compatibility only) ─────────────

const (
	LegacyRoleCandidate = 1
	LegacyRoleHR        = 2
	LegacyRoleAdmin     = 3
)

// LegacyRoleToKey maps legacy numeric roles to new role keys.
var LegacyRoleToKey = map[int32]string{
	LegacyRoleCandidate: RoleCandidate,
	LegacyRoleHR:        RoleRecruiter,
	LegacyRoleAdmin:     RoleRecruitingAdmin,
}
