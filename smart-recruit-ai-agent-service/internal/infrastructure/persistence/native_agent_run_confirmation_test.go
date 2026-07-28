package persistence

import (
	"context"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
)

func TestTransitionAgentRunConfirmationConsumesWaitingStateOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&agentRunRecord{}); err != nil {
		t.Fatalf("migrate agent run: %v", err)
	}
	now := time.Now()
	row := agentRunRecord{
		SessionID:       101,
		HRID:            77,
		ClientRequestID: "confirmation-cas",
		Status:          "waiting_confirmation",
		PlanJSON:        stringPointer(`{"durable_request":{"pending":true}}`),
		OptionContext:   stringPointer(`{"confirmation_request":{"required":true}}`),
		StartedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	store := NewNativeStore(db)
	approved, transitioned, err := store.TransitionAgentRunConfirmation(
		context.Background(),
		77,
		row.ID,
		"waiting_confirmation",
		"running",
		`{"durable_request":{"pending":true}}`,
		`{"durable_request":{"approved":true}}`,
		`{"confirmation":{"required":false}}`,
		"",
		"",
	)
	if err != nil || !transitioned || approved.Status != "running" ||
		approved.PlanJSON != `{"durable_request":{"approved":true}}` ||
		approved.OptionContextJSON != `{"confirmation":{"required":false}}` {
		t.Fatalf("first transition run=%#v transitioned=%v err=%v", approved, transitioned, err)
	}
	replayed, transitioned, err := store.TransitionAgentRunConfirmation(
		context.Background(),
		77,
		row.ID,
		"waiting_confirmation",
		"running",
		`{"durable_request":{"pending":true}}`,
		`{"durable_request":{"replayed":true}}`,
		`{"confirmation":{"required":false}}`,
		"",
		"",
	)
	if err != nil || transitioned || replayed.ID != 0 {
		t.Fatalf("replay run=%#v transitioned=%v err=%v", replayed, transitioned, err)
	}
	current, found, err := store.GetAgentRun(context.Background(), 77, row.ID)
	if err != nil || !found || current.Status != "running" ||
		current.PlanJSON != `{"durable_request":{"approved":true}}` {
		t.Fatalf("current run=%#v found=%v err=%v", current, found, err)
	}
}

func TestFinalizeAgentRunSkillApprovalCommitsEventAndReadyMarkerOnce(t *testing.T) {
	db := openAgentRunConfirmationTestDB(t, &agentRunRecord{}, &agentRunEventRecord{})
	now := time.Now()
	pendingPlan := `{"durable_request":{"agent_skill_approval":{"confirmation_id":"confirmation-1","dispatch_state":"accepted_pending_event"}}}`
	readyPlan := `{"durable_request":{"agent_skill_approval":{"confirmation_id":"confirmation-1","dispatch_state":"ready"}}}`
	row := agentRunRecord{
		SessionID:       101,
		HRID:            77,
		ClientRequestID: "confirmation-finalize",
		Status:          "queued",
		PlanJSON:        stringPointer(pendingPlan),
		StartedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	store := NewNativeStore(db)
	finalized, event, inserted, err := store.FinalizeAgentRunSkillApproval(
		context.Background(),
		77,
		row.ID,
		pendingPlan,
		readyPlan,
		`{"confirmation":{"required":false}}`,
		`{"confirmation_type":"agent_skill","agent_skill_confirmation_id":"confirmation-1"}`,
	)
	if err != nil || !inserted || finalized.PlanJSON != readyPlan ||
		event.EventType != "confirmation.accepted" || event.Seq != 1 {
		t.Fatalf("finalize run=%#v event=%#v inserted=%v err=%v", finalized, event, inserted, err)
	}
	replayed, replayEvent, inserted, err := store.FinalizeAgentRunSkillApproval(
		context.Background(),
		77,
		row.ID,
		pendingPlan,
		readyPlan,
		`{"confirmation":{"required":false}}`,
		`{"confirmation_type":"agent_skill","agent_skill_confirmation_id":"confirmation-1"}`,
	)
	if err != nil || inserted || replayed.PlanJSON != readyPlan || replayEvent.Seq != 0 {
		t.Fatalf("replay run=%#v event=%#v inserted=%v err=%v", replayed, replayEvent, inserted, err)
	}
	var eventCount int64
	if err := db.Model(&agentRunEventRecord{}).Where("run_id = ? AND event_type = ?", row.ID, "confirmation.accepted").Count(&eventCount).Error; err != nil || eventCount != 1 {
		t.Fatalf("accepted event count=%d err=%v", eventCount, err)
	}
}

func TestEnsureAgentRunUserMessageBindsExactIdentityOnce(t *testing.T) {
	db := openAgentRunConfirmationTestDB(t, &agentRunRecord{}, &aiChatHistoryRecord{}, &aiChatSessionRecord{})
	now := time.Now()
	session := aiChatSessionRecord{
		ID:        101,
		HRID:      77,
		OwnerRole: chatOwnerRoleHR,
		OwnerID:   77,
		Title:     "confirmation",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	// Same text from an earlier turn must never be reused for this Run.
	prior := aiChatHistoryRecord{
		HRID:      77,
		OwnerRole: chatOwnerRoleHR,
		OwnerID:   77,
		SessionID: 101,
		Role:      "user",
		Content:   "same text",
		CreatedAt: now.Add(-time.Minute),
	}
	if err := db.Create(&prior).Error; err != nil {
		t.Fatalf("seed prior message: %v", err)
	}
	run := agentRunRecord{
		SessionID:       101,
		HRID:            77,
		ClientRequestID: "message-binding",
		Status:          "running",
		StartedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&run).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	store := NewNativeStore(db)
	message := aiagentgrpc.ChatMessageRow{
		OwnerRole: chatOwnerRoleHR,
		OwnerID:   77,
		SessionID: 101,
		Role:      "user",
		Content:   "same text",
		ModelID:   1,
	}
	first, boundRun, created, err := store.EnsureAgentRunUserMessage(context.Background(), 77, run.ID, message)
	if err != nil || !created || first.ID == 0 || first.ID == prior.ID || boundRun.MessageID != first.ID {
		t.Fatalf("first message=%#v run=%#v created=%v err=%v", first, boundRun, created, err)
	}
	second, reboundRun, created, err := store.EnsureAgentRunUserMessage(context.Background(), 77, run.ID, message)
	if err != nil || created || second.ID != first.ID || reboundRun.MessageID != first.ID {
		t.Fatalf("second message=%#v run=%#v created=%v err=%v", second, reboundRun, created, err)
	}
	var messageCount int64
	if err := db.Model(&aiChatHistoryRecord{}).Where("session_id = ? AND role = ?", 101, "user").Count(&messageCount).Error; err != nil || messageCount != 2 {
		t.Fatalf("user message count=%d err=%v", messageCount, err)
	}
}

func TestAgentRunSkillLeaseFencesOldReplicaEventsAndCompletion(t *testing.T) {
	db := openAgentRunConfirmationTestDB(t, &agentRunRecord{}, &agentRunEventRecord{})
	now := time.Now()
	oldPlan := `{"durable_request":{"agent_skill_approval":{"dispatch_state":"claimed","dispatch_lease_id":"old-lease"}}}`
	newPlan := `{"durable_request":{"agent_skill_approval":{"dispatch_state":"claimed","dispatch_lease_id":"new-lease"}}}`
	row := agentRunRecord{
		SessionID:       101,
		HRID:            77,
		ClientRequestID: "lease-fence",
		Status:          "running",
		PlanJSON:        stringPointer(oldPlan),
		StartedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	store := NewNativeStore(db)
	if _, owned, err := store.AppendAgentRunEventForSkillLease(
		context.Background(), 77, row.ID, "old-lease", "process.delta", `{"status":"running"}`,
	); err != nil || !owned {
		t.Fatalf("initial owner append owned=%v err=%v", owned, err)
	}
	if err := db.Model(&agentRunRecord{}).Where("id = ?", row.ID).Update("plan_json", newPlan).Error; err != nil {
		t.Fatal(err)
	}
	if _, owned, err := store.AppendAgentRunEventForSkillLease(
		context.Background(), 77, row.ID, "old-lease", "run.result", `{"status":"running"}`,
	); err != nil || owned {
		t.Fatalf("old lease append owned=%v err=%v", owned, err)
	}
	if _, owned, err := store.CompleteAgentRunForSkillLease(
		context.Background(), 77, row.ID, "old-lease", "stale", "succeeded", "", "",
	); err != nil || owned {
		t.Fatalf("old lease complete owned=%v err=%v", owned, err)
	}
	if _, owned, err := store.AppendAgentRunEventForSkillLease(
		context.Background(), 77, row.ID, "new-lease", "run.result", `{"status":"running"}`,
	); err != nil || !owned {
		t.Fatalf("new lease append owned=%v err=%v", owned, err)
	}
	completed, owned, err := store.CompleteAgentRunForSkillLease(
		context.Background(), 77, row.ID, "new-lease", "fresh", "succeeded", "", "",
	)
	if err != nil || !owned || completed.AssistantText != "fresh" || completed.Status != "succeeded" {
		t.Fatalf("new lease complete=%#v owned=%v err=%v", completed, owned, err)
	}
	var resultCount int64
	if err := db.Model(&agentRunEventRecord{}).
		Where("run_id = ? AND event_type = ?", row.ID, "run.result").
		Count(&resultCount).Error; err != nil || resultCount != 1 {
		t.Fatalf("result events=%d err=%v", resultCount, err)
	}
}

func TestAgentRunSkillLeaseFencesToolStepAndTraceTogether(t *testing.T) {
	db := openAgentRunConfirmationTestDB(t, &agentRunRecord{}, &agentRunStepRecord{}, &aiToolTraceRecord{})
	now := time.Now()
	plan := `{"durable_request":{"agent_skill_approval":{"dispatch_state":"claimed","dispatch_lease_id":"lease-1"}}}`
	row := agentRunRecord{
		SessionID:       101,
		HRID:            77,
		ClientRequestID: "tool-evidence-fence",
		Status:          "running",
		PlanJSON:        stringPointer(plan),
		StartedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	store := NewNativeStore(db)
	step := aiagentgrpc.AgentRunStepRow{RunID: row.ID, ToolName: "get_job_list", Status: "success"}
	trace := aiagentgrpc.ToolTraceRow{AgentRunID: row.ID, SessionID: 101, ToolName: "get_job_list", Status: "success"}
	persistedStep, persistedTrace, owned, err := store.AppendAgentRunToolTraceForSkillLease(
		context.Background(), 77, row.ID, "lease-1", step, trace,
	)
	if err != nil || !owned || persistedStep.ID == 0 || persistedTrace.AgentRunStepID != persistedStep.ID {
		t.Fatalf("initial evidence step=%#v trace=%#v owned=%v err=%v", persistedStep, persistedTrace, owned, err)
	}
	if err := db.Model(&agentRunRecord{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status":    "canceled",
		"plan_json": `{"durable_request":{"agent_skill_approval":{"dispatch_state":"claimed","dispatch_lease_id":"lease-2"}}}`,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, owned, err := store.AppendAgentRunToolTraceForSkillLease(
		context.Background(), 77, row.ID, "lease-1", step, trace,
	); err != nil || owned {
		t.Fatalf("stale evidence owned=%v err=%v", owned, err)
	}
	var stepCount, traceCount int64
	if err := db.Model(&agentRunStepRecord{}).Where("run_id = ?", row.ID).Count(&stepCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&aiToolTraceRecord{}).Where("agent_run_id = ?", row.ID).Count(&traceCount).Error; err != nil {
		t.Fatal(err)
	}
	if stepCount != 1 || traceCount != 1 {
		t.Fatalf("fenced evidence counts steps=%d traces=%d, want 1/1", stepCount, traceCount)
	}
}

func TestTransitionAgentRunStateCASAllowsExactlyOneTerminalWinner(t *testing.T) {
	db := openAgentRunConfirmationTestDB(t, &agentRunRecord{})
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	// SQLite's shared in-memory driver reports table-lock errors for concurrent
	// writers on separate connections. A single connection still exercises the
	// production UPDATE predicate with concurrent callers without driver noise.
	sqlDB.SetMaxOpenConns(1)
	now := time.Now()
	plan := `{"durable_request":{"agent_skill_approval":{"dispatch_state":"claimed","dispatch_lease_id":"lease-race"}}}`
	row := agentRunRecord{
		SessionID:       101,
		HRID:            77,
		ClientRequestID: "terminal-cas-race",
		Status:          "running",
		PlanJSON:        stringPointer(plan),
		StartedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	store := NewNativeStore(db)
	start := make(chan struct{})
	var (
		wg      sync.WaitGroup
		winners int
		mu      sync.Mutex
	)
	for _, status := range []string{"succeeded", "canceled"} {
		status := status
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, transitioned, err := store.TransitionAgentRunConfirmation(
				context.Background(), 77, row.ID, "running", status, plan, plan, "", "", "",
			)
			if err != nil {
				t.Errorf("transition %s: %v", status, err)
				return
			}
			if transitioned {
				mu.Lock()
				winners++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()
	if winners != 1 {
		t.Fatalf("terminal CAS winners=%d, want 1", winners)
	}
	current, found, err := store.GetAgentRun(context.Background(), 77, row.ID)
	if err != nil || !found || (current.Status != "succeeded" && current.Status != "canceled") {
		t.Fatalf("terminal run=%#v found=%v err=%v", current, found, err)
	}
	originalStatus := current.Status
	if _, transitioned, err := store.TransitionAgentRunConfirmation(
		context.Background(), 77, row.ID, "running", "failed", plan, plan, "", "", "",
	); err != nil || transitioned {
		t.Fatalf("terminal regression transitioned=%v err=%v", transitioned, err)
	}
	current, _, _ = store.GetAgentRun(context.Background(), 77, row.ID)
	if current.Status != originalStatus {
		t.Fatalf("terminal status regressed from %s to %s", originalStatus, current.Status)
	}
}

func openAgentRunConfirmationTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate confirmation models: %v", err)
	}
	return db
}

func stringPointer(value string) *string {
	return &value
}
