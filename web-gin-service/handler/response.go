package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"web-gin-service/pkg/logger"
)

type ErrorInfo struct {
	Code int32
	Msg  string
}

func OK(c *gin.Context, msg string, data any) {
	if msg == "" {
		msg = "success"
	}
	c.JSON(200, envelope(c, 0, msg, data))
}

func From(c *gin.Context, code int32, msg string, data any) {
	if msg == "" {
		msg = "success"
	}
	c.JSON(200, envelope(c, code, msg, data))
}

// ProtoResponse converts a protobuf response with code/msg fields to the standard JSON envelope.
// Fields other than code and msg become the "data" payload, using proto snake_case names.
func ProtoResponse(c *gin.Context, msg proto.Message) {
	mo := protojson.MarshalOptions{UseProtoNames: true}
	jsonBytes, _ := mo.Marshal(msg)
	var raw map[string]any
	_ = json.Unmarshal(jsonBytes, &raw)
	code := int32(0)
	respMsg := "success"
	if v, ok := raw["code"].(float64); ok {
		code = int32(v)
	}
	if v, ok := raw["msg"].(string); ok {
		respMsg = v
	}
	delete(raw, "code")
	delete(raw, "msg")
	delete(raw, "request_id")
	From(c, code, respMsg, raw)
}

func BadRequest(c *gin.Context, msg string) {
	c.JSON(200, envelope(c, 400, msg, nil))
}

func Internal(c *gin.Context, err error) {
	info := PublicError(err)
	logger.L().Error("internal error",
		zap.String("request_id", RequestID(c)),
		zap.Int32("public_code", info.Code),
		zap.String("public_msg", info.Msg),
		zap.Error(err),
	)
	c.JSON(200, envelope(c, info.Code, info.Msg, nil))
}

func PublicError(err error) ErrorInfo {
	if err == nil {
		return ErrorInfo{Code: 500, Msg: "服务暂时不可用，请稍后重试"}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorInfo{Code: 504, Msg: "请求处理超时，请稍后重试"}
	}
	if errors.Is(err, context.Canceled) {
		return ErrorInfo{Code: 499, Msg: "请求已取消，请重新操作"}
	}
	// gRPC status codes — preferred classification method.
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.DeadlineExceeded:
			return ErrorInfo{Code: 504, Msg: "服务响应超时，请稍后重试"}
		case codes.Unavailable:
			return ErrorInfo{Code: 503, Msg: "后端服务暂不可用，请稍后重试"}
		case codes.PermissionDenied:
			return ErrorInfo{Code: 403, Msg: "当前账号没有权限执行这个操作"}
		case codes.Unauthenticated:
			return ErrorInfo{Code: 401, Msg: "登录状态已失效，请重新登录"}
		case codes.InvalidArgument:
			return ErrorInfo{Code: 400, Msg: "请求参数不合法，请检查后重试"}
		case codes.NotFound:
			return ErrorInfo{Code: 404, Msg: "请求的资源不存在或已失效"}
		case codes.Internal:
			msg := st.Message()
			if info, ok := classifyAIError(msg); ok {
				return info
			}
			if strings.HasPrefix(msg, "oss:") {
				return ErrorInfo{Code: 404, Msg: "文件不存在或已失效，请重新上传"}
			}
			return ErrorInfo{Code: 500, Msg: "服务暂时不可用，请稍后重试"}
		case codes.ResourceExhausted:
			msg := st.Message()
			switch {
			case strings.HasPrefix(msg, "quota:ai_daily:"):
				return ErrorInfo{Code: 42901, Msg: "今日 AI 使用次数已达上限，请明天再试"}
			case strings.HasPrefix(msg, "quota:resume_presign:"):
				return ErrorInfo{Code: 42911, Msg: "简历上传过于频繁，请稍后再试"}
			case strings.HasPrefix(msg, "quota:resume_confirm:"):
				return ErrorInfo{Code: 42912, Msg: "简历上传确认过于频繁，请稍后再试"}
			case strings.HasPrefix(msg, "risk:block:"):
				return ErrorInfo{Code: 42921, Msg: "当前操作过于频繁，请稍后再试"}
			default:
				return ErrorInfo{Code: 429, Msg: "请求过于频繁，请稍后重试"}
			}
		}
	}
	// Fallback string-based matching for unwrapped errors.
	text := strings.ToLower(err.Error())
	if info, ok := classifyAIError(err.Error()); ok {
		return info
	}
	switch {
	case strings.Contains(text, "deadline exceeded"), strings.Contains(text, "timeout"), strings.Contains(text, "timed out"):
		return ErrorInfo{Code: 504, Msg: "请求处理超时，请稍后重试"}
	case strings.Contains(text, "connection refused"), strings.Contains(text, "unavailable"), strings.Contains(text, "connection error"):
		return ErrorInfo{Code: 503, Msg: "后端服务暂不可用，请稍后重试"}
	case strings.Contains(text, "dashscope"), strings.Contains(text, "chat completions"):
		return ErrorInfo{Code: 502, Msg: "AI 服务暂时不可用，请稍后重试"}
	case strings.Contains(text, "oss:"), strings.Contains(text, "not found"):
		return ErrorInfo{Code: 404, Msg: "文件不存在或已失效，请重新上传"}
	default:
		return ErrorInfo{Code: 500, Msg: "服务暂时不可用，请稍后重试"}
	}
}

// classifyAIError parses an AI error message in the format "ai:<TYPE>: <message>"
// and returns a friendly ErrorInfo. Returns ok=false if the message is not an AI error.
//
// Status code mapping:
// - timeout/empty/unknown → 502 (AI bad gateway)
// - rate_limited → 429
// - circuit_open / unavailable → 503
// - tool_failed → 502
// - canceled → 499 (only when not handled upstream)
// - partial_reply → 200-class but msg flags partial; we report 502 to keep UX aligned
func classifyAIError(msg string) (ErrorInfo, bool) {
	if !strings.HasPrefix(msg, "ai:") {
		return ErrorInfo{}, false
	}
	rest := strings.TrimPrefix(msg, "ai:")
	// Format: "<TYPE>: <user message>"
	parts := strings.SplitN(rest, ":", 2)
	aiType := strings.TrimSpace(parts[0])
	userMsg := ""
	if len(parts) == 2 {
		userMsg = strings.TrimSpace(parts[1])
	}
	if userMsg == "" {
		userMsg = "AI 服务暂时不可用，请稍后重试"
	}
	switch aiType {
	case "AI_TIMEOUT":
		return ErrorInfo{Code: 504, Msg: userMsg}, true
	case "AI_RATE_LIMITED":
		return ErrorInfo{Code: 429, Msg: userMsg}, true
	case "AI_UNAVAILABLE":
		return ErrorInfo{Code: 503, Msg: userMsg}, true
	case "AI_CIRCUIT_OPEN":
		return ErrorInfo{Code: 503, Msg: userMsg}, true
	case "AI_EMPTY_REPLY":
		return ErrorInfo{Code: 502, Msg: userMsg}, true
	case "AI_TOOL_FAILED":
		return ErrorInfo{Code: 502, Msg: userMsg}, true
	case "AI_CONTEXT_CANCELED":
		return ErrorInfo{Code: 499, Msg: userMsg}, true
	case "AI_PARTIAL_REPLY":
		return ErrorInfo{Code: 502, Msg: userMsg}, true
	case "AI_UNKNOWN":
		return ErrorInfo{Code: 502, Msg: userMsg}, true
	default:
		// Legacy "ai: <raw>" form: keep generic prompt.
		return ErrorInfo{Code: 502, Msg: "AI 服务暂时不可用，请稍后重试"}, true
	}
}

func RequestID(c *gin.Context) string {
	if value, ok := c.Get("request_id"); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}

func envelope(c *gin.Context, code int32, msg string, data any) gin.H {
	return gin.H{"code": code, "msg": msg, "data": data, "request_id": RequestID(c)}
}

// FlushSSE flushes the response writer for SSE streams and returns false if
// the client has disconnected (broken pipe), signalling the caller to stop.
func FlushSSE(w gin.ResponseWriter) bool {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
		return true
	}
	return true
}
