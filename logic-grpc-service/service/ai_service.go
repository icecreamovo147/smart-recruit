package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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
	agentRuns       *repository.AgentRunRepo
	memories        *repository.MemoryRepo
	contextBuilder  *AgentContextBuilder
	candidateAI     *CandidateAIService
	usageLogs       *repository.UsageLogRepo
	usageAuditCtx   *repository.UsageAuditContextRepo
	authzRepo       *repository.AuthzRepo
	agentRuntime    string
	authz           *ServiceAuthorizer
	llmConfigSvc    *LlmConfigService // for runtime model selection
	agentConfigRepo *repository.AgentConfigRepo
	promptRepo      *repository.PromptTemplateRepo
	mcpSvc          *MCPService
	skillSvc        *SkillService
	agentSkillRepo  agentSkillLister
	cachedADKTools  []tool.BaseTool // lazy-initialized, shared across requests
	cachedToolsMu   sync.Mutex      // guards cachedADKTools init and invalidation
	usageBuilder    *ContextUsageBuilder
}

type runtimeAIClient struct {
	client        *ai.Client
	modelID       *int64
	modelName     string
	auditProvider string
}

func NewAIService(
	chats *repository.ChatRepo,
	applications *repository.ApplicationRepo,
	jobs *repository.JobRepo,
	resumes *repository.ResumeRepo,
	summaries *repository.SessionSummaryRepo,
	toolTraces *repository.ToolTraceRepo,
	agentRuns *repository.AgentRunRepo,
	memories *repository.MemoryRepo,
	ossClient oss.Storage,
	aiClient *ai.Client,
	toolExecutor *ai.ToolExecutor,
	contextBuilder *AgentContextBuilder,
	candidateAI *CandidateAIService,
	usageLogs *repository.UsageLogRepo,
	usageAuditCtx *repository.UsageAuditContextRepo,
	authzRepo *repository.AuthzRepo,
	agentRuntime string,
	authz *ServiceAuthorizer,
	llmConfigSvc *LlmConfigService,
	agentConfigRepo *repository.AgentConfigRepo,
	promptRepo *repository.PromptTemplateRepo,
	mcpSvc *MCPService,
	skillSvc *SkillService,
	agentSkillRepo agentSkillLister,
) *AIService {
	return &AIService{
		chats: chats, applications: applications, jobs: jobs, resumes: resumes,
		summaries: summaries, toolTraces: toolTraces, agentRuns: agentRuns, memories: memories,
		oss: ossClient, ai: aiClient, toolExecutor: toolExecutor,
		contextBuilder: contextBuilder, candidateAI: candidateAI,
		usageLogs:       usageLogs,
		usageAuditCtx:   usageAuditCtx,
		authzRepo:       authzRepo,
		agentRuntime:    agentRuntime,
		authz:           authz,
		llmConfigSvc:    llmConfigSvc,
		agentConfigRepo: agentConfigRepo,
		promptRepo:      promptRepo,
		mcpSvc:          mcpSvc,
		skillSvc:        skillSvc,
		agentSkillRepo:  agentSkillRepo,
		usageBuilder:    NewContextUsageBuilder(),
	}
}

// writeHRUsageAudit writes both the usage log and the RBAC auth context for an HR AI operation.
// The usage log is created synchronously so the returned ID can be linked to the auth context record.
func (s *AIService) writeHRUsageAudit(ctx context.Context, entry AuditLogEntry, hrID int64, resourceType string, resourceID int64) {
	usageLogID := createUsageLogSync(ctx, s.usageLogs, entry)

	if s.usageAuditCtx != nil && s.authzRepo != nil {
		ua := uint64(hrID)
		roleKeys, _ := s.authzRepo.GetUserRoles(ctx, ua)
		scopeKeys, _ := s.authzRepo.GetUserScopeKeys(ctx, ua)
		if roleKeys == nil {
			roleKeys = []string{}
		}
		if scopeKeys == nil {
			scopeKeys = []string{}
		}

		writeAIUsageAuthContext(ctx, s.usageAuditCtx, AIUsageAuthContextEntry{
			UsageLogID:    uint64(usageLogID),
			ActorUserID:   ua,
			AccountType:   "staff",
			RoleKeys:      roleKeys,
			PermissionKey: "ai.hr.use",
			ScopeKeys:     scopeKeys,
			ResourceType:  resourceType,
			ResourceID:    uint64(resourceID),
			Decision:      "allowed",
			RequestID:     entry.RequestID,
		})
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

	runtimeCfg := s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent", req.GetSkillCapabilityKeys())
	runtimeClient, err := s.resolveRuntimeAIClient(ctx, req.GetModelId(), runtimeCfg)
	if err != nil {
		s.recordFailedAgentRun(ctx, req, session, requestedModelIDPtr(req.GetModelId()), s.defaultRuntimeModelName(), "model_resolution_failed", err)
		return &pb.ChatResponse{Code: errs.ErrBadRequest, Msg: fmt.Sprintf("模型不可用：%v", err), SessionId: session.ID}, nil
	}

	var replyBuilder strings.Builder
	var contextUsage *pb.ContextUsageInfo
	reply, metadata, err := s.runToolCallingChatWithUsage(ctx, req, session, runtimeClient.modelID, runtimeClient.modelName, runtimeCfg, func(delta string) error {
		replyBuilder.WriteString(delta)
		return nil
	}, nil, func(usage *pb.ContextUsageInfo) error {
		contextUsage = usage
		return nil
	}, runtimeClient.client)
	if err != nil {
		if isCanceledError(err) {
			partial := strings.TrimSpace(replyBuilder.String())
			if partial == "" {
				partial = reply
			}
			log.Info("non-streaming chat canceled, returning partial reply", zap.Int("partial_chars", len([]rune(partial))))
			s.writeHRUsageAudit(ctx, AuditLogEntry{
				UserID: req.HrId, Role: 2, ServiceType: "ai_chat",
				Endpoint: "/hr/ai/chat", Provider: runtimeClient.auditProvider, Model: runtimeClient.modelName,
				RequestChars: inputChars, ResponseChars: len([]rune(partial)), TokenUsageTotal: tokenUsageTotal(metadata.BillingTokenUsage), Status: "timeout", CostMs: int(time.Since(startTime).Milliseconds()),
			}, req.HrId, "ai", req.ApplicationId)
			return &pb.ChatResponse{Code: errs.OK, Msg: "success", Reply: partial, CreatedAt: formatTime(time.Now()), SessionId: session.ID, ContextUsage: contextUsage}, nil
		}
		s.writeHRUsageAudit(ctx, AuditLogEntry{
			UserID: req.HrId, Role: 2, ServiceType: "ai_chat",
			Endpoint: "/hr/ai/chat", Provider: runtimeClient.auditProvider, Model: runtimeClient.modelName,
			RequestChars: inputChars, Status: "error", CostMs: int(time.Since(startTime).Milliseconds()),
		}, req.HrId, "ai", req.ApplicationId)
		return nil, wrapAIError(err)
	}

	now := time.Now()
	outputChars := len([]rune(reply))
	s.writeHRUsageAudit(ctx, AuditLogEntry{
		UserID: req.HrId, Role: 2, ServiceType: "ai_chat",
		Endpoint: "/hr/ai/chat", Provider: runtimeClient.auditProvider, Model: runtimeClient.modelName,
		RequestChars: inputChars, ResponseChars: outputChars, TokenUsageTotal: tokenUsageTotal(metadata.BillingTokenUsage), CostMs: int(time.Since(startTime).Milliseconds()),
	}, req.HrId, "ai", req.ApplicationId)
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
	resp.ContextUsage = contextUsage
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

	runtimeCfg := s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent", req.GetSkillCapabilityKeys())
	runtimeClient, err := s.resolveRuntimeAIClient(ctx, req.GetModelId(), runtimeCfg)
	if err != nil {
		s.recordFailedAgentRun(ctx, req, session, requestedModelIDPtr(req.GetModelId()), s.defaultRuntimeModelName(), "model_resolution_failed", err)
		return stream.Send(&pb.ChatStreamResponse{Code: errs.ErrBadRequest, Msg: fmt.Sprintf("模型不可用：%v", err), Done: true, SessionId: session.ID})
	}

	// Send model/agent info to the frontend status bar.
	_ = stream.Send(&pb.ChatStreamResponse{
		Code:         errs.OK,
		EventType:    "model_info",
		EventMessage: runtimeClient.modelName,
		SessionId:    session.ID,
	})

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
	contextUsageSender := func(usage *pb.ContextUsageInfo) error {
		return stream.Send(&pb.ChatStreamResponse{
			Code:         errs.OK,
			Msg:          "success",
			EventType:    "context_usage",
			ContextUsage: usage,
			SessionId:    session.ID,
		})
	}
	reply, metadata, err := s.runToolCallingChatWithUsage(ctx, req, session, runtimeClient.modelID, runtimeClient.modelName, runtimeCfg, func(delta string) error {
		return stream.Send(&pb.ChatStreamResponse{Code: errs.OK, Msg: "success", Delta: delta, SessionId: session.ID})
	}, statusSender, contextUsageSender, runtimeClient.client)
	if err != nil {
		s.writeHRUsageAudit(ctx, AuditLogEntry{
			UserID: req.HrId, Role: 2, ServiceType: "ai_chat",
			Endpoint: "/hr/ai/chat/stream", Provider: runtimeClient.auditProvider, Model: runtimeClient.modelName,
			RequestChars: inputChars, Status: "error", CostMs: int(time.Since(startTime).Milliseconds()),
		}, req.HrId, "ai", req.ApplicationId)
		return err
	}

	s.writeHRUsageAudit(ctx, AuditLogEntry{
		UserID: req.HrId, Role: 2, ServiceType: "ai_chat",
		Endpoint: "/hr/ai/chat/stream", Provider: runtimeClient.auditProvider, Model: runtimeClient.modelName,
		RequestChars: inputChars, ResponseChars: len([]rune(reply)), TokenUsageTotal: tokenUsageTotal(metadata.BillingTokenUsage), CostMs: int(time.Since(startTime).Milliseconds()),
	}, req.HrId, "ai", req.ApplicationId)

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
func (s *AIService) runToolCallingChat(ctx context.Context, req *pb.ChatRequest, session *model.AIChatSession, modelID *int64, modelName string, runtimeCfg *agentRuntimeConfig, onDelta func(string) error, onStatus func(eventType, eventMessage, errorType, toolName string) error, aiClient *ai.Client) (reply string, metadata ai.ToolMetadata, err error) {
	return s.runToolCallingChatWithUsage(ctx, req, session, modelID, modelName, runtimeCfg, onDelta, onStatus, nil, aiClient)
}

// runToolCallingChatWithUsage is like runToolCallingChat but additionally
// accepts an onContextUsage callback to emit context usage info events.
func (s *AIService) runToolCallingChatWithUsage(ctx context.Context, req *pb.ChatRequest, session *model.AIChatSession, modelID *int64, modelName string, runtimeCfg *agentRuntimeConfig, onDelta func(string) error, onStatus func(eventType, eventMessage, errorType, toolName string) error, onContextUsage func(*pb.ContextUsageInfo) error, aiClient *ai.Client) (reply string, metadata ai.ToolMetadata, err error) {
	recorder := s.startAgentRun(ctx, req, session, modelID, modelName, runtimeCfg)
	fallbackObserved := false
	var processContent strings.Builder
	effectiveOnStatus := func(eventType, eventMessage, errorType, toolName string) error {
		if eventType == "process_delta" {
			processContent.WriteString(eventMessage)
		} else if eventType == "process_clear" {
			processContent.Reset()
		}
		if (eventType == "fallback" || eventType == "partial_done") && recorder != nil && !fallbackObserved {
			fallbackObserved = true
			recorder.recordFallback(ctx, eventType, errorType, eventMessage, len(metadata.ToolTraces))
		}
		if onStatus != nil {
			return onStatus(eventType, eventMessage, errorType, toolName)
		}
		return nil
	}
	sendAgentRunStatus(effectiveOnStatus, "agent_run_started", "Agent run 已开始", "", "")
	sendAgentRunStatus(effectiveOnStatus, "model_selected", modelName, "", "")
	sendAgentRunStatus(effectiveOnStatus, "planning", "正在规划本轮执行", "", "")
	sendAgentRunStatus(effectiveOnStatus, "capability_selected", "已选择可用能力", "", "")

	// Phase 1-5: Build agent context with all memory layers.
	actx, err := s.contextBuilder.Build(ctx, AgentContextInput{
		HrID:           req.HrId,
		SessionID:      session.ID,
		ApplicationID:  req.ApplicationId,
		CurrentMessage: req.Message,
	})
	if err != nil {
		if recorder != nil {
			recorder.finish(ctx, agentRunStatusFailed, "", "context_build_failed", err.Error())
		}
		return "", metadata, err
	}
	if recorder != nil {
		recorder.setSelectedMemoryIDs(ctx, selectedMemoryIDs(actx.LongTermMemories))
	}

	// If agent config has a bound prompt template, it takes priority.
	if runtimeCfg == nil {
		runtimeCfg = s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent")
	}
	if runtimeCfg.SystemPrompt != "" {
		actx.SystemPromptTemplate = runtimeCfg.SystemPrompt
	}

	logger.L().Info("[AI问答] 用户提问",
		zap.String("question", req.Message),
		zap.Int64("hr_id", req.HrId),
		zap.Int64("session_id", session.ID),
	)

	var historyToolNames []string
	if runtimeCfg != nil && runtimeCfg.HasConfig {
		historyToolNames = runtimeCfg.ToolNames
	} else {
		historyToolNames = agentRunToolInfoNames(ai.RecruitingTools())
	}
	selectedSkillsForHistory, err := selectAgentSkills(ctx, s.agentSkillRepo, "hr_recruiting_agent", req.GetMessage(), req.GetAgentSkillIds(), agentSkillAvailableCapabilities(runtimeCfg, historyToolNames))
	if err != nil {
		logger.L().Warn("select Agent Skills for history failed", zap.Error(err))
		selectedSkillsForHistory = nil
	}
	selectedSkillsForHistory = manualAgentSkills(selectedSkillsForHistory)
	userAlreadyPersisted := currentMessageAlreadyPersisted(actx, req.Message)
	messages := buildToolCallingMessages(actx, req.Message)
	var contextUsage *pb.ContextUsageInfo

	// Emit the session-level accumulated context usage. This is intentionally
	// based on persisted chat history plus the in-flight message, not on
	// transient tool-calling message snapshots that can shrink between rounds.
	contextUsage = s.buildSessionContextUsage(ctx, req.HrId, session.ID, actx, messages, modelID, modelName, "initial", "estimator", req.Message, "")
	if contextUsage != nil {
		if onContextUsage != nil {
			usageInfo := contextUsage
			_ = onContextUsage(usageInfo)
		}
	}
	contextUsageJSON := marshalContextUsageJSON(contextUsage)

	// Save user message before the model call so it persists even on cancel.
	if !userAlreadyPersisted {
		userHistory := &model.AIChatHistory{
			SessionID:           session.ID,
			HrID:                req.HrId,
			Role:                "user",
			Content:             req.Message,
			AgentSkillIDsJSON:   marshalInt64Slice(selectedAgentSkillIDs(selectedSkillsForHistory)),
			AgentSkillNamesJSON: marshalStringSlice(selectedAgentSkillNames(selectedSkillsForHistory)),
		}
		if err := s.chats.Add(ctx, userHistory); err != nil {
			if recorder != nil {
				recorder.finish(ctx, agentRunStatusFailed, "", "persist_failed", err.Error())
				sendAgentRunStatus(effectiveOnStatus, "agent_run_done", "Agent run 保存失败", "persist_failed", "")
			}
			return "", metadata, err
		}
		if recorder != nil && userHistory.ID > 0 {
			_ = s.agentRuns.UpdateRunMessageID(ctx, recorder.runID, uint64(userHistory.ID))
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

	if aiClient == nil {
		aiClient = s.ai
	}
	if recorder != nil {
		recorder.markRunning(ctx)
	}
	usageUpdater := func(_ []*schema.Message, stage string) error {
		usageInfo := s.buildSessionContextUsage(ctx, req.HrId, session.ID, actx, messages, modelID, modelName, stage, "estimator", "", "")
		if usageInfo != nil {
			contextUsage = usageInfo
			contextUsageJSON = marshalContextUsageJSON(contextUsage)
			if onContextUsage != nil {
				return onContextUsage(usageInfo)
			}
		}
		return nil
	}
	if s.agentRuntime == "adk" {
		reply, metadata, err = s.runADKChat(ctx, req, session, messages, modelID, modelName, runtimeCfg, wrappedDelta, effectiveOnStatus, usageUpdater, aiClient, recorder)
	} else {
		reply, metadata, err = s.runLegacyChat(ctx, req, session, messages, modelID, modelName, runtimeCfg, wrappedDelta, effectiveOnStatus, usageUpdater, aiClient, recorder)
	}
	if err != nil {
		if isCanceledError(err) {
			partial := strings.TrimSpace(partialReply.String())
			if partial != "" {
				contextUsage = s.buildSessionContextUsage(ctx, req.HrId, session.ID, actx, messages, modelID, modelName, "final", "estimator", "", partial)
				contextUsageJSON = marshalContextUsageJSON(contextUsage)
				saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = s.chats.Add(saveCtx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "assistant", Content: partial + "\n\n（回复已中断）", ProcessContent: processContent.String(), ContextUsageJSON: contextUsageJSON, ModelID: modelID, ModelName: modelName})
			}
			logger.L().Info("chat canceled, partial reply saved if non-empty", zap.Int("partial_chars", len(partial)))
			if recorder != nil {
				recorder.recordRecovery(ctx, "canceled", "用户中断或连接关闭")
				recorder.finish(ctx, agentRunStatusCanceled, partial, "canceled", err.Error())
				sendAgentRunStatus(effectiveOnStatus, "agent_run_done", "Agent run 已取消", "canceled", "")
			}
			return reply, metadata, err
		}
		// Phase 3: deterministic fallback from collected tool traces when LLM fails.
		if len(metadata.ToolTraces) > 0 {
			fallback := ai.BuildHRFallbackReply(metadata.ToolTraces)
			aiErr := ai.ClassifyAIError(err)
			_ = effectiveOnStatus("fallback", "已基于已查询数据给出保守回复", string(aiErr.Type), "")
			_ = effectiveOnStatus("partial_done", "已基于已查询数据给出保守回复", string(aiErr.Type), "")
			if recorder != nil {
				if !fallbackObserved {
					recorder.recordFallback(ctx, "llm_failed_after_tools", string(aiErr.Type), err.Error(), len(metadata.ToolTraces))
				}
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
			contextUsage = s.buildSessionContextUsage(ctx, req.HrId, session.ID, actx, messages, modelID, modelName, "final", "estimator", "", fallback)
			contextUsageJSON = marshalContextUsageJSON(contextUsage)
			_ = s.chats.Add(saveCtx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "assistant", Content: fallback, ProcessContent: processContent.String(), ContextUsageJSON: contextUsageJSON, ModelID: modelID, ModelName: modelName})
			if recorder != nil {
				recorder.finish(ctx, agentRunStatusPartial, fallback, string(aiErr.Type), err.Error())
				sendAgentRunStatus(effectiveOnStatus, "agent_run_done", "Agent run 已部分完成", string(aiErr.Type), "")
			}
			return fallback, metadata, nil
		}
		if recorder != nil {
			aiErr := ai.ClassifyAIError(err)
			recorder.finish(ctx, agentRunStatusFailed, reply, string(aiErr.Type), err.Error())
			sendAgentRunStatus(effectiveOnStatus, "agent_run_done", "Agent run 失败", string(aiErr.Type), "")
		}
		return reply, metadata, wrapAIError(err)
	}
	logger.L().Info("[AI问答] LLM最终回复",
		append(ai.TokenUsageLogFields(metadata.ContextTokenUsage),
			zap.String("reply", reply),
			zap.Int("reply_chars", len([]rune(reply))),
		)...,
	)

	// Save full assistant reply on success.
	estimatorFinal := s.buildSessionContextUsage(ctx, req.HrId, session.ID, actx, messages, modelID, modelName, "final", "estimator", "", reply)
	contextUsage = s.buildProviderFinalContextUsage(ctx, estimatorFinal, metadata.ContextTokenUsage)
	contextUsageJSON = marshalContextUsageJSON(contextUsage)
	if contextUsage != nil && onContextUsage != nil {
		if err := onContextUsage(contextUsage); err != nil {
			return reply, metadata, err
		}
	}
	if err := s.chats.Add(ctx, &model.AIChatHistory{SessionID: session.ID, HrID: req.HrId, Role: "assistant", Content: reply, ProcessContent: processContent.String(), ContextUsageJSON: contextUsageJSON, ModelID: modelID, ModelName: modelName}); err != nil {
		if recorder != nil {
			recorder.finish(ctx, agentRunStatusFailed, reply, "persist_failed", err.Error())
			sendAgentRunStatus(effectiveOnStatus, "agent_run_done", "Agent run 保存失败", "persist_failed", "")
		}
		return reply, metadata, err
	}

	// Async refresh summary if needed.
	go s.maybeRefreshSummary(session.ID, req.HrId)

	// Async write long-term memory if applicable.
	go s.maybeWriteMemory(req.HrId, req.ApplicationId, reply, metadata)
	if recorder != nil {
		status := agentRunStatusSucceeded
		if fallbackObserved {
			status = agentRunStatusPartial
		}
		recorder.finish(ctx, status, reply, "", "")
		sendAgentRunStatus(effectiveOnStatus, "agent_run_done", "Agent run 已完成", "", "")
	}

	return reply, metadata, nil
}

// runADKChat executes the HR AI conversation through the Eino ADK ChatModelAgent.
func (s *AIService) runADKChat(
	ctx context.Context,
	req *pb.ChatRequest,
	session *model.AIChatSession,
	messages []*schema.Message,
	modelID *int64,
	modelName string,
	runtimeCfg *agentRuntimeConfig,
	onDelta func(string) error,
	onStatus func(string, string, string, string) error,
	onMessagesUpdated ai.MessageUpdateCallback,
	aiClient *ai.Client,
	recorder *agentRunRecorder,
) (string, ai.ToolMetadata, error) {
	if req.HrId <= 0 {
		return "", ai.ToolMetadata{}, fmt.Errorf("hrID must be positive, got %d", req.HrId)
	}

	// Thread-safe lazy-init of cached tools via getOrInitADKTools.
	adkTools, err := s.getOrInitADKTools()
	if err != nil {
		logger.L().Warn("[ADK降级] 工具创建失败，自动切换到 Legacy 路径", zap.Error(err))
		return s.runLegacyChat(ctx, req, session, messages, modelID, modelName, runtimeCfg, onDelta, onStatus, onMessagesUpdated, aiClient, recorder)
	}

	state := &ai.AgentRunState{}

	// Store per-request values in context so tool closures can retrieve them
	// without capturing them (enabling tool caching).
	ctx = ai.WithOwnerID(ctx, req.HrId)
	ctx = ai.WithAgentRunState(ctx, state)

	if runtimeCfg == nil {
		runtimeCfg = s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent", req.GetSkillCapabilityKeys())
	}
	traceFn := func(toolCallID, toolName, argsJSON, resultContent string, duration time.Duration, execErr error) {
		stepID := uint64(0)
		if recorder != nil {
			source, key := resolveCapabilityForTool(runtimeCfg, toolName)
			stepID = recorder.recordTool(ctx, toolCallID, toolName, argsJSON, resultContent, source, key, duration, execErr)
		}
		go s.recordToolTrace(session.ID, req.HrId, toolCallID, toolName, argsJSON, resultContent, duration, recorderRunID(recorder), stepID, execErr)
	}

	if runtimeCfg.HasConfig {
		if len(runtimeCfg.ToolNames) > 0 {
			adkTools = filterToolsByName(adkTools, runtimeCfg.ToolNames)
		} else {
			adkTools = nil // agent config exists but all tools disabled
		}
	}
	availableCapabilities := agentSkillAvailableCapabilities(runtimeCfg, adkToolNames(ctx, adkTools))
	maxIterations := 0
	if runtimeCfg.MaxIterations > 0 {
		maxIterations = runtimeCfg.MaxIterations
	}

	if s.mcpSvc != nil && len(runtimeCfg.MCPCapabilityKeys) > 0 {
		if mcpTools, err := s.mcpSvc.CollectBoundMCPCallableTools(ctx, runtimeCfg.MCPCapabilityKeys); err == nil && len(mcpTools) > 0 {
			merged := make([]tool.BaseTool, 0, len(adkTools)+len(mcpTools))
			merged = append(merged, adkTools...)
			merged = append(merged, mcpTools...)
			adkTools = merged
			availableCapabilities = addRuntimeCapabilityRefs(availableCapabilities, "mcp", runtimeCfg.MCPCapabilityKeys)
		}
	}

	instruction := extractSystemInstruction(messages)
	if s.skillSvc != nil && len(runtimeCfg.SkillCapabilityKeys) > 0 {
		if skillTools, skillInstructions, err := s.skillSvc.CollectBoundSkillCallableTools(ctx, runtimeCfg.SkillCapabilityKeys); err == nil {
			if len(skillTools) > 0 {
				merged := make([]tool.BaseTool, 0, len(adkTools)+len(skillTools))
				merged = append(merged, adkTools...)
				merged = append(merged, skillTools...)
				adkTools = merged
				availableCapabilities = addRuntimeCapabilityRefs(availableCapabilities, "skill", runtimeCfg.SkillCapabilityKeys)
			}
			instruction = appendSkillInstructions(instruction, skillInstructions)
		} else {
			logger.L().Warn("collect bound SKILL callable tools failed", zap.Error(err))
		}
	}
	availableToolNames := adkToolNames(ctx, adkTools)
	availableCapabilities = addRuntimeToolCapabilities(availableCapabilities, availableToolNames)
	agentSkills, err := selectAgentSkills(ctx, s.agentSkillRepo, "hr_recruiting_agent", req.GetMessage(), req.GetAgentSkillIds(), availableCapabilities)
	if err != nil {
		logger.L().Warn("select Agent Skills failed", zap.Error(err))
	} else if len(agentSkills) > 0 {
		instruction = appendAgentSkillInstructionBlock(instruction, renderAgentSkillInstructionBlock(agentSkills))
		if recorder != nil {
			recorder.setSelectedAgentSkills(ctx, agentSkills)
		}
		logSelectedAgentSkills(agentSkills)
	}
	planner := ai.NewRecruitingPlanner()
	availableToolNames = adkToolNames(ctx, adkTools)
	plan := planner.Plan(ai.RecruitingPlannerInput{
		Message:        req.GetMessage(),
		AvailableTools: availableToolNames,
		ApplicationID:  req.GetApplicationId(),
	})
	plan = applyAgentSkillPlannerConstraints(plan, agentSkills, availableToolNames)
	if recorder != nil {
		recorder.recordRecruitingPlan(ctx, plan)
	}
	instruction = appendRecruitingPlannerInstructionBlock(instruction, plan.InstructionBlock())
	logger.L().Info("[Planner] HR Agent structured plan selected",
		zap.String("intent", plan.Intent),
		zap.Strings("required_tools", plan.RequiredTools),
	)
	logger.L().Info("[提示词诊断] HR Agent 当前使用的 System Prompt",
		zap.Int("总字符数", len([]rune(instruction))),
		zap.String("前200字符", truncateString(instruction, 200)),
		zap.String("后200字符", tailString(instruction, 200)),
	)

	return aiClient.ChatWithADKAgent(ctx, ai.AgentRunInput{
		AgentName:         "hr_recruiting_agent",
		Instruction:       instruction,
		Messages:          messages,
		Tools:             adkTools,
		MaxIterations:     maxIterations,
		OwnerID:           req.HrId,
		SessionID:         session.ID,
		State:             state,
		OnMessagesUpdated: onMessagesUpdated,
	}, onDelta, traceFn, onStatus)
}

// runLegacyChat delegates to the existing ChatWithTools (Eino Tool Calling) path.
func (s *AIService) runLegacyChat(
	ctx context.Context,
	req *pb.ChatRequest,
	session *model.AIChatSession,
	messages []*schema.Message,
	modelID *int64,
	modelName string,
	runtimeCfg *agentRuntimeConfig,
	onDelta func(string) error,
	onStatus func(string, string, string, string) error,
	onMessagesUpdated ai.MessageUpdateCallback,
	aiClient *ai.Client,
	recorder *agentRunRecorder,
) (string, ai.ToolMetadata, error) {
	if runtimeCfg == nil {
		runtimeCfg = s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent", req.GetSkillCapabilityKeys())
	}
	tools := ai.RecruitingTools()
	if runtimeCfg.HasConfig {
		if len(runtimeCfg.ToolNames) > 0 {
			tools = filterToolInfosByName(tools, runtimeCfg.ToolNames)
		} else {
			tools = nil // agent config exists but all tools disabled
		}
	}
	availableCapabilities := agentSkillAvailableCapabilities(runtimeCfg, agentRunToolInfoNames(tools))

	if s.mcpSvc != nil && len(runtimeCfg.MCPCapabilityKeys) > 0 {
		if mcpTools, err := s.mcpSvc.CollectBoundMCPToolInfos(ctx, runtimeCfg.MCPCapabilityKeys); err == nil && len(mcpTools) > 0 {
			tools = append(tools, mcpTools...)
			availableCapabilities = addRuntimeCapabilityRefs(availableCapabilities, "mcp", runtimeCfg.MCPCapabilityKeys)
		}
	}
	var executor ai.ToolRunner = s.toolExecutor
	if s.skillSvc != nil && len(runtimeCfg.SkillCapabilityKeys) > 0 {
		if skillCallableTools, skillInstructions, err := s.skillSvc.CollectBoundSkillCallableTools(ctx, runtimeCfg.SkillCapabilityKeys); err == nil {
			if len(skillCallableTools) > 0 {
				for _, t := range skillCallableTools {
					info, err := t.Info(ctx)
					if err == nil {
						tools = append(tools, info)
					}
				}
				executor = &compositeToolRunner{
					primary:    executor,
					skillTools: skillInvokableToolsByName(ctx, skillCallableTools),
				}
				availableCapabilities = addRuntimeCapabilityRefs(availableCapabilities, "skill", runtimeCfg.SkillCapabilityKeys)
			}
			messages = appendSkillInstructionsToMessages(messages, skillInstructions)
		} else {
			logger.L().Warn("collect bound SKILL tool infos failed", zap.Error(err))
		}
	}
	availableToolNames := agentRunToolInfoNames(tools)
	availableCapabilities = addRuntimeToolCapabilities(availableCapabilities, availableToolNames)
	agentSkills, err := selectAgentSkills(ctx, s.agentSkillRepo, "hr_recruiting_agent", req.GetMessage(), req.GetAgentSkillIds(), availableCapabilities)
	if err != nil {
		logger.L().Warn("select Agent Skills failed", zap.Error(err))
	} else if len(agentSkills) > 0 {
		messages = appendAgentSkillInstructionBlockToMessages(messages, renderAgentSkillInstructionBlock(agentSkills))
		if recorder != nil {
			recorder.setSelectedAgentSkills(ctx, agentSkills)
		}
		logSelectedAgentSkills(agentSkills)
	}
	planner := ai.NewRecruitingPlanner()
	availableToolNames = agentRunToolInfoNames(tools)
	plan := planner.Plan(ai.RecruitingPlannerInput{
		Message:        req.GetMessage(),
		AvailableTools: availableToolNames,
		ApplicationID:  req.GetApplicationId(),
	})
	plan = applyAgentSkillPlannerConstraints(plan, agentSkills, availableToolNames)
	if recorder != nil {
		recorder.recordRecruitingPlan(ctx, plan)
	}
	messages = appendRecruitingPlannerInstructionBlockToMessages(messages, plan.InstructionBlock())

	traceFn := func(toolCallID, toolName, argsJSON, resultContent string, duration time.Duration, execErr error) {
		stepID := uint64(0)
		if recorder != nil {
			source, key := resolveCapabilityForTool(runtimeCfg, toolName)
			stepID = recorder.recordTool(ctx, toolCallID, toolName, argsJSON, resultContent, source, key, duration, execErr)
		}
		go s.recordToolTrace(session.ID, req.HrId, toolCallID, toolName, argsJSON, resultContent, duration, recorderRunID(recorder), stepID, execErr)
	}

	return aiClient.ChatWithToolsWithMessageCallback(ctx, messages, tools, executor, req.HrId, onDelta, traceFn, onStatus, onMessagesUpdated)
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

func appendSkillInstructions(base string, skillInstructions []string) string {
	if len(skillInstructions) == 0 {
		return base
	}
	addition := "Bound SKILL instructions:\n" + strings.Join(skillInstructions, "\n\n")
	if strings.TrimSpace(base) == "" {
		return addition
	}
	return base + "\n\n" + addition
}

func appendSkillInstructionsToMessages(messages []*schema.Message, skillInstructions []string) []*schema.Message {
	if len(skillInstructions) == 0 {
		return messages
	}
	addition := "Bound SKILL instructions:\n" + strings.Join(skillInstructions, "\n\n")
	copied := append([]*schema.Message(nil), messages...)
	for _, m := range copied {
		if m.Role == schema.System {
			m.Content = appendSkillInstructions(m.Content, skillInstructions)
			return copied
		}
	}
	return append([]*schema.Message{schema.SystemMessage(addition)}, copied...)
}

func appendRecruitingPlannerInstructionBlockToMessages(messages []*schema.Message, block string) []*schema.Message {
	block = strings.TrimSpace(block)
	if block == "" {
		return messages
	}
	copied := append([]*schema.Message(nil), messages...)
	for _, m := range copied {
		if m.Role == schema.System {
			m.Content = appendRecruitingPlannerInstructionBlock(m.Content, block)
			return copied
		}
	}
	return append([]*schema.Message{schema.SystemMessage(block)}, copied...)
}

func agentRunToolInfoNames(tools []*schema.ToolInfo) []string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		if t != nil && strings.TrimSpace(t.Name) != "" {
			names = append(names, t.Name)
		}
	}
	return names
}

type compositeToolRunner struct {
	primary    ai.ToolRunner
	skillTools map[string]tool.InvokableTool
}

func (r *compositeToolRunner) Execute(ctx context.Context, hrID int64, toolName string, args map[string]any) (ai.ToolResult, error) {
	if t, ok := r.skillTools[toolName]; ok {
		payload, _ := json.Marshal(args)
		result, err := t.InvokableRun(ctx, string(payload))
		return ai.ToolResult{Content: result}, err
	}
	return r.primary.Execute(ctx, hrID, toolName, args)
}

func skillInvokableToolsByName(ctx context.Context, tools []tool.BaseTool) map[string]tool.InvokableTool {
	result := map[string]tool.InvokableTool{}
	for _, t := range tools {
		invokable, ok := t.(tool.InvokableTool)
		if !ok {
			continue
		}
		info, err := t.Info(ctx)
		if err != nil || info == nil || info.Name == "" {
			continue
		}
		result[info.Name] = invokable
	}
	return result
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

	runtimeCfg := s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent")
	runtimeClient, err := s.resolveRuntimeAIClient(ctx, req.GetModelId(), runtimeCfg)
	if err != nil {
		return &pb.AnalyzeApplicationResponse{Code: errs.ErrBadRequest, Msg: fmt.Sprintf("模型不可用：%v", err)}, nil
	}

	result, err := runtimeClient.client.GenerateApplicationAnalysisWithUsage(ctx, input, nil)
	if err != nil {
		s.writeHRUsageAudit(ctx, AuditLogEntry{
			UserID: req.HrId, Role: 2, ServiceType: "ai_analyze",
			Endpoint: "/hr/ai/analyze-application", Provider: runtimeClient.auditProvider, Model: runtimeClient.modelName,
			RequestChars: inputChars, Status: "error", CostMs: int(time.Since(startTime).Milliseconds()),
		}, req.HrId, "application", req.ApplicationId)
		return nil, wrapAIError(err)
	}
	reply := result.Content
	s.writeHRUsageAudit(ctx, AuditLogEntry{
		UserID: req.HrId, Role: 2, ServiceType: "ai_analyze",
		Endpoint: "/hr/ai/analyze-application", Provider: runtimeClient.auditProvider, Model: runtimeClient.modelName,
		RequestChars: inputChars, ResponseChars: len([]rune(reply)), TokenUsageTotal: tokenUsageTotal(result.TokenUsage), CostMs: int(time.Since(startTime).Milliseconds()),
	}, req.HrId, "application", req.ApplicationId)
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
	return &pb.ChatSessionListResponse{Code: errs.OK, Msg: "success", Total: total, List: list, ModelName: s.ai.ModelName()}, nil
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

func sendAgentRunStatus(onStatus func(string, string, string, string) error, eventType, eventMessage, errorType, toolName string) {
	if onStatus == nil {
		return
	}
	_ = onStatus(eventType, eventMessage, errorType, toolName)
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
func (s *AIService) recordToolTrace(sessionID, hrID int64, toolCallID, toolName, argsJSON, resultContent string, duration time.Duration, runID, stepID uint64, execErr error) {
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
		SessionID:      uint64(sessionID),
		HrID:           uint64(hrID),
		AgentRunID:     optionalUint64(runID),
		AgentRunStepID: optionalUint64(stepID),
		ToolCallID:     toolCallID,
		ToolName:       toolName,
		ArgumentsJSON:  argsJSON,
		ResultJSON:     resultContent,
		ResultSummary:  resultSummary,
		Status:         status,
		DurationMs:     duration.Milliseconds(),
		ErrorMessage:   errMsg,
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

func recorderRunID(r *agentRunRecorder) uint64 {
	if r == nil {
		return 0
	}
	return r.runID
}

func optionalUint64(v uint64) *uint64 {
	if v == 0 {
		return nil
	}
	return &v
}

func toPBAgentRun(run model.AgentRun, steps []model.AgentRunStep) *pb.AgentRunItem {
	item := &pb.AgentRunItem{
		Id:           int64(run.ID),
		SessionId:    int64(run.SessionID),
		HrId:         int64(run.HrID),
		AgentType:    run.AgentType,
		AgentName:    run.AgentName,
		ModelName:    run.ModelName,
		Status:       run.Status,
		PlanJson:     run.PlanJSON,
		FinalAnswer:  run.FinalAnswer,
		ErrorType:    run.ErrorType,
		ErrorMessage: run.ErrorMessage,
		StartedAt:    formatTime(run.StartedAt),
		CreatedAt:    formatTime(run.CreatedAt),
		Steps:        make([]*pb.AgentRunStepItem, 0, len(steps)),
	}
	if run.MessageID != nil {
		item.MessageId = int64(*run.MessageID)
	}
	if run.HistoryID != nil {
		item.HistoryId = int64(*run.HistoryID)
	}
	if run.AgentID != nil {
		item.AgentId = int64(*run.AgentID)
	}
	if run.ModelID != nil {
		item.ModelId = int64(*run.ModelID)
	}
	if run.CompletedAt != nil {
		item.CompletedAt = formatTime(*run.CompletedAt)
	}
	for _, step := range steps {
		item.Steps = append(item.Steps, toPBAgentRunStep(step))
	}
	return item
}

func toPBAgentRunStep(step model.AgentRunStep) *pb.AgentRunStepItem {
	item := &pb.AgentRunStepItem{
		Id:               int64(step.ID),
		RunId:            int64(step.RunID),
		StepIndex:        int32(step.StepIndex),
		StepType:         step.StepType,
		CapabilitySource: step.CapabilitySource,
		CapabilityKey:    step.CapabilityKey,
		ToolName:         step.ToolName,
		InputJson:        desensitizeArgsJSON(step.InputJSON),
		OutputJson:       desensitizeResultContent(step.OutputJSON),
		Status:           step.Status,
		DurationMs:       step.DurationMs,
		ErrorMessage:     step.ErrorMessage,
		StartedAt:        formatTime(step.StartedAt),
		CreatedAt:        formatTime(step.CreatedAt),
	}
	if step.CompletedAt != nil {
		item.CompletedAt = formatTime(*step.CompletedAt)
	}
	return item
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

// buildContextUsage computes a ContextUsageInfo from the actual messages sent
// to the model and the resolved runtime model config.
// Returns nil if model config cannot be resolved.
func (s *AIService) buildContextUsage(ctx context.Context, messages []*schema.Message, modelID *int64, modelName string, stage, source string) *pb.ContextUsageInfo {
	if s.usageBuilder == nil {
		return nil
	}
	estimator := NewTokenEstimator()
	breakdown := estimator.EstimateMessages(messages)
	return s.buildContextUsageFromBreakdown(ctx, breakdown, modelID, modelName, stage, source)
}

// buildSessionContextUsage estimates the user-visible context size for the
// whole chat session. It deliberately ignores transient tool-calling messages:
// the displayed Context should represent accumulated session conversation,
// which should not drop just because an agent runtime rewrites its inner state.
func (s *AIService) buildSessionContextUsage(ctx context.Context, hrID, sessionID int64, actx *AgentContext, baseMessages []*schema.Message, modelID *int64, modelName string, stage, source, pendingUser, pendingAssistant string) *pb.ContextUsageInfo {
	if s.usageBuilder == nil {
		return nil
	}
	estimator := NewTokenEstimator()
	breakdown := &pb.ContextUsageBreakdown{}
	for _, msg := range baseMessages {
		if msg != nil && msg.Role == schema.System {
			breakdown.SystemPromptTokens = int32(estimator.EstimateText(msg.Content))
			break
		}
	}
	if actx != nil {
		if actx.SessionSummary != "" {
			breakdown.SummaryTokens = int32(estimator.EstimateText(actx.SessionSummary))
		}
		for _, memory := range actx.LongTermMemories {
			breakdown.MemoryTokens += int32(estimator.EstimateText(memory.Content))
		}
	}
	if s.chats != nil && sessionID > 0 {
		rows, err := s.chats.ListAllBySession(ctx, hrID, sessionID)
		if err == nil {
			for _, row := range rows {
				if row.Role == "user" && pendingUser != "" && strings.TrimSpace(row.Content) == strings.TrimSpace(pendingUser) {
					pendingUser = ""
				}
				breakdown.RecentMessageTokens += int32(estimator.EstimateText(row.Content))
			}
		}
	}
	if pendingUser != "" {
		breakdown.CurrentMessageTokens += int32(estimator.EstimateText(pendingUser))
	}
	if pendingAssistant != "" {
		breakdown.RecentMessageTokens += int32(estimator.EstimateText(pendingAssistant))
	}
	return s.buildContextUsageFromBreakdown(ctx, breakdown, modelID, modelName, stage, source)
}

func (s *AIService) buildContextUsageFromBreakdown(ctx context.Context, breakdown *pb.ContextUsageBreakdown, modelID *int64, modelName string, stage, source string) *pb.ContextUsageInfo {
	if s.usageBuilder == nil || breakdown == nil {
		return nil
	}
	// Resolve model config from llmConfigSvc when available.
	resolvedModelID := int64(0)
	var cwTokens, maxOutTokens int32
	if modelID != nil {
		resolvedModelID = *modelID
	}
	if s.llmConfigSvc != nil {
		var rc *LlmRuntimeModelConfig
		var err error
		if modelID != nil {
			rc, err = s.llmConfigSvc.GetModelRuntimeConfig(ctx, *modelID)
		} else {
			rc, err = s.llmConfigSvc.GetDefaultModelRuntimeConfig(ctx)
		}
		if err == nil && rc != nil {
			resolvedModelID = rc.ModelID
			cwTokens = rc.ContextWindowTokens
			maxOutTokens = rc.MaxOutputTokens
			modelName = rc.ModelName
		}
	}

	input := UsageBuildInput{
		ModelID:              resolvedModelID,
		ModelName:            modelName,
		ContextWindowTokens:  cwTokens,
		MaxOutputTokens:      maxOutTokens,
		SystemPromptTokens:   breakdown.GetSystemPromptTokens(),
		RecentMessageTokens:  breakdown.GetRecentMessageTokens(),
		SummaryTokens:        breakdown.GetSummaryTokens(),
		MemoryTokens:         breakdown.GetMemoryTokens(),
		CurrentMessageTokens: breakdown.GetCurrentMessageTokens(),
		SkillTokens:          breakdown.GetSkillTokens(),
		ToolResultTokens:     breakdown.GetToolResultTokens(),
		Stage:                stage,
		Source:               source,
	}

	return s.usageBuilder.Build(input)
}

// buildProviderFinalContextUsage converts the estimated final snapshot to a
// provider-backed snapshot when provider token usage is available.
//
// Rules:
//   - If usage is nil or PromptTokens <= 0, returns the estimator snapshot with
//     source=estimator and estimated=true unchanged.
//   - If provider usage exists, copies model metadata, breakdown, and session
//     accumulated estimate from the estimated snapshot, sets actual fields
//     from provider usage, and marks source=provider, estimated=false.
//     Remaining/ratio continue to describe the session accumulated estimate,
//     not the provider's per-request prompt token count.
func (s *AIService) buildProviderFinalContextUsage(ctx context.Context, estimated *pb.ContextUsageInfo, usage *schema.TokenUsage) *pb.ContextUsageInfo {
	if estimated == nil {
		return nil
	}
	if usage == nil || usage.PromptTokens <= 0 {
		estimated.Source = "estimator"
		estimated.Estimated = true
		estimated.Stage = "final"
		return estimated
	}

	info := &pb.ContextUsageInfo{
		ModelId:                estimated.ModelId,
		ModelName:              estimated.ModelName,
		ContextWindowTokens:    estimated.ContextWindowTokens,
		MaxOutputTokens:        estimated.MaxOutputTokens,
		PromptTokensEstimated:  estimated.PromptTokensEstimated,
		PromptTokensActual:     int32(usage.PromptTokens),
		CompletionTokensActual: int32(usage.CompletionTokens),
		TotalTokensActual:      int32(usage.TotalTokens),
		Breakdown:              estimated.Breakdown,
		Source:                 "provider",
		Estimated:              false,
		Stage:                  "final",
	}

	if estimated.ContextWindowTokens > 0 {
		info.RemainingTokensEstimated = estimated.RemainingTokensEstimated
		info.UsageRatio = estimated.UsageRatio
	} else {
		info.RemainingTokensEstimated = 0
		info.UsageRatio = 0
	}

	return info
}

// joinMessagesForEstimate concatenates recent message content for token estimation.
func joinMessagesForEstimate(messages []model.AIChatHistory) string {
	var b strings.Builder
	for _, m := range messages {
		b.WriteString(m.Content)
		b.WriteString("\n")
	}
	return b.String()
}

// joinMemoryForEstimate concatenates memory content for token estimation.
func joinMemoryForEstimate(memories []model.AIMemory) string {
	var b strings.Builder
	for _, m := range memories {
		b.WriteString(m.Content)
		b.WriteString("\n")
	}
	return b.String()
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

func (s *AIService) resolveRuntimeAIClient(ctx context.Context, requestedModelID int64, runtimeCfg *agentRuntimeConfig) (*runtimeAIClient, error) {
	result := &runtimeAIClient{
		client:        s.ai,
		modelName:     s.defaultRuntimeModelName(),
		auditProvider: "dashscope",
	}
	var temperatureOverride *float64
	if runtimeCfg != nil {
		temperatureOverride = runtimeCfg.TemperatureOverride
	}

	if requestedModelID > 0 {
		if s.llmConfigSvc == nil {
			return nil, fmt.Errorf("runtime model config service is unavailable")
		}
		modelRuntimeCfg, err := s.llmConfigSvc.GetModelRuntimeConfig(ctx, requestedModelID)
		if err != nil {
			return nil, err
		}
		cm, err := ai.NewChatModelWithParams(
			ctx,
			modelRuntimeCfg.ProviderType,
			modelRuntimeCfg.APIKey,
			modelRuntimeCfg.ModelName,
			modelRuntimeCfg.BaseURL,
			modelRuntimeTimeout(modelRuntimeCfg.Timeout, s.ai.Timeout()),
			modelRuntimeCfg.Params.WithTemperatureOverride(temperatureOverride),
		)
		if err != nil {
			return nil, fmt.Errorf("create chat model %d: %w", requestedModelID, err)
		}
		modelID := requestedModelID
		result.client = s.ai.CloneWithRuntimeConfig(modelRuntimeCfg.ModelName, cm, modelRuntimeCfg.Timeout, modelRuntimeCfg.Concurrency)
		result.modelID = &modelID
		result.modelName = modelRuntimeCfg.ModelName
		result.auditProvider = auditProviderName(modelRuntimeCfg.ProviderName, modelRuntimeCfg.ProviderType)
		logger.L().Info("runtime model selected", zap.Int64("model_id", requestedModelID), zap.String("model", modelRuntimeCfg.ModelName))
		return result, nil
	}

	if temperatureOverride == nil {
		if s.llmConfigSvc != nil {
			if modelID, providerType, providerName, _, modelName, _, err := s.llmConfigSvc.GetDefaultModelDetails(ctx); err == nil && modelName == result.modelName {
				result.modelID = &modelID
				result.auditProvider = auditProviderName(providerName, providerType)
			}
		}
		return result, nil
	}
	if s.llmConfigSvc == nil {
		return nil, fmt.Errorf("runtime model config service is unavailable for temperature_override")
	}
	modelRuntimeCfg, err := s.llmConfigSvc.GetDefaultModelRuntimeConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("default model details are unavailable for temperature_override: %w", err)
	}
	cm, err := ai.NewChatModelWithParams(
		ctx,
		modelRuntimeCfg.ProviderType,
		modelRuntimeCfg.APIKey,
		modelRuntimeCfg.ModelName,
		modelRuntimeCfg.BaseURL,
		modelRuntimeTimeout(modelRuntimeCfg.Timeout, s.ai.Timeout()),
		modelRuntimeCfg.Params.WithTemperatureOverride(temperatureOverride),
	)
	if err != nil {
		return nil, fmt.Errorf("create default chat model for temperature_override: %w", err)
	}
	result.client = s.ai.CloneWithRuntimeConfig(modelRuntimeCfg.ModelName, cm, modelRuntimeCfg.Timeout, modelRuntimeCfg.Concurrency)
	result.modelID = &modelRuntimeCfg.ModelID
	result.modelName = modelRuntimeCfg.ModelName
	result.auditProvider = auditProviderName(modelRuntimeCfg.ProviderName, modelRuntimeCfg.ProviderType)
	return result, nil
}

func modelRuntimeTimeout(modelTimeout, fallback time.Duration) time.Duration {
	if modelTimeout > 0 {
		return modelTimeout
	}
	return fallback
}

func auditProviderName(providerName, providerType string) string {
	if strings.TrimSpace(providerName) != "" {
		return strings.TrimSpace(providerName)
	}
	if strings.TrimSpace(providerType) != "" {
		return strings.TrimSpace(providerType)
	}
	return "unknown"
}

func (s *AIService) defaultRuntimeModelName() string {
	if s != nil && s.ai != nil {
		return s.ai.ModelName()
	}
	return "unknown"
}

func requestedModelIDPtr(modelID int64) *int64 {
	if modelID <= 0 {
		return nil
	}
	return &modelID
}

// agentRuntimeConfig holds resolved runtime configuration for an agent.
type agentRuntimeConfig struct {
	HasConfig           bool // true when an agent config record exists in DB
	AgentID             int64
	AgentName           string
	SystemPrompt        string
	ToolNames           []string
	MCPCapabilityKeys   map[string]bool
	SkillCapabilityKeys map[string]bool
	MaxIterations       int
	TemperatureOverride *float64
}

func (s *AIService) getAgentRuntimeConfig(ctx context.Context, agentType string, selectedSkillKeySets ...[]string) *agentRuntimeConfig {
	cfg := &agentRuntimeConfig{
		MaxIterations: 0, // 0 means use ADK default
	}

	// 1. Read agent config from DB
	if s.agentConfigRepo != nil {
		agentCfg, err := s.agentConfigRepo.GetByAgentType(ctx, agentType)
		if err == nil && agentCfg != nil && agentCfg.IsEnabled == 1 {
			cfg.HasConfig = true
			cfg.AgentID = agentCfg.ID
			cfg.AgentName = agentCfg.Name
			if agentCfg.MaxIterations > 0 {
				cfg.MaxIterations = int(agentCfg.MaxIterations)
			}
			if agentCfg.TemperatureOverride != nil {
				cfg.TemperatureOverride = agentCfg.TemperatureOverride
			}

			// 2. Read unified capability bindings. Legacy tool bindings are
			// exposed as builtin capabilities by the repository fallback.
			bindings, err := s.agentConfigRepo.ListCapabilityBindings(ctx, agentCfg.ID)
			if err == nil {
				for _, b := range bindings {
					if b.IsEnabled != 1 {
						continue
					}
					switch b.CapabilitySource {
					case "builtin":
						cfg.ToolNames = append(cfg.ToolNames, b.CapabilityKey)
					case "mcp":
						if cfg.MCPCapabilityKeys == nil {
							cfg.MCPCapabilityKeys = map[string]bool{}
						}
						cfg.MCPCapabilityKeys[b.CapabilityKey] = true
					case "skill":
						if cfg.SkillCapabilityKeys == nil {
							cfg.SkillCapabilityKeys = map[string]bool{}
						}
						cfg.SkillCapabilityKeys[b.CapabilityKey] = true
					}
				}
			}

			// 3. Read bound prompt template
			if agentCfg.PromptTemplateID != nil && *agentCfg.PromptTemplateID > 0 && s.promptRepo != nil {
				tmpl, err := s.promptRepo.GetByID(ctx, *agentCfg.PromptTemplateID)
				if err == nil && tmpl != nil {
					cfg.SystemPrompt = tmpl.Content
				}
			}
		}
	}

	for _, keys := range selectedSkillKeySets {
		for _, key := range keys {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			if cfg.SkillCapabilityKeys == nil {
				cfg.SkillCapabilityKeys = map[string]bool{}
			}
			cfg.SkillCapabilityKeys[key] = true
		}
	}

	return cfg
}

// GetToolTraces returns tool call traces for a given session, desensitized.
// Only the HR user who owns the session can query its traces.
func (s *AIService) GetToolTraces(ctx context.Context, req *pb.GetToolTracesRequest) (*pb.GetToolTracesResponse, error) {
	if req.SessionId <= 0 {
		return &pb.GetToolTracesResponse{Code: errs.ErrBadRequest, Msg: "session_id 不能为空"}, nil
	}

	// Verify the session belongs to this HR user.
	session, err := s.chats.GetSessionOwned(ctx, req.HrId, req.SessionId)
	if err != nil {
		logger.L().Error("get session owned failed", zap.Int64("hr_id", req.HrId), zap.Int64("session_id", req.SessionId), zap.Error(err))
		return nil, err
	}
	if session == nil {
		return &pb.GetToolTracesResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限访问"}, nil
	}

	// Query tool traces filtered by hr_id and session_id.
	traces, err := s.toolTraces.ListBySession(ctx, req.HrId, req.SessionId, 500)
	if err != nil {
		logger.L().Error("list tool traces failed", zap.Int64("hr_id", req.HrId), zap.Int64("session_id", req.SessionId), zap.Error(err))
		return nil, err
	}

	items := make([]*pb.ToolTraceItem, 0, len(traces))
	for _, t := range traces {
		items = append(items, &pb.ToolTraceItem{
			Id:            int64(t.ID),
			SessionId:     int64(t.SessionID),
			ToolName:      t.ToolName,
			ArgsJson:      desensitizeArgsJSON(t.ArgumentsJSON),
			ResultContent: desensitizeResultContent(t.ResultJSON),
			DurationMs:    t.DurationMs,
			ErrorMsg:      t.ErrorMessage,
			CreatedAt:     t.CreatedAt.Format(time.RFC3339),
		})
	}

	return &pb.GetToolTracesResponse{Code: errs.OK, Msg: "success", List: items}, nil
}

func (s *AIService) GetAgentRuns(ctx context.Context, req *pb.GetAgentRunsRequest) (*pb.GetAgentRunsResponse, error) {
	if req.SessionId <= 0 {
		return &pb.GetAgentRunsResponse{Code: errs.ErrBadRequest, Msg: "session_id 不能为空"}, nil
	}
	if s.agentRuns == nil {
		return &pb.GetAgentRunsResponse{Code: errs.OK, Msg: "success", List: []*pb.AgentRunItem{}}, nil
	}
	session, err := s.chats.GetSessionOwned(ctx, req.HrId, req.SessionId)
	if err != nil {
		logger.L().Error("get session owned failed", zap.Int64("hr_id", req.HrId), zap.Int64("session_id", req.SessionId), zap.Error(err))
		return nil, err
	}
	if session == nil {
		return &pb.GetAgentRunsResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限访问"}, nil
	}
	runs, err := s.agentRuns.ListRunsBySession(ctx, req.HrId, req.SessionId, 100)
	if err != nil {
		logger.L().Error("list agent runs failed", zap.Int64("hr_id", req.HrId), zap.Int64("session_id", req.SessionId), zap.Error(err))
		return nil, err
	}
	runIDs := make([]uint64, 0, len(runs))
	for _, run := range runs {
		runIDs = append(runIDs, run.ID)
	}
	steps, err := s.agentRuns.ListStepsByRunIDs(ctx, runIDs)
	if err != nil {
		logger.L().Error("list agent run steps failed", zap.Int64("hr_id", req.HrId), zap.Int64("session_id", req.SessionId), zap.Error(err))
		return nil, err
	}
	stepsByRun := make(map[uint64][]model.AgentRunStep, len(runs))
	for _, step := range steps {
		stepsByRun[step.RunID] = append(stepsByRun[step.RunID], step)
	}
	items := make([]*pb.AgentRunItem, 0, len(runs))
	for _, run := range runs {
		items = append(items, toPBAgentRun(run, stepsByRun[run.ID]))
	}
	return &pb.GetAgentRunsResponse{Code: errs.OK, Msg: "success", List: items}, nil
}

// desensitizeArgsJSON masks sensitive fields (phone, ID card) in tool argument JSON.
// It uses regex-based matching to find and replace common PII patterns.
func desensitizeArgsJSON(jsonStr string) string {
	if jsonStr == "" {
		return ""
	}
	// Mask phone numbers: 1xx-xxxx-xxxx or 1xxxxxxxxx
	rePhone := regexp.MustCompile(`1[3-9]\d{1}[\s\-]?\d{4}[\s\-]?\d{4}`)
	jsonStr = rePhone.ReplaceAllString(jsonStr, "****")

	// Mask ID card numbers (18 digits, possibly with X suffix)
	reIDCard := regexp.MustCompile(`\d{6}[\s\-]?\d{8}[\s\-]?[\dXx]{4}`)
	jsonStr = reIDCard.ReplaceAllString(jsonStr, "****")

	return jsonStr
}

// desensitizeResultContent truncates and masks sensitive data in tool result content.
// For large results (e.g. resume text), it truncates to a reasonable length.
func desensitizeResultContent(content string) string {
	if content == "" {
		return ""
	}
	// First apply PII masking (phone, ID card).
	content = desensitizeArgsJSON(content)

	// Truncate very large results (e.g. full resume text) to a summary length.
	runes := []rune(content)
	if len(runes) > 2000 {
		return string(runes[:2000]) + fmt.Sprintf("... [已截断，总字符数: %d]", len(runes))
	}
	return content
}

// truncateString returns the first n runes of s.
func truncateString(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// tailString returns the last n runes of s.
func tailString(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[len(runes)-n:])
}
