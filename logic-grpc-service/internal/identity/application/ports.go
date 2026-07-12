package application

import (
	"context"
	"time"

	"logic-grpc-service/model"
)

// UserRepository is the Identity-owned user persistence port.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByID(ctx context.Context, userID int64) (*model.User, error)
	UpdateAccountType(ctx context.Context, userID int64, accountType string) error
	UpdateEmail(ctx context.Context, userID int64, email string) error
	UpdateRole(ctx context.Context, userID int64, role int32) error
	UpdateStatus(ctx context.Context, userID int64, status string) error
	IncrementTokenVersion(ctx context.Context, userID int64) error
}

// RefreshTokenRepository is the Identity-owned refresh-token lifecycle port.
type RefreshTokenRepository interface {
	Create(ctx context.Context, userID int64, plainToken string, familyID string, expiresAt time.Time, ip, userAgent string) error
	Revoke(ctx context.Context, plainToken string) error
	RevokeFamily(ctx context.Context, familyID string) error
}

// AuthorizationRepository is the Identity-owned RBAC and scope persistence port.
type AuthorizationRepository interface {
	AssignRole(ctx context.Context, userID, roleID uint64, assignedBy *uint64) error
	AssignDataScope(ctx context.Context, userID uint64, scopeKey, resourceType string, resourceID uint64, assignedBy *uint64) error
	GetUserRoles(ctx context.Context, userID uint64) ([]string, error)
	GetUserPermissions(ctx context.Context, userID uint64) ([]string, error)
	GetUserDataScopes(ctx context.Context, userID uint64) ([]model.UserDataScope, error)
	IncrementTokenVersion(ctx context.Context, userID uint64) (int32, error)
	RecordAuthDecision(ctx context.Context, actorUserID uint64, actorRoles, permissionKey, resourceType string, resourceID uint64, decision, reason, requestID, clientIP string) error
}

// TokenVersionCache is the Identity-owned external cache side effect used after permission changes.
type TokenVersionCache interface {
	SetTokenVersion(ctx context.Context, userID uint64, version int32, ttl time.Duration) error
	DeleteTokenVersion(ctx context.Context, userID uint64) error
}
