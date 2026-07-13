package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"smart-recruit-ai-agent-service/internal/legacydomain/ai"
	"smart-recruit-ai-agent-service/internal/legacydomain/model"
	"smart-recruit-ai-agent-service/internal/legacydomain/repository"
	"smart-recruit-platform-go/logger"
	"smart-recruit-proto/recruitment/pb"
)

const (
	agentRunStatusPlanning  = AgentRunStatusPlanning
	agentRunStatusRunning   = AgentRunStatusRunning
	agentRunStatusSucceeded = AgentRunStatusSucceeded
	agentRunStatusFailed    = AgentRunStatusFailed
	agentRunStatusPartial   = AgentRunStatusPartial
	agentRunStatusCanceled  = AgentRunStatusCanceled

	agentRunFinalWriteTimeout = 5 * time.Second
)

// TransitionDurableRunStatus validates from->to then persists the status update.
// Identical from/to is an idempotent no-op that still returns nil without writing.
func TransitionDurableRunStatus(ctx context.Context, repo *repository.AgentRunRepo, runID uint64, from, to string, completedAt, cancelRequestedAt, canceledAt *time.Time) error {
	if err := ValidateAgentRunTransition(from, to); err != nil {
		return err
	}
	if from == to {
		return nil
	}
	if repo == nil {
		return fmt.Errorf("agent run repo is nil")
	}
	return repo.UpdateRunStatusFields(ctx, runID, to, completedAt, cancelRequestedAt, canceledAt)
}

// ShouldApplyRunEvent reports whether an incoming event sequence should be applied
// given the last applied sequence. Duplicate and stale sequences return false.
func ShouldApplyRunEvent(lastAppliedSeq, incomingSeq int64) bool {
	return !IsStaleOrDuplicateEventSeq(lastAppliedSeq, incomingSeq)
}

type agentRunRecorder struct {
	repo                  *repository.AgentRunRepo
	runID                 uint64
	session               *model.AIChatSession
	hrID                  int64
	selectedAgentSkillIDs []int64
	selectedMemoryIDs     []int64
	planState             map[string]any
	stepMu                sync.Mutex
}

type agentRunPlan struct {
	Agent              string         `json:"agent"`
	AgentType          string         `json:"agent_type"`
	Model              string         `json:"model"`
	ModelID            *int64         `json:"model_id,omitempty"`
	Capabilities       map[string]any `json:"capabilities"`
	UserMessageSummary string         `json:"user_message_summary"`
	ApplicationBound   bool           `json:"application_bound"`
	ApplicationID      int64          `json:"application_id,omitempty"`
	Runtime            string         `json:"runtime"`
}

func (s *AIService) startAgentRun(ctx context.Context, req *pb.ChatRequest, session *model.AIChatSession, modelID *int64, modelName string, runtimeCfg *agentRuntimeConfig) *agentRunRecorder {
	if s.agentRuns == nil || session == nil {
		return nil
	}
	now := time.Now()
	run := &model.AgentRun{
		SessionID: uint64(session.ID),
		HrID:      uint64(req.HrId),
		AgentType: "hr",
		AgentName: "hr_recruiting_agent",
		ModelName: modelName,
		Status:    agentRunStatusPlanning,
		StartedAt: now,
	}
	run.PlanJSON = buildAgentRunPlanJSON(req, modelID, modelName, runtimeCfg, s.agentRuntime)
	if modelID != nil {
		v := uint64(*modelID)
		run.ModelID = &v
	}
	if runtimeCfg != nil && runtimeCfg.AgentID > 0 {
		v := uint64(runtimeCfg.AgentID)
		run.AgentID = &v
		if runtimeCfg.AgentName != "" {
			run.AgentName = runtimeCfg.AgentName
		}
	}
	if err := s.agentRuns.CreateRun(ctx, run); err != nil {
		logger.L().Warn("create agent run failed", zap.Error(err), zap.Int64("session_id", session.ID))
		return nil
	}
	rec := &agentRunRecorder{
		repo:      s.agentRuns,
		runID:     run.ID,
		session:   session,
		hrID:      req.HrId,
		planState: decodeObjectJSON(run.PlanJSON),
	}
	rec.step(ctx, "status", "", "", "", safeJSON(map[string]any{"event": "agent_run_started"}), "{}", "succeeded", 0, "")
	rec.step(ctx, "status", "", "", "", safeJSON(map[string]any{"event": "model_selected", "model_id": modelID, "model_name": modelName}), "{}", "succeeded", 0, "")
	rec.step(ctx, "model", "", "", "", safeJSON(map[string]any{"event": "planning"}), run.PlanJSON, "succeeded", 0, "")
	rec.step(ctx, "status", "", "", "", safeJSON(map[string]any{"event": "capability_selected"}), capabilityOverviewJSON(runtimeCfg), "succeeded", 0, "")
	return rec
}

func (s *AIService) recordFailedAgentRun(ctx context.Context, req *pb.ChatRequest, session *model.AIChatSession, modelID *int64, modelName, errorType string, err error) {
	rec := s.startAgentRun(ctx, req, session, modelID, modelName, s.getAgentRuntimeConfig(ctx, "hr_recruiting_agent"))
	if rec == nil {
		return
	}
	message := ""
	if err != nil {
		message = err.Error()
	}
	rec.finish(ctx, agentRunStatusFailed, "", errorType, message)
}

func (r *agentRunRecorder) markRunning(ctx context.Context) {
	if r == nil {
		return
	}
	now := time.Now()
	_ = r.repo.UpdateRunStatus(ctx, r.runID, agentRunStatusRunning, "", "", "", nil)
	r.step(ctx, "status", "", "", "", safeJSON(map[string]any{"event": "running"}), "{}", "succeeded", 0, "")
	_ = now
}

func (r *agentRunRecorder) recordTool(ctx context.Context, toolCallID, toolName, argsJSON, resultContent, capabilitySource, capabilityKey string, duration time.Duration, execErr error) uint64 {
	if r == nil {
		return 0
	}
	status := agentRunStatusSucceeded
	errMsg := ""
	if execErr != nil {
		status = agentRunStatusFailed
		errMsg = execErr.Error()
	}
	input := normalizeJSONString(argsJSON)
	output := safeJSON(map[string]any{"tool_call_id": toolCallID, "result": resultContent})
	stepType := "tool"
	if isEvidenceProducingToolOutput(toolName, resultContent) {
		stepType = "evidence"
	}
	stepID := r.step(ctx, stepType, capabilitySource, capabilityKey, toolName, input, output, status, duration.Milliseconds(), errMsg)
	return stepID
}

func (r *agentRunRecorder) recordFallback(ctx context.Context, reason, errorType, message string, traceCount int) {
	if r == nil {
		return
	}
	writeCtx, cancel := agentRunFinalWriteContext()
	defer cancel()
	r.step(writeCtx, "fallback", "", "", "", safeJSON(map[string]any{"reason": reason, "error_type": errorType, "tool_traces": traceCount}), safeJSON(map[string]any{"message": message}), "succeeded", 0, "")
}

func (r *agentRunRecorder) recordRecovery(ctx context.Context, reason, message string) {
	if r == nil {
		return
	}
	writeCtx, cancel := agentRunFinalWriteContext()
	defer cancel()
	r.step(writeCtx, "recovery", "", "", "", safeJSON(map[string]any{"reason": reason}), safeJSON(map[string]any{"message": message}), "succeeded", 0, "")
}

func (r *agentRunRecorder) finish(ctx context.Context, status, finalAnswer, errorType, errorMessage string) {
	if r == nil {
		return
	}
	writeCtx, cancel := agentRunFinalWriteContext()
	defer cancel()
	done := time.Now()
	if err := r.repo.UpdateRunStatus(writeCtx, r.runID, status, finalAnswer, errorType, errorMessage, &done); err != nil {
		logger.L().Warn("finish agent run failed", zap.Uint64("run_id", r.runID), zap.Error(err))
	}
	r.updatePlanPatch(writeCtx, map[string]any{
		"decision": map[string]any{
			"status":        status,
			"error_type":    errorType,
			"has_answer":    strings.TrimSpace(finalAnswer) != "",
			"completed_at":  done.Format(time.RFC3339),
			"partial":       status == agentRunStatusPartial,
			"failed":        status == agentRunStatusFailed,
			"canceled":      status == agentRunStatusCanceled,
			"risk_flag_hit": strings.TrimSpace(errorType) != "",
		},
	})
	r.step(writeCtx, "status", "", "", "", safeJSON(map[string]any{"event": "agent_run_done", "status": status, "error_type": errorType}), "{}", statusToStepStatus(status), 0, errorMessage)
}

func agentRunFinalWriteContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), agentRunFinalWriteTimeout)
}

func (r *agentRunRecorder) step(ctx context.Context, stepType, capabilitySource, capabilityKey, toolName, inputJSON, outputJSON, status string, durationMs int64, errorMessage string) uint64 {
	if r == nil {
		return 0
	}
	r.stepMu.Lock()
	defer r.stepMu.Unlock()
	index, err := r.repo.NextStepIndex(ctx, r.runID)
	if err != nil {
		logger.L().Warn("agent run next step index failed", zap.Uint64("run_id", r.runID), zap.Error(err))
		return 0
	}
	now := time.Now()
	completed := now
	step := &model.AgentRunStep{
		RunID:            r.runID,
		StepIndex:        index,
		StepType:         stepType,
		CapabilitySource: capabilitySource,
		CapabilityKey:    capabilityKey,
		ToolName:         toolName,
		InputJSON:        inputJSON,
		OutputJSON:       outputJSON,
		Status:           status,
		DurationMs:       durationMs,
		ErrorMessage:     errorMessage,
		StartedAt:        now.Add(-time.Duration(durationMs) * time.Millisecond),
		CompletedAt:      &completed,
	}
	if err := r.repo.CreateStep(ctx, step); err != nil {
		logger.L().Warn("create agent run step failed", zap.Uint64("run_id", r.runID), zap.String("step_type", stepType), zap.Error(err))
		return 0
	}
	return step.ID
}

func buildAgentRunPlanJSON(req *pb.ChatRequest, modelID *int64, modelName string, runtimeCfg *agentRuntimeConfig, runtime string) string {
	plan := agentRunPlan{
		Agent:              "hr_recruiting_agent",
		AgentType:          "hr",
		Model:              modelName,
		ModelID:            modelID,
		Capabilities:       capabilityOverview(runtimeCfg),
		UserMessageSummary: summarizeUserMessage(req.GetMessage()),
		ApplicationBound:   req.GetApplicationId() > 0,
		ApplicationID:      req.GetApplicationId(),
		Runtime:            runtime,
	}
	return safeJSON(plan)
}

func (r *agentRunRecorder) setSelectedAgentSkillIDs(ids []int64) {
	if r == nil {
		return
	}
	r.stepMu.Lock()
	r.selectedAgentSkillIDs = append([]int64(nil), ids...)
	r.stepMu.Unlock()
	r.updatePlanPatch(context.Background(), map[string]any{
		"selected_agent_skill_ids": append([]int64(nil), ids...),
	})
	r.step(context.Background(), "prompt", "agent_skill", "", "", safeJSON(map[string]any{
		"event":                    "agent_skill_selected",
		"selected_agent_skill_ids": append([]int64(nil), ids...),
	}), "{}", "succeeded", 0, "")
}

func (r *agentRunRecorder) setSelectedAgentSkills(ctx context.Context, skills []selectedAgentSkill) {
	if r == nil {
		return
	}
	ids := selectedAgentSkillIDs(skills)
	details := selectedAgentSkillTraceItems(skills)
	r.stepMu.Lock()
	r.selectedAgentSkillIDs = append([]int64(nil), ids...)
	r.stepMu.Unlock()
	r.updatePlanPatch(ctx, map[string]any{
		"selected_agent_skill_ids": append([]int64(nil), ids...),
		"selected_agent_skills":    details,
	})
	r.step(ctx, "prompt", "agent_skill", "", "", safeJSON(map[string]any{
		"event":                    "agent_skill_selected",
		"selected_agent_skill_ids": append([]int64(nil), ids...),
		"selected_agent_skills":    details,
	}), "{}", "succeeded", 0, "")
}

func (r *agentRunRecorder) setSelectedMemoryIDs(ctx context.Context, ids []int64) {
	if r == nil {
		return
	}
	r.stepMu.Lock()
	r.selectedMemoryIDs = append([]int64(nil), ids...)
	r.stepMu.Unlock()
	r.updatePlanPatch(ctx, map[string]any{
		"selected_memory_ids": append([]int64(nil), ids...),
	})
	r.step(ctx, "memory", "memory", "", "", safeJSON(map[string]any{
		"event":               "memory_selected",
		"selected_memory_ids": append([]int64(nil), ids...),
	}), "{}", "succeeded", 0, "")
}

func (r *agentRunRecorder) recordRecruitingPlan(ctx context.Context, plan ai.RecruitingPlan) {
	if r == nil {
		return
	}
	decision := map[string]any{
		"intent":                     plan.Intent,
		"confirmation_required":      plan.ConfirmationRequirement.Required,
		"confirmation_reason":        plan.ConfirmationRequirement.Reason,
		"required_tool_count":        len(plan.RequiredTools),
		"required_data_count":        len(plan.RequiredData),
		"risk_flag_count":            len(plan.RiskChecks),
		"unavailable_tool_risk":      containsRecorderString(plan.RiskChecks, "no_builtin_recruiting_tools_available"),
		"requires_human_confirm":     plan.ConfirmationRequirement.Required,
		"requires_evidence_citation": containsRecorderString(plan.RiskChecks, "cite_tool_returned_evidence"),
	}
	patch := map[string]any{
		"recruiting_plan": plan,
		"planner_json":    plan.JSON(),
		"risk_flags":      append([]string(nil), plan.RiskChecks...),
		"decision":        decision,
	}
	r.updatePlanPatch(ctx, patch)
	r.step(ctx, "plan", "", "", "", safeJSON(map[string]any{"event": "recruiting_plan_selected"}), safeJSON(patch), "succeeded", 0, "")
}

func (r *agentRunRecorder) updatePlanPatch(ctx context.Context, patch map[string]any) {
	if r == nil || r.repo == nil || len(patch) == 0 {
		return
	}
	r.stepMu.Lock()
	if r.planState == nil {
		r.planState = map[string]any{}
	}
	for k, v := range patch {
		r.planState[k] = v
	}
	planJSON := safeJSON(r.planState)
	r.stepMu.Unlock()
	if err := r.repo.UpdateRunPlan(ctx, r.runID, planJSON); err != nil {
		logger.L().Warn("update agent run plan failed", zap.Uint64("run_id", r.runID), zap.Error(err))
	}
}

func decodeObjectJSON(raw string) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func selectedMemoryIDs(memories []model.AIMemory) []int64 {
	ids := make([]int64, 0, len(memories))
	for _, memory := range memories {
		if memory.ID > 0 {
			ids = append(ids, int64(memory.ID))
		}
	}
	return ids
}

func isEvidenceProducingToolOutput(toolName, resultContent string) bool {
	name := strings.ToLower(strings.TrimSpace(toolName))
	switch name {
	case "get_candidate_detail", "parse_resume_profile", "get_resume_profile", "evaluate_candidate_match",
		"get_candidate_match_evaluation", "compare_candidates_for_job", "get_job_detail":
		return true
	}
	payload := decodeObjectJSON(resultContent)
	if len(payload) == 0 {
		return false
	}
	for _, key := range []string{"evidence", "evaluation", "risks", "strengths_json", "risks_json", "score_breakdown_json", "profile", "candidate"} {
		if _, ok := payload[key]; ok {
			return true
		}
	}
	return false
}

func containsRecorderString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func capabilityOverviewJSON(runtimeCfg *agentRuntimeConfig) string {
	return safeJSON(capabilityOverview(runtimeCfg))
}

func capabilityOverview(runtimeCfg *agentRuntimeConfig) map[string]any {
	if runtimeCfg == nil {
		return map[string]any{"builtin_tools": []string{}, "mcp": []string{}, "skills": []string{}}
	}
	return map[string]any{
		"has_config":     runtimeCfg.HasConfig,
		"builtin_tools":  append([]string(nil), runtimeCfg.ToolNames...),
		"mcp":            sortedBoolMapKeys(runtimeCfg.MCPCapabilityKeys),
		"skills":         sortedBoolMapKeys(runtimeCfg.SkillCapabilityKeys),
		"tool_skills":    sortedBoolMapKeys(runtimeCfg.SkillCapabilityKeys),
		"max_iterations": runtimeCfg.MaxIterations,
	}
}

func summarizeUserMessage(message string) string {
	message = strings.Join(strings.Fields(message), " ")
	runes := []rune(message)
	if len(runes) <= 120 {
		return message
	}
	return string(runes[:120]) + "..."
}

func sortedBoolMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k, ok := range m {
		if ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func safeJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"marshal failed: %s"}`, err.Error())
	}
	return string(data)
}

func normalizeJSONString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return safeJSON(map[string]string{"raw": raw})
	}
	return safeJSON(v)
}

func statusToStepStatus(runStatus string) string {
	switch runStatus {
	case agentRunStatusSucceeded, agentRunStatusPartial:
		return "succeeded"
	case agentRunStatusCanceled:
		return "canceled"
	case agentRunStatusFailed:
		return "failed"
	default:
		return runStatus
	}
}

func resolveCapabilityForTool(runtimeCfg *agentRuntimeConfig, toolName string) (string, string) {
	if runtimeCfg == nil {
		return "builtin", toolName
	}
	for _, name := range runtimeCfg.ToolNames {
		if name == toolName {
			return "builtin", name
		}
	}
	if key, ok := findCapabilityKey(runtimeCfg.MCPCapabilityKeys, toolName); ok {
		return "mcp", key
	}
	if key, ok := findCapabilityKey(runtimeCfg.SkillCapabilityKeys, toolName); ok {
		return "skill", key
	}
	return "builtin", toolName
}

func findCapabilityKey(keys map[string]bool, toolName string) (string, bool) {
	for key, enabled := range keys {
		if !enabled {
			continue
		}
		if key == toolName || strings.HasSuffix(key, ":"+toolName) || strings.Contains(key, toolName) {
			return key, true
		}
	}
	return "", false
}
