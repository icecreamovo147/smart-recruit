package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/repository"
)

func TestCandidateMatchServiceEvaluateApplicationCreatesEvidenceRisksAndMissingRequirements(t *testing.T) {
	db := setupCandidateMatchServiceTestDB(t)
	seedCandidateMatchScenario(t, db, candidateMatchSeedOptions{ProfileComplete: false})
	svc := newCandidateMatchServiceForTest(db).WithRuntimePolicy(candidateMatchLegacyTestPolicy())

	snapshot, err := svc.EvaluateApplication(context.Background(), 1001, nil)
	if err != nil {
		t.Fatalf("EvaluateApplication failed: %v", err)
	}
	if snapshot.Evaluation.ID == 0 {
		t.Fatal("expected persisted evaluation ID")
	}
	if snapshot.Evaluation.Recommendation == "" {
		t.Fatal("expected recommendation")
	}
	if snapshot.Evaluation.ModelName != defaultCandidateMatchScorerVersion {
		t.Fatalf("unexpected model name %q", snapshot.Evaluation.ModelName)
	}
	if len(snapshot.Evidence) == 0 {
		t.Fatal("expected evidence rows")
	}

	var risks []candidateMatchSignal
	if err := json.Unmarshal([]byte(snapshot.Evaluation.RisksJSON), &risks); err != nil {
		t.Fatalf("unmarshal risks: %v", err)
	}
	if !candidateMatchSignalsContain(risks, "candidate_profile_incomplete") {
		t.Fatalf("expected incomplete profile risk, got %+v", risks)
	}

	var breakdown candidateMatchBreakdown
	if err := json.Unmarshal([]byte(snapshot.Evaluation.ScoreBreakdownJSON), &breakdown); err != nil {
		t.Fatalf("unmarshal breakdown: %v", err)
	}
	if !stringSliceContains(breakdown.MissingRequirements, "kubernetes") || !stringSliceContains(breakdown.MissingRequirements, "react") {
		t.Fatalf("expected missing kubernetes/react requirements, got %+v", breakdown.MissingRequirements)
	}

	assertCandidateMatchEvidence(t, snapshot.Evidence, "skill", "resume_skills")
	assertCandidateMatchEvidence(t, snapshot.Evidence, "missing_requirement", "jobs")
	assertCandidateMatchEvidence(t, snapshot.Evidence, "candidate_profile", "candidate_profiles")
	assertCandidateMatchEvidence(t, snapshot.Evidence, "risk", "resume_profiles")
}

func TestCandidateMatchServiceEvaluateApplicationStableOutputAndRerunVersioning(t *testing.T) {
	db := setupCandidateMatchServiceTestDB(t)
	seedCandidateMatchScenario(t, db, candidateMatchSeedOptions{ProfileComplete: true})
	svc := newCandidateMatchServiceForTest(db).WithRuntimePolicy(candidateMatchLegacyTestPolicy())
	svc.now = func() time.Time { return time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC) }

	first, err := svc.EvaluateApplication(context.Background(), 1001, nil)
	if err != nil {
		t.Fatalf("first EvaluateApplication failed: %v", err)
	}
	second, err := svc.EvaluateApplication(context.Background(), 1001, nil)
	if err != nil {
		t.Fatalf("second EvaluateApplication failed: %v", err)
	}

	if first.Evaluation.EvaluationVersion != 1 || second.Evaluation.EvaluationVersion != 2 {
		t.Fatalf("expected rerun versions 1 and 2, got %d and %d", first.Evaluation.EvaluationVersion, second.Evaluation.EvaluationVersion)
	}
	if first.Evaluation.OverallScore != second.Evaluation.OverallScore {
		t.Fatalf("expected stable score, got %.1f and %.1f", first.Evaluation.OverallScore, second.Evaluation.OverallScore)
	}
	if first.Evaluation.ScoreBreakdownJSON != second.Evaluation.ScoreBreakdownJSON ||
		first.Evaluation.StrengthsJSON != second.Evaluation.StrengthsJSON ||
		first.Evaluation.RisksJSON != second.Evaluation.RisksJSON {
		t.Fatalf("expected stable JSON output\nfirst=%s\nsecond=%s", first.Evaluation.ScoreBreakdownJSON, second.Evaluation.ScoreBreakdownJSON)
	}

	latest, err := repository.NewCandidateMatchRepo(db).GetLatestByApplicationID(context.Background(), 1001)
	if err != nil {
		t.Fatalf("GetLatestByApplicationID failed: %v", err)
	}
	if latest == nil || latest.ID != second.Evaluation.ID || latest.IsLatest != 1 {
		t.Fatalf("expected second evaluation latest, got %+v", latest)
	}

	var firstStored model.CandidateMatchEvaluation
	if err := db.First(&firstStored, first.Evaluation.ID).Error; err != nil {
		t.Fatalf("load first evaluation failed: %v", err)
	}
	if firstStored.IsLatest != 0 {
		t.Fatalf("expected first evaluation no longer latest, got %d", firstStored.IsLatest)
	}
}

func TestCandidateMatchServiceEvaluateApplicationUsesEnhancedScoringByDefault(t *testing.T) {
	db := setupCandidateMatchServiceTestDB(t)
	seedCandidateMatchScenario(t, db, candidateMatchSeedOptions{ProfileComplete: true})
	svc := newCandidateMatchServiceForTest(db)

	snapshot, err := svc.EvaluateApplication(context.Background(), 1001, nil)
	if err != nil {
		t.Fatalf("EvaluateApplication failed: %v", err)
	}

	var breakdown EnhancedScoreBreakdown
	if err := json.Unmarshal([]byte(snapshot.Evaluation.ScoreBreakdownJSON), &breakdown); err != nil {
		t.Fatalf("unmarshal enhanced breakdown: %v", err)
	}
	if breakdown.ScorerType == "" || breakdown.RequirementProfile == nil || len(breakdown.RequirementResults) == 0 {
		t.Fatalf("expected enhanced score breakdown, got %s", snapshot.Evaluation.ScoreBreakdownJSON)
	}
	if err := breakdown.Validate(); err != nil {
		t.Fatalf("expected valid enhanced score breakdown: %v", err)
	}
}

func TestCandidateMatchServiceEvaluateApplicationRejectsIncompleteResumeProfile(t *testing.T) {
	db := setupCandidateMatchServiceTestDB(t)
	seedCandidateMatchScenario(t, db, candidateMatchSeedOptions{ProfileComplete: true, ResumeProfileIncomplete: true})
	svc := newCandidateMatchServiceForTest(db)

	_, err := svc.EvaluateApplication(context.Background(), 1001, nil)
	if !errors.Is(err, ErrCandidateMatchIncompleteProfile) {
		t.Fatalf("expected ErrCandidateMatchIncompleteProfile, got %v", err)
	}
	var count int64
	if err := db.Model(&model.CandidateMatchEvaluation{}).Count(&count).Error; err != nil {
		t.Fatalf("count evaluations failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no evaluation for incomplete resume profile, got %d", count)
	}
}

func TestCandidateMatchServiceEvaluateApplicationDisabledByRuntimePolicy(t *testing.T) {
	db := setupCandidateMatchServiceTestDB(t)
	seedCandidateMatchScenario(t, db, candidateMatchSeedOptions{ProfileComplete: true})
	svc := newCandidateMatchServiceForTest(db).WithRuntimePolicy(AgentRuntimePolicy{
		StructuredResumeParse: true,
		CandidateMatch:        false,
		SemanticRetrieval:     true,
		MCPPolicy:             true,
		Planner:               true,
		SkillGovernance:       true,
		Fallbacks:             true,
	})

	_, err := svc.EvaluateApplication(context.Background(), 1001, nil)
	if !errors.Is(err, ErrAgentCapabilityDisabled) {
		t.Fatalf("expected ErrAgentCapabilityDisabled, got %v", err)
	}
	var count int64
	if err := db.Model(&model.CandidateMatchEvaluation{}).Count(&count).Error; err != nil {
		t.Fatalf("count evaluations failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected disabled evaluation to skip persistence, got %d evaluations", count)
	}
}

func TestCandidateMatchServiceEvaluateApplicationHandlesMissingDependencies(t *testing.T) {
	db := setupCandidateMatchServiceTestDB(t)
	svc := newCandidateMatchServiceForTest(db)

	_, err := svc.EvaluateApplication(context.Background(), 404, nil)
	if !errors.Is(err, ErrCandidateMatchMissingApplication) {
		t.Fatalf("expected missing application error, got %v", err)
	}

	seedCandidateMatchScenario(t, db, candidateMatchSeedOptions{ProfileComplete: true, SkipResumeProfile: true})
	_, err = svc.EvaluateApplication(context.Background(), 1001, nil)
	if !errors.Is(err, ErrCandidateMatchMissingResumeProfile) {
		t.Fatalf("expected missing resume profile error, got %v", err)
	}
}

func TestCandidateMatchServiceEvaluateApplicationReturnsMissingJobForDanglingApplication(t *testing.T) {
	db := setupCandidateMatchServiceTestDB(t)
	now := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	application := model.Application{
		ID:        1002,
		JobID:     9999,
		UserID:    501,
		ResumeID:  801,
		Status:    1,
		StatusKey: "applied",
		RoundNo:   1,
		IsCurrent: 1,
		AppliedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(&application).Error; err != nil {
		t.Fatalf("seed dangling application failed: %v", err)
	}
	svc := newCandidateMatchServiceForTest(db)

	_, err := svc.EvaluateApplication(context.Background(), application.ID, nil)
	if !errors.Is(err, ErrCandidateMatchMissingJob) {
		t.Fatalf("expected ErrCandidateMatchMissingJob, got %v", err)
	}
}

type candidateMatchSeedOptions struct {
	ProfileComplete         bool
	ResumeProfileIncomplete bool
	SkipResumeProfile       bool
}

func setupCandidateMatchServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
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
		t.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}

func newCandidateMatchServiceForTest(db *gorm.DB) *CandidateMatchService {
	return NewCandidateMatchService(
		repository.NewApplicationRepo(db),
		repository.NewJobRepo(db),
		repository.NewProfileRepo(db),
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		repository.NewCandidateMatchRepo(db),
	)
}

func candidateMatchLegacyTestPolicy() AgentRuntimePolicy {
	policy := DefaultAgentRuntimePolicy()
	policy.CandidateMatchSemantic = false
	return policy
}

func seedCandidateMatchScenario(t *testing.T, db *gorm.DB, opts candidateMatchSeedOptions) {
	t.Helper()
	now := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	user := model.User{ID: 501, Username: "candidate", AccountType: "candidate", Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user failed: %v", err)
	}
	job := model.Job{
		ID:           701,
		HrID:         11,
		Title:        "Senior Backend Engineer",
		Department:   "Platform",
		Location:     "Shanghai",
		Description:  "Build backend services and APIs.",
		Requirements: "Go, PostgreSQL, Kubernetes, React. Bachelor degree required.",
		Status:       1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("seed job failed: %v", err)
	}
	profile := model.CandidateProfile{
		ID:             601,
		UserID:         user.ID,
		RealName:       "Ada Lovelace",
		Phone:          "123456",
		Education:      "Bachelor",
		School:         "Example University",
		WorkExperience: "4 years backend engineering",
		Skills:         "Go, PostgreSQL",
		IsComplete:     0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if opts.ProfileComplete {
		profile.IsComplete = 1
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("seed profile failed: %v", err)
	}
	resume := model.Resume{
		ID:         801,
		UserID:     user.ID,
		OSSKey:     "resumes/501/ada.pdf",
		FileName:   "ada.pdf",
		FileType:   "pdf",
		FileSize:   2048,
		ParsedText: "Ada has built backend services with Go and PostgreSQL APIs.",
		ParsedAt:   &now,
		IsValid:    1,
		UploadedAt: now,
	}
	if err := db.Create(&resume).Error; err != nil {
		t.Fatalf("seed resume failed: %v", err)
	}
	application := model.Application{
		ID:        1001,
		JobID:     job.ID,
		UserID:    user.ID,
		ResumeID:  resume.ID,
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
	if opts.SkipResumeProfile {
		return
	}

	completedAt := now
	run := model.ResumeParseRun{
		ID:            901,
		ResumeID:      resume.ID,
		UserID:        user.ID,
		Status:        "succeeded",
		ParserVersion: "resume-profile-parser-v1",
		InputHash:     "hash",
		StartedAt:     now,
		CompletedAt:   &completedAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatalf("seed parse run failed: %v", err)
	}
	resumeProfile := model.ResumeProfile{
		ID:              10001,
		ResumeID:        resume.ID,
		UserID:          user.ID,
		ParseRunID:      run.ID,
		Version:         1,
		IsCurrent:       1,
		FullName:        "Ada Lovelace",
		Email:           "ada@example.com",
		Headline:        "Backend engineer",
		Summary:         "Builds backend services with Go and PostgreSQL.",
		TotalExperience: 4,
		HighestDegree:   "Bachelor",
		RawJSON:         `{}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if opts.ResumeProfileIncomplete {
		resumeProfile.FullName = ""
		resumeProfile.Summary = ""
	}
	if err := db.Create(&resumeProfile).Error; err != nil {
		t.Fatalf("seed resume profile failed: %v", err)
	}
	if opts.ResumeProfileIncomplete {
		return
	}
	experience := model.ResumeExperience{
		ID:               11001,
		ResumeProfileID:  resumeProfile.ID,
		Company:          "Example Inc",
		Title:            "Backend Engineer",
		Description:      "Built Go backend APIs with PostgreSQL.",
		AchievementsJSON: `["Improved service reliability"]`,
		SortOrder:        1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.Create(&experience).Error; err != nil {
		t.Fatalf("seed experience failed: %v", err)
	}
	skills := []model.ResumeSkill{
		{ID: 12001, ResumeProfileID: resumeProfile.ID, Name: "Go", Category: "language", SortOrder: 1, CreatedAt: now, UpdatedAt: now},
		{ID: 12002, ResumeProfileID: resumeProfile.ID, Name: "PostgreSQL", Category: "database", SortOrder: 2, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("seed skills failed: %v", err)
	}
}

func candidateMatchSignalsContain(signals []candidateMatchSignal, code string) bool {
	for _, signal := range signals {
		if signal.Code == code {
			return true
		}
	}
	return false
}

func stringSliceContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func assertCandidateMatchEvidence(t *testing.T, evidence []model.CandidateMatchEvidence, evidenceType, sourceTable string) {
	t.Helper()
	for _, item := range evidence {
		if item.EvidenceType == evidenceType && item.SourceTable == sourceTable && strings.TrimSpace(item.Snippet) != "" {
			return
		}
	}
	t.Fatalf("expected evidence type=%s source_table=%s in %+v", evidenceType, sourceTable, evidence)
}
