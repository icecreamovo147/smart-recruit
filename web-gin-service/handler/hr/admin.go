package hr

import (
	"strconv"

	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type AdminHandler struct {
	clients *rpc.Clients
}

func NewAdminHandler(clients *rpc.Clients) *AdminHandler {
	return &AdminHandler{clients: clients}
}

func (h *AdminHandler) CreateInviteCode(c *gin.Context) {
	var req struct {
		ExpiresAt string `json:"expires_at"` // RFC 3339, empty = never expires
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.CreateInviteCode(c.Request.Context(), &pb.CreateInviteCodeRequest{
		CreatedBy: middleware.UserID(c),
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) ListInviteCodes(c *gin.Context) {
	var req struct {
		Page     int32 `form:"page"`
		PageSize int32 `form:"page_size"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.ListInviteCodes(c.Request.Context(), &pb.ListInviteCodesRequest{
		CreatedBy: middleware.UserID(c),
		Page:      req.Page,
		PageSize:  req.PageSize,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) ExtendInviteCode(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的邀请码 ID")
		return
	}
	var req struct {
		NewExpiresAt string `json:"new_expires_at" binding:"required"` // RFC 3339
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.ExtendInviteCode(c.Request.Context(), &pb.ExtendInviteCodeRequest{
		AdminId:      middleware.UserID(c),
		Id:           id,
		NewExpiresAt: req.NewExpiresAt,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) RevokeInviteCode(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的邀请码 ID")
		return
	}
	resp, err := h.clients.Admin.RevokeInviteCode(c.Request.Context(), &pb.RevokeInviteCodeRequest{
		AdminId: middleware.UserID(c),
		Id:      id,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) ReactivateInviteCode(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的邀请码 ID")
		return
	}
	resp, err := h.clients.Admin.ReactivateInviteCode(c.Request.Context(), &pb.ReactivateInviteCodeRequest{
		AdminId: middleware.UserID(c),
		Id:      id,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ── Department CRUD ────────────────────────────────────────────────

func (h *AdminHandler) ListDepartments(c *gin.Context) {
	resp, err := h.clients.Admin.ListDepartments(c.Request.Context(), &pb.ListDepartmentsRequest{})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) CreateDepartment(c *gin.Context) {
	var req struct {
		ParentID  int64  `json:"parent_id"`
		Name      string `json:"name" binding:"required"`
		SortOrder int32  `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.CreateDepartment(c.Request.Context(), &pb.CreateDepartmentRequest{
		AdminId:   middleware.UserID(c),
		ParentId:  req.ParentID,
		Name:      req.Name,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) UpdateDepartment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的部门 ID")
		return
	}
	var req struct {
		ParentID  int64  `json:"parent_id"`
		Name      string `json:"name"`
		SortOrder int32  `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.UpdateDepartment(c.Request.Context(), &pb.UpdateDepartmentRequest{
		AdminId:   middleware.UserID(c),
		Id:        id,
		ParentId:  req.ParentID,
		Name:      req.Name,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) UpdateDepartmentStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的部门 ID")
		return
	}
	var req struct {
		IsActive int32 `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.UpdateDepartmentStatus(c.Request.Context(), &pb.UpdateDepartmentStatusRequest{
		AdminId:  middleware.UserID(c),
		Id:       id,
		IsActive: req.IsActive,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) DeleteDepartment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的部门 ID")
		return
	}
	resp, err := h.clients.Admin.DeleteDepartment(c.Request.Context(), &pb.DeleteDepartmentRequest{
		AdminId: middleware.UserID(c),
		Id:      id,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ── Location CRUD ──────────────────────────────────────────────────

func (h *AdminHandler) ListJobLocations(c *gin.Context) {
	resp, err := h.clients.Admin.ListJobLocations(c.Request.Context(), &pb.ListJobLocationsRequest{})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) CreateJobLocation(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		Code      string `json:"code"`
		SortOrder int32  `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.CreateJobLocation(c.Request.Context(), &pb.CreateJobLocationRequest{
		AdminId:   middleware.UserID(c),
		Name:      req.Name,
		Code:      req.Code,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) UpdateJobLocation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的地点 ID")
		return
	}
	var req struct {
		Name      string `json:"name"`
		Code      string `json:"code"`
		SortOrder int32  `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.UpdateJobLocation(c.Request.Context(), &pb.UpdateJobLocationRequest{
		AdminId:   middleware.UserID(c),
		Id:        id,
		Name:      req.Name,
		Code:      req.Code,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) UpdateJobLocationStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的地点 ID")
		return
	}
	var req struct {
		IsActive int32 `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.UpdateJobLocationStatus(c.Request.Context(), &pb.UpdateJobLocationStatusRequest{
		AdminId:  middleware.UserID(c),
		Id:       id,
		IsActive: req.IsActive,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) DeleteJobLocation(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的地点 ID")
		return
	}
	resp, err := h.clients.Admin.DeleteJobLocation(c.Request.Context(), &pb.DeleteJobLocationRequest{
		AdminId: middleware.UserID(c),
		Id:      id,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ── Department Location Config ─────────────────────────────────────

func (h *AdminHandler) GetDepartmentLocationConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的部门 ID")
		return
	}
	resp, err := h.clients.Admin.GetDepartmentLocationConfig(c.Request.Context(), &pb.GetDepartmentLocationConfigRequest{
		DepartmentId: id,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) ListDepartmentsLocationMap(c *gin.Context) {
	resp, err := h.clients.Admin.ListDepartmentsLocationMap(c.Request.Context(), &pb.ListDepartmentsLocationMapRequest{})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AdminHandler) UpdateDepartmentLocationConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "无效的部门 ID")
		return
	}
	var req struct {
		InheritLocations int32   `json:"inherit_locations"`
		LocationIds      []int64 `json:"location_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.UpdateDepartmentLocationConfig(c.Request.Context(), &pb.UpdateDepartmentLocationConfigRequest{
		AdminId:          middleware.UserID(c),
		DepartmentId:     id,
		InheritLocations: req.InheritLocations,
		LocationIds:      req.LocationIds,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ── Usage Audit ──────────────────────────────────────────────────────

func (h *AdminHandler) ListUsageLogs(c *gin.Context) {
	var req struct {
		Page        int32  `form:"page"`
		PageSize    int32  `form:"page_size"`
		ServiceType string `form:"service_type"`
		Provider    string `form:"provider"`
		Status      string `form:"status"`
		UserID      int64  `form:"user_id"`
		RequestID   string `form:"request_id"`
		StartTime   string `form:"start_time"`
		EndTime     string `form:"end_time"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Admin.QueryUsageLogs(c.Request.Context(), &pb.QueryUsageLogsRequest{
		Page:        req.Page,
		PageSize:    req.PageSize,
		ServiceType: req.ServiceType,
		Provider:    req.Provider,
		Status:      req.Status,
		UserId:      req.UserID,
		RequestId:   req.RequestID,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}
