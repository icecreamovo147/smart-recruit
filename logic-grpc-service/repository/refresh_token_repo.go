package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"logic-grpc-service/model"
)

// ErrTokenNotFound is returned when the refresh token does not exist.
var ErrTokenNotFound = errors.New("refresh token not found")

// ErrTokenExpired is returned when the refresh token has expired.
var ErrTokenExpired = errors.New("refresh token expired")

// ErrTokenRevoked is returned when the refresh token has been revoked.
var ErrTokenRevoked = errors.New("refresh token revoked")

// ErrTokenReuseDetected is returned when a revoked token is reused — potential attack.
var ErrTokenReuseDetected = errors.New("refresh token reuse detected")

// RefreshTokenResult carries the identity and authorization metadata of the user
// associated with a valid refresh token.
type RefreshTokenResult struct {
	UserID       int64
	Username     string
	Role         int32    // Deprecated: kept for compatibility
	AccountType  string
	Roles        []string
	Permissions  []string
	TokenVersion int32
	FamilyID     string
}

// RefreshTokenRepo manages opaque refresh token lifecycle.
// Tokens are stored as sha256 hashes; plain-text tokens never leave the caller.
type RefreshTokenRepo struct {
	db *gorm.DB
}

// NewRefreshTokenRepo creates a new RefreshTokenRepo.
func NewRefreshTokenRepo(db *gorm.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

// TokenHash returns the sha256 hex of a plain-text token.
func TokenHash(plain string) string {
	h := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(h[:])
}

// Create stores a new refresh token record, returning the plain-text token and its expiry.
func (r *RefreshTokenRepo) Create(ctx context.Context, userID int64, plainToken string, familyID string, expiresAt time.Time, ip, userAgent string) error {
	rt := &model.RefreshToken{
		UserID:           userID,
		TokenHash:        TokenHash(plainToken),
		FamilyID:         familyID,
		ExpiresAt:        expiresAt,
		CreatedIP:        &ip,
		CreatedUserAgent: &userAgent,
	}
	return r.db.WithContext(ctx).Create(rt).Error
}

// Rotate exchanges a valid refresh token for a new one within the same family.
// It runs inside a DB transaction with row locking to prevent concurrent rotation.
// If the token has already been revoked, returns ErrTokenReuseDetected and invalidates the family.
func (r *RefreshTokenRepo) Rotate(ctx context.Context, plainToken string, newPlainToken string, newExpiresAt time.Time, newIP, newUserAgent string) (*RefreshTokenResult, error) {
	var result *RefreshTokenResult
	reuseDetected := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		hash := TokenHash(plainToken)
		var rt model.RefreshToken
		// SELECT FOR UPDATE
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ?", hash).First(&rt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTokenNotFound
			}
			return err
		}

		// Check revocation before expiry so reuse detection always wins for
		// already-rotated tokens, even after the old token's expiry time.
		if rt.RevokedAt != nil {
			// Token reuse detected — invalidate the entire family
			now := time.Now()
			if err := tx.Model(&model.RefreshToken{}).
				Where("id = ?", rt.ID).
				Update("reuse_detected_at", now).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.RefreshToken{}).
				Where("family_id = ? AND revoked_at IS NULL", rt.FamilyID).
				Update("revoked_at", now).Error; err != nil {
				return err
			}
			reuseDetected = true
			return nil
		}

		// Check expiry
		if time.Now().After(rt.ExpiresAt) {
			return ErrTokenExpired
		}

		// Query the user
		var user model.User
		if err := tx.Where("id = ?", rt.UserID).First(&user).Error; err != nil {
			return err
		}

		// Insert new token in same family
		newHash := TokenHash(newPlainToken)
		newRT := &model.RefreshToken{
			UserID:           rt.UserID,
			TokenHash:        newHash,
			FamilyID:         rt.FamilyID,
			ExpiresAt:        newExpiresAt,
			CreatedIP:        &newIP,
			CreatedUserAgent: &newUserAgent,
		}
		if err := tx.Create(newRT).Error; err != nil {
			return err
		}

		// Revoke old token
		now := time.Now()
		if err := tx.Model(&model.RefreshToken{}).
			Where("id = ?", rt.ID).
			Updates(map[string]any{
				"revoked_at":       now,
				"replaced_by_hash": newHash,
			}).Error; err != nil {
			return err
		}

		result = &RefreshTokenResult{
			UserID:       user.ID,
			Username:     user.Username,
			Role:         user.Role,
			AccountType:  user.AccountType,
			TokenVersion: user.TokenVersion,
			FamilyID:     rt.FamilyID,
		}
		return nil
	})
	if err == nil && reuseDetected {
		return nil, ErrTokenReuseDetected
	}
	return result, err
}

// Revoke invalidates a single refresh token by its plain-text value.
func (r *RefreshTokenRepo) Revoke(ctx context.Context, plainToken string) error {
	hash := TokenHash(plainToken)
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("token_hash = ?", hash).
		Update("revoked_at", time.Now()).Error
}

// RevokeFamily invalidates all active tokens in the given family.
func (r *RefreshTokenRepo) RevokeFamily(ctx context.Context, familyID string) error {
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", time.Now()).Error
}
