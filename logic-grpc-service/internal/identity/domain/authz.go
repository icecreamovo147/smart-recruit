package domain

import "logic-grpc-service/pkg/authz"

// RoleKey identifies an Identity-owned RBAC role.
type RoleKey string

// PermissionKey identifies an Identity-owned permission.
type PermissionKey string

// ScopeKey identifies an Identity-owned data scope.
type ScopeKey string

// AccountType identifies the principal account boundary.
type AccountType string

// AuthorizationDecision is the persisted allow/deny decision vocabulary.
type AuthorizationDecision string

const (
	AccountTypeCandidate AccountType = "candidate"
	AccountTypeStaff     AccountType = "staff"
)

const (
	DecisionAllow AuthorizationDecision = "allow"
	DecisionDeny  AuthorizationDecision = "deny"
)

const (
	RoleCandidate       RoleKey = authz.RoleCandidate
	RoleRecruiter       RoleKey = authz.RoleRecruiter
	RoleRecruitingAdmin RoleKey = authz.RoleRecruitingAdmin
	RoleSystemAdmin     RoleKey = authz.RoleSystemAdmin
	RoleInterviewer     RoleKey = authz.RoleInterviewer
)

const (
	ScopeSelf               ScopeKey = authz.ScopeSelf
	ScopeAssignedInterviews ScopeKey = authz.ScopeAssignedInterviews
	ScopeOwnJobs            ScopeKey = authz.ScopeOwnJobs
	ScopeDepartment         ScopeKey = authz.ScopeDepartment
	ScopeLocation           ScopeKey = authz.ScopeLocation
	ScopeRecruitingAll      ScopeKey = authz.ScopeRecruitingAll
	ScopeSystemAll          ScopeKey = authz.ScopeSystemAll
)

const (
	PermAuthSessionRead            PermissionKey = authz.PermAuthSessionRead
	PermAdminInviteManage          PermissionKey = authz.PermAdminInviteManage
	PermAdminDepartmentManage      PermissionKey = authz.PermAdminDepartmentManage
	PermAdminLocationManage        PermissionKey = authz.PermAdminLocationManage
	PermAdminUserManage            PermissionKey = authz.PermAdminUserManage
	PermAdminRoleManage            PermissionKey = authz.PermAdminRoleManage
	PermAuditUsageRead             PermissionKey = authz.PermAuditUsageRead
	PermAuditSecurityRead          PermissionKey = authz.PermAuditSecurityRead
	PermSystemConfigManage         PermissionKey = authz.PermSystemConfigManage
	PermCandidateProfileManage     PermissionKey = authz.PermCandidateProfileManage
	PermCandidateResumeManage      PermissionKey = authz.PermCandidateResumeManage
	PermCandidateApplicationManage PermissionKey = authz.PermCandidateApplicationManage
)

// StaffRoles returns Identity-owned non-candidate role keys.
func StaffRoles() []RoleKey {
	roles := authz.StaffRoles()
	result := make([]RoleKey, 0, len(roles))
	for _, role := range roles {
		result = append(result, RoleKey(role))
	}
	return result
}

// AdminRoles returns Identity-owned administrative role keys.
func AdminRoles() []RoleKey {
	roles := authz.AdminRoles()
	result := make([]RoleKey, 0, len(roles))
	for _, role := range roles {
		result = append(result, RoleKey(role))
	}
	return result
}
