package application

import (
	"testing"

	"logic-grpc-service/repository"
)

func TestCurrentRepositoriesSatisfyAIAgentPorts(t *testing.T) {
	t.Parallel()

	var _ ChatRepository = (*repository.ChatRepo)(nil)
	var _ SessionSummaryRepository = (*repository.SessionSummaryRepo)(nil)
	var _ AgentRunRepository = (*repository.AgentRunRepo)(nil)
	var _ AgentRunEventRepository = (*repository.AgentRunEventRepo)(nil)
	var _ PromptRepository = (*repository.PromptTemplateRepo)(nil)
	var _ SkillRegistryRepository = (*repository.SkillRepo)(nil)
	var _ AgentSkillRepository = (*repository.AgentSkillRepo)(nil)
	var _ MemoryRepository = (*repository.MemoryRepo)(nil)
	var _ EmbeddingRepository = (*repository.AIEmbeddingRepo)(nil)
	var _ MCPGovernanceRepository = (*repository.MCPRepo)(nil)
	var _ AgentConfigRepository = (*repository.AgentConfigRepo)(nil)
	var _ EmbeddingProviderRepository = (*repository.EmbeddingProviderRepo)(nil)
	var _ EmbeddingModelRepository = (*repository.EmbeddingModelRepo)(nil)
	var _ UsageLogRepository = (*repository.UsageLogRepo)(nil)
	var _ UsageAuditContextRepository = (*repository.UsageAuditContextRepo)(nil)
}
