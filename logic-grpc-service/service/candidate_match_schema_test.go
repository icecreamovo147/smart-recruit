package service

import (
	"encoding/json"
	"testing"
)

func TestJobRequirementProfileValidation(t *testing.T) {
	t.Run("valid profile", func(t *testing.T) {
		profile := JobRequirementProfile{
			ProfileVersion: RequirementProfileVersionV1,
			Requirements: []JobRequirementItem{
				{
					ID:       "java-backend",
					Category: RequirementCategoryCoreSkill,
					Label:    "Java 后端开发经验",
					Priority: RequirementPriorityMustHave,
					Weight:   0.18,
					Knockout: false,
					Aliases:  []string{"Java", "Spring Boot"},
				},
				{
					ID:       "mysql",
					Category: RequirementCategoryCoreSkill,
					Label:    "MySQL 数据库",
					Priority: RequirementPriorityMustHave,
					Weight:   0.12,
				},
			},
		}
		if err := profile.Validate(); err != nil {
			t.Fatalf("expected valid profile, got: %v", err)
		}
	})

	t.Run("nil profile", func(t *testing.T) {
		var p *JobRequirementProfile
		if err := p.Validate(); err == nil {
			t.Fatal("expected error for nil profile")
		}
	})

	t.Run("empty requirements", func(t *testing.T) {
		profile := JobRequirementProfile{
			ProfileVersion: RequirementProfileVersionV1,
		}
		if err := profile.Validate(); err == nil {
			t.Fatal("expected error for empty requirements")
		}
	})

	t.Run("missing id", func(t *testing.T) {
		profile := JobRequirementProfile{
			ProfileVersion: RequirementProfileVersionV1,
			Requirements: []JobRequirementItem{
				{Category: RequirementCategoryCoreSkill, Label: "test", Priority: RequirementPriorityMustHave, Weight: 0.5},
			},
		}
		if err := profile.Validate(); err == nil {
			t.Fatal("expected error for missing id")
		}
	})

	t.Run("invalid priority", func(t *testing.T) {
		profile := JobRequirementProfile{
			ProfileVersion: RequirementProfileVersionV1,
			Requirements: []JobRequirementItem{
				{ID: "x", Category: RequirementCategoryCoreSkill, Label: "test", Priority: "invalid", Weight: 0.5},
			},
		}
		if err := profile.Validate(); err == nil {
			t.Fatal("expected error for invalid priority")
		}
	})

	t.Run("duplicate id", func(t *testing.T) {
		profile := JobRequirementProfile{
			ProfileVersion: RequirementProfileVersionV1,
			Requirements: []JobRequirementItem{
				{ID: "dup", Category: RequirementCategoryCoreSkill, Label: "a", Priority: RequirementPriorityMustHave, Weight: 0.3},
				{ID: "dup", Category: RequirementCategoryCoreSkill, Label: "b", Priority: RequirementPriorityNiceToHave, Weight: 0.2},
			},
		}
		if err := profile.Validate(); err == nil {
			t.Fatal("expected error for duplicate id")
		}
	})

	t.Run("zero weight", func(t *testing.T) {
		profile := JobRequirementProfile{
			ProfileVersion: RequirementProfileVersionV1,
			Requirements: []JobRequirementItem{
				{ID: "x", Category: RequirementCategoryCoreSkill, Label: "test", Priority: RequirementPriorityMustHave, Weight: 0},
			},
		}
		if err := profile.Validate(); err == nil {
			t.Fatal("expected error for zero weight")
		}
	})

	t.Run("invalid category", func(t *testing.T) {
		profile := JobRequirementProfile{
			ProfileVersion: RequirementProfileVersionV1,
			Requirements: []JobRequirementItem{
				{ID: "x", Category: "unknown_cat", Label: "test", Priority: RequirementPriorityMustHave, Weight: 0.5},
			},
		}
		if err := profile.Validate(); err == nil {
			t.Fatal("expected error for invalid category")
		}
	})

	t.Run("valid JSON roundtrip", func(t *testing.T) {
		raw := `{
			"profile_version": "job-requirement-profile-v1",
			"requirements": [
				{"id":"docker","category":"core_skill","label":"Docker","priority":"must_have","weight":0.15}
			]
		}`
		if err := ValidateJobRequirementProfileJSON(raw); err != nil {
			t.Fatalf("expected valid JSON, got: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		if err := ValidateJobRequirementProfileJSON(`{not json}`); err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func TestRequirementMatchResultValidation(t *testing.T) {
	t.Run("valid result with evidence", func(t *testing.T) {
		result := RequirementMatchResult{
			RequirementID: "java-backend",
			Status:        MatchStatusMatch,
			Score:         75.0,
			Confidence:    0.8,
			Evidence: []MatchEvidenceUnit{
				{SourceTable: "resume_skills", Snippet: "Java", Reason: "熟练使用 Java"},
			},
		}
		if err := result.Validate(); err != nil {
			t.Fatalf("expected valid, got: %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		result := RequirementMatchResult{
			RequirementID: "x",
			Status:        "unknown",
			Score:         50,
			Confidence:    0.5,
		}
		if err := result.Validate(); err == nil {
			t.Fatal("expected error for invalid status")
		}
	})

	t.Run("score out of range", func(t *testing.T) {
		result := RequirementMatchResult{
			RequirementID: "x",
			Status:        MatchStatusMissing,
			Score:         150,
			Confidence:    0.5,
		}
		if err := result.Validate(); err == nil {
			t.Fatal("expected error for score > 100")
		}
	})

	t.Run("match without evidence", func(t *testing.T) {
		result := RequirementMatchResult{
			RequirementID: "x",
			Status:        MatchStatusStrongMatch,
			Score:         95,
			Confidence:    0.9,
		}
		if err := result.Validate(); err == nil {
			t.Fatal("expected error for strong_match without evidence")
		}
	})

	t.Run("missing without evidence is ok", func(t *testing.T) {
		result := RequirementMatchResult{
			RequirementID: "x",
			Status:        MatchStatusMissing,
			Score:         0,
			Confidence:    0.9,
		}
		if err := result.Validate(); err != nil {
			t.Fatalf("expected valid for missing without evidence, got: %v", err)
		}
	})

	t.Run("evidence missing snippet", func(t *testing.T) {
		result := RequirementMatchResult{
			RequirementID: "x",
			Status:        MatchStatusMatch,
			Score:         70,
			Confidence:    0.7,
			Evidence: []MatchEvidenceUnit{
				{SourceTable: "resume_skills"},
			},
		}
		if err := result.Validate(); err == nil {
			t.Fatal("expected error for evidence without snippet")
		}
	})
}

func TestEnhancedScoreBreakdownValidation(t *testing.T) {
	t.Run("valid breakdown", func(t *testing.T) {
		breakdown := EnhancedScoreBreakdown{
			ScorerVersion: "candidate-match-scorer-v1",
			InputHash:     "abc123",
			ScorerType:    "semantic",
			Dimensions: []ScoreDimension{
				{Name: "must_have", Weight: 0.4, Score: 85},
			},
		}
		if err := breakdown.Validate(); err != nil {
			t.Fatalf("expected valid, got: %v", err)
		}
	})

	t.Run("missing scorer version", func(t *testing.T) {
		breakdown := EnhancedScoreBreakdown{
			ScorerType: "heuristic_fallback",
			Dimensions: []ScoreDimension{{Name: "skills", Weight: 1, Score: 50}},
		}
		if err := breakdown.Validate(); err == nil {
			t.Fatal("expected error for missing scorer_version")
		}
	})

	t.Run("valid with profile and results", func(t *testing.T) {
		breakdown := EnhancedScoreBreakdown{
			ScorerVersion: "v1",
			InputHash:     "hash",
			ScorerType:    "hybrid",
			RequirementProfile: &JobRequirementProfile{
				ProfileVersion: RequirementProfileVersionV1,
				Requirements: []JobRequirementItem{
					{ID: "r1", Category: RequirementCategoryCoreSkill, Label: "R1", Priority: RequirementPriorityMustHave, Weight: 0.5},
				},
			},
			RequirementResults: []RequirementMatchResult{
				{
					RequirementID: "r1",
					Status:        MatchStatusMatch,
					Score:         80,
					Confidence:    0.8,
					Evidence:      []MatchEvidenceUnit{{SourceTable: "resume_skills", Snippet: "skill"}},
				},
			},
			Dimensions: []ScoreDimension{{Name: "must_have", Weight: 0.4, Score: 80}},
		}
		if err := breakdown.Validate(); err != nil {
			t.Fatalf("expected valid, got: %v", err)
		}
	})

	t.Run("JSON roundtrip valid", func(t *testing.T) {
		raw, _ := json.Marshal(EnhancedScoreBreakdown{
			ScorerVersion: "v1",
			InputHash:     "hash",
			ScorerType:    "heuristic_fallback",
			Dimensions:    []ScoreDimension{{Name: "skills", Weight: 1, Score: 60}},
		})
		if err := ValidateEnhancedScoreBreakdownJSON(string(raw)); err != nil {
			t.Fatalf("expected valid JSON, got: %v", err)
		}
	})

	t.Run("JSON roundtrip invalid", func(t *testing.T) {
		if err := ValidateEnhancedScoreBreakdownJSON(`{}`); err == nil {
			t.Fatal("expected error for empty breakdown")
		}
	})
}

func TestMustHaveSummary(t *testing.T) {
	t.Run("computes counts", func(t *testing.T) {
		profile := JobRequirementProfile{
			ProfileVersion: RequirementProfileVersionV1,
			Requirements: []JobRequirementItem{
				{ID: "r1", Category: RequirementCategoryCoreSkill, Label: "R1", Priority: RequirementPriorityMustHave, Weight: 0.2},
				{ID: "r2", Category: RequirementCategoryCoreSkill, Label: "R2", Priority: RequirementPriorityMustHave, Weight: 0.2},
				{ID: "r3", Category: RequirementCategoryCoreSkill, Label: "R3", Priority: RequirementPriorityMustHave, Weight: 0.2},
				{ID: "r4", Category: RequirementCategoryDomain, Label: "R4", Priority: RequirementPriorityNiceToHave, Weight: 0.1},
			},
		}
		results := []RequirementMatchResult{
			{RequirementID: "r1", Status: MatchStatusStrongMatch, Score: 95, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
			{RequirementID: "r2", Status: MatchStatusPartialMatch, Score: 55, Confidence: 0.6, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
			{RequirementID: "r3", Status: MatchStatusMissing, Score: 0, Confidence: 0.9},
			{RequirementID: "r4", Status: MatchStatusMatch, Score: 90, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
		}
		resultByID := make(map[string]RequirementMatchResult)
		for _, r := range results {
			resultByID[r.RequirementID] = r
		}
		var passed, partial, missing, total int
		for _, req := range profile.Requirements {
			if req.Priority != RequirementPriorityMustHave {
				continue
			}
			total++
			if result, ok := resultByID[req.ID]; ok {
				switch result.Status {
				case MatchStatusStrongMatch, MatchStatusMatch:
					passed++
				case MatchStatusPartialMatch, MatchStatusWeakEvidence:
					partial++
				case MatchStatusMissing, MatchStatusConflict:
					missing++
				}
			}
		}
		if passed != 1 || partial != 1 || missing != 1 || total != 3 {
			t.Fatalf("expected passed=1 partial=1 missing=1 total=3, got passed=%d partial=%d missing=%d total=%d", passed, partial, missing, total)
		}
	})
}
