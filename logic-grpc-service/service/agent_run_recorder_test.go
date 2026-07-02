package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/ai"
	"logic-grpc-service/model"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

func TestAgentRunRecorderPersistsSuccessfulPlanEvidenceDecisionSkillsAndMemories(t *testing.T) {
	db := setupAgentRunRecorderTestDB(t)
	repo := repository.NewAgentRunRepo(db)
	svc := &AIService{agentRuns: repo, agentRuntime: "adk"}
	ctx := context.Background()
	session := &model.AIChatSession{ID: 10}
	modelID := int64(3)

	rec := svc.startAgentRun(ctx, &pb.ChatRequest{HrId: 20, Message: "评估候选人匹配度", ApplicationId: 30}, session, &modelID, "test-model", nil)
	if rec == nil {
		t.Fatal("expected recorder")
	}
	rec.setSelectedMemoryIDs(ctx, []int64{101, 102})
	rec.setSelectedAgentSkills(ctx, []selectedAgentSkill{{
		ID:                   201,
		Name:                 "match_governance",
		DisplayName:          "Match Governance",
		Manual:               true,
		Priority:             20,
		Reason:               "manual selection",
		RequiredCapabilities: []string{"builtin:evaluate_candidate_match"},
	}})
	rec.recordRecruitingPlan(ctx, ai.RecruitingPlan{
		Intent:        ai.IntentCandidateMatchEvaluation,
		RequiredTools: []string{"evaluate_candidate_match"},
		RequiredData:  []string{"application_id"},
		RiskChecks:    []string{"cite_tool_returned_evidence"},
	})
	rec.recordTool(ctx, "call-1", "evaluate_candidate_match", `{"application_id":30}`, `{"evidence":[{"snippet":"Go"}],"risks":[]}`, "", "", 15*time.Millisecond, nil)
	rec.finish(ctx, agentRunStatusSucceeded, "候选人匹配", "", "")

	assertRunPlanContains(t, db, rec.runID,
		`"intent":"candidate_match_evaluation"`,
		`"selected_agent_skill_ids":[201]`,
		`"selected_agent_skills":[`,
		`"reason":"manual selection"`,
		`"selected_memory_ids":[101,102]`,
		`"status":"succeeded"`,
		`"risk_flags":["cite_tool_returned_evidence"]`,
	)
	assertStepTypes(t, repo, rec.runID, "memory", "prompt", "plan", "evidence")
}

func TestAgentRunRecorderPersistsPartialFailureDecision(t *testing.T) {
	db := setupAgentRunRecorderTestDB(t)
	repo := repository.NewAgentRunRepo(db)
	rec := newTestAgentRunRecorder(t, repo)
	ctx := context.Background()

	rec.recordRecruitingPlan(ctx, ai.RecruitingPlan{
		Intent:        ai.IntentAnalytics,
		RequiredTools: []string{"query_total_applications"},
		RiskChecks:    []string{"call_tools_for_live_metrics"},
	})
	rec.recordTool(ctx, "call-1", "query_total_applications", `{}`, `{"total":12}`, "", "", time.Millisecond, nil)
	rec.recordFallback(ctx, "llm_failed_after_tools", "AI_TIMEOUT", "timeout", 1)
	rec.finish(ctx, agentRunStatusPartial, "fallback answer", "AI_TIMEOUT", "timeout")

	assertRunPlanContains(t, db, rec.runID, `"status":"partial"`, `"partial":true`, `"risk_flag_hit":true`, `"intent":"analytics"`)
	assertStepTypes(t, repo, rec.runID, "plan", "tool", "fallback")
}

func TestAgentRunRecorderPersistsFailedRunDecision(t *testing.T) {
	db := setupAgentRunRecorderTestDB(t)
	repo := repository.NewAgentRunRepo(db)
	rec := newTestAgentRunRecorder(t, repo)
	ctx := context.Background()

	rec.recordRecruitingPlan(ctx, ai.RecruitingPlan{
		Intent:     ai.IntentUnknown,
		RiskChecks: []string{"do_not_claim_unavailable_tools"},
	})
	rec.finish(ctx, agentRunStatusFailed, "", "context_build_failed", "context failed")

	assertRunPlanContains(t, db, rec.runID, `"status":"failed"`, `"failed":true`, `"risk_flag_hit":true`, `"intent":"unknown"`)
	assertStepTypes(t, repo, rec.runID, "plan")
}

func setupAgentRunRecorderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentRun{}, &model.AgentRunStep{}); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}

func newTestAgentRunRecorder(t *testing.T, repo *repository.AgentRunRepo) *agentRunRecorder {
	t.Helper()
	run := &model.AgentRun{
		SessionID: 1,
		HrID:      2,
		AgentType: "hr",
		AgentName: "hr_recruiting_agent",
		ModelName: "test-model",
		Status:    agentRunStatusPlanning,
		PlanJSON:  `{"agent":"hr_recruiting_agent"}`,
		StartedAt: time.Now(),
	}
	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}
	return &agentRunRecorder{repo: repo, runID: run.ID, planState: decodeObjectJSON(run.PlanJSON)}
}

func assertRunPlanContains(t *testing.T, db *gorm.DB, runID uint64, want ...string) {
	t.Helper()
	var run model.AgentRun
	if err := db.First(&run, runID).Error; err != nil {
		t.Fatalf("load run failed: %v", err)
	}
	for _, needle := range want {
		if !strings.Contains(run.PlanJSON, needle) {
			t.Fatalf("plan_json missing %q: %s", needle, run.PlanJSON)
		}
	}
}

func assertStepTypes(t *testing.T, repo *repository.AgentRunRepo, runID uint64, want ...string) {
	t.Helper()
	steps, err := repo.ListStepsByRunIDs(context.Background(), []uint64{runID})
	if err != nil {
		t.Fatalf("ListStepsByRunIDs failed: %v", err)
	}
	seen := make(map[string]bool, len(steps))
	for _, step := range steps {
		seen[step.StepType] = true
	}
	for _, stepType := range want {
		if !seen[stepType] {
			t.Fatalf("missing step type %q in %+v", stepType, steps)
		}
	}
}
