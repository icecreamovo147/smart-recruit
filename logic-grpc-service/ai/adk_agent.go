package ai

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"

	"logic-grpc-service/pkg/logger"
)

// AgentRunInput bundles the parameters needed by ChatWithADKAgent.
type AgentRunInput struct {
	AgentName     string
	Instruction   string
	Messages      []*schema.Message
	Tools         []tool.BaseTool
	MaxIterations int
	OwnerID       int64
	SessionID     int64
	// State is an optional pre-created AgentRunState. When non-nil, tools
	// and middleware share this state so business metadata (CandidateOptions,
	// Action) written by tools flows back to the caller. When nil,
	// ChatWithADKAgent creates a fresh state internally.
	State *AgentRunState
}

// ChatWithADKAgent executes a tool-calling conversation through Eino ADK
// ChatModelAgent. It replaces the hand-written WithTools -> Execute ->
// ToolMessage loop with the framework-managed ReAct loop.
//
// Streaming deltas are delivered via onDelta. Tool traces and metadata are
// captured through RecruitingAgentMiddleware and AgentRunState.
//
// Returns the final reply text, collected metadata, and any error.
func (c *Client) ChatWithADKAgent(
	ctx context.Context,
	input AgentRunInput,
	onDelta func(string) error,
	onToolExecuted ToolTraceCallback,
	onStatus func(eventType, eventMessage, errorType, toolName string) error,
) (string, ToolMetadata, error) {
	if c.cm == nil {
		return "", ToolMetadata{}, NewAIError(AIUnavailable, "", fmt.Errorf("ai chat model is nil"))
	}

	if c.totalTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = applyTotalBudget(ctx, c.totalTimeout)
		defer cancel()
	}

	maxIter := input.MaxIterations
	if maxIter <= 0 {
		maxIter = c.toolMaxRounds
	}
	if maxIter <= 0 {
		maxIter = 5
	}

	state := input.State
	if state == nil {
		state = &AgentRunState{}
	}

	middleware := &RecruitingAgentMiddleware{
		State:          state,
		OnToolExecuted: onToolExecuted,
		OnStatus:       onStatus,
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        input.AgentName,
		Description: "智能招聘助手，可查询岗位、候选人、投递、趋势数据并生成状态变更动作",
		Instruction: escapeADKInstruction(input.Instruction),
		Model:       c.cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: input.Tools,
			},
		},
		MaxIterations: maxIter,
		Handlers: []adk.ChatModelAgentMiddleware{
			middleware,
		},
	})
	if err != nil {
		return "", ToolMetadata{}, fmt.Errorf("create adk agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	sendStatus(onStatus, "thinking", "正在分析问题...", "", "")

	start := time.Now()
	// ADK's defaultGenModelInput prepends Instruction as a SystemMessage,
	// so we strip any existing system message from Messages to avoid doubling.
	iter := runner.Run(ctx, stripSystemMessage(input.Messages))

	var replyBuilder strings.Builder
	var modelStarted bool

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			aiErr := ClassifyAIError(event.Err)
			sendStatus(onStatus, "error", aiErr.UserMessage, string(aiErr.Type), "")
			logger.L().Error("[ADK Agent] 事件流错误",
				zap.String("error_type", string(aiErr.Type)),
				zap.Error(event.Err),
			)
			return "", state.ReadMetadata(), event.Err
		}

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput

		// Tool response events are handled by the middleware; skip here.
		if mv.Role == schema.Tool {
			continue
		}

		if mv.IsStreaming && mv.MessageStream != nil {
			for {
				chunk, chunkErr := mv.MessageStream.Recv()
				if chunkErr == io.EOF {
					break
				}
				if chunkErr != nil {
					if ctx.Err() != nil {
						return replyBuilder.String(), state.ReadMetadata(), ctx.Err()
					}
					logger.L().Warn("[ADK Agent] 流式消息接收错误", zap.Error(chunkErr))
					break
				}
				if chunk == nil {
					continue
				}
				if !modelStarted && chunk.Content != "" && len(chunk.ToolCalls) == 0 {
					modelStarted = true
					sendStatus(onStatus, "generating", "正在生成回答...", "", "")
				}
				if chunk.Content != "" && len(chunk.ToolCalls) == 0 {
					replyBuilder.WriteString(chunk.Content)
					if onDelta != nil {
						if err := onDelta(chunk.Content); err != nil {
							return replyBuilder.String(), state.ReadMetadata(), err
						}
					}
				}
			}
		} else if mv.Message != nil {
			if mv.Message.Content != "" && len(mv.Message.ToolCalls) == 0 {
				if !modelStarted {
					modelStarted = true
					sendStatus(onStatus, "generating", "正在生成回答...", "", "")
				}
				replyBuilder.WriteString(mv.Message.Content)
				if onDelta != nil {
					if err := onDelta(mv.Message.Content); err != nil {
						return replyBuilder.String(), state.ReadMetadata(), err
					}
				}
			}
		}
	}

	reply := strings.TrimSpace(replyBuilder.String())

	if reply == "" {
		meta := state.ReadMetadata()
		if len(meta.ToolTraces) > 0 {
			fallback := BuildHRFallbackReply(meta.ToolTraces)
			if input.AgentName == "candidate_assistant" {
				fallback = BuildCandidateFallbackReply(meta.ToolTraces)
			}
			sendStatus(onStatus, "partial_done", "已基于已查询数据给出保守回复", string(AIEmptyReply), "")
			if onDelta != nil {
				_ = onDelta(fallback)
			}
			logger.L().Warn("[ADK Agent] 模型空回复，使用工具结果兜底",
				zap.Int("tool_traces", len(meta.ToolTraces)),
				zap.Int("fallback_chars", len([]rune(fallback))),
			)
			return fallback, meta, nil
		}
		sendStatus(onStatus, "error", "未能生成有效回复", string(AIEmptyReply), "")
		return "", state.ReadMetadata(), NewAIError(AIEmptyReply, "", fmt.Errorf("ai returned empty reply"))
	}

	sendStatus(onStatus, "done", "回答完成", "", "")
	meta := state.ReadMetadata()
	logger.L().Info("[ADK Agent] 回复完成",
		zap.Int("reply_chars", len([]rune(reply))),
		zap.Int("tool_traces", len(meta.ToolTraces)),
		zap.Duration("total_cost", time.Since(start)),
	)
	return reply, meta, nil
}

// escapeADKInstruction doubles curly braces in instructions that contain
// JSON examples so ADK's default f-string template rendering doesn't try to
// resolve them as placeholders.
// stripSystemMessage returns a copy of msgs with only the first System message
// removed. ADK's defaultGenModelInput prepends Instruction as the system
// message, so the first system message in the input would double the system
// prompt. Any subsequent System messages (e.g. from context builders like
// long-term memory or compliance rules) are preserved.
func stripSystemMessage(msgs []*schema.Message) []*schema.Message {
	out := make([]*schema.Message, 0, len(msgs))
	skippedFirst := false
	for _, m := range msgs {
		if m.Role == schema.System && !skippedFirst {
			skippedFirst = true
			continue
		}
		out = append(out, m)
	}
	return out
}

func escapeADKInstruction(instruction string) string {
	instruction = strings.ReplaceAll(instruction, "{", "{{")
	instruction = strings.ReplaceAll(instruction, "}", "}}")
	return instruction
}

// ModelClient exposes the underlying BaseChatModel for ADK agent construction.
func (c *Client) ModelClient() model.BaseChatModel {
	return c.cm
}
