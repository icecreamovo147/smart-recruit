package recruiting_intelligence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

const LegacyCandidateMatchScorerVersion = "candidate-match-scorer-v1"

const (
	LegacyRecommendationStrongMatch    = "strong_match"
	LegacyRecommendationPossibleMatch  = "possible_match"
	LegacyRecommendationNeedsReview    = "needs_review"
	LegacyRecommendationNotRecommended = "not_recommended"
)

type LegacyCandidateMatchInput struct {
	JobID                int64
	Job                  JobRequirementSource
	Evidence             EvidenceIndex
	ProfileID            uint64
	ResumeID             int64
	ResumeParsedText     string
	ResumeText           string
	SkillNames           []string
	ApplicationEducation string
	CandidateProfile     *LegacyCandidateProfile
	FullName             string
	Headline             string
	Summary              string
	HighestDegree        string
	ExperienceYears      float64
}

type LegacyCandidateProfile struct {
	ID             uint64
	RealName       string
	Phone          string
	Education      string
	School         string
	WorkExperience string
	Skills         string
	IsComplete     int32
}

type LegacyScoreDimension struct {
	Name    string   `json:"name"`
	Weight  float64  `json:"weight"`
	Score   float64  `json:"score"`
	Matched []string `json:"matched,omitempty"`
	Missing []string `json:"missing,omitempty"`
}

type LegacyScoreBreakdown struct {
	ScorerVersion       string                 `json:"scorer_version"`
	InputHash           string                 `json:"input_hash"`
	MissingRequirements []string               `json:"missing_requirements"`
	Dimensions          []LegacyScoreDimension `json:"dimensions"`
}

type LegacyCandidateMatchResult struct {
	OverallScore   float64
	Recommendation string
	Summary        string
	Strengths      []AggregationSignal
	Risks          []AggregationSignal
	Breakdown      LegacyScoreBreakdown
	Evidence       []MatchEvidenceReference
}

var legacyCandidateTokenRE = regexp.MustCompile(`[a-z0-9+#.]+`)

// ScoreCandidateMatchLegacy restores the dev deterministic five-dimension
// scorer. ResumeText and SkillNames are the complete, in-memory dev scoring
// inputs; Evidence remains independently bounded/redacted for persistence.
// The scorer never invokes a model or mutates persistence state.
func ScoreCandidateMatchLegacy(input LegacyCandidateMatchInput) LegacyCandidateMatchResult {
	jobText := strings.Join([]string{input.Job.JobTitle, input.Job.Department, input.Job.Location, input.Job.Description, input.Job.Requirements}, " ")
	resumeText := input.ResumeText
	jobTokens := legacyUniqueTokens(jobText)
	resumeTokens := legacyTokenSet(resumeText)
	requirementTerms := legacyRequirementKeywords(input.Job.Requirements)
	matchedRequirements, missingRequirements := legacySplitMatches(requirementTerms, resumeTokens)

	skillRequired := legacySkillTerms(jobText)
	if len(skillRequired) == 0 {
		skillRequired = requirementTerms
	}
	skillTokens := make(map[string]struct{})
	for _, name := range input.SkillNames {
		for _, token := range legacyUniqueTokens(name) {
			skillTokens[token] = struct{}{}
		}
	}
	matchedSkills, missingSkills := legacySplitMatches(skillRequired, skillTokens)

	dimensions := []LegacyScoreDimension{
		{Name: "skills", Weight: 0.35, Score: legacyCoverageScore(len(matchedSkills), len(skillRequired)), Matched: matchedSkills, Missing: missingSkills},
		{Name: "requirements", Weight: 0.25, Score: legacyCoverageScore(len(matchedRequirements), len(requirementTerms)), Matched: matchedRequirements, Missing: missingRequirements},
		{Name: "experience", Weight: 0.20, Score: legacyExperienceScore(jobTokens, input.ExperienceYears, resumeTokens)},
		{Name: "education", Weight: 0.10, Score: legacyEducationScore(jobText, input.HighestDegree, input.ApplicationEducation)},
		{Name: "profile", Weight: 0.10, Score: legacyProfileScore(input)},
	}
	overall := 0.0
	for _, dimension := range dimensions {
		overall += dimension.Score * dimension.Weight
	}
	overall = roundAggregationScore(overall)

	risks := legacyRisks(input, missingRequirements, missingSkills)
	strengths := legacyStrengths(input, matchedSkills, matchedRequirements)
	recommendation := legacyRecommendation(overall, len(risks), len(missingRequirements))
	return LegacyCandidateMatchResult{
		OverallScore: overall, Recommendation: recommendation,
		Summary:   legacySummary(overall, recommendation, matchedSkills, missingRequirements, len(risks)),
		Strengths: strengths, Risks: risks,
		Breakdown: LegacyScoreBreakdown{
			ScorerVersion:       LegacyCandidateMatchScorerVersion,
			InputHash:           legacyInputHash(jobText, resumeText),
			MissingRequirements: missingRequirements,
			Dimensions:          dimensions,
		},
		Evidence: legacyEvidence(input, matchedSkills, matchedRequirements),
	}
}

func legacyUniqueTokens(text string) []string {
	seen := make(map[string]struct{})
	for _, match := range legacyCandidateTokenRE.FindAllString(strings.ToLower(text), -1) {
		token := strings.Trim(match, ".")
		if len(token) < 2 || legacyCandidateStopwords[token] {
			continue
		}
		seen[token] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for token := range seen {
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func legacyTokenSet(text string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, token := range legacyUniqueTokens(text) {
		set[token] = struct{}{}
	}
	return set
}

func legacyRequirementKeywords(value string) []string {
	tokens := legacyUniqueTokens(value)
	if len(tokens) > 16 {
		return tokens[:16]
	}
	return tokens
}

func legacySkillTerms(value string) []string {
	var out []string
	for _, token := range legacyUniqueTokens(value) {
		if legacyKnownSkills[token] {
			out = append(out, token)
		}
	}
	return out
}

func legacySplitMatches(required []string, actual map[string]struct{}) ([]string, []string) {
	matched, missing := make([]string, 0, len(required)), make([]string, 0, len(required))
	for _, requirement := range required {
		if _, ok := actual[requirement]; ok {
			matched = append(matched, requirement)
		} else {
			missing = append(missing, requirement)
		}
	}
	return matched, missing
}

func legacyCoverageScore(matched, total int) float64 {
	if total == 0 {
		return 70
	}
	return roundAggregationScore(float64(matched) / float64(total) * 100)
}

func legacyExperienceScore(jobTokens []string, years float64, resumeTokens map[string]struct{}) float64 {
	score := 55.0
	switch {
	case years >= 5:
		score += 25
	case years >= 3:
		score += 18
	case years >= 1:
		score += 10
	}
	matched := 0
	for _, token := range jobTokens {
		if _, ok := resumeTokens[token]; ok {
			matched++
		}
	}
	if len(jobTokens) > 0 {
		score += math.Min(20, float64(matched)/float64(len(jobTokens))*35)
	}
	return roundAggregationScore(math.Min(100, score))
}

func legacyEducationScore(jobText, highestDegree, applicationEducation string) float64 {
	job, candidate := strings.ToLower(jobText), strings.ToLower(highestDegree+" "+applicationEducation)
	if strings.Contains(job, "master") || strings.Contains(job, "硕士") {
		if containsAny(candidate, "master", "硕士", "phd", "doctor", "博士") {
			return 100
		}
		return 45
	}
	if strings.Contains(job, "bachelor") || strings.Contains(job, "本科") {
		if containsAny(candidate, "bachelor", "本科", "master", "硕士", "phd", "博士") {
			return 100
		}
		return 55
	}
	if strings.TrimSpace(candidate) == "" {
		return 60
	}
	return 80
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(text, value) {
			return true
		}
	}
	return false
}

func legacyProfileScore(input LegacyCandidateMatchInput) float64 {
	if input.CandidateProfile == nil {
		return 35
	}
	profile := input.CandidateProfile
	score := 40.0
	for _, field := range []string{profile.RealName, profile.Phone, profile.Education, profile.School, profile.WorkExperience, profile.Skills} {
		if strings.TrimSpace(field) != "" {
			score += 8
		}
	}
	if profile.IsComplete == 1 {
		score += 12
	}
	return roundAggregationScore(score)
}

func legacyRisks(input LegacyCandidateMatchInput, missingRequirements, missingSkills []string) []AggregationSignal {
	var risks []AggregationSignal
	if input.CandidateProfile == nil {
		risks = append(risks, AggregationSignal{Code: "candidate_profile_missing", Message: "Candidate profile is missing."})
	} else if input.CandidateProfile.IsComplete != 1 {
		risks = append(risks, AggregationSignal{Code: "candidate_profile_incomplete", Message: "Candidate profile is incomplete."})
	}
	if strings.TrimSpace(input.ResumeParsedText) == "" {
		risks = append(risks, AggregationSignal{Code: "resume_text_missing", Message: "Resume parsed text is empty, reducing evidence quality."})
	}
	if len(missingRequirements) > 0 {
		risks = append(risks, AggregationSignal{Code: "requirements_gap", Message: "Missing job requirement evidence: " + strings.Join(missingRequirements, ", ")})
	}
	if len(missingSkills) > 0 {
		risks = append(risks, AggregationSignal{Code: "skills_gap", Message: "Missing skill evidence: " + strings.Join(missingSkills, ", ")})
	}
	return risks
}

func legacyStrengths(input LegacyCandidateMatchInput, matchedSkills, matchedRequirements []string) []AggregationSignal {
	var strengths []AggregationSignal
	if len(matchedSkills) > 0 {
		strengths = append(strengths, AggregationSignal{Code: "matched_skills", Message: "Matched skills: " + strings.Join(matchedSkills, ", ")})
	}
	if len(matchedRequirements) > 0 {
		strengths = append(strengths, AggregationSignal{Code: "matched_requirements", Message: "Matched requirements: " + strings.Join(matchedRequirements, ", ")})
	}
	if input.ExperienceYears >= 3 {
		strengths = append(strengths, AggregationSignal{Code: "experience_depth", Message: fmt.Sprintf("Resume profile reports %.1f years of experience.", input.ExperienceYears)})
	}
	return strengths
}

func legacyRecommendation(score float64, riskCount, missingCount int) string {
	if score >= 82 && missingCount <= 1 {
		return LegacyRecommendationStrongMatch
	}
	if score >= 65 && riskCount <= 3 {
		return LegacyRecommendationPossibleMatch
	}
	if score >= 50 {
		return LegacyRecommendationNeedsReview
	}
	return LegacyRecommendationNotRecommended
}

func legacySummary(score float64, recommendation string, matchedSkills, missingRequirements []string, riskCount int) string {
	parts := []string{fmt.Sprintf("Overall score %.1f with recommendation %s.", score, recommendation)}
	if len(matchedSkills) > 0 {
		parts = append(parts, "Matched skills: "+strings.Join(matchedSkills, ", ")+".")
	}
	if len(missingRequirements) > 0 {
		parts = append(parts, "Missing requirements: "+strings.Join(missingRequirements, ", ")+".")
	}
	if riskCount > 0 {
		parts = append(parts, fmt.Sprintf("%d risk signal(s) require review.", riskCount))
	}
	return strings.Join(parts, " ")
}

func legacyEvidence(input LegacyCandidateMatchInput, matchedSkills, matchedRequirements []string) []MatchEvidenceReference {
	wanted := append(append([]string(nil), matchedSkills...), matchedRequirements...)
	seen := make(map[string]struct{})
	var out []MatchEvidenceReference
	for _, unit := range input.Evidence.units {
		matched := false
		for _, term := range wanted {
			if candidateEvidenceContains(unit.Snippet, term) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		key := fmt.Sprintf("%s:%d", unit.SourceTable, unit.SourceID)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, MatchEvidenceReference{SourceTable: unit.SourceTable, SourceID: unit.SourceID, Snippet: unit.Snippet, Reason: "legacy deterministic keyword match"})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SourceTable != out[j].SourceTable {
			return out[i].SourceTable < out[j].SourceTable
		}
		return out[i].SourceID < out[j].SourceID
	})
	return out
}

func legacyInputHash(jobText, resumeText string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{jobText, resumeText, LegacyCandidateMatchScorerVersion}, "\n")))
	return hex.EncodeToString(sum[:])
}

var legacyCandidateStopwords = map[string]bool{
	"and": true, "are": true, "but": true, "for": true, "have": true, "with": true,
	"the": true, "this": true, "that": true, "you": true, "your": true, "our": true,
	"will": true, "can": true, "able": true, "using": true, "use": true, "must": true,
	"plus": true, "nice": true, "good": true, "strong": true, "experience": true,
	"years": true, "year": true, "work": true, "candidate": true, "role": true,
	"required": true, "requirement": true, "requirements": true, "knowledge": true,
	"熟悉": true, "经验": true, "能力": true,
}

var legacyKnownSkills = map[string]bool{
	"aws": true, "azure": true, "c": true, "c++": true, "c#": true, "css": true,
	"docker": true, "elasticsearch": true, "gin": true, "git": true, "go": true,
	"golang": true, "grpc": true, "html": true, "java": true, "javascript": true,
	"kafka": true, "kubernetes": true, "linux": true, "mysql": true, "node": true,
	"postgres": true, "postgresql": true, "python": true, "rabbitmq": true,
	"react": true, "redis": true, "sql": true, "typescript": true, "vue": true,
}
