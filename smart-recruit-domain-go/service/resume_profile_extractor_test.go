package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/pkg/authz"
	"smart-recruit-domain-go/repository"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestCleanLLMJSONOutput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "plain JSON",
			input: `{"full_name": "Ada"}`,
			want:  `{"full_name": "Ada"}`,
		},
		{
			name:  "json code block",
			input: "```json\n{\"full_name\": \"Ada\"}\n```",
			want:  `{"full_name": "Ada"}`,
		},
		{
			name:  "code block without json marker",
			input: "```\n{\"full_name\": \"Ada\"}\n```",
			want:  `{"full_name": "Ada"}`,
		},
		{
			name:  "text before and after JSON",
			input: "Here is the extracted profile:\n\n{\"full_name\": \"Ada\"}\n\nI hope this helps.",
			want:  `{"full_name": "Ada"}`,
		},
		{
			name:  "text before code block JSON and after",
			input: "Here:\n```json\n{\"full_name\": \"Ada\"}\n```\nDone.",
			want:  `{"full_name": "Ada"}`,
		},
		{
			name:  "nested braces in content",
			input: "```json\n{\"summary\": \"works at {company}\", \"skills\": [{\"name\": \"Go\"}]}\n```",
			want:  `{"summary": "works at {company}", "skills": [{"name": "Go"}]}`,
		},
		{
			name:    "no JSON at all",
			input:   "this is just text without any JSON",
			wantErr: true,
		},
		{
			name:    "only opening brace",
			input:   `{"full_name":`,
			wantErr: true,
		},
		{
			name:  "empty JSON object",
			input: `{}`,
			want:  `{}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cleanLLMJSONOutput(tc.input)
			if tc.wantErr && got == "" {
				return
			}
			if tc.wantErr && got != "" {
				t.Fatalf("expected empty result, got %q", got)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestQuickValidateResumeProfileJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid complete profile",
			input: `{
				"full_name": "Ada Lovelace",
				"email": "ada@example.com",
				"phone": "+1-555-0100",
				"location": "London",
				"headline": "Engineer",
				"summary": "A summary",
				"total_experience_years": 5.5,
				"highest_degree": "BS",
				"educations": [{"school": "MIT", "degree": "BS", "major": "CS", "start_date": "2012", "end_date": "2016", "description": ""}],
				"experiences": [{"company": "Acme", "title": "Engineer", "start_date": "2016", "end_date": "present", "is_current": true, "description": "", "achievements": []}],
				"projects": [{"name": "Proj", "role": "Dev", "start_date": "2020", "end_date": "", "description": "", "technologies": [], "highlights": []}],
				"skills": [{"name": "Go", "category": "language", "level": "advanced", "years": 3, "evidence": "Acme"}]
			}`,
		},
		{
			name: "missing total_experience_years",
			input: `{
				"full_name": "Ada",
				"educations": [],
				"experiences": [],
				"projects": [],
				"skills": []
			}`,
			wantErr: true,
		},
		{
			name: "missing educations",
			input: `{
				"full_name": "Ada",
				"total_experience_years": 0,
				"experiences": [],
				"projects": [],
				"skills": []
			}`,
			wantErr: true,
		},
		{
			name: "missing experiences",
			input: `{
				"full_name": "Ada",
				"total_experience_years": 0,
				"educations": [],
				"projects": [],
				"skills": []
			}`,
			wantErr: true,
		},
		{
			name: "missing projects",
			input: `{
				"full_name": "Ada",
				"total_experience_years": 0,
				"educations": [],
				"experiences": [],
				"skills": []
			}`,
			wantErr: true,
		},
		{
			name: "missing skills",
			input: `{
				"full_name": "Ada",
				"total_experience_years": 0,
				"educations": [],
				"experiences": [],
				"projects": []
			}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `{"full_name":`,
			wantErr: true,
		},
		{
			name: "unknown field allowed in quick validation",
			input: `{
				"full_name": "Ada",
				"total_experience_years": 0,
				"educations": [],
				"experiences": [],
				"projects": [],
				"skills": [],
				"unknown_field": "bad"
			}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := quickValidateResumeProfileJSON(tc.input)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestHeuristicResumeProfileExtractorExtract(t *testing.T) {
	e := NewHeuristicResumeProfileExtractor()

	t.Run("extracts email and phone", func(t *testing.T) {
		text := "John Doe\njohn@example.com\n555-123-4567\nSome bio here."
		result, err := e.Extract(context.Background(), text)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "john@example.com") {
			t.Fatalf("expected email in result, got %s", result)
		}
		if !strings.Contains(result, "5551234567") {
			t.Fatalf("expected phone in result, got %s", result)
		}
	})

	t.Run("always returns valid JSON", func(t *testing.T) {
		result, err := e.Extract(context.Background(), "Some random text without any structured info.")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := quickValidateResumeProfileJSON(result); err != nil {
			t.Fatalf("heuristic output should pass quick-validate: %v", err)
		}
	})

	t.Run("extracts skills from text", func(t *testing.T) {
		text := "Skills: Golang, Python, Kubernetes, Docker"
		result, err := e.Extract(context.Background(), text)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "golang") {
			t.Fatalf("expected golang skill in result, got %s", result)
		}
		if !strings.Contains(result, "python") {
			t.Fatalf("expected python skill in result, got %s", result)
		}
	})
}

func TestFallbackResumeProfileExtractorPrimarySuccess(t *testing.T) {
	primary := stubResumeProfileExtractor{output: `{"full_name":"LLM Result","total_experience_years":5,"educations":[],"experiences":[],"projects":[],"skills":[]}`}
	fallback := stubResumeProfileExtractor{output: `{"full_name":"Heuristic Result","total_experience_years":0,"educations":[],"experiences":[],"projects":[],"skills":[]}`}
	f := NewFallbackResumeProfileExtractor(primary, fallback)

	result, err := f.Extract(context.Background(), "some text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "LLM Result") {
		t.Fatalf("expected primary result, got %s", result)
	}
}

func TestFallbackResumeProfileExtractorPrimaryFailsUsesFallback(t *testing.T) {
	primary := stubResumeProfileExtractor{err: errors.New("llm unavailable")}
	fallback := stubResumeProfileExtractor{output: `{"full_name":"Heuristic Result","total_experience_years":0,"educations":[],"experiences":[],"projects":[],"skills":[]}`}
	f := NewFallbackResumeProfileExtractor(primary, fallback)

	result, err := f.Extract(context.Background(), "some text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Heuristic Result") {
		t.Fatalf("expected fallback result, got %s", result)
	}
}

func TestFallbackResumeProfileExtractorBothFail(t *testing.T) {
	primary := stubResumeProfileExtractor{err: errors.New("primary failed")}
	fallback := stubResumeProfileExtractor{err: errors.New("fallback also failed")}
	f := NewFallbackResumeProfileExtractor(primary, fallback)

	_, err := f.Extract(context.Background(), "some text")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "primary failed") || !strings.Contains(err.Error(), "fallback also failed") {
		t.Fatalf("expected both errors in message, got %v", err)
	}
}

func TestFallbackResumeProfileExtractorDisabled(t *testing.T) {
	primary := stubResumeProfileExtractor{err: errors.New("llm unavailable")}
	fallback := stubResumeProfileExtractor{output: `{"full_name":"Heuristic","total_experience_years":0,"educations":[],"experiences":[],"projects":[],"skills":[]}`}
	f := NewFallbackResumeProfileExtractor(primary, fallback, WithFallbackEnabled(false))

	_, err := f.Extract(context.Background(), "some text")
	if err == nil {
		t.Fatalf("expected error when fallback disabled, got nil")
	}
}

func TestResumeProfileServiceWithLLMExtractor(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx := context.Background()
	resume := seedResumeProfileServiceResume(t, db, "Resume text for LLM extraction")

	rawOutput := `{
		"full_name": "Alice Smith",
		"email": "alice@example.com",
		"phone": "+1-555-0100",
		"location": "New York",
		"headline": "Senior Engineer",
		"summary": "Experienced backend engineer",
		"total_experience_years": 8.2,
		"highest_degree": "MS",
		"educations": [
			{"school": "Columbia University", "degree": "MS", "major": "CS", "start_date": "2014", "end_date": "2016", "description": "Thesis on distributed systems"}
		],
		"experiences": [
			{"company": "Tech Corp", "title": "Senior Engineer", "location": "NYC", "start_date": "2018", "end_date": "present", "is_current": true, "description": "Backend team", "achievements": ["Scaled system to 1M users"]}
		],
		"projects": [
			{"name": "Distributed Cache", "role": "Lead", "start_date": "2022", "end_date": "2023", "description": "In-memory cache layer", "technologies": ["Go", "Redis"], "highlights": ["Reduced latency 50%"]}
		],
		"skills": [
			{"name": "Go", "category": "language", "level": "expert", "years": 6, "evidence": "Tech Corp"}
		]
	}`

	svc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		stubResumeProfileExtractor{output: rawOutput},
	)
	svc.now = func() time.Time { return time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC) }

	snapshot, err := svc.ParseResume(ctx, resume.ID)
	if err != nil {
		t.Fatalf("ParseResume failed: %v", err)
	}
	if snapshot.ParseRun.Status != "succeeded" {
		t.Fatalf("expected succeeded, got %s", snapshot.ParseRun.Status)
	}
	if snapshot.Profile.FullName != "Alice Smith" {
		t.Fatalf("expected 'Alice Smith', got %q", snapshot.Profile.FullName)
	}
	if snapshot.Profile.TotalExperience != 8.2 {
		t.Fatalf("expected 8.2, got %v", snapshot.Profile.TotalExperience)
	}
	if len(snapshot.Educations) != 1 || snapshot.Educations[0].School != "Columbia University" {
		t.Fatalf("unexpected educations: %+v", snapshot.Educations)
	}
}

func TestResumeProfileServiceWithLLMExtractorSavesFailure(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx := context.Background()
	resume := seedResumeProfileServiceResume(t, db, "Resume text")

	svc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		stubResumeProfileExtractor{err: errors.New("llm call failed")},
	)

	_, err := svc.ParseResume(ctx, resume.ID)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	assertResumeProfileServiceFailureRun(t, db, resume.ID, "failed", "llm call failed")
}

func seedResumeProfileAndRBACTestDB(t *testing.T, ctx context.Context, db *gorm.DB) {
	t.Helper()

	// Migrate authz tables
	if err := db.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.UserRole{},
		&model.Permission{},
		&model.RolePermission{},
		&model.UserDataScope{},
		&model.Job{},
		&model.Application{},
		&model.Department{},
		&model.JobLocation{},
		&model.DepartmentLocation{},
		&model.AuthorizationAuditLog{},
		&model.Resume{},
		&model.ResumeParseRun{},
		&model.ResumeProfile{},
		&model.ResumeEducation{},
		&model.ResumeExperience{},
		&model.ResumeProject{},
		&model.ResumeSkill{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now()
	user := model.User{Username: "test_hr", Password: "hash", Role: 2, AccountType: "staff", TokenVersion: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	admin := model.User{Username: "test_admin", Password: "hash", Role: 3, AccountType: "staff", TokenVersion: 1}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	recruiterRole := model.Role{RoleKey: "recruiter", Name: "Recruiter", IsSystem: 1}
	adminRole := model.Role{RoleKey: "recruiting_admin", Name: "Recruiting Admin", IsSystem: 1}
	if err := db.Create(&recruiterRole).Error; err != nil {
		t.Fatalf("create recruiter role: %v", err)
	}
	if err := db.Create(&adminRole).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}

	adminIDA := uint64(admin.ID)
	ur1 := model.UserRole{UserID: uint64(user.ID), RoleID: recruiterRole.ID, AssignedBy: &adminIDA}
	ur2 := model.UserRole{UserID: uint64(admin.ID), RoleID: adminRole.ID, AssignedBy: &adminIDA}
	if err := db.Create(&ur1).Error; err != nil {
		t.Fatalf("assign recruiter: %v", err)
	}
	if err := db.Create(&ur2).Error; err != nil {
		t.Fatalf("assign admin: %v", err)
	}

	perms := []model.Permission{
		{PermissionKey: authz.PermJobCreate, Resource: "job", Action: "create", Description: "Create Jobs"},
		{PermissionKey: authz.PermJobPublish, Resource: "job", Action: "publish", Description: "Publish Jobs"},
		{PermissionKey: authz.PermApplicationStatusUpdate, Resource: "application", Action: "update_status", Description: "Update Application Status"},
		{PermissionKey: authz.PermApplicationRead, Resource: "application", Action: "read", Description: "Read Applications"},
		{PermissionKey: authz.PermAIHRUse, Resource: "ai", Action: "hr_use", Description: "Use HR AI Assistant"},
	}
	for i := range perms {
		if err := db.Create(&perms[i]).Error; err != nil {
			t.Fatalf("create perm %s: %v", perms[i].PermissionKey, err)
		}
	}
	for _, p := range perms {
		rp := model.RolePermission{RoleID: recruiterRole.ID, PermissionID: p.ID}
		if err := db.Create(&rp).Error; err != nil {
			t.Fatalf("assign perm %s: %v", p.PermissionKey, err)
		}
	}

	dept := model.Department{Name: "Engineering", FullName: "Engineering", IsActive: 1}
	loc := model.JobLocation{Name: "Beijing", IsActive: 1}
	if err := db.Create(&dept).Error; err != nil {
		t.Fatalf("create dept: %v", err)
	}
	if err := db.Create(&loc).Error; err != nil {
		t.Fatalf("create loc: %v", err)
	}

	deptID := int64(dept.ID)
	locID := int64(loc.ID)
	job := model.Job{
		HrID:         user.ID,
		Title:        "Software Engineer",
		DepartmentID: &deptID,
		LocationID:   &locID,
		Status:       1,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	resume := model.Resume{
		UserID:     user.ID,
		OSSKey:     "resumes/42/test.pdf",
		FileName:   "test.pdf",
		FileType:   "pdf",
		FileSize:   1000,
		ParsedText: "Resume text for parsing via ParseResumeProfile API",
		ParsedAt:   &now,
		IsValid:    1,
		UploadedAt: now,
	}
	if err := db.Create(&resume).Error; err != nil {
		t.Fatalf("create resume: %v", err)
	}

	application := model.Application{
		UserID:    user.ID,
		JobID:     job.ID,
		ResumeID:  resume.ID,
		Status:    1,
		StatusKey: "applied",
	}
	if err := db.Create(&application).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	// Assign department scope so the user can access jobs in Engineering
	authzRepo := repository.NewAuthzRepo(db)
	adminIDA = uint64(admin.ID)
	if err := authzRepo.AssignDataScope(ctx, uint64(user.ID), authz.ScopeDepartment, "department", uint64(deptID), &adminIDA); err != nil {
		t.Fatalf("assign dept scope: %v", err)
	}
}

func TestParseResumeProfileAPIMainline(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx := context.Background()
	seedResumeProfileAndRBACTestDB(t, ctx, db)

	// find the seeded resume
	var resume model.Resume
	if err := db.First(&resume).Error; err != nil {
		t.Fatalf("find seeded resume: %v", err)
	}
	var user model.User
	if err := db.Where("username = ?", "test_hr").First(&user).Error; err != nil {
		t.Fatalf("find test_hr user: %v", err)
	}

	rawOutput := `{
		"full_name": "Bob Johnson",
		"email": "bob@example.com",
		"phone": "",
		"location": "San Francisco",
		"headline": "Full Stack Developer",
		"summary": "5 years experience",
		"total_experience_years": 5.0,
		"highest_degree": "BS",
		"educations": [{"school": "Stanford", "degree": "BS", "major": "CS", "start_date": "2010", "end_date": "2014", "description": ""}],
		"experiences": [{"company": "Startup Inc", "title": "Developer", "location": "SF", "start_date": "2014", "end_date": "present", "is_current": true, "description": "Full stack", "achievements": ["Built product"]}],
		"projects": [{"name": "Web App", "role": "Dev", "start_date": "2020", "end_date": "", "description": "", "technologies": ["React"], "highlights": []}],
		"skills": [{"name": "React", "category": "frontend", "level": "advanced", "years": 3, "evidence": "Startup Inc"}]
	}`

	profileSvc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		stubResumeProfileExtractor{output: rawOutput},
	)
	svc := NewRecruitingIntelligenceService(
		repository.NewApplicationRepo(db),
		repository.NewJobRepo(db),
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		repository.NewCandidateMatchRepo(db),
		profileSvc,
		NewCandidateMatchService(
			repository.NewApplicationRepo(db),
			repository.NewJobRepo(db),
			repository.NewProfileRepo(db),
			repository.NewResumeRepo(db),
			repository.NewResumeProfileRepo(db),
			repository.NewCandidateMatchRepo(db),
		),
		NewServiceAuthorizer(repository.NewAuthzRepo(db), &scopeEvaluator{authzRepo: repository.NewAuthzRepo(db)}),
	)

	req := &pb.ParseResumeProfileRequest{
		StaffUserId: user.ID,
		ResumeId:    resume.ID,
	}
	resp, err := svc.ParseResumeProfile(ctx, req)
	if err != nil {
		t.Fatalf("ParseResumeProfile failed: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK code, got %d: %s", resp.Code, resp.Msg)
	}
	if resp.Profile == nil {
		t.Fatalf("expected non-nil profile in response")
	}
	if resp.Profile.Profile.FullName != "Bob Johnson" {
		t.Fatalf("expected 'Bob Johnson', got %q", resp.Profile.Profile.FullName)
	}
	if resp.Profile.ParseRun.Status != "succeeded" {
		t.Fatalf("expected succeeded parse run, got %s", resp.Profile.ParseRun.Status)
	}
}

func TestParseResumeProfileAPIMainlineWithApplicationID(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx := context.Background()
	seedResumeProfileAndRBACTestDB(t, ctx, db)

	var resume model.Resume
	if err := db.First(&resume).Error; err != nil {
		t.Fatalf("find seeded resume: %v", err)
	}
	var user model.User
	if err := db.Where("username = ?", "test_hr").First(&user).Error; err != nil {
		t.Fatalf("find test_hr user: %v", err)
	}
	var application model.Application
	if err := db.Where("resume_id = ?", resume.ID).First(&application).Error; err != nil {
		t.Fatalf("find seeded application: %v", err)
	}

	rawOutput := `{
		"full_name": "Carol Martinez",
		"email": "carol@example.com",
		"phone": "+86-138-0000-1111",
		"location": "Shanghai",
		"headline": "Backend Engineer",
		"summary": "4 years experience in distributed systems",
		"total_experience_years": 4.0,
		"highest_degree": "MS",
		"educations": [{"school": "Tsinghua", "degree": "MS", "major": "CS", "start_date": "2016", "end_date": "2019", "description": ""}],
		"experiences": [{"company": "Big Tech", "title": "SDE", "location": "Shanghai", "start_date": "2019", "end_date": "present", "is_current": true, "description": "Backend services", "achievements": ["Designed high-throughput API"]}],
		"projects": [{"name": "Distributed KV Store", "role": "Lead Dev", "start_date": "2021", "end_date": "2022", "description": "A distributed key-value store", "technologies": ["Go", "Raft"], "highlights": []}],
		"skills": [{"name": "Go", "category": "language", "level": "expert", "years": 4, "evidence": "Big Tech"}]
	}`

	profileSvc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		stubResumeProfileExtractor{output: rawOutput},
	)
	svc := NewRecruitingIntelligenceService(
		repository.NewApplicationRepo(db),
		repository.NewJobRepo(db),
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		repository.NewCandidateMatchRepo(db),
		profileSvc,
		NewCandidateMatchService(
			repository.NewApplicationRepo(db),
			repository.NewJobRepo(db),
			repository.NewProfileRepo(db),
			repository.NewResumeRepo(db),
			repository.NewResumeProfileRepo(db),
			repository.NewCandidateMatchRepo(db),
		),
		NewServiceAuthorizer(repository.NewAuthzRepo(db), &scopeEvaluator{authzRepo: repository.NewAuthzRepo(db)}),
	)

	req := &pb.ParseResumeProfileRequest{
		StaffUserId:   user.ID,
		ApplicationId: application.ID,
	}
	resp, err := svc.ParseResumeProfile(ctx, req)
	if err != nil {
		t.Fatalf("ParseResumeProfile failed: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK code, got %d: %s", resp.Code, resp.Msg)
	}
	if resp.Profile == nil {
		t.Fatalf("expected non-nil profile in response")
	}
	if resp.Profile.Profile.FullName != "Carol Martinez" {
		t.Fatalf("expected 'Carol Martinez', got %q", resp.Profile.Profile.FullName)
	}
	if resp.Profile.ParseRun.Status != "succeeded" {
		t.Fatalf("expected succeeded parse run, got %s", resp.Profile.ParseRun.Status)
	}
}

func TestFallbackResumeProfileExtractorMetadata(t *testing.T) {
	t.Run("primary success returns llm metadata", func(t *testing.T) {
		primary := &metadataCapturingExtractor{
			result: ResumeProfileExtractResult{
				RawJSON: `{"full_name":"LLM Person","total_experience_years":3,"educations":[{"school":"MIT"}],"experiences":[],"projects":[],"skills":[]}`,
				Metadata: ExtractorMetadata{
					ParserVersion: "resume-profile-llm-v1",
					ExtractorType: "llm",
					ModelName:     "gpt-4",
					PromptKey:     "test-prompt",
					PromptVersion: 1,
				},
			},
		}
		fallback := NewHeuristicResumeProfileExtractor()
		f := NewFallbackResumeProfileExtractor(primary, fallback)

		result, err := f.ExtractWithMetadata(context.Background(), "some text")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Metadata.ParserVersion != "resume-profile-llm-v1" {
			t.Fatalf("expected llm parser version, got %q", result.Metadata.ParserVersion)
		}
		if result.Metadata.ExtractorType != "llm" {
			t.Fatalf("expected llm extractor type, got %q", result.Metadata.ExtractorType)
		}
		if result.Metadata.FallbackUsed {
			t.Fatalf("expected fallbackUsed=false on primary success")
		}
	})

	t.Run("fallback returns heuristic metadata with FallbackUsed=true", func(t *testing.T) {
		primary := &metadataCapturingExtractor{
			err: errors.New("primary failed"),
		}
		fallback := NewHeuristicResumeProfileExtractor()
		f := NewFallbackResumeProfileExtractor(primary, fallback)

		result, err := f.ExtractWithMetadata(context.Background(), "some resume text here")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Metadata.ParserVersion != "resume-profile-heuristic-v1" {
			t.Fatalf("expected heuristic parser version, got %q", result.Metadata.ParserVersion)
		}
		if result.Metadata.ExtractorType != "heuristic_fallback" {
			t.Fatalf("expected heuristic_fallback extractor type, got %q", result.Metadata.ExtractorType)
		}
		if !result.Metadata.FallbackUsed {
			t.Fatalf("expected fallbackUsed=true on fallback")
		}
	})

	t.Run("fallback disabled returns primary error", func(t *testing.T) {
		primary := &metadataCapturingExtractor{
			err: errors.New("primary failed"),
		}
		fallback := NewHeuristicResumeProfileExtractor()
		f := NewFallbackResumeProfileExtractor(primary, fallback, WithFallbackEnabled(false))

		_, err := f.ExtractWithMetadata(context.Background(), "text")
		if err == nil {
			t.Fatalf("expected error when fallback disabled")
		}
	})

	t.Run("both fail returns error with version metadata", func(t *testing.T) {
		primary := &metadataCapturingExtractor{
			err: errors.New("primary failed"),
		}
		fallback := &metadataCapturingExtractor{
			err: errors.New("fallback also failed"),
		}
		f := NewFallbackResumeProfileExtractor(primary, fallback)

		_, err := f.ExtractWithMetadata(context.Background(), "text")
		if err == nil {
			t.Fatalf("expected error when both fail")
		}
	})
}

type metadataCapturingExtractor struct {
	result ResumeProfileExtractResult
	err    error
}

func (m *metadataCapturingExtractor) Extract(ctx context.Context, text string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.result.RawJSON, nil
}

func (m *metadataCapturingExtractor) ExtractWithMetadata(ctx context.Context, text string) (ResumeProfileExtractResult, error) {
	if m.err != nil {
		return ResumeProfileExtractResult{}, m.err
	}
	return m.result, nil
}

func TestParseResumeParserVersionPerRequest(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx := context.Background()

	llmExtractor := &metadataCapturingExtractor{
		result: ResumeProfileExtractResult{
			RawJSON: `{"full_name":"LLM","total_experience_years":2,"educations":[{"school":"U"}]}`,
			Metadata: ExtractorMetadata{
				ParserVersion: "resume-profile-llm-v1",
				ExtractorType: "llm",
			},
		},
	}
	heuristicExtractor := NewHeuristicResumeProfileExtractor()
	fallbackExtractor := NewFallbackResumeProfileExtractor(llmExtractor, heuristicExtractor)

	svc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		fallbackExtractor,
	)

	resume := seedResumeProfileServiceResume(t, db, "some resume text")

	snapshot, err := svc.ParseResume(ctx, resume.ID)
	if err != nil {
		t.Fatalf("ParseResume failed: %v", err)
	}
	if snapshot.ParseRun.ParserVersion != "resume-profile-llm-v1" {
		t.Fatalf("expected resume-profile-llm-v1 in parse run, got %q", snapshot.ParseRun.ParserVersion)
	}
}

func TestExtractorMetadataIsRequestScoped(t *testing.T) {
	primary := &slowMetadataExtractor{
		result: ResumeProfileExtractResult{
			RawJSON: `{"full_name":"Fast","total_experience_years":0,"educations":[{"school":"U"}],"experiences":[],"projects":[],"skills":[]}`,
			Metadata: ExtractorMetadata{
				ParserVersion: "resume-profile-llm-v1",
				ExtractorType: "llm",
			},
		},
	}
	fallback := NewHeuristicResumeProfileExtractor()
	f := NewFallbackResumeProfileExtractor(primary, fallback)

	results := make(chan string, 5)
	for i := 0; i < 5; i++ {
		go func() {
			result, err := f.ExtractWithMetadata(context.Background(), "text")
			if err != nil {
				results <- "error"
				return
			}
			results <- result.Metadata.ParserVersion
		}()
	}

	for i := 0; i < 5; i++ {
		version := <-results
		if version != "resume-profile-llm-v1" {
			t.Fatalf("expected resume-profile-llm-v1 from concurrent calls, got %q", version)
		}
	}
}

type slowMetadataExtractor struct {
	result ResumeProfileExtractResult
	err    error
}

func (m *slowMetadataExtractor) Extract(ctx context.Context, text string) (string, error) {
	result, err := m.ExtractWithMetadata(ctx, text)
	if err != nil {
		return "", err
	}
	return result.RawJSON, nil
}

func (m *slowMetadataExtractor) ExtractWithMetadata(ctx context.Context, text string) (ResumeProfileExtractResult, error) {
	if m.err != nil {
		return ResumeProfileExtractResult{}, m.err
	}
	return m.result, nil
}

func TestParseResumeServiceVersionIsRequestScoped(t *testing.T) {
	db := setupResumeProfileServiceTestDB(t)
	ctx := context.Background()

	llmExtractor := &metadataCapturingExtractor{
		result: ResumeProfileExtractResult{
			RawJSON: `{"full_name":"V1","total_experience_years":1,"educations":[{"school":"U"}],"experiences":[],"projects":[],"skills":[]}`,
			Metadata: ExtractorMetadata{
				ParserVersion: "resume-profile-llm-v1",
				ExtractorType: "llm",
			},
		},
	}
	heuristicExtractor := NewHeuristicResumeProfileExtractor()
	fallbackExtractor := NewFallbackResumeProfileExtractor(llmExtractor, heuristicExtractor)

	svc := NewResumeProfileService(
		repository.NewResumeRepo(db),
		repository.NewResumeProfileRepo(db),
		fallbackExtractor,
	)

	resume1 := seedResumeProfileServiceResume(t, db, "first resume")
	resume2 := seedResumeProfileServiceResume(t, db, "second resume")

	snapshot1, err := svc.ParseResume(ctx, resume1.ID)
	if err != nil {
		t.Fatalf("first ParseResume failed: %v", err)
	}
	if snapshot1.ParseRun.ParserVersion != "resume-profile-llm-v1" {
		t.Fatalf("expected resume-profile-llm-v1, got %q", snapshot1.ParseRun.ParserVersion)
	}

	snapshot2, err := svc.ParseResume(ctx, resume2.ID)
	if err != nil {
		t.Fatalf("second ParseResume failed: %v", err)
	}
	if snapshot2.ParseRun.ParserVersion != "resume-profile-llm-v1" {
		t.Fatalf("expected resume-profile-llm-v1, got %q", snapshot2.ParseRun.ParserVersion)
	}
}
