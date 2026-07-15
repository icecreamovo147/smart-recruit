package recruiting_intelligence

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCandidateMatchAggregatorDevParityKnockoutRiskDimensionsAndSummary(t *testing.T) {
	profile := JobRequirementProfile{ProfileVersion: JobRequirementProfileVersion, Requirements: []JobRequirementItem{
		{ID: "go", Category: RequirementCategoryCoreSkill, Label: "Go 开发能力", Description: "核心开发能力", Priority: RequirementPriorityMustHave, Weight: 0.4, Aliases: []string{"Go"}},
		{ID: "experience", Category: RequirementCategoryExperience, Label: "五年工作经验", Description: "相关工作经验", Priority: RequirementPriorityMustHave, Weight: 0.3, Knockout: true, Aliases: []string{"五年"}},
		{ID: "communication", Category: RequirementCategorySoftSkill, Label: "沟通能力", Description: "协作沟通能力", Priority: RequirementPrioritySoftSkill, Weight: 0.3, Aliases: []string{"沟通"}},
	}}
	results := []RequirementMatchResult{
		{RequirementID: "communication", Status: MatchStatusWeakEvidence, Score: 40, Confidence: 0.5, Risk: "沟通证据较弱", EvaluatorType: MatchEvaluatorLLM},
		{RequirementID: "experience", Status: MatchStatusMissing, Score: 99, Confidence: 0.9, Risk: "经验要求未满足", EvaluatorType: MatchEvaluatorLLM},
		{RequirementID: "go", Status: MatchStatusStrongMatch, Score: 95, Confidence: 0.94, EvaluatorType: MatchEvaluatorDeterministic},
	}

	got, err := NewCandidateMatchAggregator("").Aggregate(profile, results, 6, "input-hash", "enhanced", false)
	if err != nil {
		t.Fatalf("Aggregate error = %v", err)
	}
	// must_have=(95*.4+99*.3)/.7=96.7, core=95, experience=99,
	// growth=40; weighted=86.23; risk=5+10+2=17 => 69.2, then knockout cap.
	if got.OverallScore != 40 {
		t.Fatalf("overall = %.1f, want knockout cap 40", got.OverallScore)
	}
	if got.Recommendation != RecommendationStrongNotRecommend {
		t.Fatalf("recommendation = %q, want strong_not_recommend", got.Recommendation)
	}
	if got.Breakdown.RiskPenalty != 17 || len(got.Breakdown.Dimensions) != 4 {
		t.Fatalf("breakdown = %+v, want penalty 17 and four dimensions", got.Breakdown)
	}
	if got.Breakdown.Dimensions[0] != (ScoreDimension{Name: "must_have", Label: "必备要求满足度", Weight: .4, Score: 96.7}) {
		t.Fatalf("must-have dimension = %+v", got.Breakdown.Dimensions[0])
	}
	if !strings.Contains(got.Summary, "综合评分 40.0 分") || !strings.Contains(got.Summary, "关键否决条件") {
		t.Fatalf("summary = %q, want deterministic Chinese knockout summary", got.Summary)
	}
	if len(got.Risks) < 3 || got.Risks[0].Code != "knockout_missing" {
		t.Fatalf("risks = %+v", got.Risks)
	}
}

func TestCandidateMatchAggregatorRecommendationThresholds(t *testing.T) {
	tests := []struct {
		name           string
		score          float64
		mustHavePassed int
		want           string
	}{
		{name: "strong", score: 95, mustHavePassed: 2, want: RecommendationStrongRecommend},
		{name: "recommend", score: 75, mustHavePassed: 2, want: RecommendationRecommend},
		{name: "review", score: 60, mustHavePassed: 1, want: RecommendationReview},
		{name: "not", score: 40, mustHavePassed: 0, want: RecommendationNotRecommend},
		{name: "strong not", score: 20, mustHavePassed: 0, want: RecommendationStrongNotRecommend},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := aggregateRecommendation(tc.score, MustHaveSummary{Passed: tc.mustHavePassed, Total: 2}, false); got != tc.want {
				t.Fatalf("recommendation = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCandidateMatchAggregatorDeterministicAndRejectsModelCreatedRequirements(t *testing.T) {
	profile := JobRequirementProfile{ProfileVersion: JobRequirementProfileVersion, Requirements: []JobRequirementItem{
		{ID: "b", Category: RequirementCategoryOther, Label: "岗位适配", Description: "岗位基本适配", Priority: RequirementPriorityMustHave, Weight: .5},
		{ID: "a", Category: RequirementCategoryCoreSkill, Label: "Go 能力", Description: "Go 核心能力", Priority: RequirementPriorityMustHave, Weight: .5},
	}}
	results := []RequirementMatchResult{
		{RequirementID: "b", Status: MatchStatusMatch, Score: 80, Confidence: .8, Evidence: []MatchEvidenceReference{{SourceTable: "resume_skills", SourceID: 7, Snippet: "Go", Reason: "直接命中"}}},
		{RequirementID: "a", Status: MatchStatusStrongMatch, Score: 90, Confidence: .9, Evidence: []MatchEvidenceReference{{SourceTable: "resume_skills", SourceID: 7, Snippet: "Go", Reason: "直接命中"}}},
	}
	aggregator := NewCandidateMatchAggregator("test-v1")
	first, err := aggregator.Aggregate(profile, results, 3, "fixed", "enhanced", false)
	if err != nil {
		t.Fatalf("first Aggregate error = %v", err)
	}
	second, err := aggregator.Aggregate(profile, append([]RequirementMatchResult(nil), results...), 3, "fixed", "enhanced", false)
	if err != nil {
		t.Fatalf("second Aggregate error = %v", err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if !reflect.DeepEqual(firstJSON, secondJSON) {
		t.Fatalf("fixed aggregation differs:\n%s\n%s", firstJSON, secondJSON)
	}
	if first.Breakdown.RequirementResults[0].RequirementID != "a" {
		t.Fatalf("result order = %+v, want stable requirement ID order", first.Breakdown.RequirementResults)
	}
	unknown := append(results, RequirementMatchResult{RequirementID: "model-overall", Status: MatchStatusStrongMatch, Score: 100})
	if _, err := aggregator.Aggregate(profile, unknown, 3, "fixed", "enhanced", false); err == nil {
		t.Fatal("model-created requirement unexpectedly controlled aggregation")
	}
}

func TestCandidateMatchAggregatorWeakRiskPenaltyIsCapped(t *testing.T) {
	requirements := make([]JobRequirementItem, 0, 12)
	results := make([]RequirementMatchResult, 0, 12)
	for index := 0; index < 12; index++ {
		id := string(rune('a' + index))
		requirements = append(requirements, JobRequirementItem{ID: id, Category: RequirementCategoryOther, Label: "岗位要求", Description: "岗位要求说明", Priority: RequirementPriorityNiceToHave, Weight: 1.0 / 12})
		results = append(results, RequirementMatchResult{RequirementID: id, Status: MatchStatusWeakEvidence, Score: 100, Confidence: .5})
	}
	_, penalty := aggregationRiskPenalty(requirements, results)
	if penalty != 20 {
		t.Fatalf("penalty = %.1f, want capped 20", penalty)
	}
}
