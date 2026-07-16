package ai

import (
	"encoding/json"
	"strings"
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
			name:       "application analysis canonical message",
			message:    "请分析该候选人当前投递简历与岗位的匹配度，并基于真实候选人、岗位和匹配评估数据给出结论。",
			intent:     IntentCandidateMatchEvaluation,
			wantTool:   "evaluate_candidate_match",
			outputName: "candidate_match_evaluation",
		},
		{
			name:       "candidate comparison",
			message:    "比较这个岗位下候选人，按匹配度排序",
			intent:     IntentCandidateComparison,
			wantTool:   "list_applications_by_job",
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
			wantTool:   "get_candidate_detail",
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
			name:       "general chat fallback",
			message:    "帮我想一个办公室午餐主题",
			intent:     IntentGeneralChat,
			wantTool:   "",
			outputName: "general_chat",
		},
		{
			name:       "greeting hello",
			message:    "hello",
			intent:     IntentGreeting,
			wantTool:   "",
			outputName: "greeting",
		},
		{
			name:       "greeting hi",
			message:    "hi",
			intent:     IntentGreeting,
			wantTool:   "",
			outputName: "greeting",
		},
		{
			name:       "greeting chinese",
			message:    "你好",
			intent:     IntentGreeting,
			wantTool:   "",
			outputName: "greeting",
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
			if tt.intent != IntentUnknown && tt.intent != IntentGreeting && len(plan.DisplaySteps) == 0 {
				t.Fatal("display steps must be present for executable intents")
			}
			if tt.intent == IntentGreeting && !plan.DisallowsModelTools() {
				t.Fatalf("intent %q must disallow model tools", tt.intent)
			}
			if tt.intent == IntentGreeting && !strings.Contains(plan.InstructionBlock(), "MUST NOT call any tools") {
				t.Fatalf("instruction block missing hard no-tool rule for %q", tt.intent)
			}
			if tt.intent == IntentGeneralChat && !plan.DisallowsModelTools() {
				t.Fatalf("intent %q must not expose recruiting tools", tt.intent)
			}
			if tt.intent == IntentUnknown && plan.DisallowsModelTools() {
				t.Fatalf("unknown intent should keep model tools available for selected workflows")
			}
			assertPlanJSON(t, plan)
		})
	}
}

func TestRecruitingPlannerSeparatesGeneralDomainBeforeKeywordHints(t *testing.T) {
	planner := NewRecruitingPlanner()
	tests := []struct {
		name    string
		message string
	}{
		{"quick sort code chinese", "帮我写一段快速排序代码"},
		{"quick sort code english", "write quicksort in Go"},
		{"translation", "帮我翻译这句话：good morning"},
		{"general explanation", "解释一下 TCP 三次握手是什么"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := planner.Plan(RecruitingPlannerInput{Message: tt.message, AvailableTools: allPlannerTestTools()})
			if plan.Intent != IntentGeneralChat || plan.Domain != IntentDomainGeneral {
				t.Fatalf("plan intent/domain = %q/%q, want general_chat/general_chat", plan.Intent, plan.Domain)
			}
			if plan.RequiresRecruitingData || len(plan.RequiredTools) != 0 || len(plan.RequiredToolGroups) != 0 {
				t.Fatalf("plan requires recruiting data/tools: %#v", plan)
			}
			if !plan.DisallowsModelTools() {
				t.Fatalf("general turn must not expose recruiting tools: %#v", plan)
			}
			if len(plan.PossibleIntents) == 0 || !containsString(plan.PossibleIntents, IntentGeneralChat) {
				t.Fatalf("possible intents = %v, want general_chat", plan.PossibleIntents)
			}
			if strings.Contains(plan.InstructionBlock(), "Reply with a short greeting") {
				t.Fatalf("general instruction should not use greeting-only guidance: %s", plan.InstructionBlock())
			}
		})
	}
}

func TestRecruitingPlannerGreetingDoesNotRequireTools(t *testing.T) {
	planner := NewRecruitingPlanner()
	for _, msg := range []string{"hello", "hi", "Hello!", "你好", "嗨", "thanks"} {
		plan := planner.Plan(RecruitingPlannerInput{Message: msg, AvailableTools: allPlannerTestTools()})
		if plan.Intent != IntentGreeting {
			t.Fatalf("message %q intent = %q, want greeting", msg, plan.Intent)
		}
		if len(plan.RequiredTools) != 0 || len(plan.RequiredToolGroups) != 0 {
			t.Fatalf("message %q required tools/groups = %#v / %#v, want empty", msg, plan.RequiredTools, plan.RequiredToolGroups)
		}
		if !plan.DisallowsModelTools() {
			t.Fatalf("message %q should disallow model tools", msg)
		}
	}
	// Data questions must not be classified as greeting.
	plan := planner.Plan(RecruitingPlannerInput{Message: "现在有哪些岗位", AvailableTools: allPlannerTestTools()})
	if plan.Intent != IntentJobListing || plan.DisallowsModelTools() {
		t.Fatalf("job list query plan = %#v, want job_listing with tools allowed", plan)
	}
}

func TestRecruitingPlannerDisplayStepsCarryBusinessPurpose(t *testing.T) {
	plan := NewRecruitingPlanner().Plan(RecruitingPlannerInput{
		Message:        "请分析该候选人当前投递简历与岗位的匹配度",
		ApplicationID:  99,
		AvailableTools: allPlannerTestTools(),
	})
	if len(plan.DisplaySteps) < 3 {
		t.Fatalf("display steps = %#v, want evidence steps plus compose step", plan.DisplaySteps)
	}
	if plan.DisplaySteps[0].Key != "candidate_identity" || !strings.Contains(plan.DisplaySteps[0].Purpose, "候选人上下文") {
		t.Fatalf("first display step = %#v, want candidate context purpose", plan.DisplaySteps[0])
	}
	if plan.DisplaySteps[1].Key != "match_evidence" || !strings.Contains(plan.DisplaySteps[1].Purpose, "匹配评估") {
		t.Fatalf("second display step = %#v, want match evaluation purpose", plan.DisplaySteps[1])
	}
	if got := plan.DisplaySteps[len(plan.DisplaySteps)-1]; got.Key != "compose_answer" || !strings.Contains(got.Purpose, "形成结论") {
		t.Fatalf("compose display step = %#v, want conclusion purpose", got)
	}
}

func TestRecruitingPlannerDoesNotClaimUnavailableTools(t *testing.T) {
	planner := NewRecruitingPlanner()
	plan := planner.Plan(RecruitingPlannerInput{
		Message:        "评估张三和后端岗位的匹配度",
		AvailableTools: []string{"search_candidates"},
	})

	if len(plan.RequiredTools) != 0 {
		t.Fatalf("required_tools = %v, want no unrelated available tool", plan.RequiredTools)
	}
	if containsString(plan.RequiredTools, "evaluate_candidate_match") {
		t.Fatal("planner claimed unavailable evaluate_candidate_match tool")
	}
}

func TestRecruitingPlannerDeclaresEveryRequestedEvidenceGroup(t *testing.T) {
	planner := NewRecruitingPlanner()
	tools := allPlannerTestTools()
	tests := []struct {
		name       string
		message    string
		appID      int64
		wantIntent string
		wantGroups []string
	}{
		{"job inventory", "现在有哪些岗位", 0, IntentJobListing, []string{"job_inventory"}},
		{"application inventory", "列出所有投递", 0, IntentApplicationListing, []string{"application_inventory"}},
		{"candidate search", "搜索候选人张三", 0, IntentCandidateLookup, []string{"candidate_search"}},
		{"candidate detail", "查询这个候选人详情", 99, IntentCandidateLookup, []string{"candidate_search", "candidate_detail"}},
		{"multi metric analytics", "统计今天投递、趋势和状态分布", 0, IntentAnalytics, []string{"today_applications", "application_trend", "application_status", "total_applications"}},
		{"candidate comparison", "比较这个岗位下候选人", 99, IntentCandidateComparison, []string{"job_identity", "job_candidates"}},
		{"interview prep", "准备候选人的面试题", 99, IntentInterviewPrep, []string{"candidate_identity", "job_identity"}},
		{"offer support", "整理候选人的 offer 方案", 99, IntentOfferSupport, []string{"candidate_identity", "job_identity"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := planner.Plan(RecruitingPlannerInput{Message: tt.message, AvailableTools: tools, ApplicationID: tt.appID})
			if plan.Intent != tt.wantIntent {
				t.Fatalf("intent = %q, want %q", plan.Intent, tt.wantIntent)
			}
			for _, group := range tt.wantGroups {
				if !containsToolGroup(plan.RequiredToolGroups, group) {
					t.Fatalf("groups = %#v, missing %q", plan.RequiredToolGroups, group)
				}
			}
		})
	}
}

func TestRecruitingPlannerRequiresApplicationIDForScopedComplexIntents(t *testing.T) {
	planner := NewRecruitingPlanner()
	for _, message := range []string{
		"评估候选人匹配度",
		"准备候选人的面试题",
		"整理候选人的 offer 方案",
		"把这个候选人淘汰",
	} {
		plan := planner.Plan(RecruitingPlannerInput{Message: message, AvailableTools: allPlannerTestTools()})
		if !containsString(plan.MissingInputs, "application_id") {
			t.Fatalf("message %q missing_inputs = %v, want application_id", message, plan.MissingInputs)
		}
	}
}

func TestRecruitingPlannerCandidateComparisonUsesJobContext(t *testing.T) {
	planner := NewRecruitingPlanner()
	plan := planner.Plan(RecruitingPlannerInput{
		Message:        "比较岗位 88 下的候选人",
		JobID:          88,
		AvailableTools: allPlannerTestTools(),
	})
	if plan.Intent != IntentCandidateComparison {
		t.Fatalf("intent = %q, want %q", plan.Intent, IntentCandidateComparison)
	}
	if len(plan.MissingInputs) != 0 {
		t.Fatalf("missing_inputs = %v, want none for explicit job context", plan.MissingInputs)
	}
	if containsToolGroup(plan.RequiredToolGroups, "candidate_identity") {
		t.Fatalf("groups = %#v, comparison must not require one candidate identity", plan.RequiredToolGroups)
	}
	for _, group := range []string{"job_identity", "job_candidates"} {
		if !containsToolGroup(plan.RequiredToolGroups, group) {
			t.Fatalf("groups = %#v, missing %q", plan.RequiredToolGroups, group)
		}
	}

	missing := planner.Plan(RecruitingPlannerInput{Message: "比较这个岗位下候选人", AvailableTools: allPlannerTestTools()})
	if !containsString(missing.MissingInputs, "job_id") || containsString(missing.MissingInputs, "application_id") {
		t.Fatalf("missing_inputs = %v, want job_id only", missing.MissingInputs)
	}
}

func TestRecruitingPlannerRoutesReadQueryShapesBeforeAnalyticsOrActions(t *testing.T) {
	planner := NewRecruitingPlanner()
	tests := []struct {
		name       string
		input      RecruitingPlannerInput
		wantIntent string
		wantTool   string
	}{
		{"job count", RecruitingPlannerInput{Message: "现在有多少岗位", AvailableTools: allPlannerTestTools()}, IntentJobListing, "get_job_list"},
		{"job detail", RecruitingPlannerInput{Message: "岗位 88 详情", JobID: 88, AvailableTools: allPlannerTestTools()}, IntentJobDetail, "get_job_detail"},
		{"applications by status", RecruitingPlannerInput{Message: "列出已通过的投递", AvailableTools: allPlannerTestTools()}, IntentApplicationListing, "list_applications_by_status"},
		{"applications by job", RecruitingPlannerInput{Message: "这个岗位有哪些投递", ApplicationID: 99, AvailableTools: allPlannerTestTools()}, IntentApplicationListing, "list_applications_by_job"},
		{"candidate detail via search", RecruitingPlannerInput{Message: "查询候选人 Ada 的详情", AvailableTools: allPlannerTestTools()}, IntentCandidateLookup, "get_candidate_detail"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := planner.Plan(tt.input)
			if plan.Intent != tt.wantIntent || !containsString(plan.RequiredTools, tt.wantTool) {
				t.Fatalf("plan = %#v, want intent %s and tool %s", plan, tt.wantIntent, tt.wantTool)
			}
			if len(plan.RequiredToolGroups) == 0 || len(plan.RequiredToolGroups[0].Tools) == 0 || plan.RequiredToolGroups[len(plan.RequiredToolGroups)-1].Tools[0] != tt.wantTool {
				t.Fatalf("tool groups = %#v, want preferred tool %s", plan.RequiredToolGroups, tt.wantTool)
			}
		})
	}
}

func TestRecruitingPlannerJobDetailRequiresResolvableJob(t *testing.T) {
	plan := NewRecruitingPlanner().Plan(RecruitingPlannerInput{Message: "岗位详情", AvailableTools: allPlannerTestTools()})
	if plan.Intent != IntentJobDetail || !containsString(plan.MissingInputs, "job_id") {
		t.Fatalf("plan = %#v, want job_detail with missing job_id", plan)
	}
}

func TestRecruitingPlannerUnknownWithNoToolsFallsBackSafely(t *testing.T) {
	planner := NewRecruitingPlanner()
	plan := planner.Plan(RecruitingPlannerInput{Message: "继续处理一下"})

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
	if plan.DisallowsModelTools() {
		t.Fatalf("unknown turn should keep model tools available for selected workflows: %#v", plan)
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
		"get_application_snapshot",
		"get_job_list",
		"query_total_applications",
		"query_today_applications",
		"get_job_heat_ranking",
		"search_candidates",
		"get_job_detail",
		"search_jobs",
		"get_candidate_detail",
		"propose_application_status_update",
		"list_applications_by_job",
		"list_applications_by_status",
		"list_all_applications",
		"get_application_status_summary",
		"get_application_trend",
		"parse_resume_profile",
		"get_resume_profile",
		"evaluate_candidate_match",
		"get_candidate_match_evaluation",
		"compare_candidates_for_job",
	}
}

func containsToolGroup(groups []RecruitingToolGroup, want string) bool {
	for _, group := range groups {
		if group.Name == want && len(group.Tools) > 0 {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
