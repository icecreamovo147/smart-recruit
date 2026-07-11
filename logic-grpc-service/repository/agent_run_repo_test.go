package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"logic-grpc-service/model"
)

func TestAgentRunRepoCreateWithEmptyJSONFields(t *testing.T) {
	// Regression: MySQL JSON columns reject empty string '' (Error 3140).
	// Create must succeed when optional JSON snapshot fields are zero-value strings.
	db := setupTestDB(t)
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	clientReq := "req-empty-json"
	run := &model.AgentRun{
		SessionID:       9,
		HrID:            9,
		ClientRequestID: &clientReq,
		AgentType:       "hr",
		AgentName:       "hr_recruiting_agent",
		Status:          "queued",
		PlanJSON:        `{"agent":"hr"}`,
		// Intentionally leave ResultMetadataJSON / ConfirmationRequestJSON / OptionContextJSON empty.
		StartedAt: time.Now(),
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun with empty JSON fields failed: %v", err)
	}
	if run.ID == 0 {
		t.Fatal("expected run ID")
	}
	got, err := repo.GetRunByID(ctx, run.ID)
	if err != nil || got == nil {
		t.Fatalf("GetRunByID: %v %#v", err, got)
	}
	// Empty optional JSON should round-trip as empty/null, not break the row.
	if got.Status != "queued" {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestAgentRunRepoUpdateRunSnapshotPersistsModelName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	run := &model.AgentRun{
		SessionID: 1,
		HrID:      2,
		AgentType: "hr",
		AgentName: "hr_recruiting_agent",
		Status:    "running",
		PlanJSON:  `{"agent":"hr_recruiting_agent"}`,
		StartedAt: time.Now(),
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	modelName := "qwen-plus"
	if err := repo.UpdateRunSnapshot(ctx, run.ID, AgentRunSnapshotPatch{ModelName: &modelName}); err != nil {
		t.Fatalf("UpdateRunSnapshot failed: %v", err)
	}
	got, err := repo.GetRunByID(ctx, run.ID)
	if err != nil || got == nil {
		t.Fatalf("GetRunByID: %v %#v", err, got)
	}
	if got.ModelName != modelName {
		t.Fatalf("ModelName = %q, want %q", got.ModelName, modelName)
	}
}

func TestAgentRunRepoCreateStepAndUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	run := &model.AgentRun{
		SessionID: 1,
		HrID:      2,
		AgentType: "hr",
		AgentName: "hr_recruiting_agent",
		ModelName: "test-model",
		Status:    "planning",
		PlanJSON:  `{"agent":"hr_recruiting_agent"}`,
		StartedAt: time.Now(),
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}
	if run.ID == 0 {
		t.Fatal("expected run ID to be populated")
	}

	next, err := repo.NextStepIndex(ctx, run.ID)
	if err != nil {
		t.Fatalf("NextStepIndex failed: %v", err)
	}
	if next != 0 {
		t.Fatalf("expected first step index 0, got %d", next)
	}

	step := &model.AgentRunStep{
		RunID:       run.ID,
		StepIndex:   next,
		StepType:    "tool",
		ToolName:    "search_candidates",
		InputJSON:   `{"keyword":"go"}`,
		OutputJSON:  `{"count":1}`,
		Status:      "succeeded",
		DurationMs:  12,
		StartedAt:   time.Now(),
		CompletedAt: ptrTime(time.Now()),
	}
	if err := repo.CreateStep(ctx, step); err != nil {
		t.Fatalf("CreateStep failed: %v", err)
	}

	done := time.Now()
	if err := repo.UpdateRunStatus(ctx, run.ID, "succeeded", "ok", "", "", &done); err != nil {
		t.Fatalf("UpdateRunStatus failed: %v", err)
	}
	if err := repo.UpdateRunMessageID(ctx, run.ID, 42); err != nil {
		t.Fatalf("UpdateRunMessageID failed: %v", err)
	}

	runs, err := repo.ListRunsBySession(ctx, 2, 1, 10)
	if err != nil {
		t.Fatalf("ListRunsBySession failed: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != "succeeded" || runs[0].FinalAnswer != "ok" {
		t.Fatalf("unexpected runs: %+v", runs)
	}
	if runs[0].MessageID == nil || *runs[0].MessageID != 42 {
		t.Fatalf("expected message_id=42, got %+v", runs[0])
	}
	// history_id is reserved for the assistant message and must not be set by UpdateRunMessageID.
	if runs[0].HistoryID != nil {
		t.Fatalf("expected history_id to remain unset after UpdateRunMessageID, got %+v", runs[0])
	}
	if err := repo.UpdateRunHistoryID(ctx, run.ID, 99); err != nil {
		t.Fatalf("UpdateRunHistoryID failed: %v", err)
	}
	got, err := repo.GetRunByID(ctx, run.ID)
	if err != nil || got == nil || got.HistoryID == nil || *got.HistoryID != 99 {
		t.Fatalf("expected history_id=99, got %+v err=%v", got, err)
	}
	if got.MessageID == nil || *got.MessageID != 42 {
		t.Fatalf("message_id should stay 42 after UpdateRunHistoryID, got %+v", got)
	}

	steps, err := repo.ListStepsByRunIDs(ctx, []uint64{run.ID})
	if err != nil {
		t.Fatalf("ListStepsByRunIDs failed: %v", err)
	}
	if len(steps) != 1 || steps[0].ToolName != "search_candidates" || steps[0].Status != "succeeded" {
		t.Fatalf("unexpected steps: %+v", steps)
	}
}

func TestAgentRunRepoTraceRetrievalIncludesPlanAndEvidenceSteps(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	run := &model.AgentRun{
		SessionID: 11,
		HrID:      22,
		AgentType: "hr",
		AgentName: "hr_recruiting_agent",
		ModelName: "test-model",
		Status:    "succeeded",
		PlanJSON:  `{"recruiting_plan":{"intent":"candidate_match_evaluation"},"selected_agent_skill_ids":[7],"selected_memory_ids":[9],"risk_flags":["cite_tool_returned_evidence"],"decision":{"status":"succeeded"}}`,
		StartedAt: time.Now(),
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}
	steps := []model.AgentRunStep{
		{RunID: run.ID, StepIndex: 0, StepType: "plan", OutputJSON: `{"recruiting_plan":{"intent":"candidate_match_evaluation"}}`, Status: "succeeded", StartedAt: time.Now(), CompletedAt: ptrTime(time.Now())},
		{RunID: run.ID, StepIndex: 1, StepType: "evidence", ToolName: "evaluate_candidate_match", OutputJSON: `{"evidence":[{"snippet":"Go"}]}`, Status: "succeeded", StartedAt: time.Now(), CompletedAt: ptrTime(time.Now())},
	}
	for i := range steps {
		if err := repo.CreateStep(ctx, &steps[i]); err != nil {
			t.Fatalf("CreateStep %d failed: %v", i, err)
		}
	}

	runs, err := repo.ListRunsBySession(ctx, 22, 11, 10)
	if err != nil {
		t.Fatalf("ListRunsBySession failed: %v", err)
	}
	if len(runs) != 1 || !strings.Contains(runs[0].PlanJSON, `"selected_memory_ids":[9]`) {
		t.Fatalf("expected enriched plan json in trace retrieval, got %+v", runs)
	}
	loadedSteps, err := repo.ListStepsByRunIDs(ctx, []uint64{run.ID})
	if err != nil {
		t.Fatalf("ListStepsByRunIDs failed: %v", err)
	}
	if len(loadedSteps) != 2 || loadedSteps[0].StepType != "plan" || loadedSteps[1].StepType != "evidence" {
		t.Fatalf("expected plan then evidence steps, got %+v", loadedSteps)
	}
}

func TestAgentRunRepoFailureAndFallbackStep(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentRunRepo(db)
	ctx := context.Background()

	run := &model.AgentRun{
		SessionID: 9,
		HrID:      8,
		AgentType: "hr",
		AgentName: "hr_recruiting_agent",
		ModelName: "test-model",
		Status:    "running",
		StartedAt: time.Now(),
	}
	if err := repo.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	if err := repo.CreateStep(ctx, &model.AgentRunStep{
		RunID:       run.ID,
		StepIndex:   0,
		StepType:    "fallback",
		Status:      "succeeded",
		OutputJSON:  `{"message":"fallback used"}`,
		StartedAt:   time.Now(),
		CompletedAt: ptrTime(time.Now()),
	}); err != nil {
		t.Fatalf("CreateStep fallback failed: %v", err)
	}
	done := time.Now()
	if err := repo.UpdateRunStatus(ctx, run.ID, "partial", "fallback answer", "AI_TIMEOUT", "llm timeout", &done); err != nil {
		t.Fatalf("UpdateRunStatus failed: %v", err)
	}

	runs, err := repo.ListRunsBySession(ctx, 8, 9, 10)
	if err != nil {
		t.Fatalf("ListRunsBySession failed: %v", err)
	}
	if got := runs[0].Status; got != "partial" {
		t.Fatalf("expected partial status, got %s", got)
	}
	if got := runs[0].ErrorType; got != "AI_TIMEOUT" {
		t.Fatalf("expected error type AI_TIMEOUT, got %s", got)
	}
}

func TestAgentRunResumableSchemaPersistence(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	clientRequestID := "req-hars-001"

	session := &model.AIChatSession{
		HrID:  100,
		Title: "resumable-session",
	}
	if err := db.WithContext(ctx).Create(session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	run := &model.AgentRun{
		SessionID:               uint64(session.ID),
		HrID:                    100,
		ClientRequestID:         &clientRequestID,
		AgentType:               "hr",
		AgentName:               "hr_recruiting_agent",
		ModelName:               "test-model",
		Status:                  "running",
		AssistantText:           "partial answer",
		ProcessText:             "planning...",
		ResultMetadataJSON:      `{"candidates":1}`,
		ConfirmationRequestJSON: `{"skill_id":7}`,
		OptionContextJSON:       `{"option":"analyze"}`,
		LastEventSeq:            2,
		StartedAt:               time.Now(),
	}
	if err := db.WithContext(ctx).Create(run).Error; err != nil {
		t.Fatalf("create resumable run failed: %v", err)
	}

	dup := &model.AgentRun{
		SessionID:       uint64(session.ID),
		HrID:            100,
		ClientRequestID: &clientRequestID,
		AgentType:       "hr",
		AgentName:       "hr_recruiting_agent",
		ModelName:       "test-model",
		Status:          "queued",
		StartedAt:       time.Now(),
	}
	if err := db.WithContext(ctx).Create(dup).Error; err == nil {
		t.Fatal("expected unique idempotency conflict for duplicate client_request_id")
	}

	events := []model.AgentRunEvent{
		{RunID: run.ID, Seq: 1, EventType: "run.created", PayloadJSON: `{"status":"queued"}`},
		{RunID: run.ID, Seq: 2, EventType: "assistant.delta", PayloadJSON: `{"text":"partial"}`},
	}
	for i := range events {
		if err := db.WithContext(ctx).Create(&events[i]).Error; err != nil {
			t.Fatalf("create event %d failed: %v", events[i].Seq, err)
		}
	}
	if err := db.WithContext(ctx).Create(&model.AgentRunEvent{
		RunID:     run.ID,
		Seq:       2,
		EventType: "assistant.delta",
	}).Error; err == nil {
		t.Fatal("expected unique (run_id, seq) conflict")
	}

	activeRunID := int64(run.ID)
	if err := db.WithContext(ctx).Model(session).Update("active_run_id", activeRunID).Error; err != nil {
		t.Fatalf("set active_run_id failed: %v", err)
	}
	history := &model.AIChatHistory{
		SessionID:  session.ID,
		HrID:       100,
		Role:       "assistant",
		Content:    "partial answer",
		AgentRunID: &activeRunID,
	}
	if err := db.WithContext(ctx).Create(history).Error; err != nil {
		t.Fatalf("create history with agent_run_id failed: %v", err)
	}

	var loadedRun model.AgentRun
	if err := db.WithContext(ctx).First(&loadedRun, run.ID).Error; err != nil {
		t.Fatalf("reload run failed: %v", err)
	}
	if loadedRun.ClientRequestID == nil || *loadedRun.ClientRequestID != clientRequestID {
		t.Fatalf("unexpected client_request_id: %+v", loadedRun.ClientRequestID)
	}
	if loadedRun.AssistantText != "partial answer" || loadedRun.LastEventSeq != 2 {
		t.Fatalf("unexpected snapshot fields: %+v", loadedRun)
	}

	var loadedSession model.AIChatSession
	if err := db.WithContext(ctx).First(&loadedSession, session.ID).Error; err != nil {
		t.Fatalf("reload session failed: %v", err)
	}
	if loadedSession.ActiveRunID == nil || *loadedSession.ActiveRunID != activeRunID {
		t.Fatalf("unexpected active_run_id: %+v", loadedSession.ActiveRunID)
	}

	var loadedHistory model.AIChatHistory
	if err := db.WithContext(ctx).First(&loadedHistory, history.ID).Error; err != nil {
		t.Fatalf("reload history failed: %v", err)
	}
	if loadedHistory.AgentRunID == nil || *loadedHistory.AgentRunID != activeRunID {
		t.Fatalf("unexpected history agent_run_id: %+v", loadedHistory.AgentRunID)
	}

	var replay []model.AgentRunEvent
	if err := db.WithContext(ctx).Where("run_id = ? AND seq > ?", run.ID, int64(0)).Order("seq asc").Find(&replay).Error; err != nil {
		t.Fatalf("replay events failed: %v", err)
	}
	if len(replay) != 2 || replay[0].Seq != 1 || replay[1].EventType != "assistant.delta" {
		t.Fatalf("unexpected replay events: %+v", replay)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
