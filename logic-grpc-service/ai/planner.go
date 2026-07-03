package ai

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/pkg/logger"
)

const (
	IntentCandidateMatchEvaluation = "candidate_match_evaluation"
	IntentCandidateComparison      = "candidate_comparison"
	IntentAnalytics                = "analytics"
	IntentStatusChangeProposal     = "status_change_proposal"
	IntentInterviewPrep            = "interview_prep"
	IntentOfferSupport             = "offer_support"
	IntentUnknown                  = "unknown"
)

// RecruitingPlan is the deterministic, rule-based planner output passed to
// ADK before execution. It intentionally carries placeholders for future
// skill and memory selectors without persisting planner state.
type RecruitingPlan struct {
	Intent                  string                  `json:"intent"`
	RequiredTools           []string                `json:"required_tools"`
	RequiredData            []string                `json:"required_data"`
	SelectedSkills          []string                `json:"selected_skills"`
	SelectedMemories        []string                `json:"selected_memories"`
	OutputSchema            map[string]any          `json:"output_schema"`
	ConfirmationRequirement ConfirmationRequirement `json:"confirmation_requirement"`
	RiskChecks              []string                `json:"risk_checks"`
}

type ConfirmationRequirement struct {
	Required bool   `json:"required"`
	Reason   string `json:"reason,omitempty"`
}

type RecruitingPlannerInput struct {
	Message        string
	AvailableTools []string
	ApplicationID  int64
}

type RecruitingPlanner struct{}

func NewRecruitingPlanner() RecruitingPlanner {
	return RecruitingPlanner{}
}

func (RecruitingPlanner) Plan(input RecruitingPlannerInput) RecruitingPlan {
	started := time.Now()
	intent := classifyRecruitingIntent(input.Message)
	available := toolSet(input.AvailableTools)

	plan := RecruitingPlan{
		Intent:           intent,
		SelectedSkills:   []string{},
		SelectedMemories: []string{},
	}

	switch intent {
	case IntentCandidateMatchEvaluation:
		plan.RequiredTools = availableTools(available,
			"search_candidates",
			"get_candidate_detail",
			"parse_resume_profile",
			"evaluate_candidate_match",
			"get_candidate_match_evaluation",
		)
		plan.RequiredData = []string{"application_id", "resume_text", "job_requirements", "candidate_match_evaluation"}
		plan.OutputSchema = objectSchema("candidate_match_evaluation", "summary", "score", "evidence", "risks", "next_steps")
		plan.RiskChecks = []string{"verify_candidate_identity", "do_not_infer_from_missing_resume_text", "cite_tool_returned_evidence"}
	case IntentCandidateComparison:
		plan.RequiredTools = availableTools(available,
			"search_jobs",
			"list_applications_by_job",
			"compare_candidates_for_job",
		)
		plan.RequiredData = []string{"job_id", "job_requirements", "candidate_match_rankings"}
		plan.OutputSchema = objectSchema("candidate_comparison", "job", "ranked_candidates", "tradeoffs", "recommended_follow_up")
		plan.RiskChecks = []string{"verify_job_scope", "avoid_unfair_attribute_comparisons", "explain_missing_evaluations"}
	case IntentAnalytics:
		plan.RequiredTools = availableTools(available,
			"query_total_applications",
			"query_today_applications",
			"get_job_heat_ranking",
			"get_application_status_summary",
			"get_application_trend",
			"search_jobs",
		)
		plan.RequiredData = []string{"time_range", "job_filter_optional", "application_counts", "status_distribution", "trend"}
		plan.OutputSchema = objectSchema("analytics", "metrics", "filters", "observations", "caveats")
		plan.RiskChecks = []string{"state_time_window", "do_not_mix_filtered_and_global_counts", "call_tools_for_live_metrics"}
	case IntentStatusChangeProposal:
		plan.RequiredTools = availableTools(available,
			"search_candidates",
			"propose_application_status_update",
		)
		plan.RequiredData = []string{"application_id", "target_status", "candidate_identity"}
		plan.OutputSchema = objectSchema("status_change_proposal", "candidate", "current_status", "target_status", "confirmation_prompt")
		plan.ConfirmationRequirement = ConfirmationRequirement{Required: true, Reason: "status changes must remain proposed until the HR user confirms"}
		plan.RiskChecks = []string{"verify_application_id", "never_claim_database_updated", "ask_clarifying_question_when_target_status_missing"}
	case IntentInterviewPrep:
		plan.RequiredTools = availableTools(available,
			"search_candidates",
			"get_candidate_detail",
			"get_resume_profile",
			"get_job_detail",
		)
		plan.RequiredData = []string{"application_id", "resume_profile", "job_requirements", "interview_focus_areas"}
		plan.OutputSchema = objectSchema("interview_prep", "candidate_context", "focus_areas", "questions", "evaluation_rubric")
		plan.RiskChecks = []string{"base_questions_on_resume_and_job_data", "avoid_protected_attribute_questions", "ask_for_candidate_or_job_when_ambiguous"}
	case IntentOfferSupport:
		plan.RequiredTools = availableTools(available,
			"search_candidates",
			"get_candidate_detail",
			"get_job_detail",
		)
		plan.RequiredData = []string{"application_id", "candidate_identity", "job_details", "compensation_constraints_optional"}
		plan.OutputSchema = objectSchema("offer_support", "candidate", "offer_inputs", "draft_points", "approval_or_confirmation_needed")
		plan.ConfirmationRequirement = ConfirmationRequirement{Required: true, Reason: "offer actions and compensation commitments require explicit HR approval"}
		plan.RiskChecks = []string{"do_not_send_or_create_offer_without_tool_support", "avoid_unverified_compensation_terms", "verify_authorization_and_candidate_identity"}
	default:
		plan.RequiredTools = []string{}
		plan.RequiredData = []string{"clarifying_question"}
		plan.OutputSchema = objectSchema("unknown", "clarifying_question", "known_constraints")
		plan.RiskChecks = []string{"do_not_claim_unavailable_tools", "ask_for_missing_recruiting_intent_or_entities"}
	}

	plan.RiskChecks = appendUnavailableToolRisk(plan.RiskChecks, intent, available)
	logger.L().Info("[logic][planner] plan finished",
		zap.String("intent", intent),
		zap.Int("required_tools", len(plan.RequiredTools)),
		zap.Int("risk_checks", len(plan.RiskChecks)),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
	return plan
}

func (p RecruitingPlan) JSON() string {
	b, err := json.Marshal(p)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (p RecruitingPlan) InstructionBlock() string {
	return "Deterministic planner output JSON:\n" + p.JSON() + "\nFollow this plan when choosing tools. If required data is missing, ask for it instead of guessing. Do not call tools that are not listed in required_tools."
}

func classifyRecruitingIntent(message string) string {
	msg := strings.ToLower(strings.TrimSpace(message))
	switch {
	case containsAny(msg, "对比", "比较", "排名", "排序", "哪个更适合", "compare", "ranking", "rank candidates"):
		return IntentCandidateComparison
	case containsAny(msg, "匹配度", "匹配评估", "候选人匹配", "适配度", "胜任", "match evaluation", "candidate match", "fit score"):
		return IntentCandidateMatchEvaluation
	case containsAny(msg, "offer", "薪资方案", "报价", "录用通知", "发放录用", "offer support"):
		return IntentOfferSupport
	case containsAny(msg, "通过", "淘汰", "拒绝", "录用", "进入下一轮", "推进", "状态改", "状态变更", "status update", "reject", "approve"):
		return IntentStatusChangeProposal
	case containsAny(msg, "面试准备", "面试题", "面试问题", "面试提纲", "interview prep", "interview questions"):
		return IntentInterviewPrep
	case containsAny(msg, "统计", "趋势", "漏斗", "热度", "排行", "多少", "今日", "今天", "analytics", "trend", "funnel", "metrics"):
		return IntentAnalytics
	default:
		return IntentUnknown
	}
}

func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func toolSet(tools []string) map[string]bool {
	set := make(map[string]bool, len(tools))
	for _, name := range tools {
		name = strings.TrimSpace(name)
		if name != "" {
			set[name] = true
		}
	}
	return set
}

func availableTools(available map[string]bool, desired ...string) []string {
	out := make([]string, 0, len(desired))
	for _, name := range desired {
		if available[name] {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func objectSchema(name string, fields ...string) map[string]any {
	properties := make(map[string]any, len(fields))
	for _, field := range fields {
		properties[field] = map[string]any{"type": "string"}
	}
	return map[string]any{
		"name":       name,
		"type":       "object",
		"properties": properties,
	}
}

func appendUnavailableToolRisk(risks []string, intent string, available map[string]bool) []string {
	if intent == IntentUnknown {
		return risks
	}
	if len(available) == 0 {
		return append(risks, "no_builtin_recruiting_tools_available")
	}
	return risks
}
