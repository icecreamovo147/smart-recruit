package persistence

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
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

	educationStart := time.Date(2012, 9, 1, 0, 0, 0, 0, time.UTC)
	educationEnd := time.Date(2016, 6, 1, 0, 0, 0, 0, time.UTC)
	experienceStart := time.Date(2019, 3, 1, 0, 0, 0, 0, time.UTC)
	projectStart := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	projectEnd := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	rawJSON := `{"full_name":"Ada Lovelace","skills":[{"name":"Go"}]}`
	snapshot, err := store.SaveRecruitingResumeProfileDraft(ctx, aiagentgrpc.RecruitingResumeProfileDraft{
		ResumeID:             8001,
		UserID:               3001,
		ParserVersion:        "native-resume-profile-parser-v1",
		InputHash:            "hash-new",
		RawJSON:              rawJSON,
		FullName:             "Ada Lovelace",
		Email:                "ada@example.com",
		Phone:                "+86-13800000000",
		Location:             "Shanghai",
		Headline:             "Backend engineer",
		Summary:              "Builds Go services",
		TotalExperienceYears: 6.5,
		HighestDegree:        "Bachelor",
		Educations: []aiagentgrpc.RecruitingResumeEducationRow{
			{School: "University", Degree: "BS", Major: "Mathematics", StartDate: &educationStart, EndDate: &educationEnd, Description: "Computing", SortOrder: 1},
		},
		Experiences: []aiagentgrpc.RecruitingResumeExperienceRow{
			{Company: "Engines", Title: "Senior Engineer", Location: "Shanghai", StartDate: &experienceStart, IsCurrent: 1, Description: "Platform", AchievementsJSON: `["reduced latency","scaled services"]`, SortOrder: 1},
		},
		Projects: []aiagentgrpc.RecruitingResumeProjectRow{
			{Name: "Runtime", Role: "Lead", StartDate: &projectStart, EndDate: &projectEnd, Description: "Recruiting runtime", TechnologiesJSON: `["Go","gRPC"]`, HighlightsJSON: `["transactional"]`, SortOrder: 1},
		},
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
	profile := snapshot.Profile
	if profile.ResumeID != 8001 || profile.UserID != 3001 || profile.ParseRunID != snapshot.ParseRun.ID || profile.RawJSON != rawJSON || profile.FullName != "Ada Lovelace" || profile.Email != "ada@example.com" || profile.Phone != "+86-13800000000" || profile.Location != "Shanghai" || profile.Headline != "Backend engineer" || profile.Summary != "Builds Go services" || profile.TotalExperienceYears != 6.5 || profile.HighestDegree != "Bachelor" {
		t.Fatalf("saved profile mapping = %+v", profile)
	}
	if len(snapshot.Educations) != 1 || snapshot.Educations[0].School != "University" || snapshot.Educations[0].Degree != "BS" || snapshot.Educations[0].Major != "Mathematics" || !snapshot.Educations[0].StartDate.Equal(educationStart) || !snapshot.Educations[0].EndDate.Equal(educationEnd) || snapshot.Educations[0].Description != "Computing" || snapshot.Educations[0].SortOrder != 1 {
		t.Fatalf("education mapping = %+v", snapshot.Educations)
	}
	if len(snapshot.Experiences) != 1 || snapshot.Experiences[0].Company != "Engines" || snapshot.Experiences[0].Title != "Senior Engineer" || snapshot.Experiences[0].Location != "Shanghai" || !snapshot.Experiences[0].StartDate.Equal(experienceStart) || snapshot.Experiences[0].EndDate != nil || snapshot.Experiences[0].IsCurrent != 1 || snapshot.Experiences[0].Description != "Platform" || snapshot.Experiences[0].AchievementsJSON != `["reduced latency","scaled services"]` || snapshot.Experiences[0].SortOrder != 1 {
		t.Fatalf("experience mapping = %+v", snapshot.Experiences)
	}
	if len(snapshot.Projects) != 1 || snapshot.Projects[0].Name != "Runtime" || snapshot.Projects[0].Role != "Lead" || !snapshot.Projects[0].StartDate.Equal(projectStart) || !snapshot.Projects[0].EndDate.Equal(projectEnd) || snapshot.Projects[0].Description != "Recruiting runtime" || snapshot.Projects[0].TechnologiesJSON != `["Go","gRPC"]` || snapshot.Projects[0].HighlightsJSON != `["transactional"]` || snapshot.Projects[0].SortOrder != 1 {
		t.Fatalf("project mapping = %+v", snapshot.Projects)
	}
	if len(snapshot.Skills) != 1 || snapshot.Skills[0].Name != "Go" || snapshot.Skills[0].Category != "language" || snapshot.Skills[0].Level != "senior" || snapshot.Skills[0].Years != 5 || snapshot.Skills[0].Evidence != "services" || snapshot.Skills[0].SortOrder != 1 {
		t.Fatalf("skill mapping = %+v", snapshot.Skills)
	}
	var old recruitingResumeProfileRecord
	if err := db.Where("id = ?", uint64(6001)).First(&old).Error; err != nil {
		t.Fatalf("load old profile: %v", err)
	}
	if old.IsCurrent != 0 {
		t.Fatalf("old profile is_current = %d, want 0", old.IsCurrent)
	}
}

func TestNativeStoreSaveRecruitingResumeProfileDraftRollsBackEveryTableOnChildFailure(t *testing.T) {
	ctx := context.Background()
	db := newRecruitingGenerationTestDB(t)
	store := NewNativeStore(db)
	now := time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC)
	seedRecruitingGenerationBaseRows(t, db, now)
	if err := db.Exec(`CREATE TRIGGER fail_resume_skill_insert BEFORE INSERT ON resume_skills BEGIN SELECT RAISE(ABORT, 'forced child insert failure'); END`).Error; err != nil {
		t.Fatalf("create child failure trigger: %v", err)
	}
	before := recruitingResumeTableCounts(t, db)
	date := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := store.SaveRecruitingResumeProfileDraft(ctx, aiagentgrpc.RecruitingResumeProfileDraft{
		ResumeID: 8001, UserID: 3001, ParserVersion: "parser-new", InputHash: "hash-new", FullName: "New Ada",
		Educations:  []aiagentgrpc.RecruitingResumeEducationRow{{School: "University", StartDate: &date, SortOrder: 1}},
		Experiences: []aiagentgrpc.RecruitingResumeExperienceRow{{Company: "Engines", StartDate: &date, AchievementsJSON: `[]`, SortOrder: 1}},
		Projects:    []aiagentgrpc.RecruitingResumeProjectRow{{Name: "Runtime", StartDate: &date, TechnologiesJSON: `[]`, HighlightsJSON: `[]`, SortOrder: 1}},
		Skills:      []aiagentgrpc.RecruitingResumeSkillRow{{Name: "Go", SortOrder: 1}},
	})
	if err == nil {
		t.Fatal("SaveRecruitingResumeProfileDraft error = nil, want child insertion failure")
	}
	after := recruitingResumeTableCounts(t, db)
	if before != after {
		t.Fatalf("table counts after rollback = %+v, want %+v", after, before)
	}
	var old recruitingResumeProfileRecord
	if err := db.Where("id = ?", uint64(6001)).First(&old).Error; err != nil {
		t.Fatalf("load old profile: %v", err)
	}
	if old.IsCurrent != 1 || old.Version != 1 || old.FullName.String != "Ada" {
		t.Fatalf("old current/version changed after rollback = %+v", old)
	}
	var newVersionCount int64
	if err := db.Model(&recruitingResumeProfileRecord{}).Where("resume_id = ? AND version = ?", 8001, 2).Count(&newVersionCount).Error; err != nil || newVersionCount != 0 {
		t.Fatalf("new version count=%d error=%v, want zero", newVersionCount, err)
	}
}

func TestNativeStoreStructuredAdapterClassifiesAIEmptyReplyForResumeFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeStructuredTestResponse(t, w, "")
	}))
	t.Cleanup(server.Close)
	store := newStructuredRuntimeTestStore(t, server.URL+"/v1", "empty-model", 1, 5)
	sqlDB, err := store.db.DB()
	if err != nil {
		t.Fatalf("get structured runtime sql db: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close structured runtime sqlite: %v", err)
		}
	})
	if err := store.db.AutoMigrate(&promptTemplateRecord{}); err != nil {
		t.Fatalf("migrate prompt: %v", err)
	}
	prompt := promptTemplateRecord{ID: 71, Name: "resume", Content: "database resume system", Version: 2, IsActive: true, AgentType: recruitingruntime.AgentTypeResumeProfileExtractor, PromptRole: recruitingruntime.PromptRoleSystem, UpdatedAt: time.Now().UTC()}
	if err := store.db.Create(&prompt).Error; err != nil {
		t.Fatalf("seed prompt: %v", err)
	}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: true, ResumeParseTimeout: time.Second})
	runtime := recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(store), store, policy)
	result, err := recruitingruntime.NewResumeProfileExtractor(runtime, policy).Extract(context.Background(), recruitingruntime.ResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Ada Lovelace\nGo engineer"})
	if err != nil {
		t.Fatalf("Extract error = %v", err)
	}
	if !result.FallbackUsed || result.FallbackKind != recruitingruntime.ResumeExtractionEmpty {
		t.Fatalf("fallback result = %+v, want empty_response through NativeStore/Commons adapter", result)
	}
}

type recruitingResumeCounts struct {
	ParseRuns   int64
	Profiles    int64
	Educations  int64
	Experiences int64
	Projects    int64
	Skills      int64
}

func recruitingResumeTableCounts(t *testing.T, db *gorm.DB) recruitingResumeCounts {
	t.Helper()
	counts := recruitingResumeCounts{}
	for model, destination := range map[any]*int64{
		&recruitingResumeParseRunRecord{}:   &counts.ParseRuns,
		&recruitingResumeProfileRecord{}:    &counts.Profiles,
		&recruitingResumeEducationRecord{}:  &counts.Educations,
		&recruitingResumeExperienceRecord{}: &counts.Experiences,
		&recruitingResumeProjectRecord{}:    &counts.Projects,
		&recruitingResumeSkillRecord{}:      &counts.Skills,
	} {
		if err := db.Model(model).Count(destination).Error; err != nil {
			t.Fatalf("count %T: %v", model, err)
		}
	}
	return counts
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
	if source.CandidateProfile == nil || source.CandidateProfile.ID != 1 || source.CandidateProfile.RealName != "Ada Lovelace" ||
		source.CandidateProfile.Phone != "13800000000" || source.CandidateProfile.Education != "本科" ||
		source.CandidateProfile.School != "Analytical University" || source.CandidateProfile.WorkExperience != "五年后端经验" ||
		source.CandidateProfile.Skills != "Go, gRPC" || source.CandidateProfile.IsComplete != 1 {
		t.Fatalf("candidate profile source = %+v, want complete persisted candidate profile", source.CandidateProfile)
	}
	if source.ApplicationEducation != "本科" || source.ResumeParsedText != "Ada builds Go services." {
		t.Fatalf("education/resume source = %q/%q, want candidate profile education and application resume parsed text", source.ApplicationEducation, source.ResumeParsedText)
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

func TestNativeStoreGetRecruitingMatchSourcePreservesMissingProfileAndResumeTextSemantics(t *testing.T) {
	ctx := context.Background()
	db := newRecruitingGenerationTestDB(t)
	store := NewNativeStore(db)
	seedRecruitingGenerationBaseRows(t, db, time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC))
	if err := db.Delete(&recruitingCandidateProfileTestRecord{}, 1).Error; err != nil {
		t.Fatalf("delete candidate profile: %v", err)
	}
	if err := db.Model(&recruitingGenerationResumeTestRecord{}).Where("id = ?", 8001).Update("parsed_text", "").Error; err != nil {
		t.Fatalf("clear application resume parsed text: %v", err)
	}

	source, found, err := store.GetRecruitingMatchSource(ctx, 7001)
	if err != nil || !found {
		t.Fatalf("GetRecruitingMatchSource source=%+v found=%v err=%v", source, found, err)
	}
	if source.CandidateProfile != nil || source.ApplicationEducation != "" || source.ResumeParsedText != "" {
		t.Fatalf("missing semantics = profile:%+v education:%q parsed:%q", source.CandidateProfile, source.ApplicationEducation, source.ResumeParsedText)
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
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close recruiting generation sqlite: %v", err)
		}
	})
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
	for _, statement := range []string{
		"ALTER TABLE candidate_profiles ADD COLUMN phone TEXT",
		"ALTER TABLE candidate_profiles ADD COLUMN education TEXT",
		"ALTER TABLE candidate_profiles ADD COLUMN school TEXT",
		"ALTER TABLE candidate_profiles ADD COLUMN work_experience TEXT",
		"ALTER TABLE candidate_profiles ADD COLUMN skills TEXT",
		"ALTER TABLE candidate_profiles ADD COLUMN is_complete INTEGER NOT NULL DEFAULT 0",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("extend candidate profile fixture: %v", err)
		}
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
	if err := db.Exec(`UPDATE candidate_profiles SET phone = ?, education = ?, school = ?, work_experience = ?, skills = ?, is_complete = ? WHERE id = ?`,
		"13800000000", "本科", "Analytical University", "五年后端经验", "Go, gRPC", 1, 1).Error; err != nil {
		t.Fatalf("seed complete candidate profile: %v", err)
	}
	if err := db.Create(&recruitingGenerationResumeTestRecord{ID: 8001, UserID: 3001, FileName: "ada.pdf", ParsedText: "Ada builds Go services.", IsValid: 1}).Error; err != nil {
		t.Fatalf("seed resume: %v", err)
	}
	if err := db.Create(&recruitingGenerationResumeTestRecord{ID: 8002, UserID: 3001, FileName: "other.pdf", ParsedText: "WRONG RESUME", IsValid: 1}).Error; err != nil {
		t.Fatalf("seed alternate resume: %v", err)
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
