package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonsai "smart-recruit-commons/ai"
	"smart-recruit-proto/recruitment/pb"
)

const maxCandidateContextMessages = 20

// DEV-parity candidate system prompt (tool-calling agent, not context stuffing).
const candidateADKSystemPrompt = `你是智能招聘系统的候选人 AI 助手，只服务当前登录候选人。
你可以帮助候选人了解本人投递进度、基于本人简历推荐在招岗位、给出简历优化建议。
你只能回答与招聘求职相关的问题，如果用户询问无关内容（如产品评测、技术算法等），必须礼貌拒绝并引导回到招聘话题。
你只能基于工具返回的数据作答，不得访问或推测其他候选人、HR 内部评价、候选人不可见的岗位信息。
不得承诺录用结果，不得编造候选人简历中不存在的经历。
推荐岗位时可以覆盖系统内所有岗位；如果工具返回 has_applied=true，必须说明”已投递”。
当简历缺失或解析失败时，明确提示用户先上传简历，不得改用候选人资料页替代。
简历优化只输出建议，不输出改写后的简历段落。
你不能替用户投递岗位，也不能声称已经投递；投递只能由前端按钮和用户二次确认完成。

## 回复风格要求
- 回复内容整体保持精炼，每个要点控制在 1-2 句话，避免长篇展开。
- 简历优化建议以清单式输出，每条建议一行（用 - 开头），只写优化方向不写详细论证。
- 岗位推荐每个岗位用 2-3 句话简要说明核心匹配理由，不要展开项目细节。
- 不需要在正文末尾追加客套话（如"祝求职顺利"之类）。

## 回复格式要求
你的回复会以 Markdown 渲染展示给候选人，请严格遵循以下排版规范：

1. 总体结构：先给出一个简短的总体概括（1-2 句），再用结构化方式展开细节。
2. 多条同类信息（投递记录、岗位列表、面试轮次等）必须用 Markdown 无序列表（- 开头）逐条列出，每行一条。禁止把多条记录拼成一行纯文本。
3. 每条记录内用粗体（**文字**）标出最关键的信息，如投递状态、岗位名称、时间等。
4. 如果涉及时间线或先后顺序，按时间倒序排列（最新的在上）。
5. 用二级标题（## 标题）为不同话题分区，一个话题一个区块。
6. 段落之间留空行，保持视觉透气感。

Markdown 输出硬性规范：
你的正文回复会交给标准 CommonMark/Markdown 渲染器展示，必须严格输出合法 Markdown，不要依赖前端容错修正。
1. 列表项必须写成 "- 内容"，短横线后必须有 1 个空格。禁止写成 "-内容"、"-**标题**"。
2. 粗体必须写成 "**文字**"，星号内侧不能有空格。禁止写成 "** 文字**"、"**文字 **"。
3. 粗体前后如果紧贴普通文字、数字、日期或中文标点，必须补空格或自然分隔。
4. 标签式字段必须写成 "**字段：** 内容"，冒号后的正文前保留 1 个空格。
5. 标题必须写成 "## 标题"，井号后必须有空格；标题前后各空一行。
6. 多条记录必须使用 Markdown 列表逐条输出，每条记录一行。
7. 段落、标题、列表之间使用空行分隔；不要输出 HTML 标签。
8. 只有真正的代码、SQL、JSON、命令行内容才允许使用三反引号代码块；面向候选人阅读的建议、示例、清单、步骤、简历条目必须直接用 Markdown 列表/标题展示，禁止包进代码块。
9. 不要用代码块模拟排版，不要用缩进模拟列表；列表层级只能用 Markdown 列表缩进表达。

示例——当候选人询问投递进度时，应输出：

你的投递记录共 3 条，最新状态如下：

## 投递进度

- **后台开发实习生** — 2026-05-14 **淘汰**（第 1 轮面试）
- **前端开发实习生** — 2026-05-13 **待查看**
- **产品助理** — 2026-05-10 **已通过**（第 2 轮面试）

每次正文回复结束后，必须追加一段仅供系统解析的后续问题 JSON 标记，格式严格如下：
<<<CANDIDATE_SUGGESTED_QUESTIONS_JSON>>>
["问题1","问题2","问题3"]
<<<END_CANDIDATE_SUGGESTED_QUESTIONS_JSON>>>

后续问题要求：
1. 必须恰好 3 个，基于本次候选人的问题和你的当前回复生成。
2. 问题要短、自然、具体，像候选人下一步最可能直接追问的话。
3. 不要与当前回复末尾正文混写，不要在正文里额外写”你还可以问”。
4. 不得引导越权查看 HR 内部评价、他人信息，不得诱导 AI 直接投递或编造简历。`

type candidateAgentRuntimeConfig struct {
	HasConfig           bool
	ToolNames           []string
	MaxIterations       int
	TemperatureOverride *float64
	SystemPrompt        string
}

type recentChatMessageStore interface {
	ListRecentChatMessages(ctx context.Context, ownerRole int32, ownerID, sessionID int64, limit int32) ([]ChatMessageRow, error)
}

func (s *nativeAIService) CandidateChatStream(req *pb.CandidateChatRequest, stream gogrpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	return s.runCandidateChatRuntime(req, stream)
}

func (s *nativeAIService) runCandidateChatRuntime(req *pb.CandidateChatRequest, stream gogrpc.ServerStreamingServer[pb.ChatStreamResponse]) error {
	ctx := stream.Context()
	if req == nil || strings.TrimSpace(req.GetMessage()) == "" {
		return stream.Send(&pb.ChatStreamResponse{
			Code: agentRunCodeBadRequest, Msg: candidateChatEmptyMessageError, Done: true,
			CreatedAt: formatTime(time.Now()), EventType: "done",
		})
	}
	if s.store == nil {
		return errAIStoreRequired
	}
	if req.GetUserId() <= 0 {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	startedAt := time.Now()
	inputChars := len([]rune(req.GetMessage()))
	session, err := s.ensureSessionWithOptions(ctx, ownerRoleCandidate, req.GetUserId(), req.GetSessionId(), 0, req.GetMessage(), ChatSessionCreateOptions{
		SessionType: req.GetSessionType(),
		SourceType:  req.GetSourceType(),
		SourceID:    req.GetSourceId(),
		SourceTitle: req.GetSourceTitle(),
	})
	if err != nil {
		return err
	}
	if _, err := s.store.AppendChatMessage(ctx, ChatMessageRow{
		OwnerRole: ownerRoleCandidate, OwnerID: req.GetUserId(), SessionID: session.ID,
		Role: "user", Content: req.GetMessage(),
	}); err != nil {
		return err
	}

	runtimeModel, err := s.resolveCapabilityRuntimeModel(ctx, billingOwnerUser, req.GetUserId(), "ai.chat", platformAIAudienceCandidate, req.GetModelId())
	if err != nil {
		s.persistCandidateChatFailure(ctx, req.GetUserId(), session.ID, err)
		return err
	}
	modelID, modelName, providerName := runtimeModel.ID, runtimeModel.Name, runtimeModel.ProviderName
	req.ModelId = modelID
	auditOpts := candidateUsageAuditOptions{Provider: providerName, Model: modelName}
	ctx, err = s.reserveAIBilling(ctx, billingOwnerUser, req.GetUserId(), "ai.chat", "candidate_chat", providerName, modelName, inputChars, runtimeModel)
	if err != nil {
		s.persistCandidateChatFailure(ctx, req.GetUserId(), session.ID, err)
		return err
	}
	defer s.cancelUnsettledBilling(ctx, "runtime_completed_without_usage")

	modelInfoUsage := &pb.ContextUsageInfo{
		ModelId:                modelID,
		ModelName:              modelName,
		ContextWindowTokens:    runtimeModel.ContextWindowTokens,
		MaxOutputTokens:        runtimeModel.MaxOutputTokens,
		RequestedModelId:       runtimeModel.RequestedModelID,
		EffectiveModelId:       modelID,
		ModelFallbackReason:    runtimeModel.FallbackReason,
		CapabilityVersionId:    runtimeModel.CapabilityVersionID,
		CapabilitySnapshotHash: runtimeModel.CapabilitySnapshotHash,
		Estimated:              true,
		Source:                 "candidate-agent-runtime",
		Stage:                  "model_selected",
	}
	if err := stream.Send(&pb.ChatStreamResponse{
		Code: 0, Msg: "success", EventType: "model_info", ContextUsage: modelInfoUsage,
		SessionId: session.ID, CreatedAt: formatTime(time.Now()),
	}); err != nil {
		return err
	}

	var partialReply strings.Builder
	streamFilter := newCandidateSuggestionStreamFilter(func(delta string) error {
		partialReply.WriteString(delta)
		return stream.Send(&pb.ChatStreamResponse{
			Code: 0, Msg: "success", Delta: delta, SessionId: session.ID,
			CreatedAt: formatTime(time.Now()), EventType: "generating", EventMessage: "streaming answer",
		})
	})
	statusSender := func(eventType, eventMessage, errorType, toolName string) error {
		return stream.Send(&pb.ChatStreamResponse{
			Code: 0, Msg: "success", EventType: eventType, EventMessage: eventMessage,
			ErrorType: errorType, ToolName: toolName, SessionId: session.ID,
			CreatedAt: formatTime(time.Now()),
		})
	}

	runtimeCfg := s.getCandidateAgentRuntimeConfigForRelease(ctx, runtimeModel.ConfigurationRefs)
	if runtimeModel.CapabilityVersionID > 0 && (!runtimeCfg.HasConfig || strings.TrimSpace(runtimeCfg.SystemPrompt) == "") {
		return status.Error(codes.FailedPrecondition, "candidate Agent or Prompt is unavailable in the capability release")
	}
	systemPrompt := s.resolveCandidateAgentSystemPromptForRelease(ctx, runtimeCfg, runtimeModel.ConfigurationRefs)
	availablePlanTools := append([]string(nil), runtimeCfg.ToolNames...)
	if len(availablePlanTools) == 0 {
		availablePlanTools = commonsai.CandidateToolNames()
	}
	intentPlan := commonsai.NewCandidateAssistantPlanner().Plan(commonsai.CandidateAssistantPlannerInput{
		Message:        req.GetMessage(),
		AvailableTools: availablePlanTools,
	})

	if suffix := intentPlan.InstructionBlock(); suffix != "" {
		systemPrompt += "\n\n" + suffix
	}

	var reply string
	var metadata commonsai.ToolMetadata
	var execErr error
	legacyFallback := false
	useADK := s.effectiveAgentRuntime() == agentRuntimeADK
	executor := s.candidateTools

	if useADK && executor != nil {
		messages, buildErr := s.buildCandidateAgentMessages(ctx, req.GetUserId(), session.ID, req.GetMessage(), systemPrompt)
		if buildErr != nil {
			return buildErr
		}
		adkTools, toolErr := s.getOrInitCandidateADKTools()
		if toolErr != nil {
			legacyFallback = true
		} else {
			if len(runtimeCfg.ToolNames) > 0 {
				adkTools = commonsai.FilterCandidateADKToolsByName(adkTools, runtimeCfg.ToolNames)
			}
			if intentPlan.DisallowsModelTools() {
				adkTools = nil
			} else {
				adkTools = commonsai.FilterCandidateADKToolsByName(adkTools, intentPlan.RequiredTools)
			}
			adkProvider, hasADK := s.provider.(RecruitingADKChatProvider)
			if !hasADK {
				legacyFallback = true
			} else {
				state := &commonsai.AgentRunState{}
				adkCtx := commonsai.WithOwnerID(ctx, req.GetUserId())
				adkCtx = commonsai.WithAgentRunState(adkCtx, state)
				opts := ChatCompletionOptions{
					MaxIterations:       runtimeCfg.MaxIterations,
					TemperatureOverride: runtimeCfg.TemperatureOverride,
				}
				traceFn := func(toolCallID, toolName, argsJSON, resultContent string, duration time.Duration, toolErr error) {
					s.recordCandidateToolTrace(session.ID, req.GetUserId(), toolCallID, toolName, argsJSON, resultContent, duration, toolErr)
				}
				reply, metadata, execErr = adkProvider.ChatWithRecruitingADK(
					adkCtx,
					modelID,
					opts,
					commonsai.AgentRunInput{
						AgentName:     candidateAssistantAgentType,
						Instruction:   extractCandidateSystemInstruction(messages),
						Messages:      messages,
						Tools:         adkTools,
						MaxIterations: runtimeCfg.MaxIterations,
						OwnerID:       req.GetUserId(),
						SessionID:     session.ID,
						State:         state,
					},
					streamFilter.Write,
					traceFn,
					statusSender,
				)
			}
		}
	} else if useADK && executor == nil {
		legacyFallback = true
	}

	if legacyFallback || !useADK {
		if executor == nil {
			return errAIProviderRequired
		}
		toolProvider, hasToolProvider := s.provider.(RecruitingToolChatProvider)
		if !hasToolProvider {
			// Last resort: no tool loop — fail closed rather than stuffing context.
			return errAIProviderRequired
		}
		tools := commonsai.CandidateTools()
		if len(runtimeCfg.ToolNames) > 0 {
			tools = commonsai.FilterCandidateToolInfosByName(tools, runtimeCfg.ToolNames)
		}
		if intentPlan.DisallowsModelTools() {
			tools = nil
		} else {
			tools = commonsai.FilterCandidateToolInfosByName(tools, intentPlan.RequiredTools)
		}
		messages := []*schema.Message{
			schema.SystemMessage(systemPrompt),
			schema.UserMessage(req.GetMessage()),
		}
		opts := ChatCompletionOptions{
			MaxIterations:       runtimeCfg.MaxIterations,
			TemperatureOverride: runtimeCfg.TemperatureOverride,
		}
		traceFn := func(toolCallID, toolName, argsJSON, resultContent string, duration time.Duration, toolErr error) {
			s.recordCandidateToolTrace(session.ID, req.GetUserId(), toolCallID, toolName, argsJSON, resultContent, duration, toolErr)
		}
		reply, metadata, execErr = toolProvider.ChatWithRecruitingTools(
			ctx, modelID, opts, messages, tools, executor, req.GetUserId(),
			streamFilter.Write, traceFn, statusSender,
		)
	}

	auditOpts.TokenUsageTotal = tokenUsageTotalFromMeta(metadata.BillingTokenUsage)
	if metadata.BillingTokenUsage != nil {
		auditOpts.PromptTokens = metadata.BillingTokenUsage.PromptTokens
		auditOpts.CompletionTokens = metadata.BillingTokenUsage.CompletionTokens
	}

	if execErr != nil {
		if isCandidateChatCanceled(execErr) {
			s.saveCandidateInterruptedReply(req.GetUserId(), session.ID, partialReply.String())
			_ = s.recordCandidateUsageAudit(ctx, req.GetUserId(), inputChars, len([]rune(partialReply.String())), "timeout", "canceled", int(time.Since(startedAt).Milliseconds()), auditOpts)
			return execErr
		}
		if len(metadata.ToolTraces) > 0 {
			fallback := commonsai.BuildCandidateFallbackReply(metadata.ToolTraces)
			_ = stream.Send(&pb.ChatStreamResponse{
				Code: 0, Msg: "success", EventType: "partial_done",
				EventMessage: "已基于已查询数据给出保守回复", ErrorType: "provider_error",
				SessionId: session.ID, CreatedAt: formatTime(time.Now()),
			})
			if err := stream.Send(&pb.ChatStreamResponse{
				Code: 0, Msg: "success", Delta: fallback, SessionId: session.ID,
				CreatedAt: formatTime(time.Now()),
			}); err != nil {
				return err
			}
			processContent := buildCandidateProcessContent(candidateSuggestedQuestionsFallback())
			if _, saveErr := s.store.AppendChatMessage(ctx, ChatMessageRow{
				OwnerRole: ownerRoleCandidate, OwnerID: req.GetUserId(), SessionID: session.ID,
				Role: "assistant", Content: fallback, ProcessContent: processContent, ModelID: modelID, ModelName: modelName, CreatedAt: time.Now(),
			}); saveErr != nil {
				return saveErr
			}
			_ = s.recordCandidateUsageAudit(ctx, req.GetUserId(), inputChars, len([]rune(fallback)), "error", "fallback", int(time.Since(startedAt).Milliseconds()), auditOpts)
			return stream.Send(&pb.ChatStreamResponse{
				Code: 0, Msg: "success", Done: true, SessionId: session.ID,
				CreatedAt: formatTime(time.Now()), EventType: "done",
				SuggestedQuestions: candidateSuggestedQuestionsFallback(),
			})
		}
		_ = s.recordCandidateUsageAudit(ctx, req.GetUserId(), inputChars, 0, "error", "provider_error", int(time.Since(startedAt).Milliseconds()), auditOpts)
		return execErr
	}

	if err := streamFilter.Finish(); err != nil {
		if isCandidateChatCanceled(err) {
			s.saveCandidateInterruptedReply(req.GetUserId(), session.ID, partialReply.String())
			_ = s.recordCandidateUsageAudit(ctx, req.GetUserId(), inputChars, len([]rune(partialReply.String())), "timeout", "canceled", int(time.Since(startedAt).Milliseconds()), auditOpts)
			return err
		}
		return err
	}

	cleanReply, suggestedQuestions := extractCandidateSuggestedQuestions(reply)
	if strings.TrimSpace(cleanReply) == "" {
		cleanReply = strings.TrimSpace(partialReply.String())
	}
	cleanReply = normalizeCandidateMarkdownReply(cleanReply)
	if len(suggestedQuestions) != 3 {
		suggestedQuestions = intentPlan.SuggestedQuestions
		if len(suggestedQuestions) != 3 {
			suggestedQuestions = candidateSuggestedQuestionsFallback()
		}
	}
	processContent := buildCandidateProcessContent(suggestedQuestions)
	if _, err := s.store.AppendChatMessage(ctx, ChatMessageRow{
		OwnerRole: ownerRoleCandidate, OwnerID: req.GetUserId(), SessionID: session.ID,
		Role: "assistant", Content: cleanReply, ProcessContent: processContent, ModelID: modelID, ModelName: modelName, CreatedAt: time.Now(),
	}); err != nil {
		return err
	}
	if err := s.recordCandidateUsageAudit(ctx, req.GetUserId(), inputChars, len([]rune(cleanReply)), "ok", "", int(time.Since(startedAt).Milliseconds()), auditOpts); err != nil {
		return err
	}
	go s.maybeRefreshCandidateSummary(session.ID, req.GetUserId())
	return stream.Send(&pb.ChatStreamResponse{
		Code: 0, Msg: "success", Done: true, SessionId: session.ID,
		CreatedAt: formatTime(time.Now()), EventType: "done",
		SuggestedQuestions: suggestedQuestions,
	})
}

func (s *nativeAIService) persistCandidateChatFailure(ctx context.Context, userID, sessionID int64, cause error) {
	if s == nil || s.store == nil || userID <= 0 || sessionID <= 0 || cause == nil {
		return
	}
	content, errorCode, retryable := candidateChatFailurePresentation(cause)
	payload, _ := json.Marshal(map[string]any{
		"delivery_status": "failed",
		"error_code":      errorCode,
		"retryable":       retryable,
	})
	_, _ = s.store.AppendChatMessage(ctx, ChatMessageRow{
		OwnerRole: ownerRoleCandidate, OwnerID: userID, SessionID: sessionID,
		Role: "assistant", Content: content, ProcessContent: string(payload), CreatedAt: time.Now(),
	})
}

func candidateChatFailurePresentation(cause error) (content, errorCode string, retryable bool) {
	message := strings.ToLower(status.Convert(cause).Message())
	if status.Code(cause) == codes.ResourceExhausted && strings.Contains(message, "insufficient_credits") {
		return "AI 套餐额度已用完，请购买套餐或加量包后继续使用。", "insufficient_credits", false
	}
	switch status.Code(cause) {
	case codes.Unavailable, codes.DeadlineExceeded:
		return "AI 服务暂时不可用，请稍后重试。", "service_unavailable", true
	case codes.FailedPrecondition:
		return "AI 助手当前配置不可用，请稍后再试。", "capability_unavailable", true
	default:
		return "本次消息发送失败，请稍后重试。", "request_failed", true
	}
}

func (s *nativeAIService) getCandidateAgentRuntimeConfig(ctx context.Context) candidateAgentRuntimeConfig {
	return s.getCandidateAgentRuntimeConfigForRelease(ctx, CapabilityConfigurationRefs{})
}

func (s *nativeAIService) getCandidateAgentRuntimeConfigForRelease(ctx context.Context, refs CapabilityConfigurationRefs) candidateAgentRuntimeConfig {
	cfg := candidateAgentRuntimeConfig{}
	if s == nil || s.store == nil {
		return cfg
	}
	var agent *pb.AgentConfigInfo
	if len(refs.AgentIDs) > 0 {
		if store, ok := s.store.(hrRuntimeAgentByIDStore); ok {
			for _, id := range refs.AgentIDs {
				row, found, err := store.GetRuntimeAgentConfigByID(ctx, id)
				if err == nil && found && row != nil && row.GetIsEnabled() && strings.EqualFold(strings.TrimSpace(row.GetAgentType()), candidateAssistantAgentType) {
					agent = row
					break
				}
			}
		}
	} else if store, ok := s.store.(agentConfigStore); ok {
		resp, err := store.GetAgentConfig(ctx, &pb.GetAgentConfigRequest{AgentType: candidateAssistantAgentType})
		if err == nil && resp != nil && resp.GetCode() == 0 {
			agent = resp.GetAgent()
		}
	}
	if agent == nil || !agent.GetIsEnabled() {
		return cfg
	}
	cfg.HasConfig = true
	if agent.GetMaxIterations() > 0 {
		cfg.MaxIterations = int(agent.GetMaxIterations())
	}
	if v := agent.GetTemperatureOverride(); v > 0 {
		cfg.TemperatureOverride = &v
	}
	for _, b := range agent.GetToolBindings() {
		if b != nil && b.GetIsEnabled() && strings.TrimSpace(b.GetToolName()) != "" {
			cfg.ToolNames = append(cfg.ToolNames, b.GetToolName())
		}
	}
	if agent.GetPromptTemplateId() > 0 {
		if len(refs.PromptTemplateIDs) > 0 && !containsRuntimeID(refs.PromptTemplateIDs, agent.GetPromptTemplateId()) {
			return cfg
		}
		if promptStore, ok := s.store.(interface {
			GetRuntimePromptTemplateByID(context.Context, int64) (*pb.PromptTemplateInfo, bool, error)
		}); ok {
			tmpl, found, perr := promptStore.GetRuntimePromptTemplateByID(ctx, agent.GetPromptTemplateId())
			if perr == nil && found && tmpl != nil {
				if content := strings.TrimSpace(tmpl.GetContent()); content != "" {
					cfg.SystemPrompt = content
				}
			}
		}
	}
	return cfg
}

func (s *nativeAIService) resolveCandidateAgentSystemPrompt(ctx context.Context, runtimeCfg candidateAgentRuntimeConfig) string {
	return s.resolveCandidateAgentSystemPromptForRelease(ctx, runtimeCfg, CapabilityConfigurationRefs{})
}

func (s *nativeAIService) resolveCandidateAgentSystemPromptForRelease(ctx context.Context, runtimeCfg candidateAgentRuntimeConfig, refs CapabilityConfigurationRefs) string {
	if strings.TrimSpace(runtimeCfg.SystemPrompt) != "" {
		return runtimeCfg.SystemPrompt
	}
	if len(refs.PromptTemplateIDs) == 0 {
		if prompt, err := s.resolveCandidateSystemPrompt(ctx); err == nil && strings.TrimSpace(prompt) != "" {
			// Prefer DEV ADK prompt over the short legacy stuffing prompt when only the
			// hardcoded fallback is available.
			if prompt == candidateSystemPrompt {
				return candidateADKSystemPrompt
			}
			return prompt
		}
	}
	return candidateADKSystemPrompt
}

func (s *nativeAIService) buildCandidateAgentMessages(
	ctx context.Context,
	userID, sessionID int64,
	currentMessage, systemPrompt string,
) ([]*schema.Message, error) {
	prompt := strings.TrimSpace(systemPrompt)
	if prompt == "" {
		prompt = candidateADKSystemPrompt
	}
	messages := []*schema.Message{schema.SystemMessage(prompt)}

	history, err := s.listRecentCandidateMessages(ctx, userID, sessionID, maxCandidateContextMessages)
	if err != nil {
		return nil, err
	}
	contextHistory := history[:0]
	for _, item := range history {
		if !candidateChatMessageFailed(item.ProcessContent) {
			contextHistory = append(contextHistory, item)
		}
	}
	history = contextHistory
	currentMsgTrimmed := strings.TrimSpace(currentMessage)
	skipLastMatch := false
	if len(history) > 0 {
		last := history[len(history)-1]
		if last.Role == "user" && strings.TrimSpace(last.Content) == currentMsgTrimmed {
			skipLastMatch = true
		}
	}
	for i, h := range history {
		content := strings.TrimSpace(h.Content)
		if content == "" {
			continue
		}
		if skipLastMatch && i == len(history)-1 && h.Role == "user" && content == currentMsgTrimmed {
			continue
		}
		role := schema.Assistant
		if h.Role == "user" {
			role = schema.User
		}
		messages = append(messages, &schema.Message{Role: role, Content: h.Content})
	}
	if currentMsgTrimmed != "" {
		messages = append(messages, schema.UserMessage(currentMessage))
	}
	return messages, nil
}

func candidateChatMessageFailed(processContent string) bool {
	if strings.TrimSpace(processContent) == "" {
		return false
	}
	var payload struct {
		DeliveryStatus string `json:"delivery_status"`
	}
	return json.Unmarshal([]byte(processContent), &payload) == nil && payload.DeliveryStatus == "failed"
}

func (s *nativeAIService) listRecentCandidateMessages(ctx context.Context, userID, sessionID int64, limit int32) ([]ChatMessageRow, error) {
	if recent, ok := s.store.(recentChatMessageStore); ok {
		return recent.ListRecentChatMessages(ctx, ownerRoleCandidate, userID, sessionID, limit)
	}
	// Fallback: page 1 may only cover the oldest messages if history is long.
	return s.store.ListChatMessages(ctx, ownerRoleCandidate, userID, sessionID, 1, limit)
}

func (s *nativeAIService) getOrInitCandidateADKTools() ([]tool.BaseTool, error) {
	if s == nil {
		return nil, fmt.Errorf("native AI service is nil")
	}
	s.candidateToolsMu.Lock()
	defer s.candidateToolsMu.Unlock()
	if s.cachedCandidateADKTools != nil {
		return s.cachedCandidateADKTools, nil
	}
	if s.candidateTools == nil {
		return nil, fmt.Errorf("candidate tool executor is not configured")
	}
	tools, err := commonsai.NewCandidateADKTools(s.candidateTools)
	if err != nil {
		return nil, err
	}
	s.cachedCandidateADKTools = tools
	return tools, nil
}

// InvalidateCachedCandidateADKTools clears cached ADK tools so the next request
// rebuilds them (DEV parity; use after replacing the executor at runtime).
func (s *nativeAIService) InvalidateCachedCandidateADKTools() {
	if s == nil {
		return
	}
	s.candidateToolsMu.Lock()
	defer s.candidateToolsMu.Unlock()
	s.cachedCandidateADKTools = nil
}

type sessionSummaryStore interface {
	GetSessionSummary(ctx context.Context, ownerID, sessionID int64) (string, bool, error)
	UpsertSessionSummary(ctx context.Context, ownerID, sessionID int64, summary string, coveredMessageID int64, messageCount int) error
}

type sessionSummaryGenerator interface {
	GenerateSessionSummary(ctx context.Context, oldSummary string, recentMessages []string) (string, error)
}

// maybeRefreshCandidateSummary mirrors DEV: refresh rolling summary when the
// session has at least 15 recent messages. Failures are logged and ignored.
func (s *nativeAIService) maybeRefreshCandidateSummary(sessionID, userID int64) {
	defer func() { _ = recover() }()
	if s == nil || s.store == nil || sessionID <= 0 || userID <= 0 {
		return
	}
	summaryStore, ok := s.store.(sessionSummaryStore)
	if !ok {
		return
	}
	gen, ok := s.provider.(sessionSummaryGenerator)
	if !ok {
		if storeGen, storeOK := s.store.(sessionSummaryGenerator); storeOK {
			gen = storeGen
		} else {
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	recent, err := s.listRecentCandidateMessages(ctx, userID, sessionID, 16)
	if err != nil || len(recent) < 15 {
		return
	}
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	oldSummary := ""
	if existing, found, getErr := summaryStore.GetSessionSummary(ctx, userID, sessionID); getErr == nil && found {
		oldSummary = existing
	}

	msgTexts := make([]string, 0, len(recent))
	var maxMsgID int64
	for _, m := range recent {
		prefix := "用户"
		if m.Role == "assistant" {
			prefix = "助手"
		}
		truncated := m.Content
		if runes := []rune(truncated); len(runes) > 300 {
			truncated = string(runes[:300])
		}
		msgTexts = append(msgTexts, fmt.Sprintf("%s: %s", prefix, truncated))
		if m.ID > maxMsgID {
			maxMsgID = m.ID
		}
	}

	newSummary, err := gen.GenerateSessionSummary(ctx, oldSummary, msgTexts)
	if err != nil || strings.TrimSpace(newSummary) == "" {
		return
	}
	_ = summaryStore.UpsertSessionSummary(ctx, userID, sessionID, strings.TrimSpace(newSummary), maxMsgID, len(recent))
}

func (s *nativeAIService) recordCandidateToolTrace(sessionID, userID int64, toolCallID, toolName, argsJSON, resultContent string, duration time.Duration, execErr error) {
	if s == nil || s.store == nil {
		return
	}
	statusValue := "success"
	errMsg := ""
	if execErr != nil {
		statusValue = "error"
		errMsg = execErr.Error()
	}
	// Truncate large results (e.g. full resume text) for persistence safety.
	storedResult := resultContent
	if runes := []rune(storedResult); len(runes) > 4000 {
		storedResult = string(runes[:4000])
	}
	// Prefer not storing full resume_text in traces.
	if toolName == "get_my_resume_text" || toolName == "recommend_jobs_by_resume" {
		storedResult = redactCandidateResumeTrace(storedResult)
	}
	_, _ = s.store.AppendToolTrace(context.Background(), userID, ToolTraceRow{
		SessionID:     sessionID,
		ToolCallID:    toolCallID,
		ToolName:      toolName,
		ArgsJSON:      argsJSON,
		ResultContent: storedResult,
		Status:        statusValue,
		ErrorMsg:      errMsg,
		DurationMs:    duration.Milliseconds(),
		CreatedAt:     time.Now(),
	})
}

func redactCandidateResumeTrace(raw string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		runes := []rune(raw)
		if len(runes) > 500 {
			return string(runes[:500])
		}
		return raw
	}
	if _, ok := payload["resume_text"]; ok {
		payload["resume_text"] = "[redacted]"
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return `{"resume_text":"[redacted]"}`
	}
	return string(b)
}

func (s *nativeAIService) saveCandidateInterruptedReply(userID, sessionID int64, partial string) {
	content := strings.TrimSpace(partial)
	if content == "" || s == nil || s.store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = s.store.AppendChatMessage(ctx, ChatMessageRow{
		OwnerRole: ownerRoleCandidate, OwnerID: userID, SessionID: sessionID,
		Role: "assistant", Content: content + "\n\n（回复已中断）", CreatedAt: time.Now(),
	})
}

func extractCandidateSystemInstruction(messages []*schema.Message) string {
	for _, m := range messages {
		if m != nil && m.Role == schema.System {
			return m.Content
		}
	}
	return candidateADKSystemPrompt
}

func isCandidateChatCanceled(err error) bool {
	return err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}

func candidateSuggestedQuestionsFallback() []string {
	return []string{
		"我目前的应聘进度？",
		"根据简历推荐岗位",
		"帮我优化简历建议",
	}
}

func buildCandidateProcessContent(suggestedQuestions []string) string {
	if len(suggestedQuestions) == 0 {
		return ""
	}
	payload := map[string]any{
		"suggested_questions": append([]string(nil), suggestedQuestions...),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(b)
}

// candidateSuggestionStreamFilter suppresses suggested-questions markers from stream deltas.
type candidateSuggestionStreamFilter struct {
	onDelta     func(string) error
	buffer      string
	suppressing bool
}

func newCandidateSuggestionStreamFilter(onDelta func(string) error) *candidateSuggestionStreamFilter {
	return &candidateSuggestionStreamFilter{onDelta: onDelta}
}

func (f *candidateSuggestionStreamFilter) Write(delta string) error {
	if delta == "" || f.onDelta == nil || f.suppressing {
		return nil
	}
	f.buffer += delta
	if markerIndex := strings.Index(f.buffer, candidateSuggestedQuestionsStartMarker); markerIndex >= 0 {
		visible := f.buffer[:markerIndex]
		f.buffer = ""
		f.suppressing = true
		if visible != "" {
			return f.onDelta(visible)
		}
		return nil
	}
	keep := longestSuffixMatchingPrefix(f.buffer, candidateSuggestedQuestionsStartMarker)
	flushLen := len(f.buffer) - keep
	if flushLen <= 0 {
		return nil
	}
	visible := f.buffer[:flushLen]
	f.buffer = f.buffer[flushLen:]
	return f.onDelta(visible)
}

func (f *candidateSuggestionStreamFilter) Finish() error {
	if f.onDelta == nil || f.suppressing || f.buffer == "" {
		f.buffer = ""
		return nil
	}
	visible := f.buffer
	f.buffer = ""
	return f.onDelta(visible)
}

func longestSuffixMatchingPrefix(text, prefix string) int {
	max := len(text)
	if len(prefix)-1 < max {
		max = len(prefix) - 1
	}
	for n := max; n > 0; n-- {
		if strings.HasSuffix(text, prefix[:n]) {
			return n
		}
	}
	return 0
}

func normalizeCandidateMarkdownReply(content string) string {
	text := strings.ReplaceAll(content, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	lines = unwrapProseMarkdownFences(lines)
	for i, line := range lines {
		lines[i] = normalizeMarkdownBulletLine(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func unwrapProseMarkdownFences(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		marker, info, ok := markdownFenceLine(lines[i])
		if !ok {
			out = append(out, lines[i])
			continue
		}
		blockStart := i + 1
		closeIndex := -1
		for j := blockStart; j < len(lines); j++ {
			closeMarker, _, closeOK := markdownFenceLine(lines[j])
			if closeOK && closeMarker == marker {
				closeIndex = j
				break
			}
		}
		if closeIndex < 0 {
			out = append(out, lines[i])
			continue
		}
		block := lines[blockStart:closeIndex]
		if shouldUnwrapCandidateMarkdownFence(info, block) {
			indent := strings.Repeat(" ", leadingSpaces(lines[i]))
			for _, blockLine := range trimBlankEdgeLines(deindentLines(block)) {
				if strings.TrimSpace(blockLine) == "" {
					out = append(out, "")
				} else {
					out = append(out, indent+blockLine)
				}
			}
		} else {
			out = append(out, lines[i])
			out = append(out, block...)
			out = append(out, lines[closeIndex])
		}
		i = closeIndex
	}
	return out
}

func markdownFenceLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") {
		return "```", strings.TrimSpace(strings.TrimPrefix(trimmed, "```")), true
	}
	if strings.HasPrefix(trimmed, "~~~") {
		return "~~~", strings.TrimSpace(strings.TrimPrefix(trimmed, "~~~")), true
	}
	return "", "", false
}

func shouldUnwrapCandidateMarkdownFence(info string, lines []string) bool {
	lang := strings.ToLower(strings.TrimSpace(info))
	if lang != "" && lang != "md" && lang != "markdown" && lang != "text" && lang != "plain" {
		return false
	}
	nonEmpty := 0
	markdownSignals := 0
	codeSignals := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		nonEmpty++
		if looksLikeMarkdownProseLine(trimmed) {
			markdownSignals++
		}
		if looksLikeCodeLine(trimmed) {
			codeSignals++
		}
	}
	return nonEmpty > 0 && markdownSignals > 0 && codeSignals == 0
}

func looksLikeMarkdownProseLine(line string) bool {
	return strings.HasPrefix(line, "- ") ||
		strings.HasPrefix(line, "* ") ||
		strings.HasPrefix(line, "+ ") ||
		strings.HasPrefix(line, "## ") ||
		strings.Contains(line, "**")
}

func looksLikeCodeLine(line string) bool {
	upper := strings.ToUpper(line)
	return strings.HasPrefix(line, "{") ||
		strings.HasPrefix(line, "}") ||
		strings.HasPrefix(line, "[") ||
		strings.HasPrefix(line, "]") ||
		strings.HasPrefix(line, "func ") ||
		strings.HasPrefix(line, "const ") ||
		strings.HasPrefix(line, "var ") ||
		strings.HasPrefix(line, "package ") ||
		strings.HasPrefix(line, "$ ") ||
		strings.HasPrefix(line, "npm ") ||
		strings.HasPrefix(line, "pnpm ") ||
		strings.HasPrefix(line, "go test") ||
		strings.HasPrefix(upper, "SELECT ") ||
		strings.HasPrefix(upper, "INSERT ") ||
		strings.HasPrefix(upper, "UPDATE ") ||
		strings.HasPrefix(upper, "DELETE ") ||
		strings.HasPrefix(upper, "CREATE ") ||
		strings.HasPrefix(upper, "ALTER ")
}

func normalizeMarkdownBulletLine(line string) string {
	indentLen := leadingSpaces(line)
	indent := line[:indentLen]
	trimmed := strings.TrimSpace(line[indentLen:])
	for _, marker := range []string{"•", "○", "●", "◦"} {
		if trimmed == marker {
			return indent + "-"
		}
		if strings.HasPrefix(trimmed, marker+" ") {
			return indent + "- " + strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
		}
	}
	return line
}

func deindentLines(lines []string) []string {
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := leadingSpaces(line)
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return append([]string(nil), lines...)
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) >= minIndent {
			out = append(out, line[minIndent:])
		} else {
			out = append(out, strings.TrimLeft(line, " "))
		}
	}
	return out
}

func trimBlankEdgeLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[start:end]
}

func leadingSpaces(line string) int {
	count := 0
	for count < len(line) && line[count] == ' ' {
		count++
	}
	return count
}
