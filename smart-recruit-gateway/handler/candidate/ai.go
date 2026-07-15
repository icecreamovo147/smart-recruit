package candidate

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

// AIHandler handles candidate AI assistant endpoints.
type AIHandler struct {
	clients *rpc.Clients
}

func NewAIHandler(clients *rpc.Clients) *AIHandler {
	return &AIHandler{clients: clients}
}

// Chat handles non-streaming candidate AI chat by aggregating the streaming RPC.
func (h *AIHandler) Chat(c *gin.Context) {
	var req struct {
		Message   string         `json:"message" binding:"required"`
		SessionID base.FlexInt64 `json:"session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "消息不能为空")
		return
	}
	stream, err := h.clients.AI.CandidateChatStream(c.Request.Context(), &pb.CandidateChatRequest{
		UserId:    middleware.UserID(c),
		Message:   req.Message,
		SessionId: int64(req.SessionID),
	})
	if err != nil {
		base.Internal(c, err)
		return
	}

	var reply strings.Builder
	var sessionID int64
	var createdAt string
	var suggestedQuestions []string
	code := int32(0)
	msg := "success"
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			base.Internal(c, err)
			return
		}
		if chunk.GetCode() != 0 {
			base.From(c, chunk.GetCode(), chunk.GetMsg(), nil)
			return
		}
		code = chunk.GetCode()
		if chunk.GetMsg() != "" {
			msg = chunk.GetMsg()
		}
		if chunk.GetDelta() != "" {
			reply.WriteString(chunk.GetDelta())
		}
		if chunk.GetSessionId() > 0 {
			sessionID = chunk.GetSessionId()
		}
		if chunk.GetCreatedAt() != "" {
			createdAt = chunk.GetCreatedAt()
		}
		if len(chunk.GetSuggestedQuestions()) > 0 {
			suggestedQuestions = chunk.GetSuggestedQuestions()
		}
		if chunk.GetDone() {
			break
		}
	}
	base.From(c, code, msg, gin.H{
		"reply":               reply.String(),
		"created_at":          createdAt,
		"session_id":          sessionID,
		"suggested_questions": suggestedQuestions,
		"suggestedQuestions":  suggestedQuestions,
	})
}

// ChatStream handles SSE streaming candidate AI chat.
func (h *AIHandler) ChatStream(c *gin.Context) {
	var req struct {
		Message   string         `json:"message" binding:"required"`
		SessionID base.FlexInt64 `json:"session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "消息不能为空")
		return
	}
	stream, err := h.clients.AI.CandidateChatStream(c.Request.Context(), &pb.CandidateChatRequest{
		UserId:    middleware.UserID(c),
		Message:   req.Message,
		SessionId: int64(req.SessionID),
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
	ctx := c.Request.Context()
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
		case <-ctx.Done():
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
				payload := fmt.Sprintf("event: message\ndata: %s\n\n", mustMarshal(gin.H{
					"code":       info.Code,
					"msg":        info.Msg,
					"done":       true,
					"request_id": base.RequestID(c),
				}))
				if n, err := c.Writer.Write([]byte(payload)); err != nil || n == 0 {
					return
				}
				if !base.FlushSSE(c.Writer) {
					return
				}
				return
			}
			payload := gin.H{
				"code":                result.chunk.Code,
				"msg":                 result.chunk.Msg,
				"delta":               result.chunk.Delta,
				"done":                result.chunk.Done,
				"session_id":          result.chunk.SessionId,
				"created_at":          result.chunk.CreatedAt,
				"suggested_questions": result.chunk.SuggestedQuestions,
				"suggestedQuestions":  result.chunk.SuggestedQuestions,
				"event_type":          result.chunk.EventType,
				"event_message":       result.chunk.EventMessage,
				"error_type":          result.chunk.ErrorType,
				"tool_name":           result.chunk.ToolName,
				"request_id":          base.RequestID(c),
			}
			line := fmt.Sprintf("event: message\ndata: %s\n\n", mustMarshal(payload))
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

// ListSessions returns the candidate's AI chat sessions.
func (h *AIHandler) ListSessions(c *gin.Context) {
	page, pageSize := basePagination(c)
	resp, err := h.clients.AI.CandidateListSessions(c.Request.Context(), &pb.CandidateSessionListRequest{
		UserId:   middleware.UserID(c),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"total": resp.Total, "list": resp.List})
}

// CreateSession creates a new AI chat session for the candidate.
func (h *AIHandler) CreateSession(c *gin.Context) {
	var req struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数格式错误")
		return
	}
	resp, err := h.clients.AI.CandidateCreateSession(c.Request.Context(), &pb.CandidateCreateSessionRequest{
		UserId: middleware.UserID(c),
		Title:  req.Title,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"session": resp.Session})
}

// SessionMessages returns messages for a candidate's session.
func (h *AIHandler) SessionMessages(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "会话 ID 不合法")
		return
	}
	page, pageSize := basePagination(c)
	resp, err := h.clients.AI.CandidateSessionMessages(c.Request.Context(), &pb.CandidateSessionMessagesRequest{
		UserId:    middleware.UserID(c),
		SessionId: sessionID,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"list": resp.List})
}

// UpdateSession renames a candidate's AI chat session.
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
	resp, err := h.clients.AI.CandidateUpdateSession(c.Request.Context(), &pb.CandidateUpdateSessionRequest{
		UserId:    middleware.UserID(c),
		SessionId: sessionID,
		Title:     req.Title,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

// DeleteSession deletes a candidate's AI chat session.
func (h *AIHandler) DeleteSession(c *gin.Context) {
	sessionID, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "会话 ID 不合法")
		return
	}
	resp, err := h.clients.AI.CandidateDeleteSession(c.Request.Context(), &pb.CandidateDeleteSessionRequest{
		UserId:    middleware.UserID(c),
		SessionId: sessionID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func basePagination(c *gin.Context) (int32, int32) {
	page := int32(1)
	pageSize := int32(20)
	if v, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && v > 0 {
		page = int32(v)
	}
	if v, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && v > 0 && v <= 50 {
		pageSize = int32(v)
	}
	return page, pageSize
}

func mustMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
