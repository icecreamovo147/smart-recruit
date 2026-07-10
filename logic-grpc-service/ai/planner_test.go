package ai

import (
	"encoding/json"
	"testing"
)

func TestRecruitingPlannerRequiredIntentCoverage(t *testing.T) {
	planner := NewRecruitingPlanner()
	tools := allPlannerTestTools()
	tests := []struct {
		name       string
		message    string
		intent     string
		wantTool   string
		confirm    bool
		outputName string
	}{
		{
			name:       "candidate match evaluation",
			message:    "评估张三和后端岗位的匹配度",
			intent:     IntentCandidateMatchEvaluation,
			wantTool:   "evaluate_candidate_match",
			outputName: "candidate_match_evaluation",
		},
		{
			name:       "candidate comparison",
			message:    "比较这个岗位下候选人，按匹配度排序",
			intent:     IntentCandidateComparison,
			wantTool:   "compare_candidates_for_job",
			outputName: "candidate_comparison",
		},
		{
			name:       "analytics",
			message:    "看一下最近一周投递趋势和招聘漏斗统计",
			intent:     IntentAnalytics,
			wantTool:   "get_application_trend",
			outputName: "analytics",
		},
		{
			name:       "status-change proposal",
			message:    "把李四这个候选人淘汰",
			intent:     IntentStatusChangeProposal,
			wantTool:   "propose_application_status_update",
			confirm:    true,
			outputName: "status_change_proposal",
		},
		{
			name:       "interview prep",
			message:    "帮我准备王五的面试题和面试提纲",
			intent:     IntentInterviewPrep,
			wantTool:   "get_candidate_detail",
			outputName: "interview_prep",
		},
		{
			name:       "offer support",
			message:    "帮我整理赵六的 offer 薪资方案要点",
			intent:     IntentOfferSupport,
			wantTool:   "get_candidate_detail",
			confirm:    true,
			outputName: "offer_support",
		},
		{
			name:       "unknown fallback",
			message:    "帮我想一个办公室午餐主题",
			intent:     IntentUnknown,
			wantTool:   "",
			outputName: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := planner.Plan(RecruitingPlannerInput{Message: tt.message, AvailableTools: tools})
			if plan.Intent != tt.intent {
				t.Fatalf("intent = %q, want %q", plan.Intent, tt.intent)
			}
			if tt.wantTool != "" && !containsString(plan.RequiredTools, tt.wantTool) {
				t.Fatalf("required_tools = %v, want %q", plan.RequiredTools, tt.wantTool)
			}
			if tt.wantTool == "" && len(plan.RequiredTools) != 0 {
				t.Fatalf("unknown intent required_tools = %v, want none", plan.RequiredTools)
			}
			if plan.ConfirmationRequirement.Required != tt.confirm {
				t.Fatalf("confirmation required = %v, want %v", plan.ConfirmationRequirement.Required, tt.confirm)
			}
			if got, _ := plan.OutputSchema["name"].(string); got != tt.outputName {
				t.Fatalf("output schema name = %q, want %q", got, tt.outputName)
			}
			if plan.SelectedSkills == nil || plan.SelectedMemories == nil {
				t.Fatal("selected skill and memory placeholders must be present as empty arrays")
			}
			assertPlanJSON(t, plan)
		})
	}
}

func TestRecruitingPlannerDoesNotClaimUnavailableTools(t *testing.T) {
	planner := NewRecruitingPlanner()
	plan := planner.Plan(RecruitingPlannerInput{
		Message:        "评估张三和后端岗位的匹配度",
		AvailableTools: []string{"search_candidates"},
	})

	if len(plan.RequiredTools) != 1 || plan.RequiredTools[0] != "search_candidates" {
		t.Fatalf("required_tools = %v, want only available search_candidates", plan.RequiredTools)
	}
	if containsString(plan.RequiredTools, "evaluate_candidate_match") {
		t.Fatal("planner claimed unavailable evaluate_candidate_match tool")
	}
}

func TestRecruitingPlannerUnknownWithNoToolsFallsBackSafely(t *testing.T) {
	planner := NewRecruitingPlanner()
	plan := planner.Plan(RecruitingPlannerInput{Message: "随便聊聊"})

	if plan.Intent != IntentUnknown {
		t.Fatalf("intent = %q, want unknown", plan.Intent)
	}
	if len(plan.RequiredTools) != 0 {
		t.Fatalf("required_tools = %v, want empty", plan.RequiredTools)
	}
	if !containsString(plan.RequiredData, "clarifying_question") {
		t.Fatalf("required_data = %v, want clarifying_question", plan.RequiredData)
	}
	if !containsString(plan.RiskChecks, "do_not_claim_unavailable_tools") {
		t.Fatalf("risk_checks = %v, want safe unavailable-tool check", plan.RiskChecks)
	}
}

func assertPlanJSON(t *testing.T, plan RecruitingPlan) {
	t.Helper()
	var decoded RecruitingPlan
	if err := json.Unmarshal([]byte(plan.JSON()), &decoded); err != nil {
		t.Fatalf("plan JSON is invalid: %v", err)
	}
	if decoded.Intent != plan.Intent {
		t.Fatalf("decoded intent = %q, want %q", decoded.Intent, plan.Intent)
	}
}

func allPlannerTestTools() []string {
	return []string{
		"query_total_applications",
		"query_today_applications",
		"get_job_heat_ranking",
		"search_candidates",
		"get_job_detail",
		"search_jobs",
		"get_candidate_detail",
		"propose_application_status_update",
		"list_applications_by_job",
		"get_application_status_summary",
		"get_application_trend",
		"parse_resume_profile",
		"get_resume_profile",
		"evaluate_candidate_match",
		"get_candidate_match_evaluation",
		"compare_candidates_for_job",
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
