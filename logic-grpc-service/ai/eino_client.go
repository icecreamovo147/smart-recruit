package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	chatmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"

	"logic-grpc-service/pkg/logger"
)

// ToolRunner is the interface that both HR and candidate tool executors implement.
type ToolRunner interface {
	Execute(ctx context.Context, hrID int64, toolName string, args map[string]any) (ToolResult, error)
}

type Client struct {
	model   string
	cm      *openai.ChatModel
	timeout time.Duration
	sem     chan struct{}
	breaker *CircuitBreaker
}

type Options struct {
	Timeout                 time.Duration
	MaxConcurrency          int
	CircuitFailureThreshold int
	CircuitOpenTimeout      time.Duration
	HalfOpenMaxRequests     int
}

type RecruitingStats struct {
	TotalApplications int64
	TodayApplications int64
	HotJobs           []string
}

type ApplicationAnalysisInput struct {
	Question       string
	JobTitle       string
	Department     string
	Location       string
	SalaryRange    string
	Description    string
	Requirements   string
	StatusText     string
	RoundNo        int32
	ResumeFileName string
	ResumeTextNote string
	ResumeText     string
}

func NewClient(ctx context.Context, apiKey, model, baseURL string, opts ...Options) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("ai api_key is empty")
	}
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("ai model is empty")
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	opt := Options{
		Timeout:                 45 * time.Second,
		MaxConcurrency:          10,
		CircuitFailureThreshold: 5,
		CircuitOpenTimeout:      30 * time.Second,
		HalfOpenMaxRequests:     2,
	}
	if len(opts) > 0 {
		if opts[0].Timeout > 0 {
			opt.Timeout = opts[0].Timeout
		}
		if opts[0].MaxConcurrency > 0 {
			opt.MaxConcurrency = opts[0].MaxConcurrency
		}
		if opts[0].CircuitFailureThreshold > 0 {
			opt.CircuitFailureThreshold = opts[0].CircuitFailureThreshold
		}
		if opts[0].CircuitOpenTimeout > 0 {
			opt.CircuitOpenTimeout = opts[0].CircuitOpenTimeout
		}
		if opts[0].HalfOpenMaxRequests > 0 {
			opt.HalfOpenMaxRequests = opts[0].HalfOpenMaxRequests
		}
	}
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
		Timeout: opt.Timeout,
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		model:   model,
		cm:      cm,
		timeout: opt.Timeout,
		sem:     make(chan struct{}, opt.MaxConcurrency),
		breaker: NewCircuitBreaker(opt.CircuitFailureThreshold, opt.CircuitOpenTimeout, opt.HalfOpenMaxRequests),
	}, nil
}

func (c *Client) call(ctx context.Context, fn func(context.Context) error) error {
	if err := c.breaker.BeforeCall(); err != nil {
		return err
	}
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return ctx.Err()
	}

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	err := fn(callCtx)
	c.breaker.AfterCall(err)
	return err
}

// GenerateRecruitingReply answers HR questions using recruiting statistics.
// When onDelta is non-nil, uses streaming mode and calls onDelta for each chunk.
func (c *Client) GenerateRecruitingReply(ctx context.Context, question string, stats RecruitingStats, onDelta func(string) error) (string, error) {
	if c.cm == nil {
		return "", fmt.Errorf("ai chat model is nil")
	}
	start := time.Now()
	msgs := buildRecruitingMessages(question, stats)
	promptLen := msgCharCount(msgs)

	if onDelta != nil {
		var reply string
		err := c.call(ctx, func(callCtx context.Context) error {
			var streamErr error
			reply, streamErr = c.stream(callCtx, msgs, onDelta)
			return streamErr
		})
		logger.L().Info("ai recruiting stream done",
			zap.String("model", c.model),
			zap.Bool("stream", true),
			zap.Int("prompt_chars", promptLen),
			zap.Int("reply_chars", len([]rune(reply))),
			zap.Duration("cost", time.Since(start)),
			zap.Error(err),
		)
		return reply, err
	}
	var resp *schema.Message
	err := c.call(ctx, func(callCtx context.Context) error {
		var genErr error
		resp, genErr = c.cm.Generate(callCtx, msgs)
		return genErr
	})
	if err != nil {
		logger.L().Error("ai recruiting call failed",
			zap.String("model", c.model),
			zap.Int("prompt_chars", promptLen),
			zap.Duration("cost", time.Since(start)),
			zap.Error(err),
		)
		return "", err
	}
	if strings.TrimSpace(resp.Content) == "" {
		return "", fmt.Errorf("ai returned empty reply")
	}
	logger.L().Info("ai recruiting call done",
		zap.String("model", c.model),
		zap.Bool("stream", false),
		zap.Int("prompt_chars", promptLen),
		zap.Int("reply_chars", len([]rune(resp.Content))),
		zap.Duration("cost", time.Since(start)),
	)
	return resp.Content, nil
}

// GenerateApplicationAnalysis analyzes a candidate's resume against a job posting.
// When onDelta is non-nil, uses streaming mode and calls onDelta for each chunk.
func (c *Client) GenerateApplicationAnalysis(ctx context.Context, input ApplicationAnalysisInput, onDelta func(string) error) (string, error) {
	if c.cm == nil {
		return "", fmt.Errorf("ai chat model is nil")
	}
	start := time.Now()
	msgs := buildApplicationAnalysisMessages(input)
	promptLen := msgCharCount(msgs)

	if onDelta != nil {
		var reply string
		err := c.call(ctx, func(callCtx context.Context) error {
			var streamErr error
			reply, streamErr = c.stream(callCtx, msgs, onDelta)
			return streamErr
		})
		logger.L().Info("ai analysis stream done",
			zap.String("model", c.model),
			zap.Bool("stream", true),
			zap.Int("prompt_chars", promptLen),
			zap.Int("reply_chars", len([]rune(reply))),
			zap.Duration("cost", time.Since(start)),
			zap.Error(err),
		)
		return reply, err
	}
	var resp *schema.Message
	err := c.call(ctx, func(callCtx context.Context) error {
		var genErr error
		resp, genErr = c.cm.Generate(callCtx, msgs)
		return genErr
	})
	if err != nil {
		logger.L().Error("ai analysis call failed",
			zap.String("model", c.model),
			zap.Int("prompt_chars", promptLen),
			zap.Duration("cost", time.Since(start)),
			zap.Error(err),
		)
		return "", err
	}
	if strings.TrimSpace(resp.Content) == "" {
		return "", fmt.Errorf("ai returned empty reply")
	}
	logger.L().Info("ai analysis call done",
		zap.String("model", c.model),
		zap.Bool("stream", false),
		zap.Int("prompt_chars", promptLen),
		zap.Int("reply_chars", len([]rune(resp.Content))),
		zap.Duration("cost", time.Since(start)),
	)
	return resp.Content, nil
}

// GenerateCandidateSuggestedQuestions proposes exactly three safe follow-up
// questions for a candidate after an assistant reply.
func (c *Client) GenerateCandidateSuggestedQuestions(ctx context.Context, userMessage, assistantReply string) ([]string, error) {
	if c.cm == nil {
		return nil, fmt.Errorf("ai chat model is nil")
	}
	msgs := []*schema.Message{
		schema.SystemMessage(`你是招聘系统候选人端 AI 助手的后续问题生成器。
请根据候选人刚刚的问题和 AI 助手回复，生成 3 个候选人下一步可能会问的问题。
要求：
1. 只返回 JSON 字符串数组，不要 Markdown，不要解释。
2. 数组必须恰好 3 个字符串。
3. 每个问题要短、自然、具体，可直接作为下一轮用户消息。
4. 不要诱导查看 HR 内部评价、其他候选人信息。
5. 不要让 AI 直接投递岗位，投递只能由用户点击按钮完成。
6. 不要诱导编造简历经历、技能、学校、公司、项目或证书。`),
		schema.UserMessage(fmt.Sprintf("候选人问题：%s\n\nAI回复：%s", userMessage, assistantReply)),
	}
	var resp *schema.Message
	err := c.call(ctx, func(callCtx context.Context) error {
		var genErr error
		resp, genErr = c.cm.Generate(callCtx, msgs)
		return genErr
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || strings.TrimSpace(resp.Content) == "" {
		return nil, fmt.Errorf("ai returned empty suggested questions")
	}
	questions := parseSuggestedQuestions(resp.Content)
	if len(questions) != 3 {
		return nil, fmt.Errorf("ai returned %d suggested questions", len(questions))
	}
	return questions, nil
}

func parseSuggestedQuestions(content string) []string {
	raw := strings.TrimSpace(content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	if start := strings.Index(raw, "["); start >= 0 {
		if end := strings.LastIndex(raw, "]"); end >= start {
			raw = raw[start : end+1]
		}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	result := make([]string, 0, 3)
	seen := map[string]struct{}{}
	for _, item := range items {
		q := strings.TrimSpace(item)
		if q == "" {
			continue
		}
		runes := []rune(q)
		if len(runes) > 40 {
			q = string(runes[:40])
		}
		if _, ok := seen[q]; ok {
			continue
		}
		seen[q] = struct{}{}
		result = append(result, q)
		if len(result) == 3 {
			break
		}
	}
	return result
}

// ToolTraceCallback is invoked after each tool execution for audit/logging.
// Parameters: toolCallID, toolName, argumentsJSON, resultContent, execErr (nil on success).
type ToolTraceCallback func(toolCallID, toolName, argsJSON, resultContent string, execErr error)

// isContextCanceled returns true when the error is due to context cancellation
// (user abort or connection drop) vs deadline exceeded (timeout).
func isContextCanceled(err error) bool {
	return errors.Is(err, context.Canceled)
}

// ChatWithTools sends messages with tool definitions to the LLM. The LLM may
// decide to call tools (function calling) to query real data from MySQL. This
// method loops: LLM → tool calls → execute → feedback → LLM again, up to 5 rounds.
// When the LLM returns a text answer instead of tool calls, it streams the reply
// via onDelta and returns the full text.
// onToolExecuted is an optional callback invoked after each tool execution for trace recording.
func (c *Client) ChatWithTools(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo, executor ToolRunner, hrID int64, onDelta func(string) error, onToolExecuted ToolTraceCallback) (string, ToolMetadata, error) {
	if c.cm == nil {
		return "", ToolMetadata{}, fmt.Errorf("ai chat model is nil")
	}
	start := time.Now()
	round := 0
	const maxRounds = 5
	var metadata ToolMetadata

	for {
		round++
		select {
		case <-ctx.Done():
			return "", metadata, ctx.Err()
		default:
		}

		logger.L().Info("[AI意图] 询问LLM，等待决策...",
			zap.Int("round", round),
		)

		toolModel, err := c.cm.WithTools(tools)
		if err != nil {
			return "", metadata, fmt.Errorf("bind tools: %w", err)
		}

		var resp *schema.Message
		err = c.call(ctx, func(callCtx context.Context) error {
			var streamErr error
			resp, streamErr = c.streamToolModel(callCtx, toolModel, messages, onDelta)
			return streamErr
		})
		if err != nil {
			if isContextCanceled(err) {
				logger.L().Info("[AI意图] LLM调用被取消（用户中断或连接断开）",
					zap.Int("round", round),
					zap.Error(err),
				)
			} else {
				logger.L().Error("[AI意图] LLM调用失败",
					zap.Int("round", round),
					zap.Error(err),
				)
			}
			return "", metadata, err
		}

		// LLM decided to call one or more tools.
		if len(resp.ToolCalls) > 0 {
			if round >= maxRounds {
				logger.L().Warn("[AI意图] 工具调用达到上限，强制基于已有结果回复",
					zap.Int("round", round),
					zap.Int("max_rounds", maxRounds),
					zap.Int("tool_count", len(resp.ToolCalls)),
					zap.Duration("elapsed", time.Since(start)),
				)
				return c.finalAnswerWithoutTools(ctx, messages, metadata, onDelta, start, round)
			}
			messages = append(messages, resp)

			toolNames := make([]string, 0, len(resp.ToolCalls))
			for _, tc := range resp.ToolCalls {
				toolNames = append(toolNames, tc.Function.Name)
			}
			logger.L().Info("[AI意图] LLM决定调用工具查询数据",
				zap.Int("round", round),
				zap.Int("tool_count", len(resp.ToolCalls)),
				zap.Strings("tools", toolNames),
			)

			for _, tc := range resp.ToolCalls {
				var args map[string]any
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					args = map[string]any{}
				}
				logger.L().Info("[AI意图]   执行工具",
					zap.String("tool", tc.Function.Name),
					zap.Any("args", args),
				)
				result, execErr := executor.Execute(ctx, hrID, tc.Function.Name, args)
				if execErr != nil {
					data, _ := json.Marshal(map[string]string{"error": execErr.Error()})
					result = ToolResult{Content: string(data)}
				}
				if onToolExecuted != nil {
					onToolExecuted(tc.ID, tc.Function.Name, tc.Function.Arguments, result.Content, execErr)
				}
				metadata.merge(result.Metadata)
				messages = append(messages, schema.ToolMessage(result.Content, tc.ID, schema.WithToolName(tc.Function.Name)))
				logger.L().Info("[AI意图]   工具返回",
					zap.String("tool", tc.Function.Name),
					zap.String("result", result.Content),
				)
			}
			logger.L().Info("[AI意图] 将工具结果反馈给LLM，继续下一轮...")
			continue
		}

		// No tool calls — LLM is answering the user.
		reply := strings.TrimSpace(resp.Content)
		if reply == "" {
			return "", metadata, fmt.Errorf("ai returned empty reply")
		}
		logger.L().Info("[AI意图] LLM决定直接回复（不再需要工具）",
			zap.String("reply", reply),
			zap.Int("total_rounds", round),
			zap.Int("reply_chars", len([]rune(reply))),
			zap.Duration("total_cost", time.Since(start)),
		)
		return reply, metadata, nil
	}
}

func (c *Client) finalAnswerWithoutTools(ctx context.Context, messages []*schema.Message, metadata ToolMetadata, onDelta func(string) error, start time.Time, round int) (string, ToolMetadata, error) {
	messages = append(messages, schema.SystemMessage("工具调用轮次已达到上限。请停止调用工具，必须仅基于当前对话和已经返回的工具结果直接回答用户；如果信息仍不足，请说明已查询到的信息和需要用户补充的具体条件。"))

	var reply string
	err := c.call(ctx, func(callCtx context.Context) error {
		var streamErr error
		reply, streamErr = c.stream(callCtx, messages, onDelta)
		return streamErr
	})
	if err != nil {
		return "", metadata, err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "", metadata, fmt.Errorf("ai returned empty reply after tool round limit")
	}
	logger.L().Info("[AI意图] 工具上限兜底回复完成",
		zap.String("reply", reply),
		zap.Int("total_rounds", round),
		zap.Int("reply_chars", len([]rune(reply))),
		zap.Duration("total_cost", time.Since(start)),
	)
	return reply, metadata, nil
}

func (c *Client) streamToolModel(ctx context.Context, toolModel chatmodel.ToolCallingChatModel, messages []*schema.Message, onDelta func(string) error) (*schema.Message, error) {
	streamStart := time.Now()
	stream, err := toolModel.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	chunks := make([]*schema.Message, 0, 16)
	chunkCount := 0
	firstChunkAt := time.Time{}
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if ctx.Err() != nil || isContextCanceled(err) {
				return nil, err
			}
			if msg, ok := partialToolStreamMessage(chunks); ok {
				logger.L().Warn("ai tool stream interrupted after partial reply; using partial content",
					zap.Error(err),
					zap.Int("chunks", chunkCount),
					zap.Int("reply_chars", len([]rune(strings.TrimSpace(msg.Content)))),
					zap.Duration("elapsed", time.Since(streamStart)),
				)
				return msg, nil
			}
			return nil, err
		}
		if chunk == nil {
			continue
		}
		if firstChunkAt.IsZero() {
			firstChunkAt = time.Now()
		}
		chunkCount++
		chunks = append(chunks, chunk)
		if chunk.Content != "" && len(chunk.ToolCalls) == 0 && onDelta != nil {
			if err := onDelta(chunk.Content); err != nil {
				return nil, err
			}
		}
	}
	msg, err := schema.ConcatMessages(chunks)
	if err != nil {
		return nil, err
	}
	ttfb := time.Duration(0)
	if !firstChunkAt.IsZero() {
		ttfb = firstChunkAt.Sub(streamStart)
	}
	logger.L().Info("ai tool stream details",
		zap.Duration("ttfb", ttfb),
		zap.Int("chunks", chunkCount),
		zap.Int("tool_calls", len(msg.ToolCalls)),
		zap.Duration("total_stream_cost", time.Since(streamStart)),
	)
	return msg, nil
}

func partialToolStreamMessage(chunks []*schema.Message) (*schema.Message, bool) {
	if len(chunks) == 0 {
		return nil, false
	}
	msg, err := schema.ConcatMessages(chunks)
	if err != nil || msg == nil {
		return nil, false
	}
	if len(msg.ToolCalls) > 0 {
		return nil, false
	}
	if strings.TrimSpace(msg.Content) == "" {
		return nil, false
	}
	return msg, true
}

// GenerateSessionSummary creates or updates a rolling session summary by feeding the
// previous summary and new messages to the LLM.
func (c *Client) GenerateSessionSummary(ctx context.Context, oldSummary string, recentMessages []string) (string, error) {
	if c.cm == nil {
		return "", fmt.Errorf("ai chat model is nil")
	}
	msgText := ""
	for i, m := range recentMessages {
		msgText += fmt.Sprintf("[%d] %s\n", i+1, m)
	}
	prompt := fmt.Sprintf(
		`你是一个会话摘要助手。请根据之前的摘要和新增的对话消息，更新会话摘要。

要求：
- 保留关键事实：候选人姓名、岗位名称、分析结论、HR 偏好、待办事项
- 去除重复寒暄和临时错误信息
- 使用简洁的中文
- 如果之前没有摘要，直接根据新消息生成摘要

之前的摘要：
%s

新增对话消息：
%s

请输出更新后的摘要：`, oldSummary, msgText)

	msgs := []*schema.Message{
		schema.SystemMessage("你是一个会话摘要助手。请根据对话生成简洁的摘要。"),
		schema.UserMessage(prompt),
	}
	var resp *schema.Message
	err := c.call(ctx, func(callCtx context.Context) error {
		var genErr error
		resp, genErr = c.cm.Generate(callCtx, msgs)
		return genErr
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

func (c *Client) stream(ctx context.Context, messages []*schema.Message, onDelta func(string) error) (string, error) {
	streamStart := time.Now()
	stream, err := c.cm.Stream(ctx, messages)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var builder strings.Builder
	chunkCount := 0
	firstChunkAt := time.Time{}
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return builder.String(), err
		}
		if chunk == nil || chunk.Content == "" {
			continue
		}
		if firstChunkAt.IsZero() {
			firstChunkAt = time.Now()
		}
		chunkCount++
		builder.WriteString(chunk.Content)
		if onDelta != nil {
			if err := onDelta(chunk.Content); err != nil {
				return builder.String(), err
			}
		}
	}
	reply := builder.String()
	if strings.TrimSpace(reply) == "" {
		return "", fmt.Errorf("ai returned empty reply")
	}
	ttfb := time.Duration(0)
	if !firstChunkAt.IsZero() {
		ttfb = firstChunkAt.Sub(streamStart)
	}
	logger.L().Info("ai stream details",
		zap.Duration("ttfb", ttfb),
		zap.Int("chunks", chunkCount),
		zap.Duration("total_stream_cost", time.Since(streamStart)),
	)
	return reply, nil
}

// msgCharCount estimates total character count across all messages.
func msgCharCount(msgs []*schema.Message) int {
	n := 0
	for _, m := range msgs {
		n += len([]rune(m.Content))
	}
	return n
}

func buildRecruitingMessages(question string, stats RecruitingStats) []*schema.Message {
	return []*schema.Message{
		schema.SystemMessage("你是智能招聘系统的数据分析助手。你必须基于系统提供的真实招聘统计数据回答 HR 的问题，不要编造不存在的数据。你只能回答与招聘系统相关的问题，如果用户询问无关内容，必须礼貌拒绝并引导回到招聘话题。回答要简洁、专业、中文输出。"),
		schema.UserMessage(fmt.Sprintf(`HR 问题：%s

系统实时统计数据：
- 当前累计投递数：%d
- 今日新增投递数：%d
- 热门岗位排行：%s

请根据这些数据回答 HR。`, question, stats.TotalApplications, stats.TodayApplications, strings.Join(stats.HotJobs, "、"))),
	}
}

func buildApplicationAnalysisMessages(input ApplicationAnalysisInput) []*schema.Message {
	resumeText := strings.TrimSpace(input.ResumeText)
	if resumeText == "" {
		resumeText = "简历文本暂未解析成功。请明确说明当前无法基于 PDF 简历正文充分评估候选人，不要使用候选人在系统资料页填写的信息进行补充判断。"
	}
	question := strings.TrimSpace(input.Question)
	if question == "" {
		question = "请分析该候选人与投递岗位的匹配程度，并给出是否建议通过的判断。"
	}
	return []*schema.Message{
		schema.SystemMessage("你是智能招聘系统的简历评估助手。你必须以 PDF 简历中提取出来的文字内容为主要依据，并结合岗位信息进行分析；不得使用候选人在系统资料页填写的信息，也不得编造经历。你提供的是辅助建议，最终决策权属于 HR。输出中文，结构清晰，精炼输出。匹配点和风险点各不超过 3 条，每条控制在 2 句话以内。结论和建议合并为一段，不超过 4 句话。"),
		schema.UserMessage(fmt.Sprintf(`HR 问题：%s

投递信息：
- 当前状态：%s
- 投递轮次：第 %d 轮

岗位信息：
- 岗位：%s
- 部门：%s
- 地点：%s
- 薪资：%s
- 岗位描述：%s
- 任职要求：%s

PDF 简历解析状态：
%s

PDF 简历正文：
%s

请以 PDF 简历正文为主、结合岗位要求，给出专业、谨慎的分析结果。`, question, input.StatusText, input.RoundNo, input.JobTitle, input.Department, input.Location, input.SalaryRange, input.Description, input.Requirements, input.ResumeTextNote, resumeText)),
	}
}
