package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	aiagentruntime "smart-recruit-ai-agent-service/internal/runtime"
	"smart-recruit-platform-go/errs"
	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

type ChatProvider interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

type ModelAwareChatProvider interface {
	CompleteWithModel(ctx context.Context, prompt string, modelID int64) (string, error)
}

type AIStore interface {
	EnsureChatSession(ctx context.Context, ownerRole int32, ownerID int64, title string, applicationID int64) (ChatSessionRow, error)
	GetChatSession(ctx context.Context, ownerRole int32, ownerID, sessionID int64) (ChatSessionRow, bool, error)
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
	CompleteAgentRun(ctx context.Context, ownerID, runID int64, assistantText, status, errorType, errorMessage string) (AgentRunRow, bool, error)
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

type activePromptStore interface {
	GetActivePromptByAgentType(context.Context, *pb.GetActivePromptByAgentTypeRequest) (*pb.GetActivePromptByAgentTypeResponse, error)
}

type RuntimeDeps struct {
	Store            AIStore
	Provider         ChatProvider
	EmbeddingWorker  bool
	AgentRunWorker   bool
	RuntimeName      string
	EmbeddingConfigs pb.EmbeddingConfigServiceServer
	Auth             pb.AuthServiceClient
	Applications     pb.ApplicationOwnerServiceClient
	Jobs             pb.JobServiceClient
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

type RecruitingApplicationContext struct {
	ApplicationID   int64
	JobID           int64
	CandidateUserID int64
	CandidateName   string
	ResumeID        int64
	IsCurrent       int32
}

type RecruitingResumeParseRunRow struct {
	ID            uint64
	ResumeID      int64
	UserID        int64
	AgentRunID    *uint64
	Status        string
	ParserVersion string
	InputHash     string
	ErrorMessage  string
	StartedAt     time.Time
	CompletedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type RecruitingResumeProfileRow struct {
	ID                   uint64
	ResumeID             int64
	UserID               int64
	ParseRunID           uint64
	Version              int32
	IsCurrent            int32
	FullName             string
	Email                string
	Phone                string
	Location             string
	Headline             string
	Summary              string
	TotalExperienceYears float64
	HighestDegree        string
	RawJSON              string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type RecruitingResumeEducationRow struct {
	ID          uint64
	School      string
	Degree      string
	Major       string
	StartDate   *time.Time
	EndDate     *time.Time
	Description string
	SortOrder   int32
}

type RecruitingResumeExperienceRow struct {
	ID               uint64
	Company          string
	Title            string
	Location         string
	StartDate        *time.Time
	EndDate          *time.Time
	IsCurrent        int32
	Description      string
	AchievementsJSON string
	SortOrder        int32
}

type RecruitingResumeProjectRow struct {
	ID               uint64
	Name             string
	Role             string
	StartDate        *time.Time
	EndDate          *time.Time
	Description      string
	TechnologiesJSON string
	HighlightsJSON   string
	SortOrder        int32
}

type RecruitingResumeSkillRow struct {
	ID        uint64
	Name      string
	Category  string
	Level     string
	Years     float64
	Evidence  string
	SortOrder int32
}

type RecruitingResumeProfileSnapshot struct {
	ParseRun    RecruitingResumeParseRunRow
	Profile     RecruitingResumeProfileRow
	Educations  []RecruitingResumeEducationRow
	Experiences []RecruitingResumeExperienceRow
	Projects    []RecruitingResumeProjectRow
	Skills      []RecruitingResumeSkillRow
}

type RecruitingCandidateMatchEvaluationRow struct {
	ID                 uint64
	ApplicationID      int64
	JobID              int64
	CandidateUserID    int64
	ResumeProfileID    uint64
	AgentRunID         *uint64
	EvaluationVersion  int32
	IsLatest           int32
	OverallScore       float64
	Recommendation     string
	Summary            string
	StrengthsJSON      string
	RisksJSON          string
	ScoreBreakdownJSON string
	ModelName          string
	EvaluatedAt        time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type RecruitingCandidateMatchEvidenceRow struct {
	ID           uint64
	EvidenceType string
	Dimension    string
	SourceTable  string
	SourceID     *uint64
	Snippet      string
	Weight       float64
	ScoreImpact  float64
	MetadataJSON string
	CreatedAt    time.Time
}

type RecruitingCandidateMatchSnapshot struct {
	Evaluation RecruitingCandidateMatchEvaluationRow
	Evidence   []RecruitingCandidateMatchEvidenceRow
}

var (
	errAIStoreRequired     = errors.New("ai store is required for native AI runtime")
	errAIProviderRequired  = errors.New("ai provider is required for native AI runtime")
	errChatSessionNotFound = errors.New("chat session not found")
)

const (
	ownerRoleCandidate int32 = 1
	ownerRoleHR        int32 = 2

	agentRunEventPollInterval = 500 * time.Millisecond

	agentRunCodeBadRequest int32 = 400

	candidateAssistantAgentType    = "candidate_assistant"
	candidatePromptRoleSystem      = "system"
	candidateChatContextPageSize   = int32(100)
	candidateChatEmptyMessageError = "message is required"

	candidateSystemPrompt = "你是智能招聘系统的候选人端 AI 助手。你只服务当前登录候选人，回答应围绕候选人自己的求职、简历、岗位、投递、面试和 Offer 相关问题。\n" +
		"必须保护招聘系统数据边界：不要透露 HR 内部备注、其他候选人信息、未授权的招聘数据或系统实现细节。\n" +
		"当缺少实时工具或数据时，明确说明当前无法读取实时系统数据，并给出安全、可执行的求职建议。"

	agentRunStatusQueued              = "queued"
	agentRunStatusPlanning            = "planning"
	agentRunStatusRunning             = "running"
	agentRunStatusWaitingConfirmation = "waiting_confirmation"
	agentRunStatusCancelRequested     = "cancel_requested"
	agentRunStatusSucceeded           = "succeeded"
	agentRunStatusFailed              = "failed"
	agentRunStatusCanceled            = "canceled"
	agentRunStatusPartial             = "partial"
)

func NewNativeRuntimeDeps(deps RuntimeDeps) aiagentruntime.Deps {
	ai := NewNativeAIService(deps.Store, deps.Provider)
	embedding := deps.EmbeddingConfigs
	if embedding == nil {
		embedding = nativeEmbeddingConfigService{store: deps.Store}
	}
	var recruitingStore recruitingReadStore
	if deps.Store != nil {
		recruitingStore, _ = deps.Store.(recruitingReadStore)
	}
	return aiagentruntime.Deps{
		AI:                     ai,
		LlmConfig:              nativeLlmConfigService{store: deps.Store},
		Prompt:                 nativePromptService{store: deps.Store},
		AgentConfig:            nativeAgentConfigService{store: deps.Store},
		MCP:                    nativeMCPService{store: deps.Store},
		Skill:                  nativeSkillService{store: deps.Store},
		AgentSkill:             nativeAgentSkillService{store: deps.Store},
		RecruitingIntelligence: nativeRecruitingIntelligenceService{store: recruitingStore, auth: deps.Auth, applications: deps.Applications, jobs: deps.Jobs},
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
	return &nativeAIService{store: store, provider: provider, eventHub: newAgentRunEventHub()}
}

type nativeAIService struct {
	pb.UnimplementedAIServiceServer
	store           AIStore
	provider        ChatProvider
	eventHubMu      sync.Mutex
	eventHub        *agentRunEventHub
	runCancelMu     sync.Mutex
	runCancels      map[int64]*agentRunCancelEntry
	runTransitionMu sync.Mutex
}

type agentRunCancelEntry struct {
	cancel context.CancelFunc
}

type agentRunEventHub struct {
	mu          sync.RWMutex
	subscribers map[int64]map[chan AgentRunEventRow]struct{}
}

func newAgentRunEventHub() *agentRunEventHub {
	return &agentRunEventHub{subscribers: make(map[int64]map[chan AgentRunEventRow]struct{})}
}

func (h *agentRunEventHub) subscribe(runID int64) (<-chan AgentRunEventRow, func()) {
	ch := make(chan AgentRunEventRow, 32)
	h.mu.Lock()
	if h.subscribers[runID] == nil {
		h.subscribers[runID] = make(map[chan AgentRunEventRow]struct{})
	}
	h.subscribers[runID][ch] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		if subscribers := h.subscribers[runID]; subscribers != nil {
			delete(subscribers, ch)
			if len(subscribers) == 0 {
				delete(h.subscribers, runID)
			}
		}
		close(ch)
		h.mu.Unlock()
	}
	return ch, unsubscribe
}

func (h *agentRunEventHub) publish(row AgentRunEventRow) {
	if h == nil || row.RunID == 0 {
		return
	}
	h.mu.RLock()
	for ch := range h.subscribers[row.RunID] {
		select {
		case ch <- row:
		default:
		}
	}
	h.mu.RUnlock()
}

func (s *nativeAIService) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	if req == nil {
		return nil, errors.New("chat request is required")
	}
	session, err := s.ensureSession(ctx, ownerRoleHR, req.GetHrId(), req.GetSessionId(), req.GetApplicationId(), req.GetMessage())
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetMessage()) != "" && s.store != nil {
		if _, err := s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: ownerRoleHR, OwnerID: req.GetHrId(), SessionID: session.ID, Role: "user", Content: req.GetMessage(), ModelID: req.GetModelId()}); err != nil {
			return nil, err
		}
	}
	reply, err := s.complete(ctx, req.GetMessage(), req.GetModelId())
	if err != nil {
		if errors.Is(err, errAIProviderRequired) {
			return &pb.ChatResponse{Code: configCodeUnavailable, Msg: err.Error(), CreatedAt: formatTime(time.Now()), SessionId: session.ID, ApplicationId: req.GetApplicationId()}, nil
		}
		return nil, err
	}
	if s.store != nil {
		if _, err := s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: ownerRoleHR, OwnerID: req.GetHrId(), SessionID: session.ID, Role: "assistant", Content: reply, ModelID: req.GetModelId(), CreatedAt: time.Now()}); err != nil {
			return nil, err
		}
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
	rows, err := s.listMessages(ctx, ownerRoleHR, req.GetHrId(), 0, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: 0, Msg: "success", List: mapChatMessages(rows)}, nil
}

func (s *nativeAIService) AnalyzeApplication(ctx context.Context, req *pb.AnalyzeApplicationRequest) (*pb.AnalyzeApplicationResponse, error) {
	reply, err := s.complete(ctx, fmt.Sprintf("Analyze application %d", req.GetApplicationId()), 0)
	if err != nil {
		if errors.Is(err, errAIProviderRequired) {
			return &pb.AnalyzeApplicationResponse{Code: configCodeUnavailable, Msg: err.Error()}, nil
		}
		return nil, err
	}
	return &pb.AnalyzeApplicationResponse{Code: 0, Msg: "success", Reply: reply}, nil
}

func (s *nativeAIService) ListChatSessions(ctx context.Context, req *pb.ChatSessionListRequest) (*pb.ChatSessionListResponse, error) {
	rows, total, err := s.listSessions(ctx, ownerRoleHR, req.GetHrId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatSessionListResponse{Code: 0, Msg: "success", Total: total, List: mapChatSessions(rows)}, nil
}

func (s *nativeAIService) CreateChatSession(ctx context.Context, req *pb.CreateChatSessionRequest) (*pb.CreateChatSessionResponse, error) {
	session, err := s.ensureSession(ctx, ownerRoleHR, req.GetHrId(), 0, 0, req.GetTitle())
	if err != nil {
		return nil, err
	}
	return &pb.CreateChatSessionResponse{Code: 0, Msg: "success", Session: mapChatSession(session)}, nil
}

func (s *nativeAIService) SessionMessages(ctx context.Context, req *pb.SessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	rows, err := s.listMessages(ctx, ownerRoleHR, req.GetHrId(), req.GetSessionId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: 0, Msg: "success", List: mapChatMessages(rows)}, nil
}

func (s *nativeAIService) CreateApplicationAnalysisSession(ctx context.Context, req *pb.CreateApplicationAnalysisSessionRequest) (*pb.CreateApplicationAnalysisSessionResponse, error) {
	session, err := s.ensureSession(ctx, ownerRoleHR, req.GetHrId(), 0, req.GetApplicationId(), fmt.Sprintf("Application %d analysis", req.GetApplicationId()))
	if err != nil {
		return nil, err
	}
	return &pb.CreateApplicationAnalysisSessionResponse{Code: 0, Msg: "success", Session: mapChatSession(session)}, nil
}

func (s *nativeAIService) UpdateSession(ctx context.Context, req *pb.UpdateSessionRequest) (*pb.CommonResponse, error) {
	if s.store == nil {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: errAIStoreRequired.Error()}, nil
	}
	if err := s.store.UpdateChatSessionTitle(ctx, ownerRoleHR, req.GetHrId(), req.GetSessionId(), req.GetTitle()); err != nil {
		return nil, err
	}
	return commonOK(), nil
}

func (s *nativeAIService) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.CommonResponse, error) {
	if s.store == nil {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: errAIStoreRequired.Error()}, nil
	}
	if err := s.store.DeleteChatSession(ctx, ownerRoleHR, req.GetHrId(), req.GetSessionId()); err != nil {
		return nil, err
	}
	return commonOK(), nil
}

func (s *nativeAIService) CandidateChatStream(req *pb.CandidateChatRequest, stream gogrpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	ctx := stream.Context()
	if req == nil || strings.TrimSpace(req.GetMessage()) == "" {
		return stream.Send(&pb.ChatStreamResponse{Code: agentRunCodeBadRequest, Msg: candidateChatEmptyMessageError, Done: true, CreatedAt: formatTime(time.Now()), EventType: "done"})
	}
	session, err := s.ensureSession(ctx, ownerRoleCandidate, req.GetUserId(), req.GetSessionId(), 0, req.GetMessage())
	if err != nil {
		return err
	}
	userMessage, err := s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: ownerRoleCandidate, OwnerID: req.GetUserId(), SessionID: session.ID, Role: "user", Content: req.GetMessage()})
	if err != nil {
		return err
	}
	prompt, err := s.buildCandidateProviderPrompt(ctx, req.GetUserId(), session.ID, userMessage)
	if err != nil {
		return err
	}
	reply, err := s.complete(ctx, prompt, 0)
	if err != nil {
		return err
	}
	if s.store != nil {
		if _, err := s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: ownerRoleCandidate, OwnerID: req.GetUserId(), SessionID: session.ID, Role: "assistant", Content: reply, CreatedAt: time.Now()}); err != nil {
			return err
		}
	}
	return stream.Send(&pb.ChatStreamResponse{Code: 0, Msg: "success", Delta: reply, Done: true, SessionId: session.ID, CreatedAt: formatTime(time.Now()), EventType: "done"})
}

func (s *nativeAIService) buildCandidateProviderPrompt(ctx context.Context, userID, sessionID int64, currentUserMessage ChatMessageRow) (string, error) {
	systemPrompt, err := s.resolveCandidateSystemPrompt(ctx)
	if err != nil {
		return "", err
	}
	messages, err := s.store.ListChatMessages(ctx, ownerRoleCandidate, userID, sessionID, 1, candidateChatContextPageSize)
	if err != nil {
		return "", err
	}
	messages = ensureCandidateCurrentMessage(messages, currentUserMessage, userID, sessionID)
	return renderCandidateProviderPrompt(systemPrompt, messages, userID, sessionID), nil
}

func (s *nativeAIService) resolveCandidateSystemPrompt(ctx context.Context) (string, error) {
	if promptStore, ok := s.store.(activePromptStore); ok {
		resp, err := promptStore.GetActivePromptByAgentType(ctx, &pb.GetActivePromptByAgentTypeRequest{AgentType: candidateAssistantAgentType, PromptRole: candidatePromptRoleSystem})
		if err != nil {
			return "", err
		}
		if content := strings.TrimSpace(resp.GetTemplate().GetContent()); content != "" {
			return content, nil
		}
	}
	return candidateSystemPrompt, nil
}

func ensureCandidateCurrentMessage(messages []ChatMessageRow, current ChatMessageRow, userID, sessionID int64) []ChatMessageRow {
	if current.ID == 0 {
		return messages
	}
	for _, message := range messages {
		if message.ID == current.ID {
			return messages
		}
	}
	if current.OwnerRole == ownerRoleCandidate && current.OwnerID == userID && current.SessionID == sessionID {
		return append(messages, current)
	}
	return messages
}

func renderCandidateProviderPrompt(systemPrompt string, messages []ChatMessageRow, userID, sessionID int64) string {
	var b strings.Builder
	b.WriteString("System:\n")
	b.WriteString(strings.TrimSpace(systemPrompt))
	b.WriteString("\n\nConversation:\n")
	for _, message := range messages {
		if message.OwnerRole != ownerRoleCandidate || message.OwnerID != userID || message.SessionID != sessionID {
			continue
		}
		role := normalizeConversationRole(message.Role)
		content := strings.TrimSpace(message.Content)
		if role == "" || content == "" {
			continue
		}
		b.WriteString(role)
		b.WriteString(":\n")
		b.WriteString(content)
		b.WriteString("\n\n")
	}
	b.WriteString("Assistant:")
	return b.String()
}

func normalizeConversationRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "user", "assistant":
		return strings.ToLower(strings.TrimSpace(role))
	default:
		return ""
	}
}

func (s *nativeAIService) CandidateListSessions(ctx context.Context, req *pb.CandidateSessionListRequest) (*pb.ChatSessionListResponse, error) {
	rows, total, err := s.listSessions(ctx, ownerRoleCandidate, req.GetUserId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatSessionListResponse{Code: 0, Msg: "success", Total: total, List: mapChatSessions(rows)}, nil
}

func (s *nativeAIService) CandidateCreateSession(ctx context.Context, req *pb.CandidateCreateSessionRequest) (*pb.CreateChatSessionResponse, error) {
	session, err := s.ensureSession(ctx, ownerRoleCandidate, req.GetUserId(), 0, 0, req.GetTitle())
	if err != nil {
		return nil, err
	}
	return &pb.CreateChatSessionResponse{Code: 0, Msg: "success", Session: mapChatSession(session)}, nil
}

func (s *nativeAIService) CandidateSessionMessages(ctx context.Context, req *pb.CandidateSessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	rows, err := s.listMessages(ctx, ownerRoleCandidate, req.GetUserId(), req.GetSessionId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: 0, Msg: "success", List: mapChatMessages(rows)}, nil
}

func (s *nativeAIService) CandidateUpdateSession(ctx context.Context, req *pb.CandidateUpdateSessionRequest) (*pb.CommonResponse, error) {
	if s.missingStore() {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: errAIStoreRequired.Error()}, nil
	}
	if err := s.store.UpdateChatSessionTitle(ctx, ownerRoleCandidate, req.GetUserId(), req.GetSessionId(), req.GetTitle()); err != nil {
		return nil, err
	}
	return commonOK(), nil
}

func (s *nativeAIService) CandidateDeleteSession(ctx context.Context, req *pb.CandidateDeleteSessionRequest) (*pb.CommonResponse, error) {
	if s.missingStore() {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: errAIStoreRequired.Error()}, nil
	}
	if err := s.store.DeleteChatSession(ctx, ownerRoleCandidate, req.GetUserId(), req.GetSessionId()); err != nil {
		return nil, err
	}
	return commonOK(), nil
}

func (s *nativeAIService) GetToolTraces(ctx context.Context, req *pb.GetToolTracesRequest) (*pb.GetToolTracesResponse, error) {
	if s.store == nil {
		return &pb.GetToolTracesResponse{Code: configCodeUnavailable, Msg: errAIStoreRequired.Error()}, nil
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
		return &pb.GetAgentRunsResponse{Code: configCodeUnavailable, Msg: errAIStoreRequired.Error()}, nil
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
		return &pb.CreateAgentRunResponse{Code: configCodeUnavailable, Msg: errAIStoreRequired.Error()}, nil
	}
	run, idempotent, err := s.store.CreateAgentRun(ctx, fallbackAgentRun(req.GetHrId(), req.GetSessionId(), req.GetClientRequestId(), req.GetMessage(), req.GetModelId()))
	if err != nil {
		return nil, err
	}
	if idempotent {
		return &pb.CreateAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run), IdempotentReplay: true}, nil
	}
	created, err := s.appendAgentRunEvent(ctx, run.ID, "run.created", `{"status":"queued"}`)
	if err != nil {
		return nil, err
	}
	if created.Seq > 0 {
		run.LastEventSeq = created.Seq
	}
	s.dispatchAgentRun(run, req.GetMessage(), req.GetModelId())
	return &pb.CreateAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run), IdempotentReplay: false}, nil
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
		return &pb.GetActiveAgentRunResponse{Code: configCodeUnavailable, Msg: errAIStoreRequired.Error()}, nil
	}
	run, found, err := s.store.GetActiveAgentRun(ctx, req.GetHrId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	return &pb.GetActiveAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run), HasActiveRun: found}, nil
}

func (s *nativeAIService) SubscribeAgentRunEvents(req *pb.SubscribeAgentRunEventsRequest, stream gogrpc.ServerStreamingServer[pb.AgentRunEvent]) error {
	if req == nil {
		return errors.New("subscribe agent run events request is required")
	}
	if s.store == nil {
		return errAIStoreRequired
	}
	ctx := stream.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if _, found, err := s.getRun(ctx, req.GetHrId(), req.GetRunId()); err != nil {
		return err
	} else if !found {
		return status.Error(codes.NotFound, "agent run not found")
	}
	liveEvents, unsubscribe := s.agentRunEvents().subscribe(req.GetRunId())
	defer unsubscribe()

	lastSeq := req.GetAfterSeq()
	sendRow := func(row AgentRunEventRow) (bool, error) {
		if row.RunID != req.GetRunId() || (row.Seq > 0 && row.Seq <= lastSeq) {
			return false, nil
		}
		event := mapAgentRunEvent(row)
		if err := stream.Send(event); err != nil {
			return false, err
		}
		if row.Seq > lastSeq {
			lastSeq = row.Seq
		}
		return isTerminalAgentRunEvent(event), nil
	}
	replay := func() (bool, error) {
		rows, err := s.store.ListAgentRunEvents(ctx, req.GetHrId(), req.GetRunId(), lastSeq)
		if err != nil {
			return false, err
		}
		for _, row := range rows {
			terminal, err := sendRow(row)
			if terminal || err != nil {
				return terminal, err
			}
		}
		return false, nil
	}

	terminal, err := replay()
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	if terminal {
		return nil
	}

	ticker := time.NewTicker(agentRunEventPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case row, ok := <-liveEvents:
			if !ok {
				return nil
			}
			terminal, err := sendRow(row)
			if err != nil {
				return err
			}
			if terminal {
				return nil
			}
		case <-ticker.C:
			terminal, err := replay()
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			if terminal {
				return nil
			}
		}
	}
}

func (s *nativeAIService) CancelAgentRun(ctx context.Context, req *pb.CancelAgentRunRequest) (*pb.CancelAgentRunResponse, error) {
	if req == nil {
		return nil, errors.New("cancel agent run request is required")
	}
	run, found, err := s.getRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.CancelAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}

	if isTerminalAgentRunStatus(run.Status) || run.Status == agentRunStatusCancelRequested {
		return &pb.CancelAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
	}
	if !isCancelableAgentRunStatus(run.Status) {
		return &pb.CancelAgentRunResponse{Code: agentRunCodeBadRequest, Msg: illegalAgentRunTransitionMessage("cancel", run.Status), Run: mapAgentRunSnapshot(run)}, nil
	}

	s.runTransitionMu.Lock()
	defer s.runTransitionMu.Unlock()
	run, found, err = s.getRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.CancelAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}
	if isTerminalAgentRunStatus(run.Status) || run.Status == agentRunStatusCancelRequested {
		return &pb.CancelAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
	}
	if !isCancelableAgentRunStatus(run.Status) {
		return &pb.CancelAgentRunResponse{Code: agentRunCodeBadRequest, Msg: illegalAgentRunTransitionMessage("cancel", run.Status), Run: mapAgentRunSnapshot(run)}, nil
	}
	run, found, err = s.updateRun(ctx, req.GetHrId(), req.GetRunId(), agentRunStatusCancelRequested)
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.CancelAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}
	s.cancelAgentRunExecution(req.GetRunId())
	return &pb.CancelAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
}

func (s *nativeAIService) ConfirmAgentRun(ctx context.Context, req *pb.ConfirmAgentRunRequest) (*pb.ConfirmAgentRunResponse, error) {
	if req == nil {
		return nil, errors.New("confirm agent run request is required")
	}
	run, found, err := s.getRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.ConfirmAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}
	if run.Status == agentRunStatusRunning || isTerminalAgentRunStatus(run.Status) {
		return &pb.ConfirmAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
	}
	if run.Status != agentRunStatusWaitingConfirmation {
		return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: illegalAgentRunTransitionMessage("confirm", run.Status), Run: mapAgentRunSnapshot(run)}, nil
	}

	s.runTransitionMu.Lock()
	defer s.runTransitionMu.Unlock()
	run, found, err = s.getRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.ConfirmAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}
	if run.Status == agentRunStatusRunning || isTerminalAgentRunStatus(run.Status) {
		return &pb.ConfirmAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
	}
	if run.Status != agentRunStatusWaitingConfirmation {
		return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: illegalAgentRunTransitionMessage("confirm", run.Status), Run: mapAgentRunSnapshot(run)}, nil
	}
	run, found, err = s.updateRun(ctx, req.GetHrId(), req.GetRunId(), agentRunStatusRunning)
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.ConfirmAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}
	if _, err := s.appendAgentRunEvent(ctx, run.ID, "confirmation.accepted", fmt.Sprintf(`{"status":%q}`, agentRunStatusRunning)); err != nil {
		return nil, err
	}
	return &pb.ConfirmAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
}

func (s *nativeAIService) ensureSession(ctx context.Context, ownerRole int32, ownerID, sessionID, applicationID int64, seed string) (ChatSessionRow, error) {
	if s.store == nil {
		return ChatSessionRow{}, errAIStoreRequired
	}
	if sessionID > 0 {
		session, found, err := s.store.GetChatSession(ctx, ownerRole, ownerID, sessionID)
		if err != nil {
			return ChatSessionRow{}, err
		}
		if !found {
			return ChatSessionRow{}, status.Error(codes.NotFound, errChatSessionNotFound.Error())
		}
		return session, nil
	}
	title := strings.TrimSpace(seed)
	if title == "" {
		title = "New chat"
	}
	if len([]rune(title)) > 32 {
		title = string([]rune(title)[:32])
	}
	return s.store.EnsureChatSession(ctx, ownerRole, ownerID, title, applicationID)
}

func (s *nativeAIService) missingStore() bool {
	return s == nil || s.store == nil
}

func (s *nativeAIService) complete(ctx context.Context, prompt string, modelID int64) (string, error) {
	if s.provider == nil {
		return "", errAIProviderRequired
	}
	if modelAware, ok := s.provider.(ModelAwareChatProvider); ok {
		reply, err := modelAware.CompleteWithModel(ctx, prompt, modelID)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(reply) == "" {
			return "AI provider returned an empty response.", nil
		}
		return reply, nil
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

func (s *nativeAIService) dispatchAgentRun(run AgentRunRow, message string, modelID int64) {
	ctx, cancel := context.WithCancel(context.Background())
	s.storeAgentRunCancel(run.ID, cancel)
	go func() {
		_ = s.executeAgentRun(ctx, run, message, modelID)
	}()
}

func (s *nativeAIService) executeAgentRun(ctx context.Context, run AgentRunRow, message string, modelID int64) error {
	defer s.clearAgentRunCancel(run.ID)

	current, shouldExecute, err := s.beginAgentRunExecution(ctx, run)
	if err != nil || !shouldExecute {
		return err
	}
	reply, err := s.complete(ctx, message, modelID)
	if err != nil {
		if agentRunExecutionCanceled(ctx, err) {
			return s.finishAgentRunCanceled(ctx, current)
		}
		return s.finishAgentRunFailed(ctx, current, err)
	}
	if agentRunExecutionCanceled(ctx, nil) {
		return s.finishAgentRunCanceled(ctx, current)
	}
	return s.finishAgentRunSucceeded(ctx, current, reply, modelID)
}

func (s *nativeAIService) beginAgentRunExecution(ctx context.Context, run AgentRunRow) (AgentRunRow, bool, error) {
	storeCtx := agentRunStoreContext(ctx)
	s.runTransitionMu.Lock()
	defer s.runTransitionMu.Unlock()

	current, found, err := s.getRun(storeCtx, run.OwnerID, run.ID)
	if err != nil || !found {
		return current, false, err
	}
	if agentRunExecutionCanceled(ctx, nil) || current.Status == agentRunStatusCancelRequested {
		if err := s.completeAgentRunCanceledLocked(storeCtx, current); err != nil {
			return AgentRunRow{}, false, err
		}
		return current, false, nil
	}
	if isTerminalAgentRunStatus(current.Status) || current.Status == agentRunStatusWaitingConfirmation {
		return current, false, nil
	}
	if current.Status == agentRunStatusRunning {
		return current, true, nil
	}
	if !isExecutableAgentRunStatus(current.Status) {
		return current, false, nil
	}
	next, found, err := s.updateRun(storeCtx, current.OwnerID, current.ID, agentRunStatusRunning)
	if err != nil || !found {
		return next, false, err
	}
	return next, true, nil
}

func (s *nativeAIService) finishAgentRunSucceeded(ctx context.Context, run AgentRunRow, reply string, modelID int64) error {
	storeCtx := agentRunStoreContext(ctx)
	s.runTransitionMu.Lock()
	defer s.runTransitionMu.Unlock()

	current, found, err := s.getRun(storeCtx, run.OwnerID, run.ID)
	if err != nil || !found {
		return err
	}
	if current.Status == agentRunStatusCancelRequested || current.Status == agentRunStatusCanceled {
		return s.completeAgentRunCanceledLocked(storeCtx, current)
	}
	if isTerminalAgentRunStatus(current.Status) {
		return nil
	}
	if !isExecutableAgentRunStatus(current.Status) {
		return nil
	}
	if strings.TrimSpace(reply) != "" {
		if _, err := s.appendAgentRunEvent(storeCtx, run.ID, "assistant.delta", fmt.Sprintf(`{"status":%q,"delta":%q}`, agentRunStatusRunning, reply)); err != nil {
			return err
		}
		if _, err := s.store.AppendChatMessage(storeCtx, ChatMessageRow{OwnerRole: ownerRoleHR, OwnerID: run.OwnerID, SessionID: run.SessionID, Role: "assistant", Content: reply, ModelID: modelID, CreatedAt: time.Now()}); err != nil {
			return err
		}
	}
	if _, _, err := s.store.CompleteAgentRun(storeCtx, run.OwnerID, run.ID, reply, agentRunStatusSucceeded, "", ""); err != nil {
		return err
	}
	_, err = s.appendAgentRunEvent(storeCtx, run.ID, "run.completed", fmt.Sprintf(`{"status":%q}`, agentRunStatusSucceeded))
	return err
}

func (s *nativeAIService) finishAgentRunFailed(ctx context.Context, run AgentRunRow, runErr error) error {
	storeCtx := agentRunStoreContext(ctx)
	s.runTransitionMu.Lock()
	defer s.runTransitionMu.Unlock()

	current, found, err := s.getRun(storeCtx, run.OwnerID, run.ID)
	if err != nil || !found {
		return err
	}
	if current.Status == agentRunStatusCancelRequested || current.Status == agentRunStatusCanceled {
		return s.completeAgentRunCanceledLocked(storeCtx, current)
	}
	if isTerminalAgentRunStatus(current.Status) {
		return nil
	}
	errorMessage := runErr.Error()
	if _, eventErr := s.appendAgentRunEvent(storeCtx, run.ID, "run.error", fmt.Sprintf(`{"status":%q,"error_type":"provider","error_message":%q}`, agentRunStatusFailed, errorMessage)); eventErr != nil {
		return eventErr
	}
	if _, _, completeErr := s.store.CompleteAgentRun(storeCtx, run.OwnerID, run.ID, "", agentRunStatusFailed, "provider", errorMessage); completeErr != nil {
		return completeErr
	}
	_, eventErr := s.appendAgentRunEvent(storeCtx, run.ID, "run.completed", fmt.Sprintf(`{"status":%q,"error_type":"provider","error_message":%q}`, agentRunStatusFailed, errorMessage))
	return eventErr
}

func (s *nativeAIService) finishAgentRunCanceled(ctx context.Context, run AgentRunRow) error {
	storeCtx := agentRunStoreContext(ctx)
	s.runTransitionMu.Lock()
	defer s.runTransitionMu.Unlock()

	current, found, err := s.getRun(storeCtx, run.OwnerID, run.ID)
	if err != nil || !found {
		return err
	}
	return s.completeAgentRunCanceledLocked(storeCtx, current)
}

func (s *nativeAIService) completeAgentRunCanceledLocked(ctx context.Context, run AgentRunRow) error {
	if run.Status == agentRunStatusCanceled || isTerminalAgentRunStatus(run.Status) {
		return nil
	}
	completed, found, err := s.store.CompleteAgentRun(ctx, run.OwnerID, run.ID, run.AssistantText, agentRunStatusCanceled, "", "")
	if err != nil || !found {
		return err
	}
	_, err = s.appendAgentRunEvent(ctx, completed.ID, "run.canceled", fmt.Sprintf(`{"status":%q}`, agentRunStatusCanceled))
	return err
}

func agentRunStoreContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

func agentRunExecutionCanceled(ctx context.Context, err error) bool {
	if ctx != nil && ctx.Err() != nil {
		return true
	}
	return errors.Is(err, context.Canceled)
}

func isCancelableAgentRunStatus(status string) bool {
	switch status {
	case agentRunStatusQueued, agentRunStatusPlanning, agentRunStatusRunning, agentRunStatusWaitingConfirmation:
		return true
	default:
		return false
	}
}

func isExecutableAgentRunStatus(status string) bool {
	switch status {
	case agentRunStatusQueued, agentRunStatusPlanning, agentRunStatusRunning:
		return true
	default:
		return false
	}
}

func illegalAgentRunTransitionMessage(action, currentStatus string) string {
	if strings.TrimSpace(currentStatus) == "" {
		currentStatus = "unknown"
	}
	return fmt.Sprintf("cannot %s agent run from status %q", action, currentStatus)
}

func (s *nativeAIService) appendAgentRunEvent(ctx context.Context, runID int64, eventType, payload string) (AgentRunEventRow, error) {
	if s.store == nil {
		return AgentRunEventRow{}, errAIStoreRequired
	}
	row, err := s.store.AppendAgentRunEvent(ctx, runID, eventType, payload)
	if err != nil {
		return row, err
	}
	if row.RunID == 0 {
		row.RunID = runID
	}
	if row.EventType == "" {
		row.EventType = eventType
	}
	if row.PayloadJSON == "" {
		row.PayloadJSON = payload
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
	}
	s.agentRunEvents().publish(row)
	return row, nil
}

func (s *nativeAIService) agentRunEvents() *agentRunEventHub {
	s.eventHubMu.Lock()
	defer s.eventHubMu.Unlock()
	if s.eventHub == nil {
		s.eventHub = newAgentRunEventHub()
	}
	return s.eventHub
}

func (s *nativeAIService) storeAgentRunCancel(runID int64, cancel context.CancelFunc) {
	if runID == 0 || cancel == nil {
		return
	}
	s.runCancelMu.Lock()
	defer s.runCancelMu.Unlock()
	if s.runCancels == nil {
		s.runCancels = make(map[int64]*agentRunCancelEntry)
	}
	s.runCancels[runID] = &agentRunCancelEntry{cancel: cancel}
}

func (s *nativeAIService) cancelAgentRunExecution(runID int64) {
	s.runCancelMu.Lock()
	entry := s.runCancels[runID]
	s.runCancelMu.Unlock()
	if entry != nil && entry.cancel != nil {
		entry.cancel()
	}
}

func (s *nativeAIService) clearAgentRunCancel(runID int64) {
	s.runCancelMu.Lock()
	entry := s.runCancels[runID]
	delete(s.runCancels, runID)
	s.runCancelMu.Unlock()
	if entry != nil && entry.cancel != nil {
		entry.cancel()
	}
}

func mapAgentRunEvent(row AgentRunEventRow) *pb.AgentRunEvent {
	event := &pb.AgentRunEvent{RunId: row.RunID, Seq: row.Seq, EventType: row.EventType, PayloadJson: row.PayloadJSON, CreatedAt: formatTime(row.CreatedAt)}
	applyAgentRunPayload(event, row.PayloadJSON)
	return event
}

func isTerminalAgentRunEvent(event *pb.AgentRunEvent) bool {
	if event == nil {
		return false
	}
	switch event.GetEventType() {
	case "run.completed", "run.canceled":
		return true
	case "run.status_changed", "status.changed":
		return isTerminalAgentRunStatus(event.GetStatus())
	default:
		return false
	}
}

func isTerminalAgentRunStatus(status string) bool {
	switch status {
	case agentRunStatusSucceeded, agentRunStatusFailed, agentRunStatusCanceled, agentRunStatusPartial:
		return true
	default:
		return false
	}
}

func applyAgentRunPayload(event *pb.AgentRunEvent, payloadJSON string) {
	if strings.TrimSpace(payloadJSON) == "" {
		return
	}
	var payload struct {
		Status       string `json:"status"`
		Delta        string `json:"delta"`
		SnapshotText string `json:"snapshot_text"`
		ToolName     string `json:"tool_name"`
		ErrorType    string `json:"error_type"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return
	}
	event.Status = payload.Status
	event.Delta = payload.Delta
	event.SnapshotText = payload.SnapshotText
	event.ToolName = payload.ToolName
	event.ErrorType = payload.ErrorType
	event.ErrorMessage = payload.ErrorMessage
}

func (s *nativeAIService) listSessions(ctx context.Context, ownerRole int32, ownerID int64, page, pageSize int32) ([]ChatSessionRow, int64, error) {
	if s.store == nil {
		return nil, 0, errAIStoreRequired
	}
	return s.store.ListChatSessions(ctx, ownerRole, ownerID, normalizePage(page), normalizePageSize(pageSize))
}

func (s *nativeAIService) listMessages(ctx context.Context, ownerRole int32, ownerID, sessionID int64, page, pageSize int32) ([]ChatMessageRow, error) {
	if s.store == nil {
		return nil, errAIStoreRequired
	}
	return s.store.ListChatMessages(ctx, ownerRole, ownerID, sessionID, normalizePage(page), normalizePageSize(pageSize))
}

func (s *nativeAIService) getRun(ctx context.Context, ownerID, runID int64) (AgentRunRow, bool, error) {
	if s.store == nil {
		return AgentRunRow{}, false, errAIStoreRequired
	}
	return s.store.GetAgentRun(ctx, ownerID, runID)
}

func (s *nativeAIService) updateRun(ctx context.Context, ownerID, runID int64, status string) (AgentRunRow, bool, error) {
	if s.store == nil {
		return AgentRunRow{}, false, errAIStoreRequired
	}
	run, found, err := s.store.UpdateAgentRunStatus(ctx, ownerID, runID, status)
	if err == nil && found {
		if _, eventErr := s.appendAgentRunEvent(ctx, runID, "run.status_changed", fmt.Sprintf(`{"status":%q}`, status)); eventErr != nil {
			return AgentRunRow{}, false, eventErr
		}
	}
	return run, found, err
}

type nativeRecruitingIntelligenceService struct {
	pb.UnimplementedRecruitingIntelligenceServiceServer
	store        recruitingReadStore
	auth         pb.AuthServiceClient
	applications pb.ApplicationOwnerServiceClient
	jobs         pb.JobServiceClient
}

type recruitingReadStore interface {
	GetRecruitingApplicationByID(ctx context.Context, applicationID int64) (RecruitingApplicationContext, bool, error)
	GetLatestRecruitingApplicationByResumeID(ctx context.Context, resumeID int64) (RecruitingApplicationContext, bool, error)
	GetRecruitingResumeProfileByID(ctx context.Context, profileID uint64) (RecruitingResumeProfileRow, bool, error)
	GetCurrentRecruitingResumeProfileByResumeID(ctx context.Context, resumeID int64) (RecruitingResumeProfileRow, bool, error)
	GetRecruitingResumeProfileSnapshot(ctx context.Context, profileID uint64) (RecruitingResumeProfileSnapshot, bool, error)
	GetRecruitingCandidateMatchEvaluationSnapshot(ctx context.Context, evaluationID uint64) (RecruitingCandidateMatchSnapshot, bool, error)
	GetRecruitingCandidateMatchEvaluationSnapshotByApplicationVersion(ctx context.Context, applicationID int64, version int32) (RecruitingCandidateMatchSnapshot, bool, error)
	GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(ctx context.Context, applicationID int64) (RecruitingCandidateMatchSnapshot, bool, error)
	ListCurrentRecruitingApplicationsByJobID(ctx context.Context, jobID int64) ([]RecruitingApplicationContext, error)
	ListLatestRecruitingCandidateMatchEvaluationsByApplicationIDs(ctx context.Context, applicationIDs []int64) ([]RecruitingCandidateMatchEvaluationRow, error)
}

type recruitingAuthError struct {
	code    int32
	message string
	err     error
}

func (e *recruitingAuthError) Error() string {
	if e == nil {
		return ""
	}
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.message, e.err)
	}
	return e.message
}

func (s nativeRecruitingIntelligenceService) verifyRecruitingStaffActor(ctx context.Context, staffUserID int64) *recruitingAuthError {
	if staffUserID <= 0 {
		return &recruitingAuthError{code: errs.ErrBadRequest, message: "staff user id is required"}
	}
	authUserID := platformmetadata.GetAuthUserID(ctx)
	if authUserID <= 0 {
		return &recruitingAuthError{code: errs.ErrForbidden, message: "authenticated user not found in context"}
	}
	if authUserID != staffUserID {
		return &recruitingAuthError{code: errs.ErrForbidden, message: fmt.Sprintf("authenticated user %d cannot access staff user %d", authUserID, staffUserID)}
	}
	return nil
}

func (s nativeRecruitingIntelligenceService) authorizeRecruitingApplication(ctx context.Context, staffUserID, applicationID int64) (*pb.GetApplicationSnapshotResponse, *recruitingAuthError) {
	if authErr := s.verifyRecruitingStaffActor(ctx, staffUserID); authErr != nil {
		return nil, authErr
	}
	if applicationID <= 0 {
		return nil, &recruitingAuthError{code: errs.ErrBadRequest, message: "application id is required"}
	}
	if s.applications == nil {
		return nil, &recruitingAuthError{code: errs.ErrInternal, message: "application owner service is not configured"}
	}
	resp, err := s.applications.GetApplicationSnapshot(ctx, &pb.GetApplicationSnapshotRequest{ApplicationId: applicationID})
	if err != nil {
		return nil, &recruitingAuthError{code: errs.ErrInternal, message: "get application snapshot failed", err: err}
	}
	if resp == nil {
		return nil, &recruitingAuthError{code: errs.ErrInternal, message: "get application snapshot returned nil response"}
	}
	switch resp.GetCode() {
	case errs.OK:
		return resp, nil
	case errs.ErrForbidden:
		return nil, &recruitingAuthError{code: errs.ErrForbidden, message: recruitingMessageOrDefault(resp.GetMsg(), "application access forbidden")}
	case errs.ErrBadRequest, 404:
		return nil, &recruitingAuthError{code: 404, message: recruitingMessageOrDefault(resp.GetMsg(), "application not found")}
	default:
		return nil, &recruitingAuthError{code: resp.GetCode(), message: recruitingMessageOrDefault(resp.GetMsg(), "get application snapshot was denied")}
	}
}

func (s nativeRecruitingIntelligenceService) authorizeRecruitingJob(ctx context.Context, staffUserID, jobID int64) (*pb.Job, *recruitingAuthError) {
	if authErr := s.verifyRecruitingStaffActor(ctx, staffUserID); authErr != nil {
		return nil, authErr
	}
	if jobID <= 0 {
		return nil, &recruitingAuthError{code: errs.ErrBadRequest, message: "job id is required"}
	}
	if s.jobs == nil {
		return nil, &recruitingAuthError{code: errs.ErrInternal, message: "job service is not configured"}
	}
	const (
		pageSize = int32(100)
		maxPages = int32(1000)
	)
	var seen int64
	for page := int32(1); page <= maxPages; page++ {
		resp, err := s.jobs.ListHRJobs(ctx, &pb.ListHRJobsRequest{HrId: staffUserID, Page: page, PageSize: pageSize})
		if err != nil {
			return nil, &recruitingAuthError{code: errs.ErrInternal, message: "list hr jobs failed", err: err}
		}
		if resp == nil {
			return nil, &recruitingAuthError{code: errs.ErrInternal, message: "list hr jobs returned nil response"}
		}
		if resp.GetCode() != errs.OK {
			return nil, &recruitingAuthError{code: resp.GetCode(), message: recruitingMessageOrDefault(resp.GetMsg(), "list hr jobs was denied")}
		}
		jobs := resp.GetList()
		for _, job := range jobs {
			if job.GetJobId() == jobID {
				return job, nil
			}
		}
		if len(jobs) == 0 {
			return nil, &recruitingAuthError{code: errs.ErrForbidden, message: "job not in staff scope"}
		}
		seen += int64(len(jobs))
		if resp.GetTotal() > 0 && seen >= resp.GetTotal() {
			return nil, &recruitingAuthError{code: errs.ErrForbidden, message: "job not in staff scope"}
		}
		if int32(len(jobs)) < pageSize {
			return nil, &recruitingAuthError{code: errs.ErrForbidden, message: "job not in staff scope"}
		}
	}
	return nil, &recruitingAuthError{code: errs.ErrForbidden, message: "job not in staff scope"}
}

func recruitingMessageOrDefault(message, fallback string) string {
	if trimmed := strings.TrimSpace(message); trimmed != "" {
		return trimmed
	}
	return fallback
}

func (s nativeRecruitingIntelligenceService) GetResumeProfile(ctx context.Context, req *pb.GetResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	if req.GetApplicationId() <= 0 && req.GetResumeId() <= 0 && req.GetProfileId() == 0 {
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "resume_id, profile_id, or application_id is required"}, nil
	}
	if s.store == nil {
		return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "recruiting read store is not configured"}, nil
	}
	resumeID, accessResp := s.resolveResumeProfileAccess(ctx, req)
	if accessResp != nil {
		return accessResp, nil
	}
	profileID := req.GetProfileId()
	if profileID == 0 {
		profile, found, err := s.store.GetCurrentRecruitingResumeProfileByResumeID(ctx, resumeID)
		if err != nil {
			return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
		}
		if !found {
			return &pb.GetResumeProfileResponse{Code: 404, Msg: "current resume profile not found"}, nil
		}
		profileID = profile.ID
	}
	snapshot, found, err := s.store.GetRecruitingResumeProfileSnapshot(ctx, profileID)
	if err != nil {
		return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
	}
	if !found {
		return &pb.GetResumeProfileResponse{Code: 404, Msg: "resume profile not found"}, nil
	}
	if snapshot.Profile.ResumeID != resumeID {
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "profile_id does not belong to requested resume/application"}, nil
	}
	return &pb.GetResumeProfileResponse{Code: errs.OK, Msg: "success", Profile: recruitingResumeProfileSnapshotPB(snapshot)}, nil
}

func (nativeRecruitingIntelligenceService) ParseResumeProfile(context.Context, *pb.ParseResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	return &pb.GetResumeProfileResponse{Code: configCodeUnsupported, Msg: "resume profile parsing worker is not configured in native runtime"}, nil
}

func (nativeRecruitingIntelligenceService) EvaluateCandidateMatch(context.Context, *pb.EvaluateCandidateMatchRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	return &pb.GetCandidateMatchEvaluationResponse{Code: configCodeUnsupported, Msg: "candidate match evaluation worker is not configured in native runtime"}, nil
}

func (s nativeRecruitingIntelligenceService) GetCandidateMatchEvaluation(ctx context.Context, req *pb.GetCandidateMatchEvaluationRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	if req.GetApplicationId() <= 0 {
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrBadRequest, Msg: "application_id is required"}, nil
	}
	if s.store == nil {
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: "recruiting read store is not configured"}, nil
	}
	if _, authErr := s.authorizeRecruitingApplication(ctx, req.GetStaffUserId(), req.GetApplicationId()); authErr != nil {
		return recruitingMatchAuthResponse(authErr), nil
	}
	var (
		snapshot RecruitingCandidateMatchSnapshot
		found    bool
		err      error
	)
	switch {
	case req.GetEvaluationId() > 0:
		snapshot, found, err = s.store.GetRecruitingCandidateMatchEvaluationSnapshot(ctx, req.GetEvaluationId())
		if err == nil && found && snapshot.Evaluation.ApplicationID != req.GetApplicationId() {
			found = false
		}
	case req.GetEvaluationVersion() > 0:
		snapshot, found, err = s.store.GetRecruitingCandidateMatchEvaluationSnapshotByApplicationVersion(ctx, req.GetApplicationId(), req.GetEvaluationVersion())
	default:
		snapshot, found, err = s.store.GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(ctx, req.GetApplicationId())
	}
	if err != nil {
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
	}
	if !found {
		return &pb.GetCandidateMatchEvaluationResponse{Code: 404, Msg: "candidate match evaluation not found"}, nil
	}
	return &pb.GetCandidateMatchEvaluationResponse{Code: errs.OK, Msg: "success", Evaluation: recruitingCandidateMatchSnapshotPB(snapshot)}, nil
}

func (s nativeRecruitingIntelligenceService) CompareCandidatesForJob(ctx context.Context, req *pb.CompareCandidatesForJobRequest) (*pb.CompareCandidatesForJobResponse, error) {
	if _, authErr := s.authorizeRecruitingJob(ctx, req.GetStaffUserId(), req.GetJobId()); authErr != nil {
		return recruitingComparisonAuthResponse(req.GetJobId(), authErr), nil
	}
	if s.store == nil {
		return &pb.CompareCandidatesForJobResponse{Code: errs.ErrInternal, Msg: "recruiting read store is not configured", JobId: req.GetJobId()}, nil
	}
	applications, err := s.store.ListCurrentRecruitingApplicationsByJobID(ctx, req.GetJobId())
	if err != nil {
		return &pb.CompareCandidatesForJobResponse{Code: errs.ErrInternal, Msg: err.Error(), JobId: req.GetJobId()}, nil
	}
	applicationIDs := make([]int64, 0, len(applications))
	for _, application := range applications {
		applicationIDs = append(applicationIDs, application.ApplicationID)
	}
	evaluations, err := s.store.ListLatestRecruitingCandidateMatchEvaluationsByApplicationIDs(ctx, applicationIDs)
	if err != nil {
		return &pb.CompareCandidatesForJobResponse{Code: errs.ErrInternal, Msg: err.Error(), JobId: req.GetJobId()}, nil
	}
	byApplicationID := make(map[int64]RecruitingCandidateMatchEvaluationRow, len(evaluations))
	for _, evaluation := range evaluations {
		byApplicationID[evaluation.ApplicationID] = evaluation
	}
	candidates := make([]*pb.CandidateComparisonItem, 0, len(applications))
	missing := make([]int64, 0)
	for _, application := range applications {
		item := &pb.CandidateComparisonItem{
			ApplicationId:   application.ApplicationID,
			CandidateUserId: application.CandidateUserID,
			CandidateName:   application.CandidateName,
			ResumeId:        application.ResumeID,
		}
		if evaluation, ok := byApplicationID[application.ApplicationID]; ok {
			item.EvaluationId = evaluation.ID
			item.EvaluationVersion = evaluation.EvaluationVersion
			item.OverallScore = evaluation.OverallScore
			item.Recommendation = evaluation.Recommendation
			item.Summary = evaluation.Summary
			item.EvaluatedAt = formatTime(evaluation.EvaluatedAt)
			item.HasEvaluation = true
		} else {
			missing = append(missing, application.ApplicationID)
		}
		candidates = append(candidates, item)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].GetHasEvaluation() != candidates[j].GetHasEvaluation() {
			return candidates[i].GetHasEvaluation()
		}
		if candidates[i].GetOverallScore() != candidates[j].GetOverallScore() {
			return candidates[i].GetOverallScore() > candidates[j].GetOverallScore()
		}
		return candidates[i].GetApplicationId() < candidates[j].GetApplicationId()
	})
	return &pb.CompareCandidatesForJobResponse{Code: errs.OK, Msg: "success", JobId: req.GetJobId(), Candidates: candidates, MissingApplicationIds: missing}, nil
}

func (s nativeRecruitingIntelligenceService) resolveResumeProfileAccess(ctx context.Context, req *pb.GetResumeProfileRequest) (int64, *pb.GetResumeProfileResponse) {
	if req.GetApplicationId() > 0 {
		if _, authErr := s.authorizeRecruitingApplication(ctx, req.GetStaffUserId(), req.GetApplicationId()); authErr != nil {
			return 0, recruitingResumeAuthResponse(authErr)
		}
		application, found, err := s.store.GetRecruitingApplicationByID(ctx, req.GetApplicationId())
		if err != nil {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}
		}
		if !found || application.ResumeID <= 0 {
			return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "application resume not found"}
		}
		if req.GetResumeId() > 0 && req.GetResumeId() != application.ResumeID {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "application_id and resume_id refer to different resumes"}
		}
		if req.GetProfileId() > 0 {
			profile, profileFound, profileErr := s.store.GetRecruitingResumeProfileByID(ctx, req.GetProfileId())
			if profileErr != nil {
				return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: profileErr.Error()}
			}
			if !profileFound {
				return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "resume profile not found"}
			}
			if profile.ResumeID != application.ResumeID {
				return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "profile_id does not belong to requested resume/application"}
			}
		}
		return application.ResumeID, nil
	}

	resumeID := req.GetResumeId()
	if req.GetProfileId() > 0 {
		profile, found, err := s.store.GetRecruitingResumeProfileByID(ctx, req.GetProfileId())
		if err != nil {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}
		}
		if !found {
			return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "resume profile not found"}
		}
		if resumeID > 0 && resumeID != profile.ResumeID {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "profile_id does not belong to requested resume/application"}
		}
		resumeID = profile.ResumeID
	}
	if resumeID <= 0 {
		return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "resume_id, profile_id, or application_id is required"}
	}
	application, found, err := s.store.GetLatestRecruitingApplicationByResumeID(ctx, resumeID)
	if err != nil {
		return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}
	}
	if !found {
		return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "application context not found for resume"}
	}
	if _, authErr := s.authorizeRecruitingApplication(ctx, req.GetStaffUserId(), application.ApplicationID); authErr != nil {
		return 0, recruitingResumeAuthResponse(authErr)
	}
	return resumeID, nil
}

func recruitingResumeAuthResponse(authErr *recruitingAuthError) *pb.GetResumeProfileResponse {
	return &pb.GetResumeProfileResponse{Code: authErr.code, Msg: authErr.message}
}

func recruitingMatchAuthResponse(authErr *recruitingAuthError) *pb.GetCandidateMatchEvaluationResponse {
	return &pb.GetCandidateMatchEvaluationResponse{Code: authErr.code, Msg: authErr.message}
}

func recruitingComparisonAuthResponse(jobID int64, authErr *recruitingAuthError) *pb.CompareCandidatesForJobResponse {
	return &pb.CompareCandidatesForJobResponse{Code: authErr.code, Msg: authErr.message, JobId: jobID}
}

func recruitingResumeProfileSnapshotPB(snapshot RecruitingResumeProfileSnapshot) *pb.ResumeProfileSnapshotInfo {
	out := &pb.ResumeProfileSnapshotInfo{
		ParseRun:    recruitingResumeParseRunPB(snapshot.ParseRun),
		Profile:     recruitingResumeProfilePB(snapshot.Profile),
		Educations:  make([]*pb.ResumeEducationInfo, 0, len(snapshot.Educations)),
		Experiences: make([]*pb.ResumeExperienceInfo, 0, len(snapshot.Experiences)),
		Projects:    make([]*pb.ResumeProjectInfo, 0, len(snapshot.Projects)),
		Skills:      make([]*pb.ResumeSkillInfo, 0, len(snapshot.Skills)),
	}
	for _, row := range snapshot.Educations {
		out.Educations = append(out.Educations, recruitingResumeEducationPB(row))
	}
	for _, row := range snapshot.Experiences {
		out.Experiences = append(out.Experiences, recruitingResumeExperiencePB(row))
	}
	for _, row := range snapshot.Projects {
		out.Projects = append(out.Projects, recruitingResumeProjectPB(row))
	}
	for _, row := range snapshot.Skills {
		out.Skills = append(out.Skills, recruitingResumeSkillPB(row))
	}
	return out
}

func recruitingResumeParseRunPB(row RecruitingResumeParseRunRow) *pb.ResumeParseRunInfo {
	out := &pb.ResumeParseRunInfo{
		Id:            row.ID,
		ResumeId:      row.ResumeID,
		UserId:        row.UserID,
		Status:        row.Status,
		ParserVersion: row.ParserVersion,
		InputHash:     row.InputHash,
		ErrorMessage:  row.ErrorMessage,
		StartedAt:     formatTime(row.StartedAt),
		CompletedAt:   formatTimePtr(row.CompletedAt),
		CreatedAt:     formatTime(row.CreatedAt),
		UpdatedAt:     formatTime(row.UpdatedAt),
	}
	if row.AgentRunID != nil {
		out.AgentRunId = *row.AgentRunID
	}
	return out
}

func recruitingResumeProfilePB(row RecruitingResumeProfileRow) *pb.ResumeProfileInfo {
	return &pb.ResumeProfileInfo{
		Id:                   row.ID,
		ResumeId:             row.ResumeID,
		UserId:               row.UserID,
		ParseRunId:           row.ParseRunID,
		Version:              row.Version,
		IsCurrent:            row.IsCurrent,
		FullName:             row.FullName,
		Email:                row.Email,
		Phone:                row.Phone,
		Location:             row.Location,
		Headline:             row.Headline,
		Summary:              row.Summary,
		TotalExperienceYears: row.TotalExperienceYears,
		HighestDegree:        row.HighestDegree,
		RawJson:              row.RawJSON,
		CreatedAt:            formatTime(row.CreatedAt),
		UpdatedAt:            formatTime(row.UpdatedAt),
	}
}

func recruitingResumeEducationPB(row RecruitingResumeEducationRow) *pb.ResumeEducationInfo {
	return &pb.ResumeEducationInfo{Id: row.ID, School: row.School, Degree: row.Degree, Major: row.Major, StartDate: formatTimePtr(row.StartDate), EndDate: formatTimePtr(row.EndDate), Description: row.Description, SortOrder: row.SortOrder}
}

func recruitingResumeExperiencePB(row RecruitingResumeExperienceRow) *pb.ResumeExperienceInfo {
	return &pb.ResumeExperienceInfo{Id: row.ID, Company: row.Company, Title: row.Title, Location: row.Location, StartDate: formatTimePtr(row.StartDate), EndDate: formatTimePtr(row.EndDate), IsCurrent: row.IsCurrent, Description: row.Description, AchievementsJson: row.AchievementsJSON, SortOrder: row.SortOrder}
}

func recruitingResumeProjectPB(row RecruitingResumeProjectRow) *pb.ResumeProjectInfo {
	return &pb.ResumeProjectInfo{Id: row.ID, Name: row.Name, Role: row.Role, StartDate: formatTimePtr(row.StartDate), EndDate: formatTimePtr(row.EndDate), Description: row.Description, TechnologiesJson: row.TechnologiesJSON, HighlightsJson: row.HighlightsJSON, SortOrder: row.SortOrder}
}

func recruitingResumeSkillPB(row RecruitingResumeSkillRow) *pb.ResumeSkillInfo {
	return &pb.ResumeSkillInfo{Id: row.ID, Name: row.Name, Category: row.Category, Level: row.Level, Years: row.Years, Evidence: row.Evidence, SortOrder: row.SortOrder}
}

func recruitingCandidateMatchSnapshotPB(snapshot RecruitingCandidateMatchSnapshot) *pb.CandidateMatchEvaluationSnapshotInfo {
	out := &pb.CandidateMatchEvaluationSnapshotInfo{
		Evaluation: recruitingCandidateMatchEvaluationPB(snapshot.Evaluation),
		Evidence:   make([]*pb.CandidateMatchEvidenceInfo, 0, len(snapshot.Evidence)),
	}
	for _, row := range snapshot.Evidence {
		out.Evidence = append(out.Evidence, recruitingCandidateMatchEvidencePB(row))
	}
	return out
}

func recruitingCandidateMatchEvaluationPB(row RecruitingCandidateMatchEvaluationRow) *pb.CandidateMatchEvaluationInfo {
	out := &pb.CandidateMatchEvaluationInfo{
		Id:                      row.ID,
		ApplicationId:           row.ApplicationID,
		JobId:                   row.JobID,
		CandidateUserId:         row.CandidateUserID,
		ResumeProfileId:         row.ResumeProfileID,
		EvaluationVersion:       row.EvaluationVersion,
		IsLatest:                row.IsLatest,
		OverallScore:            row.OverallScore,
		Recommendation:          row.Recommendation,
		Summary:                 row.Summary,
		StrengthsJson:           row.StrengthsJSON,
		RisksJson:               row.RisksJSON,
		MissingRequirementsJson: recruitingMissingRequirementsJSON(row.ScoreBreakdownJSON),
		ScoreBreakdownJson:      row.ScoreBreakdownJSON,
		ModelName:               row.ModelName,
		EvaluatedAt:             formatTime(row.EvaluatedAt),
		CreatedAt:               formatTime(row.CreatedAt),
		UpdatedAt:               formatTime(row.UpdatedAt),
		DimensionsJson:          recruitingDimensionsJSON(row.ScoreBreakdownJSON),
	}
	if row.AgentRunID != nil {
		out.AgentRunId = *row.AgentRunID
	}
	return out
}

func recruitingCandidateMatchEvidencePB(row RecruitingCandidateMatchEvidenceRow) *pb.CandidateMatchEvidenceInfo {
	out := &pb.CandidateMatchEvidenceInfo{
		Id:           row.ID,
		EvidenceType: row.EvidenceType,
		Dimension:    row.Dimension,
		SourceTable:  row.SourceTable,
		Snippet:      row.Snippet,
		Weight:       row.Weight,
		ScoreImpact:  row.ScoreImpact,
		MetadataJson: row.MetadataJSON,
		CreatedAt:    formatTime(row.CreatedAt),
	}
	if row.SourceID != nil {
		out.SourceId = *row.SourceID
	}
	return out
}

func recruitingMissingRequirementsJSON(scoreBreakdown string) string {
	var payload struct {
		MissingRequirements []string `json:"missing_requirements"`
		RequirementResults  []struct {
			RequirementID string `json:"requirement_id"`
			Status        string `json:"status"`
		} `json:"requirement_results"`
	}
	if err := json.Unmarshal([]byte(scoreBreakdown), &payload); err != nil {
		return "[]"
	}
	missing := payload.MissingRequirements
	if len(missing) == 0 {
		for _, result := range payload.RequirementResults {
			if result.Status == "missing" || result.Status == "conflict" {
				missing = append(missing, result.RequirementID)
			}
		}
	}
	raw, err := json.Marshal(missing)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func recruitingDimensionsJSON(scoreBreakdown string) string {
	var payload struct {
		Dimensions []any `json:"dimensions"`
	}
	if err := json.Unmarshal([]byte(scoreBreakdown), &payload); err != nil || payload.Dimensions == nil {
		return "[]"
	}
	raw, err := json.Marshal(payload.Dimensions)
	if err != nil {
		return "[]"
	}
	return string(raw)
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
	store AIStore
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
		return &pb.ListProvidersResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	rows, total, err := s.store.ListLlmProviders(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &pb.ListProvidersResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeLlmConfigService) ListModels(ctx context.Context, req *pb.ListModelsRequest) (*pb.ListModelsResponse, error) {
	if s.store == nil {
		return &pb.ListModelsResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	rows, total, err := s.store.ListLlmModels(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetProviderId())
	if err != nil {
		return nil, err
	}
	return &pb.ListModelsResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativePromptService) ListPromptTemplates(ctx context.Context, req *pb.ListPromptTemplatesRequest) (*pb.ListPromptTemplatesResponse, error) {
	if s.store == nil {
		return &pb.ListPromptTemplatesResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	rows, total, err := s.store.ListPromptTemplates(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetAgentType())
	if err != nil {
		return nil, err
	}
	return &pb.ListPromptTemplatesResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeAgentConfigService) ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	if s.store == nil {
		return &pb.ListAgentsResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	rows, total, err := s.store.ListAgentConfigs(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetAgentType())
	if err != nil {
		return nil, err
	}
	return &pb.ListAgentsResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (nativeAgentConfigService) ListCapabilities(_ context.Context, req *pb.ListCapabilitiesRequest) (*pb.ListCapabilitiesResponse, error) {
	return &pb.ListCapabilitiesResponse{Code: 0, Msg: "success", List: builtinCapabilities(req.GetAgentType())}, nil
}
func (s nativeMCPService) ListMCPServers(ctx context.Context, req *pb.ListMCPServersRequest) (*pb.ListMCPServersResponse, error) {
	if s.store == nil {
		return &pb.ListMCPServersResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	rows, total, err := s.store.ListMCPServers(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &pb.ListMCPServersResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeMCPService) ListMCPToolPolicies(ctx context.Context, req *pb.ListMCPToolPoliciesRequest) (*pb.ListMCPToolPoliciesResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.ListMCPToolPoliciesResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.ListMCPToolPolicies(ctx, req)
}
func (s nativeMCPService) ListMCPToolLogs(ctx context.Context, req *pb.ListMCPToolLogsRequest) (*pb.ListMCPToolLogsResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.ListMCPToolLogsResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.ListMCPToolLogs(ctx, req)
}
func (s nativeSkillService) ListSkills(ctx context.Context, req *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error) {
	store, ok := s.store.(skillGovernanceStore)
	if !ok {
		return &pb.ListSkillsResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.ListSkills(ctx, req)
}
func (s nativeAgentSkillService) ListAgentSkills(ctx context.Context, req *pb.ListAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	if s.store == nil {
		return &pb.ListAgentSkillsResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	rows, total, err := s.store.ListAgentSkills(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetKeyword(), req.GetEnabledOnly())
	if err != nil {
		return nil, err
	}
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeAgentSkillService) ListAvailableAgentSkills(ctx context.Context, req *pb.ListAvailableAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	if s.store == nil {
		return &pb.ListAgentSkillsResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	rows, total, err := s.store.ListAgentSkills(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), "", true)
	if err != nil {
		return nil, err
	}
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeEmbeddingConfigService) ListEmbeddingProviders(ctx context.Context, req *pb.ListEmbeddingProvidersRequest) (*pb.ListEmbeddingProvidersResponse, error) {
	if s.store == nil {
		return &pb.ListEmbeddingProvidersResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
	}
	rows, total, err := s.store.ListEmbeddingProviders(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &pb.ListEmbeddingProvidersResponse{Code: 0, Msg: "success", Total: total, List: rows}, nil
}
func (s nativeEmbeddingConfigService) ListEmbeddingModels(ctx context.Context, req *pb.ListEmbeddingModelsRequest) (*pb.ListEmbeddingModelsResponse, error) {
	if s.store == nil {
		return &pb.ListEmbeddingModelsResponse{Code: configCodeUnavailable, Msg: "ai configuration store is not configured"}, nil
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
