package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"smart-recruit-ai-agent-service/internal/application/hr_tools"
	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	"smart-recruit-ai-agent-service/internal/domain/model"
	"smart-recruit-ai-agent-service/internal/domain/policy"
	mcpinfra "smart-recruit-ai-agent-service/internal/infrastructure/mcp"
	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
	aiagentruntime "smart-recruit-ai-agent-service/internal/runtime"
	commonsai "smart-recruit-commons/ai"
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

type ChatCompletionOptions struct {
	TemperatureOverride *float64
}

type RuntimeOptionsChatProvider interface {
	CompleteWithOptions(ctx context.Context, prompt string, modelID int64, opts ChatCompletionOptions) (string, error)
}

// RecruitingToolChatProvider runs the model-driven tool-calling loop for HR chat.
type RecruitingToolChatProvider interface {
	ChatWithRecruitingTools(
		ctx context.Context,
		modelID int64,
		opts ChatCompletionOptions,
		messages []*schema.Message,
		tools []*schema.ToolInfo,
		executor commonsai.ToolRunner,
		hrID int64,
		onDelta func(string) error,
		onToolExecuted commonsai.ToolTraceCallback,
		onStatus func(eventType, eventMessage, errorType, toolName string) error,
	) (string, commonsai.ToolMetadata, error)
}

type agentSkillDetailStore interface {
	GetAgentSkill(context.Context, *pb.GetAgentSkillRequest) (*pb.AgentSkillResponse, error)
	ListAgentSkillVersions(context.Context, *pb.ListAgentSkillVersionsRequest) (*pb.ListAgentSkillVersionsResponse, error)
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
	AppendToolTrace(ctx context.Context, ownerID int64, trace ToolTraceRow) (ToolTraceRow, error)
	AppendAgentRunStep(ctx context.Context, step AgentRunStepRow) (AgentRunStepRow, error)
	ListAgentRunSteps(ctx context.Context, runID int64) ([]AgentRunStepRow, error)
	CreateAgentRun(ctx context.Context, run AgentRunRow) (AgentRunRow, bool, error)
	ListAgentRuns(ctx context.Context, ownerID, sessionID int64) ([]AgentRunRow, error)
	GetAgentRun(ctx context.Context, ownerID, runID int64) (AgentRunRow, bool, error)
	GetActiveAgentRun(ctx context.Context, ownerID, sessionID int64) (AgentRunRow, bool, error)
	UpdateAgentRunStatus(ctx context.Context, ownerID, runID int64, status string) (AgentRunRow, bool, error)
	UpdateAgentRunPlan(ctx context.Context, ownerID, runID int64, planJSON, optionContextJSON string) (AgentRunRow, bool, error)
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

type candidateContextStore interface {
	LoadCandidateRuntimeContext(ctx context.Context, userID int64, limit int32) (CandidateRuntimeContext, error)
}

type candidateUsageAuditStore interface {
	RecordCandidateUsageAudit(ctx context.Context, row CandidateUsageAuditRow) (int64, error)
}

type applicationSnapshotClient interface {
	GetApplicationSnapshot(ctx context.Context, in *pb.GetApplicationSnapshotRequest, opts ...gogrpc.CallOption) (*pb.GetApplicationSnapshotResponse, error)
}

type RuntimeDeps struct {
	Store            AIStore
	Provider         ChatProvider
	RecruitingPolicy recruitingruntime.RuntimePolicy
	EmbeddingWorker  bool
	AgentRunWorker   bool
	RuntimeName      string
	EmbeddingConfigs pb.EmbeddingConfigServiceServer
	Auth             pb.AuthServiceClient
	Applications     pb.ApplicationOwnerServiceClient
	AppList          pb.ApplicationServiceClient
	Jobs             pb.JobServiceClient
	MCPRunner        mcpinfra.Runner
	EmbeddingRunner  embeddinginfra.EmbeddingRunner
}

type ChatSessionRow struct {
	ID            int64
	Title         string
	ApplicationID int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ChatMessageRow struct {
	ID              int64
	OwnerRole       int32
	OwnerID         int64
	SessionID       int64
	Role            string
	Content         string
	ProcessContent  string
	ModelID         int64
	ModelName       string
	AgentSkillIDs   []int64
	AgentSkillNames []string
	CreatedAt       time.Time
}

type ToolTraceRow struct {
	ID              int64
	SessionID       int64
	AgentRunID      int64
	AgentRunStepID  int64
	ToolCallID      string
	ToolName        string
	ArgsJSON        string
	ResultContent   string
	Status          string
	DurationMs      int64
	ErrorMsg        string
	CreatedAt       time.Time
}

type AgentRunStepRow struct {
	ID               int64
	RunID            int64
	StepIndex        int32
	StepType         string
	CapabilitySource string
	CapabilityKey    string
	ToolName         string
	InputJSON        string
	OutputJSON       string
	Status           string
	DurationMs       int64
	ErrorMsg         string
	StartedAt        time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CandidateRuntimeContext struct {
	Applications []CandidateApplicationContext `json:"applications"`
	Resume       CandidateResumeContext        `json:"resume"`
	Jobs         []CandidateJobContext         `json:"jobs"`
	Interviews   []CandidateInterviewContext   `json:"interviews"`
	Offers       []CandidateOfferContext       `json:"offers"`
}

type CandidateApplicationContext struct {
	ApplicationID int64  `json:"application_id"`
	JobID         int64  `json:"job_id"`
	JobTitle      string `json:"job_title"`
	Status        int32  `json:"status"`
	StatusKey     string `json:"status_key,omitempty"`
	StatusText    string `json:"status_text,omitempty"`
	RoundNo       int32  `json:"round_no"`
	AppliedAt     string `json:"applied_at,omitempty"`
}

type CandidateResumeContext struct {
	Available  bool   `json:"available"`
	ResumeID   int64  `json:"resume_id,omitempty"`
	FileName   string `json:"file_name,omitempty"`
	TextLength int    `json:"text_length,omitempty"`
	Summary    string `json:"summary,omitempty"`
	Message    string `json:"message,omitempty"`
}

type CandidateJobContext struct {
	JobID       int64  `json:"job_id"`
	Title       string `json:"title"`
	Department  string `json:"department,omitempty"`
	Location    string `json:"location,omitempty"`
	SalaryRange string `json:"salary_range,omitempty"`
	Status      int32  `json:"status"`
	StatusText  string `json:"status_text,omitempty"`
	HasApplied  bool   `json:"has_applied"`
}

type CandidateInterviewContext struct {
	InterviewID   int64  `json:"interview_id"`
	ApplicationID int64  `json:"application_id"`
	JobTitle      string `json:"job_title,omitempty"`
	RoundNo       int32  `json:"round_no"`
	Title         string `json:"title,omitempty"`
	Mode          string `json:"mode,omitempty"`
	ScheduledAt   string `json:"scheduled_at,omitempty"`
	Status        string `json:"status,omitempty"`
	CandidateNote string `json:"candidate_note,omitempty"`
}

type CandidateOfferContext struct {
	OfferID       int64  `json:"offer_id"`
	ApplicationID int64  `json:"application_id"`
	JobID         int64  `json:"job_id"`
	Title         string `json:"title,omitempty"`
	Status        string `json:"status,omitempty"`
	SalaryRange   string `json:"salary_range,omitempty"`
	WorkLocation  string `json:"work_location,omitempty"`
	StartDate     string `json:"start_date,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

type CandidateUsageAuditRow struct {
	UserID          int64
	ServiceType     string
	Endpoint        string
	Provider        string
	Model           string
	RequestChars    int
	ResponseChars   int
	EstimatedTokens int
	Status          string
	ErrorCode       string
	CostMs          int
	RequestID       string
	IP              string
	RoleKeys        []string
	PermissionKey   string
	ScopeKeys       []string
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
	PlanJSON          string
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

type agentRunDurablePayload struct {
	Message                      string   `json:"message,omitempty"`
	ActionType                   string   `json:"action_type,omitempty"`
	ActionPayloadJSON            string   `json:"action_payload_json,omitempty"`
	ApplicationID                int64    `json:"application_id,omitempty"`
	ModelID                      int64    `json:"model_id,omitempty"`
	SkillCapabilityKeys          []string `json:"skill_capability_keys,omitempty"`
	AgentSkillIDs                []int64  `json:"agent_skill_ids,omitempty"`
	AgentSkillSelectionConfirmed bool     `json:"agent_skill_selection_confirmed,omitempty"`
	AgentSkillSelectionMessageID int64    `json:"agent_skill_selection_message_id,omitempty"`
	ConfirmationPayloadJSON      string   `json:"confirmation_payload_json,omitempty"`
	ConfirmationClientRequestID  string   `json:"confirmation_client_request_id,omitempty"`
}

type RecruitingApplicationContext struct {
	ApplicationID   int64
	JobID           int64
	CandidateUserID int64
	CandidateName   string
	ResumeID        int64
	IsCurrent       int32
}

type RecruitingCandidateProfileRow struct {
	ID             uint64
	RealName       string
	Phone          string
	Education      string
	School         string
	WorkExperience string
	Skills         string
	IsComplete     int32
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

type RecruitingResumeSource struct {
	ResumeID   int64
	UserID     int64
	FileName   string
	ParsedText string
}

type RecruitingResumeProfileDraft struct {
	ResumeID             int64
	UserID               int64
	ParserVersion        string
	InputHash            string
	RawJSON              string
	FullName             string
	Email                string
	Phone                string
	Location             string
	Headline             string
	Summary              string
	TotalExperienceYears float64
	HighestDegree        string
	Educations           []RecruitingResumeEducationRow
	Experiences          []RecruitingResumeExperienceRow
	Projects             []RecruitingResumeProjectRow
	Skills               []RecruitingResumeSkillRow
}

type RecruitingJobContext struct {
	JobID        int64
	Title        string
	Department   string
	Location     string
	Description  string
	Requirements string
}

type RecruitingMatchSource struct {
	Application          RecruitingApplicationContext
	Job                  RecruitingJobContext
	Profile              RecruitingResumeProfileSnapshot
	CandidateProfile     *RecruitingCandidateProfileRow
	ApplicationEducation string
	ResumeParsedText     string
}

type RecruitingCandidateMatchDraft struct {
	ApplicationID      int64
	JobID              int64
	CandidateUserID    int64
	ResumeProfileID    uint64
	AgentRunID         *uint64
	OverallScore       float64
	Recommendation     string
	Summary            string
	StrengthsJSON      string
	RisksJSON          string
	ScoreBreakdownJSON string
	ModelName          string
	Evidence           []RecruitingCandidateMatchEvidenceRow
	ScorerVersion      string
	RequirementCount   int
	FallbackUsed       bool
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

	candidateAssistantAgentType            = "candidate_assistant"
	candidatePromptRoleSystem              = "system"
	candidateChatContextPageSize           = int32(100)
	candidateRuntimeContextLimit           = int32(20)
	candidateChatEmptyMessageError         = "message is required"
	candidateSuggestedQuestionsStartMarker = "<<<CANDIDATE_SUGGESTED_QUESTIONS_JSON>>>"
	candidateSuggestedQuestionsEndMarker   = "<<<END_CANDIDATE_SUGGESTED_QUESTIONS_JSON>>>"

	candidateSystemPrompt = "你是智能招聘系统的候选人端 AI 助手。你只服务当前登录候选人，回答应围绕候选人自己的求职、简历、岗位、投递、面试和 Offer 相关问题。\n" +
		"必须保护招聘系统数据边界：不要透露 HR 内部备注、其他候选人信息、未授权的招聘数据或系统实现细节。\n" +
		"当缺少实时工具或数据时，明确说明当前无法读取实时系统数据，并给出安全、可执行的求职建议。"

	hrRecruitingAgentType             = "hr_recruiting_agent"
	hrCandidateSearchCapability       = "candidate_search"
	hrApplicationSnapshotTool         = "get_application_snapshot"
	hrRuntimePromptRoleSystem         = "system"
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
	mcpRunner := deps.MCPRunner
	if mcpRunner == nil {
		mcpRunner = mcpinfra.NewRunner()
	}
	var embeddingService *embeddinginfra.EmbeddingService
	if store, ok := deps.Store.(embeddinginfra.EmbeddingStore); ok {
		embeddingService = embeddinginfra.NewEmbeddingService(store, deps.EmbeddingRunner)
	}
	ai := newNativeAIServiceWithRunner(deps.Store, deps.Provider, deps.Applications, deps.Jobs, deps.AppList, mcpRunner, embeddingService)
	embedding := deps.EmbeddingConfigs
	if embedding == nil {
		embedding = nativeEmbeddingConfigService{store: deps.Store, embedding: embeddingService}
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
		MCP:                    nativeMCPService{store: deps.Store, runner: mcpRunner},
		Skill:                  nativeSkillService{store: deps.Store},
		AgentSkill:             nativeAgentSkillService{store: deps.Store, embedding: embeddingService},
		RecruitingIntelligence: nativeRecruitingIntelligenceService{store: recruitingStore, provider: deps.Provider, structured: newRecruitingStructuredRuntime(deps.Store, deps.Provider, deps.RecruitingPolicy), policy: deps.RecruitingPolicy, auth: deps.Auth, applications: deps.Applications, jobs: deps.Jobs},
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
	return newNativeAIService(store, provider, nil, nil, nil)
}

func newNativeAIService(store AIStore, provider ChatProvider, applications applicationSnapshotClient, jobs hr_tools.JobClient, appList hr_tools.ApplicationListClient) *nativeAIService {
	return newNativeAIServiceWithRunner(store, provider, applications, jobs, appList, nil, nil)
}

func newNativeAIServiceWithRunner(store AIStore, provider ChatProvider, applications applicationSnapshotClient, jobs hr_tools.JobClient, appList hr_tools.ApplicationListClient, mcpRunner mcpinfra.Runner, embedding *embeddinginfra.EmbeddingService) *nativeAIService {
	return &nativeAIService{store: store, provider: provider, applications: applications, jobs: jobs, appList: appList, mcpRunner: mcpRunner, embedding: embedding, eventHub: newAgentRunEventHub()}
}

type nativeAIService struct {
	pb.UnimplementedAIServiceServer
	store           AIStore
	provider        ChatProvider
	applications    applicationSnapshotClient
	jobs            hr_tools.JobClient
	appList         hr_tools.ApplicationListClient
	mcpRunner       mcpinfra.Runner
	embedding       *embeddinginfra.EmbeddingService
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
	result, err := s.runHRChatRuntime(ctx, req, nil)
	if err != nil {
		return nil, err
	}
	if result.providerUnavailable && !result.fallbackUsed {
		return &pb.ChatResponse{Code: configCodeUnavailable, Msg: errAIProviderRequired.Error(), CreatedAt: formatTime(time.Now()), SessionId: result.session.ID, ApplicationId: req.GetApplicationId(), ContextUsage: result.contextUsage}, nil
	}
	return &pb.ChatResponse{Code: 0, Msg: "success", Reply: result.reply, CreatedAt: formatTime(time.Now()), SessionId: result.session.ID, ApplicationId: req.GetApplicationId(), CandidateName: result.candidateName, JobTitle: result.jobTitle, Status: result.status, ContextUsage: result.contextUsage}, nil
}

func (s *nativeAIService) ChatStream(req *pb.ChatRequest, stream gogrpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	if req == nil {
		return errors.New("chat request is required")
	}
	result, err := s.runHRChatRuntime(stream.Context(), req, func(event *pb.ChatStreamResponse) error {
		event.SessionId = firstNonZeroInt64(event.GetSessionId(), req.GetSessionId())
		event.ApplicationId = req.GetApplicationId()
		if event.CreatedAt == "" {
			event.CreatedAt = formatTime(time.Now())
		}
		return stream.Send(event)
	})
	if err != nil {
		return err
	}
	if result.providerUnavailable && !result.fallbackUsed {
		return stream.Send(&pb.ChatStreamResponse{Code: configCodeUnavailable, Msg: errAIProviderRequired.Error(), Done: true, SessionId: result.session.ID, ApplicationId: req.GetApplicationId(), CreatedAt: formatTime(time.Now()), EventType: "error", EventMessage: errAIProviderRequired.Error(), ErrorType: "AI_PROVIDER_UNAVAILABLE", ContextUsage: result.contextUsage})
	}
	return stream.Send(&pb.ChatStreamResponse{Code: 0, Msg: "success", Delta: result.reply, Done: true, SessionId: result.session.ID, ApplicationId: req.GetApplicationId(), CandidateName: result.candidateName, JobTitle: result.jobTitle, Status: result.status, CreatedAt: formatTime(time.Now()), EventType: "done", EventMessage: "completed", ContextUsage: result.contextUsage})
}

type hrChatStreamEmitter func(*pb.ChatStreamResponse) error

type hrChatRuntimeOptions struct {
	reuseExistingUserMessage bool
	agentRunID               int64
}

type hrChatRuntimeResult struct {
	session             ChatSessionRow
	reply               string
	contextUsage        *pb.ContextUsageInfo
	candidateName       string
	jobTitle            string
	status              int32
	providerUnavailable bool
	fallbackUsed        bool
}

type hrRuntimeGovernanceContext struct {
	Agent                        *pb.AgentConfigInfo
	Prompt                       *pb.PromptTemplateInfo
	CapabilityKeys               []string
	ToolNames                    []string
	ExecutableToolNames          []string
	SelectedAgentSkills          []hrRuntimeAgentSkill
	AgentSkillSelectionMode      string
	AgentSkillSelectionConfirmed bool
	AgentSkillSelectionMessageID int64
}

type hrRuntimeAgentSkill struct {
	ID                   int64    `json:"id"`
	Name                 string   `json:"name"`
	DisplayName          string   `json:"display_name,omitempty"`
	Description          string   `json:"description,omitempty"`
	SkillMD              string   `json:"skill_md,omitempty"`
	Manual               bool     `json:"manual"`
	Reason               string   `json:"reason,omitempty"`
	RiskLevel            string   `json:"risk_level,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
}

type hrRuntimeAgentConfigStore interface {
	GetAgentConfig(context.Context, *pb.GetAgentConfigRequest) (*pb.GetAgentConfigResponse, error)
}

type hrRuntimePromptTemplateStore interface {
	GetRuntimePromptTemplateByID(ctx context.Context, id int64) (*pb.PromptTemplateInfo, bool, error)
}

func (s *nativeAIService) runHRChatRuntime(ctx context.Context, req *pb.ChatRequest, emit hrChatStreamEmitter) (hrChatRuntimeResult, error) {
	return s.runHRChatRuntimeWithOptions(ctx, req, emit, hrChatRuntimeOptions{})
}

func (s *nativeAIService) runHRChatRuntimeWithOptions(ctx context.Context, req *pb.ChatRequest, emit hrChatStreamEmitter, opts hrChatRuntimeOptions) (hrChatRuntimeResult, error) {
	session, err := s.ensureSession(ctx, ownerRoleHR, req.GetHrId(), req.GetSessionId(), req.GetApplicationId(), req.GetMessage())
	if err != nil {
		return hrChatRuntimeResult{}, err
	}
	result := hrChatRuntimeResult{session: session}
	send := func(event *pb.ChatStreamResponse) error {
		if emit == nil || event == nil {
			return nil
		}
		event.SessionId = session.ID
		event.ApplicationId = req.GetApplicationId()
		event.CandidateName = result.candidateName
		event.JobTitle = result.jobTitle
		event.Status = result.status
		return emit(event)
	}
	if err := send(&pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "thinking", EventMessage: "planning HR recruiting context", CreatedAt: formatTime(time.Now())}); err != nil {
		return result, err
	}
	governance, err := s.loadHRRuntimeGovernance(ctx, req)
	if err != nil {
		return result, err
	}

	history, err := s.hrRecentMessages(ctx, req.GetHrId(), session.ID)
	if err != nil {
		return result, err
	}
	var userMessage ChatMessageRow
	if opts.reuseExistingUserMessage {
		userMessage = findHRUserMessage(history, req.GetMessage())
	}
	if strings.TrimSpace(req.GetMessage()) != "" && s.store != nil {
		if userMessage.ID == 0 {
			userMessage, err = s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: ownerRoleHR, OwnerID: req.GetHrId(), SessionID: session.ID, Role: "user", Content: req.GetMessage(), ModelID: req.GetModelId(), AgentSkillIDs: hrRuntimeAgentSkillIDs(governance), AgentSkillNames: hrRuntimeAgentSkillNames(governance)})
			if err != nil {
				return result, err
			}
		}
	}

	// Application snapshot + MCP remain available as pre-context tools.
	traces, err := s.executeHRContextTools(ctx, req, session.ID, opts.agentRunID, governance, send)
	if err != nil {
		return result, err
	}
	for _, trace := range traces {
		if trace.ToolName == "get_application_snapshot" {
			result.candidateName, result.jobTitle, result.status = extractApplicationTraceMetadata(trace.ResultContent)
		}
	}

	executor := &hr_tools.Executor{Jobs: s.jobs, Applications: s.appList, Snapshots: s.applications}
	toolSchemas := hrRecruitingToolSchemas(governance.ExecutableToolNames)
	toolProvider, hasToolProvider := s.provider.(RecruitingToolChatProvider)
	canRunTools := len(toolSchemas) > 0 && (s.jobs != nil || s.appList != nil || s.applications != nil)

	var reply string
	if hasToolProvider && canRunTools {
		messages := buildHRToolCallingMessages(req, history, userMessage, traces, governance)
		result.contextUsage = estimateHRContextUsage(req.GetModelId(), renderHRProviderPrompt(req, history, userMessage, traces, governance), req.GetMessage(), traces)
		if err := send(&pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "context_usage", EventMessage: "context usage estimated", ContextUsage: result.contextUsage, CreatedAt: formatTime(time.Now())}); err != nil {
			return result, err
		}
		onStatus := func(eventType, eventMessage, errorType, toolName string) error {
			event := &pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: eventType, EventMessage: eventMessage, ToolName: toolName, CreatedAt: formatTime(time.Now())}
			if errorType != "" {
				event.ErrorType = errorType
				if eventType == "error" {
					event.Msg = eventMessage
				}
			}
			return send(event)
		}
		onTool := func(toolCallID, toolName, argsJSON, resultContent string, duration time.Duration, execErr error) {
			trace := ToolTraceRow{
				SessionID:     session.ID,
				AgentRunID:    opts.agentRunID,
				ToolCallID:    toolCallID,
				ToolName:      toolName,
				ArgsJSON:      argsJSON,
				ResultContent: resultContent,
				DurationMs:    duration.Milliseconds(),
				CreatedAt:     time.Now(),
			}
			if execErr != nil {
				trace.ErrorMsg = execErr.Error()
				trace.Status = "error"
			} else {
				trace.Status = "success"
			}
			persisted, persistErr := s.persistHRToolTrace(ctx, req.GetHrId(), trace)
			if persistErr == nil && persisted.ID != 0 {
				trace = persisted
			}
			traces = append(traces, trace)
		}
		var deltaBuilder strings.Builder
		toolReply, _, toolErr := toolProvider.ChatWithRecruitingTools(
			ctx,
			req.GetModelId(),
			hrRuntimeCompletionOptions(governance),
			messages,
			toolSchemas,
			executor,
			req.GetHrId(),
			func(delta string) error {
				deltaBuilder.WriteString(delta)
				return send(&pb.ChatStreamResponse{Code: 0, Msg: "success", Delta: delta, EventType: "generating", EventMessage: "streaming answer", CreatedAt: formatTime(time.Now())})
			},
			onTool,
			onStatus,
		)
		if toolErr != nil {
			if errors.Is(toolErr, errAIProviderRequired) || strings.Contains(toolErr.Error(), "api_key") || strings.Contains(toolErr.Error(), "provider") {
				result.providerUnavailable = true
			}
			if !hasUsefulToolResults(traces) {
				if result.providerUnavailable {
					return result, nil
				}
				return result, toolErr
			}
			result.fallbackUsed = true
			reply = commonsai.BuildHRFallbackReply(toCommonsToolTraces(traces))
			if err := send(&pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "fallback", EventMessage: "model failed after useful tool results; using deterministic fallback", CreatedAt: formatTime(time.Now())}); err != nil {
				return result, err
			}
		} else {
			reply = toolReply
			if strings.TrimSpace(reply) == "" {
				reply = deltaBuilder.String()
			}
		}
	} else {
		// Complete-only providers (tests / degraded): still force live job tools when intent needs them.
		planned, planErr := s.preExecutePlannedHRTools(ctx, req, session.ID, opts.agentRunID, governance, executor, send)
		if planErr != nil {
			return result, planErr
		}
		traces = append(traces, planned...)
		contextPrompt := renderHRProviderPrompt(req, history, userMessage, traces, governance)
		result.contextUsage = estimateHRContextUsage(req.GetModelId(), contextPrompt, req.GetMessage(), traces)
		if err := send(&pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "context_usage", EventMessage: "context usage estimated", ContextUsage: result.contextUsage, CreatedAt: formatTime(time.Now())}); err != nil {
			return result, err
		}
		if err := send(&pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "generating", EventMessage: "calling model with HR context", CreatedAt: formatTime(time.Now())}); err != nil {
			return result, err
		}
		reply, err = s.complete(ctx, contextPrompt, req.GetModelId(), hrRuntimeCompletionOptions(governance))
		if err != nil {
			if errors.Is(err, errAIProviderRequired) {
				result.providerUnavailable = true
			}
			if !hasUsefulToolResults(traces) {
				if errors.Is(err, errAIProviderRequired) {
					return result, nil
				}
				return result, err
			}
			result.fallbackUsed = true
			reply = commonsai.BuildHRFallbackReply(toCommonsToolTraces(traces))
			if err := send(&pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "fallback", EventMessage: "model failed after useful tool results; using deterministic fallback", CreatedAt: formatTime(time.Now())}); err != nil {
				return result, err
			}
		}
	}
	result.reply = reply
	processContent := buildHRProcessContent(traces, result.contextUsage, result.fallbackUsed, governance)
	if s.store != nil {
		if _, err := s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: ownerRoleHR, OwnerID: req.GetHrId(), SessionID: session.ID, Role: "assistant", Content: reply, ProcessContent: processContent, ModelID: req.GetModelId(), AgentSkillIDs: hrRuntimeAgentSkillIDs(governance), AgentSkillNames: hrRuntimeAgentSkillNames(governance), CreatedAt: time.Now()}); err != nil {
			return result, err
		}
	}
	return result, nil
}

func findHRUserMessage(messages []ChatMessageRow, content string) ChatMessageRow {
	needle := strings.TrimSpace(content)
	if needle == "" {
		return ChatMessageRow{}
	}
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message.Role == "user" && strings.TrimSpace(message.Content) == needle {
			return message
		}
	}
	return ChatMessageRow{}
}

func hrRecruitingToolSchemas(allow []string) []*schema.ToolInfo {
	if len(allow) == 0 {
		return nil
	}
	set := map[string]bool{}
	for _, name := range allow {
		name = hr_tools.NormalizeToolName(name)
		if hr_tools.ExecutableByThisRunner[name] {
			set[name] = true
		}
	}
	if len(set) == 0 {
		return nil
	}
	all := commonsai.RecruitingTools()
	out := make([]*schema.ToolInfo, 0, len(set))
	for _, tool := range all {
		if tool != nil && set[tool.Name] {
			out = append(out, tool)
		}
	}
	return out
}

func buildHRToolCallingMessages(req *pb.ChatRequest, history []ChatMessageRow, current ChatMessageRow, traces []ToolTraceRow, governance hrRuntimeGovernanceContext) []*schema.Message {
	system := renderHRProviderPrompt(req, nil, ChatMessageRow{}, traces, governance)
	// strip trailing "User:\n...\nAssistant:" section when history is empty in render — keep system body only
	if idx := strings.LastIndex(system, "\nUser:\n"); idx >= 0 {
		system = strings.TrimSpace(system[:idx])
	}
	messages := []*schema.Message{schema.SystemMessage(system)}
	for _, message := range ensureHRCurrentMessage(history, current) {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		switch message.Role {
		case "assistant":
			messages = append(messages, schema.AssistantMessage(content, nil))
		default:
			messages = append(messages, schema.UserMessage(content))
		}
	}
	if current.ID == 0 {
		if msg := strings.TrimSpace(req.GetMessage()); msg != "" {
			// Avoid duplicating if already in history.
			if len(messages) == 1 || messages[len(messages)-1].Content != msg {
				messages = append(messages, schema.UserMessage(msg))
			}
		}
	}
	return messages
}

func (s *nativeAIService) preExecutePlannedHRTools(ctx context.Context, req *pb.ChatRequest, sessionID, agentRunID int64, governance hrRuntimeGovernanceContext, executor *hr_tools.Executor, emit hrChatStreamEmitter) ([]ToolTraceRow, error) {
	if executor == nil || (s.jobs == nil && s.appList == nil && s.applications == nil) {
		return nil, nil
	}
	plan := commonsai.NewRecruitingPlanner().Plan(commonsai.RecruitingPlannerInput{
		Message:        req.GetMessage(),
		AvailableTools: governance.ExecutableToolNames,
		ApplicationID:  req.GetApplicationId(),
	})
	tools := plan.RequiredTools
	if len(tools) == 0 {
		// Explicit job inventory questions always pull the live list even if planner misses.
		msg := strings.ToLower(req.GetMessage())
		if strings.Contains(msg, "岗位") || strings.Contains(msg, "职位") || strings.Contains(msg, "job") {
			for _, name := range governance.ExecutableToolNames {
				if name == "get_job_list" || name == "search_jobs" {
					tools = append(tools, name)
				}
			}
		}
	}
	// Deduplicate and only execute tools this runner supports.
	seen := map[string]bool{}
	ordered := make([]string, 0, len(tools))
	for _, name := range tools {
		name = hr_tools.NormalizeToolName(name)
		if !hr_tools.ExecutableByThisRunner[name] || seen[name] {
			continue
		}
		seen[name] = true
		ordered = append(ordered, name)
	}
	if len(ordered) == 0 {
		return nil, nil
	}
	// Prefer a single inventory tool for free-form "what jobs" questions.
	if len(ordered) > 1 {
		preferred := ""
		for _, name := range ordered {
			if name == "get_job_list" {
				preferred = name
				break
			}
		}
		if preferred == "" {
			preferred = ordered[0]
		}
		ordered = []string{preferred}
	}
	traces := make([]ToolTraceRow, 0, len(ordered))
	for _, name := range ordered {
		if emit != nil {
			if err := emit(&pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "tool_calling", EventMessage: "querying " + name, ToolName: name, CreatedAt: formatTime(time.Now())}); err != nil {
				return traces, err
			}
		}
		start := time.Now()
		args := map[string]any{}
		if name == "search_jobs" {
			args["keyword"] = ""
		}
		result, execErr := executor.Execute(ctx, req.GetHrId(), name, args)
		trace := ToolTraceRow{
			SessionID:     sessionID,
			AgentRunID:    agentRunID,
			ToolCallID:    fmt.Sprintf("planned-%s-%d", name, time.Now().UnixNano()),
			ToolName:      name,
			ArgsJSON:      marshalJSONString(args),
			ResultContent: result.Content,
			DurationMs:    time.Since(start).Milliseconds(),
			CreatedAt:     time.Now(),
		}
		if execErr != nil {
			trace.ErrorMsg = execErr.Error()
			trace.Status = "error"
		} else {
			trace.Status = "success"
		}
		persisted, persistErr := s.persistHRToolTrace(ctx, req.GetHrId(), trace)
		if persistErr != nil {
			return traces, persistErr
		}
		if persisted.ID != 0 {
			trace = persisted
		}
		if emit != nil {
			event := &pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "tool_done", EventMessage: name + " finished", ToolName: name, CreatedAt: formatTime(time.Now())}
			if trace.ErrorMsg != "" {
				event.EventType = "error"
				event.EventMessage = trace.ErrorMsg
				event.ErrorType = "TOOL_ERROR"
				event.Msg = trace.ErrorMsg
			}
			if err := emit(event); err != nil {
				return traces, err
			}
		}
		traces = append(traces, trace)
	}
	return traces, nil
}

func (s *nativeAIService) hrRecentMessages(ctx context.Context, hrID, sessionID int64) ([]ChatMessageRow, error) {
	if s == nil || s.store == nil || sessionID == 0 {
		return nil, nil
	}
	return s.store.ListChatMessages(ctx, ownerRoleHR, hrID, sessionID, 1, 20)
}

func (s *nativeAIService) loadHRRuntimeGovernance(ctx context.Context, req *pb.ChatRequest) (hrRuntimeGovernanceContext, error) {
	var runtime hrRuntimeGovernanceContext
	if s == nil || s.store == nil {
		return runtime, nil
	}
	agent, err := s.loadDefaultHRAgentConfig(ctx)
	if err != nil {
		return runtime, err
	}
	runtime.Agent = agent
	runtime.CapabilityKeys = hrRuntimeCapabilityKeys(agent)
	runtime.ToolNames = hrRuntimeToolNames(agent)
	runtime.ExecutableToolNames = hr_tools.ResolveBuiltinToolNames(runtime.ToolNames, runtime.CapabilityKeys)
	// Prefer concrete tool names for prompt/runtime metadata once resolved.
	if len(runtime.ExecutableToolNames) > 0 {
		runtime.ToolNames = append([]string(nil), runtime.ExecutableToolNames...)
	}
	runtime.Prompt = s.loadHRRuntimePrompt(ctx, agent)
	runtime.AgentSkillSelectionConfirmed = req.GetAgentSkillSelectionConfirmed()
	runtime.AgentSkillSelectionMessageID = req.GetAgentSkillSelectionMessageId()
	runtime.SelectedAgentSkills = s.selectHRRuntimeAgentSkills(ctx, req, runtime.CapabilityKeys)
	s.enrichHRRuntimeAgentSkillBodies(ctx, &runtime)
	switch {
	case len(req.GetAgentSkillIds()) > 0:
		runtime.AgentSkillSelectionMode = "manual"
	case req.GetAgentSkillSelectionConfirmed():
		runtime.AgentSkillSelectionMode = "confirmed_empty"
	case len(runtime.SelectedAgentSkills) > 0:
		runtime.AgentSkillSelectionMode = "auto"
	default:
		runtime.AgentSkillSelectionMode = "none"
	}
	return runtime, nil
}

func (s *nativeAIService) loadDefaultHRAgentConfig(ctx context.Context) (*pb.AgentConfigInfo, error) {
	rows, _, err := s.store.ListAgentConfigs(ctx, 1, 100, hrRecruitingAgentType)
	if err != nil {
		return nil, err
	}
	var selected *pb.AgentConfigInfo
	for _, row := range rows {
		if row == nil || !row.GetIsEnabled() {
			continue
		}
		if row.GetIsDefault() {
			selected = row
			break
		}
		if selected == nil {
			selected = row
		}
	}
	if selected == nil {
		return nil, nil
	}
	if configStore, ok := s.store.(hrRuntimeAgentConfigStore); ok {
		resp, err := configStore.GetAgentConfig(ctx, &pb.GetAgentConfigRequest{AgentType: hrRecruitingAgentType})
		if err != nil {
			return nil, err
		}
		if resp != nil && resp.GetCode() == 0 && resp.GetAgent() != nil {
			return resp.GetAgent(), nil
		}
	}
	return selected, nil
}

func (s *nativeAIService) loadHRRuntimePrompt(ctx context.Context, agent *pb.AgentConfigInfo) *pb.PromptTemplateInfo {
	if s == nil || s.store == nil {
		return nil
	}
	if agent != nil && agent.GetPromptTemplateId() > 0 {
		if promptStore, ok := s.store.(hrRuntimePromptTemplateStore); ok {
			template, found, err := promptStore.GetRuntimePromptTemplateByID(ctx, agent.GetPromptTemplateId())
			if err == nil && found && hrPromptTemplateUsable(template) {
				return template
			}
		}
	}
	if promptStore, ok := s.store.(activePromptStore); ok {
		for _, agentType := range []string{hrRecruitingAgentType, "hr_agent"} {
			resp, err := promptStore.GetActivePromptByAgentType(ctx, &pb.GetActivePromptByAgentTypeRequest{AgentType: agentType, PromptRole: hrRuntimePromptRoleSystem})
			if err == nil && resp != nil && resp.GetCode() == 0 && hrPromptTemplateUsable(resp.GetTemplate()) {
				return resp.GetTemplate()
			}
		}
	}
	return nil
}

func hrPromptTemplateUsable(template *pb.PromptTemplateInfo) bool {
	if template == nil || strings.TrimSpace(template.GetContent()) == "" {
		return false
	}
	if !template.GetIsActive() {
		return false
	}
	role := strings.TrimSpace(template.GetPromptRole())
	if role != "" && !strings.EqualFold(role, hrRuntimePromptRoleSystem) {
		return false
	}
	agentType := strings.TrimSpace(template.GetAgentType())
	if agentType == "" {
		return true
	}
	return strings.EqualFold(agentType, hrRecruitingAgentType) || strings.EqualFold(agentType, "hr_agent")
}

func (s *nativeAIService) enrichHRRuntimeAgentSkillBodies(ctx context.Context, runtime *hrRuntimeGovernanceContext) {
	if s == nil || s.store == nil || runtime == nil || len(runtime.SelectedAgentSkills) == 0 {
		return
	}
	detailStore, ok := s.store.(agentSkillDetailStore)
	if !ok {
		return
	}
	for i := range runtime.SelectedAgentSkills {
		skill := &runtime.SelectedAgentSkills[i]
		if skill.ID <= 0 || strings.TrimSpace(skill.SkillMD) != "" {
			continue
		}
		resp, err := detailStore.GetAgentSkill(ctx, &pb.GetAgentSkillRequest{Id: skill.ID})
		if err != nil || resp == nil || resp.GetCode() != 0 || resp.GetSkill() == nil {
			continue
		}
		md := ""
		currentVersionID := resp.GetSkill().GetCurrentVersionId()
		if versions, vErr := detailStore.ListAgentSkillVersions(ctx, &pb.ListAgentSkillVersionsRequest{SkillId: skill.ID}); vErr == nil && versions != nil && versions.GetCode() == 0 {
			for _, version := range versions.GetList() {
				if version == nil {
					continue
				}
				if currentVersionID > 0 && version.GetId() == currentVersionID {
					md = strings.TrimSpace(version.GetSkillMd())
					break
				}
				if md == "" {
					md = strings.TrimSpace(version.GetSkillMd())
				}
			}
		}
		if md == "" {
			md = strings.TrimSpace(resp.GetSkill().GetDescription())
		}
		skill.SkillMD = md
		if strings.TrimSpace(skill.Description) == "" {
			skill.Description = strings.TrimSpace(resp.GetSkill().GetDescription())
		}
	}
}

func (s *nativeAIService) selectHRRuntimeAgentSkills(ctx context.Context, req *pb.ChatRequest, capabilityKeys []string) []hrRuntimeAgentSkill {
	if s == nil || s.store == nil || req.GetAgentSkillSelectionConfirmed() && len(req.GetAgentSkillIds()) == 0 {
		return nil
	}
	rows, _, err := s.store.ListAgentSkills(ctx, 1, 100, "", true)
	if err != nil || len(rows) == 0 {
		return nil
	}
	available := make(map[string]bool, len(capabilityKeys)*2)
	for _, key := range capabilityKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		available[key] = true
		// Accept both bare keys and "builtin:key" forms used by admin UIs.
		normalized := hr_tools.NormalizeToolName(key)
		available[normalized] = true
		if !strings.Contains(key, ":") {
			available["builtin:"+key] = true
		}
	}
	selectedCapabilities := normalizedStringSet(req.GetSkillCapabilityKeys())
	if len(selectedCapabilities) > 0 {
		for key := range available {
			normalized := hr_tools.NormalizeToolName(key)
			if !selectedCapabilities[key] && !selectedCapabilities[normalized] && !selectedCapabilities["builtin:"+normalized] {
				delete(available, key)
			}
		}
	}
	candidates := make([]model.AgentSkill, 0, len(rows))
	byID := make(map[uint64]*pb.AgentSkillInfo, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		id := uint64(row.GetId())
		byID[id] = row
		candidates = append(candidates, model.AgentSkill{
			ID:                   id,
			Name:                 row.GetName(),
			AgentType:            row.GetAgentType(),
			Enabled:              row.GetIsEnabled(),
			ManualInvocable:      row.GetIsManualInvocable(),
			Category:             row.GetCategory(),
			Scenario:             row.GetScenario(),
			Priority:             int(row.GetPriority()),
			RiskLevel:            row.GetRiskLevel(),
			RequiredCapabilities: row.GetRequiredCapabilities(),
			SemanticTags:         row.GetSemanticTags(),
		})
	}
	manualIDs := make([]uint64, 0, len(req.GetAgentSkillIds()))
	for _, id := range req.GetAgentSkillIds() {
		if id > 0 {
			manualIDs = append(manualIDs, uint64(id))
		}
	}
	semanticScores := map[uint64]float64(nil)
	if len(manualIDs) == 0 && s.embedding != nil {
		if scores, _ := s.embedding.SemanticScores(ctx, req.GetMessage(), 20); len(scores) > 0 {
			semanticScores = scores
		}
	}
	selected := policy.SelectAgentSkills(candidates, model.AgentSkillSelectionRequest{
		AgentType:             hrRecruitingAgentType,
		Question:              req.GetMessage(),
		ManualIDs:             manualIDs,
		AvailableCapabilities: available,
		SemanticScores:        semanticScores,
		MaxSkills:             3,
	})
	result := make([]hrRuntimeAgentSkill, 0, len(selected))
	for _, skill := range selected {
		row := byID[skill.ID]
		if row == nil {
			continue
		}
		result = append(result, hrRuntimeAgentSkill{
			ID:                   row.GetId(),
			Name:                 row.GetName(),
			DisplayName:          row.GetDisplayName(),
			Description:          row.GetDescription(),
			Manual:               skill.Manual,
			Reason:               skill.Reason,
			RiskLevel:            skill.RiskLevel,
			RequiredCapabilities: row.GetRequiredCapabilities(),
		})
	}
	return result
}

func hrRuntimeCapabilityKeys(agent *pb.AgentConfigInfo) []string {
	if agent == nil || len(agent.GetCapabilityBindings()) == 0 {
		return []string{hrCandidateSearchCapability, "resume_intelligence", "interview_context"}
	}
	keys := make([]string, 0, len(agent.GetCapabilityBindings()))
	for _, binding := range agent.GetCapabilityBindings() {
		if binding == nil || !binding.GetIsEnabled() {
			continue
		}
		key := strings.TrimSpace(binding.GetCapabilityKey())
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func hrRuntimeToolNames(agent *pb.AgentConfigInfo) []string {
	if agent == nil || len(agent.GetToolBindings()) == 0 {
		return []string{hrApplicationSnapshotTool}
	}
	tools := make([]string, 0, len(agent.GetToolBindings()))
	for _, binding := range agent.GetToolBindings() {
		if binding == nil || !binding.GetIsEnabled() {
			continue
		}
		name := strings.TrimSpace(binding.GetToolName())
		if name != "" {
			tools = append(tools, name)
		}
	}
	sort.Strings(tools)
	return tools
}

func hrRuntimeAgentSkillIDs(governance hrRuntimeGovernanceContext) []int64 {
	ids := make([]int64, 0, len(governance.SelectedAgentSkills))
	for _, skill := range governance.SelectedAgentSkills {
		if skill.ID > 0 {
			ids = append(ids, skill.ID)
		}
	}
	return ids
}

func hrRuntimeAgentSkillNames(governance hrRuntimeGovernanceContext) []string {
	names := make([]string, 0, len(governance.SelectedAgentSkills))
	for _, skill := range governance.SelectedAgentSkills {
		if strings.TrimSpace(skill.Name) != "" {
			names = append(names, strings.TrimSpace(skill.Name))
		}
	}
	return names
}

func hrRuntimeCompletionOptions(governance hrRuntimeGovernanceContext) ChatCompletionOptions {
	if governance.Agent == nil || governance.Agent.GetTemperatureOverride() <= 0 {
		return ChatCompletionOptions{}
	}
	temperature := governance.Agent.GetTemperatureOverride()
	return ChatCompletionOptions{TemperatureOverride: &temperature}
}

func (s *nativeAIService) executeHRContextTools(ctx context.Context, req *pb.ChatRequest, sessionID, agentRunID int64, governance hrRuntimeGovernanceContext, emit hrChatStreamEmitter) ([]ToolTraceRow, error) {
	traces := make([]ToolTraceRow, 0, 1)
	if req.GetApplicationId() <= 0 {
		mcpTraces, err := s.executeHRMCPTools(ctx, req, sessionID, agentRunID, governance, emit)
		return append(traces, mcpTraces...), err
	}
	if !hrRuntimeAllowsApplicationSnapshot(req, governance) {
		trace := ToolTraceRow{SessionID: sessionID, AgentRunID: agentRunID, ToolName: hrApplicationSnapshotTool, ArgsJSON: fmt.Sprintf(`{"application_id":%d}`, req.GetApplicationId()), Status: "error", ErrorMsg: "application snapshot tool is not enabled by current agent capabilities", CreatedAt: time.Now()}
		persisted, _ := s.persistHRToolTrace(ctx, req.GetHrId(), trace)
		if persisted.ID != 0 {
			trace = persisted
		}
		traces = append(traces, trace)
	} else if s == nil || s.applications == nil {
		trace := ToolTraceRow{SessionID: sessionID, AgentRunID: agentRunID, ToolName: hrApplicationSnapshotTool, ArgsJSON: fmt.Sprintf(`{"application_id":%d}`, req.GetApplicationId()), Status: "error", ErrorMsg: "application owner client is not configured", CreatedAt: time.Now()}
		persisted, _ := s.persistHRToolTrace(ctx, req.GetHrId(), trace)
		if persisted.ID != 0 {
			trace = persisted
		}
		traces = append(traces, trace)
	} else {
		if emit != nil {
			if err := emit(&pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "tool_calling", EventMessage: "loading application snapshot", ToolName: hrApplicationSnapshotTool, CreatedAt: formatTime(time.Now())}); err != nil {
				return nil, err
			}
		}
		start := time.Now()
		snapshot, err := s.applications.GetApplicationSnapshot(ctx, &pb.GetApplicationSnapshotRequest{ApplicationId: req.GetApplicationId()})
		duration := time.Since(start)
		trace := ToolTraceRow{SessionID: sessionID, AgentRunID: agentRunID, ToolCallID: fmt.Sprintf("snapshot-%d", req.GetApplicationId()), ToolName: hrApplicationSnapshotTool, ArgsJSON: fmt.Sprintf(`{"application_id":%d}`, req.GetApplicationId()), DurationMs: duration.Milliseconds(), CreatedAt: time.Now()}
		if err != nil {
			trace.ErrorMsg = err.Error()
			trace.Status = "error"
		} else if snapshot.GetCode() != 0 {
			trace.ErrorMsg = strings.TrimSpace(snapshot.GetMsg())
			if trace.ErrorMsg == "" {
				trace.ErrorMsg = "application snapshot unavailable"
			}
			trace.Status = "error"
		} else {
			trace.Status = "success"
			trace.ResultContent = marshalJSONString(map[string]any{
				"application_id":    snapshot.GetApplicationId(),
				"candidate_user_id": snapshot.GetCandidateUserId(),
				"job_id":            snapshot.GetJobId(),
				"job_title":         snapshot.GetJobTitle(),
				"candidate_name":    snapshot.GetCandidateName(),
				"resume_id":         snapshot.GetResumeId(),
				"legacy_status":     snapshot.GetLegacyStatus(),
				"status_key":        snapshot.GetStatusKey(),
				"round_no":          snapshot.GetRoundNo(),
				"is_current":        snapshot.GetIsCurrent(),
			})
		}
		persisted, persistErr := s.persistHRToolTrace(ctx, req.GetHrId(), trace)
		if persistErr != nil {
			return nil, persistErr
		}
		if persisted.ID != 0 {
			trace = persisted
		}
		if emit != nil {
			event := &pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "tool_done", EventMessage: "application snapshot loaded", ToolName: "get_application_snapshot", CreatedAt: formatTime(time.Now())}
			if trace.ErrorMsg != "" {
				event.EventType = "error"
				event.EventMessage = trace.ErrorMsg
				event.ErrorType = "TOOL_ERROR"
				event.Msg = trace.ErrorMsg
			}
			if err := emit(event); err != nil {
				return nil, err
			}
		}
		traces = append(traces, trace)
	}
	mcpTraces, err := s.executeHRMCPTools(ctx, req, sessionID, agentRunID, governance, emit)
	if err != nil {
		return traces, err
	}
	return append(traces, mcpTraces...), nil
}

func (s *nativeAIService) executeHRMCPTools(ctx context.Context, req *pb.ChatRequest, sessionID, agentRunID int64, governance hrRuntimeGovernanceContext, emit hrChatStreamEmitter) ([]ToolTraceRow, error) {
	calls := hrRuntimeSelectedMCPTools(req, governance)
	if len(calls) == 0 {
		return nil, nil
	}
	if s == nil || s.mcpRunner == nil {
		traces := make([]ToolTraceRow, 0, len(calls))
		for _, call := range calls {
			trace := ToolTraceRow{SessionID: sessionID, AgentRunID: agentRunID, ToolName: call.runtimeName, ArgsJSON: call.argsJSON(req.GetMessage()), Status: "error", ErrorMsg: "mcp runner is not configured", CreatedAt: time.Now()}
			persisted, err := s.persistHRToolTrace(ctx, req.GetHrId(), trace)
			if err != nil {
				return traces, err
			}
			if persisted.ID != 0 {
				trace = persisted
			}
			traces = append(traces, trace)
		}
		return traces, nil
	}
	service := nativeMCPService{store: s.store, runner: s.mcpRunner}
	traces := make([]ToolTraceRow, 0, len(calls))
	for _, call := range calls {
		argsJSON := call.argsJSON(req.GetMessage())
		if emit != nil {
			if err := emit(&pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "tool_calling", EventMessage: "calling MCP tool", ToolName: call.runtimeName, CreatedAt: formatTime(time.Now())}); err != nil {
				return traces, err
			}
		}
		resp, err := service.CallMCPTool(ctx, &pb.CallMCPToolRequest{ServerId: call.serverID, ToolName: call.toolName, ArgsJson: argsJSON, CalledByHrId: req.GetHrId(), SessionId: sessionID, CallerRole: "hr_agent", CallerScope: "agent_runtime", ConfirmationApproved: req.GetAgentSkillSelectionConfirmed()})
		trace := ToolTraceRow{SessionID: sessionID, AgentRunID: agentRunID, ToolCallID: fmt.Sprintf("mcp-%d-%s", call.serverID, call.toolName), ToolName: call.runtimeName, ArgsJSON: service.RedactedArgsForTool(ctx, call.serverID, call.toolName, argsJSON), CreatedAt: time.Now()}
		if err != nil {
			trace.ErrorMsg = err.Error()
			trace.Status = "error"
		} else if resp.GetCode() != 0 || strings.TrimSpace(resp.GetErrorMsg()) != "" {
			trace.ErrorMsg = strings.TrimSpace(resp.GetErrorMsg())
			if trace.ErrorMsg == "" {
				trace.ErrorMsg = strings.TrimSpace(resp.GetMsg())
			}
			trace.DurationMs = resp.GetDurationMs()
			trace.Status = "error"
		} else {
			trace.ResultContent = resp.GetResultContent()
			trace.DurationMs = resp.GetDurationMs()
			trace.Status = "success"
		}
		persisted, persistErr := s.persistHRToolTrace(ctx, req.GetHrId(), trace)
		if persistErr != nil {
			return traces, persistErr
		}
		if persisted.ID != 0 {
			trace = persisted
		}
		if emit != nil {
			event := &pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "tool_done", EventMessage: "MCP tool finished", ToolName: call.runtimeName, CreatedAt: formatTime(time.Now())}
			if trace.ErrorMsg != "" {
				event.EventType = "error"
				event.EventMessage = trace.ErrorMsg
				event.ErrorType = "MCP_TOOL_ERROR"
				event.Msg = trace.ErrorMsg
			}
			if err := emit(event); err != nil {
				return traces, err
			}
		}
		traces = append(traces, trace)
	}
	return traces, nil
}

type hrRuntimeMCPToolCall struct {
	serverID    int64
	toolName    string
	runtimeName string
}

func (call hrRuntimeMCPToolCall) argsJSON(message string) string {
	return marshalJSONString(map[string]any{"query": strings.TrimSpace(message)})
}

func hrRuntimeSelectedMCPTools(req *pb.ChatRequest, governance hrRuntimeGovernanceContext) []hrRuntimeMCPToolCall {
	if governance.Agent == nil {
		return nil
	}
	selected := normalizedStringSet(req.GetSkillCapabilityKeys())
	// When the client does not pass skill_capability_keys, expose all MCP tools
	// bound on the agent (keys only narrow the set).
	restrict := len(selected) > 0
	calls := make([]hrRuntimeMCPToolCall, 0)
	seen := map[string]bool{}
	for _, binding := range governance.Agent.GetCapabilityBindings() {
		if binding == nil || !binding.GetIsEnabled() || !strings.EqualFold(binding.GetCapabilitySource(), "mcp") {
			continue
		}
		key := strings.TrimSpace(binding.GetCapabilityKey())
		if key == "" || seen[key] {
			continue
		}
		if restrict && !selected[key] {
			continue
		}
		serverID, toolName, ok := parseHRMCPBindingKey(key)
		if !ok {
			continue
		}
		calls = append(calls, hrRuntimeMCPToolCall{serverID: serverID, toolName: toolName, runtimeName: hrMCPRuntimeToolName(serverID, toolName)})
		seen[key] = true
	}
	return calls
}

func parseHRMCPBindingKey(key string) (int64, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(key), ":", 2)
	if len(parts) != 2 {
		return 0, "", false
	}
	serverID, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, "", false
	}
	toolName := strings.TrimSpace(parts[1])
	return serverID, toolName, serverID > 0 && toolName != ""
}

func hrMCPRuntimeToolName(serverID int64, toolName string) string {
	clean := strings.NewReplacer(":", "_", "-", "_", ".", "_", " ", "_").Replace(strings.TrimSpace(toolName))
	return fmt.Sprintf("mcp_%d_%s", serverID, clean)
}

func (s *nativeAIService) persistHRToolTrace(ctx context.Context, hrID int64, trace ToolTraceRow) (ToolTraceRow, error) {
	if s == nil || s.store == nil {
		return trace, nil
	}
	if strings.TrimSpace(trace.Status) == "" {
		if strings.TrimSpace(trace.ErrorMsg) != "" {
			trace.Status = "error"
		} else {
			trace.Status = "success"
		}
	}
	// When attached to a durable run, create an agent_run_steps row and link the trace.
	if trace.AgentRunID > 0 && strings.TrimSpace(trace.ToolName) != "" {
		now := time.Now()
		if trace.CreatedAt.IsZero() {
			trace.CreatedAt = now
		}
		step := AgentRunStepRow{
			RunID:            trace.AgentRunID,
			StepType:         "tool",
			CapabilitySource: "builtin",
			CapabilityKey:    trace.ToolName,
			ToolName:         trace.ToolName,
			InputJSON:        trace.ArgsJSON,
			OutputJSON:       trace.ResultContent,
			Status:           trace.Status,
			DurationMs:       trace.DurationMs,
			ErrorMsg:         trace.ErrorMsg,
			StartedAt:        trace.CreatedAt.Add(-time.Duration(trace.DurationMs) * time.Millisecond),
			CreatedAt:        trace.CreatedAt,
			UpdatedAt:        now,
		}
		if strings.HasPrefix(trace.ToolName, "mcp_") {
			step.CapabilitySource = "mcp"
		}
		if step.Status == "success" || step.Status == "error" {
			completed := now
			step.CompletedAt = &completed
		}
		if persistedStep, err := s.store.AppendAgentRunStep(ctx, step); err == nil && persistedStep.ID != 0 {
			trace.AgentRunStepID = persistedStep.ID
		}
	}
	return s.store.AppendToolTrace(ctx, hrID, trace)
}

func hrRuntimeAllowsApplicationSnapshot(req *pb.ChatRequest, governance hrRuntimeGovernanceContext) bool {
	if governance.Agent != nil && len(governance.Agent.GetToolBindings()) > 0 && !stringSliceContains(governance.ToolNames, hrApplicationSnapshotTool) {
		return false
	}
	if governance.Agent != nil && len(governance.Agent.GetCapabilityBindings()) > 0 && !stringSliceContains(governance.CapabilityKeys, hrCandidateSearchCapability) {
		return false
	}
	selected := normalizedStringSet(req.GetSkillCapabilityKeys())
	if len(selected) > 0 && !selected[hrCandidateSearchCapability] {
		return false
	}
	return true
}

func renderHRProviderPrompt(req *pb.ChatRequest, history []ChatMessageRow, current ChatMessageRow, traces []ToolTraceRow, governance hrRuntimeGovernanceContext) string {
	var b strings.Builder
	b.WriteString("System:\n你是 Smart Recruit 的 HR 招聘助手。必须优先依据系统工具返回的招聘数据回答；如果工具不可用，明确说明限制，不要编造候选人、岗位或投递数据。涉及岗位列表、投递统计、候选人信息等实时数据时，必须使用工具查询结果，禁止凭常识编造。\n")
	if governance.Prompt != nil && strings.TrimSpace(governance.Prompt.GetContent()) != "" {
		b.WriteString("\nActive prompt template:\n")
		b.WriteString(strings.TrimSpace(governance.Prompt.GetContent()))
		b.WriteString("\n")
	}
	if governance.Agent != nil {
		instruction := strings.TrimSpace(governance.Agent.GetInstruction())
		if instruction != "" {
			b.WriteString("\nAgent instruction:\n")
			b.WriteString(instruction)
			b.WriteString("\n")
		}
		b.WriteString("\nActive agent config:\n")
		b.WriteString(marshalJSONString(map[string]any{
			"agent_id":             governance.Agent.GetId(),
			"agent_type":           governance.Agent.GetAgentType(),
			"name":                 governance.Agent.GetName(),
			"display_name":         governance.Agent.GetDisplayName(),
			"instruction":          governance.Agent.GetInstruction(),
			"max_iterations":       governance.Agent.GetMaxIterations(),
			"temperature_override": governance.Agent.GetTemperatureOverride(),
			"capability_keys":      governance.CapabilityKeys,
			"tool_names":           governance.ToolNames,
			"executable_tools":     governance.ExecutableToolNames,
		}))
		b.WriteString("\n")
	}
	for _, skill := range governance.SelectedAgentSkills {
		if strings.TrimSpace(skill.SkillMD) == "" {
			continue
		}
		b.WriteString("\n## Active Agent Skill: ")
		if skill.DisplayName != "" {
			b.WriteString(skill.DisplayName)
		} else {
			b.WriteString(skill.Name)
		}
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(skill.SkillMD))
		b.WriteString("\n")
	}
	if len(history) > 0 {
		b.WriteString("\nRecent conversation:\n")
		for _, message := range ensureHRCurrentMessage(history, current) {
			if strings.TrimSpace(message.Content) == "" {
				continue
			}
			b.WriteString(message.Role)
			b.WriteString(":\n")
			b.WriteString(strings.TrimSpace(message.Content))
			b.WriteString("\n")
		}
	}
	if len(traces) > 0 {
		b.WriteString("\nTool results:\n")
		for _, trace := range traces {
			b.WriteString("- ")
			b.WriteString(trace.ToolName)
			if trace.ErrorMsg != "" {
				b.WriteString(" error: ")
				b.WriteString(trace.ErrorMsg)
			} else {
				b.WriteString(": ")
				b.WriteString(trace.ResultContent)
			}
			b.WriteString("\n")
		}
	}
	if req.GetAgentSkillSelectionConfirmed() || len(req.GetAgentSkillIds()) > 0 || len(req.GetSkillCapabilityKeys()) > 0 {
		b.WriteString("\nRuntime selection:\n")
		b.WriteString(marshalJSONString(map[string]any{
			"agent_skill_ids":                  req.GetAgentSkillIds(),
			"skill_capability_keys":            req.GetSkillCapabilityKeys(),
			"agent_skill_selection_confirmed":  req.GetAgentSkillSelectionConfirmed(),
			"agent_skill_selection_message_id": req.GetAgentSkillSelectionMessageId(),
		}))
		b.WriteString("\n")
	}
	if len(governance.SelectedAgentSkills) > 0 || governance.AgentSkillSelectionMode != "" {
		b.WriteString("\nSelected Agent Skills:\n")
		b.WriteString(marshalJSONString(map[string]any{
			"mode":         governance.AgentSkillSelectionMode,
			"confirmed":    governance.AgentSkillSelectionConfirmed,
			"message_id":   governance.AgentSkillSelectionMessageID,
			"agent_skills": governance.SelectedAgentSkills,
		}))
		b.WriteString("\n")
	}
	b.WriteString("\nUser:\n")
	b.WriteString(strings.TrimSpace(req.GetMessage()))
	b.WriteString("\nAssistant:")
	return b.String()
}

func ensureHRCurrentMessage(messages []ChatMessageRow, current ChatMessageRow) []ChatMessageRow {
	if current.ID == 0 {
		return messages
	}
	for _, message := range messages {
		if message.ID == current.ID {
			return messages
		}
	}
	out := make([]ChatMessageRow, 0, len(messages)+1)
	out = append(out, messages...)
	out = append(out, current)
	return out
}

func estimateHRContextUsage(modelID int64, prompt, current string, traces []ToolTraceRow) *pb.ContextUsageInfo {
	system := estimateTokens("你是 Smart Recruit 的 HR 招聘助手。")
	currentTokens := estimateTokens(current)
	toolTokens := 0
	for _, trace := range traces {
		toolTokens += estimateTokens(trace.ResultContent) + estimateTokens(trace.ErrorMsg)
	}
	promptTokens := estimateTokens(prompt)
	window := int32(8192)
	remaining := int32(math.Max(0, float64(int(window)-promptTokens)))
	return &pb.ContextUsageInfo{
		ModelId:                  modelID,
		ContextWindowTokens:      window,
		MaxOutputTokens:          1024,
		PromptTokensEstimated:    int32(promptTokens),
		RemainingTokensEstimated: remaining,
		UsageRatio:               float64(promptTokens) / float64(window),
		Estimated:                true,
		Source:                   "native-hr-runtime",
		Stage:                    "pre_generation",
		Breakdown: &pb.ContextUsageBreakdown{
			SystemPromptTokens:   int32(system),
			CurrentMessageTokens: int32(currentTokens),
			ToolResultTokens:     int32(toolTokens),
			RecentMessageTokens:  int32(maxInt(promptTokens-system-currentTokens-toolTokens, 0)),
		},
	}
}

func estimateTokens(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	runes := len([]rune(trimmed))
	return maxInt((runes+3)/4, 1)
}

func toCommonsToolTraces(rows []ToolTraceRow) []commonsai.ToolTrace {
	traces := make([]commonsai.ToolTrace, 0, len(rows))
	for _, row := range rows {
		trace := commonsai.ToolTrace{ToolName: row.ToolName, Result: row.ResultContent, Cost: time.Duration(row.DurationMs) * time.Millisecond}
		if strings.TrimSpace(row.ArgsJSON) != "" {
			var args map[string]any
			if err := json.Unmarshal([]byte(row.ArgsJSON), &args); err == nil {
				trace.Arguments = args
			}
		}
		if strings.TrimSpace(row.ErrorMsg) != "" {
			trace.Error = errors.New(row.ErrorMsg)
		}
		traces = append(traces, trace)
	}
	return traces
}

func hasUsefulToolResults(rows []ToolTraceRow) bool {
	for _, row := range rows {
		if strings.TrimSpace(row.ErrorMsg) == "" && strings.TrimSpace(row.ResultContent) != "" {
			return true
		}
	}
	return false
}

func extractApplicationTraceMetadata(raw string) (candidateName, jobTitle string, status int32) {
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return "", "", 0
	}
	candidateName, _ = data["candidate_name"].(string)
	jobTitle, _ = data["job_title"].(string)
	status = int32(numberFromAny(data["legacy_status"]))
	return candidateName, jobTitle, status
}

func buildHRProcessContent(traces []ToolTraceRow, usage *pb.ContextUsageInfo, fallbackUsed bool, governance hrRuntimeGovernanceContext) string {
	payload := map[string]any{
		"runtime":       "native-hr-runtime",
		"fallback_used": fallbackUsed,
		"tool_count":    len(traces),
	}
	if usage != nil {
		payload["context_usage"] = map[string]any{
			"prompt_tokens_estimated":    usage.GetPromptTokensEstimated(),
			"remaining_tokens_estimated": usage.GetRemainingTokensEstimated(),
			"source":                     usage.GetSource(),
			"stage":                      usage.GetStage(),
		}
	}
	if governance.Agent != nil || governance.Prompt != nil || len(governance.SelectedAgentSkills) > 0 || governance.AgentSkillSelectionMode != "" {
		payload["governance"] = map[string]any{
			"agent_id":                         agentConfigID(governance.Agent),
			"agent_type":                       agentConfigType(governance.Agent),
			"prompt_template_id":               promptTemplateID(governance.Prompt),
			"capability_keys":                  governance.CapabilityKeys,
			"tool_names":                       governance.ToolNames,
			"agent_skill_selection_mode":       governance.AgentSkillSelectionMode,
			"agent_skill_selection_confirmed":  governance.AgentSkillSelectionConfirmed,
			"agent_skill_selection_message_id": governance.AgentSkillSelectionMessageID,
			"agent_skills":                     governance.SelectedAgentSkills,
		}
	}
	return marshalJSONString(payload)
}

func marshalJSONString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func numberFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		n, _ := v.Float64()
		return n
	default:
		return 0
	}
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
	startedAt := time.Now()
	inputChars := len([]rune(req.GetMessage()))
	session, err := s.ensureSession(ctx, ownerRoleCandidate, req.GetUserId(), req.GetSessionId(), 0, req.GetMessage())
	if err != nil {
		return err
	}
	userMessage, err := s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: ownerRoleCandidate, OwnerID: req.GetUserId(), SessionID: session.ID, Role: "user", Content: req.GetMessage()})
	if err != nil {
		return err
	}
	runtimeContext, err := s.loadCandidateRuntimeContext(ctx, req.GetUserId(), session.ID)
	if err != nil {
		return err
	}
	prompt, err := s.buildCandidateProviderPrompt(ctx, req.GetUserId(), session.ID, userMessage, runtimeContext)
	if err != nil {
		return err
	}
	reply, err := s.complete(ctx, prompt, 0)
	if err != nil {
		if fallback := buildCandidateContextFallbackReply(runtimeContext); fallback != "" {
			questions := candidateSuggestedQuestions(req.GetMessage(), fallback)
			if _, saveErr := s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: ownerRoleCandidate, OwnerID: req.GetUserId(), SessionID: session.ID, Role: "assistant", Content: fallback, CreatedAt: time.Now()}); saveErr != nil {
				return saveErr
			}
			if auditErr := s.recordCandidateUsageAudit(ctx, req.GetUserId(), inputChars, len([]rune(fallback)), "error", "provider_error", int(time.Since(startedAt).Milliseconds())); auditErr != nil {
				return auditErr
			}
			if sendErr := stream.Send(&pb.ChatStreamResponse{Code: 0, Msg: "success", EventType: "partial_done", EventMessage: "已基于已查询数据给出保守回复", ErrorType: "provider_error", SessionId: session.ID, CreatedAt: formatTime(time.Now())}); sendErr != nil {
				return sendErr
			}
			return stream.Send(&pb.ChatStreamResponse{Code: 0, Msg: "success", Delta: fallback, Done: true, SessionId: session.ID, CreatedAt: formatTime(time.Now()), EventType: "done", SuggestedQuestions: questions})
		}
		if auditErr := s.recordCandidateUsageAudit(ctx, req.GetUserId(), inputChars, 0, "error", "provider_error", int(time.Since(startedAt).Milliseconds())); auditErr != nil {
			return errors.Join(err, auditErr)
		}
		return err
	}
	cleanReply, suggestedQuestions := extractCandidateSuggestedQuestions(reply)
	if len(suggestedQuestions) != 3 {
		suggestedQuestions = candidateSuggestedQuestions(req.GetMessage(), cleanReply)
	}
	if s.store != nil {
		if _, err := s.store.AppendChatMessage(ctx, ChatMessageRow{OwnerRole: ownerRoleCandidate, OwnerID: req.GetUserId(), SessionID: session.ID, Role: "assistant", Content: cleanReply, CreatedAt: time.Now()}); err != nil {
			return err
		}
	}
	if err := s.recordCandidateUsageAudit(ctx, req.GetUserId(), inputChars, len([]rune(cleanReply)), "ok", "", int(time.Since(startedAt).Milliseconds())); err != nil {
		return err
	}
	return stream.Send(&pb.ChatStreamResponse{Code: 0, Msg: "success", Delta: cleanReply, Done: true, SessionId: session.ID, CreatedAt: formatTime(time.Now()), EventType: "done", SuggestedQuestions: suggestedQuestions})
}

func (s *nativeAIService) buildCandidateProviderPrompt(ctx context.Context, userID, sessionID int64, currentUserMessage ChatMessageRow, runtimeContext CandidateRuntimeContext) (string, error) {
	systemPrompt, err := s.resolveCandidateSystemPrompt(ctx)
	if err != nil {
		return "", err
	}
	messages, err := s.store.ListChatMessages(ctx, ownerRoleCandidate, userID, sessionID, 1, candidateChatContextPageSize)
	if err != nil {
		return "", err
	}
	messages = ensureCandidateCurrentMessage(messages, currentUserMessage, userID, sessionID)
	return renderCandidateProviderPrompt(systemPrompt, messages, userID, sessionID, runtimeContext), nil
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

func (s *nativeAIService) loadCandidateRuntimeContext(ctx context.Context, userID, sessionID int64) (CandidateRuntimeContext, error) {
	contextStore, ok := s.store.(candidateContextStore)
	if !ok {
		return CandidateRuntimeContext{}, nil
	}
	snapshot, err := contextStore.LoadCandidateRuntimeContext(ctx, userID, candidateRuntimeContextLimit)
	if err != nil {
		return CandidateRuntimeContext{}, err
	}
	for _, trace := range candidateContextToolTraces(snapshot, sessionID) {
		if _, err := s.store.AppendToolTrace(ctx, userID, trace); err != nil {
			return CandidateRuntimeContext{}, err
		}
	}
	return snapshot, nil
}

func candidateContextToolTraces(snapshot CandidateRuntimeContext, sessionID int64) []ToolTraceRow {
	now := time.Now()
	traces := make([]ToolTraceRow, 0, 5)
	appendTrace := func(name string, payload any) {
		raw, _ := json.Marshal(payload)
		traces = append(traces, ToolTraceRow{
			SessionID:     sessionID,
			ToolName:      name,
			ArgsJSON:      `{"scope":"current_candidate"}`,
			ResultContent: string(raw),
			CreatedAt:     now,
		})
	}
	appendTrace("list_my_applications", map[string]any{"applications": snapshot.Applications})
	appendTrace("get_my_resume_text", sanitizeCandidateResumeTrace(snapshot.Resume))
	appendTrace("list_jobs_for_recommendation", map[string]any{"jobs": snapshot.Jobs})
	appendTrace("list_candidate_interviews", map[string]any{"interviews": snapshot.Interviews})
	appendTrace("list_my_offers", map[string]any{"offers": snapshot.Offers})
	return traces
}

func sanitizeCandidateResumeTrace(resume CandidateResumeContext) CandidateResumeContext {
	resume.Summary = ""
	return resume
}

func (s *nativeAIService) recordCandidateUsageAudit(ctx context.Context, userID int64, requestChars, responseChars int, statusValue, errorCode string, costMs int) error {
	auditStore, ok := s.store.(candidateUsageAuditStore)
	if !ok {
		return nil
	}
	if statusValue == "" {
		statusValue = "ok"
	}
	_, err := auditStore.RecordCandidateUsageAudit(ctx, CandidateUsageAuditRow{
		UserID:          userID,
		ServiceType:     "ai_chat",
		Endpoint:        "/candidate/ai/chat/stream",
		Provider:        "openai_compatible",
		RequestChars:    requestChars,
		ResponseChars:   responseChars,
		EstimatedTokens: estimateTokenUsage(requestChars, responseChars),
		Status:          statusValue,
		ErrorCode:       errorCode,
		CostMs:          costMs,
		RequestID:       platformmetadata.GetRequestID(ctx),
		IP:              platformmetadata.GetClientIP(ctx),
		RoleKeys:        []string{"candidate"},
		PermissionKey:   "ai.candidate.use",
		ScopeKeys:       []string{"self"},
	})
	return err
}

func estimateTokenUsage(requestChars, responseChars int) int {
	total := requestChars + responseChars
	if total <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / 4.0))
}

func renderCandidateProviderPrompt(systemPrompt string, messages []ChatMessageRow, userID, sessionID int64, runtimeContext CandidateRuntimeContext) string {
	var b strings.Builder
	b.WriteString("System:\n")
	b.WriteString(strings.TrimSpace(systemPrompt))
	b.WriteString("\n\nCandidate runtime context (current candidate only; do not infer other candidates or HR-only data):\n")
	b.WriteString(renderCandidateRuntimeContext(runtimeContext))
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

func renderCandidateRuntimeContext(snapshot CandidateRuntimeContext) string {
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func extractCandidateSuggestedQuestions(reply string) (string, []string) {
	raw := strings.TrimSpace(reply)
	start := strings.Index(raw, candidateSuggestedQuestionsStartMarker)
	if start < 0 {
		return raw, nil
	}
	cleanReply := strings.TrimSpace(raw[:start])
	rest := raw[start+len(candidateSuggestedQuestionsStartMarker):]
	jsonText := rest
	if end := strings.Index(rest, candidateSuggestedQuestionsEndMarker); end >= 0 {
		jsonText = rest[:end]
		after := strings.TrimSpace(rest[end+len(candidateSuggestedQuestionsEndMarker):])
		if after != "" {
			cleanReply = strings.TrimSpace(cleanReply + "\n\n" + after)
		}
	}
	return cleanReply, parseCandidateSuggestedQuestionsJSON(jsonText)
}

func parseCandidateSuggestedQuestionsJSON(content string) []string {
	var values []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &values); err != nil {
		return nil
	}
	questions := make([]string, 0, 3)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		questions = append(questions, value)
		if len(questions) == 3 {
			break
		}
	}
	if len(questions) != 3 {
		return nil
	}
	return questions
}

func candidateSuggestedQuestions(message, reply string) []string {
	lower := strings.ToLower(message + "\n" + reply)
	switch {
	case strings.Contains(lower, "offer"):
		return []string{"我的 Offer 什么时候过期？", "Offer 里有哪些需要确认？", "我可以查看哪些 Offer 记录？"}
	case strings.Contains(lower, "面试") || strings.Contains(lower, "interview"):
		return []string{"我最近有哪些面试安排？", "面试前需要准备什么？", "这个岗位面试到第几轮了？"}
	case strings.Contains(lower, "简历") || strings.Contains(lower, "resume"):
		return []string{"我的简历解析完成了吗？", "简历可以优化哪些方向？", "基于简历推荐哪些岗位？"}
	case strings.Contains(lower, "岗位") || strings.Contains(lower, "job"):
		return []string{"有哪些岗位适合我？", "这些岗位我投递过吗？", "能按匹配度推荐岗位吗？"}
	default:
		return []string{"我有哪些投递进展？", "有哪些岗位适合我？", "我接下来应该准备什么？"}
	}
}

func buildCandidateContextFallbackReply(snapshot CandidateRuntimeContext) string {
	if !candidateRuntimeContextHasData(snapshot) {
		return ""
	}
	var b strings.Builder
	b.WriteString("当前 AI 生成服务暂时不可用。我已基于你本人可见的系统数据给出保守摘要：\n\n")
	if len(snapshot.Applications) > 0 {
		b.WriteString("## 投递进度\n\n")
		for _, app := range snapshot.Applications {
			b.WriteString("- **")
			b.WriteString(nonEmpty(app.JobTitle, fmt.Sprintf("岗位 %d", app.JobID)))
			b.WriteString("** — ")
			b.WriteString(nonEmpty(app.StatusText, app.StatusKey))
			if app.AppliedAt != "" {
				b.WriteString("，投递于 ")
				b.WriteString(app.AppliedAt)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(snapshot.Interviews) > 0 {
		b.WriteString("## 面试安排\n\n")
		for _, interview := range snapshot.Interviews {
			b.WriteString("- **")
			b.WriteString(nonEmpty(interview.Title, fmt.Sprintf("第 %d 轮面试", interview.RoundNo)))
			b.WriteString("**")
			if interview.JobTitle != "" {
				b.WriteString(" — ")
				b.WriteString(interview.JobTitle)
			}
			if interview.ScheduledAt != "" {
				b.WriteString("，时间 ")
				b.WriteString(interview.ScheduledAt)
			}
			if interview.Status != "" {
				b.WriteString("，状态 ")
				b.WriteString(interview.Status)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(snapshot.Offers) > 0 {
		b.WriteString("## Offer\n\n")
		for _, offer := range snapshot.Offers {
			b.WriteString("- **")
			b.WriteString(nonEmpty(offer.Title, fmt.Sprintf("Offer %d", offer.OfferID)))
			b.WriteString("** — ")
			b.WriteString(offer.Status)
			if offer.ExpiresAt != "" {
				b.WriteString("，有效期至 ")
				b.WriteString(offer.ExpiresAt)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if snapshot.Resume.Available {
		b.WriteString("## 简历\n\n")
		b.WriteString("- 已读取你的有效简历")
		if snapshot.Resume.FileName != "" {
			b.WriteString("：**")
			b.WriteString(snapshot.Resume.FileName)
			b.WriteString("**")
		}
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

func candidateRuntimeContextHasData(snapshot CandidateRuntimeContext) bool {
	return len(snapshot.Applications) > 0 || snapshot.Resume.Available || len(snapshot.Jobs) > 0 || len(snapshot.Interviews) > 0 || len(snapshot.Offers) > 0
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringSliceContains(values []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	for _, value := range values {
		if strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}

func normalizedStringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out[trimmed] = true
		}
	}
	return out
}

func agentConfigID(agent *pb.AgentConfigInfo) int64 {
	if agent == nil {
		return 0
	}
	return agent.GetId()
}

func agentConfigType(agent *pb.AgentConfigInfo) string {
	if agent == nil {
		return ""
	}
	return agent.GetAgentType()
}

func promptTemplateID(template *pb.PromptTemplateInfo) int64 {
	if template == nil {
		return 0
	}
	return template.GetId()
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
		item := mapAgentRunItem(row)
		if steps, stepErr := s.store.ListAgentRunSteps(ctx, row.ID); stepErr == nil && len(steps) > 0 {
			item.Steps = make([]*pb.AgentRunStepItem, 0, len(steps))
			for _, step := range steps {
				item.Steps = append(item.Steps, mapAgentRunStepItem(step))
			}
		}
		items = append(items, item)
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
	payload := agentRunPayloadFromCreateRequest(req)
	run, idempotent, err := s.store.CreateAgentRun(ctx, fallbackAgentRun(req.GetHrId(), req.GetSessionId(), req.GetClientRequestId(), payload))
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
	s.dispatchAgentRun(run)
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
	activeExecution := s.cancelAgentRunExecution(req.GetRunId())
	if !activeExecution {
		if err := s.completeAgentRunCanceledLocked(ctx, run); err != nil {
			return nil, err
		}
		finalRun, finalFound, err := s.getRun(ctx, req.GetHrId(), req.GetRunId())
		if err != nil {
			return nil, err
		}
		if finalFound {
			run = finalRun
		}
	}
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
	run, found, err = s.updateAgentRunConfirmation(ctx, run, req)
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.ConfirmAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}
	run, found, err = s.updateRun(ctx, req.GetHrId(), req.GetRunId(), agentRunStatusRunning)
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.ConfirmAgentRunResponse{Code: 404, Msg: "agent run not found"}, nil
	}
	if _, err := s.appendAgentRunEvent(ctx, run.ID, "confirmation.accepted", agentRunConfirmationAcceptedPayload(req)); err != nil {
		return nil, err
	}
	s.dispatchAgentRun(run)
	return &pb.ConfirmAgentRunResponse{Code: 0, Msg: "success", Run: mapAgentRunSnapshot(run)}, nil
}

func (s *nativeAIService) updateAgentRunConfirmation(ctx context.Context, run AgentRunRow, req *pb.ConfirmAgentRunRequest) (AgentRunRow, bool, error) {
	if s.store == nil {
		return AgentRunRow{}, false, errAIStoreRequired
	}
	payload := agentRunPayloadFromRow(run)
	if len(req.GetAgentSkillIds()) > 0 {
		payload.AgentSkillIDs = append([]int64(nil), req.GetAgentSkillIds()...)
	}
	if req.GetAgentSkillSelectionConfirmed() {
		payload.AgentSkillSelectionConfirmed = true
	}
	if req.GetAgentSkillSelectionMessageId() != 0 {
		payload.AgentSkillSelectionMessageID = req.GetAgentSkillSelectionMessageId()
	}
	if strings.TrimSpace(req.GetConfirmationPayloadJson()) != "" {
		payload.ConfirmationPayloadJSON = strings.TrimSpace(req.GetConfirmationPayloadJson())
	}
	if strings.TrimSpace(req.GetClientRequestId()) != "" {
		payload.ConfirmationClientRequestID = strings.TrimSpace(req.GetClientRequestId())
	}
	planJSON := agentRunPlanJSON(payload)
	optionContextJSON := agentRunConfirmationOptionContext(req)
	return s.store.UpdateAgentRunPlan(ctx, run.OwnerID, run.ID, planJSON, optionContextJSON)
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

func (s *nativeAIService) complete(ctx context.Context, prompt string, modelID int64, opts ...ChatCompletionOptions) (string, error) {
	if s.provider == nil {
		return "", errAIProviderRequired
	}
	var completionOpts ChatCompletionOptions
	if len(opts) > 0 {
		completionOpts = opts[0]
	}
	if optionsAware, ok := s.provider.(RuntimeOptionsChatProvider); ok && completionOpts.TemperatureOverride != nil {
		reply, err := optionsAware.CompleteWithOptions(ctx, prompt, modelID, completionOpts)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(reply) == "" {
			return "AI provider returned an empty response.", nil
		}
		return reply, nil
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

func (s *nativeAIService) dispatchAgentRun(run AgentRunRow) {
	ctx, cancel := context.WithCancel(context.Background())
	s.storeAgentRunCancel(run.ID, cancel)
	go func() {
		_ = s.executeAgentRun(ctx, run)
	}()
}

func (s *nativeAIService) executeAgentRun(ctx context.Context, run AgentRunRow) error {
	defer s.clearAgentRunCancel(run.ID)

	current, shouldExecute, err := s.beginAgentRunExecution(ctx, run)
	if err != nil || !shouldExecute {
		return err
	}
	payload := agentRunPayloadFromRow(current)
	result, err := s.runHRChatRuntimeWithOptions(ctx, &pb.ChatRequest{
		HrId:                         current.OwnerID,
		SessionId:                    current.SessionID,
		Message:                      payload.Message,
		ApplicationId:                payload.ApplicationID,
		ModelId:                      payload.ModelID,
		SkillCapabilityKeys:          payload.SkillCapabilityKeys,
		AgentSkillIds:                payload.AgentSkillIDs,
		AgentSkillSelectionConfirmed: payload.AgentSkillSelectionConfirmed,
		AgentSkillSelectionMessageId: payload.AgentSkillSelectionMessageID,
	}, s.agentRunChatEmitter(current.ID), hrChatRuntimeOptions{
		reuseExistingUserMessage: run.Status == agentRunStatusRunning || run.Status == agentRunStatusWaitingConfirmation,
		agentRunID:               current.ID,
	})
	if err != nil {
		if agentRunExecutionCanceled(ctx, err) {
			return s.finishAgentRunCanceled(ctx, current)
		}
		return s.finishAgentRunFailed(ctx, current, err)
	}
	if agentRunExecutionCanceled(ctx, nil) {
		return s.finishAgentRunCanceled(ctx, current)
	}
	if result.providerUnavailable && !result.fallbackUsed {
		return s.finishAgentRunFailed(ctx, current, errAIProviderRequired)
	}
	return s.finishAgentRunSucceeded(ctx, current, result, payload.ModelID)
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

func (s *nativeAIService) finishAgentRunSucceeded(ctx context.Context, run AgentRunRow, result hrChatRuntimeResult, modelID int64) error {
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
	reply := result.reply
	if strings.TrimSpace(reply) != "" {
		if _, err := s.appendAgentRunEvent(storeCtx, run.ID, "assistant.delta", fmt.Sprintf(`{"status":%q,"delta":%q}`, agentRunStatusRunning, reply)); err != nil {
			return err
		}
	}
	if result.contextUsage != nil || result.candidateName != "" || result.jobTitle != "" || result.status != 0 {
		if _, err := s.appendAgentRunEvent(storeCtx, run.ID, "run.result", agentRunResultPayload(result)); err != nil {
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

func (s *nativeAIService) agentRunChatEmitter(runID int64) hrChatStreamEmitter {
	return func(event *pb.ChatStreamResponse) error {
		if event == nil {
			return nil
		}
		eventType := agentRunEventTypeFromChatEvent(event)
		payload := map[string]any{
			"status":        agentRunStatusRunning,
			"source_event":  event.GetEventType(),
			"event_message": event.GetEventMessage(),
		}
		if event.GetDelta() != "" {
			payload["delta"] = event.GetDelta()
		}
		if event.GetToolName() != "" {
			payload["tool_name"] = event.GetToolName()
		}
		if event.GetErrorType() != "" {
			payload["error_type"] = event.GetErrorType()
		}
		if event.GetContextUsage() != nil {
			payload["result_metadata"] = map[string]any{
				"context_usage": contextUsagePayload(event.GetContextUsage()),
			}
		}
		if event.GetEventType() == "error" && event.GetErrorType() == "TOOL_ERROR" {
			payload["error_message"] = event.GetEventMessage()
		}
		_, err := s.appendAgentRunEvent(agentRunStoreContext(context.Background()), runID, eventType, marshalJSONString(payload))
		return err
	}
}

func agentRunEventTypeFromChatEvent(event *pb.ChatStreamResponse) string {
	switch event.GetEventType() {
	case "tool_calling":
		return "tool.started"
	case "tool_done":
		return "tool.finished"
	case "context_usage", "thinking", "generating", "fallback":
		return "process.delta"
	case "error":
		if event.GetErrorType() == "TOOL_ERROR" {
			return "tool.finished"
		}
		return "run.error"
	default:
		return "process.delta"
	}
}

func agentRunResultPayload(result hrChatRuntimeResult) string {
	payload := map[string]any{
		"status": agentRunStatusSucceeded,
		"result_metadata": map[string]any{
			"application_id": result.session.ApplicationID,
			"candidate_name": result.candidateName,
			"job_title":      result.jobTitle,
			"status":         result.status,
			"context_usage":  contextUsagePayload(result.contextUsage),
			"raw_json":       buildHRProcessContent(nil, result.contextUsage, result.fallbackUsed, hrRuntimeGovernanceContext{}),
		},
	}
	return marshalJSONString(payload)
}

func contextUsagePayload(usage *pb.ContextUsageInfo) map[string]any {
	if usage == nil {
		return nil
	}
	return map[string]any{
		"model_id":                   usage.GetModelId(),
		"model_name":                 usage.GetModelName(),
		"context_window_tokens":      usage.GetContextWindowTokens(),
		"max_output_tokens":          usage.GetMaxOutputTokens(),
		"prompt_tokens_estimated":    usage.GetPromptTokensEstimated(),
		"remaining_tokens_estimated": usage.GetRemainingTokensEstimated(),
		"usage_ratio":                usage.GetUsageRatio(),
		"estimated":                  usage.GetEstimated(),
		"source":                     usage.GetSource(),
		"stage":                      usage.GetStage(),
	}
}

func agentRunPayloadFromCreateRequest(req *pb.CreateAgentRunRequest) agentRunDurablePayload {
	if req == nil {
		return agentRunDurablePayload{}
	}
	return agentRunDurablePayload{
		Message:                      req.GetMessage(),
		ActionType:                   req.GetActionType(),
		ActionPayloadJSON:            req.GetActionPayloadJson(),
		ApplicationID:                req.GetApplicationId(),
		ModelID:                      req.GetModelId(),
		SkillCapabilityKeys:          append([]string(nil), req.GetSkillCapabilityKeys()...),
		AgentSkillIDs:                append([]int64(nil), req.GetAgentSkillIds()...),
		AgentSkillSelectionConfirmed: req.GetAgentSkillSelectionConfirmed(),
		AgentSkillSelectionMessageID: req.GetAgentSkillSelectionMessageId(),
	}
}

func agentRunPlanJSON(payload agentRunDurablePayload) string {
	return marshalJSONString(map[string]any{"durable_request": payload})
}

func agentRunConfirmationAcceptedPayload(req *pb.ConfirmAgentRunRequest) string {
	payload := map[string]any{
		"status":       agentRunStatusRunning,
		"confirmation": agentRunConfirmationPayload(req),
	}
	return marshalJSONString(payload)
}

func agentRunConfirmationOptionContext(req *pb.ConfirmAgentRunRequest) string {
	return marshalJSONString(map[string]any{
		"confirmation": agentRunConfirmationPayload(req),
	})
}

func agentRunConfirmationPayload(req *pb.ConfirmAgentRunRequest) map[string]any {
	if req == nil {
		return nil
	}
	payload := map[string]any{
		"required":                    false,
		"recommended_agent_skill_ids": append([]int64(nil), req.GetAgentSkillIds()...),
		"user_message_id":             req.GetAgentSkillSelectionMessageId(),
	}
	if strings.TrimSpace(req.GetConfirmationPayloadJson()) != "" {
		payload["raw_json"] = strings.TrimSpace(req.GetConfirmationPayloadJson())
	}
	return payload
}

func agentRunPayloadFromRow(run AgentRunRow) agentRunDurablePayload {
	var payload agentRunDurablePayload
	if strings.TrimSpace(run.PlanJSON) != "" {
		var wrapper struct {
			DurableRequest agentRunDurablePayload `json:"durable_request"`
		}
		if err := json.Unmarshal([]byte(run.PlanJSON), &wrapper); err == nil {
			payload = wrapper.DurableRequest
		}
	}
	if payload.ModelID == 0 {
		payload.ModelID = run.ModelID
	}
	return payload
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

func (s *nativeAIService) cancelAgentRunExecution(runID int64) bool {
	s.runCancelMu.Lock()
	entry := s.runCancels[runID]
	s.runCancelMu.Unlock()
	if entry != nil && entry.cancel != nil {
		entry.cancel()
		return true
	}
	return false
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
		Status              string          `json:"status"`
		Delta               string          `json:"delta"`
		SnapshotText        string          `json:"snapshot_text"`
		ResultMetadata      json.RawMessage `json:"result_metadata"`
		Confirmation        json.RawMessage `json:"confirmation"`
		ConfirmationRequest json.RawMessage `json:"confirmation_request"`
		ToolName            string          `json:"tool_name"`
		ErrorType           string          `json:"error_type"`
		ErrorMessage        string          `json:"error_message"`
	}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return
	}
	event.Status = payload.Status
	event.Delta = payload.Delta
	event.SnapshotText = payload.SnapshotText
	event.ResultMetadata = parseAgentRunResultMetadata(payload.ResultMetadata)
	event.Confirmation = parseAgentRunConfirmation(payload.Confirmation)
	if event.Confirmation == nil {
		event.Confirmation = parseAgentRunConfirmation(payload.ConfirmationRequest)
	}
	event.ToolName = payload.ToolName
	event.ErrorType = payload.ErrorType
	event.ErrorMessage = payload.ErrorMessage
}

func applyAgentRunSnapshotPayload(snapshot *pb.AgentRunSnapshot, payloadJSON string) {
	if snapshot == nil || strings.TrimSpace(payloadJSON) == "" {
		return
	}
	var payload struct {
		ResultMetadata      json.RawMessage `json:"result_metadata"`
		ConfirmationRequest json.RawMessage `json:"confirmation_request"`
		Confirmation        json.RawMessage `json:"confirmation"`
	}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return
	}
	if snapshot.ResultMetadata == nil {
		snapshot.ResultMetadata = parseAgentRunResultMetadata(payload.ResultMetadata)
	}
	if snapshot.ConfirmationRequest == nil {
		snapshot.ConfirmationRequest = parseAgentRunConfirmation(payload.ConfirmationRequest)
	}
	if snapshot.ConfirmationRequest == nil {
		snapshot.ConfirmationRequest = parseAgentRunConfirmation(payload.Confirmation)
	}
}

func parseAgentRunResultMetadata(raw json.RawMessage) *pb.AgentRunResultMetadata {
	if isEmptyAgentRunJSON(raw) {
		return nil
	}
	result := &pb.AgentRunResultMetadata{}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(raw, result); err != nil {
		return &pb.AgentRunResultMetadata{RawJson: string(raw)}
	}
	return result
}

func parseAgentRunConfirmation(raw json.RawMessage) *pb.AgentRunConfirmationPayload {
	if isEmptyAgentRunJSON(raw) {
		return nil
	}
	confirmation := &pb.AgentRunConfirmationPayload{}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(raw, confirmation); err != nil {
		return &pb.AgentRunConfirmationPayload{RawJson: string(raw)}
	}
	return confirmation
}

func isEmptyAgentRunJSON(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed == "" || trimmed == "null" || trimmed == "{}"
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
	provider     ChatProvider
	structured   *recruitingruntime.Runtime
	policy       recruitingruntime.RuntimePolicy
	auth         pb.AuthServiceClient
	applications pb.ApplicationOwnerServiceClient
	jobs         pb.JobServiceClient
	observer     recruitingruntime.Observer
}

func (s nativeRecruitingIntelligenceService) recruitingObserver() recruitingruntime.Observer {
	if s.observer != nil {
		return s.observer
	}
	observer := newRecruitingRuntimeObserver()
	return observer
}

func boundedRecruitingCount(value int) int32 {
	if value <= 0 {
		return 0
	}
	if value > 1000 {
		return 1000
	}
	return int32(value)
}

func (s nativeRecruitingIntelligenceService) observeRecruiting(ctx context.Context, event recruitingruntime.Observation) {
	s.recruitingObserver().ObserveRecruitingRuntime(ctx, recruitingruntime.NormalizeObservation(event))
}

func recruitingOperationEvent(ctx context.Context, operation, resourceType string, resourceID int64, stage, category, outcome string, terminal bool, started time.Time) recruitingruntime.Observation {
	return recruitingruntime.Observation{
		Operation: operation, RequestID: platformmetadata.GetRequestID(ctx), ResourceType: resourceType, ResourceID: resourceID,
		Stage: stage, Terminal: terminal, Category: category, Outcome: outcome, Duration: time.Since(started),
	}
}

// recruitingOperationFinalizer owns the single terminal observation for one
// public method invocation. It is installed before request validation and
// emitted by defer, so every return path is covered and intermediate stages
// cannot accidentally claim overall success.
type recruitingOperationFinalizer struct {
	service nativeRecruitingIntelligenceService
	ctx     context.Context
	event   recruitingruntime.Observation
	started time.Time
	emitted bool
}

func newRecruitingOperationFinalizer(service nativeRecruitingIntelligenceService, ctx context.Context, operation, resourceType string, resourceID int64) *recruitingOperationFinalizer {
	started := time.Now()
	return &recruitingOperationFinalizer{
		service: service,
		ctx:     ctx,
		event:   recruitingOperationEvent(ctx, operation, resourceType, resourceID, "operation", "domain_validation_failure", "error", true, started),
		started: started,
	}
}

func (f *recruitingOperationFinalizer) classify(category, outcome string) {
	f.event.Category = category
	f.event.Outcome = outcome
}

func (f *recruitingOperationFinalizer) finalize() {
	if f == nil || f.emitted {
		return
	}
	f.emitted = true
	f.event.Terminal = true
	f.event.Duration = time.Since(f.started)
	f.service.observeRecruiting(f.ctx, f.event)
}

func recruitingTerminalCategoryForCode(code int32) string {
	switch code {
	case errs.OK:
		return "success"
	case errs.ErrBadRequest:
		return "domain_validation_failure"
	case errs.ErrForbidden:
		return "authorization_failure"
	case 404:
		return "not_found"
	default:
		return "source_failure"
	}
}

func recruitingTerminalCategoryForAuthError(authErr *recruitingAuthError) string {
	if authErr == nil {
		return "success"
	}
	if authErr.terminalCategory != "" {
		return authErr.terminalCategory
	}
	return recruitingTerminalCategoryForCode(authErr.code)
}

func newRecruitingStructuredRuntime(store AIStore, provider ChatProvider, policies ...recruitingruntime.RuntimePolicy) *recruitingruntime.Runtime {
	promptStore, promptOK := store.(recruitingruntime.PromptStore)
	structuredProvider, providerOK := provider.(recruitingruntime.StructuredCompletionProvider)
	if !promptOK || !providerOK {
		return nil
	}
	return recruitingruntime.NewRuntimeWithObserver(recruitingruntime.NewPromptLoader(promptStore), structuredProvider, newRecruitingRuntimeObserver(), policies...)
}

func (s nativeRecruitingIntelligenceService) hasRecruitingStructuredRuntime() bool {
	if s.structured != nil {
		return true
	}
	_, promptOK := s.store.(recruitingruntime.PromptStore)
	_, providerOK := s.provider.(recruitingruntime.StructuredCompletionProvider)
	return promptOK && providerOK
}

type recruitingReadStore interface {
	GetRecruitingApplicationByID(ctx context.Context, applicationID int64) (RecruitingApplicationContext, bool, error)
	GetLatestRecruitingApplicationByResumeID(ctx context.Context, resumeID int64) (RecruitingApplicationContext, bool, error)
	GetRecruitingResumeProfileByID(ctx context.Context, profileID uint64) (RecruitingResumeProfileRow, bool, error)
	GetCurrentRecruitingResumeProfileByResumeID(ctx context.Context, resumeID int64) (RecruitingResumeProfileRow, bool, error)
	GetRecruitingResumeProfileSnapshot(ctx context.Context, profileID uint64) (RecruitingResumeProfileSnapshot, bool, error)
	GetRecruitingCandidateMatchEvaluationSnapshot(ctx context.Context, evaluationID uint64) (RecruitingCandidateMatchSnapshot, bool, error)
	GetRecruitingCandidateMatchEvaluationSnapshotByApplicationVersion(ctx context.Context, applicationID int64, version int32) (RecruitingCandidateMatchSnapshot, bool, error)
	GetRecruitingCandidateMatchEvaluationSnapshotByApplicationAgentRunID(ctx context.Context, applicationID int64, agentRunID uint64) (RecruitingCandidateMatchSnapshot, bool, error)
	GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(ctx context.Context, applicationID int64) (RecruitingCandidateMatchSnapshot, bool, error)
	ListCurrentRecruitingApplicationsByJobID(ctx context.Context, jobID int64) ([]RecruitingApplicationContext, error)
	ListLatestRecruitingCandidateMatchEvaluationsByApplicationIDs(ctx context.Context, applicationIDs []int64) ([]RecruitingCandidateMatchEvaluationRow, error)
}

type recruitingResumeProfileGenerationStore interface {
	GetRecruitingResumeSource(ctx context.Context, resumeID int64) (RecruitingResumeSource, bool, error)
	SaveRecruitingResumeProfileDraft(ctx context.Context, draft RecruitingResumeProfileDraft) (RecruitingResumeProfileSnapshot, error)
}

type recruitingCandidateMatchGenerationStore interface {
	GetRecruitingMatchSource(ctx context.Context, applicationID int64) (RecruitingMatchSource, bool, error)
	SaveRecruitingCandidateMatchDraft(ctx context.Context, draft RecruitingCandidateMatchDraft) (RecruitingCandidateMatchSnapshot, error)
}

type recruitingAuthError struct {
	code             int32
	message          string
	err              error
	terminalCategory string
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
		return nil, &recruitingAuthError{code: errs.ErrInternal, message: "application owner service is not configured", terminalCategory: "configuration_failure"}
	}
	resp, err := s.applications.GetApplicationSnapshot(ctx, &pb.GetApplicationSnapshotRequest{ApplicationId: applicationID})
	if err != nil {
		return nil, &recruitingAuthError{code: errs.ErrInternal, message: "get application snapshot failed", err: err}
	}
	if resp == nil {
		return nil, &recruitingAuthError{code: errs.ErrInternal, message: "get application snapshot returned nil response", terminalCategory: "configuration_failure"}
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

func (s nativeRecruitingIntelligenceService) authorizeRecruitingAIPermission(ctx context.Context, staffUserID int64, resourceType string, resourceID int64) *recruitingAuthError {
	if authErr := s.verifyRecruitingStaffActor(ctx, staffUserID); authErr != nil {
		return authErr
	}
	if s.auth == nil {
		return &recruitingAuthError{code: errs.ErrInternal, message: "auth service is not configured", terminalCategory: "configuration_failure"}
	}
	resp, err := s.auth.AuthorizeInternal(ctx, &pb.AuthorizeInternalRequest{
		ActorUserId:   staffUserID,
		PermissionKey: "ai.hr.use",
		ResourceType:  resourceType,
		ResourceId:    resourceID,
		RequestId:     platformmetadata.GetRequestID(ctx),
		ClientIp:      platformmetadata.GetClientIP(ctx),
	})
	if err != nil {
		return &recruitingAuthError{code: errs.ErrInternal, message: "authorize ai permission failed", err: err}
	}
	if resp == nil {
		return &recruitingAuthError{code: errs.ErrInternal, message: "authorize ai permission returned nil response", terminalCategory: "configuration_failure"}
	}
	if resp.GetCode() == errs.OK && resp.GetAllowed() {
		return nil
	}
	if resp.GetCode() == errs.OK {
		return &recruitingAuthError{code: errs.ErrForbidden, message: recruitingMessageOrDefault(resp.GetReason(), "AI HR permission denied")}
	}
	return &recruitingAuthError{code: resp.GetCode(), message: recruitingMessageOrDefault(resp.GetMsg(), "AI HR permission denied")}
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
	resumeID, accessResp, _ := s.resolveResumeProfileAccess(ctx, req)
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

func (s nativeRecruitingIntelligenceService) ParseResumeProfile(ctx context.Context, req *pb.ParseResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	finalizer := newRecruitingOperationFinalizer(s, ctx, "resume_profile", "resume", req.GetResumeId())
	defer finalizer.finalize()
	if req.GetApplicationId() <= 0 && req.GetResumeId() <= 0 {
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "resume_id or application_id is required"}, nil
	}
	if s.store == nil {
		finalizer.classify("configuration_failure", "error")
		return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "recruiting read store is not configured"}, nil
	}
	if generationStore, ok := s.store.(recruitingResumeProfileGenerationStore); ok {
		resumeID, accessResp, accessAuthErr := s.resolveResumeProfileAccess(ctx, &pb.GetResumeProfileRequest{StaffUserId: req.GetStaffUserId(), ResumeId: req.GetResumeId(), ApplicationId: req.GetApplicationId()})
		finalizer.event.ResourceID = resumeID
		if accessResp != nil {
			if accessAuthErr != nil {
				finalizer.classify(recruitingTerminalCategoryForAuthError(accessAuthErr), "error")
			} else {
				finalizer.classify(recruitingTerminalCategoryForCode(accessResp.GetCode()), "error")
			}
			return accessResp, nil
		}
		if authErr := s.authorizeRecruitingAIPermission(ctx, req.GetStaffUserId(), "resume", resumeID); authErr != nil {
			finalizer.classify(recruitingTerminalCategoryForAuthError(authErr), "error")
			return &pb.GetResumeProfileResponse{Code: authErr.code, Msg: authErr.message}, nil
		}
		sourceStarted := time.Now()
		source, found, err := generationStore.GetRecruitingResumeSource(ctx, resumeID)
		if err != nil {
			s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "source", "source_failure", "error", false, sourceStarted))
			finalizer.classify("source_failure", "error")
			return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
		}
		if !found {
			finalizer.classify("not_found", "error")
			return &pb.GetResumeProfileResponse{Code: 404, Msg: "resume not found"}, nil
		}
		if strings.TrimSpace(source.ParsedText) == "" {
			finalizer.classify("domain_validation_failure", "error")
			return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "resume parsed_text is empty"}, nil
		}
		sourceEvent := recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "source", "success", "success", false, sourceStarted)
		sourceEvent.InputCount = 1
		s.observeRecruiting(ctx, sourceEvent)
		totalCtx, parseCtx, cancel := s.policy.ResumeExecutionContexts(ctx)
		defer cancel()
		parseCtx = recruitingruntime.WithObservationMetadata(parseCtx, platformmetadata.GetRequestID(ctx), "resume", resumeID)
		generationStarted := time.Now()
		draft, err := s.generateResumeProfileDraft(parseCtx, source)
		if err != nil {
			category := recruitingruntime.ObservationCategoryForError(err)
			if category == "provider_failure" && !s.hasRecruitingStructuredRuntime() {
				category = "configuration_failure"
			}
			if errors.Is(err, context.DeadlineExceeded) || parseCtx.Err() != nil {
				category = "timeout"
			}
			s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "generation", category, "error", false, generationStarted))
			finalizer.classify(category, "error")
			return &pb.GetResumeProfileResponse{Code: configCodeUnavailable, Msg: err.Error()}, nil
		}
		generationEvent := recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "generation", "success", "success", false, generationStarted)
		generationEvent.ParserVersion = draft.ParserVersion
		generationEvent.OutputCount = boundedRecruitingCount(1 + len(draft.Educations) + len(draft.Experiences) + len(draft.Projects) + len(draft.Skills))
		s.observeRecruiting(ctx, generationEvent)
		if err := totalCtx.Err(); err != nil {
			timeoutEvent := recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "persistence", "timeout", "error", false, time.Now())
			timeoutEvent.ParserVersion, timeoutEvent.OutputCount = draft.ParserVersion, generationEvent.OutputCount
			s.observeRecruiting(ctx, timeoutEvent)
			finalizer.classify("timeout", "error")
			finalizer.event.ParserVersion, finalizer.event.OutputCount = draft.ParserVersion, generationEvent.OutputCount
			return &pb.GetResumeProfileResponse{Code: configCodeUnavailable, Msg: "resume profile extraction failed (timeout)"}, nil
		}
		persistenceStarted := time.Now()
		snapshot, err := generationStore.SaveRecruitingResumeProfileDraft(totalCtx, draft)
		if err != nil {
			persistenceEvent := recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "persistence", "persistence_failure", "error", false, persistenceStarted)
			persistenceEvent.ParserVersion, persistenceEvent.OutputCount = draft.ParserVersion, generationEvent.OutputCount
			s.observeRecruiting(ctx, persistenceEvent)
			finalizer.classify("persistence_failure", "error")
			finalizer.event.ParserVersion, finalizer.event.OutputCount = draft.ParserVersion, generationEvent.OutputCount
			return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
		}
		s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "persistence", "success", "success", false, persistenceStarted))
		finalizer.classify("success", "success")
		finalizer.event.ParserVersion = draft.ParserVersion
		if draft.ParserVersion == recruitingruntime.ResumeHeuristicParserVersion {
			finalizer.event.Category, finalizer.event.Fallback = "fallback_success", "heuristic"
		}
		finalizer.event.OutputCount = generationEvent.OutputCount
		return &pb.GetResumeProfileResponse{Code: errs.OK, Msg: "success", Profile: recruitingResumeProfileSnapshotPB(snapshot)}, nil
	}
	resp, err := s.GetResumeProfile(ctx, &pb.GetResumeProfileRequest{
		StaffUserId:   req.GetStaffUserId(),
		ResumeId:      req.GetResumeId(),
		ApplicationId: req.GetApplicationId(),
	})
	if err != nil || resp == nil || resp.GetCode() == errs.OK {
		if err != nil || resp == nil {
			finalizer.classify("source_failure", "error")
		} else {
			finalizer.classify("success", "success")
		}
		return resp, err
	}
	finalizer.classify(recruitingTerminalCategoryForCode(resp.GetCode()), "error")
	if resp.GetCode() == 404 && resp.GetMsg() == "current resume profile not found" {
		resp.Msg = "current resume profile not found and resume profile parser is not configured in native runtime"
	}
	return resp, nil
}

func (s nativeRecruitingIntelligenceService) EvaluateCandidateMatch(ctx context.Context, req *pb.EvaluateCandidateMatchRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	finalizer := newRecruitingOperationFinalizer(s, ctx, "candidate_match", "application", req.GetApplicationId())
	defer finalizer.finalize()
	if req.GetApplicationId() <= 0 {
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrBadRequest, Msg: "application_id is required"}, nil
	}
	if s.store == nil {
		finalizer.classify("configuration_failure", "error")
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: "recruiting read store is not configured"}, nil
	}
	if _, authErr := s.authorizeRecruitingApplication(ctx, req.GetStaffUserId(), req.GetApplicationId()); authErr != nil {
		finalizer.classify(recruitingTerminalCategoryForAuthError(authErr), "error")
		return recruitingMatchAuthResponse(authErr), nil
	}
	if generationStore, ok := s.store.(recruitingCandidateMatchGenerationStore); ok && s.policy.CandidateMatchEnabled() {
		if authErr := s.authorizeRecruitingAIPermission(ctx, req.GetStaffUserId(), "application", req.GetApplicationId()); authErr != nil {
			finalizer.classify(recruitingTerminalCategoryForAuthError(authErr), "error")
			return recruitingMatchAuthResponse(authErr), nil
		}
		totalCtx, generationCtx, cancel := s.policy.CandidateMatchExecutionContexts(ctx)
		defer cancel()
		generationCtx = recruitingruntime.WithObservationMetadata(generationCtx, platformmetadata.GetRequestID(ctx), "application", req.GetApplicationId())
		sourceStarted := time.Now()
		source, found, err := generationStore.GetRecruitingMatchSource(generationCtx, req.GetApplicationId())
		if err != nil {
			category := "source_failure"
			if generationCtx.Err() != nil || totalCtx.Err() != nil {
				category = "timeout"
				s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "source", category, "error", false, sourceStarted))
				finalizer.classify(category, "error")
				return &pb.GetCandidateMatchEvaluationResponse{Code: configCodeUnavailable, Msg: "candidate match evaluation failed (timeout or cancellation)"}, nil
			}
			s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "source", category, "error", false, sourceStarted))
			finalizer.classify(category, "error")
			return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
		}
		if !found {
			finalizer.classify("not_found", "error")
			return &pb.GetCandidateMatchEvaluationResponse{Code: 404, Msg: "application match source not found"}, nil
		}
		sourceEvent := recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "source", "success", "success", false, sourceStarted)
		sourceEvent.InputCount = 1
		s.observeRecruiting(ctx, sourceEvent)
		generationStarted := time.Now()
		draft, err := s.generateCandidateMatchDraft(generationCtx, source, req.GetAgentRunId())
		if err != nil {
			category := recruitingruntime.ObservationCategoryForError(err)
			if s.policy.CandidateMatchExecutionPlan().StructuredCompletionEnabled() && !s.hasRecruitingStructuredRuntime() {
				category = "configuration_failure"
			}
			if generationCtx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				category = "timeout"
			}
			s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "aggregation", category, "error", false, generationStarted))
			finalizer.classify(category, "error")
			return &pb.GetCandidateMatchEvaluationResponse{Code: configCodeUnavailable, Msg: err.Error()}, nil
		}
		aggregationEvent := recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "aggregation", "success", "success", false, generationStarted)
		aggregationEvent.ScorerVersion = draft.ScorerVersion
		aggregationEvent.RequirementCount = boundedRecruitingCount(draft.RequirementCount)
		aggregationEvent.EvidenceCount = boundedRecruitingCount(len(draft.Evidence))
		s.observeRecruiting(ctx, aggregationEvent)
		if totalCtx.Err() != nil {
			timeoutEvent := recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "persistence", "timeout", "error", false, time.Now())
			timeoutEvent.ScorerVersion, timeoutEvent.RequirementCount, timeoutEvent.EvidenceCount = draft.ScorerVersion, aggregationEvent.RequirementCount, aggregationEvent.EvidenceCount
			s.observeRecruiting(ctx, timeoutEvent)
			finalizer.classify("timeout", "error")
			finalizer.event.ScorerVersion, finalizer.event.RequirementCount, finalizer.event.EvidenceCount = draft.ScorerVersion, aggregationEvent.RequirementCount, aggregationEvent.EvidenceCount
			return &pb.GetCandidateMatchEvaluationResponse{Code: configCodeUnavailable, Msg: "candidate match evaluation failed (timeout or cancellation)"}, nil
		}
		persistenceStarted := time.Now()
		snapshot, err := generationStore.SaveRecruitingCandidateMatchDraft(totalCtx, draft)
		if err != nil {
			persistenceEvent := recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "persistence", "persistence_failure", "error", false, persistenceStarted)
			persistenceEvent.ScorerVersion, persistenceEvent.RequirementCount, persistenceEvent.EvidenceCount = draft.ScorerVersion, aggregationEvent.RequirementCount, aggregationEvent.EvidenceCount
			s.observeRecruiting(ctx, persistenceEvent)
			finalizer.classify("persistence_failure", "error")
			finalizer.event.ScorerVersion, finalizer.event.RequirementCount, finalizer.event.EvidenceCount = draft.ScorerVersion, aggregationEvent.RequirementCount, aggregationEvent.EvidenceCount
			return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
		}
		s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "persistence", "success", "success", false, persistenceStarted))
		finalizer.classify("success", "success")
		finalizer.event.ScorerVersion, finalizer.event.RequirementCount, finalizer.event.EvidenceCount = draft.ScorerVersion, aggregationEvent.RequirementCount, aggregationEvent.EvidenceCount
		if draft.FallbackUsed {
			finalizer.event.Category, finalizer.event.Fallback = "fallback_success", "legacy_deterministic"
		}
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.OK, Msg: "success", Evaluation: recruitingCandidateMatchSnapshotPB(snapshot)}, nil
	}
	var (
		snapshot RecruitingCandidateMatchSnapshot
		found    bool
		err      error
	)
	if req.GetAgentRunId() > 0 {
		snapshot, found, err = s.store.GetRecruitingCandidateMatchEvaluationSnapshotByApplicationAgentRunID(ctx, req.GetApplicationId(), req.GetAgentRunId())
	} else {
		snapshot, found, err = s.store.GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(ctx, req.GetApplicationId())
	}
	if err != nil {
		finalizer.classify("source_failure", "error")
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
	}
	if !found {
		finalizer.classify("not_found", "error")
		if req.GetAgentRunId() > 0 {
			return &pb.GetCandidateMatchEvaluationResponse{Code: 404, Msg: "candidate match evaluation not found for agent_run_id and matcher is not configured in native runtime"}, nil
		}
		return &pb.GetCandidateMatchEvaluationResponse{Code: configCodeUnsupported, Msg: "candidate match evaluation not found and matcher is not configured in native runtime"}, nil
	}
	finalizer.classify("success", "success")
	return &pb.GetCandidateMatchEvaluationResponse{Code: errs.OK, Msg: "success", Evaluation: recruitingCandidateMatchSnapshotPB(snapshot)}, nil
}

func (s nativeRecruitingIntelligenceService) generateResumeProfileDraft(ctx context.Context, source RecruitingResumeSource) (RecruitingResumeProfileDraft, error) {
	structured := s.structured
	if structured == nil {
		promptStore, promptOK := s.store.(recruitingruntime.PromptStore)
		structuredProvider, providerOK := s.provider.(recruitingruntime.StructuredCompletionProvider)
		if promptOK && providerOK {
			structured = recruitingruntime.NewRuntimeWithObserver(recruitingruntime.NewPromptLoader(promptStore), structuredProvider, newRecruitingRuntimeObserver(), s.policy)
		}
	}
	result, err := recruitingruntime.NewResumeProfileExtractor(structured, s.policy).Extract(ctx, recruitingruntime.ResumeSource{
		ResumeID: source.ResumeID, UserID: source.UserID, FileName: source.FileName, ParsedText: source.ParsedText,
	})
	if err != nil {
		return RecruitingResumeProfileDraft{}, err
	}
	profile := result.Profile
	draft := RecruitingResumeProfileDraft{
		ResumeID:             source.ResumeID,
		UserID:               source.UserID,
		ParserVersion:        result.ParserVersion,
		InputHash:            result.InputHash,
		RawJSON:              result.RawJSON,
		FullName:             profile.FullName,
		Email:                profile.Email,
		Phone:                profile.Phone,
		Location:             profile.Location,
		Headline:             profile.Headline,
		Summary:              profile.Summary,
		TotalExperienceYears: profile.TotalExperienceYears,
		HighestDegree:        profile.HighestDegree,
		Educations:           make([]RecruitingResumeEducationRow, 0, len(profile.Educations)),
		Experiences:          make([]RecruitingResumeExperienceRow, 0, len(profile.Experiences)),
		Projects:             make([]RecruitingResumeProjectRow, 0, len(profile.Projects)),
		Skills:               make([]RecruitingResumeSkillRow, 0, len(profile.Skills)),
	}
	for index, row := range profile.Educations {
		draft.Educations = append(draft.Educations, RecruitingResumeEducationRow{
			School:      row.School,
			Degree:      row.Degree,
			Major:       row.Major,
			StartDate:   row.StartDate,
			EndDate:     row.EndDate,
			Description: row.Description,
			SortOrder:   int32(index + 1),
		})
	}
	for index, row := range profile.Experiences {
		isCurrent := int32(0)
		if row.IsCurrent {
			isCurrent = 1
		}
		draft.Experiences = append(draft.Experiences, RecruitingResumeExperienceRow{
			Company:          row.Company,
			Title:            row.Title,
			Location:         row.Location,
			StartDate:        row.StartDate,
			EndDate:          row.EndDate,
			IsCurrent:        isCurrent,
			Description:      row.Description,
			AchievementsJSON: marshalJSONString(row.Achievements),
			SortOrder:        int32(index + 1),
		})
	}
	for index, row := range profile.Projects {
		draft.Projects = append(draft.Projects, RecruitingResumeProjectRow{
			Name:             row.Name,
			Role:             row.Role,
			StartDate:        row.StartDate,
			EndDate:          row.EndDate,
			Description:      row.Description,
			TechnologiesJSON: marshalJSONString(row.Technologies),
			HighlightsJSON:   marshalJSONString(row.Highlights),
			SortOrder:        int32(index + 1),
		})
	}
	for index, row := range profile.Skills {
		draft.Skills = append(draft.Skills, RecruitingResumeSkillRow{
			Name:      row.Name,
			Category:  row.Category,
			Level:     row.Level,
			Years:     row.Years,
			Evidence:  row.Evidence,
			SortOrder: int32(index + 1),
		})
	}
	return draft, nil
}

func (s nativeRecruitingIntelligenceService) generateCandidateMatchDraft(ctx context.Context, source RecruitingMatchSource, agentRunID uint64) (RecruitingCandidateMatchDraft, error) {
	if !s.policy.CandidateMatchEnabled() {
		return RecruitingCandidateMatchDraft{}, recruitingruntime.ErrCandidateMatchPolicy
	}
	draft, _, err := s.generateCandidateMatchDraftWithShadow(ctx, source, agentRunID)
	return draft, err
}

type candidateMatchShadowComparison struct {
	PrimaryMode           recruitingruntime.CandidateMatchExecutionMode
	ShadowMode            recruitingruntime.CandidateMatchExecutionMode
	PrimaryScore          float64
	ShadowScore           float64
	ScoreDelta            float64
	PrimaryRecommendation string
	ShadowRecommendation  string
	PrimaryEvidenceCount  int
	ShadowEvidenceCount   int
	ShadowSucceeded       bool
}

func (s nativeRecruitingIntelligenceService) generateCandidateMatchDraftWithShadow(ctx context.Context, source RecruitingMatchSource, agentRunID uint64) (RecruitingCandidateMatchDraft, *candidateMatchShadowComparison, error) {
	plan := s.policy.CandidateMatchExecutionPlan()
	var (
		primary RecruitingCandidateMatchDraft
		err     error
	)
	switch plan.Primary() {
	case recruitingruntime.CandidateMatchExecutionDeterministic:
		primary = s.generateDeterministicCandidateMatchDraft(source, agentRunID, false)
	case recruitingruntime.CandidateMatchExecutionEnhanced:
		if s.structured == nil {
			// Candidate matching never routes through generic aggregate model
			// output. Policy decides whether missing structured dependencies fail
			// closed or switch to the deterministic legacy scorer below.
			err = errAIProviderRequired
		} else {
			primary, err = s.generateEnhancedCandidateMatchDraft(ctx, source, agentRunID)
		}
		if err != nil && s.policy.FallbacksEnabled() && candidateMatchFallbackEligible(err) {
			primary = s.generateDeterministicCandidateMatchDraft(source, agentRunID, true)
			err = nil
		}
	default:
		return RecruitingCandidateMatchDraft{}, nil, recruitingruntime.ErrCandidateMatchPolicy
	}
	if err != nil {
		return RecruitingCandidateMatchDraft{}, nil, err
	}
	if plan.Shadow() == recruitingruntime.CandidateMatchExecutionDisabled {
		return primary, nil, nil
	}

	comparison := &candidateMatchShadowComparison{
		PrimaryMode: plan.Primary(), ShadowMode: plan.Shadow(), PrimaryScore: primary.OverallScore,
		PrimaryRecommendation: primary.Recommendation, PrimaryEvidenceCount: len(primary.Evidence),
	}
	var shadow RecruitingCandidateMatchDraft
	switch plan.Shadow() {
	case recruitingruntime.CandidateMatchExecutionDeterministic:
		shadow = s.generateDeterministicCandidateMatchDraft(source, 0, false)
	case recruitingruntime.CandidateMatchExecutionEnhanced:
		if s.structured == nil {
			return primary, comparison, nil
		}
		shadow, err = s.generateEnhancedCandidateMatchDraft(ctx, source, 0)
	}
	if err != nil {
		return primary, comparison, nil
	}
	comparison.ShadowSucceeded = true
	comparison.ShadowScore = shadow.OverallScore
	comparison.ScoreDelta = shadow.OverallScore - primary.OverallScore
	comparison.ShadowRecommendation = shadow.Recommendation
	comparison.ShadowEvidenceCount = len(shadow.Evidence)
	return primary, comparison, nil
}

func candidateMatchFallbackEligible(err error) bool {
	return err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, recruitingruntime.ErrCandidateMatchPolicy)
}

func (s nativeRecruitingIntelligenceService) generateDeterministicCandidateMatchDraft(source RecruitingMatchSource, agentRunID uint64, fallbackUsed bool) RecruitingCandidateMatchDraft {
	index := recruitingruntime.BuildEvidenceIndex(recruitingCandidateEvidenceSource(source))
	resumeText, skillNames := recruitingLegacyScoringSource(source)
	result := recruitingruntime.ScoreCandidateMatchLegacy(recruitingruntime.LegacyCandidateMatchInput{
		JobID:    source.Job.JobID,
		Job:      recruitingruntime.JobRequirementSource{JobTitle: source.Job.Title, Department: source.Job.Department, Location: source.Job.Location, Description: source.Job.Description, Requirements: source.Job.Requirements},
		Evidence: index, ProfileID: source.Profile.Profile.ID, ResumeID: source.Application.ResumeID,
		ResumeParsedText: source.ResumeParsedText, ResumeText: resumeText, SkillNames: skillNames, ApplicationEducation: source.ApplicationEducation,
		CandidateProfile: recruitingLegacyCandidateProfile(source.CandidateProfile), FullName: source.Profile.Profile.FullName,
		Headline: source.Profile.Profile.Headline, Summary: source.Profile.Profile.Summary,
		HighestDegree: source.Profile.Profile.HighestDegree, ExperienceYears: source.Profile.Profile.TotalExperienceYears,
	})
	breakdown := map[string]any{
		"scorer_version": result.Breakdown.ScorerVersion, "input_hash": result.Breakdown.InputHash,
		"missing_requirements": result.Breakdown.MissingRequirements, "dimensions": result.Breakdown.Dimensions,
		"scorer_type": "legacy_deterministic", "fallback_used": fallbackUsed,
	}
	draft := RecruitingCandidateMatchDraft{
		ApplicationID: source.Application.ApplicationID, JobID: source.Application.JobID,
		CandidateUserID: source.Application.CandidateUserID, ResumeProfileID: source.Profile.Profile.ID,
		OverallScore: result.OverallScore, Recommendation: result.Recommendation, Summary: result.Summary,
		StrengthsJSON: marshalJSONString(result.Strengths), RisksJSON: marshalJSONString(result.Risks),
		ScoreBreakdownJSON: marshalJSONString(breakdown), ModelName: recruitingruntime.LegacyCandidateMatchScorerVersion,
		Evidence:      legacyCandidateMatchEvidenceRows(result.Evidence),
		ScorerVersion: recruitingruntime.LegacyCandidateMatchScorerVersion, FallbackUsed: fallbackUsed,
	}
	if agentRunID > 0 {
		draft.AgentRunID = &agentRunID
	}
	return draft
}

// recruitingLegacyScoringSource mirrors dev's buildCandidateMatchResumeText
// and resumeSkillNames inputs. These complete values exist only in memory for
// deterministic scoring and hashing; persisted evidence remains independently
// bounded and redacted through EvidenceIndex.
func recruitingLegacyScoringSource(source RecruitingMatchSource) (string, []string) {
	parts := []string{
		source.ResumeParsedText,
		source.Profile.Profile.FullName,
		source.Profile.Profile.Headline,
		source.Profile.Profile.Summary,
		source.Profile.Profile.HighestDegree,
	}
	if profile := source.CandidateProfile; profile != nil {
		parts = append(parts, profile.RealName, profile.Education, profile.School, profile.WorkExperience, profile.Skills)
	}
	for _, education := range source.Profile.Educations {
		parts = append(parts, education.School, education.Degree, education.Major, education.Description)
	}
	for _, experience := range source.Profile.Experiences {
		parts = append(parts, experience.Company, experience.Title, experience.Description, experience.AchievementsJSON)
	}
	for _, project := range source.Profile.Projects {
		parts = append(parts, project.Name, project.Role, project.Description, project.TechnologiesJSON, project.HighlightsJSON)
	}
	skillNames := make([]string, 0, len(source.Profile.Skills))
	for _, skill := range source.Profile.Skills {
		parts = append(parts, skill.Name, skill.Category, skill.Level, skill.Evidence)
		if strings.TrimSpace(skill.Name) != "" {
			skillNames = append(skillNames, skill.Name)
		}
	}
	sort.Strings(skillNames)
	return strings.Join(parts, " "), skillNames
}

func legacyCandidateMatchEvidenceRows(references []recruitingruntime.MatchEvidenceReference) []RecruitingCandidateMatchEvidenceRow {
	rows := make([]RecruitingCandidateMatchEvidenceRow, 0, len(references))
	for _, reference := range references {
		sourceID := reference.SourceID
		rows = append(rows, RecruitingCandidateMatchEvidenceRow{
			EvidenceType: "legacy_match", Dimension: "legacy", SourceTable: reference.SourceTable, SourceID: &sourceID,
			Snippet:      truncateRecruitingSensitiveSnippet(reference.Snippet, recruitingruntime.MaxCandidateEvidenceRunes),
			MetadataJSON: marshalJSONString(map[string]any{"reason": reference.Reason, "scorer_type": "legacy_deterministic"}),
		})
	}
	return rows
}

func (s nativeRecruitingIntelligenceService) generateStructuredCandidateMatchDraft(ctx context.Context, source RecruitingMatchSource, agentRunID uint64) (RecruitingCandidateMatchDraft, error) {
	return s.generateEnhancedCandidateMatchDraft(ctx, source, agentRunID)
}

func (s nativeRecruitingIntelligenceService) generateEnhancedCandidateMatchDraft(ctx context.Context, source RecruitingMatchSource, agentRunID uint64) (RecruitingCandidateMatchDraft, error) {
	if s.structured == nil {
		return RecruitingCandidateMatchDraft{}, errAIProviderRequired
	}
	jobSource := recruitingruntime.JobRequirementSource{
		JobTitle: source.Job.Title, Department: source.Job.Department, Location: source.Job.Location,
		Description: source.Job.Description, Requirements: source.Job.Requirements,
	}
	extraction, err := recruitingruntime.NewJobRequirementExtractor(s.structured, s.policy).Extract(ctx, jobSource)
	if err != nil {
		return RecruitingCandidateMatchDraft{}, err
	}
	if extraction.FallbackUsed {
		return RecruitingCandidateMatchDraft{}, recruitingruntime.ErrJobRequirementFallback
	}

	evidenceIndex := recruitingruntime.BuildEvidenceIndex(recruitingCandidateEvidenceSource(source))
	var results []recruitingruntime.RequirementMatchResult
	results, err = recruitingruntime.NewCandidateRequirementEvaluator(s.structured, s.policy).Evaluate(ctx, extraction.Profile, evidenceIndex)
	if err != nil {
		return RecruitingCandidateMatchDraft{}, err
	}

	fallbackUsed := extraction.FallbackUsed
	modelName := extraction.ModelName
	for _, result := range results {
		fallbackUsed = fallbackUsed || result.FallbackUsed
		if modelName == "" && result.ModelName != "" {
			modelName = result.ModelName
		}
	}
	if fallbackUsed {
		return RecruitingCandidateMatchDraft{}, recruitingruntime.ErrProvider
	}
	scorerType := string(recruitingruntime.CandidateMatchExecutionEnhanced)
	aggregated, err := recruitingruntime.NewCandidateMatchAggregator(recruitingruntime.CandidateMatchScorerVersion).Aggregate(
		extraction.Profile, results, source.Profile.Profile.TotalExperienceYears, extraction.InputHash, scorerType, fallbackUsed,
	)
	if err != nil {
		return RecruitingCandidateMatchDraft{}, err
	}
	if modelName == "" {
		modelName = recruitingruntime.CandidateMatchScorerVersion
	}
	draft := RecruitingCandidateMatchDraft{
		ApplicationID: source.Application.ApplicationID, JobID: source.Application.JobID,
		CandidateUserID: source.Application.CandidateUserID, ResumeProfileID: source.Profile.Profile.ID,
		OverallScore: aggregated.OverallScore, Recommendation: aggregated.Recommendation, Summary: aggregated.Summary,
		StrengthsJSON: marshalJSONString(aggregated.Strengths), RisksJSON: marshalJSONString(aggregated.Risks),
		ScoreBreakdownJSON: marshalJSONString(aggregated.Breakdown), ModelName: modelName,
		Evidence:      recruitingCandidateMatchEvidenceRows(extraction.Profile, results),
		ScorerVersion: recruitingruntime.CandidateMatchScorerVersion, RequirementCount: len(results), FallbackUsed: fallbackUsed,
	}
	if agentRunID > 0 {
		draft.AgentRunID = &agentRunID
	}
	return draft, nil
}

func recruitingCandidateEvidenceSource(source RecruitingMatchSource) recruitingruntime.CandidateEvidenceSource {
	out := recruitingruntime.CandidateEvidenceSource{
		Skills:      make([]recruitingruntime.CandidateEvidenceSkill, 0, len(source.Profile.Skills)),
		Experiences: make([]recruitingruntime.CandidateEvidenceExperience, 0, len(source.Profile.Experiences)),
		Projects:    make([]recruitingruntime.CandidateEvidenceProject, 0, len(source.Profile.Projects)),
		Educations:  make([]recruitingruntime.CandidateEvidenceEducation, 0, len(source.Profile.Educations)),
	}
	if source.Application.ResumeID > 0 {
		out.Resume = &recruitingruntime.CandidateEvidenceResume{ID: uint64(source.Application.ResumeID), ParsedText: source.ResumeParsedText}
	}
	if source.CandidateProfile != nil {
		profile := source.CandidateProfile
		out.CandidateProfile = &recruitingruntime.CandidateEvidenceProfile{
			ID: profile.ID, Education: profile.Education, School: profile.School,
			WorkExperience: profile.WorkExperience, Skills: profile.Skills,
		}
	}
	for _, row := range source.Profile.Skills {
		out.Skills = append(out.Skills, recruitingruntime.CandidateEvidenceSkill{ID: row.ID, Name: row.Name, Category: row.Category, Level: row.Level, Evidence: row.Evidence})
	}
	for _, row := range source.Profile.Experiences {
		out.Experiences = append(out.Experiences, recruitingruntime.CandidateEvidenceExperience{ID: row.ID, Company: row.Company, Title: row.Title, Description: row.Description, Achievements: recruitingJSONStringSlice(row.AchievementsJSON)})
	}
	for _, row := range source.Profile.Projects {
		out.Projects = append(out.Projects, recruitingruntime.CandidateEvidenceProject{ID: row.ID, Name: row.Name, Role: row.Role, Description: row.Description, Technologies: recruitingJSONStringSlice(row.TechnologiesJSON), Highlights: recruitingJSONStringSlice(row.HighlightsJSON)})
	}
	for _, row := range source.Profile.Educations {
		out.Educations = append(out.Educations, recruitingruntime.CandidateEvidenceEducation{ID: row.ID, School: row.School, Degree: row.Degree, Major: row.Major, Description: row.Description})
	}
	return out
}

func recruitingLegacyCandidateProfile(profile *RecruitingCandidateProfileRow) *recruitingruntime.LegacyCandidateProfile {
	if profile == nil {
		return nil
	}
	return &recruitingruntime.LegacyCandidateProfile{
		ID: profile.ID, RealName: profile.RealName, Phone: profile.Phone, Education: profile.Education,
		School: profile.School, WorkExperience: profile.WorkExperience, Skills: profile.Skills, IsComplete: profile.IsComplete,
	}
}

func recruitingJSONStringSlice(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return nil
	}
	return values
}

func recruitingCandidateMatchEvidenceRows(profile recruitingruntime.JobRequirementProfile, results []recruitingruntime.RequirementMatchResult) []RecruitingCandidateMatchEvidenceRow {
	weightByID := make(map[string]float64, len(profile.Requirements))
	for _, requirement := range profile.Requirements {
		weightByID[requirement.ID] = requirement.Weight
	}
	rows := make([]RecruitingCandidateMatchEvidenceRow, 0)
	seen := make(map[string]struct{})
	for _, result := range results {
		for _, reference := range result.Evidence {
			key := fmt.Sprintf("%s:%s:%d", result.RequirementID, reference.SourceTable, reference.SourceID)
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			sourceID := reference.SourceID
			rows = append(rows, RecruitingCandidateMatchEvidenceRow{
				EvidenceType: "requirement_match", Dimension: result.RequirementID,
				SourceTable: reference.SourceTable, SourceID: &sourceID,
				Snippet: truncateRecruitingSensitiveSnippet(reference.Snippet, recruitingruntime.MaxCandidateEvidenceRunes),
				Weight:  weightByID[result.RequirementID], ScoreImpact: result.Score,
				MetadataJSON: marshalJSONString(map[string]any{"requirement_id": result.RequirementID, "reason": reference.Reason, "evaluator_type": result.EvaluatorType}),
			})
		}
	}
	return rows
}

func truncateRecruitingSensitiveSnippet(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if limit <= 0 {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	return string(runes[:limit])
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

func (s nativeRecruitingIntelligenceService) resolveResumeProfileAccess(ctx context.Context, req *pb.GetResumeProfileRequest) (int64, *pb.GetResumeProfileResponse, *recruitingAuthError) {
	if req.GetApplicationId() > 0 {
		if _, authErr := s.authorizeRecruitingApplication(ctx, req.GetStaffUserId(), req.GetApplicationId()); authErr != nil {
			return 0, recruitingResumeAuthResponse(authErr), authErr
		}
		application, found, err := s.store.GetRecruitingApplicationByID(ctx, req.GetApplicationId())
		if err != nil {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
		}
		if !found || application.ResumeID <= 0 {
			return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "application resume not found"}, nil
		}
		if req.GetResumeId() > 0 && req.GetResumeId() != application.ResumeID {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "application_id and resume_id refer to different resumes"}, nil
		}
		if req.GetProfileId() > 0 {
			profile, profileFound, profileErr := s.store.GetRecruitingResumeProfileByID(ctx, req.GetProfileId())
			if profileErr != nil {
				return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: profileErr.Error()}, nil
			}
			if !profileFound {
				return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "resume profile not found"}, nil
			}
			if profile.ResumeID != application.ResumeID {
				return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "profile_id does not belong to requested resume/application"}, nil
			}
		}
		return application.ResumeID, nil, nil
	}

	resumeID := req.GetResumeId()
	if req.GetProfileId() > 0 {
		profile, found, err := s.store.GetRecruitingResumeProfileByID(ctx, req.GetProfileId())
		if err != nil {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
		}
		if !found {
			return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "resume profile not found"}, nil
		}
		if resumeID > 0 && resumeID != profile.ResumeID {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "profile_id does not belong to requested resume/application"}, nil
		}
		resumeID = profile.ResumeID
	}
	if resumeID <= 0 {
		return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "resume_id, profile_id, or application_id is required"}, nil
	}
	application, found, err := s.store.GetLatestRecruitingApplicationByResumeID(ctx, resumeID)
	if err != nil {
		return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
	}
	if !found {
		return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "application context not found for resume"}, nil
	}
	if _, authErr := s.authorizeRecruitingApplication(ctx, req.GetStaffUserId(), application.ApplicationID); authErr != nil {
		return 0, recruitingResumeAuthResponse(authErr), authErr
	}
	return resumeID, nil, nil
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
	store  AIStore
	runner mcpinfra.Runner
}
type nativeSkillService struct {
	pb.UnimplementedSkillServiceServer
	store AIStore
}
type nativeAgentSkillService struct {
	pb.UnimplementedAgentSkillServiceServer
	store     AIStore
	embedding *embeddinginfra.EmbeddingService
}
type nativeEmbeddingConfigService struct {
	pb.UnimplementedEmbeddingConfigServiceServer
	store     AIStore
	embedding *embeddinginfra.EmbeddingService
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
	return &pb.AgentRunItem{Id: row.ID, SessionId: row.SessionID, MessageId: row.MessageID, HistoryId: row.HistoryID, HrId: row.OwnerID, AgentType: row.AgentType, AgentId: row.AgentID, AgentName: row.AgentName, ModelId: row.ModelID, ModelName: row.ModelName, Status: row.Status, PlanJson: row.PlanJSON, FinalAnswer: row.AssistantText, ErrorType: row.ErrorType, ErrorMessage: row.ErrorMessage, StartedAt: formatTime(row.StartedAt), CompletedAt: formatTimePtr(row.CompletedAt), CreatedAt: formatTime(row.CreatedAt)}
}

func mapAgentRunStepItem(row AgentRunStepRow) *pb.AgentRunStepItem {
	return &pb.AgentRunStepItem{
		Id:               row.ID,
		RunId:            row.RunID,
		StepIndex:        row.StepIndex,
		StepType:         row.StepType,
		CapabilitySource: row.CapabilitySource,
		CapabilityKey:    row.CapabilityKey,
		ToolName:         row.ToolName,
		InputJson:        row.InputJSON,
		OutputJson:       row.OutputJSON,
		Status:           row.Status,
		DurationMs:       row.DurationMs,
		ErrorMessage:     row.ErrorMsg,
		StartedAt:        formatTime(row.StartedAt),
		CompletedAt:      formatTimePtr(row.CompletedAt),
		CreatedAt:        formatTime(row.CreatedAt),
	}
}

func mapAgentRunSnapshot(row AgentRunRow) *pb.AgentRunSnapshot {
	if row.ID == 0 {
		return nil
	}
	snapshot := &pb.AgentRunSnapshot{RunId: row.ID, SessionId: row.SessionID, HrId: row.OwnerID, ClientRequestId: row.ClientRequestID, MessageId: row.MessageID, HistoryId: row.HistoryID, Status: row.Status, AssistantText: row.AssistantText, ProcessText: row.ProcessText, OptionContextJson: row.OptionContextJSON, LastEventSeq: row.LastEventSeq, ErrorType: row.ErrorType, ErrorMessage: row.ErrorMessage, ModelId: row.ModelID, ModelName: row.ModelName, AgentType: row.AgentType, AgentId: row.AgentID, AgentName: row.AgentName, StartedAt: formatTime(row.StartedAt), CompletedAt: formatTimePtr(row.CompletedAt), CancelRequestedAt: formatTimePtr(row.CancelRequestedAt), CanceledAt: formatTimePtr(row.CanceledAt), CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
	applyAgentRunSnapshotPayload(snapshot, row.OptionContextJSON)
	applyAgentRunSnapshotPayload(snapshot, row.ProcessText)
	return snapshot
}

func fallbackAgentRun(hrID, sessionID int64, clientRequestID string, payload agentRunDurablePayload) AgentRunRow {
	now := time.Now()
	return AgentRunRow{SessionID: sessionID, OwnerID: hrID, ClientRequestID: clientRequestID, Status: "queued", AssistantText: "", PlanJSON: agentRunPlanJSON(payload), ModelID: payload.ModelID, AgentType: "hr", AgentName: "hr_recruiting_agent", StartedAt: now, CreatedAt: now, UpdatedAt: now}
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
