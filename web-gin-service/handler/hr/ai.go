package hr

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

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

// CreateAgentRun creates a durable resumable HR Agent run (or returns an
// idempotent existing run for the same client_request_id). Lifecycle state is
// owned by logic-grpc; the gateway only forwards the command.
func (h *AIHandler) CreateAgentRun(c *gin.Context) {
	var req struct {
		SessionID                    base.FlexInt64 `json:"session_id" binding:"required"`
		ClientRequestID              string         `json:"client_request_id"`
		Message                      string         `json:"message"`
		ActionType                   string         `json:"action_type"`
		ActionPayloadJSON            string         `json:"action_payload_json"`
		ApplicationID                base.FlexInt64 `json:"application_id"`
		ModelID                      base.FlexInt64 `json:"model_id"`
		SkillCapabilityKeys          []string       `json:"skill_capability_keys"`
		AgentSkillIDs                []int64        `json:"agent_skill_ids"`
		AgentSkillSelectionConfirmed bool           `json:"agent_skill_selection_confirmed"`
		AgentSkillSelectionMessageID base.FlexInt64 `json:"agent_skill_selection_message_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数不合法")
		return
	}
	if int64(req.SessionID) <= 0 {
		base.BadRequest(c, "会话 ID 不合法")
		return
	}
	resp, err := h.clients.AI.CreateAgentRun(c.Request.Context(), &pb.CreateAgentRunRequest{
		HrId:                         middleware.UserID(c),
		SessionId:                    int64(req.SessionID),
		ClientRequestId:              req.ClientRequestID,
		Message:                      req.Message,
		ActionType:                   req.ActionType,
		ActionPayloadJson:            req.ActionPayloadJSON,
		ApplicationId:                int64(req.ApplicationID),
		ModelId:                      int64(req.ModelID),
		SkillCapabilityKeys:          req.SkillCapabilityKeys,
		AgentSkillIds:                req.AgentSkillIDs,
		AgentSkillSelectionConfirmed: req.AgentSkillSelectionConfirmed,
		AgentSkillSelectionMessageId: int64(req.AgentSkillSelectionMessageID),
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"run":               agentRunSnapshotPayload(resp.GetRun()),
		"idempotent_replay": resp.GetIdempotentReplay(),
	})
}

// GetAgentRun returns the latest durable run snapshot for a run id.
func (h *AIHandler) GetAgentRun(c *gin.Context) {
	runID, err := parsePositiveInt64Param(c, "run_id", "运行 ID 不合法")
	if err != nil {
		return
	}
	resp, err := h.clients.AI.GetAgentRun(c.Request.Context(), &pb.GetAgentRunRequest{
		HrId:  middleware.UserID(c),
		RunId: runID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"run": agentRunSnapshotPayload(resp.GetRun())})
}

// GetActiveAgentRun returns the active durable run for a chat session, if any.
func (h *AIHandler) GetActiveAgentRun(c *gin.Context) {
	sessionID, err := parsePositiveInt64Param(c, "session_id", "会话 ID 不合法")
	if err != nil {
		return
	}
	resp, err := h.clients.AI.GetActiveAgentRun(c.Request.Context(), &pb.GetActiveAgentRunRequest{
		HrId:      middleware.UserID(c),
		SessionId: sessionID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"run":            agentRunSnapshotPayload(resp.GetRun()),
		"has_active_run": resp.GetHasActiveRun(),
	})
}

// SubscribeAgentRunEvents streams durable run events over SSE.
//
// Replay cursor resolution (header wins when present and valid):
//  1. Last-Event-ID header
//  2. after_seq query parameter
//
// HTTP client disconnect cancels only this subscription (request context →
// gRPC stream). It must never call CancelAgentRun.
func (h *AIHandler) SubscribeAgentRunEvents(c *gin.Context) {
	runID, err := parsePositiveInt64Param(c, "run_id", "运行 ID 不合法")
	if err != nil {
		return
	}
	afterSeq, ok := resolveAfterSeq(c)
	if !ok {
		base.BadRequest(c, "after_seq 不合法")
		return
	}

	ctx := c.Request.Context()
	stream, err := h.clients.AI.SubscribeAgentRunEvents(ctx, &pb.SubscribeAgentRunEventsRequest{
		HrId:     middleware.UserID(c),
		RunId:    runID,
		AfterSeq: afterSeq,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	type recvResult struct {
		event *pb.AgentRunEvent
		err   error
	}
	recvCh := make(chan recvResult, 1)
	go func() {
		defer close(recvCh)
		for {
			event, recvErr := stream.Recv()
			select {
			case recvCh <- recvResult{event: event, err: recvErr}:
			case <-ctx.Done():
				return
			}
			if recvErr != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			// Disconnect ends only the SSE/gRPC subscription; do not cancel the run.
			return
		case result, ok := <-recvCh:
			if !ok {
				return
			}
			if result.err == io.EOF {
				return
			}
			if result.err != nil {
				if ctx.Err() != nil {
					return
				}
				info := base.PublicError(result.err)
				payload := fmt.Sprintf("data: %s\n\n", mustMarshalHR(gin.H{
					"code":       info.Code,
					"msg":        info.Msg,
					"done":       true,
					"request_id": base.RequestID(c),
				}))
				if n, writeErr := c.Writer.Write([]byte(payload)); writeErr != nil || n == 0 {
					return
				}
				_ = base.FlushSSE(c.Writer)
				return
			}
			eventPayload := agentRunEventPayload(result.event)
			eventPayload["request_id"] = base.RequestID(c)
			line := fmt.Sprintf("id: %d\ndata: %s\n\n", result.event.GetSeq(), mustMarshalHR(eventPayload))
			if n, writeErr := c.Writer.Write([]byte(line)); writeErr != nil || n == 0 {
				return
			}
			if !base.FlushSSE(c.Writer) {
				return
			}
		}
	}
}

// CancelAgentRun forwards an explicit cancel command. SSE disconnect must not
// invoke this handler.
func (h *AIHandler) CancelAgentRun(c *gin.Context) {
	runID, err := parsePositiveInt64Param(c, "run_id", "运行 ID 不合法")
	if err != nil {
		return
	}
	var req struct {
		ClientRequestID string `json:"client_request_id"`
	}
	// Body is optional; empty body is treated as no client_request_id.
	_ = c.ShouldBindJSON(&req)
	resp, err := h.clients.AI.CancelAgentRun(c.Request.Context(), &pb.CancelAgentRunRequest{
		HrId:            middleware.UserID(c),
		RunId:           runID,
		ClientRequestId: req.ClientRequestID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"run": agentRunSnapshotPayload(resp.GetRun())})
}

// ConfirmAgentRun resumes a run waiting for skill/confirmation input.
func (h *AIHandler) ConfirmAgentRun(c *gin.Context) {
	runID, err := parsePositiveInt64Param(c, "run_id", "运行 ID 不合法")
	if err != nil {
		return
	}
	var req struct {
		ClientRequestID              string         `json:"client_request_id"`
		AgentSkillIDs                []int64        `json:"agent_skill_ids"`
		AgentSkillSelectionConfirmed bool           `json:"agent_skill_selection_confirmed"`
		AgentSkillSelectionMessageID base.FlexInt64 `json:"agent_skill_selection_message_id"`
		ConfirmationPayloadJSON      string         `json:"confirmation_payload_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body defaults for confirm with only path run_id.
		req = struct {
			ClientRequestID              string         `json:"client_request_id"`
			AgentSkillIDs                []int64        `json:"agent_skill_ids"`
			AgentSkillSelectionConfirmed bool           `json:"agent_skill_selection_confirmed"`
			AgentSkillSelectionMessageID base.FlexInt64 `json:"agent_skill_selection_message_id"`
			ConfirmationPayloadJSON      string         `json:"confirmation_payload_json"`
		}{}
	}
	resp, err := h.clients.AI.ConfirmAgentRun(c.Request.Context(), &pb.ConfirmAgentRunRequest{
		HrId:                         middleware.UserID(c),
		RunId:                        runID,
		ClientRequestId:              req.ClientRequestID,
		AgentSkillIds:                req.AgentSkillIDs,
		AgentSkillSelectionConfirmed: req.AgentSkillSelectionConfirmed,
		AgentSkillSelectionMessageId: int64(req.AgentSkillSelectionMessageID),
		ConfirmationPayloadJson:      req.ConfirmationPayloadJSON,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"run": agentRunSnapshotPayload(resp.GetRun())})
}

// resolveAfterSeq prefers Last-Event-ID over the after_seq query parameter.
// Returns ok=false when a provided value is present but not a valid integer.
func resolveAfterSeq(c *gin.Context) (int64, bool) {
	if lastEventID := strings.TrimSpace(c.GetHeader("Last-Event-ID")); lastEventID != "" {
		seq, err := strconv.ParseInt(lastEventID, 10, 64)
		if err != nil || seq < 0 {
			return 0, false
		}
		return seq, true
	}
	raw := strings.TrimSpace(c.Query("after_seq"))
	if raw == "" {
		return 0, true
	}
	seq, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || seq < 0 {
		return 0, false
	}
	return seq, true
}

func parsePositiveInt64Param(c *gin.Context, name, badMsg string) (int64, error) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		base.BadRequest(c, badMsg)
		if err == nil {
			err = strconv.ErrSyntax
		}
		return 0, err
	}
	return id, nil
}

func agentRunSnapshotPayload(run *pb.AgentRunSnapshot) map[string]any {
	if run == nil {
		return nil
	}
	return gin.H{
		"run_id":               run.GetRunId(),
		"session_id":           run.GetSessionId(),
		"hr_id":                run.GetHrId(),
		"client_request_id":    run.GetClientRequestId(),
		"message_id":           run.GetMessageId(),
		"history_id":           run.GetHistoryId(),
		"status":               run.GetStatus(),
		"assistant_text":       run.GetAssistantText(),
		"process_text":         run.GetProcessText(),
		"result_metadata":      agentRunResultMetadataPayload(run.GetResultMetadata()),
		"confirmation_request": agentRunConfirmationPayload(run.GetConfirmationRequest()),
		"option_context_json":  run.GetOptionContextJson(),
		"last_event_seq":       run.GetLastEventSeq(),
		"error_type":           run.GetErrorType(),
		"error_message":        run.GetErrorMessage(),
		"model_id":             run.GetModelId(),
		"model_name":           run.GetModelName(),
		"agent_type":           run.GetAgentType(),
		"agent_id":             run.GetAgentId(),
		"agent_name":           run.GetAgentName(),
		"started_at":           run.GetStartedAt(),
		"completed_at":         run.GetCompletedAt(),
		"cancel_requested_at":  run.GetCancelRequestedAt(),
		"canceled_at":          run.GetCanceledAt(),
		"created_at":           run.GetCreatedAt(),
		"updated_at":           run.GetUpdatedAt(),
	}
}

func agentRunEventPayload(event *pb.AgentRunEvent) gin.H {
	if event == nil {
		return gin.H{}
	}
	return gin.H{
		"run_id":          event.GetRunId(),
		"seq":             event.GetSeq(),
		"event_type":      event.GetEventType(),
		"payload_json":    event.GetPayloadJson(),
		"status":          event.GetStatus(),
		"delta":           event.GetDelta(),
		"snapshot_text":   event.GetSnapshotText(),
		"result_metadata": agentRunResultMetadataPayload(event.GetResultMetadata()),
		"confirmation":    agentRunConfirmationPayload(event.GetConfirmation()),
		"tool_name":       event.GetToolName(),
		"error_type":      event.GetErrorType(),
		"error_message":   event.GetErrorMessage(),
		"created_at":      event.GetCreatedAt(),
	}
}

func agentRunResultMetadataPayload(meta *pb.AgentRunResultMetadata) map[string]any {
	if meta == nil {
		return nil
	}
	var contextUsage map[string]any
	if cu := meta.GetContextUsage(); cu != nil {
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
	return gin.H{
		"action":              meta.GetAction(),
		"application_id":      meta.GetApplicationId(),
		"action_status":       meta.GetActionStatus(),
		"candidate_name":      meta.GetCandidateName(),
		"job_title":           meta.GetJobTitle(),
		"status":              meta.GetStatus(),
		"candidate_options":   meta.GetCandidateOptions(),
		"suggested_questions": meta.GetSuggestedQuestions(),
		"context_usage":       contextUsage,
		"error_type":          meta.GetErrorType(),
		"error_message":       meta.GetErrorMessage(),
		"raw_json":            meta.GetRawJson(),
	}
}

func agentRunConfirmationPayload(confirmation *pb.AgentRunConfirmationPayload) map[string]any {
	if confirmation == nil {
		return nil
	}
	candidates := make([]gin.H, 0, len(confirmation.GetCandidates()))
	for _, candidate := range confirmation.GetCandidates() {
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
		"required":                    confirmation.GetRequired(),
		"reason":                      confirmation.GetReason(),
		"candidates":                  candidates,
		"recommended_agent_skill_ids": confirmation.GetRecommendedAgentSkillIds(),
		"user_message_id":             confirmation.GetUserMessageId(),
		"raw_json":                    confirmation.GetRawJson(),
	}
}

func mustMarshalHR(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
