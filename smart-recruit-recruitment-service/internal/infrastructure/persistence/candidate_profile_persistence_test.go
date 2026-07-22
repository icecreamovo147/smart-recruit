package persistence

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-recruitment-service/internal/domain/model"
	profilepkg "smart-recruit-recruitment-service/internal/domain/profile"
)

func TestCandidateProfileStructuredHistoryRoundTrip(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&candidateProfileRecord{}, &candidateEducationRecord{}, &candidateExperienceRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	store := &nativeStore{db: db, now: func() time.Time { return time.Date(2026, 7, 23, 1, 0, 0, 0, time.UTC) }}
	bundle := profilepkg.Bundle{
		Profile: model.CandidateProfile{
			UserID:            42,
			RealName:          "Ada",
			Phone:             "13800000000",
			Skills:            "Go",
			City:              "上海市/浦东新区",
			YearsOfExperience: 3.5,
			JobStatus:         "employed",
			ExpectedPosition:  "Backend",
			ExpectedSalaryMin: 30000,
			ExpectedSalaryMax: 45000,
			Summary:           "Go engineer",
		},
		Educations: []profilepkg.EducationInput{{
			School:    "Demo University",
			Degree:    "硕士",
			Major:     "CS",
			SortOrder: 0,
		}},
		Experiences: []profilepkg.ExperienceInput{{
			Company:   "Demo Corp",
			Title:     "Engineer",
			IsCurrent: 1,
			SortOrder: 0,
		}},
	}
	if err := store.saveProfileBundle(context.Background(), bundle); err != nil {
		t.Fatalf("saveProfileBundle: %v", err)
	}
	loaded, err := store.loadProfileBundle(context.Background(), 42)
	if err != nil {
		t.Fatalf("loadProfileBundle: %v", err)
	}
	if loaded.Profile.City != "上海市/浦东新区" || loaded.Profile.YearsOfExperience != 3.5 {
		t.Fatalf("unexpected profile: %+v", loaded.Profile)
	}
	if len(loaded.Educations) != 1 || loaded.Educations[0].School != "Demo University" {
		t.Fatalf("unexpected educations: %+v", loaded.Educations)
	}
	if len(loaded.Experiences) != 1 || loaded.Experiences[0].Company != "Demo Corp" {
		t.Fatalf("unexpected experiences: %+v", loaded.Experiences)
	}
}
