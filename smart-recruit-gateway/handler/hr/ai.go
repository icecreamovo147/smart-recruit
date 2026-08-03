package hr

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	base "smart-recruit-gateway/handler"
	"smart-recruit-gateway/middleware"
	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

type AIHandler struct {
	clients *rpc.Clients
}

type agentSkillSelectionCandidateHTTP struct {
	SkillID             int64   `json:"skill_id"`
	VersionID           int64   `json:"version_id"`
	Version             string  `json:"version"`
	CompiledHash        string  `json:"compiled_hash"`
	Name                string  `json:"name"`
	DisplayName         string  `json:"display_name"`
	Reason              string  `json:"reason"`
	Score               int32   `json:"score"`
	Priority            int32   `json:"priority"`
	Category            string  `json:"category"`
	Scenario            string  `json:"scenario"`
	CompositionRole     string  `json:"composition_role"`
	Risk                string  `json:"risk"`
	ActivationPolicy    string  `json:"activation_policy"`
	CoreEstimatedTokens int32   `json:"core_estimated_tokens"`
	Recommended         bool    `json:"recommended"`
	VectorScore         float64 `json:"vector_score"`
	LexicalScore        float64 `json:"lexical_score"`
	MetadataScore       float64 `json:"metadata_score"`
	RelevanceScore      float64 `json:"relevance_score"`
	BusinessBoost       float64 `json:"business_boost"`
	FinalRankScore      float64 `json:"final_rank_score"`
	RelevanceMode       string  `json:"relevance_mode"`
	PoolRank            int32   `json:"pool_rank"`
	RankingConfidence   string  `json:"ranking_confidence"`
}

type agentSkillSelectionHTTP struct {
	Required                        bool                               `json:"required"`
	Reason                          string                             `json:"reason"`
	Candidates                      []agentSkillSelectionCandidateHTTP `json:"candidates"`
	RecommendedAgentSkillVersionIDs []int64                            `json:"recommended_agent_skill_version_ids"`
	UserMessageID                   int64                              `json:"user_message_id"`
}

type agentRunConfirmationHTTP struct {
	Required                        bool                               `json:"required"`
	Reason                          string                             `json:"reason"`
	Candidates                      []agentSkillSelectionCandidateHTTP `json:"candidates"`
	RawJSON                         string                             `json:"raw_json"`
	AgentSkillConfirmationID        string                             `json:"agent_skill_confirmation_id"`
	RecommendedAgentSkillVersionIDs []int64                            `json:"recommended_agent_skill_version_ids"`
	AgentSkillUserMessageID         int64                              `json:"agent_skill_user_message_id"`
	AgentSkillConfirmationExpiresAt string                             `json:"agent_skill_confirmation_expires_at"`
}

type agentSkillSectionRuntimeEvidenceHTTP struct {
	SectionID       int64   `json:"section_id"`
	SectionKey      string  `json:"section_key"`
	ContentHash     string  `json:"content_hash"`
	EstimatedTokens int32   `json:"estimated_tokens"`
	FinalRankScore  float64 `json:"final_rank_score"`
	Included        bool    `json:"included"`
	DecisionReason  string  `json:"decision_reason"`
}

type agentSkillRuntimeEvidenceHTTP struct {
	SkillID             int64                                  `json:"skill_id"`
	VersionID           int64                                  `json:"version_id"`
	Version             string                                 `json:"version"`
	CompiledHash        string                                 `json:"compiled_hash"`
	SkillName           string                                 `json:"skill_name"`
	DisplayName         string                                 `json:"display_name"`
	CompositionRole     string                                 `json:"composition_role"`
	Risk                string                                 `json:"risk"`
	ActivationPolicy    string                                 `json:"activation_policy"`
	SelectionMode       string                                 `json:"selection_mode"`
	RelevanceMode       string                                 `json:"relevance_mode"`
	CoreEstimatedTokens int32                                  `json:"core_estimated_tokens"`
	LoadedTokens        int32                                  `json:"loaded_tokens"`
	Sections            []agentSkillSectionRuntimeEvidenceHTTP `json:"sections"`
	Included            bool                                   `json:"included"`
	DecisionReason      string                                 `json:"decision_reason"`
	VectorScore         float64                                `json:"vector_score"`
	LexicalScore        float64                                `json:"lexical_score"`
	MetadataScore       float64                                `json:"metadata_score"`
	RelevanceScore      float64                                `json:"relevance_score"`
	BusinessBoost       float64                                `json:"business_boost"`
	FinalRankScore      float64                                `json:"final_rank_score"`
}

type chatMessageHTTP struct {
	Role                      string                          `json:"role"`
	Content                   string                          `json:"content"`
	CreatedAt                 string                          `json:"created_at"`
	ModelID                   int64                           `json:"model_id"`
	ModelName                 string                          `json:"model_name"`
	AgentSkillNames           []string                        `json:"agent_skill_names"`
	ProcessContent            string                          `json:"process_content"`
	ContextUsage              map[string]any                  `json:"context_usage"`
	AgentSkillVersionIDs      []int64                         `json:"agent_skill_version_ids"`
	AgentSkillRuntimeEvidence []agentSkillRuntimeEvidenceHTTP `json:"agent_skill_runtime_evidence"`
}

func NewAIHandler(clients *rpc.Clients) *AIHandler {
	return &AIHandler{clients: clients}
}

func (h *AIHandler) Chat(c *gin.Context) {
	var req struct {
		Message              string         `json:"message" binding:"required"`
		ApplicationID        base.FlexInt64 `json:"application_id"`
		SessionID            base.FlexInt64 `json:"session_id"`
		ModelID              base.FlexInt64 `json:"model_id"`
		CapabilityKeys       []string       `json:"capability_keys"`
		AgentSkillVersionIDs []int64        `json:"agent_skill_version_ids"`
	}
	if err := bindStrictJSON(c, &req); err != nil || strings.TrimSpace(req.Message) == "" {
		base.BadRequest(c, "消息不能为空")
		return
	}
	if err := validateAgentSkillVersionIDs(req.AgentSkillVersionIDs); err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	resp, err := h.clients.AI.Chat(c.Request.Context(), &pb.ChatRequest{
		HrId:                 middleware.UserID(c),
		Message:              req.Message,
		ApplicationId:        int64(req.ApplicationID),
		SessionId:            int64(req.SessionID),
		ModelId:              int64(req.ModelID),
		CapabilityKeys:       req.CapabilityKeys,
		AgentSkillVersionIds: req.AgentSkillVersionIDs,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	contextUsage := mapHRContextUsage(resp.GetContextUsage())
	base.From(c, resp.Code, resp.Msg, gin.H{
		"reply":               resp.Reply,
		"created_at":          resp.CreatedAt,
		"action":              resp.Action,
		"application_id":      resp.ApplicationId,
		"action_status":       resp.ActionStatus,
		"candidate_name":      resp.CandidateName,
		"job_title":           resp.JobTitle,
		"status":              resp.Status,
		"session_id":          resp.SessionId,
		"context_usage":       contextUsage,
		"suggested_questions": nonNilStringList(resp.GetSuggestedQuestions()),
		"agent_skill_runtime_evidence": agentSkillRuntimeEvidenceListPayload(
			resp.GetAgentSkillRuntimeEvidence(),
		),
	})
}

func (h *AIHandler) ChatStream(c *gin.Context) {
	var req struct {
		Message              string         `json:"message" binding:"required"`
		ApplicationID        base.FlexInt64 `json:"application_id"`
		SessionID            base.FlexInt64 `json:"session_id"`
		ModelID              base.FlexInt64 `json:"model_id"`
		CapabilityKeys       []string       `json:"capability_keys"`
		AgentSkillVersionIDs []int64        `json:"agent_skill_version_ids"`
	}
	if err := bindStrictJSON(c, &req); err != nil || strings.TrimSpace(req.Message) == "" {
		base.BadRequest(c, "消息不能为空")
		return
	}
	if err := validateAgentSkillVersionIDs(req.AgentSkillVersionIDs); err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	stream, err := h.clients.AI.ChatStream(ctx, &pb.ChatRequest{
		HrId:                 middleware.UserID(c),
		Message:              req.Message,
		ApplicationId:        int64(req.ApplicationID),
		SessionId:            int64(req.SessionID),
		ModelId:              int64(req.ModelID),
		CapabilityKeys:       req.CapabilityKeys,
		AgentSkillVersionIds: req.AgentSkillVersionIDs,
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
				messageKey, message := base.LocalizedMessage(info.Code, info.Msg)
				payload := fmt.Sprintf("event: message\ndata: %s\n\n", mustMarshalHR(gin.H{
					"code":        info.Code,
					"message_key": messageKey,
					"msg":         message,
					"done":        true,
					"request_id":  base.RequestID(c),
				}))
				if n, err := c.Writer.Write([]byte(payload)); err != nil || n == 0 {
					return
				}
				if !base.FlushSSE(c.Writer) {
					return
				}
				return
			}
			contextUsage := mapHRContextUsage(result.chunk.GetContextUsage())
			messageKey, message := base.LocalizedMessage(result.chunk.Code, result.chunk.Msg)
			eventMessageKey, eventMessage := base.LocalizedSystemMessage(result.chunk.EventMessage)
			payload := gin.H{
				"code":                  result.chunk.Code,
				"message_key":           messageKey,
				"msg":                   message,
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
				"suggested_questions":   nonNilStringList(result.chunk.GetSuggestedQuestions()),
				"event_type":            result.chunk.EventType,
				"event_message_key":     eventMessageKey,
				"event_message":         eventMessage,
				"error_type":            result.chunk.ErrorType,
				"tool_name":             result.chunk.ToolName,
				"context_usage":         contextUsage,
				"agent_skill_selection": agentSkillSelectionPayload(result.chunk.GetAgentSkillSelection()),
				"agent_skill_runtime_evidence": agentSkillRuntimeEvidenceListPayload(
					result.chunk.GetAgentSkillRuntimeEvidence(),
				),
				"request_id": base.RequestID(c),
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

func agentSkillSelectionPayload(selection *pb.AgentSkillSelection) *agentSkillSelectionHTTP {
	if selection == nil {
		return nil
	}
	candidates := make([]agentSkillSelectionCandidateHTTP, 0, len(selection.GetCandidates()))
	for _, candidate := range selection.GetCandidates() {
		if candidate == nil {
			continue
		}
		candidates = append(candidates, agentSkillSelectionCandidatePayload(candidate))
	}
	return &agentSkillSelectionHTTP{
		Required:                        selection.GetRequired(),
		Reason:                          selection.GetReason(),
		Candidates:                      candidates,
		RecommendedAgentSkillVersionIDs: nonNilInt64List(selection.GetRecommendedAgentSkillVersionIds()),
		UserMessageID:                   selection.GetUserMessageId(),
	}
}

func (h *AIHandler) ListCapabilities(c *gin.Context) {
	resp, err := h.clients.AgentConfig.ListCapabilities(c.Request.Context(), &pb.ListCapabilitiesRequest{
		AgentType: "hr_recruiting_agent",
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	list := availableCapabilities(resp.List)
	base.From(c, resp.Code, resp.Msg, gin.H{"list": list})
}

func availableCapabilities(items []*pb.CapabilityInfo) []*pb.CapabilityInfo {
	list := make([]*pb.CapabilityInfo, 0, len(items))
	for _, item := range items {
		if item != nil && item.GetIsAvailable() {
			list = append(list, item)
		}
	}
	return list
}

func (h *AIHandler) History(c *gin.Context) {
	page, pageSize := basePagination(c)
	resp, err := h.clients.AI.History(c.Request.Context(), &pb.ChatHistoryRequest{HrId: middleware.UserID(c), Page: page, PageSize: pageSize})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": chatMessageListPayload(resp.GetList())})
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
	base.From(c, resp.Code, resp.Msg, gin.H{"list": chatMessageListPayload(resp.GetList())})
}

func (h *AIHandler) PreviewChatContext(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil || sessionID <= 0 {
		base.BadRequest(c, "会话 ID 不合法")
		return
	}
	var req struct {
		ModelID              base.FlexInt64 `json:"model_id"`
		CapabilityKeys       []string       `json:"capability_keys"`
		AgentSkillVersionIDs []int64        `json:"agent_skill_version_ids"`
	}
	if err := bindStrictJSON(c, &req); err != nil {
		base.BadRequest(c, "请求参数格式错误")
		return
	}
	if err := validateAgentSkillVersionIDs(req.AgentSkillVersionIDs); err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	resp, err := h.clients.AI.PreviewChatContext(c.Request.Context(), &pb.PreviewChatContextRequest{
		HrId:                 middleware.UserID(c),
		SessionId:            sessionID,
		ModelId:              int64(req.ModelID),
		CapabilityKeys:       req.CapabilityKeys,
		AgentSkillVersionIds: req.AgentSkillVersionIDs,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"selected_model_id": resp.SelectedModelId,
		"context_usage":     mapHRContextUsage(resp.GetContextUsage()),
	})
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
	releaseVersionID := resolveTenantAICapabilityVersion(c, h.clients, "ai.application_analysis")
	if releaseVersionID <= 0 {
		return
	}
	resp, err := h.clients.AI.CreateApplicationAnalysisSession(c.Request.Context(), &pb.CreateApplicationAnalysisSessionRequest{HrId: middleware.UserID(c), ApplicationId: int64(req.ApplicationID), ModelId: int64(req.ModelID)})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{
		"session":  resp.Session,
		"messages": chatMessageListPayload(resp.GetMessages()),
	})
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
	releaseVersionID := resolveTenantAICapabilityVersion(c, h.clients, "ai.application_analysis")
	if releaseVersionID <= 0 {
		return
	}
	resp, err := h.clients.AI.AnalyzeApplication(c.Request.Context(), &pb.AnalyzeApplicationRequest{HrId: middleware.UserID(c), ApplicationId: int64(req.ApplicationID), ModelId: int64(req.ModelID), CapabilityVersionId: releaseVersionID})
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
		"context_usage":  resp.ContextUsage,
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
	base.From(c, resp.Code, resp.Msg, gin.H{"list": agentRunItemListPayload(resp.GetList())})
}

// CreateAgentRun creates a durable resumable HR Agent run (or returns an
// idempotent existing run for the same client_request_id). Lifecycle state is
// owned by the AI Agent backend; the gateway only forwards the command.
func (h *AIHandler) CreateAgentRun(c *gin.Context) {
	var req struct {
		SessionID            base.FlexInt64 `json:"session_id" binding:"required"`
		ClientRequestID      string         `json:"client_request_id"`
		Message              string         `json:"message"`
		ActionType           string         `json:"action_type"`
		ActionPayloadJSON    string         `json:"action_payload_json"`
		ApplicationID        base.FlexInt64 `json:"application_id"`
		ModelID              base.FlexInt64 `json:"model_id"`
		CapabilityKeys       []string       `json:"capability_keys"`
		AgentSkillVersionIDs []int64        `json:"agent_skill_version_ids"`
	}
	if err := bindStrictJSON(c, &req); err != nil {
		base.BadRequest(c, "请求参数不合法")
		return
	}
	if int64(req.SessionID) <= 0 {
		base.BadRequest(c, "会话 ID 不合法")
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		base.BadRequest(c, "消息内容不能为空")
		return
	}
	if err := validateAgentSkillVersionIDs(req.AgentSkillVersionIDs); err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	resp, err := h.clients.AI.CreateAgentRun(c.Request.Context(), &pb.CreateAgentRunRequest{
		HrId:                 middleware.UserID(c),
		SessionId:            int64(req.SessionID),
		ClientRequestId:      req.ClientRequestID,
		Message:              req.Message,
		ActionType:           req.ActionType,
		ActionPayloadJson:    req.ActionPayloadJSON,
		ApplicationId:        int64(req.ApplicationID),
		ModelId:              int64(req.ModelID),
		CapabilityKeys:       req.CapabilityKeys,
		AgentSkillVersionIds: req.AgentSkillVersionIDs,
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
				messageKey, message := base.LocalizedMessage(info.Code, info.Msg)
				payload := fmt.Sprintf("data: %s\n\n", mustMarshalHR(gin.H{
					"code":        info.Code,
					"message_key": messageKey,
					"msg":         message,
					"done":        true,
					"request_id":  base.RequestID(c),
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
		ClientRequestID              string  `json:"client_request_id"`
		ConfirmationPayloadJSON      string  `json:"confirmation_payload_json"`
		AgentSkillConfirmationID     string  `json:"agent_skill_confirmation_id"`
		AgentSkillDecision           string  `json:"agent_skill_confirmation_decision"`
		SelectedAgentSkillVersionIDs []int64 `json:"selected_agent_skill_version_ids"`
	}
	if err := bindStrictJSON(c, &req); err != nil {
		base.BadRequest(c, "请求参数不合法")
		return
	}
	decision, err := parseAgentSkillConfirmationDecision(req.AgentSkillDecision)
	if err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	hasSkillConfirmation := strings.TrimSpace(req.AgentSkillConfirmationID) != ""
	hasMCPConfirmation := strings.TrimSpace(req.ConfirmationPayloadJSON) != ""
	if !hasSkillConfirmation && !hasMCPConfirmation {
		base.BadRequest(c, "confirmation payload is required")
		return
	}
	if hasSkillConfirmation && hasMCPConfirmation {
		base.BadRequest(c, "agent skill and MCP confirmations must be submitted independently")
		return
	}
	if hasSkillConfirmation {
		if err := validateAgentSkillVersionIDs(req.SelectedAgentSkillVersionIDs); err != nil {
			base.BadRequest(c, err.Error())
			return
		}
		if decision == pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_UNSPECIFIED {
			base.BadRequest(c, "agent_skill_confirmation_decision is required")
			return
		}
		if decision == pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_APPROVE &&
			len(req.SelectedAgentSkillVersionIDs) == 0 {
			base.BadRequest(c, "selected_agent_skill_version_ids are required for approval")
			return
		}
	} else if decision != pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_UNSPECIFIED ||
		len(req.SelectedAgentSkillVersionIDs) > 0 {
		base.BadRequest(c, "agent skill decision requires agent_skill_confirmation_id")
		return
	}
	resp, err := h.clients.AI.ConfirmAgentRun(c.Request.Context(), &pb.ConfirmAgentRunRequest{
		HrId:                           middleware.UserID(c),
		RunId:                          runID,
		ClientRequestId:                strings.TrimSpace(req.ClientRequestID),
		ConfirmationPayloadJson:        strings.TrimSpace(req.ConfirmationPayloadJSON),
		AgentSkillConfirmationId:       strings.TrimSpace(req.AgentSkillConfirmationID),
		AgentSkillConfirmationDecision: decision,
		SelectedAgentSkillVersionIds:   req.SelectedAgentSkillVersionIDs,
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
	errorKey, errorMessage := localizedAgentError(run.GetErrorMessage())
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
		"error_message_key":    errorKey,
		"error_message":        errorMessage,
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
	errorKey, errorMessage := localizedAgentError(event.GetErrorMessage())
	return gin.H{
		"run_id":            event.GetRunId(),
		"seq":               event.GetSeq(),
		"event_type":        event.GetEventType(),
		"payload_json":      event.GetPayloadJson(),
		"status":            event.GetStatus(),
		"delta":             event.GetDelta(),
		"snapshot_text":     event.GetSnapshotText(),
		"result_metadata":   agentRunResultMetadataPayload(event.GetResultMetadata()),
		"confirmation":      agentRunConfirmationPayload(event.GetConfirmation()),
		"tool_name":         event.GetToolName(),
		"error_type":        event.GetErrorType(),
		"error_message_key": errorKey,
		"error_message":     errorMessage,
		"created_at":        event.GetCreatedAt(),
	}
}

func mapHRContextUsage(cu *pb.ContextUsageInfo) map[string]any {
	if cu == nil {
		return nil
	}
	var breakdown map[string]any
	if bd := cu.GetBreakdown(); bd != nil {
		breakdown = gin.H{
			"system_prompt_tokens":     bd.GetSystemPromptTokens(),
			"recent_message_tokens":    bd.GetRecentMessageTokens(),
			"summary_tokens":           bd.GetSummaryTokens(),
			"memory_tokens":            bd.GetMemoryTokens(),
			"current_message_tokens":   bd.GetCurrentMessageTokens(),
			"skill_tokens":             bd.GetSkillTokens(),
			"tool_result_tokens":       bd.GetToolResultTokens(),
			"tool_schema_tokens":       bd.GetToolSchemaTokens(),
			"protocol_overhead_tokens": bd.GetProtocolOverheadTokens(),
		}
	}
	return gin.H{
		"model_id":                   cu.GetModelId(),
		"model_name":                 cu.GetModelName(),
		"requested_model_id":         cu.GetRequestedModelId(),
		"effective_model_id":         cu.GetEffectiveModelId(),
		"model_fallback_reason":      cu.GetModelFallbackReason(),
		"capability_version_id":      cu.GetCapabilityVersionId(),
		"capability_snapshot_hash":   cu.GetCapabilitySnapshotHash(),
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
		"input_budget_tokens":        cu.GetInputBudgetTokens(),
		"safety_margin_tokens":       cu.GetSafetyMarginTokens(),
		"budget_usage_ratio":         cu.GetBudgetUsageRatio(),
		"budget_status":              cu.GetBudgetStatus(),
		"included_message_count":     cu.GetIncludedMessageCount(),
		"omitted_message_count":      cu.GetOmittedMessageCount(),
		"summary_applied":            cu.GetSummaryApplied(),
		"memory_applied":             cu.GetMemoryApplied(),
	}
}

func agentRunResultMetadataPayload(meta *pb.AgentRunResultMetadata) map[string]any {
	if meta == nil {
		return nil
	}
	contextUsage := mapHRContextUsage(meta.GetContextUsage())
	errorKey, errorMessage := localizedAgentError(meta.GetErrorMessage())
	return gin.H{
		"action":              meta.GetAction(),
		"application_id":      meta.GetApplicationId(),
		"action_status":       meta.GetActionStatus(),
		"candidate_name":      meta.GetCandidateName(),
		"job_title":           meta.GetJobTitle(),
		"status":              meta.GetStatus(),
		"candidate_options":   meta.GetCandidateOptions(),
		"suggested_questions": nonNilStringList(meta.GetSuggestedQuestions()),
		"context_usage":       contextUsage,
		"error_type":          meta.GetErrorType(),
		"error_message_key":   errorKey,
		"error_message":       errorMessage,
		"raw_json":            meta.GetRawJson(),
		"agent_skill_runtime_evidence": agentSkillRuntimeEvidenceListPayload(
			meta.GetAgentSkillRuntimeEvidence(),
		),
	}
}

func agentRunItemListPayload(items []*pb.AgentRunItem) []gin.H {
	result := make([]gin.H, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, gin.H{
			"id":              item.GetId(),
			"session_id":      item.GetSessionId(),
			"message_id":      item.GetMessageId(),
			"history_id":      item.GetHistoryId(),
			"hr_id":           item.GetHrId(),
			"agent_type":      item.GetAgentType(),
			"agent_id":        item.GetAgentId(),
			"agent_name":      item.GetAgentName(),
			"model_id":        item.GetModelId(),
			"model_name":      item.GetModelName(),
			"status":          item.GetStatus(),
			"plan_json":       item.GetPlanJson(),
			"final_answer":    item.GetFinalAnswer(),
			"error_type":      item.GetErrorType(),
			"error_message":   item.GetErrorMessage(),
			"started_at":      item.GetStartedAt(),
			"completed_at":    item.GetCompletedAt(),
			"created_at":      item.GetCreatedAt(),
			"steps":           item.GetSteps(),
			"result_metadata": agentRunResultMetadataPayload(item.GetResultMetadata()),
		})
	}
	return result
}

func localizedAgentError(message string) (string, string) {
	if strings.TrimSpace(message) == "" {
		return "", ""
	}
	return base.LocalizedMessage(500, message)
}

func agentRunConfirmationPayload(confirmation *pb.AgentRunConfirmationPayload) *agentRunConfirmationHTTP {
	if confirmation == nil {
		return nil
	}
	candidates := make([]agentSkillSelectionCandidateHTTP, 0, len(confirmation.GetCandidates()))
	for _, candidate := range confirmation.GetCandidates() {
		if candidate == nil {
			continue
		}
		candidates = append(candidates, agentSkillSelectionCandidatePayload(candidate))
	}
	return &agentRunConfirmationHTTP{
		Required:                        confirmation.GetRequired(),
		Reason:                          confirmation.GetReason(),
		Candidates:                      candidates,
		RawJSON:                         confirmation.GetRawJson(),
		AgentSkillConfirmationID:        confirmation.GetAgentSkillConfirmationId(),
		RecommendedAgentSkillVersionIDs: nonNilInt64List(confirmation.GetRecommendedAgentSkillVersionIds()),
		AgentSkillUserMessageID:         confirmation.GetAgentSkillUserMessageId(),
		AgentSkillConfirmationExpiresAt: confirmation.GetAgentSkillConfirmationExpiresAt(),
	}
}

func parseAgentSkillConfirmationDecision(value string) (pb.AgentSkillConfirmationDecision, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "":
		return pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_UNSPECIFIED, nil
	case "approve":
		return pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_APPROVE, nil
	case "reject":
		return pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_REJECT, nil
	default:
		return pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_UNSPECIFIED,
			fmt.Errorf("agent_skill_confirmation_decision must be approve or reject")
	}
}

func validateAgentSkillVersionIDs(values []int64) error {
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			return fmt.Errorf("agent skill version ids must be positive")
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("agent skill version ids must be unique")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func agentSkillSelectionCandidatePayload(candidate *pb.AgentSkillSelectionCandidate) agentSkillSelectionCandidateHTTP {
	if candidate == nil {
		return agentSkillSelectionCandidateHTTP{}
	}
	return agentSkillSelectionCandidateHTTP{
		SkillID:             candidate.GetSkillId(),
		VersionID:           candidate.GetVersionId(),
		Version:             candidate.GetVersion(),
		CompiledHash:        candidate.GetCompiledHash(),
		Name:                candidate.GetName(),
		DisplayName:         candidate.GetDisplayName(),
		Reason:              candidate.GetReason(),
		Score:               candidate.GetScore(),
		Priority:            candidate.GetPriority(),
		Category:            candidate.GetCategory(),
		Scenario:            candidate.GetScenario(),
		CompositionRole:     agentSkillCompositionRoleValue(candidate.GetCompositionRole()),
		Risk:                agentSkillRiskValue(candidate.GetRisk()),
		ActivationPolicy:    agentSkillActivationPolicyValue(candidate.GetActivationPolicy()),
		CoreEstimatedTokens: candidate.GetCoreEstimatedTokens(),
		Recommended:         candidate.GetRecommended(),
		VectorScore:         candidate.GetVectorScore(),
		LexicalScore:        candidate.GetLexicalScore(),
		MetadataScore:       candidate.GetMetadataScore(),
		RelevanceScore:      candidate.GetRelevanceScore(),
		BusinessBoost:       candidate.GetBusinessBoost(),
		FinalRankScore:      candidate.GetFinalRankScore(),
		RelevanceMode:       candidate.GetRelevanceMode(),
		PoolRank:            candidate.GetPoolRank(),
		RankingConfidence:   candidate.GetRankingConfidence(),
	}
}

func agentSkillRuntimeEvidenceListPayload(
	evidence []*pb.AgentSkillRuntimeEvidence,
) []agentSkillRuntimeEvidenceHTTP {
	result := make([]agentSkillRuntimeEvidenceHTTP, 0, len(evidence))
	for _, item := range evidence {
		if item == nil {
			continue
		}
		sections := make([]agentSkillSectionRuntimeEvidenceHTTP, 0, len(item.GetSections()))
		for _, section := range item.GetSections() {
			if section == nil {
				continue
			}
			sections = append(sections, agentSkillSectionRuntimeEvidenceHTTP{
				SectionID:       section.GetSectionId(),
				SectionKey:      section.GetSectionKey(),
				ContentHash:     section.GetContentHash(),
				EstimatedTokens: section.GetEstimatedTokens(),
				FinalRankScore:  section.GetFinalRankScore(),
				Included:        section.GetIncluded(),
				DecisionReason:  section.GetDecisionReason(),
			})
		}
		result = append(result, agentSkillRuntimeEvidenceHTTP{
			SkillID:             item.GetSkillId(),
			VersionID:           item.GetVersionId(),
			Version:             item.GetVersion(),
			CompiledHash:        item.GetCompiledHash(),
			SkillName:           item.GetSkillName(),
			DisplayName:         item.GetDisplayName(),
			CompositionRole:     agentSkillCompositionRoleValue(item.GetCompositionRole()),
			Risk:                agentSkillRiskValue(item.GetRisk()),
			ActivationPolicy:    agentSkillActivationPolicyValue(item.GetActivationPolicy()),
			SelectionMode:       item.GetSelectionMode(),
			RelevanceMode:       item.GetRelevanceMode(),
			CoreEstimatedTokens: item.GetCoreEstimatedTokens(),
			LoadedTokens:        item.GetLoadedTokens(),
			Sections:            sections,
			Included:            item.GetIncluded(),
			DecisionReason:      item.GetDecisionReason(),
			VectorScore:         item.GetVectorScore(),
			LexicalScore:        item.GetLexicalScore(),
			MetadataScore:       item.GetMetadataScore(),
			RelevanceScore:      item.GetRelevanceScore(),
			BusinessBoost:       item.GetBusinessBoost(),
			FinalRankScore:      item.GetFinalRankScore(),
		})
	}
	return result
}

func chatMessageListPayload(messages []*pb.ChatMessage) []chatMessageHTTP {
	result := make([]chatMessageHTTP, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		result = append(result, chatMessagePayload(message))
	}
	return result
}

func chatMessagePayload(message *pb.ChatMessage) chatMessageHTTP {
	if message == nil {
		return chatMessageHTTP{
			AgentSkillNames:           []string{},
			AgentSkillVersionIDs:      []int64{},
			AgentSkillRuntimeEvidence: []agentSkillRuntimeEvidenceHTTP{},
		}
	}
	return chatMessageHTTP{
		Role:                      message.GetRole(),
		Content:                   message.GetContent(),
		CreatedAt:                 message.GetCreatedAt(),
		ModelID:                   message.GetModelId(),
		ModelName:                 message.GetModelName(),
		AgentSkillNames:           nonNilStringList(message.GetAgentSkillNames()),
		ProcessContent:            message.GetProcessContent(),
		ContextUsage:              mapHRContextUsage(message.GetContextUsage()),
		AgentSkillVersionIDs:      nonNilInt64List(message.GetAgentSkillVersionIds()),
		AgentSkillRuntimeEvidence: agentSkillRuntimeEvidenceListPayload(message.GetAgentSkillRuntimeEvidence()),
	}
}

func nonNilStringList(values []string) []string {
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func nonNilInt64List(values []int64) []int64 {
	result := make([]int64, len(values))
	copy(result, values)
	return result
}

func mustMarshalHR(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
