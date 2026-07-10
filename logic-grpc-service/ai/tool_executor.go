package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
	"logic-grpc-service/resumeparser"
)

// ToolExecutor bridges LLM tool calls to MySQL repository methods.
type ToolExecutor struct {
	applications     *repository.ApplicationRepo
	jobs             *repository.JobRepo
	resumes          *repository.ResumeRepo
	profiles         *repository.ProfileRepo
	resumeProfiles   *repository.ResumeProfileRepo
	candidateMatches *repository.CandidateMatchRepo
	oss              oss.Storage
	authzRepo        *repository.AuthzRepo // optional: for scope-aware access checks
}

type ToolResult struct {
	Content  string
	Metadata ToolMetadata
}

type ToolMetadata struct {
	CandidateOptions []ToolCandidateOption
	Action           *ToolAction

	// BillingTokenUsage accumulates every model call's token usage in a single
	// user action. Used for audit/cost tracking.
	BillingTokenUsage *schema.TokenUsage
	// ContextTokenUsage is overwritten by each model call and holds the latest
	// model call's token usage. Used for context window occupancy display.
	ContextTokenUsage *schema.TokenUsage

	// ToolTraces accumulates one entry per executed tool call within a single
	// ChatWithTools invocation. Populated by ChatWithTools in eino_client.go.
	// Used by fallback-reply builders when the LLM fails after tools succeeded.
	ToolTraces []ToolTrace
}

// ToolTrace is the in-memory record of a single tool execution kept for fallback
// reply construction. Result is the raw JSON returned by the executor (already
// safe to inspect); Cost is the wall-clock time the tool took.
type ToolTrace struct {
	ToolName  string
	Arguments map[string]any
	Result    string
	Cost      time.Duration
	Error     error
}

type ToolCandidateOption struct {
	ApplicationID int64  `json:"application_id"`
	CandidateName string `json:"candidate_name"`
	MaskedPhone   string `json:"masked_phone"`
	JobTitle      string `json:"job_title"`
	StatusText    string `json:"status_text"`
	RoundNo       int32  `json:"round_no"`
	IsCurrent     int32  `json:"is_current"`
	AppliedAt     string `json:"applied_at"`
}

type ToolAction struct {
	Action        string
	ApplicationID int64
	ActionStatus  int32
	CandidateName string
	JobTitle      string
	Status        int32
}

func (m *ToolMetadata) merge(other ToolMetadata) {
	// CandidateOptions is append-only: multiple tools may each contribute
	// candidate records (e.g., search_candidates called multiple times).
	if len(other.CandidateOptions) > 0 {
		m.CandidateOptions = append(m.CandidateOptions, other.CandidateOptions...)
	}
	// Action is overwrite: at most one status-update action per agent turn.
	if other.Action != nil {
		m.Action = other.Action
	}
	m.addBillingTokenUsage(other.BillingTokenUsage)
	if other.ContextTokenUsage != nil {
		m.setContextTokenUsage(other.ContextTokenUsage)
	}
}

// recordTrace appends a single tool execution to the metadata trace log.
func (m *ToolMetadata) recordTrace(t ToolTrace) {
	m.ToolTraces = append(m.ToolTraces, t)
}

// addBillingTokenUsage accumulates usage into BillingTokenUsage.
func (m *ToolMetadata) addBillingTokenUsage(usage *schema.TokenUsage) {
	if usage == nil {
		return
	}
	if m.BillingTokenUsage == nil {
		m.BillingTokenUsage = &schema.TokenUsage{}
	}
	m.BillingTokenUsage.PromptTokens += usage.PromptTokens
	m.BillingTokenUsage.PromptTokenDetails.CachedTokens += usage.PromptTokenDetails.CachedTokens
	m.BillingTokenUsage.CompletionTokens += usage.CompletionTokens
	m.BillingTokenUsage.CompletionTokensDetails.ReasoningTokens += usage.CompletionTokensDetails.ReasoningTokens
	m.BillingTokenUsage.TotalTokens += usage.TotalTokens
}

// setContextTokenUsage overwrites ContextTokenUsage with the given usage.
// This represents the latest model call's token usage for context window display.
func (m *ToolMetadata) setContextTokenUsage(usage *schema.TokenUsage) {
	if usage == nil {
		return
	}
	if m.ContextTokenUsage == nil {
		m.ContextTokenUsage = &schema.TokenUsage{}
	}
	*m.ContextTokenUsage = *usage
}

// recordModelUsage records a model call's token usage for both billing
// (cumulative) and context (latest-call) semantics.
func (m *ToolMetadata) recordModelUsage(usage *schema.TokenUsage) {
	m.addBillingTokenUsage(usage)
	m.setContextTokenUsage(usage)
}

func NewToolExecutor(apps *repository.ApplicationRepo, jobs *repository.JobRepo, resumes *repository.ResumeRepo, ossClient oss.Storage, authzRepo *repository.AuthzRepo, intelligenceRepos ...any) *ToolExecutor {
	executor := &ToolExecutor{applications: apps, jobs: jobs, resumes: resumes, oss: ossClient, authzRepo: authzRepo}
	if len(intelligenceRepos) > 0 {
		if repo, ok := intelligenceRepos[0].(*repository.ProfileRepo); ok {
			executor.profiles = repo
		}
	}
	if len(intelligenceRepos) > 1 {
		if repo, ok := intelligenceRepos[1].(*repository.ResumeProfileRepo); ok {
			executor.resumeProfiles = repo
		}
	}
	if len(intelligenceRepos) > 2 {
		if repo, ok := intelligenceRepos[2].(*repository.CandidateMatchRepo); ok {
			executor.candidateMatches = repo
		}
	}
	return executor
}

// hasJobAccess checks whether the given HR user has access to the specified job,
// considering both ownership and data scopes. Returns true if access is allowed.
func (e *ToolExecutor) hasJobAccess(ctx context.Context, hrID int64, jobID int64) (bool, error) {
	// First check ownership (fast path).
	if e.jobs == nil {
		return false, nil
	}
	owned, err := e.jobs.BelongsToHR(ctx, hrID, jobID)
	if err != nil {
		return false, err
	}
	if owned {
		return true, nil
	}

	// If authzRepo is available, also check data scopes.
	if e.authzRepo == nil {
		return false, nil
	}

	scopeKeys, err := e.authzRepo.GetUserScopeKeys(ctx, uint64(hrID))
	if err != nil {
		return false, err
	}

	for _, sk := range scopeKeys {
		switch sk {
		case authz.ScopeRecruitingAll, authz.ScopeSystemAll:
			// Full access — no ownership or scope filtering needed.
			// Verify the job exists (BelongsToHR already confirmed it doesn't belong to HR).
			job, jerr := e.jobs.GetByID(ctx, jobID)
			if jerr != nil {
				return false, jerr
			}
			if job != nil {
				return true, nil
			}
		case authz.ScopeDepartment:
			deptIDs, derr := e.authzRepo.GetUserDepartmentIDs(ctx, uint64(hrID))
			if derr != nil {
				return false, derr
			}
			job, jerr := e.jobs.GetByID(ctx, jobID)
			if jerr != nil {
				return false, jerr
			}
			if job != nil && job.DepartmentID != nil {
				for _, dID := range deptIDs {
					if uint64(*job.DepartmentID) == dID {
						return true, nil
					}
				}
			}
		case authz.ScopeLocation:
			locIDs, lerr := e.authzRepo.GetUserLocationIDs(ctx, uint64(hrID))
			if lerr != nil {
				return false, lerr
			}
			job, jerr := e.jobs.GetByID(ctx, jobID)
			if jerr != nil {
				return false, jerr
			}
			if job != nil && job.LocationID != nil {
				for _, lID := range locIDs {
					if uint64(*job.LocationID) == lID {
						return true, nil
					}
				}
			}
		case authz.ScopeOwnJobs:
			// Already checked via BelongsToHR above.
		}
	}
	return false, nil
}

// Execute runs the named tool with the given JSON-decoded arguments for the specified HR.
func (e *ToolExecutor) Execute(ctx context.Context, hrID int64, toolName string, args map[string]any) (ToolResult, error) {
	started := time.Now()
	result, err := e.executeInner(ctx, hrID, toolName, args)
	durationMs := time.Since(started).Milliseconds()
	status := "succeeded"
	if err != nil {
		status = "failed"
	}
	logger.L().Info("[logic][tool_executor] tool execution finished",
		zap.String("tool_name", toolName),
		zap.Int64("hr_id", hrID),
		zap.String("status", status),
		zap.Int64("duration_ms", durationMs),
		zap.Int("result_length", len(result.Content)),
	)
	if err != nil {
		logger.L().Warn("[logic][tool_executor] tool execution failed",
			zap.String("tool_name", toolName),
			zap.Int64("hr_id", hrID),
			zap.Int64("duration_ms", durationMs),
			zap.Error(err))
	}
	return result, err
}

func (e *ToolExecutor) executeInner(ctx context.Context, hrID int64, toolName string, args map[string]any) (ToolResult, error) {
	logger.L().Info("[logic][tool_executor] tool execution started",
		zap.String("tool_name", toolName),
		zap.Int64("hr_id", hrID))
	switch toolName {
	case "query_total_applications":
		return e.queryTotal(ctx, hrID)
	case "query_today_applications":
		return e.queryToday(ctx, hrID, args)
	case "get_job_heat_ranking":
		return e.jobHeatRanking(ctx, hrID, args)
	case "search_candidates":
		return e.searchCandidates(ctx, hrID, args)
	case "list_all_applications":
		return e.listAllApplications(ctx, hrID, args)
	case "get_candidate_detail":
		return e.candidateDetail(ctx, hrID, args)
	case "get_job_list":
		return e.jobList(ctx, hrID)
	case "get_job_detail":
		return e.jobDetail(ctx, hrID, args)
	case "search_jobs":
		return e.searchJobs(ctx, hrID, args)
	case "list_applications_by_job":
		return e.listApplicationsByJob(ctx, hrID, args)
	case "list_applications_by_status":
		return e.listApplicationsByStatus(ctx, hrID, args)
	case "get_application_status_summary":
		return e.applicationStatusSummary(ctx, hrID, args)
	case "get_application_trend":
		return e.applicationTrend(ctx, hrID, args)
	case "propose_application_status_update":
		return e.proposeApplicationStatusUpdate(ctx, hrID, args)
	case "parse_resume_profile":
		return e.parseResumeProfile(ctx, hrID, args)
	case "get_resume_profile":
		return e.getResumeProfile(ctx, hrID, args)
	case "evaluate_candidate_match":
		return e.evaluateCandidateMatch(ctx, hrID, args)
	case "get_candidate_match_evaluation":
		return e.getCandidateMatchEvaluation(ctx, hrID, args)
	case "compare_candidates_for_job":
		return e.compareCandidatesForJob(ctx, hrID, args)
	default:
		return ToolResult{}, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (e *ToolExecutor) queryTotal(ctx context.Context, hrID int64) (ToolResult, error) {
	total, err := e.applications.TotalByHR(ctx, hrID)
	if err != nil {
		return ToolResult{}, err
	}
	b, _ := json.Marshal(map[string]any{"total_applications": total})
	return ToolResult{Content: string(b)}, nil
}

func (e *ToolExecutor) queryToday(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	if jobID > 0 {
		ok, err := e.hasJobAccess(ctx, hrID, jobID)
		if err != nil {
			return ToolResult{}, err
		}
		if !ok {
			return ToolResult{Content: `{"error": "岗位不存在或无权限访问"}`}, nil
		}
	}
	today, err := e.applications.TodayByHR(ctx, hrID, jobID)
	if err != nil {
		return ToolResult{}, err
	}
	b, _ := json.Marshal(map[string]any{"job_id": jobID, "today_applications": today})
	return ToolResult{Content: string(b)}, nil
}

func (e *ToolExecutor) jobHeatRanking(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	topN := 5
	if v, ok := args["top_n"]; ok {
		switch n := v.(type) {
		case float64:
			topN = int(n)
		case int:
			topN = n
		}
	}
	rows, err := e.applications.HotJobs(ctx, hrID, topN)
	if err != nil {
		return ToolResult{}, err
	}
	type hotEntry struct {
		Title string `json:"title"`
		Total int64  `json:"total"`
	}
	entries := make([]hotEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, hotEntry{Title: r.Title, Total: r.Total})
	}
	b, _ := json.Marshal(map[string]any{"hot_jobs": entries})
	return ToolResult{Content: string(b)}, nil
}

func (e *ToolExecutor) searchCandidates(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	keyword, _ := args["keyword"].(string)
	if strings.TrimSpace(keyword) == "" {
		return ToolResult{Content: `{"error": "keyword is required"}`}, nil
	}
	rows, err := e.applications.SearchCandidateApplications(ctx, hrID, keyword, 10)
	if err != nil {
		return ToolResult{}, err
	}
	if len(rows) == 0 {
		return ToolResult{Content: fmt.Sprintf(`{"candidates": [], "message": "未找到与「%s」相关的投递记录"}`, keyword)}, nil
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
	entries := make([]candidateEntry, 0, len(rows))
	options := make([]ToolCandidateOption, 0, len(rows))
	for _, r := range rows {
		statusText := applicationStatusTextPreferred(r.StatusKey, r.Status)
		name := r.RealName
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("候选人 %d", r.UserID)
		}
		appliedAt := r.AppliedAt.Format("2006-01-02 15:04")
		entries = append(entries, candidateEntry{
			ApplicationID: r.ApplicationID,
			RealName:      name,
			Phone:         maskPhoneTool(r.Phone),
			JobTitle:      r.JobTitle,
			Status:        statusText,
			RoundNo:       r.RoundNo,
			IsCurrent:     r.IsCurrent,
			AppliedAt:     appliedAt,
		})
		options = append(options, ToolCandidateOption{
			ApplicationID: r.ApplicationID,
			CandidateName: name,
			MaskedPhone:   maskPhoneTool(r.Phone),
			JobTitle:      r.JobTitle,
			StatusText:    statusText,
			RoundNo:       r.RoundNo,
			IsCurrent:     r.IsCurrent,
			AppliedAt:     appliedAt,
		})
	}
	b, _ := json.Marshal(map[string]any{"candidates": entries})
	result := ToolResult{Content: string(b)}
	if len(options) > 1 {
		result.Metadata.CandidateOptions = options
	}
	return result, nil
}

func (e *ToolExecutor) candidateDetail(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	var appID int64
	switch v := args["application_id"].(type) {
	case float64:
		appID = int64(v)
	case int64:
		appID = v
	}
	if appID <= 0 {
		return ToolResult{Content: `{"error": "application_id is required"}`}, nil
	}
	detail, err := e.applications.GetDetailOwned(ctx, hrID, appID)
	if err != nil {
		return ToolResult{}, err
	}
	if detail == nil {
		return ToolResult{Content: `{"error": "投递记录不存在或无权限访问"}`}, nil
	}

	resumeText := detail.ParsedText
	resumeNote := "简历文本暂未解析"
	if strings.TrimSpace(resumeText) == "" && detail.OSSKey != "" && e.oss != nil {
		data, err := e.oss.DownloadObject(ctx, detail.OSSKey)
		if err == nil {
			if len(data) >= 4 && !resumeparser.ValidateMagicBytes(detail.FileType, data[:min(len(data), 8)]) {
				resumeNote = fmt.Sprintf("候选人上传的简历文件头与声明格式 %s 不匹配，已停止解析。", strings.ToUpper(detail.FileType))
			} else if parser, parserErr := resumeparser.DefaultRegistry.GetParser(detail.FileType); parserErr == nil {
				text, extractErr := parser.ExtractText(ctx, data)
				if extractErr == nil && strings.TrimSpace(text) != "" {
					resumeText = text
					resumeNote = fmt.Sprintf("已从 OSS 读取简历，提取文本约 %d 个字符", len([]rune(text)))
					_ = e.resumes.UpdateParsedText(ctx, detail.ResumeID, text)
				}
			}
		}
	} else if strings.TrimSpace(resumeText) != "" {
		resumeNote = fmt.Sprintf("已使用缓存的简历解析文本，约 %d 个字符", len([]rune(resumeText)))
	}

	analysisText, stats := resumeparser.PrepareForAnalysis(resumeText)
	if !resumeparser.IsAnalysisTextUseful(analysisText, stats) {
		analysisText = ""
		resumeNote += " 解析文本经过净化后仍疑似乱码或有效信息不足，不得据此编造经历。"
	} else if stats.RemovedLines > 0 || stats.CleanedChars < stats.OriginalChars {
		resumeNote += fmt.Sprintf(" 已在提交 AI 前过滤乱码/重复噪声，保留有效文本约 %d 个字符，移除疑似噪声行 %d 行。", stats.CleanedChars, stats.RemovedLines)
	}

	result := map[string]any{
		"application_id": detail.ApplicationID,
		"candidate_name": detail.RealName,
		"job_title":      detail.JobTitle,
		"department":     detail.Department,
		"location":       detail.Location,
		"salary_range":   detail.SalaryRange,
		"description":    detail.Description,
		"requirements":   detail.Requirements,
		"status":         detail.Status,
		"status_key":     detail.StatusKey,
		"status_text":    applicationStatusTextPreferred(detail.StatusKey, detail.Status),
		"round_no":       detail.RoundNo,
		"resume_file":    detail.FileName,
		"resume_note":    resumeNote,
		"resume_text":    analysisText,
		"applied_at":     detail.AppliedAt.Format(time.RFC3339),
	}
	b, _ := json.Marshal(result)
	return ToolResult{Content: string(b)}, nil
}

func (e *ToolExecutor) jobList(ctx context.Context, hrID int64) (ToolResult, error) {
	jobs, _, err := e.jobs.ListByHR(ctx, hrID, 1, 100)
	if err != nil {
		return ToolResult{}, err
	}
	type jobEntry struct {
		ID          int64  `json:"job_id"`
		Title       string `json:"title"`
		Department  string `json:"department"`
		Location    string `json:"location"`
		SalaryRange string `json:"salary_range"`
		Status      int32  `json:"status"`
		StatusText  string `json:"status_text"`
	}
	entries := make([]jobEntry, 0, len(jobs))
	for _, j := range jobs {
		st := "招募中"
		if j.Status == 0 {
			st = "已下架"
		}
		entries = append(entries, jobEntry{
			ID: j.ID, Title: j.Title, Department: j.Department,
			Location: j.Location, SalaryRange: j.SalaryRange,
			Status: j.Status, StatusText: st,
		})
	}
	b, _ := json.Marshal(map[string]any{"jobs": entries})
	return ToolResult{Content: string(b)}, nil
}

func (e *ToolExecutor) jobDetail(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	if jobID <= 0 {
		return ToolResult{Content: `{"error": "job_id is required"}`}, nil
	}
	job, err := e.jobs.GetOwned(ctx, hrID, jobID)
	if err != nil {
		return ToolResult{}, err
	}
	if job == nil {
		return ToolResult{Content: `{"error": "岗位不存在或无权限访问"}`}, nil
	}
	counts, err := e.applications.StatusSummaryByHR(ctx, hrID, jobID)
	if err != nil {
		return ToolResult{}, err
	}
	statusCounts := statusCountMap(counts)
	result := map[string]any{
		"job_id":             job.ID,
		"title":              job.Title,
		"department":         job.Department,
		"location":           job.Location,
		"salary_range":       job.SalaryRange,
		"description":        job.Description,
		"requirements":       job.Requirements,
		"status":             job.Status,
		"status_text":        jobStatusTextTool(job.Status),
		"application_counts": statusCounts,
		"created_at":         job.CreatedAt.Format(time.RFC3339),
		"updated_at":         job.UpdatedAt.Format(time.RFC3339),
	}
	b, _ := json.Marshal(result)
	return ToolResult{Content: string(b)}, nil
}

func (e *ToolExecutor) searchJobs(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	keyword, _ := args["keyword"].(string)
	page, pageSize := pageArgs(args)
	var status *int32
	if v, ok := optionalInt32Arg(args, "status"); ok {
		status = &v
	}
	jobs, total, err := e.jobs.SearchByHR(ctx, hrID, strings.TrimSpace(keyword), status, page, pageSize)
	if err != nil {
		return ToolResult{}, err
	}
	counts, err := e.jobs.BatchApplicationCounts(ctx, jobIDs(jobs))
	if err != nil {
		return ToolResult{}, err
	}
	entries := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		entries = append(entries, map[string]any{
			"job_id":            job.ID,
			"title":             job.Title,
			"department":        job.Department,
			"location":          job.Location,
			"salary_range":      job.SalaryRange,
			"status":            job.Status,
			"status_text":       jobStatusTextTool(job.Status),
			"application_count": counts[job.ID],
			"created_at":        job.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	b, _ := json.Marshal(map[string]any{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"jobs":      entries,
	})
	return ToolResult{Content: string(b)}, nil
}

func (e *ToolExecutor) listApplicationsByJob(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	if jobID <= 0 {
		return ToolResult{Content: `{"error": "job_id is required"}`}, nil
	}
	ok, err := e.hasJobAccess(ctx, hrID, jobID)
	if err != nil {
		return ToolResult{}, err
	}
	if !ok {
		return ToolResult{Content: `{"error": "岗位不存在或无权限访问"}`}, nil
	}
	page, pageSize := pageArgs(args)
	filter := repository.ApplicationListFilter{JobID: jobID, CurrentOnly: boolArg(args, "current_only", true)}
	if v, ok := optionalInt32Arg(args, "status"); ok {
		filter.Status = &v
	}
	rows, total, err := e.applications.ListByHRFiltered(ctx, hrID, filter, page, pageSize)
	if err != nil {
		return ToolResult{}, err
	}
	return e.applicationRowsResult(rows, total, page, pageSize)
}

func (e *ToolExecutor) listApplicationsByStatus(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	status, ok := optionalInt32Arg(args, "status")
	if !ok {
		return ToolResult{Content: `{"error": "status is required"}`}, nil
	}
	page, pageSize := pageArgs(args)
	filter := repository.ApplicationListFilter{Status: &status, CurrentOnly: boolArg(args, "current_only", true)}
	if jobID := int64Arg(args, "job_id"); jobID > 0 {
		filter.JobID = jobID
	}
	rows, total, err := e.applications.ListByHRFiltered(ctx, hrID, filter, page, pageSize)
	if err != nil {
		return ToolResult{}, err
	}
	return e.applicationRowsResult(rows, total, page, pageSize)
}

func (e *ToolExecutor) applicationStatusSummary(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	if jobID > 0 {
		ok, err := e.hasJobAccess(ctx, hrID, jobID)
		if err != nil {
			return ToolResult{}, err
		}
		if !ok {
			return ToolResult{Content: `{"error": "岗位不存在或无权限访问"}`}, nil
		}
	}
	rows, err := e.applications.StatusSummaryByHR(ctx, hrID, jobID)
	if err != nil {
		return ToolResult{}, err
	}
	b, _ := json.Marshal(map[string]any{
		"job_id": jobID,
		"counts": statusCountMap(rows),
	})
	return ToolResult{Content: string(b)}, nil
}

func (e *ToolExecutor) applicationTrend(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	days := intArg(args, "days", 7)
	if days < 1 {
		days = 7
	}
	if days > 90 {
		days = 90
	}
	jobID := int64Arg(args, "job_id")
	if jobID > 0 {
		ok, err := e.hasJobAccess(ctx, hrID, jobID)
		if err != nil {
			return ToolResult{}, err
		}
		if !ok {
			return ToolResult{Content: `{"error": "岗位不存在或无权限访问"}`}, nil
		}
	}
	rows, err := e.applications.TrendByHR(ctx, hrID, jobID, days)
	if err != nil {
		return ToolResult{}, err
	}
	if rows == nil {
		rows = make([]repository.ApplicationTrendRow, 0)
	}
	var total int64
	for _, row := range rows {
		total += row.Total
	}
	b, _ := json.Marshal(map[string]any{
		"job_id": jobID,
		"days":   days,
		"total":  total,
		"trend":  rows,
	})
	return ToolResult{Content: string(b)}, nil
}

func (e *ToolExecutor) proposeApplicationStatusUpdate(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	var appID int64
	switch v := args["application_id"].(type) {
	case float64:
		appID = int64(v)
	case int64:
		appID = v
	}
	// Accept either status_key (string) or legacy status (numeric 2/3).
	var status int32
	var statusKey string
	if sk, ok := args["status_key"].(string); ok && sk != "" {
		statusKey = sk
		// Map status_key to legacy numeric for Action compatibility.
		if legacy, ok := model.StatusKeyToLegacy[statusKey]; ok {
			status = legacy
		}
	} else {
		switch v := args["status"].(type) {
		case float64:
			status = int32(v)
		case int32:
			status = v
		case int:
			status = int32(v)
		}
	}
	if appID <= 0 {
		return ToolResult{Content: `{"error": "application_id is required"}`}, nil
	}
	if statusKey == "" && status != 2 && status != 3 {
		return ToolResult{Content: `{"error": "status must be 2(通过) or 3(淘汰), or provide a valid status_key"}`}, nil
	}
	if statusKey != "" {
		if _, ok := model.StatusKeyToLegacy[statusKey]; !ok {
			return ToolResult{Content: fmt.Sprintf(`{"error": "invalid status_key: %s"}`, statusKey)}, nil
		}
	}
	detail, err := e.applications.GetDetailOwned(ctx, hrID, appID)
	if err != nil {
		return ToolResult{}, err
	}
	if detail == nil {
		return ToolResult{Content: `{"error": "投递记录不存在或无权限访问"}`}, nil
	}
	action := "reject_application"
	if status == 2 {
		action = "approve_application"
	}
	name := strings.TrimSpace(detail.RealName)
	if name == "" {
		name = fmt.Sprintf("候选人 %d", detail.UserID)
	}
	content := fmt.Sprintf(`{"action": "%s", "application_id": %d, "action_status": %d, "candidate_name": "%s", "job_title": "%s", "message": "请向 HR 请求确认后再更新状态，当前工具不会直接修改数据库。"}`,
		action, detail.ApplicationID, status, jsonEscape(name), jsonEscape(detail.JobTitle))
	return ToolResult{
		Content: content,
		Metadata: ToolMetadata{Action: &ToolAction{
			Action:        action,
			ApplicationID: detail.ApplicationID,
			ActionStatus:  status,
			CandidateName: name,
			JobTitle:      detail.JobTitle,
			Status:        detail.Status,
		}},
	}, nil
}

func (e *ToolExecutor) listAllApplications(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	page, pageSize := pageArgs(args)
	rows, total, err := e.applications.ListAllByHR(ctx, hrID, page, pageSize)
	if err != nil {
		return ToolResult{}, err
	}
	return e.applicationRowsResult(rows, total, page, pageSize)
}

func (e *ToolExecutor) applicationRowsResult(rows []repository.JobApplicationRow, total int64, page, pageSize int32) (ToolResult, error) {
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
		StatusKey     string `json:"status_key"`
		StatusText    string `json:"status_text"`
		RoundNo       int32  `json:"round_no"`
		AppliedAt     string `json:"applied_at"`
	}
	entries := make([]appEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, appEntry{
			ApplicationID: r.ApplicationID,
			JobID:         r.JobID,
			JobTitle:      r.JobTitle,
			RealName:      r.RealName,
			Phone:         maskPhoneTool(r.Phone),
			Education:     r.Education,
			School:        r.School,
			Skills:        r.Skills,
			Status:        r.Status,
			StatusKey:     r.StatusKey,
			StatusText:    applicationStatusTextPreferred(r.StatusKey, r.Status),
			RoundNo:       r.RoundNo,
			AppliedAt:     r.AppliedAt.Format("2006-01-02 15:04"),
		})
	}
	b, _ := json.Marshal(map[string]any{
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"candidates": entries,
	})
	return ToolResult{Content: string(b)}, nil
}

func (e *ToolExecutor) parseResumeProfile(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	if e.resumes == nil || e.resumeProfiles == nil {
		return ToolResult{}, fmt.Errorf("resume profile tools are not configured")
	}
	resumeID, application, err := e.resolveResumeContext(ctx, args)
	if err != nil {
		return ToolResult{}, err
	}
	if err := e.authorizeResumeContext(ctx, hrID, resumeID, application); err != nil {
		return ToolResult{}, err
	}
	resume, err := e.resumes.GetByID(ctx, resumeID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("get resume: %w", err)
	}
	if resume == nil {
		return ToolResult{}, fmt.Errorf("resume not found")
	}
	text := strings.TrimSpace(resume.ParsedText)
	if text == "" {
		return ToolResult{}, fmt.Errorf("resume parsed_text is empty")
	}
	snapshot := e.buildResumeProfileSnapshot(ctx, resume, application, text)
	if err := e.resumeProfiles.SaveProfileVersion(ctx, snapshot); err != nil {
		return ToolResult{}, fmt.Errorf("save resume profile: %w", err)
	}
	return jsonToolResult(map[string]any{
		"tool":    "parse_resume_profile",
		"source":  "generated",
		"profile": resumeProfileSnapshotMap(snapshot),
		"trace":   map[string]any{"resume_id": resume.ID, "application_id": applicationIDOrZero(application), "profile_id": snapshot.Profile.ID, "version": snapshot.Profile.Version},
	})
}

func (e *ToolExecutor) getResumeProfile(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	if e.resumeProfiles == nil {
		return ToolResult{}, fmt.Errorf("resume profile tools are not configured")
	}
	profileID := uint64(int64Arg(args, "profile_id"))
	resumeID, application, err := e.resolveResumeContext(ctx, args)
	if err != nil && profileID == 0 {
		return ToolResult{}, err
	}
	if profileID > 0 {
		profile, err := e.resumeProfiles.GetByID(ctx, profileID)
		if err != nil {
			return ToolResult{}, fmt.Errorf("get resume profile: %w", err)
		}
		if profile == nil {
			return ToolResult{}, fmt.Errorf("resume profile not found")
		}
		if resumeID > 0 && profile.ResumeID != resumeID {
			return ToolResult{}, fmt.Errorf("profile_id does not belong to requested resume/application")
		}
		resumeID = profile.ResumeID
	}
	if err := e.authorizeResumeContext(ctx, hrID, resumeID, application); err != nil {
		return ToolResult{}, err
	}
	if profileID == 0 {
		profile, err := e.resumeProfiles.GetCurrentByResumeID(ctx, resumeID)
		if err != nil {
			return ToolResult{}, fmt.Errorf("get current resume profile: %w", err)
		}
		if profile == nil {
			return ToolResult{}, fmt.Errorf("current resume profile not found")
		}
		profileID = profile.ID
	}
	snapshot, err := e.resumeProfiles.GetSnapshot(ctx, profileID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("get resume profile snapshot: %w", err)
	}
	return jsonToolResult(map[string]any{
		"tool":    "get_resume_profile",
		"source":  "stored",
		"profile": resumeProfileSnapshotMap(snapshot),
		"trace":   map[string]any{"resume_id": snapshot.Profile.ResumeID, "application_id": applicationIDOrZero(application), "profile_id": snapshot.Profile.ID, "version": snapshot.Profile.Version},
	})
}

func (e *ToolExecutor) evaluateCandidateMatch(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	snapshot, source, err := e.evaluateCandidateMatchSnapshot(ctx, hrID, int64Arg(args, "application_id"), true)
	if err != nil {
		return ToolResult{}, err
	}
	return jsonToolResult(map[string]any{
		"tool":       "evaluate_candidate_match",
		"source":     source,
		"evaluation": candidateMatchSnapshotMap(snapshot),
		"trace":      candidateMatchTrace(snapshot),
	})
}

func (e *ToolExecutor) getCandidateMatchEvaluation(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	if e.candidateMatches == nil {
		return ToolResult{}, fmt.Errorf("candidate match tools are not configured")
	}
	applicationID := int64Arg(args, "application_id")
	if applicationID <= 0 {
		return ToolResult{}, fmt.Errorf("application_id is required")
	}
	application, err := e.authorizeApplication(ctx, hrID, applicationID)
	if err != nil {
		return ToolResult{}, err
	}
	var evaluation *model.CandidateMatchEvaluation
	if evaluationID := uint64(int64Arg(args, "evaluation_id")); evaluationID > 0 {
		snapshot, err := e.candidateMatches.GetSnapshot(ctx, evaluationID)
		if err == nil && snapshot.Evaluation.ApplicationID == application.ID {
			evaluation = &snapshot.Evaluation
		}
	} else if version := int32(intArg(args, "evaluation_version", 0)); version > 0 {
		evaluation, err = e.candidateMatches.GetByApplicationIDAndVersion(ctx, applicationID, version)
		if err != nil {
			return ToolResult{}, fmt.Errorf("get candidate match evaluation: %w", err)
		}
	} else {
		evaluation, err = e.candidateMatches.GetLatestByApplicationID(ctx, applicationID)
		if err != nil {
			return ToolResult{}, fmt.Errorf("get candidate match evaluation: %w", err)
		}
	}
	if evaluation == nil {
		return ToolResult{}, fmt.Errorf("candidate match evaluation not found")
	}
	snapshot, err := e.candidateMatches.GetSnapshot(ctx, evaluation.ID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("get candidate match snapshot: %w", err)
	}
	return jsonToolResult(map[string]any{
		"tool":       "get_candidate_match_evaluation",
		"source":     "stored",
		"evaluation": candidateMatchSnapshotMap(snapshot),
		"trace":      candidateMatchTrace(snapshot),
	})
}

func (e *ToolExecutor) compareCandidatesForJob(ctx context.Context, hrID int64, args map[string]any) (ToolResult, error) {
	if e.applications == nil || e.candidateMatches == nil {
		return ToolResult{}, fmt.Errorf("candidate comparison tools are not configured")
	}
	jobID := int64Arg(args, "job_id")
	if jobID <= 0 {
		return ToolResult{}, fmt.Errorf("job_id is required")
	}
	ok, err := e.hasJobAccess(ctx, hrID, jobID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("authorize job: %w", err)
	}
	if !ok {
		return ToolResult{}, fmt.Errorf("job not found or access denied")
	}
	rows, err := e.applications.ListCurrentByJob(ctx, jobID)
	if err != nil {
		return ToolResult{}, fmt.Errorf("list job applications: %w", err)
	}
	items := make([]map[string]any, 0, len(rows))
	missing := make([]int64, 0)
	for _, row := range rows {
		evaluation, err := e.candidateMatches.GetLatestByApplicationID(ctx, row.ApplicationID)
		source := "stored"
		if err != nil {
			return ToolResult{}, fmt.Errorf("get candidate match evaluation: %w", err)
		}
		if evaluation == nil {
			generated, generatedSource, err := e.evaluateCandidateMatchSnapshot(ctx, hrID, row.ApplicationID, true)
			if err != nil {
				missing = append(missing, row.ApplicationID)
				items = append(items, comparisonItemMap(row, nil, "missing: "+err.Error()))
				continue
			}
			evaluation = &generated.Evaluation
			source = generatedSource
		}
		items = append(items, comparisonItemMap(row, evaluation, source))
	}
	sort.SliceStable(items, func(i, j int) bool {
		iHas, _ := items[i]["has_evaluation"].(bool)
		jHas, _ := items[j]["has_evaluation"].(bool)
		if iHas != jHas {
			return iHas
		}
		iScore, _ := items[i]["overall_score"].(float64)
		jScore, _ := items[j]["overall_score"].(float64)
		if iScore != jScore {
			return iScore > jScore
		}
		return items[i]["application_id"].(int64) < items[j]["application_id"].(int64)
	})
	return jsonToolResult(map[string]any{
		"tool":                    "compare_candidates_for_job",
		"job_id":                  jobID,
		"candidates":              items,
		"missing_application_ids": missing,
		"trace":                   map[string]any{"job_id": jobID, "candidate_count": len(items)},
	})
}

func pageArgs(args map[string]any) (int32, int32) {
	page := int32(intArg(args, "page", 1))
	pageSize := int32(intArg(args, "page_size", 10))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	return page, pageSize
}

func (e *ToolExecutor) resolveResumeContext(ctx context.Context, args map[string]any) (int64, *model.Application, error) {
	applicationID := int64Arg(args, "application_id")
	resumeID := int64Arg(args, "resume_id")
	if applicationID <= 0 && resumeID <= 0 {
		return 0, nil, fmt.Errorf("resume_id or application_id is required")
	}
	if applicationID > 0 {
		if e.applications == nil {
			return 0, nil, fmt.Errorf("application repository is not configured")
		}
		application, err := e.applications.GetByID(ctx, applicationID)
		if err != nil {
			return 0, nil, fmt.Errorf("get application: %w", err)
		}
		if application == nil || application.ResumeID <= 0 {
			return 0, nil, fmt.Errorf("application resume not found")
		}
		if resumeID > 0 && resumeID != application.ResumeID {
			return 0, nil, fmt.Errorf("application_id and resume_id refer to different resumes")
		}
		return application.ResumeID, application, nil
	}
	return resumeID, nil, nil
}

func (e *ToolExecutor) authorizeResumeContext(ctx context.Context, hrID int64, resumeID int64, application *model.Application) error {
	if hrID <= 0 {
		return fmt.Errorf("hr_id is required")
	}
	if resumeID <= 0 {
		return fmt.Errorf("resume_id is required")
	}
	if application == nil {
		if e.applications == nil {
			return fmt.Errorf("application repository is not configured")
		}
		latest, err := e.applications.GetLatestByResumeID(ctx, resumeID)
		if err != nil {
			return fmt.Errorf("get application by resume: %w", err)
		}
		if latest == nil {
			return fmt.Errorf("application context not found for resume")
		}
		application = latest
	}
	ok, err := e.hasJobAccess(ctx, hrID, application.JobID)
	if err != nil {
		return fmt.Errorf("authorize job: %w", err)
	}
	if !ok {
		return fmt.Errorf("application not found or access denied")
	}
	return nil
}

func (e *ToolExecutor) authorizeApplication(ctx context.Context, hrID int64, applicationID int64) (*model.Application, error) {
	if applicationID <= 0 {
		return nil, fmt.Errorf("application_id is required")
	}
	if e.applications == nil {
		return nil, fmt.Errorf("application repository is not configured")
	}
	application, err := e.applications.GetByID(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}
	if application == nil {
		return nil, fmt.Errorf("application not found")
	}
	ok, err := e.hasJobAccess(ctx, hrID, application.JobID)
	if err != nil {
		return nil, fmt.Errorf("authorize job: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("application not found or access denied")
	}
	return application, nil
}

func (e *ToolExecutor) buildResumeProfileSnapshot(ctx context.Context, resume *model.Resume, application *model.Application, text string) *repository.ResumeProfileSnapshot {
	now := time.Now().UTC()
	name, email, phone := resumeIdentityFromText(text)
	if e.profiles != nil {
		userID := resume.UserID
		if application != nil && application.UserID > 0 {
			userID = application.UserID
		}
		if profile, err := e.profiles.GetByUserID(ctx, userID); err == nil && profile != nil {
			if strings.TrimSpace(name) == "" {
				name = strings.TrimSpace(profile.RealName)
			}
			if strings.TrimSpace(phone) == "" {
				phone = strings.TrimSpace(profile.Phone)
			}
		}
	}
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("候选人 %d", resume.UserID)
	}
	skills := skillsFromText(text)
	skillRows := make([]model.ResumeSkill, 0, len(skills))
	for i, skill := range skills {
		skillRows = append(skillRows, model.ResumeSkill{Name: skill, Category: "inferred", Evidence: snippetForTerm(text, skill), SortOrder: int32(i + 1)})
	}
	raw, _ := json.Marshal(map[string]any{
		"full_name":              name,
		"email":                  email,
		"phone":                  phone,
		"summary":                truncateRunes(strings.Join(strings.Fields(text), " "), 600),
		"total_experience_years": inferredExperienceYears(text),
		"skills":                 skills,
	})
	return &repository.ResumeProfileSnapshot{
		ParseRun: model.ResumeParseRun{
			ResumeID:      resume.ID,
			UserID:        resume.UserID,
			Status:        "succeeded",
			ParserVersion: "ai-tool-resume-profile-v1",
			InputHash:     sha256Hex(text),
			StartedAt:     now,
			CompletedAt:   &now,
		},
		Profile: model.ResumeProfile{
			FullName:        name,
			Email:           email,
			Phone:           phone,
			Headline:        inferredHeadline(text, skills),
			Summary:         truncateRunes(strings.Join(strings.Fields(text), " "), 1000),
			TotalExperience: inferredExperienceYears(text),
			HighestDegree:   inferredDegree(text),
			RawJSON:         string(raw),
		},
		Skills: skillRows,
	}
}

func (e *ToolExecutor) evaluateCandidateMatchSnapshot(ctx context.Context, hrID int64, applicationID int64, createMissingProfile bool) (*repository.CandidateMatchSnapshot, string, error) {
	if e.jobs == nil || e.resumes == nil || e.resumeProfiles == nil || e.candidateMatches == nil {
		return nil, "", fmt.Errorf("candidate match tools are not configured")
	}
	application, err := e.authorizeApplication(ctx, hrID, applicationID)
	if err != nil {
		return nil, "", err
	}
	job, err := e.jobs.GetByID(ctx, application.JobID)
	if err != nil {
		return nil, "", fmt.Errorf("get job: %w", err)
	}
	if job == nil {
		return nil, "", fmt.Errorf("job not found")
	}
	resume, err := e.resumes.GetByID(ctx, application.ResumeID)
	if err != nil {
		return nil, "", fmt.Errorf("get resume: %w", err)
	}
	if resume == nil {
		return nil, "", fmt.Errorf("resume not found")
	}
	profile, err := e.resumeProfiles.GetCurrentByResumeID(ctx, resume.ID)
	if err != nil {
		return nil, "", fmt.Errorf("get current resume profile: %w", err)
	}
	source := "generated"
	if profile == nil {
		if !createMissingProfile {
			return nil, "", fmt.Errorf("current resume profile not found")
		}
		text := strings.TrimSpace(resume.ParsedText)
		if text == "" {
			return nil, "", fmt.Errorf("resume parsed_text is empty")
		}
		profileSnapshot := e.buildResumeProfileSnapshot(ctx, resume, application, text)
		if err := e.resumeProfiles.SaveProfileVersion(ctx, profileSnapshot); err != nil {
			return nil, "", fmt.Errorf("save resume profile: %w", err)
		}
		profile = &profileSnapshot.Profile
		source = "generated_with_profile"
	}
	resumeSnapshot, err := e.resumeProfiles.GetSnapshot(ctx, profile.ID)
	if err != nil {
		return nil, "", fmt.Errorf("get resume profile snapshot: %w", err)
	}
	match := buildCandidateMatchSnapshot(job, application, resumeSnapshot)
	if err := e.candidateMatches.SaveEvaluationVersion(ctx, match); err != nil {
		return nil, "", fmt.Errorf("save candidate match evaluation: %w", err)
	}
	return match, source, nil
}

func buildCandidateMatchSnapshot(job *model.Job, application *model.Application, profile *repository.ResumeProfileSnapshot) *repository.CandidateMatchSnapshot {
	jobTerms := tokenSetTool(strings.Join([]string{job.Title, job.Department, job.Location, job.Description, job.Requirements}, " "))
	resumeText := strings.Join([]string{profile.Profile.Summary, profile.Profile.Headline, profile.Profile.HighestDegree, resumeSkillText(profile.Skills)}, " ")
	resumeTerms := tokenSetTool(resumeText)
	matched, missing := splitTokenMatches(jobTerms, resumeTerms)
	score := 45.0
	if len(jobTerms) > 0 {
		score += 50 * float64(len(matched)) / float64(len(jobTerms))
	}
	if profile.Profile.TotalExperience > 0 {
		score += math.Min(profile.Profile.TotalExperience, 10)
	}
	if profile.Profile.HighestDegree != "" {
		score += 3
	}
	score = math.Min(100, math.Round(score*10)/10)
	recommendation := "review"
	if score >= 80 {
		recommendation = "strong_match"
	} else if score < 60 {
		recommendation = "weak_match"
	}
	strengths, _ := json.Marshal([]map[string]any{{"code": "matched_terms", "message": fmt.Sprintf("Matched %d job terms", len(matched)), "terms": matched}})
	risks, _ := json.Marshal([]map[string]any{{"code": "missing_terms", "message": fmt.Sprintf("Missing %d job terms", len(missing)), "terms": missing}})
	breakdown, _ := json.Marshal(map[string]any{
		"scorer_version":       "ai-tool-candidate-match-v1",
		"missing_requirements": missing,
		"dimensions":           []map[string]any{{"name": "term_coverage", "weight": 0.8, "score": score, "matched": matched, "missing": missing}},
	})
	now := time.Now().UTC()
	evidence := make([]model.CandidateMatchEvidence, 0, len(matched)+len(missing))
	for _, term := range matched {
		evidence = append(evidence, model.CandidateMatchEvidence{EvidenceType: "strength", Dimension: "requirements", SourceTable: "resume_profiles", SourceID: &profile.Profile.ID, Snippet: term, Weight: 1, ScoreImpact: 1})
	}
	for _, term := range missing {
		evidence = append(evidence, model.CandidateMatchEvidence{EvidenceType: "risk", Dimension: "requirements", SourceTable: "jobs", Snippet: term, Weight: 1, ScoreImpact: -1})
	}
	return &repository.CandidateMatchSnapshot{
		Evaluation: model.CandidateMatchEvaluation{
			ApplicationID:      application.ID,
			JobID:              job.ID,
			CandidateUserID:    application.UserID,
			ResumeProfileID:    profile.Profile.ID,
			OverallScore:       score,
			Recommendation:     recommendation,
			Summary:            fmt.Sprintf("Score %.1f based on %d matched and %d missing job terms.", score, len(matched), len(missing)),
			StrengthsJSON:      string(strengths),
			RisksJSON:          string(risks),
			ScoreBreakdownJSON: string(breakdown),
			ModelName:          "ai-tool-candidate-match-v1",
			EvaluatedAt:        now,
		},
		Evidence: evidence,
	}
}

func intArg(args map[string]any, key string, fallback int) int {
	if args == nil {
		return fallback
	}
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	default:
		return fallback
	}
}

func jsonToolResult(payload map[string]any) (ToolResult, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Content: string(b)}, nil
}

func applicationIDOrZero(application *model.Application) int64 {
	if application == nil {
		return 0
	}
	return application.ID
}

func sha256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func resumeIdentityFromText(text string) (string, string, string) {
	fields := strings.Fields(text)
	name := ""
	if len(fields) > 0 && len([]rune(fields[0])) <= 32 {
		name = strings.Trim(fields[0], " ,;:")
	}
	email := ""
	phone := ""
	for _, field := range fields {
		cleaned := strings.Trim(field, " ,;:()[]<>")
		if email == "" && strings.Contains(cleaned, "@") && strings.Contains(cleaned, ".") {
			email = cleaned
		}
		digits := onlyDigits(cleaned)
		if phone == "" && len(digits) >= 7 {
			phone = digits
		}
	}
	return name, email, phone
}

func skillsFromText(text string) []string {
	catalog := []string{"go", "golang", "java", "python", "javascript", "typescript", "vue", "react", "mysql", "postgresql", "redis", "kubernetes", "docker", "grpc", "api", "microservice", "spring", "linux", "aws", "azure"}
	lower := strings.ToLower(text)
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, skill := range catalog {
		if strings.Contains(lower, skill) {
			label := skill
			if skill == "golang" {
				label = "go"
			}
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			out = append(out, label)
		}
	}
	sort.Strings(out)
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

func inferredExperienceYears(text string) float64 {
	lower := strings.ToLower(text)
	for _, token := range strings.Fields(lower) {
		token = strings.Trim(token, "+,.;:")
		if strings.HasSuffix(token, "years") || strings.HasSuffix(token, "year") {
			continue
		}
		var years float64
		if _, err := fmt.Sscanf(token, "%f", &years); err == nil && years >= 0 && years <= 60 {
			if strings.Contains(lower, fmt.Sprintf("%g years", years)) || strings.Contains(lower, fmt.Sprintf("%g year", years)) || strings.Contains(text, fmt.Sprintf("%g年", years)) {
				return years
			}
		}
	}
	return 0
}

func inferredDegree(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "phd"), strings.Contains(lower, "doctor"), strings.Contains(text, "博士"):
		return "博士"
	case strings.Contains(lower, "master"), strings.Contains(text, "硕士"):
		return "硕士"
	case strings.Contains(lower, "bachelor"), strings.Contains(text, "本科"), strings.Contains(text, "学士"):
		return "本科"
	default:
		return ""
	}
}

func inferredHeadline(text string, skills []string) string {
	if len(skills) > 0 {
		return "Experienced in " + strings.Join(skills, ", ")
	}
	return truncateRunes(strings.Join(strings.Fields(text), " "), 80)
}

func truncateRunes(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func onlyDigits(text string) string {
	var b strings.Builder
	for _, r := range text {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func snippetForTerm(text, term string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, strings.ToLower(term))
	if idx < 0 {
		return ""
	}
	start := idx - 80
	if start < 0 {
		start = 0
	}
	end := idx + len(term) + 80
	if end > len(text) {
		end = len(text)
	}
	return strings.TrimSpace(text[start:end])
}

func resumeSkillText(skills []model.ResumeSkill) string {
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		names = append(names, skill.Name, skill.Evidence)
	}
	return strings.Join(names, " ")
}

func tokenSetTool(text string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, raw := range strings.Fields(strings.ToLower(text)) {
		token := strings.Trim(raw, " \t\r\n,.;:()[]{}<>!?\"'")
		if len(token) < 2 || toolStopwords[token] {
			continue
		}
		out[token] = struct{}{}
	}
	return out
}

var toolStopwords = map[string]bool{
	"and": true, "or": true, "the": true, "for": true, "with": true, "from": true, "this": true, "that": true,
	"岗位": true, "要求": true, "负责": true, "具备": true, "熟悉": true, "经验": true,
}

func splitTokenMatches(required, actual map[string]struct{}) ([]string, []string) {
	matched := make([]string, 0)
	missing := make([]string, 0)
	for token := range required {
		if _, ok := actual[token]; ok {
			matched = append(matched, token)
		} else {
			missing = append(missing, token)
		}
	}
	sort.Strings(matched)
	sort.Strings(missing)
	if len(missing) > 20 {
		missing = missing[:20]
	}
	return matched, missing
}

func resumeProfileSnapshotMap(snapshot *repository.ResumeProfileSnapshot) map[string]any {
	if snapshot == nil {
		return nil
	}
	skills := make([]map[string]any, 0, len(snapshot.Skills))
	for _, skill := range snapshot.Skills {
		skills = append(skills, map[string]any{"id": skill.ID, "name": skill.Name, "category": skill.Category, "level": skill.Level, "years": skill.Years, "evidence": skill.Evidence})
	}
	return map[string]any{
		"parse_run": map[string]any{"id": snapshot.ParseRun.ID, "resume_id": snapshot.ParseRun.ResumeID, "status": snapshot.ParseRun.Status, "parser_version": snapshot.ParseRun.ParserVersion, "input_hash": snapshot.ParseRun.InputHash, "error_message": snapshot.ParseRun.ErrorMessage},
		"profile":   map[string]any{"id": snapshot.Profile.ID, "resume_id": snapshot.Profile.ResumeID, "user_id": snapshot.Profile.UserID, "version": snapshot.Profile.Version, "is_current": snapshot.Profile.IsCurrent, "full_name": snapshot.Profile.FullName, "email": snapshot.Profile.Email, "phone": snapshot.Profile.Phone, "location": snapshot.Profile.Location, "headline": snapshot.Profile.Headline, "summary": snapshot.Profile.Summary, "total_experience_years": snapshot.Profile.TotalExperience, "highest_degree": snapshot.Profile.HighestDegree},
		"skills":    skills,
	}
}

func candidateMatchSnapshotMap(snapshot *repository.CandidateMatchSnapshot) map[string]any {
	if snapshot == nil {
		return nil
	}
	evidence := make([]map[string]any, 0, len(snapshot.Evidence))
	for _, row := range snapshot.Evidence {
		evidence = append(evidence, map[string]any{"id": row.ID, "evidence_type": row.EvidenceType, "dimension": row.Dimension, "source_table": row.SourceTable, "source_id": row.SourceID, "snippet": row.Snippet, "weight": row.Weight, "score_impact": row.ScoreImpact})
	}
	row := snapshot.Evaluation
	return map[string]any{
		"evaluation": map[string]any{"id": row.ID, "application_id": row.ApplicationID, "job_id": row.JobID, "candidate_user_id": row.CandidateUserID, "resume_profile_id": row.ResumeProfileID, "evaluation_version": row.EvaluationVersion, "is_latest": row.IsLatest, "overall_score": row.OverallScore, "recommendation": row.Recommendation, "summary": row.Summary, "strengths_json": row.StrengthsJSON, "risks_json": row.RisksJSON, "score_breakdown_json": row.ScoreBreakdownJSON, "model_name": row.ModelName, "evaluated_at": row.EvaluatedAt.Format(time.RFC3339)},
		"evidence":   evidence,
	}
}

func candidateMatchTrace(snapshot *repository.CandidateMatchSnapshot) map[string]any {
	if snapshot == nil {
		return nil
	}
	return map[string]any{"evaluation_id": snapshot.Evaluation.ID, "application_id": snapshot.Evaluation.ApplicationID, "job_id": snapshot.Evaluation.JobID, "resume_profile_id": snapshot.Evaluation.ResumeProfileID, "evaluation_version": snapshot.Evaluation.EvaluationVersion}
}

func comparisonItemMap(row repository.JobApplicationRow, evaluation *model.CandidateMatchEvaluation, source string) map[string]any {
	item := map[string]any{
		"application_id":    row.ApplicationID,
		"candidate_user_id": row.UserID,
		"candidate_name":    row.RealName,
		"resume_id":         row.ResumeID,
		"has_evaluation":    evaluation != nil,
		"source":            source,
	}
	if evaluation != nil {
		item["evaluation_id"] = evaluation.ID
		item["evaluation_version"] = evaluation.EvaluationVersion
		item["overall_score"] = evaluation.OverallScore
		item["recommendation"] = evaluation.Recommendation
		item["summary"] = evaluation.Summary
		item["evaluated_at"] = evaluation.EvaluatedAt.Format(time.RFC3339)
	}
	return item
}

func int64Arg(args map[string]any, key string) int64 {
	if args == nil {
		return 0
	}
	switch v := args[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

func optionalInt32Arg(args map[string]any, key string) (int32, bool) {
	if args == nil {
		return 0, false
	}
	switch v := args[key].(type) {
	case float64:
		return int32(v), true
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	default:
		return 0, false
	}
}

func boolArg(args map[string]any, key string, fallback bool) bool {
	if args == nil {
		return fallback
	}
	if v, ok := args[key].(bool); ok {
		return v
	}
	return fallback
}

func jobIDs(jobs []model.Job) []int64 {
	ids := make([]int64, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.ID)
	}
	return ids
}

func statusCountMap(rows []repository.ApplicationStatusCountRow) []map[string]any {
	counts := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		counts = append(counts, map[string]any{
			"status":      row.Status,
			"status_key":  row.StatusKey,
			"status_text": applicationStatusTextPreferred(row.StatusKey, row.Status),
			"total":       row.Total,
		})
	}
	return counts
}

func maskPhoneTool(phone string) string {
	phone = strings.TrimSpace(phone)
	runes := []rune(phone)
	if len(runes) < 7 {
		return phone
	}
	return string(runes[:3]) + "****" + string(runes[len(runes)-4:])
}

func applicationStatusTextTool(status int32) string {
	if text, ok := model.HRStatusLabels[model.LegacyStatusToKey[status]]; ok {
		return text
	}
	switch status {
	case 0:
		return "待查看"
	case 1:
		return "已查看"
	case 2:
		return "通过"
	case 3:
		return "淘汰"
	default:
		return "未知"
	}
}

// applicationStatusTextPreferred returns the HR-facing status text.
// Prefers statusKey (new status machine) and falls back to legacy numeric mapping.
func applicationStatusTextPreferred(statusKey string, status int32) string {
	if statusKey != "" {
		if text, ok := model.HRStatusLabels[statusKey]; ok {
			return text
		}
	}
	return applicationStatusTextTool(status)
}

func applicationStatusTextByKey(key string) string {
	if text, ok := model.HRStatusLabels[key]; ok {
		return text
	}
	return "未知"
}

func jobStatusTextTool(status int32) string {
	if status == 1 {
		return "招募中"
	}
	return "已下架"
}

func jsonEscape(value string) string {
	b, _ := json.Marshal(value)
	return strings.Trim(string(b), `"`)
}
