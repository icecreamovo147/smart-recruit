package service

import (
	"context"
	"testing"
)

func TestNewAIAgentRuntimeWiresComponents(t *testing.T) {
	policy := DefaultAgentRuntimePolicy()
	runtime := NewAIAgentRuntime(AIAgentRuntimeDeps{
		AI:              &AIService{},
		CandidateAI:     &CandidateAIService{},
		LlmConfig:       &LlmConfigService{},
		Prompt:          &PromptService{},
		AgentConfig:     &AgentConfigService{},
		MCP:             &MCPService{},
		Skill:           &SkillService{},
		AgentSkill:      &AgentSkillService{},
		ResumeProfile:   &ResumeProfileService{},
		CandidateMatch:  &CandidateMatchService{},
		Intelligence:    &RecruitingIntelligenceService{},
		Embedding:       &EmbeddingService{},
		EmbeddingConfig: &EmbeddingConfigService{},
		RuntimePolicy:   policy,
		RuntimeName:     "adk",
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
	if runtime.Prompt == nil {
		t.Fatal("Prompt service is nil")
	}
	if runtime.AgentConfig == nil {
		t.Fatal("AgentConfig service is nil")
	}
	if runtime.MCP == nil {
		t.Fatal("MCP service is nil")
	}
	if runtime.Skill == nil {
		t.Fatal("Skill service is nil")
	}
	if runtime.AgentSkill == nil {
		t.Fatal("AgentSkill service is nil")
	}
	if runtime.ResumeProfile == nil {
		t.Fatal("ResumeProfile service is nil")
	}
	if runtime.CandidateMatch == nil {
		t.Fatal("CandidateMatch service is nil")
	}
	if runtime.Intelligence == nil {
		t.Fatal("RecruitingIntelligence service is nil")
	}
	if runtime.Embedding == nil {
		t.Fatal("Embedding service is nil")
	}
	if runtime.EmbeddingConfig == nil {
		t.Fatal("EmbeddingConfig service is nil")
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
