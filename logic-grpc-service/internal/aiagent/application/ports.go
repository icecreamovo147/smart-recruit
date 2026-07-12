package application

import (
	"context"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

// ChatRepository is the AI Agent-owned chat session and history persistence port.
type ChatRepository interface {
	Add(ctx context.Context, history *model.AIChatHistory) error
	List(ctx context.Context, hrID int64, page, pageSize int32) ([]model.AIChatHistory, error)
	CreateSession(ctx context.Context, session *model.AIChatSession) error
	ListSessions(ctx context.Context, hrID int64, page, pageSize int32) ([]model.AIChatSession, int64, error)
	GetSessionOwned(ctx context.Context, hrID, sessionID int64) (*model.AIChatSession, error)
	ListBySession(ctx context.Context, hrID, sessionID int64, page, pageSize int32) ([]model.AIChatHistory, error)
	ListAllBySession(ctx context.Context, hrID, sessionID int64) ([]model.AIChatHistory, error)
	ListRecentBySession(ctx context.Context, hrID, sessionID int64, limit int) ([]model.AIChatHistory, error)
	CountBySession(ctx context.Context, hrID, sessionID int64) (int64, error)
	MaxMessageIDBySession(ctx context.Context, hrID, sessionID int64) (int64, error)
	UpdateUserMessageAgentSkills(ctx context.Context, hrID, sessionID, messageID int64, skillIDsJSON, skillNamesJSON string) error
	UpdateSessionTitle(ctx context.Context, hrID, sessionID int64, title string) (int64, error)
	DeleteSession(ctx context.Context, hrID, sessionID int64) (int64, error)
	AddOwned(ctx context.Context, ownerRole int32, ownerID int64, history *model.AIChatHistory) error
	CreateSessionOwned(ctx context.Context, ownerRole int32, ownerID int64, session *model.AIChatSession) error
	ListSessionsOwned(ctx context.Context, ownerRole int32, ownerID int64, page, pageSize int32) ([]model.AIChatSession, int64, error)
	GetSessionOwnedBy(ctx context.Context, ownerRole int32, ownerID, sessionID int64) (*model.AIChatSession, error)
	ListBySessionOwned(ctx context.Context, ownerRole int32, ownerID, sessionID int64, page, pageSize int32) ([]model.AIChatHistory, error)
	ListRecentBySessionOwned(ctx context.Context, ownerRole int32, ownerID, sessionID int64, limit int) ([]model.AIChatHistory, error)
	UpdateSessionTitleOwned(ctx context.Context, ownerRole int32, ownerID, sessionID int64, title string) (int64, error)
	DeleteSessionOwned(ctx context.Context, ownerRole int32, ownerID, sessionID int64) (int64, error)
}

// SessionSummaryRepository is the AI Agent-owned chat summary persistence port.
type SessionSummaryRepository interface {
	GetBySession(ctx context.Context, hrID, sessionID int64) (*model.AISessionSummary, error)
	Upsert(ctx context.Context, summary *model.AISessionSummary) error
}

// AgentRunRepository is the AI Agent-owned durable run and step persistence port.
type AgentRunRepository interface {
	CreateRun(ctx context.Context, run *model.AgentRun) error
	GetRunByID(ctx context.Context, runID uint64) (*model.AgentRun, error)
	GetRunByClientRequestID(ctx context.Context, hrID, sessionID uint64, clientRequestID string) (*model.AgentRun, error)
	GetActiveRunBySession(ctx context.Context, sessionID int64) (*model.AgentRun, error)
	SetSessionActiveRun(ctx context.Context, sessionID int64, runID *int64) error
	UpdateRunStatus(ctx context.Context, runID uint64, status, finalAnswer, errorType, errorMessage string, completedAt *time.Time) error
	UpdateRunStatusFields(ctx context.Context, runID uint64, status string, completedAt, cancelRequestedAt, canceledAt *time.Time) error
	UpdateRunSnapshot(ctx context.Context, runID uint64, patch repository.AgentRunSnapshotPatch) error
	UpdateRunPlan(ctx context.Context, runID uint64, planJSON string) error
	UpdateRunMessageID(ctx context.Context, runID, messageID uint64) error
	UpdateRunHistoryID(ctx context.Context, runID, historyID uint64) error
	CountAssistantHistoryByAgentRunID(ctx context.Context, runID uint64) (int64, error)
	CreateStep(ctx context.Context, step *model.AgentRunStep) error
	NextStepIndex(ctx context.Context, runID uint64) (int, error)
	ListRunsBySession(ctx context.Context, hrID, sessionID int64, limit int) ([]model.AgentRun, error)
	ListStepsByRunIDs(ctx context.Context, runIDs []uint64) ([]model.AgentRunStep, error)
}

// AgentRunEventRepository is the AI Agent-owned run event stream persistence port.
type AgentRunEventRepository interface {
	AppendEvent(ctx context.Context, runID uint64, eventType, payloadJSON string) (*model.AgentRunEvent, error)
	AppendEventAtSeq(ctx context.Context, runID uint64, seq int64, eventType, payloadJSON string) (*model.AgentRunEvent, bool, error)
	ListEventsAfter(ctx context.Context, runID uint64, afterSeq int64) ([]model.AgentRunEvent, error)
	GetEventBySeq(ctx context.Context, runID uint64, seq int64) (*model.AgentRunEvent, error)
}

// PromptRepository is the AI Agent-owned prompt template and version persistence port.
type PromptRepository interface {
	Create(ctx context.Context, t *model.PromptTemplate) error
	GetByID(ctx context.Context, id int64) (*model.PromptTemplate, error)
	Update(ctx context.Context, t *model.PromptTemplate) error
	UpdatePartial(ctx context.Context, id int64, updates map[string]any) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, page, pageSize int32, agentType string) ([]model.PromptTemplate, int64, error)
	GetActiveByAgentType(ctx context.Context, agentType, promptRole string) (*model.PromptTemplate, error)
	CreateVersion(ctx context.Context, v *model.PromptVersion) error
	ListVersions(ctx context.Context, templateID int64, page, pageSize int32) ([]model.PromptVersion, int64, error)
	GetVersion(ctx context.Context, templateID int64, version int32) (*model.PromptVersion, error)
}

// SkillRegistryRepository is the AI Agent-owned reusable tool Skill registry port.
type SkillRegistryRepository interface {
	CreateSkill(ctx context.Context, skill *model.Skill) error
	UpdateSkillPartial(ctx context.Context, id int64, updates map[string]any) error
	GetSkillByID(ctx context.Context, id int64) (*model.Skill, error)
	ListSkills(ctx context.Context, page, pageSize int32) ([]model.Skill, int64, error)
	CreateVersionWithTools(ctx context.Context, version *model.SkillVersion, tools []model.SkillTool, activate bool) error
	ListVersions(ctx context.Context, skillID int64) ([]model.SkillVersion, error)
	GetVersionByID(ctx context.Context, versionID int64) (*model.SkillVersion, error)
	ListTools(ctx context.Context, skillVersionID int64, enabledOnly bool) ([]model.SkillTool, error)
	ListEnabledCurrentSkillTools(ctx context.Context) ([]repository.SkillToolWithSkill, error)
}

// AgentSkillRepository is the AI Agent-owned agent-specific Skill port.
type AgentSkillRepository interface {
	CreateSkill(ctx context.Context, skill *model.AgentSkill) error
	CreateSkillWithVersion(ctx context.Context, skill *model.AgentSkill, version *model.AgentSkillVersion, activate bool) error
	UpdateSkillPartial(ctx context.Context, id int64, updates map[string]any) error
	GetSkillByID(ctx context.Context, id int64) (*model.AgentSkill, error)
	ListSkills(ctx context.Context, page, pageSize int32, enabledOnly bool, keyword string) ([]model.AgentSkill, int64, error)
	ListAvailable(ctx context.Context, page, pageSize int32) ([]model.AgentSkill, int64, error)
	ListEnabled(ctx context.Context) ([]repository.AgentSkillRuntimeRecord, error)
	CreateVersion(ctx context.Context, version *model.AgentSkillVersion, activate bool, actorUserID *int64) error
	ListVersions(ctx context.Context, skillID int64) ([]model.AgentSkillVersion, error)
	ActivateVersion(ctx context.Context, skillID, versionID int64, actorUserID *int64) error
}

// MemoryRepository is the AI Agent-owned long-term memory persistence port.
type MemoryRepository interface {
	Create(ctx context.Context, memory *model.AIMemory) error
	ListRelevant(ctx context.Context, hrID int64, scopeType string, scopeID int64, memoryTypes []string, limit int) ([]model.AIMemory, error)
	ListByHR(ctx context.Context, hrID int64, limit int) ([]model.AIMemory, error)
	ListRecallCandidates(ctx context.Context, hrID int64, scopes []repository.MemoryRecallScope, memoryTypes []string, limit int) ([]model.AIMemory, error)
}

// EmbeddingRepository is the AI Agent-owned semantic embedding persistence port.
type EmbeddingRepository interface {
	Upsert(ctx context.Context, row *model.AIEmbedding) error
	GetByObject(ctx context.Context, objectType string, objectID uint64, modelName string) (*model.AIEmbedding, error)
	ListByObjectIDs(ctx context.Context, objectType string, objectIDs []uint64, modelName string, status string) ([]model.AIEmbedding, error)
	ListCandidates(ctx context.Context, query repository.AIEmbeddingQuery) ([]model.AIEmbedding, error)
	MarkStatusByObject(ctx context.Context, objectType string, objectID uint64, status string) (int64, error)
}

// MCPGovernanceRepository is the AI Agent-owned MCP server, policy, and audit-log port.
type MCPGovernanceRepository interface {
	CreateServer(ctx context.Context, s *model.MCPServer) error
	UpdateServer(ctx context.Context, s *model.MCPServer) error
	UpdateServerPartial(ctx context.Context, id int64, updates map[string]any) error
	DeleteServer(ctx context.Context, id int64) error
	GetServerByID(ctx context.Context, id int64) (*model.MCPServer, error)
	ListServers(ctx context.Context, page, pageSize int32) ([]model.MCPServer, int64, error)
	ListEnabledServers(ctx context.Context) ([]model.MCPServer, error)
	CreateToolPolicy(ctx context.Context, policy *model.MCPToolPolicy) error
	UpdateToolPolicy(ctx context.Context, policy *model.MCPToolPolicy) error
	DeleteToolPolicy(ctx context.Context, id int64) error
	GetToolPolicyByID(ctx context.Context, id int64) (*model.MCPToolPolicy, error)
	GetEnabledToolPolicy(ctx context.Context, serverID int64, toolName string) (*model.MCPToolPolicy, error)
	ListToolPolicies(ctx context.Context, serverID int64, page, pageSize int32) ([]model.MCPToolPolicy, int64, error)
	CreateToolLog(ctx context.Context, log *model.MCPToolLog) error
	CountToolLogsSince(ctx context.Context, serverID int64, toolName string, since time.Time) (int64, error)
	ListToolLogsByServer(ctx context.Context, serverID int64, page, pageSize int32) ([]model.MCPToolLog, int64, error)
	ListToolLogsBySession(ctx context.Context, sessionID int64, page, pageSize int32) ([]model.MCPToolLog, int64, error)
}

// AgentConfigRepository is the AI Agent-owned runtime configuration and capability binding port.
type AgentConfigRepository interface {
	GetByAgentType(ctx context.Context, agentType string) (*model.AgentConfig, error)
	ListEnabledCapabilityBindingsByAgentType(ctx context.Context, agentType string) ([]model.AgentCapabilityBinding, error)
	GetAgentConfigWithBindings(ctx context.Context, agentType string) (*model.AgentConfig, []model.AgentToolBinding, error)
}

// EmbeddingProviderRepository is the AI Agent-owned embedding provider configuration port.
type EmbeddingProviderRepository interface {
	ListEnabled(ctx context.Context) ([]model.EmbeddingProviderConfig, error)
}

// EmbeddingModelRepository is the AI Agent-owned embedding model configuration port.
type EmbeddingModelRepository interface {
	GetDefaultModel(ctx context.Context) (*model.EmbeddingModelConfig, error)
	GetEnabledModelsByProvider(ctx context.Context, providerID int64) ([]model.EmbeddingModelConfig, error)
}

// UsageLogRepository is the AI Agent-owned third-party AI usage audit log port.
type UsageLogRepository interface {
	Create(ctx context.Context, log *model.ThirdPartyUsageLog) error
	List(ctx context.Context, filter repository.UsageLogFilter, page, pageSize int) ([]model.ThirdPartyUsageLog, int64, error)
}

// UsageAuditContextRepository is the AI Agent-owned authorization context audit port.
type UsageAuditContextRepository interface {
	Create(ctx context.Context, c *model.AIUsageAuthContext) error
	CreateWithTx(ctx context.Context, tx *gorm.DB, c *model.AIUsageAuthContext) error
	ListByActor(ctx context.Context, actorUserID uint64, page, pageSize int) ([]model.AIUsageAuthContext, int64, error)
	ListByPermission(ctx context.Context, permissionKey string, page, pageSize int) ([]model.AIUsageAuthContext, int64, error)
}
