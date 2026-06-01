package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// AIErrorType is a stable, taggable classification of AI-related failures.
// The string form is propagated to the gateway/frontend via gRPC status messages
// so the frontend can show the right user-facing prompt without re-parsing.
type AIErrorType string

const (
	AITimeout         AIErrorType = "AI_TIMEOUT"
	AIRateLimited     AIErrorType = "AI_RATE_LIMITED"
	AIUnavailable     AIErrorType = "AI_UNAVAILABLE"
	AIEmptyReply      AIErrorType = "AI_EMPTY_REPLY"
	AIToolFailed      AIErrorType = "AI_TOOL_FAILED"
	AIContextCanceled AIErrorType = "AI_CONTEXT_CANCELED"
	AICircuitOpen     AIErrorType = "AI_CIRCUIT_OPEN"
	AIPartialReply    AIErrorType = "AI_PARTIAL_REPLY"
	AIUnknown         AIErrorType = "AI_UNKNOWN"
)

// AIError is the shared error envelope for AI-pipeline failures. The Type drives
// frontend prompts and metrics; UserMessage is the canonical Chinese prompt; Cause
// is preserved for logs.
type AIError struct {
	Type        AIErrorType
	UserMessage string
	Cause       error
}

func (e *AIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.UserMessage, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.UserMessage)
}

func (e *AIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewAIError constructs an AIError; if userMsg is empty, a default per-type prompt is used.
func NewAIError(t AIErrorType, userMsg string, cause error) *AIError {
	if userMsg == "" {
		userMsg = defaultUserMessage(t)
	}
	return &AIError{Type: t, UserMessage: userMsg, Cause: cause}
}

// defaultUserMessage returns the canonical Chinese user-facing prompt per error type.
// These strings come directly from .ai-guides/ai-assistant-high-availability-plan.md §4.
func defaultUserMessage(t AIErrorType) string {
	switch t {
	case AITimeout:
		return "AI 服务响应较慢，本次回答已超时，请稍后重试。"
	case AIRateLimited:
		return "AI 服务当前繁忙，请稍后再试。"
	case AIUnavailable:
		return "AI 服务暂时不可用，请稍后重试。"
	case AIEmptyReply:
		return "AI 未能生成有效回复，请稍后重试。"
	case AIToolFailed:
		return "数据查询失败，请稍后重试或缩小查询范围。"
	case AIContextCanceled:
		return "已停止生成。"
	case AICircuitOpen:
		return "AI 服务暂时不可用，系统已保护性暂停请求，请稍后再试。"
	case AIPartialReply:
		return "AI 仅生成了部分回复，请重新提问以获取完整答案。"
	default:
		return "AI 服务发生未知错误，请稍后重试。"
	}
}

// UserMessageFor returns the canonical user prompt for a given type.
// Exposed so callers (gateway, frontend message defaults) can reuse the catalog.
func UserMessageFor(t AIErrorType) string {
	return defaultUserMessage(t)
}

// ClassifyAIError maps any error coming out of the AI pipeline to a typed AIError.
// It keeps the original cause so logs are not lossy.
//
// Order matters: explicit sentinels first, then context errors, then string-based
// heuristics for upstream provider errors that don't expose typed sentinels.
func ClassifyAIError(err error) *AIError {
	if err == nil {
		return nil
	}
	// If it's already a typed AIError, return as-is.
	var aiErr *AIError
	if errors.As(err, &aiErr) {
		return aiErr
	}

	// Sentinel errors first.
	if errors.Is(err, ErrCircuitOpen) {
		return NewAIError(AICircuitOpen, "", err)
	}
	if errors.Is(err, context.Canceled) {
		return NewAIError(AIContextCanceled, "", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return NewAIError(AITimeout, "", err)
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "circuit"):
		return NewAIError(AICircuitOpen, "", err)
	case strings.Contains(msg, "rate limit"),
		strings.Contains(msg, "rate_limit"),
		strings.Contains(msg, "ratelimit"),
		strings.Contains(msg, "429"),
		strings.Contains(msg, "too many requests"):
		return NewAIError(AIRateLimited, "", err)
	case strings.Contains(msg, "deadline exceeded"),
		strings.Contains(msg, "timeout"),
		strings.Contains(msg, "timed out"):
		return NewAIError(AITimeout, "", err)
	case strings.Contains(msg, "empty reply"),
		strings.Contains(msg, "empty response"),
		strings.Contains(msg, "no choices"):
		return NewAIError(AIEmptyReply, "", err)
	case strings.Contains(msg, "tool"),
		strings.Contains(msg, "execute"):
		// Best-effort: explicit tool-call failure path also wraps via AIToolFailed.
		return NewAIError(AIToolFailed, "", err)
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "unavailable"),
		strings.Contains(msg, "503"),
		strings.Contains(msg, "502"),
		strings.Contains(msg, "504"):
		return NewAIError(AIUnavailable, "", err)
	}
	return NewAIError(AIUnknown, "", err)
}

// IsRetryable reports whether an AI error class should be retried at the call layer.
// Non-idempotent or user-driven errors (canceled, circuit-open) are NOT retried.
func IsRetryable(t AIErrorType) bool {
	switch t {
	case AITimeout, AIRateLimited, AIUnavailable:
		return true
	default:
		return false
	}
}
