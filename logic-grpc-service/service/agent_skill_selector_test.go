package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"logic-grpc-service/ai"
	"logic-grpc-service/repository"
)

type fakeAgentSkillLister struct {
	rows []repository.AgentSkillRuntimeRecord
}

func (f fakeAgentSkillLister) ListEnabled(context.Context) ([]repository.AgentSkillRuntimeRecord, error) {
	return f.rows, nil
}

func TestSelectAgentSkillsManualPriorityAndAutoMatch(t *testing.T) {
	repo := fakeAgentSkillLister{rows: []repository.AgentSkillRuntimeRecord{
		{ID: 1, Name: "screening", DisplayName: "Screening", Description: "简历筛选", BodyMarkdown: "筛选候选人", IsManualInvocable: 1},
		{ID: 2, Name: "offer", DisplayName: "Offer", Description: "offer negotiation", BodyMarkdown: "薪资 沟通 offer", IsManualInvocable: 1},
		{ID: 3, Name: "interview", DisplayName: "Interview", Description: "面试安排", BodyMarkdown: "安排面试", IsManualInvocable: 1},
		{ID: 4, Name: "analytics", DisplayName: "Analytics", Description: "招聘趋势", BodyMarkdown: "数据分析", IsManualInvocable: 1},
	}}

	selected, err := selectAgentSkills(context.Background(), repo, "hr_recruiting_agent", "帮我安排面试并准备 offer 沟通", []int64{3}, nil)
	if err != nil {
		t.Fatalf("selectAgentSkills returned error: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("len(selected) = %d, want 2", len(selected))
	}
	if selected[0].ID != 3 || !selected[0].Manual {
		t.Fatalf("first selected = %+v, want manual skill 3 first", selected[0])
	}
	if selected[1].ID != 2 || selected[1].Manual {
		t.Fatalf("second selected = %+v, want auto skill 2", selected[1])
	}
}

func TestSelectAgentSkillsSkipsNonManualSkillForManualIDs(t *testing.T) {
	repo := fakeAgentSkillLister{rows: []repository.AgentSkillRuntimeRecord{
		{ID: 7, Name: "internal", DisplayName: "Internal", Description: "内部流程", BodyMarkdown: "内部流程", IsManualInvocable: 0},
	}}

	selected, err := selectAgentSkills(context.Background(), repo, "hr_recruiting_agent", "内部流程", []int64{7}, nil)
	if err != nil {
		t.Fatalf("selectAgentSkills returned error: %v", err)
	}
	if len(selected) != 1 || selected[0].Manual {
		t.Fatalf("selected = %+v, want only auto-selected non-manual skill", selected)
	}
}

func TestSelectAgentSkillsUsesPriorityAndRequiredCapabilities(t *testing.T) {
	requiredMatch := mustJSONList(t, []string{"builtin:evaluate_candidate_match"})
	requiredMissing := mustJSONList(t, []string{"builtin:compare_candidates_for_job"})
	repo := fakeAgentSkillLister{rows: []repository.AgentSkillRuntimeRecord{
		{ID: 1, Name: "match_low", DisplayName: "Match Low", Description: "candidate match", BodyMarkdown: "candidate match", Priority: 1, RequiredCapabilities: requiredMatch, IsManualInvocable: 1},
		{ID: 2, Name: "match_high", DisplayName: "Match High", Description: "candidate match", BodyMarkdown: "candidate match", Priority: 30, RequiredCapabilities: requiredMatch, IsManualInvocable: 1},
		{ID: 3, Name: "match_missing", DisplayName: "Match Missing", Description: "candidate match", BodyMarkdown: "candidate match", Priority: 100, RequiredCapabilities: requiredMissing, IsManualInvocable: 1},
	}}

	selected, err := selectAgentSkills(context.Background(), repo, "hr_recruiting_agent", "candidate match", nil, map[string]bool{"builtin:evaluate_candidate_match": true})
	if err != nil {
		t.Fatalf("selectAgentSkills returned error: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("len(selected) = %d, want 2", len(selected))
	}
	if selected[0].ID != 2 || selected[1].ID != 1 {
		t.Fatalf("selected order = %+v, want high-priority available skill first and unavailable skill skipped", selected)
	}
}

func TestSelectAgentSkillsFiltersByAgentType(t *testing.T) {
	repo := fakeAgentSkillLister{rows: []repository.AgentSkillRuntimeRecord{
		{ID: 1, Name: "candidate", DisplayName: "Candidate", Description: "candidate match", BodyMarkdown: "candidate match", AgentType: "candidate_assistant", IsManualInvocable: 1},
		{ID: 2, Name: "hr", DisplayName: "HR", Description: "candidate match", BodyMarkdown: "candidate match", AgentType: "hr_recruiting_agent", IsManualInvocable: 1},
	}}

	selected, err := selectAgentSkills(context.Background(), repo, "hr_recruiting_agent", "candidate match", []int64{1, 2}, nil)
	if err != nil {
		t.Fatalf("selectAgentSkills returned error: %v", err)
	}
	if len(selected) != 1 || selected[0].ID != 2 {
		t.Fatalf("selected = %+v, want only HR skill", selected)
	}
}

func TestSelectAgentSkillsAllowsStrongSemanticOnlyCandidates(t *testing.T) {
	repo := fakeAgentSkillLister{rows: []repository.AgentSkillRuntimeRecord{
		{ID: 1, Name: "screening", DisplayName: "Screening", Description: "candidate screening", BodyMarkdown: "review candidate", IsManualInvocable: 1},
		{ID: 2, Name: "offer", DisplayName: "Offer", Description: "candidate offer", BodyMarkdown: "prepare candidate offer", IsManualInvocable: 1},
		{ID: 3, Name: "unrelated", DisplayName: "Unrelated", Description: "薪资", BodyMarkdown: "沟通", IsManualInvocable: 1},
	}}

	selected, err := selectAgentSkillsWithSemantic(context.Background(), repo, "hr_recruiting_agent", "candidate", nil, nil, map[int64]float64{2: 0.91, 3: 0.99})
	if err != nil {
		t.Fatalf("selectAgentSkillsWithSemantic returned error: %v", err)
	}
	if len(selected) != 3 {
		t.Fatalf("selected = %+v, want rule matches plus strong semantic-only candidate", selected)
	}
	if selected[0].ID != 2 || !strings.Contains(selected[0].Reason, "semantic") {
		t.Fatalf("selected = %+v, want semantic reranked rule candidate 2 first", selected)
	}
	foundSemanticOnly := false
	for _, skill := range selected {
		if skill.ID == 3 && strings.Contains(skill.Reason, "semantic-only") {
			foundSemanticOnly = true
		}
	}
	if !foundSemanticOnly {
		t.Fatalf("selected = %+v, want strong semantic-only candidate 3 included", selected)
	}
}

func TestSelectAgentSkillsFiltersWeakSemanticOnlyCandidates(t *testing.T) {
	repo := fakeAgentSkillLister{rows: []repository.AgentSkillRuntimeRecord{
		{ID: 1, Name: "screening", DisplayName: "Screening", Description: "candidate screening", BodyMarkdown: "review candidate", IsManualInvocable: 1},
		{ID: 2, Name: "unrelated", DisplayName: "Unrelated", Description: "薪资", BodyMarkdown: "沟通", IsManualInvocable: 1},
	}}

	selected, err := selectAgentSkillsWithSemantic(context.Background(), repo, "hr_recruiting_agent", "candidate", nil, nil, map[int64]float64{2: 0.1})
	if err != nil {
		t.Fatalf("selectAgentSkillsWithSemantic returned error: %v", err)
	}
	for _, skill := range selected {
		if skill.ID == 2 {
			t.Fatalf("weak semantic-only skill should be filtered: %+v", selected)
		}
	}
}

func TestAgentSkillAvailableCapabilitiesRequiresCollectedRuntimeCapabilities(t *testing.T) {
	runtimeCfg := &agentRuntimeConfig{
		MCPCapabilityKeys:   map[string]bool{"1:lookup": true},
		SkillCapabilityKeys: map[string]bool{"skill_tool": true},
	}

	available := agentSkillAvailableCapabilities(runtimeCfg, []string{"search_jobs"})
	if available["mcp:1:lookup"] || available["skill:skill_tool"] {
		t.Fatalf("configured MCP/SKILL capabilities should not be available before runtime collection: %#v", available)
	}
	available = addRuntimeCapabilityRefs(available, "mcp", runtimeCfg.MCPCapabilityKeys)
	if !available["mcp:1:lookup"] {
		t.Fatalf("collected MCP capability was not added: %#v", available)
	}
}

func TestApplyAgentSkillPlannerConstraints(t *testing.T) {
	base := ai.RecruitingPlan{
		Intent:        ai.IntentCandidateMatchEvaluation,
		RequiredTools: []string{"search_candidates"},
		OutputSchema:  map[string]any{"name": "candidate_match_evaluation"},
	}
	skills := []selectedAgentSkill{{
		ID:                   9,
		Name:                 "match_governance",
		DisplayName:          "Match Governance",
		Priority:             20,
		Reason:               "metadata and content match",
		RequiredCapabilities: []string{"builtin:evaluate_candidate_match", "mcp:server:tool"},
		OutputSchema:         `{"type":"object","required":["score"]}`,
	}}

	plan := applyAgentSkillPlannerConstraints(base, skills, []string{"search_candidates", "evaluate_candidate_match"})
	if !containsString(plan.RequiredTools, "search_candidates") || !containsString(plan.RequiredTools, "evaluate_candidate_match") {
		t.Fatalf("required_tools = %#v, want base and skill required builtin tools", plan.RequiredTools)
	}
	if len(plan.SelectedSkills) != 1 || !strings.Contains(plan.SelectedSkills[0], "Match Governance") || !strings.Contains(plan.SelectedSkills[0], "priority:20") {
		t.Fatalf("selected skills reasons not populated: %#v", plan.SelectedSkills)
	}
	schemas, ok := plan.OutputSchema["agent_skill_output_schemas"].([]map[string]any)
	if !ok || len(schemas) != 1 {
		t.Fatalf("agent skill output schemas not attached: %#v", plan.OutputSchema["agent_skill_output_schemas"])
	}
}

func TestRenderAgentSkillInstructionBlock(t *testing.T) {
	block := renderAgentSkillInstructionBlock([]selectedAgentSkill{{
		ID:          9,
		DisplayName: "Follow-up",
		Content:     "Use concise next steps.",
	}})
	if block == "" || !strings.Contains(block, "Agent Skill instructions") || !strings.Contains(block, "Use concise next steps.") {
		t.Fatalf("unexpected block: %q", block)
	}
}

func mustJSONList(t *testing.T, values []string) string {
	t.Helper()
	b, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal JSON list: %v", err)
	}
	return string(b)
}
