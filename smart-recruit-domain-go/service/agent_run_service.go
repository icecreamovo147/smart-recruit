package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"go.uber.org/zap"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/repository"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/logger"
	"smart-recruit-proto/recruitment/pb"
)

// durableRequestPayload is stored under plan_json so workers can rebuild ChatRequest.
type durableRequestPayload struct {
	Message                      string   `json:"message"`
	ActionType                   string   `json:"action_type,omitempty"`
	ActionPayloadJSON            string   `json:"action_payload_json,omitempty"`
	ApplicationID                int64    `json:"application_id,omitempty"`
	ModelID                      int64    `json:"model_id,omitempty"`
	SkillCapabilityKeys          []string `json:"skill_capability_keys,omitempty"`
	AgentSkillIDs                []int64  `json:"agent_skill_ids,omitempty"`
	AgentSkillSelectionConfirmed bool     `json:"agent_skill_selection_confirmed,omitempty"`
	AgentSkillSelectionMessageID int64    `json:"agent_skill_selection_message_id,omitempty"`
	ConfirmationPayloadJSON      string   `json:"confirmation_payload_json,omitempty"`
	ConfirmClientRequestID       string   `json:"confirm_client_request_id,omitempty"`
	CancelClientRequestID        string   `json:"cancel_client_request_id,omitempty"`
}

// CreateAgentRun creates or returns an idempotent durable HR Agent run and dispatches work.
func (s *AIService) CreateAgentRun(ctx context.Context, req *pb.CreateAgentRunRequest) (*pb.CreateAgentRunResponse, error) {
	if req == nil {
		return &pb.CreateAgentRunResponse{Code: errs.ErrBadRequest, Msg: "请求不能为空"}, nil
	}
	if req.GetHrId() <= 0 {
		return &pb.CreateAgentRunResponse{Code: errs.ErrBadRequest, Msg: "hr_id 不能为空"}, nil
	}
	if req.GetSessionId() <= 0 {
		return &pb.CreateAgentRunResponse{Code: errs.ErrBadRequest, Msg: "session_id 不能为空"}, nil
	}
	clientRequestID := strings.TrimSpace(req.GetClientRequestId())
	if clientRequestID == "" {
		return &pb.CreateAgentRunResponse{Code: errs.ErrBadRequest, Msg: "client_request_id 不能为空"}, nil
	}
	message := strings.TrimSpace(req.GetMessage())
	actionType := strings.TrimSpace(req.GetActionType())
	if message == "" && actionType == "" {
		return &pb.CreateAgentRunResponse{Code: errs.ErrBadRequest, Msg: "message 或 action_type 不能同时为空"}, nil
	}
	if s.agentRuns == nil || s.chats == nil {
		return &pb.CreateAgentRunResponse{Code: errs.ErrInternal, Msg: "agent run 服务未就绪"}, nil
	}

	session, err := s.chats.GetSessionOwned(ctx, req.GetHrId(), req.GetSessionId())
	if err != nil {
		logger.L().Error("create agent run: get session failed", zap.Error(err), zap.Int64("hr_id", req.GetHrId()), zap.Int64("session_id", req.GetSessionId()))
		return nil, err
	}
	if session == nil {
		return &pb.CreateAgentRunResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限访问"}, nil
	}

	existing, err := s.agentRuns.GetRunByClientRequestID(ctx, uint64(req.GetHrId()), uint64(req.GetSessionId()), clientRequestID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &pb.CreateAgentRunResponse{
			Code:             errs.OK,
			Msg:              "success",
			Run:              toPBAgentRunSnapshot(existing),
			IdempotentReplay: true,
		}, nil
	}

	applicationID := req.GetApplicationId()
	if applicationID <= 0 && session.ApplicationID > 0 {
		applicationID = session.ApplicationID
	}
	payload := durableRequestPayload{
		Message:                      message,
		ActionType:                   actionType,
		ActionPayloadJSON:            req.GetActionPayloadJson(),
		ApplicationID:                applicationID,
		ModelID:                      req.GetModelId(),
		SkillCapabilityKeys:          append([]string(nil), req.GetSkillCapabilityKeys()...),
		AgentSkillIDs:                append([]int64(nil), req.GetAgentSkillIds()...),
		AgentSkillSelectionConfirmed: req.GetAgentSkillSelectionConfirmed(),
		AgentSkillSelectionMessageID: req.GetAgentSkillSelectionMessageId(),
	}
	planState := map[string]any{
		planRequestKey: payload,
		"agent":        "hr_recruiting_agent",
		"agent_type":   "hr",
		"runtime":      s.agentRuntime,
	}
	if req.GetModelId() > 0 {
		planState["model_id"] = req.GetModelId()
	}

	now := time.Now()
	clientReq := clientRequestID
	run := &model.AgentRun{
		SessionID:       uint64(session.ID),
		HrID:            uint64(req.GetHrId()),
		ClientRequestID: &clientReq,
		AgentType:       "hr",
		AgentName:       "hr_recruiting_agent",
		Status:          AgentRunStatusQueued,
		PlanJSON:        safeJSON(planState),
		StartedAt:       now,
	}
	if req.GetModelId() > 0 {
		mid := uint64(req.GetModelId())
		run.ModelID = &mid
	}
	if err := s.agentRuns.CreateRun(ctx, run); err != nil {
		// Race on unique client_request_id: return existing.
		existing, getErr := s.agentRuns.GetRunByClientRequestID(ctx, uint64(req.GetHrId()), uint64(req.GetSessionId()), clientRequestID)
		if getErr == nil && existing != nil {
			return &pb.CreateAgentRunResponse{
				Code:             errs.OK,
				Msg:              "success",
				Run:              toPBAgentRunSnapshot(existing),
				IdempotentReplay: true,
			}, nil
		}
		logger.L().Error("create agent run failed", zap.Error(err), zap.String("client_request_id", clientRequestID))
		return nil, err
	}

	activeID := int64(run.ID)
	if err := s.agentRuns.SetSessionActiveRun(ctx, session.ID, &activeID); err != nil {
		logger.L().Warn("set session active_run_id failed", zap.Error(err), zap.Uint64("run_id", run.ID))
	}

	if _, err := s.appendRunEvent(ctx, run.ID, AgentRunEventCreated, map[string]any{
		"status":            AgentRunStatusQueued,
		"client_request_id": clientRequestID,
		"session_id":        session.ID,
		"hr_id":             req.GetHrId(),
	}); err != nil {
		logger.L().Warn("append run.created failed", zap.Error(err), zap.Uint64("run_id", run.ID))
	}

	if err := s.transitionDurableRun(ctx, run, AgentRunStatusQueued, AgentRunStatusPlanning, nil, nil, nil); err != nil {
		logger.L().Warn("transition queued->planning failed", zap.Error(err), zap.Uint64("run_id", run.ID))
	} else {
		run.Status = AgentRunStatusPlanning
	}

	s.dispatchDurableAgentRun(run.ID)

	fresh, err := s.agentRuns.GetRunByID(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	if fresh == nil {
		fresh = run
	}
	return &pb.CreateAgentRunResponse{
		Code:             errs.OK,
		Msg:              "success",
		Run:              toPBAgentRunSnapshot(fresh),
		IdempotentReplay: false,
	}, nil
}

// GetAgentRun returns a durable run snapshot with ownership checks.
func (s *AIService) GetAgentRun(ctx context.Context, req *pb.GetAgentRunRequest) (*pb.GetAgentRunResponse, error) {
	run, code, msg, err := s.loadOwnedAgentRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if code != errs.OK {
		return &pb.GetAgentRunResponse{Code: code, Msg: msg}, nil
	}
	return &pb.GetAgentRunResponse{Code: errs.OK, Msg: "success", Run: toPBAgentRunSnapshot(run)}, nil
}

// GetActiveAgentRun returns the session active run when present.
func (s *AIService) GetActiveAgentRun(ctx context.Context, req *pb.GetActiveAgentRunRequest) (*pb.GetActiveAgentRunResponse, error) {
	if req.GetHrId() <= 0 || req.GetSessionId() <= 0 {
		return &pb.GetActiveAgentRunResponse{Code: errs.ErrBadRequest, Msg: "hr_id 与 session_id 不能为空"}, nil
	}
	if s.agentRuns == nil || s.chats == nil {
		return &pb.GetActiveAgentRunResponse{Code: errs.OK, Msg: "success", HasActiveRun: false}, nil
	}
	session, err := s.chats.GetSessionOwned(ctx, req.GetHrId(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	if session == nil {
		return &pb.GetActiveAgentRunResponse{Code: errs.ErrForbidden, Msg: "会话不存在或无权限访问"}, nil
	}
	run, err := s.agentRuns.GetActiveRunBySession(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	if run == nil || run.HrID != uint64(req.GetHrId()) {
		return &pb.GetActiveAgentRunResponse{Code: errs.OK, Msg: "success", HasActiveRun: false}, nil
	}
	// If pointer is stale (terminal), clear and report none.
	if IsTerminalAgentRunStatus(run.Status) {
		_ = s.agentRuns.SetSessionActiveRun(ctx, session.ID, nil)
		return &pb.GetActiveAgentRunResponse{Code: errs.OK, Msg: "success", HasActiveRun: false}, nil
	}
	return &pb.GetActiveAgentRunResponse{
		Code:         errs.OK,
		Msg:          "success",
		Run:          toPBAgentRunSnapshot(run),
		HasActiveRun: true,
	}, nil
}

// CancelAgentRun marks cancel_requested; worker observes and reaches canceled.
func (s *AIService) CancelAgentRun(ctx context.Context, req *pb.CancelAgentRunRequest) (*pb.CancelAgentRunResponse, error) {
	run, code, msg, err := s.loadOwnedAgentRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if code != errs.OK {
		return &pb.CancelAgentRunResponse{Code: code, Msg: msg}, nil
	}
	if IsTerminalAgentRunStatus(run.Status) {
		return &pb.CancelAgentRunResponse{Code: errs.OK, Msg: "success", Run: toPBAgentRunSnapshot(run)}, nil
	}
	if run.Status == AgentRunStatusCancelRequested {
		return &pb.CancelAgentRunResponse{Code: errs.OK, Msg: "success", Run: toPBAgentRunSnapshot(run)}, nil
	}

	now := time.Now()
	from := run.Status
	if err := s.transitionDurableRun(ctx, run, from, AgentRunStatusCancelRequested, nil, &now, nil); err != nil {
		// Reload in case concurrent terminal transition.
		fresh, getErr := s.agentRuns.GetRunByID(ctx, run.ID)
		if getErr != nil {
			return nil, getErr
		}
		if fresh != nil && (IsTerminalAgentRunStatus(fresh.Status) || fresh.Status == AgentRunStatusCancelRequested) {
			return &pb.CancelAgentRunResponse{Code: errs.OK, Msg: "success", Run: toPBAgentRunSnapshot(fresh)}, nil
		}
		logger.L().Warn("cancel agent run transition failed", zap.Error(err), zap.Uint64("run_id", run.ID), zap.String("from", from))
		return &pb.CancelAgentRunResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	if cancelID := strings.TrimSpace(req.GetClientRequestId()); cancelID != "" {
		_ = s.patchDurableRequest(ctx, run.ID, func(p *durableRequestPayload) {
			p.CancelClientRequestID = cancelID
		})
	}
	fresh, err := s.agentRuns.GetRunByID(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	if fresh == nil {
		fresh = run
		fresh.Status = AgentRunStatusCancelRequested
		fresh.CancelRequestedAt = &now
	}
	return &pb.CancelAgentRunResponse{Code: errs.OK, Msg: "success", Run: toPBAgentRunSnapshot(fresh)}, nil
}

// ConfirmAgentRun resumes a run waiting for skill confirmation.
func (s *AIService) ConfirmAgentRun(ctx context.Context, req *pb.ConfirmAgentRunRequest) (*pb.ConfirmAgentRunResponse, error) {
	run, code, msg, err := s.loadOwnedAgentRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return nil, err
	}
	if code != errs.OK {
		return &pb.ConfirmAgentRunResponse{Code: code, Msg: msg}, nil
	}

	// Idempotent: already resumed/running/terminal after same confirm.
	if run.Status != AgentRunStatusWaitingConfirmation {
		if IsTerminalAgentRunStatus(run.Status) || IsActiveAgentRunStatus(run.Status) {
			return &pb.ConfirmAgentRunResponse{Code: errs.OK, Msg: "success", Run: toPBAgentRunSnapshot(run)}, nil
		}
		return &pb.ConfirmAgentRunResponse{Code: errs.ErrBadRequest, Msg: "当前状态不可确认"}, nil
	}

	skillIDs := append([]int64(nil), req.GetAgentSkillIds()...)
	if err := s.patchDurableRequest(ctx, run.ID, func(p *durableRequestPayload) {
		p.AgentSkillIDs = skillIDs
		p.AgentSkillSelectionConfirmed = req.GetAgentSkillSelectionConfirmed() || true
		if req.GetAgentSkillSelectionMessageId() > 0 {
			p.AgentSkillSelectionMessageID = req.GetAgentSkillSelectionMessageId()
		}
		if payload := strings.TrimSpace(req.GetConfirmationPayloadJson()); payload != "" {
			p.ConfirmationPayloadJSON = payload
		}
		if cid := strings.TrimSpace(req.GetClientRequestId()); cid != "" {
			p.ConfirmClientRequestID = cid
		}
	}); err != nil {
		logger.L().Warn("patch durable request on confirm failed", zap.Error(err), zap.Uint64("run_id", run.ID))
	}

	if _, err := s.appendRunEvent(ctx, run.ID, AgentRunEventConfirmationAccepted, map[string]any{
		"agent_skill_ids":                  skillIDs,
		"agent_skill_selection_confirmed":  true,
		"agent_skill_selection_message_id": req.GetAgentSkillSelectionMessageId(),
	}); err != nil {
		logger.L().Warn("append confirmation.accepted failed", zap.Error(err), zap.Uint64("run_id", run.ID))
	}

	// Clear confirmation request snapshot after accept.
	empty := ""
	_ = s.agentRuns.UpdateRunSnapshot(ctx, run.ID, repository.AgentRunSnapshotPatch{
		ConfirmationRequestJSON: &empty,
	})

	if err := s.transitionDurableRun(ctx, run, AgentRunStatusWaitingConfirmation, AgentRunStatusRunning, nil, nil, nil); err != nil {
		fresh, getErr := s.agentRuns.GetRunByID(ctx, run.ID)
		if getErr != nil {
			return nil, getErr
		}
		if fresh != nil && fresh.Status != AgentRunStatusWaitingConfirmation {
			return &pb.ConfirmAgentRunResponse{Code: errs.OK, Msg: "success", Run: toPBAgentRunSnapshot(fresh)}, nil
		}
		return &pb.ConfirmAgentRunResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	s.dispatchDurableAgentRun(run.ID)

	fresh, err := s.agentRuns.GetRunByID(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	if fresh == nil {
		run.Status = AgentRunStatusRunning
		fresh = run
	}
	return &pb.ConfirmAgentRunResponse{Code: errs.OK, Msg: "success", Run: toPBAgentRunSnapshot(fresh)}, nil
}

// agentRunEventStream is the minimal stream surface for SubscribeAgentRunEvents.
// Tests can implement this without full gRPC server stream plumbing.
type agentRunEventStream interface {
	Send(*pb.AgentRunEvent) error
	Context() context.Context
}

// SubscribeAgentRunEvents replays events after after_seq then live-tails until terminal or stream cancel.
// Stream context cancellation ends only the subscription — never the run.
func (s *AIService) SubscribeAgentRunEvents(req *pb.SubscribeAgentRunEventsRequest, stream pb.AIService_SubscribeAgentRunEventsServer) error {
	return s.subscribeAgentRunEvents(req, stream)
}

func (s *AIService) subscribeAgentRunEvents(req *pb.SubscribeAgentRunEventsRequest, stream agentRunEventStream) error {
	ctx := stream.Context()
	run, code, msg, err := s.loadOwnedAgentRun(ctx, req.GetHrId(), req.GetRunId())
	if err != nil {
		return err
	}
	if code != errs.OK {
		return nil
	}
	_ = msg

	afterSeq := req.GetAfterSeq()
	if s.agentRunEvents == nil {
		return nil
	}

	// Replay persisted events first.
	events, err := s.agentRunEvents.ListEventsAfter(ctx, run.ID, afterSeq)
	if err != nil {
		return err
	}
	for i := range events {
		if err := stream.Send(toPBAgentRunEvent(&events[i])); err != nil {
			// Subscriber gone — do not cancel run.
			return nil
		}
		if events[i].Seq > afterSeq {
			afterSeq = events[i].Seq
		}
	}

	// Live tail via hub + periodic DB poll for multi-instance safety.
	var liveCh <-chan *model.AgentRunEvent
	var unsub func()
	if s.eventHub != nil {
		liveCh, unsub = s.eventHub.Subscribe(run.ID)
		defer unsub()
	}

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		// Terminal + caught up: end subscription cleanly.
		fresh, err := s.agentRuns.GetRunByID(ctx, run.ID)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if fresh != nil && IsTerminalAgentRunStatus(fresh.Status) && fresh.LastEventSeq <= afterSeq {
			return nil
		}

		if liveCh == nil {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				more, err := s.agentRunEvents.ListEventsAfter(ctx, run.ID, afterSeq)
				if err != nil {
					if ctx.Err() != nil {
						return nil
					}
					return err
				}
				for i := range more {
					if err := stream.Send(toPBAgentRunEvent(&more[i])); err != nil {
						return nil
					}
					if more[i].Seq > afterSeq {
						afterSeq = more[i].Seq
					}
				}
			}
			continue
		}

		select {
		case <-ctx.Done():
			// Subscription disconnect only — run continues.
			return nil
		case ev, ok := <-liveCh:
			if !ok {
				liveCh = nil
				continue
			}
			if ev == nil || ev.Seq <= afterSeq {
				continue
			}
			if err := stream.Send(toPBAgentRunEvent(ev)); err != nil {
				return nil
			}
			afterSeq = ev.Seq
		case <-ticker.C:
			more, err := s.agentRunEvents.ListEventsAfter(ctx, run.ID, afterSeq)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			for i := range more {
				if err := stream.Send(toPBAgentRunEvent(&more[i])); err != nil {
					return nil
				}
				if more[i].Seq > afterSeq {
					afterSeq = more[i].Seq
				}
			}
		}
	}
}

func (s *AIService) loadOwnedAgentRun(ctx context.Context, hrID, runID int64) (*model.AgentRun, int32, string, error) {
	if hrID <= 0 || runID <= 0 {
		return nil, errs.ErrBadRequest, "hr_id 与 run_id 不能为空", nil
	}
	if s.agentRuns == nil {
		return nil, errs.ErrInternal, "agent run 服务未就绪", nil
	}
	run, err := s.agentRuns.GetRunByID(ctx, uint64(runID))
	if err != nil {
		return nil, 0, "", err
	}
	if run == nil || run.HrID != uint64(hrID) {
		return nil, errs.ErrForbidden, "运行不存在或无权限访问", nil
	}
	return run, errs.OK, "success", nil
}

func (s *AIService) transitionDurableRun(ctx context.Context, run *model.AgentRun, from, to string, completedAt, cancelRequestedAt, canceledAt *time.Time) error {
	if err := TransitionDurableRunStatus(ctx, s.agentRuns, run.ID, from, to, completedAt, cancelRequestedAt, canceledAt); err != nil {
		return err
	}
	if from != to {
		if _, err := s.appendRunEvent(ctx, run.ID, AgentRunEventStatusChanged, map[string]any{
			"from":   from,
			"to":     to,
			"status": to,
		}); err != nil {
			logger.L().Warn("append run.status_changed failed", zap.Error(err), zap.Uint64("run_id", run.ID))
		}
	}
	return nil
}

func (s *AIService) appendRunEvent(ctx context.Context, runID uint64, eventType string, payload any) (*model.AgentRunEvent, error) {
	if s.agentRunEvents == nil {
		return nil, nil
	}
	payloadJSON := "{}"
	switch v := payload.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			payloadJSON = v
		}
	case nil:
		payloadJSON = "{}"
	default:
		payloadJSON = safeJSON(v)
	}
	ev, err := s.agentRunEvents.AppendEvent(ctx, runID, eventType, payloadJSON)
	if err != nil {
		return nil, err
	}
	if s.eventHub != nil && ev != nil {
		s.eventHub.Publish(runID, ev)
	}
	return ev, nil
}

func (s *AIService) patchDurableRequest(ctx context.Context, runID uint64, mut func(*durableRequestPayload)) error {
	run, err := s.agentRuns.GetRunByID(ctx, runID)
	if err != nil || run == nil {
		return err
	}
	plan := decodeObjectJSON(run.PlanJSON)
	payload := loadDurableRequest(plan)
	if mut != nil {
		mut(&payload)
	}
	plan[planRequestKey] = payload
	return s.agentRuns.UpdateRunPlan(ctx, runID, safeJSON(plan))
}

func loadDurableRequest(plan map[string]any) durableRequestPayload {
	raw, ok := plan[planRequestKey]
	if !ok || raw == nil {
		return durableRequestPayload{}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return durableRequestPayload{}
	}
	var payload durableRequestPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return durableRequestPayload{}
	}
	return payload
}

func chatRequestFromDurable(hrID int64, sessionID int64, payload durableRequestPayload) *pb.ChatRequest {
	req := &pb.ChatRequest{
		HrId:                         hrID,
		SessionId:                    sessionID,
		Message:                      payload.Message,
		ApplicationId:                payload.ApplicationID,
		ModelId:                      payload.ModelID,
		SkillCapabilityKeys:          append([]string(nil), payload.SkillCapabilityKeys...),
		AgentSkillIds:                append([]int64(nil), payload.AgentSkillIDs...),
		AgentSkillSelectionConfirmed: payload.AgentSkillSelectionConfirmed,
		AgentSkillSelectionMessageId: payload.AgentSkillSelectionMessageID,
	}
	return req
}

func toPBAgentRunSnapshot(run *model.AgentRun) *pb.AgentRunSnapshot {
	if run == nil {
		return nil
	}
	snap := &pb.AgentRunSnapshot{
		RunId:             int64(run.ID),
		SessionId:         int64(run.SessionID),
		HrId:              int64(run.HrID),
		Status:            run.Status,
		AssistantText:     run.AssistantText,
		ProcessText:       run.ProcessText,
		OptionContextJson: run.OptionContextJSON,
		LastEventSeq:      run.LastEventSeq,
		ErrorType:         run.ErrorType,
		ErrorMessage:      run.ErrorMessage,
		ModelName:         run.ModelName,
		AgentType:         run.AgentType,
		AgentName:         run.AgentName,
		StartedAt:         formatTime(run.StartedAt),
		CreatedAt:         formatTime(run.CreatedAt),
		UpdatedAt:         formatTime(run.UpdatedAt),
	}
	if run.ClientRequestID != nil {
		snap.ClientRequestId = *run.ClientRequestID
	}
	if run.MessageID != nil {
		snap.MessageId = int64(*run.MessageID)
	}
	if run.HistoryID != nil {
		snap.HistoryId = int64(*run.HistoryID)
	}
	if run.ModelID != nil {
		snap.ModelId = int64(*run.ModelID)
	}
	if run.AgentID != nil {
		snap.AgentId = int64(*run.AgentID)
	}
	if run.CompletedAt != nil {
		snap.CompletedAt = formatTime(*run.CompletedAt)
	}
	if run.CancelRequestedAt != nil {
		snap.CancelRequestedAt = formatTime(*run.CancelRequestedAt)
	}
	if run.CanceledAt != nil {
		snap.CanceledAt = formatTime(*run.CanceledAt)
	}
	if strings.TrimSpace(run.ResultMetadataJSON) != "" {
		var meta pb.AgentRunResultMetadata
		if err := json.Unmarshal([]byte(run.ResultMetadataJSON), &meta); err == nil {
			snap.ResultMetadata = &meta
		} else {
			snap.ResultMetadata = &pb.AgentRunResultMetadata{RawJson: run.ResultMetadataJSON}
		}
	}
	if strings.TrimSpace(run.ConfirmationRequestJSON) != "" {
		var conf pb.AgentRunConfirmationPayload
		if err := json.Unmarshal([]byte(run.ConfirmationRequestJSON), &conf); err == nil {
			snap.ConfirmationRequest = &conf
		} else {
			snap.ConfirmationRequest = &pb.AgentRunConfirmationPayload{RawJson: run.ConfirmationRequestJSON}
		}
	}
	return snap
}

func toPBAgentRunEvent(ev *model.AgentRunEvent) *pb.AgentRunEvent {
	if ev == nil {
		return nil
	}
	out := &pb.AgentRunEvent{
		RunId:       int64(ev.RunID),
		Seq:         ev.Seq,
		EventType:   ev.EventType,
		PayloadJson: ev.PayloadJSON,
		CreatedAt:   formatTime(ev.CreatedAt),
	}
	payload := decodeObjectJSON(ev.PayloadJSON)
	if status, ok := payload["status"].(string); ok {
		out.Status = status
	}
	if delta, ok := payload["delta"].(string); ok {
		out.Delta = delta
	}
	if snap, ok := payload["snapshot_text"].(string); ok {
		out.SnapshotText = snap
	}
	if tool, ok := payload["tool_name"].(string); ok {
		out.ToolName = tool
	}
	if et, ok := payload["error_type"].(string); ok {
		out.ErrorType = et
	}
	if em, ok := payload["error_message"].(string); ok {
		out.ErrorMessage = em
	}
	if ev.EventType == AgentRunEventConfirmationRequired {
		var conf pb.AgentRunConfirmationPayload
		if err := json.Unmarshal([]byte(ev.PayloadJSON), &conf); err == nil {
			out.Confirmation = &conf
		} else {
			out.Confirmation = &pb.AgentRunConfirmationPayload{RawJson: ev.PayloadJSON}
		}
	}
	return out
}
