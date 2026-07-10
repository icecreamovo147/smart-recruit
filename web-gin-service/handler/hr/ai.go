package hr

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type AIHandler struct {
	clients *rpc.Clients
}

func NewAIHandler(clients *rpc.Clients) *AIHandler {
	return &AIHandler{clients: clients}
}

func (h *AIHandler) Chat(c *gin.Context) {
	var req struct {
		Message                      string         `json:"message" binding:"required"`
		ApplicationID                base.FlexInt64 `json:"application_id"`
		SessionID                    base.FlexInt64 `json:"session_id"`
		ModelID                      base.FlexInt64 `json:"model_id"`
		SkillCapabilityKeys          []string       `json:"skill_capability_keys"`
		AgentSkillIDs                []int64        `json:"agent_skill_ids"`
		AgentSkillSelectionConfirmed bool           `json:"agent_skill_selection_confirmed"`
		AgentSkillSelectionMessageID base.FlexInt64 `json:"agent_skill_selection_message_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "消息不能为空")
		return
	}
	resp, err := h.clients.AI.Chat(c.Request.Context(), &pb.ChatRequest{HrId: middleware.UserID(c), Message: req.Message, ApplicationId: int64(req.ApplicationID), SessionId: int64(req.SessionID), ModelId: int64(req.ModelID), SkillCapabilityKeys: req.SkillCapabilityKeys, AgentSkillIds: req.AgentSkillIDs, AgentSkillSelectionConfirmed: req.AgentSkillSelectionConfirmed, AgentSkillSelectionMessageId: int64(req.AgentSkillSelectionMessageID)})
	if err != nil {
		base.Internal(c, err)
		return
	}
	var contextUsage map[string]any
	if cu := resp.GetContextUsage(); cu != nil {
		var breakdown map[string]any
		if bd := cu.GetBreakdown(); bd != nil {
			breakdown = gin.H{
				"system_prompt_tokens":   bd.GetSystemPromptTokens(),
				"recent_message_tokens":  bd.GetRecentMessageTokens(),
				"summary_tokens":         bd.GetSummaryTokens(),
				"memory_tokens":          bd.GetMemoryTokens(),
				"current_message_tokens": bd.GetCurrentMessageTokens(),
				"skill_tokens":           bd.GetSkillTokens(),
				"tool_result_tokens":     bd.GetToolResultTokens(),
			}
		}
		contextUsage = gin.H{
			"model_id":                   cu.GetModelId(),
			"model_name":                 cu.GetModelName(),
			"context_window_tokens":      cu.GetContextWindowTokens(),
			"max_output_tokens":          cu.GetMaxOutputTokens(),
			"prompt_tokens_estimated":    cu.GetPromptTokensEstimated(),
			"prompt_tokens_actual":       cu.GetPromptTokensActual(),
			"completion_tokens_actual":   cu.GetCompletionTokensActual(),
			"total_tokens_actual":        cu.GetTotalTokensActual(),
			"remaining_tokens_estimated": cu.GetRemainingTokensEstimated(),
			"usage_ratio":                cu.GetUsageRatio(),
			"estimated":                  cu.GetEstimated(),
			"source":                     cu.GetSource(),
			"stage":                      cu.GetStage(),
			"breakdown":                  breakdown,
		}
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"reply":          resp.Reply,
		"created_at":     resp.CreatedAt,
		"action":         resp.Action,
		"application_id": resp.ApplicationId,
		"action_status":  resp.ActionStatus,
		"candidate_name": resp.CandidateName,
		"job_title":      resp.JobTitle,
		"status":         resp.Status,
		"session_id":     resp.SessionId,
		"context_usage":  contextUsage,
	})
}

func (h *AIHandler) ChatStream(c *gin.Context) {
	var req struct {
		Message                      string         `json:"message" binding:"required"`
		ApplicationID                base.FlexInt64 `json:"application_id"`
		SessionID                    base.FlexInt64 `json:"session_id"`
		ModelID                      base.FlexInt64 `json:"model_id"`
		SkillCapabilityKeys          []string       `json:"skill_capability_keys"`
		AgentSkillIDs                []int64        `json:"agent_skill_ids"`
		AgentSkillSelectionConfirmed bool           `json:"agent_skill_selection_confirmed"`
		AgentSkillSelectionMessageID base.FlexInt64 `json:"agent_skill_selection_message_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "消息不能为空")
		return
	}
	ctx := c.Request.Context()
	stream, err := h.clients.AI.ChatStream(ctx, &pb.ChatRequest{HrId: middleware.UserID(c), Message: req.Message, ApplicationId: int64(req.ApplicationID), SessionId: int64(req.SessionID), ModelId: int64(req.ModelID), SkillCapabilityKeys: req.SkillCapabilityKeys, AgentSkillIds: req.AgentSkillIDs, AgentSkillSelectionConfirmed: req.AgentSkillSelectionConfirmed, AgentSkillSelectionMessageId: int64(req.AgentSkillSelectionMessageID)})
	if err != nil {
		base.Internal(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	type recvResult struct {
		chunk *pb.ChatStreamResponse
		err   error
	}
	recvCh := make(chan recvResult, 1)
	cancelCtx := c.Request.Context()
	go func() {
		defer close(recvCh)
		for {
			chunk, err := stream.Recv()
			select {
			case recvCh <- recvResult{chunk: chunk, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-cancelCtx.Done():
			return
		case result, ok := <-recvCh:
			if !ok {
				return
			}
			if result.err == io.EOF {
				return
			}
			if result.err != nil {
				if c.Request.Context().Err() != nil {
					return
				}
				info := base.PublicError(result.err)
				payload := fmt.Sprintf("event: message\ndata: %s\n\n", mustMarshalHR(gin.H{"code": info.Code, "msg": info.Msg, "done": true, "request_id": base.RequestID(c)}))
				if n, err := c.Writer.Write([]byte(payload)); err != nil || n == 0 {
					return
				}
				if !base.FlushSSE(c.Writer) {
					return
				}
				return
			}
			var contextUsage map[string]any
			if cu := result.chunk.GetContextUsage(); cu != nil {
				var breakdown map[string]any
				if bd := cu.GetBreakdown(); bd != nil {
					breakdown = gin.H{
						"system_prompt_tokens":   bd.GetSystemPromptTokens(),
						"recent_message_tokens":  bd.GetRecentMessageTokens(),
						"summary_tokens":         bd.GetSummaryTokens(),
						"memory_tokens":          bd.GetMemoryTokens(),
						"current_message_tokens": bd.GetCurrentMessageTokens(),
						"skill_tokens":           bd.GetSkillTokens(),
						"tool_result_tokens":     bd.GetToolResultTokens(),
					}
				}
				contextUsage = gin.H{
					"model_id":                   cu.GetModelId(),
					"model_name":                 cu.GetModelName(),
					"context_window_tokens":      cu.GetContextWindowTokens(),
					"max_output_tokens":          cu.GetMaxOutputTokens(),
					"prompt_tokens_estimated":    cu.GetPromptTokensEstimated(),
					"prompt_tokens_actual":       cu.GetPromptTokensActual(),
					"completion_tokens_actual":   cu.GetCompletionTokensActual(),
					"total_tokens_actual":        cu.GetTotalTokensActual(),
					"remaining_tokens_estimated": cu.GetRemainingTokensEstimated(),
					"usage_ratio":                cu.GetUsageRatio(),
					"estimated":                  cu.GetEstimated(),
					"source":                     cu.GetSource(),
					"stage":                      cu.GetStage(),
					"breakdown":                  breakdown,
				}
			}
			payload := gin.H{
				"code":                  result.chunk.Code,
				"msg":                   result.chunk.Msg,
				"delta":                 result.chunk.Delta,
				"done":                  result.chunk.Done,
				"action":                result.chunk.Action,
				"application_id":        result.chunk.ApplicationId,
				"action_status":         result.chunk.ActionStatus,
				"candidate_name":        result.chunk.CandidateName,
				"job_title":             result.chunk.JobTitle,
				"status":                result.chunk.Status,
				"session_id":            result.chunk.SessionId,
				"created_at":            result.chunk.CreatedAt,
				"candidate_options":     result.chunk.CandidateOptions,
				"event_type":            result.chunk.EventType,
				"event_message":         result.chunk.EventMessage,
				"error_type":            result.chunk.ErrorType,
				"tool_name":             result.chunk.ToolName,
				"context_usage":         contextUsage,
				"agent_skill_selection": agentSkillSelectionPayload(result.chunk.GetAgentSkillSelection()),
				"request_id":            base.RequestID(c),
			}
			line := fmt.Sprintf("event: message\ndata: %s\n\n", mustMarshalHR(payload))
			if n, err := c.Writer.Write([]byte(line)); err != nil || n == 0 {
				return
			}
			if !base.FlushSSE(c.Writer) {
				return
			}
			if result.chunk.Done {
				return
			}
		}
	}
}

func agentSkillSelectionPayload(selection *pb.AgentSkillSelection) map[string]any {
	if selection == nil {
		return nil
	}
	candidates := make([]gin.H, 0, len(selection.GetCandidates()))
	for _, candidate := range selection.GetCandidates() {
		candidates = append(candidates, gin.H{
			"id":                 candidate.GetId(),
			"name":               candidate.GetName(),
			"display_name":       candidate.GetDisplayName(),
			"reason":             candidate.GetReason(),
			"score":              candidate.GetScore(),
			"priority":           candidate.GetPriority(),
			"category":           candidate.GetCategory(),
			"scenario":           candidate.GetScenario(),
			"risk_level":         candidate.GetRiskLevel(),
			"recommended":        candidate.GetRecommended(),
			"vector_score":       candidate.GetVectorScore(),
			"lexical_score":      candidate.GetLexicalScore(),
			"metadata_score":     candidate.GetMetadataScore(),
			"relevance_score":    candidate.GetRelevanceScore(),
			"business_boost":     candidate.GetBusinessBoost(),
			"final_rank_score":   candidate.GetFinalRankScore(),
			"relevance_mode":     candidate.GetRelevanceMode(),
			"pool_rank":          candidate.GetPoolRank(),
			"ranking_confidence": candidate.GetRankingConfidence(),
		})
	}
	return gin.H{
		"required":                    selection.GetRequired(),
		"reason":                      selection.GetReason(),
		"candidates":                  candidates,
		"recommended_agent_skill_ids": selection.GetRecommendedAgentSkillIds(),
		"user_message_id":             selection.GetUserMessageId(),
	}
}

func (h *AIHandler) ListSkillCapabilities(c *gin.Context) {
	resp, err := h.clients.AgentConfig.ListCapabilities(c.Request.Context(), &pb.ListCapabilitiesRequest{
		AgentType: "hr_recruiting_agent",
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	list := make([]*pb.CapabilityInfo, 0, len(resp.List))
	for _, item := range resp.List {
		if item.GetSource() == "skill" && item.GetIsAvailable() {
			list = append(list, item)
		}
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": list})
}

func (h *AIHandler) History(c *gin.Context) {
	page, pageSize := basePagination(c)
	resp, err := h.clients.AI.History(c.Request.Context(), &pb.ChatHistoryRequest{HrId: middleware.UserID(c), Page: page, PageSize: pageSize})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": resp.List})
}

func (h *AIHandler) ListSessions(c *gin.Context) {
	page, pageSize := basePagination(c)
	resp, err := h.clients.AI.ListChatSessions(c.Request.Context(), &pb.ChatSessionListRequest{HrId: middleware.UserID(c), Page: page, PageSize: pageSize})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"total": resp.Total, "list": resp.List, "model_name": resp.ModelName})
}

func (h *AIHandler) CreateSession(c *gin.Context) {
	var req struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数格式错误")
		return
	}
	resp, err := h.clients.AI.CreateChatSession(c.Request.Context(), &pb.CreateChatSessionRequest{HrId: middleware.UserID(c), Title: req.Title})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"session": resp.Session})
}

func (h *AIHandler) SessionMessages(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "会话 ID 不合法")
		return
	}
	page, pageSize := basePagination(c)
	resp, err := h.clients.AI.SessionMessages(c.Request.Context(), &pb.SessionMessagesRequest{HrId: middleware.UserID(c), SessionId: sessionID, Page: page, PageSize: pageSize})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": resp.List})
}

func (h *AIHandler) CreateApplicationAnalysisSession(c *gin.Context) {
	var req struct {
		ApplicationID base.FlexInt64 `json:"application_id" binding:"required"`
		ModelID       base.FlexInt64 `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "投递记录 ID 不能为空")
		return
	}
	resp, err := h.clients.AI.CreateApplicationAnalysisSession(c.Request.Context(), &pb.CreateApplicationAnalysisSessionRequest{HrId: middleware.UserID(c), ApplicationId: int64(req.ApplicationID), ModelId: int64(req.ModelID)})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"session": resp.Session, "messages": resp.Messages})
}

func (h *AIHandler) AnalyzeApplication(c *gin.Context) {
	var req struct {
		ApplicationID base.FlexInt64 `json:"application_id" binding:"required"`
		ModelID       base.FlexInt64 `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "投递记录 ID 不能为空")
		return
	}
	resp, err := h.clients.AI.AnalyzeApplication(c.Request.Context(), &pb.AnalyzeApplicationRequest{HrId: middleware.UserID(c), ApplicationId: int64(req.ApplicationID), ModelId: int64(req.ModelID)})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"reply":          resp.Reply,
		"candidate_name": resp.CandidateName,
		"job_title":      resp.JobTitle,
		"status":         resp.Status,
		"round_no":       resp.RoundNo,
	})
}

func (h *AIHandler) UpdateSession(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "会话 ID 不合法")
		return
	}
	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "会话名称不能为空")
		return
	}
	resp, err := h.clients.AI.UpdateSession(c.Request.Context(), &pb.UpdateSessionRequest{HrId: middleware.UserID(c), SessionId: sessionID, Title: req.Title})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *AIHandler) DeleteSession(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "会话 ID 不合法")
		return
	}
	resp, err := h.clients.AI.DeleteSession(c.Request.Context(), &pb.DeleteSessionRequest{HrId: middleware.UserID(c), SessionId: sessionID})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// GetToolTraces returns tool call traces for a given session.
// Only accessible by the HR user who owns the session.
func (h *AIHandler) GetToolTraces(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "会话 ID 不合法")
		return
	}
	resp, err := h.clients.AI.GetToolTraces(c.Request.Context(), &pb.GetToolTracesRequest{
		HrId:      middleware.UserID(c),
		SessionId: sessionID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": resp.List})
}

// GetAgentRuns returns agent runs and their steps for a given session.
// Only accessible by the HR user who owns the session.
func (h *AIHandler) GetAgentRuns(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "会话 ID 不合法")
		return
	}
	resp, err := h.clients.AI.GetAgentRuns(c.Request.Context(), &pb.GetAgentRunsRequest{
		HrId:      middleware.UserID(c),
		SessionId: sessionID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": resp.List})
}

func mustMarshalHR(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
