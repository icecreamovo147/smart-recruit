package hr_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	gogrpc "google.golang.org/grpc"

	commonsai "smart-recruit-commons/ai"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

// ApplicationListClient is the subset of ApplicationService used by HR tools.
type ApplicationListClient interface {
	ListJobApplications(ctx context.Context, in *pb.ListJobApplicationsRequest, opts ...gogrpc.CallOption) (*pb.ListJobApplicationsResponse, error)
}

// SnapshotClient loads application snapshots for candidate detail tools.
type SnapshotClient interface {
	GetApplicationSnapshot(ctx context.Context, in *pb.GetApplicationSnapshotRequest, opts ...gogrpc.CallOption) (*pb.GetApplicationSnapshotResponse, error)
}

func (e *Executor) queryTotalApplications(ctx context.Context, hrID int64, _ map[string]any) (commonsai.ToolResult, error) {
	jobs, err := e.listHRJobsInventory(ctx, hrID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	var total int64
	for _, job := range jobs {
		if job == nil {
			continue
		}
		total += job.GetApplicationCount()
	}
	// Prefer live sum from applications when Application client is present.
	if e.Applications != nil {
		apps, appErr := e.listAllApplications(ctx, hrID, 0, true)
		if appErr == nil {
			total = int64(len(apps))
		}
	}
	return jsonResult(map[string]any{"total_applications": total}), nil
}

func (e *Executor) queryTodayApplications(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	apps, err := e.listAllApplications(ctx, hrID, jobID, true)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	today := time.Now().Format("2006-01-02")
	var count int64
	for _, app := range apps {
		if app == nil {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(app.GetAppliedAt()), today) {
			count++
		}
	}
	return jsonResult(map[string]any{
		"job_id":             jobID,
		"today_applications": count,
		"date":               today,
	}), nil
}

func (e *Executor) jobHeatRanking(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	topN := 5
	if v, ok := optionalInt32Arg(args, "top_n"); ok && v > 0 {
		topN = int(v)
	}
	jobs, err := e.listHRJobsInventory(ctx, hrID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	type entry struct {
		Title string `json:"title"`
		Total int64  `json:"total"`
		JobID int64  `json:"job_id"`
	}
	items := make([]entry, 0, len(jobs))
	for _, job := range jobs {
		if job == nil {
			continue
		}
		items = append(items, entry{Title: job.GetTitle(), Total: job.GetApplicationCount(), JobID: job.GetJobId()})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Total == items[j].Total {
			return items[i].JobID < items[j].JobID
		}
		return items[i].Total > items[j].Total
	})
	if topN > len(items) {
		topN = len(items)
	}
	return jsonResult(map[string]any{"hot_jobs": items[:topN]}), nil
}

func (e *Executor) listAllApplicationsTool(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	page, pageSize := pageArgs(args, 1, 10)
	apps, err := e.listAllApplications(ctx, hrID, 0, true)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	return pageApplications(apps, page, pageSize), nil
}

func (e *Executor) listApplicationsByJob(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	if jobID <= 0 {
		return jsonResult(map[string]any{"error": "job_id is required"}), nil
	}
	page, pageSize := pageArgs(args, 1, 10)
	currentOnly := true
	if v, ok := args["current_only"].(bool); ok {
		currentOnly = v
	}
	apps, err := e.listAllApplications(ctx, hrID, jobID, currentOnly)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if v, ok := optionalInt32Arg(args, "status"); ok {
		apps = filterApplicationsByStatus(apps, &v)
	}
	return pageApplications(apps, page, pageSize), nil
}

func (e *Executor) listApplicationsByStatus(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	status, ok := optionalInt32Arg(args, "status")
	if !ok {
		return jsonResult(map[string]any{"error": "status is required"}), nil
	}
	jobID := int64Arg(args, "job_id")
	page, pageSize := pageArgs(args, 1, 10)
	currentOnly := true
	if v, ok := args["current_only"].(bool); ok {
		currentOnly = v
	}
	apps, err := e.listAllApplications(ctx, hrID, jobID, currentOnly)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	apps = filterApplicationsByStatus(apps, &status)
	return pageApplications(apps, page, pageSize), nil
}

func (e *Executor) applicationStatusSummary(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	apps, err := e.listAllApplications(ctx, hrID, jobID, true)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	counts := map[int32]int64{}
	for _, app := range apps {
		if app == nil {
			continue
		}
		counts[app.GetStatus()]++
	}
	entries := make([]map[string]any, 0, len(counts))
	for status, total := range counts {
		entries = append(entries, map[string]any{
			"status":      status,
			"status_text": applicationStatusText(status),
			"total":       total,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i]["status"].(int32) < entries[j]["status"].(int32)
	})
	return jsonResult(map[string]any{
		"job_id":  jobID,
		"summary": entries,
		"total":   int64(len(apps)),
	}), nil
}

func (e *Executor) applicationTrend(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	days := int32(7)
	if v, ok := optionalInt32Arg(args, "days"); ok && v > 0 {
		days = v
	}
	if days > 90 {
		days = 90
	}
	jobID := int64Arg(args, "job_id")
	apps, err := e.listAllApplications(ctx, hrID, jobID, true)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	cutoff := time.Now().AddDate(0, 0, -int(days-1)).Truncate(24 * time.Hour)
	byDay := map[string]int64{}
	for d := 0; d < int(days); d++ {
		day := cutoff.AddDate(0, 0, d).Format("2006-01-02")
		byDay[day] = 0
	}
	for _, app := range apps {
		if app == nil {
			continue
		}
		day := strings.TrimSpace(app.GetAppliedAt())
		if len(day) >= 10 {
			day = day[:10]
		}
		if _, ok := byDay[day]; ok {
			byDay[day]++
		}
	}
	points := make([]map[string]any, 0, len(byDay))
	for d := 0; d < int(days); d++ {
		day := cutoff.AddDate(0, 0, d).Format("2006-01-02")
		points = append(points, map[string]any{"date": day, "applications": byDay[day]})
	}
	return jsonResult(map[string]any{"days": days, "job_id": jobID, "trend": points}), nil
}

func (e *Executor) searchCandidates(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	keyword, _ := args["keyword"].(string)
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return jsonResult(map[string]any{"error": "keyword is required"}), nil
	}
	keywordLower := strings.ToLower(keyword)
	jobs, err := e.listHRJobsInventory(ctx, hrID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	// List per job so we can attach job titles (JobApplication has no job_id field).
	enriched := make([]map[string]any, 0)
	for _, job := range jobs {
		if job == nil {
			continue
		}
		jobApps, listErr := e.listJobApplications(ctx, hrID, job.GetJobId(), true)
		if listErr != nil {
			continue
		}
		for _, app := range jobApps {
			if app == nil {
				continue
			}
			haystack := strings.ToLower(strings.Join([]string{
				app.GetRealName(),
				app.GetPhone(),
				app.GetSchool(),
				app.GetEducation(),
				job.GetTitle(),
			}, " "))
			if !strings.Contains(haystack, keywordLower) {
				continue
			}
			enriched = append(enriched, map[string]any{
				"application_id": app.GetApplicationId(),
				"real_name":      app.GetRealName(),
				"phone":          maskPhone(app.GetPhone()),
				"job_title":      job.GetTitle(),
				"status":         applicationStatusText(app.GetStatus()),
				"round_no":       app.GetRoundNo(),
				"is_current":     app.GetIsCurrent(),
				"applied_at":     app.GetAppliedAt(),
			})
		}
	}
	message := ""
	if len(enriched) == 0 {
		message = "未找到匹配的候选人"
	}
	return jsonResult(map[string]any{"candidates": enriched, "message": message}), nil
}

func (e *Executor) getCandidateDetail(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	applicationID := int64Arg(args, "application_id")
	if applicationID <= 0 {
		return jsonResult(map[string]any{"error": "application_id is required"}), nil
	}
	if e.Snapshots == nil {
		return commonsai.ToolResult{}, fmt.Errorf("application snapshot client is not configured")
	}
	// Scope check: application must belong to one of HR's jobs.
	owned, err := e.applicationOwnedByHR(ctx, hrID, applicationID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if !owned {
		return jsonResult(map[string]any{"error": "投递记录不存在或无权限访问"}), nil
	}
	resp, err := e.Snapshots.GetApplicationSnapshot(ctx, &pb.GetApplicationSnapshotRequest{ApplicationId: applicationID})
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if resp == nil {
		return commonsai.ToolResult{}, fmt.Errorf("get application snapshot returned nil")
	}
	if resp.GetCode() != errs.OK {
		msg := strings.TrimSpace(resp.GetMsg())
		if msg == "" {
			msg = "application snapshot unavailable"
		}
		return jsonResult(map[string]any{"error": msg}), nil
	}
	return jsonResult(map[string]any{
		"application_id":    resp.GetApplicationId(),
		"candidate_user_id": resp.GetCandidateUserId(),
		"candidate_name":    resp.GetCandidateName(),
		"job_id":            resp.GetJobId(),
		"job_title":         resp.GetJobTitle(),
		"resume_id":         resp.GetResumeId(),
		"legacy_status":     resp.GetLegacyStatus(),
		"status_key":        resp.GetStatusKey(),
		"round_no":          resp.GetRoundNo(),
		"is_current":        resp.GetIsCurrent(),
	}), nil
}

func (e *Executor) proposeApplicationStatusUpdate(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	applicationID := int64Arg(args, "application_id")
	status, ok := optionalInt32Arg(args, "status")
	if applicationID <= 0 {
		return jsonResult(map[string]any{"error": "application_id is required"}), nil
	}
	if !ok || (status != 2 && status != 3) {
		return jsonResult(map[string]any{"error": "status must be 2 (通过) or 3 (淘汰)"}), nil
	}
	owned, err := e.applicationOwnedByHR(ctx, hrID, applicationID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if !owned {
		return jsonResult(map[string]any{"error": "投递记录不存在或无权限访问"}), nil
	}
	detail, _ := e.getCandidateDetail(ctx, hrID, map[string]any{"application_id": applicationID})
	candidateName := ""
	jobTitle := ""
	if detail.Content != "" {
		var payload map[string]any
		_ = jsonUnmarshal(detail.Content, &payload)
		candidateName, _ = payload["candidate_name"].(string)
		jobTitle, _ = payload["job_title"].(string)
	}
	return jsonResult(map[string]any{
		"action":          "propose_application_status_update",
		"application_id":  applicationID,
		"proposed_status": status,
		"status_text":     applicationStatusText(status),
		"candidate_name":  candidateName,
		"job_title":       jobTitle,
		"requires_confirm": true,
		"message":         "此工具仅生成待确认动作，不会直接修改数据库；请 HR 明确确认后由系统执行状态变更。",
	}), nil
}

func (e *Executor) listAllApplications(ctx context.Context, hrID, jobID int64, currentOnly bool) ([]*pb.JobApplication, error) {
	if e.Applications == nil {
		return nil, fmt.Errorf("application list service is not configured")
	}
	if jobID > 0 {
		return e.listJobApplications(ctx, hrID, jobID, currentOnly)
	}
	jobs, err := e.listHRJobsInventory(ctx, hrID)
	if err != nil {
		return nil, err
	}
	all := make([]*pb.JobApplication, 0)
	for _, job := range jobs {
		if job == nil {
			continue
		}
		apps, listErr := e.listJobApplications(ctx, hrID, job.GetJobId(), currentOnly)
		if listErr != nil {
			return nil, listErr
		}
		all = append(all, apps...)
	}
	return all, nil
}

func (e *Executor) listJobApplications(ctx context.Context, hrID, jobID int64, currentOnly bool) ([]*pb.JobApplication, error) {
	if e.Applications == nil {
		return nil, fmt.Errorf("application list service is not configured")
	}
	const pageSize = int32(100)
	var all []*pb.JobApplication
	var seen int64
	for page := int32(1); page <= 50; page++ {
		resp, err := e.Applications.ListJobApplications(ctx, &pb.ListJobApplicationsRequest{
			HrId:     hrID,
			JobId:    jobID,
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, fmt.Errorf("list job applications returned nil")
		}
		if resp.GetCode() != errs.OK {
			msg := strings.TrimSpace(resp.GetMsg())
			if msg == "" {
				msg = "list job applications was denied"
			}
			return nil, fmt.Errorf("%s", msg)
		}
		batch := resp.GetList()
		for _, app := range batch {
			if app == nil {
				continue
			}
			if currentOnly && app.GetIsCurrent() == 0 {
				continue
			}
			all = append(all, app)
		}
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

func (e *Executor) applicationOwnedByHR(ctx context.Context, hrID, applicationID int64) (bool, error) {
	apps, err := e.listAllApplications(ctx, hrID, 0, false)
	if err != nil {
		// Fallback: try snapshot job_hr_id when list fails.
		if e.Snapshots != nil {
			resp, snapErr := e.Snapshots.GetApplicationSnapshot(ctx, &pb.GetApplicationSnapshotRequest{ApplicationId: applicationID})
			if snapErr != nil {
				return false, err
			}
			if resp != nil && resp.GetCode() == errs.OK && resp.GetJobHrId() == hrID {
				return true, nil
			}
		}
		return false, err
	}
	for _, app := range apps {
		if app != nil && app.GetApplicationId() == applicationID {
			return true, nil
		}
	}
	return false, nil
}

func pageApplications(apps []*pb.JobApplication, page, pageSize int32) commonsai.ToolResult {
	total := int64(len(apps))
	start := int((page - 1) * pageSize)
	if start > len(apps) {
		start = len(apps)
	}
	end := start + int(pageSize)
	if end > len(apps) {
		end = len(apps)
	}
	entries := make([]map[string]any, 0, end-start)
	for _, app := range apps[start:end] {
		if app == nil {
			continue
		}
		entries = append(entries, map[string]any{
			"application_id": app.GetApplicationId(),
			"real_name":      app.GetRealName(),
			"phone":          maskPhone(app.GetPhone()),
			"education":      app.GetEducation(),
			"school":         app.GetSchool(),
			"status":         app.GetStatus(),
			"status_text":    applicationStatusText(app.GetStatus()),
			"round_no":       app.GetRoundNo(),
			"is_current":     app.GetIsCurrent(),
			"applied_at":     app.GetAppliedAt(),
		})
	}
	return jsonResult(map[string]any{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"applications": entries,
	})
}

func filterApplicationsByStatus(apps []*pb.JobApplication, status *int32) []*pb.JobApplication {
	if status == nil {
		return apps
	}
	out := make([]*pb.JobApplication, 0, len(apps))
	for _, app := range apps {
		if app != nil && app.GetStatus() == *status {
			out = append(out, app)
		}
	}
	return out
}

func applicationStatusText(status int32) string {
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
		return fmt.Sprintf("状态%d", status)
	}
}

func maskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func jsonUnmarshal(raw string, dest any) error {
	return json.Unmarshal([]byte(raw), dest)
}
