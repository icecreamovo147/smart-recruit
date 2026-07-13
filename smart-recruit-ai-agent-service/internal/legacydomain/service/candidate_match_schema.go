package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	RequirementProfileVersionV1 = "job-requirement-profile-v1"

	RequirementPriorityMustHave   = "must_have"
	RequirementPriorityNiceToHave = "nice_to_have"
	RequirementPrioritySoftSkill  = "soft_skill"

	RequirementCategoryCoreSkill  = "core_skill"
	RequirementCategoryExperience = "experience"
	RequirementCategoryEducation  = "education"
	RequirementCategoryCert       = "certification"
	RequirementCategoryDomain     = "domain"
	RequirementCategoryLanguage   = "language"
	RequirementCategorySoftSkill  = "soft_skill"
	RequirementCategoryOther      = "other"

	MatchStatusStrongMatch  = "strong_match"
	MatchStatusMatch        = "match"
	MatchStatusPartialMatch = "partial_match"
	MatchStatusWeakEvidence = "weak_evidence"
	MatchStatusMissing      = "missing"
	MatchStatusConflict     = "conflict"
)

type JobRequirementProfile struct {
	ProfileVersion string               `json:"profile_version"`
	Requirements   []JobRequirementItem `json:"requirements"`
}

type JobRequirementItem struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	Weight      float64  `json:"weight"`
	Knockout    bool     `json:"knockout"`
	Aliases     []string `json:"aliases,omitempty"`
}

func (p *JobRequirementProfile) Validate() error {
	if p == nil {
		return fmt.Errorf("requirement profile is nil")
	}
	if p.ProfileVersion == "" {
		return fmt.Errorf("profile_version is required")
	}
	if len(p.Requirements) == 0 {
		return fmt.Errorf("at least one requirement is required")
	}
	seen := make(map[string]bool)
	for i, req := range p.Requirements {
		if req.ID == "" {
			return fmt.Errorf("requirements[%d].id is required", i)
		}
		if seen[req.ID] {
			return fmt.Errorf("duplicate requirement id: %s", req.ID)
		}
		seen[req.ID] = true
		if req.Label == "" {
			return fmt.Errorf("requirements[%d].label is required for id=%s", i, req.ID)
		}
		if req.Priority != RequirementPriorityMustHave &&
			req.Priority != RequirementPriorityNiceToHave &&
			req.Priority != RequirementPrioritySoftSkill {
			return fmt.Errorf("requirements[%d].priority must be must_have, nice_to_have, or soft_skill", i)
		}
		validCategories := map[string]bool{
			RequirementCategoryCoreSkill:  true,
			RequirementCategoryExperience: true,
			RequirementCategoryEducation:  true,
			RequirementCategoryCert:       true,
			RequirementCategoryDomain:     true,
			RequirementCategoryLanguage:   true,
			RequirementCategorySoftSkill:  true,
			RequirementCategoryOther:      true,
		}
		if !validCategories[req.Category] {
			return fmt.Errorf("requirements[%d].category is invalid: %s", i, req.Category)
		}
		if req.Weight <= 0 {
			return fmt.Errorf("requirements[%d].weight must be positive", i)
		}
	}
	return nil
}

type RequirementMatchResult struct {
	RequirementID string              `json:"requirement_id"`
	Status        string              `json:"status"`
	Score         float64             `json:"score"`
	Confidence    float64             `json:"confidence"`
	Evidence      []MatchEvidenceUnit `json:"evidence"`
	Risk          string              `json:"risk,omitempty"`
}

type MatchEvidenceUnit struct {
	SourceTable string `json:"source_table"`
	SourceID    uint64 `json:"source_id,omitempty"`
	Snippet     string `json:"snippet"`
	Reason      string `json:"reason,omitempty"`
}

func (r *RequirementMatchResult) Validate() error {
	if r == nil {
		return fmt.Errorf("requirement match result is nil")
	}
	if r.RequirementID == "" {
		return fmt.Errorf("requirement_id is required")
	}
	validStatuses := map[string]bool{
		MatchStatusStrongMatch:  true,
		MatchStatusMatch:        true,
		MatchStatusPartialMatch: true,
		MatchStatusWeakEvidence: true,
		MatchStatusMissing:      true,
		MatchStatusConflict:     true,
	}
	if !validStatuses[r.Status] {
		return fmt.Errorf("invalid status %q for requirement %s", r.Status, r.RequirementID)
	}
	if r.Score < 0 || r.Score > 100 {
		return fmt.Errorf("score must be 0-100 for requirement %s", r.RequirementID)
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return fmt.Errorf("confidence must be 0-1 for requirement %s", r.RequirementID)
	}
	if (r.Status == MatchStatusMatch || r.Status == MatchStatusStrongMatch) && len(r.Evidence) == 0 {
		return fmt.Errorf("requirement %s has status %s but no evidence", r.RequirementID, r.Status)
	}
	for i, ev := range r.Evidence {
		if ev.SourceTable == "" {
			return fmt.Errorf("requirement %s evidence[%d].source_table is required", r.RequirementID, i)
		}
		if strings.TrimSpace(ev.Snippet) == "" {
			return fmt.Errorf("requirement %s evidence[%d].snippet is required", r.RequirementID, i)
		}
	}
	return nil
}

type EnhancedScoreBreakdown struct {
	ScorerVersion      string                   `json:"scorer_version"`
	InputHash          string                   `json:"input_hash"`
	ScorerType         string                   `json:"scorer_type"`
	FallbackUsed       bool                     `json:"fallback_used"`
	RequirementProfile *JobRequirementProfile   `json:"requirement_profile,omitempty"`
	RequirementResults []RequirementMatchResult `json:"requirement_results,omitempty"`
	Dimensions         []ScoreDimension         `json:"dimensions"`
	MustHaveResult     *MustHaveSummary         `json:"must_have_result,omitempty"`
	ScoringPolicy      *ScoringPolicy           `json:"scoring_policy,omitempty"`
}

type ScoreDimension struct {
	Name    string   `json:"name"`
	Label   string   `json:"label,omitempty"`
	Weight  float64  `json:"weight"`
	Score   float64  `json:"score"`
	Matched []string `json:"matched,omitempty"`
	Missing []string `json:"missing,omitempty"`
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

func (b *EnhancedScoreBreakdown) Validate() error {
	if b == nil {
		return fmt.Errorf("score breakdown is nil")
	}
	if b.ScorerVersion == "" {
		return fmt.Errorf("scorer_version is required")
	}
	if b.InputHash == "" {
		return fmt.Errorf("input_hash is required")
	}
	if b.ScorerType == "" {
		return fmt.Errorf("scorer_type is required")
	}
	if len(b.Dimensions) == 0 {
		return fmt.Errorf("at least one dimension is required")
	}
	if b.RequirementProfile != nil {
		if err := b.RequirementProfile.Validate(); err != nil {
			return fmt.Errorf("requirement_profile: %w", err)
		}
	}
	for i, req := range b.RequirementResults {
		if err := req.Validate(); err != nil {
			return fmt.Errorf("requirement_results[%d]: %w", i, err)
		}
	}
	return nil
}

func ValidateJobRequirementProfileJSON(raw string) error {
	var profile JobRequirementProfile
	if err := json.Unmarshal([]byte(raw), &profile); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return profile.Validate()
}

func ValidateRequirementMatchResultJSON(raw string) error {
	var result RequirementMatchResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return result.Validate()
}

func ValidateEnhancedScoreBreakdownJSON(raw string) error {
	var breakdown EnhancedScoreBreakdown
	if err := json.Unmarshal([]byte(raw), &breakdown); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return breakdown.Validate()
}

type RequirementExtractor interface {
	Extract(ctx context.Context, jobTitle, department, description, requirements string) (*JobRequirementProfile, string, error)
}

type RequirementExtractorV2 interface {
	RequirementExtractor
	ExtractWithMetadata(ctx context.Context, jobTitle, department, description, requirements string) (*JobRequirementExtractResult, error)
}

type JobRequirementExtractResult struct {
	Profile   *JobRequirementProfile
	InputHash string
	Metadata  ExtractorMetadata
}
