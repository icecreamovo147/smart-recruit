package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"smart-recruit-domain-go/model"
)

func TestAgentRunEventRepoAppendAndReplay(t *testing.T) {
	db := setupTestDB(t)
	runRepo := NewAgentRunRepo(db)
	eventRepo := NewAgentRunEventRepo(db)
	ctx := context.Background()

	run := &model.AgentRun{
		SessionID: 1,
		HrID:      2,
		AgentType: "hr",
		AgentName: "hr_recruiting_agent",
		ModelName: "test-model",
		Status:    "queued",
		StartedAt: time.Now(),
	}
	if err := runRepo.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	e1, err := eventRepo.AppendEvent(ctx, run.ID, "run.created", `{"status":"queued"}`)
	if err != nil {
		t.Fatalf("AppendEvent 1 failed: %v", err)
	}
	if e1.Seq != 1 {
		t.Fatalf("expected seq 1, got %d", e1.Seq)
	}
	e2, err := eventRepo.AppendEvent(ctx, run.ID, "run.status_changed", `{"status":"planning"}`)
	if err != nil {
		t.Fatalf("AppendEvent 2 failed: %v", err)
	}
	if e2.Seq != 2 {
		t.Fatalf("expected seq 2, got %d", e2.Seq)
	}

	loaded, err := runRepo.GetRunByID(ctx, run.ID)
	if err != nil || loaded == nil {
		t.Fatalf("GetRunByID failed: %v", err)
	}
	if loaded.LastEventSeq != 2 {
		t.Fatalf("expected last_event_seq=2, got %d", loaded.LastEventSeq)
	}

	replay, err := eventRepo.ListEventsAfter(ctx, run.ID, 0)
	if err != nil {
		t.Fatalf("ListEventsAfter(0) failed: %v", err)
	}
	if len(replay) != 2 || replay[0].Seq != 1 || replay[1].Seq != 2 {
		t.Fatalf("unexpected full replay: %+v", replay)
	}

	partial, err := eventRepo.ListEventsAfter(ctx, run.ID, 1)
	if err != nil {
		t.Fatalf("ListEventsAfter(1) failed: %v", err)
	}
	if len(partial) != 1 || partial[0].Seq != 2 || partial[0].EventType != "run.status_changed" {
		t.Fatalf("unexpected partial replay: %+v", partial)
	}
}

func TestAgentRunEventRepoDuplicateAndStaleApplicationSafe(t *testing.T) {
	db := setupTestDB(t)
	runRepo := NewAgentRunRepo(db)
	eventRepo := NewAgentRunEventRepo(db)
	ctx := context.Background()

	run := &model.AgentRun{
		SessionID: 3,
		HrID:      4,
		AgentType: "hr",
		AgentName: "hr_recruiting_agent",
		ModelName: "test-model",
		Status:    "running",
		StartedAt: time.Now(),
	}
	if err := runRepo.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	first, dup, err := eventRepo.AppendEventAtSeq(ctx, run.ID, 1, "assistant.delta", `{"text":"a"}`)
	if err != nil || dup {
		t.Fatalf("first AppendEventAtSeq failed: err=%v dup=%v", err, dup)
	}
	if first.Seq != 1 {
		t.Fatalf("expected seq 1, got %d", first.Seq)
	}

	again, dup, err := eventRepo.AppendEventAtSeq(ctx, run.ID, 1, "assistant.delta", `{"text":"a"}`)
	if err != nil {
		t.Fatalf("duplicate AppendEventAtSeq failed: %v", err)
	}
	if !dup {
		t.Fatal("expected duplicate=true for same seq")
	}
	if again.ID != first.ID || again.Seq != 1 {
		t.Fatalf("duplicate should return existing event, got %+v", again)
	}

	if _, _, err := eventRepo.AppendEventAtSeq(ctx, run.ID, 1, "assistant.delta", `{"text":"ignored"}`); err != nil {
		t.Fatalf("second duplicate should remain safe: %v", err)
	}

	// Stale: seq behind last_event_seq without an existing row should fail safely.
	if err := db.WithContext(ctx).Where("run_id = ? AND seq = ?", run.ID, 1).Delete(&model.AgentRunEvent{}).Error; err != nil {
		t.Fatalf("delete event for stale test failed: %v", err)
	}
	_, _, err = eventRepo.AppendEventAtSeq(ctx, run.ID, 1, "assistant.delta", `{"text":"stale"}`)
	if err == nil || !strings.Contains(err.Error(), "stale event seq") {
		t.Fatalf("expected stale event error, got %v", err)
	}

	// Out-of-order ahead of next expected seq should fail.
	_, _, err = eventRepo.AppendEventAtSeq(ctx, run.ID, 3, "assistant.delta", `{"text":"gap"}`)
	if err == nil || !strings.Contains(err.Error(), "out-of-order") {
		t.Fatalf("expected out-of-order error, got %v", err)
	}

	loaded, err := runRepo.GetRunByID(ctx, run.ID)
	if err != nil || loaded == nil {
		t.Fatalf("GetRunByID failed: %v", err)
	}
	if loaded.LastEventSeq != 1 {
		t.Fatalf("last_event_seq should remain 1 after safe rejects, got %d", loaded.LastEventSeq)
	}
}

func TestAgentRunRepoSnapshotAndActiveRunHelpers(t *testing.T) {
	db := setupTestDB(t)
	runRepo := NewAgentRunRepo(db)
	ctx := context.Background()
	clientRequestID := "req-hars-002"

	session := &model.AIChatSession{HrID: 10, Title: "active-run"}
	if err := db.WithContext(ctx).Create(session).Error; err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	run := &model.AgentRun{
		SessionID:       uint64(session.ID),
		HrID:            10,
		ClientRequestID: &clientRequestID,
		AgentType:       "hr",
		AgentName:       "hr_recruiting_agent",
		ModelName:       "test-model",
		Status:          "planning",
		StartedAt:       time.Now(),
	}
	if err := runRepo.CreateRun(ctx, run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	found, err := runRepo.GetRunByClientRequestID(ctx, 10, uint64(session.ID), clientRequestID)
	if err != nil || found == nil || found.ID != run.ID {
		t.Fatalf("GetRunByClientRequestID failed: err=%v found=%+v", err, found)
	}

	assistant := "hello"
	process := "thinking"
	meta := `{"n":1}`
	confirm := `{"skill_id":1}`
	option := `{"option":"analyze"}`
	seq := int64(3)
	if err := runRepo.UpdateRunSnapshot(ctx, run.ID, AgentRunSnapshotPatch{
		AssistantText:           &assistant,
		ProcessText:             &process,
		ResultMetadataJSON:      &meta,
		ConfirmationRequestJSON: &confirm,
		OptionContextJSON:       &option,
		LastEventSeq:            &seq,
	}); err != nil {
		t.Fatalf("UpdateRunSnapshot failed: %v", err)
	}

	activeID := int64(run.ID)
	if err := runRepo.SetSessionActiveRun(ctx, session.ID, &activeID); err != nil {
		t.Fatalf("SetSessionActiveRun failed: %v", err)
	}
	active, err := runRepo.GetActiveRunBySession(ctx, session.ID)
	if err != nil || active == nil {
		t.Fatalf("GetActiveRunBySession failed: %v", err)
	}
	if active.AssistantText != "hello" || active.ProcessText != "thinking" || active.LastEventSeq != 3 {
		t.Fatalf("unexpected active snapshot: %+v", active)
	}
	if active.ResultMetadataJSON != meta || active.ConfirmationRequestJSON != confirm || active.OptionContextJSON != option {
		t.Fatalf("unexpected snapshot json fields: %+v", active)
	}

	now := time.Now()
	if err := runRepo.UpdateRunStatusFields(ctx, run.ID, "cancel_requested", nil, &now, nil); err != nil {
		t.Fatalf("UpdateRunStatusFields failed: %v", err)
	}
	reloaded, err := runRepo.GetRunByID(ctx, run.ID)
	if err != nil || reloaded == nil {
		t.Fatalf("GetRunByID failed: %v", err)
	}
	if reloaded.Status != "cancel_requested" || reloaded.CancelRequestedAt == nil {
		t.Fatalf("expected cancel_requested with timestamp, got %+v", reloaded)
	}

	if err := runRepo.SetSessionActiveRun(ctx, session.ID, nil); err != nil {
		t.Fatalf("clear active run failed: %v", err)
	}
	cleared, err := runRepo.GetActiveRunBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetActiveRunBySession after clear failed: %v", err)
	}
	if cleared != nil {
		t.Fatalf("expected nil active run after clear, got %+v", cleared)
	}
}
