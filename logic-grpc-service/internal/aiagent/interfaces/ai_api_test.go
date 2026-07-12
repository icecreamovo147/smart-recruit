package interfaces

import (
	"testing"

	"logic-grpc-service/service"
)

func TestCurrentAIServicesSatisfyAIAgentAPIs(t *testing.T) {
	t.Parallel()

	var _ HRChatAPI = (*service.AIService)(nil)
	var _ AgentRunAPI = (*service.AIService)(nil)
	var _ CandidateChatAPI = (*service.CandidateAIService)(nil)
	var _ PromptAPI = (*service.PromptService)(nil)
	var _ AgentConfigAPI = (*service.AgentConfigService)(nil)
	var _ SkillAPI = (*service.SkillService)(nil)
	var _ AgentSkillAPI = (*service.AgentSkillService)(nil)
	var _ MCPAPI = (*service.MCPService)(nil)
	var _ EmbeddingConfigAPI = (*service.EmbeddingConfigService)(nil)
	var _ LLMConfigAPI = (*service.LlmConfigService)(nil)
	var _ UsageAuditAPI = (*service.UsageStatsService)(nil)
}
