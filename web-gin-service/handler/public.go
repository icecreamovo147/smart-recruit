package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type PublicHandler struct {
	clients *rpc.Clients
}

func NewPublicHandler(clients *rpc.Clients) *PublicHandler {
	return &PublicHandler{clients: clients}
}

func (h *PublicHandler) ListJobs(c *gin.Context) {
	page, pageSize := pagination(c)
	cursor, hasCursor := c.GetQuery("cursor")
	if hasCursor {
		if err := validateCursor(cursor); err != nil {
			BadRequest(c, "cursor 参数格式错误")
			return
		}
		page = 0
	}
	resp, err := h.clients.Job.ListPublicJobs(c.Request.Context(), &pb.ListPublicJobsRequest{Page: page, PageSize: pageSize, Keyword: c.Query("keyword"), Cursor: cursor})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

func (h *PublicHandler) JobDetail(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("job_id"), 10, 64)
	if err != nil {
		BadRequest(c, "岗位 ID 不合法")
		return
	}
	resp, err := h.clients.Job.GetJobDetail(c.Request.Context(), &pb.GetJobDetailRequest{JobId: jobID})
	if err != nil {
		Internal(c, err)
		return
	}
	if resp.Job == nil {
		BadRequest(c, resp.Msg)
		return
	}
	From(c, resp.Code, resp.Msg, resp.Job)
}

func pagination(c *gin.Context) (int32, int32) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return int32(page), int32(pageSize)
}

// validateCursor validates the cursor format as a basic sanity check.
// Valid cursors are non-empty strings up to 256 characters containing only
// printable ASCII characters. This is a defense-in-depth check; the actual
// cursor parsing is done server-side.
func validateCursor(cursor string) error {
	if len(cursor) == 0 || len(cursor) > 256 {
		return errors.New("cursor length out of range")
	}
	for _, b := range cursor {
		if b < 0x20 || b > 0x7e {
			return errors.New("cursor contains invalid characters")
		}
	}
	return nil
}
