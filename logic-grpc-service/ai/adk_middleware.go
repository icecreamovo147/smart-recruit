package ai

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"go.uber.org/zap"

	"logic-grpc-service/pkg/logger"
)

// AgentRunState carries per-run mutable metadata that tools and middleware
// write during a single agent invocation. Eino ADK Tool default only returns
// string/structured output, so business metadata (CandidateOptions, Action,
// ToolTraces) must be captured through middleware or tool closures.
type AgentRunState struct {
	Metadata ToolMetadata
	Mu       sync.Mutex
}

// Merge atomically merges tool-level metadata into the run state.
func (s *AgentRunState) Merge(meta ToolMetadata) {
	if s == nil {
		return
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Metadata.merge(meta)
}

// RecordTrace atomically appends a tool trace entry.
func (s *AgentRunState) RecordTrace(t ToolTrace) {
	if s == nil {
		return
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Metadata.recordTrace(t)
}

// ReadMetadata returns a copy of the current metadata under lock.
// Use this for reads from goroutines other than the tool execution goroutine.
func (s *AgentRunState) ReadMetadata() ToolMetadata {
	if s == nil {
		return ToolMetadata{}
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	// Return a shallow copy — the caller must not mutate slices/maps.
	return s.Metadata
}

// RecruitingAgentMiddleware implements adk.ChatModelAgentMiddleware to
// inject status events, tool traces, timing, and metadata collection into
// ADK ChatModelAgent runs. Embed BaseChatModelAgentMiddleware for no-op
// defaults on unused methods.
type RecruitingAgentMiddleware struct {
	adk.BaseChatModelAgentMiddleware
	State          *AgentRunState
	OnToolExecuted ToolTraceCallback
	OnStatus       func(eventType, eventMessage, errorType, toolName string) error
}

// WrapInvokableToolCall wraps synchronous tool invocation to capture timing,
// traces, status events, and metadata. Tool errors are returned as JSON error
// strings with nil error so the model can continue reasoning.
func (m *RecruitingAgentMiddleware) WrapInvokableToolCall(
	ctx context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		sendStatus(m.OnStatus, "tool_calling", "正在查询"+tCtx.Name+"...", "", tCtx.Name)

		start := time.Now()
		output, execErr := endpoint(ctx, argumentsInJSON, opts...)
		cost := time.Since(start)

		resultContent := output
		if execErr != nil {
			resultContent = marshalToolError(execErr)
		}

		m.State.RecordTrace(ToolTrace{
			ToolName:  tCtx.Name,
			Arguments: parseArgsForTrace(argumentsInJSON),
			Result:    resultContent,
			Cost:      cost,
			Error:     execErr,
		})

		if m.OnToolExecuted != nil {
			m.OnToolExecuted(tCtx.CallID, tCtx.Name, argumentsInJSON, resultContent, execErr)
		}

		sendStatus(m.OnStatus, "tool_done", "数据查询完成", "", "")
		logger.L().Info("[ADK工具] 执行完成",
			zap.String("tool", tCtx.Name),
			zap.Duration("cost", cost),
			zap.Int("result_chars", len([]rune(resultContent))),
		)

		// Return JSON error as content (nil error) so the model can see it.
		if execErr != nil {
			return resultContent, nil
		}
		return output, nil
	}, nil
}

// marshalToolError converts a tool execution error into a JSON error object
// safe for model consumption. The error message is business-facing — no SQL,
// stack traces, or internal paths.
func marshalToolError(err error) string {
	if err == nil {
		return `{"error":false}`
	}
	data, _ := json.Marshal(map[string]any{
		"error":   true,
		"message": err.Error(),
	})
	return string(data)
}

// parseArgsForTrace attempts to decode the JSON arguments string into a
// map for human-readable trace logging. On failure it returns the raw string.
func parseArgsForTrace(argumentsInJSON string) map[string]any {
	var args map[string]any
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		args = map[string]any{"_raw": argumentsInJSON}
	}
	return args
}
