package persistence

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
)

func TestNativeStoreSaveRecruitingResumeProfileDraftVersionsAndChildren(t *testing.T) {
	ctx := context.Background()
	db := newRecruitingGenerationTestDB(t)
	store := NewNativeStore(db)
	now := time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC)
	seedRecruitingGenerationBaseRows(t, db, now)

	source, found, err := store.GetRecruitingResumeSource(ctx, 8001)
	if err != nil {
		t.Fatalf("GetRecruitingResumeSource error = %v", err)
	}
	if !found || source.UserID != 3001 || source.ParsedText == "" {
		t.Fatalf("resume source = %+v found=%v, want seeded parsed text", source, found)
	}

	snapshot, err := store.SaveRecruitingResumeProfileDraft(ctx, aiagentgrpc.RecruitingResumeProfileDraft{
		ResumeID:      8001,
		UserID:        3001,
		ParserVersion: "native-resume-profile-parser-v1",
		InputHash:     "hash-new",
		RawJSON:       `{"full_name":"Ada Lovelace","skills":[{"name":"Go"}]}`,
		FullName:      "Ada Lovelace",
		Headline:      "Backend engineer",
		Summary:       "Builds Go services",
		Skills: []aiagentgrpc.RecruitingResumeSkillRow{
			{Name: "Go", Category: "language", Level: "senior", Years: 5, Evidence: "services", SortOrder: 1},
		},
	})
	if err != nil {
		t.Fatalf("SaveRecruitingResumeProfileDraft error = %v", err)
	}
	if snapshot.Profile.Version != 2 || snapshot.Profile.IsCurrent != 1 {
		t.Fatalf("saved profile version/current = %d/%d, want 2/1", snapshot.Profile.Version, snapshot.Profile.IsCurrent)
	}
	if snapshot.ParseRun.Status != "succeeded" || snapshot.ParseRun.InputHash != "hash-new" {
		t.Fatalf("parse run = %+v, want succeeded hash-new", snapshot.ParseRun)
	}
	if len(snapshot.Skills) != 1 || snapshot.Skills[0].Name != "Go" {
		t.Fatalf("skills = %+v, want generated Go skill", snapshot.Skills)
	}
	var old recruitingResumeProfileRecord
	if err := db.Where("id = ?", uint64(6001)).First(&old).Error; err != nil {
		t.Fatalf("load old profile: %v", err)
	}
	if old.IsCurrent != 0 {
		t.Fatalf("old profile is_current = %d, want 0", old.IsCurrent)
	}
}

func TestNativeStoreSaveRecruitingCandidateMatchDraftVersionsLatestAndAgentRun(t *testing.T) {
	ctx := context.Background()
	db := newRecruitingGenerationTestDB(t)
	store := NewNativeStore(db)
	now := time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC)
	seedRecruitingGenerationBaseRows(t, db, now)

	source, found, err := store.GetRecruitingMatchSource(ctx, 7001)
	if err != nil {
		t.Fatalf("GetRecruitingMatchSource error = %v", err)
	}
	if !found || source.Job.Requirements == "" || source.Profile.Profile.ID != 6001 {
		t.Fatalf("match source = %+v found=%v, want job requirements and current profile", source, found)
	}

	agentRunID := uint64(12001)
	sourceID := uint64(6401)
	snapshot, err := store.SaveRecruitingCandidateMatchDraft(ctx, aiagentgrpc.RecruitingCandidateMatchDraft{
		ApplicationID:      7001,
		JobID:              9001,
		CandidateUserID:    3001,
		ResumeProfileID:    6001,
		AgentRunID:         &agentRunID,
		OverallScore:       91.5,
		Recommendation:     "strong_match",
		Summary:            "Strong Go backend match",
		StrengthsJSON:      `["Go"]`,
		RisksJSON:          `[]`,
		ScoreBreakdownJSON: `{"missing_requirements":[],"dimensions":[{"name":"backend","score":91.5}]}`,
		ModelName:          "native-candidate-match-scorer-v1",
		Evidence: []aiagentgrpc.RecruitingCandidateMatchEvidenceRow{
			{EvidenceType: "skill", Dimension: "backend", SourceTable: "resume_skills", SourceID: &sourceID, Snippet: "Go", Weight: 0.8, ScoreImpact: 5.5, MetadataJSON: `{"source":"resume"}`},
		},
	})
	if err != nil {
		t.Fatalf("SaveRecruitingCandidateMatchDraft error = %v", err)
	}
	if snapshot.Evaluation.EvaluationVersion != 2 || snapshot.Evaluation.IsLatest != 1 {
		t.Fatalf("evaluation version/latest = %d/%d, want 2/1", snapshot.Evaluation.EvaluationVersion, snapshot.Evaluation.IsLatest)
	}
	if snapshot.Evaluation.AgentRunID == nil || *snapshot.Evaluation.AgentRunID != agentRunID {
		t.Fatalf("agent run id = %v, want %d", snapshot.Evaluation.AgentRunID, agentRunID)
	}
	if len(snapshot.Evidence) != 1 || snapshot.Evidence[0].Snippet != "Go" {
		t.Fatalf("evidence = %+v, want generated evidence", snapshot.Evidence)
	}
	var old recruitingCandidateMatchEvaluationRecord
	if err := db.Where("id = ?", uint64(9101)).First(&old).Error; err != nil {
		t.Fatalf("load old evaluation: %v", err)
	}
	if old.IsLatest != 0 {
		t.Fatalf("old evaluation is_latest = %d, want 0", old.IsLatest)
	}
}

func newRecruitingGenerationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&recruitingApplicationTestRecord{},
		&recruitingCandidateProfileTestRecord{},
		&recruitingGenerationResumeTestRecord{},
		&recruitingGenerationJobTestRecord{},
		&recruitingResumeParseRunRecord{},
		&recruitingResumeProfileRecord{},
		&recruitingResumeEducationRecord{},
		&recruitingResumeExperienceRecord{},
		&recruitingResumeProjectRecord{},
		&recruitingResumeSkillRecord{},
		&recruitingCandidateMatchEvaluationRecord{},
		&recruitingCandidateMatchEvidenceRecord{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func seedRecruitingGenerationBaseRows(t *testing.T, db *gorm.DB, now time.Time) {
	t.Helper()
	if err := db.Create(&recruitingGenerationJobTestRecord{ID: 9001, Title: "Backend Engineer", Department: "Engineering", Location: "Shanghai", Description: "Build platform services", Requirements: "Go and distributed systems"}).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if err := db.Create(&recruitingApplicationTestRecord{ID: 7001, JobID: 9001, UserID: 3001, ResumeID: 8001, StatusKey: "applied", IsCurrent: 1, AppliedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	if err := db.Create(&recruitingCandidateProfileTestRecord{ID: 1, UserID: 3001, RealName: "Ada Lovelace"}).Error; err != nil {
		t.Fatalf("seed candidate profile: %v", err)
	}
	if err := db.Create(&recruitingGenerationResumeTestRecord{ID: 8001, UserID: 3001, FileName: "ada.pdf", ParsedText: "Ada builds Go services.", IsValid: 1}).Error; err != nil {
		t.Fatalf("seed resume: %v", err)
	}
	completedAt := now.Add(time.Minute)
	if err := db.Create(&recruitingResumeParseRunRecord{ID: 5001, ResumeID: 8001, UserID: 3001, Status: "succeeded", ParserVersion: ns("legacy"), InputHash: ns("hash-old"), StartedAt: now, CompletedAt: &completedAt, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed parse run: %v", err)
	}
	if err := db.Create(&recruitingResumeProfileRecord{ID: 6001, ResumeID: 8001, UserID: 3001, ParseRunID: 5001, Version: 1, IsCurrent: 1, FullName: ns("Ada"), Summary: ns("Go backend"), CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if err := db.Create(&recruitingResumeSkillRecord{ID: 6401, ResumeProfileID: 6001, Name: "Go", Category: ns("language"), Level: ns("senior"), Years: nf(5), Evidence: ns("services"), SortOrder: 1, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed skill: %v", err)
	}
	if err := db.Create(&recruitingCandidateMatchEvaluationRecord{ID: 9101, ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, ResumeProfileID: 6001, EvaluationVersion: 1, IsLatest: 1, OverallScore: nf(70), Recommendation: ns("possible_match"), Summary: ns("Old"), ScoreBreakdownJSON: ns(`{"dimensions":[]}`), ModelName: ns("legacy"), EvaluatedAt: now, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed evaluation: %v", err)
	}
}

type recruitingGenerationResumeTestRecord struct {
	ID         int64  `gorm:"primaryKey"`
	UserID     int64  `gorm:"column:user_id"`
	FileName   string `gorm:"column:file_name"`
	ParsedText string `gorm:"column:parsed_text"`
	IsValid    int32  `gorm:"column:is_valid"`
}

func (recruitingGenerationResumeTestRecord) TableName() string { return "resumes" }

type recruitingGenerationJobTestRecord struct {
	ID           int64  `gorm:"primaryKey"`
	Title        string `gorm:"column:title"`
	Department   string `gorm:"column:department"`
	Location     string `gorm:"column:location"`
	Description  string `gorm:"column:description"`
	Requirements string `gorm:"column:requirements"`
}

func (recruitingGenerationJobTestRecord) TableName() string { return "jobs" }
