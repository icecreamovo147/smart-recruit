package dto

import "smart-recruit-ai-agent-service/internal/domain/model"

type MCPPolicyResult struct {
	Evaluation model.MCPPolicyEvaluation
}

type SkillVersionResult struct {
	Version        model.SkillVersion
	VersionCreated bool
	Activated      bool
}

type EmbeddingRuntimeResult struct {
	State model.EmbeddingRuntimeState
}

type CandidateMatchResult struct {
	Aggregation model.CandidateMatchAggregation
	Persisted   bool
}
