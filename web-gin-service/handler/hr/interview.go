package hr

import (
	"strconv"

	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type InterviewHandler struct {
	clients *rpc.Clients
}

func NewInterviewHandler(clients *rpc.Clients) *InterviewHandler {
	return &InterviewHandler{clients: clients}
}

func (h *InterviewHandler) Schedule(c *gin.Context) {
	var req struct {
		ApplicationID   int64  `json:"application_id" binding:"required"`
		InterviewerID   int64  `json:"interviewer_id" binding:"required"`
		RoundNo         int32  `json:"round_no"`
		Title           string `json:"title"`
		Mode            string `json:"mode"`
		MeetingURL      string `json:"meeting_url"`
		Location        string `json:"location"`
		DurationMinutes int32  `json:"duration_minutes"`
		CandidateNote   string `json:"candidate_note"`
		InternalNote    string `json:"internal_note"`
		ScheduledAt     string `json:"scheduled_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Interview.ScheduleInterview(c.Request.Context(), &pb.ScheduleInterviewRequest{
		HrId:            middleware.UserID(c),
		ApplicationId:   req.ApplicationID,
		InterviewerId:   req.InterviewerID,
		RoundNo:         req.RoundNo,
		Title:           req.Title,
		Mode:            req.Mode,
		MeetingUrl:      req.MeetingURL,
		Location:        req.Location,
		DurationMinutes: req.DurationMinutes,
		CandidateNote:   req.CandidateNote,
		InternalNote:    req.InternalNote,
		ScheduledAt:     req.ScheduledAt,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *InterviewHandler) Update(c *gin.Context) {
	interviewID, err := strconv.ParseInt(c.Param("interview_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "面试 ID 不合法")
		return
	}
	var req struct {
		Title           string `json:"title"`
		Mode            string `json:"mode"`
		MeetingURL      string `json:"meeting_url"`
		Location        string `json:"location"`
		DurationMinutes int32  `json:"duration_minutes"`
		CandidateNote   string `json:"candidate_note"`
		InternalNote    string `json:"internal_note"`
		ScheduledAt     string `json:"scheduled_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Interview.UpdateInterview(c.Request.Context(), &pb.UpdateInterviewRequest{
		HrId:            middleware.UserID(c),
		InterviewId:     interviewID,
		Title:           req.Title,
		Mode:            req.Mode,
		MeetingUrl:      req.MeetingURL,
		Location:        req.Location,
		DurationMinutes: req.DurationMinutes,
		CandidateNote:   req.CandidateNote,
		InternalNote:    req.InternalNote,
		ScheduledAt:     req.ScheduledAt,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *InterviewHandler) Cancel(c *gin.Context) {
	interviewID, err := strconv.ParseInt(c.Param("interview_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "面试 ID 不合法")
		return
	}
	var req struct {
		CancelReason string `json:"cancel_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Interview.CancelInterview(c.Request.Context(), &pb.CancelInterviewRequest{
		HrId:         middleware.UserID(c),
		InterviewId:  interviewID,
		CancelReason: req.CancelReason,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *InterviewHandler) ListByApplication(c *gin.Context) {
	applicationID, err := strconv.ParseInt(c.Param("application_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "投递记录 ID 不合法")
		return
	}
	resp, err := h.clients.Interview.ListApplicationInterviews(c.Request.Context(), &pb.ListApplicationInterviewsRequest{
		HrId:          middleware.UserID(c),
		ApplicationId: applicationID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *InterviewHandler) Get(c *gin.Context) {
	interviewID, err := strconv.ParseInt(c.Param("interview_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "面试 ID 不合法")
		return
	}
	resp, err := h.clients.Interview.GetInterview(c.Request.Context(), &pb.GetInterviewRequest{
		UserId:      middleware.UserID(c),
		InterviewId: interviewID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// ListMy returns the current staff user's assigned interviews (interviewer view).
func (h *InterviewHandler) ListMy(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	resp, err := h.clients.Interview.ListMyInterviews(c.Request.Context(), &pb.ListMyInterviewsRequest{
		InterviewerId: middleware.UserID(c),
		Status:        status,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// SubmitFeedback handles feedback submission from an interviewer.
func (h *InterviewHandler) SubmitFeedback(c *gin.Context) {
	interviewID, err := strconv.ParseInt(c.Param("interview_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "面试 ID 不合法")
		return
	}
	var req struct {
		ApplicationID     int64  `json:"application_id" binding:"required"`
		Recommendation    string `json:"recommendation" binding:"required"`
		Score             int32  `json:"score"`
		DimensionScores   string `json:"dimension_scores_json"`
		Comments          string `json:"comments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Interview.SubmitFeedback(c.Request.Context(), &pb.SubmitFeedbackRequest{
		InterviewerId:     middleware.UserID(c),
		InterviewId:       interviewID,
		ApplicationId:     req.ApplicationID,
		Recommendation:    req.Recommendation,
		Score:             req.Score,
		DimensionScoresJson: req.DimensionScores,
		Comments:          req.Comments,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// GetFeedback returns the feedback submitted by the current user for an interview.
func (h *InterviewHandler) GetFeedback(c *gin.Context) {
	interviewID, err := strconv.ParseInt(c.Param("interview_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "面试 ID 不合法")
		return
	}
	resp, err := h.clients.Interview.GetFeedback(c.Request.Context(), &pb.GetFeedbackRequest{
		InterviewerId: middleware.UserID(c),
		InterviewId:   interviewID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

