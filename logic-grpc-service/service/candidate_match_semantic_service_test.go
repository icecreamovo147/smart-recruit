package service

import (
	"context"
	"strings"
	"testing"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

func TestDeterministicMatchSkillFound(t *testing.T) {
	matcher := NewRequirementMatcher(nil)
	index := &EvidenceIndex{
		Units: []EvidenceUnit{
			{SourceTable: "resume_skills", SourceType: "skill", SourceID: 1, Snippet: "Go", NormalizedTerms: []string{"go"}},
			{SourceTable: "resume_skills", SourceType: "skill", SourceID: 2, Snippet: "Golang", NormalizedTerms: []string{"golang"}},
		},
	}
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 1},
	}

	req := JobRequirementItem{
		ID:       "go",
		Label:    "Go 开发经验",
		Priority: RequirementPriorityMustHave,
		Aliases:  []string{"Go", "Golang"},
	}

	result := matcher.matchOne(context.Background(), req, index, snapshot)
	if result.Status != MatchStatusMatch && result.Status != MatchStatusStrongMatch {
		t.Fatalf("expected match or strong_match for Go skill, got status=%s score=%.1f", result.Status, result.Score)
	}
	if result.Score < 50 {
		t.Fatalf("expected score >= 50 for matched skill, got %.1f", result.Score)
	}
	if len(result.Evidence) == 0 {
		t.Fatal("expected evidence for matched skill")
	}
}

func TestDeterministicMatchSkillMissing(t *testing.T) {
	matcher := NewRequirementMatcher(nil)
	index := &EvidenceIndex{
		Units: []EvidenceUnit{
			{SourceTable: "resume_skills", SourceType: "skill", SourceID: 1, Snippet: "Python", NormalizedTerms: []string{"python"}},
		},
	}
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 1},
	}

	req := JobRequirementItem{
		ID:       "kubernetes",
		Label:    "Kubernetes 容器编排",
		Priority: RequirementPriorityMustHave,
		Aliases:  []string{"Kubernetes", "K8s"},
	}

	result := matcher.matchOne(context.Background(), req, index, snapshot)
	if result.Status != MatchStatusMissing {
		t.Fatalf("expected missing for unmatched skill, got status=%s", result.Status)
	}
}

func TestDeterministicMatchPartialMatch(t *testing.T) {
	matcher := NewRequirementMatcher(nil)
	index := &EvidenceIndex{
		Units: []EvidenceUnit{
			{SourceTable: "resume_projects", SourceType: "project", SourceID: 1, Snippet: "Built Java backend services", NormalizedTerms: []string{"java", "backend"}},
		},
	}
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 1},
	}

	req := JobRequirementItem{
		ID:       "spring-boot",
		Label:    "Spring Boot",
		Priority: RequirementPriorityMustHave,
		Aliases:  []string{"Spring Boot", "springboot"},
	}

	result := matcher.matchOne(context.Background(), req, index, snapshot)
	if result.Status != MatchStatusPartialMatch && result.Status != MatchStatusMissing {
		t.Fatalf("expected partial_match or missing for indirect match, got status=%s score=%.1f", result.Status, result.Score)
	}
}

func TestDeterministicMatchEducation(t *testing.T) {
	matcher := NewRequirementMatcher(nil)
	index := &EvidenceIndex{}

	type eduCase struct {
		name            string
		reqLabel        string
		candidateDegree string
		wantMatch       bool
		wantScoreGE     float64
	}

	tests := []eduCase{
		{name: "JD本科+候选人本科", reqLabel: "本科及以上学历", candidateDegree: "Bachelor", wantMatch: true, wantScoreGE: 80},
		{name: "JD本科+候选人大专", reqLabel: "本科及以上学历", candidateDegree: "大专", wantMatch: false, wantScoreGE: 0},
		{name: "JD本科+候选人硕士", reqLabel: "本科及以上学历", candidateDegree: "Master", wantMatch: true, wantScoreGE: 80},
		{name: "JD硕士+候选人本科", reqLabel: "硕士及以上学历", candidateDegree: "Bachelor", wantMatch: false, wantScoreGE: 0},
		{name: "JD硕士+候选人博士", reqLabel: "硕士及以上学历", candidateDegree: "PhD", wantMatch: true, wantScoreGE: 80},
		{name: "JD本科+候选人本科(中文)", reqLabel: "本科及以上学历", candidateDegree: "本科", wantMatch: true, wantScoreGE: 80},
		{name: "JD博士+候选人硕士", reqLabel: "博士学历", candidateDegree: "Master", wantMatch: false, wantScoreGE: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := &repository.ResumeProfileSnapshot{
				Profile: model.ResumeProfile{ID: 1},
				Educations: []model.ResumeEducation{
					{ID: 1, School: "University", Degree: tc.candidateDegree, Major: "CS"},
				},
			}
			req := JobRequirementItem{
				ID:       "education-test",
				Label:    tc.reqLabel,
				Category: RequirementCategoryEducation,
				Priority: RequirementPriorityMustHave,
			}
			result := matcher.matchOne(context.Background(), req, index, snapshot)
			if tc.wantMatch && result.Status == MatchStatusMissing {
				t.Fatalf("expected match for %s, got status=%s score=%.1f", tc.name, result.Status, result.Score)
			}
			if !tc.wantMatch && result.Status == MatchStatusMatch {
				t.Fatalf("expected no match for %s, got status=%s score=%.1f", tc.name, result.Status, result.Score)
			}
			if result.Score < tc.wantScoreGE {
				t.Fatalf("expected score >= %.1f for %s, got %.1f", tc.wantScoreGE, tc.name, result.Score)
			}
		})
	}
}

func TestDeterministicMatchExperience(t *testing.T) {
	matcher := NewRequirementMatcher(nil)
	index := &EvidenceIndex{}
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 1, TotalExperience: 4},
		Experiences: []model.ResumeExperience{
			{ID: 1, Company: "Example Inc", Title: "Backend Engineer"},
		},
	}

	t.Run("matching experience years", func(t *testing.T) {
		req := JobRequirementItem{
			ID:       "experience-years",
			Label:    "3年以上工作经验",
			Category: RequirementCategoryExperience,
			Priority: RequirementPriorityMustHave,
		}
		result := matcher.matchOne(context.Background(), req, index, snapshot)
		if result.Status != MatchStatusMatch {
			t.Fatalf("expected match for 4 years experience, got status=%s score=%.1f", result.Status, result.Score)
		}
	})

	t.Run("low experience", func(t *testing.T) {
		snapshotLow := &repository.ResumeProfileSnapshot{
			Profile: model.ResumeProfile{ID: 2, TotalExperience: 0.5},
		}
		req := JobRequirementItem{
			ID:       "experience-years",
			Label:    "3年以上工作经验",
			Category: RequirementCategoryExperience,
			Priority: RequirementPriorityMustHave,
		}
		result := matcher.matchOne(context.Background(), req, index, snapshotLow)
		if result.Status == MatchStatusMatch || result.Status == MatchStatusStrongMatch {
			t.Fatalf("expected no strong match for low experience, got status=%s score=%.1f", result.Status, result.Score)
		}
	})
}

func TestRequirementMatcherMatchAll(t *testing.T) {
	matcher := NewRequirementMatcher(nil)
	profile := &JobRequirementProfile{
		ProfileVersion: RequirementProfileVersionV1,
		Requirements: []JobRequirementItem{
			{ID: "go", Label: "Go", Priority: RequirementPriorityMustHave, Category: RequirementCategoryCoreSkill, Weight: 0.5, Aliases: []string{"Go"}},
			{ID: "redis", Label: "Redis", Priority: RequirementPriorityNiceToHave, Category: RequirementCategoryCoreSkill, Weight: 0.3, Aliases: []string{"Redis"}},
			{ID: "education-bachelor", Label: "本科", Priority: RequirementPriorityMustHave, Category: RequirementCategoryEducation, Weight: 0.2, Knockout: true},
		},
	}
	index := &EvidenceIndex{
		Units: []EvidenceUnit{
			{SourceTable: "resume_skills", SourceType: "skill", SourceID: 1, Snippet: "Go", NormalizedTerms: []string{"go"}},
		},
	}
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 1, HighestDegree: "Bachelor"},
		Educations: []model.ResumeEducation{
			{ID: 1, School: "University", Degree: "Bachelor", Major: "CS"},
		},
	}

	results := matcher.MatchAll(context.Background(), profile, index, snapshot)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	resultByID := make(map[string]RequirementMatchResult)
	for _, r := range results {
		resultByID[r.RequirementID] = r
	}

	if goResult, ok := resultByID["go"]; !ok {
		t.Fatal("expected result for go")
	} else if goResult.Status == MatchStatusMissing {
		t.Fatalf("expected go to match, got status=%s", goResult.Status)
	}

	if eduResult, ok := resultByID["education-bachelor"]; !ok {
		t.Fatal("expected result for education-bachelor")
	} else if eduResult.Status == MatchStatusMissing {
		t.Fatalf("expected education to match, got status=%s", eduResult.Status)
	}
}

func TestBuildEvidenceTextForLLM(t *testing.T) {
	index := &EvidenceIndex{
		Units: []EvidenceUnit{
			{SourceTable: "resume_skills", SourceType: "skill", SourceID: 1, Snippet: "Go", NormalizedTerms: []string{"go"}},
			{SourceTable: "resume_skills", SourceType: "skill", SourceID: 2, Snippet: "Kubernetes", NormalizedTerms: []string{"kubernetes"}},
			{SourceTable: "resume_projects", SourceType: "project", SourceID: 3, Snippet: "Built microservices with Go", NormalizedTerms: []string{"go", "microservices"}},
		},
	}

	req := JobRequirementItem{
		ID:       "go",
		Label:    "Go 开发经验",
		Priority: RequirementPriorityMustHave,
		Aliases:  []string{"Go", "Golang"},
	}

	text := buildEvidenceTextForLLM(index, req)
	if text == "" {
		t.Fatal("expected non-empty evidence text")
	}
	if !strings.Contains(text, "resume_skills") {
		t.Fatal("expected evidence text to contain source table references")
	}
}

func TestBuildCandidateInfoForLLM(t *testing.T) {
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 1, TotalExperience: 4, HighestDegree: "Bachelor"},
		Skills: []model.ResumeSkill{
			{ID: 1, Name: "Go", Category: "language"},
			{ID: 2, Name: "Kubernetes", Category: "platform"},
		},
		Experiences: []model.ResumeExperience{
			{ID: 1, Company: "Example Inc", Title: "Backend Engineer"},
		},
	}

	info := buildCandidateInfoForLLM(snapshot)
	if info == "" {
		t.Fatal("expected non-empty candidate info")
	}
	if !strings.Contains(info, "4.0") {
		t.Fatal("expected info to contain experience years")
	}
	if !strings.Contains(info, "Bachelor") {
		t.Fatal("expected info to contain highest degree")
	}
}

func TestBuildCandidateInfoForLLMNil(t *testing.T) {
	info := buildCandidateInfoForLLM(nil)
	if info == "" {
		t.Fatal("expected fallback message for nil snapshot")
	}
	if info != "No profile data available." {
		t.Fatalf("expected 'No profile data available.', got %q", info)
	}
}

func TestBuildEvidenceTextForLLMEmptyIndex(t *testing.T) {
	text := buildEvidenceTextForLLM(&EvidenceIndex{}, JobRequirementItem{ID: "test", Label: "test"})
	if text == "" {
		t.Fatal("expected fallback text for empty index")
	}
}

func TestLLMMatcherSchemaValidationDegradation(t *testing.T) {
	result := RequirementMatchResult{
		RequirementID: "test",
		Status:        MatchStatusStrongMatch,
		Score:         100,
		Confidence:    0.95,
	}
	err := result.Validate()
	if err == nil {
		t.Fatal("expected validation error: strong_match without evidence")
	}
	result.Status = MatchStatusWeakEvidence
	result.Score = 30
	result.Confidence = 0.3
	if err := result.Validate(); err != nil {
		t.Fatalf("expected valid weak_evidence without evidence, got: %v", err)
	}
}

func TestRequirementMatcherEvidenceRequiredForMatch(t *testing.T) {
	matcher := NewRequirementMatcher(nil)
	index := &EvidenceIndex{}
	snapshot := &repository.ResumeProfileSnapshot{
		Profile: model.ResumeProfile{ID: 1},
	}

	req := JobRequirementItem{
		ID:       "java",
		Label:    "Java",
		Priority: RequirementPriorityMustHave,
		Aliases:  []string{"Java"},
	}

	result := matcher.matchOne(context.Background(), req, index, snapshot)
	if result.Status == MatchStatusMatch || result.Status == MatchStatusStrongMatch {
		t.Fatalf("expected no match/strong_match without evidence, got status=%s", result.Status)
	}
}
