package infrastructure

import (
	"gorm.io/gorm"

	"logic-grpc-service/repository"
)

// UserRepository adapts the current GORM user repository to Identity application ports.
type UserRepository = repository.UserRepo

// RefreshTokenRepository adapts the current GORM refresh-token repository to Identity application ports.
type RefreshTokenRepository = repository.RefreshTokenRepo

// AuthorizationRepository adapts the current GORM RBAC repository to Identity application ports.
type AuthorizationRepository = repository.AuthzRepo

// NewUserRepository creates an Identity-owned user repository adapter.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return repository.NewUserRepo(db)
}

// NewRefreshTokenRepository creates an Identity-owned refresh-token repository adapter.
func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return repository.NewRefreshTokenRepo(db)
}

// NewAuthorizationRepository creates an Identity-owned authorization repository adapter.
func NewAuthorizationRepository(db *gorm.DB) *AuthorizationRepository {
	return repository.NewAuthzRepo(db)
}
