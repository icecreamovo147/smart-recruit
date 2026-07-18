package hr

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	base "smart-recruit-gateway/handler"
	"smart-recruit-gateway/middleware"
	"smart-recruit-gateway/pkg/logger"
	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

type RecruitingIntelligenceHandler struct {
	clients *rpc.Clients
}

func NewRecruitingIntelligenceHandler(clients *rpc.Clients) *RecruitingIntelligenceHandler {
	return &RecruitingIntelligenceHandler{clients: clients}
}

func (h *RecruitingIntelligenceHandler) GetResumeProfileByApplication(c *gin.Context) {
	started := time.Now()
	userID := middleware.UserID(c)
	requestID := base.RequestID(c)
	applicationID, err := parseApplicationIDParam(c)
	logger.L().Info("[HR][HTTP] GetResumeProfileByApplication received",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("application_id", applicationID))
	if err != nil {
		logger.L().Warn("[HR][HTTP] GetResumeProfileByApplication bad request",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.BadRequest(c, "投递记录 ID 不合法")
		return
	}
	logger.L().Info("[HR][HTTP] GetResumeProfileByApplication grpc call started",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("application_id", applicationID))
	resp, err := h.clients.RecruitingIntelligence.GetResumeProfile(c.Request.Context(), &pb.GetResumeProfileRequest{
		StaffUserId:   userID,
		ApplicationId: applicationID,
	})
	if err != nil {
		logger.L().Error("[HR][HTTP] GetResumeProfileByApplication grpc error",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.Internal(c, err)
		return
	}
	logger.L().Info("[HR][HTTP] GetResumeProfileByApplication finished",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("application_id", applicationID),
		zap.Int32("business_code", resp.GetCode()),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
	base.ProtoResponse(c, resp)
}

func (h *RecruitingIntelligenceHandler) GetResumeProfile(c *gin.Context) {
	started := time.Now()
	userID := middleware.UserID(c)
	requestID := base.RequestID(c)
	req := &pb.GetResumeProfileRequest{
		StaffUserId:   userID,
		ResumeId:      parseInt64Query(c, "resume_id"),
		ApplicationId: parseInt64Query(c, "application_id"),
		ProfileId:     parseUint64Query(c, "profile_id"),
	}
	logger.L().Info("[HR][HTTP] GetResumeProfile received",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("resume_id", req.GetResumeId()),
		zap.Int64("application_id", req.GetApplicationId()),
		zap.Uint64("profile_id", req.GetProfileId()))
	logger.L().Info("[HR][HTTP] GetResumeProfile grpc call started",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("resume_id", req.GetResumeId()),
		zap.Int64("application_id", req.GetApplicationId()),
		zap.Uint64("profile_id", req.GetProfileId()))
	resp, err := h.clients.RecruitingIntelligence.GetResumeProfile(c.Request.Context(), req)
	if err != nil {
		logger.L().Error("[HR][HTTP] GetResumeProfile grpc error",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.Internal(c, err)
		return
	}
	logger.L().Info("[HR][HTTP] GetResumeProfile finished",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int32("business_code", resp.GetCode()),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
	base.ProtoResponse(c, resp)
}

func (h *RecruitingIntelligenceHandler) ParseResumeProfile(c *gin.Context) {
	started := time.Now()
	userID := middleware.UserID(c)
	requestID := base.RequestID(c)
	var req struct {
		ResumeID      int64 `json:"resume_id"`
		ApplicationID int64 `json:"application_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.L().Warn("[HR][HTTP] ParseResumeProfile bad request",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.BadRequest(c, "请求参数错误")
		return
	}
	logger.L().Info("[HR][HTTP] ParseResumeProfile received",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("application_id", req.ApplicationID),
		zap.Int64("resume_id", req.ResumeID))
	logger.L().Info("[HR][HTTP] ParseResumeProfile grpc call started",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("application_id", req.ApplicationID),
		zap.Int64("resume_id", req.ResumeID))
	resp, err := h.clients.RecruitingIntelligence.ParseResumeProfile(c.Request.Context(), &pb.ParseResumeProfileRequest{
		StaffUserId:   userID,
		ResumeId:      req.ResumeID,
		ApplicationId: req.ApplicationID,
	})
	if err != nil {
		logger.L().Error("[HR][HTTP] ParseResumeProfile grpc error",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.Internal(c, err)
		return
	}
	logger.L().Info("[HR][HTTP] ParseResumeProfile finished",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int32("business_code", resp.GetCode()),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
	base.ProtoResponse(c, resp)
}

func (h *RecruitingIntelligenceHandler) EvaluateCandidateMatch(c *gin.Context) {
	started := time.Now()
	userID := middleware.UserID(c)
	requestID := base.RequestID(c)
	applicationID, err := parseApplicationIDParam(c)
	if err != nil {
		logger.L().Warn("[HR][HTTP] EvaluateCandidateMatch bad request",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.BadRequest(c, "投递记录 ID 不合法")
		return
	}
	var body struct {
		AgentRunID uint64 `json:"agent_run_id"`
	}
	_ = c.ShouldBindJSON(&body)
	logger.L().Info("[HR][HTTP] EvaluateCandidateMatch received",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("application_id", applicationID),
		zap.Uint64("agent_run_id", body.AgentRunID))
	logger.L().Info("[HR][HTTP] EvaluateCandidateMatch grpc call started",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("application_id", applicationID),
		zap.Uint64("agent_run_id", body.AgentRunID))
	resp, err := h.clients.RecruitingIntelligence.EvaluateCandidateMatch(c.Request.Context(), &pb.EvaluateCandidateMatchRequest{
		StaffUserId:   userID,
		ApplicationId: applicationID,
		AgentRunId:    body.AgentRunID,
	})
	if err != nil {
		logger.L().Error("[HR][HTTP] EvaluateCandidateMatch grpc error",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.Internal(c, err)
		return
	}
	logger.L().Info("[HR][HTTP] EvaluateCandidateMatch finished",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int32("business_code", resp.GetCode()),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
	base.ProtoResponse(c, resp)
}

func (h *RecruitingIntelligenceHandler) GetCandidateMatchEvaluation(c *gin.Context) {
	started := time.Now()
	userID := middleware.UserID(c)
	requestID := base.RequestID(c)
	applicationID, err := parseApplicationIDParam(c)
	if err != nil {
		logger.L().Warn("[HR][HTTP] GetCandidateMatchEvaluation bad request",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.BadRequest(c, "投递记录 ID 不合法")
		return
	}
	evalID := parseUint64Query(c, "evaluation_id")
	evalVersion := int32(parseInt64Query(c, "evaluation_version"))
	logger.L().Info("[HR][HTTP] GetCandidateMatchEvaluation received",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("application_id", applicationID),
		zap.Uint64("evaluation_id", evalID),
		zap.Int32("evaluation_version", evalVersion))
	logger.L().Info("[HR][HTTP] GetCandidateMatchEvaluation grpc call started",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("application_id", applicationID),
		zap.Uint64("evaluation_id", evalID),
		zap.Int32("evaluation_version", evalVersion))
	resp, err := h.clients.RecruitingIntelligence.GetCandidateMatchEvaluation(c.Request.Context(), &pb.GetCandidateMatchEvaluationRequest{
		StaffUserId:       userID,
		ApplicationId:     applicationID,
		EvaluationId:      evalID,
		EvaluationVersion: evalVersion,
	})
	if err != nil {
		logger.L().Error("[HR][HTTP] GetCandidateMatchEvaluation grpc error",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.Internal(c, err)
		return
	}
	logger.L().Info("[HR][HTTP] GetCandidateMatchEvaluation finished",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int32("business_code", resp.GetCode()),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
	base.ProtoResponse(c, resp)
}

func (h *RecruitingIntelligenceHandler) CompareCandidatesForJob(c *gin.Context) {
	started := time.Now()
	userID := middleware.UserID(c)
	requestID := base.RequestID(c)
	jobID, err := parseInt64Param(c, "job_id")
	if err != nil {
		logger.L().Warn("[HR][HTTP] CompareCandidatesForJob bad request",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.BadRequest(c, "岗位 ID 不合法")
		return
	}
	logger.L().Info("[HR][HTTP] CompareCandidatesForJob received",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("job_id", jobID))
	logger.L().Info("[HR][HTTP] CompareCandidatesForJob grpc call started",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int64("job_id", jobID))
	resp, err := h.clients.RecruitingIntelligence.CompareCandidatesForJob(c.Request.Context(), &pb.CompareCandidatesForJobRequest{
		StaffUserId: userID,
		JobId:       jobID,
	})
	if err != nil {
		logger.L().Error("[HR][HTTP] CompareCandidatesForJob grpc error",
			zap.String("request_id", requestID),
			zap.Int64("user_id", userID), zap.Error(err))
		base.Internal(c, err)
		return
	}
	logger.L().Info("[HR][HTTP] CompareCandidatesForJob finished",
		zap.String("request_id", requestID),
		zap.Int64("user_id", userID),
		zap.Int32("business_code", resp.GetCode()),
		zap.Int("candidate_count", len(resp.GetCandidates())),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
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
