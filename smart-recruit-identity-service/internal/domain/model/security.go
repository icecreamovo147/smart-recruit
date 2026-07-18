package model

import "time"

const (
	LegacyRoleCandidate int32 = 1
	LegacyRoleHR        int32 = 2
	LegacyRoleAdmin     int32 = 3

	AccountTypeCandidate = "candidate"
	AccountTypeStaff     = "staff"

	UserStatusActive = "active"

	RoleCandidate       = "candidate"
	RoleRecruiter       = "recruiter"
	RoleRecruitingAdmin = "recruiting_admin"
	RoleSystemAdmin     = "system_admin"

	ScopeOwnJobs       = "own_jobs"
	ScopeRecruitingAll = "recruiting_all"
	ScopeSystemAll     = "system_all"

	PermAdminRoleManage   = "admin.role.manage"
	PermAdminUserManage   = "admin.user.manage"
	PermAuditSecurityRead = "audit.security.read"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         int32
	Email        string
	AccountType  string
	Status       string
	TokenVersion int32
	CreatedAt    time.Time
}

type Role struct {
	ID          uint64
	RoleKey     string
	Name        string
	Description string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Permission struct {
	ID            uint64
	PermissionKey string
	Resource      string
	Action        string
	Description   string
}

type DataScope struct {
	ID           uint64
	UserID       uint64
	ScopeKey     string
	ResourceType string
	ResourceID   uint64
	AssignedBy   *uint64
	AssignedAt   time.Time
	RevokedAt    *time.Time
}

type ScopeAssignment struct {
	ScopeKey     string
	ResourceType string
	ResourceID   int64
}

type Principal struct {
	UserID       int64
	Username     string
	AccountType  string
	Roles        []string
	Permissions  []string
	DataScopes   []ScopeAssignment
	TokenVersion int32
	Email        string
	LegacyRole   int32
}

func (p *Principal) HasRole(roleKey string) bool {
	if p == nil {
		return false
	}
	for _, role := range p.Roles {
		if role == roleKey {
			return true
		}
	}
	return false
}

func (p *Principal) HasPermission(permissionKey string) bool {
	if p == nil {
		return false
	}
	for _, permission := range p.Permissions {
		if permission == permissionKey {
			return true
		}
	}
	return false
}

func (p *Principal) HasScope(scopeKey string) bool {
	if p == nil {
		return false
	}
	for _, scope := range p.DataScopes {
		if scope.ScopeKey == scopeKey {
			return true
		}
	}
	return false
}

func (p *Principal) IsStaff() bool {
	return p != nil && p.AccountType == AccountTypeStaff
}

func (p *Principal) IsCandidate() bool {
	return p != nil && p.AccountType == AccountTypeCandidate
}

type InviteCode struct {
	ID        int64
	Code      string
	CreatedBy int64
	IsActive  bool
	ExpiresAt *time.Time
}

func (c InviteCode) UsableAt(now time.Time) bool {
	if !c.IsActive {
		return false
	}
	return c.ExpiresAt == nil || c.ExpiresAt.After(now)
}

type RefreshSession struct {
	UserID       int64
	Username     string
	Role         int32
	AccountType  string
	TokenVersion int32
	FamilyID     string
}

type AuthAuditDecision struct {
	ActorUserID   uint64
	ActorRoles    string
	PermissionKey string
	ResourceType  string
	ResourceID    uint64
	Decision      string
	Reason        string
	RequestID     string
	ClientIP      string
}

type AuthAuditLog struct {
	ID            uint64
	ActorUserID   uint64
	ActorRoles    string
	PermissionKey string
	ResourceType  string
	ResourceID    uint64
	Decision      string
	Reason        string
	RequestID     string
	ClientIP      string
	CreatedAt     time.Time
}
