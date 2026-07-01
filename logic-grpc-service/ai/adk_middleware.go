package ai

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
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

// RecordModelUsage atomically records a model call's token usage for both
// billing (cumulative) and context (latest-call) semantics.
func (s *AgentRunState) RecordModelUsage(usage *schema.TokenUsage) {
	if s == nil || usage == nil {
		return
	}
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Metadata.recordModelUsage(usage)
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
	State             *AgentRunState
	OnToolExecuted    ToolTraceCallback
	OnStatus          func(eventType, eventMessage, errorType, toolName string) error
	OnMessagesUpdated MessageUpdateCallback
}

const adkContextUsageToolSeenKey = "smart_recruit.context_usage_tool_seen"

// BeforeModelRewriteState observes the exact ADK message state before each
// model invocation. After a tool has run, this state includes tool results and
// is the authoritative input for context usage estimation.
func (m *RecruitingAgentMiddleware) BeforeModelRewriteState(
	ctx context.Context,
	state *adk.ChatModelAgentState,
	mc *adk.ModelContext,
) (context.Context, *adk.ChatModelAgentState, error) {
	if m.OnMessagesUpdated == nil || state == nil {
		return ctx, state, nil
	}
	if seen, _, err := adk.GetRunLocalValue(ctx, adkContextUsageToolSeenKey); err == nil {
		if v, ok := seen.(bool); ok && v {
			_ = adk.SetRunLocalValue(ctx, adkContextUsageToolSeenKey, false)
			if err := m.OnMessagesUpdated(state.Messages, "tool_result"); err != nil {
				return ctx, state, err
			}
		}
	}
	return ctx, state, nil
}

// AfterModelRewriteState observes the state after model output is appended.
// When the agent finishes, this is the final prompt state used for persistence.
func (m *RecruitingAgentMiddleware) AfterModelRewriteState(
	ctx context.Context,
	state *adk.ChatModelAgentState,
	mc *adk.ModelContext,
) (context.Context, *adk.ChatModelAgentState, error) {
	if m.OnMessagesUpdated == nil || state == nil {
		return ctx, state, nil
	}
	if len(state.Messages) == 0 {
		return ctx, state, nil
	}
	last := state.Messages[len(state.Messages)-1]
	if last != nil && len(last.ToolCalls) == 0 {
		if err := m.OnMessagesUpdated(state.Messages, "final"); err != nil {
			return ctx, state, err
		}
	}
	return ctx, state, nil
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
			m.OnToolExecuted(tCtx.CallID, tCtx.Name, argumentsInJSON, resultContent, cost, execErr)
		}
		if m.OnMessagesUpdated != nil {
			_ = adk.SetRunLocalValue(ctx, adkContextUsageToolSeenKey, true)
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
