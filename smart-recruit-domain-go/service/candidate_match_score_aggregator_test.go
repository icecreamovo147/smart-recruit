package service

import (
	"strings"
	"testing"
)

func TestScoreAggregationMustHaveScore(t *testing.T) {
	agg := NewScoreAggregator("test-v1")

	profile := &JobRequirementProfile{
		ProfileVersion: RequirementProfileVersionV1,
		Requirements: []JobRequirementItem{
			{ID: "r1", Label: "Requirement 1", Priority: RequirementPriorityMustHave, Weight: 0.5},
			{ID: "r2", Label: "Requirement 2", Priority: RequirementPriorityMustHave, Weight: 0.3},
			{ID: "r3", Label: "Requirement 3", Priority: RequirementPriorityNiceToHave, Weight: 0.2},
		},
	}
	results := []RequirementMatchResult{
		{RequirementID: "r1", Status: MatchStatusStrongMatch, Score: 95, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
		{RequirementID: "r2", Status: MatchStatusPartialMatch, Score: 50, Confidence: 0.6, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
		{RequirementID: "r3", Status: MatchStatusMatch, Score: 80, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
	}

	aggregated := agg.Aggregate(profile, results, nil, nil, "hash", "test", false)

	if aggregated.OverallScore <= 0 {
		t.Fatalf("expected positive overall score, got %.1f", aggregated.OverallScore)
	}
	if aggregated.Recommendation == "" {
		t.Fatal("expected recommendation")
	}
	if aggregated.Summary == "" {
		t.Fatal("expected summary")
	}
	if len(aggregated.Breakdown.Dimensions) != 4 {
		t.Fatalf("expected 4 dimensions, got %d", len(aggregated.Breakdown.Dimensions))
	}
}

func TestScoreAggregationKnockoutCap(t *testing.T) {
	agg := NewScoreAggregator("test-v1")

	profile := &JobRequirementProfile{
		ProfileVersion: RequirementProfileVersionV1,
		Requirements: []JobRequirementItem{
			{ID: "r1", Label: "Knockout Req", Priority: RequirementPriorityMustHave, Weight: 0.5, Knockout: true},
			{ID: "r2", Label: "Nice Skill", Priority: RequirementPriorityNiceToHave, Weight: 0.5},
		},
	}
	results := []RequirementMatchResult{
		{RequirementID: "r1", Status: MatchStatusMissing, Score: 0, Confidence: 0.9},
		{RequirementID: "r2", Status: MatchStatusStrongMatch, Score: 95, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
	}

	aggregated := agg.Aggregate(profile, results, nil, nil, "hash", "test", false)

	if aggregated.OverallScore > 40 {
		t.Fatalf("expected score capped to <=40 for knockout missing, got %.1f", aggregated.OverallScore)
	}
	if aggregated.Recommendation != RecommendationStrongNotRecommend {
		t.Fatalf("expected strong_not_recommend for knockout missing, got %s", aggregated.Recommendation)
	}
}

func TestScoreAggregationEmptyProfile(t *testing.T) {
	agg := NewScoreAggregator("test-v1")

	profile := &JobRequirementProfile{
		ProfileVersion: RequirementProfileVersionV1,
	}

	aggregated := agg.Aggregate(profile, nil, nil, nil, "hash", "test", false)

	if aggregated.OverallScore != 0 {
		t.Fatalf("expected score 0 for empty profile, got %.1f", aggregated.OverallScore)
	}
	if aggregated.Recommendation != RecommendationNotRecommend {
		t.Fatalf("expected not_recommend for empty, got %s", aggregated.Recommendation)
	}
}

func TestScoreAggregationAllMatch(t *testing.T) {
	agg := NewScoreAggregator("test-v1")

	profile := &JobRequirementProfile{
		ProfileVersion: RequirementProfileVersionV1,
		Requirements: []JobRequirementItem{
			{ID: "r1", Label: "Java", Priority: RequirementPriorityMustHave, Weight: 0.5},
			{ID: "r2", Label: "MySQL", Priority: RequirementPriorityMustHave, Weight: 0.5},
		},
	}
	results := []RequirementMatchResult{
		{RequirementID: "r1", Status: MatchStatusStrongMatch, Score: 95, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
		{RequirementID: "r2", Status: MatchStatusMatch, Score: 85, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
	}

	aggregated := agg.Aggregate(profile, results, nil, nil, "hash", "test", false)

	if aggregated.OverallScore < 50 {
		t.Fatalf("expected moderate score for all matched, got %.1f", aggregated.OverallScore)
	}
	if aggregated.Recommendation != RecommendationReview {
		t.Fatalf("expected review (score=62 -> review threshold 50-70), got %s (score=%.1f)", aggregated.Recommendation, aggregated.OverallScore)
	}
}

func TestScoreAggregationAllMissing(t *testing.T) {
	agg := NewScoreAggregator("test-v1")

	profile := &JobRequirementProfile{
		ProfileVersion: RequirementProfileVersionV1,
		Requirements: []JobRequirementItem{
			{ID: "r1", Label: "Java", Priority: RequirementPriorityMustHave, Weight: 0.5},
			{ID: "r2", Label: "K8s", Priority: RequirementPriorityMustHave, Weight: 0.5},
		},
	}
	results := []RequirementMatchResult{
		{RequirementID: "r1", Status: MatchStatusMissing, Score: 0, Confidence: 0.9},
		{RequirementID: "r2", Status: MatchStatusMissing, Score: 0, Confidence: 0.9},
	}

	aggregated := agg.Aggregate(profile, results, nil, nil, "hash", "test", false)

	if aggregated.OverallScore > 30 {
		t.Fatalf("expected very low score for all missing, got %.1f", aggregated.OverallScore)
	}
	if aggregated.Recommendation != RecommendationStrongNotRecommend && aggregated.Recommendation != RecommendationNotRecommend {
		t.Fatalf("expected not recommend for all missing, got %s", aggregated.Recommendation)
	}
}

func TestScoreAggregationChineseSummary(t *testing.T) {
	agg := NewScoreAggregator("test-v1")

	profile := &JobRequirementProfile{
		ProfileVersion: RequirementProfileVersionV1,
		Requirements: []JobRequirementItem{
			{ID: "go", Label: "Go 开发", Priority: RequirementPriorityMustHave, Weight: 0.6},
			{ID: "redis", Label: "Redis", Priority: RequirementPriorityMustHave, Weight: 0.4},
		},
	}
	results := []RequirementMatchResult{
		{RequirementID: "go", Status: MatchStatusStrongMatch, Score: 95, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
		{RequirementID: "redis", Status: MatchStatusMissing, Score: 0, Confidence: 0.9},
	}

	aggregated := agg.Aggregate(profile, results, nil, nil, "hash", "test", false)

	if aggregated.Summary == "" {
		t.Fatal("expected Chinese summary")
	}

	if strings.Contains(aggregated.Summary, "综合评分") {
		t.Log("summary contains Chinese score text")
	} else {
		t.Fatalf("expected Chinese summary containing 综合评分, got: %s", aggregated.Summary)
	}

	if aggregated.Recommendation != "" {
		label := ComputeRecommendationLabel(aggregated.Recommendation)
		if label == aggregated.Recommendation {
			t.Fatalf("expected Chinese label for recommendation %s, got same as key", aggregated.Recommendation)
		}
	}
}

func TestScoreAggregationStrengthsAndRisks(t *testing.T) {
	agg := NewScoreAggregator("test-v1")

	profile := &JobRequirementProfile{
		ProfileVersion: RequirementProfileVersionV1,
		Requirements: []JobRequirementItem{
			{ID: "r1", Label: "Java", Priority: RequirementPriorityMustHave, Weight: 0.5},
			{ID: "r2", Label: "K8s", Priority: RequirementPriorityMustHave, Weight: 0.5},
		},
	}
	results := []RequirementMatchResult{
		{RequirementID: "r1", Status: MatchStatusStrongMatch, Score: 95, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
		{RequirementID: "r2", Status: MatchStatusMissing, Score: 0, Confidence: 0.9, Risk: "未找到相关经验"},
	}

	aggregated := agg.Aggregate(profile, results, nil, nil, "hash", "test", false)

	if len(aggregated.Strengths) == 0 {
		t.Fatal("expected strengths for matched requirement")
	}
	if len(aggregated.Risks) == 0 {
		t.Fatal("expected risks for missing requirement")
	}

	strengthMsgs := make([]string, len(aggregated.Strengths))
	for i, s := range aggregated.Strengths {
		strengthMsgs[i] = s.Message
	}
	riskMsgs := make([]string, len(aggregated.Risks))
	for i, r := range aggregated.Risks {
		riskMsgs[i] = r.Message
	}

	hasStrength := false
	for _, msg := range strengthMsgs {
		if strings.Contains(msg, "Java") {
			hasStrength = true
			break
		}
	}
	if !hasStrength {
		t.Fatalf("expected strength mentioning Java, got %v", strengthMsgs)
	}

	hasRisk := false
	for _, msg := range riskMsgs {
		if strings.Contains(msg, "K8s") || strings.Contains(msg, "未找到") {
			hasRisk = true
			break
		}
	}
	if !hasRisk {
		t.Fatalf("expected risk mentioning K8s or '未找到', got %v", riskMsgs)
	}
}

func TestComputeRecommendationLabel(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{RecommendationStrongRecommend, "强烈推荐"},
		{RecommendationRecommend, "推荐推进"},
		{RecommendationReview, "谨慎评估"},
		{RecommendationNotRecommend, "不建议推进"},
		{RecommendationStrongNotRecommend, "强烈不建议"},
		{"unknown", "unknown"},
	}
	for _, tc := range tests {
		got := ComputeRecommendationLabel(tc.key)
		if got != tc.want {
			t.Fatalf("ComputeRecommendationLabel(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestScoreAggregationMustHaveSummary(t *testing.T) {
	profile := &JobRequirementProfile{
		ProfileVersion: RequirementProfileVersionV1,
		Requirements: []JobRequirementItem{
			{ID: "r1", Priority: RequirementPriorityMustHave, Weight: 0.3},
			{ID: "r2", Priority: RequirementPriorityMustHave, Weight: 0.3},
			{ID: "r3", Priority: RequirementPriorityMustHave, Weight: 0.4},
			{ID: "r4", Priority: RequirementPriorityNiceToHave, Weight: 0.2},
		},
	}
	resultByID := map[string]RequirementMatchResult{
		"r1": {RequirementID: "r1", Status: MatchStatusStrongMatch, Score: 95, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
		"r2": {RequirementID: "r2", Status: MatchStatusPartialMatch, Score: 50, Confidence: 0.6, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
		"r3": {RequirementID: "r3", Status: MatchStatusMissing, Score: 0, Confidence: 0.9},
		"r4": {RequirementID: "r4", Status: MatchStatusMatch, Score: 90, Confidence: 0.9, Evidence: []MatchEvidenceUnit{{SourceTable: "t", Snippet: "s"}}},
	}

	summary := computeMustHaveSummary(profile, resultByID)
	if summary.Passed != 1 || summary.Partial != 1 || summary.Missing != 1 || summary.Total != 3 {
		t.Fatalf("expected passed=1 partial=1 missing=1 total=3, got passed=%d partial=%d missing=%d total=%d",
			summary.Passed, summary.Partial, summary.Missing, summary.Total)
	}
}
