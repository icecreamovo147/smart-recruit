package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"go.uber.org/zap"

	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

const defaultMatchConfidence = 0.85

type RequirementMatcher struct {
	llmMatcher *LLMRequirementMatcher
}

func NewRequirementMatcher(llmMatcher *LLMRequirementMatcher) *RequirementMatcher {
	return &RequirementMatcher{
		llmMatcher: llmMatcher,
	}
}

func (m *RequirementMatcher) MatchAll(ctx context.Context, profile *JobRequirementProfile, index *EvidenceIndex, snapshot *repository.ResumeProfileSnapshot) []RequirementMatchResult {
	results := make([]RequirementMatchResult, 0, len(profile.Requirements))

	for _, req := range profile.Requirements {
		result := m.matchOne(ctx, req, index, snapshot)

		if err := result.Validate(); err != nil {
			logger.L().Warn("requirement match result failed validation, degrading to weak_evidence",
				zap.String("requirement_id", req.ID),
				zap.Error(err),
			)
			result.Status = MatchStatusWeakEvidence
			result.Score = 30
			result.Confidence = 0.3
			result.Evidence = nil
			result.Risk = "匹配结果不合法，已降级为弱证据"
		}

		results = append(results, result)
	}

	return results
}

func (m *RequirementMatcher) matchOne(ctx context.Context, req JobRequirementItem, index *EvidenceIndex, snapshot *repository.ResumeProfileSnapshot) RequirementMatchResult {
	result := m.deterministicMatch(req, index, snapshot)
	if result.Status != MatchStatusMissing {
		logger.L().Debug("deterministic match succeeded",
			zap.String("requirement_id", req.ID),
			zap.String("status", result.Status),
			zap.Float64("score", result.Score),
		)
		return result
	}

	if m.llmMatcher != nil && m.llmMatcher.llmConfigSvc != nil {
		llmResult, err := m.llmMatcher.Match(ctx, req, index, snapshot)
		if err == nil && llmResult.Status != MatchStatusMissing {
			logger.L().Debug("llm semantic match succeeded",
				zap.String("requirement_id", req.ID),
				zap.String("status", llmResult.Status),
				zap.Float64("score", llmResult.Score),
			)
			return llmResult
		}
		if err != nil {
			logger.L().Warn("llm semantic match failed, keeping deterministic result",
				zap.String("requirement_id", req.ID),
				zap.Error(err),
			)
		}
	}

	return result
}

func (m *RequirementMatcher) deterministicMatch(req JobRequirementItem, index *EvidenceIndex, snapshot *repository.ResumeProfileSnapshot) RequirementMatchResult {
	if index == nil {
		index = &EvidenceIndex{}
	}
	aliases := GenerateRequirementAliases(&req)

	matchedEvidence := findEvidenceByTerm(aliases, index)

	if len(matchedEvidence) > 0 {
		score := 70.0
		confidence := defaultMatchConfidence
		status := MatchStatusMatch

		exactMatch := false
		for _, alias := range aliases {
			for _, ev := range matchedEvidence {
				if strings.Contains(strings.ToLower(ev.Snippet), strings.ToLower(alias)) {
					exactMatch = true
					break
				}
			}
			if exactMatch {
				break
			}
		}

		if exactMatch {
			score = 85.0
			status = MatchStatusMatch
		} else {
			score = 60.0
			status = MatchStatusPartialMatch
			confidence = 0.65
		}

		if strings.EqualFold(req.Priority, RequirementPriorityMustHave) && len(matchedEvidence) >= 2 {
			score = math.Min(95, score+10)
			status = MatchStatusStrongMatch
			confidence = 0.92
		}

		risk := ""
		if status == MatchStatusPartialMatch {
			risk = "有相关证据但不够直接"
		}

		return RequirementMatchResult{
			RequirementID: req.ID,
			Status:        status,
			Score:         roundCandidateMatchScore(score),
			Confidence:    confidence,
			Evidence:      matchedEvidence,
			Risk:          risk,
		}
	}

	if req.Category == RequirementCategoryEducation {
		return m.matchEducation(req, snapshot)
	}

	if req.Category == RequirementCategoryExperience {
		return m.matchExperience(req, snapshot)
	}

	return RequirementMatchResult{
		RequirementID: req.ID,
		Status:        MatchStatusMissing,
		Score:         0,
		Confidence:    0.5,
		Risk:          fmt.Sprintf("未找到与「%s」相关的证据", req.Label),
	}
}

type degreeLevel int

const (
	degreeUnknown degreeLevel = iota
	degreeAssociate
	degreeBachelor
	degreeMaster
	degreeDoctor
)

var degreeKeywords = map[string]struct {
	level degreeLevel
	label string
}{
	"大专":        {degreeAssociate, "大专"},
	"associate": {degreeAssociate, "大专"},
	"本科":        {degreeBachelor, "本科"},
	"bachelor":  {degreeBachelor, "本科"},
	"硕士":        {degreeMaster, "硕士"},
	"master":    {degreeMaster, "硕士"},
	"博士":        {degreeDoctor, "博士"},
	"phd":       {degreeDoctor, "博士"},
	"doctor":    {degreeDoctor, "博士"},
}

func parseDegreeLevel(text string) (degreeLevel, string) {
	lower := strings.ToLower(text)
	for keyword, info := range degreeKeywords {
		if strings.Contains(lower, keyword) {
			return info.level, info.label
		}
	}
	return degreeUnknown, ""
}

func (m *RequirementMatcher) matchEducation(req JobRequirementItem, snapshot *repository.ResumeProfileSnapshot) RequirementMatchResult {
	if snapshot == nil || len(snapshot.Educations) == 0 {
		return RequirementMatchResult{
			RequirementID: req.ID,
			Status:        MatchStatusMissing,
			Score:         0,
			Confidence:    0.8,
			Risk:          "无教育经历数据",
		}
	}

	reqText := req.Label + " " + req.Description + " " + strings.Join(req.Aliases, " ")
	requiredLevel, requiredLabel := parseDegreeLevel(reqText)
	if requiredLevel == degreeUnknown {
		return RequirementMatchResult{
			RequirementID: req.ID,
			Status:        MatchStatusMatch,
			Score:         80,
			Confidence:    0.7,
			Risk:          "",
		}
	}

	candidateHighest := degreeUnknown
	for _, edu := range snapshot.Educations {
		eduText := edu.Degree + " " + edu.Major + " " + edu.Description
		level, _ := parseDegreeLevel(eduText)
		if level > candidateHighest {
			candidateHighest = level
		}
	}

	score := 0.0
	status := MatchStatusMissing
	risk := "教育背景不满足要求"

	if candidateHighest >= requiredLevel {
		score = 100
		status = MatchStatusMatch
		risk = ""
	} else if candidateHighest == degreeAssociate && requiredLevel <= degreeBachelor {
		score = 30
		status = MatchStatusWeakEvidence
		risk = "学历为大专，不满足" + requiredLabel + "及以上要求"
	} else {
		score = 20
		status = MatchStatusMissing
		risk = "学历不满足" + requiredLabel + "及以上要求"
	}

	var bestEvidence []MatchEvidenceUnit
	for _, edu := range snapshot.Educations {
		bestEvidence = append(bestEvidence, MatchEvidenceUnit{
			SourceTable: "resume_educations",
			SourceID:    edu.ID,
			Snippet:     truncateEvidenceSnippet(edu.School + " | " + edu.Degree + " | " + edu.Major),
		})
	}

	return RequirementMatchResult{
		RequirementID: req.ID,
		Status:        status,
		Score:         roundCandidateMatchScore(score),
		Confidence:    0.85,
		Evidence:      bestEvidence,
		Risk:          risk,
	}
}

func (m *RequirementMatcher) matchExperience(req JobRequirementItem, snapshot *repository.ResumeProfileSnapshot) RequirementMatchResult {
	if snapshot == nil {
		return RequirementMatchResult{
			RequirementID: req.ID,
			Status:        MatchStatusMissing,
			Score:         0,
			Confidence:    0.7,
			Risk:          "无简历画像数据",
		}
	}
	years := snapshot.Profile.TotalExperience
	if years <= 0 && len(snapshot.Experiences) == 0 {
		return RequirementMatchResult{
			RequirementID: req.ID,
			Status:        MatchStatusMissing,
			Score:         0,
			Confidence:    0.7,
			Risk:          "无工作经验数据",
		}
	}

	score := 0.0
	status := MatchStatusWeakEvidence
	risk := "经验不足"

	if years >= 5 {
		score = 90
		status = MatchStatusStrongMatch
		risk = ""
	} else if years >= 3 {
		score = 75
		status = MatchStatusMatch
		risk = ""
	} else if years >= 1 {
		score = 55
		status = MatchStatusPartialMatch
		risk = "经验年限偏少"
	} else if years > 0 {
		score = 30
		risk = "经验年限非常有限"
	}

	var evidence []MatchEvidenceUnit
	for _, exp := range snapshot.Experiences {
		evidence = append(evidence, MatchEvidenceUnit{
			SourceTable: "resume_experiences",
			SourceID:    exp.ID,
			Snippet:     truncateEvidenceSnippet(strings.Join([]string{exp.Title, exp.Company}, " | ")),
		})
	}
	if len(evidence) == 0 && years > 0 {
		evidence = append(evidence, MatchEvidenceUnit{
			SourceTable: "resume_profiles",
			SourceID:    snapshot.Profile.ID,
			Snippet:     fmt.Sprintf("总工作经验 %.1f 年", years),
		})
	}

	return RequirementMatchResult{
		RequirementID: req.ID,
		Status:        status,
		Score:         roundCandidateMatchScore(score),
		Confidence:    0.8,
		Evidence:      evidence,
		Risk:          risk,
	}
}
