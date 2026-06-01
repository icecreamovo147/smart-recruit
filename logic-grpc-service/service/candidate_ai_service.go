package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/ai"
	"logic-grpc-service/model"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ownerRoleCandidate = 1
const candidateSuggestedQuestionsStartMarker = "<<<CANDIDATE_SUGGESTED_QUESTIONS_JSON>>>"
const candidateSuggestedQuestionsEndMarker = "<<<END_CANDIDATE_SUGGESTED_QUESTIONS_JSON>>>"

// CandidateAIService handles AI assistant requests for candidates.
type CandidateAIService struct {
	usageLogs             *repository.UsageLogRepo
	chats                 *repository.ChatRepo
	applications          *repository.ApplicationRepo
	jobs                  *repository.JobRepo
	resumes               *repository.ResumeRepo
	aiClient              *ai.Client
	toolExecutor          *ai.CandidateToolExecutor
	agentRuntime          string
	toolTraces            *repository.ToolTraceRepo
	summaries             *repository.SessionSummaryRepo
	cachedCandidateADKTools []tool.BaseTool // lazy-initialized, shared across requests
	cachedToolsMu           sync.Mutex       // guards cachedCandidateADKTools init and invalidation
}

func NewCandidateAIService(
	usageLogs *repository.UsageLogRepo,
	chats *repository.ChatRepo,
	applications *repository.ApplicationRepo,
	jobs *repository.JobRepo,
	resumes *repository.ResumeRepo,
	aiClient *ai.Client,
	toolExecutor *ai.CandidateToolExecutor,
	agentRuntime string,
	toolTraces *repository.ToolTraceRepo,
	summaries *repository.SessionSummaryRepo,
) *CandidateAIService {
	return &CandidateAIService{
		usageLogs: usageLogs,
		chats: chats, applications: applications, jobs: jobs, resumes: resumes,
		aiClient: aiClient, toolExecutor: toolExecutor,
		agentRuntime: agentRuntime,
		toolTraces: toolTraces,
		summaries:  summaries,
	}
}

const candidateSystemPrompt = `你是智能招聘系统的候选人 AI 助手，只服务当前登录候选人。
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

` + standardMarkdownReplyRules + `

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

// StreamChat runs a streaming chat for a candidate and saves messages.
func (s *CandidateAIService) StreamChat(ctx context.Context, userID int64, message string, sessionID int64, onDelta func(string) error, onDone func(reply string, sessionID int64) error) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("消息不能为空")
	}

	startTime := time.Now()
	inputChars := len([]rune(message))
	log := logger.With(zap.Int64("user_id", userID), zap.Int64("session_id", sessionID))
	log.Info("candidate chat stream started", zap.Int("msg_len", inputChars))

	session, err := s.getOrCreateSession(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("会话不存在或无权限访问")
	}

	if err := s.chats.AddOwned(ctx, ownerRoleCandidate, userID,
		&model.AIChatHistory{SessionID: session.ID, Role: "user", Content: message}); err != nil {
		return err
	}

	var partialReply strings.Builder
	streamFilter := newCandidateSuggestionStreamFilter(func(delta string) error {
		partialReply.WriteString(delta)
		if onDelta == nil {
			return nil
		}
		return onDelta(delta)
	})

	var reply string
	var metadata ai.ToolMetadata
	var execErr error

	var legacyFallback bool
	if s.agentRuntime == "adk" {
		if userID <= 0 {
			return fmt.Errorf("userID must be positive, got %d", userID)
		}
		messages, buildErr := s.buildCandidateAgentMessages(ctx, userID, session.ID, message)
		if buildErr != nil {
			return buildErr
		}
		adkTools, toolErr := s.getOrInitCandidateADKTools()
		if toolErr != nil {
			logger.L().Warn("[候选人ADK降级] 工具创建失败，降级到 Legacy 路径", zap.Error(toolErr))
			legacyFallback = true
		} else {
			state := &ai.AgentRunState{}
			adkCtx := ai.WithOwnerID(ctx, userID)
			adkCtx = ai.WithAgentRunState(adkCtx, state)
			traceFn := func(toolCallID, toolName, argsJSON, resultContent string, execErr error) {
				go s.recordToolTrace(session.ID, userID, toolCallID, toolName, argsJSON, resultContent, execErr)
			}
			reply, metadata, execErr = s.aiClient.ChatWithADKAgent(adkCtx, ai.AgentRunInput{
				AgentName:   "candidate_assistant",
				Instruction: extractSystemInstruction(messages),
				Messages:    messages,
				Tools:       adkTools,
				OwnerID:     userID,
				SessionID:   session.ID,
				State:       state,
			}, streamFilter.Write, traceFn, nil)
		}
	}
	if legacyFallback || s.agentRuntime != "adk" {
		messages := []*schema.Message{
			schema.SystemMessage(candidateSystemPrompt),
			schema.UserMessage(message),
		}
		tools := ai.CandidateTools()
		reply, metadata, execErr = s.aiClient.ChatWithTools(ctx, messages, tools, s.toolExecutor, userID, streamFilter.Write, nil, nil)
	}
	if execErr != nil {
		if isCanceledError(execErr) {
			s.saveCandidateInterruptedReply(userID, session.ID, partialReply.String())
			logger.L().Info("candidate chat stream canceled, partial reply saved if non-empty",
				zap.Int64("user_id", userID),
				zap.Int64("session_id", session.ID),
				zap.Int("partial_chars", len([]rune(strings.TrimSpace(partialReply.String())))),
			)
			return execErr
		}
		// Phase 3: deterministic fallback from collected tool traces when LLM fails.
		if len(metadata.ToolTraces) > 0 {
			fallback := ai.BuildCandidateFallbackReply(metadata.ToolTraces)
			aiErr := ai.ClassifyAIError(execErr)
			logger.L().Warn("[候选人AI兜底] LLM 失败，使用工具结果生成兜底回复",
				zap.String("error_type", string(aiErr.Type)),
				zap.Int("tool_traces", len(metadata.ToolTraces)),
				zap.Int("fallback_chars", len([]rune(fallback))),
			)
			if onDelta != nil {
				_ = onDelta(fallback)
			}
			saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.chats.AddOwned(saveCtx, ownerRoleCandidate, userID,
				&model.AIChatHistory{SessionID: session.ID, Role: "assistant", Content: fallback})
			return onDone(fallback, session.ID)
		}
		writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
			UserID: userID, Role: 1, ServiceType: "ai_chat",
			Endpoint: "/candidate/ai/chat/stream", Provider: "dashscope", Model: s.aiClient.ModelName(),
			RequestChars: inputChars, Status: "error", CostMs: int(time.Since(startTime).Milliseconds()),
		})
		return wrapAIError(execErr)
	}
	if err := streamFilter.Finish(); err != nil {
		if isCanceledError(err) {
			s.saveCandidateInterruptedReply(userID, session.ID, partialReply.String())
			return err
		}
		return err
	}
	cleanReply, _ := extractCandidateSuggestedQuestions(reply)

	if err := s.chats.AddOwned(ctx, ownerRoleCandidate, userID,
		&model.AIChatHistory{SessionID: session.ID, Role: "assistant", Content: cleanReply}); err != nil {
		return err
	}
	go s.maybeRefreshSummary(session.ID, userID)

	writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
		UserID: userID, Role: 1, ServiceType: "ai_chat",
		Endpoint: "/candidate/ai/chat/stream", Provider: "dashscope", Model: s.aiClient.ModelName(),
		RequestChars: inputChars, ResponseChars: len([]rune(cleanReply)), CostMs: int(time.Since(startTime).Milliseconds()),
	})
	logger.L().Info("[候选人AI] 回复完成",
		zap.Int64("user_id", userID),
		zap.Int64("session_id", session.ID),
		zap.Int("reply_chars", len([]rune(cleanReply))),
	)

	return onDone(cleanReply, session.ID)
}

// ListSessions returns candidate's AI chat sessions.
func (s *CandidateAIService) ListSessions(ctx context.Context, userID int64, page, pageSize int32) ([]model.AIChatSession, int64, error) {
	return s.chats.ListSessionsOwned(ctx, ownerRoleCandidate, userID, page, pageSize)
}

// CreateSession creates a new AI chat session for a candidate.
func (s *CandidateAIService) CreateSession(ctx context.Context, userID int64, title string) (*model.AIChatSession, error) {
	if strings.TrimSpace(title) == "" {
		title = "新对话"
	}
	session := &model.AIChatSession{Title: title}
	if err := s.chats.CreateSessionOwned(ctx, ownerRoleCandidate, userID, session); err != nil {
		return nil, err
	}
	return session, nil
}

// SessionMessages returns messages for a candidate's session.
func (s *CandidateAIService) SessionMessages(ctx context.Context, userID, sessionID int64, page, pageSize int32) ([]model.AIChatHistory, error) {
	session, err := s.chats.GetSessionOwnedBy(ctx, ownerRoleCandidate, userID, sessionID)
	if err != nil || session == nil {
		return nil, err
	}
	return s.chats.ListBySessionOwned(ctx, ownerRoleCandidate, userID, sessionID, page, pageSize)
}

// DeleteSession deletes a candidate's session.
func (s *CandidateAIService) DeleteSession(ctx context.Context, userID, sessionID int64) error {
	rows, err := s.chats.DeleteSessionOwned(ctx, ownerRoleCandidate, userID, sessionID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("会话不存在或无权限操作")
	}
	return nil
}

// UpdateSessionTitle renames a candidate's session.
func (s *CandidateAIService) UpdateSessionTitle(ctx context.Context, userID, sessionID int64, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("会话名称不能为空")
	}
	rows, err := s.chats.UpdateSessionTitleOwned(ctx, ownerRoleCandidate, userID, sessionID, limitSessionTitle(title))
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("会话不存在或无权限操作")
	}
	return nil
}

func (s *CandidateAIService) getOrCreateSession(ctx context.Context, userID, sessionID int64) (*model.AIChatSession, error) {
	if sessionID > 0 {
		return s.chats.GetSessionOwnedBy(ctx, ownerRoleCandidate, userID, sessionID)
	}
	session := &model.AIChatSession{Title: "新对话"}
	if err := s.chats.CreateSessionOwned(ctx, ownerRoleCandidate, userID, session); err != nil {
		return nil, err
	}
	return session, nil
}

// ---- gRPC handler methods ----

// StreamChatGRPC handles SSE streaming chat via gRPC.
func (s *CandidateAIService) StreamChatGRPC(req *pb.CandidateChatRequest, stream pb.AIService_CandidateChatStreamServer) error {
	ctx := stream.Context()

	if strings.TrimSpace(req.Message) == "" {
		return stream.Send(&pb.ChatStreamResponse{Code: errs.ErrBadRequest, Msg: "消息不能为空", Done: true})
	}

	startTime := time.Now()
	inputChars := len([]rune(req.Message))
	log := logger.With(zap.Int64("user_id", req.UserId), zap.Int64("session_id", req.SessionId))
	log.Info("candidate chat stream started", zap.Int("msg_len", inputChars))

	session, err := s.getOrCreateSession(ctx, req.UserId, req.SessionId)
	if err != nil {
		return err
	}
	if session == nil {
		return stream.Send(&pb.ChatStreamResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限访问", Done: true})
	}

	if err := s.chats.AddOwned(ctx, ownerRoleCandidate, req.UserId,
		&model.AIChatHistory{SessionID: session.ID, Role: "user", Content: req.Message}); err != nil {
		return err
	}

	var partialReply strings.Builder
	streamFilter := newCandidateSuggestionStreamFilter(func(delta string) error {
		partialReply.WriteString(delta)
		return stream.Send(&pb.ChatStreamResponse{Code: errs.OK, Msg: "success", Delta: delta, SessionId: session.ID})
	})
	statusSender := func(eventType, eventMessage, errorType, toolName string) error {
		return stream.Send(&pb.ChatStreamResponse{
			Code: errs.OK, Msg: "success", EventType: eventType, EventMessage: eventMessage,
			ErrorType: errorType, ToolName: toolName, SessionId: session.ID,
		})
	}

	var reply string
	var metadata ai.ToolMetadata
	var execErr error

	var legacyFallback bool
	if s.agentRuntime == "adk" {
		if req.UserId <= 0 {
			return fmt.Errorf("userID must be positive, got %d", req.UserId)
		}
		messages, buildErr := s.buildCandidateAgentMessages(ctx, req.UserId, session.ID, req.Message)
		if buildErr != nil {
			return buildErr
		}
		adkTools, toolErr := s.getOrInitCandidateADKTools()
		if toolErr != nil {
			logger.L().Warn("[候选人ADK降级] 工具创建失败，降级到 Legacy 路径", zap.Error(toolErr))
			legacyFallback = true
		} else {
			state := &ai.AgentRunState{}
			adkCtx := ai.WithOwnerID(ctx, req.UserId)
			adkCtx = ai.WithAgentRunState(adkCtx, state)
			traceFn := func(toolCallID, toolName, argsJSON, resultContent string, execErr error) {
				go s.recordToolTrace(session.ID, req.UserId, toolCallID, toolName, argsJSON, resultContent, execErr)
			}
			reply, metadata, execErr = s.aiClient.ChatWithADKAgent(adkCtx, ai.AgentRunInput{
				AgentName:   "candidate_assistant",
				Instruction: extractSystemInstruction(messages),
				Messages:    messages,
				Tools:       adkTools,
				OwnerID:     req.UserId,
				SessionID:   session.ID,
				State:       state,
			}, streamFilter.Write, traceFn, statusSender)
		}
	}
	if legacyFallback || s.agentRuntime != "adk" {
		messages := []*schema.Message{
			schema.SystemMessage(candidateSystemPrompt),
			schema.UserMessage(req.Message),
		}
		tools := ai.CandidateTools()
		reply, metadata, execErr = s.aiClient.ChatWithTools(ctx, messages, tools, s.toolExecutor, req.UserId, streamFilter.Write, nil, statusSender)
	}
	if execErr != nil {
		if isCanceledError(execErr) {
			partial := partialReply.String()
			s.saveCandidateInterruptedReply(req.UserId, session.ID, partial)
			writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
				UserID: req.UserId, Role: 1, ServiceType: "ai_chat",
				Endpoint: "/candidate/ai/chat/stream", Provider: "dashscope", Model: s.aiClient.ModelName(),
				RequestChars: inputChars, ResponseChars: len([]rune(partial)),
				Status: "timeout", CostMs: int(time.Since(startTime).Milliseconds()),
			})
			logger.L().Info("candidate chat stream canceled, partial reply saved if non-empty",
				zap.Int64("user_id", req.UserId),
				zap.Int64("session_id", session.ID),
				zap.Int("partial_chars", len([]rune(strings.TrimSpace(partialReply.String())))),
			)
			return execErr
		}
		// Phase 3: deterministic fallback from collected tool traces when LLM fails.
		if len(metadata.ToolTraces) > 0 {
			fallback := ai.BuildCandidateFallbackReply(metadata.ToolTraces)
			aiErr := ai.ClassifyAIError(execErr)
			// Phase 4: partial_done status event before fallback content.
			_ = stream.Send(&pb.ChatStreamResponse{Code: errs.OK, Msg: "success", EventType: "partial_done", EventMessage: "已基于已查询数据给出保守回复", ErrorType: string(aiErr.Type), SessionId: session.ID})
			logger.L().Warn("[候选人AI兜底] LLM 失败，使用工具结果生成兜底回复",
				zap.String("error_type", string(aiErr.Type)),
				zap.Int64("user_id", req.UserId),
				zap.Int64("session_id", session.ID),
				zap.Int("tool_traces", len(metadata.ToolTraces)),
				zap.Int("fallback_chars", len([]rune(fallback))),
			)
			if err := stream.Send(&pb.ChatStreamResponse{Code: errs.OK, Msg: "success", Delta: fallback, SessionId: session.ID}); err != nil {
				return err
			}
			saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.chats.AddOwned(saveCtx, ownerRoleCandidate, req.UserId,
				&model.AIChatHistory{SessionID: session.ID, Role: "assistant", Content: fallback})
			writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
				UserID: req.UserId, Role: 1, ServiceType: "ai_chat",
				Endpoint: "/candidate/ai/chat/stream", Provider: "dashscope", Model: s.aiClient.ModelName(),
				RequestChars: inputChars, ResponseChars: len([]rune(fallback)),
				Status: "error", ErrorCode: string(aiErr.Type), CostMs: int(time.Since(startTime).Milliseconds()),
			})
			return stream.Send(&pb.ChatStreamResponse{
				Code: errs.OK, Msg: "success", Done: true,
				CreatedAt:          time.Now().Format("2006-01-02 15:04:05"),
				SessionId:          session.ID,
				SuggestedQuestions: candidateSuggestedQuestions(req.Message, fallback),
			})
		}
		writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
			UserID: req.UserId, Role: 1, ServiceType: "ai_chat",
			Endpoint: "/candidate/ai/chat/stream", Provider: "dashscope", Model: s.aiClient.ModelName(),
			RequestChars: inputChars, Status: "error", CostMs: int(time.Since(startTime).Milliseconds()),
		})
		return wrapAIError(execErr)
	}
	if err := streamFilter.Finish(); err != nil {
		if isCanceledError(err) {
			partial := partialReply.String()
			s.saveCandidateInterruptedReply(req.UserId, session.ID, partial)
			writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
				UserID: req.UserId, Role: 1, ServiceType: "ai_chat",
				Endpoint: "/candidate/ai/chat/stream", Provider: "dashscope", Model: s.aiClient.ModelName(),
				RequestChars: inputChars, ResponseChars: len([]rune(partial)),
				Status: "timeout", CostMs: int(time.Since(startTime).Milliseconds()),
			})
			return err
		}
		return err
	}
	cleanReply, suggestedQuestions := extractCandidateSuggestedQuestions(reply)

	// Save messages
	if err := s.chats.AddOwned(ctx, ownerRoleCandidate, req.UserId,
		&model.AIChatHistory{SessionID: session.ID, Role: "assistant", Content: cleanReply}); err != nil {
		return err
	}
	go s.maybeRefreshSummary(session.ID, req.UserId)

	logger.L().Info("[候选人AI] 回复完成",
		zap.Int64("user_id", req.UserId),
		zap.Int64("session_id", session.ID),
		zap.Int("reply_chars", len([]rune(cleanReply))),
	)
	writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
		UserID: req.UserId, Role: 1, ServiceType: "ai_chat",
		Endpoint: "/candidate/ai/chat/stream", Provider: "dashscope", Model: s.aiClient.ModelName(),
		RequestChars: inputChars, ResponseChars: len([]rune(cleanReply)), CostMs: int(time.Since(startTime).Milliseconds()),
	})

	if len(suggestedQuestions) != 3 {
		suggestedQuestions = candidateSuggestedQuestions(req.Message, cleanReply)
	}
	return stream.Send(&pb.ChatStreamResponse{
		Code: errs.OK, Msg: "success", Done: true,
		CreatedAt:          time.Now().Format("2006-01-02 15:04:05"),
		SessionId:          session.ID,
		SuggestedQuestions: suggestedQuestions,
	})
}

func (s *CandidateAIService) saveCandidateInterruptedReply(userID, sessionID int64, partial string) {
	content := strings.TrimSpace(partial)
	if content == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.chats.AddOwned(ctx, ownerRoleCandidate, userID,
		&model.AIChatHistory{SessionID: sessionID, Role: "assistant", Content: content + "\n\n（回复已中断）"}); err != nil {
		logger.L().Warn("candidate interrupted reply save failed",
			zap.Int64("user_id", userID),
			zap.Int64("session_id", sessionID),
			zap.Error(err),
		)
	}
}

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

func extractCandidateSuggestedQuestions(reply string) (string, []string) {
	raw := strings.TrimSpace(reply)
	start := strings.Index(raw, candidateSuggestedQuestionsStartMarker)
	if start < 0 {
		return raw, nil
	}

	cleanReply := strings.TrimSpace(raw[:start])
	rest := raw[start+len(candidateSuggestedQuestionsStartMarker):]
	jsonText := rest
	if end := strings.Index(rest, candidateSuggestedQuestionsEndMarker); end >= 0 {
		jsonText = rest[:end]
		after := strings.TrimSpace(rest[end+len(candidateSuggestedQuestionsEndMarker):])
		if after != "" {
			cleanReply = strings.TrimSpace(cleanReply + "\n\n" + after)
		}
	}

	return cleanReply, parseCandidateSuggestedQuestionsJSON(jsonText)
}

func parseCandidateSuggestedQuestionsJSON(content string) []string {
	raw := strings.TrimSpace(content)
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
	for _, item := range items {
		question := strings.TrimSpace(item)
		if question == "" || containsString(result, question) {
			continue
		}
		runes := []rune(question)
		if len(runes) > 40 {
			question = string(runes[:40])
		}
		result = append(result, question)
		if len(result) == 3 {
			break
		}
	}
	if len(result) != 3 {
		return nil
	}
	return result
}

// candidateSuggestedQuestions returns a fixed set of 3 safe fallback questions.
// It no longer reads userMessage or reply to produce keyword-based suggestions;
// the model is expected to generate context-aware suggested questions.
// These fallbacks are only used when the model's suggested-question JSON cannot be
// parsed or returns fewer than 3 items.
func candidateSuggestedQuestions(_, _ string) []string {
	return []string{
		"我目前的应聘进度？",
		"根据简历推荐岗位",
		"帮我优化简历建议",
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// ListSessionsGRPC returns candidate's sessions as a proto response.
func (s *CandidateAIService) ListSessionsGRPC(ctx context.Context, req *pb.CandidateSessionListRequest) (*pb.ChatSessionListResponse, error) {
	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	rows, total, err := s.chats.ListSessionsOwned(ctx, ownerRoleCandidate, req.UserId, page, pageSize)
	if err != nil {
		return nil, err
	}
	list := make([]*pb.ChatSession, 0, len(rows))
	for _, row := range rows {
		list = append(list, &pb.ChatSession{
			SessionId: row.ID,
			Title:     row.Title,
			CreatedAt: row.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: row.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &pb.ChatSessionListResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

// CreateSessionGRPC creates a session and returns proto response.
func (s *CandidateAIService) CreateSessionGRPC(ctx context.Context, req *pb.CandidateCreateSessionRequest) (*pb.CreateChatSessionResponse, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return &pb.CreateChatSessionResponse{Code: errs.ErrBadRequest, Msg: "会话名称不能为空"}, nil
	}
	session := &model.AIChatSession{Title: title}
	if err := s.chats.CreateSessionOwned(ctx, ownerRoleCandidate, req.UserId, session); err != nil {
		return nil, err
	}
	return &pb.CreateChatSessionResponse{
		Code: errs.OK,
		Msg:  "success",
		Session: &pb.ChatSession{
			SessionId: session.ID,
			Title:     session.Title,
			CreatedAt: session.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: session.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// SessionMessagesGRPC returns session messages as proto response.
func (s *CandidateAIService) SessionMessagesGRPC(ctx context.Context, req *pb.CandidateSessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	session, err := s.chats.GetSessionOwnedBy(ctx, ownerRoleCandidate, req.UserId, req.SessionId)
	if err != nil || session == nil {
		return &pb.ChatHistoryResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限访问"}, nil
	}

	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}

	rows, err := s.chats.ListBySessionOwned(ctx, ownerRoleCandidate, req.UserId, req.SessionId, page, pageSize)
	if err != nil {
		return nil, err
	}
	list := make([]*pb.ChatMessage, 0, len(rows))
	for _, row := range rows {
		list = append(list, &pb.ChatMessage{
			Role:      row.Role,
			Content:   row.Content,
			CreatedAt: row.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &pb.ChatHistoryResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

// UpdateSessionGRPC renames a session and returns proto response.
func (s *CandidateAIService) UpdateSessionGRPC(ctx context.Context, req *pb.CandidateUpdateSessionRequest) (*pb.CommonResponse, error) {
	if err := s.UpdateSessionTitle(ctx, req.UserId, req.SessionId, req.Title); err != nil {
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "会话已重命名"}, nil
}

// DeleteSessionGRPC deletes a session and returns proto response.
func (s *CandidateAIService) DeleteSessionGRPC(ctx context.Context, req *pb.CandidateDeleteSessionRequest) (*pb.CommonResponse, error) {
	rows, err := s.chats.DeleteSessionOwned(ctx, ownerRoleCandidate, req.UserId, req.SessionId)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限操作"}, nil
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "会话已删除"}, nil
}

// recordToolTrace persists a single tool execution trace asynchronously.
// Failures are logged but do not affect the chat flow.
// Note: For candidate traces, the HrID field stores the userID.
func (s *CandidateAIService) recordToolTrace(sessionID, userID int64, toolCallID, toolName, argsJSON, resultContent string, execErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := "success"
	errMsg := ""
	if execErr != nil {
		status = "error"
		errMsg = execErr.Error()
	}

	// Generate a short result summary for large results.
	resultSummary := resultContent
	runes := []rune(resultSummary)
	if len(runes) > 500 {
		resultSummary = string(runes[:500])
	}

	trace := &model.AIToolTrace{
		SessionID:     uint64(sessionID),
		HrID:          uint64(userID),
		ToolCallID:    toolCallID,
		ToolName:      toolName,
		ArgumentsJSON: argsJSON,
		ResultJSON:    resultContent,
		ResultSummary: resultSummary,
		Status:        status,
		ErrorMessage:  errMsg,
	}

	if err := s.toolTraces.Create(ctx, trace); err != nil {
		logger.L().Warn("[候选人工具轨迹] 写入失败",
			zap.String("tool", toolName),
			zap.String("status", status),
			zap.Error(err),
		)
	} else {
		logger.L().Info("[候选人工具轨迹] 已记录",
			zap.String("tool", toolName),
			zap.String("status", status),
			zap.Int("result_chars", len([]rune(resultContent))),
		)
	}
}

// maybeRefreshSummary checks if the candidate session needs summary refresh
// and triggers it asynchronously. It uses a simple heuristic: refresh if the
// session has more than 15 messages.
func (s *CandidateAIService) maybeRefreshSummary(sessionID, userID int64) {
	defer func() {
		if r := recover(); r != nil {
			logger.L().Error("[候选人] summary refresh panic recovered", zap.Any("panic", r), zap.Int64("session_id", sessionID))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if session has enough messages to warrant a summary.
	recent, err := s.chats.ListRecentBySessionOwned(ctx, ownerRoleCandidate, userID, sessionID, 16)
	if err != nil {
		logger.L().Warn("[候选人] summary refresh: failed to check message count", zap.Error(err), zap.Int64("session_id", sessionID))
		return
	}
	if len(recent) < 15 {
		return
	}

	// Release the short-lived context before creating a longer one.
	cancel()
	// Extend timeout for LLM call.
	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log := logger.With(zap.Int64("session_id", sessionID), zap.Int64("user_id", userID))

	// Get existing summary.
	oldSummary := ""
	existing, err := s.summaries.GetBySession(ctx, userID, sessionID)
	if err == nil && existing != nil {
		oldSummary = existing.Summary
	}

	// Format messages for summarization.
	msgTexts := make([]string, 0, len(recent))
	var maxMsgID int64
	for _, m := range recent {
		prefix := "用户"
		if m.Role == "assistant" {
			prefix = "助手"
		}
		truncated := m.Content
		runes := []rune(truncated)
		if len(runes) > 300 {
			truncated = string(runes[:300])
		}
		msgTexts = append(msgTexts, fmt.Sprintf("%s: %s", prefix, truncated))
		if m.ID > maxMsgID {
			maxMsgID = m.ID
		}
	}

	newSummary, err := s.aiClient.GenerateSessionSummary(ctx, oldSummary, msgTexts)
	if err != nil {
		log.Error("[候选人] summary: LLM generation failed", zap.Error(err))
		return
	}
	if strings.TrimSpace(newSummary) == "" {
		log.Warn("[候选人] summary: LLM returned empty summary, skipping upsert")
		return
	}

	if err := s.summaries.Upsert(ctx, &model.AISessionSummary{
		SessionID:        uint64(sessionID),
		HrID:             uint64(userID),
		Summary:          newSummary,
		CoveredMessageID: uint64(maxMsgID),
		MessageCount:     len(recent),
	}); err != nil {
		log.Error("[候选人] summary: upsert failed", zap.Error(err))
		return
	}

	log.Info("[候选人] summary refreshed",
		zap.Int("summary_chars", len([]rune(newSummary))),
		zap.Int64("covered_message_id", maxMsgID),
	)
}

// getOrInitCandidateADKTools returns the cached candidate ADK tools, initializing
// them under lock on first call. Safe for concurrent use; supports invalidation
// via InvalidateCachedCandidateADKTools for hot-reload scenarios.
func (s *CandidateAIService) getOrInitCandidateADKTools() ([]tool.BaseTool, error) {
	s.cachedToolsMu.Lock()
	defer s.cachedToolsMu.Unlock()
	if s.cachedCandidateADKTools != nil {
		return s.cachedCandidateADKTools, nil
	}
	tools, err := ai.NewCandidateADKTools(s.toolExecutor)
	if err != nil {
		return nil, err
	}
	s.cachedCandidateADKTools = tools
	return tools, nil
}

// InvalidateCachedCandidateADKTools clears the cached candidate ADK tools so
// the next request re-creates them. Use after replacing the executor at runtime.
func (s *CandidateAIService) InvalidateCachedCandidateADKTools() {
	s.cachedToolsMu.Lock()
	defer s.cachedToolsMu.Unlock()
	s.cachedCandidateADKTools = nil
}
