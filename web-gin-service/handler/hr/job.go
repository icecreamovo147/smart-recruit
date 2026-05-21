package hr

import (
	"strconv"

	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type JobHandler struct {
	clients *rpc.Clients
}

func NewJobHandler(clients *rpc.Clients) *JobHandler {
	return &JobHandler{clients: clients}
}

func (h *JobHandler) Create(c *gin.Context) {
	var req jobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Job.CreateJob(c.Request.Context(), &pb.CreateJobRequest{HrId: middleware.UserID(c), Title: req.Title, Department: req.Department, DepartmentId: req.DepartmentID, Location: req.Location, LocationId: req.LocationID, SalaryRange: req.SalaryRange, Description: req.Description, Requirements: req.Requirements})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *JobHandler) Update(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("job_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "岗位 ID 不合法")
		return
	}
	var req jobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Job.UpdateJob(c.Request.Context(), &pb.UpdateJobRequest{HrId: middleware.UserID(c), JobId: jobID, Title: req.Title, Department: req.Department, DepartmentId: req.DepartmentID, Location: req.Location, LocationId: req.LocationID, SalaryRange: req.SalaryRange, Description: req.Description, Requirements: req.Requirements})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *JobHandler) Offline(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("job_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "岗位 ID 不合法")
		return
	}
	resp, err := h.clients.Job.OfflineJob(c.Request.Context(), &pb.OfflineJobRequest{HrId: middleware.UserID(c), JobId: jobID})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *JobHandler) Online(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("job_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "岗位 ID 不合法")
		return
	}
	resp, err := h.clients.Job.OnlineJob(c.Request.Context(), &pb.OfflineJobRequest{HrId: middleware.UserID(c), JobId: jobID})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *JobHandler) List(c *gin.Context) {
	page, pageSize := basePagination(c)
	cursor, hasCursor := c.GetQuery("cursor")
	if hasCursor {
		page = 0
	}
	resp, err := h.clients.Job.ListHRJobs(c.Request.Context(), &pb.ListHRJobsRequest{HrId: middleware.UserID(c), Page: page, PageSize: pageSize, Cursor: cursor})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *JobHandler) JobOptions(c *gin.Context) {
	resp, err := h.clients.Job.ListJobOptions(c.Request.Context(), &pb.ListJobOptionsRequest{})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

type jobRequest struct {
	Title         string `json:"title" binding:"required"`
	Department    string `json:"department"`
	DepartmentID  int64  `json:"department_id"`
	Location      string `json:"location"`
	LocationID    int64  `json:"location_id"`
	SalaryRange   string `json:"salary_range"`
	Description   string `json:"description"`
	Requirements  string `json:"requirements"`
}

func basePagination(c *gin.Context) (int32, int32) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	return int32(page), int32(pageSize)
}
