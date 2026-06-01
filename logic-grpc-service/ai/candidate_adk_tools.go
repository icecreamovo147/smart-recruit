package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// --- typed input/output structs ---

type listMyApplicationsOutput struct {
	Total        int32            `json:"total"`
	Applications []myAppEntry     `json:"applications"`
	Message      string           `json:"message,omitempty"`
}

type myAppEntry struct {
	ApplicationID int64  `json:"application_id"`
	JobID         int64  `json:"job_id"`
	JobTitle      string `json:"job_title"`
	Status        int32  `json:"status"`
	StatusText    string `json:"status_text"`
	RoundNo       int32  `json:"round_no"`
	AppliedAt     string `json:"applied_at"`
}

type getMyApplicationDetailInput struct {
	ApplicationID int64 `json:"application_id" jsonschema:"required" jsonschema_description:"投递记录 ID"`
}

type myApplicationDetailOutput struct {
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

type getMyResumeOutput struct {
	ResumeAvailable bool   `json:"resume_available"`
	FileName        string `json:"file_name"`
	TextLength      int    `json:"text_length,omitempty"`
	ResumeText      string `json:"resume_text,omitempty"`
	Message         string `json:"message,omitempty"`
}

type listJobsForRecommendationOutput struct {
	Total int32               `json:"total"`
	Jobs  []candidateJobEntry `json:"jobs"`
}

type candidateJobEntry struct {
	JobID       int64  `json:"job_id"`
	Title       string `json:"title"`
	Department  string `json:"department"`
	Location    string `json:"location"`
	SalaryRange string `json:"salary_range"`
	Status      int32  `json:"status"`
	StatusText  string `json:"status_text"`
	HasApplied  bool   `json:"has_applied"`
}

type getJobDetailForCandidateInput struct {
	JobID int64 `json:"job_id" jsonschema:"required" jsonschema_description:"岗位 ID"`
}

type jobDetailForCandidateOutput struct {
	JobID       int64  `json:"job_id"`
	Title       string `json:"title"`
	Department  string `json:"department"`
	Location    string `json:"location"`
	SalaryRange string `json:"salary_range"`
	Description string `json:"description"`
	Requirements string `json:"requirements"`
	Status      int32  `json:"status"`
	StatusText  string `json:"status_text"`
	HasApplied  bool   `json:"has_applied"`
}

type recommendJobsByResumeOutput struct {
	ResumeText  string             `json:"resume_text"`
	ResumeFile  string             `json:"resume_file"`
	TotalJobs   int                `json:"total_jobs"`
	Jobs        []candidateJobEntry `json:"jobs"`
	Instruction string             `json:"instruction"`
}

// NewCandidateADKTools creates all candidate tools as tool.BaseTool with
// typed input/output via utils.InferTool. Each tool delegates to
// *CandidateToolExecutor so business logic lives in a single place. Each tool
// is wrapped with a uniform error handler so the model receives JSON error
// strings.
//
// The executor is captured at creation time (allowing tools to be cached), but
// per-request values (user ID, AgentRunState) are retrieved from context at
// call time via WithOwnerID / WithAgentRunState.
func NewCandidateADKTools(executor *CandidateToolExecutor) ([]tool.BaseTool, error) {
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

	listMyAppsTool, err := utils.InferTool(
		"list_my_applications",
		"查询当前候选人自己的投递列表和状态。返回每条投递的岗位名称、状态、投递时间、轮次等信息",
		func(ctx context.Context, _ struct{}) (listMyApplicationsOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			result, err := executor.Execute(ctx, ownerID, "list_my_applications", nil)
			if err != nil {
				return listMyApplicationsOutput{}, err
			}
			var out listMyApplicationsOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return listMyApplicationsOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list_my_applications: %w", err)
	}

	getMyAppDetailTool, err := utils.InferTool(
		"get_my_application_detail",
		"查询某条本人投递的详细进度，包括岗位信息和当前状态",
		func(ctx context.Context, input getMyApplicationDetailInput) (myApplicationDetailOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			args := map[string]any{"application_id": input.ApplicationID}
			result, err := executor.Execute(ctx, ownerID, "get_my_application_detail", args)
			if err != nil {
				return myApplicationDetailOutput{}, err
			}
			var out myApplicationDetailOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return myApplicationDetailOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get_my_application_detail: %w", err)
	}

	getMyResumeTool, err := utils.InferTool(
		"get_my_resume_text",
		"获取当前候选人有效上传简历的解析文本。如果候选人没有上传简历或简历解析为空，会返回提示信息",
		func(ctx context.Context, _ struct{}) (getMyResumeOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			result, err := executor.Execute(ctx, ownerID, "get_my_resume_text", nil)
			if err != nil {
				return getMyResumeOutput{}, err
			}
			var out getMyResumeOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return getMyResumeOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get_my_resume_text: %w", err)
	}

	listJobsForRecTool, err := utils.InferTool(
		"list_jobs_for_recommendation",
		"查询系统内岗位列表，并标记当前候选人是否已投递每个岗位。返回岗位 ID、名称、部门、地点、薪资、招募状态、has_applied 字段",
		func(ctx context.Context, _ struct{}) (listJobsForRecommendationOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			result, err := executor.Execute(ctx, ownerID, "list_jobs_for_recommendation", nil)
			if err != nil {
				return listJobsForRecommendationOutput{}, err
			}
			var out listJobsForRecommendationOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return listJobsForRecommendationOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list_jobs_for_recommendation: %w", err)
	}

	getJobDetailForCandidateTool, err := utils.InferTool(
		"get_job_detail_for_candidate",
		"查询指定岗位的详情，并标记当前候选人是否已投递该岗位",
		func(ctx context.Context, input getJobDetailForCandidateInput) (jobDetailForCandidateOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			args := map[string]any{"job_id": input.JobID}
			result, err := executor.Execute(ctx, ownerID, "get_job_detail_for_candidate", args)
			if err != nil {
				return jobDetailForCandidateOutput{}, err
			}
			var out jobDetailForCandidateOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return jobDetailForCandidateOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get_job_detail_for_candidate: %w", err)
	}

	recommendByResumeTool, err := utils.InferTool(
		"recommend_jobs_by_resume",
		"基于当前候选人上传简历的解析文本和系统内所有岗位，生成 3-5 个推荐岗位，包含匹配理由、不足点和建议投递优先级",
		func(ctx context.Context, _ struct{}) (recommendJobsByResumeOutput, error) {
			ownerID := ownerIDFromContext(ctx)
			result, err := executor.Execute(ctx, ownerID, "recommend_jobs_by_resume", nil)
			if err != nil {
				return recommendJobsByResumeOutput{}, err
			}
			var out recommendJobsByResumeOutput
			if err := json.Unmarshal([]byte(result.Content), &out); err != nil {
				return recommendJobsByResumeOutput{}, err
			}
			return out, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("recommend_jobs_by_resume: %w", err)
	}

	baseTools := []tool.BaseTool{
		listMyAppsTool, getMyAppDetailTool, getMyResumeTool,
		listJobsForRecTool, getJobDetailForCandidateTool, recommendByResumeTool,
	}

	result := make([]tool.BaseTool, 0, len(baseTools))
	for _, t := range baseTools {
		result = append(result, utils.WrapToolWithErrorHandler(t, errorHandler))
	}
	return result, nil
}
