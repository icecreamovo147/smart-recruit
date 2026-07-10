package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/ai"
	"logic-grpc-service/config"
	"logic-grpc-service/model"
	"logic-grpc-service/pkg/crypto"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
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

func TestMaybeRequestAgentSkillSelectionEmitsPayload(t *testing.T) {
	skills := []selectedAgentSkill{
		{ID: 11, Name: "match", DisplayName: "Match", Reason: "hybrid score", FinalRankScore: 0.9},
		{ID: 12, Name: "risk", DisplayName: "Risk", Reason: "hybrid score", FinalRankScore: 0.7},
	}
	var got *pb.AgentSkillSelection
	err := maybeRequestAgentSkillSelection(context.Background(), &pb.ChatRequest{Message: "candidate match"}, skills, 99, func(selection *pb.AgentSkillSelection) error {
		got = selection
		return nil
	})
	if !errors.Is(err, errAgentSkillSelectionRequired) {
		t.Fatalf("err = %v, want errAgentSkillSelectionRequired", err)
	}
	if got == nil || !got.Required || got.Reason != "multiple_auto_candidates" {
		t.Fatalf("selection payload = %+v, want required multiple_auto_candidates", got)
	}
	if len(got.Candidates) != 2 || len(got.RecommendedAgentSkillIds) != 1 || got.RecommendedAgentSkillIds[0] != 11 {
		t.Fatalf("selection payload candidates/recommended = %+v", got)
	}
	if !got.Candidates[0].Recommended {
		t.Fatalf("first candidate should be recommended: %+v", got.Candidates[0])
	}
	if got.UserMessageId != 99 {
		t.Fatalf("UserMessageId = %d, want 99", got.UserMessageId)
	}
}

func TestMaybeRequestAgentSkillSelectionBypassesConfirmedRequest(t *testing.T) {
	called := false
	err := maybeRequestAgentSkillSelection(
		context.Background(),
		&pb.ChatRequest{Message: "candidate match", AgentSkillSelectionConfirmed: true},
		[]selectedAgentSkill{{ID: 1}, {ID: 2}},
		99,
		func(selection *pb.AgentSkillSelection) error {
			called = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if called {
		t.Fatalf("selection callback should not be called for confirmed requests")
	}
}

func TestRequestConfirmedNoAgentSkills(t *testing.T) {
	if !requestConfirmedNoAgentSkills(&pb.ChatRequest{AgentSkillSelectionConfirmed: true}) {
		t.Fatalf("confirmed empty selection should skip automatic Agent Skill selection")
	}
	if requestConfirmedNoAgentSkills(&pb.ChatRequest{AgentSkillSelectionConfirmed: true, AgentSkillIds: []int64{1}}) {
		t.Fatalf("confirmed explicit Skill IDs should not be treated as selecting none")
	}
	if requestConfirmedNoAgentSkills(&pb.ChatRequest{}) {
		t.Fatalf("unconfirmed request should allow automatic Agent Skill selection")
	}
}

func TestUpdateConfirmedUserMessageAgentSkillsIgnoresStaleMessageID(t *testing.T) {
	db := setupAgentRunServiceTestDB(t)
	chats := repository.NewChatRepo(db)
	svc := &AIService{chats: chats}
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 42, Title: "stale confirmation"}
	if err := chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	updated, err := svc.updateConfirmedUserMessageAgentSkills(ctx,
		&pb.ChatRequest{
			HrId:                         42,
			SessionId:                    session.ID,
			Message:                      "有哪些岗位是我所发布的",
			AgentSkillSelectionConfirmed: true,
			AgentSkillSelectionMessageId: 999,
			AgentSkillIds:                nil,
		},
		session,
		nil,
	)
	if err != nil {
		t.Fatalf("updateConfirmedUserMessageAgentSkills returned error: %v", err)
	}
	if updated {
		t.Fatalf("updated = true, want false for stale message ID")
	}
}

func TestUpdateConfirmedUserMessageAgentSkillsUpdatesExistingMessage(t *testing.T) {
	db := setupAgentRunServiceTestDB(t)
	chats := repository.NewChatRepo(db)
	svc := &AIService{chats: chats}
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 42, Title: "confirmation"}
	if err := chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	msg := &model.AIChatHistory{SessionID: session.ID, HrID: 42, Role: "user", Content: "有哪些岗位是我所发布的"}
	if err := chats.Add(ctx, msg); err != nil {
		t.Fatalf("Add message failed: %v", err)
	}

	updated, err := svc.updateConfirmedUserMessageAgentSkills(ctx,
		&pb.ChatRequest{
			HrId:                         42,
			SessionId:                    session.ID,
			Message:                      msg.Content,
			AgentSkillSelectionConfirmed: true,
			AgentSkillSelectionMessageId: msg.ID,
			AgentSkillIds:                []int64{11},
		},
		session,
		[]selectedAgentSkill{{ID: 11, Name: "published_jobs", DisplayName: "我发布的岗位"}},
	)
	if err != nil {
		t.Fatalf("updateConfirmedUserMessageAgentSkills returned error: %v", err)
	}
	if !updated {
		t.Fatalf("updated = false, want true")
	}
	var stored model.AIChatHistory
	if err := db.First(&stored, msg.ID).Error; err != nil {
		t.Fatalf("load message failed: %v", err)
	}
	if stored.AgentSkillIDsJSON != `[11]` {
		t.Fatalf("AgentSkillIDsJSON = %q, want [11]", stored.AgentSkillIDsJSON)
	}
	if stored.AgentSkillNamesJSON != `["我发布的岗位"]` {
		t.Fatalf("AgentSkillNamesJSON = %q", stored.AgentSkillNamesJSON)
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
		"parse_resume_profile",
		"get_resume_profile",
		"evaluate_candidate_match",
		"get_candidate_match_evaluation",
		"compare_candidates_for_job",
	}
	for _, name := range required {
		if !toolNames[name] {
			t.Errorf("required tool %q is missing from RecruitingTools", name)
		}
	}
}

func TestAgentRuntimeConfigDoesNotEnableMCPWithoutBinding(t *testing.T) {
	db := setupAgentRuntimeConfigTestDB(t)
	repo := repository.NewAgentConfigRepo(db)
	ctx := context.Background()

	agent := &model.AgentConfig{
		Name:          "hr_agent_without_mcp",
		DisplayName:   "HR Agent",
		AgentType:     "hr_recruiting_agent",
		MaxIterations: 5,
		IsEnabled:     1,
	}
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := repo.ReplaceCapabilityBindings(ctx, agent.ID, []model.AgentCapabilityBinding{
		{CapabilitySource: "builtin", CapabilityKey: "search_jobs", IsEnabled: 1},
	}); err != nil {
		t.Fatalf("replace bindings: %v", err)
	}

	svc := &AIService{agentConfigRepo: repo}
	cfg := svc.getAgentRuntimeConfig(ctx, "hr_recruiting_agent")
	if !cfg.HasConfig {
		t.Fatal("expected runtime config")
	}
	if len(cfg.MCPCapabilityKeys) != 0 {
		t.Fatalf("expected no MCP capabilities, got %#v", cfg.MCPCapabilityKeys)
	}
	if len(cfg.ToolNames) != 1 || cfg.ToolNames[0] != "search_jobs" {
		t.Fatalf("expected builtin search_jobs, got %#v", cfg.ToolNames)
	}
}

func TestAgentRuntimeConfigEnablesOnlyBoundMCPCapability(t *testing.T) {
	db := setupAgentRuntimeConfigTestDB(t)
	repo := repository.NewAgentConfigRepo(db)
	ctx := context.Background()

	agent := &model.AgentConfig{
		Name:          "hr_agent_with_mcp",
		DisplayName:   "HR Agent",
		AgentType:     "hr_recruiting_agent",
		MaxIterations: 5,
		IsEnabled:     1,
	}
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := repo.ReplaceCapabilityBindings(ctx, agent.ID, []model.AgentCapabilityBinding{
		{CapabilitySource: "builtin", CapabilityKey: "search_jobs", IsEnabled: 1},
		{CapabilitySource: "mcp", CapabilityKey: "7:lookup_candidate", IsEnabled: 1},
	}); err != nil {
		t.Fatalf("replace bindings: %v", err)
	}

	svc := &AIService{agentConfigRepo: repo}
	cfg := svc.getAgentRuntimeConfig(ctx, "hr_recruiting_agent")
	if !cfg.MCPCapabilityKeys["7:lookup_candidate"] {
		t.Fatalf("expected bound MCP capability, got %#v", cfg.MCPCapabilityKeys)
	}
	if cfg.MCPCapabilityKeys["7:unbound_tool"] {
		t.Fatalf("unexpected unbound MCP capability present: %#v", cfg.MCPCapabilityKeys)
	}
}

func TestAgentRuntimeConfigFallsBackToLegacyBuiltinBindings(t *testing.T) {
	db := setupAgentRuntimeConfigTestDB(t)
	repo := repository.NewAgentConfigRepo(db)
	ctx := context.Background()

	agent := &model.AgentConfig{
		Name:          "hr_agent_legacy",
		DisplayName:   "HR Agent",
		AgentType:     "hr_recruiting_agent",
		MaxIterations: 5,
		IsEnabled:     1,
	}
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := db.WithContext(ctx).Create(&model.AgentToolBinding{
		AgentID:   agent.ID,
		ToolName:  "get_job_detail",
		IsEnabled: 1,
	}).Error; err != nil {
		t.Fatalf("create legacy binding: %v", err)
	}

	svc := &AIService{agentConfigRepo: repo}
	cfg := svc.getAgentRuntimeConfig(ctx, "hr_recruiting_agent")
	if len(cfg.ToolNames) != 1 || cfg.ToolNames[0] != "get_job_detail" {
		t.Fatalf("expected legacy builtin binding, got %#v", cfg.ToolNames)
	}
}

func TestAgentRunFinishUsesBackgroundContextAfterCancellation(t *testing.T) {
	db := setupAgentRunServiceTestDB(t)
	repo := repository.NewAgentRunRepo(db)
	run := &model.AgentRun{
		SessionID: 1,
		HrID:      2,
		AgentType: "hr",
		AgentName: "hr_recruiting_agent",
		ModelName: "test-model",
		Status:    agentRunStatusRunning,
		StartedAt: time.Now(),
	}
	if err := repo.CreateRun(context.Background(), run); err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec := &agentRunRecorder{repo: repo, runID: run.ID}
	rec.finish(ctx, agentRunStatusCanceled, "partial", "canceled", context.Canceled.Error())

	runs, err := repo.ListRunsBySession(context.Background(), 2, 1, 10)
	if err != nil {
		t.Fatalf("ListRunsBySession failed: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != agentRunStatusCanceled {
		t.Fatalf("expected canceled run despite canceled ctx, got %+v", runs)
	}
	if runs[0].CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
	steps, err := repo.ListStepsByRunIDs(context.Background(), []uint64{run.ID})
	if err != nil {
		t.Fatalf("ListStepsByRunIDs failed: %v", err)
	}
	if len(steps) != 1 || steps[0].Status != agentRunStatusCanceled {
		t.Fatalf("expected canceled final step, got %+v", steps)
	}
}

func TestChatModelResolutionFailureCreatesFailedAgentRun(t *testing.T) {
	db := setupAgentRunServiceTestDB(t)
	aiClient, err := ai.NewClient(context.Background(), "test-key", "default-model", "")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	svc := &AIService{
		chats:        repository.NewChatRepo(db),
		agentRuns:    repository.NewAgentRunRepo(db),
		ai:           aiClient,
		llmConfigSvc: NewLlmConfigService(repository.NewProviderRepo(db), repository.NewModelConfigRepo(db), crypto.EncryptionKey{}),
	}

	resp, err := svc.Chat(context.Background(), &pb.ChatRequest{HrId: 7, Message: "hello", ModelId: 404})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if resp == nil || !strings.Contains(resp.Msg, "模型不可用") {
		t.Fatalf("expected model unavailable response, got %+v", resp)
	}

	runs, err := svc.agentRuns.ListRunsBySession(context.Background(), 7, resp.SessionId, 10)
	if err != nil {
		t.Fatalf("ListRunsBySession failed: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one failed run, got %+v", runs)
	}
	if runs[0].Status != agentRunStatusFailed || runs[0].ErrorType != "model_resolution_failed" {
		t.Fatalf("unexpected run failure state: %+v", runs[0])
	}
	steps, err := svc.agentRuns.ListStepsByRunIDs(context.Background(), []uint64{runs[0].ID})
	if err != nil {
		t.Fatalf("ListStepsByRunIDs failed: %v", err)
	}
	if len(steps) == 0 || steps[len(steps)-1].Status != agentRunStatusFailed {
		t.Fatalf("expected failed final step, got %+v", steps)
	}
}

func TestBuildSessionContextUsageUsesAccumulatedHistory(t *testing.T) {
	db := setupAgentRunServiceTestDB(t)
	chats := repository.NewChatRepo(db)
	ctx := context.Background()
	session := &model.AIChatSession{HrID: 12, Title: "context test"}
	if err := chats.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	for _, row := range []model.AIChatHistory{
		{SessionID: session.ID, HrID: 12, Role: "user", Content: strings.Repeat("历史问题", 40)},
		{SessionID: session.ID, HrID: 12, Role: "assistant", Content: strings.Repeat("历史回答", 60)},
	} {
		if err := chats.Add(ctx, &row); err != nil {
			t.Fatalf("Add history failed: %v", err)
		}
	}

	svc := &AIService{chats: chats, usageBuilder: NewContextUsageBuilder()}
	actx := &AgentContext{
		SessionSummary:   strings.Repeat("摘要", 30),
		LongTermMemories: []model.AIMemory{{Content: strings.Repeat("记忆", 20)}},
	}
	initial := svc.buildSessionContextUsage(
		ctx,
		12,
		session.ID,
		actx,
		[]*schema.Message{schema.SystemMessage(strings.Repeat("系统提示", 25))},
		nil,
		"test-model",
		"initial",
		"estimator",
		strings.Repeat("当前问题", 10),
		"",
	)
	if initial == nil {
		t.Fatal("expected initial context usage")
	}
	toolStage := svc.buildSessionContextUsage(
		ctx,
		12,
		session.ID,
		actx,
		[]*schema.Message{schema.SystemMessage(strings.Repeat("系统提示", 25))},
		nil,
		"test-model",
		"tool_result",
		"estimator",
		"",
		"",
	)
	if toolStage == nil {
		t.Fatal("expected tool-stage context usage")
	}
	withoutCurrent := initial.PromptTokensEstimated - initial.Breakdown.CurrentMessageTokens
	if toolStage.PromptTokensEstimated < withoutCurrent {
		t.Fatalf("session usage should keep accumulated history across runtime updates: initial=%d without_current=%d tool_stage=%d",
			initial.PromptTokensEstimated, withoutCurrent, toolStage.PromptTokensEstimated)
	}
}

func TestUserMessagePersistFailureFinalizesFailedAgentRun(t *testing.T) {
	db := setupAgentRunServiceTestDB(t)
	aiClient, err := ai.NewClient(context.Background(), "test-key", "default-model", "")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	chats := repository.NewChatRepo(db)
	svc := &AIService{
		chats:     chats,
		summaries: repository.NewSessionSummaryRepo(db),
		memories:  repository.NewMemoryRepo(db),
		agentRuns: repository.NewAgentRunRepo(db),
		ai:        aiClient,
	}
	svc.contextBuilder = NewAgentContextBuilder(chats, svc.summaries, svc.memories, aiClient, zeroConfig(), nil)

	missingSession := &model.AIChatSession{ID: 999, HrID: 3}
	_, _, err = svc.runToolCallingChat(context.Background(),
		&pb.ChatRequest{HrId: 3, SessionId: missingSession.ID, Message: "persist me"},
		missingSession, nil, "default-model", nil, nil, nil, aiClient)
	if err == nil {
		t.Fatal("expected user message persist failure")
	}

	runs, err := svc.agentRuns.ListRunsBySession(context.Background(), 3, missingSession.ID, 10)
	if err != nil {
		t.Fatalf("ListRunsBySession failed: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected one run for failed persist, got %+v", runs)
	}
	if runs[0].Status != agentRunStatusFailed || runs[0].ErrorType != "persist_failed" {
		t.Fatalf("expected persist_failed run, got %+v", runs[0])
	}
}

func setupAgentRunServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AIChatSession{},
		&model.AIChatHistory{},
		&model.AISessionSummary{},
		&model.AIMemory{},
		&model.AgentRun{},
		&model.AgentRunStep{},
		&model.LlmProvider{},
		&model.LlmModel{},
	); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}

func zeroConfig() config.Config {
	return config.Config{}
}

func setupAgentRuntimeConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentConfig{}, &model.AgentToolBinding{}, &model.AgentCapabilityBinding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestDesensitizeArgsJSON_Phone(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"11-digit phone", `{"phone": "13812345678"}`},
		{"phone with spaces", `{"phone": "138 1234 5678"}`},
		{"phone with dash", `{"phone": "138-1234-5678"}`},
		{"phone in text", `联系手机号13812345678请查收`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := desensitizeArgsJSON(tc.input)
			if got == tc.input {
				t.Errorf("desensitizeArgsJSON(%q) = %q, want phone masked", tc.input, got)
			}
		})
	}
}

func TestDesensitizeArgsJSON_IDCard(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"18-digit ID", `{"id_card": "110101199001011234"}`},
		{"ID with X suffix", `{"id_card": "11010119900101123X"}`},
		{"ID with spaces", `{"id_card": "110101 19900101 1234"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := desensitizeArgsJSON(tc.input)
			if got == tc.input {
				t.Errorf("desensitizeArgsJSON(%q) = %q, want ID card masked", tc.input, got)
			}
		})
	}
}

func TestDesensitizeArgsJSON_Empty(t *testing.T) {
	if got := desensitizeArgsJSON(""); got != "" {
		t.Errorf("desensitizeArgsJSON(\"\") = %q, want empty", got)
	}
}

func TestDesensitizeArgsJSON_NoPII(t *testing.T) {
	input := `{"tool": "search_jobs", "keyword": "工程师"}`
	if got := desensitizeArgsJSON(input); got != input {
		t.Errorf("desensitizeArgsJSON(%q) = %q, want unchanged", input, got)
	}
}

func TestDesensitizeResultContent_Empty(t *testing.T) {
	if got := desensitizeResultContent(""); got != "" {
		t.Errorf("desensitizeResultContent(\"\") = %q, want empty", got)
	}
}

func TestDesensitizeResultContent_Short(t *testing.T) {
	input := "查询成功，共找到 5 个匹配岗位"
	if got := desensitizeResultContent(input); got != input {
		t.Errorf("desensitizeResultContent(%q) = %q, want unchanged", input, got)
	}
}

func TestDesensitizeResultContent_Truncate(t *testing.T) {
	input := make([]rune, 2500)
	for i := range input {
		input[i] = 'x'
	}
	got := desensitizeResultContent(string(input))
	if len([]rune(got)) > 2100 {
		t.Errorf("desensitizeResultContent truncated result too long: %d chars", len([]rune(got)))
	}
	if !strings.Contains(got, "已截断") {
		t.Error("desensitizeResultContent should include truncation notice")
	}
}

func TestDesensitizeResultContent_PhoneMasked(t *testing.T) {
	input := `{"result": "联系手机: 13812345678, 身份证: 110101199001011234"}`
	got := desensitizeResultContent(input)
	if strings.Contains(got, "13812345678") {
		t.Error("desensitizeResultContent should mask phone number")
	}
	if strings.Contains(got, "110101199001011234") {
		t.Error("desensitizeResultContent should mask ID card")
	}
}

func TestDesensitizeResultContent_AtBoundary(t *testing.T) {
	input := make([]rune, 2000)
	for i := range input {
		input[i] = 'a'
	}
	got := desensitizeResultContent(string(input))
	if strings.Contains(got, "已截断") {
		t.Error("desensitizeResultContent should not truncate at exactly 2000 chars")
	}
	if len([]rune(got)) != 2000 {
		t.Errorf("desensitizeResultContent should keep 2000 chars unchanged, got %d", len([]rune(got)))
	}
}

func TestBuildProviderFinalContextUsage_ProviderAvailable(t *testing.T) {
	svc := &AIService{usageBuilder: NewContextUsageBuilder()}

	estimated := &pb.ContextUsageInfo{
		ModelId:               1,
		ModelName:             "test-model",
		ContextWindowTokens:   128000,
		MaxOutputTokens:       4096,
		PromptTokensEstimated: 5000,
		Breakdown:             &pb.ContextUsageBreakdown{SystemPromptTokens: 100},
		Source:                "estimator",
		Estimated:             true,
		Stage:                 "final",
	}

	usage := &schema.TokenUsage{PromptTokens: 3200, CompletionTokens: 800, TotalTokens: 4000}
	result := svc.buildProviderFinalContextUsage(context.Background(), estimated, usage)

	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Source != "provider" {
		t.Errorf("source = %q, want provider", result.Source)
	}
	if result.Estimated {
		t.Error("estimated should be false when provider usage is available")
	}
	if result.PromptTokensActual != 3200 {
		t.Errorf("prompt_tokens_actual = %d, want 3200", result.PromptTokensActual)
	}
	if result.CompletionTokensActual != 800 {
		t.Errorf("completion_tokens_actual = %d, want 800", result.CompletionTokensActual)
	}
	if result.TotalTokensActual != 4000 {
		t.Errorf("total_tokens_actual = %d, want 4000", result.TotalTokensActual)
	}
	if result.Stage != "final" {
		t.Errorf("stage = %q, want final", result.Stage)
	}
	// Model metadata should be preserved from estimated snapshot
	if result.ModelId != 1 || result.ModelName != "test-model" {
		t.Error("model metadata should be preserved from estimated snapshot")
	}
	// Breakdown should remain from estimated snapshot
	if result.Breakdown == nil || result.Breakdown.SystemPromptTokens != 100 {
		t.Error("breakdown should be preserved from estimated snapshot")
	}
	// Estimated prompt tokens should remain as reference
	if result.PromptTokensEstimated != 5000 {
		t.Errorf("prompt_tokens_estimated should remain as reference = %d", result.PromptTokensEstimated)
	}
}

func TestBuildProviderFinalContextUsage_ProviderMissing(t *testing.T) {
	svc := &AIService{usageBuilder: NewContextUsageBuilder()}

	estimated := &pb.ContextUsageInfo{
		ModelId:               1,
		ModelName:             "test-model",
		ContextWindowTokens:   128000,
		MaxOutputTokens:       4096,
		PromptTokensEstimated: 5000,
		Breakdown:             &pb.ContextUsageBreakdown{},
		Source:                "estimator",
		Estimated:             true,
		Stage:                 "tool_result",
	}

	result := svc.buildProviderFinalContextUsage(context.Background(), estimated, nil)

	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Source != "estimator" {
		t.Errorf("source = %q, want estimator when provider missing", result.Source)
	}
	if !result.Estimated {
		t.Error("estimated should be true when provider usage is missing")
	}
	if result.PromptTokensActual != 0 {
		t.Errorf("prompt_tokens_actual should be 0 when provider is missing, got %d", result.PromptTokensActual)
	}
	if result.Stage != "final" {
		t.Errorf("stage = %q, want final", result.Stage)
	}
}

func TestBuildProviderFinalContextUsage_ProviderWithZeroPromptTokens(t *testing.T) {
	svc := &AIService{usageBuilder: NewContextUsageBuilder()}

	estimated := &pb.ContextUsageInfo{
		ModelId:               1,
		ModelName:             "test-model",
		ContextWindowTokens:   128000,
		PromptTokensEstimated: 5000,
		Breakdown:             &pb.ContextUsageBreakdown{},
		Source:                "estimator",
		Estimated:             true,
		Stage:                 "tool_result",
	}

	result := svc.buildProviderFinalContextUsage(context.Background(), estimated, &schema.TokenUsage{PromptTokens: 0, CompletionTokens: 0, TotalTokens: 0})

	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Source != "estimator" {
		t.Errorf("source = %q, want estimator when prompt_tokens is 0", result.Source)
	}
	if !result.Estimated {
		t.Error("estimated should be true when prompt_tokens is 0")
	}
}

func TestBuildProviderFinalContextUsage_RemainingAndRatioUseSessionEstimate(t *testing.T) {
	svc := &AIService{usageBuilder: NewContextUsageBuilder()}

	estimated := &pb.ContextUsageInfo{
		ModelId:                  1,
		ModelName:                "test-model",
		ContextWindowTokens:      128000,
		MaxOutputTokens:          4096,
		PromptTokensEstimated:    100000,
		RemainingTokensEstimated: 23904,
		UsageRatio:               0.78125,
		Breakdown:                &pb.ContextUsageBreakdown{},
		Source:                   "estimator",
		Estimated:                true,
		Stage:                    "final",
	}

	usage := &schema.TokenUsage{PromptTokens: 32000, CompletionTokens: 8000, TotalTokens: 40000}
	result := svc.buildProviderFinalContextUsage(context.Background(), estimated, usage)

	if result.RemainingTokensEstimated != estimated.RemainingTokensEstimated {
		t.Errorf("remaining = %d, want %d", result.RemainingTokensEstimated, estimated.RemainingTokensEstimated)
	}
	if result.UsageRatio != estimated.UsageRatio {
		t.Errorf("ratio = %f, want %f", result.UsageRatio, estimated.UsageRatio)
	}
}

func TestBuildProviderFinalContextUsage_NilEstimated(t *testing.T) {
	svc := &AIService{usageBuilder: NewContextUsageBuilder()}
	result := svc.buildProviderFinalContextUsage(context.Background(), nil, &schema.TokenUsage{PromptTokens: 100})
	if result != nil {
		t.Error("result should be nil when estimated is nil")
	}
}

func TestBuildProviderFinalContextUsage_NoContextWindow(t *testing.T) {
	svc := &AIService{usageBuilder: NewContextUsageBuilder()}

	estimated := &pb.ContextUsageInfo{
		PromptTokensEstimated: 5000,
		Breakdown:             &pb.ContextUsageBreakdown{},
		Source:                "estimator",
		Estimated:             true,
		Stage:                 "final",
	}

	usage := &schema.TokenUsage{PromptTokens: 3200, CompletionTokens: 800, TotalTokens: 4000}
	result := svc.buildProviderFinalContextUsage(context.Background(), estimated, usage)

	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.RemainingTokensEstimated != 0 {
		t.Errorf("remaining should be 0 when context_window is 0, got %d", result.RemainingTokensEstimated)
	}
	if result.UsageRatio != 0 {
		t.Errorf("ratio should be 0 when context_window is 0, got %f", result.UsageRatio)
	}
}
