package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"smart-recruit-platform-go/logger"
	"smart-recruit-recruitment-service/internal/legacydomain/model"
	"smart-recruit-recruitment-service/internal/legacydomain/repository"
)

const defaultCandidateMatchScorerVersion = "candidate-match-scorer-v1"

var (
	ErrCandidateMatchMissingApplication   = errors.New("application not found")
	ErrCandidateMatchMissingJob           = errors.New("job not found")
	ErrCandidateMatchMissingResume        = errors.New("resume not found")
	ErrCandidateMatchMissingResumeProfile = errors.New("current resume profile not found")
	ErrCandidateMatchIncompleteProfile    = errors.New("resume profile is incomplete")
)

type CandidateMatchService struct {
	applications   *repository.ApplicationRepo
	jobs           *repository.JobRepo
	profiles       *repository.ProfileRepo
	resumes        *repository.ResumeRepo
	resumeProfiles *repository.ResumeProfileRepo
	matches        *repository.CandidateMatchRepo
	promptRepo     *repository.PromptTemplateRepo
	scorerVersion  string
	now            func() time.Time
	policy         AgentRuntimePolicy
	reqExtractor   RequirementExtractorV2
}

func NewCandidateMatchService(
	applications *repository.ApplicationRepo,
	jobs *repository.JobRepo,
	profiles *repository.ProfileRepo,
	resumes *repository.ResumeRepo,
	resumeProfiles *repository.ResumeProfileRepo,
	matches *repository.CandidateMatchRepo,
) *CandidateMatchService {
	return &CandidateMatchService{
		applications:   applications,
		jobs:           jobs,
		profiles:       profiles,
		resumes:        resumes,
		resumeProfiles: resumeProfiles,
		matches:        matches,
		scorerVersion:  defaultCandidateMatchScorerVersion,
		now:            time.Now,
		policy:         DefaultAgentRuntimePolicy(),
		reqExtractor:   NewJobRequirementExtractorHeuristic(),
	}
}

func (s *CandidateMatchService) WithRuntimePolicy(policy AgentRuntimePolicy) *CandidateMatchService {
	if s != nil {
		s.policy = policy.withDefaults()
	}
	return s
}

func (s *CandidateMatchService) WithRequirementExtractor(extractor RequirementExtractorV2) *CandidateMatchService {
	if s != nil {
		s.reqExtractor = extractor
	}
	return s
}

func (s *CandidateMatchService) WithPromptRepo(repo *repository.PromptTemplateRepo) *CandidateMatchService {
	if s != nil {
		s.promptRepo = repo
	}
	return s
}

func (s *CandidateMatchService) EvaluateApplication(ctx context.Context, applicationID int64, agentRunID *uint64) (*repository.CandidateMatchSnapshot, error) {
	log := logger.GetRequestLogger(ctx)
	policy := s.policy.withDefaults()
	if !policy.CandidateMatch {
		log.Warn("[domain][candidate_match] capability disabled",
			zap.String("capability", "candidate_match"),
			zap.Int64("application_id", applicationID))
		return nil, fmt.Errorf("%w: candidate_match", ErrAgentCapabilityDisabled)
	}
	started := time.Now()
	ctx, cancel := contextWithPolicyTimeout(ctx, policy.CandidateMatchTimeout)
	defer cancel()

	log.Info("[domain][candidate_match] EvaluateApplication started",
		zap.Int64("application_id", applicationID))

	baseApplication, err := s.applications.GetByID(ctx, applicationID)
	if err != nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", err)
		return nil, fmt.Errorf("get application: %w", err)
	}
	if baseApplication == nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", ErrCandidateMatchMissingApplication)
		return nil, ErrCandidateMatchMissingApplication
	}
	log.Info("[domain][candidate_match] application loaded",
		zap.Int64("application_id", applicationID),
		zap.Int64("job_id", baseApplication.JobID))

	job, err := s.jobs.GetByID(ctx, baseApplication.JobID)
	if err != nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", err)
		return nil, fmt.Errorf("get job: %w", err)
	}
	if job == nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", ErrCandidateMatchMissingJob)
		return nil, ErrCandidateMatchMissingJob
	}
	log.Info("[domain][candidate_match] job loaded",
		zap.Int64("job_id", job.ID),
		zap.String("job_title", job.Title))

	application, err := s.applications.GetDetail(ctx, applicationID)
	if err != nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", err)
		return nil, fmt.Errorf("get application detail: %w", err)
	}
	if application == nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", ErrCandidateMatchMissingApplication)
		return nil, ErrCandidateMatchMissingApplication
	}
	log.Info("[domain][candidate_match] application detail loaded",
		zap.Int64("user_id", application.UserID),
		zap.Int64("resume_id", application.ResumeID))

	profile, err := s.profiles.GetByUserID(ctx, application.UserID)
	if err != nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", err)
		return nil, fmt.Errorf("get candidate profile: %w", err)
	}
	log.Info("[domain][candidate_match] candidate profile loaded",
		zap.Bool("profile_exists", profile != nil),
		zap.Int32("profile_complete", func() int32 {
			if profile != nil {
				return profile.IsComplete
			}
			return 0
		}()))

	resume, err := s.resumes.GetByID(ctx, application.ResumeID)
	if err != nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", err)
		return nil, fmt.Errorf("get resume: %w", err)
	}
	if resume == nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", ErrCandidateMatchMissingResume)
		return nil, ErrCandidateMatchMissingResume
	}
	log.Info("[domain][candidate_match] resume loaded",
		zap.Int64("resume_id", resume.ID))

	currentProfile, err := s.resumeProfiles.GetCurrentByResumeID(ctx, resume.ID)
	if err != nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", err)
		return nil, fmt.Errorf("get current resume profile: %w", err)
	}
	if currentProfile == nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", ErrCandidateMatchMissingResumeProfile)
		return nil, ErrCandidateMatchMissingResumeProfile
	}
	log.Info("[domain][candidate_match] current resume profile loaded",
		zap.Uint64("profile_id", currentProfile.ID))

	resumeSnapshot, err := s.resumeProfiles.GetSnapshot(ctx, currentProfile.ID)
	if err != nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", err)
		return nil, fmt.Errorf("get resume profile snapshot: %w", err)
	}
	if isIncompleteResumeProfile(resumeSnapshot) {
		log.Warn("[domain][candidate_match] resume profile is incomplete",
			zap.Int64("application_id", applicationID),
			zap.Uint64("profile_id", currentProfile.ID))
		s.logEvaluationFinished(applicationID, nil, started, "failed", ErrCandidateMatchIncompleteProfile)
		return nil, ErrCandidateMatchIncompleteProfile
	}

	log.Info("[domain][candidate_match] scoring started",
		zap.Int64("application_id", applicationID),
		zap.Bool("semantic_enabled", policy.CandidateMatchSemantic),
		zap.Bool("shadow_mode", policy.CandidateMatchShadow))

	var result candidateMatchScoreResult
	if policy.CandidateMatchSemantic {
		result = s.scoreEnhanced(ctx, job, application, profile, resume, resumeSnapshot)
		runShadowIfEnabled(ctx, &policy, s, job, application, profile, resume, resumeSnapshot, result)
	} else {
		result = s.score(job, application, profile, resume, resumeSnapshot)
		runShadowIfEnabled(ctx, &policy, s, job, application, profile, resume, resumeSnapshot, result)
	}
	log.Info("[domain][candidate_match] scoring finished",
		zap.Float64("overall_score", result.OverallScore),
		zap.String("recommendation", result.Recommendation),
		zap.Int("strengths", len(result.Strengths)),
		zap.Int("risks", len(result.Risks)),
		zap.Int("evidence", len(result.Evidence)))
	snapshot := &repository.CandidateMatchSnapshot{
		Evaluation: model.CandidateMatchEvaluation{
			ApplicationID:   application.ApplicationID,
			JobID:           job.ID,
			CandidateUserID: application.UserID,
			ResumeProfileID: resumeSnapshot.Profile.ID,
			AgentRunID:      agentRunID,
			OverallScore:    result.OverallScore,
			Recommendation:  result.Recommendation,
			Summary:         result.Summary,
			StrengthsJSON:   mustCandidateMatchJSON(result.Strengths),
			RisksJSON:       mustCandidateMatchJSON(result.Risks),
			ScoreBreakdownJSON: func() string {
				if result.ScoreBreakdownJSON != "" {
					return result.ScoreBreakdownJSON
				}
				return mustCandidateMatchJSON(result.Breakdown)
			}(),
			ModelName:   s.scorerVersion,
			EvaluatedAt: s.now().UTC(),
		},
		Evidence: result.Evidence,
	}
	log.Info("[domain][candidate_match] saving evaluation")
	if err := s.matches.SaveEvaluationVersion(ctx, snapshot); err != nil {
		s.logEvaluationFinished(applicationID, nil, started, "failed", err)
		return nil, fmt.Errorf("save candidate match evaluation: %w", err)
	}
	log.Info("[domain][candidate_match] evaluation saved",
		zap.Uint64("evaluation_id", snapshot.Evaluation.ID))
	s.logEvaluationFinished(applicationID, snapshot, started, "succeeded", nil)
	return snapshot, nil
}

func (s *CandidateMatchService) logEvaluationFinished(applicationID int64, snapshot *repository.CandidateMatchSnapshot, started time.Time, status string, err error) {
	fields := []zap.Field{
		zap.String("event", "agent.candidate_match.evaluate"),
		zap.Int64("application_id", applicationID),
		zap.String("status", status),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()),
	}
	if snapshot != nil {
		fields = append(fields,
			zap.Uint64("evaluation_id", snapshot.Evaluation.ID),
			zap.Int32("evaluation_version", snapshot.Evaluation.EvaluationVersion),
			zap.Float64("overall_score", snapshot.Evaluation.OverallScore),
			zap.String("recommendation", snapshot.Evaluation.Recommendation),
		)
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.L().Warn("candidate match evaluation finished", fields...)
		return
	}
	logger.L().Info("candidate match evaluation finished", fields...)
}

func (s *CandidateMatchService) EvaluateApplicationEnhanced(ctx context.Context, applicationID int64, agentRunID *uint64) (*repository.CandidateMatchSnapshot, error) {
	originalPolicy := s.policy
	s.policy = s.policy.withDefaults()
	s.policy.CandidateMatchSemantic = true
	defer func() { s.policy = originalPolicy }()
	return s.EvaluateApplication(ctx, applicationID, agentRunID)
}

func (s *CandidateMatchService) scoreEnhanced(ctx context.Context, job *model.Job, application *repository.ApplicationDetailRow, profile *model.CandidateProfile, resume *model.Resume, snapshot *repository.ResumeProfileSnapshot) candidateMatchScoreResult {
	extractor := s.reqExtractor
	if extractor == nil {
		extractor = NewJobRequirementExtractorHeuristic()
	}
	extractResult, err := extractor.ExtractWithMetadata(ctx, job.Title, job.Department, job.Description, job.Requirements)
	if err != nil || extractResult == nil || extractResult.Profile == nil {
		if s.policy.Fallbacks {
			return s.score(job, application, profile, resume, snapshot)
		}
		return candidateMatchScoreResult{
			OverallScore:   0,
			Recommendation: RecommendationNotRecommend,
			Summary:        "增强评估服务不可用，且 fallback 已禁用",
			Risks:          []candidateMatchSignal{{Code: "enhanced_unavailable", Message: "语义评估服务不可用，fallback 已禁用"}},
		}
	}

	requirementProfile := extractResult.Profile
	inputHash := extractResult.InputHash
	evidenceIndex := BuildEvidenceIndex(snapshot, profile, resume)

	var llmMatcher *LLMRequirementMatcher
	if f, ok := s.reqExtractor.(*fallbackRequirementExtractor); ok {
		if llmExt, ok := f.primary.(*LLMJobRequirementExtractor); ok {
			llmMatcher = NewLLMRequirementMatcher(llmExt.LlmConfigSvc, s.promptRepo)
		}
	}
	matcher := NewRequirementMatcher(llmMatcher)
	matchResults := matcher.MatchAll(ctx, requirementProfile, evidenceIndex, snapshot)

	scorerType := extractResult.Metadata.ExtractorType
	if scorerType == "" {
		scorerType = "hybrid"
	}
	agg := NewScoreAggregator(s.scorerVersion)
	aggregated := agg.Aggregate(requirementProfile, matchResults, evidenceIndex, snapshot, inputHash, scorerType, extractResult.Metadata.FallbackUsed)
	if err := aggregated.Breakdown.Validate(); err != nil {
		logger.L().Warn("enhanced score breakdown validation failed, falling back to legacy", zap.Error(err))
		if s.policy.Fallbacks {
			return s.score(job, application, profile, resume, snapshot)
		}
		return candidateMatchScoreResult{
			OverallScore:   0,
			Recommendation: RecommendationNotRecommend,
			Summary:        "增强评估结果不合法，且 fallback 已禁用",
			Risks:          []candidateMatchSignal{{Code: "enhanced_invalid", Message: "语义评估结果不合法，fallback 已禁用"}},
		}
	}

	return candidateMatchScoreResult{
		OverallScore:       aggregated.OverallScore,
		Recommendation:     aggregated.Recommendation,
		Summary:            aggregated.Summary,
		Strengths:          aggregated.Strengths,
		Risks:              aggregated.Risks,
		ScoreBreakdownJSON: mustCandidateMatchJSON(aggregated.Breakdown),
		Evidence:           s.mapRequirementEvidenceToModel(matchResults, job.ID),
		Breakdown: candidateMatchBreakdown{
			ScorerVersion:       s.scorerVersion,
			InputHash:           inputHash,
			MissingRequirements: extractMissingRequirementIDs(matchResults, requirementProfile),
			Dimensions:          convertScoreDimensions(aggregated.Breakdown.Dimensions),
		},
	}
}

func (s *CandidateMatchService) score(job *model.Job, application *repository.ApplicationDetailRow, profile *model.CandidateProfile, resume *model.Resume, snapshot *repository.ResumeProfileSnapshot) candidateMatchScoreResult {
	jobText := strings.Join([]string{job.Title, job.Department, job.Location, job.Description, job.Requirements}, " ")
	resumeText := buildCandidateMatchResumeText(resume, snapshot, profile)
	jobTokens := uniqueSortedTokens(jobText)
	resumeTokens := tokenSet(resumeText)
	requirementTerms := requirementKeywords(job.Requirements)
	matchedRequirements, missingRequirements := splitMatches(requirementTerms, resumeTokens)

	skillRequired := candidateMatchSkillTerms(jobText)
	if len(skillRequired) == 0 {
		skillRequired = requirementTerms
	}
	skillTerms := resumeSkillNames(snapshot.Skills)
	skillSet := make(map[string]struct{}, len(skillTerms))
	for _, term := range skillTerms {
		for _, token := range uniqueSortedTokens(term) {
			skillSet[token] = struct{}{}
		}
	}
	matchedSkills, missingSkills := splitMatches(skillRequired, skillSet)

	experienceScore := experienceDimensionScore(jobTokens, snapshot.Profile.TotalExperience, resumeTokens)
	educationScore := educationDimensionScore(jobText, snapshot.Profile.HighestDegree, application.Education)
	profileScore := profileDimensionScore(profile)
	skillsScore := coverageScore(len(matchedSkills), len(skillRequired))
	requirementsScore := coverageScore(len(matchedRequirements), len(requirementTerms))

	dimensions := []candidateMatchDimension{
		{Name: "skills", Weight: 0.35, Score: skillsScore, Matched: matchedSkills, Missing: missingSkills},
		{Name: "requirements", Weight: 0.25, Score: requirementsScore, Matched: matchedRequirements, Missing: missingRequirements},
		{Name: "experience", Weight: 0.20, Score: experienceScore},
		{Name: "education", Weight: 0.10, Score: educationScore},
		{Name: "profile", Weight: 0.10, Score: profileScore},
	}

	overall := 0.0
	breakdown := candidateMatchBreakdown{
		ScorerVersion:       s.scorerVersion,
		InputHash:           candidateMatchInputHash(jobText, resumeText, s.scorerVersion),
		MissingRequirements: missingRequirements,
		Dimensions:          dimensions,
	}
	for _, dimension := range dimensions {
		overall += dimension.Score * dimension.Weight
	}
	overall = roundCandidateMatchScore(overall)

	risks := candidateMatchRisks(profile, resume, missingRequirements, missingSkills)
	strengths := candidateMatchStrengths(matchedSkills, matchedRequirements, snapshot)
	recommendation := candidateMatchRecommendation(overall, risks, missingRequirements)

	return candidateMatchScoreResult{
		OverallScore:   overall,
		Recommendation: recommendation,
		Summary:        candidateMatchSummary(overall, recommendation, matchedSkills, missingRequirements, risks),
		Strengths:      strengths,
		Risks:          risks,
		Breakdown:      breakdown,
		Evidence:       candidateMatchEvidence(job, profile, resume, snapshot, matchedSkills, matchedRequirements, missingRequirements, risks),
	}
}

type candidateMatchScoreResult struct {
	OverallScore       float64
	Recommendation     string
	Summary            string
	Strengths          []candidateMatchSignal
	Risks              []candidateMatchSignal
	Breakdown          candidateMatchBreakdown
	ScoreBreakdownJSON string
	Evidence           []model.CandidateMatchEvidence
}

type candidateMatchBreakdown struct {
	ScorerVersion       string                    `json:"scorer_version"`
	InputHash           string                    `json:"input_hash"`
	MissingRequirements []string                  `json:"missing_requirements"`
	Dimensions          []candidateMatchDimension `json:"dimensions"`
}

type candidateMatchDimension struct {
	Name    string   `json:"name"`
	Weight  float64  `json:"weight"`
	Score   float64  `json:"score"`
	Matched []string `json:"matched,omitempty"`
	Missing []string `json:"missing,omitempty"`
}

type candidateMatchSignal struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var candidateMatchTokenRE = regexp.MustCompile(`[a-z0-9+#.]+`)

func uniqueSortedTokens(text string) []string {
	matches := candidateMatchTokenRE.FindAllString(strings.ToLower(text), -1)
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		token := strings.Trim(match, ".")
		if len(token) < 2 || candidateMatchStopwords[token] {
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

func tokenSet(text string) map[string]struct{} {
	tokens := uniqueSortedTokens(text)
	set := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		set[token] = struct{}{}
	}
	return set
}

func requirementKeywords(requirements string) []string {
	tokens := uniqueSortedTokens(requirements)
	if len(tokens) > 16 {
		tokens = tokens[:16]
	}
	return tokens
}

func candidateMatchSkillTerms(text string) []string {
	tokens := uniqueSortedTokens(text)
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if candidateMatchKnownSkills[token] {
			out = append(out, token)
		}
	}
	return out
}

func splitMatches(required []string, actual map[string]struct{}) ([]string, []string) {
	matched := make([]string, 0, len(required))
	missing := make([]string, 0, len(required))
	for _, requirement := range required {
		if _, ok := actual[requirement]; ok {
			matched = append(matched, requirement)
		} else {
			missing = append(missing, requirement)
		}
	}
	return matched, missing
}

func resumeSkillNames(skills []model.ResumeSkill) []string {
	out := make([]string, 0, len(skills))
	for _, skill := range skills {
		if strings.TrimSpace(skill.Name) != "" {
			out = append(out, skill.Name)
		}
	}
	sort.Strings(out)
	return out
}

func buildCandidateMatchResumeText(resume *model.Resume, snapshot *repository.ResumeProfileSnapshot, profile *model.CandidateProfile) string {
	parts := []string{
		resume.ParsedText,
		snapshot.Profile.FullName,
		snapshot.Profile.Headline,
		snapshot.Profile.Summary,
		snapshot.Profile.HighestDegree,
	}
	if profile != nil {
		parts = append(parts, profile.RealName, profile.Education, profile.School, profile.WorkExperience, profile.Skills)
	}
	for _, education := range snapshot.Educations {
		parts = append(parts, education.School, education.Degree, education.Major, education.Description)
	}
	for _, experience := range snapshot.Experiences {
		parts = append(parts, experience.Company, experience.Title, experience.Description, experience.AchievementsJSON)
	}
	for _, project := range snapshot.Projects {
		parts = append(parts, project.Name, project.Role, project.Description, project.TechnologiesJSON, project.HighlightsJSON)
	}
	for _, skill := range snapshot.Skills {
		parts = append(parts, skill.Name, skill.Category, skill.Level, skill.Evidence)
	}
	return strings.Join(parts, " ")
}

func isIncompleteResumeProfile(snapshot *repository.ResumeProfileSnapshot) bool {
	if snapshot == nil || snapshot.Profile.ID == 0 {
		return true
	}
	return strings.TrimSpace(snapshot.Profile.FullName) == "" ||
		(len(snapshot.Educations) == 0 && len(snapshot.Experiences) == 0 && len(snapshot.Projects) == 0 && len(snapshot.Skills) == 0)
}

func coverageScore(matched, total int) float64 {
	if total == 0 {
		return 70
	}
	return roundCandidateMatchScore(float64(matched) / float64(total) * 100)
}

func experienceDimensionScore(jobTokens []string, years float64, resumeTokens map[string]struct{}) float64 {
	score := 55.0
	if years >= 5 {
		score += 25
	} else if years >= 3 {
		score += 18
	} else if years >= 1 {
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
	return roundCandidateMatchScore(math.Min(100, score))
}

func educationDimensionScore(jobText, highestDegree, applicationEducation string) float64 {
	text := strings.ToLower(strings.Join([]string{highestDegree, applicationEducation}, " "))
	job := strings.ToLower(jobText)
	if strings.Contains(job, "master") || strings.Contains(job, "硕士") {
		if strings.Contains(text, "master") || strings.Contains(text, "硕士") || strings.Contains(text, "phd") || strings.Contains(text, "doctor") || strings.Contains(text, "博士") {
			return 100
		}
		return 45
	}
	if strings.Contains(job, "bachelor") || strings.Contains(job, "本科") {
		if strings.Contains(text, "bachelor") || strings.Contains(text, "本科") || strings.Contains(text, "master") || strings.Contains(text, "硕士") || strings.Contains(text, "phd") || strings.Contains(text, "博士") {
			return 100
		}
		return 55
	}
	if strings.TrimSpace(text) == "" {
		return 60
	}
	return 80
}

func profileDimensionScore(profile *model.CandidateProfile) float64 {
	if profile == nil {
		return 35
	}
	score := 40.0
	fields := []string{profile.RealName, profile.Phone, profile.Education, profile.School, profile.WorkExperience, profile.Skills}
	for _, field := range fields {
		if strings.TrimSpace(field) != "" {
			score += 8
		}
	}
	if profile.IsComplete == 1 {
		score += 12
	}
	return roundCandidateMatchScore(math.Min(100, score))
}

func candidateMatchRisks(profile *model.CandidateProfile, resume *model.Resume, missingRequirements, missingSkills []string) []candidateMatchSignal {
	var risks []candidateMatchSignal
	if profile == nil {
		risks = append(risks, candidateMatchSignal{Code: "candidate_profile_missing", Message: "Candidate profile is missing."})
	} else if profile.IsComplete != 1 {
		risks = append(risks, candidateMatchSignal{Code: "candidate_profile_incomplete", Message: "Candidate profile is incomplete."})
	}
	if strings.TrimSpace(resume.ParsedText) == "" {
		risks = append(risks, candidateMatchSignal{Code: "resume_text_missing", Message: "Resume parsed text is empty, reducing evidence quality."})
	}
	if len(missingRequirements) > 0 {
		risks = append(risks, candidateMatchSignal{Code: "requirements_gap", Message: "Missing job requirement evidence: " + strings.Join(missingRequirements, ", ")})
	}
	if len(missingSkills) > 0 {
		risks = append(risks, candidateMatchSignal{Code: "skills_gap", Message: "Missing skill evidence: " + strings.Join(missingSkills, ", ")})
	}
	return risks
}

func candidateMatchStrengths(matchedSkills, matchedRequirements []string, snapshot *repository.ResumeProfileSnapshot) []candidateMatchSignal {
	var strengths []candidateMatchSignal
	if len(matchedSkills) > 0 {
		strengths = append(strengths, candidateMatchSignal{Code: "matched_skills", Message: "Matched skills: " + strings.Join(matchedSkills, ", ")})
	}
	if len(matchedRequirements) > 0 {
		strengths = append(strengths, candidateMatchSignal{Code: "matched_requirements", Message: "Matched requirements: " + strings.Join(matchedRequirements, ", ")})
	}
	if snapshot.Profile.TotalExperience >= 3 {
		strengths = append(strengths, candidateMatchSignal{Code: "experience_depth", Message: fmt.Sprintf("Resume profile reports %.1f years of experience.", snapshot.Profile.TotalExperience)})
	}
	return strengths
}

func candidateMatchRecommendation(score float64, risks []candidateMatchSignal, missingRequirements []string) string {
	if score >= 82 && len(missingRequirements) <= 1 {
		return "strong_match"
	}
	if score >= 65 && len(risks) <= 3 {
		return "possible_match"
	}
	if score >= 50 {
		return "needs_review"
	}
	return "not_recommended"
}

func candidateMatchSummary(score float64, recommendation string, matchedSkills, missingRequirements []string, risks []candidateMatchSignal) string {
	parts := []string{
		fmt.Sprintf("Overall score %.1f with recommendation %s.", score, recommendation),
	}
	if len(matchedSkills) > 0 {
		parts = append(parts, "Matched skills: "+strings.Join(matchedSkills, ", ")+".")
	}
	if len(missingRequirements) > 0 {
		parts = append(parts, "Missing requirements: "+strings.Join(missingRequirements, ", ")+".")
	}
	if len(risks) > 0 {
		parts = append(parts, fmt.Sprintf("%d risk signal(s) require review.", len(risks)))
	}
	return strings.Join(parts, " ")
}

func candidateMatchEvidence(job *model.Job, profile *model.CandidateProfile, resume *model.Resume, snapshot *repository.ResumeProfileSnapshot, matchedSkills, matchedRequirements, missingRequirements []string, risks []candidateMatchSignal) []model.CandidateMatchEvidence {
	var evidence []model.CandidateMatchEvidence
	for _, skill := range snapshot.Skills {
		skillTokens := tokenSet(skill.Name)
		for _, matched := range matchedSkills {
			if _, ok := skillTokens[matched]; ok {
				sourceID := skill.ID
				evidence = append(evidence, model.CandidateMatchEvidence{
					EvidenceType: "skill",
					Dimension:    "skills",
					SourceTable:  "resume_skills",
					SourceID:     &sourceID,
					Snippet:      skill.Name,
					Weight:       0.35,
					ScoreImpact:  100,
				})
			}
		}
	}
	for _, experience := range snapshot.Experiences {
		text := strings.Join([]string{experience.Title, experience.Description, experience.AchievementsJSON}, " ")
		tokens := tokenSet(text)
		for _, requirement := range matchedRequirements {
			if _, ok := tokens[requirement]; ok {
				sourceID := experience.ID
				evidence = append(evidence, model.CandidateMatchEvidence{
					EvidenceType: "experience",
					Dimension:    "requirements",
					SourceTable:  "resume_experiences",
					SourceID:     &sourceID,
					Snippet:      truncateCandidateMatchSnippet(text),
					Weight:       0.25,
					ScoreImpact:  75,
				})
			}
		}
	}
	for _, requirement := range matchedRequirements {
		if strings.Contains(strings.ToLower(resume.ParsedText), requirement) {
			sourceID := uint64(resume.ID)
			evidence = append(evidence, model.CandidateMatchEvidence{
				EvidenceType: "resume_text",
				Dimension:    "requirements",
				SourceTable:  "resumes",
				SourceID:     &sourceID,
				Snippet:      truncateCandidateMatchSnippet(resume.ParsedText),
				Weight:       0.10,
				ScoreImpact:  50,
			})
			break
		}
	}
	if len(missingRequirements) > 0 {
		sourceID := uint64(job.ID)
		evidence = append(evidence, model.CandidateMatchEvidence{
			EvidenceType: "missing_requirement",
			Dimension:    "requirements",
			SourceTable:  "jobs",
			SourceID:     &sourceID,
			Snippet:      strings.Join(missingRequirements, ", "),
			Weight:       0.25,
			ScoreImpact:  -25,
		})
	}
	if profile != nil {
		sourceID := uint64(profile.ID)
		evidence = append(evidence, model.CandidateMatchEvidence{
			EvidenceType: "candidate_profile",
			Dimension:    "profile",
			SourceTable:  "candidate_profiles",
			SourceID:     &sourceID,
			Snippet:      truncateCandidateMatchSnippet(strings.Join([]string{profile.RealName, profile.Education, profile.School, profile.Skills}, " ")),
			Weight:       0.10,
			ScoreImpact:  profileDimensionScore(profile),
		})
	}
	for _, risk := range risks {
		evidence = append(evidence, model.CandidateMatchEvidence{
			EvidenceType: "risk",
			Dimension:    "risk",
			SourceTable:  "resume_profiles",
			SourceID:     &snapshot.Profile.ID,
			Snippet:      risk.Message,
			Weight:       0,
			ScoreImpact:  -10,
		})
	}
	sort.SliceStable(evidence, func(i, j int) bool {
		left := evidence[i]
		right := evidence[j]
		if left.EvidenceType != right.EvidenceType {
			return left.EvidenceType < right.EvidenceType
		}
		if left.SourceTable != right.SourceTable {
			return left.SourceTable < right.SourceTable
		}
		return left.Snippet < right.Snippet
	})
	return evidence
}

func candidateMatchInputHash(jobText, resumeText, scorerVersion string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{jobText, resumeText, scorerVersion}, "\n")))
	return hex.EncodeToString(sum[:])
}

func roundCandidateMatchScore(score float64) float64 {
	return math.Round(score*10) / 10
}

func truncateCandidateMatchSnippet(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= 240 {
		return text
	}
	return text[:240]
}

func mustCandidateMatchJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func extractMissingRequirementIDs(results []RequirementMatchResult, profile *JobRequirementProfile) []string {
	labelsByID := make(map[string]string)
	if profile != nil {
		for _, req := range profile.Requirements {
			labelsByID[req.ID] = req.Label
		}
	}
	var missing []string
	for _, result := range results {
		if result.Status == MatchStatusMissing || result.Status == MatchStatusConflict {
			if label := labelsByID[result.RequirementID]; label != "" {
				missing = append(missing, label)
			} else {
				missing = append(missing, result.RequirementID)
			}
		}
	}
	sort.Strings(missing)
	return missing
}

func convertScoreDimensions(dimensions []ScoreDimension) []candidateMatchDimension {
	out := make([]candidateMatchDimension, 0, len(dimensions))
	for _, dimension := range dimensions {
		out = append(out, candidateMatchDimension{
			Name:    dimension.Name,
			Weight:  dimension.Weight,
			Score:   dimension.Score,
			Matched: dimension.Matched,
			Missing: dimension.Missing,
		})
	}
	return out
}

func (s *CandidateMatchService) mapRequirementEvidenceToModel(results []RequirementMatchResult, jobID int64) []model.CandidateMatchEvidence {
	var evidence []model.CandidateMatchEvidence
	seen := make(map[string]bool)
	for _, result := range results {
		for _, ev := range result.Evidence {
			key := result.RequirementID + ":" + ev.SourceTable + ":" + ev.Snippet
			if seen[key] {
				continue
			}
			seen[key] = true
			sourceID := ev.SourceID
			evidence = append(evidence, model.CandidateMatchEvidence{
				EvidenceType: "requirement_match",
				Dimension:    result.RequirementID,
				SourceTable:  ev.SourceTable,
				SourceID:     &sourceID,
				Snippet:      ev.Snippet,
				Weight:       0,
				ScoreImpact:  result.Score,
				MetadataJSON: mustCandidateMatchJSON(map[string]any{
					"requirement_id": result.RequirementID,
					"reason":         ev.Reason,
				}),
			})
		}
	}
	if len(evidence) == 0 {
		sourceID := uint64(jobID)
		evidence = append(evidence, model.CandidateMatchEvidence{
			EvidenceType: "no_match",
			Dimension:    "overall",
			SourceTable:  "jobs",
			SourceID:     &sourceID,
			Snippet:      "No requirement-level evidence found",
			MetadataJSON: "{}",
		})
	}
	return evidence
}

func runShadowIfEnabled(ctx context.Context, policy *AgentRuntimePolicy, s *CandidateMatchService, job *model.Job, application *repository.ApplicationDetailRow, profile *model.CandidateProfile, resume *model.Resume, resumeSnapshot *repository.ResumeProfileSnapshot, primaryResult candidateMatchScoreResult) {
	if !policy.CandidateMatchShadow {
		return
	}
	go func() {
		shadowCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		started := time.Now()
		shadowType := "enhanced"
		var shadowResult candidateMatchScoreResult
		if policy.CandidateMatchSemantic {
			shadowType = "legacy"
			shadowResult = s.score(job, application, profile, resume, resumeSnapshot)
		} else {
			shadowResult = s.scoreEnhanced(shadowCtx, job, application, profile, resume, resumeSnapshot)
		}
		recommendationDelta := ""
		if primaryResult.Recommendation != shadowResult.Recommendation {
			recommendationDelta = primaryResult.Recommendation + "→" + shadowResult.Recommendation
		}
		logger.L().Info("[domain][candidate_match] shadow mode comparison",
			zap.String("shadow_type", shadowType),
			zap.Int64("shadow_duration_ms", time.Since(started).Milliseconds()),
			zap.Float64("primary_score", primaryResult.OverallScore),
			zap.Float64("shadow_score", shadowResult.OverallScore),
			zap.Float64("score_delta", shadowResult.OverallScore-primaryResult.OverallScore),
			zap.String("recommendation_delta", recommendationDelta),
			zap.Int("primary_evidence", len(primaryResult.Evidence)),
			zap.Int("shadow_evidence", len(shadowResult.Evidence)))
	}()
}

var candidateMatchStopwords = map[string]bool{
	"and": true, "are": true, "but": true, "for": true, "have": true, "with": true,
	"the": true, "this": true, "that": true, "you": true, "your": true, "our": true,
	"will": true, "can": true, "able": true, "using": true, "use": true, "must": true,
	"plus": true, "nice": true, "good": true, "strong": true, "experience": true,
	"years": true, "year": true, "work": true, "candidate": true, "role": true,
	"required": true, "requirement": true, "requirements": true, "knowledge": true,
	"熟悉": true, "经验": true, "能力": true,
}

var candidateMatchKnownSkills = map[string]bool{
	"aws": true, "azure": true, "c": true, "c++": true, "c#": true, "css": true,
	"docker": true, "elasticsearch": true, "gin": true, "git": true, "go": true,
	"golang": true, "grpc": true, "html": true, "java": true, "javascript": true,
	"kafka": true, "kubernetes": true, "linux": true, "mysql": true, "node": true,
	"postgres": true, "postgresql": true, "python": true, "rabbitmq": true,
	"react": true, "redis": true, "sql": true, "typescript": true, "vue": true,
}
