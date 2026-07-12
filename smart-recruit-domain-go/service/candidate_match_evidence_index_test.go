package service

import (
	"strings"
	"testing"
	"time"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/repository"
)

func now() time.Time {
	return time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
}

func TestBuildEvidenceIndexSkills(t *testing.T) {
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 10001},
		Skills: []model.ResumeSkill{
			{ID: 1, Name: "Go", Category: "language"},
			{ID: 2, Name: "Kubernetes", Category: "platform"},
			{ID: 3, Name: "PostgreSQL", Category: "database"},
		},
	}
	index := BuildEvidenceIndex(snapshot, nil, nil)

	skillCount := 0
	for _, unit := range index.Units {
		if unit.SourceTable == "resume_skills" {
			skillCount++
			if unit.SourceID == 0 {
				t.Fatal("expected non-zero SourceID for skill evidence")
			}
			if unit.Snippet == "" {
				t.Fatal("expected non-empty snippet for skill evidence")
			}
		}
	}
	if skillCount != 3 {
		t.Fatalf("expected 3 skill evidence units, got %d", skillCount)
	}
}

func TestBuildEvidenceIndexExperience(t *testing.T) {
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 10001},
		Experiences: []model.ResumeExperience{
			{
				ID:               10,
				Company:          "Example Inc",
				Title:            "Backend Engineer",
				Description:      "Built Go backend APIs with PostgreSQL.",
				AchievementsJSON: `["Improved service reliability"]`,
			},
		},
	}
	index := BuildEvidenceIndex(snapshot, nil, nil)

	found := false
	for _, unit := range index.Units {
		if unit.SourceTable == "resume_experiences" {
			found = true
			if unit.SourceID != 10 {
				t.Fatal("expected SourceID 10")
			}
			if !strings.Contains(unit.Snippet, "Backend Engineer") {
				t.Fatal("expected snippet to contain job title")
			}
			if len(unit.NormalizedTerms) == 0 {
				t.Fatal("expected normalized terms")
			}
		}
	}
	if !found {
		t.Fatal("expected experience evidence unit")
	}
}

func TestBuildEvidenceIndexProject(t *testing.T) {
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 10001},
		Projects: []model.ResumeProject{
			{
				ID:               20,
				Name:             "Smart Recruit System",
				Role:             "Backend Developer",
				Description:      "Built recruitment platform with Go and React",
				TechnologiesJSON: `["Go", "React", "PostgreSQL"]`,
			},
		},
	}
	index := BuildEvidenceIndex(snapshot, nil, nil)

	found := false
	for _, unit := range index.Units {
		if unit.SourceTable == "resume_projects" {
			found = true
			if unit.SourceID != 20 {
				t.Fatal("expected SourceID 20")
			}
			if len(unit.NormalizedTerms) == 0 {
				t.Fatal("expected normalized terms for project")
			}
		}
	}
	if !found {
		t.Fatal("expected project evidence unit")
	}
}

func TestBuildEvidenceIndexEducation(t *testing.T) {
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 10001},
		Educations: []model.ResumeEducation{
			{
				ID:     30,
				School: "Example University",
				Degree: "Bachelor",
				Major:  "Computer Science",
			},
		},
	}
	index := BuildEvidenceIndex(snapshot, nil, nil)

	found := false
	for _, unit := range index.Units {
		if unit.SourceTable == "resume_educations" {
			found = true
			if unit.SourceID != 30 {
				t.Fatal("expected SourceID 30")
			}
			if !strings.Contains(unit.Snippet, "Example University") {
				t.Fatal("expected snippet to contain school name")
			}
		}
	}
	if !found {
		t.Fatal("expected education evidence unit")
	}
}

func TestBuildEvidenceIndexResumeText(t *testing.T) {
	now := now()
	resume := &model.Resume{
		ID:         40,
		ParsedText: "Ada has built backend services with Go and PostgreSQL. Contact: ada@example.com or 13800138000.",
		ParsedAt:   &now,
	}
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 10001},
	}
	index := BuildEvidenceIndex(snapshot, nil, resume)

	found := false
	for _, unit := range index.Units {
		if unit.SourceTable == "resumes" {
			found = true
			if strings.Contains(unit.Snippet, "ada@example.com") {
				t.Fatal("expected email to be stripped from snippet")
			}
			if strings.Contains(unit.Snippet, "13800138000") {
				t.Fatal("expected phone to be stripped from snippet")
			}
			if len(unit.NormalizedTerms) == 0 {
				t.Fatal("expected normalized terms for resume text")
			}
		}
	}
	if !found {
		t.Fatal("expected resume text evidence unit")
	}
}

func TestBuildEvidenceIndexCandidateProfile(t *testing.T) {
	profile := &model.CandidateProfile{
		ID:             50,
		Education:      "Bachelor",
		School:         "Example University",
		WorkExperience: "4 years backend engineering",
		Skills:         "Go, PostgreSQL",
		Phone:          "13800138000",
	}
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 10001},
	}
	index := BuildEvidenceIndex(snapshot, profile, nil)

	found := false
	for _, unit := range index.Units {
		if unit.SourceTable == "candidate_profiles" {
			found = true
			if unit.SourceID != 50 {
				t.Fatal("expected SourceID 50")
			}
			if !strings.Contains(unit.Snippet, "Bachelor") {
				t.Fatal("expected snippet to contain education info")
			}
		}
	}
	if !found {
		t.Fatal("expected candidate profile evidence unit")
	}
}

func TestBuildEvidenceIndexSensitiveFieldExclusion(t *testing.T) {
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 10001},
		Skills: []model.ResumeSkill{
			{ID: 1, Name: "13800138000", Category: "contact"},
			{ID: 2, Name: "test@example.com", Category: "contact"},
			{ID: 3, Name: "Go", Category: "language"},
		},
	}
	index := BuildEvidenceIndex(snapshot, nil, nil)

	for _, unit := range index.Units {
		if unit.SourceTable == "resume_skills" && unit.SourceID <= 2 {
			t.Fatalf("expected phone/email skill to be excluded, got id=%d snippet=%s", unit.SourceID, unit.Snippet)
		}
	}
}

func TestBuildEvidenceIndexSnippetLengthControl(t *testing.T) {
	longText := strings.Repeat("A very long resume text that should be truncated to the maximum allowed length. ", 50)
	resume := &model.Resume{
		ID:         60,
		ParsedText: longText,
	}
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 10001},
	}
	index := BuildEvidenceIndex(snapshot, nil, resume)

	for _, unit := range index.Units {
		runes := []rune(unit.Snippet)
		if runes[len(runes)-1] == '…' {
			if len(runes) > maxEvidenceSnippetLen+1 { // +1 for ellipsis
				t.Fatalf("expected truncated snippet <= %d runes, got %d", maxEvidenceSnippetLen+1, len(runes))
			}
		}
	}
}

func TestBuildEvidenceIndexEmptyProfile(t *testing.T) {
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 10001},
	}
	index := BuildEvidenceIndex(snapshot, nil, nil)
	if len(index.Units) != 0 {
		t.Fatalf("expected empty evidence index for empty profile, got %d units", len(index.Units))
	}
}

func TestBuildEvidenceIndexMaxLimit(t *testing.T) {
	skills := make([]model.ResumeSkill, 150)
	for i := range skills {
		skills[i] = model.ResumeSkill{ID: uint64(100 + i), Name: "Skill", Category: "lang"}
	}
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 10001},
		Skills:  skills,
	}
	index := BuildEvidenceIndex(snapshot, nil, nil)
	if len(index.Units) > maxTotalSnippets {
		t.Fatalf("expected at most %d units, got %d", maxTotalSnippets, len(index.Units))
	}
}

func TestStripSensitiveFields(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "Call me at 13800138000", want: "Call me at [PHONE]"},
		{input: "Email: test@example.com", want: "Email: [EMAIL]"},
		{input: "No sensitive data here", want: "No sensitive data here"},
		{input: "Phone: 13800138000, Email: test@example.com", want: "Phone: [PHONE], Email: [EMAIL]"},
	}
	for _, tc := range tests {
		got := stripSensitiveFields(tc.input)
		if got != tc.want {
			t.Fatalf("stripSensitiveFields(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFindEvidenceByTerm(t *testing.T) {
	index := &EvidenceIndex{
		Units: []EvidenceUnit{
			{SourceTable: "resume_skills", SourceType: "skill", SourceID: 1, Snippet: "Go", NormalizedTerms: []string{"go"}},
			{SourceTable: "resume_skills", SourceType: "skill", SourceID: 2, Snippet: "Kubernetes", NormalizedTerms: []string{"kubernetes"}},
		},
	}

	matches := findEvidenceByTerm([]string{"go", "python"}, index)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].SourceID != 1 {
		t.Fatalf("expected matched SourceID 1, got %d", matches[0].SourceID)
	}
}
