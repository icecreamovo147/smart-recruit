package hr

import (
	"strconv"

	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type RecruitingIntelligenceHandler struct {
	clients *rpc.Clients
}

func NewRecruitingIntelligenceHandler(clients *rpc.Clients) *RecruitingIntelligenceHandler {
	return &RecruitingIntelligenceHandler{clients: clients}
}

func (h *RecruitingIntelligenceHandler) GetResumeProfileByApplication(c *gin.Context) {
	applicationID, err := parseApplicationIDParam(c)
	if err != nil {
		base.BadRequest(c, "投递记录 ID 不合法")
		return
	}
	resp, err := h.clients.RecruitingIntelligence.GetResumeProfile(c.Request.Context(), &pb.GetResumeProfileRequest{
		StaffUserId:   middleware.UserID(c),
		ApplicationId: applicationID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *RecruitingIntelligenceHandler) GetResumeProfile(c *gin.Context) {
	req := &pb.GetResumeProfileRequest{
		StaffUserId:   middleware.UserID(c),
		ResumeId:      parseInt64Query(c, "resume_id"),
		ApplicationId: parseInt64Query(c, "application_id"),
		ProfileId:     parseUint64Query(c, "profile_id"),
	}
	resp, err := h.clients.RecruitingIntelligence.GetResumeProfile(c.Request.Context(), req)
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *RecruitingIntelligenceHandler) ParseResumeProfile(c *gin.Context) {
	var req struct {
		ResumeID      int64 `json:"resume_id"`
		ApplicationID int64 `json:"application_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.RecruitingIntelligence.ParseResumeProfile(c.Request.Context(), &pb.ParseResumeProfileRequest{
		StaffUserId:   middleware.UserID(c),
		ResumeId:      req.ResumeID,
		ApplicationId: req.ApplicationID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *RecruitingIntelligenceHandler) EvaluateCandidateMatch(c *gin.Context) {
	applicationID, err := parseApplicationIDParam(c)
	if err != nil {
		base.BadRequest(c, "投递记录 ID 不合法")
		return
	}
	var req struct {
		AgentRunID uint64 `json:"agent_run_id"`
	}
	_ = c.ShouldBindJSON(&req)
	resp, err := h.clients.RecruitingIntelligence.EvaluateCandidateMatch(c.Request.Context(), &pb.EvaluateCandidateMatchRequest{
		StaffUserId:   middleware.UserID(c),
		ApplicationId: applicationID,
		AgentRunId:    req.AgentRunID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *RecruitingIntelligenceHandler) GetCandidateMatchEvaluation(c *gin.Context) {
	applicationID, err := parseApplicationIDParam(c)
	if err != nil {
		base.BadRequest(c, "投递记录 ID 不合法")
		return
	}
	resp, err := h.clients.RecruitingIntelligence.GetCandidateMatchEvaluation(c.Request.Context(), &pb.GetCandidateMatchEvaluationRequest{
		StaffUserId:       middleware.UserID(c),
		ApplicationId:     applicationID,
		EvaluationId:      parseUint64Query(c, "evaluation_id"),
		EvaluationVersion: int32(parseInt64Query(c, "evaluation_version")),
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *RecruitingIntelligenceHandler) CompareCandidatesForJob(c *gin.Context) {
	jobID, err := parseInt64Param(c, "job_id")
	if err != nil {
		base.BadRequest(c, "岗位 ID 不合法")
		return
	}
	resp, err := h.clients.RecruitingIntelligence.CompareCandidatesForJob(c.Request.Context(), &pb.CompareCandidatesForJobRequest{
		StaffUserId: middleware.UserID(c),
		JobId:       jobID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func parseInt64Param(c *gin.Context, name string) (int64, error) {
	return strconv.ParseInt(c.Param(name), 10, 64)
}

func parseApplicationIDParam(c *gin.Context) (int64, error) {
	if value := c.Param("application_id"); value != "" {
		return strconv.ParseInt(value, 10, 64)
	}
	return parseInt64Param(c, "id")
}

func parseInt64Query(c *gin.Context, name string) int64 {
	value, _ := strconv.ParseInt(c.Query(name), 10, 64)
	return value
}

func parseUint64Query(c *gin.Context, name string) uint64 {
	value, _ := strconv.ParseUint(c.Query(name), 10, 64)
	return value
}
