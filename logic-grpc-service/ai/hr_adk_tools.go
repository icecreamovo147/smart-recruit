package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"logic-grpc-service/repository"
)

// contextKey is used for per-request values stored in context.Context.
type contextKey string

func (k contextKey) String() string { return "ai_context_" + string(k) }

const (
	contextKeyOwnerID contextKey = "owner_id"
	contextKeyState   contextKey = "state"
)

// WithOwnerID stores the user identifier (HR ID or candidate User ID) in the
// context so ADK tool closures can retrieve it without capturing per-request
// state, enabling tool caching across requests.
func WithOwnerID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, contextKeyOwnerID, id)
}

// WithAgentRunState stores the per-run AgentRunState in the context so tool
// closures can propagate metadata (CandidateOptions, Action) without capturing
// the state pointer at tool-creation time.
func WithAgentRunState(ctx context.Context, state *AgentRunState) context.Context {
	return context.WithValue(ctx, contextKeyState, state)
}

// ownerIDFromContext extracts the owner ID previously stored by WithOwnerID.
// Returns 0 if not found.
func ownerIDFromContext(ctx context.Context) int64 {
	if v, ok := ctx.Value(contextKeyOwnerID).(int64); ok {
		return v
	}
	return 0
}

// agentStateFromContext extracts the AgentRunState previously stored by
// WithAgentRunState. Returns nil if not found.
func agentStateFromContext(ctx context.Context) *AgentRunState {
	if v, ok := ctx.Value(contextKeyState).(*AgentRunState); ok {
		return v
	}
	return nil
}

// --- typed input/output structs ---

type queryTotalInput struct{}

type queryTotalOutput struct {
	TotalApplications int64 `json:"total_applications"`
}

type queryTodayInput struct {
	JobID int64 `json:"job_id,omitempty" jsonschema_description:"岗位 ID；不传则统计所有岗位"`
}

type queryTodayOutput struct {
	JobID             int64 `json:"job_id"`
	TodayApplications int64 `json:"today_applications"`
}

type jobHeatRankingInput struct {
	TopN int `json:"top_n,omitempty" jsonschema_description:"返回前几名，默认 5"`
}

type hotEntry struct {
	Title string `json:"title"`
	Total int64  `json:"total"`
}

type jobHeatRankingOutput struct {
	HotJobs []hotEntry `json:"hot_jobs"`
}

type searchCandidatesInput struct {
	Keyword string `json:"keyword" jsonschema:"required" jsonschema_description:"搜索关键词，匹配候选人姓名、电话或岗位名称"`
}

type candidateEntry struct {
	ApplicationID int64  `json:"application_id"`
	RealName      string `json:"real_name"`
	Phone         string `json:"phone"`
	JobTitle      string `json:"job_title"`
	Status        string `json:"status"`
	RoundNo       int32  `json:"round_no"`
	IsCurrent     int32  `json:"is_current"`
	AppliedAt     string `json:"applied_at"`
}

type searchCandidatesOutput struct {
	Candidates []candidateEntry `json:"candidates"`
	Message    string           `json:"message,omitempty"`
}

type getJobDetailInput struct {
	JobID int64 `json:"job_id" jsonschema:"required" jsonschema_description:"岗位 ID"`
}

type statusCountEntry struct {
	Status     int32  `json:"status"`
	StatusText string `json:"status_text"`
	Total      int64  `json:"total"`
}

type getJobDetailOutput struct {
	JobID             int64              `json:"job_id"`
	Title             string             `json:"title"`
	Department        string             `json:"department"`
	Location          string             `json:"location"`
	SalaryRange       string             `json:"salary_range"`
	Description       string             `json:"description"`
	Requirements      string             `json:"requirements"`
	Status            int32              `json:"status"`
	StatusText        string             `json:"status_text"`
	ApplicationCounts []statusCountEntry `json:"application_counts"`
	CreatedAt         string             `json:"created_at"`
	UpdatedAt         string             `json:"updated_at"`
}

type searchJobsInput struct {
	Keyword  string `json:"keyword,omitempty" jsonschema_description:"搜索关键词，可匹配岗位名称、部门、地点、描述或要求；为空时列出岗位"`
	Status   *int32 `json:"status,omitempty" jsonschema_description:"岗位状态：1 招募中，0 已下架；不传则不限状态"`
	Page     int32  `json:"page,omitempty" jsonschema_description:"页码，从 1 开始，默认 1"`
	PageSize int32  `json:"page_size,omitempty" jsonschema_description:"每页数量，默认 10，最多 50"`
}

type jobEntry struct {
	JobID            int64  `json:"job_id"`
	Title            string `json:"title"`
	Department       string `json:"department"`
	Location         string `json:"location"`
	SalaryRange      string `json:"salary_range"`
	Status           int32  `json:"status"`
	StatusText       string `json:"status_text"`
	ApplicationCount int64  `json:"application_count"`
	CreatedAt        string `json:"created_at"`
}

type searchJobsOutput struct {
	Total    int64      `json:"total"`
	Page     int32      `json:"page"`
	PageSize int32      `json:"page_size"`
	Jobs     []jobEntry `json:"jobs"`
}

type getCandidateDetailInput struct {
	ApplicationID int64 `json:"application_id" jsonschema:"required" jsonschema_description:"投递记录 ID"`
}

type getCandidateDetailOutput struct {
	ApplicationID int64  `json:"application_id"`
	CandidateName string `json:"candidate_name"`
	JobTitle      string `json:"job_title"`
	Department    string `json:"department"`
	Location      string `json:"location"`
	SalaryRange   string `json:"salary_range"`
	Description   string `json:"description"`
	Requirements  string `json:"requirements"`
	Status        int32  `json:"status"`
	RoundNo       int32  `json:"round_no"`
	ResumeFile    string `json:"resume_file"`
	ResumeNote    string `json:"resume_note"`
	ResumeText    string `json:"resume_text"`
	AppliedAt     string `json:"applied_at"`
}

type proposeStatusUpdateInput struct {
	ApplicationID int64 `json:"application_id" jsonschema:"required" jsonschema_description:"投递记录 ID"`
	Status        int32 `json:"status" jsonschema:"required" jsonschema_description:"目标状态：2 表示通过，3 表示淘汰"`
}

type proposeStatusUpdateOutput struct {
	Action        string `json:"action"`
	ApplicationID int64  `json:"application_id"`
	ActionStatus  int32  `json:"action_status"`
	CandidateName string `json:"candidate_name"`
	JobTitle      string `json:"job_title"`
	Message       string `json:"message"`
}

type listAllApplicationsInput struct {
	Page     int32 `json:"page,omitempty" jsonschema_description:"页码，从 1 开始，默认 1"`
	PageSize int32 `json:"page_size,omitempty" jsonschema_description:"每页数量，默认 10"`
}

type appEntry struct {
	ApplicationID int64  `json:"application_id"`
	JobID         int64  `json:"job_id"`
	JobTitle      string `json:"job_title"`
	RealName      string `json:"real_name"`
	Phone         string `json:"phone"`
	Education     string `json:"education"`
	School        string `json:"school"`
	Skills        string `json:"skills"`
	Status        int32  `json:"status"`
	StatusText    string `json:"status_text"`
	RoundNo       int32  `json:"round_no"`
	AppliedAt     string `json:"applied_at"`
}

type listApplicationsOutput struct {
	Total      int64      `json:"total"`
	Page       int32      `json:"page"`
	PageSize   int32      `json:"page_size"`
	Candidates []appEntry `json:"candidates"`
}

type listApplicationsByJobInput struct {
	JobID       int64  `json:"job_id" jsonschema:"required" jsonschema_description:"岗位 ID"`
	Status      *int32 `json:"status,omitempty" jsonschema_description:"投递状态：0 待查看，1 已查看，2 通过，3 淘汰；不传则不限状态"`
	CurrentOnly *bool  `json:"current_only,omitempty" jsonschema_description:"是否只看当前有效投递，默认 true"`
	Page        int32  `json:"page,omitempty" jsonschema_description:"页码，从 1 开始，默认 1"`
	PageSize    int32  `json:"page_size,omitempty" jsonschema_description:"每页数量，默认 10，最多 50"`
}

type listApplicationsByStatusInput struct {
	Status      int32 `json:"status" jsonschema:"required" jsonschema_description:"投递状态：0 待查看，1 已查看，2 通过，3 淘汰"`
	JobID       int64 `json:"job_id,omitempty" jsonschema_description:"岗位 ID；不传则查询所有岗位"`
	CurrentOnly *bool `json:"current_only,omitempty" jsonschema_description:"是否只看当前有效投递，默认 true"`
	Page        int32 `json:"page,omitempty" jsonschema_description:"页码，从 1 开始，默认 1"`
	PageSize    int32 `json:"page_size,omitempty" jsonschema_description:"每页数量，默认 10，最多 50"`
}

type applicationStatusSummaryInput struct {
	JobID int64 `json:"job_id,omitempty" jsonschema_description:"岗位 ID；不传则统计所有岗位"`
}

type applicationStatusSummaryOutput struct {
	JobID  int64              `json:"job_id"`
	Counts []statusCountEntry `json:"counts"`
}

type applicationTrendInput struct {
	Days  int   `json:"days,omitempty" jsonschema_description:"最近天数，默认 7，最多 90"`
	JobID int64 `json:"job_id,omitempty" jsonschema_description:"岗位 ID；不传则统计所有岗位"`
}

type applicationTrendOutput struct {
	JobID int64                            `json:"job_id"`
	Days  int                              `json:"days"`
	Total int64                            `json:"total"`
	Trend []repository.ApplicationTrendRow `json:"trend"`
}

type getJobListOutput struct {
	Jobs []jobEntry `json:"jobs"`
}

// NewRecruitingADKTools creates all HR tools as tool.BaseTool with typed
// input/output via utils.InferTool. Each tool delegates to *ToolExecutor so
// business logic lives in a single place. Each tool is wrapped with a uniform
// error handler that returns JSON error strings so the model can correct
// itself.
//
// The executor is captured at creation time (allowing tools to be cached), but
// per-request values (owner ID, AgentRunState) are retrieved from context at
// call time via WithOwnerID / WithAgentRunState.
func NewRecruitingADKTools(executor *ToolExecutor) ([]tool.BaseTool, error) {
	if executor == nil {
		return nil, fmt.Errorf("tool executor must not be nil")
	}

	errorHandler := func(ctx context.Context, err error) string {
		data, _ := json.Marshal(map[string]any{
			"error":   true,
			"message": err.Error(),
		})
		return string(data)
	}

	queryTotalTool, err := utils.InferTool(
		"query_total_applications",
		"查询当前 HR 所有岗位的累计投递总数",
		func(ctx context.Context, input queryTotalInput) (queryTotalOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			result, err := executor.Execute(ctx, ownerID, "query_total_applications", nil)
			if err != nil {
				return queryTotalOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out queryTotalOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return queryTotalOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("query_total_applications: %w", err)
	}

	queryTodayTool, err := utils.InferTool(
		"query_today_applications",
		"查询今日新增投递数，可按岗位限定。用户询问今天某岗位投递了多少人时，先定位岗位 ID，再调用此工具",
		func(ctx context.Context, input queryTodayInput) (queryTodayOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{}
			if input.JobID > 0 {
				args["job_id"] = input.JobID
			}
			result, err := executor.Execute(ctx, ownerID, "query_today_applications", args)
			if err != nil {
				return queryTodayOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out queryTodayOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return queryTodayOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("query_today_applications: %w", err)
	}

	heatRankingTool, err := utils.InferTool(
		"get_job_heat_ranking",
		"查询投递数最高的岗位排行",
		func(ctx context.Context, input jobHeatRankingInput) (jobHeatRankingOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{}
			if input.TopN > 0 {
				args["top_n"] = input.TopN
			}
			result, err := executor.Execute(ctx, ownerID, "get_job_heat_ranking", args)
			if err != nil {
				return jobHeatRankingOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out jobHeatRankingOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return jobHeatRankingOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get_job_heat_ranking: %w", err)
	}

	searchCandidatesTool, err := utils.InferTool(
		"search_candidates",
		"搜索候选人投递记录，用于定位候选人或投递记录。姓名精确匹配，电话和岗位名称支持模糊匹配。需要确定候选人对应 application_id 时优先使用此工具",
		func(ctx context.Context, input searchCandidatesInput) (searchCandidatesOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{"keyword": input.Keyword}
			result, err := executor.Execute(ctx, ownerID, "search_candidates", args)
			if err != nil {
				return searchCandidatesOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			// The executor may return a JSON result with either {"candidates":...}
			// or an empty list with a message field.
			var out searchCandidatesOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return searchCandidatesOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("search_candidates: %w", err)
	}

	getJobDetailTool, err := utils.InferTool(
		"get_job_detail",
		"查询指定岗位的完整信息、岗位要求以及该岗位的投递状态分布",
		func(ctx context.Context, input getJobDetailInput) (getJobDetailOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{"job_id": input.JobID}
			result, err := executor.Execute(ctx, ownerID, "get_job_detail", args)
			if err != nil {
				return getJobDetailOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out getJobDetailOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return getJobDetailOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get_job_detail: %w", err)
	}

	searchJobsTool, err := utils.InferTool(
		"search_jobs",
		"按关键词和上下架状态搜索当前 HR 发布的岗位。用户提到岗位名称、部门、地点等模糊条件，或需要先定位岗位 ID 时优先使用此工具",
		func(ctx context.Context, input searchJobsInput) (searchJobsOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{}
			if input.Keyword != "" {
				args["keyword"] = input.Keyword
			}
			if input.Status != nil {
				args["status"] = *input.Status
			}
			if input.Page > 0 {
				args["page"] = input.Page
			}
			if input.PageSize > 0 {
				args["page_size"] = input.PageSize
			}
			result, err := executor.Execute(ctx, ownerID, "search_jobs", args)
			if err != nil {
				return searchJobsOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out searchJobsOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return searchJobsOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("search_jobs: %w", err)
	}

	getCandidateDetailTool, err := utils.InferTool(
		"get_candidate_detail",
		"获取指定投递记录的完整信息，包括岗位信息和候选人上传简历的解析正文，用于简历分析和岗位匹配度评估",
		func(ctx context.Context, input getCandidateDetailInput) (getCandidateDetailOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{"application_id": input.ApplicationID}
			result, err := executor.Execute(ctx, ownerID, "get_candidate_detail", args)
			if err != nil {
				return getCandidateDetailOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out getCandidateDetailOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return getCandidateDetailOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get_candidate_detail: %w", err)
	}

	proposeStatusUpdateTool, err := utils.InferTool(
		"propose_application_status_update",
		"当 HR 明确要求对某个投递进行状态变更时调用，此工具只生成待确认动作，不会直接修改数据库",
		func(ctx context.Context, input proposeStatusUpdateInput) (proposeStatusUpdateOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{
				"application_id": input.ApplicationID,
				"status":         input.Status,
			}
			result, err := executor.Execute(ctx, ownerID, "propose_application_status_update", args)
			if err != nil {
				return proposeStatusUpdateOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out proposeStatusUpdateOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return proposeStatusUpdateOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("propose_application_status_update: %w", err)
	}

	listAllApplicationsTool, err := utils.InferTool(
		"list_all_applications",
		"分页列出当前 HR 所有岗位下的全部投递候选人，用于浏览整体候选人情况",
		func(ctx context.Context, input listAllApplicationsInput) (listApplicationsOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{}
			if input.Page > 0 {
				args["page"] = input.Page
			}
			if input.PageSize > 0 {
				args["page_size"] = input.PageSize
			}
			result, err := executor.Execute(ctx, ownerID, "list_all_applications", args)
			if err != nil {
				return listApplicationsOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out listApplicationsOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return listApplicationsOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list_all_applications: %w", err)
	}

	listAppsByJobTool, err := utils.InferTool(
		"list_applications_by_job",
		"分页列出指定岗位下的投递候选人，可按投递状态筛选",
		func(ctx context.Context, input listApplicationsByJobInput) (listApplicationsOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{"job_id": input.JobID}
			if input.Status != nil {
				args["status"] = *input.Status
			}
			if input.CurrentOnly != nil {
				args["current_only"] = *input.CurrentOnly
			}
			if input.Page > 0 {
				args["page"] = input.Page
			}
			if input.PageSize > 0 {
				args["page_size"] = input.PageSize
			}
			result, err := executor.Execute(ctx, ownerID, "list_applications_by_job", args)
			if err != nil {
				return listApplicationsOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out listApplicationsOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return listApplicationsOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list_applications_by_job: %w", err)
	}

	listAppsByStatusTool, err := utils.InferTool(
		"list_applications_by_status",
		"分页列出当前 HR 所有岗位下某个状态的投递候选人，也可限定岗位",
		func(ctx context.Context, input listApplicationsByStatusInput) (listApplicationsOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{"status": input.Status}
			if input.JobID > 0 {
				args["job_id"] = input.JobID
			}
			if input.CurrentOnly != nil {
				args["current_only"] = *input.CurrentOnly
			}
			if input.Page > 0 {
				args["page"] = input.Page
			}
			if input.PageSize > 0 {
				args["page_size"] = input.PageSize
			}
			result, err := executor.Execute(ctx, ownerID, "list_applications_by_status", args)
			if err != nil {
				return listApplicationsOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out listApplicationsOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return listApplicationsOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list_applications_by_status: %w", err)
	}

	statusSummaryTool, err := utils.InferTool(
		"get_application_status_summary",
		"查询投递状态分布统计，可按岗位限定",
		func(ctx context.Context, input applicationStatusSummaryInput) (applicationStatusSummaryOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{}
			if input.JobID > 0 {
				args["job_id"] = input.JobID
			}
			result, err := executor.Execute(ctx, ownerID, "get_application_status_summary", args)
			if err != nil {
				return applicationStatusSummaryOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out applicationStatusSummaryOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return applicationStatusSummaryOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get_application_status_summary: %w", err)
	}

	trendTool, err := utils.InferTool(
		"get_application_trend",
		"查询近 N 天投递趋势，可按岗位限定。若只问今天投递人数，优先使用 query_today_applications",
		func(ctx context.Context, input applicationTrendInput) (applicationTrendOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			args := map[string]any{}
			if input.Days > 0 {
				args["days"] = input.Days
			}
			if input.JobID > 0 {
				args["job_id"] = input.JobID
			}
			result, err := executor.Execute(ctx, ownerID, "get_application_trend", args)
			if err != nil {
				return applicationTrendOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out applicationTrendOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return applicationTrendOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get_application_trend: %w", err)
	}

	getJobListTool, err := utils.InferTool(
		"get_job_list",
		"查询当前 HR 发布的所有在招岗位列表",
		func(ctx context.Context, _ struct{}) (getJobListOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			state := agentStateFromContext(ctx)
			result, err := executor.Execute(ctx, ownerID, "get_job_list", nil)
			if err != nil {
				return getJobListOutput{}, err
			}
			if state != nil {
				state.Merge(result.Metadata)
			}
			var out getJobListOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return getJobListOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get_job_list: %w", err)
	}

	baseTools := []tool.BaseTool{
		queryTotalTool, queryTodayTool, heatRankingTool,
		searchCandidatesTool, getJobDetailTool, searchJobsTool,
		getCandidateDetailTool, proposeStatusUpdateTool,
		listAllApplicationsTool, listAppsByJobTool, listAppsByStatusTool,
		statusSummaryTool, trendTool, getJobListTool,
	}

	result := make([]tool.BaseTool, 0, len(baseTools))
	for _, t := range baseTools {
		result = append(result, utils.WrapToolWithErrorHandler(t, errorHandler))
	}
	return result, nil
}
