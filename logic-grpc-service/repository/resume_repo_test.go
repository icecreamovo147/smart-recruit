package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/errs"
)

func TestConfirmUpload_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResumeRepo(db)
	ctx := context.Background()

	resume := &model.Resume{
		UserID: 1, OSSKey: "resumes/1/test.pdf",
		FileName: "test.pdf", FileType: "pdf", FileSize: 1000,
		IsValid: 1, UploadedAt: time.Now(),
	}
	if err := repo.ConfirmUpload(ctx, resume); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if resume.ID == 0 {
		t.Errorf("expected ID to be set after insert")
	}

	var count int64
	db.Model(&model.Resume{}).Where("user_id = ? AND is_valid = 1", int64(1)).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 valid resume, got %d", count)
	}
}

func TestConfirmUpload_ReplacesOld(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResumeRepo(db)
	ctx := context.Background()

	r1 := &model.Resume{
		UserID: 1, OSSKey: "resumes/1/old.pdf",
		FileName: "old.pdf", FileType: "pdf", FileSize: 500,
		IsValid: 1, UploadedAt: time.Now(),
	}
	if err := repo.ConfirmUpload(ctx, r1); err != nil {
		t.Fatalf("first upload should succeed: %v", err)
	}

	r2 := &model.Resume{
		UserID: 1, OSSKey: "resumes/1/new.pdf",
		FileName: "new.pdf", FileType: "pdf", FileSize: 800,
		IsValid: 1, UploadedAt: time.Now(),
	}
	if err := repo.ConfirmUpload(ctx, r2); err != nil {
		t.Fatalf("second upload should succeed: %v", err)
	}

	var old model.Resume
	db.First(&old, r1.ID)
	if old.IsValid != 0 {
		t.Errorf("old resume should be invalid (is_valid=0), got %d", old.IsValid)
	}

	var newR model.Resume
	db.First(&newR, r2.ID)
	if newR.IsValid != 1 {
		t.Errorf("new resume should be valid (is_valid=1), got %d", newR.IsValid)
	}

	var count int64
	db.Model(&model.Resume{}).Where("user_id = ? AND is_valid = 1", int64(1)).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 valid resume, got %d", count)
	}
}

func TestConfirmUpload_UniqueConstraintEnforced(t *testing.T) {
	// Verifies the partial unique index guarantees exactly one valid resume per user.
	// SQLite serializes write transactions, so this runs sequentially.
	// Full concurrent upload testing requires MySQL.
	db := setupTestDB(t)
	repo := NewResumeRepo(db)
	ctx := context.Background()

	userID := int64(1)

	for i := 0; i < 5; i++ {
		r := &model.Resume{
			UserID:   userID,
			OSSKey:   fmt.Sprintf("resumes/%d/seq_%d.pdf", userID, i),
			FileName: fmt.Sprintf("seq_%d.pdf", i),
			FileType: "pdf", FileSize: int64(1000 + i),
			IsValid: 1, UploadedAt: time.Now(),
		}
		err := repo.ConfirmUpload(ctx, r)
		if err != nil && !errors.Is(err, errs.ErrDuplicateResumeSentinel) {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
	}

	// Exactly one valid resume
	var validCount int64
	db.Model(&model.Resume{}).Where("user_id = ? AND is_valid = 1", userID).Count(&validCount)
	if validCount != 1 {
		t.Errorf("expected exactly 1 valid resume, got %d", validCount)
	}

	// All 5 exist in the table
	var total int64
	db.Model(&model.Resume{}).Where("user_id = ?", userID).Count(&total)
	if total != 5 {
		t.Errorf("expected 5 total resumes, got %d", total)
	}
}
