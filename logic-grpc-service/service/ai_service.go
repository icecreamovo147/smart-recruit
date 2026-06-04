package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"logic-grpc-service/ai"
	"logic-grpc-service/model"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type AIService struct {
	chats           *repository.ChatRepo
	applications    *repository.ApplicationRepo
	jobs            *repository.JobRepo
	resumes         *repository.ResumeRepo
	oss             oss.Storage
	ai              *ai.Client
	toolExecutor    *ai.ToolExecutor
	summaries       *repository.SessionSummaryRepo
	toolTraces      *repository.ToolTraceRepo
	memories        *repository.MemoryRepo
	contextBuilder  *AgentContextBuilder
	candidateAI     *CandidateAIService
	usageLogs       *repository.UsageLogRepo
	agentRuntime    string
	authz           *ServiceAuthorizer
	cachedADKTools []tool.BaseTool // lazy-initialized, shared across requests
	cachedToolsMu  sync.Mutex       // guards cachedADKTools init and invalidation
}

func NewAIService(
	chats *repository.ChatRepo,
	applications *repository.ApplicationRepo,
	jobs *repository.JobRepo,
	resumes *repository.ResumeRepo,
	summaries *repository.SessionSummaryRepo,
	toolTraces *repository.ToolTraceRepo,
	memories *repository.MemoryRepo,
	ossClient oss.Storage,
	aiClient *ai.Client,
	toolExecutor *ai.ToolExecutor,
	contextBuilder *AgentContextBuilder,
	candidateAI *CandidateAIService,
	usageLogs *repository.UsageLogRepo,
	agentRuntime string,
	authz *ServiceAuthorizer,
) *AIService {
	return &AIService{
		chats: chats, applications: applications, jobs: jobs, resumes: resumes,
		summaries: summaries, toolTraces: toolTraces, memories: memories,
		oss: ossClient, ai: aiClient, toolExecutor: toolExecutor,
		contextBuilder: contextBuilder, candidateAI: candidateAI,
		usageLogs: usageLogs,
		agentRuntime: agentRuntime,
		authz: authz,
	}
}

// Chat is the non-streaming HR AI chat endpoint.
// It reuses the same Eino Tool Calling path as ChatStream: the model decides
// which tools to call based on the system prompt and tool descriptions.
func (s *AIService) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	if strings.TrimSpace(req.Message) == "" {
		return &pb.ChatResponse{Code: errs.ErrBadRequest, Msg: "消息不能为空"}, nil
	}
	startTime := time.Now()
	inputChars := len([]rune(req.Message))
	log := logger.With(zap.Int64("hr_id", req.HrId), zap.Int64("session_id", req.SessionId))
	session, err := s.getOrCreateChatSession(ctx, req.HrId, req.SessionId)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return &pb.ChatResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限访问"}, nil
	}
	if session.ApplicationID > 0 {
		req.ApplicationId = session.ApplicationID
	}

	var replyBuilder strings.Builder
	reply, metadata, err := s.runToolCallingChat(ctx, req, session, func(delta string) error {
		replyBuilder.WriteString(delta)
		return nil
	}, nil)
	if err != nil {
		if isCanceledError(err) {
			partial := strings.TrimSpace(replyBuilder.String())
			if partial == "" {
				partial = reply
			}
			log.Info("non-streaming chat canceled, returning partial reply", zap.Int("partial_chars", len([]rune(partial))))
			writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
				UserID: req.HrId, Role: 2, ServiceType: "ai_chat",
				Endpoint: "/hr/ai/chat", Provider: "dashscope", Model: s.ai.ModelName(),
				RequestChars: inputChars, ResponseChars: len([]rune(partial)), Status: "timeout", CostMs: int(time.Since(startTime).Milliseconds()),
			})
			return &pb.ChatResponse{Code: errs.OK, Msg: "success", Reply: partial, CreatedAt: formatTime(time.Now()), SessionId: session.ID}, nil
		}
		writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
			UserID: req.HrId, Role: 2, ServiceType: "ai_chat",
			Endpoint: "/hr/ai/chat", Provider: "dashscope", Model: s.ai.ModelName(),
			RequestChars: inputChars, Status: "error", CostMs: int(time.Since(startTime).Milliseconds()),
		})
		return nil, wrapAIError(err)
	}

	now := time.Now()
	outputChars := len([]rune(reply))
	writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
		UserID: req.HrId, Role: 2, ServiceType: "ai_chat",
		Endpoint: "/hr/ai/chat", Provider: "dashscope", Model: s.ai.ModelName(),
		RequestChars: inputChars, ResponseChars: outputChars, CostMs: int(time.Since(startTime).Milliseconds()),
	})
	log.Info("chat completed", zap.Int("reply_len", outputChars))
	resp := &pb.ChatResponse{Code: errs.OK, Msg: "success", Reply: reply, CreatedAt: formatTime(now), SessionId: session.ID}
	if metadata.Action != nil {
		resp.Action = metadata.Action.Action
		resp.ApplicationId = metadata.Action.ApplicationID
		resp.ActionStatus = metadata.Action.ActionStatus
		resp.CandidateName = metadata.Action.CandidateName
		resp.JobTitle = metadata.Action.JobTitle
		resp.Status = metadata.Action.Status
	}
	return resp, nil
}

func (s *AIService) ChatStream(req *pb.ChatRequest, stream pb.AIService_ChatStreamServer) error {
	ctx := stream.Context()
	log := logger.With(zap.Int64("hr_id", req.HrId), zap.Int64("session_id", req.SessionId), zap.Int64("application_id", req.ApplicationId))
	if strings.TrimSpace(req.Message) == "" {
		return stream.Send(&pb.ChatStreamResponse{Code: errs.ErrBadRequest, Msg: "消息不能为空", Done: true})
	}
	startTime := time.Now()
	inputChars := len([]rune(req.Message))
	log.Info("chat stream started", zap.Int("msg_len", inputChars))
	session, err := s.getOrCreateStreamChatSession(ctx, req)
	if err != nil {
		return err
	}
	if session == nil {
		return stream.Send(&pb.ChatStreamResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限访问", Done: true})
	}
	if session.ApplicationID > 0 {
		req.ApplicationId = session.ApplicationID
	}

	now := time.Now()
	statusSender := func(eventType, eventMessage, errorType, toolName string) error {
		return stream.Send(&pb.ChatStreamResponse{
			Code:         errs.OK,
			Msg:          "success",
			EventType:    eventType,
			EventMessage: eventMessage,
			ErrorType:    errorType,
			ToolName:     toolName,
			SessionId:    session.ID,
		})
	}
	reply, metadata, err := s.runToolCallingChat(ctx, req, session, func(delta string) error {
		return stream.Send(&pb.ChatStreamResponse{Code: errs.OK, Msg: "success", Delta: delta, SessionId: session.ID})
	}, statusSender)
	if err != nil {
		writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
			UserID: req.HrId, Role: 2, ServiceType: "ai_chat",
			Endpoint: "/hr/ai/chat/stream", Provider: "dashscope", Model: s.ai.ModelName(),
			RequestChars: inputChars, Status: "error", CostMs: int(time.Since(startTime).Milliseconds()),
		})
		return err
	}

	writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
		UserID: req.HrId, Role: 2, ServiceType: "ai_chat",
		Endpoint: "/hr/ai/chat/stream", Provider: "dashscope", Model: s.ai.ModelName(),
		RequestChars: inputChars, ResponseChars: len([]rune(reply)), CostMs: int(time.Since(startTime).Milliseconds()),
	})

	optionsJSON := ""
	if len(metadata.CandidateOptions) > 0 {
		if data, err := json.Marshal(metadata.CandidateOptions); err == nil {
			optionsJSON = string(data)
		}
	}
	done := &pb.ChatStreamResponse{Code: errs.OK, Msg: "success", Done: true, CreatedAt: formatTime(now), SessionId: session.ID, CandidateOptions: optionsJSON}
	if metadata.Action != nil {
		done.Action = metadata.Action.Action
		done.ApplicationId = metadata.Action.ApplicationID
		done.ActionStatus = metadata.Action.ActionStatus
		done.CandidateName = metadata.Action.CandidateName
		done.JobTitle = metadata.Action.JobTitle
		done.Status = metadata.Action.Status
	}
	return stream.Send(done)
}

// runToolCallingChat is the shared Eino Tool Calling pipeline for HR AI chat.
// It handles context building, tool execution, message persistence, summary refresh,
// and memory writing. Both streaming (ChatStream) and non-streaming (Chat) endpoints
// use this method. The onDelta callback receives incremental reply text; for
// non-streaming callers it accumulates the full reply, for streaming callers it
// writes SSE deltas.
func (s *AIService) runToolCallingChat(ctx context.Context, req *pb.ChatRequest, session *model.AIChatSession, onDelta func(string) error, onStatus func(eventType, eventMessage, errorType, toolName string) error) (reply string, metadata ai.ToolMetadata, err error) {
	// Phase 1-5: Build agent context with all memory layers.
	actx, err := s.contextBuilder.Build(ctx, AgentContextInput{
		HrID:           req.HrId,
		SessionID:      session.ID,
		ApplicationID:  req.ApplicationId,
		CurrentMessage: req.Message,
	})
	if err != nil {
		return "", metadata, err
	}

	logger.L().Info("[AI问答] 用户提问",
		zap.String("question", req.Message),
		zap.Int64("hr_id", req.HrId),
		zap.Int64("session_id", session.ID),
	)

	userAlreadyPersisted := currentMessageAlreadyPersisted(actx, req.Message)
	messages := buildToolCallingMessages(actx, req.Message)

	// Save user message before the model call so it persists even on cancel.
	if !userAlreadyPersisted {
		if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "user", Content: req.Message}); err != nil {
			return "", metadata, err
		}
	}

	// Accumulate partial assistant reply for cancel-save.
	var partialReply strings.Builder
	wrappedDelta := func(delta string) error {
		partialReply.WriteString(delta)
		if onDelta != nil {
			return onDelta(delta)
		}
		return nil
	}

	if s.agentRuntime == "adk" {
		reply, metadata, err = s.runADKChat(ctx, req, session, messages, wrappedDelta, onStatus)
	} else {
		reply, metadata, err = s.runLegacyChat(ctx, req, session, messages, wrappedDelta, onStatus)
	}
	if err != nil {
		if isCanceledError(err) {
			partial := strings.TrimSpace(partialReply.String())
			if partial != "" {
				saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = s.chats.Add(saveCtx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "assistant", Content: partial + "\n\n（回复已中断）"})
			}
			logger.L().Info("chat canceled, partial reply saved if non-empty", zap.Int("partial_chars", len(partial)))
			return reply, metadata, err
		}
		// Phase 3: deterministic fallback from collected tool traces when LLM fails.
		if len(metadata.ToolTraces) > 0 {
			fallback := ai.BuildHRFallbackReply(metadata.ToolTraces)
			aiErr := ai.ClassifyAIError(err)
			if onStatus != nil {
				_ = onStatus("partial_done", "已基于已查询数据给出保守回复", string(aiErr.Type), "")
			}
			logger.L().Warn("[AI兜底] LLM 失败，使用工具结果生成兜底回复",
				zap.String("error_type", string(aiErr.Type)),
				zap.Int("tool_traces", len(metadata.ToolTraces)),
				zap.Int("fallback_chars", len([]rune(fallback))),
			)
			if onDelta != nil {
				_ = onDelta(fallback)
			}
			saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.chats.Add(saveCtx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "assistant", Content: fallback})
			return fallback, metadata, nil
		}
		return reply, metadata, wrapAIError(err)
	}
	logger.L().Info("[AI问答] LLM最终回复",
		zap.String("reply", reply),
		zap.Int("reply_chars", len([]rune(reply))),
	)

	// Save full assistant reply on success.
	if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "assistant", Content: reply}); err != nil {
		return reply, metadata, err
	}

	// Async refresh summary if needed.
	go s.maybeRefreshSummary(session.ID, req.HrId)

	// Async write long-term memory if applicable.
	go s.maybeWriteMemory(req.HrId, req.ApplicationId, reply, metadata)

	return reply, metadata, nil
}

// runADKChat executes the HR AI conversation through the Eino ADK ChatModelAgent.
func (s *AIService) runADKChat(
	ctx context.Context,
	req *pb.ChatRequest,
	session *model.AIChatSession,
	messages []*schema.Message,
	onDelta func(string) error,
	onStatus func(string, string, string, string) error,
) (string, ai.ToolMetadata, error) {
	if req.HrId <= 0 {
		return "", ai.ToolMetadata{}, fmt.Errorf("hrID must be positive, got %d", req.HrId)
	}

	// Thread-safe lazy-init of cached tools via getOrInitADKTools.
	adkTools, err := s.getOrInitADKTools()
	if err != nil {
		logger.L().Warn("[ADK降级] 工具创建失败，自动切换到 Legacy 路径", zap.Error(err))
		return s.runLegacyChat(ctx, req, session, messages, onDelta, onStatus)
	}

	state := &ai.AgentRunState{}

	// Store per-request values in context so tool closures can retrieve them
	// without capturing them (enabling tool caching).
	ctx = ai.WithOwnerID(ctx, req.HrId)
	ctx = ai.WithAgentRunState(ctx, state)

	traceFn := func(toolCallID, toolName, argsJSON, resultContent string, execErr error) {
		go s.recordToolTrace(session.ID, req.HrId, toolCallID, toolName, argsJSON, resultContent, execErr)
	}

	return s.ai.ChatWithADKAgent(ctx, ai.AgentRunInput{
		AgentName:     "hr_recruiting_agent",
		Instruction:   extractSystemInstruction(messages),
		Messages:      messages,
		Tools:         adkTools,
		MaxIterations: 0,
		OwnerID:       req.HrId,
		SessionID:     session.ID,
		State:         state,
	}, onDelta, traceFn, onStatus)
}

// runLegacyChat delegates to the existing ChatWithTools (Eino Tool Calling) path.
func (s *AIService) runLegacyChat(
	ctx context.Context,
	req *pb.ChatRequest,
	session *model.AIChatSession,
	messages []*schema.Message,
	onDelta func(string) error,
	onStatus func(string, string, string, string) error,
) (string, ai.ToolMetadata, error) {
	tools := ai.RecruitingTools()

	traceFn := func(toolCallID, toolName, argsJSON, resultContent string, execErr error) {
		go s.recordToolTrace(session.ID, req.HrId, toolCallID, toolName, argsJSON, resultContent, execErr)
	}

	return s.ai.ChatWithTools(ctx, messages, tools, s.toolExecutor, req.HrId, onDelta, traceFn, onStatus)
}

// extractSystemInstruction pulls the system prompt content from the messages
// list to use as the ADK agent Instruction (the messages are forwarded
// separately as history).
func extractSystemInstruction(messages []*schema.Message) string {
	for _, m := range messages {
		if m.Role == schema.System {
			return m.Content
		}
	}
	return ""
}

func (s *AIService) getOrCreateStreamChatSession(ctx context.Context, req *pb.ChatRequest) (*model.AIChatSession, error) {
	if req.SessionId > 0 || req.ApplicationId == 0 {
		return s.getOrCreateChatSession(ctx, req.HrId, req.SessionId)
	}
	detail, err := s.applications.GetDetailOwned(ctx, req.HrId, req.ApplicationId)
	if err != nil || detail == nil {
		return nil, err
	}
	candidateName := displayCandidateName(detail)
	title := fmt.Sprintf("%s - %s 简历分析", candidateName, detail.JobTitle)
	session := &model.AIChatSession{HrID: req.HrId, Title: limitSessionTitle(title), ApplicationID: req.ApplicationId}
	if err := s.chats.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// AnalyzeApplication is an explicit button-triggered functional interface for resume analysis.
// It directly calls GenerateApplicationAnalysis with pre-loaded resume/position data.
// It does NOT participate in natural-language intent recognition — that path is handled
// exclusively by runToolCallingChat (Eino Tool Calling) for both Chat and ChatStream.
func (s *AIService) AnalyzeApplication(ctx context.Context, req *pb.AnalyzeApplicationRequest) (*pb.AnalyzeApplicationResponse, error) {
	detail, input, err := s.applicationAnalysisInput(ctx, req.HrId, req.ApplicationId, "")
	if err != nil {
		logger.L().Error("analyze application failed", zap.Int64("application_id", req.ApplicationId), zap.Error(err))
		return nil, err
	}
	if detail == nil {
		return &pb.AnalyzeApplicationResponse{Code: errs.ErrForbidden, Msg: "无权限查看该投递记录"}, nil
	}
	startTime := time.Now()
	inputChars := len([]rune(input.Question)) + len([]rune(input.JobTitle)) + len([]rune(input.ResumeText))
	reply, err := s.ai.GenerateApplicationAnalysis(ctx, input, nil)
	if err != nil {
		writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
			UserID: req.HrId, Role: 2, ServiceType: "ai_analyze",
			Endpoint: "/hr/ai/analyze-application", Provider: "dashscope", Model: s.ai.ModelName(),
			RequestChars: inputChars, Status: "error", CostMs: int(time.Since(startTime).Milliseconds()),
		})
		return nil, wrapAIError(err)
	}
	writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
		UserID: req.HrId, Role: 2, ServiceType: "ai_analyze",
		Endpoint: "/hr/ai/analyze-application", Provider: "dashscope", Model: s.ai.ModelName(),
		RequestChars: inputChars, ResponseChars: len([]rune(reply)), CostMs: int(time.Since(startTime).Milliseconds()),
	})
	return &pb.AnalyzeApplicationResponse{
		Code:          errs.OK,
		Msg:           "success",
		Reply:         reply,
		CandidateName: displayCandidateName(detail),
		JobTitle:      detail.JobTitle,
		Status:        detail.Status,
		RoundNo:       detail.RoundNo,
	}, nil
}

func (s *AIService) History(ctx context.Context, req *pb.ChatHistoryRequest) (*pb.ChatHistoryResponse, error) {
	rows, err := s.chats.List(ctx, req.HrId, page(req.Page), pageSize(req.PageSize))
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: errs.OK, Msg: "success", List: toPBChatMessages(rows)}, nil
}

func (s *AIService) ListChatSessions(ctx context.Context, req *pb.ChatSessionListRequest) (*pb.ChatSessionListResponse, error) {
	rows, total, err := s.chats.ListSessions(ctx, req.HrId, page(req.Page), pageSize(req.PageSize))
	if err != nil {
		logger.L().Error("list sessions failed", zap.Int64("hr_id", req.HrId), zap.Error(err))
		return nil, err
	}
	list := make([]*pb.ChatSession, 0, len(rows))
	for _, row := range rows {
		list = append(list, toPBChatSession(row))
	}
	return &pb.ChatSessionListResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *AIService) CreateChatSession(ctx context.Context, req *pb.CreateChatSessionRequest) (*pb.CreateChatSessionResponse, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return &pb.CreateChatSessionResponse{Code: errs.ErrBadRequest, Msg: "会话名称不能为空"}, nil
	}
	session := &model.AIChatSession{HrID: req.HrId, Title: title}
	if err := s.chats.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return &pb.CreateChatSessionResponse{Code: errs.OK, Msg: "success", Session: toPBChatSession(*session)}, nil
}

func (s *AIService) SessionMessages(ctx context.Context, req *pb.SessionMessagesRequest) (*pb.ChatHistoryResponse, error) {
	session, err := s.chats.GetSessionOwned(ctx, req.HrId, req.SessionId)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return &pb.ChatHistoryResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限访问"}, nil
	}
	rows, err := s.chats.ListBySession(ctx, req.HrId, req.SessionId, page(req.Page), pageSize(req.PageSize))
	if err != nil {
		return nil, err
	}
	return &pb.ChatHistoryResponse{Code: errs.OK, Msg: "success", List: toPBChatMessages(rows)}, nil
}

func (s *AIService) CreateApplicationAnalysisSession(ctx context.Context, req *pb.CreateApplicationAnalysisSessionRequest) (*pb.CreateApplicationAnalysisSessionResponse, error) {
	detail, err := s.applications.GetDetailOwned(ctx, req.HrId, req.ApplicationId)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return &pb.CreateApplicationAnalysisSessionResponse{Code: errs.ErrForbidden, Msg: "无权限查看该投递记录"}, nil
	}
	candidateName := displayCandidateName(detail)
	title := fmt.Sprintf("%s - %s 简历分析", candidateName, detail.JobTitle)
	session := &model.AIChatSession{HrID: req.HrId, Title: limitSessionTitle(title), ApplicationID: req.ApplicationId}
	if err := s.chats.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	userMessage := fmt.Sprintf("请帮我分析%s投递%s岗位的简历。", candidateName, detail.JobTitle)
	if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "user", Content: userMessage}); err != nil {
		return nil, err
	}
	rows, err := s.chats.ListBySession(ctx, req.HrId, session.ID, 1, 100)
	if err != nil {
		return nil, err
	}
	logger.L().Info("analysis session created",
		zap.Int64("session_id", session.ID),
		zap.Int64("application_id", req.ApplicationId),
		zap.Int64("hr_id", req.HrId),
	)
	return &pb.CreateApplicationAnalysisSessionResponse{Code: errs.OK, Msg: "success", Session: toPBChatSession(*session), Messages: toPBChatMessages(rows)}, nil
}

func currentMessageAlreadyPersisted(actx *AgentContext, message string) bool {
	if actx == nil || len(actx.RecentMessages) == 0 {
		return false
	}
	last := actx.RecentMessages[len(actx.RecentMessages)-1]
	return last.Role == "user" && strings.TrimSpace(last.Content) == strings.TrimSpace(message)
}

func isCanceledError(err error) bool {
	return errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled
}

func (s *AIService) UpdateSession(ctx context.Context, req *pb.UpdateSessionRequest) (*pb.CommonResponse, error) {
	if strings.TrimSpace(req.Title) == "" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "会话名称不能为空"}, nil
	}
	rows, err := s.chats.UpdateSessionTitle(ctx, req.HrId, req.SessionId, strings.TrimSpace(req.Title))
	if err != nil {
		logger.L().Error("update session failed", zap.Int64("session_id", req.SessionId), zap.Error(err))
		return nil, err
	}
	if rows == 0 {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限操作"}, nil
	}
	logger.L().Info("session renamed", zap.Int64("session_id", req.SessionId))
	return &pb.CommonResponse{Code: errs.OK, Msg: "会话名称已更新"}, nil
}

func (s *AIService) DeleteSession(ctx context.Context, req *pb.DeleteSessionRequest) (*pb.CommonResponse, error) {
	rows, err := s.chats.DeleteSession(ctx, req.HrId, req.SessionId)
	if err != nil {
		logger.L().Error("delete session failed", zap.Int64("session_id", req.SessionId), zap.Error(err))
		return nil, err
	}
	if rows == 0 {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限操作"}, nil
	}
	logger.L().Info("session deleted", zap.Int64("session_id", req.SessionId))
	return &pb.CommonResponse{Code: errs.OK, Msg: "会话已删除"}, nil
}

func (s *AIService) getOrCreateChatSession(ctx context.Context, hrID, sessionID int64) (*model.AIChatSession, error) {
	if sessionID > 0 {
		return s.chats.GetSessionOwned(ctx, hrID, sessionID)
	}
	session := &model.AIChatSession{HrID: hrID, Title: "新对话"}
	if err := s.chats.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// recordToolTrace persists a single tool execution trace asynchronously.
// Failures are logged but do not affect the chat flow.
func (s *AIService) recordToolTrace(sessionID, hrID int64, toolCallID, toolName, argsJSON, resultContent string, execErr error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := "success"
	errMsg := ""
	if execErr != nil {
		status = "error"
		errMsg = execErr.Error()
	}

	// Generate a short result summary for large results.
	resultSummary := truncateResultSummary(resultContent, 500)

	trace := &model.AIToolTrace{
		SessionID:     uint64(sessionID),
		HrID:          uint64(hrID),
		ToolCallID:    toolCallID,
		ToolName:      toolName,
		ArgumentsJSON: argsJSON,
		ResultJSON:    resultContent,
		ResultSummary: resultSummary,
		Status:        status,
		ErrorMessage:  errMsg,
	}

	if err := s.toolTraces.Create(ctx, trace); err != nil {
		logger.L().Warn("[工具轨迹] 写入失败",
			zap.String("tool", toolName),
			zap.String("status", status),
			zap.Error(err),
		)
	} else {
		logger.L().Info("[工具轨迹] 已记录",
			zap.String("tool", toolName),
			zap.String("status", status),
			zap.Int("result_chars", len([]rune(resultContent))),
		)
	}
}

// maybeRefreshSummary checks if the session needs summary refresh and triggers it.
func (s *AIService) maybeRefreshSummary(sessionID, hrID int64) {
	defer func() {
		if r := recover(); r != nil {
			logger.L().Error("summary refresh panic recovered", zap.Any("panic", r), zap.Int64("session_id", sessionID))
		}
	}()

	needsRefresh, err := s.contextBuilder.ShouldRefreshSummary(context.Background(), hrID, sessionID)
	if err != nil {
		logger.L().Warn("summary refresh check failed", zap.Error(err), zap.Int64("session_id", sessionID))
		return
	}
	if !needsRefresh {
		return
	}
	s.contextBuilder.RefreshSessionSummary(sessionID, hrID)
}

// maybeWriteMemory evaluates whether to persist a long-term memory from the current turn.
func (s *AIService) maybeWriteMemory(hrID, applicationID int64, assistantReply string, metadata ai.ToolMetadata) {
	defer func() {
		if r := recover(); r != nil {
			logger.L().Error("memory write panic recovered", zap.Any("panic", r), zap.Int64("hr_id", hrID))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Rule 2: Write conclusion after application analysis (metadata Action + analysis response).
	if metadata.Action != nil && applicationID > 0 {
		conclusion := buildAnalysisConclusion(metadata, assistantReply)
		if conclusion != "" {
			s.writeMemory(ctx, hrID, "application", uint64(applicationID), "conclusion", conclusion, "agent", 0.85)
		}
	}
}

func (s *AIService) writeMemory(ctx context.Context, hrID int64, scopeType string, scopeID uint64, memoryType, content, source string, confidence float64) {
	if strings.TrimSpace(content) == "" {
		return
	}
	memory := &model.AIMemory{
		HrID:       uint64(hrID),
		ScopeType:  scopeType,
		ScopeID:    scopeID,
		MemoryType: memoryType,
		Content:    content,
		Source:     source,
		Confidence: confidence,
	}
	if err := s.memories.Create(ctx, memory); err != nil {
		logger.L().Warn("[长期记忆] 写入失败",
			zap.String("scope_type", scopeType),
			zap.String("memory_type", memoryType),
			zap.Error(err),
		)
	} else {
		logger.L().Info("[长期记忆] 已写入",
			zap.String("scope_type", scopeType),
			zap.String("memory_type", memoryType),
			zap.Int("content_chars", len([]rune(content))),
		)
	}
}

// buildAnalysisConclusion creates a concise conclusion from application analysis results.
func buildAnalysisConclusion(metadata ai.ToolMetadata, assistantReply string) string {
	if metadata.Action == nil {
		return ""
	}
	// Extract a concise summary from the assistant reply (first 200 chars).
	runes := []rune(strings.TrimSpace(assistantReply))
	summary := assistantReply
	if len(runes) > 200 {
		summary = string(runes[:200]) + "..."
	}
	return fmt.Sprintf("投递 %d（候选人：%s，岗位：%s）的分析结论：%s",
		metadata.Action.ApplicationID,
		metadata.Action.CandidateName,
		metadata.Action.JobTitle,
		summary,
	)
}

// truncateResultSummary creates a short summary for large tool results.
// For small results, the result itself is used. For large results, it is truncated.
func truncateResultSummary(result string, maxChars int) string {
	runes := []rune(result)
	if len(runes) <= maxChars {
		return result
	}
	return string(runes[:maxChars]) + fmt.Sprintf("... [总字符数: %d]", len(runes))
}

func (s *AIService) applicationAnalysisInput(ctx context.Context, hrID, applicationID int64, question string) (*repository.ApplicationDetailRow, ai.ApplicationAnalysisInput, error) {
	detail, err := s.applications.GetDetailOwned(ctx, hrID, applicationID)
	if err != nil || detail == nil {
		return detail, ai.ApplicationAnalysisInput{}, err
	}

	// Pipeline: load text → clean for AI
	loaded := loadOrRefreshResumeText(ctx, detail, s.oss, s.resumes)
	resumeText, resumeNote := prepareResumeForAI(loaded.Text, loaded.Note)

	return detail, ai.ApplicationAnalysisInput{
		Question:       question,
		JobTitle:       detail.JobTitle,
		Department:     detail.Department,
		Location:       detail.Location,
		SalaryRange:    detail.SalaryRange,
		Description:    detail.Description,
		Requirements:   detail.Requirements,
		StatusText:     applicationStatusText(detail.Status),
		RoundNo:        detail.RoundNo,
		ResumeFileName: detail.FileName,
		ResumeTextNote: resumeNote,
		ResumeText:     resumeText,
	}, nil
}

// getOrInitADKTools returns the cached ADK tools, initializing them under lock
// on first call. This is safe for concurrent use and supports invalidation via
// InvalidateCachedADKTools for future hot-reload scenarios.
func (s *AIService) getOrInitADKTools() ([]tool.BaseTool, error) {
	s.cachedToolsMu.Lock()
	defer s.cachedToolsMu.Unlock()
	if s.cachedADKTools != nil {
		return s.cachedADKTools, nil
	}
	tools, err := ai.NewRecruitingADKTools(s.toolExecutor)
	if err != nil {
		return nil, err
	}
	s.cachedADKTools = tools
	return tools, nil
}

// InvalidateCachedADKTools clears the cached ADK tools so the next request
// re-creates them. Use this after replacing the ToolExecutor at runtime (e.g.
// during configuration hot-reload).
func (s *AIService) InvalidateCachedADKTools() {
	s.cachedToolsMu.Lock()
	defer s.cachedToolsMu.Unlock()
	s.cachedADKTools = nil
}
