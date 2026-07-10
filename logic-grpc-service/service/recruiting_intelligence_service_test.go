package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/repository"
)

func TestRecruitingIntelligenceResolveResumeIDRejectsMixedApplicationResume(t *testing.T) {
	db := setupRecruitingIntelligenceServiceTestDB(t)
	svc := newRecruitingIntelligenceServiceForTest(db)
	seedRecruitingIntelligenceApplication(t, db, 1001, 801)

	resumeID, err := svc.resolveResumeID(context.Background(), 802, 1001)
	if err == nil {
		t.Fatalf("expected mismatched application_id/resume_id error")
	}
	if resumeID != 0 {
		t.Fatalf("expected no resume id on mismatch, got %d", resumeID)
	}
}

func TestRecruitingIntelligenceResolveResumeIDUsesApplicationResumeWhenConsistent(t *testing.T) {
	db := setupRecruitingIntelligenceServiceTestDB(t)
	svc := newRecruitingIntelligenceServiceForTest(db)
	seedRecruitingIntelligenceApplication(t, db, 1001, 801)

	resumeID, err := svc.resolveResumeID(context.Background(), 801, 1001)
	if err != nil {
		t.Fatalf("resolveResumeID returned error: %v", err)
	}
	if resumeID != 801 {
		t.Fatalf("expected application resume id 801, got %d", resumeID)
	}
}

func TestRecruitingIntelligenceValidateResumeIdentifierConsistencyRejectsMixedProfile(t *testing.T) {
	db := setupRecruitingIntelligenceServiceTestDB(t)
	svc := newRecruitingIntelligenceServiceForTest(db)
	seedRecruitingIntelligenceApplication(t, db, 1001, 801)
	seedRecruitingIntelligenceResumeProfile(t, db, 2001, 802)

	err := svc.validateResumeIdentifierConsistency(context.Background(), 0, 2001, 1001)
	if err == nil {
		t.Fatalf("expected mismatched application_id/profile_id error")
	}
}

func TestRecruitingIntelligenceValidateResumeIdentifierConsistencyAllowsMatchingProfile(t *testing.T) {
	db := setupRecruitingIntelligenceServiceTestDB(t)
	svc := newRecruitingIntelligenceServiceForTest(db)
	seedRecruitingIntelligenceApplication(t, db, 1001, 801)
	seedRecruitingIntelligenceResumeProfile(t, db, 2001, 801)

	err := svc.validateResumeIdentifierConsistency(context.Background(), 801, 2001, 1001)
	if err != nil {
		t.Fatalf("validateResumeIdentifierConsistency returned error: %v", err)
	}
}

func TestRecruitingIntelligenceResolveResumeProfilePropagatesRepositoryError(t *testing.T) {
	db := setupRecruitingIntelligenceServiceTestDB(t)
	svc := newRecruitingIntelligenceServiceForTest(db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db failed: %v", err)
	}

	snapshot, resp := svc.resolveResumeProfile(context.Background(), 801, 0, 0)
	if snapshot != nil {
		t.Fatalf("expected no snapshot on repository error")
	}
	if resp == nil {
		t.Fatalf("expected non-OK response on repository error")
	}
	if resp.Code != errs.ErrInternal {
		t.Fatalf("expected ErrInternal, got code %d msg %q", resp.Code, resp.Msg)
	}
}

func setupRecruitingIntelligenceServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Application{},
		&model.ResumeProfile{},
	); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}

func newRecruitingIntelligenceServiceForTest(db *gorm.DB) *RecruitingIntelligenceService {
	return NewRecruitingIntelligenceService(
		repository.NewApplicationRepo(db),
		nil,
		nil,
		repository.NewResumeProfileRepo(db),
		nil,
		nil,
		nil,
		NewServiceAuthorizer(nil, nil),
	)
}

func seedRecruitingIntelligenceApplication(t *testing.T, db *gorm.DB, applicationID int64, resumeID int64) {
	t.Helper()
	now := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	application := model.Application{
		ID:        applicationID,
		JobID:     701,
		UserID:    501,
		ResumeID:  resumeID,
		Status:    1,
		StatusKey: "applied",
		RoundNo:   1,
		IsCurrent: 1,
		AppliedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(&application).Error; err != nil {
		t.Fatalf("seed application failed: %v", err)
	}
}

func seedRecruitingIntelligenceResumeProfile(t *testing.T, db *gorm.DB, profileID uint64, resumeID int64) {
	t.Helper()
	now := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	profile := model.ResumeProfile{
		ID:         profileID,
		ResumeID:   resumeID,
		UserID:     501,
		ParseRunID: 901,
		Version:    1,
		IsCurrent:  1,
		FullName:   "Ada Lovelace",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("seed resume profile failed: %v", err)
	}
}
