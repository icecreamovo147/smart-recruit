package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"logic-grpc-service/model"
)

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
	if runs[0].MessageID == nil || *runs[0].MessageID != 42 || runs[0].HistoryID == nil || *runs[0].HistoryID != 42 {
		t.Fatalf("expected message/history id to be updated, got %+v", runs[0])
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

func ptrTime(t time.Time) *time.Time {
	return &t
}
