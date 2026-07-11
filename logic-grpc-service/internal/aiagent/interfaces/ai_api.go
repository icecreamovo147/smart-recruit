package interfaces

import (
	"context"

	"logic-grpc-service/recruitment/pb"
)

// HRChatAPI is the AI Agent-owned HR chat and trace gRPC contract.
type HRChatAPI interface {
	Chat(context.Context, *pb.ChatRequest) (*pb.ChatResponse, error)
	ChatStream(*pb.ChatRequest, pb.AIService_ChatStreamServer) error
	AnalyzeApplication(context.Context, *pb.AnalyzeApplicationRequest) (*pb.AnalyzeApplicationResponse, error)
	History(context.Context, *pb.ChatHistoryRequest) (*pb.ChatHistoryResponse, error)
	ListChatSessions(context.Context, *pb.ChatSessionListRequest) (*pb.ChatSessionListResponse, error)
	CreateChatSession(context.Context, *pb.CreateChatSessionRequest) (*pb.CreateChatSessionResponse, error)
	SessionMessages(context.Context, *pb.SessionMessagesRequest) (*pb.ChatHistoryResponse, error)
	CreateApplicationAnalysisSession(context.Context, *pb.CreateApplicationAnalysisSessionRequest) (*pb.CreateApplicationAnalysisSessionResponse, error)
	UpdateSession(context.Context, *pb.UpdateSessionRequest) (*pb.CommonResponse, error)
	DeleteSession(context.Context, *pb.DeleteSessionRequest) (*pb.CommonResponse, error)
	GetToolTraces(context.Context, *pb.GetToolTracesRequest) (*pb.GetToolTracesResponse, error)
	GetAgentRuns(context.Context, *pb.GetAgentRunsRequest) (*pb.GetAgentRunsResponse, error)
}

// AgentRunAPI is the AI Agent-owned durable agent run gRPC contract.
type AgentRunAPI interface {
	CreateAgentRun(context.Context, *pb.CreateAgentRunRequest) (*pb.CreateAgentRunResponse, error)
	GetAgentRun(context.Context, *pb.GetAgentRunRequest) (*pb.GetAgentRunResponse, error)
	GetActiveAgentRun(context.Context, *pb.GetActiveAgentRunRequest) (*pb.GetActiveAgentRunResponse, error)
	CancelAgentRun(context.Context, *pb.CancelAgentRunRequest) (*pb.CancelAgentRunResponse, error)
	ConfirmAgentRun(context.Context, *pb.ConfirmAgentRunRequest) (*pb.ConfirmAgentRunResponse, error)
	SubscribeAgentRunEvents(*pb.SubscribeAgentRunEventsRequest, pb.AIService_SubscribeAgentRunEventsServer) error
}

// CandidateChatAPI is the AI Agent-owned candidate assistant gRPC contract.
type CandidateChatAPI interface {
	StreamChatGRPC(*pb.CandidateChatRequest, pb.AIService_CandidateChatStreamServer) error
	ListSessionsGRPC(context.Context, *pb.CandidateSessionListRequest) (*pb.ChatSessionListResponse, error)
	CreateSessionGRPC(context.Context, *pb.CandidateCreateSessionRequest) (*pb.CreateChatSessionResponse, error)
	SessionMessagesGRPC(context.Context, *pb.CandidateSessionMessagesRequest) (*pb.ChatHistoryResponse, error)
	UpdateSessionGRPC(context.Context, *pb.CandidateUpdateSessionRequest) (*pb.CommonResponse, error)
	DeleteSessionGRPC(context.Context, *pb.CandidateDeleteSessionRequest) (*pb.CommonResponse, error)
}

// PromptAPI is the AI Agent-owned prompt management gRPC contract.
type PromptAPI interface {
	ListPromptTemplates(context.Context, *pb.ListPromptTemplatesRequest) (*pb.ListPromptTemplatesResponse, error)
	CreatePromptTemplate(context.Context, *pb.CreatePromptTemplateRequest) (*pb.PromptTemplateResponse, error)
	UpdatePromptTemplate(context.Context, *pb.UpdatePromptTemplateRequest) (*pb.PromptTemplateResponse, error)
	DeletePromptTemplate(context.Context, *pb.DeletePromptTemplateRequest) (*pb.CommonResponse, error)
	GetPromptVersionHistory(context.Context, *pb.GetPromptVersionHistoryRequest) (*pb.GetPromptVersionHistoryResponse, error)
	RollbackPromptVersion(context.Context, *pb.RollbackPromptVersionRequest) (*pb.PromptTemplateResponse, error)
	RenderPrompt(context.Context, *pb.RenderPromptRequest) (*pb.RenderPromptResponse, error)
	GetActivePromptByAgentType(context.Context, *pb.GetActivePromptByAgentTypeRequest) (*pb.GetActivePromptByAgentTypeResponse, error)
}

// AgentConfigAPI is the AI Agent-owned runtime configuration gRPC contract.
type AgentConfigAPI interface {
	ListAgents(context.Context, *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error)
	ListCapabilities(context.Context, *pb.ListCapabilitiesRequest) (*pb.ListCapabilitiesResponse, error)
	CreateAgent(context.Context, *pb.CreateAgentRequest) (*pb.AgentConfigResponse, error)
	UpdateAgent(context.Context, *pb.UpdateAgentRequest) (*pb.AgentConfigResponse, error)
	DeleteAgent(context.Context, *pb.DeleteAgentRequest) (*pb.CommonResponse, error)
	GetAgentConfig(context.Context, *pb.GetAgentConfigRequest) (*pb.GetAgentConfigResponse, error)
}

// SkillAPI is the AI Agent-owned reusable Skill registry gRPC contract.
type SkillAPI interface {
	ListSkills(context.Context, *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error)
	CreateSkill(context.Context, *pb.CreateSkillRequest) (*pb.SkillResponse, error)
	UpdateSkill(context.Context, *pb.UpdateSkillRequest) (*pb.SkillResponse, error)
	CreateSkillVersion(context.Context, *pb.CreateSkillVersionRequest) (*pb.SkillVersionResponse, error)
	ListSkillVersions(context.Context, *pb.ListSkillVersionsRequest) (*pb.ListSkillVersionsResponse, error)
	ActivateSkillVersion(context.Context, *pb.ActivateSkillVersionRequest) (*pb.SkillResponse, error)
	ListSkillTools(context.Context, *pb.ListSkillToolsRequest) (*pb.ListSkillToolsResponse, error)
	UpdateSkillTool(context.Context, *pb.UpdateSkillToolRequest) (*pb.SkillToolResponse, error)
}

// AgentSkillAPI is the AI Agent-owned agent-specific Skill gRPC contract.
type AgentSkillAPI interface {
	ListAgentSkills(context.Context, *pb.ListAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error)
	ListAvailableAgentSkills(context.Context, *pb.ListAvailableAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error)
	DebugSemanticRetrieval(context.Context, *pb.DebugSemanticRetrievalRequest) (*pb.DebugSemanticRetrievalResponse, error)
	GetAgentSkill(context.Context, *pb.GetAgentSkillRequest) (*pb.AgentSkillResponse, error)
	CreateAgentSkill(context.Context, *pb.CreateAgentSkillRequest) (*pb.AgentSkillResponse, error)
	UpdateAgentSkill(context.Context, *pb.UpdateAgentSkillRequest) (*pb.AgentSkillResponse, error)
	CreateAgentSkillVersion(context.Context, *pb.CreateAgentSkillVersionRequest) (*pb.AgentSkillVersionResponse, error)
	ListAgentSkillVersions(context.Context, *pb.ListAgentSkillVersionsRequest) (*pb.ListAgentSkillVersionsResponse, error)
	ActivateAgentSkillVersion(context.Context, *pb.ActivateAgentSkillVersionRequest) (*pb.AgentSkillResponse, error)
	UpdateAgentSkillStatus(context.Context, *pb.UpdateAgentSkillStatusRequest) (*pb.AgentSkillResponse, error)
	PreviewAgentSkill(context.Context, *pb.PreviewAgentSkillRequest) (*pb.PreviewAgentSkillResponse, error)
}

// MCPAPI is the AI Agent-owned MCP governance and tool execution gRPC contract.
type MCPAPI interface {
	ListMCPServers(context.Context, *pb.ListMCPServersRequest) (*pb.ListMCPServersResponse, error)
	CreateMCPServer(context.Context, *pb.CreateMCPServerRequest) (*pb.MCPServerResponse, error)
	UpdateMCPServer(context.Context, *pb.UpdateMCPServerRequest) (*pb.MCPServerResponse, error)
	DeleteMCPServer(context.Context, *pb.DeleteMCPServerRequest) (*pb.CommonResponse, error)
	TestMCPConnection(context.Context, *pb.TestMCPConnectionRequest) (*pb.TestMCPConnectionResponse, error)
	ListMCPTools(context.Context, *pb.ListMCPToolsRequest) (*pb.ListMCPToolsResponse, error)
	CallMCPTool(context.Context, *pb.CallMCPToolRequest) (*pb.CallMCPToolResponse, error)
	ListMCPToolPolicies(context.Context, *pb.ListMCPToolPoliciesRequest) (*pb.ListMCPToolPoliciesResponse, error)
	CreateMCPToolPolicy(context.Context, *pb.CreateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error)
	UpdateMCPToolPolicy(context.Context, *pb.UpdateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error)
	DeleteMCPToolPolicy(context.Context, *pb.DeleteMCPToolPolicyRequest) (*pb.CommonResponse, error)
	ListMCPToolLogs(context.Context, *pb.ListMCPToolLogsRequest) (*pb.ListMCPToolLogsResponse, error)
}

// EmbeddingConfigAPI is the AI Agent-owned embedding provider/model configuration gRPC contract.
type EmbeddingConfigAPI interface {
	ListEmbeddingProviders(context.Context, *pb.ListEmbeddingProvidersRequest) (*pb.ListEmbeddingProvidersResponse, error)
	CreateEmbeddingProvider(context.Context, *pb.CreateEmbeddingProviderRequest) (*pb.EmbeddingProviderResponse, error)
	UpdateEmbeddingProvider(context.Context, *pb.UpdateEmbeddingProviderRequest) (*pb.EmbeddingProviderResponse, error)
	DeleteEmbeddingProvider(context.Context, *pb.DeleteEmbeddingProviderRequest) (*pb.CommonResponse, error)
	ListEmbeddingModels(context.Context, *pb.ListEmbeddingModelsRequest) (*pb.ListEmbeddingModelsResponse, error)
	CreateEmbeddingModel(context.Context, *pb.CreateEmbeddingModelRequest) (*pb.EmbeddingModelResponse, error)
	UpdateEmbeddingModel(context.Context, *pb.UpdateEmbeddingModelRequest) (*pb.EmbeddingModelResponse, error)
	SetDefaultEmbeddingModel(context.Context, *pb.SetDefaultEmbeddingModelRequest) (*pb.CommonResponse, error)
	TestEmbeddingModel(context.Context, *pb.TestEmbeddingModelRequest) (*pb.TestEmbeddingModelResponse, error)
	BackfillEmbeddings(context.Context, *pb.BackfillEmbeddingsRequest) (*pb.BackfillEmbeddingsResponse, error)
}

// LLMConfigAPI is the AI Agent-owned model provider fallback and runtime configuration gRPC contract.
type LLMConfigAPI interface {
	ListProviders(context.Context, *pb.ListProvidersRequest) (*pb.ListProvidersResponse, error)
	CreateProvider(context.Context, *pb.CreateProviderRequest) (*pb.ProviderResponse, error)
	UpdateProvider(context.Context, *pb.UpdateProviderRequest) (*pb.ProviderResponse, error)
	DeleteProvider(context.Context, *pb.DeleteProviderRequest) (*pb.CommonResponse, error)
	TestProviderConnection(context.Context, *pb.TestProviderConnectionRequest) (*pb.TestProviderConnectionResponse, error)
	ListModels(context.Context, *pb.ListModelsRequest) (*pb.ListModelsResponse, error)
	CreateModel(context.Context, *pb.CreateModelRequest) (*pb.ModelResponse, error)
	UpdateModel(context.Context, *pb.UpdateModelRequest) (*pb.ModelResponse, error)
	DeleteModel(context.Context, *pb.DeleteModelRequest) (*pb.CommonResponse, error)
}

// UsageAuditAPI is the AI Agent-owned AI usage audit query gRPC contract.
type UsageAuditAPI interface {
	GetUsageStats(context.Context, *pb.GetUsageStatsRequest) (*pb.GetUsageStatsResponse, error)
	GetUsageTrend(context.Context, *pb.GetUsageTrendRequest) (*pb.GetUsageTrendResponse, error)
}
