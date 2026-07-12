package ai

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/repository"
)

func TestToolExecutorRecruitingIntelligenceToolsExecute(t *testing.T) {
	db := setupToolExecutorIntelligenceDB(t)
	executor := NewToolExecutor(
		repository.NewApplicationRepo(db),
		repository.NewJobRepo(db),
		repository.NewResumeRepo(db),
		nil,
		nil,
		repository.NewProfileRepo(db),
		repository.NewResumeProfileRepo(db),
		repository.NewCandidateMatchRepo(db),
	)
	seedToolExecutorIntelligenceScenario(t, db)

	ctx := context.Background()
	parseResult, err := executor.Execute(ctx, 11, "parse_resume_profile", map[string]any{"application_id": int64(1001)})
	if err != nil {
		t.Fatalf("parse_resume_profile failed: %v", err)
	}
	assertToolJSONHasTrace(t, parseResult.Content, "profile")

	getProfileResult, err := executor.Execute(ctx, 11, "get_resume_profile", map[string]any{"application_id": int64(1001)})
	if err != nil {
		t.Fatalf("get_resume_profile failed: %v", err)
	}
	assertToolJSONHasTrace(t, getProfileResult.Content, "profile")

	evaluateResult, err := executor.Execute(ctx, 11, "evaluate_candidate_match", map[string]any{"application_id": int64(1001)})
	if err != nil {
		t.Fatalf("evaluate_candidate_match failed: %v", err)
	}
	assertToolJSONHasTrace(t, evaluateResult.Content, "evaluation")

	getEvaluationResult, err := executor.Execute(ctx, 11, "get_candidate_match_evaluation", map[string]any{"application_id": int64(1001)})
	if err != nil {
		t.Fatalf("get_candidate_match_evaluation failed: %v", err)
	}
	assertToolJSONHasTrace(t, getEvaluationResult.Content, "evaluation")

	compareResult, err := executor.Execute(ctx, 11, "compare_candidates_for_job", map[string]any{"job_id": int64(701)})
	if err != nil {
		t.Fatalf("compare_candidates_for_job failed: %v", err)
	}
	var comparePayload map[string]any
	if err := json.Unmarshal([]byte(compareResult.Content), &comparePayload); err != nil {
		t.Fatalf("compare result is not JSON: %v", err)
	}
	candidates, ok := comparePayload["candidates"].([]any)
	if !ok || len(candidates) != 1 {
		t.Fatalf("expected one comparison candidate, got %v", comparePayload["candidates"])
	}
}

func TestToolExecutorRecruitingIntelligenceToolsRejectMissingParameters(t *testing.T) {
	executor := &ToolExecutor{}
	if _, err := executor.Execute(context.Background(), 11, "evaluate_candidate_match", nil); err == nil {
		t.Fatal("expected missing application_id error")
	}
	if _, err := executor.Execute(context.Background(), 11, "compare_candidates_for_job", nil); err == nil {
		t.Fatal("expected missing job_id error")
	}
}

func setupToolExecutorIntelligenceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Job{},
		&model.CandidateProfile{},
		&model.Resume{},
		&model.ResumeParseRun{},
		&model.ResumeProfile{},
		&model.ResumeEducation{},
		&model.ResumeExperience{},
		&model.ResumeProject{},
		&model.ResumeSkill{},
		&model.Application{},
		&model.CandidateMatchEvaluation{},
		&model.CandidateMatchEvidence{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func seedToolExecutorIntelligenceScenario(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	rows := []any{
		&model.User{ID: 501, Username: "candidate", AccountType: "candidate", Status: "active", CreatedAt: now, UpdatedAt: now},
		&model.Job{ID: 701, HrID: 11, Title: "Senior Backend Engineer", Department: "Platform", Location: "Shanghai", Description: "Build backend APIs.", Requirements: "Go PostgreSQL Kubernetes Bachelor", Status: 1, CreatedAt: now, UpdatedAt: now},
		&model.CandidateProfile{UserID: 501, RealName: "Ada Lovelace", Phone: "13800000000", Education: "Bachelor", Skills: "Go, PostgreSQL", IsComplete: 1, CreatedAt: now, UpdatedAt: now},
		&model.Resume{ID: 801, UserID: 501, FileName: "ada.pdf", FileType: "pdf", OSSKey: "resume/ada.pdf", ParsedText: "Ada Lovelace ada@example.com 13800000000 has 6 years building Go PostgreSQL Kubernetes APIs. Bachelor degree.", ParsedAt: &now, IsValid: 1, UploadedAt: now},
		&model.Application{ID: 1001, JobID: 701, UserID: 501, ResumeID: 801, Status: 0, StatusKey: "applied", RoundNo: 1, IsCurrent: 1, AppliedAt: now, UpdatedAt: now},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("seed %#v: %v", row, err)
		}
	}
}

func assertToolJSONHasTrace(t *testing.T, raw string, requiredKey string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("tool result is not JSON: %v", err)
	}
	if _, ok := payload[requiredKey]; !ok {
		t.Fatalf("tool result missing %q: %v", requiredKey, payload)
	}
	if _, ok := payload["trace"]; !ok {
		t.Fatalf("tool result missing trace: %v", payload)
	}
}
