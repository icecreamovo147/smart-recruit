package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

const defaultResumeProfileParserVersion = "resume-profile-parser-v1"

var (
	ErrResumeProfileMissingResume = errors.New("resume not found")
	ErrResumeProfileMissingText   = errors.New("resume parsed_text is empty")
	ErrResumeProfileInvalidOutput = errors.New("resume profile extractor output is invalid")
)

type ResumeProfileExtractor interface {
	Extract(ctx context.Context, text string) (string, error)
}

type ResumeProfileService struct {
	resumes       *repository.ResumeRepo
	profiles      *repository.ResumeProfileRepo
	extractor     ResumeProfileExtractor
	parserVersion string
	now           func() time.Time
	policy        AgentRuntimePolicy
}

func NewResumeProfileService(resumes *repository.ResumeRepo, profiles *repository.ResumeProfileRepo, extractor ResumeProfileExtractor) *ResumeProfileService {
	return &ResumeProfileService{
		resumes:       resumes,
		profiles:      profiles,
		extractor:     extractor,
		parserVersion: defaultResumeProfileParserVersion,
		now:           time.Now,
		policy:        DefaultAgentRuntimePolicy(),
	}
}

func (s *ResumeProfileService) WithRuntimePolicy(policy AgentRuntimePolicy) *ResumeProfileService {
	if s != nil {
		s.policy = policy.withDefaults()
	}
	return s
}

func (s *ResumeProfileService) ParseResume(ctx context.Context, resumeID int64) (*repository.ResumeProfileSnapshot, error) {
	if s == nil || s.resumes == nil || s.profiles == nil || s.extractor == nil {
		return nil, fmt.Errorf("resume profile service is not configured")
	}
	policy := s.policy.withDefaults()
	if !policy.StructuredResumeParse {
		logger.L().Warn("agent capability disabled",
			zap.String("capability", "structured_resume_parse"),
			zap.Int64("resume_id", resumeID))
		return nil, fmt.Errorf("%w: structured_resume_parse", ErrAgentCapabilityDisabled)
	}
	started := time.Now()
	ctx, cancel := contextWithPolicyTimeout(ctx, policy.ResumeParseTimeout)
	defer cancel()

	resume, err := s.resumes.GetByID(ctx, resumeID)
	if err != nil {
		s.logParseFinished(resumeID, "", started, "failed", err)
		return nil, fmt.Errorf("get resume: %w", err)
	}
	if resume == nil {
		s.logParseFinished(resumeID, "", started, "failed", ErrResumeProfileMissingResume)
		return nil, ErrResumeProfileMissingResume
	}

	text := strings.TrimSpace(resume.ParsedText)
	inputHash := hashResumeProfileInput(text)
	if text == "" {
		s.logParseFinished(resumeID, inputHash, started, "failed", ErrResumeProfileMissingText)
		return nil, s.saveFailure(ctx, resume, inputHash, ErrResumeProfileMissingText)
	}

	raw, err := s.extractor.Extract(ctx, text)
	if err != nil {
		s.logParseFinished(resumeID, inputHash, started, "failed", err)
		return nil, s.saveFailure(ctx, resume, inputHash, fmt.Errorf("extract resume profile: %w", err))
	}

	extracted, err := decodeExtractedResumeProfile(raw)
	if err != nil {
		s.logParseFinished(resumeID, inputHash, started, "failed", err)
		return nil, s.saveFailure(ctx, resume, inputHash, err)
	}

	snapshot, err := s.buildSnapshot(resume, inputHash, raw, extracted)
	if err != nil {
		s.logParseFinished(resumeID, inputHash, started, "failed", err)
		return nil, s.saveFailure(ctx, resume, inputHash, err)
	}
	if err := s.profiles.SaveProfileVersion(ctx, snapshot); err != nil {
		s.logParseFinished(resumeID, inputHash, started, "failed", err)
		return nil, fmt.Errorf("save resume profile version: %w", err)
	}
	s.logParseFinished(resumeID, inputHash, started, "succeeded", nil)
	return snapshot, nil
}

func (s *ResumeProfileService) logParseFinished(resumeID int64, inputHash string, started time.Time, status string, err error) {
	fields := []zap.Field{
		zap.String("event", "agent.resume_profile.parse"),
		zap.Int64("resume_id", resumeID),
		zap.String("input_hash", inputHash),
		zap.String("status", status),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.L().Warn("resume profile parse finished", fields...)
		return
	}
	logger.L().Info("resume profile parse finished", fields...)
}

func (s *ResumeProfileService) saveFailure(ctx context.Context, resume *model.Resume, inputHash string, cause error) error {
	now := s.currentTime()
	message := strings.TrimSpace(cause.Error())
	if len(message) > 4000 {
		message = message[:4000]
	}
	snapshot := &repository.ResumeProfileSnapshot{
		ParseRun: model.ResumeParseRun{
			ResumeID:      resume.ID,
			UserID:        resume.UserID,
			Status:        "failed",
			ParserVersion: s.version(),
			InputHash:     inputHash,
			ErrorMessage:  message,
			StartedAt:     now,
			CompletedAt:   &now,
		},
		Profile: model.ResumeProfile{},
	}
	if err := s.profiles.SaveProfileVersion(ctx, snapshot); err != nil {
		return fmt.Errorf("%w; additionally failed to save parse failure: %v", cause, err)
	}
	return cause
}

func (s *ResumeProfileService) buildSnapshot(resume *model.Resume, inputHash, raw string, extracted extractedResumeProfile) (*repository.ResumeProfileSnapshot, error) {
	normalized, err := normalizeExtractedResumeProfile(extracted)
	if err != nil {
		return nil, err
	}
	now := s.currentTime()
	return &repository.ResumeProfileSnapshot{
		ParseRun: model.ResumeParseRun{
			ResumeID:      resume.ID,
			UserID:        resume.UserID,
			Status:        "succeeded",
			ParserVersion: s.version(),
			InputHash:     inputHash,
			StartedAt:     now,
			CompletedAt:   &now,
		},
		Profile: model.ResumeProfile{
			FullName:        normalized.Profile.FullName,
			Email:           normalized.Profile.Email,
			Phone:           normalized.Profile.Phone,
			Location:        normalized.Profile.Location,
			Headline:        normalized.Profile.Headline,
			Summary:         normalized.Profile.Summary,
			TotalExperience: normalized.Profile.TotalExperience,
			HighestDegree:   normalized.Profile.HighestDegree,
			RawJSON:         raw,
		},
		Educations:  normalized.Educations,
		Experiences: normalized.Experiences,
		Projects:    normalized.Projects,
		Skills:      normalized.Skills,
	}, nil
}

func (s *ResumeProfileService) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *ResumeProfileService) version() string {
	if strings.TrimSpace(s.parserVersion) == "" {
		return defaultResumeProfileParserVersion
	}
	return strings.TrimSpace(s.parserVersion)
}

type extractedResumeProfile struct {
	FullName             string                      `json:"full_name"`
	Email                string                      `json:"email"`
	Phone                string                      `json:"phone"`
	Location             string                      `json:"location"`
	Headline             string                      `json:"headline"`
	Summary              string                      `json:"summary"`
	TotalExperienceYears *float64                    `json:"total_experience_years"`
	HighestDegree        string                      `json:"highest_degree"`
	Educations           []extractedResumeEducation  `json:"educations"`
	Experiences          []extractedResumeExperience `json:"experiences"`
	Projects             []extractedResumeProject    `json:"projects"`
	Skills               []extractedResumeSkill      `json:"skills"`
}

type extractedResumeEducation struct {
	School      string `json:"school"`
	Degree      string `json:"degree"`
	Major       string `json:"major"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Description string `json:"description"`
}

type extractedResumeExperience struct {
	Company      string   `json:"company"`
	Title        string   `json:"title"`
	Location     string   `json:"location"`
	StartDate    string   `json:"start_date"`
	EndDate      string   `json:"end_date"`
	IsCurrent    bool     `json:"is_current"`
	Description  string   `json:"description"`
	Achievements []string `json:"achievements"`
}

type extractedResumeProject struct {
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	StartDate    string   `json:"start_date"`
	EndDate      string   `json:"end_date"`
	Description  string   `json:"description"`
	Technologies []string `json:"technologies"`
	Highlights   []string `json:"highlights"`
}

type extractedResumeSkill struct {
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Level    string  `json:"level"`
	Years    float64 `json:"years"`
	Evidence string  `json:"evidence"`
}

type normalizedResumeProfile struct {
	Profile     model.ResumeProfile
	Educations  []model.ResumeEducation
	Experiences []model.ResumeExperience
	Projects    []model.ResumeProject
	Skills      []model.ResumeSkill
}

func decodeExtractedResumeProfile(raw string) (extractedResumeProfile, error) {
	var extracted extractedResumeProfile
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&extracted); err != nil {
		return extracted, fmt.Errorf("%w: %v", ErrResumeProfileInvalidOutput, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return extracted, fmt.Errorf("%w: multiple JSON values", ErrResumeProfileInvalidOutput)
	}
	return extracted, nil
}

func normalizeExtractedResumeProfile(extracted extractedResumeProfile) (normalizedResumeProfile, error) {
	if strings.TrimSpace(extracted.FullName) == "" {
		return normalizedResumeProfile{}, fmt.Errorf("%w: full_name is required", ErrResumeProfileInvalidOutput)
	}
	if extracted.TotalExperienceYears == nil {
		return normalizedResumeProfile{}, fmt.Errorf("%w: total_experience_years is required", ErrResumeProfileInvalidOutput)
	}
	if *extracted.TotalExperienceYears < 0 {
		return normalizedResumeProfile{}, fmt.Errorf("%w: total_experience_years cannot be negative", ErrResumeProfileInvalidOutput)
	}
	email := normalizeWhitespace(extracted.Email)
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			return normalizedResumeProfile{}, fmt.Errorf("%w: email is invalid", ErrResumeProfileInvalidOutput)
		}
	}

	normalized := normalizedResumeProfile{
		Profile: model.ResumeProfile{
			FullName:        normalizeName(extracted.FullName),
			Email:           strings.ToLower(email),
			Phone:           normalizeWhitespace(extracted.Phone),
			Location:        normalizeWhitespace(extracted.Location),
			Headline:        normalizeWhitespace(extracted.Headline),
			Summary:         normalizeWhitespace(extracted.Summary),
			TotalExperience: roundYears(*extracted.TotalExperienceYears),
			HighestDegree:   normalizeWhitespace(extracted.HighestDegree),
		},
	}
	if normalized.Profile.FullName == "" {
		return normalizedResumeProfile{}, fmt.Errorf("%w: full_name is required", ErrResumeProfileInvalidOutput)
	}

	for i, education := range extracted.Educations {
		school := normalizeOrganization(education.School)
		if school == "" {
			return normalizedResumeProfile{}, fmt.Errorf("%w: educations[%d].school is required", ErrResumeProfileInvalidOutput, i)
		}
		start, err := parseResumeDate(education.StartDate)
		if err != nil {
			return normalizedResumeProfile{}, fmt.Errorf("%w: educations[%d].start_date: %v", ErrResumeProfileInvalidOutput, i, err)
		}
		end, err := parseResumeDate(education.EndDate)
		if err != nil {
			return normalizedResumeProfile{}, fmt.Errorf("%w: educations[%d].end_date: %v", ErrResumeProfileInvalidOutput, i, err)
		}
		normalized.Educations = append(normalized.Educations, model.ResumeEducation{
			School:      school,
			Degree:      normalizeWhitespace(education.Degree),
			Major:       normalizeWhitespace(education.Major),
			StartDate:   start,
			EndDate:     end,
			Description: normalizeWhitespace(education.Description),
			SortOrder:   int32(i + 1),
		})
	}

	for i, experience := range extracted.Experiences {
		company := normalizeOrganization(experience.Company)
		if company == "" {
			return normalizedResumeProfile{}, fmt.Errorf("%w: experiences[%d].company is required", ErrResumeProfileInvalidOutput, i)
		}
		start, err := parseResumeDate(experience.StartDate)
		if err != nil {
			return normalizedResumeProfile{}, fmt.Errorf("%w: experiences[%d].start_date: %v", ErrResumeProfileInvalidOutput, i, err)
		}
		end, err := parseResumeDate(experience.EndDate)
		if err != nil {
			return normalizedResumeProfile{}, fmt.Errorf("%w: experiences[%d].end_date: %v", ErrResumeProfileInvalidOutput, i, err)
		}
		achievements, err := marshalNormalizedStrings(experience.Achievements)
		if err != nil {
			return normalizedResumeProfile{}, fmt.Errorf("%w: experiences[%d].achievements: %v", ErrResumeProfileInvalidOutput, i, err)
		}
		var isCurrent int32
		if experience.IsCurrent || strings.EqualFold(strings.TrimSpace(experience.EndDate), "present") || strings.EqualFold(strings.TrimSpace(experience.EndDate), "current") {
			isCurrent = 1
			end = nil
		}
		normalized.Experiences = append(normalized.Experiences, model.ResumeExperience{
			Company:          company,
			Title:            normalizeWhitespace(experience.Title),
			Location:         normalizeWhitespace(experience.Location),
			StartDate:        start,
			EndDate:          end,
			IsCurrent:        isCurrent,
			Description:      normalizeWhitespace(experience.Description),
			AchievementsJSON: achievements,
			SortOrder:        int32(i + 1),
		})
	}

	for i, project := range extracted.Projects {
		name := normalizeWhitespace(project.Name)
		if name == "" {
			return normalizedResumeProfile{}, fmt.Errorf("%w: projects[%d].name is required", ErrResumeProfileInvalidOutput, i)
		}
		start, err := parseResumeDate(project.StartDate)
		if err != nil {
			return normalizedResumeProfile{}, fmt.Errorf("%w: projects[%d].start_date: %v", ErrResumeProfileInvalidOutput, i, err)
		}
		end, err := parseResumeDate(project.EndDate)
		if err != nil {
			return normalizedResumeProfile{}, fmt.Errorf("%w: projects[%d].end_date: %v", ErrResumeProfileInvalidOutput, i, err)
		}
		technologies, err := marshalNormalizedStrings(project.Technologies)
		if err != nil {
			return normalizedResumeProfile{}, fmt.Errorf("%w: projects[%d].technologies: %v", ErrResumeProfileInvalidOutput, i, err)
		}
		highlights, err := marshalNormalizedStrings(project.Highlights)
		if err != nil {
			return normalizedResumeProfile{}, fmt.Errorf("%w: projects[%d].highlights: %v", ErrResumeProfileInvalidOutput, i, err)
		}
		normalized.Projects = append(normalized.Projects, model.ResumeProject{
			Name:             name,
			Role:             normalizeWhitespace(project.Role),
			StartDate:        start,
			EndDate:          end,
			Description:      normalizeWhitespace(project.Description),
			TechnologiesJSON: technologies,
			HighlightsJSON:   highlights,
			SortOrder:        int32(i + 1),
		})
	}

	normalized.Skills = normalizeResumeSkills(extracted.Skills)
	if len(normalized.Educations) == 0 && len(normalized.Experiences) == 0 && len(normalized.Projects) == 0 && len(normalized.Skills) == 0 {
		return normalizedResumeProfile{}, fmt.Errorf("%w: profile must include at least one education, experience, project, or skill", ErrResumeProfileInvalidOutput)
	}
	return normalized, nil
}

func normalizeResumeSkills(skills []extractedResumeSkill) []model.ResumeSkill {
	byName := make(map[string]model.ResumeSkill)
	for _, skill := range skills {
		name := normalizeSkillName(skill.Name)
		if name == "" {
			continue
		}
		existing, found := byName[strings.ToLower(name)]
		candidate := model.ResumeSkill{
			Name:      name,
			Category:  strings.ToLower(normalizeWhitespace(skill.Category)),
			Level:     strings.ToLower(normalizeWhitespace(skill.Level)),
			Years:     roundYears(skill.Years),
			Evidence:  normalizeWhitespace(skill.Evidence),
			SortOrder: int32(len(byName) + 1),
		}
		if candidate.Years < 0 {
			candidate.Years = 0
		}
		if !found {
			byName[strings.ToLower(name)] = candidate
			continue
		}
		if candidate.Category != "" {
			existing.Category = candidate.Category
		}
		if candidate.Level != "" {
			existing.Level = candidate.Level
		}
		if candidate.Years > existing.Years {
			existing.Years = candidate.Years
		}
		if candidate.Evidence != "" && !strings.Contains(existing.Evidence, candidate.Evidence) {
			if existing.Evidence != "" {
				existing.Evidence += "; "
			}
			existing.Evidence += candidate.Evidence
		}
		byName[strings.ToLower(name)] = existing
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	normalized := make([]model.ResumeSkill, 0, len(names))
	for i, key := range names {
		skill := byName[key]
		skill.SortOrder = int32(i + 1)
		normalized = append(normalized, skill)
	}
	return normalized
}

func normalizeName(value string) string {
	return normalizeWhitespace(value)
}

func normalizeOrganization(value string) string {
	return normalizeWhitespace(value)
}

func normalizeSkillName(value string) string {
	value = normalizeWhitespace(value)
	switch strings.ToLower(value) {
	case "golang":
		return "Go"
	case "nodejs", "node.js":
		return "Node.js"
	case "javascript":
		return "JavaScript"
	case "typescript":
		return "TypeScript"
	case "postgresql":
		return "PostgreSQL"
	case "mysql":
		return "MySQL"
	case "grpc":
		return "gRPC"
	}
	return value
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func parseResumeDate(value string) (*time.Time, error) {
	value = normalizeWhitespace(value)
	if value == "" || strings.EqualFold(value, "present") || strings.EqualFold(value, "current") {
		return nil, nil
	}
	layouts := []string{"2006-01-02", "2006-01", "2006"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("expected YYYY, YYYY-MM, or YYYY-MM-DD")
}

func marshalNormalizedStrings(values []string) (string, error) {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = normalizeWhitespace(value)
		if value != "" {
			normalized = append(normalized, value)
		}
	}
	if len(normalized) == 0 {
		return "[]", nil
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func hashResumeProfileInput(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func roundYears(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
