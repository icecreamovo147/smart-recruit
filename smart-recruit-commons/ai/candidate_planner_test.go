package ai

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCandidateAssistantPlannerRequiredIntentCoverage(t *testing.T) {
	planner := NewCandidateAssistantPlanner()
	tools := candidatePlannerTestTools()
	tests := []struct {
		name       string
		message    string
		intent     string
		wantTools  []string
		disallow   bool
		outputName string
	}{
		{
			name:       "greeting hello",
			message:    "hello",
			intent:     CandidateIntentGreeting,
			disallow:   true,
			outputName: "candidate_greeting",
		},
		{
			name:       "general chat",
			message:    "帮我写一段快速排序代码",
			intent:     CandidateIntentGeneralChat,
			disallow:   true,
			outputName: "candidate_general_chat",
		},
		{
			name:       "application progress",
			message:    "我目前的应聘进度？",
			intent:     CandidateIntentApplicationProgress,
			wantTools:  []string{"list_my_applications"},
			outputName: "candidate_application_progress",
		},
		{
			name:       "application detail",
			message:    "查看第2轮那个投递的详细进度",
			intent:     CandidateIntentApplicationDetail,
			wantTools:  []string{"get_my_application_detail", "list_my_applications"},
			outputName: "candidate_application_detail",
		},
		{
			name:       "resume advice",
			message:    "可以帮我优化一下简历吗",
			intent:     CandidateIntentResumeAdvice,
			wantTools:  []string{"get_my_resume_text"},
			outputName: "candidate_resume_advice",
		},
		{
			name:       "job recommendation",
			message:    "根据我的简历推荐一些岗位",
			intent:     CandidateIntentJobRecommendation,
			wantTools:  []string{"recommend_jobs_by_resume"},
			outputName: "candidate_job_recommendation",
		},
		{
			name:       "job listing",
			message:    "现在有哪些在招岗位",
			intent:     CandidateIntentJobListing,
			wantTools:  []string{"list_jobs_for_recommendation"},
			outputName: "candidate_job_listing",
		},
		{
			name:       "job detail",
			message:    "深圳的软件开发实习生岗位具体职责是什么？",
			intent:     CandidateIntentJobDetail,
			wantTools:  []string{"get_job_detail_for_candidate", "list_jobs_for_recommendation"},
			outputName: "candidate_job_detail",
		},
		{
			name:       "unknown",
			message:    "随便聊聊",
			intent:     CandidateIntentUnknown,
			disallow:   true,
			outputName: "candidate_unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := planner.Plan(CandidateAssistantPlannerInput{Message: tt.message, AvailableTools: tools})
			if plan.Intent != tt.intent {
				t.Fatalf("intent = %q, want %q", plan.Intent, tt.intent)
			}
			if !sameStringSet(plan.RequiredTools, tt.wantTools) {
				t.Fatalf("required tools = %v, want %v", plan.RequiredTools, tt.wantTools)
			}
			if plan.DisallowsModelTools() != tt.disallow {
				t.Fatalf("DisallowsModelTools = %v, want %v", plan.DisallowsModelTools(), tt.disallow)
			}
			if got, _ := plan.OutputSchema["name"].(string); got != tt.outputName {
				t.Fatalf("output schema name = %q, want %q", got, tt.outputName)
			}
			if tt.disallow && !strings.Contains(plan.InstructionBlock(), "MUST NOT call") {
				t.Fatalf("instruction missing no-tool hard rule: %s", plan.InstructionBlock())
			}
			assertCandidatePlanJSON(t, plan)
		})
	}
}

func TestCandidateAssistantPlannerDoesNotClaimUnavailableTools(t *testing.T) {
	plan := NewCandidateAssistantPlanner().Plan(CandidateAssistantPlannerInput{
		Message:        "根据我的简历推荐一些岗位",
		AvailableTools: []string{"get_my_resume_text"},
	})
	if len(plan.RequiredTools) != 0 {
		t.Fatalf("required tools = %v, want no unavailable recommendation tool", plan.RequiredTools)
	}
	if !containsString(plan.RiskChecks, "no_candidate_tools_available") {
		t.Fatalf("risk checks = %v, want unavailable tool risk", plan.RiskChecks)
	}
}

func TestCandidateAssistantPlannerGreetingNeverRequiresTools(t *testing.T) {
	for _, msg := range []string{"hello", "hi", "你好", "谢谢"} {
		plan := NewCandidateAssistantPlanner().Plan(CandidateAssistantPlannerInput{Message: msg, AvailableTools: candidatePlannerTestTools()})
		if plan.Intent != CandidateIntentGreeting || len(plan.RequiredTools) != 0 || !plan.DisallowsModelTools() {
			t.Fatalf("plan for %q = %#v, want no-tool greeting", msg, plan)
		}
		if len(plan.SuggestedQuestions) != 3 {
			t.Fatalf("suggested questions = %v, want 3", plan.SuggestedQuestions)
		}
	}
}

func assertCandidatePlanJSON(t *testing.T, plan CandidateAssistantPlan) {
	t.Helper()
	var decoded CandidateAssistantPlan
	if err := json.Unmarshal([]byte(plan.JSON()), &decoded); err != nil {
		t.Fatalf("candidate plan JSON is invalid: %v", err)
	}
	if decoded.Intent != plan.Intent {
		t.Fatalf("decoded intent = %q, want %q", decoded.Intent, plan.Intent)
	}
}

func candidatePlannerTestTools() []string {
	return []string{
		"list_my_applications",
		"get_my_application_detail",
		"get_my_resume_text",
		"list_jobs_for_recommendation",
		"get_job_detail_for_candidate",
		"recommend_jobs_by_resume",
	}
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	counts := make(map[string]int, len(got))
	for _, value := range got {
		counts[value]++
	}
	for _, value := range want {
		if counts[value] == 0 {
			return false
		}
		counts[value]--
	}
	return true
}
