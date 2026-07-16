package grpc

import (
	"context"
	"strings"

	"smart-recruit-proto/recruitment/pb"
)

const (
	configCodeUnavailable int32 = 500
	configCodeUnsupported int32 = 501
)

type llmConfigStore interface {
	CreateLlmProvider(context.Context, *pb.CreateProviderRequest) (*pb.ProviderResponse, error)
	UpdateLlmProvider(context.Context, *pb.UpdateProviderRequest) (*pb.ProviderResponse, error)
	DeleteLlmProvider(context.Context, *pb.DeleteProviderRequest) (*pb.CommonResponse, error)
	TestLlmProviderConnection(context.Context, *pb.TestProviderConnectionRequest) (*pb.TestProviderConnectionResponse, error)
	CreateLlmModel(context.Context, *pb.CreateModelRequest) (*pb.ModelResponse, error)
	UpdateLlmModel(context.Context, *pb.UpdateModelRequest) (*pb.ModelResponse, error)
	DeleteLlmModel(context.Context, *pb.DeleteModelRequest) (*pb.CommonResponse, error)
}

type promptConfigStore interface {
	CreatePromptTemplate(context.Context, *pb.CreatePromptTemplateRequest) (*pb.PromptTemplateResponse, error)
	UpdatePromptTemplate(context.Context, *pb.UpdatePromptTemplateRequest) (*pb.PromptTemplateResponse, error)
	DeletePromptTemplate(context.Context, *pb.DeletePromptTemplateRequest) (*pb.CommonResponse, error)
	GetPromptVersionHistory(context.Context, *pb.GetPromptVersionHistoryRequest) (*pb.GetPromptVersionHistoryResponse, error)
	RollbackPromptVersion(context.Context, *pb.RollbackPromptVersionRequest) (*pb.PromptTemplateResponse, error)
	RenderPrompt(context.Context, *pb.RenderPromptRequest) (*pb.RenderPromptResponse, error)
	GetActivePromptByAgentType(context.Context, *pb.GetActivePromptByAgentTypeRequest) (*pb.GetActivePromptByAgentTypeResponse, error)
}

type agentConfigStore interface {
	CreateAgent(context.Context, *pb.CreateAgentRequest) (*pb.AgentConfigResponse, error)
	UpdateAgent(context.Context, *pb.UpdateAgentRequest) (*pb.AgentConfigResponse, error)
	DeleteAgent(context.Context, *pb.DeleteAgentRequest) (*pb.CommonResponse, error)
	GetAgentConfig(context.Context, *pb.GetAgentConfigRequest) (*pb.GetAgentConfigResponse, error)
}

type embeddingConfigStore interface {
	CreateEmbeddingProvider(context.Context, *pb.CreateEmbeddingProviderRequest) (*pb.EmbeddingProviderResponse, error)
	UpdateEmbeddingProvider(context.Context, *pb.UpdateEmbeddingProviderRequest) (*pb.EmbeddingProviderResponse, error)
	DeleteEmbeddingProvider(context.Context, *pb.DeleteEmbeddingProviderRequest) (*pb.CommonResponse, error)
	CreateEmbeddingModel(context.Context, *pb.CreateEmbeddingModelRequest) (*pb.EmbeddingModelResponse, error)
	UpdateEmbeddingModel(context.Context, *pb.UpdateEmbeddingModelRequest) (*pb.EmbeddingModelResponse, error)
	SetDefaultEmbeddingModel(context.Context, *pb.SetDefaultEmbeddingModelRequest) (*pb.CommonResponse, error)
	TestEmbeddingModel(context.Context, *pb.TestEmbeddingModelRequest) (*pb.TestEmbeddingModelResponse, error)
	BackfillEmbeddings(context.Context, *pb.BackfillEmbeddingsRequest) (*pb.BackfillEmbeddingsResponse, error)
}

func (s nativeLlmConfigService) CreateProvider(ctx context.Context, req *pb.CreateProviderRequest) (*pb.ProviderResponse, error) {
	store, ok := s.store.(llmConfigStore)
	if !ok {
		return &pb.ProviderResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.CreateLlmProvider(ctx, req)
}

func (s nativeLlmConfigService) UpdateProvider(ctx context.Context, req *pb.UpdateProviderRequest) (*pb.ProviderResponse, error) {
	store, ok := s.store.(llmConfigStore)
	if !ok {
		return &pb.ProviderResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.UpdateLlmProvider(ctx, req)
}

func (s nativeLlmConfigService) DeleteProvider(ctx context.Context, req *pb.DeleteProviderRequest) (*pb.CommonResponse, error) {
	store, ok := s.store.(llmConfigStore)
	if !ok {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.DeleteLlmProvider(ctx, req)
}

func (s nativeLlmConfigService) TestProviderConnection(ctx context.Context, req *pb.TestProviderConnectionRequest) (*pb.TestProviderConnectionResponse, error) {
	store, ok := s.store.(llmConfigStore)
	if !ok {
		return &pb.TestProviderConnectionResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured", Success: false}, nil
	}
	return store.TestLlmProviderConnection(ctx, req)
}

func (s nativeLlmConfigService) CreateModel(ctx context.Context, req *pb.CreateModelRequest) (*pb.ModelResponse, error) {
	store, ok := s.store.(llmConfigStore)
	if !ok {
		return &pb.ModelResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.CreateLlmModel(ctx, req)
}

func (s nativeLlmConfigService) UpdateModel(ctx context.Context, req *pb.UpdateModelRequest) (*pb.ModelResponse, error) {
	store, ok := s.store.(llmConfigStore)
	if !ok {
		return &pb.ModelResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.UpdateLlmModel(ctx, req)
}

func (s nativeLlmConfigService) DeleteModel(ctx context.Context, req *pb.DeleteModelRequest) (*pb.CommonResponse, error) {
	store, ok := s.store.(llmConfigStore)
	if !ok {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.DeleteLlmModel(ctx, req)
}

func (s nativePromptService) CreatePromptTemplate(ctx context.Context, req *pb.CreatePromptTemplateRequest) (*pb.PromptTemplateResponse, error) {
	store, ok := s.store.(promptConfigStore)
	if !ok {
		return &pb.PromptTemplateResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.CreatePromptTemplate(ctx, req)
}

func (s nativePromptService) UpdatePromptTemplate(ctx context.Context, req *pb.UpdatePromptTemplateRequest) (*pb.PromptTemplateResponse, error) {
	store, ok := s.store.(promptConfigStore)
	if !ok {
		return &pb.PromptTemplateResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.UpdatePromptTemplate(ctx, req)
}

func (s nativePromptService) DeletePromptTemplate(ctx context.Context, req *pb.DeletePromptTemplateRequest) (*pb.CommonResponse, error) {
	store, ok := s.store.(promptConfigStore)
	if !ok {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.DeletePromptTemplate(ctx, req)
}

func (s nativePromptService) GetPromptVersionHistory(ctx context.Context, req *pb.GetPromptVersionHistoryRequest) (*pb.GetPromptVersionHistoryResponse, error) {
	store, ok := s.store.(promptConfigStore)
	if !ok {
		return &pb.GetPromptVersionHistoryResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.GetPromptVersionHistory(ctx, req)
}

func (s nativePromptService) RollbackPromptVersion(ctx context.Context, req *pb.RollbackPromptVersionRequest) (*pb.PromptTemplateResponse, error) {
	store, ok := s.store.(promptConfigStore)
	if !ok {
		return &pb.PromptTemplateResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.RollbackPromptVersion(ctx, req)
}

func (s nativePromptService) RenderPrompt(ctx context.Context, req *pb.RenderPromptRequest) (*pb.RenderPromptResponse, error) {
	store, ok := s.store.(promptConfigStore)
	if !ok {
		return &pb.RenderPromptResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.RenderPrompt(ctx, req)
}

func (s nativePromptService) GetActivePromptByAgentType(ctx context.Context, req *pb.GetActivePromptByAgentTypeRequest) (*pb.GetActivePromptByAgentTypeResponse, error) {
	store, ok := s.store.(promptConfigStore)
	if !ok {
		return &pb.GetActivePromptByAgentTypeResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.GetActivePromptByAgentType(ctx, req)
}

func (s nativeAgentConfigService) CreateAgent(ctx context.Context, req *pb.CreateAgentRequest) (*pb.AgentConfigResponse, error) {
	store, ok := s.store.(agentConfigStore)
	if !ok {
		return &pb.AgentConfigResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.CreateAgent(ctx, req)
}

func (s nativeAgentConfigService) UpdateAgent(ctx context.Context, req *pb.UpdateAgentRequest) (*pb.AgentConfigResponse, error) {
	store, ok := s.store.(agentConfigStore)
	if !ok {
		return &pb.AgentConfigResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.UpdateAgent(ctx, req)
}

func (s nativeAgentConfigService) DeleteAgent(ctx context.Context, req *pb.DeleteAgentRequest) (*pb.CommonResponse, error) {
	store, ok := s.store.(agentConfigStore)
	if !ok {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.DeleteAgent(ctx, req)
}

func (s nativeAgentConfigService) GetAgentConfig(ctx context.Context, req *pb.GetAgentConfigRequest) (*pb.GetAgentConfigResponse, error) {
	store, ok := s.store.(agentConfigStore)
	if !ok {
		return &pb.GetAgentConfigResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.GetAgentConfig(ctx, req)
}

func (s nativeEmbeddingConfigService) CreateEmbeddingProvider(ctx context.Context, req *pb.CreateEmbeddingProviderRequest) (*pb.EmbeddingProviderResponse, error) {
	store, ok := s.store.(embeddingConfigStore)
	if !ok {
		return &pb.EmbeddingProviderResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.CreateEmbeddingProvider(ctx, req)
}

func (s nativeEmbeddingConfigService) UpdateEmbeddingProvider(ctx context.Context, req *pb.UpdateEmbeddingProviderRequest) (*pb.EmbeddingProviderResponse, error) {
	store, ok := s.store.(embeddingConfigStore)
	if !ok {
		return &pb.EmbeddingProviderResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.UpdateEmbeddingProvider(ctx, req)
}

func (s nativeEmbeddingConfigService) DeleteEmbeddingProvider(ctx context.Context, req *pb.DeleteEmbeddingProviderRequest) (*pb.CommonResponse, error) {
	store, ok := s.store.(embeddingConfigStore)
	if !ok {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.DeleteEmbeddingProvider(ctx, req)
}

func (s nativeEmbeddingConfigService) CreateEmbeddingModel(ctx context.Context, req *pb.CreateEmbeddingModelRequest) (*pb.EmbeddingModelResponse, error) {
	store, ok := s.store.(embeddingConfigStore)
	if !ok {
		return &pb.EmbeddingModelResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.CreateEmbeddingModel(ctx, req)
}

func (s nativeEmbeddingConfigService) UpdateEmbeddingModel(ctx context.Context, req *pb.UpdateEmbeddingModelRequest) (*pb.EmbeddingModelResponse, error) {
	store, ok := s.store.(embeddingConfigStore)
	if !ok {
		return &pb.EmbeddingModelResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.UpdateEmbeddingModel(ctx, req)
}

func (s nativeEmbeddingConfigService) SetDefaultEmbeddingModel(ctx context.Context, req *pb.SetDefaultEmbeddingModelRequest) (*pb.CommonResponse, error) {
	store, ok := s.store.(embeddingConfigStore)
	if !ok {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	return store.SetDefaultEmbeddingModel(ctx, req)
}

func (s nativeEmbeddingConfigService) TestEmbeddingModel(ctx context.Context, req *pb.TestEmbeddingModelRequest) (*pb.TestEmbeddingModelResponse, error) {
	if s.embedding != nil {
		return s.embedding.TestModel(ctx, req)
	}
	store, ok := s.store.(embeddingConfigStore)
	if !ok {
		return &pb.TestEmbeddingModelResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured", Success: false}, nil
	}
	return store.TestEmbeddingModel(ctx, req)
}

func (s nativeEmbeddingConfigService) BackfillEmbeddings(ctx context.Context, req *pb.BackfillEmbeddingsRequest) (*pb.BackfillEmbeddingsResponse, error) {
	if s.embedding != nil {
		return s.embedding.Backfill(ctx, req)
	}
	store, ok := s.store.(embeddingConfigStore)
	if !ok {
		return &pb.BackfillEmbeddingsResponse{Code: configCodeUnsupported, Msg: "embedding backfill worker is not configured in native runtime"}, nil
	}
	return store.BackfillEmbeddings(ctx, req)
}

func builtinCapabilities(agentType string) []*pb.CapabilityInfo {
	items := []*pb.CapabilityInfo{
		{
			Source:      "builtin",
			Key:         "candidate_search",
			Name:        "candidate_search",
			DisplayName: "Candidate Search",
			Description: "Search candidate and application context from recruitment data.",
			IsAvailable: true,
			RuntimeType: "native",
		},
		{
			Source:      "builtin",
			Key:         "resume_intelligence",
			Name:        "resume_intelligence",
			DisplayName: "Resume Intelligence",
			Description: "Read resume intelligence already persisted by the recruitment domain.",
			IsAvailable: true,
			RuntimeType: "native",
		},
		{
			Source:      "builtin",
			Key:         "interview_context",
			Name:        "interview_context",
			DisplayName: "Interview Context",
			Description: "Use interview schedule and result context when it is available.",
			IsAvailable: true,
			RuntimeType: "native",
		},
		// Executable recruiting tools exposed to Agent admin bindings / runtime allowlists.
		{Source: "builtin", Key: "get_job_list", Name: "get_job_list", DisplayName: "岗位列表", Description: "List open jobs owned by the current HR.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "search_jobs", Name: "search_jobs", DisplayName: "搜索岗位", Description: "Search HR jobs by keyword and status.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "get_job_detail", Name: "get_job_detail", DisplayName: "岗位详情", Description: "Get a single job detail within HR scope.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "get_application_snapshot", Name: "get_application_snapshot", DisplayName: "投递快照", Description: "Load application snapshot context when application_id is present.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "query_total_applications", Name: "query_total_applications", DisplayName: "累计投递", Description: "Count total applications for the current HR.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "query_today_applications", Name: "query_today_applications", DisplayName: "今日投递", Description: "Count today's new applications.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "get_job_heat_ranking", Name: "get_job_heat_ranking", DisplayName: "岗位热度", Description: "Rank jobs by application volume.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "list_all_applications", Name: "list_all_applications", DisplayName: "全部投递", Description: "List applications across HR jobs.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "list_applications_by_job", Name: "list_applications_by_job", DisplayName: "按岗位投递", Description: "List applications for one job.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "list_applications_by_status", Name: "list_applications_by_status", DisplayName: "按状态投递", Description: "List applications filtered by status.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "get_application_status_summary", Name: "get_application_status_summary", DisplayName: "状态分布", Description: "Aggregate application status counts.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "get_application_trend", Name: "get_application_trend", DisplayName: "投递趋势", Description: "Daily application trend over recent days.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "search_candidates", Name: "search_candidates", DisplayName: "搜索候选人", Description: "Search candidates by name/phone/job.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "get_candidate_detail", Name: "get_candidate_detail", DisplayName: "候选人详情", Description: "Load candidate application snapshot.", IsAvailable: true, RuntimeType: "native"},
		{Source: "builtin", Key: "propose_application_status_update", Name: "propose_application_status_update", DisplayName: "状态变更提案", Description: "Propose status change without mutating DB.", IsAvailable: true, RuntimeType: "native"},
	}
	if strings.TrimSpace(agentType) == "candidate_assistant" {
		return items[:1]
	}
	return items
}
