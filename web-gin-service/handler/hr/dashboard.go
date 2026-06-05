package hr

import (
	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type DashboardHandler struct {
	clients *rpc.Clients
}

func NewDashboardHandler(clients *rpc.Clients) *DashboardHandler {
	return &DashboardHandler{clients: clients}
}

func (h *DashboardHandler) Summary(c *gin.Context) {
	hrID := middleware.UserID(c)
	acctType := middleware.AccountType(c)

	ctx := c.Request.Context()

	// Fetch dashboard report from analytics service.
	reportResp, err := h.clients.Admin.GetDashboardReport(ctx, &pb.GetDashboardReportRequest{
		StaffUserId: hrID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}

	// Fetch unread notification count.
	unreadResp, err := h.clients.Notification.UnreadNotificationCount(ctx, &pb.UnreadNotificationCountRequest{
		UserId:      hrID,
		AccountType: acctType,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}

	// Fetch jobs for department distribution.
	jobsResp, err := h.clients.Job.ListHRJobs(ctx, &pb.ListHRJobsRequest{
		HrId:     hrID,
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}

	// Compute KPI from analytics report.
	onlineJobs := reportResp.GetOnlineJobs()
	offlineJobs := reportResp.GetOfflineJobs()
	totalApplications := reportResp.GetTotalApplications()
	todayApplications := reportResp.GetTodayApplications()
	pendingActions := reportResp.GetPendingActions()

	// Build department distribution from job list.
	deptMap := make(map[string]int64)
	for _, job := range jobsResp.List {
		deptMap[job.Department]++
	}

	// Compute pending actions and stage distribution from analytics report.
	var pendingActionsFromReport int64
	stageCounts := make(map[string]int64)
	for _, item := range reportResp.GetStageDistribution() {
		stageCounts[item.StageKey] = item.Count
		if item.StageKey == "applied" || item.StageKey == "viewed" {
			pendingActionsFromReport += item.Count
		}
	}
	_ = pendingActionsFromReport // prefer report's pending_actions

	unreadNotifications := int64(0)
	if unreadResp != nil {
		unreadNotifications = unreadResp.Unread
	}

	// Build job distribution by department.
	deptLabels := make([]string, 0, len(deptMap))
	deptValues := make([]int64, 0, len(deptMap))
	for dept, count := range deptMap {
		deptLabels = append(deptLabels, dept)
		deptValues = append(deptValues, count)
	}

	// Build stage distribution from report.
	stageLabels := []string{"已投递", "已查看", "筛选中", "筛选通过", "面试中", "面试通过", "待发Offer", "Offer已发", "已入职", "淘汰"}
	stageKeys := []string{"applied", "viewed", "screening", "screen_passed", "interviewing", "interview_passed", "offer_pending", "offer_sent", "hired", "rejected"}
	stageValues := make([]int64, len(stageLabels))
	for i, key := range stageKeys {
		stageValues[i] = stageCounts[key]
	}

	// Build trend data from report.
	trendDates := make([]string, 0)
	trendApps := make([]int64, 0)
	for _, p := range reportResp.GetTrend() {
		trendDates = append(trendDates, p.Date)
		trendApps = append(trendApps, p.Applications)
	}

	base.OK(c, "success", gin.H{
		"kpi": gin.H{
			"online_jobs":          onlineJobs,
			"offline_jobs":         offlineJobs,
			"total_applications":   totalApplications,
			"today_applications":   todayApplications,
			"unread_notifications": unreadNotifications,
			"pending_actions":      pendingActions,
		},
		"trend": gin.H{
			"dates":        trendDates,
			"applications": trendApps,
		},
		"stage_distribution": gin.H{
			"labels": stageLabels,
			"values": stageValues,
		},
		"job_distribution": gin.H{
			"labels": deptLabels,
			"values": deptValues,
		},
	})
}
