package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"logic-grpc-service/model"
	"logic-grpc-service/oss"
	"logic-grpc-service/repository"
	"logic-grpc-service/resumeparser"
)

// ToolExecutor bridges LLM tool calls to MySQL repository methods.
type ToolExecutor struct {
	applications *repository.ApplicationRepo
	jobs         *repository.JobRepo
	resumes      *repository.ResumeRepo
	oss          oss.Storage
}

type ToolResult struct {
	Content  string
	Metadata ToolMetadata
}

type ToolMetadata struct {
	CandidateOptions []ToolCandidateOption
	Action           *ToolAction
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
}

// recordTrace appends a single tool execution to the metadata trace log.
func (m *ToolMetadata) recordTrace(t ToolTrace) {
	m.ToolTraces = append(m.ToolTraces, t)
}

func NewToolExecutor(apps *repository.ApplicationRepo, jobs *repository.JobRepo, resumes *repository.ResumeRepo, ossClient oss.Storage) *ToolExecutor {
	return &ToolExecutor{applications: apps, jobs: jobs, resumes: resumes, oss: ossClient}
}

// Execute runs the named tool with the given JSON-decoded arguments for the specified HR.
func (e *ToolExecutor) Execute(ctx context.Context, hrID int64, toolName string, args map[string]any) (ToolResult, error) {
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
		ok, err := e.jobs.BelongsToHR(ctx, hrID, jobID)
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
		statusText := applicationStatusTextTool(r.Status)
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
	ok, err := e.jobs.BelongsToHR(ctx, hrID, jobID)
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
		ok, err := e.jobs.BelongsToHR(ctx, hrID, jobID)
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
		ok, err := e.jobs.BelongsToHR(ctx, hrID, jobID)
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
	var status int32
	switch v := args["status"].(type) {
	case float64:
		status = int32(v)
	case int32:
		status = v
	case int:
		status = int32(v)
	}
	if appID <= 0 {
		return ToolResult{Content: `{"error": "application_id is required"}`}, nil
	}
	if status != 2 && status != 3 {
		return ToolResult{Content: `{"error": "status must be 2(通过) or 3(淘汰)"}`}, nil
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
			StatusText:    applicationStatusTextTool(r.Status),
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
			"status_text": applicationStatusTextTool(row.Status),
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
