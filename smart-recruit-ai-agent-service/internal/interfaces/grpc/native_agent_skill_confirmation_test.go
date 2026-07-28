package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domainagentskill "smart-recruit-ai-agent-service/internal/domain/agentskill"
	commonsai "smart-recruit-commons/ai"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-platform-go/observability"
	"smart-recruit-proto/recruitment/pb"
)

type agentSkillRunTestStore struct {
	*agentRunTestStore
	resolution       RuntimeModelInfo
	finalizeFailures int
}

type cancelTransitionRaceStore struct {
	*agentRunTestStore
	hookMu    sync.Mutex
	attempts  int
	beforeCAS func(context.Context, *agentRunTestStore, int, int64, string, string)
}

type cancelTransitionLockWaitStore struct {
	*cancelTransitionRaceStore
	initialRead sync.Once
	readDone    chan struct{}
}

func (s *cancelTransitionLockWaitStore) GetAgentRun(
	ctx context.Context,
	ownerID int64,
	runID int64,
) (AgentRunRow, bool, error) {
	run, found, err := s.agentRunTestStore.GetAgentRun(ctx, ownerID, runID)
	s.initialRead.Do(func() { close(s.readDone) })
	return run, found, err
}

func (s *cancelTransitionRaceStore) TransitionAgentRunConfirmation(
	ctx context.Context,
	ownerID int64,
	runID int64,
	fromStatus string,
	toStatus string,
	expectedPlanJSON string,
	planJSON string,
	optionContextJSON string,
	errorType string,
	errorMessage string,
) (AgentRunRow, bool, error) {
	s.hookMu.Lock()
	s.attempts++
	attempt := s.attempts
	hook := s.beforeCAS
	s.hookMu.Unlock()
	if hook != nil {
		hook(ctx, s.agentRunTestStore, attempt, runID, fromStatus, toStatus)
	}
	return s.agentRunTestStore.TransitionAgentRunConfirmation(
		ctx,
		ownerID,
		runID,
		fromStatus,
		toStatus,
		expectedPlanJSON,
		planJSON,
		optionContextJSON,
		errorType,
		errorMessage,
	)
}

func (s *cancelTransitionRaceStore) transitionAttempts() int {
	s.hookMu.Lock()
	defer s.hookMu.Unlock()
	return s.attempts
}

func (s *agentSkillRunTestStore) FinalizeAgentRunSkillApproval(
	ctx context.Context,
	ownerID int64,
	runID int64,
	expectedPlanJSON string,
	readyPlanJSON string,
	optionContextJSON string,
	eventPayloadJSON string,
) (AgentRunRow, AgentRunEventRow, bool, error) {
	s.mu.Lock()
	if s.finalizeFailures > 0 {
		s.finalizeFailures--
		s.mu.Unlock()
		return AgentRunRow{}, AgentRunEventRow{}, false, errors.New("injected confirmation finalize failure")
	}
	s.mu.Unlock()
	return s.agentRunTestStore.FinalizeAgentRunSkillApproval(
		ctx,
		ownerID,
		runID,
		expectedPlanJSON,
		readyPlanJSON,
		optionContextJSON,
		eventPayloadJSON,
	)
}

func (s *agentSkillRunTestStore) ResolveCapabilityRuntimeModel(
	_ context.Context,
	_, _ string,
	_ int64,
	requestedModelID int64,
) (CapabilityRuntimeModelResolution, error) {
	model := s.resolution
	if requestedModelID > 0 {
		model.ID = requestedModelID
		model.RequestedModelID = requestedModelID
	}
	return CapabilityRuntimeModelResolution{
		EffectiveModelID:       model.ID,
		ModelName:              model.Name,
		ProviderName:           model.ProviderName,
		RequestedModelID:       model.RequestedModelID,
		CapabilityVersionID:    model.CapabilityVersionID,
		CapabilitySnapshotHash: model.CapabilitySnapshotHash,
		ContextWindowTokens:    model.ContextWindowTokens,
		MaxOutputTokens:        model.MaxOutputTokens,
		ConfigurationRefs:      model.ConfigurationRefs,
		SkillRuntimePolicy:     model.SkillRuntimePolicy,
	}, nil
}

func (s *agentRunTestStore) TransitionAgentRunConfirmation(
	_ context.Context,
	ownerID int64,
	runID int64,
	fromStatus string,
	toStatus string,
	expectedPlanJSON string,
	planJSON string,
	optionContextJSON string,
	errorType string,
	errorMessage string,
) (AgentRunRow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok || run.OwnerID != ownerID || run.Status != fromStatus || run.PlanJSON != expectedPlanJSON {
		return AgentRunRow{}, false, nil
	}
	now := time.Now()
	run.Status = toStatus
	run.PlanJSON = planJSON
	run.OptionContextJSON = optionContextJSON
	run.ErrorType = errorType
	run.ErrorMessage = errorMessage
	run.UpdatedAt = now
	if isTerminalAgentRunStatus(toStatus) {
		run.CompletedAt = &now
	}
	if toStatus == agentRunStatusCanceled {
		run.CanceledAt = &now
	}
	s.runs[runID] = run
	s.statusUpdates = append(s.statusUpdates, toStatus)
	return run, true, nil
}

func (s *agentRunTestStore) FinalizeAgentRunSkillApproval(
	_ context.Context,
	ownerID int64,
	runID int64,
	expectedPlanJSON string,
	readyPlanJSON string,
	optionContextJSON string,
	eventPayloadJSON string,
) (AgentRunRow, AgentRunEventRow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok || run.OwnerID != ownerID || run.Status != agentRunStatusQueued || run.PlanJSON != expectedPlanJSON {
		return run, AgentRunEventRow{}, false, nil
	}
	s.runEventSeq[runID]++
	event := AgentRunEventRow{
		RunID:       runID,
		Seq:         s.runEventSeq[runID],
		EventType:   "confirmation.accepted",
		PayloadJSON: eventPayloadJSON,
		CreatedAt:   time.Now(),
	}
	s.runEvents[runID] = append(s.runEvents[runID], event)
	run.PlanJSON = readyPlanJSON
	run.OptionContextJSON = optionContextJSON
	run.LastEventSeq = event.Seq
	run.UpdatedAt = event.CreatedAt
	s.runs[runID] = run
	return run, event, true, nil
}

func (s *agentRunTestStore) EnsureAgentRunUserMessage(
	_ context.Context,
	ownerID int64,
	runID int64,
	message ChatMessageRow,
) (ChatMessageRow, AgentRunRow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok || run.OwnerID != ownerID {
		return ChatMessageRow{}, AgentRunRow{}, false, errInvalidAgentSkillConfirmation
	}
	if run.MessageID > 0 {
		for _, existing := range s.messages {
			if existing.ID == run.MessageID &&
				existing.OwnerID == ownerID &&
				existing.SessionID == run.SessionID &&
				existing.Role == "user" {
				return existing, run, false, nil
			}
		}
		return ChatMessageRow{}, AgentRunRow{}, false, errInvalidAgentSkillConfirmation
	}
	s.nextMessageID++
	message.ID = s.nextMessageID
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}
	s.messages = append(s.messages, message)
	run.MessageID = message.ID
	run.UpdatedAt = message.CreatedAt
	s.runs[runID] = run
	return message, run, true, nil
}

func TestAgentSkillDurableConfirmationPreflightRefreshAndApprove(t *testing.T) {
	service, store, provider, actor := newAgentSkillConfirmationRuntime(
		t,
		domainagentskill.RiskLevelHigh,
		[]int64{101},
	)
	run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
	if provider.callCount() != 0 {
		t.Fatalf("provider calls before approval = %d, want zero", provider.callCount())
	}
	snapshot := mapAgentRunSnapshot(run)
	confirmation := snapshot.GetConfirmationRequest()
	if confirmation == nil ||
		!confirmation.GetRequired() ||
		confirmation.GetAgentSkillConfirmationId() == "" ||
		len(confirmation.GetCandidates()) != 1 ||
		confirmation.GetCandidates()[0].GetVersionId() != 101 ||
		confirmation.GetCandidates()[0].GetCompiledHash() == "" ||
		confirmation.GetCandidates()[0].GetRisk() != pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_HIGH ||
		confirmation.GetCandidates()[0].GetActivationPolicy() != pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_CONFIRM ||
		len(confirmation.GetRecommendedAgentSkillVersionIds()) != 1 ||
		confirmation.GetRecommendedAgentSkillVersionIds()[0] != 101 {
		t.Fatalf("confirmation snapshot = %#v", confirmation)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, confirmation.GetAgentSkillConfirmationExpiresAt())
	if err != nil || time.Until(expiresAt) < 9*time.Minute || time.Until(expiresAt) > 11*time.Minute {
		t.Fatalf("expires_at = %q err=%v", confirmation.GetAgentSkillConfirmationExpiresAt(), err)
	}
	refreshed, err := service.GetAgentRun(actor, &pb.GetAgentRunRequest{HrId: 77, RunId: run.ID})
	if err != nil || refreshed.GetRun().GetConfirmationRequest().GetAgentSkillConfirmationId() != confirmation.GetAgentSkillConfirmationId() {
		t.Fatalf("refresh response=%#v err=%v", refreshed, err)
	}
	events := store.runEvents[run.ID]
	if len(events) == 0 || events[len(events)-1].EventType != "confirmation.required" ||
		strings.Contains(events[len(events)-1].PayloadJSON, "resume screening core") ||
		strings.Contains(events[len(events)-1].PayloadJSON, "resume screening private user message") {
		t.Fatalf("confirmation event is missing or leaked content: %#v", events)
	}

	response, err := service.ConfirmAgentRun(actor, &pb.ConfirmAgentRunRequest{
		HrId:                           77,
		RunId:                          run.ID,
		ClientRequestId:                "approve-1",
		AgentSkillConfirmationId:       confirmation.GetAgentSkillConfirmationId(),
		AgentSkillConfirmationDecision: pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_APPROVE,
		SelectedAgentSkillVersionIds:   []int64{101},
	})
	if err != nil || response.GetCode() != 0 ||
		(response.GetRun().GetStatus() != agentRunStatusQueued &&
			response.GetRun().GetStatus() != agentRunStatusRunning) {
		t.Fatalf("approve response=%#v err=%v", response, err)
	}
	waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
	close(provider.release)
	waitUntilAgentRunTest(t, time.Second, func() bool {
		final, found := store.runSnapshot(run.ID)
		return found && final.Status == agentRunStatusSucceeded
	})
	if prompts := provider.promptsSnapshot(); len(prompts) != 1 || !strings.Contains(prompts[0], "resume screening core") {
		t.Fatalf("approved prompt = %#v", prompts)
	}
	if messages := store.messagesSnapshot(); countAgentSkillTestMessages(messages, "user") != 1 {
		t.Fatalf("user message was duplicated across resume: %#v", messages)
	}
	metrics := service.metrics.Prometheus()
	if strings.Count(metrics, "smart_recruit_agent_skill_confirmation_total{") != 2 ||
		!strings.Contains(metrics, `result="confirmation_required",reason="confirmation_required"`) ||
		!strings.Contains(metrics, `result="passed",reason="approved"`) {
		t.Fatalf("confirmation metrics were not emitted exactly once per transition:\n%s", metrics)
	}
}

func TestAgentSkillDurableConfirmationRejectCancelAndMalformedCancelNeverExecute(t *testing.T) {
	t.Run("reject is terminal", func(t *testing.T) {
		service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		confirmationID := mapAgentRunSnapshot(run).GetConfirmationRequest().GetAgentSkillConfirmationId()
		response, err := service.ConfirmAgentRun(actor, &pb.ConfirmAgentRunRequest{
			HrId:                           77,
			RunId:                          run.ID,
			AgentSkillConfirmationId:       confirmationID,
			AgentSkillConfirmationDecision: pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_REJECT,
		})
		if err != nil || response.GetCode() != 0 || response.GetRun().GetStatus() != agentRunStatusCanceled {
			t.Fatalf("reject response=%#v err=%v", response, err)
		}
		if provider.callCount() != 0 || len(store.toolTraces) != 0 {
			t.Fatalf("reject executed provider/tool: provider=%d traces=%#v", provider.callCount(), store.toolTraces)
		}
		if store.countEvents(run.ID, "run.canceled") != 1 {
			t.Fatalf("events=%#v", store.eventTypes(run.ID))
		}
	})

	t.Run("cancel ignores malformed pending payload", func(t *testing.T) {
		service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		store.mu.Lock()
		corrupt := store.runs[run.ID]
		corrupt.PlanJSON = `{"durable_request":{"pending_agent_skill_confirmation":`
		store.runs[run.ID] = corrupt
		store.mu.Unlock()
		response, err := service.CancelAgentRun(actor, &pb.CancelAgentRunRequest{HrId: 77, RunId: run.ID})
		if err != nil || response.GetCode() != 0 || response.GetRun().GetStatus() != agentRunStatusCanceled {
			t.Fatalf("cancel response=%#v err=%v", response, err)
		}
		if provider.callCount() != 0 {
			t.Fatalf("provider calls after malformed cancel = %d", provider.callCount())
		}
	})

	t.Run("valid cancel atomically terminates", func(t *testing.T) {
		service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		response, err := service.CancelAgentRun(actor, &pb.CancelAgentRunRequest{HrId: 77, RunId: run.ID})
		if err != nil || response.GetCode() != 0 || response.GetRun().GetStatus() != agentRunStatusCanceled {
			t.Fatalf("cancel response=%#v err=%v", response, err)
		}
		if provider.callCount() != 0 || store.countEvents(run.ID, "run.canceled") != 1 {
			t.Fatalf("cancel provider=%d events=%#v", provider.callCount(), store.eventTypes(run.ID))
		}
		if output := service.metrics.Prometheus(); strings.Count(output, `reason="canceled"`) != 1 {
			t.Fatalf("cancel metric not emitted exactly once:\n%s", output)
		}
	})

	t.Run("running approved execution cancels under its lease", func(t *testing.T) {
		service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		response, err := service.ConfirmAgentRun(actor, agentSkillApprovalRequest(run, "cancel-running"))
		if err != nil || response.GetCode() != 0 {
			t.Fatalf("approve response=%#v err=%v", response, err)
		}
		waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
		if _, err := service.CancelAgentRun(actor, &pb.CancelAgentRunRequest{HrId: 77, RunId: run.ID}); err != nil {
			t.Fatalf("cancel running approval: %v", err)
		}
		waitUntilAgentRunTest(t, time.Second, func() bool {
			final, found := store.runSnapshot(run.ID)
			return found && final.Status == agentRunStatusCanceled
		})
		if store.countEvents(run.ID, "run.canceled") != 1 {
			t.Fatalf("canceled events=%v", store.eventTypes(run.ID))
		}
	})
}

func TestAgentSkillDurableConfirmationRejectsActorPayloadAndSnapshotTampering(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*agentSkillRunTestStore, AgentRunRow, *pb.ConfirmAgentRunRequest) context.Context
	}{
		{
			name: "cross user",
			mutate: func(_ *agentSkillRunTestStore, _ AgentRunRow, _ *pb.ConfirmAgentRunRequest) context.Context {
				return agentSkillTestActor(9, 999)
			},
		},
		{
			name: "cross tenant",
			mutate: func(_ *agentSkillRunTestStore, _ AgentRunRow, _ *pb.ConfirmAgentRunRequest) context.Context {
				return agentSkillTestActor(10, 88)
			},
		},
		{
			name: "altered selected version order",
			mutate: func(_ *agentSkillRunTestStore, _ AgentRunRow, req *pb.ConfirmAgentRunRequest) context.Context {
				req.SelectedAgentSkillVersionIds = []int64{102, 101}
				return agentSkillTestActor(9, 88)
			},
		},
		{
			name: "mcp payload isolation",
			mutate: func(_ *agentSkillRunTestStore, _ AgentRunRow, req *pb.ConfirmAgentRunRequest) context.Context {
				req.ConfirmationPayloadJson = `{"type":"mcp_tool","approved":true}`
				return agentSkillTestActor(9, 88)
			},
		},
		{
			name: "changed release snapshot",
			mutate: func(store *agentSkillRunTestStore, _ AgentRunRow, _ *pb.ConfirmAgentRunRequest) context.Context {
				store.resolution.CapabilitySnapshotHash = strings.Repeat("c", 64)
				return agentSkillTestActor(9, 88)
			},
		},
		{
			name: "changed package hash",
			mutate: func(store *agentSkillRunTestStore, _ AgentRunRow, _ *pb.ConfirmAgentRunRequest) context.Context {
				store.agentSkillVersionDocs[0].CoreMarkdown += " tampered"
				return agentSkillTestActor(9, 88)
			},
		},
		{
			name: "changed message digest",
			mutate: func(store *agentSkillRunTestStore, run AgentRunRow, _ *pb.ConfirmAgentRunRequest) context.Context {
				payload := agentRunPayloadFromRow(run)
				payload.Message = "changed message"
				store.mu.Lock()
				changed := store.runs[run.ID]
				changed.PlanJSON = agentRunPlanJSON(payload)
				store.runs[run.ID] = changed
				store.mu.Unlock()
				return agentSkillTestActor(9, 88)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101, 102})
			run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
			confirmationID := mapAgentRunSnapshot(run).GetConfirmationRequest().GetAgentSkillConfirmationId()
			req := &pb.ConfirmAgentRunRequest{
				HrId:                           77,
				RunId:                          run.ID,
				AgentSkillConfirmationId:       confirmationID,
				AgentSkillConfirmationDecision: pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_APPROVE,
				SelectedAgentSkillVersionIds:   []int64{101, 102},
			}
			requestCtx := tt.mutate(store, run, req)
			response, err := service.ConfirmAgentRun(requestCtx, req)
			if err != nil || response.GetCode() != agentRunCodeBadRequest {
				t.Fatalf("tampered response=%#v err=%v", response, err)
			}
			current, _ := store.runSnapshot(run.ID)
			if current.Status != agentRunStatusWaitingConfirmation || provider.callCount() != 0 {
				t.Fatalf("tamper changed run/provider: status=%s calls=%d", current.Status, provider.callCount())
			}
		})
	}
}

func TestAgentSkillDurableConfirmationCannotCrossRuns(t *testing.T) {
	service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
	store.seedChatSession(ownerRoleHR, 77, 102, "second skill confirmation")
	first := createWaitingAgentSkillRun(t, service, actor, 101, nil)
	second := createWaitingAgentSkillRun(t, service, actor, 102, nil)
	firstID := mapAgentRunSnapshot(first).GetConfirmationRequest().GetAgentSkillConfirmationId()
	response, err := service.ConfirmAgentRun(actor, &pb.ConfirmAgentRunRequest{
		HrId:                           77,
		RunId:                          second.ID,
		AgentSkillConfirmationId:       firstID,
		AgentSkillConfirmationDecision: pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_APPROVE,
		SelectedAgentSkillVersionIds:   []int64{101},
	})
	if err != nil || response.GetCode() != agentRunCodeBadRequest {
		t.Fatalf("cross-run response=%#v err=%v", response, err)
	}
	current, _ := store.runSnapshot(second.ID)
	if current.Status != agentRunStatusWaitingConfirmation || provider.callCount() != 0 {
		t.Fatalf("cross-run confirmation changed state: status=%s calls=%d", current.Status, provider.callCount())
	}
}

func TestAgentSkillDurableConfirmationExpiryAndConcurrentConsumption(t *testing.T) {
	t.Run("expired confirmation fails closed and refresh is terminal", func(t *testing.T) {
		service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		payload := agentRunPayloadFromRow(run)
		payload.PendingAgentSkillConfirmation.ExpiresAt = time.Now().Add(-time.Minute).Format(time.RFC3339Nano)
		store.mu.Lock()
		changed := store.runs[run.ID]
		changed.PlanJSON = agentRunPlanJSON(payload)
		store.runs[run.ID] = changed
		store.mu.Unlock()

		refreshed, err := service.GetActiveAgentRun(actor, &pb.GetActiveAgentRunRequest{HrId: 77, SessionId: 101})
		if err != nil || refreshed.GetHasActiveRun() || refreshed.GetRun().GetStatus() != agentRunStatusFailed ||
			refreshed.GetRun().GetErrorType() != "AGENT_SKILL_CONFIRMATION_EXPIRED" {
			t.Fatalf("expired refresh=%#v err=%v", refreshed, err)
		}
		if provider.callCount() != 0 {
			t.Fatalf("expired provider calls=%d", provider.callCount())
		}
	})

	t.Run("only one concurrent approval consumes", func(t *testing.T) {
		service, _, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		confirmationID := mapAgentRunSnapshot(run).GetConfirmationRequest().GetAgentSkillConfirmationId()
		var wg sync.WaitGroup
		codes := make(chan int32, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				response, err := service.ConfirmAgentRun(actor, &pb.ConfirmAgentRunRequest{
					HrId:                           77,
					RunId:                          run.ID,
					ClientRequestId:                string(rune('a' + index)),
					AgentSkillConfirmationId:       confirmationID,
					AgentSkillConfirmationDecision: pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_APPROVE,
					SelectedAgentSkillVersionIds:   []int64{101},
				})
				if err != nil {
					codes <- -1
					return
				}
				codes <- response.GetCode()
			}(i)
		}
		wg.Wait()
		close(codes)
		successes, rejected := 0, 0
		for code := range codes {
			if code == 0 {
				successes++
			} else if code == agentRunCodeBadRequest {
				rejected++
			}
		}
		if successes != 1 || rejected != 1 {
			t.Fatalf("concurrent result successes=%d rejected=%d", successes, rejected)
		}
		waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
		close(provider.release)
	})
}

func TestAgentSkillApprovalHandoffRecoversAcrossDurabilityBoundaries(t *testing.T) {
	t.Run("retry after approval CAS before accepted event", func(t *testing.T) {
		service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		request := agentSkillApprovalRequest(run, "approval-recovery-cas-event")
		store.finalizeFailures = 1

		if _, err := service.ConfirmAgentRun(actor, request); err == nil {
			t.Fatal("first confirmation unexpectedly survived injected finalize failure")
		}
		current, _ := store.runSnapshot(run.ID)
		currentPayload := agentRunPayloadFromRow(current)
		if current.Status != agentRunStatusQueued ||
			currentPayload.AgentSkillApproval == nil ||
			currentPayload.AgentSkillApproval.DispatchState != agentSkillDispatchPendingEvent ||
			store.countEvents(run.ID, "confirmation.accepted") != 0 ||
			provider.callCount() != 0 {
			t.Fatalf("partial handoff run=%#v payload=%#v events=%v provider=%d", current, currentPayload, store.eventTypes(run.ID), provider.callCount())
		}

		response, err := service.ConfirmAgentRun(actor, request)
		if err != nil || response.GetCode() != 0 {
			t.Fatalf("retry response=%#v err=%v", response, err)
		}
		waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
		if store.countEvents(run.ID, "confirmation.accepted") != 1 {
			t.Fatalf("accepted events=%v", store.eventTypes(run.ID))
		}
		if output := service.metrics.Prometheus(); strings.Count(output, `result="passed",reason="approved"`) != 1 {
			t.Fatalf("approved metric was not emitted exactly once:\n%s", output)
		}
		close(provider.release)
	})

	t.Run("retry after accepted event before dispatch", func(t *testing.T) {
		service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		request := agentSkillApprovalRequest(run, "approval-recovery-event-dispatch")
		ready := prepareApprovedAgentSkillRunWithoutDispatch(t, service, actor, run, request)
		if ready.Status != agentRunStatusQueued ||
			store.countEvents(run.ID, "confirmation.accepted") != 1 ||
			provider.callCount() != 0 {
			t.Fatalf("ready run=%#v events=%v provider=%d", ready, store.eventTypes(run.ID), provider.callCount())
		}

		if _, err := service.GetAgentRun(actor, &pb.GetAgentRunRequest{HrId: 77, RunId: run.ID}); err != nil {
			t.Fatalf("refresh did not dispatch ready approval: %v", err)
		}
		waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
		if store.countEvents(run.ID, "confirmation.accepted") != 1 {
			t.Fatalf("accepted event duplicated: %v", store.eventTypes(run.ID))
		}
		close(provider.release)
	})

	t.Run("expired worker claim is requeued and claimed once", func(t *testing.T) {
		service, _, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		request := agentSkillApprovalRequest(run, "approval-recovery-stale-claim")
		ready := prepareApprovedAgentSkillRunWithoutDispatch(t, service, actor, run, request)
		claimed, shouldExecute, err := service.beginAgentRunExecution(context.Background(), ready)
		if err != nil || !shouldExecute || claimed.Status != agentRunStatusRunning {
			t.Fatalf("claim run=%#v execute=%v err=%v", claimed, shouldExecute, err)
		}
		claimedPayload := agentRunPayloadFromRow(claimed)
		claimedPayload.AgentSkillApproval.DispatchLeaseExpiresAt = time.Now().Add(-time.Minute).Format(time.RFC3339Nano)
		stale, transitioned, err := service.transitionAgentRunState(
			context.Background(),
			claimed,
			agentRunStatusRunning,
			agentRunStatusRunning,
			agentRunPlanJSON(claimedPayload),
			claimed.OptionContextJSON,
			"",
			"",
		)
		if err != nil || !transitioned {
			t.Fatalf("seed stale claim run=%#v transitioned=%v err=%v", stale, transitioned, err)
		}

		if _, err := service.GetAgentRun(actor, &pb.GetAgentRunRequest{HrId: 77, RunId: run.ID}); err != nil {
			t.Fatalf("recover stale claim: %v", err)
		}
		waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
		if _, err := service.GetAgentRun(actor, &pb.GetAgentRunRequest{HrId: 77, RunId: run.ID}); err != nil {
			t.Fatalf("refresh claimed run: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
		if provider.callCount() != 1 {
			t.Fatalf("provider calls after duplicate recovery=%d", provider.callCount())
		}
		close(provider.release)
	})

	t.Run("duplicate dispatchers share one durable claim", func(t *testing.T) {
		service, _, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		request := agentSkillApprovalRequest(run, "approval-duplicate-dispatch")
		ready := prepareApprovedAgentSkillRunWithoutDispatch(t, service, actor, run, request)
		service.dispatchAgentRun(ready)
		service.dispatchAgentRun(ready)
		waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
		time.Sleep(20 * time.Millisecond)
		if provider.callCount() != 1 {
			t.Fatalf("provider calls after duplicate dispatch=%d", provider.callCount())
		}
		close(provider.release)
	})
}

func TestAgentSkillRecoveryNeverDispatchesOrdinaryRun(t *testing.T) {
	store := newAgentRunTestStore()
	store.seedChatSession(ownerRoleHR, 77, 101, "ordinary run")
	provider := newBlockingAgentRunProvider("ordinary reply")
	service := &nativeAIService{store: store, provider: provider}
	run := fallbackAgentRun(77, 101, "ordinary-run", agentRunDurablePayload{
		Message: "ordinary provider request",
		ModelID: 1,
	})
	created, _, err := store.CreateAgentRun(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	service.dispatchAgentRun(created)
	waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
	for i := 0; i < 8; i++ {
		if _, err := service.GetAgentRun(context.Background(), &pb.GetAgentRunRequest{
			HrId:  77,
			RunId: created.ID,
		}); err != nil {
			t.Fatalf("poll %d: %v", i, err)
		}
	}
	time.Sleep(20 * time.Millisecond)
	if provider.callCount() != 1 {
		t.Fatalf("ordinary provider calls after polling = %d, want 1", provider.callCount())
	}
	close(provider.release)
}

func TestAgentSkillConfirmationDeadlineUsesConfirmationAndLeaseState(t *testing.T) {
	t.Run("approval after ordinary run timeout remains valid", func(t *testing.T) {
		service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		store.mu.Lock()
		stale := store.runs[run.ID]
		stale.StartedAt = time.Now().Add(-5 * time.Minute)
		store.runs[run.ID] = stale
		store.mu.Unlock()

		active, err := service.GetActiveAgentRun(actor, &pb.GetActiveAgentRunRequest{HrId: 77, SessionId: 101})
		if err != nil || !active.GetHasActiveRun() ||
			active.GetRun().GetStatus() != agentRunStatusWaitingConfirmation {
			t.Fatalf("old waiting run was timed out: response=%#v err=%v", active, err)
		}
		response, err := service.ConfirmAgentRun(actor, agentSkillApprovalRequest(run, "late-approval"))
		if err != nil || response.GetCode() != 0 {
			t.Fatalf("late approval response=%#v err=%v", response, err)
		}
		waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
		close(provider.release)
	})

	t.Run("stale crashed claim recovers despite original start", func(t *testing.T) {
		service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
		ready := prepareApprovedAgentSkillRunWithoutDispatch(
			t,
			service,
			actor,
			run,
			agentSkillApprovalRequest(run, "crashed-claim"),
		)
		claimed, shouldExecute, err := service.beginAgentRunExecution(context.Background(), ready)
		if err != nil || !shouldExecute {
			t.Fatalf("claim execute=%v run=%#v err=%v", shouldExecute, claimed, err)
		}
		payload := agentRunPayloadFromRow(claimed)
		payload.AgentSkillApproval.DispatchLeaseExpiresAt = time.Now().Add(-time.Minute).Format(time.RFC3339Nano)
		claimed.StartedAt = time.Now().Add(-20 * time.Minute)
		store.mu.Lock()
		stored := store.runs[claimed.ID]
		stored.StartedAt = claimed.StartedAt
		store.runs[claimed.ID] = stored
		store.mu.Unlock()
		stale, transitioned, err := service.transitionAgentRunState(
			context.Background(),
			claimed,
			agentRunStatusRunning,
			agentRunStatusRunning,
			agentRunPlanJSON(payload),
			claimed.OptionContextJSON,
			"",
			"",
		)
		if err != nil || !transitioned || !service.agentRunExceededDeadline(stale, time.Now()) {
			t.Fatalf("stale claim seed=%#v transitioned=%v deadline=%v err=%v", stale, transitioned, service.agentRunExceededDeadline(stale, time.Now()), err)
		}
		if _, err := service.GetAgentRun(actor, &pb.GetAgentRunRequest{HrId: 77, RunId: run.ID}); err != nil {
			t.Fatalf("recover stale claim: %v", err)
		}
		waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
		close(provider.release)
	})
}

type ignoringCancelAgentRunProvider struct {
	reply   string
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func newIgnoringCancelAgentRunProvider(reply string) *ignoringCancelAgentRunProvider {
	return &ignoringCancelAgentRunProvider{
		reply:   reply,
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (p *ignoringCancelAgentRunProvider) Complete(context.Context, string) (string, error) {
	p.once.Do(func() { close(p.started) })
	<-p.release
	return p.reply, nil
}

func TestAgentSkillExecutionLeaseFencesOldReplicaCompletion(t *testing.T) {
	service, store, _, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
	run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
	ready := prepareApprovedAgentSkillRunWithoutDispatch(
		t,
		service,
		actor,
		run,
		agentSkillApprovalRequest(run, "cross-replica-fence"),
	)

	oldProvider := newIgnoringCancelAgentRunProvider("stale old reply")
	oldService := newNativeAIService(store, oldProvider, nil, nil, nil)
	oldService.skillPackageV2Enabled = true
	oldService.metrics = observability.NewRegistry("ai-agent")
	oldService.dispatchAgentRun(ready)
	select {
	case <-oldProvider.started:
	case <-time.After(time.Second):
		t.Fatal("old replica provider did not start")
	}

	current, _ := store.runSnapshot(run.ID)
	oldLease := agentRunTestSkillLease(current)
	payload := agentRunPayloadFromRow(current)
	payload.AgentSkillApproval.DispatchLeaseExpiresAt = time.Now().Add(-time.Minute).Format(time.RFC3339Nano)
	_, transitioned, err := oldService.transitionAgentRunState(
		context.Background(),
		current,
		agentRunStatusRunning,
		agentRunStatusRunning,
		agentRunRuntimePlanJSON(payload, nil, hrRuntimeGovernanceContext{}, commonsai.RecruitingPlan{}, 0, "", nil, ""),
		current.OptionContextJSON,
		"",
		"",
	)
	if err != nil || !transitioned {
		t.Fatalf("expire old lease transitioned=%v err=%v", transitioned, err)
	}

	newProvider := newBlockingAgentRunProvider("fresh new reply")
	newService := newNativeAIService(store, newProvider, nil, nil, nil)
	newService.skillPackageV2Enabled = true
	newService.metrics = observability.NewRegistry("ai-agent")
	if _, err := newService.GetAgentRun(actor, &pb.GetAgentRunRequest{HrId: 77, RunId: run.ID}); err != nil {
		t.Fatalf("new replica recovery: %v", err)
	}
	waitUntilAgentRunTest(t, time.Second, func() bool { return newProvider.callCount() == 1 })
	fresh, _ := store.runSnapshot(run.ID)
	if fresh.Status != agentRunStatusRunning || agentRunTestSkillLease(fresh) == oldLease {
		t.Fatalf("fresh lease was not claimed: old=%q run=%#v", oldLease, fresh)
	}

	close(oldProvider.release)
	time.Sleep(30 * time.Millisecond)
	if current, _ := store.runSnapshot(run.ID); current.Status != agentRunStatusRunning ||
		current.AssistantText == "stale old reply" ||
		store.countEvents(run.ID, "run.result") != 0 {
		t.Fatalf("old replica wrote after losing lease: run=%#v events=%v", current, store.eventTypes(run.ID))
	}
	close(newProvider.release)
	waitUntilAgentRunTest(t, time.Second, func() bool {
		final, found := store.runSnapshot(run.ID)
		return found && final.Status == agentRunStatusSucceeded
	})
	final, _ := store.runSnapshot(run.ID)
	if final.AssistantText != "fresh new reply" ||
		store.countEvents(run.ID, "run.result") != 1 ||
		countAgentSkillTestMessages(store.messagesSnapshot(), "assistant") != 1 {
		t.Fatalf("final fenced result=%#v events=%v messages=%#v", final, store.eventTypes(run.ID), store.messagesSnapshot())
	}
}

type countingAgentSkillToolRunner struct {
	mu    sync.Mutex
	calls int
}

func (r *countingAgentSkillToolRunner) Execute(context.Context, int64, string, map[string]any) (commonsai.ToolResult, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return commonsai.ToolResult{Content: `{"ok":true}`}, nil
}

func (r *countingAgentSkillToolRunner) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func TestAgentSkillExecutionLeaseFencesOldReplicaToolAttemptAndEvidenceAfterCancel(t *testing.T) {
	service, store, _, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
	run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
	ready := prepareApprovedAgentSkillRunWithoutDispatch(
		t,
		service,
		actor,
		run,
		agentSkillApprovalRequest(run, "cross-replica-tool-fence"),
	)
	claimed, shouldExecute, err := service.beginAgentRunExecution(context.Background(), ready)
	if err != nil || !shouldExecute || claimed.Status != agentRunStatusRunning {
		t.Fatalf("claim run=%#v execute=%v err=%v", claimed, shouldExecute, err)
	}
	approval := agentRunPayloadFromRow(claimed).AgentSkillApproval
	executionCtx := withAgentSkillApproval(context.Background(), claimed.ID, claimed.OwnerID, approval)
	delegate := &countingAgentSkillToolRunner{}
	runner := newHRRuntimeToolRunner(delegate, []string{"get_job_list"}, service, claimed.ID)

	attempt := make(chan error, 1)
	release := make(chan struct{})
	go func() {
		<-release // Models a provider that ignores its canceled request context.
		_, attemptErr := runner.Execute(executionCtx, claimed.OwnerID, "get_job_list", nil)
		attempt <- attemptErr
	}()

	cancelReplica := newNativeAIService(store, nil, nil, nil, nil)
	response, err := cancelReplica.CancelAgentRun(actor, &pb.CancelAgentRunRequest{HrId: claimed.OwnerID, RunId: claimed.ID})
	if err != nil || response.GetCode() != 0 || response.GetRun().GetStatus() != agentRunStatusCanceled {
		t.Fatalf("cross-replica cancel response=%#v err=%v", response, err)
	}
	close(release)
	if attemptErr := <-attempt; !errors.Is(attemptErr, errAgentRunExecutionLeaseLost) {
		t.Fatalf("old tool attempt error=%v, want lease lost", attemptErr)
	}
	if delegate.callCount() != 0 {
		t.Fatalf("old tool delegate calls=%d, want 0", delegate.callCount())
	}
	_, persistErr := service.persistHRToolTrace(executionCtx, claimed.OwnerID, ToolTraceRow{
		SessionID:  claimed.SessionID,
		AgentRunID: claimed.ID,
		ToolName:   "get_job_list",
		Status:     "error",
		ErrorMsg:   errAgentRunExecutionLeaseLost.Error(),
	})
	if !errors.Is(persistErr, errAgentRunExecutionLeaseLost) {
		t.Fatalf("old trace persist error=%v, want lease lost", persistErr)
	}
	if len(store.toolTraces) != 0 || len(store.runSteps[claimed.ID]) != 0 {
		t.Fatalf("old evidence persisted traces=%#v steps=%#v", store.toolTraces, store.runSteps[claimed.ID])
	}
}

func TestCancelAgentRunRetriesAfterConcurrentPlanChange(t *testing.T) {
	base := newAgentRunTestStore()
	run := fallbackAgentRun(77, 101, "cancel-plan-race", agentRunDurablePayload{
		Message: "cancel after plan update",
		ModelID: 123,
	})
	run.Status = agentRunStatusRunning
	created, _, err := base.CreateAgentRun(context.Background(), run)
	if err != nil {
		t.Fatalf("CreateAgentRun seed: %v", err)
	}
	updatedPlan := agentRunPlanJSON(agentRunDurablePayload{
		Message: "runtime plan update won first",
		ModelID: 123,
	})
	store := &cancelTransitionRaceStore{agentRunTestStore: base}
	store.beforeCAS = func(_ context.Context, base *agentRunTestStore, attempt int, runID int64, _, _ string) {
		if attempt != 1 {
			return
		}
		base.mu.Lock()
		current := base.runs[runID]
		current.PlanJSON = updatedPlan
		current.UpdatedAt = time.Now()
		base.runs[runID] = current
		base.mu.Unlock()
	}
	service := newNativeAIService(store, nil, nil, nil, nil)

	response, err := service.CancelAgentRun(context.Background(), &pb.CancelAgentRunRequest{
		HrId:  77,
		RunId: created.ID,
	})
	if err != nil || response.GetCode() != 0 || response.GetRun().GetStatus() != agentRunStatusCanceled {
		t.Fatalf("cancel response=%#v err=%v", response, err)
	}
	final, found := base.runSnapshot(created.ID)
	if !found || final.Status != agentRunStatusCanceled || final.PlanJSON != updatedPlan {
		t.Fatalf("final run=%#v found=%v", final, found)
	}
	if attempts := store.transitionAttempts(); attempts != 3 {
		t.Fatalf("transition attempts=%d, want first miss plus cancel-request and terminal CAS", attempts)
	}
}

func TestCancelAgentRunRetriesAfterConcurrentQueuedClaim(t *testing.T) {
	base := newAgentRunTestStore()
	run := fallbackAgentRun(77, 101, "cancel-claim-race", agentRunDurablePayload{
		Message: "cancel during claim",
		ModelID: 123,
	})
	run.Status = agentRunStatusQueued
	created, _, err := base.CreateAgentRun(context.Background(), run)
	if err != nil {
		t.Fatalf("CreateAgentRun seed: %v", err)
	}
	store := &cancelTransitionRaceStore{agentRunTestStore: base}
	store.beforeCAS = func(_ context.Context, base *agentRunTestStore, attempt int, runID int64, _, _ string) {
		if attempt != 1 {
			return
		}
		base.mu.Lock()
		current := base.runs[runID]
		current.Status = agentRunStatusRunning
		current.PlanJSON = agentRunPlanJSON(agentRunDurablePayload{
			Message: "claimed runtime plan",
			ModelID: 123,
		})
		current.UpdatedAt = time.Now()
		base.runs[runID] = current
		base.mu.Unlock()
	}
	service := newNativeAIService(store, nil, nil, nil, nil)

	response, err := service.CancelAgentRun(context.Background(), &pb.CancelAgentRunRequest{
		HrId:  77,
		RunId: created.ID,
	})
	if err != nil || response.GetCode() != 0 || response.GetRun().GetStatus() != agentRunStatusCanceled {
		t.Fatalf("cancel response=%#v err=%v", response, err)
	}
	final, found := base.runSnapshot(created.ID)
	if !found || final.Status != agentRunStatusCanceled {
		t.Fatalf("final run=%#v found=%v", final, found)
	}
	if attempts := store.transitionAttempts(); attempts != 3 {
		t.Fatalf("transition attempts=%d, want queued miss plus cancel-request and terminal CAS", attempts)
	}
}

func TestCancelAgentRunRetryExhaustionAndContextCancellation(t *testing.T) {
	t.Run("bounded exhaustion is retryable", func(t *testing.T) {
		base := newAgentRunTestStore()
		run := fallbackAgentRun(77, 101, "cancel-exhaustion", agentRunDurablePayload{
			Message: "continuously changing plan",
			ModelID: 123,
		})
		run.Status = agentRunStatusRunning
		created, _, err := base.CreateAgentRun(context.Background(), run)
		if err != nil {
			t.Fatalf("CreateAgentRun seed: %v", err)
		}
		store := &cancelTransitionRaceStore{agentRunTestStore: base}
		store.beforeCAS = func(_ context.Context, base *agentRunTestStore, attempt int, runID int64, _, _ string) {
			base.mu.Lock()
			current := base.runs[runID]
			current.PlanJSON = agentRunPlanJSON(agentRunDurablePayload{
				Message: fmt.Sprintf("runtime update %d", attempt),
				ModelID: 123,
			})
			current.UpdatedAt = time.Now()
			base.runs[runID] = current
			base.mu.Unlock()
		}
		service := newNativeAIService(store, nil, nil, nil, nil)

		response, err := service.CancelAgentRun(context.Background(), &pb.CancelAgentRunRequest{
			HrId:  77,
			RunId: created.ID,
		})
		if response != nil || status.Code(err) != codes.Aborted {
			t.Fatalf("cancel response=%#v err=%v code=%v, want retryable Aborted", response, err, status.Code(err))
		}
		if attempts := store.transitionAttempts(); attempts != agentRunCancelTransitionMaxAttempts {
			t.Fatalf("transition attempts=%d, want bounded %d", attempts, agentRunCancelTransitionMaxAttempts)
		}
		final, found := base.runSnapshot(created.ID)
		if !found || final.Status != agentRunStatusRunning {
			t.Fatalf("final run=%#v found=%v, cancellation must not use unconditional update", final, found)
		}
	})

	t.Run("context cancellation stops retries", func(t *testing.T) {
		base := newAgentRunTestStore()
		run := fallbackAgentRun(77, 101, "cancel-context", agentRunDurablePayload{
			Message: "cancel context",
			ModelID: 123,
		})
		run.Status = agentRunStatusRunning
		created, _, err := base.CreateAgentRun(context.Background(), run)
		if err != nil {
			t.Fatalf("CreateAgentRun seed: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		store := &cancelTransitionRaceStore{agentRunTestStore: base}
		store.beforeCAS = func(_ context.Context, base *agentRunTestStore, _ int, runID int64, _, _ string) {
			base.mu.Lock()
			current := base.runs[runID]
			current.PlanJSON = agentRunPlanJSON(agentRunDurablePayload{
				Message: "context canceled update",
				ModelID: 123,
			})
			current.UpdatedAt = time.Now()
			base.runs[runID] = current
			base.mu.Unlock()
			cancel()
		}
		service := newNativeAIService(store, nil, nil, nil, nil)

		response, err := service.CancelAgentRun(ctx, &pb.CancelAgentRunRequest{
			HrId:  77,
			RunId: created.ID,
		})
		if response != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel response=%#v err=%v, want context canceled", response, err)
		}
		if attempts := store.transitionAttempts(); attempts != 1 {
			t.Fatalf("transition attempts=%d, want immediate stop after first miss", attempts)
		}
	})
}

func TestCancelAgentRunLockWaitHonorsContext(t *testing.T) {
	tests := []struct {
		name     string
		newCtx   func() (context.Context, context.CancelFunc)
		cancel   func(context.CancelFunc)
		wantCode codes.Code
	}{
		{
			name: "canceled",
			newCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			cancel:   func(cancel context.CancelFunc) { cancel() },
			wantCode: codes.Canceled,
		},
		{
			name: "deadline exceeded",
			newCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 20*time.Millisecond)
			},
			cancel:   func(context.CancelFunc) {},
			wantCode: codes.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := newAgentRunTestStore()
			run := fallbackAgentRun(77, 101, "cancel-lock-wait-"+tt.name, agentRunDurablePayload{
				Message: "cancel while transition serialization is held",
				ModelID: 123,
			})
			run.Status = agentRunStatusRunning
			created, _, err := base.CreateAgentRun(context.Background(), run)
			if err != nil {
				t.Fatalf("CreateAgentRun seed: %v", err)
			}
			store := &cancelTransitionLockWaitStore{
				cancelTransitionRaceStore: &cancelTransitionRaceStore{agentRunTestStore: base},
				readDone:                  make(chan struct{}),
			}
			service := newNativeAIService(store, nil, nil, nil, nil)
			service.runTransitionMu.Lock()
			lockHeld := true
			defer func() {
				if lockHeld {
					service.runTransitionMu.Unlock()
				}
			}()

			ctx, cancel := tt.newCtx()
			defer cancel()
			type result struct {
				response *pb.CancelAgentRunResponse
				err      error
			}
			resultCh := make(chan result, 1)
			go func() {
				response, cancelErr := service.CancelAgentRun(ctx, &pb.CancelAgentRunRequest{
					HrId:  77,
					RunId: created.ID,
				})
				resultCh <- result{response: response, err: cancelErr}
			}()

			select {
			case <-store.readDone:
			case <-time.After(time.Second):
				t.Fatal("CancelAgentRun did not finish its initial read")
			}
			tt.cancel(cancel)

			select {
			case got := <-resultCh:
				if got.response != nil || status.Code(got.err) != tt.wantCode {
					t.Fatalf("cancel response=%#v err=%v code=%v, want %v", got.response, got.err, status.Code(got.err), tt.wantCode)
				}
			case <-time.After(250 * time.Millisecond):
				service.runTransitionMu.Unlock()
				lockHeld = false
				<-resultCh
				t.Fatalf("CancelAgentRun did not return promptly with %v while transition lock was held", tt.wantCode)
			}
			if attempts := store.transitionAttempts(); attempts != 0 {
				t.Fatalf("store transition attempts=%d, want 0 while lock acquisition was canceled", attempts)
			}
			service.runTransitionMu.Unlock()
			lockHeld = false
		})
	}
}

func TestCancelAgentRunAndGovernedCompletionHaveOneTerminalWinnerAcrossReplicas(t *testing.T) {
	service, store, _, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
	run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
	ready := prepareApprovedAgentSkillRunWithoutDispatch(
		t,
		service,
		actor,
		run,
		agentSkillApprovalRequest(run, "cross-replica-terminal-race"),
	)
	claimed, shouldExecute, err := service.beginAgentRunExecution(context.Background(), ready)
	if err != nil || !shouldExecute {
		t.Fatalf("claim run=%#v execute=%v err=%v", claimed, shouldExecute, err)
	}
	cancelReplica := newNativeAIService(store, nil, nil, nil, nil)
	start := make(chan struct{})
	var (
		wg            sync.WaitGroup
		cancelErr     error
		completionErr error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, cancelErr = cancelReplica.CancelAgentRun(actor, &pb.CancelAgentRunRequest{HrId: claimed.OwnerID, RunId: claimed.ID})
	}()
	go func() {
		defer wg.Done()
		<-start
		_, _, completionErr = service.completeAgentRunExecution(
			context.Background(), claimed, "winner", agentRunStatusSucceeded, "", "",
		)
	}()
	close(start)
	wg.Wait()
	if cancelErr != nil {
		t.Fatalf("cancel race error: %v", cancelErr)
	}
	if completionErr != nil && !errors.Is(completionErr, errAgentRunExecutionLeaseLost) {
		t.Fatalf("completion race error: %v", completionErr)
	}
	final, found := store.runSnapshot(claimed.ID)
	if !found || (final.Status != agentRunStatusSucceeded && final.Status != agentRunStatusCanceled) {
		t.Fatalf("terminal run=%#v found=%v", final, found)
	}
	terminalStatus := final.Status
	if _, err := cancelReplica.CancelAgentRun(actor, &pb.CancelAgentRunRequest{HrId: claimed.OwnerID, RunId: claimed.ID}); err != nil {
		t.Fatalf("idempotent cancel: %v", err)
	}
	after, _ := store.runSnapshot(claimed.ID)
	if after.Status != terminalStatus {
		t.Fatalf("terminal status regressed from %s to %s", terminalStatus, after.Status)
	}
}

func TestCancelAgentRunFinalizesOrphanedGovernedCancelRequest(t *testing.T) {
	service, store, _, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
	run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
	ready := prepareApprovedAgentSkillRunWithoutDispatch(
		t,
		service,
		actor,
		run,
		agentSkillApprovalRequest(run, "orphaned-cancel-request"),
	)
	claimed, shouldExecute, err := service.beginAgentRunExecution(context.Background(), ready)
	if err != nil || !shouldExecute {
		t.Fatalf("claim run=%#v execute=%v err=%v", claimed, shouldExecute, err)
	}
	cancelRequested, transitioned, err := service.transitionAgentRunState(
		context.Background(),
		claimed,
		agentRunStatusRunning,
		agentRunStatusCancelRequested,
		claimed.PlanJSON,
		claimed.OptionContextJSON,
		"",
		"",
	)
	if err != nil || !transitioned {
		t.Fatalf("seed cancel request run=%#v transitioned=%v err=%v", cancelRequested, transitioned, err)
	}
	restarted := newNativeAIService(store, nil, nil, nil, nil)
	response, err := restarted.CancelAgentRun(actor, &pb.CancelAgentRunRequest{HrId: 77, RunId: claimed.ID})
	if err != nil || response.GetCode() != 0 || response.GetRun().GetStatus() != agentRunStatusCanceled {
		t.Fatalf("recover cancel response=%#v err=%v", response, err)
	}
}

func TestAgentSkillApprovalMetricRecoversFromDurableAcceptedEvent(t *testing.T) {
	service, store, _, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
	run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
	ready := prepareApprovedAgentSkillRunWithoutDispatch(
		t,
		service,
		actor,
		run,
		agentSkillApprovalRequest(run, "metric-recovery"),
	)
	if store.countEvents(run.ID, "confirmation.accepted") != 1 {
		t.Fatalf("accepted event missing: %v", store.eventTypes(run.ID))
	}

	provider := newBlockingAgentRunProvider("recovered")
	restarted := newNativeAIService(store, provider, nil, nil, nil)
	restarted.skillPackageV2Enabled = true
	restarted.metrics = observability.NewRegistry("ai-agent")
	for i := 0; i < 4; i++ {
		if _, err := restarted.GetAgentRun(actor, &pb.GetAgentRunRequest{HrId: 77, RunId: ready.ID}); err != nil {
			t.Fatalf("recovery poll %d: %v", i, err)
		}
	}
	waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
	output := restarted.metrics.Prometheus()
	if strings.Count(output, `result="passed",reason="approved"`) != 1 {
		t.Fatalf("recovered approval metric was not exactly once:\n%s", output)
	}
	close(provider.release)
}

func TestAgentSkillConfirmationFailsClosedWithoutAtomicStore(t *testing.T) {
	base := newAgentRunTestStore()
	run := AgentRunRow{
		ID:       1,
		OwnerID:  77,
		Status:   agentRunStatusWaitingConfirmation,
		PlanJSON: `{"durable_request":{"pending_agent_skill_confirmation":{"id":"opaque"}}}`,
	}
	service := &nativeAIService{store: &agentSkillNonAtomicStore{AIStore: base}}
	if _, _, err := service.transitionAgentRunConfirmation(
		context.Background(),
		run,
		agentRunStatusQueued,
		run.PlanJSON,
		"{}",
		"",
		"",
	); !errors.Is(err, errInvalidAgentSkillConfirmation) {
		t.Fatalf("non-atomic transition error=%v, want fail closed", err)
	}
}

func TestAgentSkillConfirmationUsesDistinctMessageIdentityForRepeatedText(t *testing.T) {
	service, store, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
	prior := store.seedChatMessage(ChatMessageRow{
		OwnerRole: ownerRoleHR,
		OwnerID:   77,
		SessionID: 101,
		Role:      "user",
		Content:   "resume screening private user message",
		ModelID:   1,
	})
	run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
	pending := agentRunPayloadFromRow(run).PendingAgentSkillConfirmation
	if run.MessageID <= 0 || pending == nil ||
		run.MessageID != pending.UserMessageID ||
		run.MessageID == prior.ID {
		t.Fatalf("message identity prior=%d run=%d pending=%#v", prior.ID, run.MessageID, pending)
	}

	response, err := service.ConfirmAgentRun(actor, agentSkillApprovalRequest(run, "repeated-message-approval"))
	if err != nil || response.GetCode() != 0 {
		t.Fatalf("approve repeated message response=%#v err=%v", response, err)
	}
	waitUntilAgentRunTest(t, time.Second, func() bool { return provider.callCount() == 1 })
	if messages := store.messagesSnapshot(); countAgentSkillTestMessages(messages, "user") != 2 {
		t.Fatalf("repeated text reused or duplicated wrong identity: %#v", messages)
	}
	close(provider.release)
}

type agentSkillNonAtomicStore struct {
	AIStore
}

func agentSkillApprovalRequest(run AgentRunRow, clientRequestID string) *pb.ConfirmAgentRunRequest {
	confirmation := mapAgentRunSnapshot(run).GetConfirmationRequest()
	return &pb.ConfirmAgentRunRequest{
		HrId:                           run.OwnerID,
		RunId:                          run.ID,
		ClientRequestId:                clientRequestID,
		AgentSkillConfirmationId:       confirmation.GetAgentSkillConfirmationId(),
		AgentSkillConfirmationDecision: pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_APPROVE,
		SelectedAgentSkillVersionIds:   append([]int64(nil), confirmation.GetRecommendedAgentSkillVersionIds()...),
	}
}

func prepareApprovedAgentSkillRunWithoutDispatch(
	t *testing.T,
	service *nativeAIService,
	ctx context.Context,
	run AgentRunRow,
	request *pb.ConfirmAgentRunRequest,
) AgentRunRow {
	t.Helper()
	payload := agentRunPayloadFromRow(run)
	if err := validateAgentRunSkillConfirmation(ctx, run, payload, request, time.Now()); err != nil {
		t.Fatalf("validate confirmation: %v", err)
	}
	if _, err := service.revalidateAgentRunSkillBinding(ctx, run, payload); err != nil {
		t.Fatalf("revalidate confirmation: %v", err)
	}
	pending := *payload.PendingAgentSkillConfirmation
	payload = approvedAgentRunSkillPayload(
		payload,
		pending,
		request.GetSelectedAgentSkillVersionIds(),
		request.GetClientRequestId(),
		time.Now(),
	)
	queued, transitioned, err := service.transitionAgentRunConfirmation(
		ctx,
		run,
		agentRunStatusQueued,
		agentRunPlanJSON(payload),
		agentRunSkillResolutionContextJSON(pending, "approved"),
		"",
		"",
	)
	if err != nil || !transitioned {
		t.Fatalf("approval CAS run=%#v transitioned=%v err=%v", queued, transitioned, err)
	}
	ready, err := service.finalizeApprovedAgentSkillHandoff(ctx, queued)
	if err != nil {
		t.Fatalf("finalize approval: %v", err)
	}
	return ready
}

func TestAgentSkillCriticalManualOnlyAndDirectStreamGovernance(t *testing.T) {
	t.Run("critical manual exact version requires durable confirmation", func(t *testing.T) {
		service, _, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelCritical, []int64{101})
		run := createWaitingAgentSkillRun(t, service, actor, 101, []int64{101})
		confirmation := mapAgentRunSnapshot(run).GetConfirmationRequest()
		if confirmation.GetCandidates()[0].GetRisk() != pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_CRITICAL ||
			confirmation.GetCandidates()[0].GetActivationPolicy() != pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_MANUAL_ONLY ||
			provider.callCount() != 0 {
			t.Fatalf("critical confirmation=%#v provider=%d", confirmation, provider.callCount())
		}
	})

	t.Run("direct stream returns structured recoverable state", func(t *testing.T) {
		service, _, provider, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
		stream := &captureChatStream{ctx: actor}
		err := service.ChatStream(&pb.ChatRequest{HrId: 77, SessionId: 101, Message: "resume screening private user message"}, stream)
		if status.Code(err) != codes.FailedPrecondition ||
			!strings.Contains(status.Convert(err).Message(), "AGENT_SKILL_CONFIRMATION_REQUIRES_DURABLE_RUN") {
			t.Fatalf("stream error=%v", err)
		}
		var selection *pb.ChatStreamResponse
		for _, response := range stream.responses {
			if response.GetEventType() == "agent_skill_selection_required" {
				selection = response
			}
		}
		if selection == nil || selection.GetAgentSkillSelection().GetCandidates()[0].GetVersionId() != 101 ||
			!selection.GetDone() || provider.callCount() != 0 {
			t.Fatalf("selection=%#v calls=%d", selection, provider.callCount())
		}
	})
}

func newAgentSkillConfirmationRuntime(
	t *testing.T,
	risk domainagentskill.RiskLevel,
	versionIDs []int64,
) (*nativeAIService, *agentSkillRunTestStore, *blockingAgentRunProvider, context.Context) {
	t.Helper()
	base := newAgentRunTestStore()
	base.seedChatSession(ownerRoleHR, 77, 101, "skill confirmation")
	base.agentConfigs = []*pb.AgentConfigInfo{{
		Id: 10, Name: "hr", AgentType: hrRecruitingAgentType, PromptTemplateId: 20, IsDefault: true, IsEnabled: true,
	}}
	base.promptByID[20] = &pb.PromptTemplateInfo{
		Id: 20, AgentType: hrRecruitingAgentType, PromptRole: hrRuntimePromptRoleSystem, IsActive: true, Content: "system",
	}
	for index, versionID := range versionIDs {
		role := domainagentskill.CompositionRolePrimary
		versionRisk := risk
		core := "resume screening core"
		if index > 0 {
			role = domainagentskill.CompositionRoleSupporting
			versionRisk = domainagentskill.RiskLevelLow
			core = "resume supporting core"
		}
		base.agentSkillVersionDocs = append(
			base.agentSkillVersionDocs,
			runtimeSkillVersionDocument(versionID, int64(index+1), role, versionRisk, core),
		)
	}
	store := &agentSkillRunTestStore{
		agentRunTestStore: base,
		resolution:        runtimeSkillModel(versionIDs, CapabilitySkillRuntimePolicy{}),
	}
	provider := newBlockingAgentRunProvider("approved reply")
	service := newNativeAIService(store, provider, nil, nil, nil)
	service.skillPackageV2Enabled = true
	service.metrics = observability.NewRegistry("ai-agent")
	return service, store, provider, agentSkillTestActor(9, 88)
}

func createWaitingAgentSkillRun(
	t *testing.T,
	service *nativeAIService,
	ctx context.Context,
	sessionID int64,
	versionIDs []int64,
) AgentRunRow {
	t.Helper()
	response, err := service.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{
		HrId:                 77,
		SessionId:            sessionID,
		ClientRequestId:      "skill-confirm-" + time.Now().Format("150405.000000000"),
		Message:              "resume screening private user message",
		ModelId:              1,
		AgentSkillVersionIds: append([]int64(nil), versionIDs...),
	})
	if err != nil || response.GetCode() != 0 || response.GetRun() == nil {
		t.Fatalf("CreateAgentRun response=%#v err=%v", response, err)
	}
	runID := response.GetRun().GetRunId()
	var run AgentRunRow
	waitUntilAgentRunTest(t, time.Second, func() bool {
		var found bool
		run, found = service.store.(*agentSkillRunTestStore).runSnapshot(runID)
		return found && run.Status == agentRunStatusWaitingConfirmation
	})
	return run
}

func agentSkillTestActor(tenantID, userID int64) context.Context {
	return metadata.WithTenantActor(context.Background(), metadata.TenantContext{
		TenantID:    tenantID,
		UserID:      userID,
		AccountType: "staff",
		ClientApp:   "hr",
	})
}

func countAgentSkillTestMessages(messages []ChatMessageRow, role string) int {
	count := 0
	for _, message := range messages {
		if message.Role == role {
			count++
		}
	}
	return count
}

func TestAgentSkillConfirmationPersistedBindingContainsDigestsNotBodies(t *testing.T) {
	service, _, _, actor := newAgentSkillConfirmationRuntime(t, domainagentskill.RiskLevelHigh, []int64{101})
	run := createWaitingAgentSkillRun(t, service, actor, 101, nil)
	var wrapper map[string]any
	if err := json.Unmarshal([]byte(run.PlanJSON), &wrapper); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(run.PlanJSON, "resume screening core") {
		t.Fatalf("plan leaked Skill body: %s", run.PlanJSON)
	}
	payload := agentRunPayloadFromRow(run)
	pending := payload.PendingAgentSkillConfirmation
	if pending == nil || !validSHA256Hex(pending.MessageDigest) ||
		len(pending.CompiledHashes) != 1 || !validSHA256Hex(pending.CompiledHashes[0]) {
		t.Fatalf("pending binding=%#v", pending)
	}
}
