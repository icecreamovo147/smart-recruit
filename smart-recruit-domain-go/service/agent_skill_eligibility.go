package service

import (
	"strings"
	"unicode"
)

type AgentSkillEligibility struct {
	Allowed    bool
	Reason     string
	QueryClass string
}

var recruitingActionTerms = []string{
	"筛", "筛选", "匹配", "评估", "分析", "推荐", "查找", "搜索", "安排", "预约", "邀约", "约",
	"生成", "撰写", "创建", "发布", "修改", "优化", "发送", "沟通", "跟进", "比较", "排序", "总结",
	"提取", "解析", "录入", "更新", "淘汰", "通过", "拒绝", "推进", "入职",
	"screen", "match", "evaluate", "analyze", "recommend", "search", "schedule",
	"generate", "write", "create", "post", "send", "compare", "rank", "summarize", "extract",
	"parse", "benchmark",
}

var recruitingObjectTerms = []string{
	"简历", "候选人", "候选", "人才", "面试", "offer", "jd", "职位", "岗位", "招聘", "薪资", "背调",
	"面评", "评价", "申请", "投递", "人才库", "面试官", "录用", "入职", "职级", "岗位要求",
	"resume", "candidate", "talent", "interview", "job", "position", "recruiting", "recruitment",
	"salary", "compensation", "application", "applicant", "hiring",
}

var recruitingStrongPhrases = []string{
	"筛简历", "筛选简历", "约面试", "安排面试", "发offer", "发 offer", "生成jd", "生成 jd",
	"候选人匹配", "简历匹配", "薪资分析", "候选人评估", "面试安排",
	"screen resume", "screen resumes", "candidate match", "match candidate", "schedule interview",
	"send offer", "generate jd", "salary benchmark", "salary analysis",
}

var contextualContinuationTerms = []string{
	"继续", "再来", "上一条", "刚才", "这个", "这些", "按这个", "按刚才", "重新", "再", "继续按",
	"continue", "again", "same", "previous", "that", "those",
}

func evaluateAgentSkillAutoEligibility(question string) AgentSkillEligibility {
	normalized := normalizeAgentSkillQuery(question)
	if normalized == "" {
		return AgentSkillEligibility{Reason: "short_low_information", QueryClass: "empty"}
	}
	if hasAnyTerm(normalized, recruitingStrongPhrases) {
		return AgentSkillEligibility{Allowed: true, Reason: "business_action", QueryClass: "strong_phrase"}
	}

	hasAction := hasAnyTerm(normalized, recruitingActionTerms)
	hasObject := hasAnyTerm(normalized, recruitingObjectTerms)
	hasContinuation := hasAnyTerm(normalized, contextualContinuationTerms)
	if hasAction && hasObject {
		return AgentSkillEligibility{Allowed: true, Reason: "business_action", QueryClass: "action_object"}
	}
	if hasContinuation && hasAction {
		return AgentSkillEligibility{Allowed: true, Reason: "contextual_followup", QueryClass: "continuation_action"}
	}
	if isLowInformationAgentSkillQuery(normalized) {
		return AgentSkillEligibility{Reason: "short_low_information", QueryClass: "low_information"}
	}
	if hasObject && meaningfulRuneCount(normalized) >= 8 {
		return AgentSkillEligibility{Allowed: true, Reason: "business_signal", QueryClass: "business_object"}
	}
	return AgentSkillEligibility{Reason: "no_recruiting_intent", QueryClass: "out_of_domain"}
}

func allowSemanticOnlyAgentSkillCandidate(question string, vectorScore float64) bool {
	if vectorScore < skillSemanticOnlyVectorGate {
		return false
	}
	eligibility := evaluateAgentSkillAutoEligibility(question)
	if eligibility.Allowed {
		return true
	}
	return false
}

func normalizeAgentSkillQuery(question string) string {
	return strings.ToLower(strings.TrimSpace(question))
}

func hasAnyTerm(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func isLowInformationAgentSkillQuery(text string) bool {
	count := meaningfulRuneCount(text)
	if count == 0 {
		return true
	}
	if count <= 4 {
		return true
	}
	fields := strings.Fields(text)
	if isMostlyASCII(text) && len(fields) > 0 && len(fields) <= 2 && count <= 12 {
		return true
	}
	return false
}

func isMostlyASCII(text string) bool {
	total := 0
	ascii := 0
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		total++
		if r <= unicode.MaxASCII {
			ascii++
		}
	}
	return total > 0 && ascii*2 >= total
}

func meaningfulRuneCount(text string) int {
	count := 0
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		count++
	}
	return count
}
