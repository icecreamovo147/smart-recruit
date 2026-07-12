package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

type stubResumeProfileExtractor struct {
	output string
	err    error
}

func (s stubResumeProfileExtractor) Extract(ctx context.Context, text string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.output, nil
}

type cancelingResumeProfileExtractor struct {
	output string
	cancel context.CancelFunc
}

func (s cancelingResumeProfileExtractor) Extract(ctx context.Context, text string) (string, error) {
	if s.cancel != nil {
		s.cancel()
	}
	return s.output, nil
}

func TestResumeProfileServiceParseResumeStoresNormalizedSnapshot(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx := context.Background()
	resume := seedResumeProfileServiceResume(t, db, "  Resume text  ")

	raw := `{
		"full_name": "  Ada   Lovelace ",
		"email": "ADA@EXAMPLE.COM",
		"phone": "  +1 555 0100 ",
		"location": " London ",
		"headline": " Backend   Engineer ",
		"summary": " Builds   systems ",
		"total_experience_years": 5.46,
		"highest_degree": " BS ",
		"educations": [
			{"school": " Example   University ", "degree": "BS", "major": "Computer Science", "start_date": "2012", "end_date": "2016-06", "description": " Honors "}
		],
		"experiences": [
			{"company": " Example   Co ", "title": "Senior Engineer", "location": "Remote", "start_date": "2019-01", "end_date": "present", "is_current": true, "description": " Platform ", "achievements": [" scaled   services ", ""]}
		],
		"projects": [
			{"name": " Recruiter   AI ", "role": "Lead", "start_date": "2023-02-03", "end_date": "", "description": " Evidence backed project ", "technologies": ["golang", " Vue "], "highlights": ["reduced manual review"]}
		],
		"skills": [
			{"name": "golang", "category": " Language ", "level": " Advanced ", "years": 3.24, "evidence": " Example Co "},
			{"name": " Go ", "category": "backend", "level": "", "years": 4.01, "evidence": " Recruiter AI "},
			{"name": " TypeScript ", "category": "language", "level": "intermediate", "years": 2, "evidence": "UI work"}
		]
	}`

	svc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		stubResumeProfileExtractor{output: raw},
	)
	svc.now = func() time.Time { return time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC) }

	snapshot, err := svc.ParseResume(ctx, resume.ID)
	if err != nil {
		t.Fatalf("ParseResume failed: %v", err)
	}
	if snapshot.ParseRun.Status != "succeeded" || snapshot.ParseRun.ErrorMessage != "" {
		t.Fatalf("unexpected parse run: %+v", snapshot.ParseRun)
	}
	if snapshot.Profile.FullName != "Ada Lovelace" {
		t.Fatalf("expected normalized full name, got %q", snapshot.Profile.FullName)
	}
	if snapshot.Profile.Email != "ada@example.com" {
		t.Fatalf("expected lower-case email, got %q", snapshot.Profile.Email)
	}
	if snapshot.Profile.TotalExperience != 5.5 {
		t.Fatalf("expected rounded experience 5.5, got %v", snapshot.Profile.TotalExperience)
	}
	if len(snapshot.Educations) != 1 || snapshot.Educations[0].School != "Example University" || snapshot.Educations[0].SortOrder != 1 {
		t.Fatalf("unexpected educations: %+v", snapshot.Educations)
	}
	if snapshot.Educations[0].StartDate == nil || snapshot.Educations[0].StartDate.Format("2006-01-02") != "2012-01-01" {
		t.Fatalf("unexpected education start date: %+v", snapshot.Educations[0].StartDate)
	}
	if len(snapshot.Experiences) != 1 || snapshot.Experiences[0].Company != "Example Co" || snapshot.Experiences[0].IsCurrent != 1 || snapshot.Experiences[0].EndDate != nil {
		t.Fatalf("unexpected experiences: %+v", snapshot.Experiences)
	}
	if snapshot.Experiences[0].AchievementsJSON != `["scaled services"]` {
		t.Fatalf("unexpected achievements JSON: %s", snapshot.Experiences[0].AchievementsJSON)
	}
	if len(snapshot.Projects) != 1 || snapshot.Projects[0].Name != "Recruiter AI" || snapshot.Projects[0].TechnologiesJSON != `["golang","Vue"]` {
		t.Fatalf("unexpected projects: %+v", snapshot.Projects)
	}
	if len(snapshot.Skills) != 2 {
		t.Fatalf("expected deduplicated skills, got %+v", snapshot.Skills)
	}
	if snapshot.Skills[0].Name != "Go" || snapshot.Skills[0].Years != 4 || snapshot.Skills[0].Evidence != "Example Co; Recruiter AI" {
		t.Fatalf("unexpected Go skill: %+v", snapshot.Skills[0])
	}
	if snapshot.Skills[1].Name != "TypeScript" {
		t.Fatalf("expected TypeScript second after stable sort, got %+v", snapshot.Skills)
	}

	current, err := repository.NewResumeProfileRepo(db).GetCurrentByResumeID(ctx, resume.ID)
	if err != nil {
		t.Fatalf("GetCurrentByResumeID failed: %v", err)
	}
	if current == nil || current.ID != snapshot.Profile.ID {
		t.Fatalf("expected persisted current profile, got %+v", current)
	}
}

func TestParseResumeDateNormalizesCommonLLMDateVariants(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		endDate bool
		want    string
		wantNil bool
		wantErr bool
	}{
		{name: "year", value: "2012", want: "2012-01-01"},
		{name: "year month slash", value: "2012/9", want: "2012-09-01"},
		{name: "year month dot", value: "2012.09", want: "2012-09-01"},
		{name: "chinese year month", value: "2012年9月", want: "2012-09-01"},
		{name: "chinese full date", value: "2012年9月3日", want: "2012-09-03"},
		{name: "range start", value: "2012.09-2016.06", want: "2012-09-01"},
		{name: "range end", value: "2012.09-2016.06", endDate: true, want: "2016-06-01"},
		{name: "year range start", value: "2012-2016", want: "2012-01-01"},
		{name: "year range end", value: "2012-2016", endDate: true, want: "2016-01-01"},
		{name: "open ended english", value: "present", wantNil: true},
		{name: "open ended chinese", value: "至今", endDate: true, wantNil: true},
		{name: "month year remains invalid", value: "01-2020", wantErr: true},
		{name: "invalid month", value: "2020-13", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var (
				got *time.Time
				err error
			)
			if tc.endDate {
				got, err = parseResumeEndDate(tc.value)
			} else {
				got, err = parseResumeDate(tc.value)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil date, got %s", got.Format("2006-01-02"))
				}
				return
			}
			if got == nil || got.Format("2006-01-02") != tc.want {
				t.Fatalf("expected %s, got %+v", tc.want, got)
			}
		})
	}
}

func TestResumeProfileServiceParseResumePersistsWhenRequestContextCanceledAfterExtraction(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	resume := seedResumeProfileServiceResume(t, db, "Resume text")

	raw := `{
		"full_name": "Fallback Person",
		"email": "fallback@example.com",
		"phone": "",
		"location": "",
		"headline": "Backend Engineer",
		"summary": "Parsed by fallback",
		"total_experience_years": 3,
		"highest_degree": "",
		"educations": [],
		"experiences": [],
		"projects": [],
		"skills": [{"name": "Go", "category": "language", "level": "advanced", "years": 3, "evidence": "resume"}]
	}`

	svc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		cancelingResumeProfileExtractor{output: raw, cancel: cancel},
	)

	snapshot, err := svc.ParseResume(ctx, resume.ID)
	if err != nil {
		t.Fatalf("ParseResume failed after extractor canceled request context: %v", err)
	}
	if snapshot.Profile.FullName != "Fallback Person" {
		t.Fatalf("expected persisted fallback profile, got %+v", snapshot.Profile)
	}
	current, err := repository.NewResumeProfileRepo(db).GetCurrentByResumeID(context.Background(), resume.ID)
	if err != nil {
		t.Fatalf("GetCurrentByResumeID failed: %v", err)
	}
	if current == nil || current.FullName != "Fallback Person" {
		t.Fatalf("expected current fallback profile, got %+v", current)
	}
}

func TestResumeProfileServiceParseResumeStoresFailureForMissingText(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx := context.Background()
	resume := seedResumeProfileServiceResume(t, db, "   ")

	svc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		stubResumeProfileExtractor{output: `{}`},
	)

	_, err := svc.ParseResume(ctx, resume.ID)
	if !errors.Is(err, ErrResumeProfileMissingText) {
		t.Fatalf("expected missing text error, got %v", err)
	}
	assertResumeProfileServiceFailureRun(t, db, resume.ID, "failed", "parsed_text is empty")
}

func TestResumeProfileServiceParseResumeDisabledByRuntimePolicy(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx := context.Background()
	resume := seedResumeProfileServiceResume(t, db, "resume text")

	svc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		stubResumeProfileExtractor{output: `{}`},
	).WithRuntimePolicy(AgentRuntimePolicy{
		StructuredResumeParse: false,
		CandidateMatch:        true,
		SemanticRetrieval:     true,
		MCPPolicy:             true,
		Planner:               true,
		SkillGovernance:       true,
		Fallbacks:             true,
	})

	_, err := svc.ParseResume(ctx, resume.ID)
	if !errors.Is(err, ErrAgentCapabilityDisabled) {
		t.Fatalf("expected ErrAgentCapabilityDisabled, got %v", err)
	}
	var count int64
	if err := db.Model(&model.ResumeParseRun{}).Count(&count).Error; err != nil {
		t.Fatalf("count parse runs failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected disabled parse to skip persistence, got %d parse runs", count)
	}
}

func TestResumeProfileServiceParseResumeRejectsMalformedAndIncompleteOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantErr string
	}{
		{name: "malformed JSON", output: `{"full_name":`, wantErr: "unexpected EOF"},
		{name: "unknown field", output: `{"full_name":"Ada","total_experience_years":1,"unknown":true,"skills":[{"name":"Go"}]}`, wantErr: "unknown field"},
		{name: "missing required field", output: `{"full_name":"Ada","skills":[{"name":"Go"}]}`, wantErr: "total_experience_years is required"},
		{name: "empty profile evidence", output: `{"full_name":"Ada","total_experience_years":1,"educations":[],"experiences":[],"projects":[],"skills":[]}`, wantErr: "at least one education"},
		{name: "invalid date", output: `{"full_name":"Ada","total_experience_years":1,"experiences":[{"company":"Example","start_date":"01-2020"}]}`, wantErr: "expected YYYY"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := setupResumeProfileServiceTestDB(t)
			ctx := context.Background()
			resume := seedResumeProfileServiceResume(t, db, "resume text")
			svc := NewResumeProfileService(
				repository.NewResumeRepo(db),
				repository.NewResumeProfileRepo(db),
				stubResumeProfileExtractor{output: tc.output},
			)

			_, err := svc.ParseResume(ctx, resume.ID)
			if !errors.Is(err, ErrResumeProfileInvalidOutput) {
				t.Fatalf("expected invalid output error, got %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error to contain %q, got %v", tc.wantErr, err)
			}
			assertResumeProfileServiceFailureRun(t, db, resume.ID, "failed", tc.wantErr)
		})
	}
}

func TestResumeProfileServiceParseResumeDoesNotStoreRunForMissingResume(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	svc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		stubResumeProfileExtractor{output: `{}`},
	)

	_, err := svc.ParseResume(context.Background(), 404)
	if !errors.Is(err, ErrResumeProfileMissingResume) {
		t.Fatalf("expected missing resume error, got %v", err)
	}
	var count int64
	if err := db.Model(&model.ResumeParseRun{}).Count(&count).Error; err != nil {
		t.Fatalf("count parse runs failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no parse run without resume owner, got %d", count)
	}
}

func TestResumeProfileServiceParseResumeStoresFailureForExtractorError(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx := context.Background()
	resume := seedResumeProfileServiceResume(t, db, "resume text")
	svc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		stubResumeProfileExtractor{err: errors.New("extractor unavailable")},
	)

	_, err := svc.ParseResume(ctx, resume.ID)
	if err == nil || !strings.Contains(err.Error(), "extract resume profile") {
		t.Fatalf("expected extractor error, got %v", err)
	}
	assertResumeProfileServiceFailureRun(t, db, resume.ID, "failed", "extractor unavailable")
}

func setupResumeProfileServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Resume{},
		&model.ResumeParseRun{},
		&model.ResumeProfile{},
		&model.ResumeEducation{},
		&model.ResumeExperience{},
		&model.ResumeProject{},
		&model.ResumeSkill{},
	); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}

func seedResumeProfileServiceResume(t *testing.T, db *gorm.DB, parsedText string) *model.Resume {
	t.Helper()
	now := time.Now()
	resume := &model.Resume{
		UserID:     42,
		OSSKey:     "resumes/42/test.pdf",
		FileName:   "test.pdf",
		FileType:   "pdf",
		FileSize:   1000,
		ParsedText: parsedText,
		ParsedAt:   &now,
		IsValid:    1,
		UploadedAt: now,
	}
	if err := db.Create(resume).Error; err != nil {
		t.Fatalf("seed resume failed: %v", err)
	}
	return resume
}

func assertResumeProfileServiceFailureRun(t *testing.T, db *gorm.DB, resumeID int64, status, errorText string) {
	t.Helper()
	var run model.ResumeParseRun
	if err := db.Where("resume_id = ?", resumeID).Order("id DESC").First(&run).Error; err != nil {
		t.Fatalf("load parse run failed: %v", err)
	}
	if run.Status != status {
		t.Fatalf("expected status %q, got %+v", status, run)
	}
	if run.CompletedAt == nil {
		t.Fatalf("expected completed_at on failure run: %+v", run)
	}
	if !strings.Contains(run.ErrorMessage, errorText) {
		t.Fatalf("expected error message to contain %q, got %q", errorText, run.ErrorMessage)
	}
	var count int64
	if err := db.Model(&model.ResumeProfile{}).Where("parse_run_id = ?", run.ID).Count(&count).Error; err != nil {
		t.Fatalf("count failure profiles failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected failure run to avoid empty profile storage, got %d profile rows", count)
	}
}
