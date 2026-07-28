package recruiting_intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/eino/schema"

	"smart-recruit-ai-agent-service/internal/domain/agentskill"
)

func TestStrictRuntimeExactReleasedContractForAllStructuredAgents(t *testing.T) {
	agentTypes := []string{
		AgentTypeResumeProfileExtractor,
		AgentTypeJobRequirementExtractor,
		AgentTypeCandidateMatchEvaluator,
	}
	for _, agentType := range agentTypes {
		t.Run(agentType, func(t *testing.T) {
			runtimePackage := strictRuntimePackage(t, 101, agentType, agentskill.RiskLevelLow, strictTestSchema())
			provider := &strictSequenceProvider{results: []StructuredCompletionResult{{
				Content: `{"ok":true}`, ProviderKey: "provider-a", ModelName: "model-a",
				TokenUsage: &schema.TokenUsage{PromptTokens: 10, CompletionTokens: 3, TotalTokens: 13},
			}}}
			runtime := strictTestRuntime(provider, strictPackageLoader{packages: []ReleasedAgentSkillPackage{runtimePackage}})
			ctx := strictCapabilityContext(context.Background(), true, []int64{101})
			ctx, validation := WithStrictValidationCollector(ctx)
			ctx, billing := WithBillingUsageCollector(ctx)

			result, err := runtime.Complete(ctx, agentType, "PRIVATE_SOURCE")
			if err != nil {
				t.Fatalf("Complete() error = %v", err)
			}
			if !result.StrictContractApplied || result.Content != `{"ok":true}` {
				t.Fatalf("result = %#v", result)
			}
			if provider.CallCount() != 1 || !strings.Contains(provider.SystemPrompts()[0], `"required":["ok"]`) {
				t.Fatalf("provider calls=%d system=%q", provider.CallCount(), provider.SystemPrompts())
			}
			assertStrictOutcomes(t, validation.Outcomes(), "passed", "strict_valid")
			if usages := billing.Usages(); len(usages) != 1 || usages[0].TokenUsage.TotalTokens != 13 {
				t.Fatalf("billing usages = %#v", usages)
			}
		})
	}
}

func TestStrictRuntimeRepairsAtMostOnceAndBillsBothCalls(t *testing.T) {
	runtimePackage := strictRuntimePackage(t, 101, AgentTypeResumeProfileExtractor, agentskill.RiskLevelMedium, strictTestSchema())
	provider := &strictSequenceProvider{results: []StructuredCompletionResult{
		{Content: "```json\n{\"ok\":\"PRIVATE_BAD_VALUE\"}\n```", ProviderKey: "provider-a", ModelName: "model-a", TokenUsage: &schema.TokenUsage{TotalTokens: 4}},
		{Content: `{"ok":true}`, ProviderKey: "provider-a", ModelName: "model-a", TokenUsage: &schema.TokenUsage{TotalTokens: 6}},
	}}
	runtime := strictTestRuntime(provider, strictPackageLoader{packages: []ReleasedAgentSkillPackage{runtimePackage}})
	ctx := strictCapabilityContext(context.Background(), true, []int64{101})
	ctx, validation := WithStrictValidationCollector(ctx)
	ctx, billing := WithBillingUsageCollector(ctx)

	result, err := runtime.Complete(ctx, AgentTypeResumeProfileExtractor, "PRIVATE_SOURCE")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if result.Content != `{"ok":true}` || provider.CallCount() != 2 {
		t.Fatalf("result=%#v calls=%d", result, provider.CallCount())
	}
	prompts := provider.UserPrompts()
	if len(prompts) != 2 || !strings.Contains(prompts[1], "Previous response") || strings.Contains(prompts[1], "PRIVATE_SOURCE") {
		t.Fatalf("repair prompt = %q", prompts)
	}
	assertStrictOutcomes(t, validation.Outcomes(), "passed", "strict_valid")
	usages := billing.Usages()
	if len(usages) != 2 || usages[0].TokenUsage.TotalTokens != 4 || usages[1].TokenUsage.TotalTokens != 6 {
		t.Fatalf("billing usages = %#v", usages)
	}
}

func TestStrictRuntimeFinalInvalidFailsClosedWithoutBodyLeak(t *testing.T) {
	runtimePackage := strictRuntimePackage(t, 101, AgentTypeResumeProfileExtractor, agentskill.RiskLevelLow, strictTestSchema())
	provider := &strictSequenceProvider{results: []StructuredCompletionResult{
		{Content: "PRIVATE_FIRST_INVALID"},
		{Content: `{"ok":"PRIVATE_SECOND_INVALID"}`},
		{Content: `{"ok":true}`},
	}}
	runtime := strictTestRuntime(provider, strictPackageLoader{packages: []ReleasedAgentSkillPackage{runtimePackage}})
	ctx := strictCapabilityContext(context.Background(), true, []int64{101})
	ctx, validation := WithStrictValidationCollector(ctx)

	_, err := runtime.Complete(ctx, AgentTypeResumeProfileExtractor, "PRIVATE_SOURCE")
	if err == nil || !IsStrictOutputError(err) {
		t.Fatalf("Complete() error = %v", err)
	}
	if provider.CallCount() != 2 {
		t.Fatalf("provider calls = %d, want 2", provider.CallCount())
	}
	for _, secret := range []string{"PRIVATE_FIRST_INVALID", "PRIVATE_SECOND_INVALID", "PRIVATE_SOURCE", "ok", "strict-test-v1"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked %q: %v", secret, err)
		}
	}
	assertStrictOutcomes(t, validation.Outcomes(), "failed", "strict_invalid")
}

func TestStrictRuntimeProviderErrorRemainsProviderErrorAndDoesNotRepair(t *testing.T) {
	runtimePackage := strictRuntimePackage(t, 101, AgentTypeResumeProfileExtractor, agentskill.RiskLevelLow, strictTestSchema())
	provider := &strictSequenceProvider{
		results: []StructuredCompletionResult{{Content: `{"ok":true}`}},
		errs:    []error{errors.New("PRIVATE_PROVIDER_FAILURE")},
	}
	runtime := strictTestRuntime(provider, strictPackageLoader{packages: []ReleasedAgentSkillPackage{runtimePackage}})
	ctx := strictCapabilityContext(context.Background(), true, []int64{101})
	ctx, validation := WithStrictValidationCollector(ctx)

	_, err := runtime.Complete(ctx, AgentTypeResumeProfileExtractor, "PRIVATE_SOURCE")
	var runtimeErr *RuntimeError
	if !errors.As(err, &runtimeErr) || runtimeErr.Kind != ErrorKindProvider || !runtimeErr.StrictContractApplied {
		t.Fatalf("Complete() error = %#v", err)
	}
	if provider.CallCount() != 1 || strings.Contains(err.Error(), "PRIVATE_PROVIDER_FAILURE") {
		t.Fatalf("provider calls=%d error=%v", provider.CallCount(), err)
	}
	assertStrictOutcomes(t, validation.Outcomes(), "failed", "strict_invalid")
}

func TestStrictResolverFailsClosedBeforeProvider(t *testing.T) {
	valid := strictRuntimePackage(t, 101, AgentTypeResumeProfileExtractor, agentskill.RiskLevelLow, strictTestSchema())
	second := strictRuntimePackage(t, 102, AgentTypeResumeProfileExtractor, agentskill.RiskLevelMedium, strictTestSchema())
	high := strictRuntimePackage(t, 103, AgentTypeResumeProfileExtractor, agentskill.RiskLevelHigh, strictTestSchema())
	critical := strictRuntimePackage(t, 104, AgentTypeResumeProfileExtractor, agentskill.RiskLevelCritical, strictTestSchema())
	drifted := valid
	drifted.CompiledHash = strings.Repeat("0", 64)
	duplicateRoot := valid
	duplicateRoot.ManifestJSON = strings.Replace(
		duplicateRoot.ManifestJSON,
		`"display_name":`,
		`"display_name":"shadow","display_name":`,
		1,
	)
	duplicateNested := valid
	duplicateNested.ManifestJSON = strings.Replace(
		duplicateNested.ManifestJSON,
		`"properties":{"ok":`,
		`"properties":{"ok":{"type":"string"},"ok":`,
		1,
	)
	supportingContract := valid
	var supportingManifest agentskill.Manifest
	if err := json.Unmarshal([]byte(supportingContract.ManifestJSON), &supportingManifest); err != nil {
		t.Fatal(err)
	}
	supportingManifest.Composition.Role = agentskill.CompositionRoleSupporting
	supportingJSON, err := json.Marshal(supportingManifest)
	if err != nil {
		t.Fatal(err)
	}
	supportingContract.ManifestJSON = string(supportingJSON)

	tests := []struct {
		name     string
		ids      []int64
		packages []ReleasedAgentSkillPackage
	}{
		{name: "missing exact version", ids: []int64{101}, packages: nil},
		{name: "multiple matching primary", ids: []int64{101, 102}, packages: []ReleasedAgentSkillPackage{valid, second}},
		{name: "package hash drift", ids: []int64{101}, packages: []ReleasedAgentSkillPackage{drifted}},
		{name: "duplicate root manifest key", ids: []int64{101}, packages: []ReleasedAgentSkillPackage{duplicateRoot}},
		{name: "duplicate nested schema key", ids: []int64{101}, packages: []ReleasedAgentSkillPackage{duplicateNested}},
		{name: "high confirm denied unattended", ids: []int64{103}, packages: []ReleasedAgentSkillPackage{high}},
		{name: "critical manual denied unattended", ids: []int64{104}, packages: []ReleasedAgentSkillPackage{critical}},
		{name: "supporting output contract rejected", ids: []int64{101}, packages: []ReleasedAgentSkillPackage{supportingContract}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &strictSequenceProvider{results: []StructuredCompletionResult{{Content: `{"ok":true}`}}}
			runtime := strictTestRuntime(provider, strictPackageLoader{packages: test.packages})
			ctx := strictCapabilityContext(context.Background(), true, test.ids)
			ctx, validation := WithStrictValidationCollector(ctx)
			_, err := runtime.Complete(ctx, AgentTypeResumeProfileExtractor, "PRIVATE_SOURCE")
			if err == nil || !IsStrictOutputError(err) {
				t.Fatalf("Complete() error = %v", err)
			}
			if provider.CallCount() != 0 {
				t.Fatalf("provider calls = %d", provider.CallCount())
			}
			assertStrictOutcomes(t, validation.Outcomes(), "failed", "strict_invalid")
		})
	}
}

func TestStrictResolverNoMatchAndFeatureOffPreserveExistingBehavior(t *testing.T) {
	jobPackage := strictRuntimePackage(t, 101, AgentTypeJobRequirementExtractor, agentskill.RiskLevelLow, strictTestSchema())
	tests := []struct {
		name    string
		enabled bool
		ids     []int64
	}{
		{name: "feature off", enabled: false, ids: []int64{101}},
		{name: "no version ids", enabled: true},
		{name: "no agent type match", enabled: true, ids: []int64{101}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &strictSequenceProvider{results: []StructuredCompletionResult{{Content: "legacy structured output"}}}
			runtime := strictTestRuntime(provider, strictPackageLoader{packages: []ReleasedAgentSkillPackage{jobPackage}})
			ctx := strictCapabilityContext(context.Background(), test.enabled, test.ids)
			ctx, validation := WithStrictValidationCollector(ctx)
			result, err := runtime.Complete(ctx, AgentTypeResumeProfileExtractor, "PRIVATE_SOURCE")
			if err != nil || result.StrictContractApplied || result.Content != "legacy structured output" {
				t.Fatalf("result=%#v error=%v", result, err)
			}
			if len(validation.Outcomes()) != 0 {
				t.Fatalf("strict outcomes = %#v", validation.Outcomes())
			}
		})
	}
}

func TestStrictResolverNilLoaderFailsClosedOnlyForEnabledExactVersions(t *testing.T) {
	resolver := NewStrictContractResolver(nil)

	for _, test := range []struct {
		name    string
		enabled bool
		ids     []int64
		wantErr bool
	}{
		{name: "feature off", enabled: false, ids: []int64{101}},
		{name: "no version ids", enabled: true},
		{name: "enabled exact versions", enabled: true, ids: []int64{101}, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			contract, err := resolver.Resolve(
				strictCapabilityContext(context.Background(), test.enabled, test.ids),
				AgentTypeResumeProfileExtractor,
			)
			if test.wantErr {
				if contract != nil || !errors.Is(err, ErrStrictPackageLoaderUnavailable) || !IsStrictOutputError(err) {
					t.Fatalf("contract=%#v error=%v, want typed package-loader-unavailable error", contract, err)
				}
				return
			}
			if contract != nil || err != nil {
				t.Fatalf("contract=%#v error=%v, want unchanged legacy behavior", contract, err)
			}
		})
	}
}

func TestStrictSchemaCannotBypassDownstreamDomainValidationOrFallback(t *testing.T) {
	broadSchema := json.RawMessage(`{"type":"object"}`)
	policy := NewRuntimePolicy(RuntimePolicyConfig{
		StructuredResumeParse: true, CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: true,
	})

	t.Run("resume profile", func(t *testing.T) {
		runtimePackage := strictRuntimePackage(t, 101, AgentTypeResumeProfileExtractor, agentskill.RiskLevelLow, broadSchema)
		provider := &strictSequenceProvider{results: []StructuredCompletionResult{{Content: `{}`}}}
		runtime := strictTestRuntime(provider, strictPackageLoader{packages: []ReleasedAgentSkillPackage{runtimePackage}})
		_, err := NewResumeProfileExtractor(runtime, policy).Extract(
			strictCapabilityContext(context.Background(), true, []int64{101}),
			ResumeSource{ParsedText: "PRIVATE_RESUME Go"},
		)
		if err == nil || !IsStrictOutputError(err) || provider.CallCount() != 1 {
			t.Fatalf("error=%v calls=%d", err, provider.CallCount())
		}
	})

	t.Run("job requirement", func(t *testing.T) {
		runtimePackage := strictRuntimePackage(t, 102, AgentTypeJobRequirementExtractor, agentskill.RiskLevelLow, broadSchema)
		provider := &strictSequenceProvider{results: []StructuredCompletionResult{{Content: `{}`}}}
		runtime := strictTestRuntime(provider, strictPackageLoader{packages: []ReleasedAgentSkillPackage{runtimePackage}})
		_, err := NewJobRequirementExtractor(runtime, policy).Extract(
			strictCapabilityContext(context.Background(), true, []int64{102}),
			JobRequirementSource{JobTitle: "PRIVATE_JOB", Requirements: "Go"},
		)
		if err == nil || !IsStrictOutputError(err) || provider.CallCount() != 1 {
			t.Fatalf("error=%v calls=%d", err, provider.CallCount())
		}
	})

	t.Run("candidate matcher provenance", func(t *testing.T) {
		runtimePackage := strictRuntimePackage(t, 103, AgentTypeCandidateMatchEvaluator, agentskill.RiskLevelLow, broadSchema)
		provider := &strictSequenceProvider{results: []StructuredCompletionResult{{Content: `{}`}}}
		runtime := strictTestRuntime(provider, strictPackageLoader{packages: []ReleasedAgentSkillPackage{runtimePackage}})
		evaluator := NewCandidateRequirementEvaluator(runtime, policy)
		profile := testRequirementProfile(JobRequirementItem{
			ID: "go", Category: RequirementCategoryCoreSkill, Label: "Go 开发能力",
			Description: "要求具备 Go 开发能力。", Priority: RequirementPriorityMustHave, Weight: 1, Aliases: []string{"Go"},
		})
		_, err := evaluator.Evaluate(
			strictCapabilityContext(context.Background(), true, []int64{103}),
			profile,
			BuildEvidenceIndex(CandidateEvidenceSource{}),
		)
		if err == nil || !IsStrictOutputError(err) || provider.CallCount() != 1 {
			t.Fatalf("error=%v calls=%d", err, provider.CallCount())
		}
	})
}

func strictTestSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"],"additionalProperties":false}`)
}

func strictCapabilityContext(ctx context.Context, enabled bool, ids []int64) context.Context {
	return WithCapabilityRuntime(ctx, 1, 2, 3, "", "snapshot", []int64{10}, ids, enabled)
}

func strictTestRuntime(provider StructuredCompletionProvider, loader ReleasedAgentSkillPackageLoader) *Runtime {
	prompts := NewPromptLoader(strictPromptStore{})
	return NewRuntimeWithObserverAndStrictContracts(
		prompts,
		provider,
		nil,
		NewStrictContractResolver(loader),
		NewRuntimePolicy(RuntimePolicyConfig{StructuredResumeParse: true, CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: true}),
	)
}

type strictPromptStore struct{}

func (strictPromptStore) LoadActiveRecruitingPrompt(_ context.Context, agentType, role string) (PromptDescriptor, error) {
	return PromptDescriptor{ID: 1, Name: "structured", Version: 1, AgentType: agentType, Role: role, Content: "SYSTEM"}, nil
}

type strictPackageLoader struct {
	packages []ReleasedAgentSkillPackage
	err      error
}

func (l strictPackageLoader) LoadReleasedAgentSkillPackages(_ context.Context, _ []int64) ([]ReleasedAgentSkillPackage, error) {
	return append([]ReleasedAgentSkillPackage(nil), l.packages...), l.err
}

type strictSequenceProvider struct {
	mu      sync.Mutex
	results []StructuredCompletionResult
	errs    []error
	systems []string
	users   []string
}

func (p *strictSequenceProvider) CompleteStructured(_ context.Context, systemPrompt, userPrompt string) (StructuredCompletionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	index := len(p.systems)
	p.systems = append(p.systems, systemPrompt)
	p.users = append(p.users, userPrompt)
	if index < len(p.errs) && p.errs[index] != nil {
		return StructuredCompletionResult{}, p.errs[index]
	}
	if index >= len(p.results) {
		return StructuredCompletionResult{}, errors.New("unexpected provider call")
	}
	return p.results[index], nil
}

func (p *strictSequenceProvider) CallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.systems)
}

func (p *strictSequenceProvider) SystemPrompts() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.systems...)
}

func (p *strictSequenceProvider) UserPrompts() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.users...)
}

func strictRuntimePackage(
	t *testing.T,
	versionID int64,
	agentType string,
	risk agentskill.RiskLevel,
	outputSchema json.RawMessage,
) ReleasedAgentSkillPackage {
	t.Helper()
	compiled, err := agentskill.Compile(agentskill.PackageDraft{
		Manifest: agentskill.Manifest{
			SchemaVersion: 2, SkillName: "strict-test", DisplayName: "Strict test",
			AgentType: agentType, Priority: 10, RiskLevel: risk,
			Composition: agentskill.Composition{Role: agentskill.CompositionRolePrimary},
			OutputContract: agentskill.OutputContract{
				Mode: agentskill.OutputModeStrict, SchemaID: "strict-test-v1", Schema: outputSchema,
			},
			RequiredCapabilities: []string{}, TriggerKeywords: []string{}, SemanticTags: []string{}, EvaluationCriteria: []string{},
		},
		Core: agentskill.Core{ContentMarkdown: "Strict test core."},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return ReleasedAgentSkillPackage{
		ID: versionID, SkillID: versionID + 1000, Version: "v1",
		ManifestJSON: compiled.ManifestJSON, CoreMarkdown: compiled.Core.ContentMarkdown,
		CompiledMarkdown: compiled.CompiledMarkdown, CompiledHash: compiled.CompiledHash,
		CoreEstimatedTokens: compiled.Core.EstimatedTokens,
	}
}

func assertStrictOutcomes(t *testing.T, outcomes []StrictValidationOutcome, result, reason string) {
	t.Helper()
	if len(outcomes) != 1 || outcomes[0].Result != result || outcomes[0].Reason != reason {
		t.Fatalf("strict outcomes = %#v", outcomes)
	}
}
