package grpc

import (
	"context"
	"testing"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
)

type structuredRuntimeStore struct {
	*fakeAIStore
}

func (s *structuredRuntimeStore) LoadActiveRecruitingPrompt(context.Context, string, string) (recruitingruntime.PromptDescriptor, error) {
	return recruitingruntime.PromptDescriptor{}, nil
}

func (s *structuredRuntimeStore) CompleteStructured(context.Context, string, string) (recruitingruntime.StructuredCompletionResult, error) {
	return recruitingruntime.StructuredCompletionResult{}, nil
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
