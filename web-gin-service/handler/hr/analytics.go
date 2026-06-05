package hr

import (
	"strconv"

	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

// AnalyticsHandler handles analytics and report endpoints.
type AnalyticsHandler struct {
	clients *rpc.Clients
}

func NewAnalyticsHandler(clients *rpc.Clients) *AnalyticsHandler {
	return &AnalyticsHandler{clients: clients}
}

// DashboardReport returns the scoped dashboard KPI and distribution data.
func (h *AnalyticsHandler) DashboardReport(c *gin.Context) {
	staffUserID := middleware.UserID(c)
	resp, err := h.clients.Admin.GetDashboardReport(c.Request.Context(), &pb.GetDashboardReportRequest{
		StaffUserId: staffUserID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// FunnelReport returns the recruitment funnel with conversion rates.
func (h *AnalyticsHandler) FunnelReport(c *gin.Context) {
	staffUserID := middleware.UserID(c)
	jobIDStr := c.Query("job_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var jobID int64
	if jobIDStr != "" {
		jobID, _ = strconv.ParseInt(jobIDStr, 10, 64)
	}

	resp, err := h.clients.Admin.GetFunnelReport(c.Request.Context(), &pb.GetFunnelReportRequest{
		StaffUserId: staffUserID,
		JobId:       jobID,
		StartDate:   startDate,
		EndDate:     endDate,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// TimeInStageReport returns average time spent in each pipeline stage.
func (h *AnalyticsHandler) TimeInStageReport(c *gin.Context) {
	staffUserID := middleware.UserID(c)
	jobIDStr := c.Query("job_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var jobID int64
	if jobIDStr != "" {
		jobID, _ = strconv.ParseInt(jobIDStr, 10, 64)
	}

	resp, err := h.clients.Admin.GetTimeInStageReport(c.Request.Context(), &pb.GetTimeInStageReportRequest{
		StaffUserId: staffUserID,
		JobId:       jobID,
		StartDate:   startDate,
		EndDate:     endDate,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// InterviewOfferMetrics returns interview pass rate and offer acceptance rate.
func (h *AnalyticsHandler) InterviewOfferMetrics(c *gin.Context) {
	staffUserID := middleware.UserID(c)
	jobIDStr := c.Query("job_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var jobID int64
	if jobIDStr != "" {
		jobID, _ = strconv.ParseInt(jobIDStr, 10, 64)
	}

	resp, err := h.clients.Admin.GetInterviewOfferMetrics(c.Request.Context(), &pb.GetInterviewOfferMetricsRequest{
		StaffUserId: staffUserID,
		JobId:       jobID,
		StartDate:   startDate,
		EndDate:     endDate,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// AuthAuditLogs returns authorization audit logs (security audit).
func (h *AnalyticsHandler) AuthAuditLogs(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var actorUserID int64
	if aid := c.Query("actor_user_id"); aid != "" {
		actorUserID, _ = strconv.ParseInt(aid, 10, 64)
	}
	permissionKey := c.Query("permission_key")
	decision := c.Query("decision")

	resp, err := h.clients.Admin.QueryAuthAuditLogs(c.Request.Context(), &pb.QueryAuthAuditLogsRequest{
		Page:          page,
		PageSize:      pageSize,
		ActorUserId:   actorUserID,
		PermissionKey: permissionKey,
		Decision:      decision,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func parsePagination(c *gin.Context) (int32, int32) {
	page := int32(1)
	pageSize := int32(20)
	if p := c.Query("page"); p != "" {
		if v, err := strconv.ParseInt(p, 10, 32); err == nil && v > 0 {
			page = int32(v)
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.ParseInt(ps, 10, 32); err == nil && v > 0 && v <= 100 {
			pageSize = int32(v)
		}
	}
	return page, pageSize
}
