package service

import (
	"context"
	"errors"

	"smart-recruit-domain-go/mq"
)

var errNilAIAgentRuntime = errors.New("ai agent runtime is nil")
var errNilAIAgentMQ = errors.New("ai agent runtime mq connection is nil")

type AIAgentRuntime struct {
	AI                *AIService
	CandidateAI       *CandidateAIService
	LlmConfig         *LlmConfigService
	Prompt            *PromptService
	AgentConfig       *AgentConfigService
	MCP               *MCPService
	Skill             *SkillService
	AgentSkill        *AgentSkillService
	ResumeProfile     *ResumeProfileService
	CandidateMatch    *CandidateMatchService
	Intelligence      *RecruitingIntelligenceService
	Embedding         *EmbeddingService
	EmbeddingConfig   *EmbeddingConfigService
	EmbeddingConsumer *EmbeddingConsumer
	AgentRunConsumer  *AgentRunConsumer
	RuntimePolicy     AgentRuntimePolicy
	RuntimeName       string
}

type AIAgentRuntimeDeps struct {
	AI                *AIService
	CandidateAI       *CandidateAIService
	LlmConfig         *LlmConfigService
	Prompt            *PromptService
	AgentConfig       *AgentConfigService
	MCP               *MCPService
	Skill             *SkillService
	AgentSkill        *AgentSkillService
	ResumeProfile     *ResumeProfileService
	CandidateMatch    *CandidateMatchService
	Intelligence      *RecruitingIntelligenceService
	Embedding         *EmbeddingService
	EmbeddingConfig   *EmbeddingConfigService
	EmbeddingConsumer *EmbeddingConsumer
	AgentRunConsumer  *AgentRunConsumer
	RuntimePolicy     AgentRuntimePolicy
	RuntimeName       string
}

type AIAgentRuntimeStartError struct {
	Component string
	Err       error
}

func NewAIAgentRuntime(deps AIAgentRuntimeDeps) *AIAgentRuntime {
	return &AIAgentRuntime{
		AI:                deps.AI,
		CandidateAI:       deps.CandidateAI,
		LlmConfig:         deps.LlmConfig,
		Prompt:            deps.Prompt,
		AgentConfig:       deps.AgentConfig,
		MCP:               deps.MCP,
		Skill:             deps.Skill,
		AgentSkill:        deps.AgentSkill,
		ResumeProfile:     deps.ResumeProfile,
		CandidateMatch:    deps.CandidateMatch,
		Intelligence:      deps.Intelligence,
		Embedding:         deps.Embedding,
		EmbeddingConfig:   deps.EmbeddingConfig,
		EmbeddingConsumer: deps.EmbeddingConsumer,
		AgentRunConsumer:  deps.AgentRunConsumer,
		RuntimePolicy:     deps.RuntimePolicy,
		RuntimeName:       deps.RuntimeName,
	}
}

func (r *AIAgentRuntime) Start(ctx context.Context, mqConn *mq.Conn) []AIAgentRuntimeStartError {
	if r == nil {
		return []AIAgentRuntimeStartError{{Component: "ai-agent-runtime", Err: errNilAIAgentRuntime}}
	}
	var errs []AIAgentRuntimeStartError
	if mqConn == nil {
		return append(errs, AIAgentRuntimeStartError{Component: "ai-agent-runtime-mq", Err: errNilAIAgentMQ})
	}
	if r.EmbeddingConsumer != nil {
		if err := r.EmbeddingConsumer.Start(ctx, mqConn); err != nil {
			errs = append(errs, AIAgentRuntimeStartError{Component: "embedding-consumer", Err: err})
		}
	}
	if r.AgentRunConsumer != nil {
		if err := r.AgentRunConsumer.Start(ctx, mqConn); err != nil {
			errs = append(errs, AIAgentRuntimeStartError{Component: "agent-run-consumer", Err: err})
		}
	}
	return errs
}
