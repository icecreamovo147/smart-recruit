package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"smart-recruit-domain-go/ai"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/repository"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func setupDurableAgentRunTest(t *testing.T) (*AIService, *gorm.DB) {
	t.Helper()
	dsn := "file:durable_" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Serialize SQLite access so worker goroutines + event appends do not hit table locks.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.AIChatSession{},
		&model.AIChatHistory{},
		&model.AgentRun{},
		&model.AgentRunEvent{},
		&model.AgentRunStep{},
		&model.EventOutbox{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	runRepo := repository.NewAgentRunRepo(db)
	eventRepo := repository.NewAgentRunEventRepo(db)
	chatRepo := repository.NewChatRepo(db)
	svc := &AIService{
		chats:          chatRepo,
		agentRuns:      runRepo,
		agentRunEvents: eventRepo,
		eventHub:       newAgentRunEventHub(),
		agentRuntime:   "adk",
	}
	// Default noop executor so Create does not invoke real AI.
	svc.durableRunExecutor = func(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error) {
		return &durableRunExecuteResult{Reply: "ok"}, nil
	}
	return svc, db
}

func waitRunStatus(t *testing.T, repo *repository.AgentRunRepo, runID uint64, want string, timeout time.Duration) *model.AgentRun {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		run, err := repo.GetRunByID(context.Background(), runID)
		if err != nil {
			t.Fatalf("GetRunByID: %v", err)
		}
		if run != nil && run.Status == want {
			return run
		}
		time.Sleep(15 * time.Millisecond)
	}
	run, _ := repo.GetRunByID(context.Background(), runID)
	t.Fatalf("timeout waiting status=%s, got %+v", want, run)
	return nil
}

func TestCreateAgentRunIdempotentByClientRequestID(t *testing.T) {
	svc, _ := setupDurableAgentRunTest(t)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 42, OwnerRole: 2, OwnerID: 42, Title: "idem"}
	if err := svc.chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	req := &pb.CreateAgentRunRequest{
		HrId:            42,
		SessionId:       session.ID,
		ClientRequestId: "req-idem-1",
		Message:         "hello durable",
	}

	var starts int32
	svc.durableRunExecutor = func(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error) {
		atomic.AddInt32(&starts, 1)
		time.Sleep(30 * time.Millisecond)
		return &durableRunExecuteResult{Reply: "done"}, nil
	}

	r1, err := svc.CreateAgentRun(ctx, req)
	if err != nil {
		t.Fatalf("CreateAgentRun1: %v", err)
	}
	if r1.Code != errs.OK || r1.GetRun() == nil || r1.IdempotentReplay {
		t.Fatalf("unexpected first create: %+v", r1)
	}
	r2, err := svc.CreateAgentRun(ctx, req)
	if err != nil {
		t.Fatalf("CreateAgentRun2: %v", err)
	}
	if !r2.IdempotentReplay || r2.GetRun().GetRunId() != r1.GetRun().GetRunId() {
		t.Fatalf("expected idempotent replay same run, got r1=%+v r2=%+v", r1, r2)
	}
	// Only one active worker start for the first create (second must not redispatch).
	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&starts); got != 1 {
		t.Fatalf("expected single worker start, got %d", got)
	}
}

func TestCancelAgentRunRequestedThenCanceled(t *testing.T) {
	svc, _ := setupDurableAgentRunTest(t)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 7, OwnerRole: 2, OwnerID: 7, Title: "c"}
	if err := svc.chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	started := make(chan struct{})
	svc.durableRunExecutor = func(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error) {
		close(started)
		// Block until cancel observed on work ctx.
		<-ctx.Done()
		return &durableRunExecuteResult{Canceled: true, Reply: "partial"}, ctx.Err()
	}

	create, err := svc.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{
		HrId: 7, SessionId: session.ID, ClientRequestId: "cancel-1", Message: "long job",
	})
	if err != nil || create.GetRun() == nil {
		t.Fatalf("create: %+v err=%v", create, err)
	}
	runID := create.GetRun().GetRunId()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not start")
	}

	cancelResp, err := svc.CancelAgentRun(ctx, &pb.CancelAgentRunRequest{HrId: 7, RunId: runID})
	if err != nil {
		t.Fatalf("CancelAgentRun: %v", err)
	}
	if cancelResp.GetRun().GetStatus() != AgentRunStatusCancelRequested && cancelResp.GetRun().GetStatus() != AgentRunStatusCanceled {
		t.Fatalf("expected cancel_requested or canceled, got %s", cancelResp.GetRun().GetStatus())
	}

	final := waitRunStatus(t, svc.agentRuns, uint64(runID), AgentRunStatusCanceled, 3*time.Second)
	if final.CancelRequestedAt == nil && final.CanceledAt == nil {
		t.Fatalf("expected cancel timestamps, got %+v", final)
	}
}

func TestWaitingConfirmationPersistsAndConfirmResumesSameRun(t *testing.T) {
	svc, _ := setupDurableAgentRunTest(t)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 9, OwnerRole: 2, OwnerID: 9, Title: "w"}
	if err := svc.chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	var phase int32
	svc.durableRunExecutor = func(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error) {
		n := atomic.AddInt32(&phase, 1)
		if n == 1 {
			conf := `{"required":true,"reason":"pick skills"}`
			_ = svc.agentRuns.UpdateRunSnapshot(ctx, input.Run.ID, repository.AgentRunSnapshotPatch{
				ConfirmationRequestJSON: &conf,
			})
			return &durableRunExecuteResult{
				WaitingConfirmation: true,
				ConfirmationJSON:    conf,
			}, errAgentSkillSelectionRequired
		}
		return &durableRunExecuteResult{Reply: "confirmed answer"}, nil
	}

	create, err := svc.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{
		HrId: 9, SessionId: session.ID, ClientRequestId: "confirm-1", Message: "need skills",
	})
	if err != nil || create.GetRun() == nil {
		t.Fatalf("create: %+v err=%v", create, err)
	}
	runID := create.GetRun().GetRunId()
	waitRunStatus(t, svc.agentRuns, uint64(runID), AgentRunStatusWaitingConfirmation, 3*time.Second)

	confirm, err := svc.ConfirmAgentRun(ctx, &pb.ConfirmAgentRunRequest{
		HrId:                         9,
		RunId:                        runID,
		AgentSkillIds:                []int64{11, 12},
		AgentSkillSelectionConfirmed: true,
	})
	if err != nil {
		t.Fatalf("ConfirmAgentRun: %v", err)
	}
	if confirm.GetRun().GetRunId() != runID {
		t.Fatalf("confirm must resume same run_id")
	}
	final := waitRunStatus(t, svc.agentRuns, uint64(runID), AgentRunStatusSucceeded, 3*time.Second)
	if final.ID != uint64(runID) {
		t.Fatalf("run id changed")
	}
	if got := atomic.LoadInt32(&phase); got < 2 {
		t.Fatalf("expected second worker phase after confirm, got %d", got)
	}
}

func TestToPBAgentRunEventConfirmationRequiredIncludesCandidates(t *testing.T) {
	payload := `{"required":true,"reason":"multiple_auto_candidates","candidates":[{"id":11,"name":"match","display_name":"Match"}],"recommended_agent_skill_ids":[11],"user_message_id":99}`
	ev := toPBAgentRunEvent(&model.AgentRunEvent{
		RunID:       7,
		Seq:         3,
		EventType:   AgentRunEventConfirmationRequired,
		PayloadJSON: payload,
	})
	if ev.GetConfirmation() == nil {
		t.Fatalf("expected structured confirmation on event")
	}
	if len(ev.GetConfirmation().GetCandidates()) != 1 || ev.GetConfirmation().GetCandidates()[0].GetId() != 11 {
		t.Fatalf("unexpected candidates: %+v", ev.GetConfirmation().GetCandidates())
	}
	if ev.GetConfirmation().GetUserMessageId() != 99 {
		t.Fatalf("UserMessageId = %d, want 99", ev.GetConfirmation().GetUserMessageId())
	}
}

func TestDurableEventAppendOrderingAndReplayAfterSeq(t *testing.T) {
	svc, _ := setupDurableAgentRunTest(t)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 3, OwnerRole: 2, OwnerID: 3, Title: "e"}
	if err := svc.chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	// Create run without worker concurrency: no-op dispatch by completing immediately first,
	// then append more events on the terminal run for ordering/replay assertions.
	svc.durableRunExecutor = func(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error) {
		return &durableRunExecuteResult{Reply: "seed"}, nil
	}
	create, err := svc.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{
		HrId: 3, SessionId: session.ID, ClientRequestId: "events-1", Message: "trace",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	runID := uint64(create.GetRun().GetRunId())
	waitRunStatus(t, svc.agentRuns, runID, AgentRunStatusSucceeded, 3*time.Second)

	// Append additional ordered events after worker finished (no concurrent writers).
	for i, et := range []string{AgentRunEventAssistantDelta, AgentRunEventProcessDelta, AgentRunEventHeartbeat} {
		if _, err := svc.appendRunEvent(ctx, runID, et, map[string]any{"i": i}); err != nil {
			t.Fatalf("append %s: %v", et, err)
		}
	}
	all, err := svc.agentRunEvents.ListEventsAfter(ctx, runID, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) < 3 {
		t.Fatalf("expected events, got %d", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i].Seq <= all[i-1].Seq {
			t.Fatalf("seq not monotonic: %+v", all)
		}
	}
	after := all[1].Seq
	rest, err := svc.agentRunEvents.ListEventsAfter(ctx, runID, after)
	if err != nil {
		t.Fatalf("list after: %v", err)
	}
	for _, ev := range rest {
		if ev.Seq <= after {
			t.Fatalf("replay leaked seq<=after: %+v", ev)
		}
	}
}

func TestFinalAssistantMessagePersistedOnce(t *testing.T) {
	svc, _ := setupDurableAgentRunTest(t)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 5, OwnerRole: 2, OwnerID: 5, Title: "f"}
	if err := svc.chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	svc.durableRunExecutor = func(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error) {
		return &durableRunExecuteResult{Reply: "final answer only once"}, nil
	}
	create, err := svc.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{
		HrId: 5, SessionId: session.ID, ClientRequestId: "final-1", Message: "q",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	runID := uint64(create.GetRun().GetRunId())
	run := waitRunStatus(t, svc.agentRuns, runID, AgentRunStatusSucceeded, 3*time.Second)

	// Second complete must not duplicate assistant history.
	if err := svc.completeDurableRun(ctx, run, session, AgentRunStatusSucceeded, "final answer only once", "", "", "", ""); err != nil {
		t.Fatalf("second complete: %v", err)
	}
	count, err := svc.agentRuns.CountAssistantHistoryByAgentRunID(ctx, runID)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 assistant history, got %d", count)
	}
}

// TestFinalAssistantAfterUserMessageID simulates the production durable path where
// the user history row is linked via UpdateRunMessageID before final assistant persist.
// message_id must not suppress writing the assistant row (FR-016).
func TestFinalAssistantAfterUserMessageID(t *testing.T) {
	svc, _ := setupDurableAgentRunTest(t)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 15, OwnerRole: 2, OwnerID: 15, Title: "user-then-final"}
	if err := svc.chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	svc.durableRunExecutor = func(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error) {
		// Mimic production: persist user message and associate message_id mid-run.
		user := &model.AIChatHistory{
			SessionID: session.ID,
			HrID:      15,
			Role:      "user",
			Content:   input.Request.GetMessage(),
		}
		if err := svc.chats.Add(ctx, user); err != nil {
			return nil, err
		}
		if err := svc.agentRuns.UpdateRunMessageID(ctx, input.Run.ID, uint64(user.ID)); err != nil {
			return nil, err
		}
		// Also simulate legacy pollution path: history_id accidentally equals user id.
		_ = svc.agentRuns.UpdateRunHistoryID(ctx, input.Run.ID, uint64(user.ID))
		return &durableRunExecuteResult{
			Reply:              "assistant after user link",
			ResultMetadataJSON: `{"candidate_options":[{"id":1}]}`,
		}, nil
	}

	create, err := svc.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{
		HrId: 15, SessionId: session.ID, ClientRequestId: "final-user-1", Message: "need analysis",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	runID := uint64(create.GetRun().GetRunId())
	run := waitRunStatus(t, svc.agentRuns, runID, AgentRunStatusSucceeded, 3*time.Second)

	if run.MessageID == nil || *run.MessageID == 0 {
		t.Fatalf("expected user message_id set, got %+v", run)
	}
	if run.HistoryID == nil || *run.HistoryID == 0 {
		t.Fatalf("expected assistant history_id set, got %+v", run)
	}
	if *run.HistoryID == *run.MessageID {
		t.Fatalf("assistant history_id must differ from user message_id: message=%d history=%d", *run.MessageID, *run.HistoryID)
	}
	count, err := svc.agentRuns.CountAssistantHistoryByAgentRunID(ctx, runID)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 assistant history after user-linked run, got %d", count)
	}
	// Second complete still idempotent.
	if err := svc.completeDurableRun(ctx, run, session, AgentRunStatusSucceeded, "assistant after user link", "", "", "", ""); err != nil {
		t.Fatalf("second complete: %v", err)
	}
	count2, err := svc.agentRuns.CountAssistantHistoryByAgentRunID(ctx, runID)
	if err != nil {
		t.Fatalf("count2: %v", err)
	}
	if count2 != 1 {
		t.Fatalf("expected still 1 assistant history, got %d", count2)
	}
	if strings.TrimSpace(run.OptionContextJSON) == "" && strings.TrimSpace(run.ResultMetadataJSON) == "" {
		// reload
		run, _ = svc.agentRuns.GetRunByID(ctx, runID)
	}
	run, _ = svc.agentRuns.GetRunByID(ctx, runID)
	if run == nil || strings.TrimSpace(run.OptionContextJSON) == "" {
		t.Fatalf("expected option_context_json snapshot, got %+v", run)
	}
}

func TestDispatchDurableAgentRunWritesOutboxEvent(t *testing.T) {
	svc, db := setupDurableAgentRunTest(t)
	svc.agentRunOutbox = NewOutboxPublisher(repository.NewOutboxRepo(db), nil)

	svc.dispatchDurableAgentRun(123)

	var events []model.EventOutbox
	if err := db.Find(&events).Error; err != nil {
		t.Fatalf("find outbox events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(events))
	}
	if events[0].EventType != agentRunExecuteEventType || events[0].AggregateType != agentRunAggregateType || events[0].AggregateID != 123 {
		t.Fatalf("unexpected outbox event: %+v", events[0])
	}
	var payload agentRunExecuteEnvelope
	if err := json.Unmarshal([]byte(events[0].Payload), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.RunID != 123 || strings.TrimSpace(payload.EventID) == "" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestAgentRunConsumerReturnsRetryableWorkerError(t *testing.T) {
	svc, db := setupDurableAgentRunTest(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	consumer := NewAgentRunConsumer(svc)
	err = consumer.handle(context.Background(), []byte(`{"event_id":"evt-retry","run_id":999}`))
	if err == nil {
		t.Fatal("expected retryable worker error, got nil")
	}
}

func TestProcessClearAppendsSnapshotEvent(t *testing.T) {
	svc, _ := setupDurableAgentRunTest(t)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 31, OwnerRole: 2, OwnerID: 31, Title: "process-clear"}
	if err := svc.chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	run := &model.AgentRun{HrID: 31, SessionID: uint64(session.ID), Status: AgentRunStatusRunning}
	if err := svc.agentRuns.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	var process strings.Builder
	if err := svc.handleDurableStatusEvent(ctx, run.ID, &process, "process_delta", "thinking", "", ""); err != nil {
		t.Fatalf("process_delta: %v", err)
	}
	if err := svc.handleDurableStatusEvent(ctx, run.ID, &process, "process_clear", "", "", ""); err != nil {
		t.Fatalf("process_clear: %v", err)
	}

	events, err := svc.agentRunEvents.ListEventsAfter(ctx, run.ID, 0)
	if err != nil {
		t.Fatalf("ListAfterSeq: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	clearEvent := events[1]
	if clearEvent.EventType != AgentRunEventProcessSnapshot {
		t.Fatalf("expected process snapshot, got %s", clearEvent.EventType)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(clearEvent.PayloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal clear payload: %v", err)
	}
	if payload["snapshot_text"] != "" {
		t.Fatalf("expected empty snapshot_text, got %q", payload["snapshot_text"])
	}
}

func TestDurableStatusEventFiltersLifecycleFromProcessText(t *testing.T) {
	svc, _ := setupDurableAgentRunTest(t)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 32, OwnerRole: 2, OwnerID: 32, Title: "process-lifecycle"}
	if err := svc.chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	run := &model.AgentRun{HrID: 32, SessionID: uint64(session.ID), Status: AgentRunStatusRunning}
	if err := svc.agentRuns.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	var process strings.Builder
	events := []struct {
		eventType string
		message   string
	}{
		{eventType: "agent_run_started", message: "Agent run 已开始"},
		{eventType: "capability_selected", message: "已选择可用能力"},
		{eventType: "planning", message: "正在规划本轮执行"},
		{eventType: "agent_run_done", message: "Agent run 已完成"},
	}
	for _, event := range events {
		if err := svc.handleDurableStatusEvent(ctx, run.ID, &process, event.eventType, event.message, "", ""); err != nil {
			t.Fatalf("handle %s: %v", event.eventType, err)
		}
	}

	if got, want := process.String(), "正在规划本轮执行\n"; got != want {
		t.Fatalf("process = %q, want %q", got, want)
	}
	stored, err := svc.agentRuns.GetRunByID(ctx, run.ID)
	if err != nil || stored == nil {
		t.Fatalf("GetRunByID: %v %#v", err, stored)
	}
	if stored.ProcessText != "正在规划本轮执行\n" {
		t.Fatalf("stored ProcessText = %q", stored.ProcessText)
	}
	persisted, err := svc.agentRunEvents.ListEventsAfter(ctx, run.ID, 0)
	if err != nil {
		t.Fatalf("ListAfterSeq: %v", err)
	}
	if len(persisted) != 1 {
		t.Fatalf("expected only visible planning event, got %d events: %+v", len(persisted), persisted)
	}
	if persisted[0].EventType != AgentRunEventProcessDelta {
		t.Fatalf("expected process.delta, got %s", persisted[0].EventType)
	}
	if strings.Contains(persisted[0].PayloadJSON, "Agent run") || strings.Contains(persisted[0].PayloadJSON, "已选择可用能力") {
		t.Fatalf("lifecycle text leaked into process event: %s", persisted[0].PayloadJSON)
	}
}

func TestBuildResultMetadataSerializesCandidateOptionsAsString(t *testing.T) {
	raw := buildResultMetadataJSON(ai.ToolMetadata{
		CandidateOptions: []ai.ToolCandidateOption{{
			ApplicationID: 42,
			CandidateName: "候选人A",
			JobTitle:      "后端工程师",
		}},
	})
	var payload struct {
		CandidateOptions string `json:"candidate_options"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if strings.TrimSpace(payload.CandidateOptions) == "" {
		t.Fatalf("expected candidate_options JSON string, got %q", payload.CandidateOptions)
	}
	var options []ai.ToolCandidateOption
	if err := json.Unmarshal([]byte(payload.CandidateOptions), &options); err != nil {
		t.Fatalf("candidate_options should be JSON string: %v", err)
	}
	if len(options) != 1 || options[0].ApplicationID != 42 {
		t.Fatalf("unexpected candidate options: %+v", options)
	}
}

// TestGetActiveAgentRunRecoverySnapshot verifies refresh recovery: active run
// lookup returns a non-terminal snapshot with last_event_seq, and clearing the
// active pointer after terminal does not resurrect a stale run.
func TestGetActiveAgentRunRecoverySnapshot(t *testing.T) {
	svc, _ := setupDurableAgentRunTest(t)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 21, OwnerRole: 2, OwnerID: 21, Title: "active-recovery"}
	if err := svc.chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	hold := make(chan struct{})
	svc.durableRunExecutor = func(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error) {
		// Persist partial snapshot mid-run so GetActive can hydrate UI.
		assistant := "partial answer"
		process := "planning…"
		_ = svc.agentRuns.UpdateRunSnapshot(ctx, input.Run.ID, repository.AgentRunSnapshotPatch{
			AssistantText: &assistant,
			ProcessText:   &process,
		})
		if _, err := svc.appendRunEvent(ctx, input.Run.ID, AgentRunEventAssistantDelta, map[string]any{
			"delta": "partial answer",
		}); err != nil {
			return nil, err
		}
		select {
		case <-hold:
			return &durableRunExecuteResult{Reply: "partial answer complete"}, nil
		case <-ctx.Done():
			return &durableRunExecuteResult{Canceled: true}, ctx.Err()
		}
	}

	create, err := svc.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{
		HrId: 21, SessionId: session.ID, ClientRequestId: "active-1", Message: "recover me",
	})
	if err != nil {
		t.Fatalf("CreateAgentRun: %v", err)
	}
	runID := create.GetRun().GetRunId()

	// Wait until events / snapshot are visible.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		active, err := svc.GetActiveAgentRun(ctx, &pb.GetActiveAgentRunRequest{
			HrId: 21, SessionId: session.ID,
		})
		if err != nil {
			t.Fatalf("GetActiveAgentRun: %v", err)
		}
		if active.GetHasActiveRun() && active.GetRun() != nil && active.GetRun().GetLastEventSeq() > 0 {
			if active.GetRun().GetRunId() != runID {
				t.Fatalf("active run id mismatch: got %d want %d", active.GetRun().GetRunId(), runID)
			}
			if active.GetRun().GetAssistantText() == "" && active.GetRun().GetProcessText() == "" {
				// continue until snapshot fields appear
			} else {
				// Replay cursor is last_event_seq for frontend subscribe(after_seq).
				if active.GetRun().GetLastEventSeq() <= 0 {
					t.Fatalf("expected last_event_seq > 0 for reconnect, got %+v", active.GetRun())
				}
				// Disconnect does not cancel — still active.
				if IsTerminalAgentRunStatus(active.GetRun().GetStatus()) {
					t.Fatalf("expected non-terminal active run, got %s", active.GetRun().GetStatus())
				}
				close(hold)
				waitRunStatus(t, svc.agentRuns, uint64(runID), AgentRunStatusSucceeded, 3*time.Second)
				// After success, active pointer must clear for refresh.
				after, err := svc.GetActiveAgentRun(ctx, &pb.GetActiveAgentRunRequest{
					HrId: 21, SessionId: session.ID,
				})
				if err != nil {
					t.Fatalf("GetActiveAgentRun after complete: %v", err)
				}
				if after.GetHasActiveRun() {
					t.Fatalf("expected no active run after success, got %+v", after.GetRun())
				}
				return
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
	close(hold)
	t.Fatalf("timeout waiting for active run recovery snapshot")
}

func TestSubscribeDisconnectDoesNotCancelRun(t *testing.T) {
	svc, _ := setupDurableAgentRunTest(t)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 11, OwnerRole: 2, OwnerID: 11, Title: "s"}
	if err := svc.chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	hold := make(chan struct{})
	svc.durableRunExecutor = func(ctx context.Context, input durableRunExecuteInput) (*durableRunExecuteResult, error) {
		select {
		case <-hold:
			return &durableRunExecuteResult{Reply: "later"}, nil
		case <-ctx.Done():
			return &durableRunExecuteResult{Canceled: true}, ctx.Err()
		}
	}

	create, err := svc.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{
		HrId: 11, SessionId: session.ID, ClientRequestId: "sub-1", Message: "streaming",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	runID := create.GetRun().GetRunId()

	// Wait until worker is running or planning/running active.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		run, _ := svc.agentRuns.GetRunByID(ctx, uint64(runID))
		if run != nil && (run.Status == AgentRunStatusRunning || run.Status == AgentRunStatusPlanning) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	subCtx, cancel := context.WithCancel(context.Background())
	stream := &fakeAgentRunEventStream{ctx: subCtx}
	done := make(chan error, 1)
	go func() {
		done <- svc.subscribeAgentRunEvents(&pb.SubscribeAgentRunEventsRequest{
			HrId: 11, RunId: runID, AfterSeq: 0,
		}, stream)
	}()
	time.Sleep(40 * time.Millisecond)
	cancel() // disconnect subscriber only
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("subscribe ended with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("subscribe did not end after ctx cancel")
	}

	run, err := svc.agentRuns.GetRunByID(context.Background(), uint64(runID))
	if err != nil || run == nil {
		t.Fatalf("load run: %v", err)
	}
	if run.Status == AgentRunStatusCanceled {
		t.Fatalf("subscribe disconnect must not cancel run, status=%s", run.Status)
	}
	if !IsActiveAgentRunStatus(run.Status) && run.Status != AgentRunStatusSucceeded {
		// still active (held) expected
		t.Fatalf("expected active run after subscribe cancel, got %s", run.Status)
	}

	close(hold)
	waitRunStatus(t, svc.agentRuns, uint64(runID), AgentRunStatusSucceeded, 3*time.Second)
}

// fakeAgentRunEventStream implements agentRunEventStream for unit tests.
type fakeAgentRunEventStream struct {
	ctx  context.Context
	mu   sync.Mutex
	sent []*pb.AgentRunEvent
}

func (f *fakeAgentRunEventStream) Send(ev *pb.AgentRunEvent) error {
	if f.ctx.Err() != nil {
		return f.ctx.Err()
	}
	f.mu.Lock()
	f.sent = append(f.sent, ev)
	f.mu.Unlock()
	return nil
}

func (f *fakeAgentRunEventStream) Context() context.Context { return f.ctx }
