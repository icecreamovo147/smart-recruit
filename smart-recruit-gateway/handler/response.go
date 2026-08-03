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

	"smart-recruit-gateway/pkg/logger"
	"smart-recruit-platform-go/i18n"
)

type ErrorInfo struct {
	Code int32
	Msg  string
}

func OK(c *gin.Context, msg string, data any) {
	if msg == "" {
		msg = "common.success"
	}
	c.JSON(200, envelope(c, 0, msg, data))
}

func From(c *gin.Context, code int32, msg string, data any) {
	if msg == "" {
		msg = "common.success"
	}
	c.JSON(200, envelope(c, code, msg, data))
}

// ProtoResponse converts a protobuf response with code/msg fields to the standard JSON envelope.
// Fields other than code and msg become the "data" payload, using proto snake_case names.
func ProtoResponse(c *gin.Context, msg proto.Message) {
	mo := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}
	jsonBytes, _ := mo.Marshal(msg)
	var raw map[string]any
	_ = json.Unmarshal(jsonBytes, &raw)
	code := int32(0)
	respMsg := "common.success"
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
	logger.L().Error("log.gateway.internal_error",
		zap.String("request_id", RequestID(c)),
		zap.Int32("public_code", info.Code),
		zap.String("message_key", resolveMessageKey(info.Code, info.Msg)),
		zap.String("cause", errorText(err)),
	)
	c.JSON(200, envelope(c, info.Code, info.Msg, nil))
}

func PublicError(err error) ErrorInfo {
	if err == nil {
		return ErrorInfo{Code: 500, Msg: "common.unknown_error"}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorInfo{Code: 504, Msg: "common.timeout"}
	}
	if errors.Is(err, context.Canceled) {
		return ErrorInfo{Code: 499, Msg: "common.canceled"}
	}
	// gRPC status codes — preferred classification method.
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.DeadlineExceeded:
			return ErrorInfo{Code: 504, Msg: "common.timeout"}
		case codes.Unavailable:
			return ErrorInfo{Code: 503, Msg: "common.backend_unavailable"}
		case codes.PermissionDenied:
			return ErrorInfo{Code: 403, Msg: "common.forbidden"}
		case codes.Unauthenticated:
			return ErrorInfo{Code: 401, Msg: "common.unauthenticated"}
		case codes.InvalidArgument:
			return ErrorInfo{Code: 400, Msg: "common.invalid_request"}
		case codes.NotFound:
			return ErrorInfo{Code: 404, Msg: "common.not_found"}
		case codes.AlreadyExists:
			return ErrorInfo{Code: 409, Msg: st.Message()}
		case codes.FailedPrecondition:
			return ErrorInfo{Code: 409, Msg: st.Message()}
		case codes.Internal:
			msg := st.Message()
			if info, ok := classifyAIError(msg); ok {
				return info
			}
			if strings.HasPrefix(msg, "oss:") {
				return ErrorInfo{Code: 404, Msg: "common.not_found"}
			}
			return ErrorInfo{Code: 500, Msg: "common.unknown_error"}
		case codes.ResourceExhausted:
			msg := st.Message()
			switch {
			case strings.Contains(msg, "insufficient_credits"):
				return ErrorInfo{Code: 40201, Msg: "ai.insufficient_credits"}
			case strings.HasPrefix(msg, "quota:ai_daily:"):
				return ErrorInfo{Code: 42901, Msg: "ai.daily_quota_exceeded"}
			case strings.HasPrefix(msg, "quota:resume_presign:"):
				return ErrorInfo{Code: 42911, Msg: "common.too_many_requests"}
			case strings.HasPrefix(msg, "quota:resume_confirm:"):
				return ErrorInfo{Code: 42912, Msg: "common.too_many_requests"}
			case strings.HasPrefix(msg, "risk:block:"):
				return ErrorInfo{Code: 42921, Msg: "common.too_many_requests"}
			default:
				return ErrorInfo{Code: 429, Msg: "common.too_many_requests"}
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
		return ErrorInfo{Code: 504, Msg: "common.timeout"}
	case strings.Contains(text, "connection refused"), strings.Contains(text, "unavailable"), strings.Contains(text, "connection error"):
		return ErrorInfo{Code: 503, Msg: "common.backend_unavailable"}
	case strings.Contains(text, "dashscope"), strings.Contains(text, "chat completions"):
		return ErrorInfo{Code: 502, Msg: "ai.unavailable"}
	case strings.Contains(text, "oss:"), strings.Contains(text, "not found"):
		return ErrorInfo{Code: 404, Msg: "common.not_found"}
	default:
		return ErrorInfo{Code: 500, Msg: "common.unknown_error"}
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
	switch aiType {
	case "AI_TIMEOUT":
		return ErrorInfo{Code: 504, Msg: "ai.stream_timeout"}, true
	case "AI_RATE_LIMITED":
		return ErrorInfo{Code: 429, Msg: "ai.too_many_requests"}, true
	case "AI_UNAVAILABLE":
		return ErrorInfo{Code: 503, Msg: "ai.unavailable"}, true
	case "AI_CIRCUIT_OPEN":
		return ErrorInfo{Code: 503, Msg: "ai.unavailable"}, true
	case "AI_EMPTY_REPLY":
		return ErrorInfo{Code: 502, Msg: "ai.invalid_response"}, true
	case "AI_TOOL_FAILED":
		return ErrorInfo{Code: 502, Msg: "ai.stream_failed"}, true
	case "AI_CONTEXT_CANCELED":
		return ErrorInfo{Code: 499, Msg: "common.canceled"}, true
	case "AI_PARTIAL_REPLY":
		return ErrorInfo{Code: 502, Msg: "ai.stream_failed"}, true
	case "AI_UNKNOWN":
		return ErrorInfo{Code: 502, Msg: "ai.unavailable"}, true
	default:
		// Legacy "ai: <raw>" form: keep generic prompt.
		return ErrorInfo{Code: 502, Msg: "ai.unavailable"}, true
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
	messageKey, message := LocalizedMessage(code, msg)
	return gin.H{
		"code":        code,
		"message_key": messageKey,
		"msg":         message,
		"data":        data,
		"request_id":  RequestID(c),
	}
}

// LocalizedMessage resolves an internal key or legacy response text into the
// stable public message contract. Raw upstream text is never returned.
func LocalizedMessage(code int32, message string) (string, string) {
	messageKey := resolveMessageKey(code, message)
	return messageKey, i18n.T(messageKey)
}

// LocalizedSystemMessage localizes an optional SSE/Agent system message. An
// unknown upstream value is replaced with a safe current-language fallback.
func LocalizedSystemMessage(message string) (string, string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", ""
	}
	if i18n.Has(message) {
		return message, i18n.T(message)
	}
	if key, ok := i18n.KeyForText(message); ok {
		return key, i18n.T(key)
	}
	return "common.operation_failed", i18n.T("common.operation_failed")
}

func resolveMessageKey(code int32, message string) string {
	message = strings.TrimSpace(message)
	if i18n.Has(message) {
		return message
	}
	if key, ok := i18n.KeyForText(message); ok {
		return key
	}
	if code == 0 || message == "success" || message == "ok" {
		return "common.success"
	}
	switch code {
	case 400:
		return "common.invalid_request"
	case 401:
		return "common.unauthenticated"
	case 40201:
		return "ai.insufficient_credits"
	case 403, 4030:
		return "common.forbidden"
	case 404:
		return "common.not_found"
	case 42901:
		return "ai.daily_quota_exceeded"
	case 42902:
		return "ai.too_many_requests"
	case 429, 42911, 42912, 42921:
		return "common.too_many_requests"
	case 499:
		return "common.canceled"
	case 502:
		return "ai.unavailable"
	case 503:
		return "common.backend_unavailable"
	case 504:
		return "common.timeout"
	default:
		return "common.operation_failed"
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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
