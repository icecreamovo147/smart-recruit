package recruiting_intelligence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	commonsai "smart-recruit-commons/ai"
)

const validJobRequirementJSON = `{
  "profile_version":"job-requirement-profile-v1",
  "requirements":[
    {"id":"go-backend","category":"core_skill","label":"Go 后端开发能力","description":"用于评估后端开发能力。","priority":"must_have","weight":0.6,"knockout":false,"aliases":["Go","Golang"]},
    {"id":"communication","category":"soft_skill","label":"沟通能力","description":"用于评估团队沟通能力。","priority":"soft_skill","weight":0.4,"knockout":false,"aliases":["沟通"]}
  ]
}`

func TestJobRequirementExtractorUsesDatabaseSystemPrompt(t *testing.T) {
	store := &jobRequirementPromptStore{prompt: validJobRequirementPrompt()}
	provider := &jobRequirementProvider{content: validJobRequirementJSON, model: "model-a"}
	policy := enhancedJobRequirementPolicy(true)
	extractor := NewJobRequirementExtractor(NewRuntime(NewPromptLoader(store), provider, policy), policy)
	source := JobRequirementSource{JobTitle: "Go 工程师", Department: "平台部", Location: "上海", Description: "负责后端服务", Requirements: "熟悉 Go"}

	result, err := extractor.Extract(context.Background(), source)
	if err != nil {
		t.Fatalf("Extract error = %v", err)
	}
	if len(store.calls) != 1 || store.calls[0] != [2]string{AgentTypeJobRequirementExtractor, PromptRoleSystem} {
		t.Fatalf("prompt calls = %#v", store.calls)
	}
	if provider.system != "database job requirement system prompt" {
		t.Fatalf("system prompt = %q", provider.system)
	}
	for _, expected := range []string{"岗位名称：Go 工程师", "所属部门：平台部", "工作地点：上海", "岗位描述：\n负责后端服务", "岗位要求：\n熟悉 Go"} {
		if !strings.Contains(provider.user, expected) {
			t.Fatalf("user message missing %q: %q", expected, provider.user)
		}
	}
	if result.ParserVersion != JobRequirementLLMVersion || result.ExtractorType != "llm" || result.ModelName != "model-a" || result.Prompt.ID != 41 || result.FallbackUsed {
		t.Fatalf("result provenance = %+v", result)
	}
	if result.InputHash == "" || len(result.Profile.Requirements) != 2 {
		t.Fatalf("result = %+v", result)
	}
}

func TestJobRequirementStrictSchemaValidation(t *testing.T) {
	tooMany := `{"profile_version":"job-requirement-profile-v1","requirements":[`
	for index := 0; index <= MaxJobRequirements; index++ {
		if index > 0 {
			tooMany += ","
		}
		tooMany += fmt.Sprintf(`{"id":"item-%d","category":"other","label":"要求","description":"说明","priority":"nice_to_have","weight":%g,"knockout":false,"aliases":[]}`, index, 1.0/float64(MaxJobRequirements+1))
	}
	tooMany += `]}`

	tests := []struct {
		name    string
		content string
		kind    JobRequirementExtractionErrorKind
	}{
		{name: "leading text", content: "result: " + validJobRequirementJSON, kind: JobRequirementExtractionJSON},
		{name: "trailing text", content: validJobRequirementJSON + " done", kind: JobRequirementExtractionJSON},
		{name: "multiple json values", content: validJobRequirementJSON + ` {}`, kind: JobRequirementExtractionJSON},
		{name: "top level null", content: `null`, kind: JobRequirementExtractionSchema},
		{name: "top level array", content: `[]`, kind: JobRequirementExtractionSchema},
		{name: "unknown field", content: strings.Replace(validJobRequirementJSON, `"profile_version":`, `"unknown":true,"profile_version":`, 1), kind: JobRequirementExtractionSchema},
		{name: "missing profile key", content: strings.Replace(validJobRequirementJSON, `  "profile_version":"job-requirement-profile-v1",`+"\n", "", 1), kind: JobRequirementExtractionSchema},
		{name: "wrong profile version", content: strings.Replace(validJobRequirementJSON, JobRequirementProfileVersion, "v2", 1), kind: JobRequirementExtractionSchema},
		{name: "missing item key", content: strings.Replace(validJobRequirementJSON, `"description":"用于评估后端开发能力。",`, "", 1), kind: JobRequirementExtractionSchema},
		{name: "missing knockout", content: strings.Replace(validJobRequirementJSON, `"knockout":false,`, "", 1), kind: JobRequirementExtractionSchema},
		{name: "missing aliases", content: strings.Replace(validJobRequirementJSON, `,"aliases":["Go","Golang"]`, "", 1), kind: JobRequirementExtractionSchema},
		{name: "null requirement item", content: `{"profile_version":"job-requirement-profile-v1","requirements":[null]}`, kind: JobRequirementExtractionSchema},
		{name: "null aliases", content: strings.Replace(validJobRequirementJSON, `["Go","Golang"]`, `null`, 1), kind: JobRequirementExtractionSchema},
		{name: "string aliases", content: strings.Replace(validJobRequirementJSON, `["Go","Golang"]`, `"Go"`, 1), kind: JobRequirementExtractionSchema},
		{name: "string knockout", content: strings.Replace(validJobRequirementJSON, `"knockout":false`, `"knockout":"false"`, 1), kind: JobRequirementExtractionSchema},
		{name: "invalid category", content: strings.Replace(validJobRequirementJSON, `"category":"core_skill"`, `"category":"unknown"`, 1), kind: JobRequirementExtractionSchema},
		{name: "invalid priority", content: strings.Replace(validJobRequirementJSON, `"priority":"must_have"`, `"priority":"critical"`, 1), kind: JobRequirementExtractionSchema},
		{name: "invalid id", content: strings.Replace(validJobRequirementJSON, `"id":"go-backend"`, `"id":"Go Backend"`, 1), kind: JobRequirementExtractionSchema},
		{name: "duplicate id", content: strings.Replace(validJobRequirementJSON, `"id":"communication"`, `"id":"go-backend"`, 1), kind: JobRequirementExtractionSchema},
		{name: "empty label", content: strings.Replace(validJobRequirementJSON, `"label":"Go 后端开发能力"`, `"label":""`, 1), kind: JobRequirementExtractionSchema},
		{name: "empty description", content: strings.Replace(validJobRequirementJSON, `"description":"用于评估后端开发能力。"`, `"description":""`, 1), kind: JobRequirementExtractionSchema},
		{name: "english label", content: strings.Replace(validJobRequirementJSON, `"label":"Go 后端开发能力"`, `"label":"Go backend engineering"`, 1), kind: JobRequirementExtractionSchema},
		{name: "english description", content: strings.Replace(validJobRequirementJSON, `"description":"用于评估后端开发能力。"`, `"description":"Evaluates backend engineering expertise."`, 1), kind: JobRequirementExtractionSchema},
		{name: "empty alias", content: strings.Replace(validJobRequirementJSON, `["Go","Golang"]`, `[""]`, 1), kind: JobRequirementExtractionSchema},
		{name: "duplicate alias", content: strings.Replace(validJobRequirementJSON, `["Go","Golang"]`, `["Go","go"]`, 1), kind: JobRequirementExtractionSchema},
		{name: "empty requirements", content: `{"profile_version":"job-requirement-profile-v1","requirements":[]}`, kind: JobRequirementExtractionSchema},
		{name: "too many requirements", content: tooMany, kind: JobRequirementExtractionSchema},
		{name: "zero weight", content: strings.Replace(validJobRequirementJSON, `"weight":0.6`, `"weight":0`, 1), kind: JobRequirementExtractionSchema},
		{name: "weight above one", content: strings.Replace(validJobRequirementJSON, `"weight":0.6`, `"weight":1.1`, 1), kind: JobRequirementExtractionSchema},
		{name: "material total drift", content: strings.Replace(validJobRequirementJSON, `"weight":0.6`, `"weight":0.59`, 1), kind: JobRequirementExtractionSchema},
	}
	policy := enhancedJobRequirementPolicy(false)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider := &jobRequirementProvider{content: tc.content}
			extractor := NewJobRequirementExtractor(jobRequirementRuntime(provider, validJobRequirementPrompt(), policy), policy)
			_, err := extractor.Extract(context.Background(), JobRequirementSource{JobTitle: "Engineer"})
			var extractionErr *JobRequirementExtractionError
			if !errors.As(err, &extractionErr) || extractionErr.Kind != tc.kind {
				t.Fatalf("error = %v, want kind %s", err, tc.kind)
			}
			if tc.name == "top level null" || tc.name == "null requirement item" {
				if !errors.Is(err, ErrJobRequirementSchema) {
					t.Fatalf("error = %v, want ErrJobRequirementSchema", err)
				}
			}
		})
	}
}

func TestJobRequirementMinorWeightDriftNormalization(t *testing.T) {
	content := strings.Replace(validJobRequirementJSON, `"weight":0.6`, `"weight":0.5999997`, 1)
	policy := enhancedJobRequirementPolicy(false)
	result, err := NewJobRequirementExtractor(jobRequirementRuntime(&jobRequirementProvider{content: content}, validJobRequirementPrompt(), policy), policy).Extract(context.Background(), JobRequirementSource{JobTitle: "Engineer"})
	if err != nil {
		t.Fatalf("Extract error = %v", err)
	}
	total := 0.0
	for _, item := range result.Profile.Requirements {
		total += item.Weight
	}
	if total != 1 || result.Profile.Requirements[1].Weight != 0.4000003 {
		t.Fatalf("normalized weights = %#v (total %.12f)", result.Profile.Requirements, total)
	}
}

func TestJobRequirementMinorWeightDriftRejectsInvalidResidual(t *testing.T) {
	content := strings.Replace(validJobRequirementJSON, `"weight":0.6`, `"weight":1.0`, 1)
	content = strings.Replace(content, `"weight":0.4`, `"weight":0.0000005`, 1)
	policy := enhancedJobRequirementPolicy(false)
	_, err := NewJobRequirementExtractor(jobRequirementRuntime(&jobRequirementProvider{content: content}, validJobRequirementPrompt(), policy), policy).Extract(context.Background(), JobRequirementSource{JobTitle: "Engineer"})
	var extractionErr *JobRequirementExtractionError
	if !errors.As(err, &extractionErr) || extractionErr.Kind != JobRequirementExtractionSchema || !errors.Is(err, ErrJobRequirementSchema) {
		t.Fatalf("error = %v, want schema error", err)
	}
}

func TestJobRequirementChineseDisplayTextAllowsTechnicalTerms(t *testing.T) {
	content := strings.Replace(validJobRequirementJSON, `"label":"Go 后端开发能力"`, `"label":"Go 与 gRPC 服务开发能力"`, 1)
	content = strings.Replace(content, `"description":"用于评估后端开发能力。"`, `"description":"用于评估 Kubernetes 容器编排与 gRPC 服务开发能力。"`, 1)
	policy := enhancedJobRequirementPolicy(false)
	if _, err := NewJobRequirementExtractor(jobRequirementRuntime(&jobRequirementProvider{content: content}, validJobRequirementPrompt(), policy), policy).Extract(context.Background(), JobRequirementSource{JobTitle: "Engineer"}); err != nil {
		t.Fatalf("Extract error = %v", err)
	}
}

func TestJobRequirementMachineFieldsRejectSurroundingWhitespaceWithFallbackPolicy(t *testing.T) {
	tests := []struct {
		name         string
		from         string
		to           string
		fallbacks    bool
		wantFallback bool
	}{
		{name: "id disabled", from: `"id":"go-backend"`, to: `"id":" go-backend "`, fallbacks: false},
		{name: "category disabled", from: `"category":"core_skill"`, to: `"category":" core_skill "`, fallbacks: false},
		{name: "priority disabled", from: `"priority":"must_have"`, to: `"priority":" must_have "`, fallbacks: false},
		{name: "id enabled", from: `"id":"go-backend"`, to: `"id":" go-backend "`, fallbacks: true, wantFallback: true},
		{name: "category enabled", from: `"category":"core_skill"`, to: `"category":" core_skill "`, fallbacks: true, wantFallback: true},
		{name: "priority enabled", from: `"priority":"must_have"`, to: `"priority":" must_have "`, fallbacks: true, wantFallback: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := strings.Replace(validJobRequirementJSON, tc.from, tc.to, 1)
			policy := enhancedJobRequirementPolicy(tc.fallbacks)
			result, err := NewJobRequirementExtractor(jobRequirementRuntime(&jobRequirementProvider{content: content}, validJobRequirementPrompt(), policy), policy).Extract(context.Background(), JobRequirementSource{JobTitle: "Go 工程师"})
			if tc.wantFallback {
				if err != nil {
					t.Fatalf("Extract error = %v", err)
				}
				if !result.FallbackUsed || result.FallbackKind != JobRequirementExtractionSchema || result.ExtractorType != "heuristic" {
					t.Fatalf("fallback result = %+v", result)
				}
				return
			}
			var extractionErr *JobRequirementExtractionError
			if !errors.As(err, &extractionErr) || extractionErr.Kind != JobRequirementExtractionSchema || !errors.Is(err, ErrJobRequirementSchema) {
				t.Fatalf("error = %v, want schema error", err)
			}
		})
	}
}

func TestJobRequirementSimplifiedChineseDisplayPolicy(t *testing.T) {
	tests := []struct {
		name         string
		from         string
		to           string
		fallbacks    bool
		wantFallback bool
	}{
		{name: "traditional label disabled", from: `"label":"Go 后端开发能力"`, to: `"label":"後端開發能力"`},
		{name: "traditional description disabled", from: `"description":"用于评估后端开发能力。"`, to: `"description":"用於評估後端開發能力。"`},
		{name: "japanese kanji label disabled", from: `"label":"Go 后端开发能力"`, to: `"label":"後端開発能力"`},
		{name: "japanese kana description disabled", from: `"description":"用于评估后端开发能力。"`, to: `"description":"バックエンド開発能力を評価する。"`},
		{name: "traditional label enabled", from: `"label":"Go 后端开发能力"`, to: `"label":"後端開發能力"`, fallbacks: true, wantFallback: true},
		{name: "traditional description enabled", from: `"description":"用于评估后端开发能力。"`, to: `"description":"用於評估後端開發能力。"`, fallbacks: true, wantFallback: true},
		{name: "japanese kanji label enabled", from: `"label":"Go 后端开发能力"`, to: `"label":"後端開発能力"`, fallbacks: true, wantFallback: true},
		{name: "japanese kana description enabled", from: `"description":"用于评估后端开发能力。"`, to: `"description":"バックエンド開発能力を評価する。"`, fallbacks: true, wantFallback: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := strings.Replace(validJobRequirementJSON, tc.from, tc.to, 1)
			policy := enhancedJobRequirementPolicy(tc.fallbacks)
			result, err := NewJobRequirementExtractor(jobRequirementRuntime(&jobRequirementProvider{content: content}, validJobRequirementPrompt(), policy), policy).Extract(context.Background(), JobRequirementSource{JobTitle: "Go 工程师"})
			if tc.wantFallback {
				if err != nil {
					t.Fatalf("Extract error = %v", err)
				}
				if !result.FallbackUsed || result.FallbackKind != JobRequirementExtractionSchema || result.ExtractorType != "heuristic" {
					t.Fatalf("fallback result = %+v", result)
				}
				return
			}
			var extractionErr *JobRequirementExtractionError
			if !errors.As(err, &extractionErr) || extractionErr.Kind != JobRequirementExtractionSchema || !errors.Is(err, ErrJobRequirementSchema) {
				t.Fatalf("error = %v, want schema error", err)
			}
		})
	}
}

func TestJobRequirementEnglishDisplayTextFallbackEnabled(t *testing.T) {
	content := strings.Replace(validJobRequirementJSON, `"description":"用于评估后端开发能力。"`, `"description":"Evaluates Go and gRPC expertise."`, 1)
	policy := enhancedJobRequirementPolicy(true)
	result, err := NewJobRequirementExtractor(jobRequirementRuntime(&jobRequirementProvider{content: content}, validJobRequirementPrompt(), policy), policy).Extract(context.Background(), JobRequirementSource{JobTitle: "Go 工程师"})
	if err != nil {
		t.Fatalf("Extract error = %v", err)
	}
	if !result.FallbackUsed || result.FallbackKind != JobRequirementExtractionSchema || result.ExtractorType != "heuristic" {
		t.Fatalf("fallback result = %+v", result)
	}
}

func TestJobRequirementFallbackMatrix(t *testing.T) {
	tests := []struct {
		name     string
		prompt   PromptDescriptor
		provider *jobRequirementProvider
		kind     JobRequirementExtractionErrorKind
	}{
		{name: "prompt", prompt: PromptDescriptor{}, provider: &jobRequirementProvider{content: validJobRequirementJSON}, kind: JobRequirementExtractionPrompt},
		{name: "provider", prompt: validJobRequirementPrompt(), provider: &jobRequirementProvider{err: errors.New("provider unavailable")}, kind: JobRequirementExtractionProvider},
		{name: "adapter empty", prompt: validJobRequirementPrompt(), provider: &jobRequirementProvider{err: commonsai.NewAIError(commonsai.AIEmptyReply, "", errors.New("empty provider body"))}, kind: JobRequirementExtractionEmpty},
		{name: "adapter timeout", prompt: validJobRequirementPrompt(), provider: &jobRequirementProvider{err: commonsai.NewAIError(commonsai.AITimeout, "", errors.New("provider timeout"))}, kind: JobRequirementExtractionTimeout},
		{name: "timeout", prompt: validJobRequirementPrompt(), provider: &jobRequirementProvider{err: context.DeadlineExceeded}, kind: JobRequirementExtractionTimeout},
		{name: "empty", prompt: validJobRequirementPrompt(), provider: &jobRequirementProvider{content: "  "}, kind: JobRequirementExtractionEmpty},
		{name: "json", prompt: validJobRequirementPrompt(), provider: &jobRequirementProvider{content: "not-json"}, kind: JobRequirementExtractionJSON},
		{name: "schema", prompt: validJobRequirementPrompt(), provider: &jobRequirementProvider{content: `{"profile_version":"job-requirement-profile-v1","requirements":[]}`}, kind: JobRequirementExtractionSchema},
	}
	policy := enhancedJobRequirementPolicy(true)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			extractor := NewJobRequirementExtractor(jobRequirementRuntime(tc.provider, tc.prompt, policy), policy)
			result, err := extractor.Extract(context.Background(), JobRequirementSource{JobTitle: "Go Engineer", Requirements: "Go, PostgreSQL, 3 years experience"})
			if err != nil {
				t.Fatalf("Extract error = %v", err)
			}
			if !result.FallbackUsed || result.FallbackKind != tc.kind || result.ParserVersion != JobRequirementHeuristicVersion || result.ExtractorType != "heuristic" {
				t.Fatalf("fallback provenance = %+v", result)
			}
			if result.ModelName != "" || result.Prompt.ID != 0 || strings.Contains(fmt.Sprint(result), "provider unavailable") {
				t.Fatalf("fallback provenance leaked primary details: %+v", result)
			}
		})
	}
}

func TestJobRequirementFallbackDisabledAndPolicyGates(t *testing.T) {
	disabledFallback := enhancedJobRequirementPolicy(false)
	_, err := NewJobRequirementExtractor(jobRequirementRuntime(&jobRequirementProvider{content: `{"profile_version":"job-requirement-profile-v1","requirements":[]}`}, validJobRequirementPrompt(), disabledFallback), disabledFallback).Extract(context.Background(), JobRequirementSource{JobTitle: "Engineer"})
	var extractionErr *JobRequirementExtractionError
	if !errors.As(err, &extractionErr) || extractionErr.Kind != JobRequirementExtractionSchema {
		t.Fatalf("fallback-disabled error = %v", err)
	}

	for _, tc := range []struct {
		name   string
		policy RuntimePolicy
		ok     bool
	}{
		{name: "overall disabled blocks", policy: NewRuntimePolicy(RuntimePolicyConfig{CandidateMatch: false, CandidateMatchSemantic: true, CandidateMatchShadow: true, Fallbacks: true})},
		{name: "deterministic only blocks", policy: NewRuntimePolicy(RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: false, CandidateMatchShadow: false, Fallbacks: true})},
		{name: "enhanced primary allows", policy: NewRuntimePolicy(RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, CandidateMatchShadow: false, Fallbacks: true}), ok: true},
		{name: "enhanced shadow allows", policy: NewRuntimePolicy(RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: false, CandidateMatchShadow: true, Fallbacks: true}), ok: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &jobRequirementProvider{content: validJobRequirementJSON}
			extractor := NewJobRequirementExtractor(jobRequirementRuntime(provider, validJobRequirementPrompt(), tc.policy), tc.policy)
			_, err := extractor.Extract(context.Background(), JobRequirementSource{JobTitle: "Engineer"})
			if tc.ok && err != nil {
				t.Fatalf("Extract error = %v", err)
			}
			if !tc.ok {
				var policyErr *JobRequirementExtractionError
				if !errors.As(err, &policyErr) || policyErr.Kind != JobRequirementExtractionPolicy || provider.calls != 0 {
					t.Fatalf("policy result: error=%v provider calls=%d", err, provider.calls)
				}
			}
		})
	}
}

func TestJobRequirementHeuristicIsDeterministicAndValid(t *testing.T) {
	source := JobRequirementSource{
		JobTitle:     "Senior Go Backend Engineer",
		Department:   "Platform",
		Location:     "Shanghai",
		Description:  "负责微服务与分布式系统开发，需要良好沟通能力。",
		Requirements: "Go, PostgreSQL, Kubernetes, 3 years experience, bachelor degree",
	}
	first, err := extractJobRequirementsHeuristically(source)
	if err != nil {
		t.Fatalf("first extraction = %v", err)
	}
	second, err := extractJobRequirementsHeuristically(source)
	if err != nil {
		t.Fatalf("second extraction = %v", err)
	}
	if first.RawJSON != second.RawJSON || first.InputHash != second.InputHash {
		t.Fatalf("heuristic output is not deterministic:\n%s\n%s", first.RawJSON, second.RawJSON)
	}
	if err := normalizeAndValidateJobRequirementProfile(&first.Profile); err != nil {
		t.Fatalf("heuristic profile invalid: %v", err)
	}
	ids := make(map[string]struct{})
	total := 0.0
	for _, item := range first.Profile.Requirements {
		if _, exists := ids[item.ID]; exists {
			t.Fatalf("duplicate id %q", item.ID)
		}
		ids[item.ID] = struct{}{}
		total += item.Weight
	}
	if total != 1 {
		t.Fatalf("weight total = %.12f", total)
	}
	for _, expected := range []string{"go", "postgresql", "kubernetes", "experience-3-years", "bachelor-degree", "communication"} {
		if _, exists := ids[expected]; !exists {
			t.Errorf("missing heuristic requirement %q in %#v", expected, ids)
		}
	}

	general, err := extractJobRequirementsHeuristically(JobRequirementSource{JobTitle: "实习生"})
	if err != nil || len(general.Profile.Requirements) != 1 || general.Profile.Requirements[0].ID != "general-requirement" || general.Profile.Requirements[0].Description == "" {
		t.Fatalf("general fallback = %+v, %v", general, err)
	}
}

func TestStableJobRequirementsReturnsDefensiveSortedCopy(t *testing.T) {
	profile := JobRequirementProfile{Requirements: []JobRequirementItem{{ID: "z", Aliases: []string{"Z"}}, {ID: "a", Aliases: []string{"A"}}}}
	stable := StableJobRequirements(profile)
	stable[0].Aliases[0] = "changed"
	if stable[0].ID != "a" || profile.Requirements[1].Aliases[0] != "A" {
		t.Fatalf("stable=%+v profile=%+v", stable, profile)
	}
}

func validJobRequirementPrompt() PromptDescriptor {
	return PromptDescriptor{ID: 41, Name: "job-requirement-v2", Version: 2, AgentType: AgentTypeJobRequirementExtractor, Role: PromptRoleSystem, Content: "database job requirement system prompt"}
}

func enhancedJobRequirementPolicy(fallbacks bool) RuntimePolicy {
	return NewRuntimePolicy(RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: fallbacks, CandidateMatchTimeout: time.Second})
}

func jobRequirementRuntime(provider *jobRequirementProvider, prompt PromptDescriptor, policy RuntimePolicy) *Runtime {
	return NewRuntime(NewPromptLoader(&jobRequirementPromptStore{prompt: prompt}), provider, policy)
}

type jobRequirementPromptStore struct {
	prompt PromptDescriptor
	err    error
	calls  [][2]string
}

func (s *jobRequirementPromptStore) LoadActiveRecruitingPrompt(_ context.Context, agentType, role string) (PromptDescriptor, error) {
	s.calls = append(s.calls, [2]string{agentType, role})
	return s.prompt, s.err
}

type jobRequirementProvider struct {
	content string
	model   string
	err     error
	system  string
	user    string
	calls   int
}

func (p *jobRequirementProvider) CompleteStructured(_ context.Context, systemPrompt, userPrompt string) (StructuredCompletionResult, error) {
	p.calls++
	p.system = systemPrompt
	p.user = userPrompt
	if p.err != nil {
		return StructuredCompletionResult{}, p.err
	}
	return StructuredCompletionResult{Content: p.content, ModelName: p.model}, nil
}
