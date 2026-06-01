package service

import (
	"strings"
	"testing"

	"logic-grpc-service/ai"
	"logic-grpc-service/model"
)

func TestBuildToolCallingMessagesNoToolsForGreeting(t *testing.T) {
	actx := &AgentContext{
		HrID:      1,
		SessionID: 100,
	}
	messages := buildToolCallingMessages(actx, "你好")
	if len(messages) == 0 {
		t.Fatal("expected at least a system message")
	}
	sys := messages[0].Content
	if !strings.Contains(sys, "如果用户只是问候、感谢、询问你能做什么、请求使用说明，不要调用工具，直接简洁回答") {
		t.Error("system prompt should instruct model not to call tools for greetings/thanks/help")
	}
	if !strings.Contains(sys, "当前用户消息优先级最高") {
		t.Error("system prompt should prioritize current user message")
	}
	if !strings.Contains(sys, "Markdown 输出硬性规范") || !strings.Contains(sys, "禁止写成 \"-内容\"") {
		t.Error("system prompt should include strict standard markdown formatting rules")
	}
}

func TestCandidateSystemPromptIncludesStandardMarkdownRules(t *testing.T) {
	if !strings.Contains(candidateSystemPrompt, "Markdown 输出硬性规范") {
		t.Fatal("candidate system prompt should include strict standard markdown formatting rules")
	}
	if !strings.Contains(candidateSystemPrompt, "2026-05-25 **淘汰**") {
		t.Error("candidate system prompt should include an example for bold text adjacent to dates")
	}
	if strings.Contains(candidateSystemPrompt, "[“问题1”") {
		t.Error("candidate suggested-question JSON example should use standard ASCII JSON quotes")
	}
}

func TestBuildToolCallingMessagesStatusChangeMustUseTool(t *testing.T) {
	actx := &AgentContext{
		HrID:      1,
		SessionID: 100,
	}
	messages := buildToolCallingMessages(actx, "把这个候选人淘汰")
	if len(messages) == 0 {
		t.Fatal("expected at least a system message")
	}
	sys := messages[0].Content
	if !strings.Contains(sys, "propose_application_status_update") {
		t.Error("system prompt should instruct model to call propose_application_status_update for status changes")
	}
	if !strings.Contains(sys, "不要声称已经更新") {
		t.Error("system prompt should forbid claiming status was updated")
	}
}

func TestBuildToolCallingMessagesDataQueryMustUseTool(t *testing.T) {
	actx := &AgentContext{
		HrID:      1,
		SessionID: 100,
	}
	messages := buildToolCallingMessages(actx, "今天有多少投递")
	if len(messages) == 0 {
		t.Fatal("expected at least a system message")
	}
	sys := messages[0].Content
	if !strings.Contains(sys, "必须调用最匹配的工具") {
		t.Error("system prompt should instruct model to call tools for recruiting data queries")
	}
}

func TestBuildToolCallingMessagesMissingParamsAskUser(t *testing.T) {
	actx := &AgentContext{
		HrID:      1,
		SessionID: 100,
	}
	messages := buildToolCallingMessages(actx, "帮我查一下")
	if len(messages) == 0 {
		t.Fatal("expected at least a system message")
	}
	sys := messages[0].Content
	if !strings.Contains(sys, "如果缺少必要参数，不要猜测，应直接追问用户补充") {
		t.Error("system prompt should instruct model to ask user when params are missing")
	}
}

func TestBuildToolCallingMessagesApplicationContext(t *testing.T) {
	actx := &AgentContext{
		HrID:          1,
		SessionID:     100,
		ApplicationID: 42,
	}
	messages := buildToolCallingMessages(actx, "分析简历")
	if len(messages) == 0 {
		t.Fatal("expected at least a system message")
	}
	sys := messages[0].Content
	if !strings.Contains(sys, "投递记录 ID 是 42") {
		t.Error("system prompt should include bound application ID")
	}
	if !strings.Contains(sys, "get_candidate_detail") {
		t.Error("system prompt should instruct model to call get_candidate_detail for resume analysis")
	}
}

func TestBuildToolCallingMessagesIncludesRecentHistory(t *testing.T) {
	actx := &AgentContext{
		HrID:      1,
		SessionID: 100,
		RecentMessages: []model.AIChatHistory{
			{Role: "user", Content: "你好"},
			{Role: "assistant", Content: "你好，有什么可以帮你的？"},
		},
	}
	messages := buildToolCallingMessages(actx, "有哪些岗位")
	if len(messages) < 3 {
		t.Fatalf("expected at least 3 messages (system + 2 history), got %d", len(messages))
	}
}

func TestRecruitingToolsProposeStatusUpdateCoversAllPhrases(t *testing.T) {
	tools := ai.RecruitingTools()
	var found *struct{ Desc string }
	for _, tool := range tools {
		if tool.Name == "propose_application_status_update" {
			found = &struct{ Desc string }{Desc: tool.Desc}
			break
		}
	}
	if found == nil {
		t.Fatal("propose_application_status_update tool not found")
	}
	requiredPhrases := []string{"通过", "淘汰", "拒绝", "录用", "进入下一轮"}
	for _, phrase := range requiredPhrases {
		if !strings.Contains(found.Desc, phrase) {
			t.Errorf("propose_application_status_update description should contain %q", phrase)
		}
	}
	if !strings.Contains(found.Desc, "待确认动作") {
		t.Error("propose_application_status_update description should mention 待确认动作")
	}
	if !strings.Contains(found.Desc, "不会直接修改数据库") {
		t.Error("propose_application_status_update description should clarify it does not modify database")
	}
}

func TestRecruitingToolsCandidateDetailRequiresCall(t *testing.T) {
	tools := ai.RecruitingTools()
	var found *struct{ Desc string }
	for _, tool := range tools {
		if tool.Name == "get_candidate_detail" {
			found = &struct{ Desc string }{Desc: tool.Desc}
			break
		}
	}
	if found == nil {
		t.Fatal("get_candidate_detail tool not found")
	}
	if !strings.Contains(found.Desc, "必须调用") || !strings.Contains(found.Desc, "resume_text") {
		t.Error("get_candidate_detail description should emphasize mandatory call and resume_text")
	}
}

func TestCandidateSuggestedQuestionsReturnsFixedFallback(t *testing.T) {
	questions := candidateSuggestedQuestions("", "")
	if len(questions) != 3 {
		t.Fatalf("expected 3 fallback questions, got %d: %v", len(questions), questions)
	}
	for i, q := range questions {
		if q == "" {
			t.Errorf("question %d is empty", i)
		}
	}
	expected := []string{"我目前的应聘进度？", "根据简历推荐岗位", "帮我优化简历建议"}
	for i, q := range questions {
		if q != expected[i] {
			t.Errorf("question[%d] = %q, want %q", i, q, expected[i])
		}
	}
}

func TestCandidateSuggestedQuestionsIgnoresInput(t *testing.T) {
	// Regardless of input, the fallback must return the same 3 fixed questions.
	questions1 := candidateSuggestedQuestions("上传简历失败怎么办", "你还没有上传简历")
	questions2 := candidateSuggestedQuestions("帮我推荐岗位", "以下是根据你的简历推荐")
	questions3 := candidateSuggestedQuestions("", "")
	if len(questions1) != 3 || len(questions2) != 3 || len(questions3) != 3 {
		t.Fatal("all calls must return exactly 3 questions")
	}
	for i := 0; i < 3; i++ {
		if questions1[i] != questions2[i] || questions2[i] != questions3[i] {
			t.Fatal("all calls must return identical questions regardless of input")
		}
	}
}

func TestRecruitingToolsAllRequiredToolsPresent(t *testing.T) {
	tools := ai.RecruitingTools()
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	required := []string{
		"query_total_applications",
		"query_today_applications",
		"get_job_heat_ranking",
		"search_candidates",
		"get_job_detail",
		"search_jobs",
		"get_candidate_detail",
		"propose_application_status_update",
		"list_all_applications",
		"list_applications_by_job",
		"list_applications_by_status",
		"get_application_status_summary",
		"get_application_trend",
		"get_job_list",
	}
	for _, name := range required {
		if !toolNames[name] {
			t.Errorf("required tool %q is missing from RecruitingTools", name)
		}
	}
}
