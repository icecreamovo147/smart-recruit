package repository

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

func TestCreateNewRound_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewApplicationRepo(db)
	ctx := context.Background()

	app := &model.Application{
		UserID: 1, JobID: 1, ResumeID: 1, Status: 0,
	}
	err := repo.CreateNewRound(ctx, app)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if app.RoundNo != 1 {
		t.Errorf("expected RoundNo=1, got %d", app.RoundNo)
	}
	if app.IsCurrent != 1 {
		t.Errorf("expected IsCurrent=1, got %d", app.IsCurrent)
	}
	if app.ID == 0 {
		t.Errorf("expected ID to be set after insert")
	}
}

func TestCreateNewRound_DuplicateSequential(t *testing.T) {
	db := setupTestDB(t)
	repo := NewApplicationRepo(db)
	ctx := context.Background()

	app1 := &model.Application{
		UserID: 1, JobID: 1, ResumeID: 1, Status: 0,
	}
	if err := repo.CreateNewRound(ctx, app1); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	app2 := &model.Application{
		UserID: 1, JobID: 1, ResumeID: 1, Status: 0,
	}
	err := repo.CreateNewRound(ctx, app2)
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected gorm.ErrDuplicatedKey, got: %v", err)
	}
}

func TestCreateNewRound_UniqueConstraintBlocksDuplicate(t *testing.T) {
	// Verifies the partial unique index prevents duplicate active applications.
	// SQLite cannot run truly concurrent write transactions, so this test
	// executes sequentially but validates the DB-level constraint.
	// Full concurrency testing requires MySQL.
	db := setupTestDB(t)
	repo := NewApplicationRepo(db)
	ctx := context.Background()

	userID, jobID := int64(1), int64(1)

	// First application
	app1 := &model.Application{UserID: userID, JobID: jobID, ResumeID: 1, Status: 0}
	if err := repo.CreateNewRound(ctx, app1); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	// Attempt multiple duplicates — each must fail with the same constraint
	for i := 0; i < 5; i++ {
		app := &model.Application{UserID: userID, JobID: jobID, ResumeID: 1, Status: 0}
		err := repo.CreateNewRound(ctx, app)
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			t.Fatalf("attempt %d: expected gorm.ErrDuplicatedKey, got: %v", i, err)
		}
	}

	// DB-level: exactly one active application
	var count int64
	db.Model(&model.Application{}).
		Where("user_id = ? AND job_id = ? AND is_current = ? AND status <> ?", userID, jobID, 1, 3).
		Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 active application, got %d", count)
	}
}

func TestCreateNewRound_AfterRejected(t *testing.T) {
	db := setupTestDB(t)
	repo := NewApplicationRepo(db)
	ctx := context.Background()

	app1 := &model.Application{UserID: 1, JobID: 1, ResumeID: 1, Status: 0}
	if err := repo.CreateNewRound(ctx, app1); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}
	if app1.RoundNo != 1 {
		t.Errorf("expected RoundNo=1, got %d", app1.RoundNo)
	}

	// Mark as rejected and not current
	db.Model(&model.Application{}).Where("id = ?", app1.ID).
		Updates(map[string]any{"status": 3, "is_current": 0})

	// New application after rejection should succeed
	app2 := &model.Application{UserID: 1, JobID: 1, ResumeID: 1, Status: 0}
	if err := repo.CreateNewRound(ctx, app2); err != nil {
		t.Fatalf("second create after rejection should succeed: %v", err)
	}
	if app2.RoundNo != 2 {
		t.Errorf("expected RoundNo=2, got %d", app2.RoundNo)
	}
	if app2.IsCurrent != 1 {
		t.Errorf("expected IsCurrent=1, got %d", app2.IsCurrent)
	}
}
