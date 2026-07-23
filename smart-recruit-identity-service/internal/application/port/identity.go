package port

import (
	"context"
	"time"
)

const (
	AccessTokenTTL  = 24 * time.Hour
	RefreshTokenTTL = 30 * 24 * time.Hour
)

type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

type PasswordVerifier interface {
	VerifyPassword(hash, password string) bool
}

type PasswordService interface {
	PasswordHasher
	PasswordVerifier
}

type TokenGenerator interface {
	NewRefreshToken() (string, error)
	NewFamilyID() (string, error)
}

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type ActorVerifier interface {
	VerifyActorMatch(ctx context.Context, userID int64) error
}

type AdminAuthorizer interface {
	AuthorizePermission(ctx context.Context, permissionKey string) error
}

type TokenVersionCache interface {
	SetTokenVersion(ctx context.Context, userID uint64, version int32, ttl time.Duration) error
	DeleteTokenVersion(ctx context.Context, userID uint64) error
}
