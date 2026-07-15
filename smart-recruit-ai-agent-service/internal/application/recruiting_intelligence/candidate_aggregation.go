package recruiting_intelligence

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

const CandidateMatchScorerVersion = "candidate-match-scorer-v1"

const (
	RecommendationStrongRecommend    = "strong_recommend"
	RecommendationRecommend          = "recommend"
	RecommendationReview             = "review"
	RecommendationNotRecommend       = "not_recommend"
	RecommendationStrongNotRecommend = "strong_not_recommend"
)

var ErrCandidateAggregation = errors.New("candidate match aggregation failed")

type ScoreDimension struct {
	Name   string  `json:"name"`
	Label  string  `json:"label"`
	Weight float64 `json:"weight"`
	Score  float64 `json:"score"`
}

type MustHaveSummary struct {
	Passed  int `json:"passed"`
	Partial int `json:"partial"`
	Missing int `json:"missing"`
	Total   int `json:"total"`
}

type ScoringPolicy struct {
	MustHaveWeight        float64 `json:"must_have_weight"`
	CoreSkillWeight       float64 `json:"core_skill_weight"`
	ExperienceWeight      float64 `json:"experience_weight"`
	GrowthPotentialWeight float64 `json:"growth_potential_weight"`
	RiskPenaltyMax        float64 `json:"risk_penalty_max"`
	KnockoutEnabled       bool    `json:"knockout_enabled"`
}

type AggregationSignal struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type EnhancedScoreBreakdown struct {
	ScorerVersion      string                   `json:"scorer_version"`
	InputHash          string                   `json:"input_hash"`
	ScorerType         string                   `json:"scorer_type"`
	FallbackUsed       bool                     `json:"fallback_used"`
	RequirementProfile JobRequirementProfile    `json:"requirement_profile"`
	RequirementResults []RequirementMatchResult `json:"requirement_results"`
	Dimensions         []ScoreDimension         `json:"dimensions"`
	MustHaveResult     MustHaveSummary          `json:"must_have_result"`
	ScoringPolicy      ScoringPolicy            `json:"scoring_policy"`
	RiskPenalty        float64                  `json:"risk_penalty"`
}

type CandidateAggregationResult struct {
	OverallScore   float64
	Recommendation string
	Summary        string
	Strengths      []AggregationSignal
	Risks          []AggregationSignal
	Breakdown      EnhancedScoreBreakdown
}

type CandidateMatchAggregator struct {
	scorerVersion string
}

func NewCandidateMatchAggregator(scorerVersion string) *CandidateMatchAggregator {
	if strings.TrimSpace(scorerVersion) == "" {
		scorerVersion = CandidateMatchScorerVersion
	}
	return &CandidateMatchAggregator{scorerVersion: scorerVersion}
}

func (a *CandidateMatchAggregator) Aggregate(profile JobRequirementProfile, results []RequirementMatchResult, experienceYears float64, inputHash, scorerType string, fallbackUsed bool) (CandidateAggregationResult, error) {
	if a == nil || len(profile.Requirements) == 0 || strings.TrimSpace(inputHash) == "" || strings.TrimSpace(scorerType) == "" {
		return CandidateAggregationResult{}, ErrCandidateAggregation
	}
	requirements := StableJobRequirements(profile)
	profile = JobRequirementProfile{ProfileVersion: profile.ProfileVersion, Requirements: requirements}
	resultByID := make(map[string]RequirementMatchResult, len(results))
	for _, result := range results {
		if _, duplicate := resultByID[result.RequirementID]; duplicate {
			return CandidateAggregationResult{}, fmt.Errorf("%w: duplicate requirement result", ErrCandidateAggregation)
		}
		resultByID[result.RequirementID] = result
	}
	stableResults := make([]RequirementMatchResult, 0, len(requirements))
	for _, requirement := range requirements {
		result, exists := resultByID[requirement.ID]
		if !exists {
			return CandidateAggregationResult{}, fmt.Errorf("%w: missing requirement result", ErrCandidateAggregation)
		}
		stableResults = append(stableResults, result)
		delete(resultByID, requirement.ID)
	}
	if len(resultByID) != 0 {
		return CandidateAggregationResult{}, fmt.Errorf("%w: unknown requirement result", ErrCandidateAggregation)
	}

	mustHave := weightedRequirementScore(requirements, stableResults, func(item JobRequirementItem) bool { return item.Priority == RequirementPriorityMustHave }, 0)
	coreSkills := weightedRequirementScore(requirements, stableResults, func(item JobRequirementItem) bool { return item.Category == RequirementCategoryCoreSkill }, 50)
	experience := weightedRequirementScore(requirements, stableResults, func(item JobRequirementItem) bool { return item.Category == RequirementCategoryExperience }, experienceDefaultScore(experienceYears))
	growth := weightedRequirementScore(requirements, stableResults, func(item JobRequirementItem) bool {
		return item.Priority == RequirementPriorityNiceToHave || item.Priority == RequirementPrioritySoftSkill
	}, 50)

	knockoutMissing, riskPenalty := aggregationRiskPenalty(requirements, stableResults)
	overall := mustHave*0.40 + coreSkills*0.25 + experience*0.20 + growth*0.10
	overall = roundAggregationScore(math.Max(0, overall-riskPenalty))
	if knockoutMissing > 0 {
		overall = math.Min(overall, 40)
	}
	mustHaveSummary := aggregateMustHave(requirements, stableResults)
	recommendation := aggregateRecommendation(overall, mustHaveSummary, knockoutMissing > 0)

	return CandidateAggregationResult{
		OverallScore:   overall,
		Recommendation: recommendation,
		Summary:        aggregateChineseSummary(overall, mustHaveSummary, knockoutMissing),
		Strengths:      aggregateStrengths(requirements, stableResults),
		Risks:          aggregateRisks(requirements, stableResults, knockoutMissing),
		Breakdown: EnhancedScoreBreakdown{
			ScorerVersion: a.scorerVersion, InputHash: inputHash, ScorerType: scorerType, FallbackUsed: fallbackUsed,
			RequirementProfile: profile, RequirementResults: stableResults,
			Dimensions: []ScoreDimension{
				{Name: "must_have", Label: "必备要求满足度", Weight: 0.40, Score: mustHave},
				{Name: "core_skills", Label: "核心技能匹配", Weight: 0.25, Score: coreSkills},
				{Name: "experience", Label: "项目/经验证据", Weight: 0.20, Score: experience},
				{Name: "growth", Label: "成长潜力与加分项", Weight: 0.10, Score: growth},
			},
			MustHaveResult: mustHaveSummary,
			ScoringPolicy:  ScoringPolicy{MustHaveWeight: 0.40, CoreSkillWeight: 0.25, ExperienceWeight: 0.20, GrowthPotentialWeight: 0.10, RiskPenaltyMax: 20, KnockoutEnabled: true},
			RiskPenalty:    riskPenalty,
		},
	}, nil
}

func weightedRequirementScore(requirements []JobRequirementItem, results []RequirementMatchResult, include func(JobRequirementItem) bool, defaultScore float64) float64 {
	byID := make(map[string]RequirementMatchResult, len(results))
	for _, result := range results {
		byID[result.RequirementID] = result
	}
	var totalWeight, weighted float64
	for _, requirement := range requirements {
		if !include(requirement) {
			continue
		}
		result := byID[requirement.ID]
		weighted += result.Score * requirement.Weight
		totalWeight += requirement.Weight
	}
	if totalWeight == 0 {
		return defaultScore
	}
	return roundAggregationScore(weighted / totalWeight)
}

func experienceDefaultScore(years float64) float64 {
	switch {
	case years >= 5:
		return 85
	case years >= 3:
		return 70
	case years >= 1:
		return 50
	default:
		return 30
	}
}

func aggregationRiskPenalty(requirements []JobRequirementItem, results []RequirementMatchResult) (int, float64) {
	knockoutByID := make(map[string]bool, len(requirements))
	for _, requirement := range requirements {
		knockoutByID[requirement.ID] = requirement.Knockout
	}
	knockoutMissing := 0
	penalty := 0.0
	for _, result := range results {
		switch result.Status {
		case MatchStatusMissing, MatchStatusConflict:
			penalty += 5
			if knockoutByID[result.RequirementID] {
				knockoutMissing++
				penalty += 10
			}
		case MatchStatusWeakEvidence:
			penalty += 2
		}
	}
	return knockoutMissing, math.Min(penalty, 20)
}

func aggregateMustHave(requirements []JobRequirementItem, results []RequirementMatchResult) MustHaveSummary {
	byID := make(map[string]RequirementMatchResult, len(results))
	for _, result := range results {
		byID[result.RequirementID] = result
	}
	var summary MustHaveSummary
	for _, requirement := range requirements {
		if requirement.Priority != RequirementPriorityMustHave {
			continue
		}
		summary.Total++
		switch byID[requirement.ID].Status {
		case MatchStatusStrongMatch, MatchStatusMatch:
			summary.Passed++
		case MatchStatusPartialMatch, MatchStatusWeakEvidence:
			summary.Partial++
		default:
			summary.Missing++
		}
	}
	return summary
}

func aggregateRecommendation(score float64, mustHave MustHaveSummary, knockout bool) string {
	if knockout {
		return RecommendationStrongNotRecommend
	}
	ratio := 0.0
	if mustHave.Total > 0 {
		ratio = float64(mustHave.Passed) / float64(mustHave.Total)
	}
	switch {
	case score >= 85 && ratio >= 0.8:
		return RecommendationStrongRecommend
	case score >= 70 && ratio >= 0.6:
		return RecommendationRecommend
	case score >= 50:
		return RecommendationReview
	case score >= 30:
		return RecommendationNotRecommend
	default:
		return RecommendationStrongNotRecommend
	}
}

func aggregateChineseSummary(score float64, mustHave MustHaveSummary, knockoutMissing int) string {
	if mustHave.Total == 0 {
		return "暂无岗位要求数据，无法生成评估摘要"
	}
	parts := []string{fmt.Sprintf("综合评分 %.1f 分。", score)}
	if knockoutMissing > 0 {
		parts = append(parts, fmt.Sprintf("存在 %d 项关键否决条件未满足，建议慎重评估。", knockoutMissing))
	}
	parts = append(parts, fmt.Sprintf("必备要求: %d/%d 项满足，%d 项部分满足，%d 项缺失。", mustHave.Passed, mustHave.Total, mustHave.Partial, mustHave.Missing))
	if mustHave.Passed >= mustHave.Total/2 {
		parts = append(parts, "候选人在核心要求上有较好的匹配度。")
	} else {
		parts = append(parts, "候选人在多项核心要求上存在差距，需进一步核实。")
	}
	return strings.Join(parts, " ")
}

func aggregateStrengths(requirements []JobRequirementItem, results []RequirementMatchResult) []AggregationSignal {
	labels := requirementLabels(requirements)
	var matched, strong []string
	for _, result := range results {
		if result.Status == MatchStatusStrongMatch || result.Status == MatchStatusMatch {
			matched = append(matched, labels[result.RequirementID])
		}
		if result.Status == MatchStatusStrongMatch {
			strong = append(strong, labels[result.RequirementID])
		}
	}
	var signals []AggregationSignal
	if len(matched) > 0 {
		signals = append(signals, AggregationSignal{Code: "matched_requirements", Message: "已满足要求: " + strings.Join(matched, "、")})
	}
	if len(strong) > 0 {
		signals = append(signals, AggregationSignal{Code: "strong_match_requirements", Message: "高度匹配: " + strings.Join(strong, "、")})
	}
	return signals
}

func aggregateRisks(requirements []JobRequirementItem, results []RequirementMatchResult, knockoutMissing int) []AggregationSignal {
	labels := requirementLabels(requirements)
	var signals []AggregationSignal
	if knockoutMissing > 0 {
		signals = append(signals, AggregationSignal{Code: "knockout_missing", Message: fmt.Sprintf("存在 %d 项关键否决条件未满足", knockoutMissing)})
	}
	var missing []string
	for _, result := range results {
		if result.Status == MatchStatusMissing || result.Status == MatchStatusConflict {
			missing = append(missing, labels[result.RequirementID])
		}
	}
	if len(missing) > 0 {
		signals = append(signals, AggregationSignal{Code: "missing_requirements", Message: "缺失要求: " + strings.Join(missing, "、")})
	}
	for _, result := range results {
		if strings.TrimSpace(result.Risk) != "" {
			signals = append(signals, AggregationSignal{Code: "requirement_risk_" + result.RequirementID, Message: result.Risk})
		}
	}
	return signals
}

func requirementLabels(requirements []JobRequirementItem) map[string]string {
	labels := make(map[string]string, len(requirements))
	for _, requirement := range requirements {
		labels[requirement.ID] = requirement.Label
	}
	return labels
}

func roundAggregationScore(value float64) float64 {
	return math.Round(math.Max(0, math.Min(100, value))*10) / 10
}

// StableAggregationSignals is useful to callers that merge signals from
// independent deterministic scorers.
func StableAggregationSignals(signals []AggregationSignal) []AggregationSignal {
	cloned := append([]AggregationSignal(nil), signals...)
	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].Code != cloned[j].Code {
			return cloned[i].Code < cloned[j].Code
		}
		return cloned[i].Message < cloned[j].Message
	})
	return cloned
}
