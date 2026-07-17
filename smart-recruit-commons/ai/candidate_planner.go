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
	CandidateIntentGreeting            = "greeting"
	CandidateIntentGeneralChat         = "general_chat"
	CandidateIntentApplicationProgress = "application_progress"
	CandidateIntentApplicationDetail   = "application_detail"
	CandidateIntentResumeAdvice        = "resume_advice"
	CandidateIntentJobRecommendation   = "job_recommendation"
	CandidateIntentJobListing          = "job_listing"
	CandidateIntentJobDetail           = "job_detail"
	CandidateIntentUnknown             = "unknown"
)

type CandidateAssistantPlan struct {
	Intent                  string                  `json:"intent"`
	Domain                  string                  `json:"domain"`
	Confidence              float64                 `json:"confidence"`
	RequiresCandidateData   bool                    `json:"requires_candidate_data"`
	KeywordHints            []string                `json:"keyword_hints"`
	PossibleIntents         []string                `json:"possible_intents"`
	ClassificationSource    string                  `json:"classification_source"`
	RequiredTools           []string                `json:"required_tools"`
	RequiredData            []string                `json:"required_data"`
	RequiredToolGroups      []RecruitingToolGroup   `json:"required_tool_groups"`
	DisplaySteps            []RecruitingDisplayStep `json:"display_steps"`
	MissingInputs           []string                `json:"missing_inputs"`
	OutputSchema            map[string]any          `json:"output_schema"`
	RiskChecks              []string                `json:"risk_checks"`
	SuggestedQuestions      []string                `json:"suggested_questions,omitempty"`
	ConfirmationRequirement ConfirmationRequirement `json:"confirmation_requirement"`
}

type CandidateAssistantPlannerInput struct {
	Message        string
	AvailableTools []string
}

type CandidateAssistantPlanner struct{}

func NewCandidateAssistantPlanner() CandidateAssistantPlanner {
	return CandidateAssistantPlanner{}
}

func (CandidateAssistantPlanner) Plan(input CandidateAssistantPlannerInput) CandidateAssistantPlan {
	started := time.Now()
	classification := classifyCandidateAssistantMessage(input.Message)
	available := toolSet(input.AvailableTools)
	plan := CandidateAssistantPlan{
		Intent:                classification.Intent,
		Domain:                classification.Domain,
		Confidence:            classification.Confidence,
		RequiresCandidateData: classification.RequiresRecruitingData,
		KeywordHints:          classification.KeywordHints,
		PossibleIntents:       classification.PossibleIntents,
		ClassificationSource:  classification.Source,
		RequiredToolGroups:    []RecruitingToolGroup{},
		MissingInputs:         []string{},
	}

	switch classification.Intent {
	case CandidateIntentGreeting:
		plan.RequiredTools = []string{}
		plan.RequiredData = []string{"greeting_reply"}
		plan.OutputSchema = objectSchema("candidate_greeting", "greeting", "capability_hints")
		plan.RiskChecks = []string{"do_not_call_any_tools", "do_not_fetch_candidate_data", "offer_capabilities_without_querying"}
		plan.SuggestedQuestions = []string{"我目前的应聘进度？", "根据简历推荐岗位", "帮我优化简历建议"}
	case CandidateIntentGeneralChat:
		plan.RequiredTools = []string{}
		plan.RequiredData = []string{"general_user_message"}
		plan.OutputSchema = objectSchema("candidate_general_chat", "answer", "candidate_scope_note")
		plan.RiskChecks = []string{"do_not_call_candidate_tools", "do_not_fetch_candidate_data", "redirect_to_job_search_scope_when_needed"}
	case CandidateIntentApplicationProgress:
		plan.RequiredToolGroups = []RecruitingToolGroup{toolGroup(available, "candidate_applications", "list_my_applications")}
		plan.RequiredData = []string{"candidate_application_list"}
		plan.OutputSchema = objectSchema("candidate_application_progress", "applications", "total", "next_questions")
		plan.RiskChecks = []string{"only_current_candidate_data", "do_not_invent_applications", "summarize_tool_returned_status"}
	case CandidateIntentApplicationDetail:
		plan.RequiredToolGroups = []RecruitingToolGroup{
			toolGroup(available, "candidate_applications", "list_my_applications"),
			toolGroup(available, "candidate_application_detail", "get_my_application_detail"),
		}
		plan.RequiredData = []string{"candidate_application_list", "application_id_or_resolvable_reference", "application_detail"}
		plan.OutputSchema = objectSchema("candidate_application_detail", "application", "status", "timeline", "caveats")
		plan.RiskChecks = []string{"only_current_candidate_data", "ask_clarifying_question_when_application_reference_is_ambiguous", "do_not_expose_hr_internal_notes"}
	case CandidateIntentResumeAdvice:
		plan.RequiredToolGroups = []RecruitingToolGroup{toolGroup(available, "candidate_resume", "get_my_resume_text")}
		plan.RequiredData = []string{"resume_text_or_unavailable_reason"}
		plan.OutputSchema = objectSchema("candidate_resume_advice", "summary", "suggestions", "missing_information")
		plan.RiskChecks = []string{"only_use_tool_returned_resume_text", "do_not_fabricate_resume_experience", "do_not_write_full_resume_sections"}
	case CandidateIntentJobRecommendation:
		plan.RequiredToolGroups = []RecruitingToolGroup{toolGroup(available, "candidate_job_recommendation", "recommend_jobs_by_resume")}
		plan.RequiredData = []string{"resume_text", "open_jobs", "has_applied_flags"}
		plan.OutputSchema = objectSchema("candidate_job_recommendation", "recommended_jobs", "reasons", "priority")
		plan.RiskChecks = []string{"respect_has_applied_flag", "do_not_claim_application_submitted", "state_resume_missing_when_tool_reports_no_resume"}
	case CandidateIntentJobListing:
		plan.RequiredToolGroups = []RecruitingToolGroup{toolGroup(available, "candidate_job_inventory", "list_jobs_for_recommendation")}
		plan.RequiredData = []string{"open_jobs", "has_applied_flags"}
		plan.OutputSchema = objectSchema("candidate_job_listing", "jobs", "total", "filters")
		plan.RiskChecks = []string{"only_show_candidate_visible_jobs", "do_not_invent_jobs", "include_has_applied_when_available"}
	case CandidateIntentJobDetail:
		plan.RequiredToolGroups = []RecruitingToolGroup{
			toolGroup(available, "candidate_job_inventory", "list_jobs_for_recommendation"),
			toolGroup(available, "candidate_job_detail", "get_job_detail_for_candidate"),
		}
		plan.RequiredData = []string{"job_id_or_resolvable_reference", "job_detail", "has_applied_flag"}
		plan.OutputSchema = objectSchema("candidate_job_detail", "job", "requirements", "salary", "has_applied")
		plan.RiskChecks = []string{"only_show_candidate_visible_job_fields", "ask_clarifying_question_when_job_reference_is_ambiguous", "do_not_invent_interview_process"}
	default:
		plan.RequiredTools = []string{}
		plan.RequiredData = []string{"clarifying_question"}
		plan.OutputSchema = objectSchema("candidate_unknown", "clarifying_question", "supported_intents")
		plan.RiskChecks = []string{"do_not_call_candidate_tools", "ask_for_candidate_intent", "avoid_unsolicited_candidate_data_queries"}
	}

	plan.RequiredTools = toolsFromGroups(plan.RequiredToolGroups)
	plan.DisplaySteps = candidateDisplayStepsForPlan(plan.Intent, plan.RequiredToolGroups)
	plan.RiskChecks = appendCandidateUnavailableToolRisk(plan.RiskChecks, plan.Intent, plan.RequiredTools, available)
	logger.L().Info("[domain][candidate_planner] plan finished",
		zap.String("intent", plan.Intent),
		zap.Int("required_tools", len(plan.RequiredTools)),
		zap.Int64("duration_ms", time.Since(started).Milliseconds()))
	return plan
}

func classifyCandidateAssistantMessage(message string) IntentClassification {
	msg := strings.ToLower(strings.TrimSpace(message))
	switch {
	case isGreetingMessage(msg):
		return candidateAssistantClassification(CandidateIntentGreeting, IntentDomainGeneral, 0.97, false, collectKeywordHints(msg, "hello", "hi", "hey", "你好", "您好", "嗨", "谢谢", "thanks"), CandidateIntentGreeting)
	case isCandidateGeneralChatMessage(msg):
		return candidateAssistantClassification(CandidateIntentGeneralChat, IntentDomainGeneral, 0.9, false, collectKeywordHints(msg, generalChatKeywords()...), CandidateIntentGeneralChat)
	case isCandidateJobRecommendationQuery(msg):
		return candidateAssistantClassification(CandidateIntentJobRecommendation, IntentDomainRecruiting, 0.9, true, collectKeywordHints(msg, "推荐", "适合我", "匹配岗位", "岗位匹配", "recommend", "match jobs"), CandidateIntentJobRecommendation)
	case isCandidateResumeAdviceQuery(msg):
		return candidateAssistantClassification(CandidateIntentResumeAdvice, IntentDomainRecruiting, 0.9, true, collectKeywordHints(msg, "简历", "优化", "建议", "修改", "提升", "润色", "短板", "resume"), CandidateIntentResumeAdvice)
	case isCandidateApplicationDetailQuery(msg):
		return candidateAssistantClassification(CandidateIntentApplicationDetail, IntentDomainRecruiting, 0.86, true, collectKeywordHints(msg, "投递", "应聘", "申请", "详情", "详细", "第", "application detail"), CandidateIntentApplicationDetail)
	case isCandidateApplicationProgressQuery(msg):
		return candidateAssistantClassification(CandidateIntentApplicationProgress, IntentDomainRecruiting, 0.88, true, collectKeywordHints(msg, "投递", "应聘", "申请", "进度", "状态", "面试", "offer", "application", "progress", "status"), CandidateIntentApplicationProgress)
	case isCandidateJobDetailQuery(msg):
		return candidateAssistantClassification(CandidateIntentJobDetail, IntentDomainRecruiting, 0.86, true, collectKeywordHints(msg, "岗位", "职位", "详情", "职责", "要求", "薪资", "地点", "流程", "job detail"), CandidateIntentJobDetail)
	case isCandidateJobListingQuery(msg):
		return candidateAssistantClassification(CandidateIntentJobListing, IntentDomainRecruiting, 0.84, true, collectKeywordHints(msg, "岗位", "职位", "在招", "列表", "有哪些", "open jobs", "job list"), CandidateIntentJobListing)
	default:
		return candidateAssistantClassification(CandidateIntentUnknown, IntentDomainAmbiguous, 0.45, false, []string{}, CandidateIntentUnknown)
	}
}

func candidateAssistantClassification(intent, domain string, confidence float64, requiresData bool, hints []string, possible ...string) IntentClassification {
	if len(possible) == 0 {
		possible = []string{intent}
	}
	return IntentClassification{
		Domain:                 domain,
		Intent:                 intent,
		Confidence:             confidence,
		RequiresRecruitingData: requiresData,
		KeywordHints:           hints,
		PossibleIntents:        possible,
		Source:                 "deterministic_candidate_planner",
	}
}

func (p CandidateAssistantPlan) JSON() string {
	b, err := json.Marshal(p)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (p CandidateAssistantPlan) InstructionBlock() string {
	base := "Deterministic candidate planner output JSON:\n" + p.JSON() + "\nFollow this plan when choosing candidate tools. If required data is missing or a referenced job/application is ambiguous, ask a clarifying question instead of guessing. Do not call tools that are not listed in required_tools."
	if p.DisallowsModelTools() {
		if p.Intent == CandidateIntentGreeting {
			base += "\nHARD RULE: required_tools is empty for this greeting. You MUST NOT call any tools. Do not query applications, resumes, jobs, interviews, or offers. Reply with a short greeting and capability guidance only."
		} else {
			base += "\nHARD RULE: this turn is not a live candidate-data request. You MUST NOT call candidate tools or query applications, resumes, jobs, interviews, or offers. Give a brief direct answer or ask what job-search task the candidate wants help with."
		}
	}
	return base
}

func (p CandidateAssistantPlan) DisallowsModelTools() bool {
	return p.Intent == CandidateIntentGreeting || p.Intent == CandidateIntentGeneralChat || p.Intent == CandidateIntentUnknown
}

func isCandidateGeneralChatMessage(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" || isGreetingMessage(msg) {
		return false
	}
	if containsAny(msg, "代码", "算法", "函数", "编程", "程序", "实现", "示例代码", "quick sort", "quicksort", "leetcode") {
		return true
	}
	if containsAny(msg, "翻译", "总结", "解释一下", "是什么", "为什么", "写一段", "写一个", "想一个", "主题") && !candidateHasLiveDataSignal(msg) {
		return true
	}
	return false
}

func candidateHasLiveDataSignal(message string) bool {
	return containsAny(message, "岗位", "职位", "投递", "应聘", "申请", "简历", "面试", "offer", "求职", "job", "position", "application", "resume", "interview")
}

func isCandidateJobRecommendationQuery(message string) bool {
	return containsAny(message, "推荐", "适合我", "匹配岗位", "岗位匹配", "recommend", "match jobs") &&
		containsAny(message, "岗位", "职位", "工作", "机会", "简历", "job", "position", "resume")
}

func isCandidateResumeAdviceQuery(message string) bool {
	return containsAny(message, "简历", "resume", "cv") &&
		containsAny(message, "优化", "建议", "修改", "提升", "润色", "短板", "不足", "亮点", "advice", "improve", "polish")
}

func isCandidateApplicationProgressQuery(message string) bool {
	return containsAny(message, "投递", "应聘", "申请", "进度", "状态", "面试", "offer", "录用", "未通过", "待查看", "已通过", "application", "applications", "progress", "status")
}

func isCandidateApplicationDetailQuery(message string) bool {
	if !isCandidateApplicationProgressQuery(message) {
		return false
	}
	return containsAny(message, "详情", "详细", "具体", "某条", "第", "这个", "那个", "detail", "specific")
}

func isCandidateJobListingQuery(message string) bool {
	return containsAny(message, "有哪些岗位", "哪些岗位", "岗位列表", "在招岗位", "职位列表", "有什么岗位", "岗位有哪些", "open jobs", "job list", "which jobs")
}

func isCandidateJobDetailQuery(message string) bool {
	return containsAny(message, "岗位", "职位", "工作", "job", "position") &&
		containsAny(message, "详情", "职责", "要求", "薪资", "地点", "部门", "流程", "介绍", "具体", "detail", "requirement", "salary", "location")
}

func candidateDisplayStepsForPlan(intent string, groups []RecruitingToolGroup) []RecruitingDisplayStep {
	steps := make([]RecruitingDisplayStep, 0, len(groups)+1)
	for _, group := range groups {
		steps = append(steps, RecruitingDisplayStep{
			Key:       group.Name,
			Purpose:   candidateDisplayPurposeForGroup(group.Name),
			ToolGroup: group.Name,
			Tools:     append([]string(nil), group.Tools...),
		})
	}
	if intent != CandidateIntentUnknown && intent != CandidateIntentGreeting && intent != CandidateIntentGeneralChat {
		steps = append(steps, RecruitingDisplayStep{
			Key:     "compose_answer",
			Purpose: candidateComposePurpose(intent),
		})
	}
	return steps
}

func candidateDisplayPurposeForGroup(group string) string {
	switch group {
	case "candidate_applications":
		return "读取当前候选人的投递记录"
	case "candidate_application_detail":
		return "读取当前候选人指定投递详情"
	case "candidate_resume":
		return "读取当前候选人的有效简历解析文本"
	case "candidate_job_recommendation":
		return "基于当前候选人简历和在招岗位生成推荐"
	case "candidate_job_inventory":
		return "读取候选人可见的在招岗位"
	case "candidate_job_detail":
		return "读取候选人可见的岗位详情"
	default:
		return "读取候选人可见数据"
	}
}

func candidateComposePurpose(intent string) string {
	switch intent {
	case CandidateIntentApplicationProgress:
		return "整理投递进度并说明下一步可追问内容"
	case CandidateIntentApplicationDetail:
		return "整理指定投递详情并说明可见状态"
	case CandidateIntentResumeAdvice:
		return "基于简历正文生成简短优化建议"
	case CandidateIntentJobRecommendation:
		return "整理推荐岗位、匹配理由和投递优先级"
	case CandidateIntentJobListing:
		return "整理候选人可见岗位列表"
	case CandidateIntentJobDetail:
		return "整理岗位职责、要求和候选人投递标记"
	default:
		return "生成候选人助手回复"
	}
}

func appendCandidateUnavailableToolRisk(risks []string, intent string, requiredTools []string, available map[string]bool) []string {
	if intent == CandidateIntentUnknown || intent == CandidateIntentGreeting || intent == CandidateIntentGeneralChat {
		return risks
	}
	if len(available) == 0 || len(requiredTools) == 0 {
		return append(risks, "no_candidate_tools_available")
	}
	return risks
}

func CandidateToolNames() []string {
	tools := CandidateTools()
	names := make([]string, 0, len(tools))
	seen := map[string]bool{}
	for _, tool := range tools {
		if tool == nil || strings.TrimSpace(tool.Name) == "" || seen[tool.Name] {
			continue
		}
		seen[tool.Name] = true
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}
