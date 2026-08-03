package hr_tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	commonsai "smart-recruit-commons/ai"
	"smart-recruit-platform-go/businessclock"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

// ApplicationListClient is the subset of ApplicationService used by HR tools.
type ApplicationListClient interface {
	ListJobApplications(ctx context.Context, in *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error)
}

// SnapshotClient loads application snapshots for candidate detail tools.
type SnapshotClient interface {
	GetApplicationSnapshot(ctx context.Context, in *pb.GetApplicationSnapshotRequest) (*pb.GetApplicationSnapshotResponse, error)
}

const (
	applicationAggregationMaxJobs        = 100
	applicationAggregationMaxPagesPerJob = int32(10)
	applicationAggregationPageSize       = int32(100)
	applicationAggregationMaxRows        = 5000
	applicationAggregationWorkers        = 4
	applicationAggregationMaxWarnings    = 10
)

type aggregatedApplication struct {
	JobID    int64
	JobTitle string
	App      *pb.JobApplication
}

type applicationAggregation struct {
	Items          []aggregatedApplication
	Partial        bool
	Truncated      bool
	FailedJobCount int
	Warnings       []string
}

func (a applicationAggregation) applications() []*pb.JobApplication {
	apps := make([]*pb.JobApplication, 0, len(a.Items))
	for _, item := range a.Items {
		if item.App != nil {
			apps = append(apps, item.App)
		}
	}
	return apps
}

func (a applicationAggregation) metadata() map[string]any {
	return map[string]any{
		"partial":          a.Partial,
		"truncated":        a.Truncated,
		"failed_job_count": a.FailedJobCount,
		"warnings":         append([]string(nil), a.Warnings...),
	}
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
		aggregation, appErr := e.listAllApplications(ctx, hrID, 0, true)
		if appErr != nil {
			return commonsai.ToolResult{}, appErr
		}
		total = int64(len(aggregation.Items))
		payload := aggregation.metadata()
		payload["total_applications"] = total
		return jsonResult(payload), nil
	}
	return jsonResult(map[string]any{"total_applications": total}), nil
}

func (e *Executor) queryTodayApplications(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	aggregation, err := e.listAllApplications(ctx, hrID, jobID, true)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	apps := aggregation.applications()
	today := businessclock.Now().Format("2006-01-02")
	var count int64
	for _, app := range apps {
		if app == nil {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(app.GetAppliedAt()), today) {
			count++
		}
	}
	payload := aggregation.metadata()
	payload["job_id"] = jobID
	payload["today_applications"] = count
	payload["date"] = today
	return jsonResult(payload), nil
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
	aggregation, err := e.listAllApplications(ctx, hrID, 0, true)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	return pageApplications(aggregation.applications(), page, pageSize, aggregation.metadata()), nil
}

func (e *Executor) listApplicationsByJob(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	if jobID <= 0 {
		return toolFailure("invalid_argument", "list_applications_by_job", "job_id is required", nil)
	}
	page, pageSize := pageArgs(args, 1, 10)
	currentOnly := true
	if v, ok := args["current_only"].(bool); ok {
		currentOnly = v
	}
	aggregation, err := e.listAllApplications(ctx, hrID, jobID, currentOnly)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	apps := aggregation.applications()
	if v, ok := optionalInt32Arg(args, "status"); ok {
		apps = filterApplicationsByStatus(apps, &v)
	}
	return pageApplications(apps, page, pageSize, aggregation.metadata()), nil
}

func (e *Executor) listApplicationsByStatus(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	status, ok := optionalInt32Arg(args, "status")
	if !ok {
		return toolFailure("invalid_argument", "list_applications_by_status", "status is required", nil)
	}
	jobID := int64Arg(args, "job_id")
	page, pageSize := pageArgs(args, 1, 10)
	currentOnly := true
	if v, ok := args["current_only"].(bool); ok {
		currentOnly = v
	}
	aggregation, err := e.listAllApplications(ctx, hrID, jobID, currentOnly)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	apps := aggregation.applications()
	apps = filterApplicationsByStatus(apps, &status)
	return pageApplications(apps, page, pageSize, aggregation.metadata()), nil
}

func (e *Executor) applicationStatusSummary(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	jobID := int64Arg(args, "job_id")
	aggregation, err := e.listAllApplications(ctx, hrID, jobID, true)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	apps := aggregation.applications()
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
	payload := aggregation.metadata()
	payload["job_id"] = jobID
	payload["summary"] = entries
	payload["total"] = int64(len(apps))
	return jsonResult(payload), nil
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
	aggregation, err := e.listAllApplications(ctx, hrID, jobID, true)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	apps := aggregation.applications()
	cutoff := businessclock.StartOfDay(businessclock.Now()).AddDate(0, 0, -int(days-1))
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
	payload := aggregation.metadata()
	payload["days"] = days
	payload["job_id"] = jobID
	payload["trend"] = points
	return jsonResult(payload), nil
}

func (e *Executor) searchCandidates(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	keyword, _ := args["keyword"].(string)
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return toolFailure("invalid_argument", "search_candidates", "keyword is required", nil)
	}
	keywordLower := strings.ToLower(keyword)
	aggregation, err := e.listAllApplications(ctx, hrID, 0, true)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	enriched := make([]map[string]any, 0)
	for _, item := range aggregation.Items {
		app := item.App
		if app == nil {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{
			app.GetRealName(),
			app.GetPhone(),
			app.GetSchool(),
			app.GetEducation(),
			item.JobTitle,
		}, " "))
		if !strings.Contains(haystack, keywordLower) {
			continue
		}
		enriched = append(enriched, map[string]any{
			"application_id": app.GetApplicationId(),
			"real_name":      app.GetRealName(),
			"phone":          maskPhone(app.GetPhone()),
			"job_title":      item.JobTitle,
			"status":         applicationStatusText(app.GetStatus()),
			"round_no":       app.GetRoundNo(),
			"is_current":     app.GetIsCurrent(),
			"applied_at":     app.GetAppliedAt(),
		})
	}
	message := ""
	if len(enriched) == 0 {
		message = "未找到匹配的候选人"
	}
	payload := aggregation.metadata()
	payload["candidates"] = enriched
	payload["message"] = message
	return jsonResult(payload), nil
}

func (e *Executor) getCandidateDetail(ctx context.Context, hrID int64, args map[string]any) (commonsai.ToolResult, error) {
	applicationID := int64Arg(args, "application_id")
	if applicationID <= 0 {
		return toolFailure("invalid_argument", "get_candidate_detail", "application_id is required", nil)
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
		return toolFailure("forbidden_or_not_found", "get_candidate_detail", "投递记录不存在或无权限访问", nil)
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
		return toolFailure("downstream", "get_candidate_detail", msg, nil)
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
		return toolFailure("invalid_argument", "propose_application_status_update", "application_id is required", nil)
	}
	if !ok || (status != 2 && status != 3) {
		return toolFailure("invalid_argument", "propose_application_status_update", "status must be 2 (通过) or 3 (淘汰)", nil)
	}
	owned, err := e.applicationOwnedByHR(ctx, hrID, applicationID)
	if err != nil {
		return commonsai.ToolResult{}, err
	}
	if !owned {
		return toolFailure("forbidden_or_not_found", "propose_application_status_update", "投递记录不存在或无权限访问", nil)
	}
	detail, detailErr := e.getCandidateDetail(ctx, hrID, map[string]any{"application_id": applicationID})
	if detailErr != nil {
		return commonsai.ToolResult{}, detailErr
	}
	candidateName := ""
	jobTitle := ""
	if detail.Content != "" {
		var payload map[string]any
		_ = jsonUnmarshal(detail.Content, &payload)
		candidateName, _ = payload["candidate_name"].(string)
		jobTitle, _ = payload["job_title"].(string)
	}
	return jsonResult(map[string]any{
		"action":           "propose_application_status_update",
		"application_id":   applicationID,
		"proposed_status":  status,
		"status_text":      applicationStatusText(status),
		"candidate_name":   candidateName,
		"job_title":        jobTitle,
		"requires_confirm": true,
		"message":          "此工具仅生成待确认动作，不会直接修改数据库；请 HR 明确确认后由系统执行状态变更。",
	}), nil
}

func (e *Executor) listAllApplications(ctx context.Context, hrID, jobID int64, currentOnly bool) (applicationAggregation, error) {
	if e.Applications == nil {
		return applicationAggregation{}, fmt.Errorf("application list service is not configured")
	}
	if jobID > 0 {
		apps, truncated, err := e.listJobApplications(ctx, hrID, jobID, currentOnly)
		if err != nil {
			return applicationAggregation{}, err
		}
		items := make([]aggregatedApplication, 0, len(apps))
		for _, app := range apps {
			items = append(items, aggregatedApplication{JobID: jobID, App: app})
		}
		return applicationAggregation{Items: items, Truncated: truncated}, nil
	}
	jobs, err := e.listHRJobsInventory(ctx, hrID)
	if err != nil {
		return applicationAggregation{}, err
	}
	filtered := make([]*pb.Job, 0, len(jobs))
	for _, job := range jobs {
		if job == nil {
			continue
		}
		filtered = append(filtered, job)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].GetJobId() < filtered[j].GetJobId() })
	jobLimitTruncated := len(filtered) > applicationAggregationMaxJobs
	if jobLimitTruncated {
		filtered = filtered[:applicationAggregationMaxJobs]
	}
	if len(filtered) == 0 {
		return applicationAggregation{}, nil
	}
	type jobResult struct {
		job       *pb.Job
		apps      []*pb.JobApplication
		truncated bool
		err       error
	}
	jobsCh := make(chan *pb.Job)
	results := make(chan jobResult, len(filtered))
	workerCount := applicationAggregationWorkers
	if len(filtered) < workerCount {
		workerCount = len(filtered)
	}
	var workers sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobsCh {
				apps, truncated, listErr := e.listJobApplications(ctx, hrID, job.GetJobId(), currentOnly)
				select {
				case results <- jobResult{job: job, apps: apps, truncated: truncated, err: listErr}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobsCh)
		for _, job := range filtered {
			select {
			case jobsCh <- job:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()
	jobResults := make([]jobResult, 0, len(filtered))
	for {
		select {
		case <-ctx.Done():
			return applicationAggregation{}, ctx.Err()
		case result, ok := <-results:
			if !ok {
				goto collected
			}
			jobResults = append(jobResults, result)
		}
	}

collected:
	sort.Slice(jobResults, func(i, j int) bool { return jobResults[i].job.GetJobId() < jobResults[j].job.GetJobId() })
	aggregation := applicationAggregation{Truncated: jobLimitTruncated}
	for _, result := range jobResults {
		if result.err != nil {
			aggregation.FailedJobCount++
			if len(aggregation.Warnings) < applicationAggregationMaxWarnings {
				aggregation.Warnings = append(aggregation.Warnings, fmt.Sprintf("job %d applications unavailable", result.job.GetJobId()))
			}
			continue
		}
		aggregation.Truncated = aggregation.Truncated || result.truncated
		sort.SliceStable(result.apps, func(i, j int) bool {
			return result.apps[i].GetApplicationId() < result.apps[j].GetApplicationId()
		})
		for _, app := range result.apps {
			if app == nil {
				continue
			}
			if len(aggregation.Items) >= applicationAggregationMaxRows {
				aggregation.Truncated = true
				continue
			}
			aggregation.Items = append(aggregation.Items, aggregatedApplication{JobID: result.job.GetJobId(), JobTitle: result.job.GetTitle(), App: app})
		}
	}
	aggregation.Partial = aggregation.FailedJobCount > 0
	if aggregation.FailedJobCount == len(filtered) {
		return applicationAggregation{}, fmt.Errorf("application aggregation failed for all %d jobs", len(filtered))
	}
	return aggregation, nil
}

func (e *Executor) listJobApplications(ctx context.Context, hrID, jobID int64, currentOnly bool) ([]*pb.JobApplication, bool, error) {
	if e.Applications == nil {
		return nil, false, fmt.Errorf("application list service is not configured")
	}
	var all []*pb.JobApplication
	var seen int64
	for page := int32(1); page <= applicationAggregationMaxPagesPerJob; page++ {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		resp, err := e.Applications.ListJobApplications(ctx, &pb.ListJobApplicationsRequest{
			HrId:     hrID,
			JobId:    jobID,
			Page:     page,
			PageSize: applicationAggregationPageSize,
		})
		if err != nil {
			return nil, false, err
		}
		if resp == nil {
			return nil, false, fmt.Errorf("list job applications returned nil")
		}
		if resp.GetCode() != errs.OK {
			msg := strings.TrimSpace(resp.GetMsg())
			if msg == "" {
				msg = "list job applications was denied"
			}
			return nil, false, fmt.Errorf("%s", msg)
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
			if len(all) >= applicationAggregationMaxRows {
				return all[:applicationAggregationMaxRows], true, nil
			}
		}
		seen += int64(len(batch))
		if len(batch) == 0 {
			return all, false, nil
		}
		if resp.GetTotal() > 0 && seen >= resp.GetTotal() {
			return all, false, nil
		}
		if int32(len(batch)) < applicationAggregationPageSize {
			return all, false, nil
		}
	}
	return all, true, nil
}

func (e *Executor) applicationOwnedByHR(ctx context.Context, hrID, applicationID int64) (bool, error) {
	aggregation, err := e.listAllApplications(ctx, hrID, 0, false)
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
	for _, app := range aggregation.applications() {
		if app != nil && app.GetApplicationId() == applicationID {
			return true, nil
		}
	}
	if aggregation.Partial || aggregation.Truncated {
		if e.Snapshots != nil {
			resp, snapErr := e.Snapshots.GetApplicationSnapshot(ctx, &pb.GetApplicationSnapshotRequest{ApplicationId: applicationID})
			if snapErr != nil {
				return false, snapErr
			}
			if resp != nil && resp.GetCode() == errs.OK {
				return resp.GetJobHrId() == hrID, nil
			}
		}
		return false, fmt.Errorf("application ownership is indeterminate because the HR aggregation was partial")
	}
	return false, nil
}

func pageApplications(apps []*pb.JobApplication, page, pageSize int32, metadata map[string]any) commonsai.ToolResult {
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
	payload := metadata
	if payload == nil {
		payload = map[string]any{}
	}
	payload["total"] = total
	payload["page"] = page
	payload["page_size"] = pageSize
	payload["applications"] = entries
	return jsonResult(payload)
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
