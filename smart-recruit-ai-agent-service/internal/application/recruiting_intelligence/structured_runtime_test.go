package recruiting_intelligence

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

type runtimePromptStore struct {
	prompt PromptDescriptor
	calls  [][2]string
	err    error
}

func (s *runtimePromptStore) LoadActiveRecruitingPrompt(_ context.Context, agentType, role string) (PromptDescriptor, error) {
	s.calls = append(s.calls, [2]string{agentType, role})
	return s.prompt, s.err
}

type runtimeProvider struct {
	system       string
	user         string
	err          error
	withoutUsage bool
}

func (p *runtimeProvider) CompleteStructured(_ context.Context, systemPrompt, userPrompt string) (StructuredCompletionResult, error) {
	p.system = systemPrompt
	p.user = userPrompt
	result := StructuredCompletionResult{Content: `{"ok":true}`, ProviderKey: "provider-a", ModelName: "model-a"}
	if !p.withoutUsage {
		result.TokenUsage = &schema.TokenUsage{PromptTokens: 12, CompletionTokens: 3, TotalTokens: 15}
	}
	return result, p.err
}

func TestRuntimeCollectsEstimatedUsageWhenProviderOmitsTokenCounts(t *testing.T) {
	store := &runtimePromptStore{prompt: PromptDescriptor{ID: 10, Name: "resume", Version: 1, AgentType: AgentTypeResumeProfileExtractor, Role: PromptRoleSystem, Content: "system"}}
	runtime := NewRuntime(NewPromptLoader(store), &runtimeProvider{withoutUsage: true})
	ctx, collector := WithBillingUsageCollector(context.Background())
	if _, err := runtime.Complete(ctx, AgentTypeResumeProfileExtractor, "user"); err != nil {
		t.Fatal(err)
	}
	usages := collector.Usages()
	if len(usages) != 1 || usages[0].TokenUsage != nil || usages[0].EstimatedInputTokens <= 0 || usages[0].EstimatedOutputTokens <= 0 {
		t.Fatalf("usages=%+v", usages)
	}
}

func TestRuntimeCollectsStructuredProviderUsageForBilling(t *testing.T) {
	store := &runtimePromptStore{prompt: PromptDescriptor{ID: 10, Name: "resume", Version: 1, AgentType: AgentTypeResumeProfileExtractor, Role: PromptRoleSystem, Content: "system"}}
	runtime := NewRuntime(NewPromptLoader(store), &runtimeProvider{})
	ctx, collector := WithBillingUsageCollector(context.Background())
	result, err := runtime.Complete(ctx, AgentTypeResumeProfileExtractor, "user")
	if err != nil {
		t.Fatal(err)
	}
	usages := collector.Usages()
	if result.ProviderKey != "provider-a" || result.TokenUsage == nil || len(usages) != 1 {
		t.Fatalf("result=%+v usages=%+v", result, usages)
	}
	if usages[0].ProviderKey != "provider-a" || usages[0].ModelName != "model-a" || usages[0].TokenUsage.TotalTokens != 15 {
		t.Fatalf("usage=%+v", usages[0])
	}
}

func TestRuntimeReloadsExactActiveSystemPromptPerRequest(t *testing.T) {
	store := &runtimePromptStore{prompt: PromptDescriptor{ID: 10, Name: "resume-v1", Version: 1, AgentType: AgentTypeResumeProfileExtractor, Role: PromptRoleSystem, Content: "system-v1"}}
	provider := &runtimeProvider{}
	runtime := NewRuntime(NewPromptLoader(store), provider)

	first, err := runtime.Complete(context.Background(), AgentTypeResumeProfileExtractor, "user-one")
	if err != nil {
		t.Fatalf("first Complete: %v", err)
	}
	store.prompt = PromptDescriptor{ID: 11, Name: "resume-v2", Version: 2, AgentType: AgentTypeResumeProfileExtractor, Role: PromptRoleSystem, Content: "system-v2"}
	second, err := runtime.Complete(context.Background(), AgentTypeResumeProfileExtractor, "user-two")
	if err != nil {
		t.Fatalf("second Complete: %v", err)
	}
	if first.Prompt.Version != 1 || second.Prompt.Version != 2 || provider.system != "system-v2" || provider.user != "user-two" {
		t.Fatalf("first/second/provider = %#v/%#v/%#v", first, second, provider)
	}
	if len(store.calls) != 2 {
		t.Fatalf("prompt calls = %d, want one per request", len(store.calls))
	}
	for _, call := range store.calls {
		if call != [2]string{AgentTypeResumeProfileExtractor, PromptRoleSystem} {
			t.Fatalf("prompt lookup = %#v", call)
		}
	}
}

func TestRuntimeRejectsInvalidPromptAndSanitizesProviderError(t *testing.T) {
	store := &runtimePromptStore{prompt: PromptDescriptor{ID: 10, AgentType: AgentTypeResumeProfileExtractor, Role: "user", Content: "secret prompt body"}}
	runtime := NewRuntime(NewPromptLoader(store), &runtimeProvider{})
	_, err := runtime.Complete(context.Background(), AgentTypeResumeProfileExtractor, "secret user body")
	if !errors.Is(err, ErrPromptInvalid) {
		t.Fatalf("invalid prompt error = %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaked prompt body: %v", err)
	}

	store.prompt.Role = PromptRoleSystem
	provider := &runtimeProvider{err: errors.New("provider included private model output")}
	runtime = NewRuntime(NewPromptLoader(store), provider)
	_, err = runtime.Complete(context.Background(), AgentTypeResumeProfileExtractor, "secret user body")
	var runtimeErr *RuntimeError
	if !errors.As(err, &runtimeErr) || runtimeErr.Kind != ErrorKindProvider {
		t.Fatalf("provider error = %#v", err)
	}
	if strings.Contains(err.Error(), "private model output") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaked provider/message body: %v", err)
	}
}
