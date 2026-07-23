package recruiting_intelligence

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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	commonsai "smart-recruit-commons/ai"
)

const (
	ResumeLLMParserVersion       = "resume-profile-llm-v1"
	ResumeHeuristicParserVersion = "resume-profile-heuristic-v1"
)

var (
	ErrResumeEmptyResponse = errors.New("resume extractor returned an empty response")
	ErrResumeJSON          = errors.New("resume extractor returned invalid JSON")
	ErrResumeSchema        = errors.New("resume extractor returned an invalid schema")
	ErrResumeFallback      = errors.New("resume heuristic fallback failed")
)

type ResumeExtractionErrorKind string

const (
	ResumeExtractionPrompt   ResumeExtractionErrorKind = "prompt"
	ResumeExtractionProvider ResumeExtractionErrorKind = "provider"
	ResumeExtractionTimeout  ResumeExtractionErrorKind = "timeout"
	ResumeExtractionEmpty    ResumeExtractionErrorKind = "empty_response"
	ResumeExtractionJSON     ResumeExtractionErrorKind = "json"
	ResumeExtractionSchema   ResumeExtractionErrorKind = "schema"
	ResumeExtractionPolicy   ResumeExtractionErrorKind = "policy"
	ResumeExtractionFallback ResumeExtractionErrorKind = "fallback"
)

// ResumeExtractionError is deliberately safe for RPC responses and logs. It
// never includes the resume, prompt, or model output.
type ResumeExtractionError struct {
	Kind  ResumeExtractionErrorKind
	Cause error
}

func (e *ResumeExtractionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("resume profile extraction failed (%s): %v", e.Kind, e.Cause)
	}
	return fmt.Sprintf("resume profile extraction failed (%s)", e.Kind)
}

func (e *ResumeExtractionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type ResumeSource struct {
	ResumeID   int64
	UserID     int64
	FileName   string
	ParsedText string
}

type ResumeProfile struct {
	FullName             string
	Email                string
	Phone                string
	Location             string
	Headline             string
	Summary              string
	TotalExperienceYears float64
	HighestDegree        string
	Educations           []ResumeEducation
	Experiences          []ResumeExperience
	Projects             []ResumeProject
	Skills               []ResumeSkill
}

type ResumeEducation struct {
	School      string
	Degree      string
	Major       string
	StartDate   *time.Time
	EndDate     *time.Time
	Description string
}

type ResumeExperience struct {
	Company      string
	Title        string
	Location     string
	StartDate    *time.Time
	EndDate      *time.Time
	IsCurrent    bool
	Description  string
	Achievements []string
}

type ResumeProject struct {
	Name         string
	Role         string
	StartDate    *time.Time
	EndDate      *time.Time
	Description  string
	Technologies []string
	Highlights   []string
}

type ResumeSkill struct {
	Name     string
	Category string
	Level    string
	Years    float64
	Evidence string
}

type ResumeExtractionResult struct {
	Profile       ResumeProfile
	RawJSON       string
	InputHash     string
	ParserVersion string
	ExtractorType string
	ModelName     string
	Prompt        PromptDescriptor
	FallbackUsed  bool
	FallbackKind  ResumeExtractionErrorKind
}

type ResumeProfileExtractor struct {
	runtime *Runtime
	policy  RuntimePolicy
}

func NewResumeProfileExtractor(runtime *Runtime, policy RuntimePolicy) *ResumeProfileExtractor {
	return &ResumeProfileExtractor{runtime: runtime, policy: policy.effective()}
}

func (e *ResumeProfileExtractor) Extract(ctx context.Context, source ResumeSource) (ResumeExtractionResult, error) {
	started := time.Now()
	if e == nil || !e.policy.StructuredResumeParseEnabled() {
		return ResumeExtractionResult{}, &ResumeExtractionError{Kind: ResumeExtractionPolicy, Cause: ErrCapabilityDisabled}
	}
	primary, err := e.extractPrimary(ctx, source)
	if err == nil {
		observeRuntimeOutcome(e.runtime, ctx, "resume_profile_extraction", AgentTypeResumeProfileExtractor, "success", "none", started)
		return primary, nil
	}
	kind := classifyResumeExtractionError(err)
	if kind == ResumeExtractionPolicy || !e.policy.FallbacksEnabled() {
		observeRuntimeOutcome(e.runtime, ctx, "resume_profile_extraction", AgentTypeResumeProfileExtractor, "error", "disabled", started)
		return ResumeExtractionResult{}, err
	}
	fallback, fallbackErr := extractResumeHeuristically(source)
	if fallbackErr != nil {
		observeRuntimeOutcome(e.runtime, ctx, "resume_profile_extraction", AgentTypeResumeProfileExtractor, "error", string(kind), started)
		return ResumeExtractionResult{}, &ResumeExtractionError{Kind: ResumeExtractionFallback, Cause: errors.Join(err, ErrResumeFallback, fallbackErr)}
	}
	fallback.FallbackUsed = true
	fallback.FallbackKind = kind
	observeRuntimeOutcome(e.runtime, ctx, "resume_profile_extraction", AgentTypeResumeProfileExtractor, "success", string(kind), started)
	return fallback, nil
}

func (e *ResumeProfileExtractor) extractPrimary(ctx context.Context, source ResumeSource) (ResumeExtractionResult, error) {
	if e.runtime == nil {
		return ResumeExtractionResult{}, &ResumeExtractionError{Kind: ResumeExtractionProvider, Cause: ErrProvider}
	}
	completion, err := e.runtime.Complete(ctx, AgentTypeResumeProfileExtractor, resumeProfileUserMessage(source))
	if err != nil {
		return ResumeExtractionResult{}, &ResumeExtractionError{Kind: classifyResumeExtractionError(err), Cause: err}
	}
	if strings.TrimSpace(completion.Content) == "" {
		return ResumeExtractionResult{}, &ResumeExtractionError{Kind: ResumeExtractionEmpty, Cause: ErrResumeEmptyResponse}
	}
	profile, raw, err := decodeResumeProfile(completion.Content)
	if err != nil {
		return ResumeExtractionResult{}, err
	}
	return ResumeExtractionResult{
		Profile:       profile,
		RawJSON:       raw,
		InputHash:     resumeInputHash(source.ParsedText),
		ParserVersion: ResumeLLMParserVersion,
		ExtractorType: "llm",
		ModelName:     completion.ModelName,
		Prompt:        completion.Prompt,
	}, nil
}

func resumeProfileUserMessage(source ResumeSource) string {
	metadata, _ := json.Marshal(map[string]any{
		"resume_id":   source.ResumeID,
		"user_id":     source.UserID,
		"file_name":   source.FileName,
		"text_length": len([]rune(source.ParsedText)),
	})
	return "Resume metadata:\n" + string(metadata) + "\nResume parsed text:\n" + strings.TrimSpace(source.ParsedText)
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

var (
	resumeProfileKeys    = []string{"full_name", "email", "phone", "location", "headline", "summary", "total_experience_years", "highest_degree", "educations", "experiences", "projects", "skills"}
	resumeEducationKeys  = []string{"school", "degree", "major", "start_date", "end_date", "description"}
	resumeExperienceKeys = []string{"company", "title", "location", "start_date", "end_date", "is_current", "description", "achievements"}
	resumeProjectKeys    = []string{"name", "role", "start_date", "end_date", "description", "technologies", "highlights"}
	resumeSkillKeys      = []string{"name", "category", "level", "years", "evidence"}
)

func (value *extractedResumeProfile) UnmarshalJSON(data []byte) error {
	type plain extractedResumeProfile
	if err := decodeRequiredResumeObject(data, resumeProfileKeys, (*plain)(value)); err != nil {
		return err
	}
	return nil
}

func (value *extractedResumeEducation) UnmarshalJSON(data []byte) error {
	type plain extractedResumeEducation
	return decodeRequiredResumeObject(data, resumeEducationKeys, (*plain)(value))
}

func (value *extractedResumeExperience) UnmarshalJSON(data []byte) error {
	type plain extractedResumeExperience
	return decodeRequiredResumeObject(data, resumeExperienceKeys, (*plain)(value))
}

func (value *extractedResumeProject) UnmarshalJSON(data []byte) error {
	type plain extractedResumeProject
	return decodeRequiredResumeObject(data, resumeProjectKeys, (*plain)(value))
}

func (value *extractedResumeSkill) UnmarshalJSON(data []byte) error {
	type plain extractedResumeSkill
	return decodeRequiredResumeObject(data, resumeSkillKeys, (*plain)(value))
}

// decodeRequiredResumeObject enforces the exact database-prompt schema at
// every object layer. Unknown information is represented by the prompt's
// typed empty values ("", 0, false, or []), never by omitting a key or null.
func decodeRequiredResumeObject(data []byte, required []string, destination any) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return errors.New("expected JSON object")
	}
	allowed := make(map[string]struct{}, len(required))
	for _, key := range required {
		allowed[key] = struct{}{}
		value, exists := fields[key]
		if !exists {
			return fmt.Errorf("%w: required key %q is missing", ErrResumeSchema, key)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%w: required key %q must use its schema type", ErrResumeSchema, key)
		}
	}
	for key := range fields {
		if _, exists := allowed[key]; !exists {
			return fmt.Errorf("%w: unknown field %q", ErrResumeSchema, key)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func decodeResumeProfile(content string) (ResumeProfile, string, error) {
	raw, err := canonicalizeResumeProfileJSON(content)
	if err != nil {
		return ResumeProfile{}, "", err
	}
	var extracted extractedResumeProfile
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&extracted); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) || errors.Is(err, ErrResumeSchema) || strings.HasPrefix(err.Error(), "json: unknown field ") {
			return ResumeProfile{}, "", &ResumeExtractionError{Kind: ResumeExtractionSchema, Cause: errors.Join(ErrResumeSchema, err)}
		}
		return ResumeProfile{}, "", &ResumeExtractionError{Kind: ResumeExtractionJSON, Cause: errors.Join(ErrResumeJSON, err)}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ResumeProfile{}, "", &ResumeExtractionError{Kind: ResumeExtractionJSON, Cause: errors.Join(ErrResumeJSON, errors.New("multiple JSON values or trailing data"))}
	}
	profile, err := normalizeResumeProfile(extracted)
	if err != nil {
		return ResumeProfile{}, "", &ResumeExtractionError{Kind: ResumeExtractionSchema, Cause: errors.Join(ErrResumeSchema, err)}
	}
	return profile, raw, nil
}

// canonicalizeResumeProfileJSON repairs common LLM drifts before strict schema
// validation: markdown fences, surrounding prose, null/missing typed empties,
// unknown keys, and dotted/slashed date separators that appear in Chinese resumes.
func canonicalizeResumeProfileJSON(content string) (string, error) {
	raw := extractJSONObject(strings.TrimSpace(content))
	if raw == "" {
		return "", &ResumeExtractionError{Kind: ResumeExtractionJSON, Cause: errors.Join(ErrResumeJSON, errors.New("no JSON object found"))}
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		return "", &ResumeExtractionError{Kind: ResumeExtractionJSON, Cause: errors.Join(ErrResumeJSON, err)}
	}
	if root == nil {
		return "", &ResumeExtractionError{Kind: ResumeExtractionJSON, Cause: errors.Join(ErrResumeJSON, errors.New("expected JSON object"))}
	}
	normalized, err := normalizeResumeObjectMap(root, resumeProfileKeys, resumeProfileDefault)
	if err != nil {
		return "", &ResumeExtractionError{Kind: ResumeExtractionSchema, Cause: errors.Join(ErrResumeSchema, err)}
	}
	for _, key := range []string{"educations", "experiences", "projects", "skills"} {
		items, err := decodeJSONObjectArray(normalized[key])
		if err != nil {
			return "", &ResumeExtractionError{Kind: ResumeExtractionSchema, Cause: errors.Join(ErrResumeSchema, fmt.Errorf("%s: %w", key, err))}
		}
		keys, defaultFn := resumeArraySchema(key)
		repaired := make([]json.RawMessage, 0, len(items))
		for index, item := range items {
			object, err := normalizeResumeObjectMap(item, keys, defaultFn)
			if err != nil {
				return "", &ResumeExtractionError{Kind: ResumeExtractionSchema, Cause: errors.Join(ErrResumeSchema, fmt.Errorf("%s[%d]: %w", key, index, err))}
			}
			for _, dateKey := range []string{"start_date", "end_date"} {
				if rawDate, ok := object[dateKey]; ok {
					object[dateKey] = json.RawMessage(strconv.Quote(normalizeResumeDateInput(unquoteJSONString(rawDate))))
				}
			}
			encoded, err := json.Marshal(object)
			if err != nil {
				return "", &ResumeExtractionError{Kind: ResumeExtractionJSON, Cause: errors.Join(ErrResumeJSON, err)}
			}
			repaired = append(repaired, encoded)
		}
		encoded, err := json.Marshal(repaired)
		if err != nil {
			return "", &ResumeExtractionError{Kind: ResumeExtractionJSON, Cause: errors.Join(ErrResumeJSON, err)}
		}
		normalized[key] = encoded
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", &ResumeExtractionError{Kind: ResumeExtractionJSON, Cause: errors.Join(ErrResumeJSON, err)}
	}
	return string(encoded), nil
}

func resumeArraySchema(key string) ([]string, func(string) json.RawMessage) {
	switch key {
	case "educations":
		return resumeEducationKeys, resumeEducationDefault
	case "experiences":
		return resumeExperienceKeys, resumeExperienceDefault
	case "projects":
		return resumeProjectKeys, resumeProjectDefault
	default:
		return resumeSkillKeys, resumeSkillDefault
	}
}

func resumeProfileDefault(key string) json.RawMessage {
	switch key {
	case "total_experience_years":
		return json.RawMessage("0")
	case "educations", "experiences", "projects", "skills":
		return json.RawMessage("[]")
	default:
		return json.RawMessage(`""`)
	}
}

func resumeEducationDefault(string) json.RawMessage { return json.RawMessage(`""`) }

func resumeExperienceDefault(key string) json.RawMessage {
	switch key {
	case "is_current":
		return json.RawMessage("false")
	case "achievements":
		return json.RawMessage("[]")
	default:
		return json.RawMessage(`""`)
	}
}

func resumeProjectDefault(key string) json.RawMessage {
	switch key {
	case "technologies", "highlights":
		return json.RawMessage("[]")
	default:
		return json.RawMessage(`""`)
	}
}

func resumeSkillDefault(key string) json.RawMessage {
	if key == "years" {
		return json.RawMessage("0")
	}
	return json.RawMessage(`""`)
}

func normalizeResumeObjectMap(fields map[string]json.RawMessage, required []string, defaultFn func(string) json.RawMessage) (map[string]json.RawMessage, error) {
	if fields == nil {
		return nil, errors.New("expected JSON object")
	}
	out := make(map[string]json.RawMessage, len(required))
	for _, key := range required {
		value, exists := fields[key]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			out[key] = defaultFn(key)
			continue
		}
		out[key] = value
	}
	return out, nil
}

func decodeJSONObjectArray(raw json.RawMessage) ([]map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return []map[string]json.RawMessage{}, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(trimmed, &items); err != nil {
		return nil, err
	}
	out := make([]map[string]json.RawMessage, 0, len(items))
	for _, item := range items {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(item, &object); err != nil {
			return nil, err
		}
		out = append(out, object)
	}
	return out, nil
}

func extractJSONObject(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSpace(trimmed)
		if len(trimmed) >= 4 && strings.EqualFold(trimmed[:4], "json") {
			trimmed = strings.TrimSpace(trimmed[4:])
		}
		if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
			trimmed = strings.TrimSpace(trimmed[:idx])
		}
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end < start {
		return ""
	}
	return trimmed[start : end+1]
}

func unquoteJSONString(raw json.RawMessage) string {
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value
	}
	return strings.Trim(string(bytes.TrimSpace(raw)), `"`)
}

func normalizeResumeDateInput(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(".", "-", "/", "-", "年", "-", "月", "-", "日", "")
	normalized := strings.Trim(replacer.Replace(value), "-")
	normalized = strings.Join(strings.FieldsFunc(normalized, func(r rune) bool { return r == '-' }), "-")
	return normalized
}

func normalizeResumeProfile(extracted extractedResumeProfile) (ResumeProfile, error) {
	if extracted.TotalExperienceYears == nil {
		return ResumeProfile{}, errors.New("total_experience_years is required")
	}
	if *extracted.TotalExperienceYears < 0 {
		return ResumeProfile{}, errors.New("total_experience_years cannot be negative")
	}
	if extracted.Educations == nil || extracted.Experiences == nil || extracted.Projects == nil || extracted.Skills == nil {
		return ResumeProfile{}, errors.New("educations, experiences, projects, and skills are required arrays")
	}
	email := normalizeResumeEmail(extracted.Email)
	if email != "" {
		address, err := mail.ParseAddress(email)
		if err != nil || !strings.EqualFold(address.Address, email) {
			return ResumeProfile{}, errors.New("email is invalid")
		}
		email = strings.ToLower(email)
	}
	profile := ResumeProfile{
		FullName:             normalizeResumeText(extracted.FullName),
		Email:                email,
		Phone:                normalizeResumeText(extracted.Phone),
		Location:             normalizeResumeText(extracted.Location),
		Headline:             normalizeResumeText(extracted.Headline),
		Summary:              normalizeResumeText(extracted.Summary),
		TotalExperienceYears: roundResumeYears(*extracted.TotalExperienceYears),
		HighestDegree:        normalizeResumeText(extracted.HighestDegree),
		Educations:           make([]ResumeEducation, 0, len(extracted.Educations)),
		Experiences:          make([]ResumeExperience, 0, len(extracted.Experiences)),
		Projects:             make([]ResumeProject, 0, len(extracted.Projects)),
		Skills:               make([]ResumeSkill, 0, len(extracted.Skills)),
	}
	for index, item := range extracted.Educations {
		school := normalizeResumeText(item.School)
		if school == "" {
			return ResumeProfile{}, fmt.Errorf("educations[%d].school is required", index)
		}
		start, err := parseResumeDate(item.StartDate, false)
		if err != nil {
			return ResumeProfile{}, fmt.Errorf("educations[%d].start_date is invalid", index)
		}
		end, err := parseResumeDate(item.EndDate, true)
		if err != nil {
			return ResumeProfile{}, fmt.Errorf("educations[%d].end_date is invalid", index)
		}
		profile.Educations = append(profile.Educations, ResumeEducation{School: school, Degree: normalizeResumeText(item.Degree), Major: normalizeResumeText(item.Major), StartDate: start, EndDate: end, Description: normalizeResumeText(item.Description)})
	}
	for index, item := range extracted.Experiences {
		company := normalizeResumeText(item.Company)
		if company == "" {
			return ResumeProfile{}, fmt.Errorf("experiences[%d].company is required", index)
		}
		if item.Achievements == nil {
			return ResumeProfile{}, fmt.Errorf("experiences[%d].achievements is required as an array", index)
		}
		start, err := parseResumeDate(item.StartDate, false)
		if err != nil {
			return ResumeProfile{}, fmt.Errorf("experiences[%d].start_date is invalid", index)
		}
		end, err := parseResumeDate(item.EndDate, true)
		if err != nil {
			return ResumeProfile{}, fmt.Errorf("experiences[%d].end_date is invalid", index)
		}
		current := item.IsCurrent || isCurrentResumeDate(item.EndDate)
		if current {
			end = nil
		}
		profile.Experiences = append(profile.Experiences, ResumeExperience{Company: company, Title: normalizeResumeText(item.Title), Location: normalizeResumeText(item.Location), StartDate: start, EndDate: end, IsCurrent: current, Description: normalizeResumeText(item.Description), Achievements: normalizeStringList(item.Achievements)})
	}
	for index, item := range extracted.Projects {
		name := normalizeResumeText(item.Name)
		if name == "" {
			return ResumeProfile{}, fmt.Errorf("projects[%d].name is required", index)
		}
		if item.Technologies == nil || item.Highlights == nil {
			return ResumeProfile{}, fmt.Errorf("projects[%d].technologies and highlights are required arrays", index)
		}
		start, err := parseResumeDate(item.StartDate, false)
		if err != nil {
			return ResumeProfile{}, fmt.Errorf("projects[%d].start_date is invalid", index)
		}
		end, err := parseResumeDate(item.EndDate, true)
		if err != nil {
			return ResumeProfile{}, fmt.Errorf("projects[%d].end_date is invalid", index)
		}
		profile.Projects = append(profile.Projects, ResumeProject{Name: name, Role: normalizeResumeText(item.Role), StartDate: start, EndDate: end, Description: normalizeResumeText(item.Description), Technologies: normalizeStringList(item.Technologies), Highlights: normalizeStringList(item.Highlights)})
	}
	bySkill := make(map[string]ResumeSkill)
	for index, item := range extracted.Skills {
		name := normalizeSkillName(item.Name)
		if name == "" {
			return ResumeProfile{}, fmt.Errorf("skills[%d].name is required", index)
		}
		if item.Years < 0 {
			return ResumeProfile{}, fmt.Errorf("skills[%d].years cannot be negative", index)
		}
		key := strings.ToLower(name)
		candidate := ResumeSkill{Name: name, Category: strings.ToLower(normalizeResumeText(item.Category)), Level: strings.ToLower(normalizeResumeText(item.Level)), Years: roundResumeYears(item.Years), Evidence: normalizeResumeText(item.Evidence)}
		if existing, ok := bySkill[key]; ok {
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
				existing.Evidence = strings.Trim(strings.Join([]string{existing.Evidence, candidate.Evidence}, "; "), "; ")
			}
			bySkill[key] = existing
			continue
		}
		bySkill[key] = candidate
	}
	keys := make([]string, 0, len(bySkill))
	for key := range bySkill {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		profile.Skills = append(profile.Skills, bySkill[key])
	}
	if profile.FullName == "" && len(profile.Educations) == 0 && len(profile.Experiences) == 0 && len(profile.Projects) == 0 && len(profile.Skills) == 0 {
		return ResumeProfile{}, errors.New("profile must contain a name or material education, experience, project, or skill")
	}
	return profile, nil
}

func extractResumeHeuristically(source ResumeSource) (ResumeExtractionResult, error) {
	profile := extractedResumeProfile{
		Educations:  []extractedResumeEducation{},
		Experiences: []extractedResumeExperience{},
		Projects:    []extractedResumeProject{},
		Skills:      []extractedResumeSkill{},
	}
	zero := 0.0
	profile.TotalExperienceYears = &zero
	lines := strings.Split(source.ParsedText, "\n")
	for _, line := range lines {
		candidate := normalizeResumeText(line)
		if profile.FullName == "" && candidate != "" && len([]rune(candidate)) <= 80 && !strings.Contains(candidate, "@") && !strings.ContainsAny(candidate, "0123456789") {
			profile.FullName = candidate
		}
		if profile.Email == "" {
			for _, field := range strings.Fields(candidate) {
				field = strings.Trim(field, ",;()[]<>\"'")
				if address, err := mail.ParseAddress(field); err == nil && strings.EqualFold(address.Address, field) {
					profile.Email = field
					break
				}
			}
		}
	}
	lower := strings.ToLower(source.ParsedText)
	keywords := []string{"angular", "aws", "azure", "c#", "c++", "docker", "gcp", "git", "go", "golang", "grpc", "java", "javascript", "kafka", "kubernetes", "mongodb", "mysql", "node.js", "nodejs", "postgresql", "python", "react", "redis", "rust", "sql", "typescript", "vue"}
	seen := make(map[string]struct{})
	for _, keyword := range keywords {
		if !containsResumeKeyword(lower, keyword) {
			continue
		}
		name := normalizeSkillName(keyword)
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		profile.Skills = append(profile.Skills, extractedResumeSkill{Name: name})
	}
	raw, err := json.Marshal(profile)
	if err != nil {
		return ResumeExtractionResult{}, err
	}
	normalized, err := normalizeResumeProfile(profile)
	if err != nil {
		return ResumeExtractionResult{}, err
	}
	return ResumeExtractionResult{Profile: normalized, RawJSON: string(raw), InputHash: resumeInputHash(source.ParsedText), ParserVersion: ResumeHeuristicParserVersion, ExtractorType: "heuristic"}, nil
}

func classifyResumeExtractionError(err error) ResumeExtractionErrorKind {
	var extractionErr *ResumeExtractionError
	if errors.As(err, &extractionErr) {
		return extractionErr.Kind
	}
	var runtimeErr *RuntimeError
	if errors.As(err, &runtimeErr) {
		if runtimeErr.Kind == ErrorKindPolicy {
			return ResumeExtractionPolicy
		}
		if runtimeErr.Kind == ErrorKindPrompt {
			return ResumeExtractionPrompt
		}
		if errors.Is(runtimeErr, context.DeadlineExceeded) || errors.Is(runtimeErr, context.Canceled) {
			return ResumeExtractionTimeout
		}
		var aiErr *commonsai.AIError
		if errors.As(runtimeErr, &aiErr) && aiErr.Type == commonsai.AIEmptyReply {
			return ResumeExtractionEmpty
		}
		return ResumeExtractionProvider
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ResumeExtractionTimeout
	}
	return ResumeExtractionProvider
}

func resumeInputHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

func normalizeResumeText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func normalizeSkillName(value string) string {
	value = normalizeResumeText(value)
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
	default:
		return value
	}
}

func normalizeStringList(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		value = normalizeResumeText(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func roundResumeYears(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func normalizeResumeEmail(value string) string {
	value = normalizeResumeText(value)
	for _, prefix := range []string{"邮箱：", "邮箱:", "email:", "Email:", "E-mail:", "e-mail:"} {
		value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
	}
	return value
}

func parseResumeDate(value string, allowCurrent bool) (*time.Time, error) {
	value = strings.TrimSpace(strings.ToLower(normalizeResumeDateInput(value)))
	if value == "" || (allowCurrent && isCurrentResumeDate(value)) {
		return nil, nil
	}
	parts := strings.Split(value, "-")
	if len(parts) < 1 || len(parts) > 3 || len(parts[0]) != 4 {
		return nil, errors.New("expected YYYY, YYYY-MM, or YYYY-MM-DD")
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil || year < 1900 || year > 2099 {
		return nil, errors.New("invalid year")
	}
	month, day := 1, 1
	if len(parts) >= 2 {
		month, err = strconv.Atoi(parts[1])
		if err != nil || month < 1 || month > 12 {
			return nil, errors.New("invalid month")
		}
	}
	if len(parts) == 3 {
		day, err = strconv.Atoi(parts[2])
		if err != nil || day < 1 || day > 31 {
			return nil, errors.New("invalid day")
		}
	}
	parsed, err := time.Parse("2006-01-02", fmt.Sprintf("%04d-%02d-%02d", year, month, day))
	if err != nil {
		return nil, errors.New("invalid date")
	}
	return &parsed, nil
}

func isCurrentResumeDate(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "present", "current", "now", "ongoing", "至今", "现在", "目前":
		return true
	default:
		return false
	}
}

var resumeKeywordBoundary = regexp.MustCompile(`[a-z0-9+#.]`)

func containsResumeKeyword(text, keyword string) bool {
	for start := 0; start < len(text); {
		index := strings.Index(text[start:], keyword)
		if index < 0 {
			return false
		}
		index += start
		beforeOK := index == 0 || !resumeKeywordBoundary.MatchString(text[index-1:index])
		after := index + len(keyword)
		afterOK := after == len(text) || !resumeKeywordBoundary.MatchString(text[after:after+1])
		if beforeOK && afterOK {
			return true
		}
		start = index + 1
	}
	return false
}
