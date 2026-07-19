package repository

import (
	"context"
	"errors"
	"time"

	"smart-recruit-identity-service/internal/domain/model"
)

var (
	ErrTokenNotFound      = errors.New("refresh token not found")
	ErrTokenExpired       = errors.New("refresh token expired")
	ErrTokenReuseDetected = errors.New("refresh token reuse detected")
	ErrLastAdmin          = errors.New("last system admin cannot be revoked")
	ErrUserRoleNotFound   = errors.New("user role not found")
)

type UserRepository interface {
	GetByUsername(context.Context, string) (*model.User, error)
	GetByID(context.Context, int64) (*model.User, error)
	Create(context.Context, *model.User) error
	UpdateEmail(context.Context, int64, string) error
	ListStaff(context.Context, int32, int32, string) ([]model.User, int64, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, userID int64, plainToken, familyID, clientApp string, tenantID, membershipID int64, expiresAt time.Time, ip, userAgent string) error
	GetActive(ctx context.Context, plainToken string) (*model.RefreshSession, error)
	Rotate(ctx context.Context, plainToken, newPlainToken string, newExpiresAt time.Time, newIP, newUserAgent string) (*model.RefreshSession, error)
	RotateToTenant(ctx context.Context, plainToken, newPlainToken string, tenantID, membershipID int64, newExpiresAt time.Time, newIP, newUserAgent string) (*model.RefreshSession, error)
	Revoke(ctx context.Context, plainToken string) error
}

type TenantRepository interface {
	GetByID(context.Context, int64) (*model.Tenant, error)
	GetDefault(context.Context) (*model.Tenant, error)
	Create(context.Context, *model.Tenant) error
	List(ctx context.Context, offset, limit int, keyword, status string) ([]model.Tenant, int64, error)
	UpdateStatus(ctx context.Context, tenantID int64, status string) (*model.Tenant, error)
	ListTenantMemberships(ctx context.Context, tenantID int64, offset, limit int) ([]model.TenantMembership, int64, error)
	ListUserMemberships(context.Context, int64) ([]model.TenantMembership, error)
	GetActiveMembership(ctx context.Context, userID, tenantID int64) (*model.TenantMembership, error)
	LoadTenantPrincipal(ctx context.Context, userID, tenantID int64, clientApp string) (*model.Principal, error)
	EnsureMembership(ctx context.Context, userID, tenantID int64, status string) (*model.TenantMembership, error)
	AssignMembershipRole(ctx context.Context, membershipID int64, roleID uint64, assignedBy *uint64) error
	RevokeMembershipRole(ctx context.Context, membershipID int64, roleID uint64) (bool, error)
	CountActiveTenantMembersWithRole(ctx context.Context, tenantID int64, roleID uint64) (int64, error)
	AssignMembershipDataScope(ctx context.Context, membershipID int64, scopeKey, resourceType string, resourceID uint64, assignedBy *uint64) error
	RevokeMembershipDataScope(ctx context.Context, membershipID, scopeID int64) (bool, error)
	RevokeTenantDataScope(ctx context.Context, tenantID, scopeID int64) (int64, bool, error)
}

type AuthzRepository interface {
	GetRoleByKey(context.Context, string) (*model.Role, error)
	ListRoles(context.Context) ([]model.Role, error)
	ListPermissions(context.Context) ([]model.Permission, error)
	GetUserRoles(context.Context, uint64) ([]string, error)
	GetUserPermissions(context.Context, uint64) ([]string, error)
	GetUserDataScopes(context.Context, uint64) ([]model.DataScope, error)
	LoadPrincipal(context.Context, uint64) (*model.Principal, error)
	LoadPlatformPrincipal(context.Context, uint64) (*model.Principal, error)
	AssignRole(ctx context.Context, userID, roleID uint64, assignedBy *uint64) error
	RevokeRoleWithLastAdminGuard(ctx context.Context, userID, roleID uint64, roleKey string, revokedBy *uint64) (bool, error)
	RestoreRole(ctx context.Context, userID, roleID uint64) error
	CountActiveUsersWithRole(context.Context, string) (int64, error)
	AssignDataScope(ctx context.Context, userID uint64, scopeKey, resourceType string, resourceID uint64, assignedBy *uint64) error
	RevokeDataScope(context.Context, uint64) error
	GetScopeOwnerID(context.Context, uint64) (uint64, error)
	IncrementTokenVersion(context.Context, uint64) (int32, error)
}

type InviteCodeRepository interface {
	GetByCode(context.Context, string) (*model.InviteCode, error)
}

type AuditRepository interface {
	RecordAuthDecision(context.Context, model.AuthAuditDecision) error
	QueryAuthAuditLogs(ctx context.Context, actorUserID *uint64, permissionKey, decision string, offset, limit int) ([]model.AuthAuditLog, int64, error)
}
