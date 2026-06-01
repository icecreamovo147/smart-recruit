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
		Message       string `json:"message" binding:"required"`
		ApplicationID int64  `json:"application_id"`
		SessionID     int64  `json:"session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "消息不能为空")
		return
	}
	resp, err := h.clients.AI.Chat(c.Request.Context(), &pb.ChatRequest{HrId: middleware.UserID(c), Message: req.Message, ApplicationId: req.ApplicationID, SessionId: req.SessionID})
	if err != nil {
		base.Internal(c, err)
		return
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
	})
}

func (h *AIHandler) ChatStream(c *gin.Context) {
	var req struct {
		Message       string `json:"message" binding:"required"`
		ApplicationID int64  `json:"application_id"`
		SessionID     int64  `json:"session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "消息不能为空")
		return
	}
	stream, err := h.clients.AI.ChatStream(c.Request.Context(), &pb.ChatRequest{HrId: middleware.UserID(c), Message: req.Message, ApplicationId: req.ApplicationID, SessionId: req.SessionID})
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
				payload := fmt.Sprintf("event: message\ndata: %s\n\n", mustMarshalHR(gin.H{"code": info.Code, "msg": info.Msg, "done": true, "request_id": base.RequestID(c)}))
				if n, err := c.Writer.Write([]byte(payload)); err != nil || n == 0 {
					return
				}
				if !base.FlushSSE(c.Writer) {
					return
				}
				return
			}
			payload := gin.H{
				"code":              result.chunk.Code,
				"msg":               result.chunk.Msg,
				"delta":             result.chunk.Delta,
				"done":              result.chunk.Done,
				"action":            result.chunk.Action,
				"application_id":    result.chunk.ApplicationId,
				"action_status":     result.chunk.ActionStatus,
				"candidate_name":    result.chunk.CandidateName,
				"job_title":         result.chunk.JobTitle,
				"status":            result.chunk.Status,
				"session_id":        result.chunk.SessionId,
				"created_at":        result.chunk.CreatedAt,
				"candidate_options": result.chunk.CandidateOptions,
				"event_type":        result.chunk.EventType,
				"event_message":     result.chunk.EventMessage,
				"error_type":        result.chunk.ErrorType,
				"tool_name":         result.chunk.ToolName,
				"request_id":        base.RequestID(c),
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
	base.From(c, resp.Code, resp.Msg, gin.H{"total": resp.Total, "list": resp.List})
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
		ApplicationID int64 `json:"application_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "投递记录 ID 不能为空")
		return
	}
	resp, err := h.clients.AI.CreateApplicationAnalysisSession(c.Request.Context(), &pb.CreateApplicationAnalysisSessionRequest{HrId: middleware.UserID(c), ApplicationId: req.ApplicationID})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, gin.H{"session": resp.Session, "messages": resp.Messages})
}

func (h *AIHandler) AnalyzeApplication(c *gin.Context) {
	var req struct {
		ApplicationID int64 `json:"application_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "投递记录 ID 不能为空")
		return
	}
	resp, err := h.clients.AI.AnalyzeApplication(c.Request.Context(), &pb.AnalyzeApplicationRequest{HrId: middleware.UserID(c), ApplicationId: req.ApplicationID})
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

func mustMarshalHR(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
