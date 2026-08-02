package grpc

import (
	"context"
	"strings"
	"testing"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	domainagentskill "smart-recruit-ai-agent-service/internal/domain/agentskill"
	"smart-recruit-platform-go/observability"
)

type structuredRuntimeStore struct {
	*fakeAIStore
	result recruitingruntime.StructuredCompletionResult
	calls  int
}

func (s *structuredRuntimeStore) LoadActiveRecruitingPrompt(_ context.Context, agentType, role string) (recruitingruntime.PromptDescriptor, error) {
	return recruitingruntime.PromptDescriptor{ID: 1, Name: "structured", Version: 1, AgentType: agentType, Role: role, Content: "system"}, nil
}

func (s *structuredRuntimeStore) CompleteStructured(context.Context, string, string) (recruitingruntime.StructuredCompletionResult, error) {
	s.calls++
	return s.result, nil
}

func TestRecruitingStrictRuntimeWiresExactReleaseAndEmitsOneMetric(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantResult string
		wantReason string
		wantCalls  int
		wantError  bool
	}{
		{name: "valid", content: `{"ok":true}`, wantResult: "passed", wantReason: "strict_valid", wantCalls: 1},
		{name: "invalid after repair", content: `{}`, wantResult: "failed", wantReason: "strict_invalid", wantCalls: 2, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base := newFakeAIStore()
			document := runtimeSkillVersionDocument(
				101,
				1,
				domainagentskill.CompositionRolePrimary,
				domainagentskill.RiskLevelLow,
				"strict structured core",
			)
			document.Manifest.AgentType = recruitingruntime.AgentTypeResumeProfileExtractor
			document.Manifest.OutputContract = domainagentskill.OutputContract{
				Mode:     domainagentskill.OutputModeStrict,
				SchemaID: "resume-test-v1",
				Schema:   []byte(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"],"additionalProperties":false}`),
			}
			base.agentSkillPackages = append(base.agentSkillPackages, runtimeSkillPackage(t, document, nil))
			store := &structuredRuntimeStore{
				fakeAIStore: base,
				result:      recruitingruntime.StructuredCompletionResult{Content: test.content},
			}
			runtime := newRecruitingStructuredRuntime(store, store)
			if runtime == nil {
				t.Fatal("structured runtime was not wired")
			}
			metrics := observability.NewRegistry("ai-agent")
			meter := newNativeAIService(base, nil, nil, nil, nil)
			meter.metrics = metrics
			meter.skillPackageV2Enabled = true
			service := nativeRecruitingIntelligenceService{meter: meter}
			model := runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{})
			ctx := withRecruitingCapabilityRuntimeFeature(context.Background(), model, true)
			ctx, collector := service.withStrictValidationMetrics(ctx)

			result, err := runtime.Complete(ctx, recruitingruntime.AgentTypeResumeProfileExtractor, "private source")
			if (err != nil) != test.wantError || (!test.wantError && !result.StrictContractApplied) || store.calls != test.wantCalls {
				t.Fatalf("result=%#v error=%v calls=%d", result, err, store.calls)
			}
			service.finalizeStrictValidationMetrics(context.Background(), collector)
			output := metrics.Prometheus()
			if strings.Count(output, `smart_recruit_agent_skill_output_validation_total{`) != 1 ||
				!strings.Contains(output, `result="`+test.wantResult+`",reason="`+test.wantReason+`"`) {
				t.Fatalf("strict validation metric was not emitted exactly once:\n%s", output)
			}
		})
	}
}

func (s *structuredRuntimeStore) Complete(context.Context, string) (string, error) {
	return "", nil
}

func TestNewRecruitingStructuredRuntimeRequiresBothAdapters(t *testing.T) {
	legacy := newFakeAIStore()
	if got := newRecruitingStructuredRuntime(legacy, &fakeChatProvider{}); got != nil {
		t.Fatal("legacy provider/store unexpectedly produced structured runtime")
	}
	structured := &structuredRuntimeStore{fakeAIStore: legacy}
	if got := newRecruitingStructuredRuntime(structured, structured); got == nil {
		t.Fatal("structured prompt/provider adapters did not produce runtime")
	}
}
