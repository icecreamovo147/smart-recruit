package infrastructure

import "logic-grpc-service/repository"

// ChatRepository adapts the current chat repository to AI Agent application ports.
type ChatRepository = repository.ChatRepo

// SessionSummaryRepository adapts the current session summary repository to AI Agent application ports.
type SessionSummaryRepository = repository.SessionSummaryRepo

// AgentRunRepository adapts the current durable agent run repository to AI Agent application ports.
type AgentRunRepository = repository.AgentRunRepo

// AgentRunEventRepository adapts the current durable event stream repository to AI Agent application ports.
type AgentRunEventRepository = repository.AgentRunEventRepo

// PromptRepository adapts current prompt template persistence to AI Agent application ports.
type PromptRepository = repository.PromptTemplateRepo

// SkillRegistryRepository adapts the current reusable Skill registry to AI Agent application ports.
type SkillRegistryRepository = repository.SkillRepo

// AgentSkillRepository adapts current agent-specific Skill persistence to AI Agent application ports.
type AgentSkillRepository = repository.AgentSkillRepo

// MemoryRepository adapts current long-term memory persistence to AI Agent application ports.
type MemoryRepository = repository.MemoryRepo

// EmbeddingRepository adapts current semantic embedding persistence to AI Agent application ports.
type EmbeddingRepository = repository.AIEmbeddingRepo

// MCPGovernanceRepository adapts current MCP server, policy, and audit persistence to AI Agent application ports.
type MCPGovernanceRepository = repository.MCPRepo

// AgentConfigRepository adapts current Agent configuration persistence to AI Agent application ports.
type AgentConfigRepository = repository.AgentConfigRepo

// EmbeddingProviderRepository adapts current embedding provider configuration persistence.
type EmbeddingProviderRepository = repository.EmbeddingProviderRepo

// EmbeddingModelRepository adapts current embedding model configuration persistence.
type EmbeddingModelRepository = repository.EmbeddingModelRepo

// UsageLogRepository adapts current third-party usage logs to AI Agent audit ports.
type UsageLogRepository = repository.UsageLogRepo

// UsageAuditContextRepository adapts current usage authorization context persistence to AI Agent audit ports.
type UsageAuditContextRepository = repository.UsageAuditContextRepo
