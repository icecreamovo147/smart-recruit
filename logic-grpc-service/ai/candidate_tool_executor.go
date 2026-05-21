package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"logic-grpc-service/repository"
)

// CandidateToolExecutor bridges candidate LLM tool calls to repository methods.
type CandidateToolExecutor struct {
	applications *repository.ApplicationRepo
	jobs         *repository.JobRepo
	resumes      *repository.ResumeRepo
}

func NewCandidateToolExecutor(apps *repository.ApplicationRepo, jobs *repository.JobRepo, resumes *repository.ResumeRepo) *CandidateToolExecutor {
	return &CandidateToolExecutor{applications: apps, jobs: jobs, resumes: resumes}
}

func (e *CandidateToolExecutor) Execute(ctx context.Context, userID int64, toolName string, args map[string]any) (ToolResult, error) {
	switch toolName {
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
		return ToolResult{}, fmt.Errorf("unknown candidate tool: %s", toolName)
	}
}

func (e *CandidateToolExecutor) listMyApplications(ctx context.Context, userID int64) (ToolResult, error) {
	rows, _, err := e.applications.ListMy(ctx, userID, 1, 50)
	if err != nil {
		return ToolResult{}, err
	}

	type appEntry struct {
		ApplicationID int64  `json:"application_id"`
		JobID         int64  `json:"job_id"`
		JobTitle      string `json:"job_title"`
		Status        int32  `json:"status"`
		StatusText    string `json:"status_text"`
		RoundNo       int32  `json:"round_no"`
		AppliedAt     string `json:"applied_at"`
	}

	if len(rows) == 0 {
		return ToolResult{Content: `{"applications": [], "message": "你目前还没有投递记录"}`}, nil
	}

	entries := make([]appEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, appEntry{
			ApplicationID: r.ApplicationID,
			JobID:         r.JobID,
			JobTitle:      r.JobTitle,
			Status:        r.Status,
			StatusText:    applicationStatusTextTool(r.Status),
			RoundNo:       r.RoundNo,
			AppliedAt:     r.AppliedAt.Format("2006-01-02 15:04"),
		})
	}
	b, _ := json.Marshal(map[string]any{
		"total":        len(entries),
		"applications": entries,
	})
	return ToolResult{Content: string(b)}, nil
}

func (e *CandidateToolExecutor) getMyApplicationDetail(ctx context.Context, userID int64, args map[string]any) (ToolResult, error) {
	appID := int64Arg(args, "application_id")
	if appID <= 0 {
		return ToolResult{Content: `{"error": "application_id is required"}`}, nil
	}

	detail, err := e.applications.GetDetailByUser(ctx, userID, appID)
	if err != nil {
		return ToolResult{}, err
	}
	if detail == nil {
		return ToolResult{Content: `{"error": "投递记录不存在或无权限访问"}`}, nil
	}

	result := map[string]any{
		"application_id": detail.ApplicationID,
		"job_id":         detail.JobID,
		"job_title":      detail.JobTitle,
		"department":     detail.Department,
		"location":       detail.Location,
		"salary_range":   detail.SalaryRange,
		"status":         detail.Status,
		"status_text":    applicationStatusTextTool(detail.Status),
		"round_no":       detail.RoundNo,
		"applied_at":     detail.AppliedAt.Format(time.RFC3339),
	}
	b, _ := json.Marshal(result)
	return ToolResult{Content: string(b)}, nil
}

func (e *CandidateToolExecutor) getMyResumeText(ctx context.Context, userID int64) (ToolResult, error) {
	resume, err := e.resumes.GetValidByUserID(ctx, userID)
	if err != nil {
		return ToolResult{}, err
	}
	if resume == nil {
		return ToolResult{Content: `{"resume_available": false, "message": "你还没有上传简历。请先上传简历后，我才能基于你的简历内容提供岗位推荐和优化建议。"}`}, nil
	}

	text := strings.TrimSpace(resume.ParsedText)
	if text == "" {
		return ToolResult{Content: fmt.Sprintf(`{"resume_available": false, "file_name": "%s", "message": "你的简历已上传，但解析文本暂不可用。请尝试重新上传简历，或等待系统完成解析后再使用此功能。"}`, resume.FileName)}, nil
	}

	if len([]rune(text)) < 20 {
		return ToolResult{Content: fmt.Sprintf(`{"resume_available": false, "file_name": "%s", "message": "简历解析文本内容过短（仅 %d 个字符），可能无法提供有效的分析和推荐。请检查上传的简历文件是否完整。"}`, resume.FileName, len([]rune(text)))}, nil
	}

	return ToolResult{Content: fmt.Sprintf(`{"resume_available": true, "file_name": "%s", "text_length": %d, "resume_text": "%s"}`, resume.FileName, len([]rune(text)), jsonEscape(text))}, nil
}

func (e *CandidateToolExecutor) listJobsForRecommendation(ctx context.Context, userID int64) (ToolResult, error) {
	rows, err := e.jobs.ListForCandidateWithApplicationMark(ctx, userID)
	if err != nil {
		return ToolResult{}, err
	}

	type jobEntry struct {
		JobID       int64  `json:"job_id"`
		Title       string `json:"title"`
		Department  string `json:"department"`
		Location    string `json:"location"`
		SalaryRange string `json:"salary_range"`
		Status      int32  `json:"status"`
		StatusText  string `json:"status_text"`
		HasApplied  bool   `json:"has_applied"`
	}

	entries := make([]jobEntry, 0, len(rows))
	for _, r := range rows {
		st := "招募中"
		if r.Status == 0 {
			st = "已下架"
		}
		entries = append(entries, jobEntry{
			JobID: r.ID, Title: r.Title, Department: r.Department,
			Location: r.Location, SalaryRange: r.SalaryRange,
			Status: r.Status, StatusText: st, HasApplied: r.HasApplied,
		})
	}
	if entries == nil {
		entries = []jobEntry{}
	}
	b, _ := json.Marshal(map[string]any{
		"total": len(entries),
		"jobs":  entries,
	})
	return ToolResult{Content: string(b)}, nil
}

func (e *CandidateToolExecutor) getJobDetailForCandidate(ctx context.Context, userID int64, args map[string]any) (ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	if jobID <= 0 {
		return ToolResult{Content: `{"error": "job_id is required"}`}, nil
	}

	job, hasApplied, err := e.jobs.GetForCandidate(ctx, userID, jobID)
	if err != nil {
		return ToolResult{}, err
	}
	if job == nil {
		return ToolResult{Content: `{"error": "岗位不存在"}`}, nil
	}

	result := map[string]any{
		"job_id":       job.ID,
		"title":        job.Title,
		"department":   job.Department,
		"location":     job.Location,
		"salary_range": job.SalaryRange,
		"description":  job.Description,
		"requirements": job.Requirements,
		"status":       job.Status,
		"status_text":  jobStatusTextTool(job.Status),
		"has_applied":  hasApplied,
	}
	b, _ := json.Marshal(result)
	return ToolResult{Content: string(b)}, nil
}

func (e *CandidateToolExecutor) recommendJobsByResume(ctx context.Context, userID int64) (ToolResult, error) {
	resume, err := e.resumes.GetValidByUserID(ctx, userID)
	if err != nil {
		return ToolResult{}, err
	}
	if resume == nil || strings.TrimSpace(resume.ParsedText) == "" {
		return ToolResult{Content: `{"error": "no_resume", "message": "你还没有上传简历或简历解析文本为空。请先上传简历后，我才能为你推荐匹配的岗位。"}`}, nil
	}

	rows, err := e.jobs.ListForCandidateWithApplicationMark(ctx, userID)
	if err != nil {
		return ToolResult{}, err
	}

	type jobEntry struct {
		JobID       int64  `json:"job_id"`
		Title       string `json:"title"`
		Department  string `json:"department"`
		Location    string `json:"location"`
		SalaryRange string `json:"salary_range"`
		Status      int32  `json:"status"`
		HasApplied  bool   `json:"has_applied"`
	}

	entries := make([]jobEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, jobEntry{
			JobID: r.ID, Title: r.Title, Department: r.Department,
			Location: r.Location, SalaryRange: r.SalaryRange,
			Status: r.Status, HasApplied: r.HasApplied,
		})
	}
	if entries == nil {
		entries = []jobEntry{}
	}

	b, _ := json.Marshal(map[string]any{
		"resume_text": resume.ParsedText,
		"resume_file": resume.FileName,
		"total_jobs":  len(entries),
		"jobs":        entries,
		"instruction": "请基于以上简历内容和岗位列表，为候选人推荐 3-5 个最匹配的岗位。对每个推荐岗位说明匹配理由、候选人的不足点、建议投递优先级。如果岗位的 has_applied 为 true，必须标注'已投递'。只输出推荐结果，不要输出其他内容。",
	})
	return ToolResult{Content: string(b)}, nil
}
