package repository

import (
	"context"
	"testing"
	"time"

	"logic-grpc-service/model"
)

func TestResumeProfileRepoSaveProfileVersionCreatesCurrentSnapshot(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResumeProfileRepo(db)
	ctx := context.Background()

	snapshot := &ResumeProfileSnapshot{
		ParseRun: model.ResumeParseRun{
			ResumeID:      10,
			UserID:        20,
			Status:        "succeeded",
			ParserVersion: "resume-parser-v1",
			InputHash:     "hash-1",
		},
		Profile: model.ResumeProfile{
			FullName:        "Ada Lovelace",
			Headline:        "Backend Engineer",
			TotalExperience: 5.5,
			RawJSON:         `{"name":"Ada Lovelace"}`,
		},
		Educations: []model.ResumeEducation{{School: "Example University", Degree: "BS", SortOrder: 1}},
		Experiences: []model.ResumeExperience{{
			Company:          "Example Co",
			Title:            "Engineer",
			AchievementsJSON: `["scaled services"]`,
		}},
		Projects: []model.ResumeProject{{Name: "Recruiting Platform", TechnologiesJSON: `["Go"]`}},
		Skills:   []model.ResumeSkill{{Name: "Go", Category: "language", Level: "advanced"}},
	}

	if err := repo.SaveProfileVersion(ctx, snapshot); err != nil {
		t.Fatalf("SaveProfileVersion failed: %v", err)
	}
	if snapshot.ParseRun.ID == 0 || snapshot.Profile.ID == 0 {
		t.Fatalf("expected parse run and profile IDs to be populated: %+v", snapshot)
	}
	if snapshot.Profile.Version != 1 || snapshot.Profile.IsCurrent != 1 {
		t.Fatalf("expected version 1 current profile, got version=%d current=%d", snapshot.Profile.Version, snapshot.Profile.IsCurrent)
	}
	if snapshot.ParseRun.CompletedAt == nil {
		t.Fatal("expected completed_at to be set for succeeded run")
	}

	current, err := repo.GetCurrentByResumeID(ctx, 10)
	if err != nil {
		t.Fatalf("GetCurrentByResumeID failed: %v", err)
	}
	if current == nil || current.ID != snapshot.Profile.ID {
		t.Fatalf("unexpected current profile: %+v", current)
	}

	loaded, err := repo.GetSnapshot(ctx, snapshot.Profile.ID)
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}
	if len(loaded.Educations) != 1 || len(loaded.Experiences) != 1 || len(loaded.Projects) != 1 || len(loaded.Skills) != 1 {
		t.Fatalf("expected all child rows to be loaded, got %+v", loaded)
	}
}

func TestResumeProfileRepoSaveProfileVersionBumpsVersionAndReplacesChildren(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResumeProfileRepo(db)
	ctx := context.Background()

	first := &ResumeProfileSnapshot{
		ParseRun: model.ResumeParseRun{ResumeID: 11, UserID: 21, Status: "succeeded"},
		Profile:  model.ResumeProfile{FullName: "Grace Hopper"},
		Skills:   []model.ResumeSkill{{Name: "COBOL"}},
	}
	if err := repo.SaveProfileVersion(ctx, first); err != nil {
		t.Fatalf("first SaveProfileVersion failed: %v", err)
	}

	second := &ResumeProfileSnapshot{
		ParseRun: model.ResumeParseRun{ResumeID: 11, UserID: 21, Status: "succeeded"},
		Profile:  model.ResumeProfile{FullName: "Grace Hopper", Headline: "Computer Scientist"},
		Skills:   []model.ResumeSkill{{Name: "Compilers"}, {Name: "Leadership"}},
	}
	if err := repo.SaveProfileVersion(ctx, second); err != nil {
		t.Fatalf("second SaveProfileVersion failed: %v", err)
	}
	if second.Profile.Version != 2 || second.Profile.IsCurrent != 1 {
		t.Fatalf("expected second profile version 2 current, got version=%d current=%d", second.Profile.Version, second.Profile.IsCurrent)
	}

	var old model.ResumeProfile
	if err := db.First(&old, first.Profile.ID).Error; err != nil {
		t.Fatalf("load old profile failed: %v", err)
	}
	if old.IsCurrent != 0 {
		t.Fatalf("expected old profile to be non-current, got %d", old.IsCurrent)
	}

	current, err := repo.GetCurrentByResumeID(ctx, 11)
	if err != nil {
		t.Fatalf("GetCurrentByResumeID failed: %v", err)
	}
	if current == nil || current.ID != second.Profile.ID {
		t.Fatalf("expected second profile current, got %+v", current)
	}

	second.Profile.Headline = "Updated Scientist"
	second.Skills = []model.ResumeSkill{{Name: "Distributed Systems"}}
	if err := repo.SaveProfileVersion(ctx, second); err != nil {
		t.Fatalf("idempotent SaveProfileVersion failed: %v", err)
	}
	if second.Profile.Version != 2 {
		t.Fatalf("same parse run should preserve version 2, got %d", second.Profile.Version)
	}
	loaded, err := repo.GetSnapshot(ctx, second.Profile.ID)
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}
	if len(loaded.Skills) != 1 || loaded.Skills[0].Name != "Distributed Systems" {
		t.Fatalf("expected replaced skills, got %+v", loaded.Skills)
	}
}

func TestResumeProfileRepoKeepsRunningParseRunOpen(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResumeProfileRepo(db)
	ctx := context.Background()

	started := time.Now().Add(-time.Minute)
	snapshot := &ResumeProfileSnapshot{
		ParseRun: model.ResumeParseRun{ResumeID: 12, UserID: 22, Status: "running", StartedAt: started},
		Profile:  model.ResumeProfile{FullName: "Running Candidate"},
	}
	if err := repo.SaveProfileVersion(ctx, snapshot); err != nil {
		t.Fatalf("SaveProfileVersion failed: %v", err)
	}
	if snapshot.ParseRun.CompletedAt != nil {
		t.Fatalf("running parse run should not get completed_at, got %v", snapshot.ParseRun.CompletedAt)
	}
}

func TestResumeProfileRepoFailedRunWithoutProfileDoesNotReplaceCurrent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResumeProfileRepo(db)
	ctx := context.Background()

	success := &ResumeProfileSnapshot{
		ParseRun: model.ResumeParseRun{ResumeID: 13, UserID: 23, Status: "succeeded"},
		Profile:  model.ResumeProfile{FullName: "Existing Candidate"},
		Skills:   []model.ResumeSkill{{Name: "Go"}},
	}
	if err := repo.SaveProfileVersion(ctx, success); err != nil {
		t.Fatalf("success SaveProfileVersion failed: %v", err)
	}

	failure := &ResumeProfileSnapshot{
		ParseRun: model.ResumeParseRun{ResumeID: 13, UserID: 23, Status: "failed", ErrorMessage: "invalid JSON"},
	}
	if err := repo.SaveProfileVersion(ctx, failure); err != nil {
		t.Fatalf("failure SaveProfileVersion failed: %v", err)
	}
	if failure.ParseRun.ID == 0 || failure.ParseRun.CompletedAt == nil {
		t.Fatalf("expected failed parse run to be persisted as terminal, got %+v", failure.ParseRun)
	}

	current, err := repo.GetCurrentByResumeID(ctx, 13)
	if err != nil {
		t.Fatalf("GetCurrentByResumeID failed: %v", err)
	}
	if current == nil || current.ID != success.Profile.ID || current.FullName != "Existing Candidate" {
		t.Fatalf("expected existing profile to remain current, got %+v", current)
	}

	var failureProfiles int64
	if err := db.Model(&model.ResumeProfile{}).Where("parse_run_id = ?", failure.ParseRun.ID).Count(&failureProfiles).Error; err != nil {
		t.Fatalf("count failure profiles failed: %v", err)
	}
	if failureProfiles != 0 {
		t.Fatalf("expected no empty profile for failed run, got %d", failureProfiles)
	}
}
