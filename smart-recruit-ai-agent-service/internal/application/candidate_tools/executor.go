// Package candidate_tools provides candidate AI builtin tool execution (DEV parity).
package candidate_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	commonsai "smart-recruit-commons/ai"
)

// ApplicationListItem is a candidate's application row for AI tools.
type ApplicationListItem struct {
	ApplicationID int64  `json:"application_id"`
	JobID         int64  `json:"job_id"`
	JobTitle      string `json:"job_title"`
	Status        int32  `json:"status"`
	StatusText    string `json:"status_text"`
	RoundNo       int32  `json:"round_no"`
	AppliedAt     string `json:"applied_at"`
}

// ApplicationDetail is a single application detail for AI tools.
type ApplicationDetail struct {
	ApplicationID int64  `json:"application_id"`
	JobID         int64  `json:"job_id"`
	JobTitle      string `json:"job_title"`
	Department    string `json:"department"`
	Location      string `json:"location"`
	SalaryRange   string `json:"salary_range"`
	Status        int32  `json:"status"`
	StatusText    string `json:"status_text"`
	RoundNo       int32  `json:"round_no"`
	AppliedAt     string `json:"applied_at"`
}

// ResumeText is the candidate resume payload for AI tools.
type ResumeText struct {
	Available  bool
	FileName   string
	TextLength int
	ResumeText string
	Message    string
}

// JobListItem is a job row with application mark for AI tools.
type JobListItem struct {
	JobID       int64  `json:"job_id"`
	Title       string `json:"title"`
	Department  string `json:"department"`
	Location    string `json:"location"`
	SalaryRange string `json:"salary_range"`
	Status      int32  `json:"status"`
	StatusText  string `json:"status_text"`
	HasApplied  bool   `json:"has_applied"`
}

// JobDetail is a job detail with application mark for AI tools.
type JobDetail struct {
	JobID        int64  `json:"job_id"`
	Title        string `json:"title"`
	Department   string `json:"department"`
	Location     string `json:"location"`
	SalaryRange  string `json:"salary_range"`
	Description  string `json:"description"`
	Requirements string `json:"requirements"`
	Status       int32  `json:"status"`
	StatusText   string `json:"status_text"`
	HasApplied   bool   `json:"has_applied"`
}

// DataStore is the persistence surface required by candidate tools.
// Implemented by adapters over NativeStore to avoid import cycles.
type DataStore interface {
	ListMyApplicationsForAI(ctx context.Context, userID int64, limit int32) ([]ApplicationListItem, error)
	GetMyApplicationDetailForAI(ctx context.Context, userID, applicationID int64) (ApplicationDetail, bool, error)
	GetMyResumeTextForAI(ctx context.Context, userID int64) (ResumeText, error)
	ListJobsForCandidateAI(ctx context.Context, userID int64, limit int32) ([]JobListItem, error)
	GetJobDetailForCandidateAI(ctx context.Context, userID, jobID int64) (JobDetail, bool, error)
}

// Executor implements commonsai.ToolRunner for candidate builtins.
type Executor struct {
	Store DataStore
}

// NewExecutor constructs a candidate tool executor.
func NewExecutor(store DataStore) *Executor {
	return &Executor{Store: store}
}

// Execute routes a model-selected tool call to domain data scoped by userID.
func (e *Executor) Execute(ctx context.Context, userID int64, toolName string, args map[string]any) (commonsai.ToolResult, error) {
	if e == nil || e.Store == nil {
		return commonsai.ToolResult{}, fmt.Errorf("candidate tool executor is not configured")
	}
	if userID <= 0 {
		return commonsai.ToolResult{}, fmt.Errorf("user_id is required")
	}
	if args == nil {
		args = map[string]any{}
	}
	switch strings.TrimSpace(toolName) {
	case "list_my_applications":
		return e.listMyApplications(ctx, userID)
	case "get_my_application_detail":
		return e.getMyApplicationDetail(ctx, userID, args)
	case "get_my_resume_text":
		return e.getMyResumeText(ctx, userID)
	case "list_jobs_for_recommendation":
		return e.listJobsForRecommendation(ctx, userID)
	case "get_job_detail_for_candidate":
		return e.getJobDetailForCandidate(ctx, userID, args)
	case "recommend_jobs_by_resume":
		return e.recommendJobsByResume(ctx, userID)
	default:
		return commonsai.ToolResult{}, fmt.Errorf("unknown candidate tool: %s", toolName)
	}
}

func (e *Executor) listMyApplications(ctx context.Context, userID int64) (commonsai.ToolResult, error) {
	rows, err := e.Store.ListMyApplicationsForAI(ctx, userID, 50)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if len(rows) == 0 {
		return jsonToolResult(map[string]any{
			"applications": []any{},
			"message":      "你目前还没有投递记录",
		})
	}
	return jsonToolResult(map[string]any{
		"total":        len(rows),
		"applications": rows,
	})
}

func (e *Executor) getMyApplicationDetail(ctx context.Context, userID int64, args map[string]any) (commonsai.ToolResult, error) {
	appID := int64Arg(args, "application_id")
	if appID <= 0 {
		return jsonToolResult(map[string]any{"error": "application_id is required"})
	}
	detail, found, err := e.Store.GetMyApplicationDetailForAI(ctx, userID, appID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if !found {
		return jsonToolResult(map[string]any{"error": "投递记录不存在或无权限访问"})
	}
	return jsonToolResult(detail)
}

func (e *Executor) getMyResumeText(ctx context.Context, userID int64) (commonsai.ToolResult, error) {
	resume, err := e.Store.GetMyResumeTextForAI(ctx, userID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if !resume.Available {
		payload := map[string]any{
			"resume_available": false,
			"message":          resume.Message,
		}
		if resume.FileName != "" {
			payload["file_name"] = resume.FileName
		}
		return jsonToolResult(payload)
	}
	return jsonToolResult(map[string]any{
		"resume_available": true,
		"file_name":        resume.FileName,
		"text_length":      resume.TextLength,
		"resume_text":      resume.ResumeText,
	})
}

func (e *Executor) listJobsForRecommendation(ctx context.Context, userID int64) (commonsai.ToolResult, error) {
	rows, err := e.Store.ListJobsForCandidateAI(ctx, userID, 100)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if rows == nil {
		rows = []JobListItem{}
	}
	return jsonToolResult(map[string]any{
		"total": len(rows),
		"jobs":  rows,
	})
}

func (e *Executor) getJobDetailForCandidate(ctx context.Context, userID int64, args map[string]any) (commonsai.ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	if jobID <= 0 {
		return jsonToolResult(map[string]any{"error": "job_id is required"})
	}
	detail, found, err := e.Store.GetJobDetailForCandidateAI(ctx, userID, jobID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if !found {
		return jsonToolResult(map[string]any{"error": "岗位不存在"})
	}
	return jsonToolResult(detail)
}

func (e *Executor) recommendJobsByResume(ctx context.Context, userID int64) (commonsai.ToolResult, error) {
	resume, err := e.Store.GetMyResumeTextForAI(ctx, userID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if !resume.Available || strings.TrimSpace(resume.ResumeText) == "" {
		return jsonToolResult(map[string]any{
			"error":   "no_resume",
			"message": "你还没有上传简历或简历解析文本为空。请先上传简历后，我才能为你推荐匹配的岗位。",
		})
	}
	jobs, err := e.Store.ListJobsForCandidateAI(ctx, userID, 100)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	entries := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		entries = append(entries, map[string]any{
			"job_id":       j.JobID,
			"title":        j.Title,
			"department":   j.Department,
			"location":     j.Location,
			"salary_range": j.SalaryRange,
			"status":       j.Status,
			"has_applied":  j.HasApplied,
		})
	}
	return jsonToolResult(map[string]any{
		"resume_text": resume.ResumeText,
		"resume_file": resume.FileName,
		"total_jobs":  len(entries),
		"jobs":        entries,
		"instruction": "请基于以上简历内容和岗位列表，为候选人推荐 3-5 个最匹配的岗位。对每个推荐岗位说明匹配理由、候选人的不足点、建议投递优先级。如果岗位的 has_applied 为 true，必须标注'已投递'。只输出推荐结果，不要输出其他内容。",
	})
}

func jsonToolResult(v any) (commonsai.ToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	return commonsai.ToolResult{Content: string(b)}, nil
}

func int64Arg(args map[string]any, key string) int64 {
	if args == nil {
		return 0
	}
	raw, ok := args[key]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		var n int64
		_, _ = fmt.Sscan(strings.TrimSpace(v), &n)
		return n
	default:
		return 0
	}
}
