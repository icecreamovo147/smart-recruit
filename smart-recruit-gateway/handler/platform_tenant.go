package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

type PlatformTenantHandler struct {
	clients *rpc.Clients
}

func NewPlatformTenantHandler(clients *rpc.Clients) *PlatformTenantHandler {
	return &PlatformTenantHandler{clients: clients}
}

func (h *PlatformTenantHandler) Create(c *gin.Context) {
	var req struct {
		Slug     string `json:"slug" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Timezone string `json:"timezone"`
		Locale   string `json:"locale"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Tenant.CreateTenant(c.Request.Context(), &pb.CreateTenantRequest{Slug: req.Slug, Name: req.Name, Timezone: req.Timezone, Locale: req.Locale})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) List(c *gin.Context) {
	resp, err := h.clients.Tenant.ListTenants(c.Request.Context(), &pb.ListTenantsRequest{
		Page: int32(queryInt(c, "page", 1)), PageSize: int32(queryInt(c, "page_size", 20)),
		Keyword: c.Query("keyword"), Status: c.Query("status"),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) Get(c *gin.Context) {
	tenantID, ok := platformIDParam(c, "tenant_id")
	if !ok {
		return
	}
	resp, err := h.clients.Tenant.GetTenant(c.Request.Context(), &pb.GetTenantRequest{TenantId: tenantID})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) UpdateStatus(c *gin.Context) {
	tenantID, err := strconv.ParseInt(c.Param("tenant_id"), 10, 64)
	if err != nil || tenantID <= 0 {
		BadRequest(c, "企业 ID 无效")
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Tenant.UpdateTenantStatus(c.Request.Context(), &pb.UpdateTenantStatusRequest{TenantId: tenantID, Status: req.Status, Reason: strings.TrimSpace(req.Reason)})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) UpdateMembershipStatus(c *gin.Context) {
	tenantID, ok := platformIDParam(c, "tenant_id")
	if !ok {
		return
	}
	membershipID, ok := platformIDParam(c, "membership_id")
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Tenant.UpdateTenantMembershipStatus(c.Request.Context(), &pb.UpdateTenantMembershipStatusRequest{
		TenantId: tenantID, MembershipId: membershipID, Status: req.Status, Reason: strings.TrimSpace(req.Reason),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) Dashboard(c *gin.Context) {
	resp, err := h.clients.Tenant.GetPlatformDashboard(c.Request.Context(), &pb.GetPlatformDashboardRequest{})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) AuditLogs(c *gin.Context) {
	resp, err := h.clients.Tenant.QueryPlatformAuditLogs(c.Request.Context(), &pb.QueryPlatformAuditLogsRequest{
		TenantId: queryInt64(c, "tenant_id"), ActorUserId: queryInt64(c, "actor_user_id"),
		Action: c.Query("action"), RequestId: c.Query("request_id"), StartTime: c.Query("start_time"), EndTime: c.Query("end_time"),
		Page: int32(queryInt(c, "page", 1)), PageSize: int32(queryInt(c, "page_size", 20)),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) ListPlans(c *gin.Context) {
	resp, err := h.clients.Tenant.ListPlatformPlans(c.Request.Context(), &pb.ListPlatformPlansRequest{Status: c.Query("status")})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) SavePlanVersion(c *gin.Context) {
	planID, ok := platformIDParam(c, "plan_id")
	if !ok {
		return
	}
	var req struct {
		VersionID    int64                         `json:"version_id"`
		ChangeNote   string                        `json:"change_note" binding:"required"`
		Entitlements []*pb.PlatformEntitlementItem `json:"entitlements" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Tenant.SavePlatformPlanVersion(c.Request.Context(), &pb.SavePlatformPlanVersionRequest{PlanId: planID, VersionId: req.VersionID, ChangeNote: req.ChangeNote, Entitlements: req.Entitlements})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) PublishPlanVersion(c *gin.Context) {
	planID, ok := platformIDParam(c, "plan_id")
	if !ok {
		return
	}
	versionID, ok := platformIDParam(c, "version_id")
	if !ok {
		return
	}
	var req struct {
		EffectiveAt string `json:"effective_at" binding:"required"`
		Reason      string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Tenant.PublishPlatformPlanVersion(c.Request.Context(), &pb.PublishPlatformPlanVersionRequest{PlanId: planID, VersionId: versionID, EffectiveAt: req.EffectiveAt, Reason: req.Reason})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) GetSubscription(c *gin.Context) {
	tenantID, ok := platformIDParam(c, "tenant_id")
	if !ok {
		return
	}
	resp, err := h.clients.Tenant.GetTenantSubscription(c.Request.Context(), &pb.GetTenantSubscriptionRequest{TenantId: tenantID})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) UpdateSubscription(c *gin.Context) {
	tenantID, ok := platformIDParam(c, "tenant_id")
	if !ok {
		return
	}
	var req struct {
		PlanVersionID int64  `json:"plan_version_id" binding:"required"`
		StartsAt      string `json:"starts_at" binding:"required"`
		EndsAt        string `json:"ends_at"`
		Reason        string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Tenant.UpdateTenantSubscription(c.Request.Context(), &pb.UpdateTenantSubscriptionRequest{TenantId: tenantID, PlanVersionId: req.PlanVersionID, StartsAt: req.StartsAt, EndsAt: req.EndsAt, Reason: req.Reason})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) UpdateEntitlementOverride(c *gin.Context) {
	tenantID, ok := platformIDParam(c, "tenant_id")
	if !ok {
		return
	}
	var req struct {
		EntitlementKey string `json:"entitlement_key" binding:"required"`
		ValueType      string `json:"value_type" binding:"required"`
		ValueJSON      string `json:"value_json" binding:"required"`
		ExpiresAt      string `json:"expires_at"`
		Reason         string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Tenant.UpdateTenantEntitlementOverride(c.Request.Context(), &pb.UpdateTenantEntitlementOverrideRequest{TenantId: tenantID, EntitlementKey: req.EntitlementKey, ValueType: req.ValueType, ValueJson: req.ValueJSON, ExpiresAt: req.ExpiresAt, Reason: req.Reason})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) GetUsage(c *gin.Context) {
	tenantID, ok := platformIDParam(c, "tenant_id")
	if !ok {
		return
	}
	resp, err := h.clients.Tenant.GetTenantUsage(c.Request.Context(), &pb.GetTenantUsageRequest{TenantId: tenantID})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) ListAlerts(c *gin.Context) {
	resp, err := h.clients.Tenant.ListQuotaAlerts(c.Request.Context(), &pb.ListQuotaAlertsRequest{TenantId: queryInt64(c, "tenant_id"), Status: c.Query("status"), MetricKey: c.Query("metric_key"), Page: int32(queryInt(c, "page", 1)), PageSize: int32(queryInt(c, "page_size", 20))})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) UpdateAlert(c *gin.Context) {
	alertID, ok := platformIDParam(c, "alert_id")
	if !ok {
		return
	}
	var req struct {
		Status         string `json:"status" binding:"required"`
		AssigneeUserID int64  `json:"assignee_user_id"`
		ResolutionNote string `json:"resolution_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Tenant.UpdateQuotaAlert(c.Request.Context(), &pb.UpdateQuotaAlertRequest{AlertId: alertID, Status: req.Status, AssigneeUserId: req.AssigneeUserID, ResolutionNote: req.ResolutionNote})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PlatformTenantHandler) ListMemberships(c *gin.Context) {
	tenantID, err := strconv.ParseInt(c.Param("tenant_id"), 10, 64)
	if err != nil || tenantID <= 0 {
		BadRequest(c, "企业 ID 无效")
		return
	}
	resp, err := h.clients.Tenant.ListTenantMemberships(c.Request.Context(), &pb.ListTenantMembershipsRequest{
		TenantId: tenantID, Page: int32(queryInt(c, "page", 1)), PageSize: int32(queryInt(c, "page_size", 20)),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func queryInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func queryInt64(c *gin.Context, key string) int64 {
	value, _ := strconv.ParseInt(c.Query(key), 10, 64)
	return value
}

func platformIDParam(c *gin.Context, key string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil || value <= 0 {
		BadRequest(c, "ID 无效")
		return 0, false
	}
	return value, true
}
