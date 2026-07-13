package repository

import (
	"context"
	"time"

	"smart-recruit-ai-agent-service/internal/domain/model"
)

type MCPPolicyRepository interface {
	GetEnabledToolPolicy(ctx context.Context, serverID uint64, toolName string) (*model.MCPToolPolicy, error)
	CountToolCallsSince(ctx context.Context, serverID uint64, toolName string, since time.Time) (int64, error)
}

type SkillRepository interface {
	ListEnabledAgentSkills(ctx context.Context) ([]model.AgentSkill, error)
	CreateSkillVersion(ctx context.Context, version model.SkillVersion) (*model.SkillVersion, error)
	ActivateSkillVersion(ctx context.Context, skillID uint64, version int64) error
}

type EmbeddingConfigRepository interface {
	GetDefaultEmbeddingProvider(ctx context.Context) (*model.EmbeddingProviderConfig, error)
	GetDefaultEmbeddingModel(ctx context.Context, providerID uint64) (*model.EmbeddingModelConfig, error)
}

type CandidateMatchRepository interface {
	SaveCandidateMatchAggregation(ctx context.Context, applicationID uint64, aggregation model.CandidateMatchAggregation) error
}
