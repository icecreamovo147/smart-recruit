package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"logic-grpc-service/ai"
	"logic-grpc-service/model"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type AIService struct {
	chats          *repository.ChatRepo
	applications   *repository.ApplicationRepo
	jobs           *repository.JobRepo
	resumes        *repository.ResumeRepo
	oss            *oss.Client
	ai             *ai.Client
	toolExecutor   *ai.ToolExecutor
	summaries      *repository.SessionSummaryRepo
	toolTraces     *repository.ToolTraceRepo
	memories       *repository.MemoryRepo
	contextBuilder *AgentContextBuilder
	candidateAI    *CandidateAIService
}

func NewAIService(
	chats *repository.ChatRepo,
	applications *repository.ApplicationRepo,
	jobs *repository.JobRepo,
	resumes *repository.ResumeRepo,
	summaries *repository.SessionSummaryRepo,
	toolTraces *repository.ToolTraceRepo,
	memories *repository.MemoryRepo,
	ossClient *oss.Client,
	aiClient *ai.Client,
	toolExecutor *ai.ToolExecutor,
	contextBuilder *AgentContextBuilder,
	candidateAI *CandidateAIService,
) *AIService {
	return &AIService{
		chats: chats, applications: applications, jobs: jobs, resumes: resumes,
		summaries: summaries, toolTraces: toolTraces, memories: memories,
		oss: ossClient, ai: aiClient, toolExecutor: toolExecutor,
		contextBuilder: contextBuilder, candidateAI: candidateAI,
	}
}

func (s *AIService) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	if strings.TrimSpace(req.Message) == "" {
		return &pb.ChatResponse{Code: errs.ErrBadRequest, Msg: "消息不能为空"}, nil
	}
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
	if req.ApplicationId > 0 {
		return s.chatWithApplication(ctx, req, session.ID)
	}
	total, err := s.applications.TotalByHR(ctx, req.HrId)
	if err != nil {
		return nil, err
	}
	today, err := s.applications.TodayByHR(ctx, req.HrId)
	if err != nil {
		return nil, err
	}
	hotRows, err := s.applications.HotJobs(ctx, req.HrId, 3)
	if err != nil {
		return nil, err
	}
	hot := make([]string, 0, len(hotRows))
	for _, row := range hotRows {
		hot = append(hot, fmt.Sprintf("%s（%d份）", row.Title, row.Total))
	}
	reply, err := s.ai.GenerateRecruitingReply(ctx, req.Message, ai.RecruitingStats{TotalApplications: total, TodayApplications: today, HotJobs: hot}, nil)
	if err != nil {
		log.Error("ai recruiting reply failed", zap.Error(err))
		return nil, wrapAIError(err)
	}
	now := time.Now()
	if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "user", Content: req.Message}); err != nil {
		return nil, err
	}
	if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "assistant", Content: reply}); err != nil {
		return nil, err
	}
	log.Info("chat completed", zap.Int("reply_len", len([]rune(reply))))
	return &pb.ChatResponse{Code: errs.OK, Msg: "success", Reply: reply, CreatedAt: formatTime(now), SessionId: session.ID}, nil
}

func (s *AIService) ChatStream(req *pb.ChatRequest, stream pb.AIService_ChatStreamServer) error {
	ctx := stream.Context()
	log := logger.With(zap.Int64("hr_id", req.HrId), zap.Int64("session_id", req.SessionId), zap.Int64("application_id", req.ApplicationId))
	if strings.TrimSpace(req.Message) == "" {
		return stream.Send(&pb.ChatStreamResponse{Code: errs.ErrBadRequest, Msg: "消息不能为空", Done: true})
	}
	log.Info("chat stream started", zap.Int("msg_len", len([]rune(req.Message))))
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
	return s.chatWithTools(ctx, req, session, stream)
}

func (s *AIService) chatWithTools(ctx context.Context, req *pb.ChatRequest, session *model.AIChatSession, stream pb.AIService_ChatStreamServer) error {
	now := time.Now()

	// Phase 1-5: Build agent context with all memory layers.
	actx, err := s.contextBuilder.Build(ctx, AgentContextInput{
		HrID:           req.HrId,
		SessionID:      session.ID,
		ApplicationID:  req.ApplicationId,
		CurrentMessage: req.Message,
	})
	if err != nil {
		return err
	}

	logger.L().Info("[AI问答] 用户提问",
		zap.String("question", req.Message),
		zap.Int64("hr_id", req.HrId),
		zap.Int64("session_id", session.ID),
	)

	userAlreadyPersisted := currentMessageAlreadyPersisted(actx, req.Message)
	messages := buildToolCallingMessages(actx, req.Message)
	tools := ai.RecruitingTools()

	// Phase 3: Tool trace callback — records each tool execution asynchronously.
	traceFn := func(toolCallID, toolName, argsJSON, resultContent string, execErr error) {
		go s.recordToolTrace(session.ID, req.HrId, toolCallID, toolName, argsJSON, resultContent, execErr)
	}

	// Save user message before the model call so it persists even on cancel.
	// Application-analysis sessions may already have persisted the first user
	// message when the session was explicitly created from the ledger page.
	if !userAlreadyPersisted {
		if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "user", Content: req.Message}); err != nil {
			return err
		}
	}

	if reply, ok := directHRAssistantReply(req.Message); ok {
		logger.L().Info("[AI问答] 直接回复（无需工具）",
			zap.String("question", req.Message),
			zap.Int64("hr_id", req.HrId),
			zap.Int64("session_id", session.ID),
		)
		if err := stream.Send(&pb.ChatStreamResponse{Code: errs.OK, Msg: "success", Delta: reply, SessionId: session.ID}); err != nil {
			return err
		}
		if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "assistant", Content: reply}); err != nil {
			return err
		}
		go s.maybeRefreshSummary(session.ID, req.HrId)
		go s.maybeWriteMemory(req.HrId, session.ID, req.ApplicationId, req.Message, reply, ai.ToolMetadata{})
		return stream.Send(&pb.ChatStreamResponse{Code: errs.OK, Msg: "success", Done: true, CreatedAt: formatTime(now), SessionId: session.ID})
	}

	// Accumulate partial assistant reply for cancel-save.
	var partialReply strings.Builder
	reply, metadata, err := s.ai.ChatWithTools(ctx, messages, tools, s.toolExecutor, req.HrId, func(delta string) error {
		partialReply.WriteString(delta)
		return stream.Send(&pb.ChatStreamResponse{Code: errs.OK, Msg: "success", Delta: delta, SessionId: session.ID})
	}, traceFn)
	if err != nil {
		if isCanceledError(err) {
			partial := strings.TrimSpace(partialReply.String())
			if partial != "" {
				saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = s.chats.Add(saveCtx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "assistant", Content: partial + "\n\n（回复已中断）"})
			}
			logger.L().Info("chat stream canceled, partial reply saved if non-empty", zap.Int("partial_chars", len(partial)))
			// Return the original cancel error so upstream knows this was a user abort, not an AI failure.
			return err
		}
		return wrapAIError(err)
	}
	logger.L().Info("[AI问答] LLM最终回复",
		zap.String("reply", reply),
		zap.Int("reply_chars", len([]rune(reply))),
	)

	// Save full assistant reply on success.
	if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "assistant", Content: reply}); err != nil {
		return err
	}

	// Phase 2: Async refresh summary if needed.
	go s.maybeRefreshSummary(session.ID, req.HrId)

	// Phase 4: Async write long-term memory if applicable.
	go s.maybeWriteMemory(req.HrId, session.ID, req.ApplicationId, req.Message, reply, metadata)

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

func (s *AIService) AnalyzeApplication(ctx context.Context, req *pb.AnalyzeApplicationRequest) (*pb.AnalyzeApplicationResponse, error) {
	detail, input, err := s.applicationAnalysisInput(ctx, req.HrId, req.ApplicationId, "")
	if err != nil {
		logger.L().Error("analyze application failed", zap.Int64("application_id", req.ApplicationId), zap.Error(err))
		return nil, err
	}
	if detail == nil {
		return &pb.AnalyzeApplicationResponse{Code: errs.ErrForbidden, Msg: "无权限查看该投递记录"}, nil
	}
	reply, err := s.ai.GenerateApplicationAnalysis(ctx, input, nil)
	if err != nil {
		return nil, wrapAIError(err)
	}
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

func (s *AIService) chatWithApplication(ctx context.Context, req *pb.ChatRequest, sessionID int64) (*pb.ChatResponse, error) {
	detail, input, err := s.applicationAnalysisInput(ctx, req.HrId, req.ApplicationId, req.Message)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return &pb.ChatResponse{Code: errs.ErrForbidden, Msg: "无权限查看该投递记录"}, nil
	}
	if actionStatus, ok := detectApplicationAction(req.Message); ok {
		action := "reject_application"
		actionText := "淘汰"
		if actionStatus == 2 {
			action = "approve_application"
			actionText = "通过"
		}
		reply := fmt.Sprintf("我识别到你想将「%s」投递「%s」的申请标记为「%s」。请确认后我会为你更新投递状态。", displayCandidateName(detail), detail.JobTitle, actionText)
		now := time.Now()
		if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: sessionID, HrID: req.HrId, Role: "user", Content: req.Message}); err != nil {
			return nil, err
		}
		logger.L().Info("[AI问答] LLM最终回复",
			zap.String("reply", reply),
			zap.Int("reply_chars", len([]rune(reply))),
		)

		if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: sessionID, HrID: req.HrId, Role: "assistant", Content: reply}); err != nil {
			return nil, err
		}
		return &pb.ChatResponse{Code: errs.OK, Msg: "success", Reply: reply, CreatedAt: formatTime(now), Action: action, ApplicationId: req.ApplicationId, ActionStatus: actionStatus, CandidateName: displayCandidateName(detail), JobTitle: detail.JobTitle, Status: detail.Status, SessionId: sessionID}, nil
	}
	reply, err := s.ai.GenerateApplicationAnalysis(ctx, input, nil)
	if err != nil {
		return nil, wrapAIError(err)
	}
	// If context was canceled (e.g. user aborted), don't persist the half-baked reply.
	if err := ctx.Err(); err != nil {
		logger.L().Info("application analysis canceled before persistence", zap.Error(err))
		return nil, err
	}
	now := time.Now()
	if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: sessionID, HrID: req.HrId, Role: "user", Content: req.Message}); err != nil {
		return nil, err
	}
	if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: sessionID, HrID: req.HrId, Role: "assistant", Content: reply}); err != nil {
		return nil, err
	}
	return &pb.ChatResponse{Code: errs.OK, Msg: "success", Reply: reply, CreatedAt: formatTime(now), ApplicationId: req.ApplicationId, CandidateName: displayCandidateName(detail), JobTitle: detail.JobTitle, Status: detail.Status, SessionId: sessionID}, nil
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
		title = "新对话"
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

func directHRAssistantReply(message string) (string, bool) {
	normalized := normalizeDirectHRMessage(message)
	if normalized == "" {
		return "", false
	}

	greetings := map[string]struct{}{
		"hi": {}, "hello": {}, "hey": {},
		"你好": {}, "您好": {}, "哈喽": {}, "嗨": {},
		"在吗": {}, "在不在": {},
		"早上好": {}, "上午好": {}, "中午好": {}, "下午好": {}, "晚上好": {},
	}
	if _, ok := greetings[normalized]; ok {
		return "你好，我是 HR 端的 AI 数据助手。你可以问我岗位、投递、候选人、招聘趋势等数据，也可以让我分析候选人简历或辅助生成状态变更确认。", true
	}

	helpRequests := map[string]struct{}{
		"help": {}, "帮助": {}, "使用说明": {},
		"你是谁": {}, "你能做什么": {}, "你可以做什么": {},
		"有什么功能": {}, "有哪些功能": {}, "怎么用": {}, "如何使用": {},
	}
	if _, ok := helpRequests[normalized]; ok {
		return "我是 HR 端的 AI 数据助手，可以帮你查询岗位和投递数据、查看候选人情况、分析简历与岗位匹配度、统计招聘趋势，并在你明确要求时生成通过或淘汰的待确认操作。", true
	}

	thanks := map[string]struct{}{
		"谢谢": {}, "谢谢你": {}, "感谢": {}, "好的": {}, "好": {}, "ok": {}, "嗯": {},
	}
	if _, ok := thanks[normalized]; ok {
		return "不客气。需要查询岗位、候选人或投递数据时，直接告诉我就行。", true
	}

	return "", false
}

func normalizeDirectHRMessage(message string) string {
	text := strings.ToLower(strings.TrimSpace(message))
	replacer := strings.NewReplacer(
		" ", "", "\t", "", "\n", "", "\r", "",
		"。", "", ".", "", "！", "", "!", "", "？", "", "?", "",
		"，", "", ",", "", "、", "", ";", "", "；", "",
		"～", "", "~", "", "呀", "", "啊", "", "呢", "",
	)
	return replacer.Replace(text)
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

func (s *AIService) generateApplicationAnalysisMessage(sessionID, hrID int64, input ai.ApplicationAnalysisInput) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	log := logger.With(zap.Int64("session_id", sessionID), zap.Int64("hr_id", hrID))
	log.Info("async analysis started")
	start := time.Now()
	reply, err := s.ai.GenerateApplicationAnalysis(ctx, input, nil)
	if err != nil {
		log.Error("async analysis failed", zap.Error(err), zap.Duration("elapsed", time.Since(start)))
		reply = fmt.Sprintf("简历分析暂时没有完成：%s。你可以稍后在当前会话中继续追问，或重新发起 AI 分析。", err.Error())
	}
	_ = s.chats.Add(ctx, &model.AIChatHistory{SessionID: sessionID, HrID: hrID, Role: "assistant", Content: reply})
	log.Info("async analysis completed", zap.Duration("elapsed", time.Since(start)), zap.Int("reply_len", len([]rune(reply))))
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
func (s *AIService) maybeWriteMemory(hrID, sessionID, applicationID int64, userMsg, assistantReply string, metadata ai.ToolMetadata) {
	defer func() {
		if r := recover(); r != nil {
			logger.L().Error("memory write panic recovered", zap.Any("panic", r), zap.Int64("hr_id", hrID))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Rule 1: Detect explicit HR preferences in user message.
	if pref := detectPreference(userMsg); pref != "" {
		s.writeMemory(ctx, hrID, "hr", 0, "preference", pref, "user", 0.9)
	}

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

// detectPreference checks if the user message contains an explicit preference statement.
// This is a keyword-based rule; can be replaced with LLM classification later.
func detectPreference(message string) string {
	prefIndicators := []string{
		"优先", "偏好", "更看重", "不太看重", "不喜欢", "更喜欢",
		"以后", "未来", "之后都", "以后都",
		"重点关注", "不关注", "不用看",
	}
	text := message
	hasIndicator := false
	for _, kw := range prefIndicators {
		if strings.Contains(text, kw) {
			hasIndicator = true
			break
		}
	}
	if !hasIndicator {
		return ""
	}
	// Only capture if message is substantial enough (more than just the keyword itself).
	runes := []rune(strings.TrimSpace(message))
	if len(runes) < 10 {
		return ""
	}
	return "HR 表达了以下偏好：" + strings.TrimSpace(message)
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
