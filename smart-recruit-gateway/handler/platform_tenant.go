package handler

import (
	"strconv"

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

func (h *PlatformTenantHandler) UpdateStatus(c *gin.Context) {
	tenantID, err := strconv.ParseInt(c.Param("tenant_id"), 10, 64)
	if err != nil || tenantID <= 0 {
		BadRequest(c, "企业 ID 无效")
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Tenant.UpdateTenantStatus(c.Request.Context(), &pb.UpdateTenantStatusRequest{TenantId: tenantID, Status: req.Status})
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
