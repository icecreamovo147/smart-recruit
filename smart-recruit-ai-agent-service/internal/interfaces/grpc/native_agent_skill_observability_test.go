package grpc

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	domainagentskill "smart-recruit-ai-agent-service/internal/domain/agentskill"
	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
	commonsai "smart-recruit-commons/ai"
	"smart-recruit-platform-go/observability"
	"smart-recruit-proto/recruitment/pb"
)

func TestAgentSkillFeatureOffOmitsPackagesAndKeepsPrivacySafeEvidence(t *testing.T) {
	store := newFakeAIStore()
	store.agentConfigs = []*pb.AgentConfigInfo{{
		Id: 10, Name: "hr", AgentType: hrRecruitingAgentType, PromptTemplateId: 20, IsDefault: true, IsEnabled: true,
	}}
	store.promptByID[20] = &pb.PromptTemplateInfo{
		Id: 20, AgentType: hrRecruitingAgentType, PromptRole: hrRuntimePromptRoleSystem, IsActive: true, Content: "system",
	}
	store.agentSkillVersionDocs = []embeddinginfra.AgentSkillVersionEmbeddingDocument{
		runtimeSkillVersionDocument(101, 1, domainagentskill.CompositionRolePrimary, domainagentskill.RiskLevelLow, "PRIVATE_SKILL_BODY"),
	}
	metrics := observability.NewRegistry("ai-agent")
	service := newNativeAIService(store, nil, nil, nil, nil)
	service.metrics = metrics
	service.skillPackageV2Enabled = false

	model := runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{})
	model.ConfigurationRefs.AgentIDs = []int64{10}
	model.ConfigurationRefs.PromptTemplateIDs = []int64{20}
	runtime, err := service.loadHRRuntimeGovernanceForAgentCore(
		context.Background(),
		&pb.ChatRequest{HrId: 77, Message: "PRIVATE_USER_MESSAGE"},
		0,
		false,
		model,
		true,
	)
	if err != nil {
		t.Fatalf("load governance returned error: %v", err)
	}
	if len(runtime.SelectedAgentSkills) != 0 || runtime.AgentSkillConfirmationRequired {
		t.Fatalf("feature-off runtime loaded Skill: %#v", runtime.SelectedAgentSkills)
	}
	for _, governanceError := range runtime.GovernanceErrors {
		if governanceError.Source == "agent_skill" {
			t.Fatalf("feature-off Skill must not fail the request, errors=%#v", runtime.GovernanceErrors)
		}
	}
	if len(runtime.AgentSkillRuntimeEvidence) != 1 ||
		runtime.AgentSkillRuntimeEvidence[0].GetVersionId() != 101 ||
		runtime.AgentSkillRuntimeEvidence[0].GetDecisionReason() != "skill_v2_disabled" {
		t.Fatalf("feature-off evidence = %#v", runtime.AgentSkillRuntimeEvidence)
	}
	process := buildHRProcessContent(nil, nil, false, runtime, commonsai.RecruitingPlan{}, "adk", nil)
	for _, forbidden := range []string{"PRIVATE_SKILL_BODY", "PRIVATE_USER_MESSAGE"} {
		if strings.Contains(process, forbidden) {
			t.Fatalf("feature-off evidence leaked %q: %s", forbidden, process)
		}
	}
	output := metrics.Prometheus()
	if strings.Count(output, "smart_recruit_agent_skill_retrieval_duration_seconds_count{") != 1 ||
		!strings.Contains(output, `result="disabled",reason="skill_v2_disabled"`) {
		t.Fatalf("feature-off metrics were not emitted exactly once:\n%s", output)
	}
}

func TestAgentSkillFeatureOffStillHandlesReleasedChatWithoutLegacyFallback(t *testing.T) {
	base := newFakeAIStore()
	base.agentConfigs = []*pb.AgentConfigInfo{{
		Id: 10, Name: "hr", AgentType: hrRecruitingAgentType, PromptTemplateId: 20, IsDefault: true, IsEnabled: true,
	}}
	base.promptByID[20] = &pb.PromptTemplateInfo{
		Id: 20, AgentType: hrRecruitingAgentType, PromptRole: hrRuntimePromptRoleSystem, IsActive: true, Content: "system",
	}
	document := runtimeSkillVersionDocument(
		101,
		1,
		domainagentskill.CompositionRolePrimary,
		domainagentskill.RiskLevelLow,
		"PRIVATE_SKILL_BODY",
	)
	base.agentSkillPackages = []embeddinginfra.AgentSkillRuntimePackage{
		runtimeSkillPackage(t, document, nil),
	}
	store := &releasedSkillFakeStore{
		fakeAIStore: base,
		resolution:  runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
	}
	provider := &fakeChatProvider{reply: "safe reply"}
	service := newNativeAIService(store, provider, nil, nil, nil)
	service.skillPackageV2Enabled = false
	service.metrics = observability.NewRegistry("ai-agent")

	response, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "hello"})
	if err != nil {
		t.Fatalf("feature-off Chat returned error: %v", err)
	}
	if response.GetReply() != "safe reply" || provider.calls != 1 {
		t.Fatalf("feature-off Chat response=%#v provider calls=%d", response, provider.calls)
	}
	if len(response.GetAgentSkillRuntimeEvidence()) != 1 ||
		response.GetAgentSkillRuntimeEvidence()[0].GetDecisionReason() != "skill_v2_disabled" ||
		response.GetAgentSkillRuntimeEvidence()[0].GetIncluded() {
		t.Fatalf("feature-off Chat evidence=%#v", response.GetAgentSkillRuntimeEvidence())
	}
	if len(provider.prompts) != 1 || strings.Contains(provider.prompts[0], "PRIVATE_SKILL_BODY") {
		t.Fatalf("feature-off runtime fell back to Skill content: %#v", provider.prompts)
	}
}

func TestRecordAgentSkillRuntimeDecisionEmitsEachDecisionOnce(t *testing.T) {
	metrics := observability.NewRegistry("ai-agent")
	service := &nativeAIService{metrics: metrics}
	evidence := []*pb.AgentSkillRuntimeEvidence{{
		SkillId: 9, VersionId: 101, Version: "2.0.0", CompiledHash: strings.Repeat("a", 64),
		CompositionRole: pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY,
		Risk:            pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_LOW,
		SelectionMode:   "auto", LoadedTokens: 42, Included: true, DecisionReason: "core_included",
		Sections: []*pb.AgentSkillSectionRuntimeEvidence{{
			SectionId: 1001, SectionKey: "details", ContentHash: strings.Repeat("b", 64),
			EstimatedTokens: 99, Included: false, DecisionReason: "section_budget_exceeded",
		}},
	}}
	service.recordAgentSkillRuntimeDecision(
		withAgentSkillExecutionMode(context.Background(), true),
		"v2",
		"auto",
		evidence,
		nil,
		20*time.Millisecond,
	)

	output := metrics.Prometheus()
	if strings.Count(output, "smart_recruit_agent_skill_selection_total{") != 1 {
		t.Fatalf("selection decision count/series is not exactly one:\n%s", output)
	}
	if !strings.Contains(output, `smart_recruit_agent_skill_tokens_count{runtime_mode="v2",execution_mode="durable",selection_mode="auto",role="primary",risk="low",result="included",reason="core_included"} 1`) {
		t.Fatalf("token observation missing:\n%s", output)
	}
	if !strings.Contains(output, `smart_recruit_agent_skill_budget_drop_total{runtime_mode="v2",execution_mode="durable",selection_mode="auto",role="primary",risk="low",result="dropped",reason="section_budget_exceeded"} 1`) {
		t.Fatalf("section budget drop missing:\n%s", output)
	}
	if strings.Contains(output, "2.0.0") || strings.Contains(output, strings.Repeat("a", 64)) || strings.Contains(output, "details") {
		t.Fatalf("metric labels leaked version/hash/section identity:\n%s", output)
	}
}

func TestRecordAgentSkillRuntimeDecisionExposesBelowGateReason(t *testing.T) {
	metrics := observability.NewRegistry("ai-agent")
	service := &nativeAIService{metrics: metrics}
	service.recordAgentSkillRuntimeDecision(
		context.Background(),
		"v2",
		"none",
		[]*pb.AgentSkillRuntimeEvidence{{
			SkillId: 9, VersionId: 101,
			CompositionRole: pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY,
			Risk:            pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_LOW,
			Included:        false,
			DecisionReason:  "below_relevance_gate",
			RelevanceScore:  0.12,
		}},
		nil,
		10*time.Millisecond,
	)

	output := metrics.Prometheus()
	if !strings.Contains(
		output,
		`smart_recruit_agent_skill_selection_total{runtime_mode="v2",execution_mode="direct",selection_mode="none",role="primary",risk="low",result="dropped",reason="below_relevance_gate"} 1`,
	) {
		t.Fatalf("below-gate metric missing:\n%s", output)
	}
}

func TestAgentSkillPreflightMetricsAreSuppressedBeforeDurableExecution(t *testing.T) {
	metrics := observability.NewRegistry("ai-agent")
	service := &nativeAIService{metrics: metrics}
	evidence := []*pb.AgentSkillRuntimeEvidence{{
		CompositionRole: pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY,
		Risk:            pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_LOW,
		LoadedTokens:    42,
		Included:        true,
		DecisionReason:  "core_included",
	}}
	service.recordAgentSkillRuntimeDecision(
		withoutAgentSkillMetrics(context.Background()),
		"v2",
		"auto",
		evidence,
		nil,
		time.Millisecond,
	)
	service.recordAgentSkillRuntimeDecision(
		withAgentSkillExecutionMode(context.Background(), true),
		"v2",
		"auto",
		evidence,
		nil,
		time.Millisecond,
	)
	output := metrics.Prometheus()
	if strings.Count(output, "smart_recruit_agent_skill_selection_total{") != 1 ||
		!strings.Contains(output, `execution_mode="durable"`) {
		t.Fatalf("preflight duplicated durable execution metrics:\n%s", output)
	}
}

func TestAgentSkillJudgeIsNonBlockingBoundedAndRedacted(t *testing.T) {
	metrics := observability.NewRegistry("ai-agent")
	block := make(chan struct{})
	runner := &capturingAgentSkillJudgeRunner{block: block, called: make(chan struct{}, 1)}
	dispatcher := newAgentSkillJudgeDispatcher(true, runner, metrics)
	t.Cleanup(dispatcher.Close)
	input := AgentSkillJudgeInput{
		RedactedResponse: agentSkillJudgeResponseProfile(
			"候选人 Alice 和 Bob 应聘 Senior Go Engineer，邮箱 alice@example.invalid，电话 13800138000，期望月薪 25000 元。",
		),
		EvaluationCriteria: []string{"回答准确"},
	}
	started := time.Now()
	dispatcher.TrySubmit(input, observability.AgentSkillMetricLabels{
		RuntimeMode: "v2", ExecutionMode: "durable", SelectionMode: "auto",
		Role: "none", Risk: "unknown",
	})
	if time.Since(started) > 100*time.Millisecond {
		t.Fatal("judge submission blocked the request")
	}
	select {
	case <-runner.called:
	case <-time.After(time.Second):
		t.Fatal("judge worker did not receive queued input")
	}
	runner.mu.Lock()
	captured := runner.input
	runner.mu.Unlock()
	for _, forbidden := range []string{"Alice", "Bob", "Senior Go Engineer", "alice@example.invalid", "13800138000", "25000"} {
		if strings.Contains(captured.RedactedResponse, forbidden) {
			t.Fatalf("judge input leaked %q: %#v", forbidden, captured)
		}
	}
	var profile map[string]any
	if err := json.Unmarshal([]byte(captured.RedactedResponse), &profile); err != nil {
		t.Fatalf("judge response profile is invalid JSON: %v", err)
	}
	if profile["schema_version"] != float64(1) ||
		profile["response_present"] != true ||
		profile["format"] != "text" {
		t.Fatalf("judge response profile = %#v", profile)
	}
	close(block)
}

func TestAgentSkillJudgeResponseProfileContainsNoReversibleRecruitingText(t *testing.T) {
	response := strings.Join([]string{
		"候选人张三应聘高级后端工程师，现居上海浦东。",
		"候选人李四应聘数据科学家，期望月薪 35000 元。",
		"联系方式 alice@example.invalid / 13800138000。",
		"身份证 110101199001011234，银行卡 6222 0212 3456 7890 123。",
	}, "\n")
	got := agentSkillJudgeResponseProfile(response)
	for _, forbidden := range []string{
		"张三", "李四", "高级后端工程师", "数据科学家", "上海浦东",
		"35000", "alice@example.invalid", "13800138000",
		"110101199001011234", "6222 0212 3456 7890 123",
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("response profile leaked %q: %s", forbidden, got)
		}
	}
	var profile map[string]any
	if err := json.Unmarshal([]byte(got), &profile); err != nil {
		t.Fatalf("response profile is invalid JSON: %v", err)
	}
	allowed := map[string]struct{}{
		"schema_version": {}, "response_present": {}, "bounded_rune_count": {},
		"bounded_line_count": {}, "truncated": {}, "format": {},
	}
	for key := range profile {
		if _, ok := allowed[key]; !ok {
			t.Fatalf("response profile contains non-allowlisted field %q: %#v", key, profile)
		}
	}
	if profile["bounded_line_count"] != float64(4) {
		t.Fatalf("bounded_line_count = %#v, want 4", profile["bounded_line_count"])
	}
}

func TestAgentSkillJudgeResponseProfileBoundsStructuralSignals(t *testing.T) {
	response := strings.Repeat("界\n", agentSkillJudgeMaxResponseRunes+100)
	var profile map[string]any
	if err := json.Unmarshal([]byte(agentSkillJudgeResponseProfile(response)), &profile); err != nil {
		t.Fatalf("response profile is invalid JSON: %v", err)
	}
	if profile["bounded_rune_count"] != float64(agentSkillJudgeMaxResponseRunes) ||
		profile["bounded_line_count"] != float64(agentSkillJudgeMaxResponseLines) ||
		profile["truncated"] != true {
		t.Fatalf("unbounded response profile: %#v", profile)
	}
}

func TestTrySubmitAgentSkillJudgeBuildsPrivacySafeInput(t *testing.T) {
	metrics := observability.NewRegistry("ai-agent")
	runner := &capturingAgentSkillJudgeRunner{called: make(chan struct{}, 1)}
	dispatcher := newAgentSkillJudgeDispatcher(true, runner, metrics)
	t.Cleanup(dispatcher.Close)
	service := &nativeAIService{metrics: metrics, agentSkillJudge: dispatcher}
	governance := hrRuntimeGovernanceContext{
		AgentSkillSelectionMode: "auto",
		SelectedAgentSkills: []hrRuntimeAgentSkill{{
			Included: true,
			EvaluationCriteria: []string{
				"不得提及候选人 Alice 或岗位 Senior Go Engineer",
				"不得包含期望月薪 25000 元",
				"不得包含银行卡 6222 0212 3456 7890 123",
			},
		}},
	}

	service.trySubmitAgentSkillJudge(
		context.Background(),
		"Alice 与 Bob 正在应聘 Senior Go Engineer，期望月薪 25000 元，银行卡 6222 0212 3456 7890 123。",
		"Alice",
		"Senior Go Engineer",
		governance,
	)
	select {
	case <-runner.called:
	case <-time.After(time.Second):
		t.Fatal("judge worker did not receive privacy-safe input")
	}
	runner.mu.Lock()
	captured := cloneAgentSkillJudgeInput(runner.input)
	runner.mu.Unlock()
	serialized := captured.RedactedResponse + strings.Join(captured.EvaluationCriteria, "\n")
	for _, forbidden := range []string{
		"Alice", "Bob", "Senior Go Engineer", "25000", "6222 0212 3456 7890 123",
	} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("production judge input leaked %q: %#v", forbidden, captured)
		}
	}
	if len(captured.EvaluationCriteria) != 3 {
		t.Fatalf("evaluation criteria = %#v, want 3 bounded values", captured.EvaluationCriteria)
	}
}

func TestAgentSkillJudgeCriteriaRedactsPIIBeforeBoundaryTruncation(t *testing.T) {
	tests := []struct {
		name            string
		prefixRunes     int
		sensitive       string
		sensitiveValues []string
		forbidden       string
	}{
		{
			name:        "email",
			prefixRunes: agentSkillJudgeMaxCriterionRunes - 5,
			sensitive:   "alice@example.invalid",
			forbidden:   "alice",
		},
		{
			name:        "phone",
			prefixRunes: agentSkillJudgeMaxCriterionRunes - 5,
			sensitive:   "13800138000",
			forbidden:   "13800",
		},
		{
			name:        "identifier",
			prefixRunes: agentSkillJudgeMaxCriterionRunes - 6,
			sensitive:   "110101199001011234",
			forbidden:   "110101",
		},
		{
			name:            "candidate name",
			prefixRunes:     agentSkillJudgeMaxCriterionRunes - 2,
			sensitive:       "候选人张三",
			sensitiveValues: []string{"候选人张三"},
			forbidden:       "候选",
		},
		{
			name:            "job title",
			prefixRunes:     agentSkillJudgeMaxCriterionRunes - 4,
			sensitive:       "Senior Platform Engineer",
			sensitiveValues: []string{"Senior Platform Engineer"},
			forbidden:       "Seni",
		},
		{
			name:        "salary",
			prefixRunes: agentSkillJudgeMaxCriterionRunes - 5,
			sensitive:   "期望月薪 25000 元",
			forbidden:   "25000",
		},
		{
			name:        "bank card",
			prefixRunes: agentSkillJudgeMaxCriterionRunes - 5,
			sensitive:   "6222 0212 3456 7890 123",
			forbidden:   "6222",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := strings.Repeat("界", tt.prefixRunes) + tt.sensitive
			got := redactAndBoundAgentSkillJudgeText(value, agentSkillJudgeMaxCriterionRunes, tt.sensitiveValues...)
			if strings.Contains(got, tt.forbidden) || strings.Contains(got, tt.sensitive) {
				t.Fatalf("boundary-crossing PII survived redaction: %q", got)
			}
			if count := len([]rune(got)); count > agentSkillJudgeMaxCriterionRunes {
				t.Fatalf("redacted rune count = %d, want <= %d", count, agentSkillJudgeMaxCriterionRunes)
			}
		})
	}
}

func TestAgentSkillJudgeDispatcherCloseCancelsWorkerAndIsIdempotent(t *testing.T) {
	metrics := observability.NewRegistry("ai-agent")
	runner := &contextAwareAgentSkillJudgeRunner{
		started:  make(chan struct{}, 1),
		finished: make(chan struct{}, 1),
	}
	dispatcher := newAgentSkillJudgeDispatcher(true, runner, metrics)
	dispatcher.TrySubmit(AgentSkillJudgeInput{
		RedactedResponse:   "safe",
		EvaluationCriteria: []string{"accurate"},
	}, agentSkillJudgeTestLabels())
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("judge worker did not start")
	}

	closed := make(chan struct{})
	go func() {
		dispatcher.Close()
		dispatcher.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("dispatcher Close did not cancel and join the worker")
	}
	select {
	case <-runner.finished:
	default:
		t.Fatal("runner did not observe dispatcher cancellation")
	}

	started := time.Now()
	dispatcher.TrySubmit(AgentSkillJudgeInput{
		RedactedResponse:   "safe",
		EvaluationCriteria: []string{"accurate"},
	}, agentSkillJudgeTestLabels())
	if time.Since(started) > 100*time.Millisecond {
		t.Fatal("submission after shutdown blocked")
	}
	if output := metrics.Prometheus(); !strings.Contains(output, `result="unavailable",reason="judge_shutdown"`) {
		t.Fatalf("shutdown rejection is not observable:\n%s", output)
	}
}

func TestAgentSkillJudgeDispatcherQueueSaturationRemainsNonBlocking(t *testing.T) {
	metrics := observability.NewRegistry("ai-agent")
	runner := &contextAwareAgentSkillJudgeRunner{
		started:  make(chan struct{}, 1),
		finished: make(chan struct{}, 1),
	}
	dispatcher := newAgentSkillJudgeDispatcher(true, runner, metrics)
	t.Cleanup(dispatcher.Close)
	input := AgentSkillJudgeInput{
		RedactedResponse:   "safe",
		EvaluationCriteria: []string{"accurate"},
	}
	dispatcher.TrySubmit(input, agentSkillJudgeTestLabels())
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("judge worker did not start")
	}
	for range agentSkillJudgeQueueSize {
		dispatcher.TrySubmit(input, agentSkillJudgeTestLabels())
	}
	started := time.Now()
	dispatcher.TrySubmit(input, agentSkillJudgeTestLabels())
	if time.Since(started) > 100*time.Millisecond {
		t.Fatal("submission to saturated queue blocked")
	}
	if output := metrics.Prometheus(); !strings.Contains(output, `result="unavailable",reason="judge_queue_full"`) {
		t.Fatalf("queue saturation is not observable:\n%s", output)
	}
}

func TestAgentSkillJudgeDispatcherUsesTenSecondContextDeadline(t *testing.T) {
	metrics := observability.NewRegistry("ai-agent")
	runner := &deadlineAgentSkillJudgeRunner{deadline: make(chan time.Time, 1)}
	dispatcher := newAgentSkillJudgeDispatcher(true, runner, metrics)
	t.Cleanup(dispatcher.Close)
	before := time.Now()
	dispatcher.TrySubmit(AgentSkillJudgeInput{
		RedactedResponse:   "safe",
		EvaluationCriteria: []string{"accurate"},
	}, agentSkillJudgeTestLabels())
	select {
	case deadline := <-runner.deadline:
		got := deadline.Sub(before)
		if got < agentSkillJudgeTimeout-250*time.Millisecond || got > agentSkillJudgeTimeout+250*time.Millisecond {
			t.Fatalf("judge context deadline = %v after submission, want %v", got, agentSkillJudgeTimeout)
		}
	case <-time.After(time.Second):
		t.Fatal("judge runner did not receive a context deadline")
	}
}

func TestAgentSkillJudgeEnabledWithoutRunnerIsObservable(t *testing.T) {
	metrics := observability.NewRegistry("ai-agent")
	dispatcher := newAgentSkillJudgeDispatcher(true, nil, metrics)
	dispatcher.TrySubmit(AgentSkillJudgeInput{
		RedactedResponse:   "safe",
		EvaluationCriteria: []string{"accurate"},
	}, observability.AgentSkillMetricLabels{
		RuntimeMode: "v2", ExecutionMode: "direct", SelectionMode: "auto",
		Role: "none", Risk: "unknown",
	})
	output := metrics.Prometheus()
	if !strings.Contains(output, `result="unavailable",reason="judge_runner_unavailable"`) {
		t.Fatalf("missing-runner state is not observable:\n%s", output)
	}
}

func TestAgentSkillRuntimeEvidenceSerializationNeverContainsContent(t *testing.T) {
	governance := hrRuntimeGovernanceContext{
		SelectedAgentSkills: []hrRuntimeAgentSkill{{
			ID: 1, VersionID: 101, Version: "2.0.0", CompiledHash: strings.Repeat("a", 64),
			Name: "screening", DisplayName: "筛选", CoreMarkdown: "PRIVATE_CORE_CONTENT",
			CoreEstimatedTokens: 20, LoadedTokens: 30, Included: true, DecisionReason: "core_included",
			Sections: []hrRuntimeAgentSkillSection{{
				ID: 1001, Key: "rubric", Title: "Rubric", ContentMarkdown: "PRIVATE_SECTION_CONTENT",
				ContentHash: strings.Repeat("b", 64), EstimatedTokens: 10, Included: true, DecisionReason: "section_included",
			}},
		}},
		AgentSkillSelectionMode: "auto",
	}
	governance.AgentSkillRuntimeEvidence = []*pb.AgentSkillRuntimeEvidence{
		hrRuntimeAgentSkillEvidenceToPB(governance.SelectedAgentSkills[0]),
	}
	raw := buildHRProcessContent(nil, nil, false, governance, commonsai.RecruitingPlan{}, "adk", nil)
	if !json.Valid([]byte(raw)) {
		t.Fatalf("process evidence is invalid JSON: %s", raw)
	}
	for _, forbidden := range []string{"PRIVATE_CORE_CONTENT", "PRIVATE_SECTION_CONTENT"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("process evidence leaked %q: %s", forbidden, raw)
		}
	}
	restored := agentSkillRuntimeEvidenceFromProcessContent(raw)
	if len(restored) != 1 || restored[0].GetVersionId() != 101 ||
		restored[0].GetCompiledHash() != strings.Repeat("a", 64) ||
		len(restored[0].GetSections()) != 1 ||
		restored[0].GetSections()[0].GetContentHash() != strings.Repeat("b", 64) ||
		restored[0].GetLoadedTokens() != 30 {
		t.Fatalf("restored evidence lost exact safe fields: %#v", restored)
	}
}

type capturingAgentSkillJudgeRunner struct {
	mu     sync.Mutex
	input  AgentSkillJudgeInput
	block  <-chan struct{}
	called chan struct{}
	err    error
}

func (r *capturingAgentSkillJudgeRunner) Judge(ctx context.Context, input AgentSkillJudgeInput) (AgentSkillJudgeResult, error) {
	r.mu.Lock()
	r.input = cloneAgentSkillJudgeInput(input)
	r.mu.Unlock()
	select {
	case r.called <- struct{}{}:
	default:
	}
	if r.block != nil {
		select {
		case <-r.block:
		case <-ctx.Done():
			return AgentSkillJudgeResult{}, ctx.Err()
		}
	}
	if r.err != nil {
		return AgentSkillJudgeResult{}, r.err
	}
	return AgentSkillJudgeResult{Passed: true}, nil
}

var _ AgentSkillJudgeRunner = (*capturingAgentSkillJudgeRunner)(nil)

type contextAwareAgentSkillJudgeRunner struct {
	started  chan struct{}
	finished chan struct{}
}

func (r *contextAwareAgentSkillJudgeRunner) Judge(ctx context.Context, _ AgentSkillJudgeInput) (AgentSkillJudgeResult, error) {
	select {
	case r.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	select {
	case r.finished <- struct{}{}:
	default:
	}
	return AgentSkillJudgeResult{}, ctx.Err()
}

type deadlineAgentSkillJudgeRunner struct {
	deadline chan time.Time
}

func (r *deadlineAgentSkillJudgeRunner) Judge(ctx context.Context, _ AgentSkillJudgeInput) (AgentSkillJudgeResult, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return AgentSkillJudgeResult{}, context.DeadlineExceeded
	}
	r.deadline <- deadline
	return AgentSkillJudgeResult{Passed: true}, nil
}

func agentSkillJudgeTestLabels() observability.AgentSkillMetricLabels {
	return observability.AgentSkillMetricLabels{
		RuntimeMode: "v2", ExecutionMode: "durable", SelectionMode: "auto",
		Role: "none", Risk: "unknown",
	}
}
