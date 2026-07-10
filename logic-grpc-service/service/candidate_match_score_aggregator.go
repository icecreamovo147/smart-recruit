package service

import (
	"fmt"
	"math"
	"strings"

	"logic-grpc-service/repository"
)

const (
	RecommendationStrongRecommend    = "strong_recommend"
	RecommendationRecommend          = "recommend"
	RecommendationReview             = "review"
	RecommendationNotRecommend       = "not_recommend"
	RecommendationStrongNotRecommend = "strong_not_recommend"
)

type ScoreAggregator struct {
	ScorerVersion string
}

func NewScoreAggregator(scorerVersion string) *ScoreAggregator {
	return &ScoreAggregator{ScorerVersion: scorerVersion}
}

type AggregationResult struct {
	OverallScore   float64
	Recommendation string
	Summary        string
	Strengths      []candidateMatchSignal
	Risks          []candidateMatchSignal
	Breakdown      EnhancedScoreBreakdown
}

func (a *ScoreAggregator) Aggregate(
	profile *JobRequirementProfile,
	results []RequirementMatchResult,
	index *EvidenceIndex,
	snapshot *repository.ResumeProfileSnapshot,
	inputHash string,
	scorerType string,
	fallbackUsed bool,
) AggregationResult {
	if len(profile.Requirements) == 0 {
		return a.emptyResult(inputHash, scorerType, fallbackUsed)
	}

	resultByID := make(map[string]RequirementMatchResult)
	for _, r := range results {
		resultByID[r.RequirementID] = r
	}

	mustHaveResults := a.filterByPriority(profile.Requirements, results, RequirementPriorityMustHave)
	skillResults := a.filterByCategory(profile.Requirements, results, RequirementCategoryCoreSkill)
	experienceResults := a.filterByCategory(profile.Requirements, results, RequirementCategoryExperience)
	niceToHaveResults := a.filterByPriority(profile.Requirements, results, RequirementPriorityNiceToHave)
	softSkillResults := a.filterByPriority(profile.Requirements, results, RequirementPrioritySoftSkill)

	hasKnockoutMissing := false
	knockoutMissingCount := 0
	for _, req := range profile.Requirements {
		if !req.Knockout {
			continue
		}
		if result, ok := resultByID[req.ID]; ok {
			if result.Status == MatchStatusMissing || result.Status == MatchStatusConflict {
				hasKnockoutMissing = true
				knockoutMissingCount++
			}
		}
	}

	mustHaveScore := a.computeMustHaveScore(mustHaveResults)
	skillScore := a.computeSkillScore(skillResults)
	experienceScore := a.computeExperienceScore(experienceResults, snapshot)
	growthScore := a.computeGrowthScore(append(niceToHaveResults, softSkillResults...))
	riskPenalty := a.computeRiskPenalty(results, knockoutMissingCount)

	dimensions := []ScoreDimension{
		{Name: "must_have", Label: "必备要求满足度", Weight: 0.40, Score: mustHaveScore},
		{Name: "core_skills", Label: "核心技能匹配", Weight: 0.25, Score: skillScore},
		{Name: "experience", Label: "项目/经验证据", Weight: 0.20, Score: experienceScore},
		{Name: "growth", Label: "成长潜力与加分项", Weight: 0.10, Score: growthScore},
	}

	overall := mustHaveScore*0.40 + skillScore*0.25 + experienceScore*0.20 + growthScore*0.10
	overall = math.Max(0, overall-riskPenalty)
	overall = roundCandidateMatchScore(math.Min(100, overall))

	if hasKnockoutMissing {
		overall = math.Min(overall, 40)
	}

	recommendation := a.computeRecommendation(overall, mustHaveResults, hasKnockoutMissing)
	strengths := a.computeStrengths(results, profile)
	risks := a.computeRisks(results, profile, hasKnockoutMissing, knockoutMissingCount)

	mustHaveSummary := computeMustHaveSummary(profile, resultByID)

	summary := a.generateChineseSummary(overall, mustHaveSummary, knockoutMissingCount)

	return AggregationResult{
		OverallScore:   overall,
		Recommendation: recommendation,
		Summary:        summary,
		Strengths:      strengths,
		Risks:          risks,
		Breakdown: EnhancedScoreBreakdown{
			ScorerVersion:      a.ScorerVersion,
			InputHash:          inputHash,
			ScorerType:         scorerType,
			FallbackUsed:       fallbackUsed,
			RequirementProfile: profile,
			RequirementResults: results,
			Dimensions:         dimensions,
			MustHaveResult:     mustHaveSummary,
			ScoringPolicy: &ScoringPolicy{
				MustHaveWeight:        0.40,
				CoreSkillWeight:       0.25,
				ExperienceWeight:      0.20,
				GrowthPotentialWeight: 0.10,
				RiskPenaltyMax:        20,
				KnockoutEnabled:       true,
			},
		},
	}
}

func (a *ScoreAggregator) computeMustHaveScore(results []requirementScored) float64 {
	if len(results) == 0 {
		return 0
	}
	totalWeight := 0.0
	weightedScore := 0.0
	for _, r := range results {
		weightedScore += r.Score * r.Weight
		totalWeight += r.Weight
	}
	if totalWeight == 0 {
		return 0
	}
	return roundCandidateMatchScore(weightedScore / totalWeight)
}

func (a *ScoreAggregator) computeSkillScore(results []requirementScored) float64 {
	if len(results) == 0 {
		return 50
	}
	totalWeight := 0.0
	weightedScore := 0.0
	for _, r := range results {
		weightedScore += r.Score * r.Weight
		totalWeight += r.Weight
	}
	if totalWeight == 0 {
		return 50
	}
	return roundCandidateMatchScore(weightedScore / totalWeight)
}

func (a *ScoreAggregator) computeExperienceScore(results []requirementScored, snapshot *repository.ResumeProfileSnapshot) float64 {
	if len(results) > 0 {
		totalWeight := 0.0
		weightedScore := 0.0
		for _, r := range results {
			weightedScore += r.Score * r.Weight
			totalWeight += r.Weight
		}
		if totalWeight > 0 {
			return roundCandidateMatchScore(weightedScore / totalWeight)
		}
	}
	if snapshot == nil {
		return 30
	}
	years := snapshot.Profile.TotalExperience
	if years >= 5 {
		return 85
	} else if years >= 3 {
		return 70
	} else if years >= 1 {
		return 50
	}
	return 30
}

func (a *ScoreAggregator) computeGrowthScore(results []requirementScored) float64 {
	if len(results) == 0 {
		return 50
	}
	totalWeight := 0.0
	weightedScore := 0.0
	for _, r := range results {
		weightedScore += r.Score * r.Weight
		totalWeight += r.Weight
	}
	if totalWeight == 0 {
		return 50
	}
	return roundCandidateMatchScore(weightedScore / totalWeight)
}

func (a *ScoreAggregator) computeRiskPenalty(results []RequirementMatchResult, knockoutMissing int) float64 {
	penalty := 0.0
	for _, r := range results {
		if r.Status == MatchStatusMissing || r.Status == MatchStatusConflict {
			penalty += 5
		} else if r.Status == MatchStatusWeakEvidence {
			penalty += 2
		}
	}
	penalty += float64(knockoutMissing) * 10
	return math.Min(penalty, 20)
}

func (a *ScoreAggregator) computeRecommendation(score float64, mustHaveResults []requirementScored, hasKnockoutMissing bool) string {
	if hasKnockoutMissing {
		return RecommendationStrongNotRecommend
	}
	mustHavePassed := 0
	for _, r := range mustHaveResults {
		if r.Status == MatchStatusStrongMatch || r.Status == MatchStatusMatch {
			mustHavePassed++
		}
	}
	mustHaveRatio := 0.0
	if len(mustHaveResults) > 0 {
		mustHaveRatio = float64(mustHavePassed) / float64(len(mustHaveResults))
	}

	if score >= 85 && mustHaveRatio >= 0.8 {
		return RecommendationStrongRecommend
	}
	if score >= 70 && mustHaveRatio >= 0.6 {
		return RecommendationRecommend
	}
	if score >= 50 {
		return RecommendationReview
	}
	if score >= 30 {
		return RecommendationNotRecommend
	}
	return RecommendationStrongNotRecommend
}

func (a *ScoreAggregator) computeStrengths(results []RequirementMatchResult, profile *JobRequirementProfile) []candidateMatchSignal {
	var strengths []candidateMatchSignal

	var matchedLabels []string
	for _, result := range results {
		if result.Status == MatchStatusStrongMatch || result.Status == MatchStatusMatch {
			for _, req := range profile.Requirements {
				if req.ID == result.RequirementID {
					matchedLabels = append(matchedLabels, req.Label)
					break
				}
			}
		}
	}
	if len(matchedLabels) > 0 {
		strengths = append(strengths, candidateMatchSignal{
			Code:    "matched_requirements",
			Message: "已满足要求: " + strings.Join(matchedLabels, "、"),
		})
	}

	var strongMatches []string
	for _, result := range results {
		if result.Status == MatchStatusStrongMatch {
			for _, req := range profile.Requirements {
				if req.ID == result.RequirementID {
					strongMatches = append(strongMatches, req.Label)
					break
				}
			}
		}
	}
	if len(strongMatches) > 0 {
		strengths = append(strengths, candidateMatchSignal{
			Code:    "strong_match_requirements",
			Message: "高度匹配: " + strings.Join(strongMatches, "、"),
		})
	}

	return strengths
}

func (a *ScoreAggregator) computeRisks(results []RequirementMatchResult, profile *JobRequirementProfile, hasKnockoutMissing bool, knockoutMissingCount int) []candidateMatchSignal {
	var risks []candidateMatchSignal

	if hasKnockoutMissing {
		risks = append(risks, candidateMatchSignal{
			Code:    "knockout_missing",
			Message: fmt.Sprintf("存在 %d 项关键否决条件未满足", knockoutMissingCount),
		})
	}

	var missingLabels []string
	for _, result := range results {
		if result.Status == MatchStatusMissing || result.Status == MatchStatusConflict {
			for _, req := range profile.Requirements {
				if req.ID == result.RequirementID {
					missingLabels = append(missingLabels, req.Label)
					break
				}
			}
		}
	}
	if len(missingLabels) > 0 {
		risks = append(risks, candidateMatchSignal{
			Code:    "missing_requirements",
			Message: "缺失要求: " + strings.Join(missingLabels, "、"),
		})
	}

	for _, result := range results {
		if result.Risk != "" {
			risks = append(risks, candidateMatchSignal{
				Code:    "requirement_risk_" + result.RequirementID,
				Message: result.Risk,
			})
		}
	}

	return risks
}

func (a *ScoreAggregator) generateChineseSummary(score float64, mustHave *MustHaveSummary, knockoutMissing int) string {
	if mustHave.Total == 0 {
		return "暂无岗位要求数据，无法生成评估摘要"
	}
	parts := []string{
		fmt.Sprintf("综合评分 %.1f 分。", score),
	}
	if knockoutMissing > 0 {
		parts = append(parts, fmt.Sprintf("存在 %d 项关键否决条件未满足，建议慎重评估。", knockoutMissing))
	}
	parts = append(parts,
		fmt.Sprintf("必备要求: %d/%d 项满足，%d 项部分满足，%d 项缺失。", mustHave.Passed, mustHave.Total, mustHave.Partial, mustHave.Missing),
	)
	if mustHave.Passed >= mustHave.Total/2 {
		parts = append(parts, "候选人在核心要求上有较好的匹配度。")
	} else {
		parts = append(parts, "候选人在多项核心要求上存在差距，需进一步核实。")
	}
	return strings.Join(parts, " ")
}

func (a *ScoreAggregator) emptyResult(inputHash, scorerType string, fallbackUsed bool) AggregationResult {
	return AggregationResult{
		OverallScore:   0,
		Recommendation: RecommendationNotRecommend,
		Summary:        "暂无评估数据",
		Strengths:      []candidateMatchSignal{},
		Risks:          []candidateMatchSignal{{Code: "no_data", Message: "缺少岗位要求数据"}},
		Breakdown: EnhancedScoreBreakdown{
			ScorerVersion: a.ScorerVersion,
			InputHash:     inputHash,
			ScorerType:    scorerType,
			FallbackUsed:  fallbackUsed,
			Dimensions:    []ScoreDimension{},
		},
	}
}

type requirementScored struct {
	RequirementID string
	Status        string
	Score         float64
	Weight        float64
	Confidence    float64
}

func (a *ScoreAggregator) filterByPriority(requirements []JobRequirementItem, results []RequirementMatchResult, priority string) []requirementScored {
	resultByID := make(map[string]RequirementMatchResult)
	for _, r := range results {
		resultByID[r.RequirementID] = r
	}
	var scored []requirementScored
	for _, req := range requirements {
		if req.Priority != priority {
			continue
		}
		if result, ok := resultByID[req.ID]; ok {
			scored = append(scored, requirementScored{
				RequirementID: req.ID,
				Status:        result.Status,
				Score:         result.Score,
				Weight:        req.Weight,
				Confidence:    result.Confidence,
			})
		}
	}
	return scored
}

func (a *ScoreAggregator) filterByCategory(requirements []JobRequirementItem, results []RequirementMatchResult, category string) []requirementScored {
	resultByID := make(map[string]RequirementMatchResult)
	for _, r := range results {
		resultByID[r.RequirementID] = r
	}
	var scored []requirementScored
	for _, req := range requirements {
		if req.Category != category {
			continue
		}
		if result, ok := resultByID[req.ID]; ok {
			scored = append(scored, requirementScored{
				RequirementID: req.ID,
				Status:        result.Status,
				Score:         result.Score,
				Weight:        req.Weight,
				Confidence:    result.Confidence,
			})
		}
	}
	return scored
}

func computeMustHaveSummary(profile *JobRequirementProfile, resultByID map[string]RequirementMatchResult) *MustHaveSummary {
	summary := &MustHaveSummary{}
	for _, req := range profile.Requirements {
		if req.Priority != RequirementPriorityMustHave {
			continue
		}
		summary.Total++
		if result, ok := resultByID[req.ID]; ok {
			switch result.Status {
			case MatchStatusStrongMatch, MatchStatusMatch:
				summary.Passed++
			case MatchStatusPartialMatch, MatchStatusWeakEvidence:
				summary.Partial++
			case MatchStatusMissing, MatchStatusConflict:
				summary.Missing++
			}
		} else {
			summary.Missing++
		}
	}
	return summary
}

func ComputeRecommendationLabel(recommendation string) string {
	labels := map[string]string{
		RecommendationStrongRecommend:    "强烈推荐",
		RecommendationRecommend:          "推荐推进",
		RecommendationReview:             "谨慎评估",
		RecommendationNotRecommend:       "不建议推进",
		RecommendationStrongNotRecommend: "强烈不建议",
	}
	if label, ok := labels[recommendation]; ok {
		return label
	}
	return recommendation
}
