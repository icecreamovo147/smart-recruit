package recruiting_intelligence

import (
	"context"
	"errors"
	"strings"
	"testing"
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
	system string
	user   string
	err    error
}

func (p *runtimeProvider) CompleteStructured(_ context.Context, systemPrompt, userPrompt string) (StructuredCompletionResult, error) {
	p.system = systemPrompt
	p.user = userPrompt
	return StructuredCompletionResult{Content: `{"ok":true}`, ModelName: "model-a"}, p.err
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
