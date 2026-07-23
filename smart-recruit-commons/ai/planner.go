package ai

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"smart-recruit-platform-go/logger"
)

const (
	IntentGeneralChat              = "general_chat"
	IntentCandidateMatchEvaluation = "candidate_match_evaluation"
	IntentCandidateComparison      = "candidate_comparison"
	IntentCandidateLookup          = "candidate_lookup"
	IntentApplicationListing       = "application_listing"
	IntentAnalytics                = "analytics"
	IntentJobListing               = "job_listing"
	IntentJobDetail                = "job_detail"
	IntentStatusChangeProposal     = "status_change_proposal"
	IntentInterviewPrep            = "interview_prep"
	IntentOfferSupport             = "offer_support"
	IntentGreeting                 = "greeting"
	IntentUnknown                  = "unknown"
)

const (
	HRSuggestedQuestionsStartMarker = "<<<HR_SUGGESTED_QUESTIONS_JSON>>>"
	HRSuggestedQuestionsEndMarker   = "<<<END_HR_SUGGESTED_QUESTIONS_JSON>>>"
)

const (
	IntentDomainGeneral    = "general_chat"
	IntentDomainRecruiting = "recruiting_domain"
	IntentDomainAmbiguous  = "ambiguous"
)

// RecruitingPlan is the deterministic, rule-based planner output passed to
// ADK before execution. It intentionally carries placeholders for future
// skill and memory selectors without persisting planner state.
type RecruitingPlan struct {
	Intent                  string                  `json:"intent"`
	Domain                  string                  `json:"domain"`
	Confidence              float64                 `json:"confidence"`
	RequiresRecruitingData  bool                    `json:"requires_recruiting_data"`
	KeywordHints            []string                `json:"keyword_hints"`
	PossibleIntents         []string                `json:"possible_intents"`
	ClassificationSource    string                  `json:"classification_source"`
	RequiredTools           []string                `json:"required_tools"`
	RequiredData            []string                `json:"required_data"`
	RequiredToolGroups      []RecruitingToolGroup   `json:"required_tool_groups"`
	DisplaySteps            []RecruitingDisplayStep `json:"display_steps"`
	MissingInputs           []string                `json:"missing_inputs"`
	SelectedSkills          []string                `json:"selected_skills"`
	SelectedMemories        []string                `json:"selected_memories"`
	OutputSchema            map[string]any          `json:"output_schema"`
	ConfirmationRequirement ConfirmationRequirement `json:"confirmation_requirement"`
	RiskChecks              []string                `json:"risk_checks"`
	SuggestedQuestions      []string                `json:"suggested_questions,omitempty"`
}

type IntentClassification struct {
	Domain                 string
	Intent                 string
	Confidence             float64
	RequiresRecruitingData bool
	KeywordHints           []string
	PossibleIntents        []string
	Source                 string
}

// RecruitingToolGroup defines one mandatory evidence group. At least one Tool
// in each group must succeed before the runtime may answer with live facts.
type RecruitingToolGroup struct {
	Name  string   `json:"name"`
	Tools []string `json:"tools"`
}

type RecruitingDisplayStep struct {
	Key       string   `json:"key"`
	Purpose   string   `json:"purpose"`
	ToolGroup string   `json:"tool_group,omitempty"`
	Tools     []string `json:"tools,omitempty"`
}

type ConfirmationRequirement struct {
	Required bool   `json:"required"`
	Reason   string `json:"reason,omitempty"`
}

type RecruitingPlannerInput struct {
	Message        string
	AvailableTools []string
	ApplicationID  int64
	JobID          int64
}

type RecruitingPlanner struct{}

func NewRecruitingPlanner() RecruitingPlanner {
	return RecruitingPlanner{}
}

func (RecruitingPlanner) Plan(input RecruitingPlannerInput) RecruitingPlan {
	started := time.Now()
	classification := classifyRecruitingMessage(input.Message)
	intent := classification.Intent
	available := toolSet(input.AvailableTools)

	plan := RecruitingPlan{
		Intent:                 intent,
		Domain:                 classification.Domain,
		Confidence:             classification.Confidence,
		RequiresRecruitingData: classification.RequiresRecruitingData,
		KeywordHints:           classification.KeywordHints,
		PossibleIntents:        classification.PossibleIntents,
		ClassificationSource:   classification.Source,
		SelectedSkills:         []string{},
		SelectedMemories:       []string{},
		RequiredToolGroups:     []RecruitingToolGroup{},
		MissingInputs:          []string{},
	}

	switch intent {
	case IntentGeneralChat:
		plan.RequiredTools = []string{}
		plan.RequiredData = []string{"general_user_message"}
		plan.OutputSchema = objectSchema("general_chat", "answer", "caveats")
		plan.RiskChecks = []string{"do_not_call_recruiting_tools", "do_not_fetch_live_recruiting_data", "answer_without_live_data_gate"}
	case IntentCandidateMatchEvaluation:
		plan.RequiredToolGroups = []RecruitingToolGroup{
			toolGroup(available, "candidate_identity", "get_candidate_detail"),
			toolGroup(available, "match_evidence", "get_candidate_match_evaluation", "evaluate_candidate_match"),
		}
		if input.ApplicationID <= 0 {
			plan.MissingInputs = append(plan.MissingInputs, "application_id")
		}
		plan.RequiredData = []string{"application_id", "resume_text", "job_requirements", "candidate_match_evaluation"}
		plan.OutputSchema = objectSchema("candidate_match_evaluation", "summary", "score", "evidence", "risks", "next_steps")
		plan.RiskChecks = []string{"verify_candidate_identity", "do_not_infer_from_missing_resume_text", "cite_tool_returned_evidence"}
	case IntentCandidateComparison:
		plan.RequiredToolGroups = []RecruitingToolGroup{
			toolGroup(available, "job_identity", "get_job_detail"),
			toolGroup(available, "job_candidates", "list_applications_by_job"),
		}
		if input.ApplicationID <= 0 && input.JobID <= 0 {
			plan.MissingInputs = append(plan.MissingInputs, "job_id")
		}
		plan.RequiredData = []string{"job_id", "job_requirements", "candidate_match_rankings"}
		plan.OutputSchema = objectSchema("candidate_comparison", "job", "ranked_candidates", "tradeoffs", "recommended_follow_up")
		plan.RiskChecks = []string{"verify_job_scope", "avoid_unfair_attribute_comparisons", "explain_missing_evaluations"}
	case IntentAnalytics:
		plan.RequiredToolGroups = analyticsToolGroups(input.Message, available)
		plan.RequiredData = []string{"time_range", "job_filter_optional", "application_counts", "status_distribution", "trend"}
		plan.OutputSchema = objectSchema("analytics", "metrics", "filters", "observations", "caveats")
		plan.RiskChecks = []string{"state_time_window", "do_not_mix_filtered_and_global_counts", "call_tools_for_live_metrics"}
	case IntentJobListing:
		plan.RequiredToolGroups = []RecruitingToolGroup{toolGroup(available, "job_inventory", "get_job_list", "search_jobs")}
		plan.RequiredData = []string{"job_inventory", "job_status_filter_optional"}
		plan.OutputSchema = objectSchema("job_listing", "jobs", "total", "filters", "caveats")
		plan.RiskChecks = []string{"call_tools_for_live_job_data", "do_not_invent_jobs", "state_when_inventory_empty"}
	case IntentJobDetail:
		if input.ApplicationID > 0 {
			plan.RequiredToolGroups = append(plan.RequiredToolGroups, toolGroup(available, "application_context", "get_application_snapshot", "get_candidate_detail"))
		}
		plan.RequiredToolGroups = append(plan.RequiredToolGroups, toolGroup(available, "job_detail", "get_job_detail"))
		if input.ApplicationID <= 0 && input.JobID <= 0 {
			plan.MissingInputs = append(plan.MissingInputs, "job_id")
		}
		plan.RequiredData = []string{"job_id", "job_detail"}
		plan.OutputSchema = objectSchema("job_detail", "job", "caveats")
		plan.RiskChecks = []string{"call_tools_for_live_job_data", "verify_job_scope", "do_not_invent_job_details"}
	case IntentStatusChangeProposal:
		plan.RequiredToolGroups = []RecruitingToolGroup{toolGroup(available, "candidate_identity", "get_application_snapshot", "get_candidate_detail")}
		if input.ApplicationID <= 0 {
			plan.MissingInputs = append(plan.MissingInputs, "application_id")
		}
		plan.RequiredData = []string{"application_id", "target_status", "candidate_identity"}
		plan.OutputSchema = objectSchema("status_change_proposal", "candidate", "current_status", "target_status", "confirmation_prompt")
		plan.ConfirmationRequirement = ConfirmationRequirement{Required: true, Reason: "status changes must remain proposed until the HR user confirms"}
		plan.RiskChecks = []string{"verify_application_id", "never_claim_database_updated", "ask_clarifying_question_when_target_status_missing"}
	case IntentInterviewPrep:
		plan.RequiredToolGroups = []RecruitingToolGroup{
			toolGroup(available, "candidate_identity", "get_application_snapshot", "get_candidate_detail"),
			toolGroup(available, "job_identity", "get_job_detail"),
		}
		if input.ApplicationID <= 0 {
			plan.MissingInputs = append(plan.MissingInputs, "application_id")
		}
		plan.RequiredData = []string{"application_id", "resume_profile", "job_requirements", "interview_focus_areas"}
		plan.OutputSchema = objectSchema("interview_prep", "candidate_context", "focus_areas", "questions", "evaluation_rubric")
		plan.RiskChecks = []string{"base_questions_on_resume_and_job_data", "avoid_protected_attribute_questions", "ask_for_candidate_or_job_when_ambiguous"}
	case IntentOfferSupport:
		plan.RequiredToolGroups = []RecruitingToolGroup{
			toolGroup(available, "candidate_identity", "get_application_snapshot", "get_candidate_detail"),
			toolGroup(available, "job_identity", "get_job_detail"),
		}
		if input.ApplicationID <= 0 {
			plan.MissingInputs = append(plan.MissingInputs, "application_id")
		}
		plan.RequiredData = []string{"application_id", "candidate_identity", "job_details", "compensation_constraints_optional"}
		plan.OutputSchema = objectSchema("offer_support", "candidate", "offer_inputs", "draft_points", "approval_or_confirmation_needed")
		plan.ConfirmationRequirement = ConfirmationRequirement{Required: true, Reason: "offer actions and compensation commitments require explicit HR approval"}
		plan.RiskChecks = []string{"do_not_send_or_create_offer_without_tool_support", "avoid_unverified_compensation_terms", "verify_authorization_and_candidate_identity"}
	case IntentApplicationListing:
		plan.RequiredToolGroups = []RecruitingToolGroup{applicationListingToolGroup(input.Message, input.ApplicationID, available)}
		plan.RequiredData = []string{"application_inventory", "application_filter_optional"}
		plan.OutputSchema = objectSchema("application_listing", "applications", "total", "filters", "caveats")
		plan.RiskChecks = []string{"call_tools_for_live_application_data", "do_not_invent_applications", "state_when_inventory_empty"}
	case IntentCandidateLookup:
		plan.RequiredToolGroups = []RecruitingToolGroup{toolGroup(available, "candidate_search", "search_candidates")}
		if input.ApplicationID > 0 || candidateDetailRequested(input.Message) {
			plan.RequiredToolGroups = append(plan.RequiredToolGroups, toolGroup(available, "candidate_detail", "get_candidate_detail"))
		}
		plan.RequiredData = []string{"candidate_identity", "application_scope"}
		plan.OutputSchema = objectSchema("candidate_lookup", "candidates", "total", "filters", "caveats")
		plan.RiskChecks = []string{"call_tools_for_live_candidate_data", "do_not_invent_candidates", "respect_hr_scope"}
	case IntentGreeting:
		plan.RequiredTools = []string{}
		plan.RequiredData = []string{"greeting_reply"}
		plan.OutputSchema = objectSchema("greeting", "greeting", "capability_hints")
		plan.RiskChecks = []string{"do_not_call_any_tools", "do_not_fetch_live_recruiting_data", "offer_capabilities_without_querying"}
	default:
		plan.RequiredTools = []string{}
		plan.RequiredData = []string{"clarifying_question"}
		plan.OutputSchema = objectSchema("unknown", "clarifying_question", "known_constraints")
		plan.RiskChecks = []string{"do_not_claim_unavailable_tools", "ask_for_missing_recruiting_intent_or_entities", "prefer_clarifying_question_over_unsolicited_inventory_queries"}
	}
	plan.RequiredTools = toolsFromGroups(plan.RequiredToolGroups)
	plan.DisplaySteps = displayStepsForPlan(intent, plan.RequiredToolGroups)
	if !plan.ConfirmationRequirement.Required {
		plan.SuggestedQuestions = recruitingSuggestedQuestions(intent)
	}

	plan.RiskChecks = appendUnavailableToolRisk(plan.RiskChecks, intent, available)
	logger.L().Info("[domain][planner] plan finished",
		zap.String("intent", intent),
		zap.Int("required_tools", len(plan.RequiredTools)),
		zap.Int("risk_checks", len(plan.RiskChecks)),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
	return plan
}

func displayStepsForPlan(intent string, groups []RecruitingToolGroup) []RecruitingDisplayStep {
	steps := make([]RecruitingDisplayStep, 0, len(groups)+1)
	for _, group := range groups {
		purpose := displayPurposeForGroup(intent, group.Name)
		if purpose == "" {
			purpose = "查询所需的实时招聘数据"
		}
		steps = append(steps, RecruitingDisplayStep{
			Key:       group.Name,
			Purpose:   purpose,
			ToolGroup: group.Name,
			Tools:     append([]string(nil), group.Tools...),
		})
	}
	if intent != IntentUnknown && intent != IntentGreeting {
		steps = append(steps, RecruitingDisplayStep{
			Key:     "compose_answer",
			Purpose: displayComposePurpose(intent),
		})
	}
	return steps
}

func displayPurposeForGroup(intent, group string) string {
	switch group {
	case "candidate_identity":
		return "读取当前投递和候选人上下文"
	case "match_evidence":
		return "获取或生成候选人与岗位的匹配评估"
	case "job_identity", "job_detail":
		return "读取岗位详情和任职要求"
	case "job_candidates":
		return "读取该岗位下的候选人和投递数据"
	case "job_inventory":
		return "读取当前岗位列表"
	case "application_context":
		return "读取当前投递关联的候选人和岗位上下文"
	case "application_inventory":
		return "读取投递记录列表"
	case "candidate_search":
		return "搜索符合条件的候选人"
	case "candidate_detail":
		return "读取候选人详情"
	case "today_applications":
		return "统计今日投递数据"
	case "application_trend":
		return "读取投递趋势数据"
	case "job_heat":
		return "读取岗位热度排行"
	case "application_status":
		return "读取投递状态分布"
	case "total_applications":
		return "统计累计投递数据"
	default:
		if intent == IntentAnalytics {
			return "读取招聘统计指标"
		}
		return ""
	}
}

func displayComposePurpose(intent string) string {
	switch intent {
	case IntentGeneralChat:
		return "直接回答用户的通用问题，不查询招聘实时数据"
	case IntentCandidateMatchEvaluation:
		return "基于候选人、岗位和匹配评估形成结论"
	case IntentCandidateComparison:
		return "对候选人进行对比并整理建议"
	case IntentAnalytics:
		return "归纳招聘指标并说明口径"
	case IntentJobListing:
		return "整理岗位列表和可用筛选条件"
	case IntentJobDetail:
		return "整理岗位详情并说明关键信息"
	case IntentStatusChangeProposal:
		return "整理状态变更建议并等待确认"
	case IntentInterviewPrep:
		return "基于简历和岗位生成面试准备内容"
	case IntentOfferSupport:
		return "整理 Offer 支持信息和待确认项"
	case IntentApplicationListing:
		return "整理投递记录和筛选结果"
	case IntentCandidateLookup:
		return "整理候选人查询结果"
	case IntentGreeting:
		return "礼貌问候并介绍可提供的招聘数据能力"
	default:
		return "整理已获取的数据并生成回复"
	}
}

func (p RecruitingPlan) JSON() string {
	b, err := json.Marshal(p)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (p RecruitingPlan) InstructionBlock() string {
	base := "Deterministic planner output JSON:\n" + p.JSON() + "\nFollow this plan when choosing tools. If required data is missing, ask for it instead of guessing. Do not call tools that are not listed in required_tools."
	if p.DisallowsModelTools() {
		if p.Intent == IntentGreeting {
			base += "\nHARD RULE: required_tools is empty for this turn. You MUST NOT call any tools. Do not query jobs, applications, candidates, or analytics. Reply with a short greeting and capability guidance only."
		} else {
			base += "\nHARD RULE: this turn is not a live recruiting-data request. You MUST NOT call recruiting tools or query jobs, applications, candidates, resumes, or analytics. Answer the user's request directly from general model knowledge, and state limitations when appropriate."
		}
	}
	if len(p.SuggestedQuestions) == 3 {
		base += "\nAfter the user-visible answer, append exactly three short, natural follow-up questions as a JSON string array between these markers:\n" + HRSuggestedQuestionsStartMarker + "\n[\"question 1\",\"question 2\",\"question 3\"]\n" + HRSuggestedQuestionsEndMarker + "\nThe questions must follow from this turn, use generic references such as 该候选人 or 该岗位, and must not contain candidate names, phone numbers, email addresses, IDs, salaries, or other personal data. Do not mention these markers in the user-visible answer."
	}
	return base
}

func recruitingSuggestedQuestions(intent string) []string {
	var questions []string
	switch intent {
	case IntentJobListing, IntentJobDetail:
		questions = []string{"查看这些岗位的投递情况", "分析相关岗位的候选人分布", "哪些岗位需要优先推进招聘？"}
	case IntentApplicationListing, IntentCandidateLookup:
		questions = []string{"查看相关候选人的详细信息", "分析候选人与岗位的匹配度", "为候选人准备面试问题"}
	case IntentCandidateMatchEvaluation, IntentCandidateComparison:
		questions = []string{"展开说明主要匹配证据", "比较候选人的关键差异", "给出下一步面试建议"}
	case IntentAnalytics:
		questions = []string{"按岗位拆分这些招聘数据", "查看招聘漏斗的变化趋势", "哪些招聘环节需要重点关注？"}
	case IntentInterviewPrep:
		questions = []string{"补充更有针对性的面试问题", "生成面试评价要点", "查看候选人的匹配风险"}
	case IntentOfferSupport:
		questions = []string{"整理 Offer 沟通要点", "检查还需要确认哪些信息", "生成审批前核对清单"}
	default:
		questions = []string{"查看当前招聘数据概览", "有哪些正在招聘的岗位？", "帮我分析候选人匹配度"}
	}
	return append([]string(nil), questions...)
}

// DisallowsModelTools reports whether the runtime must expose zero model tools.
// Greetings and non-recruiting general turns are hard-banned from recruiting
// tools; other intents keep model tools subject to planner guidance.
func (p RecruitingPlan) DisallowsModelTools() bool {
	return p.Intent == IntentGreeting || p.Intent == IntentGeneralChat
}

func classifyRecruitingIntent(message string) string {
	return classifyRecruitingMessage(message).Intent
}

func classifyRecruitingMessage(message string) IntentClassification {
	msg := strings.ToLower(strings.TrimSpace(message))
	switch {
	case isGeneralChatMessage(msg):
		return IntentClassification{
			Domain:                 IntentDomainGeneral,
			Intent:                 IntentGeneralChat,
			Confidence:             0.92,
			RequiresRecruitingData: false,
			KeywordHints:           collectKeywordHints(msg, generalChatKeywords()...),
			PossibleIntents:        possibleIntentsForGeneralMessage(msg),
			Source:                 "deterministic_domain_classifier",
		}
	case isGreetingMessage(msg):
		return recruitingClassification(IntentGreeting, 0.97, false, collectKeywordHints(msg, "hello", "hi", "hey", "你好", "您好", "嗨", "谢谢", "thanks"), IntentGreeting)
	case containsAny(msg, "对比", "比较", "排名", "排序", "哪个更适合", "compare", "ranking", "rank candidates"):
		return recruitingClassification(IntentCandidateComparison, 0.82, true, collectKeywordHints(msg, "对比", "比较", "排名", "排序", "哪个更适合", "compare", "ranking", "rank candidates"), IntentCandidateComparison)
	case containsAny(msg, "匹配度", "匹配评估", "候选人匹配", "适配度", "胜任", "match evaluation", "candidate match", "fit score"):
		return recruitingClassification(IntentCandidateMatchEvaluation, 0.9, true, collectKeywordHints(msg, "匹配度", "匹配评估", "候选人匹配", "适配度", "胜任", "match evaluation", "candidate match", "fit score"), IntentCandidateMatchEvaluation)
	case containsAny(msg, "offer", "薪资方案", "报价", "录用通知", "发放录用", "offer support"):
		return recruitingClassification(IntentOfferSupport, 0.86, true, collectKeywordHints(msg, "offer", "薪资方案", "报价", "录用通知", "发放录用", "offer support"), IntentOfferSupport)
	case containsAny(msg, "面试准备", "面试题", "面试问题", "面试提纲", "interview prep", "interview questions"):
		return recruitingClassification(IntentInterviewPrep, 0.86, true, collectKeywordHints(msg, "面试准备", "面试题", "面试问题", "面试提纲", "interview prep", "interview questions"), IntentInterviewPrep)
	case isApplicationListingQuery(msg):
		return recruitingClassification(IntentApplicationListing, 0.88, true, collectKeywordHints(msg, "投递", "application", "投递列表", "所有投递", "全部投递", "已通过", "已淘汰"), IntentApplicationListing)
	case containsAny(msg, "候选人", "应聘者", "candidate") && containsAny(msg, "查找", "搜索", "查询", "列表", "详情", "是谁", "search", "find", "list", "detail"):
		return recruitingClassification(IntentCandidateLookup, 0.84, true, collectKeywordHints(msg, "候选人", "应聘者", "candidate", "查找", "搜索", "查询", "列表", "详情", "search", "find", "list", "detail"), IntentCandidateLookup)
	case containsAny(msg, "岗位详情", "职位详情", "job detail", "position detail") ||
		(containsAny(msg, "岗位", "职位", "job", "position") && containsAny(msg, "详情", "详细", "detail")):
		return recruitingClassification(IntentJobDetail, 0.88, true, collectKeywordHints(msg, "岗位详情", "职位详情", "job detail", "position detail", "岗位", "职位", "job", "position", "详情", "详细", "detail"), IntentJobDetail)
	case containsAny(msg,
		"有哪些岗位", "哪些岗位", "岗位列表", "现在有哪些", "当前岗位", "在招岗位", "发布的岗位",
		"有什么岗位", "岗位有哪些", "职位列表", "有哪些职位", "有多少岗位", "岗位数量", "职位数量", "job list", "list jobs", "open positions", "which jobs", "how many jobs"):
		return recruitingClassification(IntentJobListing, 0.9, true, collectKeywordHints(msg, "有哪些岗位", "岗位列表", "当前岗位", "在招岗位", "职位列表", "有多少岗位", "job list", "list jobs", "open positions", "which jobs", "how many jobs"), IntentJobListing)
	case containsAny(msg, "岗位", "职位", "jobs", "positions") && containsAny(msg, "哪些", "什么", "列表", "全部", "所有", "list", "all", "open"):
		return recruitingClassification(IntentJobListing, 0.82, true, collectKeywordHints(msg, "岗位", "职位", "jobs", "positions", "哪些", "什么", "列表", "全部", "所有", "list", "all", "open"), IntentJobListing)
	case containsAny(msg, "统计", "趋势", "漏斗", "热度", "排行", "多少", "今日", "今天", "analytics", "trend", "funnel", "metrics"):
		return recruitingClassification(IntentAnalytics, 0.78, true, collectKeywordHints(msg, "统计", "趋势", "漏斗", "热度", "排行", "多少", "今日", "今天", "analytics", "trend", "funnel", "metrics"), IntentAnalytics)
	case containsAny(msg, "通过", "淘汰", "拒绝", "录用", "进入下一轮", "推进", "状态改", "状态变更", "status update", "reject", "approve"):
		return recruitingClassification(IntentStatusChangeProposal, 0.8, true, collectKeywordHints(msg, "通过", "淘汰", "拒绝", "录用", "进入下一轮", "推进", "状态改", "状态变更", "status update", "reject", "approve"), IntentStatusChangeProposal)
	default:
		return IntentClassification{
			Domain:                 IntentDomainAmbiguous,
			Intent:                 IntentUnknown,
			Confidence:             0.45,
			RequiresRecruitingData: false,
			KeywordHints:           []string{},
			PossibleIntents:        []string{IntentUnknown},
			Source:                 "deterministic_domain_classifier",
		}
	}
}

func recruitingClassification(intent string, confidence float64, requiresData bool, hints []string, possible ...string) IntentClassification {
	if len(possible) == 0 {
		possible = []string{intent}
	}
	domain := IntentDomainRecruiting
	if !requiresData && intent == IntentGreeting {
		domain = IntentDomainGeneral
	}
	return IntentClassification{
		Domain:                 domain,
		Intent:                 intent,
		Confidence:             confidence,
		RequiresRecruitingData: requiresData,
		KeywordHints:           hints,
		PossibleIntents:        possible,
		Source:                 "deterministic_domain_classifier",
	}
}

func isGeneralChatMessage(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return false
	}
	if isGreetingMessage(msg) {
		return false
	}
	if containsAny(msg, "代码", "算法", "函数", "编程", "程序", "实现", "示例代码", "quick sort", "quicksort", "leetcode", "python", "golang", "go语言", "java", "javascript", "typescript", "sql", "正则") {
		return true
	}
	if containsAny(msg, "翻译", "润色", "改写", "总结", "解释一下", "是什么", "为什么", "怎么做", "写一段", "写一个", "想一个", "主题", "建议") && !hasLiveRecruitingSignal(msg) {
		return true
	}
	return false
}

func hasLiveRecruitingSignal(message string) bool {
	if !containsAny(message, "岗位", "职位", "投递", "简历", "候选人", "应聘者", "面试", "offer", "招聘", "job", "position", "application", "resume", "candidate", "interview") {
		return false
	}
	return containsAny(message, "当前", "现在", "实时", "系统", "这些", "这个", "该", "列表", "详情", "查询", "搜索", "统计", "趋势", "状态", "匹配度", "排序", "比较", "多少", "today", "current", "list", "detail", "search", "analytics")
}

func generalChatKeywords() []string {
	return []string{"代码", "算法", "函数", "编程", "程序", "实现", "示例代码", "quick sort", "quicksort", "leetcode", "python", "golang", "go语言", "java", "javascript", "typescript", "sql", "正则", "翻译", "润色", "改写", "总结", "解释一下", "写一段", "写一个", "想一个", "主题", "建议"}
}

func possibleIntentsForGeneralMessage(message string) []string {
	possible := []string{IntentGeneralChat}
	if containsAny(message, "排序", "排名", "ranking", "rank") {
		possible = append(possible, IntentCandidateComparison)
	}
	if containsAny(message, "多少", "统计", "趋势", "排行", "analytics", "metrics") {
		possible = append(possible, IntentAnalytics)
	}
	return possible
}

func collectKeywordHints(message string, keywords ...string) []string {
	hints := make([]string, 0, len(keywords))
	seen := map[string]bool{}
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" || seen[keyword] || !strings.Contains(message, strings.ToLower(keyword)) {
			continue
		}
		seen[keyword] = true
		hints = append(hints, keyword)
	}
	return hints
}

func isGreetingMessage(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return false
	}
	// Strip common trailing punctuation / emoji-ish suffixes for short greetings.
	msg = strings.Trim(msg, " \t\n\r!！.。?？~～、,，")
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return false
	}
	exact := map[string]bool{
		"hello": true, "hi": true, "hey": true, "hola": true,
		"你好": true, "您好": true, "嗨": true, "哈喽": true, "哈囉": true,
		"在吗": true, "在嘛": true, "早上好": true, "下午好": true, "晚上好": true,
		"早": true, "晚安": true, "thanks": true, "thank you": true,
		"谢谢": true, "多谢": true, "谢谢你": true, "thankyou": true,
	}
	if exact[msg] {
		return true
	}
	// Very short pure greeting phrases only; avoid matching real queries.
	if len([]rune(msg)) <= 12 && containsAny(msg, "hello", "hi there", "hey there", "你好呀", "你好啊", "您好呀") {
		// Reject if it still looks like a data question.
		if containsAny(msg, "岗位", "职位", "投递", "候选人", "统计", "job", "application", "candidate") {
			return false
		}
		return true
	}
	return false
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
	return out
}

func isApplicationListingQuery(message string) bool {
	if !containsAny(message, "投递", "application") {
		return false
	}
	return containsAny(message,
		"投递列表", "所有投递", "全部投递", "有哪些投递", "哪些投递", "投递有哪些", "投递记录",
		"已通过", "已淘汰", "待查看", "已查看", "这个岗位", "该岗位", "application list", "list applications")
}

func candidateDetailRequested(message string) bool {
	message = strings.ToLower(strings.TrimSpace(message))
	return containsAny(message, "详情", "详细", "detail", "profile")
}

func applicationListingToolGroup(message string, applicationID int64, available map[string]bool) RecruitingToolGroup {
	message = strings.ToLower(strings.TrimSpace(message))
	switch {
	case containsAny(message, "待查看", "已查看", "已通过", "已淘汰", "通过的投递", "淘汰的投递", "状态"):
		return toolGroup(available, "application_inventory", "list_applications_by_status", "list_all_applications")
	case applicationID > 0 || containsAny(message, "这个岗位", "该岗位", "当前岗位"):
		return toolGroup(available, "application_inventory", "list_applications_by_job", "list_all_applications")
	default:
		return toolGroup(available, "application_inventory", "list_all_applications")
	}
}

func toolGroup(available map[string]bool, name string, desired ...string) RecruitingToolGroup {
	return RecruitingToolGroup{Name: name, Tools: availableTools(available, desired...)}
}

func toolsFromGroups(groups []RecruitingToolGroup) []string {
	set := map[string]bool{}
	for _, group := range groups {
		for _, name := range group.Tools {
			set[name] = true
		}
	}
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func analyticsToolGroups(message string, available map[string]bool) []RecruitingToolGroup {
	msg := strings.ToLower(strings.TrimSpace(message))
	groups := make([]RecruitingToolGroup, 0, 4)
	if containsAny(msg, "今日", "今天", "today") {
		groups = append(groups, toolGroup(available, "today_applications", "query_today_applications"))
	}
	if containsAny(msg, "趋势", "trend") {
		groups = append(groups, toolGroup(available, "application_trend", "get_application_trend"))
	}
	if containsAny(msg, "热度", "排行", "ranking", "hot") {
		groups = append(groups, toolGroup(available, "job_heat", "get_job_heat_ranking"))
	}
	if containsAny(msg, "状态", "漏斗", "分布", "funnel", "status") {
		groups = append(groups, toolGroup(available, "application_status", "get_application_status_summary"))
	}
	if len(groups) == 0 || containsAny(msg, "累计", "总数", "多少", "统计", "total", "count", "metrics") {
		groups = append(groups, toolGroup(available, "total_applications", "query_total_applications"))
	}
	return groups
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
	if intent == IntentUnknown || intent == IntentGreeting || intent == IntentGeneralChat {
		return risks
	}
	if len(available) == 0 {
		return append(risks, "no_builtin_recruiting_tools_available")
	}
	return risks
}
