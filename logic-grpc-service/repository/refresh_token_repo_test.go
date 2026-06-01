package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

func TestRefreshTokenRepo_CreateStoresHashOnly(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepo(db)
	ctx := context.Background()
	user := createRefreshTokenTestUser(t, db)

	plain := "plain-refresh-token"
	if err := repo.Create(ctx, user.ID, plain, "family-a", time.Now().Add(time.Hour), "127.0.0.1", "test-agent"); err != nil {
		t.Fatalf("create refresh token failed: %v", err)
	}

	var stored model.RefreshToken
	if err := db.First(&stored).Error; err != nil {
		t.Fatalf("load stored refresh token failed: %v", err)
	}
	if stored.TokenHash == plain {
		t.Fatalf("expected hash storage, got plaintext token")
	}
	if stored.TokenHash != TokenHash(plain) {
		t.Fatalf("stored hash mismatch: got %q want %q", stored.TokenHash, TokenHash(plain))
	}
}

func TestRefreshTokenRepo_RotateRevokesOldAndIssuesNew(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepo(db)
	ctx := context.Background()
	user := createRefreshTokenTestUser(t, db)

	oldPlain := "old-refresh-token"
	newPlain := "new-refresh-token"
	if err := repo.Create(ctx, user.ID, oldPlain, "family-b", time.Now().Add(time.Hour), "", ""); err != nil {
		t.Fatalf("create old refresh token failed: %v", err)
	}

	result, err := repo.Rotate(ctx, oldPlain, newPlain, time.Now().Add(time.Hour), "10.0.0.1", "agent")
	if err != nil {
		t.Fatalf("rotate failed: %v", err)
	}
	if result.UserID != user.ID || result.Username != user.Username || result.Role != user.Role {
		t.Fatalf("unexpected rotation identity: %+v", result)
	}

	var oldToken model.RefreshToken
	if err := db.Where("token_hash = ?", TokenHash(oldPlain)).First(&oldToken).Error; err != nil {
		t.Fatalf("load old token failed: %v", err)
	}
	if oldToken.RevokedAt == nil {
		t.Fatalf("expected old token to be revoked")
	}
	if oldToken.ReplacedByHash == nil || *oldToken.ReplacedByHash != TokenHash(newPlain) {
		t.Fatalf("expected replaced_by_hash to point to new token")
	}

	var newToken model.RefreshToken
	if err := db.Where("token_hash = ?", TokenHash(newPlain)).First(&newToken).Error; err != nil {
		t.Fatalf("load new token failed: %v", err)
	}
	if newToken.RevokedAt != nil {
		t.Fatalf("expected new token to remain active")
	}
	if newToken.FamilyID != oldToken.FamilyID {
		t.Fatalf("expected family to be preserved")
	}
}

func TestRefreshTokenRepo_RevokedTokenReuseRevokesFamilyEvenAfterExpiry(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepo(db)
	ctx := context.Background()
	user := createRefreshTokenTestUser(t, db)

	oldPlain := "old-reused-refresh-token"
	currentPlain := "current-refresh-token"
	familyID := "family-reuse"
	now := time.Now()
	revokedAt := now.Add(-2 * time.Hour)
	reuseExpiredAt := now.Add(-time.Hour)
	currentExpiresAt := now.Add(time.Hour)
	if err := db.Create(&model.RefreshToken{
		UserID:         user.ID,
		TokenHash:      TokenHash(oldPlain),
		FamilyID:       familyID,
		ExpiresAt:      reuseExpiredAt,
		RevokedAt:      &revokedAt,
		ReplacedByHash: stringPtr(TokenHash(currentPlain)),
	}).Error; err != nil {
		t.Fatalf("create revoked token failed: %v", err)
	}
	if err := db.Create(&model.RefreshToken{
		UserID:    user.ID,
		TokenHash: TokenHash(currentPlain),
		FamilyID:  familyID,
		ExpiresAt: currentExpiresAt,
		CreatedIP: stringPtr("127.0.0.1"),
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create current token failed: %v", err)
	}

	_, err := repo.Rotate(ctx, oldPlain, "attacker-new-token", time.Now().Add(time.Hour), "", "")
	if !errors.Is(err, ErrTokenReuseDetected) {
		t.Fatalf("expected ErrTokenReuseDetected, got %v", err)
	}

	var reused model.RefreshToken
	if err := db.Where("token_hash = ?", TokenHash(oldPlain)).First(&reused).Error; err != nil {
		t.Fatalf("load reused token failed: %v", err)
	}
	if reused.ReuseDetectedAt == nil {
		t.Fatalf("expected reuse_detected_at to be set")
	}

	var current model.RefreshToken
	if err := db.Where("token_hash = ?", TokenHash(currentPlain)).First(&current).Error; err != nil {
		t.Fatalf("load current token failed: %v", err)
	}
	if current.RevokedAt == nil {
		t.Fatalf("expected active family token to be revoked")
	}
}

func createRefreshTokenTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()
	user := model.User{Username: "refresh-user", Password: "hashed-password", Role: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create test user failed: %v", err)
	}
	return user
}

func stringPtr(value string) *string {
	return &value
}
