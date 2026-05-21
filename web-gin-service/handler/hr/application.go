package hr

import (
	"strconv"

	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type ApplicationHandler struct {
	clients *rpc.Clients
}

func NewApplicationHandler(clients *rpc.Clients) *ApplicationHandler {
	return &ApplicationHandler{clients: clients}
}

func (h *ApplicationHandler) ListByJob(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("job_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "岗位 ID 不合法")
		return
	}
	page, pageSize := basePagination(c)
	resp, err := h.clients.Application.ListJobApplications(c.Request.Context(), &pb.ListJobApplicationsRequest{HrId: middleware.UserID(c), JobId: jobID, Page: page, PageSize: pageSize})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *ApplicationHandler) UpdateStatus(c *gin.Context) {
	applicationID, err := strconv.ParseInt(c.Param("application_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "投递记录 ID 不合法")
		return
	}
	var req struct {
		Status *int32 `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Status == nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Application.UpdateApplicationStatus(c.Request.Context(), &pb.UpdateApplicationStatusRequest{
		HrId:          middleware.UserID(c),
		ApplicationId: applicationID,
		Status:        *req.Status,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}
