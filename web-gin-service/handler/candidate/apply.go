package candidate

import (
	"strconv"

	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type ApplyHandler struct {
	clients *rpc.Clients
}

func NewApplyHandler(clients *rpc.Clients) *ApplyHandler {
	return &ApplyHandler{clients: clients}
}

func (h *ApplyHandler) Apply(c *gin.Context) {
	var req struct {
		JobID int64 `json:"job_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Application.ApplyJob(c.Request.Context(), &pb.ApplyJobRequest{UserId: middleware.UserID(c), JobId: req.JobID})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *ApplyHandler) Mine(c *gin.Context) {
	page, pageSize := pagination(c)
	cursor, hasCursor := c.GetQuery("cursor")
	if hasCursor {
		page = 0
	}
	resp, err := h.clients.Application.ListMyApplications(c.Request.Context(), &pb.ListMyApplicationsRequest{UserId: middleware.UserID(c), Page: page, PageSize: pageSize, Cursor: cursor})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func pagination(c *gin.Context) (int32, int32) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	return int32(page), int32(pageSize)
}
