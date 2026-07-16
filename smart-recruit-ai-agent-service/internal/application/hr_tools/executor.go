// Package hr_tools provides microservice-backed HR recruiting tool execution.
// Tools query recruitment domain data through gRPC clients instead of direct DB access.
package hr_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gogrpc "google.golang.org/grpc"

	commonsai "smart-recruit-commons/ai"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

// JobClient is the subset of JobService used by HR builtin tools.
type JobClient interface {
	ListHRJobs(ctx context.Context, in *pb.ListHRJobsRequest, opts ...gogrpc.CallOption) (*pb.ListJobsResponse, error)
	GetJobDetail(ctx context.Context, in *pb.GetJobDetailRequest, opts ...gogrpc.CallOption) (*pb.GetJobDetailResponse, error)
}

// Executor implements commonsai.ToolRunner for HR recruiting builtins.
type Executor struct {
	Jobs         JobClient
	Applications ApplicationListClient
	Snapshots    SnapshotClient
}

// DefaultJobToolNames are the default tools used when agent bindings do not name concrete tools.
var DefaultJobToolNames = []string{
	"get_job_list",
	"search_jobs",
	"get_job_detail",
	"query_total_applications",
	"query_today_applications",
	"get_job_heat_ranking",
	"get_application_status_summary",
	"list_all_applications",
	"search_candidates",
}

// KnownBuiltinToolNames is the allowlist of tool names this executor (or future
// extensions) may expose to the model.
var KnownBuiltinToolNames = map[string]bool{
	"get_job_list":                      true,
	"search_jobs":                       true,
	"get_job_detail":                    true,
	"get_application_snapshot":          true,
	"query_total_applications":          true,
	"query_today_applications":          true,
	"get_job_heat_ranking":              true,
	"search_candidates":                 true,
	"get_candidate_detail":              true,
	"list_all_applications":             true,
	"list_applications_by_job":          true,
	"list_applications_by_status":       true,
	"get_application_status_summary":    true,
	"get_application_trend":             true,
	"propose_application_status_update": true,
	"parse_resume_profile":              true,
	"get_resume_profile":                true,
	"evaluate_candidate_match":          true,
	"get_candidate_match_evaluation":    true,
	"compare_candidates_for_job":        true,
}

// ExecutableByThisRunner marks tools currently implemented by Executor.Execute.
var ExecutableByThisRunner = map[string]bool{
	"get_job_list":                      true,
	"search_jobs":                       true,
	"get_job_detail":                    true,
	"query_total_applications":          true,
	"query_today_applications":          true,
	"get_job_heat_ranking":              true,
	"list_all_applications":             true,
	"list_applications_by_job":          true,
	"list_applications_by_status":       true,
	"get_application_status_summary":    true,
	"get_application_trend":             true,
	"search_candidates":                 true,
	"get_candidate_detail":              true,
	"propose_application_status_update": true,
}

// Execute routes a model-selected tool call to the underlying domain client.
func (e *Executor) Execute(ctx context.Context, hrID int64, toolName string, args map[string]any) (commonsai.ToolResult, error) {
	if e == nil {
		return commonsai.ToolResult{}, fmt.Errorf("hr tool executor is not configured")
	}
	if hrID <= 0 {
		return jsonResult(map[string]any{"error": "hr_id is required"}), nil
	}
	if args == nil {
		args = map[string]any{}
	}
	switch NormalizeToolName(toolName) {
	case "get_job_list":
		return e.getJobList(ctx, hrID, args)
	case "search_jobs":
		return e.searchJobs(ctx, hrID, args)
	case "get_job_detail":
		return e.getJobDetail(ctx, hrID, args)
	case "query_total_applications":
		return e.queryTotalApplications(ctx, hrID, args)
	case "query_today_applications":
		return e.queryTodayApplications(ctx, hrID, args)
	case "get_job_heat_ranking":
		return e.jobHeatRanking(ctx, hrID, args)
	case "list_all_applications":
		return e.listAllApplicationsTool(ctx, hrID, args)
	case "list_applications_by_job":
		return e.listApplicationsByJob(ctx, hrID, args)
	case "list_applications_by_status":
		return e.listApplicationsByStatus(ctx, hrID, args)
	case "get_application_status_summary":
		return e.applicationStatusSummary(ctx, hrID, args)
	case "get_application_trend":
		return e.applicationTrend(ctx, hrID, args)
	case "search_candidates":
		return e.searchCandidates(ctx, hrID, args)
	case "get_candidate_detail":
		return e.getCandidateDetail(ctx, hrID, args)
	case "propose_application_status_update":
		return e.proposeApplicationStatusUpdate(ctx, hrID, args)
	default:
		return jsonResult(map[string]any{"error": fmt.Sprintf("unsupported tool: %s", toolName)}), nil
	}
}

func (e *Executor) getJobList(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	// Default: open jobs only (status=1). Callers may pass status to override.
	statusFilter := int32(1)
	hasStatus := true
	if v, ok := optionalInt32Arg(args, "status"); ok {
		statusFilter = v
	}
	jobs, err := e.listHRJobsInventory(ctx, hrID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	var statusPtr *int32
	if hasStatus {
		statusPtr = &statusFilter
	}
	filtered := filterJobs(jobs, "", statusPtr)
	return jsonResult(map[string]any{
		"total": int64(len(filtered)),
		"jobs":  toJobEntries(filtered),
	}), nil
}

func (e *Executor) searchJobs(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	keyword, _ := args["keyword"].(string)
	keyword = strings.TrimSpace(keyword)
	page, pageSize := pageArgs(args, 1, 10)
	if pageSize > 50 {
		pageSize = 50
	}
	var statusFilter *int32
	if v, ok := optionalInt32Arg(args, "status"); ok {
		statusFilter = &v
	}
	jobs, err := e.listHRJobsInventory(ctx, hrID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	filtered := filterJobs(jobs, keyword, statusFilter)
	total := int64(len(filtered))
	start := int((page - 1) * pageSize)
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + int(pageSize)
	if end > len(filtered) {
		end = len(filtered)
	}
	return jsonResult(map[string]any{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"jobs":      toJobEntries(filtered[start:end]),
	}), nil
}

func (e *Executor) getJobDetail(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	if jobID <= 0 {
		return jsonResult(map[string]any{"error": "job_id is required"}), nil
	}
	if e.Jobs == nil {
		return commonsai.ToolResult{}, fmt.Errorf("job service is not configured")
	}
	owned, err := e.findOwnedJob(ctx, hrID, jobID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if owned == nil {
		return jsonResult(map[string]any{"error": "岗位不存在或无权限访问"}), nil
	}
	resp, err := e.Jobs.GetJobDetail(ctx, &pb.GetJobDetailRequest{JobId: jobID})
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if resp == nil {
		return commonsai.ToolResult{}, fmt.Errorf("get job detail returned nil response")
	}
	if resp.GetCode() != errs.OK || resp.GetJob() == nil {
		msg := strings.TrimSpace(resp.GetMsg())
		if msg == "" {
			msg = "job detail unavailable"
		}
		return jsonResult(map[string]any{"error": msg}), nil
	}
	job := resp.GetJob()
	return jsonResult(map[string]any{
		"job_id":            job.GetJobId(),
		"title":             job.GetTitle(),
		"department":        job.GetDepartment(),
		"location":          job.GetLocation(),
		"salary_range":      job.GetSalaryRange(),
		"description":       job.GetDescription(),
		"requirements":      job.GetRequirements(),
		"status":            job.GetStatus(),
		"status_text":       jobStatusText(job.GetStatus()),
		"application_count": job.GetApplicationCount(),
		"created_at":        job.GetCreatedAt(),
	}), nil
}

func (e *Executor) findOwnedJob(ctx context.Context, hrID, jobID int64) (*pb.Job, error) {
	jobs, err := e.listHRJobsInventory(ctx, hrID)
	if err != nil {
		return nil, err
	}
	for _, job := range jobs {
		if job != nil && job.GetJobId() == jobID {
			return job, nil
		}
	}
	return nil, nil
}

func (e *Executor) listHRJobsInventory(ctx context.Context, hrID int64) ([]*pb.Job, error) {
	if e.Jobs == nil {
		return nil, fmt.Errorf("job service is not configured")
	}
	const (
		pageSize = int32(100)
		maxPages = int32(20)
	)
	var (
		all  []*pb.Job
		seen int64
	)
	for page := int32(1); page <= maxPages; page++ {
		resp, err := e.Jobs.ListHRJobs(ctx, &pb.ListHRJobsRequest{HrId: hrID, Page: page, PageSize: pageSize})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, fmt.Errorf("list hr jobs returned nil response")
		}
		if resp.GetCode() != errs.OK {
			msg := strings.TrimSpace(resp.GetMsg())
			if msg == "" {
				msg = "list hr jobs was denied"
			}
			return nil, fmt.Errorf("%s", msg)
		}
		batch := resp.GetList()
		all = append(all, batch...)
		seen += int64(len(batch))
		if len(batch) == 0 {
			break
		}
		if resp.GetTotal() > 0 && seen >= resp.GetTotal() {
			break
		}
		if int32(len(batch)) < pageSize {
			break
		}
	}
	return all, nil
}

func filterJobs(jobs []*pb.Job, keyword string, status *int32) []*pb.Job {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	out := make([]*pb.Job, 0, len(jobs))
	for _, job := range jobs {
		if job == nil {
			continue
		}
		if status != nil && job.GetStatus() != *status {
			continue
		}
		if keyword != "" {
			haystack := strings.ToLower(strings.Join([]string{
				job.GetTitle(),
				job.GetDepartment(),
				job.GetLocation(),
				job.GetDescription(),
				job.GetRequirements(),
				job.GetSalaryRange(),
			}, " "))
			if !strings.Contains(haystack, keyword) {
				continue
			}
		}
		out = append(out, job)
	}
	return out
}

func toJobEntries(jobs []*pb.Job) []map[string]any {
	entries := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		if job == nil {
			continue
		}
		entries = append(entries, map[string]any{
			"job_id":            job.GetJobId(),
			"title":             job.GetTitle(),
			"department":        job.GetDepartment(),
			"location":          job.GetLocation(),
			"salary_range":      job.GetSalaryRange(),
			"status":            job.GetStatus(),
			"status_text":       jobStatusText(job.GetStatus()),
			"application_count": job.GetApplicationCount(),
			"created_at":        job.GetCreatedAt(),
		})
	}
	return entries
}

func jobStatusText(status int32) string {
	if status == 0 {
		return "已下架"
	}
	return "招募中"
}

func jsonResult(value any) commonsai.ToolResult {
	data, err := json.Marshal(value)
	if err != nil {
		return commonsai.ToolResult{Content: `{"error":"marshal failed"}`}
	}
	return commonsai.ToolResult{Content: string(data)}
}

func pageArgs(args map[string]any, defaultPage, defaultSize int32) (int32, int32) {
	page := defaultPage
	size := defaultSize
	if v, ok := optionalInt32Arg(args, "page"); ok && v > 0 {
		page = v
	}
	if v, ok := optionalInt32Arg(args, "page_size"); ok && v > 0 {
		size = v
	}
	return page, size
}

func optionalInt32Arg(args map[string]any, key string) (int32, bool) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return 0, false
	}
	switch v := raw.(type) {
	case int32:
		return v, true
	case int:
		return int32(v), true
	case int64:
		return int32(v), true
	case float64:
		return int32(v), true
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int32(i), true
	case string:
		var n int64
		if _, err := fmt.Sscan(strings.TrimSpace(v), &n); err != nil {
			return 0, false
		}
		return int32(n), true
	default:
		return 0, false
	}
}

func int64Arg(args map[string]any, key string) int64 {
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
	case json.Number:
		i, _ := v.Int64()
		return i
	case string:
		var n int64
		_, _ = fmt.Sscan(strings.TrimSpace(v), &n)
		return n
	default:
		return 0
	}
}

// NormalizeToolName strips optional source prefixes such as "builtin:".
func NormalizeToolName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if i := strings.IndexByte(name, ':'); i >= 0 {
		source := strings.ToLower(strings.TrimSpace(name[:i]))
		if source == "builtin" || source == "tool" {
			return strings.TrimSpace(name[i+1:])
		}
	}
	return name
}

// ResolveBuiltinToolNames derives the executable builtin tool allowlist from agent bindings.
// When bindings are empty or only abstract capability labels are present, DefaultJobToolNames
// are returned so free chat still has real tools.
func ResolveBuiltinToolNames(toolBindings []string, capabilityKeys []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0)
	add := func(name string) {
		name = NormalizeToolName(name)
		if name == "" || seen[name] || !KnownBuiltinToolNames[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for _, name := range toolBindings {
		add(name)
	}
	for _, key := range capabilityKeys {
		add(key)
	}
	if len(out) == 0 {
		return append([]string(nil), DefaultJobToolNames...)
	}
	hasConcreteExecutable := false
	for _, name := range out {
		if ExecutableByThisRunner[name] {
			hasConcreteExecutable = true
			break
		}
	}
	if !hasConcreteExecutable {
		merged := append([]string(nil), DefaultJobToolNames...)
		for _, name := range out {
			if name == "get_application_snapshot" {
				merged = append(merged, name)
			}
		}
		return merged
	}
	return out
}
