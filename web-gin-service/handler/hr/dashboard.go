package hr

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/pkg/logger"
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
	role := middleware.Role(c)

	// Fetch all jobs (large page to get full list for aggregation).
	jobsResp, err := h.clients.Job.ListHRJobs(c.Request.Context(), &pb.ListHRJobsRequest{
		HrId:     hrID,
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}

	// Fetch unread notification count.
	unreadResp, err := h.clients.Notification.UnreadNotificationCount(c.Request.Context(), &pb.UnreadNotificationCountRequest{
		UserId: hrID,
		Role:   role,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}

	// Compute KPIs from job list.
	var onlineJobs, offlineJobs int64
	deptMap := make(map[string]int64)

	for _, job := range jobsResp.List {
		if job.Status == 1 {
			onlineJobs++
		} else {
			offlineJobs++
		}
		deptMap[job.Department]++
	}

	// Compute pending actions and stage distribution from per-job applications.
	// Collect unique user_ids to count distinct candidates (not total application records).
	// Cap at 50 jobs to keep response time reasonable.
	var pendingActions int64
	stageCounts := make(map[int32]int64)
	userIDs := make(map[int64]bool)
	var mu sync.Mutex
	jobCount := len(jobsResp.List)
	if jobCount > 50 {
		jobCount = 50
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	eg, ctx := errgroup.WithContext(ctx)
	for i := 0; i < jobCount; i++ {
		jobID := jobsResp.List[i].JobId
		eg.Go(func() error {
			appsResp, err := h.clients.Application.ListJobApplications(ctx, &pb.ListJobApplicationsRequest{
				HrId:     hrID,
				JobId:    jobID,
				Page:     1,
				PageSize: 500,
			})
			if err != nil {
				logger.L().Warn("dashboard: list job applications failed", zap.Int64("job_id", jobID), zap.Error(err))
				return nil // don't fail the whole request; continue with other jobs
			}
			mu.Lock()
			defer mu.Unlock()
			for _, app := range appsResp.List {
				userIDs[app.UserId] = true
				stageCounts[app.Status]++
				if app.Status == 0 {
					pendingActions++
				}
			}
			return nil
		})
	}
	_ = eg.Wait() // errors are logged per-job above
	totalCandidates := int64(len(userIDs))

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

	// Build stage distribution from collected counts (status 0-3 maps to labels).
	stageLabels := []string{"待查看", "已查看", "通过", "淘汰"}
	stageValues := make([]int64, 4)
	for i := 0; i < 4; i++ {
		stageValues[i] = stageCounts[int32(i)]
	}

	// Trend data: placeholder for now (requires time-series aggregation not available from current gRPC APIs).
	trendDates := make([]string, 7)
	trendApps := make([]int64, 7)

	base.OK(c, "success", gin.H{
		"kpi": gin.H{
			"online_jobs":          onlineJobs,
			"offline_jobs":         offlineJobs,
			"total_applications":   totalCandidates,
			"today_applications":   0,
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
