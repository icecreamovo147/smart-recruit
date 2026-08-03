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
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	commonsai "smart-recruit-commons/ai"
)

const (
	JobRequirementProfileVersion   = "job-requirement-profile-v1"
	JobRequirementLLMVersion       = "job-requirement-llm-v1"
	JobRequirementHeuristicVersion = "job-requirement-heuristic-v1"
	MaxJobRequirements             = 20
	MaxJobRequirementAliases       = 20

	RequirementCategoryCoreSkill     = "core_skill"
	RequirementCategoryExperience    = "experience"
	RequirementCategoryEducation     = "education"
	RequirementCategoryCertification = "certification"
	RequirementCategoryDomain        = "domain"
	RequirementCategoryLanguage      = "language"
	RequirementCategorySoftSkill     = "soft_skill"
	RequirementCategoryOther         = "other"

	RequirementPriorityMustHave   = "must_have"
	RequirementPriorityNiceToHave = "nice_to_have"
	RequirementPrioritySoftSkill  = "soft_skill"

	jobRequirementWeightTolerance = 1e-6
	jobRequirementWeightPrecision = 1_000_000
	maxJobRequirementTextRunes    = 500
	maxJobRequirementIDRunes      = 80
	maxJobRequirementAliasRunes   = 100
)

var (
	ErrJobRequirementEmptyResponse = errors.New("job requirement extractor returned an empty response")
	ErrJobRequirementJSON          = errors.New("job requirement extractor returned invalid JSON")
	ErrJobRequirementSchema        = errors.New("job requirement extractor returned an invalid schema")
	ErrJobRequirementFallback      = errors.New("job requirement heuristic fallback failed")

	jobRequirementIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	jobExperiencePattern    = regexp.MustCompile(`(?i)(\d+)\s*(?:\+|年以上|年|years?|yrs?)`)

	// These controlled marker sets implement a conservative zh-CN display
	// contract without pretending to perform complete language detection.
	// A display field must contain Han text and must not contain known
	// Traditional variants, Japanese kana, or common Japanese-only variants.
	traditionalDisplayMarkers = "後開發於評與專業經驗學歷證書語軟體網絡數據庫務團隊溝協應負責維護優實現構處問題雲運質領導項產國際從這個將會還讓並給裡時機關係統稱適職選"
	japaneseDisplayMarkers    = "発働験歴専応実関対処営総導産広変転図価済気読書込"
)

type JobRequirementExtractionErrorKind string

const (
	JobRequirementExtractionPrompt   JobRequirementExtractionErrorKind = "prompt"
	JobRequirementExtractionProvider JobRequirementExtractionErrorKind = "provider"
	JobRequirementExtractionTimeout  JobRequirementExtractionErrorKind = "timeout"
	JobRequirementExtractionEmpty    JobRequirementExtractionErrorKind = "empty_response"
	JobRequirementExtractionJSON     JobRequirementExtractionErrorKind = "json"
	JobRequirementExtractionSchema   JobRequirementExtractionErrorKind = "schema"
	JobRequirementExtractionPolicy   JobRequirementExtractionErrorKind = "policy"
	JobRequirementExtractionFallback JobRequirementExtractionErrorKind = "fallback"
)

// JobRequirementExtractionError is safe for logs and RPC errors: it never
// embeds job text, prompt content, or model output in its string form.
type JobRequirementExtractionError struct {
	Kind  JobRequirementExtractionErrorKind
	Cause error
}

func (e *JobRequirementExtractionError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("job requirement extraction failed (%s)", e.Kind)
}

func (e *JobRequirementExtractionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type JobRequirementSource struct {
	JobTitle     string
	Department   string
	Location     string
	Description  string
	Requirements string
}

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
	Aliases     []string `json:"aliases"`
}

type JobRequirementExtractionResult struct {
	Profile       JobRequirementProfile
	RawJSON       string
	InputHash     string
	ParserVersion string
	ExtractorType string
	ModelName     string
	Prompt        PromptDescriptor
	FallbackUsed  bool
	FallbackKind  JobRequirementExtractionErrorKind
}

type JobRequirementExtractor struct {
	runtime *Runtime
	policy  RuntimePolicy
}

func NewJobRequirementExtractor(runtime *Runtime, policy RuntimePolicy) *JobRequirementExtractor {
	return &JobRequirementExtractor{runtime: runtime, policy: policy.effective()}
}

func (e *JobRequirementExtractor) Extract(ctx context.Context, source JobRequirementSource) (JobRequirementExtractionResult, error) {
	started := time.Now()
	if e == nil || !e.policy.CandidateMatchExecutionPlan().StructuredCompletionEnabled() {
		return JobRequirementExtractionResult{}, &JobRequirementExtractionError{Kind: JobRequirementExtractionPolicy, Cause: ErrCapabilityDisabled}
	}
	primary, err := e.extractPrimary(ctx, source)
	if err == nil {
		observeRuntimeOutcome(e.runtime, ctx, "job_requirement_extraction", AgentTypeJobRequirementExtractor, "success", "none", started)
		return primary, nil
	}
	kind := classifyJobRequirementExtractionError(err)
	if IsStrictOutputError(err) || kind == JobRequirementExtractionPolicy || !e.policy.FallbacksEnabled() {
		observeRuntimeOutcome(e.runtime, ctx, "job_requirement_extraction", AgentTypeJobRequirementExtractor, "error", "disabled", started)
		return JobRequirementExtractionResult{}, err
	}
	fallback, fallbackErr := extractJobRequirementsHeuristically(source)
	if fallbackErr != nil {
		observeRuntimeOutcome(e.runtime, ctx, "job_requirement_extraction", AgentTypeJobRequirementExtractor, "error", string(kind), started)
		return JobRequirementExtractionResult{}, &JobRequirementExtractionError{
			Kind:  JobRequirementExtractionFallback,
			Cause: errors.Join(err, ErrJobRequirementFallback, fallbackErr),
		}
	}
	fallback.FallbackUsed = true
	fallback.FallbackKind = kind
	observeRuntimeOutcome(e.runtime, ctx, "job_requirement_extraction", AgentTypeJobRequirementExtractor, "success", string(kind), started)
	return fallback, nil
}

func (e *JobRequirementExtractor) extractPrimary(ctx context.Context, source JobRequirementSource) (JobRequirementExtractionResult, error) {
	if e.runtime == nil {
		return JobRequirementExtractionResult{}, &JobRequirementExtractionError{Kind: JobRequirementExtractionProvider, Cause: ErrProvider}
	}
	completion, err := e.runtime.Complete(ctx, AgentTypeJobRequirementExtractor, jobRequirementUserMessage(source))
	if err != nil {
		return JobRequirementExtractionResult{}, &JobRequirementExtractionError{Kind: classifyJobRequirementExtractionError(err), Cause: err}
	}
	if strings.TrimSpace(completion.Content) == "" {
		return JobRequirementExtractionResult{}, &JobRequirementExtractionError{Kind: JobRequirementExtractionEmpty, Cause: ErrJobRequirementEmptyResponse}
	}
	profile, raw, err := decodeJobRequirementProfile(completion.Content)
	if err != nil {
		if completion.StrictContractApplied {
			return JobRequirementExtractionResult{}, strictDomainError(err)
		}
		return JobRequirementExtractionResult{}, err
	}
	return JobRequirementExtractionResult{
		Profile:       profile,
		RawJSON:       raw,
		InputHash:     jobRequirementInputHash(source),
		ParserVersion: JobRequirementLLMVersion,
		ExtractorType: "llm",
		ModelName:     completion.ModelName,
		Prompt:        completion.Prompt,
	}, nil
}

func jobRequirementUserMessage(source JobRequirementSource) string {
	return strings.Join([]string{
		"岗位信息：",
		"岗位名称：" + strings.TrimSpace(source.JobTitle),
		"所属部门：" + strings.TrimSpace(source.Department),
		"工作地点：" + strings.TrimSpace(source.Location),
		"岗位描述：",
		strings.TrimSpace(source.Description),
		"岗位要求：",
		strings.TrimSpace(source.Requirements),
	}, "\n")
}

var (
	jobRequirementProfileKeys = []string{"profile_version", "requirements"}
	jobRequirementItemKeys    = []string{"id", "category", "label", "description", "priority", "weight", "knockout", "aliases"}
)

func (value *JobRequirementProfile) UnmarshalJSON(data []byte) error {
	type plain JobRequirementProfile
	return decodeRequiredJobRequirementObject(data, jobRequirementProfileKeys, (*plain)(value))
}

func (value *JobRequirementItem) UnmarshalJSON(data []byte) error {
	type plain JobRequirementItem
	return decodeRequiredJobRequirementObject(data, jobRequirementItemKeys, (*plain)(value))
}

func decodeRequiredJobRequirementObject(data []byte, required []string, destination any) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("%w: expected JSON object", ErrJobRequirementSchema)
	}
	allowed := make(map[string]struct{}, len(required))
	for _, key := range required {
		allowed[key] = struct{}{}
		raw, exists := fields[key]
		if !exists {
			return fmt.Errorf("%w: required key %q is missing", ErrJobRequirementSchema, key)
		}
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("%w: required key %q must use its schema type", ErrJobRequirementSchema, key)
		}
	}
	for key := range fields {
		if _, exists := allowed[key]; !exists {
			return fmt.Errorf("%w: unknown field %q", ErrJobRequirementSchema, key)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func decodeJobRequirementProfile(content string) (JobRequirementProfile, string, error) {
	raw := strings.TrimSpace(content)
	var profile JobRequirementProfile
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&profile); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) || errors.Is(err, ErrJobRequirementSchema) || strings.HasPrefix(err.Error(), "json: unknown field ") {
			return JobRequirementProfile{}, "", &JobRequirementExtractionError{Kind: JobRequirementExtractionSchema, Cause: errors.Join(ErrJobRequirementSchema, err)}
		}
		return JobRequirementProfile{}, "", &JobRequirementExtractionError{Kind: JobRequirementExtractionJSON, Cause: errors.Join(ErrJobRequirementJSON, err)}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return JobRequirementProfile{}, "", &JobRequirementExtractionError{Kind: JobRequirementExtractionJSON, Cause: errors.Join(ErrJobRequirementJSON, errors.New("multiple JSON values or trailing data"))}
	}
	if err := normalizeAndValidateJobRequirementProfile(&profile); err != nil {
		return JobRequirementProfile{}, "", &JobRequirementExtractionError{Kind: JobRequirementExtractionSchema, Cause: errors.Join(ErrJobRequirementSchema, err)}
	}
	return profile, raw, nil
}

func normalizeAndValidateJobRequirementProfile(profile *JobRequirementProfile) error {
	if profile == nil || profile.ProfileVersion != JobRequirementProfileVersion {
		return fmt.Errorf("profile_version must be %q", JobRequirementProfileVersion)
	}
	if len(profile.Requirements) == 0 || len(profile.Requirements) > MaxJobRequirements {
		return fmt.Errorf("requirements count must be between 1 and %d", MaxJobRequirements)
	}
	seenIDs := make(map[string]struct{}, len(profile.Requirements))
	total := 0.0
	for index := range profile.Requirements {
		item := &profile.Requirements[index]
		item.Label = normalizeRequirementText(item.Label)
		item.Description = normalizeRequirementText(item.Description)
		if item.ID == "" || len([]rune(item.ID)) > maxJobRequirementIDRunes || !jobRequirementIDPattern.MatchString(item.ID) {
			return fmt.Errorf("requirements[%d].id must be bounded lowercase kebab-case", index)
		}
		if _, exists := seenIDs[item.ID]; exists {
			return fmt.Errorf("requirements[%d].id is duplicated", index)
		}
		seenIDs[item.ID] = struct{}{}
		if !validJobRequirementCategory(item.Category) {
			return fmt.Errorf("requirements[%d].category is invalid", index)
		}
		if !validJobRequirementPriority(item.Priority) {
			return fmt.Errorf("requirements[%d].priority is invalid", index)
		}
		if item.Label == "" || len([]rune(item.Label)) > maxJobRequirementTextRunes {
			return fmt.Errorf("requirements[%d].label must be non-empty and bounded", index)
		}
		if item.Description == "" || len([]rune(item.Description)) > maxJobRequirementTextRunes {
			return fmt.Errorf("requirements[%d].description must be non-empty and bounded", index)
		}
		if !isConservativeSimplifiedChineseDisplayText(item.Label) {
			return fmt.Errorf("requirements[%d].label must use Simplified Chinese display text", index)
		}
		if !isConservativeSimplifiedChineseDisplayText(item.Description) {
			return fmt.Errorf("requirements[%d].description must use Simplified Chinese display text", index)
		}
		if math.IsNaN(item.Weight) || math.IsInf(item.Weight, 0) || item.Weight <= 0 || item.Weight > 1 {
			return fmt.Errorf("requirements[%d].weight must be finite and in (0,1]", index)
		}
		if item.Aliases == nil || len(item.Aliases) > MaxJobRequirementAliases {
			return fmt.Errorf("requirements[%d].aliases must be an array with at most %d items", index, MaxJobRequirementAliases)
		}
		aliases := make([]string, 0, len(item.Aliases))
		seenAliases := make(map[string]struct{}, len(item.Aliases))
		for aliasIndex, alias := range item.Aliases {
			alias = normalizeRequirementText(alias)
			if alias == "" || len([]rune(alias)) > maxJobRequirementAliasRunes {
				return fmt.Errorf("requirements[%d].aliases[%d] must be non-empty and bounded", index, aliasIndex)
			}
			key := strings.ToLower(alias)
			if _, exists := seenAliases[key]; exists {
				return fmt.Errorf("requirements[%d].aliases[%d] is duplicated", index, aliasIndex)
			}
			seenAliases[key] = struct{}{}
			aliases = append(aliases, alias)
		}
		item.Aliases = aliases
		total += item.Weight
	}
	if math.Abs(total-1) > jobRequirementWeightTolerance {
		return fmt.Errorf("requirement weights must sum to 1.0")
	}
	return normalizeMinorJobRequirementWeightDrift(profile.Requirements, total)
}

func normalizeMinorJobRequirementWeightDrift(items []JobRequirementItem, total float64) error {
	if len(items) == 0 {
		return errors.New("requirement weights cannot be normalized without requirements")
	}
	// Assign the tiny residual to the final item. This is stable and avoids
	// changing the relative meaning of every model-provided weight.
	if total != 1 {
		last := len(items) - 1
		adjusted := items[last].Weight + (1 - total)
		if math.IsNaN(adjusted) || math.IsInf(adjusted, 0) || adjusted <= 0 || adjusted > 1 {
			return errors.New("minor weight normalization would produce an invalid weight")
		}
		items[last].Weight = adjusted
	}

	normalizedTotal := 0.0
	for index := range items {
		weight := items[index].Weight
		if math.IsNaN(weight) || math.IsInf(weight, 0) || weight <= 0 || weight > 1 {
			return fmt.Errorf("requirements[%d].weight is invalid after normalization", index)
		}
		normalizedTotal += weight
	}
	if math.Abs(normalizedTotal-1) > jobRequirementWeightTolerance {
		return errors.New("requirement weights are invalid after normalization")
	}
	return nil
}

func isConservativeSimplifiedChineseDisplayText(value string) bool {
	hasHan := false
	for _, current := range value {
		if unicode.Is(unicode.Hiragana, current) || unicode.Is(unicode.Katakana, current) ||
			strings.ContainsRune(traditionalDisplayMarkers, current) || strings.ContainsRune(japaneseDisplayMarkers, current) {
			return false
		}
		if unicode.Is(unicode.Han, current) {
			hasHan = true
		}
	}
	return hasHan
}

func validJobRequirementCategory(value string) bool {
	switch value {
	case RequirementCategoryCoreSkill, RequirementCategoryExperience, RequirementCategoryEducation,
		RequirementCategoryCertification, RequirementCategoryDomain, RequirementCategoryLanguage,
		RequirementCategorySoftSkill, RequirementCategoryOther:
		return true
	default:
		return false
	}
}

func validJobRequirementPriority(value string) bool {
	switch value {
	case RequirementPriorityMustHave, RequirementPriorityNiceToHave, RequirementPrioritySoftSkill:
		return true
	default:
		return false
	}
}

type heuristicRequirementPattern struct {
	terms    []string
	id       string
	label    string
	category string
	priority string
	knockout bool
}

var heuristicRequirementPatterns = []heuristicRequirementPattern{
	{[]string{"go", "golang"}, "go", "Go 开发能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"java"}, "java", "Java 开发能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"python"}, "python", "Python 开发能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"typescript"}, "typescript", "TypeScript 开发能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"javascript"}, "javascript", "JavaScript 开发能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"vue", "vue.js"}, "vue", "Vue 前端开发能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"react", "react.js"}, "react", "React 前端开发能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"mysql"}, "mysql", "MySQL 数据库能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"postgresql", "postgres"}, "postgresql", "PostgreSQL 数据库能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"redis"}, "redis", "Redis 缓存能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"kubernetes", "k8s"}, "kubernetes", "Kubernetes 容器编排能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"docker"}, "docker", "Docker 容器化能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"grpc"}, "grpc", "gRPC 服务开发能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"kafka"}, "kafka", "Kafka 消息系统能力", RequirementCategoryCoreSkill, RequirementPriorityMustHave, false},
	{[]string{"微服务", "microservice"}, "microservices", "微服务架构能力", RequirementCategoryDomain, RequirementPriorityMustHave, false},
	{[]string{"分布式", "distributed"}, "distributed-systems", "分布式系统能力", RequirementCategoryDomain, RequirementPriorityMustHave, false},
	{[]string{"英语", "英文", "english"}, "english", "英语沟通能力", RequirementCategoryLanguage, RequirementPriorityNiceToHave, false},
	{[]string{"沟通", "communication"}, "communication", "沟通协作能力", RequirementCategorySoftSkill, RequirementPrioritySoftSkill, false},
	{[]string{"团队合作", "团队协作", "teamwork"}, "teamwork", "团队协作能力", RequirementCategorySoftSkill, RequirementPrioritySoftSkill, false},
}

func extractJobRequirementsHeuristically(source JobRequirementSource) (JobRequirementExtractionResult, error) {
	text := strings.Join([]string{source.JobTitle, source.Department, source.Location, source.Description, source.Requirements}, "\n")
	lower := strings.ToLower(text)
	items := make([]JobRequirementItem, 0, MaxJobRequirements)
	seen := make(map[string]struct{})
	add := func(item JobRequirementItem) {
		if len(items) >= MaxJobRequirements {
			return
		}
		if _, exists := seen[item.ID]; exists {
			return
		}
		seen[item.ID] = struct{}{}
		items = append(items, item)
	}
	for _, pattern := range heuristicRequirementPatterns {
		aliases := make([]string, 0, len(pattern.terms))
		matched := false
		for _, term := range pattern.terms {
			if containsRequirementTerm(lower, term) {
				matched = true
				aliases = append(aliases, term)
			}
		}
		if matched {
			add(JobRequirementItem{ID: pattern.id, Category: pattern.category, Label: pattern.label, Description: pattern.label + "将用于候选人岗位匹配。", Priority: pattern.priority, Knockout: pattern.knockout, Aliases: aliases})
		}
	}
	if match := jobExperiencePattern.FindStringSubmatch(text); len(match) > 1 {
		years, _ := strconv.Atoi(match[1])
		if years > 0 {
			label := fmt.Sprintf("%d 年以上相关工作经验", years)
			add(JobRequirementItem{ID: fmt.Sprintf("experience-%d-years", years), Category: RequirementCategoryExperience, Label: label, Description: label + "将作为岗位经验匹配要求。", Priority: RequirementPriorityMustHave, Knockout: true, Aliases: []string{fmt.Sprintf("%d年经验", years), fmt.Sprintf("%d years experience", years)}})
		}
	}
	for _, degree := range []struct {
		terms []string
		id    string
		label string
	}{
		{[]string{"博士", "phd", "doctorate"}, "doctorate-degree", "博士学历"},
		{[]string{"硕士", "master"}, "master-degree", "硕士学历"},
		{[]string{"本科", "bachelor"}, "bachelor-degree", "本科学历"},
		{[]string{"大专", "college degree"}, "college-degree", "大专学历"},
	} {
		for _, term := range degree.terms {
			if containsRequirementTerm(lower, term) {
				add(JobRequirementItem{ID: degree.id, Category: RequirementCategoryEducation, Label: degree.label, Description: degree.label + "将作为岗位学历匹配要求。", Priority: RequirementPriorityMustHave, Knockout: true, Aliases: append([]string(nil), degree.terms...)})
				break
			}
		}
	}
	if len(items) == 0 {
		label := "基本岗位要求"
		if title := normalizeRequirementText(source.JobTitle); title != "" {
			label = title + "基本岗位要求"
		}
		add(JobRequirementItem{ID: "general-requirement", Category: RequirementCategoryOther, Label: label, Description: "根据岗位名称与岗位说明评估候选人的基本适配程度。", Priority: RequirementPriorityMustHave, Aliases: []string{"岗位要求"}})
	}
	assignHeuristicRequirementWeights(items)
	profile := JobRequirementProfile{ProfileVersion: JobRequirementProfileVersion, Requirements: items}
	if err := normalizeAndValidateJobRequirementProfile(&profile); err != nil {
		return JobRequirementExtractionResult{}, err
	}
	raw, err := json.Marshal(profile)
	if err != nil {
		return JobRequirementExtractionResult{}, err
	}
	return JobRequirementExtractionResult{
		Profile:       profile,
		RawJSON:       string(raw),
		InputHash:     jobRequirementInputHash(source),
		ParserVersion: JobRequirementHeuristicVersion,
		ExtractorType: "heuristic",
	}, nil
}

func assignHeuristicRequirementWeights(items []JobRequirementItem) {
	if len(items) == 0 {
		return
	}
	units := make([]int64, len(items))
	var totalUnits int64
	for index := range items {
		switch items[index].Priority {
		case RequirementPriorityMustHave:
			units[index] = 3
		case RequirementPriorityNiceToHave:
			units[index] = 2
		default:
			units[index] = 1
		}
		totalUnits += units[index]
	}
	remaining := int64(jobRequirementWeightPrecision)
	for index := range items {
		share := units[index] * int64(jobRequirementWeightPrecision) / totalUnits
		if index == len(items)-1 {
			share = remaining
		}
		items[index].Weight = float64(share) / jobRequirementWeightPrecision
		remaining -= share
	}
}

func containsRequirementTerm(text, term string) bool {
	term = strings.ToLower(term)
	if term == "" {
		return false
	}
	for offset := 0; offset < len(text); {
		index := strings.Index(text[offset:], term)
		if index < 0 {
			return false
		}
		index += offset
		beforeOK := index == 0 || !isRequirementWordRune(rune(text[index-1]))
		after := index + len(term)
		afterOK := after == len(text) || !isRequirementWordRune(rune(text[after]))
		if containsNonASCII(term) || (beforeOK && afterOK) {
			return true
		}
		offset = index + 1
	}
	return false
}

func isRequirementWordRune(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value) || value == '+' || value == '#' || value == '.'
}

func containsNonASCII(value string) bool {
	for _, current := range value {
		if current > unicode.MaxASCII {
			return true
		}
	}
	return false
}

func classifyJobRequirementExtractionError(err error) JobRequirementExtractionErrorKind {
	var extractionErr *JobRequirementExtractionError
	if errors.As(err, &extractionErr) {
		return extractionErr.Kind
	}
	var runtimeErr *RuntimeError
	if errors.As(err, &runtimeErr) {
		if runtimeErr.Kind == ErrorKindPolicy {
			return JobRequirementExtractionPolicy
		}
		if runtimeErr.Kind == ErrorKindPrompt {
			return JobRequirementExtractionPrompt
		}
		if errors.Is(runtimeErr, context.DeadlineExceeded) || errors.Is(runtimeErr, context.Canceled) {
			return JobRequirementExtractionTimeout
		}
		var aiErr *commonsai.AIError
		if errors.As(runtimeErr, &aiErr) {
			switch aiErr.Type {
			case commonsai.AIEmptyReply:
				return JobRequirementExtractionEmpty
			case commonsai.AITimeout, commonsai.AIContextCanceled:
				return JobRequirementExtractionTimeout
			}
		}
		return JobRequirementExtractionProvider
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return JobRequirementExtractionTimeout
	}
	return JobRequirementExtractionProvider
}

func jobRequirementInputHash(source JobRequirementSource) string {
	canonical := strings.Join([]string{
		strings.TrimSpace(source.JobTitle), strings.TrimSpace(source.Department), strings.TrimSpace(source.Location),
		strings.TrimSpace(source.Description), strings.TrimSpace(source.Requirements), JobRequirementProfileVersion,
	}, "\n")
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:])
}

func normalizeRequirementText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

// StableJobRequirements returns a defensive copy ordered by ID for consumers
// that require canonical traversal without mutating extraction order.
func StableJobRequirements(profile JobRequirementProfile) []JobRequirementItem {
	items := append([]JobRequirementItem(nil), profile.Requirements...)
	for index := range items {
		items[index].Aliases = append([]string(nil), items[index].Aliases...)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

// ExtractJobRequirementsDeterministically exposes the validated heuristic
// extractor to the deterministic primary candidate-match path. It performs no
// model call and keeps the job-requirement profile internal to the runtime.
func ExtractJobRequirementsDeterministically(source JobRequirementSource) (JobRequirementExtractionResult, error) {
	return extractJobRequirementsHeuristically(source)
}
