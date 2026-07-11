package service

import (
	"context"
	"testing"
)

func TestNewAIAgentRuntimeWiresComponents(t *testing.T) {
	policy := DefaultAgentRuntimePolicy()
	runtime := NewAIAgentRuntime(AIAgentRuntimeDeps{
		AI:            &AIService{},
		CandidateAI:   &CandidateAIService{},
		LlmConfig:     &LlmConfigService{},
		Embedding:     &EmbeddingService{},
		RuntimePolicy: policy,
		RuntimeName:   "adk",
	})

	if runtime.AI == nil {
		t.Fatal("AI service is nil")
	}
	if runtime.CandidateAI == nil {
		t.Fatal("CandidateAI service is nil")
	}
	if runtime.LlmConfig == nil {
		t.Fatal("LlmConfig service is nil")
	}
	if runtime.Embedding == nil {
		t.Fatal("Embedding service is nil")
	}
	if runtime.RuntimeName != "adk" {
		t.Fatalf("RuntimeName = %q, want adk", runtime.RuntimeName)
	}
}

func TestAIAgentRuntimeStartRejectsNilRuntime(t *testing.T) {
	var runtime *AIAgentRuntime
	errs := runtime.Start(context.Background(), nil)
	if len(errs) != 1 || errs[0].Component != "ai-agent-runtime" {
		t.Fatalf("unexpected errors: %+v", errs)
	}
}

func TestAIAgentRuntimeStartRejectsNilMQ(t *testing.T) {
	runtime := &AIAgentRuntime{}
	errs := runtime.Start(context.Background(), nil)
	if len(errs) != 1 || errs[0].Component != "ai-agent-runtime-mq" {
		t.Fatalf("unexpected errors: %+v", errs)
	}
}
