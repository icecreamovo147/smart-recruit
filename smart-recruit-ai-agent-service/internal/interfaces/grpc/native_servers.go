package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	gogrpc "google.golang.org/grpc"

	aiagentruntime "smart-recruit-ai-agent-service/internal/runtime"
	"smart-recruit-proto/recruitment/pb"
)

type ChatProvider interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

type AIStore interface {
	EnsureChatSession(ctx context.Context, ownerRole int32, ownerID int64, title string, applicationID int64) (ChatSessionRow, error)
	ListChatSessions(ctx context.Context, ownerRole int32, ownerID int64, page, pageSize int32) ([]ChatSessionRow, int64, error)
	UpdateChatSessionTitle(ctx context.Context, ownerRole int32, ownerID, sessionID int64, title string) error
	DeleteChatSession(ctx context.Context, ownerRole int32, ownerID, sessionID int64) error
	AppendChatMessage(ctx context.Context, message ChatMessageRow) (ChatMessageRow, error)
	ListChatMessages(ctx context.Context, ownerRole int32, ownerID, sessionID int64, page, pageSize int32) ([]ChatMessageRow, error)
	ListToolTraces(ctx context.Context, ownerID, sessionID int64) ([]ToolTraceRow, error)
	CreateAgentRun(ctx context.Context, run AgentRunRow) (AgentRunRow, bool, error)
	ListAgentRuns(ctx context.Context, ownerID, sessionID int64) ([]AgentRunRow, error)
	GetAgentRun(ctx context.Context, ownerID, runID int64) (AgentRunRow, bool, error)
	GetActiveAgentRun(ctx context.Context, ownerID, sessionID int64) (AgentRunRow, bool, error)
	UpdateAgentRunStatus(ctx context.Context, ownerID, runID int64, status string) (AgentRunRow, bool, error)
	AppendAgentRunEvent(ctx context.Context, runID int64, eventType, payload string) (AgentRunEventRow, error)
	ListAgentRunEvents(ctx context.Context, ownerID, runID, afterSeq int64) ([]AgentRunEventRow, error)
	ListLlmProviders(ctx context.Context, page, pageSize int32) ([]*pb.LlmProviderInfo, int64, error)
	ListLlmModels(ctx context.Context, page, pageSize int32, providerID int64) ([]*pb.LlmModelInfo, int64, error)
	ListPromptTemplates(ctx context.Context, page, pageSize int32, agentType string) ([]*pb.PromptTemplateInfo, int64, error)
	ListAgentConfigs(ctx context.Context, page, pageSize int32, agentType string) ([]*pb.AgentConfigInfo, int64, error)
	ListMCPServers(ctx context.Context, page, pageSize int32) ([]*pb.MCPServerInfo, int64, error)
	ListAgentSkills(ctx context.Context, page, pageSize int32, keyword string, enabledOnly bool) ([]*pb.AgentSkillInfo, int64, error)
	ListEmbeddingProviders(ctx context.Context, page, pageSize int32) ([]*pb.EmbeddingProviderInfo, int64, error)
	ListEmbeddingModels(ctx context.Context, page, pageSize int32, providerID int64) ([]*pb.EmbeddingModelInfo, int64, error)
}

type RuntimeDeps struct {
	Store            AIStore
	Provider         ChatProvider
	EmbeddingWorker  bool
	AgentRunWorker   bool
	RuntimeName      string
	EmbeddingConfigs pb.EmbeddingConfigServiceServer
}

type ChatSessionRow struct {
	ID            int64
	Title         string
	ApplicationID int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ChatMessageRow struct {
	ID             int64
	OwnerRole      int32
	OwnerID        int64
	SessionID      int64
	Role           string
	Content        string
	ProcessContent string
	ModelID        int64
	ModelName      string
	CreatedAt      time.Time
}

type ToolTraceRow struct {
	ID            int64
	SessionID     int64
	ToolName      string
	ArgsJSON      string
	ResultContent string
	DurationMs    int64
	ErrorMsg      string
	CreatedAt     time.Time
}

type AgentRunRow struct {
	ID                int64
	SessionID         int64
	MessageID         int64
	HistoryID         int64
	OwnerID           int64
	ClientRequestID   string
	Status            string
	AssistantText     string
	ProcessText       string
	OptionContextJSON string
	LastEventSeq      int64
	ErrorType         string
	ErrorMessage      string
	ModelID           int64
	ModelName         string
	AgentType         string
	AgentID           int64
	AgentName         string
	StartedAt         time.Time
	CompletedAt       *time.Time
	CancelRequestedAt *time.Time
	CanceledAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AgentRunEventRow struct {
	RunID       int64
	Seq         int64
	EventType   string
	PayloadJSON string
	CreatedAt   time.Time
}

func NewNativeRuntimeDeps(deps RuntimeDeps) aiagentruntime.Deps {
	ai := NewNativeAIService(deps.Store, deps.Provider)
	embedding := deps.EmbeddingConfigs
	if embedding == nil {
		embedding = nativeEmbeddingConfigService{store: deps.Store}
	}
	return aiagentruntime.Deps{
		AI:                     ai,
		LlmConfig:              nativeLlmConfigService{store: deps.Store},
		Prompt:                 nativePromptService{store: deps.Store},
		AgentConfig:            nativeAgentConfigService{store: deps.Store},
		MCP:                    nativeMCPService{store: deps.Store},
		Skill:                  nativeSkillService{},
		AgentSkill:             nativeAgentSkillService{store: deps.Store},
		RecruitingIntelligence: nativeRecruitingIntelligenceService{},
		EmbeddingConfig:        embedding,
		LongTasks: aiagentruntime.LongTaskControls{
			RabbitMQRequired: true,
			EmbeddingWorker:  deps.EmbeddingWorker,
			AgentRunWorker:   deps.AgentRunWorker,
			RuntimeName:      deps.RuntimeName,
		},
	}
}

func NewNativeAIService(store AIStore, provider ChatProvider) pb.AIServiceServer {
	return &nativeAIService{store: store, provider: provider}
}

type nativeAIService struct {
	pb.UnimplementedAIServiceServer
	store    AIStore
	provider ChatProvider
}

func (s *nativeAIService) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	if req == nil {
		return nil, errors.New("chat request is required")
	}
	session, err := s.ensureSession(ctx, 1, req.GetHrId(), req.GetSessionId(), req.GetApplicationId(), req.GetMessage())
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetMessage()) != "" && s.store != nil {
		_, _ = s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: 1, OwnerID: req.GetHrId(), SessionID: session.ID, Role: "user", Content: req.GetMessage(), ModelID: req.GetModelId()})
	}
	reply, err := s.complete(ctx, req.GetMessage())
	if err != nil {
		return nil, err
	}
	if s.store != nil {
		_, _ = s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: 1, OwnerID: req.GetHrId(), SessionID: session.ID, Role: "assistant", Content: reply, ModelID: req.GetModelId(), CreatedAt: time.Now()})
	}
	return &pb.ChatResponse{Code: 0, Msg: "success", Reply: reply, CreatedAt: formatTime(time.Now()), SessionId: session.ID, ApplicationId: req.GetApplicationId()}, nil
}

func (s *nativeAIService) ChatStream(req *pb.ChatRequest, stream gogrpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	resp, err := s.Chat(stream.Context(), req)
	if err != nil {
		return err
	}
	return stream.Send(&pb.ChatStreamResponse{Code: resp.Code, Msg: resp.Msg, Delta: resp.Reply, Done: true, SessionId: resp.SessionId, CreatedAt: resp.CreatedAt, EventType: "done"})
}

func (s *nativeAIService) History(ctx context.Context, req *pb.ChatHistoryRequest) (*pb.ChatHistoryResponse, error) {
	rows, err := s.listMessages(ctx, 1, req.GetHrId(), 0, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: 0, Msg: "success", List: mapChatMessages(rows)}, nil
}

func (s *nativeAIService) AnalyzeApplication(ctx context.Context, req *pb.AnalyzeApplicationRequest) (*pb.AnalyzeApplicationResponse, error) {
	reply, err := s.complete(ctx, fmt.Sprintf("Analyze application %d", req.GetApplicationId()))
	if err != nil {
		return nil, err
	}
	return &pb.AnalyzeApplicationResponse{Code: 0, Msg: "success", Reply: reply}, nil
}

func (s *nativeAIService) ListChatSessions(ctx context.Context, req *pb.ChatSessionListRequest) (*pb.ChatSessionListResponse, error) {
	rows, total, err := s.listSessions(ctx, 1, req.GetHrId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatSessionListResponse{Code: 0, Msg: "success", Total: total, List: mapChatSessions(rows)}, nil
}

func (s *nativeAIService) CreateChatSession(ctx context.Context, req *pb.CreateChatSessionRequest) (*pb.CreateChatSessionResponse, error) {
	session, err := s.ensureSession(ctx, 1, req.GetHrId(), 0, 0, req.GetTitle())
	if err != nil {
		return nil, err
	}
	return &pb.CreateChatSessionResponse{Code: 0, Msg: "success", Session: mapChatSession(session)}, nil
}

func (s *nativeAIService) SessionMessages(ctx context.Context, req *pb.SessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	rows, err := s.listMessages(ctx, 1, req.GetHrId(), req.GetSessionId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: 0, Msg: "success", List: mapChatMessages(rows)}, nil
}

func (s *nativeAIService) CreateApplicationAnalysisSession(ctx context.Context, req *pb.CreateApplicationAnalysisSessionRequest) (*pb.CreateApplicationAnalysisSessionResponse, error) {
	session, err := s.ensureSession(ctx, 1, req.GetHrId(), 0, req.GetApplicationId(), fmt.Sprintf("Application %d analysis", req.GetApplicationId()))
	if err != nil {
		return nil, err
	}
	return &pb.CreateApplicationAnalysisSessionResponse{Code: 0, Msg: "success", Session: mapChatSession(session)}, nil
}

func (s *nativeAIService) UpdateSession(ctx context.Context, req *pb.UpdateSessionRequest) (*pb.CommonResponse, error) {
	if s.store == nil {
		return commonOK(), nil
	}
	if err := s.store.UpdateChatSessionTitle(ctx, 1, req.GetHrId(), req.GetSessionId(), req.GetTitle()); err != nil {
		return nil, err
	}
	return commonOK(), nil
}

func (s *nativeAIService) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.CommonResponse, error) {
	if s.store == nil {
		return commonOK(), nil
	}
	if err := s.store.DeleteChatSession(ctx, 1, req.GetHrId(), req.GetSessionId()); err != nil {
		return nil, err
	}
	return commonOK(), nil
}

func (s *nativeAIService) CandidateChatStream(req *pb.CandidateChatRequest, stream gogrpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	session, err := s.ensureSession(stream.Context(), 2, req.GetUserId(), req.GetSessionId(), 0, req.GetMessage())
	if err != nil {
		return err
	}
	reply, err := s.complete(stream.Context(), req.GetMessage())
	if err != nil {
		return err
	}
	return stream.Send(&pb.ChatStreamResponse{Code: 0, Msg: "success", Delta: reply, Done: true, SessionId: session.ID, CreatedAt: formatTime(time.Now()), EventType: "done"})
}

func (s *nativeAIService) CandidateListSessions(ctx context.Context, req *pb.CandidateSessionListRequest) (*pb.ChatSessionListResponse, error) {
	rows, total, err := s.listSessions(ctx, 2, req.GetUserId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatSessionListResponse{Code: 0, Msg: "success", Total: total, List: mapChatSessions(rows)}, nil
}

func (s *nativeAIService) CandidateCreateSession(ctx context.Context, req *pb.CandidateCreateSessionRequest) (*pb.CreateChatSessionResponse, error) {
	session, err := s.ensureSession(ctx, 2, req.GetUserId(), 0, 0, req.GetTitle())
	if err != nil {
		return nil, err
	}
	return &pb.CreateChatSessionResponse{Code: 0, Msg: "success", Session: mapChatSession(session)}, nil
}

func (s *nativeAIService) CandidateSessionMessages(ctx context.Context, req *pb.CandidateSessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	rows, err := s.listMessages(ctx, 2, req.GetUserId(), req.GetSessionId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: 0, Msg: "success", List: mapChatMessages(rows)}, nil
}

func (s *nativeAIService) CandidateUpdateSession(ctx context.Context, req *pb.CandidateUpdateSessionRequest) (*pb.CommonResponse, error) {
	if s.store != nil {
		if err := s.store.UpdateChatSessionTitle(ctx, 2, req.GetUserId(), req.GetSessionId(), req.GetTitle()); err != nil {
			return nil, err
		}
	}
	return commonOK(), nil
}

func (s *nativeAIService) CandidateDeleteSession(ctx context.Context, req *pb.CandidateDeleteSessionRequest) (*pb.CommonResponse, error) {
	if s.store != nil {
		if err := s.store.DeleteChatSession(ctx, 2, req.GetUserId(), req.GetSessionId()); err != nil {
			return nil, err
		}
	}
	return commonOK(), nil
}

func (s *nativeAIService) GetToolTraces(ctx context.Context, req *pb.GetToolTracesRequest) (*pb.GetToolTracesResponse, error) {
	if s.store == nil {
		return &pb.GetToolTracesResponse{Code: 0, Msg: "success"}, nil
	}
	rows, err := s.store.ListToolTraces(ctx, req.GetHrId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	items := make([]*pb.ToolTraceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.ToolTraceItem{Id: row.ID, SessionId: row.SessionID, ToolName: row.ToolName, ArgsJson: row.ArgsJSON, ResultContent: row.ResultContent, DurationMs: row.DurationMs, ErrorMsg: row.ErrorMsg, CreatedAt: formatTime(row.CreatedAt)})
	}
	return &pb.GetToolTracesResponse{Code: 0, Msg: "success", List: items}, nil
}

func (s *nativeAIService) GetAgentRuns(ctx context.Context, req *pb.GetAgentRunsRequest) (*pb.GetAgentRunsResponse, error) {
	if s.store == nil {
		return &pb.GetAgentRunsResponse{Code: 0, Msg: "success"}, nil
	}
	rows, err := s.store.ListAgentRuns(ctx, req.GetHrId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	items := make([]*pb.AgentRunItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAgentRunItem(row))
	}
	return &pb.GetAgentRunsResponse{Code: 0, Msg: "success", List: items}, nil
}

func (s *nativeAIService) CreateAgentRun(ctx context.Context, req *pb.CreateAgentRunRequest) (*pb.CreateAgentRunResponse, error) {
	if req == nil {
		return nil, errors.New("create agent run request is required")
	}
	if s.store == nil {
		run := fallbackAgentRun(req.GetHrId(), req.GetSessionId(), req.GetClientRequestId(), req.GetMessage(), req.GetModelId())
		return &pb.CreateAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
	}
	run, idempotent, err := s.store.CreateAgentRun(ctx, fallbackAgentRun(req.GetHrId(), req.GetSessionId(), req.GetClientRequestId(), req.GetMessage(), req.GetModelId()))
	if err != nil {
		return nil, err
	}
	_, _ = s.store.AppendAgentRunEvent(ctx, run.ID, "run.created", `{"status":"queued"}`)
	return &pb.CreateAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run), IdempotentReplay: idempotent}, nil
}

func (s *nativeAIService) GetAgentRun(ctx context.Context, req *pb.GetAgentRunRequest) (*pb.GetAgentRunResponse, error) {
	run, found, err := s.getRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.GetAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}
	return &pb.GetAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
}

func (s *nativeAIService) GetActiveAgentRun(ctx context.Context, req *pb.GetActiveAgentRunRequest) (*pb.GetActiveAgentRunResponse, error) {
	if s.store == nil {
		return &pb.GetActiveAgentRunResponse{Code: 0, Msg: "success"}, nil
	}
	run, found, err := s.store.GetActiveAgentRun(ctx, req.GetHrId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	return &pb.GetActiveAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run), HasActiveRun: found}, nil
}

func (s *nativeAIService) SubscribeAgentRunEvents(req *pb.SubscribeAgentRunEventsRequest, stream gogrpc.ServerStreamingServer[pb.AgentRunEvent]) error {
	if s.store == nil {
		return nil
	}
	rows, err := s.store.ListAgentRunEvents(stream.Context(), req.GetHrId(), req.GetRunId(), req.GetAfterSeq())
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := stream.Send(&pb.AgentRunEvent{RunId: row.RunID, Seq: row.Seq, EventType: row.EventType, PayloadJson: row.PayloadJSON, CreatedAt: formatTime(row.CreatedAt)}); err != nil {
			return err
		}
	}
	return nil
}

func (s *nativeAIService) CancelAgentRun(ctx context.Context, req *pb.CancelAgentRunRequest) (*pb.CancelAgentRunResponse, error) {
	run, found, err := s.updateRun(ctx, req.GetHrId(), req.GetRunId(), "cancel_requested")
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.CancelAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}
	return &pb.CancelAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
}

func (s *nativeAIService) ConfirmAgentRun(ctx context.Context, req *pb.ConfirmAgentRunRequest) (*pb.ConfirmAgentRunResponse, error) {
	run, found, err := s.updateRun(ctx, req.GetHrId(), req.GetRunId(), "running")
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.ConfirmAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}
	return &pb.ConfirmAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
}

func (s *nativeAIService) ensureSession(ctx context.Context, ownerRole int32, ownerID, sessionID, applicationID int64, seed string) (ChatSessionRow, error) {
	if sessionID > 0 {
		return ChatSessionRow{ID: sessionID, UpdatedAt: time.Now()}, nil
	}
	title := strings.TrimSpace(seed)
	if title == "" {
		title = "New chat"
	}
	if len([]rune(title)) > 32 {
		title = string([]rune(title)[:32])
	}
	if s.store == nil {
		now := time.Now()
		return ChatSessionRow{ID: time.Now().UnixNano(), Title: title, ApplicationID: applicationID, CreatedAt: now, UpdatedAt: now}, nil
	}
	return s.store.EnsureChatSession(ctx, ownerRole, ownerID, title, applicationID)
}

func (s *nativeAIService) complete(ctx context.Context, prompt string) (string, error) {
	if s.provider == nil {
		return "AI provider is not configured for this environment.", nil
	}
	reply, err := s.provider.Complete(ctx, prompt)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(reply) == "" {
		return "AI provider returned an empty response.", nil
	}
	return reply, nil
}

func (s *nativeAIService) listSessions(ctx context.Context, ownerRole int32, ownerID int64, page, pageSize int32) ([]ChatSessionRow, int64, error) {
	if s.store == nil {
		return nil, 0, nil
	}
	return s.store.ListChatSessions(ctx, ownerRole, ownerID, normalizePage(page), normalizePageSize(pageSize))
}

func (s *nativeAIService) listMessages(ctx context.Context, ownerRole int32, ownerID, sessionID int64, page, pageSize int32) ([]ChatMessageRow, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.ListChatMessages(ctx, ownerRole, ownerID, sessionID, normalizePage(page), normalizePageSize(pageSize))
}

func (s *nativeAIService) getRun(ctx context.Context, ownerID, runID int64) (AgentRunRow, bool, error) {
	if s.store == nil {
		return AgentRunRow{}, false, nil
	}
	return s.store.GetAgentRun(ctx, ownerID, runID)
}

func (s *nativeAIService) updateRun(ctx context.Context, ownerID, runID int64, status string) (AgentRunRow, bool, error) {
	if s.store == nil {
		return AgentRunRow{}, false, nil
	}
	run, found, err := s.store.UpdateAgentRunStatus(ctx, ownerID, runID, status)
	if err == nil && found {
		_, _ = s.store.AppendAgentRunEvent(ctx, runID, "status.changed", fmt.Sprintf(`{"status":%q}`, status))
	}
	return run, found, err
}

type nativeRecruitingIntelligenceService struct {
	pb.UnimplementedRecruitingIntelligenceServiceServer
}

func (nativeRecruitingIntelligenceService) GetResumeProfile(context.Context, *pb.GetResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	return &pb.GetResumeProfileResponse{Code: 404, Msg: "resume profile not found"}, nil
}

func (nativeRecruitingIntelligenceService) ParseResumeProfile(context.Context, *pb.ParseResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	return &pb.GetResumeProfileResponse{Code: 202, Msg: "resume profile parsing is queued or unavailable"}, nil
}

func (nativeRecruitingIntelligenceService) EvaluateCandidateMatch(context.Context, *pb.EvaluateCandidateMatchRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	return &pb.GetCandidateMatchEvaluationResponse{Code: 202, Msg: "candidate match evaluation is queued or unavailable"}, nil
}

func (nativeRecruitingIntelligenceService) GetCandidateMatchEvaluation(context.Context, *pb.GetCandidateMatchEvaluationRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	return &pb.GetCandidateMatchEvaluationResponse{Code: 404, Msg: "candidate match evaluation not found"}, nil
}

func (nativeRecruitingIntelligenceService) CompareCandidatesForJob(context.Context, *pb.CompareCandidatesForJobRequest) (*pb.CompareCandidatesForJobResponse, error) {
	return &pb.CompareCandidatesForJobResponse{Code: 0, Msg: "success"}, nil
}

type nativeLlmConfigService struct {
	pb.UnimplementedLlmConfigServiceServer
	store AIStore
}
type nativePromptService struct {
	pb.UnimplementedPromptServiceServer
	store AIStore
}
type nativeAgentConfigService struct {
	pb.UnimplementedAgentConfigServiceServer
	store AIStore
}
type nativeMCPService struct {
	pb.UnimplementedMCPServiceServer
	store AIStore
}
type nativeSkillService struct {
	pb.UnimplementedSkillServiceServer
}
type nativeAgentSkillService struct {
	pb.UnimplementedAgentSkillServiceServer
	store AIStore
}
type nativeEmbeddingConfigService struct {
	pb.UnimplementedEmbeddingConfigServiceServer
	store AIStore
}

func (s nativeLlmConfigService) ListProviders(ctx context.Context, req *pb.ListProvidersRequest) (*pb.ListProvidersResponse, error) {
	if s.store == nil {
		return &pb.ListProvidersResponse{Code: 0, Msg: "success"}, nil
	}
	rows, total, err := s.store.ListLlmProviders(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &pb.ListProvidersResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeLlmConfigService) ListModels(ctx context.Context, req *pb.ListModelsRequest) (*pb.ListModelsResponse, error) {
	if s.store == nil {
		return &pb.ListModelsResponse{Code: 0, Msg: "success"}, nil
	}
	rows, total, err := s.store.ListLlmModels(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetProviderId())
	if err != nil {
		return nil, err
	}
	return &pb.ListModelsResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativePromptService) ListPromptTemplates(ctx context.Context, req *pb.ListPromptTemplatesRequest) (*pb.ListPromptTemplatesResponse, error) {
	if s.store == nil {
		return &pb.ListPromptTemplatesResponse{Code: 0, Msg: "success"}, nil
	}
	rows, total, err := s.store.ListPromptTemplates(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetAgentType())
	if err != nil {
		return nil, err
	}
	return &pb.ListPromptTemplatesResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeAgentConfigService) ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	if s.store == nil {
		return &pb.ListAgentsResponse{Code: 0, Msg: "success"}, nil
	}
	rows, total, err := s.store.ListAgentConfigs(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetAgentType())
	if err != nil {
		return nil, err
	}
	return &pb.ListAgentsResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (nativeAgentConfigService) ListCapabilities(context.Context, *pb.ListCapabilitiesRequest) (*pb.ListCapabilitiesResponse, error) {
	return &pb.ListCapabilitiesResponse{Code: 0, Msg: "success"}, nil
}
func (s nativeMCPService) ListMCPServers(ctx context.Context, req *pb.ListMCPServersRequest) (*pb.ListMCPServersResponse, error) {
	if s.store == nil {
		return &pb.ListMCPServersResponse{Code: 0, Msg: "success"}, nil
	}
	rows, total, err := s.store.ListMCPServers(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &pb.ListMCPServersResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (nativeMCPService) ListMCPToolPolicies(context.Context, *pb.ListMCPToolPoliciesRequest) (*pb.ListMCPToolPoliciesResponse, error) {
	return &pb.ListMCPToolPoliciesResponse{Code: 0, Msg: "success"}, nil
}
func (nativeMCPService) ListMCPToolLogs(context.Context, *pb.ListMCPToolLogsRequest) (*pb.ListMCPToolLogsResponse, error) {
	return &pb.ListMCPToolLogsResponse{Code: 0, Msg: "success"}, nil
}
func (nativeSkillService) ListSkills(context.Context, *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	return &pb.ListSkillsResponse{Code: 0, Msg: "success"}, nil
}
func (s nativeAgentSkillService) ListAgentSkills(ctx context.Context, req *pb.ListAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	if s.store == nil {
		return &pb.ListAgentSkillsResponse{Code: 0, Msg: "success"}, nil
	}
	rows, total, err := s.store.ListAgentSkills(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetKeyword(), req.GetEnabledOnly())
	if err != nil {
		return nil, err
	}
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeAgentSkillService) ListAvailableAgentSkills(ctx context.Context, req *pb.ListAvailableAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	if s.store == nil {
		return &pb.ListAgentSkillsResponse{Code: 0, Msg: "success"}, nil
	}
	rows, total, err := s.store.ListAgentSkills(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), "", true)
	if err != nil {
		return nil, err
	}
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeEmbeddingConfigService) ListEmbeddingProviders(ctx context.Context, req *pb.ListEmbeddingProvidersRequest) (*pb.ListEmbeddingProvidersResponse, error) {
	if s.store == nil {
		return &pb.ListEmbeddingProvidersResponse{Code: 0, Msg: "success"}, nil
	}
	rows, total, err := s.store.ListEmbeddingProviders(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &pb.ListEmbeddingProvidersResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeEmbeddingConfigService) ListEmbeddingModels(ctx context.Context, req *pb.ListEmbeddingModelsRequest) (*pb.ListEmbeddingModelsResponse, error) {
	if s.store == nil {
		return &pb.ListEmbeddingModelsResponse{Code: 0, Msg: "success"}, nil
	}
	rows, total, err := s.store.ListEmbeddingModels(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetProviderId())
	if err != nil {
		return nil, err
	}
	return &pb.ListEmbeddingModelsResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}

func commonOK() *pb.CommonResponse {
	return &pb.CommonResponse{Code: 0, Msg: "success"}
}

func mapChatSession(row ChatSessionRow) *pb.ChatSession {
	return &pb.ChatSession{SessionId: row.ID, Title: row.Title, ApplicationId: row.ApplicationID, CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
}

func mapChatSessions(rows []ChatSessionRow) []*pb.ChatSession {
	items := make([]*pb.ChatSession, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapChatSession(row))
	}
	return items
}

func mapChatMessages(rows []ChatMessageRow) []*pb.ChatMessage {
	items := make([]*pb.ChatMessage, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.ChatMessage{Role: row.Role, Content: row.Content, ProcessContent: row.ProcessContent, ModelId: row.ModelID, ModelName: row.ModelName, CreatedAt: formatTime(row.CreatedAt)})
	}
	return items
}

func mapAgentRunItem(row AgentRunRow) *pb.AgentRunItem {
	return &pb.AgentRunItem{Id: row.ID, SessionId: row.SessionID, MessageId: row.MessageID, HistoryId: row.HistoryID, HrId: row.OwnerID, AgentType: row.AgentType, AgentId: row.AgentID, AgentName: row.AgentName, ModelId: row.ModelID, ModelName: row.ModelName, Status: row.Status, FinalAnswer: row.AssistantText, ErrorType: row.ErrorType, ErrorMessage: row.ErrorMessage, StartedAt: formatTime(row.StartedAt), CompletedAt: formatTimePtr(row.CompletedAt), CreatedAt: formatTime(row.CreatedAt)}
}

func mapAgentRunSnapshot(row AgentRunRow) *pb.AgentRunSnapshot {
	if row.ID == 0 {
		return nil
	}
	return &pb.AgentRunSnapshot{RunId: row.ID, SessionId: row.SessionID, HrId: row.OwnerID, ClientRequestId: row.ClientRequestID, MessageId: row.MessageID, HistoryId: row.HistoryID, Status: row.Status, AssistantText: row.AssistantText, ProcessText: row.ProcessText, OptionContextJson: row.OptionContextJSON, LastEventSeq: row.LastEventSeq, ErrorType: row.ErrorType, ErrorMessage: row.ErrorMessage, ModelId: row.ModelID, ModelName: row.ModelName, AgentType: row.AgentType, AgentId: row.AgentID, AgentName: row.AgentName, StartedAt: formatTime(row.StartedAt), CompletedAt: formatTimePtr(row.CompletedAt), CancelRequestedAt: formatTimePtr(row.CancelRequestedAt), CanceledAt: formatTimePtr(row.CanceledAt), CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
}

func fallbackAgentRun(hrID, sessionID int64, clientRequestID, message string, modelID int64) AgentRunRow {
	now := time.Now()
	return AgentRunRow{SessionID: sessionID, OwnerID: hrID, ClientRequestID: clientRequestID, Status: "queued", AssistantText: "", ModelID: modelID, AgentType: "hr", AgentName: "hr_recruiting_agent", StartedAt: now, CreatedAt: now, UpdatedAt: now}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}

func normalizePage(page int32) int32 {
	if page <= 0 {
		return 1
	}
	return page
}

func normalizePageSize(pageSize int32) int32 {
	if pageSize <= 0 {
		return 20
	}
	if pageSize > 100 {
		return 100
	}
	return pageSize
}
