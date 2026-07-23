package model

import "time"

const (
	LegacyRoleCandidate int32 = 1
	LegacyRoleHR        int32 = 2
	LegacyRoleAdmin     int32 = 3

	AccountTypeCandidate = "candidate"
	AccountTypeStaff     = "staff"
	AccountTypePlatform  = "platform"
	AccountTypeService   = "service"

	UserStatusActive = "active"

	RoleCandidate        = "candidate"
	RoleRecruiter        = "recruiter"
	RoleRecruitingAdmin  = "recruiting_admin"
	RoleSystemAdmin      = "system_admin"
	RolePlatformAdmin    = "platform_admin"
	RolePlatformOperator = "platform_operator"
	RolePlatformAuditor  = "platform_auditor"
	RoleInterviewer      = "interviewer"

	RoleScopeIdentity = "identity"
	RoleScopeTenant   = "tenant"
	RoleScopePlatform = "platform"

	ScopeOwnJobs       = "own_jobs"
	ScopeRecruitingAll = "recruiting_all"
	ScopeSystemAll     = "system_all"

	PermAdminRoleManage   = "admin.role.manage"
	PermAdminUserManage   = "admin.user.manage"
	PermAuditSecurityRead = "audit.security.read"

	PermPlatformDashboardRead      = "platform.dashboard.read"
	PermPlatformTenantRead         = "platform.tenant.read"
	PermPlatformTenantManage       = "platform.tenant.manage"
	PermPlatformMemberManage       = "platform.member.manage"
	PermPlatformAuditRead          = "platform.audit.read"
	PermPlatformUserManage         = "platform.user.manage"
	PermPlatformPlanRead           = "platform.plan.read"
	PermPlatformPlanManage         = "platform.plan.manage"
	PermPlatformPlanPublish        = "platform.plan.publish"
	PermPlatformSubscriptionManage = "platform.subscription.manage"
	PermPlatformUsageRead          = "platform.usage.read"
	PermPlatformAlertRead          = "platform.alert.read"
	PermPlatformAlertManage        = "platform.alert.manage"
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

type PlatformAccount struct {
	User
	Roles []string
}

type Role struct {
	ID          uint64
	RoleKey     string
	Name        string
	Description string
	ScopeType   string
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
	ID           int64
	ScopeKey     string
	ResourceType string
	ResourceID   int64
	AssignedAt   time.Time
}

type Principal struct {
	UserID        int64
	Username      string
	AccountType   string
	Roles         []string
	Permissions   []string
	DataScopes    []ScopeAssignment
	TokenVersion  int32
	Email         string
	LegacyRole    int32
	TenantID      int64
	MembershipID  int64
	ClientApp     string
	AvailableApps []string
	Memberships   []TenantMembership
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
	TenantID  int64
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
	TenantID     int64
	MembershipID int64
	ClientApp    string
}

type Tenant struct {
	ID              int64
	TenantKey       string
	Slug            string
	Name            string
	Status          string
	Timezone        string
	Locale          string
	IsDefault       bool
	MembershipCount int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (t Tenant) IsActive() bool { return t.Status == "active" }

type TenantMembership struct {
	ID       int64
	TenantID int64
	UserID   int64
	Username string
	Email    string
	Status   string
	Tenant   Tenant
	Roles    []string
	JoinedAt *time.Time
}

type PlatformDashboard struct {
	TotalTenants        int64
	ActiveTenants       int64
	SuspendedTenants    int64
	DisabledTenants     int64
	NewTenants30D       int64
	TotalMemberships    int64
	ActiveMemberships   int64
	TenantsWithoutAdmin int64
}

type PlatformAuditLog struct {
	ID               uint64
	ActorUserID      int64
	ActorUsername    string
	Action           string
	ResourceType     string
	ResourceID       int64
	TargetTenantID   int64
	TargetTenantName string
	BeforeJSON       string
	AfterJSON        string
	RequestID        string
	ClientIP         string
	CreatedAt        time.Time
}

type PlatformAuditFilter struct {
	TenantID    int64
	ActorUserID int64
	Action      string
	RequestID   string
	StartTime   *time.Time
	EndTime     *time.Time
}

type PlatformEntitlement struct {
	Key             string
	ValueType       string
	ValueJSON       string
	EnforcementMode string
	Source          string
}

type PlatformPlanVersion struct {
	ID           int64
	PlanID       int64
	Version      int32
	Status       string
	EffectiveAt  *time.Time
	RetiredAt    *time.Time
	ChangeNote   string
	Entitlements []PlatformEntitlement
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PlatformPlan struct {
	ID          int64
	PlanKey     string
	Name        string
	Description string
	Status      string
	Versions    []PlatformPlanVersion
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TenantSubscription struct {
	ID                    int64
	TenantID              int64
	PlanVersionID         int64
	PlanKey               string
	PlanName              string
	PlanVersion           int32
	Status                string
	StartsAt              time.Time
	EndsAt                *time.Time
	Reason                string
	EffectiveEntitlements []PlatformEntitlement
}

type TenantUsageMetric struct {
	Key             string
	UsageValue      int64
	QuotaValue      int64
	UsagePercent    int32
	EnforcementMode string
	MeasuredAt      time.Time
}

type QuotaAlert struct {
	ID               int64
	TenantID         int64
	TenantName       string
	MetricKey        string
	ThresholdPercent int32
	UsageValue       int64
	QuotaValue       int64
	Status           string
	AssigneeUserID   int64
	AcknowledgedAt   *time.Time
	ResolvedAt       *time.Time
	ResolutionNote   string
	FirstTriggeredAt time.Time
	LastTriggeredAt  time.Time
}

type QuotaAlertFilter struct {
	TenantID  int64
	Status    string
	MetricKey string
}

func (m TenantMembership) IsActive() bool {
	return m.Status == "active" && m.Tenant.IsActive()
}

type AuthAuditDecision struct {
	TenantID      int64
	MembershipID  int64
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
	TenantID      int64
	MembershipID  int64
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
