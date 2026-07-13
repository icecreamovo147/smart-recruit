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

	"smart-recruit-ai-agent-service/internal/legacydomain/ai"
	"smart-recruit-ai-agent-service/internal/legacydomain/model"
	"smart-recruit-ai-agent-service/internal/legacydomain/repository"
	"smart-recruit-domain-go/mq"
	"smart-recruit-platform-go/logger"
	"smart-recruit-proto/recruitment/pb"
)

// durableRunExecuteInput is the input for a single worker execution attempt.
type durableRunExecuteInput struct {
	Run              *model.AgentRun
	Request          *pb.ChatRequest
	Session          *model.AIChatSession
	OnDelta          func(delta string) error
	OnStatus         func(eventType, eventMessage, errorType, toolName string) error
	OnSkillSelection func(selection *pb.AgentSkillSelection) error
}

// durableRunExecuteResult captures executor outcome for the durable worker.
type durableRunExecuteResult struct {
	Reply               string
	ProcessText         string
	Metadata            ai.ToolMetadata
	WaitingConfirmation bool
	ConfirmationJSON    string
	Canceled            bool
	Partial             bool
	Failed              bool
	ErrorType           string
	ErrorMessage        string
	ResultMetadataJSON  string
}

// durableRunExecutor runs one durable attempt. Tests inject fakes; production uses AI pipeline.
type durableRunExecutor func(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error)

const (
	durableCancelPollInterval = 200 * time.Millisecond
	durableDeltaFlushInterval = 120 * time.Millisecond
	durableDeltaFlushChars    = 80
	agentRunExecuteEventType  = "agent.run.execute"
	agentRunAggregateType     = "agent_run"
)

type agentRunExecutePayload struct {
	RunID uint64 `json:"run_id"`
}

// dispatchDurableAgentRun enqueues runID onto the durable outbox/RabbitMQ
// worker path. When no dispatcher is injected (unit tests / direct service
// construction), it falls back to the legacy detached goroutine.
func (s *AIService) dispatchDurableAgentRun(runID uint64) {
	if s == nil || runID == 0 {
		return
	}
	if s.agentRunOutbox != nil && s.agentRunOutbox.repo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.agentRunOutbox.WriteEvent(
			ctx,
			agentRunExecuteEventType,
			agentRunAggregateType,
			runID,
			mq.AgentRunExecuteRoutingKey,
			agentRunExecutePayload{RunID: runID},
		); err != nil {
			logger.L().Warn("durable worker: enqueue outbox event failed, falling back to goroutine",
				zap.Uint64("run_id", runID),
				zap.Error(err))
		} else {
			s.agentRunOutbox.Signal()
			return
		}
	}
	go func() {
		if err := s.executeDurableAgentRun(runID); err != nil {
			logger.L().Warn("durable worker: goroutine execution failed",
				zap.Uint64("run_id", runID),
				zap.Error(err))
		}
	}()
}

// executeDurableAgentRun owns execution lifetime independent of HTTP/SSE/subscription contexts.
func (s *AIService) executeDurableAgentRun(runID uint64) error {
	// Detached root context — never tied to request or subscription.
	rootCtx := context.Background()
	log := logger.L().With(zap.Uint64("run_id", runID))
	if !s.tryStartDurableRun(runID) {
		log.Info("durable worker: run already active in this process")
		return nil
	}
	defer s.finishDurableRun(runID)

	run, err := s.agentRuns.GetRunByID(rootCtx, runID)
	if err != nil {
		log.Warn("durable worker: load run failed", zap.Error(err))
		return fmt.Errorf("load run %d: %w", runID, err)
	}
	if run == nil {
		log.Warn("durable worker: run not found")
		return nil
	}
	if IsTerminalAgentRunStatus(run.Status) {
		return nil
	}
	// waiting_confirmation is a parked state until ConfirmAgentRun dispatches again.
	if run.Status == AgentRunStatusWaitingConfirmation {
		return nil
	}

	session, err := s.chats.GetSessionOwned(rootCtx, int64(run.HrID), int64(run.SessionID))
	if err != nil {
		log.Warn("durable worker: session missing", zap.Error(err))
		return fmt.Errorf("load session for run %d: %w", runID, err)
	}
	if session == nil {
		log.Warn("durable worker: session missing")
		return s.failDurableRun(rootCtx, run, "session_missing", "会话不存在或无权限")
	}

	plan := decodeObjectJSON(run.PlanJSON)
	payload := loadDurableRequest(plan)
	chatReq := chatRequestFromDurable(int64(run.HrID), session.ID, payload)

	// Worker-local cancelable context; cancel command cancels this, not subscribers.
	workCtx, workCancel := context.WithCancel(rootCtx)
	defer workCancel()

	stopPoll := make(chan struct{})
	var pollOnce sync.Once
	go s.pollCancelRequested(workCtx, runID, workCancel, stopPoll)
	defer pollOnce.Do(func() { close(stopPoll) })

	// Ensure status is running (from planning or after confirm).
	if run.Status == AgentRunStatusPlanning || run.Status == AgentRunStatusQueued {
		from := run.Status
		if err := s.transitionDurableRun(rootCtx, run, from, AgentRunStatusRunning, nil, nil, nil); err != nil {
			// May already be cancel_requested.
			fresh, loadErr := s.agentRuns.GetRunByID(rootCtx, runID)
			if loadErr != nil {
				return fmt.Errorf("reload run %d after transition failure: %w", runID, loadErr)
			}
			if fresh != nil && fresh.Status == AgentRunStatusCancelRequested {
				return s.finalizeCanceledDurableRun(rootCtx, fresh, "")
			}
			if fresh != nil && (IsTerminalAgentRunStatus(fresh.Status) || fresh.Status == AgentRunStatusWaitingConfirmation) {
				return nil
			}
			log.Warn("durable worker: transition to running failed", zap.Error(err))
			return fmt.Errorf("transition run %d to running: %w", runID, err)
		} else {
			run.Status = AgentRunStatusRunning
		}
	}

	var assistantBuf strings.Builder
	var processBuf strings.Builder
	if run.AssistantText != "" {
		assistantBuf.WriteString(run.AssistantText)
	}
	if run.ProcessText != "" {
		processBuf.WriteString(run.ProcessText)
	}

	deltaBatcher := newAssistantDeltaBatcher(s, runID, &assistantBuf)

	input := durableRunExecuteInput{
		Run:     run,
		Request: chatReq,
		Session: session,
		OnDelta: func(delta string) error {
			if workCtx.Err() != nil {
				return workCtx.Err()
			}
			return deltaBatcher.Append(workCtx, delta)
		},
		OnStatus: func(eventType, eventMessage, errorType, toolName string) error {
			if workCtx.Err() != nil {
				return workCtx.Err()
			}
			return s.handleDurableStatusEvent(workCtx, runID, &processBuf, eventType, eventMessage, errorType, toolName)
		},
		OnSkillSelection: func(selection *pb.AgentSkillSelection) error {
			if selection == nil {
				return nil
			}
			confJSON := agentSkillSelectionConfirmationJSON(selection)
			// Capture for worker result via shared state on run snapshot.
			_ = s.agentRuns.UpdateRunSnapshot(workCtx, runID, repository.AgentRunSnapshotPatch{
				ConfirmationRequestJSON: &confJSON,
			})
			return nil
		},
	}

	executor := s.durableRunExecutor
	if executor == nil {
		executor = s.defaultDurableRunExecutor
	}

	result, execErr := executor(workCtx, input)
	_ = deltaBatcher.Flush(rootCtx)

	// Always re-load for concurrent cancel_requested.
	fresh, loadErr := s.agentRuns.GetRunByID(rootCtx, runID)
	if loadErr != nil {
		log.Warn("durable worker: reload after execute failed", zap.Error(loadErr))
		return fmt.Errorf("reload run %d after execute: %w", runID, loadErr)
	}
	if fresh == nil {
		log.Warn("durable worker: run missing after execute")
		return nil
	}

	if fresh.Status == AgentRunStatusCancelRequested || (result != nil && result.Canceled) || (execErr != nil && isCanceledError(execErr) && fresh.Status == AgentRunStatusCancelRequested) {
		partial := ""
		if result != nil {
			partial = result.Reply
		}
		if strings.TrimSpace(partial) == "" {
			partial = strings.TrimSpace(assistantBuf.String())
		}
		return s.finalizeCanceledDurableRun(rootCtx, fresh, partial)
	}

	if result != nil && result.WaitingConfirmation {
		return s.parkWaitingConfirmation(rootCtx, fresh, result)
	}

	if execErr != nil && errors.Is(execErr, errAgentSkillSelectionRequired) {
		// Confirmation payload should already be on snapshot via OnSkillSelection.
		if result == nil {
			result = &durableRunExecuteResult{WaitingConfirmation: true}
		}
		result.WaitingConfirmation = true
		if result.ConfirmationJSON == "" {
			result.ConfirmationJSON = fresh.ConfirmationRequestJSON
		}
		return s.parkWaitingConfirmation(rootCtx, fresh, result)
	}

	if (result != nil && result.Failed) || (execErr != nil && (result == nil || !result.Partial)) {
		errType := "execution_failed"
		errMsg := ""
		if result != nil {
			if result.ErrorType != "" {
				errType = result.ErrorType
			}
			errMsg = result.ErrorMessage
		}
		if execErr != nil && errMsg == "" {
			errMsg = execErr.Error()
		}
		return s.failDurableRun(rootCtx, fresh, errType, errMsg)
	}

	// Success or partial fallback.
	reply := ""
	processText := processBuf.String()
	if result != nil {
		reply = result.Reply
		if result.ProcessText != "" {
			processText = result.ProcessText
		}
	}
	if strings.TrimSpace(reply) == "" {
		reply = strings.TrimSpace(assistantBuf.String())
	}
	status := AgentRunStatusSucceeded
	if result != nil && result.Partial {
		status = AgentRunStatusPartial
	}

	resultMeta := ""
	if result != nil {
		resultMeta = result.ResultMetadataJSON
	}
	if err := s.completeDurableRun(rootCtx, fresh, session, status, reply, processText, resultMeta, "", ""); err != nil {
		log.Warn("durable worker: complete failed", zap.Error(err))
		return fmt.Errorf("complete run %d: %w", runID, err)
	}
	return nil
}

func (s *AIService) defaultDurableRunExecutor(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error) {
	if input.Request == nil || input.Session == nil || input.Run == nil {
		return &durableRunExecuteResult{Failed: true, ErrorType: "invalid_input", ErrorMessage: "missing run inputs"}, nil
	}
	runtimeCfg := s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent", input.Request.GetSkillCapabilityKeys())
	runtimeClient, err := s.resolveRuntimeAIClient(ctx, input.Request.GetModelId(), runtimeCfg)
	if err != nil {
		return &durableRunExecuteResult{
			Failed:       true,
			ErrorType:    "model_resolution_failed",
			ErrorMessage: err.Error(),
		}, nil
	}

	recorder := &agentRunRecorder{
		repo:      s.agentRuns,
		runID:     input.Run.ID,
		session:   input.Session,
		hrID:      input.Request.HrId,
		planState: decodeObjectJSON(input.Run.PlanJSON),
	}
	if input.Run.ModelName == "" {
		input.Run.ModelName = runtimeClient.modelName
	}
	if runtimeClient.modelName != "" {
		_ = s.agentRuns.UpdateRunSnapshot(ctx, input.Run.ID, repository.AgentRunSnapshotPatch{
			ModelName: &runtimeClient.modelName,
		})
		// Keep model name on run via plan patch.
		recorder.updatePlanPatch(ctx, map[string]any{"model": runtimeClient.modelName})
	}

	var processContent strings.Builder
	reply, metadata, err := s.runToolCallingChatWithUsage(
		ctx,
		input.Request,
		input.Session,
		runtimeClient.modelID,
		runtimeClient.modelName,
		runtimeCfg,
		input.OnDelta,
		func(eventType, eventMessage, errorType, toolName string) error {
			if eventType == "process_delta" {
				processContent.WriteString(eventMessage)
			} else if eventType == "process_clear" {
				processContent.Reset()
			}
			if input.OnStatus != nil {
				return input.OnStatus(eventType, eventMessage, errorType, toolName)
			}
			return nil
		},
		nil,
		func(selection *pb.AgentSkillSelection) error {
			if input.OnSkillSelection != nil {
				_ = input.OnSkillSelection(selection)
			}
			return nil
		},
		runtimeClient.client,
		withDurableChatHooks(&durableChatHooks{
			recorder:                  recorder,
			durable:                   true,
			skipFinalAssistantPersist: true,
		}),
	)

	result := &durableRunExecuteResult{
		Reply:       reply,
		ProcessText: processContent.String(),
		Metadata:    metadata,
	}
	if metadata.Action != nil || len(metadata.CandidateOptions) > 0 {
		result.ResultMetadataJSON = buildResultMetadataJSON(metadata)
	}

	if err != nil {
		if errors.Is(err, errAgentSkillSelectionRequired) {
			result.WaitingConfirmation = true
			// Confirmation JSON written by OnSkillSelection.
			fresh, _ := s.agentRuns.GetRunByID(ctx, input.Run.ID)
			if fresh != nil {
				result.ConfirmationJSON = fresh.ConfirmationRequestJSON
			}
			return result, err
		}
		if isCanceledError(err) {
			result.Canceled = true
			return result, err
		}
		// Fallback path may return nil err with partial content from pipeline;
		// when err is non-nil and no tool fallback already applied, mark failed.
		aiErr := ai.ClassifyAIError(err)
		result.Failed = true
		result.ErrorType = string(aiErr.Type)
		result.ErrorMessage = err.Error()
		return result, err
	}
	return result, nil
}

func (s *AIService) pollCancelRequested(ctx context.Context, runID uint64, cancel context.CancelFunc, stop <-chan struct{}) {
	ticker := time.NewTicker(durableCancelPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			run, err := s.agentRuns.GetRunByID(context.Background(), runID)
			if err != nil || run == nil {
				continue
			}
			if run.Status == AgentRunStatusCancelRequested || IsTerminalAgentRunStatus(run.Status) {
				cancel()
				return
			}
		}
	}
}

func (s *AIService) handleDurableStatusEvent(ctx context.Context, runID uint64, processBuf *strings.Builder, eventType, eventMessage, errorType, toolName string) error {
	switch eventType {
	case "process_delta":
		if processBuf != nil {
			processBuf.WriteString(eventMessage)
			text := processBuf.String()
			_ = s.agentRuns.UpdateRunSnapshot(ctx, runID, repository.AgentRunSnapshotPatch{ProcessText: &text})
		}
		_, err := s.appendRunEvent(ctx, runID, AgentRunEventProcessDelta, map[string]any{
			"delta": eventMessage,
		})
		return err
	case "process_clear":
		if processBuf != nil {
			processBuf.Reset()
			empty := ""
			_ = s.agentRuns.UpdateRunSnapshot(ctx, runID, repository.AgentRunSnapshotPatch{ProcessText: &empty})
		}
		_, err := s.appendRunEvent(ctx, runID, AgentRunEventProcessSnapshot, map[string]any{
			"snapshot_text": "",
		})
		return err
	case "tool_start", "tool_call_start":
		_, err := s.appendRunEvent(ctx, runID, AgentRunEventToolStarted, map[string]any{
			"tool_name": toolName,
			"message":   eventMessage,
		})
		return err
	case "tool_end", "tool_call_end", "tool_result":
		_, err := s.appendRunEvent(ctx, runID, AgentRunEventToolFinished, map[string]any{
			"tool_name":  toolName,
			"message":    eventMessage,
			"error_type": errorType,
		})
		return err
	default:
		// Other process/status events: store as process.delta when message present.
		if strings.TrimSpace(eventMessage) != "" {
			if processBuf != nil && isVisibleDurableProcessEvent(eventType) {
				line := eventMessage + "\n"
				processBuf.WriteString(line)
				text := processBuf.String()
				_ = s.agentRuns.UpdateRunSnapshot(ctx, runID, repository.AgentRunSnapshotPatch{ProcessText: &text})
				_, err := s.appendRunEvent(ctx, runID, AgentRunEventProcessDelta, map[string]any{
					"delta":      line,
					"event_type": eventType,
				})
				return err
			}
		}
		return nil
	}
}

func isVisibleDurableProcessEvent(eventType string) bool {
	switch eventType {
	case "planning", "status":
		return true
	default:
		return false
	}
}

func (s *AIService) parkWaitingConfirmation(ctx context.Context, run *model.AgentRun, result *durableRunExecuteResult) error {
	confJSON := ""
	if result != nil {
		confJSON = result.ConfirmationJSON
	}
	if confJSON == "" {
		confJSON = run.ConfirmationRequestJSON
	}
	if confJSON == "" {
		confJSON = `{"required":true}`
	}
	if err := s.agentRuns.UpdateRunSnapshot(ctx, run.ID, repository.AgentRunSnapshotPatch{
		ConfirmationRequestJSON: &confJSON,
	}); err != nil {
		return err
	}
	from := run.Status
	if from == AgentRunStatusCancelRequested {
		return s.finalizeCanceledDurableRun(ctx, run, "")
	}
	if from != AgentRunStatusWaitingConfirmation {
		if err := s.transitionDurableRun(ctx, run, from, AgentRunStatusWaitingConfirmation, nil, nil, nil); err != nil {
			logger.L().Warn("park waiting_confirmation failed", zap.Error(err), zap.Uint64("run_id", run.ID))
			return err
		}
	}
	if _, err := s.appendRunEvent(ctx, run.ID, AgentRunEventConfirmationRequired, confJSON); err != nil {
		logger.L().Warn("append confirmation.required failed", zap.Error(err), zap.Uint64("run_id", run.ID))
		return err
	}
	// Worker stops; ConfirmAgentRun will redispatch same run_id.
	return nil
}

func (s *AIService) finalizeCanceledDurableRun(ctx context.Context, run *model.AgentRun, partial string) error {
	now := time.Now()
	from := run.Status
	if from != AgentRunStatusCancelRequested && !IsTerminalAgentRunStatus(from) {
		if err := s.transitionDurableRun(ctx, run, from, AgentRunStatusCancelRequested, nil, &now, nil); err != nil {
			return err
		}
		from = AgentRunStatusCancelRequested
	}
	if IsTerminalAgentRunStatus(from) {
		return nil
	}
	if err := s.transitionDurableRun(ctx, run, from, AgentRunStatusCanceled, &now, nil, &now); err != nil {
		logger.L().Warn("finalize canceled failed", zap.Error(err), zap.Uint64("run_id", run.ID))
		return err
	}
	patch := repository.AgentRunSnapshotPatch{}
	if partial != "" {
		patch.AssistantText = &partial
		patch.FinalAnswer = &partial
	}
	errType := "canceled"
	errMsg := "user canceled"
	patch.ErrorType = &errType
	patch.ErrorMessage = &errMsg
	if err := s.agentRuns.UpdateRunSnapshot(ctx, run.ID, patch); err != nil {
		return err
	}
	if _, err := s.appendRunEvent(ctx, run.ID, AgentRunEventCanceled, map[string]any{
		"status":         AgentRunStatusCanceled,
		"assistant_text": partial,
	}); err != nil {
		logger.L().Warn("append run.canceled failed", zap.Error(err), zap.Uint64("run_id", run.ID))
		return err
	}
	if err := s.agentRuns.SetSessionActiveRun(ctx, int64(run.SessionID), nil); err != nil {
		return err
	}
	return nil
}

func (s *AIService) failDurableRun(ctx context.Context, run *model.AgentRun, errorType, errorMessage string) error {
	now := time.Now()
	from := run.Status
	if IsTerminalAgentRunStatus(from) {
		return nil
	}
	// cancel_requested can fail into failed per state machine.
	if from != AgentRunStatusFailed {
		if err := s.transitionDurableRun(ctx, run, from, AgentRunStatusFailed, &now, nil, nil); err != nil {
			logger.L().Warn("fail durable run transition failed", zap.Error(err), zap.Uint64("run_id", run.ID))
			return err
		}
	}
	patch := repository.AgentRunSnapshotPatch{
		ErrorType:    &errorType,
		ErrorMessage: &errorMessage,
	}
	if err := s.agentRuns.UpdateRunSnapshot(ctx, run.ID, patch); err != nil {
		return err
	}
	if _, err := s.appendRunEvent(ctx, run.ID, AgentRunEventError, map[string]any{
		"status":        AgentRunStatusFailed,
		"error_type":    errorType,
		"error_message": errorMessage,
	}); err != nil {
		logger.L().Warn("append run.error failed", zap.Error(err), zap.Uint64("run_id", run.ID))
		return err
	}
	if err := s.agentRuns.SetSessionActiveRun(ctx, int64(run.SessionID), nil); err != nil {
		return err
	}
	return nil
}

func (s *AIService) completeDurableRun(ctx context.Context, run *model.AgentRun, session *model.AIChatSession, status, reply, processText, resultMeta, errorType, errorMessage string) error {
	writeCtx, cancel := agentRunFinalWriteContext()
	defer cancel()

	// Idempotent assistant history write.
	if err := s.persistFinalAssistantOnce(writeCtx, run, session, reply, processText); err != nil {
		logger.L().Warn("persist final assistant failed", zap.Error(err), zap.Uint64("run_id", run.ID))
		return err
	}

	now := time.Now()
	from := run.Status
	if !IsTerminalAgentRunStatus(from) {
		if err := s.transitionDurableRun(writeCtx, run, from, status, &now, nil, nil); err != nil {
			// Reload — may already be terminal from concurrent path.
			fresh, _ := s.agentRuns.GetRunByID(writeCtx, run.ID)
			if fresh == nil || !IsTerminalAgentRunStatus(fresh.Status) {
				return err
			}
		}
	}

	patch := repository.AgentRunSnapshotPatch{
		AssistantText: &reply,
		FinalAnswer:   &reply,
		ProcessText:   &processText,
	}
	if resultMeta != "" {
		patch.ResultMetadataJSON = &resultMeta
		// Keep option/action context available for refresh restore (FR-006).
		patch.OptionContextJSON = &resultMeta
	}
	if errorType != "" {
		patch.ErrorType = &errorType
		patch.ErrorMessage = &errorMessage
	}
	_ = s.agentRuns.UpdateRunSnapshot(writeCtx, run.ID, patch)

	if resultMeta != "" {
		_, _ = s.appendRunEvent(writeCtx, run.ID, AgentRunEventResult, resultMeta)
	}
	if _, err := s.appendRunEvent(writeCtx, run.ID, AgentRunEventCompleted, map[string]any{
		"status":         status,
		"assistant_text": reply,
	}); err != nil {
		logger.L().Warn("append run.completed failed", zap.Error(err), zap.Uint64("run_id", run.ID))
	}
	_ = s.agentRuns.SetSessionActiveRun(writeCtx, int64(run.SessionID), nil)
	return nil
}

// persistFinalAssistantOnce writes at most one assistant history row for the run.
// message_id is the user message; history_id is the assistant message. Do not treat
// a user message id stored in history_id as evidence that the assistant already exists.
func (s *AIService) persistFinalAssistantOnce(ctx context.Context, run *model.AgentRun, session *model.AIChatSession, content, processText string) error {
	if s == nil || s.agentRuns == nil || s.chats == nil || run == nil || session == nil {
		return nil
	}
	count, err := s.agentRuns.CountAssistantHistoryByAgentRunID(ctx, run.ID)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	// history_id already points at an assistant association distinct from the user message.
	if run.HistoryID != nil && *run.HistoryID > 0 {
		if run.MessageID == nil || *run.HistoryID != *run.MessageID {
			return nil
		}
		// Legacy pollution: history_id == message_id (user). Clear-and-rewrite path continues.
	}
	if strings.TrimSpace(content) == "" {
		return nil
	}
	runID := int64(run.ID)
	var modelID *int64
	if run.ModelID != nil {
		v := int64(*run.ModelID)
		modelID = &v
	}
	history := &model.AIChatHistory{
		SessionID:      session.ID,
		HrID:           int64(run.HrID),
		Role:           "assistant",
		Content:        content,
		ProcessContent: processText,
		ModelID:        modelID,
		ModelName:      run.ModelName,
		AgentRunID:     &runID,
	}
	if err := s.chats.Add(ctx, history); err != nil {
		return err
	}
	if history.ID > 0 {
		_ = s.agentRuns.UpdateRunHistoryID(ctx, run.ID, uint64(history.ID))
		run.HistoryID = uint64Ptr(uint64(history.ID))
	}
	return nil
}

func uint64Ptr(v uint64) *uint64 { return &v }

func buildResultMetadataJSON(metadata ai.ToolMetadata) string {
	payload := map[string]any{}
	if metadata.Action != nil {
		payload["action"] = metadata.Action.Action
		payload["application_id"] = metadata.Action.ApplicationID
		payload["action_status"] = metadata.Action.ActionStatus
		payload["candidate_name"] = metadata.Action.CandidateName
		payload["job_title"] = metadata.Action.JobTitle
		payload["status"] = metadata.Action.Status
	}
	if len(metadata.CandidateOptions) > 0 {
		if data, err := json.Marshal(metadata.CandidateOptions); err == nil {
			payload["candidate_options"] = string(data)
		}
	}
	return safeJSON(payload)
}

func (s *AIService) tryStartDurableRun(runID uint64) bool {
	if s == nil || runID == 0 {
		return false
	}
	s.durableRunGuardMu.Lock()
	defer s.durableRunGuardMu.Unlock()
	if s.durableRunGuard == nil {
		s.durableRunGuard = make(map[uint64]struct{})
	}
	if _, exists := s.durableRunGuard[runID]; exists {
		return false
	}
	s.durableRunGuard[runID] = struct{}{}
	return true
}

func (s *AIService) finishDurableRun(runID uint64) {
	if s == nil || runID == 0 {
		return
	}
	s.durableRunGuardMu.Lock()
	defer s.durableRunGuardMu.Unlock()
	delete(s.durableRunGuard, runID)
}

// assistantDeltaBatcher batches token deltas to avoid one DB row per token (NFR-003).
type assistantDeltaBatcher struct {
	svc       *AIService
	runID     uint64
	assistant *strings.Builder
	buf       strings.Builder
	mu        sync.Mutex
	lastFlush time.Time
}

func newAssistantDeltaBatcher(svc *AIService, runID uint64, assistant *strings.Builder) *assistantDeltaBatcher {
	return &assistantDeltaBatcher{
		svc:       svc,
		runID:     runID,
		assistant: assistant,
		lastFlush: time.Now(),
	}
}

func (b *assistantDeltaBatcher) Append(ctx context.Context, delta string) error {
	if b == nil || delta == "" {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.WriteString(delta)
	if b.assistant != nil {
		b.assistant.WriteString(delta)
	}
	if b.buf.Len() >= durableDeltaFlushChars || time.Since(b.lastFlush) >= durableDeltaFlushInterval {
		return b.flushLocked(ctx)
	}
	return nil
}

func (b *assistantDeltaBatcher) Flush(ctx context.Context) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushLocked(ctx)
}

func (b *assistantDeltaBatcher) flushLocked(ctx context.Context) error {
	if b.buf.Len() == 0 {
		return nil
	}
	delta := b.buf.String()
	b.buf.Reset()
	b.lastFlush = time.Now()
	if b.assistant != nil {
		text := b.assistant.String()
		_ = b.svc.agentRuns.UpdateRunSnapshot(ctx, b.runID, repository.AgentRunSnapshotPatch{
			AssistantText: &text,
		})
	}
	_, err := b.svc.appendRunEvent(ctx, b.runID, AgentRunEventAssistantDelta, map[string]any{
		"delta": delta,
	})
	return err
}
