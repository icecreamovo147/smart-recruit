package infrastructure

import (
	"reflect"
	"testing"

	"logic-grpc-service/repository"
)

func TestAIAgentAdaptersUseCurrentRepositoryTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  reflect.Type
		want reflect.Type
	}{
		{"chat", reflect.TypeOf((*ChatRepository)(nil)), reflect.TypeOf((*repository.ChatRepo)(nil))},
		{"summary", reflect.TypeOf((*SessionSummaryRepository)(nil)), reflect.TypeOf((*repository.SessionSummaryRepo)(nil))},
		{"run", reflect.TypeOf((*AgentRunRepository)(nil)), reflect.TypeOf((*repository.AgentRunRepo)(nil))},
		{"run_event", reflect.TypeOf((*AgentRunEventRepository)(nil)), reflect.TypeOf((*repository.AgentRunEventRepo)(nil))},
		{"prompt", reflect.TypeOf((*PromptRepository)(nil)), reflect.TypeOf((*repository.PromptTemplateRepo)(nil))},
		{"skill", reflect.TypeOf((*SkillRegistryRepository)(nil)), reflect.TypeOf((*repository.SkillRepo)(nil))},
		{"agent_skill", reflect.TypeOf((*AgentSkillRepository)(nil)), reflect.TypeOf((*repository.AgentSkillRepo)(nil))},
		{"memory", reflect.TypeOf((*MemoryRepository)(nil)), reflect.TypeOf((*repository.MemoryRepo)(nil))},
		{"embedding", reflect.TypeOf((*EmbeddingRepository)(nil)), reflect.TypeOf((*repository.AIEmbeddingRepo)(nil))},
		{"mcp", reflect.TypeOf((*MCPGovernanceRepository)(nil)), reflect.TypeOf((*repository.MCPRepo)(nil))},
		{"agent_config", reflect.TypeOf((*AgentConfigRepository)(nil)), reflect.TypeOf((*repository.AgentConfigRepo)(nil))},
		{"embedding_provider", reflect.TypeOf((*EmbeddingProviderRepository)(nil)), reflect.TypeOf((*repository.EmbeddingProviderRepo)(nil))},
		{"embedding_model", reflect.TypeOf((*EmbeddingModelRepository)(nil)), reflect.TypeOf((*repository.EmbeddingModelRepo)(nil))},
		{"usage_log", reflect.TypeOf((*UsageLogRepository)(nil)), reflect.TypeOf((*repository.UsageLogRepo)(nil))},
		{"usage_audit", reflect.TypeOf((*UsageAuditContextRepository)(nil)), reflect.TypeOf((*repository.UsageAuditContextRepo)(nil))},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Fatalf("%s adapter drifted from current repository: got %v want %v", tc.name, tc.got, tc.want)
		}
	}
}
