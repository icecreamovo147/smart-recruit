package grpc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	candidatetools "smart-recruit-ai-agent-service/internal/application/candidate_tools"
	"smart-recruit-ai-agent-service/internal/application/contextbudget"
	"smart-recruit-ai-agent-service/internal/application/hr_tools"
	appmemory "smart-recruit-ai-agent-service/internal/application/memory"
	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	domainagentskill "smart-recruit-ai-agent-service/internal/domain/agentskill"
	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
	"smart-recruit-ai-agent-service/internal/domain/model"
	mcpinfra "smart-recruit-ai-agent-service/internal/infrastructure/mcp"
	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
	aiagentruntime "smart-recruit-ai-agent-service/internal/runtime"
	commonsai "smart-recruit-commons/ai"
	"smart-recruit-platform-go/businessclock"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/logger"
	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-platform-go/observability"
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
	MaxIterations       int
	PrepareMessages     commonsai.MessagePrepareCallback
}

type RuntimeOptionsChatProvider interface {
	CompleteWithOptions(ctx context.Context, prompt string, modelID int64, opts ChatCompletionOptions) (string, error)
}

type UsageAwareRuntimeOptionsChatProvider interface {
	CompleteWithOptionsAndUsage(ctx context.Context, prompt string, modelID int64, opts ChatCompletionOptions) (commonsai.GenerateResult, error)
}

type RuntimeModelInfo struct {
	ID                     int64
	Name                   string
	ProviderName           string
	ContextWindowTokens    int32
	MaxOutputTokens        int32
	RequestedModelID       int64
	FallbackReason         string
	CapabilityVersionID    int64
	CapabilitySnapshotHash string
	ConfigurationRefs      CapabilityConfigurationRefs
	SkillRuntimePolicy     CapabilitySkillRuntimePolicy
}

// CapabilityConfigurationRefs is the immutable resource allowlist captured by
// a published platform capability release. Runtime code must not silently
// replace these IDs with whatever happens to be current in the control plane.
type CapabilityConfigurationRefs struct {
	AgentIDs             []int64
	PromptTemplateIDs    []int64
	AgentSkillVersionIDs []int64
	MCPPolicyIDs         []int64
}

type CapabilitySkillRuntimePolicy struct {
	PolicyVersion  string
	MaxSkillTokens int
	MaxInputRatio  float64
	MaxSkills      int
}

type CapabilityRuntimeModelResolution struct {
	EffectiveModelID       int64
	ModelName              string
	ProviderName           string
	RequestedModelID       int64
	FallbackReason         string
	CapabilityVersionID    int64
	CapabilitySnapshotHash string
	ContextWindowTokens    int32
	MaxOutputTokens        int32
	ConfigurationRefs      CapabilityConfigurationRefs
	SkillRuntimePolicy     CapabilitySkillRuntimePolicy
}

type capabilityRuntimeModelResolver interface {
	ResolveCapabilityRuntimeModel(context.Context, string, string, int64, int64) (CapabilityRuntimeModelResolution, error)
}

const (
	platformAIAudienceTenantHR  = "tenant_hr"
	platformAIAudienceCandidate = "candidate"
)

type llmRuntimeModelResolver interface {
	ResolveLLMRuntimeModel(ctx context.Context, modelID int64) (int64, string, string, bool, error)
}

type llmRuntimeModelInfoResolver interface {
	ResolveLLMRuntimeModelInfo(ctx context.Context, modelID int64) (RuntimeModelInfo, bool, error)
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

// RecruitingADKChatProvider runs HR chat through Eino ADK ChatModelAgent.
type RecruitingADKChatProvider interface {
	ChatWithRecruitingADK(
		ctx context.Context,
		modelID int64,
		opts ChatCompletionOptions,
		input commonsai.AgentRunInput,
		onDelta func(string) error,
		onToolExecuted commonsai.ToolTraceCallback,
		onStatus func(eventType, eventMessage, errorType, toolName string) error,
	) (string, commonsai.ToolMetadata, error)
}

type agentSkillDetailStore interface {
	GetAgentSkill(context.Context, *pb.GetAgentSkillRequest) (*pb.AgentSkillResponse, error)
	ListAgentSkillVersions(context.Context, *pb.ListAgentSkillVersionsRequest) (*pb.ListAgentSkillVersionsResponse, error)
}

type availableAgentSkillStore interface {
	ListAvailableAgentSkills(context.Context, int32, int32, string) ([]*pb.AgentSkillInfo, int64, error)
}

type hrCandidateMatchReadStore interface {
	GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(context.Context, int64) (RecruitingCandidateMatchSnapshot, bool, error)
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
	CompleteAgentRun(ctx context.Context, ownerID, runID int64, assistantText, status, errorType, errorMessage, resultMetadataJSON string) (AgentRunRow, bool, error)
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

type chatSessionContextModelStore interface {
	UpdateChatSessionContextModel(ctx context.Context, ownerRole int32, ownerID, sessionID, selectedModelID int64, usage *pb.ContextUsageInfo) error
}

type agentRunRuntimeGovernanceStore interface {
	UpdateAgentRunRuntimeGovernance(ctx context.Context, ownerID, runID int64, model RuntimeModelInfo) error
}

type ChatSessionListFilter struct {
	Keyword     string
	SessionType string
	SourceType  string
	SourceID    int64
}

type ChatSessionCreateOptions struct {
	SessionType string
	SourceType  string
	SourceID    int64
	SourceTitle string
}

type enhancedChatSessionStore interface {
	EnsureChatSessionWithOptions(ctx context.Context, ownerRole int32, ownerID int64, title string, applicationID int64, opts ChatSessionCreateOptions) (ChatSessionRow, error)
	ListChatSessionsWithFilter(ctx context.Context, ownerRole int32, ownerID int64, page, pageSize int32, filter ChatSessionListFilter) ([]ChatSessionRow, int64, error)
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

type usageAuditStore interface {
	RecordUsageAudit(ctx context.Context, row UsageAuditRow) (int64, error)
}

type applicationSnapshotClient interface {
	GetApplicationSnapshot(ctx context.Context, in *pb.GetApplicationSnapshotRequest, opts ...gogrpc.CallOption) (*pb.GetApplicationSnapshotResponse, error)
}

type jobServiceClient interface {
	ListHRJobs(ctx context.Context, in *pb.ListHRJobsRequest, opts ...gogrpc.CallOption) (*pb.ListJobsResponse, error)
	GetJobDetail(ctx context.Context, in *pb.GetJobDetailRequest, opts ...gogrpc.CallOption) (*pb.GetJobDetailResponse, error)
}

type applicationListServiceClient interface {
	ListJobApplications(ctx context.Context, in *pb.ListJobApplicationsRequest, opts ...gogrpc.CallOption) (*pb.ListJobApplicationsResponse, error)
}

type hrToolJobClientAdapter struct {
	client jobServiceClient
}

func (a hrToolJobClientAdapter) ListHRJobs(ctx context.Context, in *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	return a.client.ListHRJobs(ctx, in)
}

func (a hrToolJobClientAdapter) GetJobDetail(ctx context.Context, in *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	return a.client.GetJobDetail(ctx, in)
}

type hrToolApplicationListClientAdapter struct {
	client applicationListServiceClient
}

func (a hrToolApplicationListClientAdapter) ListJobApplications(ctx context.Context, in *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	return a.client.ListJobApplications(ctx, in)
}

type hrToolSnapshotClientAdapter struct {
	client applicationSnapshotClient
}

func (a hrToolSnapshotClientAdapter) GetApplicationSnapshot(ctx context.Context, in *pb.GetApplicationSnapshotRequest) (*pb.GetApplicationSnapshotResponse, error) {
	return a.client.GetApplicationSnapshot(ctx, in)
}

func adaptHRToolJobClient(client jobServiceClient) hr_tools.JobClient {
	if client == nil {
		return nil
	}
	return hrToolJobClientAdapter{client: client}
}

func adaptHRToolApplicationListClient(client applicationListServiceClient) hr_tools.ApplicationListClient {
	if client == nil {
		return nil
	}
	return hrToolApplicationListClientAdapter{client: client}
}

func adaptHRToolSnapshotClient(client applicationSnapshotClient) hr_tools.SnapshotClient {
	if client == nil {
		return nil
	}
	return hrToolSnapshotClientAdapter{client: client}
}

type RuntimeDeps struct {
	Store            AIStore
	Provider         ChatProvider
	RecruitingPolicy recruitingruntime.RuntimePolicy
	MemoryService    *appmemory.Service
	EmbeddingService *embeddinginfra.EmbeddingService
	EmbeddingWorker  bool
	AgentRunWorker   bool
	RuntimeName      string
	EmbeddingConfigs pb.EmbeddingConfigServiceServer
	PlatformAI       pb.PlatformAIControlPlaneServiceServer
	Auth             pb.AuthServiceClient
	Billing          pb.BillingServiceClient
	BillingRequired  bool
	AgentRunTimeout  time.Duration
	Applications     pb.ApplicationOwnerServiceClient
	AppList          pb.ApplicationServiceClient
	Jobs             pb.JobServiceClient
	MCPRunner        mcpinfra.Runner
	EmbeddingRunner  embeddinginfra.EmbeddingRunner
	SkillPackageV2   bool
	AgentSkillJudge  bool
	SkillJudgeRunner AgentSkillJudgeRunner
	Metrics          *observability.Registry
}

type ChatSessionRow struct {
	ID                 int64
	Title              string
	ApplicationID      int64
	LatestContextUsage *pb.ContextUsageInfo
	SelectedModelID    int64
	SessionType        string
	SourceType         string
	SourceID           int64
	SourceTitle        string
	Summary            string
	LastMessagePreview string
	MessageCount       int32
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ChatMessageRow struct {
	ID                   int64
	OwnerRole            int32
	OwnerID              int64
	SessionID            int64
	Role                 string
	Content              string
	ProcessContent       string
	ModelID              int64
	ModelName            string
	ContextUsage         *pb.ContextUsageInfo
	AgentSkillVersionIDs []int64
	AgentSkillNames      []string
	CreatedAt            time.Time
}

type ToolTraceRow struct {
	ID             int64
	SessionID      int64
	AgentRunID     int64
	AgentRunStepID int64
	ToolCallID     string
	ToolName       string
	ArgsJSON       string
	ResultContent  string
	Status         string
	DurationMs     int64
	ErrorMsg       string
	CreatedAt      time.Time
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

// UsageAuditRow is a third-party usage audit record for AI calls (HR or candidate).
type UsageAuditRow struct {
	UserID            int64
	Role              int32
	AccountType       string
	ServiceType       string
	Endpoint          string
	Provider          string
	Model             string
	RequestChars      int
	ResponseChars     int
	EstimatedTokens   int
	TokenUsageTotal   int
	PromptTokens      int
	CompletionTokens  int
	CachedInputTokens int
	Status            string
	ErrorCode         string
	CostMs            int
	RequestID         string
	IP                string
	RoleKeys          []string
	PermissionKey     string
	ScopeKeys         []string
	ResourceType      string
	ResourceID        int64
}

// CandidateUsageAuditRow keeps the candidate-facing audit shape used by existing call sites.
type CandidateUsageAuditRow struct {
	UserID            int64
	ServiceType       string
	Endpoint          string
	Provider          string
	Model             string
	RequestChars      int
	ResponseChars     int
	EstimatedTokens   int
	TokenUsageTotal   int
	PromptTokens      int
	CompletionTokens  int
	CachedInputTokens int
	Status            string
	ErrorCode         string
	CostMs            int
	RequestID         string
	IP                string
	RoleKeys          []string
	PermissionKey     string
	ScopeKeys         []string
}

func candidateUsageAuditToUsageAudit(row CandidateUsageAuditRow) UsageAuditRow {
	return UsageAuditRow{
		UserID:            row.UserID,
		Role:              1,
		AccountType:       "candidate",
		ServiceType:       row.ServiceType,
		Endpoint:          row.Endpoint,
		Provider:          row.Provider,
		Model:             row.Model,
		RequestChars:      row.RequestChars,
		ResponseChars:     row.ResponseChars,
		EstimatedTokens:   row.EstimatedTokens,
		TokenUsageTotal:   row.TokenUsageTotal,
		PromptTokens:      row.PromptTokens,
		CompletionTokens:  row.CompletionTokens,
		CachedInputTokens: row.CachedInputTokens,
		Status:            row.Status,
		ErrorCode:         row.ErrorCode,
		CostMs:            row.CostMs,
		RequestID:         row.RequestID,
		IP:                row.IP,
		RoleKeys:          append([]string(nil), row.RoleKeys...),
		PermissionKey:     row.PermissionKey,
		ScopeKeys:         append([]string(nil), row.ScopeKeys...),
		ResourceType:      "ai",
		ResourceID:        0,
	}
}

type AgentRunRow struct {
	ID                 int64
	TenantID           int64
	SessionID          int64
	MessageID          int64
	HistoryID          int64
	OwnerID            int64
	ClientRequestID    string
	Status             string
	AssistantText      string
	ProcessText        string
	PlanJSON           string
	ResultMetadataJSON string
	OptionContextJSON  string
	LastEventSeq       int64
	ErrorType          string
	ErrorMessage       string
	ModelID            int64
	ModelName          string
	AgentType          string
	AgentID            int64
	AgentName          string
	StartedAt          time.Time
	CompletedAt        *time.Time
	CancelRequestedAt  *time.Time
	CanceledAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type AgentRunEventRow struct {
	RunID       int64
	Seq         int64
	EventType   string
	PayloadJSON string
	CreatedAt   time.Time
}

type agentRunDurablePayload struct {
	Message                       string                     `json:"message,omitempty"`
	ActionType                    string                     `json:"action_type,omitempty"`
	ActionPayloadJSON             string                     `json:"action_payload_json,omitempty"`
	ApplicationID                 int64                      `json:"application_id,omitempty"`
	ModelID                       int64                      `json:"model_id,omitempty"`
	AuthUserID                    int64                      `json:"auth_user_id,omitempty"`
	AuthAccountType               string                     `json:"auth_account_type,omitempty"`
	AuthTenantID                  int64                      `json:"auth_tenant_id,omitempty"`
	AuthMembershipID              int64                      `json:"auth_membership_id,omitempty"`
	AuthClientApp                 string                     `json:"auth_client_app,omitempty"`
	EffectiveAgentID              int64                      `json:"effective_agent_id,omitempty"`
	EffectiveAgentPinned          bool                       `json:"effective_agent_pinned,omitempty"`
	CapabilityKeys                []string                   `json:"capability_keys,omitempty"`
	AgentSkillVersionIDs          []int64                    `json:"agent_skill_version_ids,omitempty"`
	ConfirmationPayloadJSON       string                     `json:"confirmation_payload_json,omitempty"`
	ConfirmationClientRequestID   string                     `json:"confirmation_client_request_id,omitempty"`
	PendingMCPConfirmation        *agentRunMCPConfirmation   `json:"pending_mcp_confirmation,omitempty"`
	MCPApproval                   *agentRunMCPApproval       `json:"mcp_approval,omitempty"`
	PendingAgentSkillConfirmation *agentRunSkillConfirmation `json:"pending_agent_skill_confirmation,omitempty"`
	AgentSkillApproval            *agentRunSkillApproval     `json:"agent_skill_approval,omitempty"`
}

type agentRunMCPConfirmation struct {
	ID            string `json:"id"`
	CapabilityKey string `json:"capability_key"`
	ArgumentsHash string `json:"arguments_hash"`
	Reason        string `json:"reason,omitempty"`
	ExpiresAt     string `json:"expires_at"`
}

type agentRunMCPApproval struct {
	ConfirmationID string `json:"confirmation_id"`
	CapabilityKey  string `json:"capability_key"`
	ArgumentsHash  string `json:"arguments_hash"`
	ApprovedAt     string `json:"approved_at"`
}

type mcpConfirmationPayload struct {
	Type           string `json:"type"`
	ConfirmationID string `json:"confirmation_id"`
	CapabilityKey  string `json:"capability_key"`
	ArgumentsHash  string `json:"arguments_hash"`
	Approved       bool   `json:"approved"`
}

type mcpConfirmationRequiredError struct {
	Confirmation agentRunMCPConfirmation
}

func (e *mcpConfirmationRequiredError) Error() string {
	return "mcp tool confirmation is required"
}

type agentSkillConfirmationRequiredError struct {
	Governance    hrRuntimeGovernanceContext
	RuntimeModel  RuntimeModelInfo
	UserMessageID int64
}

func (e *agentSkillConfirmationRequiredError) Error() string {
	return "agent skill confirmation is required"
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
	ID                     uint64
	ResumeID               int64
	UserID                 int64
	AgentRunID             *uint64
	RequestedModelID       int64
	EffectiveModelID       int64
	ModelFallbackReason    string
	CapabilityVersionID    int64
	CapabilitySnapshotHash string
	Status                 string
	ParserVersion          string
	InputHash              string
	ErrorMessage           string
	StartedAt              time.Time
	CompletedAt            *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
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
	ID                     uint64
	ApplicationID          int64
	JobID                  int64
	CandidateUserID        int64
	ResumeProfileID        uint64
	AgentRunID             *uint64
	RequestedModelID       int64
	EffectiveModelID       int64
	ModelFallbackReason    string
	CapabilityVersionID    int64
	CapabilitySnapshotHash string
	EvaluationVersion      int32
	IsLatest               int32
	OverallScore           float64
	Recommendation         string
	Summary                string
	StrengthsJSON          string
	RisksJSON              string
	ScoreBreakdownJSON     string
	ModelName              string
	EvaluatedAt            time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
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
	ResumeID               int64
	UserID                 int64
	ParserVersion          string
	InputHash              string
	RawJSON                string
	FullName               string
	Email                  string
	Phone                  string
	Location               string
	Headline               string
	Summary                string
	TotalExperienceYears   float64
	HighestDegree          string
	RequestedModelID       int64
	EffectiveModelID       int64
	ModelFallbackReason    string
	CapabilityVersionID    int64
	CapabilitySnapshotHash string
	Educations             []RecruitingResumeEducationRow
	Experiences            []RecruitingResumeExperienceRow
	Projects               []RecruitingResumeProjectRow
	Skills                 []RecruitingResumeSkillRow
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
	ApplicationID          int64
	JobID                  int64
	CandidateUserID        int64
	ResumeProfileID        uint64
	AgentRunID             *uint64
	RequestedModelID       int64
	EffectiveModelID       int64
	ModelFallbackReason    string
	CapabilityVersionID    int64
	CapabilitySnapshotHash string
	OverallScore           float64
	Recommendation         string
	Summary                string
	StrengthsJSON          string
	RisksJSON              string
	ScoreBreakdownJSON     string
	ModelName              string
	Evidence               []RecruitingCandidateMatchEvidenceRow
	ScorerVersion          string
	RequirementCount       int
	FallbackUsed           bool
}

var (
	errAIStoreRequired        = errors.New("ai store is required for native AI runtime")
	errAIProviderRequired     = errors.New("ai provider is required for native AI runtime")
	errChatSessionNotFound    = errors.New("chat session not found")
	errInvalidMCPConfirmation = errors.New("mcp confirmation payload is invalid")
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

	agentRunCancelTransitionMaxAttempts = 4
)

func NewNativeRuntimeDeps(deps RuntimeDeps) aiagentruntime.Deps {
	mcpRunner := deps.MCPRunner
	if mcpRunner == nil {
		mcpRunner = mcpinfra.NewRunner()
	}
	var embeddingService *embeddinginfra.EmbeddingService
	if deps.EmbeddingService != nil {
		embeddingService = deps.EmbeddingService
	} else if store, ok := deps.Store.(embeddinginfra.EmbeddingStore); ok {
		embeddingService = embeddinginfra.NewEmbeddingService(store, deps.EmbeddingRunner)
	}
	ai := newNativeAIServiceWithRunner(deps.Store, deps.Provider, deps.Applications, deps.Jobs, deps.AppList, mcpRunner, embeddingService)
	if deps.Metrics != nil {
		ai.metrics = deps.Metrics
	}
	ai.recruitingPolicy = deps.RecruitingPolicy
	ai.auth = deps.Auth
	ai.billing = deps.Billing
	ai.billingRequired = deps.BillingRequired
	ai.agentRunTimeout = deps.AgentRunTimeout
	ai.agentRuntime = normalizeAgentRuntime(deps.RuntimeName)
	ai.memoryService = deps.MemoryService
	ai.skillPackageV2Enabled = deps.SkillPackageV2
	ai.agentSkillJudge = newAgentSkillJudgeDispatcher(deps.AgentSkillJudge, deps.SkillJudgeRunner, ai.metrics)
	if store, ok := deps.Store.(candidatetools.DataStore); ok {
		ai.candidateTools = candidatetools.NewExecutor(store)
	}
	if seeder, ok := deps.Store.(interface {
		EnsureDefaultCandidateAssistant(context.Context) error
	}); ok {
		// Best-effort DEV-parity seed; failures must not block service startup.
		_ = seeder.EnsureDefaultCandidateAssistant(context.Background())
	}
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
		AgentSkill:             nativeAgentSkillService{store: deps.Store, embedding: embeddingService},
		RecruitingIntelligence: nativeRecruitingIntelligenceService{store: recruitingStore, provider: deps.Provider, structured: newRecruitingStructuredRuntime(deps.Store, deps.Provider, deps.RecruitingPolicy), policy: deps.RecruitingPolicy, auth: deps.Auth, applications: deps.Applications, jobs: deps.Jobs, meter: ai},
		EmbeddingConfig:        embedding,
		PlatformAIControlPlane: deps.PlatformAI,
		LongTasks: aiagentruntime.LongTaskControls{
			RabbitMQRequired: true,
			EmbeddingWorker:  deps.EmbeddingWorker,
			AgentRunWorker:   deps.AgentRunWorker,
			RuntimeName:      deps.RuntimeName,
		},
	}
}

func (s *nativeAIService) Close() error {
	if s != nil {
		s.agentSkillJudge.Close()
	}
	return nil
}

func NewNativeAIService(store AIStore, provider ChatProvider) pb.AIServiceServer {
	return newNativeAIService(store, provider, nil, nil, nil)
}

func newNativeAIService(store AIStore, provider ChatProvider, applications applicationSnapshotClient, jobs jobServiceClient, appList applicationListServiceClient) *nativeAIService {
	return newNativeAIServiceWithRunner(store, provider, applications, jobs, appList, nil, nil)
}

func newNativeAIServiceWithRunner(store AIStore, provider ChatProvider, applications applicationSnapshotClient, jobs jobServiceClient, appList applicationListServiceClient, mcpRunner mcpinfra.Runner, embedding *embeddinginfra.EmbeddingService) *nativeAIService {
	return &nativeAIService{
		store:        store,
		provider:     provider,
		applications: applications,
		jobs:         jobs,
		appList:      appList,
		mcpRunner:    mcpRunner,
		embedding:    embedding,
		agentRuntime: agentRuntimeADK,
		eventHub:     newAgentRunEventHub(),
		metrics:      observability.DefaultMetrics,
	}
}

const (
	agentRuntimeADK    = "adk"
	agentRuntimeLegacy = "legacy"
)

func normalizeAgentRuntime(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case agentRuntimeLegacy:
		return agentRuntimeLegacy
	default:
		return agentRuntimeADK
	}
}

type nativeAIService struct {
	pb.UnimplementedAIServiceServer
	store                   AIStore
	provider                ChatProvider
	recruitingPolicy        recruitingruntime.RuntimePolicy
	auth                    pb.AuthServiceClient
	billing                 pb.BillingServiceClient
	billingRequired         bool
	agentRunTimeout         time.Duration
	applications            applicationSnapshotClient
	jobs                    jobServiceClient
	appList                 applicationListServiceClient
	mcpRunner               mcpinfra.Runner
	embedding               *embeddinginfra.EmbeddingService
	agentRuntime            string
	candidateTools          commonsai.ToolRunner
	candidateToolsMu        sync.Mutex
	cachedCandidateADKTools []tool.BaseTool
	eventHubMu              sync.Mutex
	eventHub                *agentRunEventHub
	runCancelMu             sync.Mutex
	runCancels              map[int64]*agentRunCancelEntry
	runTransitionMu         sync.Mutex
	agentSkillMetricMu      sync.Mutex
	agentSkillMetricKeys    map[string]struct{}
	memoryService           *appmemory.Service
	skillPackageV2Enabled   bool
	metrics                 *observability.Registry
	agentSkillJudge         *agentSkillJudgeDispatcher
}

func (s *nativeAIService) effectiveAgentRuntime() string {
	if s == nil {
		return agentRuntimeADK
	}
	return normalizeAgentRuntime(s.agentRuntime)
}

func (s *nativeAIService) hrRuntimeLabel() string {
	return s.effectiveAgentRuntime()
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
		if code := hrContextErrorCode(err); code != "" {
			return &pb.ChatResponse{Code: configCodeUnavailable, Msg: hrContextMessageKey(code), CreatedAt: formatTime(time.Now()), SessionId: result.session.ID, ApplicationId: req.GetApplicationId(), ContextUsage: result.contextUsage}, nil
		}
		return nil, err
	}
	if result.providerUnavailable && !result.fallbackUsed {
		return &pb.ChatResponse{Code: configCodeUnavailable, Msg: "ai.provider_unavailable", CreatedAt: formatTime(time.Now()), SessionId: result.session.ID, ApplicationId: req.GetApplicationId(), ContextUsage: result.contextUsage, AgentSkillRuntimeEvidence: result.governance.AgentSkillRuntimeEvidence}, nil
	}
	return &pb.ChatResponse{Code: 0, Msg: "common.success", Reply: result.reply, CreatedAt: formatTime(time.Now()), SessionId: result.session.ID, ApplicationId: req.GetApplicationId(), CandidateName: result.candidateName, JobTitle: result.jobTitle, Status: result.status, ContextUsage: result.contextUsage, SuggestedQuestions: result.suggestedQuestions, AgentSkillRuntimeEvidence: result.governance.AgentSkillRuntimeEvidence}, nil
}

func (s *nativeAIService) ChatStream(req *pb.ChatRequest, stream gogrpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	if req == nil {
		return errors.New("chat request is required")
	}
	result, err := s.runHRChatRuntime(stream.Context(), req, func(event *pb.ChatStreamResponse, _ *agentRunDisplayContext) error {
		event.SessionId = firstNonZeroInt64(event.GetSessionId(), req.GetSessionId())
		event.ApplicationId = req.GetApplicationId()
		if event.CreatedAt == "" {
			event.CreatedAt = formatTime(time.Now())
		}
		return stream.Send(event)
	})
	if err != nil {
		if code := hrContextErrorCode(err); code != "" {
			messageKey := hrContextMessageKey(code)
			return stream.Send(&pb.ChatStreamResponse{Code: configCodeUnavailable, Msg: messageKey, Done: true, SessionId: result.session.ID, ApplicationId: req.GetApplicationId(), CreatedAt: formatTime(time.Now()), EventType: "error", EventMessage: messageKey, ErrorType: code, ContextUsage: result.contextUsage})
		}
		return err
	}
	if result.providerUnavailable && !result.fallbackUsed {
		return stream.Send(&pb.ChatStreamResponse{Code: configCodeUnavailable, Msg: "ai.provider_unavailable", Done: true, SessionId: result.session.ID, ApplicationId: req.GetApplicationId(), CreatedAt: formatTime(time.Now()), EventType: "error", EventMessage: "ai.provider_unavailable", ErrorType: "AI_PROVIDER_UNAVAILABLE", ContextUsage: result.contextUsage, AgentSkillRuntimeEvidence: result.governance.AgentSkillRuntimeEvidence})
	}
	return stream.Send(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", Delta: result.reply, Done: true, SessionId: result.session.ID, ApplicationId: req.GetApplicationId(), CandidateName: result.candidateName, JobTitle: result.jobTitle, Status: result.status, CreatedAt: formatTime(time.Now()), EventType: "done", EventMessage: "ai.event.completed", ContextUsage: result.contextUsage, SuggestedQuestions: result.suggestedQuestions, AgentSkillRuntimeEvidence: result.governance.AgentSkillRuntimeEvidence})
}

func hrContextMessageKey(code string) string {
	if code == hrContextBudgetExceededCode {
		return "ai.context_budget_exceeded"
	}
	return "ai.context_config_invalid"
}

type hrChatStreamEmitter func(*pb.ChatStreamResponse, *agentRunDisplayContext) error

type agentRunDisplayContext struct {
	Plan        commonsai.RecruitingPlan
	StepKey     string
	StepPurpose string
	ToolGroup   string
}

type hrChatRuntimeOptions struct {
	reuseExistingUserMessage bool
	existingUserMessageID    int64
	agentRunID               int64
	effectiveAgentID         int64
	effectiveAgentPinned     bool
	durablePayload           agentRunDurablePayload
}

type hrChatRuntimeResult struct {
	session             ChatSessionRow
	reply               string
	governance          hrRuntimeGovernanceContext
	plan                commonsai.RecruitingPlan
	modelID             int64
	modelName           string
	providerName        string
	runtimeModel        RuntimeModelInfo
	billingTokenUsage   *schema.TokenUsage
	toolTraces          []ToolTraceRow
	streamedTextDelta   bool
	contextUsage        *pb.ContextUsageInfo
	candidateName       string
	jobTitle            string
	status              int32
	providerUnavailable bool
	providerInvoked     bool
	fallbackUsed        bool
	runtimeWarnings     []string
	suggestedQuestions  []string
}

type hrRuntimeGovernanceContext struct {
	Agent                          *pb.AgentConfigInfo
	Prompt                         *pb.PromptTemplateInfo
	PromptContent                  string
	CapabilityKeys                 []string
	ToolNames                      []string
	ExecutableToolNames            []string
	SelectedAgentSkills            []hrRuntimeAgentSkill
	AgentSkillSelectionMode        string
	AgentSkillRuntimeEvidence      []*pb.AgentSkillRuntimeEvidence
	AgentSkillConfirmationRequired bool
	GovernanceErrors               []hrRuntimeGovernanceError
	ReleaseRefs                    CapabilityConfigurationRefs
	MemorySection                  string
	MemoryEvidence                 hrRuntimeMemoryEvidence
	MCPApproval                    *agentRunMCPApproval
}

type hrRuntimeMemoryEvidence struct {
	MemoryIDs      []uint64 `json:"memory_ids,omitempty"`
	Count          int      `json:"count,omitempty"`
	Chars          int      `json:"chars,omitempty"`
	RelevanceModes []string `json:"relevance_modes,omitempty"`
}

type hrRuntimeGovernanceError struct {
	Source     string `json:"source"`
	Code       string `json:"code"`
	ResourceID int64  `json:"resource_id,omitempty"`
}

type hrAllowlistedToolRunner struct {
	delegate commonsai.ToolRunner
	allowed  map[string]bool
	service  *nativeAIService
	runID    int64
}

func newHRAllowlistedToolRunner(delegate commonsai.ToolRunner, toolNames []string) hrAllowlistedToolRunner {
	allowed := make(map[string]bool, len(toolNames))
	for _, name := range toolNames {
		if normalized := hr_tools.NormalizeToolName(name); normalized != "" {
			allowed[normalized] = true
		}
	}
	return hrAllowlistedToolRunner{delegate: delegate, allowed: allowed}
}

func newHRRuntimeToolRunner(delegate commonsai.ToolRunner, toolNames []string, service *nativeAIService, runID int64) hrAllowlistedToolRunner {
	runner := newHRAllowlistedToolRunner(delegate, toolNames)
	runner.service = service
	runner.runID = runID
	return runner
}

func (r hrAllowlistedToolRunner) Execute(ctx context.Context, hrID int64, toolName string, args map[string]any) (commonsai.ToolResult, error) {
	normalized := hr_tools.NormalizeToolName(toolName)
	if !r.allowed[normalized] {
		err := &hr_tools.ToolExecutionError{Kind: "unauthorized", ToolName: normalized, Message: fmt.Sprintf("tool %s is not enabled for the active agent", normalized)}
		return commonsai.ToolResult{Content: marshalJSONString(map[string]any{"error": err.Error(), "error_type": err.Kind})}, err
	}
	if r.delegate == nil {
		err := &hr_tools.ToolExecutionError{Kind: "unavailable", ToolName: normalized, Message: "tool executor is unavailable"}
		return commonsai.ToolResult{Content: marshalJSONString(map[string]any{"error": err.Error(), "error_type": err.Kind})}, err
	}
	if r.service != nil {
		if err := r.service.verifyAgentRunSkillExecutionLease(ctx); err != nil {
			return agentRunSkillLeaseLostToolResult(normalized, err)
		}
	}
	var (
		result commonsai.ToolResult
		err    error
	)
	if isRecruitingIntelligenceTool(normalized) {
		result, err = r.service.executeRecruitingIntelligenceTool(ctx, hrID, normalized, args, r.runID)
	} else {
		result, err = r.delegate.Execute(ctx, hrID, normalized, args)
	}
	if r.service != nil {
		if leaseErr := r.service.verifyAgentRunSkillExecutionLease(ctx); leaseErr != nil {
			return agentRunSkillLeaseLostToolResult(normalized, leaseErr)
		}
	}
	return result, err
}

func agentRunSkillLeaseLostToolResult(toolName string, cause error) (commonsai.ToolResult, error) {
	err := fmt.Errorf("%w: tool %s was canceled before durable evidence could be committed", errAgentRunExecutionLeaseLost, toolName)
	if cause != nil && !errors.Is(cause, errAgentRunExecutionLeaseLost) {
		err = fmt.Errorf("%w: %v", errAgentRunExecutionLeaseLost, cause)
	}
	return commonsai.ToolResult{Content: marshalJSONString(map[string]any{
		"error":      err.Error(),
		"error_type": "execution_lease_lost",
		"retryable":  false,
	})}, err
}

type hrRuntimeAgentConfigStore interface {
	GetAgentConfig(context.Context, *pb.GetAgentConfigRequest) (*pb.GetAgentConfigResponse, error)
}

type hrRuntimePromptTemplateStore interface {
	GetRuntimePromptTemplateByID(ctx context.Context, id int64) (*pb.PromptTemplateInfo, bool, error)
}

type releasedMCPToolResolver interface {
	ResolveReleasedMCPToolKeys(context.Context, []int64) (map[string]bool, error)
}

func (s *nativeAIService) runHRChatRuntime(ctx context.Context, req *pb.ChatRequest, emit hrChatStreamEmitter) (hrChatRuntimeResult, error) {
	return s.runHRChatRuntimeWithOptions(ctx, req, emit, hrChatRuntimeOptions{})
}

func (s *nativeAIService) runHRChatRuntimeWithOptions(ctx context.Context, req *pb.ChatRequest, emit hrChatStreamEmitter, opts hrChatRuntimeOptions) (result hrChatRuntimeResult, err error) {
	ctx = withAgentSkillExecutionMode(ctx, opts.agentRunID > 0)
	ctx = withAgentSkillApproval(ctx, opts.agentRunID, req.GetHrId(), opts.durablePayload.AgentSkillApproval)
	session, err := s.ensureSession(ctx, ownerRoleHR, req.GetHrId(), req.GetSessionId(), req.GetApplicationId(), req.GetMessage())
	if err != nil {
		return hrChatRuntimeResult{}, err
	}
	result = hrChatRuntimeResult{session: session}
	if req.GetSessionId() == 0 {
		req.SessionId = session.ID
	}
	capability := "ai.chat"
	operation := "hr_chat"
	if opts.agentRunID > 0 {
		capability = "ai.agent_run"
		operation = "agent_run"
	}
	result.runtimeModel, err = s.resolveCapabilityRuntimeModel(ctx, billingOwnerTenant, req.GetHrId(), capability, platformAIAudienceTenantHR, req.GetModelId())
	if err != nil {
		return result, err
	}
	result.modelID = result.runtimeModel.ID
	result.modelName = result.runtimeModel.Name
	result.providerName = result.runtimeModel.ProviderName
	req.ModelId = result.modelID
	ctx, err = s.reserveAIBilling(ctx, billingOwnerTenant, req.GetHrId(), capability, operation, result.providerName, result.modelName, len([]rune(req.GetMessage())), result.runtimeModel)
	if err != nil {
		return result, err
	}
	defer s.cancelUnsettledBilling(ctx, "runtime_completed_without_usage")
	if sessionStore, ok := s.store.(chatSessionContextModelStore); ok {
		if err := sessionStore.UpdateChatSessionContextModel(ctx, ownerRoleHR, req.GetHrId(), session.ID, result.modelID, nil); err != nil {
			return result, err
		}
	}
	startedAt := time.Now()
	auditEnabled := false
	auditStatus := "ok"
	auditErrorCode := ""
	defer func() {
		if !auditEnabled {
			return
		}
		s.bestEffortRecordHRUsageAudit(ctx, req, opts, emit != nil, result, auditStatus, auditErrorCode, startedAt)
	}()
	sendWithDisplay := func(event *pb.ChatStreamResponse, display *agentRunDisplayContext) error {
		if emit == nil || event == nil {
			return nil
		}
		event.SessionId = session.ID
		event.ApplicationId = req.GetApplicationId()
		event.CandidateName = result.candidateName
		event.JobTitle = result.jobTitle
		event.Status = result.status
		return emit(event, display)
	}
	send := func(event *pb.ChatStreamResponse) error {
		return sendWithDisplay(event, nil)
	}
	emitWithoutDisplay := func(event *pb.ChatStreamResponse, _ *agentRunDisplayContext) error {
		return send(event)
	}
	emitWithDisplay := func(event *pb.ChatStreamResponse, display *agentRunDisplayContext) error {
		return sendWithDisplay(event, display)
	}
	modelInfoUsage := newHRContextUsageEnvelope(result.runtimeModel, 0)
	modelInfoUsage.Estimated = true
	modelInfoUsage.Source = "model_configuration"
	modelInfoUsage.Stage = "model_selected"
	if err := send(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "model_info", ContextUsage: modelInfoUsage, CreatedAt: formatTime(time.Now())}); err != nil {
		return result, err
	}
	if err := send(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "thinking", EventMessage: "ai.event.planning", CreatedAt: formatTime(time.Now())}); err != nil {
		return result, err
	}
	var governance hrRuntimeGovernanceContext
	if result.runtimeModel.CapabilityVersionID > 0 {
		governance, err = s.loadHRRuntimeGovernanceForAgentWithRelease(ctx, req, opts.effectiveAgentID, opts.effectiveAgentPinned, result.runtimeModel)
	} else {
		governance, err = s.loadHRRuntimeGovernanceForAgent(ctx, req, opts.effectiveAgentID, opts.effectiveAgentPinned)
	}
	if err != nil {
		return result, err
	}
	governance.MCPApproval = opts.durablePayload.MCPApproval
	result.governance = governance
	if governanceHasAgentSkillError(governance, "strict_output_contract_unsupported") {
		s.recordAgentSkillStrictUnsupported(ctx, governance)
		return result, statusErrorFailedPrecondition("AGENT_SKILL_STRICT_OUTPUT_UNSUPPORTED")
	}
	if governance.AgentSkillConfirmationRequired {
		userMessageID := int64(0)
		if opts.agentRunID > 0 {
			userMessage, appendErr := s.ensureAgentRunSkillConfirmationUserMessage(ctx, req, governance, opts.agentRunID)
			if appendErr != nil {
				return result, appendErr
			}
			userMessageID = userMessage.ID
		}
		selection := agentSkillSelectionPayload(governance, userMessageID)
		if emit != nil {
			if sendErr := send(&pb.ChatStreamResponse{
				Code:                      agentRunCodeBadRequest,
				Msg:                       "ai.agent_skill_confirmation_required",
				Done:                      true,
				EventType:                 "agent_skill_selection_required",
				EventMessage:              "ai.agent_skill_confirmation_required",
				ErrorType:                 "AGENT_SKILL_CONFIRMATION_REQUIRED",
				AgentSkillSelection:       selection,
				AgentSkillRuntimeEvidence: governance.AgentSkillRuntimeEvidence,
				CreatedAt:                 formatTime(time.Now()),
			}); sendErr != nil {
				return result, sendErr
			}
		}
		if opts.agentRunID <= 0 {
			return result, statusErrorFailedPrecondition("AGENT_SKILL_CONFIRMATION_REQUIRES_DURABLE_RUN")
		}
		return result, &agentSkillConfirmationRequiredError{
			Governance:    governance,
			RuntimeModel:  result.runtimeModel,
			UserMessageID: userMessageID,
		}
	}
	if result.runtimeModel.CapabilityVersionID > 0 && (governance.Agent == nil || governance.Prompt == nil || len(governance.GovernanceErrors) > 0) {
		return result, status.Error(codes.FailedPrecondition, "Agent or Prompt is unavailable in the capability release")
	}

	history, err := s.hrContextMessages(ctx, req.GetHrId(), session.ID)
	if err != nil {
		return result, err
	}
	var userMessage ChatMessageRow
	if opts.reuseExistingUserMessage {
		if opts.existingUserMessageID > 0 {
			userMessage = findHRUserMessageByID(history, opts.existingUserMessageID)
			if userMessage.ID == 0 {
				return result, errInvalidAgentSkillConfirmation
			}
			if approval := opts.durablePayload.AgentSkillApproval; approval != nil &&
				(approval.UserMessageID != userMessage.ID ||
					approval.MessageDigest != agentSkillMessageDigest(userMessage.Content)) {
				return result, errInvalidAgentSkillConfirmation
			}
		} else if strings.EqualFold(strings.TrimSpace(opts.durablePayload.ActionType), "analyze_application") {
			// Application analysis sessions predate durable Run message binding
			// and seed their user message before creating a Run. This compatibility
			// path is not used by either Skill or MCP confirmation resume.
			userMessage = findHRUserMessage(history, req.GetMessage())
		}
	}
	if strings.TrimSpace(req.GetMessage()) != "" && s.store != nil {
		if userMessage.ID == 0 {
			message := ChatMessageRow{OwnerRole: ownerRoleHR, OwnerID: req.GetHrId(), SessionID: session.ID, Role: "user", Content: req.GetMessage(), ModelID: req.GetModelId(), AgentSkillVersionIDs: hrRuntimeAgentSkillVersionIDs(governance), AgentSkillNames: hrRuntimeAgentSkillNames(governance)}
			if opts.agentRunID > 0 {
				binder, ok := s.store.(agentRunUserMessageStore)
				if !ok {
					return result, errInvalidAgentSkillConfirmation
				}
				userMessage, _, _, err = binder.EnsureAgentRunUserMessage(ctx, req.GetHrId(), opts.agentRunID, message)
			} else {
				userMessage, err = s.store.AppendChatMessage(ctx, message)
			}
			if err != nil {
				return result, err
			}
		}
	}

	// Application snapshot + MCP remain available as pre-context tools.
	traces, err := s.executeHRContextTools(ctx, req, session.ID, opts.agentRunID, governance, emitWithoutDisplay)
	if err != nil {
		return result, err
	}
	for _, trace := range traces {
		if trace.ToolName == "get_application_snapshot" {
			result.candidateName, result.jobTitle, result.status = extractApplicationTraceMetadata(trace.ResultContent)
		}
	}

	executor := &hr_tools.Executor{
		Jobs:         adaptHRToolJobClient(s.jobs),
		Applications: adaptHRToolApplicationListClient(s.appList),
		Snapshots:    adaptHRToolSnapshotClient(s.applications),
	}
	planTools := append([]string(nil), governance.ExecutableToolNames...)
	planTools = append(planTools, governance.ToolNames...)
	plan := commonsai.NewRecruitingPlanner().Plan(commonsai.RecruitingPlannerInput{
		Message:        req.GetMessage(),
		AvailableTools: planTools,
		ApplicationID:  req.GetApplicationId(),
		JobID:          jobIDFromMessage(req.GetMessage()),
	})
	result.plan = plan
	if err := s.persistAgentRunRuntimePlan(ctx, req, opts, governance, plan, result.modelID, result.modelName, nil); err != nil {
		return result, err
	}
	planned, planErr := s.preExecutePlannedHRTools(ctx, req, session.ID, opts.agentRunID, plan, traces, executor, emitWithDisplay)
	if planErr != nil {
		return result, planErr
	}
	traces = append(traces, planned...)

	// P0: greeting/unknown plans must never expose model tools, even if agent
	// bindings enable the full inventory.
	modelToolNames := append([]string(nil), governance.ExecutableToolNames...)
	if plan.DisallowsModelTools() {
		modelToolNames = nil
	}
	toolSchemas := hrRecruitingToolSchemas(modelToolNames)
	modelToolRunner := newHRRuntimeToolRunner(executor, modelToolNames, s, opts.agentRunID)
	toolProvider, hasToolProvider := s.provider.(RecruitingToolChatProvider)
	adkProvider, hasADKProvider := s.provider.(RecruitingADKChatProvider)
	canRunTools := len(modelToolNames) > 0 && (s.jobs != nil || s.appList != nil || s.applications != nil)
	useADK := s.effectiveAgentRuntime() == agentRuntimeADK

	// From here on, every exit should leave an AI usage audit trail (aligned with legacy writeHRUsageAudit).
	auditEnabled = true

	var reply string
	gateRequired := len(plan.RequiredToolGroups) > 0
	gateSatisfied := hrPlanEvidenceSatisfied(plan, traces)
	if gateRequired && hrPlanHasAuthoritativeEmptyCollection(plan, traces) {
		reply = hrDeterministicToolReply(traces)
	} else if gateRequired && !gateSatisfied {
		reply = hrPlanGateReply(plan, traces)
		if err := sendWithDisplay(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "fallback", EventMessage: "ai.event.evidence_gate_blocked", CreatedAt: formatTime(time.Now())}, displayContextForPlanStep(plan, "compose_answer")); err != nil {
			return result, err
		}
	} else if plan.Intent == commonsai.IntentStatusChangeProposal {
		reply = hrStatusChangeConfirmationReply(traces)
	} else if gateRequired && hrPlanUsesDeterministicFacts(plan) {
		reply = hrDeterministicToolReply(traces)
	} else if canRunTools && (hasADKProvider || hasToolProvider) {
		messages := buildHRToolCallingMessages(req, history, userMessage, traces, governance)
		if len(messages) > 0 {
			messages[0].Content += "\n\n" + plan.InstructionBlock()
		}
		summaryStore, summaryGenerator := s.hrSummaryDependencies()
		budgetController := newHRContextBudgetController(ctx, result.runtimeModel, toolSchemas, req.GetMessage(), traces, governance, history, req.GetHrId(), session.ID, summaryStore, summaryGenerator)
		messages, err = budgetController.prepare(ctx, messages, "initial_tool_call")
		result.contextUsage = budgetController.usage
		if err != nil {
			return result, err
		}
		if err := sendWithDisplay(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "context_usage", EventMessage: "ai.event.context_usage_estimated", ContextUsage: result.contextUsage, CreatedAt: formatTime(time.Now())}, displayContextForPlanStep(plan, "compose_answer")); err != nil {
			return result, err
		}
		onStatus := func(eventType, eventMessage, errorType, toolName string) error {
			event := &pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: eventType, EventMessage: eventMessage, ToolName: toolName, CreatedAt: formatTime(time.Now())}
			if eventType == "process_delta" && eventMessage != "" {
				event.Delta = eventMessage
			}
			if errorType != "" {
				event.ErrorType = errorType
				if eventType == "error" {
					event.Code = configCodeUnavailable
					event.Msg = "ai.stream_failed"
					event.EventMessage = "ai.stream_failed"
				}
			}
			if eventType == "timeout_warning" || strings.TrimSpace(errorType) != "" {
				warning := strings.TrimSpace(eventMessage)
				if warning == "" {
					warning = strings.TrimSpace(errorType)
				}
				if warning != "" {
					result.runtimeWarnings = append(result.runtimeWarnings, warning)
				}
			}
			return sendWithDisplay(event, displayContextForTool(plan, toolName))
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
		streamFilter := newHRSuggestionStreamFilter(func(delta string) error {
			deltaBuilder.WriteString(delta)
			if delta != "" {
				result.streamedTextDelta = true
			}
			return sendWithDisplay(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", Delta: delta, EventType: "generating", EventMessage: "ai.event.streaming_answer", CreatedAt: formatTime(time.Now())}, displayContextForPlanStep(plan, "compose_answer"))
		})
		onDelta := streamFilter.Write

		var toolReply string
		var toolMetadata commonsai.ToolMetadata
		var toolErr error
		ranADK := false
		if useADK && hasADKProvider {
			adkTools, adkToolsErr := commonsai.NewRecruitingADKTools(modelToolRunner, modelToolNames)
			if adkToolsErr != nil {
				result.runtimeWarnings = append(result.runtimeWarnings, "adk_tools_init_failed")
			} else {
				instruction := ""
				if len(messages) > 0 && messages[0] != nil && messages[0].Role == schema.System {
					instruction = messages[0].Content
				}
				state := &commonsai.AgentRunState{}
				adkCtx := commonsai.WithOwnerID(ctx, req.GetHrId())
				adkCtx = commonsai.WithAgentRunState(adkCtx, state)
				completionOptions := hrRuntimeCompletionOptions(governance)
				completionOptions.PrepareMessages = budgetController.prepare
				result.providerInvoked = true
				toolReply, toolMetadata, toolErr = adkProvider.ChatWithRecruitingADK(
					adkCtx,
					result.modelID,
					completionOptions,
					commonsai.AgentRunInput{
						AgentName:       "hr_recruiting_agent",
						Instruction:     instruction,
						Messages:        messages,
						Tools:           adkTools,
						MaxIterations:   hrRuntimeCompletionOptions(governance).MaxIterations,
						OwnerID:         req.GetHrId(),
						SessionID:       session.ID,
						State:           state,
						PrepareMessages: budgetController.prepare,
					},
					onDelta,
					onTool,
					onStatus,
				)
				ranADK = true
			}
		}
		if !ranADK {
			if !hasToolProvider {
				return result, fmt.Errorf("recruiting tool chat provider is not configured")
			}
			completionOptions := hrRuntimeCompletionOptions(governance)
			completionOptions.PrepareMessages = budgetController.prepare
			result.providerInvoked = true
			toolReply, toolMetadata, toolErr = toolProvider.ChatWithRecruitingTools(
				ctx,
				result.modelID,
				completionOptions,
				messages,
				toolSchemas,
				modelToolRunner,
				req.GetHrId(),
				onDelta,
				onTool,
				onStatus,
			)
		}
		if toolErr == nil {
			s.recordAgentSkillAdvisoryResponse(ctx, governance)
		}
		if finishErr := streamFilter.Finish(); finishErr != nil {
			return result, finishErr
		}
		result.billingTokenUsage = cloneTokenUsage(toolMetadata.BillingTokenUsage)
		if applyToolMetadataContextUsage(result.contextUsage, toolMetadata) {
			if err := sendWithDisplay(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "context_usage", EventMessage: "ai.event.provider_context_usage", ContextUsage: result.contextUsage, CreatedAt: formatTime(time.Now())}, displayContextForPlanStep(plan, "compose_answer")); err != nil {
				return result, err
			}
		}
		if toolErr != nil {
			if errors.Is(toolErr, errAIProviderRequired) || strings.Contains(toolErr.Error(), "api_key") || strings.Contains(toolErr.Error(), "provider") {
				result.providerUnavailable = true
			}
			if !hasUsefulToolResults(traces) {
				if agentRunExecutionCanceled(ctx, toolErr) {
					auditStatus = "timeout"
					result.reply = strings.TrimSpace(deltaBuilder.String())
				} else {
					auditStatus = "error"
					auditErrorCode = "provider_error"
				}
				if result.providerUnavailable {
					return result, nil
				}
				return result, toolErr
			}
			result.fallbackUsed = true
			auditErrorCode = "fallback"
			reply = commonsai.BuildHRFallbackReply(toCommonsToolTraces(traces))
			if err := sendWithDisplay(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "fallback", EventMessage: "ai.event.deterministic_fallback", CreatedAt: formatTime(time.Now())}, displayContextForPlanStep(plan, "compose_answer")); err != nil {
				return result, err
			}
		} else {
			reply = toolReply
			if strings.TrimSpace(reply) == "" {
				reply = deltaBuilder.String()
			}
		}
	} else {
		messages := buildHRToolCallingMessages(req, history, userMessage, traces, governance)
		if len(messages) > 0 {
			messages[0].Content += "\n\n" + plan.InstructionBlock()
		}
		summaryStore, summaryGenerator := s.hrSummaryDependencies()
		budgetController := newHRContextBudgetController(ctx, result.runtimeModel, nil, req.GetMessage(), traces, governance, history, req.GetHrId(), session.ID, summaryStore, summaryGenerator)
		messages, err = budgetController.prepare(ctx, messages, "completion_model_call")
		result.contextUsage = budgetController.usage
		if err != nil {
			return result, err
		}
		selectedHistory := selectedHRHistoryRows(messages, history, userMessage)
		summary := preparedHRSummary(messages)
		renderCompletion := func() string {
			prompt := renderHRProviderPrompt(req, selectedHistory, userMessage, traces, governance)
			if summary != "" {
				prompt = "Rolling conversation summary:\n" + summary + "\n\n" + prompt
			}
			return prompt + "\n\n" + plan.InstructionBlock()
		}
		contextPrompt := renderCompletion()
		extraOmitted := 0
		inputBudget := int64(budgetController.usage.GetInputBudgetTokens())
		if result.runtimeModel.ContextWindowTokens > 0 {
			for int64(estimateTokensConservative(contextPrompt)+6) > inputBudget && len(selectedHistory) > 0 {
				selectedHistory = selectedHistory[1:]
				extraOmitted++
				contextPrompt = renderCompletion()
			}
			if int64(estimateTokensConservative(contextPrompt)+6) > inputBudget && summary != "" {
				summary = ""
				contextPrompt = renderCompletion()
			}
			if int64(estimateTokensConservative(contextPrompt)+6) > inputBudget {
				budgetController.usage.BudgetStatus = "over_budget"
				return result, &hrContextGuardError{Code: hrContextBudgetExceededCode}
			}
		}
		assemblyUsage := result.contextUsage
		result.contextUsage = estimateHRCompletionContextUsage(result.runtimeModel, contextPrompt, req.GetMessage(), selectedHistory, userMessage, traces, governance)
		result.contextUsage.OmittedMessageCount = assemblyUsage.GetOmittedMessageCount() + int32(extraOmitted)
		result.contextUsage.SummaryApplied = summary != ""
		if err := sendWithDisplay(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "context_usage", EventMessage: "ai.event.context_usage_estimated", ContextUsage: result.contextUsage, CreatedAt: formatTime(time.Now())}, displayContextForPlanStep(plan, "compose_answer")); err != nil {
			return result, err
		}
		if err := sendWithDisplay(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "generating", EventMessage: "ai.event.calling_model", CreatedAt: formatTime(time.Now())}, displayContextForPlanStep(plan, "compose_answer")); err != nil {
			return result, err
		}
		var completionResult commonsai.GenerateResult
		result.providerInvoked = true
		completionResult, err = s.completeWithUsage(ctx, contextPrompt, result.modelID, hrRuntimeCompletionOptions(governance))
		reply = completionResult.Content
		result.billingTokenUsage = cloneTokenUsage(completionResult.TokenUsage)
		if err == nil {
			s.recordAgentSkillAdvisoryResponse(ctx, governance)
		}
		if applyActualContextUsage(result.contextUsage, completionResult.TokenUsage) {
			if emitErr := sendWithDisplay(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "context_usage", EventMessage: "ai.event.provider_context_usage", ContextUsage: result.contextUsage, CreatedAt: formatTime(time.Now())}, displayContextForPlanStep(plan, "compose_answer")); emitErr != nil {
				return result, emitErr
			}
		}
		if err != nil {
			if errors.Is(err, errAIProviderRequired) {
				result.providerUnavailable = true
			}
			if !hasUsefulToolResults(traces) {
				if agentRunExecutionCanceled(ctx, err) {
					auditStatus = "timeout"
				} else {
					auditStatus = "error"
					auditErrorCode = "provider_error"
				}
				if errors.Is(err, errAIProviderRequired) {
					return result, nil
				}
				return result, err
			}
			result.fallbackUsed = true
			auditErrorCode = "fallback"
			reply = commonsai.BuildHRFallbackReply(toCommonsToolTraces(traces))
			if err := sendWithDisplay(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "fallback", EventMessage: "ai.event.deterministic_fallback", CreatedAt: formatTime(time.Now())}, displayContextForPlanStep(plan, "compose_answer")); err != nil {
				return result, err
			}
		}
	}
	if opts.agentRunID > 0 && agentRunExecutionCanceled(ctx, nil) {
		return result, ctx.Err()
	}
	cleanReply, generatedQuestions := extractHRSuggestedQuestions(reply)
	reply = cleanReply
	if !plan.ConfirmationRequirement.Required {
		result.suggestedQuestions = normalizeHRSuggestedQuestions(generatedQuestions, plan.SuggestedQuestions, result.candidateName, result.jobTitle)
	}
	result.reply = reply
	s.trySubmitAgentSkillJudge(ctx, reply, result.candidateName, result.jobTitle, governance)
	result.toolTraces = append([]ToolTraceRow(nil), traces...)
	result.contextUsage = s.estimateHRPostTurnContextUsage(
		ctx,
		req,
		result.runtimeModel,
		history,
		userMessage,
		reply,
		traces,
		governance,
		plan,
		toolSchemas,
	)
	processContent := buildHRProcessContent(traces, result.contextUsage, result.fallbackUsed, governance, plan, s.hrRuntimeLabel(), result.suggestedQuestions)
	if s.store != nil {
		message := ChatMessageRow{OwnerRole: ownerRoleHR, OwnerID: req.GetHrId(), SessionID: session.ID, Role: "assistant", Content: reply, ProcessContent: processContent, ModelID: result.modelID, ModelName: result.modelName, ContextUsage: result.contextUsage, AgentSkillVersionIDs: hrRuntimeAgentSkillVersionIDs(governance), AgentSkillNames: hrRuntimeAgentSkillNames(governance), CreatedAt: time.Now()}
		if leaseID := agentRunSkillApprovalLeaseFromContext(ctx); leaseID != "" {
			store, ok := s.store.(agentRunSkillExecutionFenceStore)
			if !ok {
				return result, errAgentRunExecutionLeaseLost
			}
			if _, owned, appendErr := store.AppendAgentRunAssistantMessageForSkillLease(
				ctx,
				req.GetHrId(),
				opts.agentRunID,
				leaseID,
				message,
			); appendErr != nil {
				return result, appendErr
			} else if !owned {
				return result, errAgentRunExecutionLeaseLost
			}
		} else if hasAgentRunSkillApprovalContext(ctx) {
			return result, errAgentRunExecutionLeaseLost
		} else if _, err := s.store.AppendChatMessage(ctx, message); err != nil {
			return result, err
		}
	}
	s.asyncExtractHRMemory(ctx, req, session.ID, req.GetMessage(), reply, 0, 0)
	return result, nil
}

// estimateHRPostTurnContextUsage reports the effective context footprint of the
// current conversation after the assistant reply is added. It intentionally
// reuses the same budget controller as the next model call, so summaries,
// history trimming, tool schemas, Prompt/Agent instructions, selected Skills,
// and protocol framing follow one calculation path.
func (s *nativeAIService) estimateHRPostTurnContextUsage(
	ctx context.Context,
	req *pb.ChatRequest,
	model RuntimeModelInfo,
	history []ChatMessageRow,
	current ChatMessageRow,
	reply string,
	traces []ToolTraceRow,
	governance hrRuntimeGovernanceContext,
	plan commonsai.RecruitingPlan,
	toolSchemas []*schema.ToolInfo,
) *pb.ContextUsageInfo {
	postTurnHistory := ensureHRCurrentMessage(history, current)
	postTurnHistory = append([]ChatMessageRow(nil), postTurnHistory...)
	if content := strings.TrimSpace(reply); content != "" {
		postTurnHistory = append(postTurnHistory, ChatMessageRow{
			OwnerRole: ownerRoleHR,
			OwnerID:   req.GetHrId(),
			SessionID: req.GetSessionId(),
			Role:      "assistant",
			Content:   content,
			ModelID:   model.ID,
			ModelName: model.Name,
		})
	}

	previewReq := proto.Clone(req).(*pb.ChatRequest)
	previewReq.Message = ""
	messages := buildHRToolCallingMessages(previewReq, postTurnHistory, ChatMessageRow{}, traces, governance)
	if len(messages) > 0 {
		messages[0].Content += "\n\n" + plan.InstructionBlock()
	}

	var summaryStore hrSessionSummaryStore
	if candidate, ok := s.store.(hrSessionSummaryStore); ok {
		summaryStore = candidate
	}
	controller := newHRContextBudgetController(
		ctx,
		model,
		toolSchemas,
		"",
		traces,
		governance,
		postTurnHistory,
		req.GetHrId(),
		req.GetSessionId(),
		summaryStore,
		nil,
	)
	_, _ = controller.prepare(ctx, messages, "post_turn")
	usage := controller.usage
	if usage == nil {
		usage = estimateHRMessagesContextUsage(model, messages, toolSchemas, "", traces, governance)
	}
	usage.PromptTokensActual = 0
	usage.CompletionTokensActual = 0
	usage.TotalTokensActual = 0
	usage.Estimated = true
	usage.Source = "conservative_estimator"
	usage.Stage = "post_turn"
	return usage
}

func (s *nativeAIService) hrSummaryDependencies() (hrSessionSummaryStore, sessionSummaryGenerator) {
	var summaryStore hrSessionSummaryStore
	if candidate, ok := s.store.(hrSessionSummaryStore); ok {
		summaryStore = candidate
	}
	var generator sessionSummaryGenerator
	if candidate, ok := s.provider.(sessionSummaryGenerator); ok {
		generator = candidate
	} else if candidate, ok := s.store.(sessionSummaryGenerator); ok {
		generator = candidate
	}
	return summaryStore, generator
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

func findHRUserMessageByID(messages []ChatMessageRow, messageID int64) ChatMessageRow {
	if messageID <= 0 {
		return ChatMessageRow{}
	}
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message.ID == messageID && message.Role == "user" {
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
		// Proposal/action tools require a dedicated confirmation transport that
		// HR chat does not yet expose; never let the model auto-execute them.
		if name == "propose_application_status_update" {
			continue
		}
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

func (s *nativeAIService) preExecutePlannedHRTools(ctx context.Context, req *pb.ChatRequest, sessionID, agentRunID int64, plan commonsai.RecruitingPlan, existing []ToolTraceRow, executor *hr_tools.Executor, emit hrChatStreamEmitter) ([]ToolTraceRow, error) {
	if executor == nil || (s.jobs == nil && s.appList == nil && s.applications == nil) {
		return nil, nil
	}
	if len(plan.RequiredToolGroups) == 0 || len(plan.MissingInputs) > 0 {
		return nil, nil
	}
	traces := make([]ToolTraceRow, 0, len(plan.RequiredToolGroups))
	allTraces := append([]ToolTraceRow(nil), existing...)
	for _, group := range plan.RequiredToolGroups {
		if hrEvidenceGroupSatisfied(group, allTraces) {
			continue
		}
		for _, candidate := range group.Tools {
			name := hr_tools.NormalizeToolName(candidate)
			if (!hr_tools.ExecutableByThisRunner[name] && !isRecruitingIntelligenceTool(name)) || name == "propose_application_status_update" {
				continue
			}
			args, ok := plannedHRToolArgs(name, req, allTraces)
			if !ok {
				continue
			}
			if emit != nil {
				if err := emit(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "tool_calling", EventMessage: "ai.event.tool_calling", ToolName: name, CreatedAt: formatTime(time.Now())}, displayContextForTool(plan, name)); err != nil {
					return traces, err
				}
			}
			start := time.Now()
			result, execErr := s.executePlannedHRTool(ctx, req.GetHrId(), name, args, executor)
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
				event := &pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "tool_done", EventMessage: "ai.event.tool_finished", ToolName: name, CreatedAt: formatTime(time.Now())}
				if trace.ErrorMsg != "" {
					event.EventType = "error"
					event.EventMessage = trace.ErrorMsg
					event.ErrorType = "TOOL_ERROR"
					event.Msg = trace.ErrorMsg
				}
				if err := emit(event, displayContextForTool(plan, name)); err != nil {
					return traces, err
				}
			}
			traces = append(traces, trace)
			allTraces = append(allTraces, trace)
			if hrEvidenceGroupSatisfied(group, allTraces) {
				break
			}
		}
	}
	return traces, nil
}

func plannedHRToolArgs(name string, req *pb.ChatRequest, traces []ToolTraceRow) (map[string]any, bool) {
	args := map[string]any{}
	switch name {
	case "get_candidate_detail":
		applicationID := req.GetApplicationId()
		if applicationID <= 0 {
			applicationID = applicationIDFromHRToolTraces(traces)
		}
		if applicationID <= 0 {
			return nil, false
		}
		args["application_id"] = applicationID
	case "get_candidate_match_evaluation", "evaluate_candidate_match", "get_resume_profile", "parse_resume_profile":
		if req.GetApplicationId() <= 0 {
			return nil, false
		}
		args["application_id"] = req.GetApplicationId()
	case "get_job_detail", "list_applications_by_job":
		jobID := jobIDFromHRToolTraces(traces)
		if jobID <= 0 {
			jobID = jobIDFromMessage(req.GetMessage())
		}
		if jobID <= 0 {
			return nil, false
		}
		args["job_id"] = jobID
	case "list_applications_by_status":
		status, ok := applicationStatusFromMessage(req.GetMessage())
		if !ok {
			return nil, false
		}
		args["status"] = status
	case "search_candidates":
		keyword := candidateKeywordFromMessage(req.GetMessage())
		if keyword == "" {
			return nil, false
		}
		args["keyword"] = keyword
	case "search_jobs":
		args["keyword"] = ""
	}
	return args, true
}

func (s *nativeAIService) executePlannedHRTool(ctx context.Context, hrID int64, name string, args map[string]any, executor *hr_tools.Executor) (commonsai.ToolResult, error) {
	if err := s.verifyAgentRunSkillExecutionLease(ctx); err != nil {
		return agentRunSkillLeaseLostToolResult(name, err)
	}
	var (
		result commonsai.ToolResult
		err    error
	)
	if isRecruitingIntelligenceTool(name) {
		result, err = s.executeRecruitingIntelligenceTool(ctx, hrID, name, args, 0)
	} else {
		result, err = executor.Execute(ctx, hrID, name, args)
	}
	if leaseErr := s.verifyAgentRunSkillExecutionLease(ctx); leaseErr != nil {
		return agentRunSkillLeaseLostToolResult(name, leaseErr)
	}
	return result, err
}

func isRecruitingIntelligenceTool(name string) bool {
	switch hr_tools.NormalizeToolName(name) {
	case "parse_resume_profile", "get_resume_profile", "evaluate_candidate_match", "get_candidate_match_evaluation", "compare_candidates_for_job":
		return true
	default:
		return false
	}
}

func (s *nativeAIService) executeRecruitingIntelligenceTool(ctx context.Context, hrID int64, name string, args map[string]any, agentRunID int64) (commonsai.ToolResult, error) {
	if s == nil {
		return hrToolErrorResult(name, "unavailable", "native HR runtime is not configured", nil)
	}
	store, ok := s.store.(recruitingReadStore)
	if (!ok || store == nil) && hr_tools.NormalizeToolName(name) == "get_candidate_match_evaluation" {
		return s.executeLegacyCandidateMatchReadTool(ctx, name, args)
	}
	if !ok || store == nil {
		return hrToolErrorResult(name, "unavailable", "recruiting intelligence store is not configured", nil)
	}
	service := nativeRecruitingIntelligenceService{
		store:        store,
		provider:     s.provider,
		structured:   newRecruitingStructuredRuntime(s.store, s.provider, s.recruitingPolicy),
		policy:       s.recruitingPolicy,
		auth:         s.auth,
		applications: s.applications,
		jobs:         s.jobs,
		meter:        s.recruitingIntelligenceMeter(),
	}
	toolName := hr_tools.NormalizeToolName(name)
	switch toolName {
	case "get_resume_profile":
		applicationID, resumeID := recruitingApplicationAndResumeArgs(args)
		resp, err := service.GetResumeProfile(ctx, &pb.GetResumeProfileRequest{StaffUserId: hrID, ApplicationId: applicationID, ResumeId: resumeID, ProfileId: uint64(numberFromAny(args["profile_id"]))})
		return recruitingToolResponse(toolName, resp, err)
	case "parse_resume_profile":
		applicationID, resumeID := recruitingApplicationAndResumeArgs(args)
		resp, err := service.ParseResumeProfile(ctx, &pb.ParseResumeProfileRequest{StaffUserId: hrID, ApplicationId: applicationID, ResumeId: resumeID})
		return recruitingToolResponse(toolName, resp, err)
	case "get_candidate_match_evaluation":
		applicationID := int64(numberFromAny(args["application_id"]))
		resp, err := service.GetCandidateMatchEvaluation(ctx, &pb.GetCandidateMatchEvaluationRequest{
			StaffUserId:       hrID,
			ApplicationId:     applicationID,
			EvaluationId:      uint64(numberFromAny(args["evaluation_id"])),
			EvaluationVersion: int32(numberFromAny(args["evaluation_version"])),
		})
		return recruitingToolResponse(toolName, resp, err)
	case "evaluate_candidate_match":
		applicationID := int64(numberFromAny(args["application_id"]))
		resp, err := service.EvaluateCandidateMatch(ctx, &pb.EvaluateCandidateMatchRequest{StaffUserId: hrID, ApplicationId: applicationID, AgentRunId: uint64(agentRunID)})
		return recruitingToolResponse(toolName, resp, err)
	case "compare_candidates_for_job":
		jobID := int64(numberFromAny(args["job_id"]))
		resp, err := service.CompareCandidatesForJob(ctx, &pb.CompareCandidatesForJobRequest{StaffUserId: hrID, JobId: jobID})
		return recruitingToolResponse(toolName, resp, err)
	default:
		return hrToolErrorResult(toolName, "unsupported", fmt.Sprintf("unsupported tool: %s", name), nil)
	}
}

func (s *nativeAIService) recruitingIntelligenceMeter() *nativeAIService {
	if s == nil || s.store == nil {
		return nil
	}
	// Production NativeStore implements the release resolver. Keeping the
	// meter absent for isolated legacy stores prevents their current/request
	// model compatibility path from masquerading as an exact v2 release.
	if _, ok := s.store.(capabilityRuntimeModelResolver); !ok {
		return nil
	}
	return s
}

func (s *nativeAIService) executeLegacyCandidateMatchReadTool(ctx context.Context, name string, args map[string]any) (commonsai.ToolResult, error) {
	applicationID := int64(numberFromAny(args["application_id"]))
	store, ok := s.store.(hrCandidateMatchReadStore)
	if !ok {
		return hrToolErrorResult(name, "unavailable", "candidate match evaluation store is not configured", nil)
	}
	snapshot, found, err := store.GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(ctx, applicationID)
	if err != nil {
		return hrToolErrorResult(name, "downstream", "candidate match evaluation query failed", err)
	}
	if !found {
		return hrToolErrorResult(name, "not_found", "candidate match evaluation not found", nil)
	}
	if snapshot.Evaluation.ApplicationID != applicationID {
		return hrToolErrorResult(name, "forbidden_or_not_found", "candidate match evaluation scope mismatch", nil)
	}
	return commonsai.ToolResult{Content: marshalJSONString(candidateMatchSnapshotToolPayload(snapshot))}, nil
}

func candidateMatchSnapshotToolPayload(snapshot RecruitingCandidateMatchSnapshot) map[string]any {
	return map[string]any{
		"application_id": snapshot.Evaluation.ApplicationID,
		"job_id":         snapshot.Evaluation.JobID,
		"overall_score":  snapshot.Evaluation.OverallScore,
		"recommendation": snapshot.Evaluation.Recommendation,
		"summary":        snapshot.Evaluation.Summary,
		"strengths_json": snapshot.Evaluation.StrengthsJSON,
		"risks_json":     snapshot.Evaluation.RisksJSON,
		"evaluation_id":  snapshot.Evaluation.ID,
		"evidence_count": len(snapshot.Evidence),
		"evaluated_at":   formatTime(snapshot.Evaluation.EvaluatedAt),
	}
}

func recruitingApplicationAndResumeArgs(args map[string]any) (int64, int64) {
	if args == nil {
		return 0, 0
	}
	return int64(numberFromAny(args["application_id"])), int64(numberFromAny(args["resume_id"]))
}

type recruitingToolPBResponse interface {
	proto.Message
	GetCode() int32
	GetMsg() string
}

func recruitingToolResponse(toolName string, resp recruitingToolPBResponse, err error) (commonsai.ToolResult, error) {
	if err != nil {
		return hrToolErrorResult(toolName, "downstream", err.Error(), err)
	}
	if resp == nil {
		return hrToolErrorResult(toolName, "unavailable", "recruiting intelligence tool returned nil response", nil)
	}
	content := protoJSONOrFallback(resp)
	if resp.GetCode() == errs.OK {
		return commonsai.ToolResult{Content: content}, nil
	}
	kind := recruitingToolErrorKind(resp.GetCode(), resp.GetMsg())
	execErr := &hr_tools.ToolExecutionError{Kind: kind, ToolName: toolName, Message: recruitingMessageOrDefault(resp.GetMsg(), "recruiting intelligence tool failed")}
	return commonsai.ToolResult{Content: marshalJSONString(map[string]any{"error": execErr.Error(), "error_type": execErr.Kind, "tool_response": jsonRawObject(content)})}, execErr
}

func protoJSONOrFallback(message proto.Message) string {
	if message == nil {
		return "{}"
	}
	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(message)
	if err != nil {
		return marshalJSONString(map[string]any{"error": err.Error()})
	}
	return string(data)
}

func jsonRawObject(content string) any {
	var payload any
	if json.Unmarshal([]byte(content), &payload) == nil {
		return payload
	}
	return content
}

func recruitingToolErrorKind(code int32, message string) string {
	switch code {
	case 404:
		return "not_found"
	case errs.ErrForbidden:
		return "forbidden_or_not_found"
	case errs.ErrBadRequest:
		return "invalid_argument"
	case configCodeUnsupported:
		if strings.Contains(strings.ToLower(message), "not found") {
			return "not_found"
		}
		return "unsupported"
	case configCodeUnavailable:
		return "unavailable"
	default:
		return "downstream"
	}
}

func hrToolErrorResult(toolName, kind, message string, cause error) (commonsai.ToolResult, error) {
	execErr := &hr_tools.ToolExecutionError{Kind: kind, ToolName: hr_tools.NormalizeToolName(toolName), Message: message, Cause: cause}
	return commonsai.ToolResult{Content: marshalJSONString(map[string]any{"error": execErr.Error(), "error_type": execErr.Kind})}, execErr
}

func candidateKeywordFromMessage(message string) string {
	keyword := strings.TrimSpace(message)
	replacer := strings.NewReplacer(
		"候选人", " ", "应聘者", " ", "搜索", " ", "查找", " ", "查询", " ",
		"列表", " ", "详情", " ", "是谁", " ", "这个", " ", "该", " ", "的", " ", "search", " ", "find", " ",
		"candidate", " ", "list", " ", "detail", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(keyword)), " ")
}

func applicationIDFromHRToolTraces(traces []ToolTraceRow) int64 {
	for index := len(traces) - 1; index >= 0; index-- {
		if traces[index].Status != "success" {
			continue
		}
		var payload struct {
			ApplicationID int64 `json:"application_id"`
			Candidates    []struct {
				ApplicationID int64 `json:"application_id"`
			} `json:"candidates"`
		}
		if json.Unmarshal([]byte(traces[index].ResultContent), &payload) != nil {
			continue
		}
		if payload.ApplicationID > 0 {
			return payload.ApplicationID
		}
		if len(payload.Candidates) == 1 && payload.Candidates[0].ApplicationID > 0 {
			return payload.Candidates[0].ApplicationID
		}
	}
	return 0
}

func jobIDFromMessage(message string) int64 {
	for _, token := range strings.FieldsFunc(message, func(r rune) bool { return r < '0' || r > '9' }) {
		if value, err := strconv.ParseInt(token, 10, 64); err == nil && value > 0 {
			return value
		}
	}
	return 0
}

func applicationStatusFromMessage(message string) (int32, bool) {
	message = strings.ToLower(message)
	switch {
	case containsAnyString(message, "待查看", "未查看", "pending"):
		return 0, true
	case containsAnyString(message, "已查看", "viewed"):
		return 1, true
	case containsAnyString(message, "已通过", "通过的投递", "approved"):
		return 2, true
	case containsAnyString(message, "已淘汰", "淘汰的投递", "rejected"):
		return 3, true
	default:
		return 0, false
	}
}

func containsAnyString(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func jobIDFromHRToolTraces(traces []ToolTraceRow) int64 {
	for index := len(traces) - 1; index >= 0; index-- {
		if strings.TrimSpace(traces[index].ResultContent) == "" || traces[index].Status != "success" {
			continue
		}
		var payload map[string]any
		if json.Unmarshal([]byte(traces[index].ResultContent), &payload) == nil {
			if jobID := int64(numberFromAny(payload["job_id"])); jobID > 0 {
				return jobID
			}
		}
	}
	return 0
}

func hrEvidenceGroupSatisfied(group commonsai.RecruitingToolGroup, traces []ToolTraceRow) bool {
	allowed := normalizedStringSet(group.Tools)
	for _, trace := range traces {
		if allowed[hr_tools.NormalizeToolName(trace.ToolName)] && trace.Status == "success" && toolResultContentUseful(trace.ResultContent) {
			return true
		}
	}
	return false
}

func hrPlanEvidenceSatisfied(plan commonsai.RecruitingPlan, traces []ToolTraceRow) bool {
	if len(plan.RequiredToolGroups) == 0 || len(plan.MissingInputs) > 0 {
		return len(plan.RequiredToolGroups) == 0 && len(plan.MissingInputs) == 0
	}
	for _, group := range plan.RequiredToolGroups {
		if len(group.Tools) == 0 || !hrEvidenceGroupSatisfied(group, traces) {
			return false
		}
	}
	return true
}

func hrPlanHasAuthoritativeEmptyCollection(plan commonsai.RecruitingPlan, traces []ToolTraceRow) bool {
	required := normalizedStringSet(plan.RequiredTools)
	for _, trace := range traces {
		if trace.Status != "success" || !required[hr_tools.NormalizeToolName(trace.ToolName)] {
			continue
		}
		var payload map[string]any
		if json.Unmarshal([]byte(trace.ResultContent), &payload) != nil {
			continue
		}
		key := ""
		switch hr_tools.NormalizeToolName(trace.ToolName) {
		case "get_job_list", "search_jobs":
			key = "jobs"
		case "list_all_applications", "list_applications_by_job", "list_applications_by_status":
			key = "applications"
		case "search_candidates":
			key = "candidates"
		}
		if key == "" {
			continue
		}
		if values, ok := payload[key].([]any); ok && len(values) == 0 {
			return true
		}
	}
	return false
}

func hrPlanGateReply(plan commonsai.RecruitingPlan, traces []ToolTraceRow) string {
	if len(plan.MissingInputs) > 0 {
		return "需要先选择或提供具体的岗位、候选人或投递记录，才能查询对应的实时数据。"
	}
	for _, group := range plan.RequiredToolGroups {
		if len(group.Tools) == 0 {
			return "当前 Agent 未启用或未绑定完成该问题所需的能力，请在 Agent 管理中启用对应工具后再试。"
		}
	}
	for _, trace := range traces {
		if trace.Status == "error" && stringSliceContains(plan.RequiredTools, trace.ToolName) {
			if hrToolTraceHasErrorType(trace, "not_found") {
				return "实时数据查询失败：当前投递暂无可用的匹配评估记录；如需基于真实简历和岗位数据生成结论，请启用并运行候选人匹配评估生成工具。"
			}
			if hrToolTraceHasErrorType(trace, "unsupported", "unavailable") {
				return "完成该实时数据查询所需的工具已绑定，但当前运行时未接入或暂不可用，无法提供可靠结果。"
			}
			return "实时数据查询失败，无法可靠回答该问题。请稍后重试。"
		}
	}
	return "当前 Agent 未启用完成该实时数据查询所需的工具，或对应工具在运行时不可用，无法提供可靠结果。"
}

func hrToolTraceHasErrorType(trace ToolTraceRow, types ...string) bool {
	if strings.TrimSpace(trace.ResultContent) == "" || len(types) == 0 {
		return false
	}
	var payload map[string]any
	if json.Unmarshal([]byte(trace.ResultContent), &payload) != nil {
		return false
	}
	errorType, _ := payload["error_type"].(string)
	if errorType == "" {
		if nested, ok := payload["tool_response"].(map[string]any); ok {
			if code := int32(numberFromAny(nested["code"])); code != 0 {
				errorType = recruitingToolErrorKind(code, stringFromAny(nested["msg"]))
			}
		}
	}
	for _, typ := range types {
		if errorType == typ {
			return true
		}
	}
	return false
}

func hrPlanUsesDeterministicFacts(plan commonsai.RecruitingPlan) bool {
	switch plan.Intent {
	case commonsai.IntentJobListing, commonsai.IntentJobDetail, commonsai.IntentApplicationListing, commonsai.IntentCandidateLookup, commonsai.IntentAnalytics:
		return true
	default:
		return false
	}
}

func hrDeterministicToolReply(traces []ToolTraceRow) string {
	reply := commonsai.BuildHRFallbackReply(toCommonsToolTraces(traces))
	reply = strings.Replace(reply, "AI 模型回答失败，以下基于已查询到的数据给出保守回复：", "以下结果来自系统实时查询：", 1)
	reply = strings.Replace(reply, "以上为系统已查询的数据。如需更详细分析，请稍后重试。", "以上为本次系统实时查询结果。", 1)
	return reply
}

func hrStatusChangeConfirmationReply(traces []ToolTraceRow) string {
	var candidateName, jobTitle string
	for _, trace := range traces {
		if trace.Status != "success" {
			continue
		}
		var payload map[string]any
		if json.Unmarshal([]byte(trace.ResultContent), &payload) != nil {
			continue
		}
		if candidateName == "" {
			candidateName, _ = payload["candidate_name"].(string)
		}
		if jobTitle == "" {
			jobTitle, _ = payload["job_title"].(string)
		}
	}
	contextLabel := "目标投递记录"
	if candidateName != "" && jobTitle != "" {
		contextLabel = fmt.Sprintf("候选人 %s 在岗位 %s 的投递记录", candidateName, jobTitle)
	} else if candidateName != "" {
		contextLabel = "候选人 " + candidateName + " 的投递记录"
	}
	return fmt.Sprintf("已核对%s。状态变更尚未执行；请明确确认目标状态后，再通过受控的确认流程提交。", contextLabel)
}

func (s *nativeAIService) hrRecentMessages(ctx context.Context, hrID, sessionID int64) ([]ChatMessageRow, error) {
	if s == nil || s.store == nil || sessionID == 0 {
		return nil, nil
	}
	if recent, ok := s.store.(recentChatMessageStore); ok {
		return recent.ListRecentChatMessages(ctx, ownerRoleHR, hrID, sessionID, 20)
	}
	return s.store.ListChatMessages(ctx, ownerRoleHR, hrID, sessionID, 1, 20)
}

// hrContextMessages loads a bounded but substantially larger tail for budget
// selection and rolling-summary coverage. Unknown-window degradation is still
// capped to 20 by the assembler.
func (s *nativeAIService) hrContextMessages(ctx context.Context, hrID, sessionID int64) ([]ChatMessageRow, error) {
	if s == nil || s.store == nil || sessionID == 0 {
		return nil, nil
	}
	if recent, ok := s.store.(recentChatMessageStore); ok {
		return recent.ListRecentChatMessages(ctx, ownerRoleHR, hrID, sessionID, 200)
	}
	return s.store.ListChatMessages(ctx, ownerRoleHR, hrID, sessionID, 1, 200)
}

func (s *nativeAIService) loadHRRuntimeGovernance(ctx context.Context, req *pb.ChatRequest) (hrRuntimeGovernanceContext, error) {
	return s.loadHRRuntimeGovernanceForAgent(ctx, req, 0, false)
}

type hrRuntimeAgentByIDStore interface {
	GetRuntimeAgentConfigByID(context.Context, int64) (*pb.AgentConfigInfo, bool, error)
}

func (s *nativeAIService) loadHRRuntimeGovernanceForAgent(ctx context.Context, req *pb.ChatRequest, effectiveAgentID int64, effectiveAgentPinned bool) (hrRuntimeGovernanceContext, error) {
	return s.loadHRRuntimeGovernanceForAgentCore(ctx, req, effectiveAgentID, effectiveAgentPinned, RuntimeModelInfo{}, false)
}

func (s *nativeAIService) loadHRRuntimeGovernanceForAgentWithRelease(ctx context.Context, req *pb.ChatRequest, effectiveAgentID int64, effectiveAgentPinned bool, model RuntimeModelInfo) (hrRuntimeGovernanceContext, error) {
	return s.loadHRRuntimeGovernanceForAgentCore(ctx, req, effectiveAgentID, effectiveAgentPinned, model, true)
}

func (s *nativeAIService) loadHRRuntimeGovernanceForAgentCore(ctx context.Context, req *pb.ChatRequest, effectiveAgentID int64, effectiveAgentPinned bool, model RuntimeModelInfo, enforceRelease bool) (hrRuntimeGovernanceContext, error) {
	var runtime hrRuntimeGovernanceContext
	refs := model.ConfigurationRefs
	runtime.ReleaseRefs = refs
	memoryRecall := s.recallHRMemory(ctx, req, 0, 0)
	runtime.MemorySection = memoryRecall.InjectText
	runtime.MemoryEvidence = memoryRecall.Evidence
	if s == nil || s.store == nil {
		return runtime, nil
	}
	var agent *pb.AgentConfigInfo
	var err error
	if !effectiveAgentPinned || effectiveAgentID > 0 {
		agent, err = s.loadHRAgentConfigForRelease(ctx, effectiveAgentID, refs.AgentIDs)
	}
	if err != nil {
		return runtime, err
	}
	runtime.Agent = agent
	runtime.CapabilityKeys = hrRuntimeCapabilityKeys(agent)
	runtime.ToolNames = hrRuntimeToolNames(agent)
	if enforceRelease {
		allowed := make(map[string]bool)
		if len(refs.MCPPolicyIDs) > 0 {
			resolver, ok := s.store.(releasedMCPToolResolver)
			if !ok {
				return runtime, errors.New("released MCP policy resolver is unavailable")
			}
			var resolveErr error
			allowed, resolveErr = resolver.ResolveReleasedMCPToolKeys(ctx, refs.MCPPolicyIDs)
			if resolveErr != nil {
				return runtime, resolveErr
			}
		}
		filtered := runtime.ToolNames[:0]
		for _, toolName := range runtime.ToolNames {
			if _, _, isMCP := parseHRMCPBindingKey(toolName); !isMCP || allowed[toolName] {
				filtered = append(filtered, toolName)
			}
		}
		runtime.ToolNames = filtered
	}
	runtime.ExecutableToolNames = hr_tools.ResolveBuiltinToolNames(runtime.ToolNames, runtime.CapabilityKeys)
	// Keep the full explicit binding list for platform-owned context tools such
	// as get_application_snapshot. ExecutableToolNames is the separate model /
	// builtin-runner allowlist and must not erase those bindings.
	runtime.Prompt, runtime.PromptContent, runtime.GovernanceErrors = s.loadHRRuntimePromptForRelease(ctx, req, agent, refs.PromptTemplateIDs, runtime.MemorySection)
	selectionMode := "none"
	if len(req.GetAgentSkillVersionIds()) > 0 {
		selectionMode = "manual"
	}
	if !s.skillPackageV2Enabled {
		runtime.AgentSkillSelectionMode = selectionMode
		runtime.AgentSkillRuntimeEvidence = disabledAgentSkillRuntimeEvidence(req, model)
		s.recordAgentSkillRuntimeDecision(
			ctx,
			"disabled",
			runtime.AgentSkillSelectionMode,
			runtime.AgentSkillRuntimeEvidence,
			nil,
			0,
		)
		return runtime, nil
	}
	var skillErrors []hrRuntimeGovernanceError
	skillStartedAt := time.Now()
	runtime.SelectedAgentSkills, runtime.AgentSkillRuntimeEvidence, runtime.AgentSkillConfirmationRequired, skillErrors =
		s.selectHRRuntimeAgentSkillPackages(ctx, req, runtime.CapabilityKeys, model, enforceRelease)
	runtime.GovernanceErrors = append(runtime.GovernanceErrors, skillErrors...)
	switch {
	case len(req.GetAgentSkillVersionIds()) > 0:
		runtime.AgentSkillSelectionMode = "manual"
	case len(runtime.SelectedAgentSkills) > 0:
		runtime.AgentSkillSelectionMode = "auto"
	default:
		runtime.AgentSkillSelectionMode = "none"
	}
	s.recordAgentSkillRuntimeDecision(
		ctx,
		"v2",
		runtime.AgentSkillSelectionMode,
		runtime.AgentSkillRuntimeEvidence,
		skillErrors,
		time.Since(skillStartedAt),
	)
	return runtime, nil
}

func (s *nativeAIService) loadHRAgentConfigForRelease(ctx context.Context, effectiveAgentID int64, allowedIDs []int64) (*pb.AgentConfigInfo, error) {
	if len(allowedIDs) == 0 {
		return s.loadHRAgentConfig(ctx, effectiveAgentID)
	}
	if effectiveAgentID > 0 {
		if !containsRuntimeID(allowedIDs, effectiveAgentID) {
			return nil, fmt.Errorf("effective HR Agent %d is outside the capability release", effectiveAgentID)
		}
		return s.loadHRAgentConfig(ctx, effectiveAgentID)
	}
	for _, id := range allowedIDs {
		agent, loadErr := s.loadHRAgentConfig(ctx, id)
		if loadErr == nil && agent != nil {
			return agent, nil
		}
	}
	return nil, errors.New("capability release has no available HR Agent")
}

func (s *nativeAIService) loadHRAgentConfig(ctx context.Context, effectiveAgentID int64) (*pb.AgentConfigInfo, error) {
	if effectiveAgentID <= 0 {
		return s.loadDefaultHRAgentConfig(ctx)
	}
	if store, ok := s.store.(hrRuntimeAgentByIDStore); ok {
		agent, found, err := store.GetRuntimeAgentConfigByID(ctx, effectiveAgentID)
		if err != nil {
			return nil, err
		}
		if !found || agent == nil || !agent.GetIsEnabled() || !strings.EqualFold(strings.TrimSpace(agent.GetAgentType()), hrRecruitingAgentType) {
			return nil, fmt.Errorf("effective HR Agent %d is unavailable", effectiveAgentID)
		}
		return agent, nil
	}
	rows, _, err := s.store.ListAgentConfigs(ctx, 1, 100, hrRecruitingAgentType)
	if err != nil {
		return nil, err
	}
	for _, agent := range rows {
		if agent != nil && agent.GetId() == effectiveAgentID && agent.GetIsEnabled() {
			return agent, nil
		}
	}
	return nil, fmt.Errorf("effective HR Agent %d is unavailable", effectiveAgentID)
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

func (s *nativeAIService) loadHRRuntimePrompt(ctx context.Context, req *pb.ChatRequest, agent *pb.AgentConfigInfo) (*pb.PromptTemplateInfo, string, []hrRuntimeGovernanceError) {
	return s.loadHRRuntimePromptForRelease(ctx, req, agent, nil, emptyMemorySection)
}

func (s *nativeAIService) loadHRRuntimePromptForRelease(ctx context.Context, req *pb.ChatRequest, agent *pb.AgentConfigInfo, allowedIDs []int64, memorySection string) (*pb.PromptTemplateInfo, string, []hrRuntimeGovernanceError) {
	if s == nil || s.store == nil {
		return nil, "", nil
	}
	if agent != nil && agent.GetPromptTemplateId() > 0 {
		promptID := agent.GetPromptTemplateId()
		if len(allowedIDs) > 0 && !containsRuntimeID(allowedIDs, promptID) {
			return nil, "", []hrRuntimeGovernanceError{{Source: "prompt", Code: "outside_capability_release", ResourceID: promptID}}
		}
		if promptStore, ok := s.store.(hrRuntimePromptTemplateStore); ok {
			template, found, err := promptStore.GetRuntimePromptTemplateByID(ctx, promptID)
			if err != nil {
				return nil, "", []hrRuntimeGovernanceError{{Source: "prompt", Code: "lookup_failed", ResourceID: promptID}}
			}
			if !found || template == nil {
				return nil, "", []hrRuntimeGovernanceError{{Source: "prompt", Code: "not_found", ResourceID: promptID}}
			}
			if !hrPromptTemplateUsable(template) {
				return nil, "", []hrRuntimeGovernanceError{{Source: "prompt", Code: "incompatible", ResourceID: promptID}}
			}
			content, renderErr := renderHRRuntimePrompt(template.GetContent(), hrRuntimePromptVariablesWithMemory(req, memorySection))
			if renderErr != nil {
				return nil, "", []hrRuntimeGovernanceError{{Source: "prompt", Code: "invalid_variables", ResourceID: promptID}}
			}
			return template, content, nil
		}
		return nil, "", []hrRuntimeGovernanceError{{Source: "prompt", Code: "store_unavailable", ResourceID: promptID}}
	}
	if len(allowedIDs) > 0 {
		if promptStore, ok := s.store.(hrRuntimePromptTemplateStore); ok {
			for _, promptID := range allowedIDs {
				template, found, lookupErr := promptStore.GetRuntimePromptTemplateByID(ctx, promptID)
				if lookupErr == nil && found && hrPromptTemplateUsable(template) {
					content, renderErr := renderHRRuntimePrompt(template.GetContent(), hrRuntimePromptVariablesWithMemory(req, memorySection))
					if renderErr == nil {
						return template, content, nil
					}
				}
			}
		}
		return nil, "", []hrRuntimeGovernanceError{{Source: "prompt", Code: "release_prompt_unavailable"}}
	}
	if promptStore, ok := s.store.(activePromptStore); ok {
		for _, agentType := range []string{hrRecruitingAgentType, "hr_agent"} {
			resp, err := promptStore.GetActivePromptByAgentType(ctx, &pb.GetActivePromptByAgentTypeRequest{AgentType: agentType, PromptRole: hrRuntimePromptRoleSystem})
			if err == nil && resp != nil && resp.GetCode() == 0 && hrPromptTemplateUsable(resp.GetTemplate()) {
				template := resp.GetTemplate()
				content, renderErr := renderHRRuntimePrompt(template.GetContent(), hrRuntimePromptVariablesWithMemory(req, memorySection))
				if renderErr != nil {
					return nil, "", []hrRuntimeGovernanceError{{Source: "prompt", Code: "invalid_variables", ResourceID: template.GetId()}}
				}
				return template, content, nil
			}
		}
	}
	return nil, "", nil
}

func containsRuntimeID(ids []int64, target int64) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func hrPromptTemplateUsable(template *pb.PromptTemplateInfo) bool {
	if template == nil || strings.TrimSpace(template.GetContent()) == "" {
		return false
	}
	if !template.GetIsActive() {
		return false
	}
	role := strings.TrimSpace(template.GetPromptRole())
	if !strings.EqualFold(role, hrRuntimePromptRoleSystem) {
		return false
	}
	agentType := strings.TrimSpace(template.GetAgentType())
	if agentType == "" {
		return false
	}
	return strings.EqualFold(agentType, hrRecruitingAgentType) || strings.EqualFold(agentType, "hr_agent")
}

var hrRuntimePromptVariableName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func hrRuntimePromptVariables(req *pb.ChatRequest) map[string]string {
	contextLine := "当前未指定投递上下文"
	if req.GetApplicationId() > 0 {
		contextLine = "当前投递 ID: " + strconv.FormatInt(req.GetApplicationId(), 10)
	}
	return map[string]string{
		"hr_id":           strconv.FormatInt(req.GetHrId(), 10),
		"session_id":      strconv.FormatInt(req.GetSessionId(), 10),
		"application_id":  strconv.FormatInt(req.GetApplicationId(), 10),
		"current_date":    time.Now().Format("2006-01-02"),
		"context_line":    contextLine,
		"summary_section": "会话历史由运行时上下文预算器统一提供。",
		"memory_section":  "当前没有额外注入的长期记忆。",
	}
}

func renderHRRuntimePrompt(content string, variables map[string]string) (string, error) {
	var rendered strings.Builder
	for cursor := 0; cursor < len(content); {
		rest := content[cursor:]
		openOffset := strings.Index(rest, "{{")
		closeOffset := strings.Index(rest, "}}")
		if openOffset < 0 {
			if closeOffset >= 0 {
				return "", fmt.Errorf("prompt contains unsupported or malformed variable")
			}
			rendered.WriteString(rest)
			break
		}
		if closeOffset >= 0 && closeOffset < openOffset {
			return "", fmt.Errorf("prompt contains unsupported or malformed variable")
		}
		open := cursor + openOffset
		if open > 0 && content[open-1] == '{' {
			return "", fmt.Errorf("prompt contains unsupported or malformed variable")
		}
		closeTail := strings.Index(content[open+2:], "}}")
		if closeTail < 0 {
			return "", fmt.Errorf("prompt contains unsupported or malformed variable")
		}
		close := open + 2 + closeTail
		if close+2 < len(content) && content[close+2] == '}' {
			return "", fmt.Errorf("prompt contains unsupported or malformed variable")
		}
		name := strings.TrimSpace(content[open+2 : close])
		if !hrRuntimePromptVariableName.MatchString(name) {
			return "", fmt.Errorf("prompt contains unsupported or malformed variable")
		}
		value, ok := variables[name]
		if !ok {
			return "", fmt.Errorf("prompt contains unsupported or malformed variable")
		}
		rendered.WriteString(content[cursor:open])
		rendered.WriteString(value)
		cursor = close + 2
	}
	return rendered.String(), nil
}

func outsideReleaseSkillErrors(ids []int64) []hrRuntimeGovernanceError {
	result := make([]hrRuntimeGovernanceError, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, hrRuntimeGovernanceError{Source: "agent_skill", Code: "outside_capability_release", ResourceID: id})
	}
	return result
}

func int64RuntimeSet(ids []int64) map[int64]bool {
	result := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id > 0 {
			result[id] = true
		}
	}
	return result
}

func hrRuntimeCapabilityKeys(agent *pb.AgentConfigInfo) []string {
	if agent == nil {
		return []string{hrCandidateSearchCapability, "resume_intelligence", "interview_context"}
	}
	if len(agent.GetCapabilityBindings()) == 0 {
		return nil
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
	if agent == nil {
		return []string{hrApplicationSnapshotTool}
	}
	if len(agent.GetToolBindings()) == 0 {
		return nil
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

func hrRuntimeAgentSkillVersionIDs(governance hrRuntimeGovernanceContext) []int64 {
	ids := make([]int64, 0, len(governance.SelectedAgentSkills))
	for _, skill := range governance.SelectedAgentSkills {
		if skill.VersionID > 0 && skill.Included {
			ids = append(ids, skill.VersionID)
		}
	}
	return ids
}

func hrRuntimeAgentSkillNames(governance hrRuntimeGovernanceContext) []string {
	names := make([]string, 0, len(governance.SelectedAgentSkills))
	for _, skill := range governance.SelectedAgentSkills {
		if !skill.Included {
			continue
		}
		// Persist the user-facing label for history badges; fall back to technical name.
		label := strings.TrimSpace(skill.DisplayName)
		if label == "" {
			label = strings.TrimSpace(skill.Name)
		}
		if label != "" {
			names = append(names, label)
		}
	}
	return names
}

func hrRuntimeCompletionOptions(governance hrRuntimeGovernanceContext) ChatCompletionOptions {
	const maxAgentIterations = 20
	var opts ChatCompletionOptions
	if governance.Agent == nil {
		return opts
	}
	if governance.Agent.GetTemperatureOverride() > 0 {
		temperature := governance.Agent.GetTemperatureOverride()
		opts.TemperatureOverride = &temperature
	}
	iterations := int(governance.Agent.GetMaxIterations())
	if iterations > maxAgentIterations {
		iterations = maxAgentIterations
	}
	if iterations > 0 {
		opts.MaxIterations = iterations
	}
	return opts
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
			if err := emit(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "tool_calling", EventMessage: "ai.event.application_snapshot_loading", ToolName: hrApplicationSnapshotTool, CreatedAt: formatTime(time.Now())}, nil); err != nil {
				return nil, err
			}
		}
		if err := s.verifyAgentRunSkillExecutionLease(ctx); err != nil {
			return traces, err
		}
		start := time.Now()
		snapshot, err := s.applications.GetApplicationSnapshot(ctx, &pb.GetApplicationSnapshotRequest{ApplicationId: req.GetApplicationId()})
		duration := time.Since(start)
		if leaseErr := s.verifyAgentRunSkillExecutionLease(ctx); leaseErr != nil {
			return traces, leaseErr
		}
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
			event := &pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "tool_done", EventMessage: "ai.event.application_snapshot_loaded", ToolName: "get_application_snapshot", CreatedAt: formatTime(time.Now())}
			if trace.ErrorMsg != "" {
				event.EventType = "error"
				event.EventMessage = trace.ErrorMsg
				event.ErrorType = "TOOL_ERROR"
				event.Msg = trace.ErrorMsg
			}
			if err := emit(event, nil); err != nil {
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
		argumentsHash := hrRuntimeMCPArgumentsHash(call, argsJSON)
		confirmationApproved := hrRuntimeMCPApprovalMatches(governance.MCPApproval, call.runtimeName, argumentsHash)
		if emit != nil {
			if err := emit(&pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "tool_calling", EventMessage: "ai.event.mcp_calling", ToolName: call.runtimeName, CreatedAt: formatTime(time.Now())}, nil); err != nil {
				return traces, err
			}
		}
		if err := s.verifyAgentRunSkillExecutionLease(ctx); err != nil {
			return traces, err
		}
		resp, err := service.CallMCPTool(ctx, &pb.CallMCPToolRequest{ServerId: call.serverID, ToolName: call.toolName, ArgsJson: argsJSON, CalledByHrId: req.GetHrId(), SessionId: sessionID, CallerRole: "hr_agent", CallerScope: "agent_runtime", ConfirmationApproved: confirmationApproved})
		if leaseErr := s.verifyAgentRunSkillExecutionLease(ctx); leaseErr != nil {
			return traces, leaseErr
		}
		trace := ToolTraceRow{SessionID: sessionID, AgentRunID: agentRunID, ToolCallID: fmt.Sprintf("mcp-%d-%s", call.serverID, call.toolName), ToolName: call.runtimeName, ArgsJSON: service.RedactedArgsForTool(ctx, call.serverID, call.toolName, argsJSON), CreatedAt: time.Now()}
		var confirmationErr *mcpConfirmationRequiredError
		if err != nil {
			trace.ErrorMsg = err.Error()
			trace.Status = "error"
		} else if resp.GetPolicyDecision() == model.MCPPolicyDecisionConfirmationRequired {
			trace.ErrorMsg = "confirmation_required"
			trace.Status = "error"
			confirmationErr = &mcpConfirmationRequiredError{Confirmation: newAgentRunMCPConfirmation(call.runtimeName, argumentsHash, resp.GetPolicyReason())}
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
		traces = append(traces, trace)
		if confirmationErr != nil {
			return traces, confirmationErr
		}
		if emit != nil {
			event := &pb.ChatStreamResponse{Code: 0, Msg: "common.success", EventType: "tool_done", EventMessage: "ai.event.mcp_finished", ToolName: call.runtimeName, CreatedAt: formatTime(time.Now())}
			if trace.ErrorMsg != "" {
				event.EventType = "error"
				event.EventMessage = trace.ErrorMsg
				event.ErrorType = "MCP_TOOL_ERROR"
				event.Msg = trace.ErrorMsg
			}
			if err := emit(event, nil); err != nil {
				return traces, err
			}
		}
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

func hrRuntimeMCPArgumentsHash(call hrRuntimeMCPToolCall, argsJSON string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d\n%s\n%s", call.serverID, call.toolName, strings.TrimSpace(argsJSON))))
	return hex.EncodeToString(sum[:])
}

func hrRuntimeMCPApprovalMatches(approval *agentRunMCPApproval, capabilityKey, argumentsHash string) bool {
	return approval != nil &&
		strings.TrimSpace(approval.ConfirmationID) != "" &&
		approval.CapabilityKey == strings.TrimSpace(capabilityKey) &&
		approval.ArgumentsHash == strings.TrimSpace(argumentsHash)
}

func newAgentRunMCPConfirmation(capabilityKey, argumentsHash, reason string) agentRunMCPConfirmation {
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		fallback := sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%d", capabilityKey, argumentsHash, time.Now().UnixNano())))
		random = fallback[:24]
	}
	return agentRunMCPConfirmation{
		ID:            hex.EncodeToString(random),
		CapabilityKey: strings.TrimSpace(capabilityKey),
		ArgumentsHash: strings.TrimSpace(argumentsHash),
		Reason:        strings.TrimSpace(reason),
		ExpiresAt:     formatTime(time.Now().Add(10 * time.Minute)),
	}
}

func hrRuntimeSelectedMCPTools(req *pb.ChatRequest, governance hrRuntimeGovernanceContext) []hrRuntimeMCPToolCall {
	if governance.Agent == nil {
		return nil
	}
	selected := normalizedStringSet(req.GetCapabilityKeys())
	if len(selected) == 0 {
		return nil
	}
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
		if !selected[key] {
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
		if execution, governed := agentRunSkillApprovalExecutionFromContext(ctx); governed {
			store, ok := s.store.(agentRunSkillExecutionFenceStore)
			if !ok || execution.RunID != trace.AgentRunID || execution.HRID != hrID {
				return ToolTraceRow{}, errAgentRunExecutionLeaseLost
			}
			_, persistedTrace, owned, err := store.AppendAgentRunToolTraceForSkillLease(
				ctx,
				hrID,
				execution.RunID,
				execution.Approval.DispatchLeaseID,
				step,
				trace,
			)
			if err != nil {
				return ToolTraceRow{}, err
			}
			if !owned {
				return ToolTraceRow{}, errAgentRunExecutionLeaseLost
			}
			return persistedTrace, nil
		}
		if hasAgentRunSkillApprovalContext(ctx) {
			return ToolTraceRow{}, errAgentRunExecutionLeaseLost
		}
		if persistedStep, err := s.store.AppendAgentRunStep(ctx, step); err == nil && persistedStep.ID != 0 {
			trace.AgentRunStepID = persistedStep.ID
		}
	}
	return s.store.AppendToolTrace(ctx, hrID, trace)
}

func hrRuntimeAllowsApplicationSnapshot(req *pb.ChatRequest, governance hrRuntimeGovernanceContext) bool {
	if governance.Agent != nil && !hrRuntimeToolSetAllowsApplicationContext(governance.ToolNames) {
		return false
	}
	selected := normalizedStringSet(req.GetCapabilityKeys())
	if len(selected) > 0 && !selected[hrCandidateSearchCapability] {
		return false
	}
	return true
}

func hrRuntimeToolSetAllowsApplicationContext(toolNames []string) bool {
	for _, name := range toolNames {
		switch hr_tools.NormalizeToolName(name) {
		case hrApplicationSnapshotTool, "get_candidate_detail", "get_resume_profile", "parse_resume_profile", "get_candidate_match_evaluation", "evaluate_candidate_match":
			return true
		}
	}
	return false
}

func renderHRProviderPrompt(req *pb.ChatRequest, history []ChatMessageRow, current ChatMessageRow, traces []ToolTraceRow, governance hrRuntimeGovernanceContext) string {
	var b strings.Builder
	b.WriteString("System:\n你是 Smart Recruit 的 HR 招聘助手。必须优先依据系统工具返回的招聘数据回答；如果工具不可用，明确说明限制，不要编造候选人、岗位或投递数据。涉及岗位列表、投递统计、候选人信息等实时数据时，必须使用工具查询结果，禁止凭常识编造。\n若用户只是问候、感谢或询问你能做什么，不要调用任何工具，直接礼貌回复并简要介绍能力。\n")
	if governance.Prompt != nil && strings.TrimSpace(governance.PromptContent) != "" {
		b.WriteString("\nActive prompt template:\n")
		b.WriteString(strings.TrimSpace(governance.PromptContent))
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
		if !skill.Included || strings.TrimSpace(skill.CoreMarkdown) == "" {
			continue
		}
		b.WriteString("\n## Active Agent Skill: ")
		if skill.DisplayName != "" {
			b.WriteString(skill.DisplayName)
		} else {
			b.WriteString(skill.Name)
		}
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(skill.CoreMarkdown))
		for _, section := range skill.Sections {
			if !section.Included || strings.TrimSpace(section.ContentMarkdown) == "" {
				continue
			}
			b.WriteString("\n\n### Reference: ")
			if strings.TrimSpace(section.Title) != "" {
				b.WriteString(strings.TrimSpace(section.Title))
			} else {
				b.WriteString(section.Key)
			}
			b.WriteString("\n")
			b.WriteString(strings.TrimSpace(section.ContentMarkdown))
		}
		if skill.CompositionRole == domainagentskill.CompositionRolePrimary &&
			skill.OutputContract.Mode == domainagentskill.OutputModeAdvisory &&
			strings.TrimSpace(skill.AdvisoryInstruction) != "" {
			b.WriteString("\n\n")
			b.WriteString(strings.TrimSpace(skill.AdvisoryInstruction))
		}
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
	if len(req.GetAgentSkillVersionIds()) > 0 || len(req.GetCapabilityKeys()) > 0 {
		b.WriteString("\nRuntime selection:\n")
		b.WriteString(marshalJSONString(map[string]any{
			"agent_skill_version_ids": req.GetAgentSkillVersionIds(),
			"capability_keys":         req.GetCapabilityKeys(),
		}))
		b.WriteString("\n")
	}
	if len(governance.SelectedAgentSkills) > 0 || governance.AgentSkillSelectionMode != "" {
		b.WriteString("\nSelected Agent Skills:\n")
		b.WriteString(marshalJSONString(map[string]any{
			"mode":         governance.AgentSkillSelectionMode,
			"agent_skills": hrRuntimeAgentSkillEvidence(governance.SelectedAgentSkills),
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

// estimateHRMessagesContextUsage meters the exact message and tool-schema
// values passed to the tool/ADK provider. Each input is assigned to exactly one
// breakdown bucket, so the breakdown is a strict partition of the estimate.
func estimateHRMessagesContextUsage(model RuntimeModelInfo, messages []*schema.Message, toolSchemas []*schema.ToolInfo, current string, traces []ToolTraceRow, governance hrRuntimeGovernanceContext) *pb.ContextUsageInfo {
	var rawSystemTokens, recentTokens, currentTokens, summaryTokens, memoryTokens int64
	currentIndex := -1
	current = strings.TrimSpace(current)
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message != nil && message.Role == schema.User && current != "" && strings.TrimSpace(message.Content) == current {
			currentIndex = index
			break
		}
	}
	for index, message := range messages {
		if message == nil {
			continue
		}
		content := message.Content
		tokens := int64(estimateTokensConservative(content))
		switch {
		case strings.HasPrefix(content, hrSummaryMessagePrefix):
			summaryTokens += int64(estimateTokensConservative(strings.TrimPrefix(content, hrSummaryMessagePrefix)))
		case strings.HasPrefix(content, contextbudget.MemoryMessagePrefix):
			memoryTokens += int64(estimateTokensConservative(strings.TrimPrefix(content, contextbudget.MemoryMessagePrefix)))
		case message.Role == schema.System:
			rawSystemTokens += tokens
		case index == currentIndex:
			currentTokens += tokens
		default:
			recentTokens += tokens
		}
	}
	skillTokens, rawSystemTokens := allocateKnownContextTokens(rawSystemTokens, estimateHRSkillFragmentTokens(governance))
	toolResultTokens, systemTokens := allocateKnownContextTokens(rawSystemTokens, estimateHRToolTraceFragmentTokens(traces))
	toolSchemaTokens := int64(estimateToolSchemaTokens(toolSchemas))
	protocolTokens := int64(len(messages)*4 + 2)
	return newEstimatedHRContextUsage(model, &pb.ContextUsageBreakdown{
		SystemPromptTokens:     saturatingInt32(systemTokens),
		SummaryTokens:          saturatingInt32(summaryTokens),
		MemoryTokens:           saturatingInt32(memoryTokens),
		RecentMessageTokens:    saturatingInt32(recentTokens),
		CurrentMessageTokens:   saturatingInt32(currentTokens),
		SkillTokens:            saturatingInt32(skillTokens),
		ToolResultTokens:       saturatingInt32(toolResultTokens),
		ToolSchemaTokens:       saturatingInt32(toolSchemaTokens),
		ProtocolOverheadTokens: saturatingInt32(protocolTokens),
	}, len(messages))
}

// estimateHRCompletionContextUsage meters the exact flattened prompt passed to
// the completion provider. It is one provider input message plus its framing.
func estimateHRCompletionContextUsage(model RuntimeModelInfo, prompt, current string, history []ChatMessageRow, currentRow ChatMessageRow, traces []ToolTraceRow, governance hrRuntimeGovernanceContext) *pb.ContextUsageInfo {
	remaining := int64(estimateTokensConservative(prompt))
	skillTokens, remaining := allocateKnownContextTokens(remaining, estimateHRSkillFragmentTokens(governance))
	toolResultTokens, remaining := allocateKnownContextTokens(remaining, estimateHRToolTraceFragmentTokens(traces))

	currentOccurrences := int64(1)
	if currentRow.ID != 0 {
		// renderHRProviderPrompt includes the current row in Recent conversation
		// and also emits req.message in the final User section.
		currentOccurrences = 2
	}
	currentWanted := int64(estimateTokensConservative(current)) * currentOccurrences
	currentTokens, remaining := allocateKnownContextTokens(remaining, currentWanted)

	var recentWanted int64
	for _, message := range history {
		if currentRow.ID != 0 && message.ID == currentRow.ID {
			continue
		}
		recentWanted += int64(estimateTokensConservative(message.Content))
	}
	recentTokens, systemTokens := allocateKnownContextTokens(remaining, recentWanted)
	return newEstimatedHRContextUsage(model, &pb.ContextUsageBreakdown{
		SystemPromptTokens:     saturatingInt32(systemTokens),
		RecentMessageTokens:    saturatingInt32(recentTokens),
		CurrentMessageTokens:   saturatingInt32(currentTokens),
		SkillTokens:            saturatingInt32(skillTokens),
		ToolResultTokens:       saturatingInt32(toolResultTokens),
		ProtocolOverheadTokens: 6,
	}, 1)
}

func allocateKnownContextTokens(available, wanted int64) (allocated, remaining int64) {
	if available <= 0 || wanted <= 0 {
		return 0, maxInt64(available, 0)
	}
	if wanted > available {
		wanted = available
	}
	return wanted, available - wanted
}

func estimateHRSkillFragmentTokens(governance hrRuntimeGovernanceContext) int64 {
	var total int64
	for _, skill := range governance.SelectedAgentSkills {
		if skill.Included {
			total += int64(skill.LoadedTokens)
		}
	}
	return total
}

func estimateHRToolTraceFragmentTokens(traces []ToolTraceRow) int64 {
	var total int64
	for _, trace := range traces {
		fragment := trace.ResultContent
		if strings.TrimSpace(trace.ErrorMsg) != "" {
			fragment = trace.ErrorMsg
		}
		total += int64(estimateTokensConservative(fragment))
	}
	return total
}

func newEstimatedHRContextUsage(model RuntimeModelInfo, breakdown *pb.ContextUsageBreakdown, includedMessageCount int) *pb.ContextUsageInfo {
	promptTokens := contextUsageBreakdownTotal(breakdown)
	usage := newHRContextUsageEnvelope(model, promptTokens)
	usage.PromptTokensEstimated = promptTokens
	usage.Estimated = true
	usage.Source = "conservative_estimator"
	usage.Stage = "pre_generation"
	usage.IncludedMessageCount = saturatingInt32(int64(maxInt(includedMessageCount, 0)))
	usage.Breakdown = breakdown
	return usage
}

func contextUsageBreakdownTotal(breakdown *pb.ContextUsageBreakdown) int32 {
	if breakdown == nil {
		return 0
	}
	total := int64(breakdown.GetSystemPromptTokens()) +
		int64(breakdown.GetRecentMessageTokens()) +
		int64(breakdown.GetSummaryTokens()) +
		int64(breakdown.GetMemoryTokens()) +
		int64(breakdown.GetCurrentMessageTokens()) +
		int64(breakdown.GetSkillTokens()) +
		int64(breakdown.GetToolResultTokens()) +
		int64(breakdown.GetToolSchemaTokens()) +
		int64(breakdown.GetProtocolOverheadTokens())
	return saturatingInt32(total)
}

func estimateTokens(value string) int {
	return estimateTokensConservative(value)
}

func estimateTokensConservative(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	ascii := 0
	nonASCII := 0
	for _, r := range trimmed {
		if r <= 0x7f {
			ascii++
		} else {
			nonASCII++
		}
	}
	return maxInt((ascii+3)/4+nonASCII, 1)
}

func estimateToolSchemaTokens(tools []*schema.ToolInfo) int {
	total := 0
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		payload, err := json.Marshal(tool)
		if err != nil {
			continue
		}
		total += estimateTokensConservative(string(payload))
	}
	return total
}

func newHRContextUsageEnvelope(model RuntimeModelInfo, promptTokens int32) *pb.ContextUsageInfo {
	usage := &pb.ContextUsageInfo{
		ModelId:                model.ID,
		ModelName:              model.Name,
		ContextWindowTokens:    model.ContextWindowTokens,
		MaxOutputTokens:        model.MaxOutputTokens,
		BudgetStatus:           "unknown_config",
		RequestedModelId:       model.RequestedModelID,
		EffectiveModelId:       model.ID,
		ModelFallbackReason:    model.FallbackReason,
		CapabilityVersionId:    model.CapabilityVersionID,
		CapabilitySnapshotHash: model.CapabilitySnapshotHash,
	}
	window := int64(model.ContextWindowTokens)
	maxOutput := int64(model.MaxOutputTokens)
	if maxOutput < 0 {
		usage.BudgetStatus = "invalid_config"
		return usage
	}
	if window <= 0 {
		return usage
	}
	safety := (window + 19) / 20
	if safety < 256 {
		safety = 256
	}
	if safety > 2048 {
		safety = 2048
	}
	usage.SafetyMarginTokens = saturatingInt32(safety)
	inputBudget := window - maxOutput - safety
	usage.InputBudgetTokens = clampInt32(inputBudget)
	if inputBudget <= 0 {
		usage.BudgetStatus = "invalid_config"
		return usage
	}
	prompt := int64(promptTokens)
	usage.RemainingTokensEstimated = saturatingInt32(maxInt64(inputBudget-prompt, 0))
	usage.UsageRatio = float64(prompt) / float64(window)
	usage.BudgetUsageRatio = float64(prompt) / float64(inputBudget)
	switch {
	case usage.BudgetUsageRatio >= 0.75:
		usage.BudgetStatus = "over_target"
	case usage.BudgetUsageRatio >= 0.60:
		usage.BudgetStatus = "approaching_target"
	default:
		usage.BudgetStatus = "within_budget"
	}
	return usage
}

func applyActualContextUsage(usage *pb.ContextUsageInfo, actual *schema.TokenUsage) bool {
	if usage == nil || actual == nil {
		return false
	}
	if actual.PromptTokens < 0 || actual.CompletionTokens < 0 || actual.TotalTokens < 0 {
		return false
	}
	if actual.PromptTokens == 0 && actual.CompletionTokens == 0 && actual.TotalTokens == 0 {
		return false
	}
	promptTokens := int64(actual.PromptTokens)
	completionTokens := int64(actual.CompletionTokens)
	totalTokens := int64(actual.TotalTokens)
	// Some providers omit total or return a total smaller than the reported
	// components. Use the larger value to avoid under-reporting while retaining
	// a provider total that may legitimately include extra token classes.
	componentTotal := promptTokens
	if completionTokens > math.MaxInt64-componentTotal {
		componentTotal = math.MaxInt64
	} else {
		componentTotal += completionTokens
	}
	if totalTokens < componentTotal {
		totalTokens = componentTotal
	}
	usage.PromptTokensActual = saturatingInt32(promptTokens)
	usage.CompletionTokensActual = saturatingInt32(completionTokens)
	usage.TotalTokensActual = saturatingInt32(totalTokens)
	usage.Estimated = false
	usage.Source = "provider_actual"
	usage.Stage = "final"
	window := int64(usage.ContextWindowTokens)
	if window > 0 {
		usage.UsageRatio = float64(promptTokens) / float64(window)
	}
	inputBudget := int64(usage.InputBudgetTokens)
	if inputBudget > 0 {
		usage.RemainingTokensEstimated = saturatingInt32(maxInt64(inputBudget-promptTokens, 0))
		usage.BudgetUsageRatio = float64(promptTokens) / float64(inputBudget)
		switch {
		case usage.BudgetUsageRatio >= 0.75:
			usage.BudgetStatus = "over_target"
		case usage.BudgetUsageRatio >= 0.60:
			usage.BudgetStatus = "approaching_target"
		default:
			usage.BudgetStatus = "within_budget"
		}
	}
	return true
}

func saturatingInt32(value int64) int32 {
	if value <= 0 {
		return 0
	}
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(value)
}

func clampInt32(value int64) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}

func applyToolMetadataContextUsage(usage *pb.ContextUsageInfo, metadata commonsai.ToolMetadata) bool {
	// BillingTokenUsage is intentionally cumulative across model calls and is
	// reserved for cost/audit reporting. The current context snapshot must use
	// only the latest model call exposed by ContextTokenUsage.
	return applyActualContextUsage(usage, metadata.ContextTokenUsage)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
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
		if strings.TrimSpace(row.ErrorMsg) == "" && strings.EqualFold(strings.TrimSpace(row.Status), "success") && toolResultContentUseful(row.ResultContent) {
			return true
		}
	}
	return false
}

func toolResultContentUseful(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err == nil {
		if message, ok := payload["error"].(string); ok && strings.TrimSpace(message) != "" {
			return false
		}
		if kind, ok := payload["error_type"].(string); ok && strings.TrimSpace(kind) != "" {
			return false
		}
	}
	return true
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

func buildHRProcessContent(traces []ToolTraceRow, usage *pb.ContextUsageInfo, fallbackUsed bool, governance hrRuntimeGovernanceContext, plan commonsai.RecruitingPlan, runtimeLabel string, suggestedQuestions []string) string {
	if strings.TrimSpace(runtimeLabel) == "" {
		runtimeLabel = agentRuntimeADK
	}
	payload := map[string]any{
		"runtime":       runtimeLabel,
		"fallback_used": fallbackUsed,
		"tool_count":    len(traces),
	}
	if summary := buildAgentRunDisplaySummary(plan, traces, usage, fallbackUsed); len(summary) > 0 {
		payload["display_summary"] = summary
	}
	if usage != nil {
		payload["context_usage"] = contextUsagePayload(usage)
	}
	if len(suggestedQuestions) > 0 {
		payload["suggested_questions"] = append([]string(nil), suggestedQuestions...)
	}
	if governance.Agent != nil || governance.Prompt != nil || len(governance.SelectedAgentSkills) > 0 || governance.AgentSkillSelectionMode != "" || len(governance.GovernanceErrors) > 0 || governance.MemoryEvidence.Count > 0 {
		governancePayload := map[string]any{
			"agent_id":                     agentConfigID(governance.Agent),
			"agent_type":                   agentConfigType(governance.Agent),
			"agent_name":                   agentConfigName(governance.Agent),
			"prompt_template_id":           promptTemplateID(governance.Prompt),
			"prompt_template_version":      promptTemplateVersion(governance.Prompt),
			"capability_keys":              governance.CapabilityKeys,
			"tool_names":                   governance.ToolNames,
			"agent_skill_selection_mode":   governance.AgentSkillSelectionMode,
			"agent_skill_runtime_evidence": governance.AgentSkillRuntimeEvidence,
			"governance_errors":            governance.GovernanceErrors,
		}
		if governance.MemoryEvidence.Count > 0 {
			governancePayload["memory_inject"] = governance.MemoryEvidence
		}
		payload["governance"] = governancePayload
	}
	if len(traces) > 0 {
		toolResults := make([]map[string]any, 0, len(traces))
		for _, trace := range traces {
			toolResults = append(toolResults, map[string]any{
				"tool_name": trace.ToolName,
				"status":    trace.Status,
			})
		}
		payload["tool_results"] = toolResults
	}
	return marshalJSONString(payload)
}

func buildAgentRunDisplaySummary(plan commonsai.RecruitingPlan, traces []ToolTraceRow, _ *pb.ContextUsageInfo, fallbackUsed bool) []string {
	lines := make([]string, 0, len(plan.DisplaySteps)+2)
	seenLines := make(map[string]struct{}, len(plan.DisplaySteps)+2)
	appendLine := func(line string) {
		line = strings.TrimSpace(line)
		if line == "" {
			return
		}
		if _, exists := seenLines[line]; exists {
			return
		}
		seenLines[line] = struct{}{}
		lines = append(lines, line)
	}
	appendLine("已分析问题并确定所需招聘数据。")
	traceByTool := make(map[string]ToolTraceRow, len(traces))
	for _, trace := range traces {
		if strings.TrimSpace(trace.ToolName) != "" {
			traceByTool[hr_tools.NormalizeToolName(trace.ToolName)] = trace
		}
	}
	for _, step := range plan.DisplaySteps {
		if step.Key == "compose_answer" {
			continue
		}
		if strings.TrimSpace(step.Purpose) == "" {
			continue
		}
		success, failed, attempted := false, false, false
		for _, tool := range step.Tools {
			trace, ok := traceByTool[hr_tools.NormalizeToolName(tool)]
			if !ok {
				continue
			}
			attempted = true
			if trace.Status == "success" {
				success = true
			}
			if trace.Status == "error" {
				failed = true
			}
		}
		switch {
		case success:
			appendLine("已完成：" + step.Purpose + "。")
		case failed:
			appendLine(step.Purpose + "暂时没有完全完成，已继续使用其他可用数据推进。")
		case attempted:
			appendLine("已尝试：" + step.Purpose + "。")
		}
	}
	if fallbackUsed {
		appendLine("部分数据暂时不足，已使用可用信息保守作答。")
	}
	if len(lines) == 1 && len(traces) > 0 {
		appendLine("已查询实时招聘数据并获取回答所需信息。")
	}
	appendLine("已整理查询结果并生成回复。")
	return lines
}

func buildAgentRunProcessSnapshot(plan commonsai.RecruitingPlan, traces []ToolTraceRow, usage *pb.ContextUsageInfo, fallbackUsed bool) string {
	return strings.Join(buildAgentRunDisplaySummary(plan, traces, usage, fallbackUsed), "\n")
}

func hrRuntimeAgentSkillEvidence(skills []hrRuntimeAgentSkill) []map[string]any {
	result := make([]map[string]any, 0, len(skills))
	for _, skill := range skills {
		result = append(result, map[string]any{
			"skill_id":              skill.ID,
			"version_id":            skill.VersionID,
			"version":               skill.Version,
			"compiled_hash":         skill.CompiledHash,
			"name":                  skill.Name,
			"display_name":          skill.DisplayName,
			"manual":                skill.Manual,
			"reason":                skill.Reason,
			"risk_level":            skill.RiskLevel,
			"composition_role":      skill.CompositionRole,
			"loaded_tokens":         skill.LoadedTokens,
			"required_capabilities": skill.RequiredCapabilities,
		})
	}
	return result
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
	case string:
		n, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return n
	default:
		return 0
	}
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func (s *nativeAIService) History(ctx context.Context, req *pb.ChatHistoryRequest) (*pb.ChatHistoryResponse, error) {
	rows, err := s.listMessages(ctx, ownerRoleHR, req.GetHrId(), 0, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: 0, Msg: "common.success", List: mapChatMessages(rows)}, nil
}

func (s *nativeAIService) AnalyzeApplication(ctx context.Context, req *pb.AnalyzeApplicationRequest) (*pb.AnalyzeApplicationResponse, error) {
	modelID := req.GetModelId()
	var runtimeModel RuntimeModelInfo
	if req.GetCapabilityVersionId() > 0 {
		resolver, ok := s.store.(capabilityRuntimeModelResolver)
		if !ok {
			return &pb.AnalyzeApplicationResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
		}
		resolved, resolveErr := resolver.ResolveCapabilityRuntimeModel(ctx, "ai.application_analysis", platformAIAudienceTenantHR, req.GetCapabilityVersionId(), modelID)
		if resolveErr != nil {
			return &pb.AnalyzeApplicationResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
		}
		modelID = resolved.EffectiveModelID
		runtimeModel = RuntimeModelInfo{ID: resolved.EffectiveModelID, Name: resolved.ModelName, ProviderName: resolved.ProviderName, ContextWindowTokens: resolved.ContextWindowTokens, MaxOutputTokens: resolved.MaxOutputTokens, RequestedModelID: resolved.RequestedModelID, FallbackReason: resolved.FallbackReason, CapabilityVersionID: resolved.CapabilityVersionID, CapabilitySnapshotHash: resolved.CapabilitySnapshotHash, ConfigurationRefs: resolved.ConfigurationRefs, SkillRuntimePolicy: resolved.SkillRuntimePolicy}
		ctx, resolveErr = s.reserveAIBilling(ctx, billingOwnerTenant, req.GetHrId(), "ai.application_analysis", "application_analysis", runtimeModel.ProviderName, runtimeModel.Name, len([]rune(fmt.Sprintf("Analyze application %d", req.GetApplicationId()))), runtimeModel)
		if resolveErr != nil {
			return nil, resolveErr
		}
		defer s.cancelUnsettledBilling(ctx, "runtime_completed_without_usage")
	}
	reply, err := s.complete(ctx, fmt.Sprintf("Analyze application %d", req.GetApplicationId()), modelID)
	if err != nil {
		if errors.Is(err, errAIProviderRequired) {
			return &pb.AnalyzeApplicationResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
		}
		return nil, err
	}
	if runtimeModel.ID > 0 {
		s.bestEffortMeterUsage(ctx, UsageAuditRow{Provider: runtimeModel.ProviderName, Model: runtimeModel.Name, EstimatedTokens: estimateTokens(fmt.Sprintf("Analyze application %d%s", req.GetApplicationId(), reply))})
	}
	return &pb.AnalyzeApplicationResponse{Code: 0, Msg: "common.success", Reply: reply, ContextUsage: newHRContextUsageEnvelope(runtimeModel, 0)}, nil
}

func (s *nativeAIService) ListChatSessions(ctx context.Context, req *pb.ChatSessionListRequest) (*pb.ChatSessionListResponse, error) {
	rows, total, err := s.listSessions(ctx, ownerRoleHR, req.GetHrId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatSessionListResponse{Code: 0, Msg: "common.success", Total: total, List: mapChatSessions(rows)}, nil
}

func (s *nativeAIService) CreateChatSession(ctx context.Context, req *pb.CreateChatSessionRequest) (*pb.CreateChatSessionResponse, error) {
	session, err := s.ensureSession(ctx, ownerRoleHR, req.GetHrId(), 0, 0, req.GetTitle())
	if err != nil {
		return nil, err
	}
	return &pb.CreateChatSessionResponse{Code: 0, Msg: "common.success", Session: mapChatSession(session)}, nil
}

func (s *nativeAIService) SessionMessages(ctx context.Context, req *pb.SessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	rows, err := s.listMessages(ctx, ownerRoleHR, req.GetHrId(), req.GetSessionId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: 0, Msg: "common.success", List: mapChatMessages(rows)}, nil
}

// PreviewChatContext recompiles the current persisted conversation for the
// requested next-turn model without invoking a model, executing tools, or
// generating/updating summaries. Persisting the selection and resulting
// snapshot is intentionally separate from the read-only compilation step so a
// refresh restores the same model-relative A / B view.
func (s *nativeAIService) PreviewChatContext(ctx context.Context, req *pb.PreviewChatContextRequest) (*pb.PreviewChatContextResponse, error) {
	if s.store == nil {
		return &pb.PreviewChatContextResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	if req.GetSessionId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "chat session id is required")
	}
	session, found, err := s.store.GetChatSession(ctx, ownerRoleHR, req.GetHrId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, status.Error(codes.NotFound, errChatSessionNotFound.Error())
	}
	model, found, err := s.resolveSelectableRuntimeModelInfo(ctx, req.GetModelId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, status.Error(codes.InvalidArgument, "selected model is unavailable")
	}
	chatReq := &pb.ChatRequest{
		HrId:                 req.GetHrId(),
		SessionId:            session.ID,
		ApplicationId:        session.ApplicationID,
		ModelId:              req.GetModelId(),
		CapabilityKeys:       append([]string(nil), req.GetCapabilityKeys()...),
		AgentSkillVersionIds: append([]int64(nil), req.GetAgentSkillVersionIds()...),
	}
	governance, err := s.loadHRRuntimeGovernance(withoutAgentSkillMetrics(ctx), chatReq)
	if err != nil {
		return nil, err
	}
	history, err := s.hrContextMessages(ctx, req.GetHrId(), session.ID)
	if err != nil {
		return nil, err
	}
	usage := s.previewHRConversationContextUsage(ctx, chatReq, model, history, governance)
	sessionStore, ok := s.store.(chatSessionContextModelStore)
	if !ok {
		return &pb.PreviewChatContextResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	if err := sessionStore.UpdateChatSessionContextModel(ctx, ownerRoleHR, req.GetHrId(), session.ID, req.GetModelId(), usage); err != nil {
		return nil, err
	}
	return &pb.PreviewChatContextResponse{
		Code:            0,
		Msg:             "common.success",
		SelectedModelId: req.GetModelId(),
		ContextUsage:    usage,
	}, nil
}

func (s *nativeAIService) previewHRConversationContextUsage(
	ctx context.Context,
	req *pb.ChatRequest,
	model RuntimeModelInfo,
	history []ChatMessageRow,
	governance hrRuntimeGovernanceContext,
) *pb.ContextUsageInfo {
	toolSchemas := hrRecruitingToolSchemas(governance.ExecutableToolNames)
	messages := buildHRToolCallingMessages(req, history, ChatMessageRow{}, nil, governance)
	var summaryStore hrSessionSummaryStore
	if candidate, ok := s.store.(hrSessionSummaryStore); ok {
		summaryStore = candidate
	}
	controller := newHRContextBudgetController(
		ctx,
		model,
		toolSchemas,
		"",
		nil,
		governance,
		history,
		req.GetHrId(),
		req.GetSessionId(),
		summaryStore,
		nil,
	)
	_, _ = controller.prepare(ctx, messages, "model_preview")
	usage := controller.usage
	if usage == nil {
		usage = estimateHRMessagesContextUsage(model, messages, toolSchemas, "", nil, governance)
	}
	usage.PromptTokensActual = 0
	usage.CompletionTokensActual = 0
	usage.TotalTokensActual = 0
	usage.Estimated = true
	usage.Source = "conservative_estimator"
	usage.Stage = "model_preview"
	return usage
}

func (s *nativeAIService) CreateApplicationAnalysisSession(ctx context.Context, req *pb.CreateApplicationAnalysisSessionRequest) (*pb.CreateApplicationAnalysisSessionResponse, error) {
	session, err := s.ensureSession(ctx, ownerRoleHR, req.GetHrId(), 0, req.GetApplicationId(), fmt.Sprintf("Application %d analysis", req.GetApplicationId()))
	if err != nil {
		return nil, err
	}
	message, err := s.store.AppendChatMessage(ctx, ChatMessageRow{
		OwnerRole: ownerRoleHR,
		OwnerID:   req.GetHrId(),
		SessionID: session.ID,
		Role:      "user",
		Content:   "请分析该候选人当前投递简历与岗位的匹配度，并基于真实候选人、岗位和匹配评估数据给出结论。",
		ModelID:   req.GetModelId(),
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateApplicationAnalysisSessionResponse{Code: 0, Msg: "common.success", Session: mapChatSession(session), Messages: mapChatMessages([]ChatMessageRow{message})}, nil
}

func (s *nativeAIService) UpdateSession(ctx context.Context, req *pb.UpdateSessionRequest) (*pb.CommonResponse, error) {
	if s.store == nil {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	if err := s.store.UpdateChatSessionTitle(ctx, ownerRoleHR, req.GetHrId(), req.GetSessionId(), req.GetTitle()); err != nil {
		return nil, err
	}
	return commonOK(), nil
}

func (s *nativeAIService) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.CommonResponse, error) {
	if s.store == nil {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	if err := s.store.DeleteChatSession(ctx, ownerRoleHR, req.GetHrId(), req.GetSessionId()); err != nil {
		return nil, err
	}
	return commonOK(), nil
}

// CandidateChatStream is implemented in native_candidate_chat.go (DEV ADK parity).

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

func (s *nativeAIService) bestEffortRecordHRUsageAudit(ctx context.Context, req *pb.ChatRequest, opts hrChatRuntimeOptions, streaming bool, result hrChatRuntimeResult, statusValue, errorCode string, startedAt time.Time) {
	if err := s.recordHRUsageAudit(ctx, req, opts, streaming, result, statusValue, errorCode, startedAt); err != nil {
		// Audit must not fail the user-facing AI response (aligned with legacy createUsageLogSync warn-only).
		_ = err
	}
}

func (s *nativeAIService) recordHRUsageAudit(ctx context.Context, req *pb.ChatRequest, opts hrChatRuntimeOptions, streaming bool, result hrChatRuntimeResult, statusValue, errorCode string, startedAt time.Time) error {
	if req == nil {
		return nil
	}
	auditStore, ok := s.store.(usageAuditStore)
	if !ok {
		return nil
	}
	if statusValue == "" {
		statusValue = "ok"
	}
	endpoint := "/hr/ai/chat"
	if opts.agentRunID > 0 {
		endpoint = "/hr/ai/agent-run"
	} else if streaming {
		endpoint = "/hr/ai/chat/stream"
	}
	requestChars := len([]rune(strings.TrimSpace(req.GetMessage())))
	responseChars := len([]rune(result.reply))
	tokenTotal := tokenUsageTotal(result.billingTokenUsage)
	if tokenTotal <= 0 && result.contextUsage != nil {
		tokenTotal = int(result.contextUsage.GetTotalTokensActual())
		if tokenTotal <= 0 {
			tokenTotal = int(result.contextUsage.GetPromptTokensActual() + result.contextUsage.GetCompletionTokensActual())
		}
		if tokenTotal <= 0 {
			tokenTotal = int(result.contextUsage.GetPromptTokensEstimated())
		}
	}
	provider := strings.TrimSpace(result.providerName)
	if provider == "" {
		provider = "unknown"
	}
	modelName := strings.TrimSpace(result.modelName)
	row := UsageAuditRow{
		UserID:          req.GetHrId(),
		Role:            2,
		AccountType:     "staff",
		ServiceType:     "ai_chat",
		Endpoint:        endpoint,
		Provider:        provider,
		Model:           modelName,
		RequestChars:    requestChars,
		ResponseChars:   responseChars,
		EstimatedTokens: estimateTokenUsage(requestChars, responseChars),
		TokenUsageTotal: tokenTotal,
		Status:          statusValue,
		ErrorCode:       errorCode,
		CostMs:          int(time.Since(startedAt).Milliseconds()),
		RequestID:       platformmetadata.GetRequestID(ctx),
		IP:              platformmetadata.GetClientIP(ctx),
		RoleKeys:        []string{"staff"},
		PermissionKey:   "ai.hr.use",
		ResourceType:    "ai",
		ResourceID:      req.GetApplicationId(),
	}
	if result.billingTokenUsage != nil {
		row.PromptTokens = result.billingTokenUsage.PromptTokens
		row.CompletionTokens = result.billingTokenUsage.CompletionTokens
	}
	_, err := auditStore.RecordUsageAudit(ctx, row)
	if result.providerInvoked {
		s.bestEffortMeterUsage(ctx, row)
	} else {
		s.cancelUnsettledBilling(ctx, "hr_runtime_completed_without_provider_call")
	}
	return err
}

func cloneTokenUsage(usage *schema.TokenUsage) *schema.TokenUsage {
	if usage == nil {
		return nil
	}
	cloned := *usage
	return &cloned
}

func tokenUsageTotal(usage *schema.TokenUsage) int {
	if usage == nil {
		return 0
	}
	if usage.TotalTokens > 0 {
		return usage.TotalTokens
	}
	return usage.PromptTokens + usage.CompletionTokens
}

type candidateUsageAuditOptions struct {
	Provider         string
	Model            string
	TokenUsageTotal  int
	PromptTokens     int
	CompletionTokens int
}

func (s *nativeAIService) recordCandidateUsageAudit(ctx context.Context, userID int64, requestChars, responseChars int, statusValue, errorCode string, costMs int, opts ...candidateUsageAuditOptions) error {
	if statusValue == "" {
		statusValue = "ok"
	}
	provider := "openai_compatible"
	modelName := ""
	tokenTotal := 0
	promptTokens := 0
	completionTokens := 0
	if len(opts) > 0 {
		if strings.TrimSpace(opts[0].Provider) != "" {
			provider = strings.TrimSpace(opts[0].Provider)
		}
		modelName = strings.TrimSpace(opts[0].Model)
		tokenTotal = opts[0].TokenUsageTotal
		promptTokens = opts[0].PromptTokens
		completionTokens = opts[0].CompletionTokens
	}
	estimated := estimateTokenUsage(requestChars, responseChars)
	if tokenTotal <= 0 {
		tokenTotal = estimated
	}
	row := CandidateUsageAuditRow{
		UserID:           userID,
		ServiceType:      "ai_chat",
		Endpoint:         "/candidate/ai/chat/stream",
		Provider:         provider,
		Model:            modelName,
		RequestChars:     requestChars,
		ResponseChars:    responseChars,
		EstimatedTokens:  estimated,
		TokenUsageTotal:  tokenTotal,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		Status:           statusValue,
		ErrorCode:        errorCode,
		CostMs:           costMs,
		RequestID:        platformmetadata.GetRequestID(ctx),
		IP:               platformmetadata.GetClientIP(ctx),
		RoleKeys:         []string{"candidate"},
		PermissionKey:    "ai.candidate.use",
		ScopeKeys:        []string{"self"},
	}
	if auditStore, ok := s.store.(usageAuditStore); ok {
		_, err := auditStore.RecordUsageAudit(ctx, candidateUsageAuditToUsageAudit(row))
		s.bestEffortMeterUsage(ctx, candidateUsageAuditToUsageAudit(row))
		return err
	}
	auditStore, ok := s.store.(candidateUsageAuditStore)
	if !ok {
		return nil
	}
	_, err := auditStore.RecordCandidateUsageAudit(ctx, row)
	return err
}

func tokenUsageTotalFromMeta(usage *schema.TokenUsage) int {
	if usage == nil {
		return 0
	}
	if usage.TotalTokens > 0 {
		return usage.TotalTokens
	}
	return usage.PromptTokens + usage.CompletionTokens
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

func extractHRSuggestedQuestions(reply string) (string, []string) {
	raw := strings.TrimSpace(reply)
	start := strings.Index(raw, commonsai.HRSuggestedQuestionsStartMarker)
	if start < 0 {
		return raw, nil
	}
	cleanReply := strings.TrimSpace(raw[:start])
	rest := raw[start+len(commonsai.HRSuggestedQuestionsStartMarker):]
	jsonText := rest
	if end := strings.Index(rest, commonsai.HRSuggestedQuestionsEndMarker); end >= 0 {
		jsonText = rest[:end]
		after := strings.TrimSpace(rest[end+len(commonsai.HRSuggestedQuestionsEndMarker):])
		if after != "" {
			cleanReply = strings.TrimSpace(cleanReply + "\n\n" + after)
		}
	}
	return cleanReply, parseHRSuggestedQuestionsJSON(jsonText)
}

func parseHRSuggestedQuestionsJSON(content string) []string {
	var values []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &values); err != nil {
		return nil
	}
	if len(values) != 3 {
		return nil
	}
	return values
}

func normalizeHRSuggestedQuestions(values, fallback []string, forbidden ...string) []string {
	normalized := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	for _, value := range values {
		question := strings.TrimSpace(value)
		key := strings.ToLower(question)
		if question == "" || len([]rune(question)) > 60 {
			return append([]string(nil), fallback...)
		}
		if _, ok := seen[key]; ok {
			return append([]string(nil), fallback...)
		}
		if domainmemory.ClassifyPIILevel(question) == domainmemory.PIILevelHigh {
			return append([]string(nil), fallback...)
		}
		for _, blocked := range forbidden {
			blocked = strings.TrimSpace(blocked)
			if blocked != "" && strings.Contains(strings.ToLower(question), strings.ToLower(blocked)) {
				return append([]string(nil), fallback...)
			}
		}
		seen[key] = struct{}{}
		normalized = append(normalized, question)
	}
	if len(normalized) != 3 {
		return append([]string(nil), fallback...)
	}
	return normalized
}

// hrSuggestionStreamFilter suppresses the model-only suggested-question block
// from user-visible streaming deltas, including when a marker spans chunks.
type hrSuggestionStreamFilter struct {
	onDelta     func(string) error
	buffer      string
	suppressing bool
}

func newHRSuggestionStreamFilter(onDelta func(string) error) *hrSuggestionStreamFilter {
	return &hrSuggestionStreamFilter{onDelta: onDelta}
}

func (f *hrSuggestionStreamFilter) Write(delta string) error {
	if delta == "" || f.onDelta == nil || f.suppressing {
		return nil
	}
	f.buffer += delta
	if markerIndex := strings.Index(f.buffer, commonsai.HRSuggestedQuestionsStartMarker); markerIndex >= 0 {
		visible := f.buffer[:markerIndex]
		f.buffer = ""
		f.suppressing = true
		if visible != "" {
			return f.onDelta(visible)
		}
		return nil
	}
	keep := longestSuffixMatchingPrefix(f.buffer, commonsai.HRSuggestedQuestionsStartMarker)
	flushLen := len(f.buffer) - keep
	if flushLen <= 0 {
		return nil
	}
	visible := f.buffer[:flushLen]
	f.buffer = f.buffer[flushLen:]
	return f.onDelta(visible)
}

func (f *hrSuggestionStreamFilter) Finish() error {
	if f.onDelta == nil || f.suppressing || f.buffer == "" {
		f.buffer = ""
		return nil
	}
	visible := f.buffer
	f.buffer = ""
	return f.onDelta(visible)
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

func agentConfigName(agent *pb.AgentConfigInfo) string {
	if agent == nil {
		return ""
	}
	return agent.GetName()
}

func promptTemplateID(template *pb.PromptTemplateInfo) int64 {
	if template == nil {
		return 0
	}
	return template.GetId()
}

func promptTemplateVersion(template *pb.PromptTemplateInfo) int32 {
	if template == nil {
		return 0
	}
	return template.GetVersion()
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
	rows, total, err := s.listCandidateSessions(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.ChatSessionListResponse{Code: 0, Msg: "common.success", Total: total, List: mapChatSessions(rows)}, nil
}

func (s *nativeAIService) CandidateCreateSession(ctx context.Context, req *pb.CandidateCreateSessionRequest) (*pb.CreateChatSessionResponse, error) {
	session, err := s.ensureSessionWithOptions(ctx, ownerRoleCandidate, req.GetUserId(), 0, 0, req.GetTitle(), ChatSessionCreateOptions{
		SessionType: req.GetSessionType(),
		SourceType:  req.GetSourceType(),
		SourceID:    req.GetSourceId(),
		SourceTitle: req.GetSourceTitle(),
	})
	if err != nil {
		return nil, err
	}
	if initial := strings.TrimSpace(req.GetInitialMessage()); initial != "" {
		if _, err := s.store.AppendChatMessage(ctx, ChatMessageRow{
			OwnerRole: ownerRoleCandidate, OwnerID: req.GetUserId(), SessionID: session.ID,
			Role: "user", Content: initial, CreatedAt: time.Now(),
		}); err != nil {
			return nil, err
		}
	}
	return &pb.CreateChatSessionResponse{Code: 0, Msg: "common.success", Session: mapChatSession(session)}, nil
}

func (s *nativeAIService) CandidateSessionMessages(ctx context.Context, req *pb.CandidateSessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	rows, err := s.listMessages(ctx, ownerRoleCandidate, req.GetUserId(), req.GetSessionId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: 0, Msg: "common.success", List: mapChatMessages(rows)}, nil
}

func (s *nativeAIService) CandidateUpdateSession(ctx context.Context, req *pb.CandidateUpdateSessionRequest) (*pb.CommonResponse, error) {
	if s.missingStore() {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	if err := s.store.UpdateChatSessionTitle(ctx, ownerRoleCandidate, req.GetUserId(), req.GetSessionId(), req.GetTitle()); err != nil {
		return nil, err
	}
	return commonOK(), nil
}

func (s *nativeAIService) CandidateDeleteSession(ctx context.Context, req *pb.CandidateDeleteSessionRequest) (*pb.CommonResponse, error) {
	if s.missingStore() {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	if err := s.store.DeleteChatSession(ctx, ownerRoleCandidate, req.GetUserId(), req.GetSessionId()); err != nil {
		return nil, err
	}
	return commonOK(), nil
}

func (s *nativeAIService) listCandidateSessions(ctx context.Context, req *pb.CandidateSessionListRequest) ([]ChatSessionRow, int64, error) {
	if s.store == nil {
		return nil, 0, errAIStoreRequired
	}
	filter := ChatSessionListFilter{
		Keyword:     req.GetKeyword(),
		SessionType: req.GetSessionType(),
		SourceType:  req.GetSourceType(),
		SourceID:    req.GetSourceId(),
	}
	if enhanced, ok := s.store.(enhancedChatSessionStore); ok {
		return enhanced.ListChatSessionsWithFilter(ctx, ownerRoleCandidate, req.GetUserId(), req.GetPage(), req.GetPageSize(), filter)
	}
	return s.listSessions(ctx, ownerRoleCandidate, req.GetUserId(), req.GetPage(), req.GetPageSize())
}

func (s *nativeAIService) GetToolTraces(ctx context.Context, req *pb.GetToolTracesRequest) (*pb.GetToolTracesResponse, error) {
	if s.store == nil {
		return &pb.GetToolTracesResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	rows, err := s.store.ListToolTraces(ctx, req.GetHrId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	items := make([]*pb.ToolTraceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.ToolTraceItem{Id: row.ID, SessionId: row.SessionID, ToolName: row.ToolName, ArgsJson: row.ArgsJSON, ResultContent: row.ResultContent, DurationMs: row.DurationMs, ErrorMsg: row.ErrorMsg, CreatedAt: formatTime(row.CreatedAt)})
	}
	return &pb.GetToolTracesResponse{Code: 0, Msg: "common.success", List: items}, nil
}

func (s *nativeAIService) GetAgentRuns(ctx context.Context, req *pb.GetAgentRunsRequest) (*pb.GetAgentRunsResponse, error) {
	if s.store == nil {
		return &pb.GetAgentRunsResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
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
	return &pb.GetAgentRunsResponse{Code: 0, Msg: "common.success", List: items}, nil
}

func (s *nativeAIService) CreateAgentRun(ctx context.Context, req *pb.CreateAgentRunRequest) (*pb.CreateAgentRunResponse, error) {
	if req == nil {
		return nil, errors.New("create agent run request is required")
	}
	req.Message = strings.TrimSpace(req.GetMessage())
	if req.GetMessage() == "" {
		return nil, status.Error(codes.InvalidArgument, "agent run message is required")
	}
	if s.store == nil {
		return &pb.CreateAgentRunResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	payload := agentRunPayloadFromCreateRequest(req)
	payload.AuthUserID = platformmetadata.GetAuthUserID(ctx)
	payload.AuthAccountType = platformmetadata.GetAuthAccountType(ctx)
	tenantActor := platformmetadata.GetTenantContext(ctx)
	payload.AuthTenantID = tenantActor.TenantID
	payload.AuthMembershipID = tenantActor.MembershipID
	payload.AuthClientApp = tenantActor.ClientApp
	if payload.AuthUserID <= 0 {
		payload.AuthUserID = req.GetHrId()
	}
	if strings.TrimSpace(payload.AuthAccountType) == "" {
		payload.AuthAccountType = "staff"
	}
	governance, err := s.loadHRRuntimeGovernance(withoutAgentSkillMetrics(ctx), &pb.ChatRequest{
		HrId:                 req.GetHrId(),
		SessionId:            req.GetSessionId(),
		Message:              req.GetMessage(),
		ApplicationId:        req.GetApplicationId(),
		ModelId:              req.GetModelId(),
		CapabilityKeys:       append([]string(nil), req.GetCapabilityKeys()...),
		AgentSkillVersionIds: append([]int64(nil), req.GetAgentSkillVersionIds()...),
	})
	if err != nil {
		return nil, err
	}
	payload.EffectiveAgentID = agentConfigID(governance.Agent)
	payload.EffectiveAgentPinned = true
	initialRun := fallbackAgentRun(req.GetHrId(), req.GetSessionId(), req.GetClientRequestId(), payload)
	initialRun.TenantID = tenantActor.TenantID
	applyHRGovernanceToAgentRun(&initialRun, governance)
	run, idempotent, err := s.store.CreateAgentRun(ctx, initialRun)
	if err != nil {
		return nil, err
	}
	if idempotent {
		return &pb.CreateAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run), IdempotentReplay: true}, nil
	}
	created, err := s.appendAgentRunEvent(ctx, run.ID, "run.created", `{"status":"queued"}`)
	if err != nil {
		return nil, err
	}
	if created.Seq > 0 {
		run.LastEventSeq = created.Seq
	}
	s.dispatchAgentRun(run)
	return &pb.CreateAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run), IdempotentReplay: false}, nil
}

func (s *nativeAIService) GetAgentRun(ctx context.Context, req *pb.GetAgentRunRequest) (*pb.GetAgentRunResponse, error) {
	run, found, err := s.getRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.GetAgentRunResponse{Code: 404, Msg: "common.operation_failed"}, nil
	}
	run, err = s.recoverApprovedAgentSkillHandoff(ctx, run, time.Now())
	if err != nil {
		return nil, err
	}
	if expired, transitioned, expireErr := s.expirePendingAgentRunConfirmation(ctx, run, time.Now()); expireErr != nil {
		return nil, expireErr
	} else if transitioned {
		run = expired
	}
	return &pb.GetAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run)}, nil
}

func (s *nativeAIService) GetActiveAgentRun(ctx context.Context, req *pb.GetActiveAgentRunRequest) (*pb.GetActiveAgentRunResponse, error) {
	if s.store == nil {
		return &pb.GetActiveAgentRunResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	run, found, err := s.store.GetActiveAgentRun(ctx, req.GetHrId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	if found {
		run, err = s.recoverApprovedAgentSkillHandoff(ctx, run, time.Now())
		if err != nil {
			return nil, err
		}
		if expired, transitioned, expireErr := s.expirePendingAgentRunConfirmation(ctx, run, time.Now()); expireErr != nil {
			return nil, expireErr
		} else if transitioned {
			run = expired
			found = false
		}
	}
	if found && s.agentRunExceededDeadline(run, time.Now()) {
		if err := s.ensureAgentRunTerminal(run, context.DeadlineExceeded); err != nil {
			return nil, err
		}
		terminal, terminalFound, err := s.getRun(ctx, run.OwnerID, run.ID)
		if err != nil {
			return nil, err
		}
		if terminalFound {
			run = terminal
		}
		found = false
	}
	return &pb.GetActiveAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run), HasActiveRun: found}, nil
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
		return &pb.CancelAgentRunResponse{Code: 404, Msg: "common.operation_failed"}, nil
	}

	if isTerminalAgentRunStatus(run.Status) {
		return &pb.CancelAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run)}, nil
	}
	if run.Status == agentRunStatusCancelRequested &&
		agentRunPayloadFromRow(run).AgentSkillApproval == nil {
		return &pb.CancelAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run)}, nil
	}
	if run.Status != agentRunStatusCancelRequested && !isCancelableAgentRunStatus(run.Status) {
		return &pb.CancelAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
	}

	if lockErr := lockMutexWithContext(ctx, &s.runTransitionMu); lockErr != nil {
		return nil, status.FromContextError(lockErr).Err()
	}
	defer s.runTransitionMu.Unlock()
	// Progress writes and worker claims use the same exact status/plan CAS from
	// other replicas. Reload after a miss so cancellation is rebuilt from the
	// winning snapshot without ever falling back to an unconditional update.
	for attempt := 0; attempt < agentRunCancelTransitionMaxAttempts; attempt++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		run, found, err = s.getRun(ctx, req.GetHrId(), req.GetRunId())
		if err != nil {
			return nil, err
		}
		if !found {
			return &pb.CancelAgentRunResponse{Code: 404, Msg: "common.operation_failed"}, nil
		}
		if isTerminalAgentRunStatus(run.Status) {
			return &pb.CancelAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run)}, nil
		}
		if run.Status == agentRunStatusCancelRequested &&
			agentRunPayloadFromRow(run).AgentSkillApproval == nil {
			return &pb.CancelAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run)}, nil
		}
		if run.Status != agentRunStatusCancelRequested && !isCancelableAgentRunStatus(run.Status) {
			return &pb.CancelAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
		}

		payload := agentRunPayloadFromRow(run)
		if run.Status == agentRunStatusWaitingConfirmation ||
			run.Status == agentRunStatusQueued ||
			run.Status == agentRunStatusPlanning {
			var pendingSkill *agentRunSkillConfirmation
			nextPlanJSON := run.PlanJSON
			nextOptionContextJSON := run.OptionContextJSON
			if run.Status == agentRunStatusWaitingConfirmation && payload.PendingAgentSkillConfirmation != nil {
				pending := *payload.PendingAgentSkillConfirmation
				pendingSkill = &pending
				payload.PendingAgentSkillConfirmation = nil
				nextPlanJSON = agentRunPlanJSON(payload)
				nextOptionContextJSON = agentRunSkillResolutionContextJSON(pending, "canceled")
			}
			canceled, transitioned, transitionErr := s.transitionAgentRunState(
				ctx,
				run,
				run.Status,
				agentRunStatusCanceled,
				nextPlanJSON,
				nextOptionContextJSON,
				"",
				"",
			)
			if transitionErr != nil {
				return nil, transitionErr
			}
			if !transitioned {
				continue
			}
			s.cancelAgentRunExecution(req.GetRunId())
			if _, eventErr := s.appendAgentRunEvent(ctx, canceled.ID, "run.status_changed", fmt.Sprintf(`{"status":%q}`, agentRunStatusCanceled)); eventErr != nil {
				return nil, eventErr
			}
			_, _ = s.appendAgentRunEvent(ctx, canceled.ID, "run.canceled", marshalJSONString(map[string]any{
				"status": agentRunStatusCanceled,
			}))
			if pendingSkill != nil {
				s.recordAgentSkillConfirmation(
					withAgentSkillExecutionMode(ctx, true),
					pendingSkill.SelectionMode,
					pendingSkill.Role,
					pendingSkill.Risk,
					"failed",
					"canceled",
				)
			}
			return &pb.CancelAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(canceled)}, nil
		}
		if run.Status == agentRunStatusCancelRequested {
			if s.cancelAgentRunExecution(req.GetRunId()) {
				return &pb.CancelAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run)}, nil
			}
			canceled, transitioned, transitionErr := s.transitionAgentRunState(
				ctx,
				run,
				agentRunStatusCancelRequested,
				agentRunStatusCanceled,
				run.PlanJSON,
				run.OptionContextJSON,
				"",
				"",
			)
			if transitionErr != nil {
				return nil, transitionErr
			}
			if !transitioned {
				continue
			}
			_, _ = s.appendAgentRunEvent(ctx, canceled.ID, "run.canceled", marshalJSONString(map[string]any{
				"status": agentRunStatusCanceled,
			}))
			return &pb.CancelAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(canceled)}, nil
		}

		cancelRequested, transitioned, transitionErr := s.transitionAgentRunState(
			ctx,
			run,
			agentRunStatusRunning,
			agentRunStatusCancelRequested,
			run.PlanJSON,
			run.OptionContextJSON,
			"",
			"",
		)
		if transitionErr != nil {
			return nil, transitionErr
		}
		if !transitioned {
			continue
		}
		activeExecution := s.cancelAgentRunExecution(req.GetRunId())
		if _, eventErr := s.appendAgentRunEvent(ctx, cancelRequested.ID, "run.status_changed", fmt.Sprintf(`{"status":%q}`, agentRunStatusCancelRequested)); eventErr != nil {
			return nil, eventErr
		}
		if activeExecution {
			return &pb.CancelAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(cancelRequested)}, nil
		}
		canceled, finalized, finalizeErr := s.transitionAgentRunState(
			ctx,
			cancelRequested,
			agentRunStatusCancelRequested,
			agentRunStatusCanceled,
			cancelRequested.PlanJSON,
			cancelRequested.OptionContextJSON,
			"",
			"",
		)
		if finalizeErr != nil {
			return nil, finalizeErr
		}
		if !finalized {
			continue
		}
		_, _ = s.appendAgentRunEvent(ctx, canceled.ID, "run.canceled", marshalJSONString(map[string]any{
			"status": agentRunStatusCanceled,
		}))
		return &pb.CancelAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(canceled)}, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	return nil, status.Error(codes.Aborted, "agent run cancellation conflicted; retry")
}

func lockMutexWithContext(ctx context.Context, mutex *sync.Mutex) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if mutex.TryLock() {
			if ctxErr := ctx.Err(); ctxErr != nil {
				mutex.Unlock()
				return ctxErr
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *nativeAIService) ConfirmAgentRun(ctx context.Context, req *pb.ConfirmAgentRunRequest) (*pb.ConfirmAgentRunResponse, error) {
	if req == nil {
		return nil, errors.New("confirm agent run request is required")
	}
	hasAgentSkillDecision := strings.TrimSpace(req.GetAgentSkillConfirmationId()) != "" ||
		req.GetAgentSkillConfirmationDecision() != pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_UNSPECIFIED ||
		len(req.GetSelectedAgentSkillVersionIds()) > 0
	run, found, err := s.getRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.ConfirmAgentRunResponse{Code: 404, Msg: "common.operation_failed"}, nil
	}
	s.runTransitionMu.Lock()
	defer s.runTransitionMu.Unlock()
	run, found, err = s.getRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.ConfirmAgentRunResponse{Code: 404, Msg: "common.operation_failed"}, nil
	}
	payload := agentRunPayloadFromRow(run)
	if payload.AgentSkillApproval != nil && hasAgentSkillDecision {
		if validateErr := validateApprovedAgentSkillRetry(ctx, run, payload, req); validateErr != nil {
			return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
		}
		if validateErr := s.validateAgentRunSkillMessageBinding(
			ctx,
			run,
			payload.AgentSkillApproval.UserMessageID,
			payload.AgentSkillApproval.MessageDigest,
		); validateErr != nil {
			return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
		}
		run, err = s.recoverStaleAgentSkillDispatchLocked(ctx, run, time.Now())
		if err != nil {
			return nil, err
		}
		run, err = s.finalizeApprovedAgentSkillHandoff(ctx, run)
		if err != nil {
			return nil, err
		}
		if run.Status == agentRunStatusQueued {
			s.dispatchAgentRun(run)
		}
		return &pb.ConfirmAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run)}, nil
	}
	if run.Status == agentRunStatusRunning || isTerminalAgentRunStatus(run.Status) {
		if hasAgentSkillDecision {
			return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
		}
		return &pb.ConfirmAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run)}, nil
	}
	if run.Status != agentRunStatusWaitingConfirmation {
		return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
	}
	payload = agentRunPayloadFromRow(run)
	if payload.PendingAgentSkillConfirmation != nil {
		if payload.PendingMCPConfirmation != nil || !hasAgentSkillDecision {
			return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
		}
		now := time.Now()
		if validateErr := validateAgentRunSkillConfirmation(ctx, run, payload, req, now); validateErr != nil {
			if errors.Is(validateErr, errExpiredAgentSkillConfirmation) {
				expired, transitioned, transitionErr := s.transitionAgentRunConfirmation(
					ctx,
					run,
					agentRunStatusFailed,
					run.PlanJSON,
					agentRunSkillResolutionContextJSON(*payload.PendingAgentSkillConfirmation, "expired"),
					"AGENT_SKILL_CONFIRMATION_EXPIRED",
					"ai.agent_skill_confirmation_expired",
				)
				if transitionErr != nil {
					return nil, transitionErr
				}
				if transitioned {
					_, _ = s.appendAgentRunEvent(ctx, expired.ID, "run.status_changed", marshalJSONString(map[string]any{
						"status":        agentRunStatusFailed,
						"error_type":    "AGENT_SKILL_CONFIRMATION_EXPIRED",
						"error_message": "ai.agent_skill_confirmation_expired",
					}))
					s.recordAgentSkillConfirmation(
						withAgentSkillExecutionMode(ctx, true),
						payload.PendingAgentSkillConfirmation.SelectionMode,
						payload.PendingAgentSkillConfirmation.Role,
						payload.PendingAgentSkillConfirmation.Risk,
						"failed",
						"expired",
					)
					run = expired
				}
			}
			return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
		}
		pending := *payload.PendingAgentSkillConfirmation
		if req.GetAgentSkillConfirmationDecision() == pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_REJECT {
			payload.PendingAgentSkillConfirmation = nil
			rejected, transitioned, transitionErr := s.transitionAgentRunConfirmation(
				ctx,
				run,
				agentRunStatusCanceled,
				agentRunPlanJSON(payload),
				agentRunSkillResolutionContextJSON(pending, "rejected"),
				"",
				"",
			)
			if transitionErr != nil {
				return nil, transitionErr
			}
			if !transitioned {
				return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
			}
			_, _ = s.appendAgentRunEvent(ctx, rejected.ID, "confirmation.rejected", marshalJSONString(map[string]any{
				"status": agentRunStatusCanceled,
			}))
			_, _ = s.appendAgentRunEvent(ctx, rejected.ID, "run.canceled", marshalJSONString(map[string]any{
				"status": agentRunStatusCanceled,
			}))
			s.recordAgentSkillConfirmation(
				withAgentSkillExecutionMode(ctx, true),
				pending.SelectionMode,
				pending.Role,
				pending.Risk,
				"failed",
				"rejected",
			)
			return &pb.ConfirmAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(rejected)}, nil
		}
		if _, validateErr := s.revalidateAgentRunSkillBinding(ctx, run, payload); validateErr != nil {
			return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
		}
		payload = approvedAgentRunSkillPayload(
			payload,
			pending,
			req.GetSelectedAgentSkillVersionIds(),
			req.GetClientRequestId(),
			now,
		)
		run, found, err = s.transitionAgentRunConfirmation(
			ctx,
			run,
			agentRunStatusQueued,
			agentRunPlanJSON(payload),
			agentRunSkillResolutionContextJSON(pending, "approved"),
			"",
			"",
		)
		if err != nil {
			return nil, err
		}
		if !found {
			return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
		}
		run, err = s.finalizeApprovedAgentSkillHandoff(ctx, run)
		if err != nil {
			return nil, err
		}
		s.dispatchAgentRun(run)
		return &pb.ConfirmAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run)}, nil
	}
	if hasAgentSkillDecision {
		return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
	}
	run, found, err = s.updateAgentRunConfirmation(ctx, run, req)
	if err != nil {
		if errors.Is(err, errInvalidMCPConfirmation) {
			return &pb.ConfirmAgentRunResponse{Code: agentRunCodeBadRequest, Msg: "common.invalid_request", Run: mapAgentRunSnapshot(run)}, nil
		}
		return nil, err
	}
	if !found {
		return &pb.ConfirmAgentRunResponse{Code: 404, Msg: "common.operation_failed"}, nil
	}
	run, found, err = s.updateRun(ctx, req.GetHrId(), req.GetRunId(), agentRunStatusRunning)
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.ConfirmAgentRunResponse{Code: 404, Msg: "common.operation_failed"}, nil
	}
	if _, err := s.appendAgentRunEvent(ctx, run.ID, "confirmation.accepted", agentRunConfirmationAcceptedPayload(req)); err != nil {
		return nil, err
	}
	s.dispatchAgentRun(run)
	return &pb.ConfirmAgentRunResponse{Code: 0, Msg: "common.success", Run: mapAgentRunSnapshot(run)}, nil
}

func (s *nativeAIService) updateAgentRunConfirmation(ctx context.Context, run AgentRunRow, req *pb.ConfirmAgentRunRequest) (AgentRunRow, bool, error) {
	if s.store == nil {
		return AgentRunRow{}, false, errAIStoreRequired
	}
	payload := agentRunPayloadFromRow(run)
	if payload.PendingMCPConfirmation != nil {
		approval, err := validateAgentRunMCPConfirmation(req.GetConfirmationPayloadJson(), *payload.PendingMCPConfirmation, time.Now())
		if err != nil {
			return run, true, err
		}
		payload.MCPApproval = approval
		payload.PendingMCPConfirmation = nil
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

func validateAgentRunMCPConfirmation(rawJSON string, pending agentRunMCPConfirmation, now time.Time) (*agentRunMCPApproval, error) {
	var submitted mcpConfirmationPayload
	if strings.TrimSpace(rawJSON) == "" || json.Unmarshal([]byte(rawJSON), &submitted) != nil {
		return nil, errInvalidMCPConfirmation
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, pending.ExpiresAt)
	if err != nil || !now.Before(expiresAt) {
		return nil, errInvalidMCPConfirmation
	}
	if submitted.Type != "mcp_tool" ||
		!submitted.Approved ||
		submitted.ConfirmationID != pending.ID ||
		submitted.CapabilityKey != pending.CapabilityKey ||
		submitted.ArgumentsHash != pending.ArgumentsHash {
		return nil, errInvalidMCPConfirmation
	}
	return &agentRunMCPApproval{
		ConfirmationID: pending.ID,
		CapabilityKey:  pending.CapabilityKey,
		ArgumentsHash:  pending.ArgumentsHash,
		ApprovedAt:     formatTime(now),
	}, nil
}

func (s *nativeAIService) ensureSession(ctx context.Context, ownerRole int32, ownerID, sessionID, applicationID int64, seed string) (ChatSessionRow, error) {
	return s.ensureSessionWithOptions(ctx, ownerRole, ownerID, sessionID, applicationID, seed, ChatSessionCreateOptions{})
}

func (s *nativeAIService) ensureSessionWithOptions(ctx context.Context, ownerRole int32, ownerID, sessionID, applicationID int64, seed string, opts ChatSessionCreateOptions) (ChatSessionRow, error) {
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
	if enhanced, ok := s.store.(enhancedChatSessionStore); ok {
		return enhanced.EnsureChatSessionWithOptions(ctx, ownerRole, ownerID, title, applicationID, opts)
	}
	return s.store.EnsureChatSession(ctx, ownerRole, ownerID, title, applicationID)
}

func (s *nativeAIService) missingStore() bool {
	return s == nil || s.store == nil
}

func (s *nativeAIService) complete(ctx context.Context, prompt string, modelID int64, opts ...ChatCompletionOptions) (string, error) {
	result, err := s.completeWithUsage(ctx, prompt, modelID, opts...)
	return result.Content, err
}

func (s *nativeAIService) completeWithUsage(ctx context.Context, prompt string, modelID int64, opts ...ChatCompletionOptions) (commonsai.GenerateResult, error) {
	if s.provider == nil {
		return commonsai.GenerateResult{}, errAIProviderRequired
	}
	var completionOpts ChatCompletionOptions
	if len(opts) > 0 {
		completionOpts = opts[0]
	}
	if usageAware, ok := s.provider.(UsageAwareRuntimeOptionsChatProvider); ok {
		result, err := usageAware.CompleteWithOptionsAndUsage(ctx, prompt, modelID, completionOpts)
		if err != nil {
			return commonsai.GenerateResult{}, err
		}
		if strings.TrimSpace(result.Content) == "" {
			result.Content = "AI provider returned an empty response."
		}
		return result, nil
	}
	if optionsAware, ok := s.provider.(RuntimeOptionsChatProvider); ok && completionOpts.TemperatureOverride != nil {
		reply, err := optionsAware.CompleteWithOptions(ctx, prompt, modelID, completionOpts)
		if err != nil {
			return commonsai.GenerateResult{}, err
		}
		if strings.TrimSpace(reply) == "" {
			reply = "AI provider returned an empty response."
		}
		return commonsai.GenerateResult{Content: reply}, nil
	}
	if modelAware, ok := s.provider.(ModelAwareChatProvider); ok {
		reply, err := modelAware.CompleteWithModel(ctx, prompt, modelID)
		if err != nil {
			return commonsai.GenerateResult{}, err
		}
		if strings.TrimSpace(reply) == "" {
			reply = "AI provider returned an empty response."
		}
		return commonsai.GenerateResult{Content: reply}, nil
	}
	reply, err := s.provider.Complete(ctx, prompt)
	if err != nil {
		return commonsai.GenerateResult{}, err
	}
	if strings.TrimSpace(reply) == "" {
		reply = "AI provider returned an empty response."
	}
	return commonsai.GenerateResult{Content: reply}, nil
}

func (s *nativeAIService) dispatchAgentRun(run AgentRunRow) {
	ctx, cancel := context.WithTimeout(context.Background(), s.effectiveAgentRunTimeout())
	cancelEntry := &agentRunCancelEntry{cancel: cancel}
	go func() {
		if err := s.executeAgentRun(ctx, run, cancelEntry); err != nil {
			if errors.Is(err, errAgentRunExecutionLeaseLost) {
				return
			}
			logger.L().Error("agent run execution failed",
				zap.Int64("run_id", run.ID),
				zap.String("status", run.Status),
				zap.Error(err),
			)
			if terminalErr := s.ensureAgentRunTerminal(run, err); terminalErr != nil {
				logger.L().Error("agent run terminal fallback failed",
					zap.Int64("run_id", run.ID),
					zap.Error(terminalErr),
				)
			}
		}
	}()
}

const defaultAgentRunTimeout = 3 * time.Minute

func (s *nativeAIService) effectiveAgentRunTimeout() time.Duration {
	if s != nil && s.agentRunTimeout > 0 {
		return s.agentRunTimeout
	}
	return defaultAgentRunTimeout
}

func (s *nativeAIService) agentRunExceededDeadline(run AgentRunRow, now time.Time) bool {
	if isTerminalAgentRunStatus(run.Status) {
		return false
	}
	payload := agentRunPayloadFromRow(run)
	if run.Status == agentRunStatusWaitingConfirmation {
		expiresAt := ""
		if payload.PendingAgentSkillConfirmation != nil {
			expiresAt = payload.PendingAgentSkillConfirmation.ExpiresAt
		} else if payload.PendingMCPConfirmation != nil {
			expiresAt = payload.PendingMCPConfirmation.ExpiresAt
		}
		deadline, err := time.Parse(time.RFC3339Nano, expiresAt)
		return err != nil || !now.Before(deadline)
	}
	if approval := payload.AgentSkillApproval; validAgentRunSkillApprovalMarker(run, approval) {
		switch {
		case run.Status == agentRunStatusQueued &&
			(approval.DispatchState == agentSkillDispatchPendingEvent ||
				approval.DispatchState == agentSkillDispatchReady):
			// Confirmation may legitimately be approved after the ordinary Run
			// timeout. The durable handoff is still recoverable until claimed.
			return false
		case run.Status == agentRunStatusRunning && approval.DispatchState == agentSkillDispatchClaimed:
			// Governed execution ownership is defined by the persisted lease,
			// including any later renewal, rather than the original creation time.
			return agentSkillDispatchLeaseExpired(approval, now)
		}
	}
	if run.StartedAt.IsZero() || now.Before(run.StartedAt) {
		return false
	}
	return now.Sub(run.StartedAt) >= s.effectiveAgentRunTimeout()
}

func (s *nativeAIService) executeAgentRun(ctx context.Context, run AgentRunRow, cancelEntry *agentRunCancelEntry) error {
	current, shouldExecute, err := s.beginAgentRunExecution(ctx, run)
	if err != nil || !shouldExecute {
		if cancelEntry != nil && cancelEntry.cancel != nil {
			cancelEntry.cancel()
		}
		return err
	}
	s.storeAgentRunCancel(run.ID, cancelEntry)
	defer s.clearAgentRunCancel(run.ID, cancelEntry)
	payload := agentRunPayloadFromRow(current)
	ctx = agentRunExecutionContext(ctx, current, payload)
	result, err := s.runHRChatRuntimeWithOptions(ctx, &pb.ChatRequest{
		HrId:                 current.OwnerID,
		SessionId:            current.SessionID,
		Message:              payload.Message,
		ApplicationId:        payload.ApplicationID,
		ModelId:              payload.ModelID,
		CapabilityKeys:       payload.CapabilityKeys,
		AgentSkillVersionIds: payload.AgentSkillVersionIDs,
	}, s.agentRunChatEmitterForExecution(current), hrChatRuntimeOptions{
		reuseExistingUserMessage: shouldReuseAgentRunUserMessage(run, payload),
		existingUserMessageID:    current.MessageID,
		agentRunID:               current.ID,
		effectiveAgentID:         payload.EffectiveAgentID,
		effectiveAgentPinned:     payload.EffectiveAgentPinned,
		durablePayload:           payload,
	})
	if err != nil {
		var skillConfirmationErr *agentSkillConfirmationRequiredError
		if errors.As(err, &skillConfirmationErr) {
			return s.finishAgentRunWaitingForSkillConfirmation(ctx, current, skillConfirmationErr)
		}
		var confirmationErr *mcpConfirmationRequiredError
		if errors.As(err, &confirmationErr) {
			return s.finishAgentRunWaitingForMCPConfirmation(ctx, current, confirmationErr.Confirmation)
		}
		if agentRunExecutionTimedOut(ctx, err) {
			return s.finishAgentRunFailed(ctx, current, context.DeadlineExceeded)
		}
		if agentRunExecutionCanceled(ctx, err) {
			return s.finishAgentRunCanceled(ctx, current)
		}
		return s.finishAgentRunFailed(ctx, current, err)
	}
	if agentRunExecutionTimedOut(ctx, nil) {
		return s.finishAgentRunFailed(ctx, current, context.DeadlineExceeded)
	}
	if agentRunExecutionCanceled(ctx, nil) {
		return s.finishAgentRunCanceled(ctx, current)
	}
	if result.providerUnavailable && !result.fallbackUsed {
		return s.finishAgentRunFailed(ctx, current, errAIProviderRequired)
	}
	return s.finishAgentRunSucceeded(ctx, current, result, payload.ModelID)
}

func (s *nativeAIService) finishAgentRunWaitingForMCPConfirmation(ctx context.Context, run AgentRunRow, confirmation agentRunMCPConfirmation) error {
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
	if isTerminalAgentRunStatus(current.Status) || current.Status == agentRunStatusWaitingConfirmation {
		return nil
	}
	payload := agentRunPayloadFromRow(current)
	payload.PendingMCPConfirmation = &confirmation
	payload.MCPApproval = nil
	optionContextJSON := marshalJSONString(map[string]any{
		"confirmation_request": map[string]any{
			"required": true,
			"reason":   confirmation.Reason,
			"raw_json": marshalJSONString(map[string]any{
				"type":            "mcp_tool",
				"confirmation_id": confirmation.ID,
				"capability_key":  confirmation.CapabilityKey,
				"arguments_hash":  confirmation.ArgumentsHash,
				"expires_at":      confirmation.ExpiresAt,
			}),
		},
	})
	if _, updated, updateErr := s.store.UpdateAgentRunPlan(storeCtx, current.OwnerID, current.ID, agentRunPlanJSON(payload), optionContextJSON); updateErr != nil {
		return updateErr
	} else if !updated {
		return fmt.Errorf("agent run not found")
	}
	waiting, updated, updateErr := s.updateRun(storeCtx, current.OwnerID, current.ID, agentRunStatusWaitingConfirmation)
	if updateErr != nil {
		return updateErr
	}
	if !updated {
		return fmt.Errorf("agent run not found")
	}
	eventPayload := marshalJSONString(map[string]any{
		"status": agentRunStatusWaitingConfirmation,
		"confirmation_request": map[string]any{
			"required": true,
			"reason":   confirmation.Reason,
			"raw_json": marshalJSONString(map[string]any{
				"type":            "mcp_tool",
				"confirmation_id": confirmation.ID,
				"capability_key":  confirmation.CapabilityKey,
				"arguments_hash":  confirmation.ArgumentsHash,
				"expires_at":      confirmation.ExpiresAt,
			}),
		},
	})
	_, eventErr := s.appendAgentRunEvent(storeCtx, waiting.ID, "confirmation.required", eventPayload)
	return eventErr
}

func shouldReuseAgentRunUserMessage(run AgentRunRow, payload agentRunDurablePayload) bool {
	if run.Status == agentRunStatusRunning ||
		run.Status == agentRunStatusWaitingConfirmation ||
		(payload.AgentSkillApproval != nil && payload.AgentSkillApproval.UserMessageID > 0) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(payload.ActionType), "analyze_application")
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
		// A running Run has already been claimed by a worker. Executing it from
		// a second dispatch would duplicate provider/tool side effects.
		if agentRunPayloadFromRow(current).AgentSkillApproval != nil {
			return current, false, nil
		}
		return current, true, nil
	}
	if !isExecutableAgentRunStatus(current.Status) {
		return current, false, nil
	}
	payload := agentRunPayloadFromRow(current)
	if approval := payload.AgentSkillApproval; approval != nil {
		if current.Status != agentRunStatusQueued ||
			approval.DispatchState != agentSkillDispatchReady ||
			!validAgentRunSkillApprovalMarker(current, approval) {
			return current, false, errInvalidAgentSkillConfirmation
		}
		leaseDuration := 2*s.effectiveAgentRunTimeout() + agentSkillDispatchLeaseGrace
		claimedPayload := claimedAgentRunSkillPayload(
			payload,
			newAgentSkillDispatchLeaseID(current.ID),
			time.Now().Add(leaseDuration),
		)
		next, claimed, claimErr := s.transitionAgentRunState(
			storeCtx,
			current,
			agentRunStatusQueued,
			agentRunStatusRunning,
			agentRunPlanJSON(claimedPayload),
			current.OptionContextJSON,
			"",
			"",
		)
		return next, claimed, claimErr
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
	if strings.TrimSpace(reply) != "" && !result.streamedTextDelta {
		if _, err := s.appendAgentRunExecutionEvent(storeCtx, run, "assistant.delta", fmt.Sprintf(`{"status":%q,"delta":%q}`, agentRunStatusRunning, reply)); err != nil {
			return err
		}
	}
	if result.plan.Intent != "" {
		payload := agentRunPayloadFromRow(run)
		planJSON := agentRunRuntimePlanJSON(payload, &pb.ChatRequest{
			HrId:          current.OwnerID,
			SessionId:     current.SessionID,
			Message:       payload.Message,
			ApplicationId: payload.ApplicationID,
			ModelId:       payload.ModelID,
		}, result.governance, result.plan, result.modelID, result.modelName, result.runtimeWarnings, s.hrRuntimeLabel())
		if leaseID := agentRunSkillExecutionLease(run); leaseID != "" {
			store, ok := s.store.(agentRunSkillExecutionFenceStore)
			if !ok {
				return errAgentRunExecutionLeaseLost
			}
			if _, owned, err := store.UpdateAgentRunPlanForSkillLease(
				storeCtx,
				run.OwnerID,
				run.ID,
				leaseID,
				planJSON,
				current.OptionContextJSON,
			); err != nil {
				return err
			} else if !owned {
				return errAgentRunExecutionLeaseLost
			}
		} else if _, _, err := s.store.UpdateAgentRunPlan(storeCtx, current.OwnerID, current.ID, planJSON, current.OptionContextJSON); err != nil {
			return err
		}
	}
	processSnapshot := buildAgentRunProcessSnapshot(result.plan, result.toolTraces, result.contextUsage, result.fallbackUsed)
	if _, err := s.appendAgentRunExecutionEvent(storeCtx, run, "process.snapshot", marshalJSONString(map[string]any{
		"status":        agentRunStatusRunning,
		"snapshot_text": processSnapshot,
	})); err != nil {
		return err
	}
	if _, err := s.appendAgentRunExecutionEvent(storeCtx, run, "run.result", agentRunResultPayload(result, s.hrRuntimeLabel())); err != nil {
		return err
	}
	if governanceStore, ok := s.store.(agentRunRuntimeGovernanceStore); ok {
		if leaseID := agentRunSkillExecutionLease(run); leaseID != "" {
			fencedStore, supported := s.store.(agentRunSkillExecutionFenceStore)
			if !supported {
				return errAgentRunExecutionLeaseLost
			}
			if owned, err := fencedStore.UpdateAgentRunRuntimeGovernanceForSkillLease(
				storeCtx,
				run.OwnerID,
				run.ID,
				leaseID,
				result.runtimeModel,
			); err != nil {
				return err
			} else if !owned {
				return errAgentRunExecutionLeaseLost
			}
		} else if err := governanceStore.UpdateAgentRunRuntimeGovernance(storeCtx, run.OwnerID, run.ID, result.runtimeModel); err != nil {
			return err
		}
	}
	resultMetadataJSON := marshalJSONString(agentRunResultMetadata(result, s.hrRuntimeLabel()))
	if _, _, err := s.completeAgentRunExecution(storeCtx, run, reply, agentRunStatusSucceeded, "", "", resultMetadataJSON); err != nil {
		return err
	}
	_, err = s.appendAgentRunExecutionEvent(storeCtx, run, "run.completed", fmt.Sprintf(`{"status":%q}`, agentRunStatusSucceeded))
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
	errorType, errorMessage := agentRunFailureDetails(runErr, "provider")
	if _, eventErr := s.appendAgentRunExecutionEvent(storeCtx, run, "run.error", fmt.Sprintf(`{"status":%q,"error_type":%q,"error_message":%q}`, agentRunStatusFailed, errorType, errorMessage)); eventErr != nil {
		return eventErr
	}
	if _, _, completeErr := s.completeAgentRunExecution(storeCtx, run, "", agentRunStatusFailed, errorType, errorMessage, ""); completeErr != nil {
		return completeErr
	}
	_, eventErr := s.appendAgentRunExecutionEvent(storeCtx, run, "run.completed", fmt.Sprintf(`{"status":%q,"error_type":%q,"error_message":%q}`, agentRunStatusFailed, errorType, errorMessage))
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
	_, found, err := s.completeAgentRunExecution(ctx, run, run.AssistantText, agentRunStatusCanceled, "", "", "")
	if err != nil || !found {
		return err
	}
	_, err = s.appendAgentRunExecutionEvent(ctx, run, "run.canceled", fmt.Sprintf(`{"status":%q}`, agentRunStatusCanceled))
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

func agentRunExecutionTimedOut(ctx context.Context, err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || (ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded))
}

func agentRunFailureDetails(runErr error, defaultType string) (string, string) {
	if strings.TrimSpace(defaultType) == "" {
		defaultType = "runtime"
	}
	if runErr == nil {
		return defaultType, "common.operation_failed"
	}
	if errors.Is(runErr, context.DeadlineExceeded) {
		return "timeout", "ai.stream_timeout"
	}
	if contextCode := hrContextErrorCode(runErr); contextCode != "" {
		return contextCode, hrContextMessageKey(contextCode)
	}
	if status.Code(runErr) == codes.FailedPrecondition &&
		status.Convert(runErr).Message() == "AGENT_SKILL_STRICT_OUTPUT_UNSUPPORTED" {
		return "agent_skill_strict_output_unsupported", "ai.agent_skill_strict_output_unsupported"
	}
	if strings.Contains(strings.ToLower(runErr.Error()), "insufficient_credits") {
		return "insufficient_credits", "ai.insufficient_credits"
	}
	return defaultType, "common.operation_failed"
}

func (s *nativeAIService) ensureAgentRunTerminal(run AgentRunRow, runErr error) error {
	if s == nil || s.store == nil || run.ID <= 0 {
		return runErr
	}
	storeCtx := context.Background()
	current, found, err := s.getRun(storeCtx, run.OwnerID, run.ID)
	if err != nil || !found || isTerminalAgentRunStatus(current.Status) {
		return err
	}
	errorType, errorMessage := agentRunFailureDetails(runErr, "runtime")
	_, found, err = s.completeAgentRunExecution(
		storeCtx,
		run,
		current.AssistantText,
		agentRunStatusFailed,
		errorType,
		errorMessage,
		"",
	)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("agent run %d not found while applying terminal fallback", current.ID)
	}
	_, err = s.appendAgentRunExecutionEvent(storeCtx, run, "run.completed", fmt.Sprintf(`{"status":%q,"error_type":%q,"error_message":%q}`, agentRunStatusFailed, errorType, errorMessage))
	return err
}

func (s *nativeAIService) agentRunChatEmitter(runID int64) hrChatStreamEmitter {
	return s.agentRunChatEmitterForExecution(AgentRunRow{ID: runID})
}

func (s *nativeAIService) agentRunChatEmitterForExecution(run AgentRunRow) hrChatStreamEmitter {
	processDisplay := newAgentRunProcessDisplayState()
	var emitMu sync.Mutex
	return func(event *pb.ChatStreamResponse, display *agentRunDisplayContext) error {
		// Runtime callbacks may arrive concurrently. Keep snapshot mutation and the
		// corresponding durable event append in one critical section so a stale
		// snapshot cannot be persisted after a newer one.
		emitMu.Lock()
		defer emitMu.Unlock()
		if event == nil {
			return nil
		}
		eventType := agentRunEventTypeFromChatEvent(event)
		eventMessage := event.GetEventMessage()
		if event.GetEventType() == "process_delta" || event.GetEventType() == "process_clear" {
			eventMessage = ""
		}
		payload := map[string]any{
			"status":        agentRunStatusRunning,
			"source_event":  event.GetEventType(),
			"event_message": eventMessage,
		}
		if display != nil {
			if display.StepKey != "" {
				payload["step_key"] = display.StepKey
			}
			if display.StepPurpose != "" {
				payload["step_purpose"] = display.StepPurpose
			}
			if display.ToolGroup != "" {
				payload["tool_group"] = display.ToolGroup
			}
		}
		if displayMessage := agentRunDisplayMessage(eventType, event, display); displayMessage != "" {
			payload["display_message"] = displayMessage
			payload["display_source"] = "planner_step"
			if display == nil || display.StepPurpose == "" {
				payload["display_source"] = "runtime_fallback"
			}
			if key := agentRunProcessDisplayKey(eventType, event, display); key != "" {
				if eventType == "tool.started" || eventType == "tool.finished" {
					processDisplay.upsert("planning", "已分析问题并确定所需招聘数据。")
				}
				processDisplay.upsert(key, displayMessage)
				payload["snapshot_text"] = processDisplay.snapshot()
			}
		}
		if event.GetDelta() != "" && eventType == "assistant.delta" {
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
		_, err := s.appendAgentRunExecutionEvent(
			agentRunStoreContext(context.Background()),
			run,
			eventType,
			marshalJSONString(payload),
		)
		return err
	}
}

func displayContextForTool(plan commonsai.RecruitingPlan, toolName string) *agentRunDisplayContext {
	toolName = hr_tools.NormalizeToolName(toolName)
	if toolName == "" {
		return nil
	}
	for _, step := range plan.DisplaySteps {
		for _, tool := range step.Tools {
			if hr_tools.NormalizeToolName(tool) == toolName {
				return &agentRunDisplayContext{
					Plan:        plan,
					StepKey:     step.Key,
					StepPurpose: step.Purpose,
					ToolGroup:   step.ToolGroup,
				}
			}
		}
	}
	return nil
}

func displayContextForPlanStep(plan commonsai.RecruitingPlan, key string) *agentRunDisplayContext {
	for _, step := range plan.DisplaySteps {
		if step.Key == key {
			return &agentRunDisplayContext{
				Plan:        plan,
				StepKey:     step.Key,
				StepPurpose: step.Purpose,
				ToolGroup:   step.ToolGroup,
			}
		}
	}
	return nil
}

func agentRunDisplayMessage(eventType string, event *pb.ChatStreamResponse, display *agentRunDisplayContext) string {
	if event == nil {
		return ""
	}
	purpose := ""
	if display != nil {
		purpose = strings.TrimSpace(display.StepPurpose)
	}
	switch eventType {
	case "tool.started":
		if purpose != "" {
			return "我正在" + purpose + "。"
		}
		return "我正在查询实时招聘数据。"
	case "tool.finished":
		if event.GetErrorType() != "" || strings.TrimSpace(event.GetMsg()) != "success" {
			if purpose != "" {
				return purpose + "暂时没有完成，我会尝试使用其他可用数据继续推进。"
			}
			return "这一步实时数据暂时没有完成，我会尝试使用其他可用数据继续推进。"
		}
		if purpose != "" {
			return "已完成：" + purpose + "。"
		}
		return "已获取一项实时招聘数据。"
	case "process.delta":
		switch event.GetEventType() {
		case "context_usage":
			return ""
		case "thinking":
			return "正在分析问题并确定所需招聘数据。"
		case "fallback":
			return "部分数据暂时不足，正在使用可用信息保守作答。"
		case "generating":
			return "正在整理查询结果并生成回复。"
		}
	case "run.error":
		if purpose != "" {
			return purpose + "过程中出现异常。"
		}
		return "执行过程中出现异常。"
	}
	return ""
}

type agentRunProcessDisplayState struct {
	mu    sync.RWMutex
	order []string
	lines map[string]string
}

func newAgentRunProcessDisplayState() *agentRunProcessDisplayState {
	return &agentRunProcessDisplayState{lines: make(map[string]string)}
}

func (s *agentRunProcessDisplayState) upsert(key, line string) {
	if s == nil {
		return
	}
	key = strings.TrimSpace(key)
	line = strings.TrimSpace(line)
	if key == "" || line == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.lines[key]; !exists {
		s.order = append(s.order, key)
	}
	s.lines[key] = line
}

func (s *agentRunProcessDisplayState) snapshot() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	lines := make([]string, 0, len(s.order))
	seen := make(map[string]struct{}, len(s.order))
	for _, key := range s.order {
		if line := strings.TrimSpace(s.lines[key]); line != "" {
			if _, exists := seen[line]; exists {
				continue
			}
			seen[line] = struct{}{}
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func agentRunProcessDisplayKey(eventType string, event *pb.ChatStreamResponse, display *agentRunDisplayContext) string {
	if display != nil && strings.TrimSpace(display.StepKey) != "" && (eventType == "tool.started" || eventType == "tool.finished") {
		return "step:" + strings.TrimSpace(display.StepKey)
	}
	switch event.GetEventType() {
	case "thinking":
		return "planning"
	case "generating":
		return "compose"
	case "fallback":
		return "fallback"
	}
	if toolName := strings.TrimSpace(event.GetToolName()); toolName != "" && (eventType == "tool.started" || eventType == "tool.finished") {
		return "tool:" + hr_tools.NormalizeToolName(toolName)
	}
	return ""
}

func agentRunEventTypeFromChatEvent(event *pb.ChatStreamResponse) string {
	if event.GetEventType() == "generating" && event.GetDelta() != "" {
		return "assistant.delta"
	}
	switch event.GetEventType() {
	case "tool_calling":
		return "tool.started"
	case "tool_done":
		return "tool.finished"
	case "process_delta", "process_clear", "context_usage", "thinking", "generating", "fallback":
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

func agentRunResultPayload(result hrChatRuntimeResult, runtimeLabel string) string {
	payload := map[string]any{
		"status":          agentRunStatusSucceeded,
		"result_metadata": agentRunResultMetadata(result, runtimeLabel),
	}
	return marshalJSONString(payload)
}

func agentRunResultMetadata(result hrChatRuntimeResult, runtimeLabel string) map[string]any {
	return map[string]any{
		"status":                       result.status,
		"suggested_questions":          append([]string(nil), result.suggestedQuestions...),
		"context_usage":                contextUsagePayload(result.contextUsage),
		"agent_skill_runtime_evidence": result.governance.AgentSkillRuntimeEvidence,
		"raw_json":                     buildHRProcessContent(result.toolTraces, result.contextUsage, result.fallbackUsed, result.governance, result.plan, runtimeLabel, result.suggestedQuestions),
	}
}

func contextUsagePayload(usage *pb.ContextUsageInfo) map[string]any {
	if usage == nil {
		return nil
	}
	payload := map[string]any{
		"model_id":                   usage.GetModelId(),
		"model_name":                 usage.GetModelName(),
		"context_window_tokens":      usage.GetContextWindowTokens(),
		"max_output_tokens":          usage.GetMaxOutputTokens(),
		"prompt_tokens_estimated":    usage.GetPromptTokensEstimated(),
		"prompt_tokens_actual":       usage.GetPromptTokensActual(),
		"completion_tokens_actual":   usage.GetCompletionTokensActual(),
		"total_tokens_actual":        usage.GetTotalTokensActual(),
		"remaining_tokens_estimated": usage.GetRemainingTokensEstimated(),
		"usage_ratio":                usage.GetUsageRatio(),
		"input_budget_tokens":        usage.GetInputBudgetTokens(),
		"safety_margin_tokens":       usage.GetSafetyMarginTokens(),
		"budget_usage_ratio":         usage.GetBudgetUsageRatio(),
		"budget_status":              usage.GetBudgetStatus(),
		"included_message_count":     usage.GetIncludedMessageCount(),
		"omitted_message_count":      usage.GetOmittedMessageCount(),
		"summary_applied":            usage.GetSummaryApplied(),
		"estimated":                  usage.GetEstimated(),
		"source":                     usage.GetSource(),
		"stage":                      usage.GetStage(),
	}
	if breakdown := usage.GetBreakdown(); breakdown != nil {
		payload["breakdown"] = map[string]any{
			"system_prompt_tokens":     breakdown.GetSystemPromptTokens(),
			"recent_message_tokens":    breakdown.GetRecentMessageTokens(),
			"summary_tokens":           breakdown.GetSummaryTokens(),
			"memory_tokens":            breakdown.GetMemoryTokens(),
			"current_message_tokens":   breakdown.GetCurrentMessageTokens(),
			"skill_tokens":             breakdown.GetSkillTokens(),
			"tool_result_tokens":       breakdown.GetToolResultTokens(),
			"tool_schema_tokens":       breakdown.GetToolSchemaTokens(),
			"protocol_overhead_tokens": breakdown.GetProtocolOverheadTokens(),
		}
	}
	return payload
}

func modelDisplayName(modelID int64) string {
	if modelID > 0 {
		return fmt.Sprintf("模型 #%d", modelID)
	}
	return "默认模型"
}

func (s *nativeAIService) resolveRuntimeModelDisplay(ctx context.Context, requestedModelID int64) (int64, string, string) {
	info := s.resolveRuntimeModelInfo(ctx, requestedModelID)
	return info.ID, info.Name, info.ProviderName
}

func (s *nativeAIService) resolveRuntimeModelInfo(ctx context.Context, requestedModelID int64) RuntimeModelInfo {
	fallback := RuntimeModelInfo{ID: requestedModelID, Name: modelDisplayName(requestedModelID)}
	if s == nil || s.store == nil {
		return fallback
	}
	if resolver, ok := s.store.(llmRuntimeModelInfoResolver); ok {
		info, found, err := resolver.ResolveLLMRuntimeModelInfo(ctx, requestedModelID)
		if err == nil && found && strings.TrimSpace(info.Name) != "" {
			info.Name = strings.TrimSpace(info.Name)
			info.ProviderName = strings.TrimSpace(info.ProviderName)
			return info
		}
	}
	if resolver, ok := s.store.(llmRuntimeModelResolver); ok {
		id, name, providerName, found, err := resolver.ResolveLLMRuntimeModel(ctx, requestedModelID)
		if err == nil && found && strings.TrimSpace(name) != "" {
			return RuntimeModelInfo{ID: id, Name: strings.TrimSpace(name), ProviderName: strings.TrimSpace(providerName)}
		}
	}
	rows, _, err := s.store.ListLlmModels(ctx, 1, 200, 0)
	if err != nil || len(rows) == 0 {
		return fallback
	}
	var firstEnabled *pb.LlmModelInfo
	for _, row := range rows {
		if row == nil || !row.GetIsEnabled() {
			continue
		}
		if firstEnabled == nil {
			firstEnabled = row
		}
		if requestedModelID > 0 && row.GetId() == requestedModelID {
			return runtimeModelInfoFromProto(row)
		}
		if requestedModelID <= 0 && row.GetIsDefault() {
			return runtimeModelInfoFromProto(row)
		}
	}
	if requestedModelID <= 0 && firstEnabled != nil {
		return runtimeModelInfoFromProto(firstEnabled)
	}
	return fallback
}

func (s *nativeAIService) resolveSelectableRuntimeModelInfo(ctx context.Context, requestedModelID int64) (RuntimeModelInfo, bool, error) {
	if s == nil || s.store == nil {
		return RuntimeModelInfo{}, false, nil
	}
	if resolver, ok := s.store.(llmRuntimeModelInfoResolver); ok {
		info, found, err := resolver.ResolveLLMRuntimeModelInfo(ctx, requestedModelID)
		if err != nil || !found {
			return RuntimeModelInfo{}, found, err
		}
		info.Name = strings.TrimSpace(info.Name)
		info.ProviderName = strings.TrimSpace(info.ProviderName)
		return info, info.ID > 0 && info.Name != "", nil
	}
	rows, _, err := s.store.ListLlmModels(ctx, 1, 200, 0)
	if err != nil {
		return RuntimeModelInfo{}, false, err
	}
	var firstEnabled *pb.LlmModelInfo
	for _, row := range rows {
		if row == nil || !row.GetIsEnabled() {
			continue
		}
		if firstEnabled == nil {
			firstEnabled = row
		}
		if requestedModelID > 0 && row.GetId() == requestedModelID {
			return runtimeModelInfoFromProto(row), true, nil
		}
		if requestedModelID <= 0 && row.GetIsDefault() {
			return runtimeModelInfoFromProto(row), true, nil
		}
	}
	if requestedModelID <= 0 && firstEnabled != nil {
		return runtimeModelInfoFromProto(firstEnabled), true, nil
	}
	return RuntimeModelInfo{}, false, nil
}

func runtimeModelInfoFromProto(model *pb.LlmModelInfo) RuntimeModelInfo {
	if model == nil {
		return RuntimeModelInfo{}
	}
	return RuntimeModelInfo{
		ID:                  model.GetId(),
		Name:                llmModelDisplayName(model),
		ProviderName:        strings.TrimSpace(model.GetProviderName()),
		ContextWindowTokens: model.GetContextWindowTokens(),
		MaxOutputTokens:     model.GetMaxTokens(),
	}
}

func llmModelDisplayName(model *pb.LlmModelInfo) string {
	if model == nil {
		return ""
	}
	if name := strings.TrimSpace(model.GetModelName()); name != "" {
		return name
	}
	if name := strings.TrimSpace(model.GetDisplayName()); name != "" {
		return name
	}
	return modelDisplayName(model.GetId())
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func agentRunPayloadFromCreateRequest(req *pb.CreateAgentRunRequest) agentRunDurablePayload {
	if req == nil {
		return agentRunDurablePayload{}
	}
	return agentRunDurablePayload{
		Message:              req.GetMessage(),
		ActionType:           req.GetActionType(),
		ActionPayloadJSON:    req.GetActionPayloadJson(),
		ApplicationID:        req.GetApplicationId(),
		ModelID:              req.GetModelId(),
		CapabilityKeys:       append([]string(nil), req.GetCapabilityKeys()...),
		AgentSkillVersionIDs: append([]int64(nil), req.GetAgentSkillVersionIds()...),
	}
}

func agentRunPlanJSON(payload agentRunDurablePayload) string {
	return marshalJSONString(map[string]any{"durable_request": payload})
}

func agentRunRuntimePlanJSON(payload agentRunDurablePayload, req *pb.ChatRequest, governance hrRuntimeGovernanceContext, plan commonsai.RecruitingPlan, runtimeModelID int64, runtimeModelName string, runtimeWarnings []string, runtimeLabel string) string {
	if strings.TrimSpace(runtimeLabel) == "" {
		runtimeLabel = agentRuntimeADK
	}
	if req != nil {
		if strings.TrimSpace(payload.Message) == "" {
			payload.Message = req.GetMessage()
		}
		if payload.ApplicationID == 0 {
			payload.ApplicationID = req.GetApplicationId()
		}
		if payload.ModelID == 0 {
			payload.ModelID = req.GetModelId()
		}
		if len(payload.CapabilityKeys) == 0 {
			payload.CapabilityKeys = append([]string(nil), req.GetCapabilityKeys()...)
		}
		if len(payload.AgentSkillVersionIDs) == 0 {
			payload.AgentSkillVersionIDs = append([]int64(nil), req.GetAgentSkillVersionIds()...)
		}
	}
	agentType := hrRecruitingAgentType
	agentName := hrRecruitingAgentType
	agentID := int64(0)
	if governance.Agent != nil {
		if value := strings.TrimSpace(governance.Agent.GetAgentType()); value != "" {
			agentType = value
		}
		if value := strings.TrimSpace(governance.Agent.GetName()); value != "" {
			agentName = value
		}
		agentID = governance.Agent.GetId()
	}
	decision := map[string]any{
		"intent":                     string(plan.Intent),
		"required_tool_count":        len(plan.RequiredTools),
		"required_data_count":        len(plan.RequiredData),
		"risk_flag_count":            len(plan.RiskChecks),
		"requires_human_confirm":     plan.ConfirmationRequirement.Required,
		"requires_evidence_citation": len(plan.RiskChecks) > 0,
	}
	if plan.ConfirmationRequirement.Required {
		decision["confirmation_required"] = true
		decision["confirmation_reason"] = plan.ConfirmationRequirement.Reason
	}
	if runtimeModelName == "" {
		runtimeModelName = modelDisplayName(payload.ModelID)
	}
	if runtimeModelID == 0 {
		runtimeModelID = payload.ModelID
	}
	runtimeWarnings = compactStrings(runtimeWarnings)
	if len(runtimeWarnings) > 0 {
		decision["runtime_warning"] = true
		decision["warning_count"] = len(runtimeWarnings)
		decision["warning_messages"] = runtimeWarnings
	}
	return marshalJSONString(map[string]any{
		"durable_request":   payload,
		"runtime":           runtimeLabel,
		"agent":             agentName,
		"agent_type":        agentType,
		"agent_id":          agentID,
		"model":             runtimeModelName,
		"model_id":          runtimeModelID,
		"application_bound": payload.ApplicationID > 0,
		"application_id":    payload.ApplicationID,
		"recruiting_plan":   plan,
		"risk_flags":        append([]string(nil), plan.RiskChecks...),
		"decision":          decision,
	})
}

func (s *nativeAIService) persistAgentRunRuntimePlan(ctx context.Context, req *pb.ChatRequest, opts hrChatRuntimeOptions, governance hrRuntimeGovernanceContext, plan commonsai.RecruitingPlan, runtimeModelID int64, runtimeModelName string, runtimeWarnings []string) error {
	if s == nil || s.store == nil || opts.agentRunID <= 0 || req == nil {
		return nil
	}
	payload := opts.durablePayload
	planJSON := agentRunRuntimePlanJSON(payload, req, governance, plan, runtimeModelID, runtimeModelName, runtimeWarnings, s.hrRuntimeLabel())
	if leaseID := agentRunSkillApprovalLeaseFromContext(ctx); leaseID != "" {
		store, ok := s.store.(agentRunSkillExecutionFenceStore)
		if !ok {
			return errAgentRunExecutionLeaseLost
		}
		if _, owned, err := store.UpdateAgentRunPlanForSkillLease(
			ctx,
			req.GetHrId(),
			opts.agentRunID,
			leaseID,
			planJSON,
			"",
		); err != nil {
			return err
		} else if !owned {
			return errAgentRunExecutionLeaseLost
		}
		return nil
	}
	if hasAgentRunSkillApprovalContext(ctx) {
		return errAgentRunExecutionLeaseLost
	}
	_, found, err := s.store.UpdateAgentRunPlan(ctx, req.GetHrId(), opts.agentRunID, planJSON, "")
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("agent run not found")
	}
	return nil
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
		"required":                            false,
		"recommended_agent_skill_version_ids": append([]int64(nil), req.GetSelectedAgentSkillVersionIds()...),
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

func agentRunExecutionContext(ctx context.Context, run AgentRunRow, payload agentRunDurablePayload) context.Context {
	authUserID := payload.AuthUserID
	if authUserID <= 0 {
		authUserID = run.OwnerID
	}
	if authUserID <= 0 {
		return ctx
	}
	accountType := strings.TrimSpace(payload.AuthAccountType)
	if accountType == "" {
		accountType = "staff"
	}
	tenantID := payload.AuthTenantID
	if tenantID <= 0 {
		tenantID = run.TenantID
	}
	ctx = platformmetadata.WithTenantActor(ctx, platformmetadata.TenantContext{
		TenantID:     tenantID,
		MembershipID: payload.AuthMembershipID,
		UserID:       authUserID,
		AccountType:  accountType,
		ClientApp:    payload.AuthClientApp,
	})
	// Durable execution has no inbound HTTP request after dispatch. Pin a stable
	// request identity so Billing reservation retries reuse the same idempotency
	// key instead of creating a second reservation for the same Agent Run.
	return context.WithValue(ctx, platformmetadata.KeyRequestID, fmt.Sprintf("agent-run:%d", run.ID))
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

func (s *nativeAIService) storeAgentRunCancel(runID int64, entry *agentRunCancelEntry) {
	if runID == 0 || entry == nil || entry.cancel == nil {
		return
	}
	s.runCancelMu.Lock()
	defer s.runCancelMu.Unlock()
	if s.runCancels == nil {
		s.runCancels = make(map[int64]*agentRunCancelEntry)
	}
	s.runCancels[runID] = entry
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

func (s *nativeAIService) clearAgentRunCancel(runID int64, expected *agentRunCancelEntry) {
	s.runCancelMu.Lock()
	entry := s.runCancels[runID]
	if entry == expected {
		delete(s.runCancels, runID)
	}
	s.runCancelMu.Unlock()
	if expected != nil && expected.cancel != nil {
		expected.cancel()
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
	applications applicationSnapshotClient
	jobs         jobServiceClient
	observer     recruitingruntime.Observer
	meter        *nativeAIService
}

func (s nativeRecruitingIntelligenceService) capabilityRuntimeContext(ctx context.Context, capability string, versionID, requestedModelID int64) (context.Context, RuntimeModelInfo, error) {
	if versionID <= 0 {
		return ctx, RuntimeModelInfo{}, nil
	}
	resolver, ok := s.store.(capabilityRuntimeModelResolver)
	if !ok {
		return ctx, RuntimeModelInfo{}, errors.New("platform AI capability resolver is unavailable")
	}
	resolved, err := resolver.ResolveCapabilityRuntimeModel(ctx, capability, platformAIAudienceTenantHR, versionID, requestedModelID)
	if err != nil {
		return ctx, RuntimeModelInfo{}, err
	}
	runtimeModel := RuntimeModelInfo{
		ID: resolved.EffectiveModelID, Name: resolved.ModelName, ProviderName: resolved.ProviderName,
		RequestedModelID: resolved.RequestedModelID, FallbackReason: resolved.FallbackReason,
		CapabilityVersionID: resolved.CapabilityVersionID, CapabilitySnapshotHash: resolved.CapabilitySnapshotHash,
		ContextWindowTokens: resolved.ContextWindowTokens, MaxOutputTokens: resolved.MaxOutputTokens,
		ConfigurationRefs: resolved.ConfigurationRefs, SkillRuntimePolicy: resolved.SkillRuntimePolicy,
	}
	return withRecruitingCapabilityRuntime(ctx, runtimeModel), runtimeModel, nil
}

func withRecruitingCapabilityRuntime(ctx context.Context, model RuntimeModelInfo) context.Context {
	return withRecruitingCapabilityRuntimeFeature(ctx, model, false)
}

func withRecruitingCapabilityRuntimeFeature(ctx context.Context, model RuntimeModelInfo, skillPackageV2 bool) context.Context {
	return recruitingruntime.WithCapabilityRuntime(
		ctx, model.RequestedModelID, model.ID, model.CapabilityVersionID,
		model.FallbackReason, model.CapabilitySnapshotHash, model.ConfigurationRefs.PromptTemplateIDs,
		model.ConfigurationRefs.AgentSkillVersionIDs, skillPackageV2,
	)
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
	return recruitingruntime.NewRuntimeWithObserverAndStrictContracts(
		recruitingruntime.NewPromptLoader(promptStore),
		structuredProvider,
		newRecruitingRuntimeObserver(),
		recruitingStrictContractResolver(store),
		policies...,
	)
}

func recruitingStrictContractResolver(store any) *recruitingruntime.StrictContractResolver {
	packageStore, ok := store.(hrRuntimeAgentSkillPackageStore)
	if !ok {
		// Keep the resolver installed even for partial/legacy store adapters.
		// The resolver remains a no-op while v2 is disabled or no exact release
		// versions are present, and fails closed before prompt/provider access
		// when an enabled exact v2 release cannot load its immutable package.
		return recruitingruntime.NewStrictContractResolver(nil)
	}
	return recruitingruntime.NewStrictContractResolver(recruitingAgentSkillPackageLoader{store: packageStore})
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
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
	}
	if s.store == nil {
		return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
	}
	resumeID, accessResp, _ := s.resolveResumeProfileAccess(ctx, req)
	if accessResp != nil {
		return accessResp, nil
	}
	profileID := req.GetProfileId()
	if profileID == 0 {
		profile, found, err := s.store.GetCurrentRecruitingResumeProfileByResumeID(ctx, resumeID)
		if err != nil {
			return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
		}
		if !found {
			return &pb.GetResumeProfileResponse{Code: 404, Msg: "ai.resume_profile_not_found"}, nil
		}
		profileID = profile.ID
	}
	snapshot, found, err := s.store.GetRecruitingResumeProfileSnapshot(ctx, profileID)
	if err != nil {
		return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
	}
	if !found {
		return &pb.GetResumeProfileResponse{Code: 404, Msg: "common.operation_failed"}, nil
	}
	if snapshot.Profile.ResumeID != resumeID {
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
	}
	return &pb.GetResumeProfileResponse{Code: errs.OK, Msg: "common.success", Profile: recruitingResumeProfileSnapshotPB(snapshot)}, nil
}

func (s nativeRecruitingIntelligenceService) ParseResumeProfile(ctx context.Context, req *pb.ParseResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	finalizer := newRecruitingOperationFinalizer(s, ctx, "resume_profile", "resume", req.GetResumeId())
	defer finalizer.finalize()
	if req.GetApplicationId() <= 0 && req.GetResumeId() <= 0 {
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
	}
	if s.store == nil {
		finalizer.classify("configuration_failure", "error")
		return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
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
			return &pb.GetResumeProfileResponse{Code: authErr.code, Msg: "common.operation_failed"}, nil
		}
		var runtimeErr error
		var runtimeModel RuntimeModelInfo
		if s.meter != nil {
			runtimeModel, runtimeErr = s.meter.resolveCapabilityRuntimeModel(ctx, billingOwnerTenant, req.GetStaffUserId(), "ai.resume_parse", platformAIAudienceTenantHR, req.GetModelId())
			if runtimeErr == nil {
				ctx = withRecruitingCapabilityRuntimeFeature(ctx, runtimeModel, s.meter.skillPackageV2Enabled)
			}
		} else {
			ctx, runtimeModel, runtimeErr = s.capabilityRuntimeContext(ctx, "ai.resume_parse", req.GetCapabilityVersionId(), req.GetModelId())
		}
		if runtimeErr != nil {
			finalizer.classify("configuration_failure", "error")
			return &pb.GetResumeProfileResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
		}
		sourceStarted := time.Now()
		source, found, err := generationStore.GetRecruitingResumeSource(ctx, resumeID)
		if err != nil {
			s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "source", "source_failure", "error", false, sourceStarted))
			finalizer.classify("source_failure", "error")
			return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
		}
		if !found {
			finalizer.classify("not_found", "error")
			return &pb.GetResumeProfileResponse{Code: 404, Msg: "common.operation_failed"}, nil
		}
		if strings.TrimSpace(source.ParsedText) == "" {
			finalizer.classify("domain_validation_failure", "error")
			return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
		}
		if s.meter != nil {
			ctx, runtimeErr = s.meter.reserveAIBilling(ctx, billingOwnerTenant, req.GetStaffUserId(), "ai.resume_parse", "resume_parse", runtimeModel.ProviderName, runtimeModel.Name, len([]rune(source.ParsedText)), runtimeModel)
			if runtimeErr != nil {
				finalizer.classify("billing_failure", "error")
				return nil, runtimeErr
			}
			defer s.meter.cancelUnsettledBilling(ctx, "resume_parse_completed_without_provider_usage")
		}
		sourceEvent := recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "source", "success", "success", false, sourceStarted)
		sourceEvent.InputCount = 1
		s.observeRecruiting(ctx, sourceEvent)
		totalCtx, parseCtx, cancel := s.policy.ResumeExecutionContexts(ctx)
		defer cancel()
		parseCtx = recruitingruntime.WithObservationMetadata(parseCtx, platformmetadata.GetRequestID(ctx), "resume", resumeID)
		parseCtx, billingUsage := recruitingruntime.WithBillingUsageCollector(parseCtx)
		parseCtx, strictValidation := s.withStrictValidationMetrics(parseCtx)
		defer s.finalizeStrictValidationMetrics(ctx, strictValidation)
		if s.meter != nil {
			defer s.meter.finalizeStructuredBilling(ctx, runtimeModel, billingUsage)
		}
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
			return &pb.GetResumeProfileResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
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
			return &pb.GetResumeProfileResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
		}
		persistenceStarted := time.Now()
		snapshot, err := generationStore.SaveRecruitingResumeProfileDraft(totalCtx, draft)
		if err != nil {
			persistenceEvent := recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "persistence", "persistence_failure", "error", false, persistenceStarted)
			persistenceEvent.ParserVersion, persistenceEvent.OutputCount = draft.ParserVersion, generationEvent.OutputCount
			s.observeRecruiting(ctx, persistenceEvent)
			finalizer.classify("persistence_failure", "error")
			finalizer.event.ParserVersion, finalizer.event.OutputCount = draft.ParserVersion, generationEvent.OutputCount
			return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
		}
		s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "resume_profile", "resume", resumeID, "persistence", "success", "success", false, persistenceStarted))
		finalizer.classify("success", "success")
		finalizer.event.ParserVersion = draft.ParserVersion
		if draft.ParserVersion == recruitingruntime.ResumeHeuristicParserVersion {
			finalizer.event.Category, finalizer.event.Fallback = "fallback_success", "heuristic"
		}
		finalizer.event.OutputCount = generationEvent.OutputCount
		return &pb.GetResumeProfileResponse{Code: errs.OK, Msg: "common.success", Profile: recruitingResumeProfileSnapshotPB(snapshot)}, nil
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
	if resp.GetCode() == 404 && resp.GetMsg() == "ai.resume_profile_not_found" {
		resp.Msg = "ai.resume_profile_unavailable"
	}
	return resp, nil
}

func (s nativeRecruitingIntelligenceService) ParseResumeProfileForCandidate(ctx context.Context, req *pb.ParseResumeProfileForCandidateRequest) (*pb.GetResumeProfileResponse, error) {
	finalizer := newRecruitingOperationFinalizer(s, ctx, "resume_profile", "resume", req.GetResumeId())
	defer finalizer.finalize()
	if req.GetCandidateUserId() <= 0 || req.GetResumeId() <= 0 {
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
	}
	if s.store == nil {
		finalizer.classify("configuration_failure", "error")
		return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
	}
	generationStore, ok := s.store.(recruitingResumeProfileGenerationStore)
	if !ok {
		finalizer.classify("configuration_failure", "error")
		return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
	}
	source, found, err := generationStore.GetRecruitingResumeSource(ctx, req.GetResumeId())
	if err != nil {
		finalizer.classify("source_failure", "error")
		return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
	}
	if !found {
		finalizer.classify("not_found", "error")
		return &pb.GetResumeProfileResponse{Code: 404, Msg: "common.operation_failed"}, nil
	}
	if source.UserID != req.GetCandidateUserId() {
		finalizer.classify("forbidden", "error")
		return &pb.GetResumeProfileResponse{Code: errs.ErrForbidden, Msg: "common.forbidden"}, nil
	}
	if strings.TrimSpace(source.ParsedText) == "" {
		finalizer.classify("domain_validation_failure", "error")
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
	}
	totalCtx, parseCtx, cancel := s.policy.ResumeExecutionContexts(ctx)
	defer cancel()
	parseCtx = recruitingruntime.WithObservationMetadata(parseCtx, platformmetadata.GetRequestID(ctx), "resume", req.GetResumeId())
	draft, err := s.generateResumeProfileDraft(parseCtx, source)
	if err != nil {
		finalizer.classify(recruitingruntime.ObservationCategoryForError(err), "error")
		return &pb.GetResumeProfileResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	if err := totalCtx.Err(); err != nil {
		finalizer.classify("timeout", "error")
		return &pb.GetResumeProfileResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	snapshot, err := generationStore.SaveRecruitingResumeProfileDraft(totalCtx, draft)
	if err != nil {
		finalizer.classify("persistence_failure", "error")
		return &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
	}
	finalizer.classify("success", "success")
	finalizer.event.ParserVersion = draft.ParserVersion
	finalizer.event.OutputCount = boundedRecruitingCount(1 + len(draft.Educations) + len(draft.Experiences) + len(draft.Projects) + len(draft.Skills))
	if draft.ParserVersion == recruitingruntime.ResumeHeuristicParserVersion {
		finalizer.event.Category, finalizer.event.Fallback = "fallback_success", "heuristic"
	}
	return &pb.GetResumeProfileResponse{Code: errs.OK, Msg: "common.success", Profile: recruitingResumeProfileSnapshotPB(snapshot)}, nil
}

func (s nativeRecruitingIntelligenceService) EvaluateCandidateMatch(ctx context.Context, req *pb.EvaluateCandidateMatchRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	finalizer := newRecruitingOperationFinalizer(s, ctx, "candidate_match", "application", req.GetApplicationId())
	defer finalizer.finalize()
	if req.GetApplicationId() <= 0 {
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
	}
	if s.store == nil {
		finalizer.classify("configuration_failure", "error")
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
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
		var runtimeErr error
		var runtimeModel RuntimeModelInfo
		if s.meter != nil {
			runtimeModel, runtimeErr = s.meter.resolveCapabilityRuntimeModel(ctx, billingOwnerTenant, req.GetStaffUserId(), "ai.match_evaluation", platformAIAudienceTenantHR, req.GetModelId())
			if runtimeErr == nil {
				ctx = withRecruitingCapabilityRuntimeFeature(ctx, runtimeModel, s.meter.skillPackageV2Enabled)
			}
		} else {
			ctx, runtimeModel, runtimeErr = s.capabilityRuntimeContext(ctx, "ai.match_evaluation", req.GetCapabilityVersionId(), req.GetModelId())
		}
		if runtimeErr != nil {
			finalizer.classify("configuration_failure", "error")
			return &pb.GetCandidateMatchEvaluationResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
		}
		if s.meter != nil {
			ctx, runtimeErr = s.meter.reserveAIBilling(ctx, billingOwnerTenant, req.GetStaffUserId(), "ai.match_evaluation", "match_evaluation", runtimeModel.ProviderName, runtimeModel.Name, 0, runtimeModel)
			if runtimeErr != nil {
				finalizer.classify("billing_failure", "error")
				return nil, runtimeErr
			}
			defer s.meter.cancelUnsettledBilling(ctx, "match_evaluation_completed_without_provider_usage")
		}
		totalCtx, generationCtx, cancel := s.policy.CandidateMatchExecutionContexts(ctx)
		defer cancel()
		generationCtx = recruitingruntime.WithObservationMetadata(generationCtx, platformmetadata.GetRequestID(ctx), "application", req.GetApplicationId())
		generationCtx, billingUsage := recruitingruntime.WithBillingUsageCollector(generationCtx)
		generationCtx, strictValidation := s.withStrictValidationMetrics(generationCtx)
		defer s.finalizeStrictValidationMetrics(ctx, strictValidation)
		if s.meter != nil {
			defer s.meter.finalizeStructuredBilling(ctx, runtimeModel, billingUsage)
		}
		sourceStarted := time.Now()
		source, found, err := generationStore.GetRecruitingMatchSource(generationCtx, req.GetApplicationId())
		if err != nil {
			category := "source_failure"
			if generationCtx.Err() != nil || totalCtx.Err() != nil {
				category = "timeout"
				s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "source", category, "error", false, sourceStarted))
				finalizer.classify(category, "error")
				return &pb.GetCandidateMatchEvaluationResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
			}
			s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "source", category, "error", false, sourceStarted))
			finalizer.classify(category, "error")
			return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
		}
		if !found {
			finalizer.classify("not_found", "error")
			return &pb.GetCandidateMatchEvaluationResponse{Code: 404, Msg: "common.operation_failed"}, nil
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
			return &pb.GetCandidateMatchEvaluationResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
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
			return &pb.GetCandidateMatchEvaluationResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
		}
		persistenceStarted := time.Now()
		snapshot, err := generationStore.SaveRecruitingCandidateMatchDraft(totalCtx, draft)
		if err != nil {
			persistenceEvent := recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "persistence", "persistence_failure", "error", false, persistenceStarted)
			persistenceEvent.ScorerVersion, persistenceEvent.RequirementCount, persistenceEvent.EvidenceCount = draft.ScorerVersion, aggregationEvent.RequirementCount, aggregationEvent.EvidenceCount
			s.observeRecruiting(ctx, persistenceEvent)
			finalizer.classify("persistence_failure", "error")
			finalizer.event.ScorerVersion, finalizer.event.RequirementCount, finalizer.event.EvidenceCount = draft.ScorerVersion, aggregationEvent.RequirementCount, aggregationEvent.EvidenceCount
			return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
		}
		s.observeRecruiting(ctx, recruitingOperationEvent(ctx, "candidate_match", "application", req.GetApplicationId(), "persistence", "success", "success", false, persistenceStarted))
		finalizer.classify("success", "success")
		finalizer.event.ScorerVersion, finalizer.event.RequirementCount, finalizer.event.EvidenceCount = draft.ScorerVersion, aggregationEvent.RequirementCount, aggregationEvent.EvidenceCount
		if draft.FallbackUsed {
			finalizer.event.Category, finalizer.event.Fallback = "fallback_success", "legacy_deterministic"
		}
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.OK, Msg: "common.success", Evaluation: recruitingCandidateMatchSnapshotPB(snapshot)}, nil
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
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
	}
	if !found {
		finalizer.classify("not_found", "error")
		if req.GetAgentRunId() > 0 {
			return &pb.GetCandidateMatchEvaluationResponse{Code: 404, Msg: "common.operation_failed"}, nil
		}
		return &pb.GetCandidateMatchEvaluationResponse{Code: configCodeUnsupported, Msg: "ai.candidate_match_unavailable"}, nil
	}
	finalizer.classify("success", "success")
	return &pb.GetCandidateMatchEvaluationResponse{Code: errs.OK, Msg: "common.success", Evaluation: recruitingCandidateMatchSnapshotPB(snapshot)}, nil
}

func (s nativeRecruitingIntelligenceService) generateResumeProfileDraft(ctx context.Context, source RecruitingResumeSource) (RecruitingResumeProfileDraft, error) {
	structured := s.structured
	if structured == nil {
		promptStore, promptOK := s.store.(recruitingruntime.PromptStore)
		structuredProvider, providerOK := s.provider.(recruitingruntime.StructuredCompletionProvider)
		if promptOK && providerOK {
			structured = recruitingruntime.NewRuntimeWithObserverAndStrictContracts(
				recruitingruntime.NewPromptLoader(promptStore),
				structuredProvider,
				newRecruitingRuntimeObserver(),
				recruitingStrictContractResolver(s.store),
				s.policy,
			)
		}
	}
	result, err := recruitingruntime.NewResumeProfileExtractor(structured, s.policy).Extract(ctx, recruitingruntime.ResumeSource{
		ResumeID: source.ResumeID, UserID: source.UserID, FileName: source.FileName, ParsedText: source.ParsedText,
	})
	if err != nil {
		return RecruitingResumeProfileDraft{}, err
	}
	profile := result.Profile
	requestedModelID, effectiveModelID, capabilityVersionID, fallbackReason, snapshotHash := recruitingruntime.CapabilityRuntimeTrace(ctx)
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
		RequestedModelID:     requestedModelID, EffectiveModelID: effectiveModelID,
		ModelFallbackReason: fallbackReason, CapabilityVersionID: capabilityVersionID,
		CapabilitySnapshotHash: snapshotHash,
		Educations:             make([]RecruitingResumeEducationRow, 0, len(profile.Educations)),
		Experiences:            make([]RecruitingResumeExperienceRow, 0, len(profile.Experiences)),
		Projects:               make([]RecruitingResumeProjectRow, 0, len(profile.Projects)),
		Skills:                 make([]RecruitingResumeSkillRow, 0, len(profile.Skills)),
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
	if err == nil {
		draft.RequestedModelID, draft.EffectiveModelID, draft.CapabilityVersionID, draft.ModelFallbackReason, draft.CapabilitySnapshotHash = recruitingruntime.CapabilityRuntimeTrace(ctx)
	}
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
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
	}
	if s.store == nil {
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
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
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
	}
	if !found {
		return &pb.GetCandidateMatchEvaluationResponse{Code: 404, Msg: "common.operation_failed"}, nil
	}
	return &pb.GetCandidateMatchEvaluationResponse{Code: errs.OK, Msg: "common.success", Evaluation: recruitingCandidateMatchSnapshotPB(snapshot)}, nil
}

func (s nativeRecruitingIntelligenceService) CompareCandidatesForJob(ctx context.Context, req *pb.CompareCandidatesForJobRequest) (*pb.CompareCandidatesForJobResponse, error) {
	if _, authErr := s.authorizeRecruitingJob(ctx, req.GetStaffUserId(), req.GetJobId()); authErr != nil {
		return recruitingComparisonAuthResponse(req.GetJobId(), authErr), nil
	}
	if s.store == nil {
		return &pb.CompareCandidatesForJobResponse{Code: errs.ErrInternal, Msg: "common.operation_failed", JobId: req.GetJobId()}, nil
	}
	applications, err := s.store.ListCurrentRecruitingApplicationsByJobID(ctx, req.GetJobId())
	if err != nil {
		return &pb.CompareCandidatesForJobResponse{Code: errs.ErrInternal, Msg: "common.operation_failed", JobId: req.GetJobId()}, nil
	}
	applicationIDs := make([]int64, 0, len(applications))
	for _, application := range applications {
		applicationIDs = append(applicationIDs, application.ApplicationID)
	}
	evaluations, err := s.store.ListLatestRecruitingCandidateMatchEvaluationsByApplicationIDs(ctx, applicationIDs)
	if err != nil {
		return &pb.CompareCandidatesForJobResponse{Code: errs.ErrInternal, Msg: "common.operation_failed", JobId: req.GetJobId()}, nil
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
	return &pb.CompareCandidatesForJobResponse{Code: errs.OK, Msg: "common.success", JobId: req.GetJobId(), Candidates: candidates, MissingApplicationIds: missing}, nil
}

func (s nativeRecruitingIntelligenceService) resolveResumeProfileAccess(ctx context.Context, req *pb.GetResumeProfileRequest) (int64, *pb.GetResumeProfileResponse, *recruitingAuthError) {
	if req.GetApplicationId() > 0 {
		if _, authErr := s.authorizeRecruitingApplication(ctx, req.GetStaffUserId(), req.GetApplicationId()); authErr != nil {
			return 0, recruitingResumeAuthResponse(authErr), authErr
		}
		application, found, err := s.store.GetRecruitingApplicationByID(ctx, req.GetApplicationId())
		if err != nil {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
		}
		if !found || application.ResumeID <= 0 {
			return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "common.operation_failed"}, nil
		}
		if req.GetResumeId() > 0 && req.GetResumeId() != application.ResumeID {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
		}
		if req.GetProfileId() > 0 {
			profile, profileFound, profileErr := s.store.GetRecruitingResumeProfileByID(ctx, req.GetProfileId())
			if profileErr != nil {
				return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
			}
			if !profileFound {
				return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "common.operation_failed"}, nil
			}
			if profile.ResumeID != application.ResumeID {
				return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
			}
		}
		return application.ResumeID, nil, nil
	}

	resumeID := req.GetResumeId()
	if req.GetProfileId() > 0 {
		profile, found, err := s.store.GetRecruitingResumeProfileByID(ctx, req.GetProfileId())
		if err != nil {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
		}
		if !found {
			return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "common.operation_failed"}, nil
		}
		if resumeID > 0 && resumeID != profile.ResumeID {
			return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
		}
		resumeID = profile.ResumeID
	}
	if resumeID <= 0 {
		return 0, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
	}
	application, found, err := s.store.GetLatestRecruitingApplicationByResumeID(ctx, resumeID)
	if err != nil {
		return 0, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: "common.operation_failed"}, nil
	}
	if !found {
		return 0, &pb.GetResumeProfileResponse{Code: 404, Msg: "common.operation_failed"}, nil
	}
	if _, authErr := s.authorizeRecruitingApplication(ctx, req.GetStaffUserId(), application.ApplicationID); authErr != nil {
		return 0, recruitingResumeAuthResponse(authErr), authErr
	}
	return resumeID, nil, nil
}

func recruitingResumeAuthResponse(authErr *recruitingAuthError) *pb.GetResumeProfileResponse {
	return &pb.GetResumeProfileResponse{Code: authErr.code, Msg: "common.operation_failed"}
}

func recruitingMatchAuthResponse(authErr *recruitingAuthError) *pb.GetCandidateMatchEvaluationResponse {
	return &pb.GetCandidateMatchEvaluationResponse{Code: authErr.code, Msg: "common.operation_failed"}
}

func recruitingComparisonAuthResponse(jobID int64, authErr *recruitingAuthError) *pb.CompareCandidatesForJobResponse {
	return &pb.CompareCandidatesForJobResponse{Code: authErr.code, Msg: "common.operation_failed", JobId: jobID}
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
		Id:                     row.ID,
		ResumeId:               row.ResumeID,
		UserId:                 row.UserID,
		Status:                 row.Status,
		ParserVersion:          row.ParserVersion,
		InputHash:              row.InputHash,
		ErrorMessage:           row.ErrorMessage,
		StartedAt:              formatTime(row.StartedAt),
		CompletedAt:            formatTimePtr(row.CompletedAt),
		CreatedAt:              formatTime(row.CreatedAt),
		UpdatedAt:              formatTime(row.UpdatedAt),
		RequestedModelId:       row.RequestedModelID,
		EffectiveModelId:       row.EffectiveModelID,
		ModelFallbackReason:    row.ModelFallbackReason,
		CapabilityVersionId:    row.CapabilityVersionID,
		CapabilitySnapshotHash: row.CapabilitySnapshotHash,
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
		RequestedModelId:        row.RequestedModelID,
		EffectiveModelId:        row.EffectiveModelID,
		ModelFallbackReason:     row.ModelFallbackReason,
		CapabilityVersionId:     row.CapabilityVersionID,
		CapabilitySnapshotHash:  row.CapabilitySnapshotHash,
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
		return &pb.ListProvidersResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	rows, total, err := s.store.ListLlmProviders(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &pb.ListProvidersResponse{Code: 0, Msg: "common.success", Total: total, List: rows}, nil
}
func (s nativeLlmConfigService) ListModels(ctx context.Context, req *pb.ListModelsRequest) (*pb.ListModelsResponse, error) {
	if s.store == nil {
		return &pb.ListModelsResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	rows, total, err := s.store.ListLlmModels(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetProviderId())
	if err != nil {
		return nil, err
	}
	return &pb.ListModelsResponse{Code: 0, Msg: "common.success", Total: total, List: rows}, nil
}
func (s nativePromptService) ListPromptTemplates(ctx context.Context, req *pb.ListPromptTemplatesRequest) (*pb.ListPromptTemplatesResponse, error) {
	if s.store == nil {
		return &pb.ListPromptTemplatesResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	rows, total, err := s.store.ListPromptTemplates(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetAgentType())
	if err != nil {
		return nil, err
	}
	return &pb.ListPromptTemplatesResponse{Code: 0, Msg: "common.success", Total: total, List: rows}, nil
}
func (s nativeAgentConfigService) ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	if s.store == nil {
		return &pb.ListAgentsResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	rows, total, err := s.store.ListAgentConfigs(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetAgentType())
	if err != nil {
		return nil, err
	}
	return &pb.ListAgentsResponse{Code: 0, Msg: "common.success", Total: total, List: rows}, nil
}
func (nativeAgentConfigService) ListCapabilities(_ context.Context, req *pb.ListCapabilitiesRequest) (*pb.ListCapabilitiesResponse, error) {
	return &pb.ListCapabilitiesResponse{Code: 0, Msg: "common.success", List: builtinCapabilities(req.GetAgentType())}, nil
}
func (s nativeMCPService) ListMCPServers(ctx context.Context, req *pb.ListMCPServersRequest) (*pb.ListMCPServersResponse, error) {
	if s.store == nil {
		return &pb.ListMCPServersResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	rows, total, err := s.store.ListMCPServers(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &pb.ListMCPServersResponse{Code: 0, Msg: "common.success", Total: total, List: rows}, nil
}
func (s nativeMCPService) ListMCPToolPolicies(ctx context.Context, req *pb.ListMCPToolPoliciesRequest) (*pb.ListMCPToolPoliciesResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.ListMCPToolPoliciesResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	return store.ListMCPToolPolicies(ctx, req)
}
func (s nativeMCPService) ListMCPToolLogs(ctx context.Context, req *pb.ListMCPToolLogsRequest) (*pb.ListMCPToolLogsResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.ListMCPToolLogsResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	return store.ListMCPToolLogs(ctx, req)
}
func (s nativeAgentSkillService) ListAgentSkills(ctx context.Context, req *pb.ListAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	if s.store == nil {
		return &pb.ListAgentSkillsResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	rows, total, err := s.store.ListAgentSkills(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetKeyword(), req.GetEnabledOnly())
	if err != nil {
		return nil, err
	}
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "common.success", Total: total, List: rows}, nil
}
func (s nativeAgentSkillService) ListAvailableAgentSkills(ctx context.Context, req *pb.ListAvailableAgentSkillsRequest) (*pb.ListAgentSkillsResponse, error) {
	if s.store == nil {
		return &pb.ListAgentSkillsResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	if store, ok := s.store.(availableAgentSkillStore); ok {
		rows, total, err := store.ListAvailableAgentSkills(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), hrRecruitingAgentType)
		if err != nil {
			return nil, err
		}
		return &pb.ListAgentSkillsResponse{Code: 0, Msg: "common.success", Total: total, List: rows}, nil
	}
	rows, _, err := s.store.ListAgentSkills(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), "", true)
	if err != nil {
		return nil, err
	}
	available := make([]*pb.AgentSkillInfo, 0, len(rows))
	detailStore, hasDetailStore := s.store.(agentSkillDetailStore)
	for _, row := range rows {
		if row == nil {
			continue
		}
		currentSummary := row.GetCurrentVersion()
		if !row.GetIsEnabled() || !row.GetIsManualInvocable() ||
			row.GetCurrentVersionId() <= 0 || currentSummary == nil ||
			!strings.EqualFold(strings.TrimSpace(currentSummary.GetAgentType()), hrRecruitingAgentType) ||
			!hasDetailStore {
			continue
		}
		versions, versionErr := detailStore.ListAgentSkillVersions(ctx, &pb.ListAgentSkillVersionsRequest{SkillId: row.GetId()})
		if versionErr != nil || versions == nil || versions.GetCode() != 0 {
			continue
		}
		for _, version := range versions.GetList() {
			if version != nil && version.GetId() == row.GetCurrentVersionId() &&
				version.GetSkillId() == row.GetId() && version.GetPackage() != nil &&
				strings.TrimSpace(version.GetPackage().GetCoreMarkdown()) != "" {
				available = append(available, row)
				break
			}
		}
	}
	return &pb.ListAgentSkillsResponse{Code: 0, Msg: "common.success", Total: int64(len(available)), List: available}, nil
}
func (s nativeEmbeddingConfigService) ListEmbeddingProviders(ctx context.Context, req *pb.ListEmbeddingProvidersRequest) (*pb.ListEmbeddingProvidersResponse, error) {
	if s.store == nil {
		return &pb.ListEmbeddingProvidersResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	rows, total, err := s.store.ListEmbeddingProviders(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &pb.ListEmbeddingProvidersResponse{Code: 0, Msg: "common.success", Total: total, List: rows}, nil
}
func (s nativeEmbeddingConfigService) ListEmbeddingModels(ctx context.Context, req *pb.ListEmbeddingModelsRequest) (*pb.ListEmbeddingModelsResponse, error) {
	if s.store == nil {
		return &pb.ListEmbeddingModelsResponse{Code: configCodeUnavailable, Msg: "common.operation_failed"}, nil
	}
	rows, total, err := s.store.ListEmbeddingModels(ctx, normalizePage(req.GetPage()), normalizePageSize(req.GetPageSize()), req.GetProviderId())
	if err != nil {
		return nil, err
	}
	return &pb.ListEmbeddingModelsResponse{Code: 0, Msg: "common.success", Total: total, List: rows}, nil
}

func commonOK() *pb.CommonResponse {
	return &pb.CommonResponse{Code: 0, Msg: "common.success"}
}

func mapChatSession(row ChatSessionRow) *pb.ChatSession {
	return &pb.ChatSession{
		SessionId:          row.ID,
		Title:              row.Title,
		ApplicationId:      row.ApplicationID,
		CreatedAt:          formatTime(row.CreatedAt),
		UpdatedAt:          formatTime(row.UpdatedAt),
		SessionType:        row.SessionType,
		SourceType:         row.SourceType,
		SourceId:           row.SourceID,
		SourceTitle:        row.SourceTitle,
		Summary:            row.Summary,
		LastMessagePreview: row.LastMessagePreview,
		MessageCount:       row.MessageCount,
		LatestContextUsage: row.LatestContextUsage,
		SelectedModelId:    row.SelectedModelID,
	}
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
		items = append(items, &pb.ChatMessage{
			Role:                      row.Role,
			Content:                   row.Content,
			ProcessContent:            row.ProcessContent,
			ModelId:                   row.ModelID,
			ModelName:                 row.ModelName,
			ContextUsage:              row.ContextUsage,
			AgentSkillVersionIds:      append([]int64(nil), row.AgentSkillVersionIDs...),
			AgentSkillNames:           append([]string(nil), row.AgentSkillNames...),
			AgentSkillRuntimeEvidence: agentSkillRuntimeEvidenceFromProcessContent(row.ProcessContent),
			CreatedAt:                 formatTime(row.CreatedAt),
		})
	}
	return items
}

func agentSkillRuntimeEvidenceFromProcessContent(processContent string) []*pb.AgentSkillRuntimeEvidence {
	if strings.TrimSpace(processContent) == "" {
		return nil
	}
	var envelope struct {
		Governance struct {
			AgentSkillRuntimeEvidence []json.RawMessage `json:"agent_skill_runtime_evidence"`
		} `json:"governance"`
	}
	if err := json.Unmarshal([]byte(processContent), &envelope); err != nil {
		return nil
	}
	items := make([]*pb.AgentSkillRuntimeEvidence, 0, len(envelope.Governance.AgentSkillRuntimeEvidence))
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	for _, raw := range envelope.Governance.AgentSkillRuntimeEvidence {
		item := &pb.AgentSkillRuntimeEvidence{}
		if err := opts.Unmarshal(raw, item); err == nil {
			items = append(items, item)
		}
	}
	return items
}

func mapAgentRunItem(row AgentRunRow) *pb.AgentRunItem {
	return &pb.AgentRunItem{Id: row.ID, SessionId: row.SessionID, MessageId: row.MessageID, HistoryId: row.HistoryID, HrId: row.OwnerID, AgentType: row.AgentType, AgentId: row.AgentID, AgentName: row.AgentName, ModelId: row.ModelID, ModelName: row.ModelName, Status: row.Status, PlanJson: row.PlanJSON, FinalAnswer: row.AssistantText, ErrorType: row.ErrorType, ErrorMessage: row.ErrorMessage, StartedAt: formatTime(row.StartedAt), CompletedAt: formatTimePtr(row.CompletedAt), CreatedAt: formatTime(row.CreatedAt), ResultMetadata: parseAgentRunResultMetadata(json.RawMessage(row.ResultMetadataJSON))}
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
	snapshot := &pb.AgentRunSnapshot{RunId: row.ID, SessionId: row.SessionID, HrId: row.OwnerID, ClientRequestId: row.ClientRequestID, MessageId: row.MessageID, HistoryId: row.HistoryID, Status: row.Status, AssistantText: row.AssistantText, ProcessText: row.ProcessText, ResultMetadata: parseAgentRunResultMetadata(json.RawMessage(row.ResultMetadataJSON)), OptionContextJson: row.OptionContextJSON, LastEventSeq: row.LastEventSeq, ErrorType: row.ErrorType, ErrorMessage: row.ErrorMessage, ModelId: row.ModelID, ModelName: row.ModelName, AgentType: row.AgentType, AgentId: row.AgentID, AgentName: row.AgentName, StartedAt: formatTime(row.StartedAt), CompletedAt: formatTimePtr(row.CompletedAt), CancelRequestedAt: formatTimePtr(row.CancelRequestedAt), CanceledAt: formatTimePtr(row.CanceledAt), CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt)}
	applyAgentRunSnapshotPayload(snapshot, row.OptionContextJSON)
	applyAgentRunSnapshotPayload(snapshot, row.ProcessText)
	return snapshot
}

func fallbackAgentRun(hrID, sessionID int64, clientRequestID string, payload agentRunDurablePayload) AgentRunRow {
	now := time.Now()
	return AgentRunRow{SessionID: sessionID, OwnerID: hrID, ClientRequestID: clientRequestID, Status: "queued", AssistantText: "", PlanJSON: agentRunPlanJSON(payload), ModelID: payload.ModelID, ModelName: modelDisplayName(payload.ModelID), AgentType: "hr", AgentName: "hr_recruiting_agent", StartedAt: now, CreatedAt: now, UpdatedAt: now}
}

func applyHRGovernanceToAgentRun(run *AgentRunRow, governance hrRuntimeGovernanceContext) {
	if run == nil || governance.Agent == nil {
		return
	}
	run.AgentID = governance.Agent.GetId()
	run.AgentType = strings.TrimSpace(governance.Agent.GetAgentType())
	run.AgentName = strings.TrimSpace(governance.Agent.GetName())
	if run.AgentType == "" {
		run.AgentType = hrRecruitingAgentType
	}
	if run.AgentName == "" {
		run.AgentName = hrRecruitingAgentType
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return businessclock.FormatRFC3339(t)
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
