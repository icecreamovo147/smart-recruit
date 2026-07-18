package grpc

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"

	commonsai "smart-recruit-commons/ai"
	"smart-recruit-proto/recruitment/pb"
)

func TestNewHRContextUsageEnvelopeBudgetBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		model      RuntimeModelInfo
		prompt     int32
		wantSafety int32
		wantBudget int32
		wantStatus string
		wantRatio  float64
	}{
		{name: "unknown window", model: RuntimeModelInfo{ContextWindowTokens: 0, MaxOutputTokens: 1024}, prompt: 100, wantStatus: "unknown_config"},
		{name: "minimum safety", model: RuntimeModelInfo{ContextWindowTokens: 4096, MaxOutputTokens: 1024}, prompt: 1000, wantSafety: 256, wantBudget: 2816, wantStatus: "within_budget", wantRatio: 1000.0 / 2816.0},
		{name: "five percent safety", model: RuntimeModelInfo{ContextWindowTokens: 8192, MaxOutputTokens: 1024}, prompt: 5000, wantSafety: 410, wantBudget: 6758, wantStatus: "approaching_target", wantRatio: 5000.0 / 6758.0},
		{name: "exact sixty percent", model: RuntimeModelInfo{ContextWindowTokens: 1256, MaxOutputTokens: 0}, prompt: 600, wantSafety: 256, wantBudget: 1000, wantStatus: "approaching_target", wantRatio: 0.60},
		{name: "exact seventy five percent", model: RuntimeModelInfo{ContextWindowTokens: 1256, MaxOutputTokens: 0}, prompt: 750, wantSafety: 256, wantBudget: 1000, wantStatus: "over_target", wantRatio: 0.75},
		{name: "maximum safety", model: RuntimeModelInfo{ContextWindowTokens: 100000, MaxOutputTokens: 4096}, prompt: 80000, wantSafety: 2048, wantBudget: 93856, wantStatus: "over_target", wantRatio: 80000.0 / 93856.0},
		{name: "max int32 window", model: RuntimeModelInfo{ContextWindowTokens: math.MaxInt32, MaxOutputTokens: 0}, prompt: 1, wantSafety: 2048, wantBudget: math.MaxInt32 - 2048, wantStatus: "within_budget", wantRatio: 1.0 / float64(math.MaxInt32-2048)},
		{name: "negative max output", model: RuntimeModelInfo{ContextWindowTokens: 8192, MaxOutputTokens: -1}, prompt: 10, wantStatus: "invalid_config"},
		{name: "invalid budget", model: RuntimeModelInfo{ContextWindowTokens: 1024, MaxOutputTokens: 900}, prompt: 10, wantSafety: 256, wantBudget: -132, wantStatus: "invalid_config"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newHRContextUsageEnvelope(tt.model, tt.prompt)
			if got.GetSafetyMarginTokens() != tt.wantSafety || got.GetInputBudgetTokens() != tt.wantBudget {
				t.Fatalf("safety/budget = %d/%d, want %d/%d", got.GetSafetyMarginTokens(), got.GetInputBudgetTokens(), tt.wantSafety, tt.wantBudget)
			}
			if got.GetBudgetStatus() != tt.wantStatus {
				t.Fatalf("status = %q, want %q", got.GetBudgetStatus(), tt.wantStatus)
			}
			if math.Abs(got.GetBudgetUsageRatio()-tt.wantRatio) > 0.000001 {
				t.Fatalf("budget ratio = %f, want %f", got.GetBudgetUsageRatio(), tt.wantRatio)
			}
		})
	}
}

func TestEstimateTokensConservativeCountsNonASCIIByRune(t *testing.T) {
	if got := estimateTokensConservative("abcdefgh"); got != 2 {
		t.Fatalf("ASCII estimate = %d, want 2", got)
	}
	if got := estimateTokensConservative("招聘助手🙂"); got != 5 {
		t.Fatalf("non-ASCII estimate = %d, want 5", got)
	}
	if got := estimateTokensConservative("abcd招聘🙂"); got != 4 {
		t.Fatalf("mixed estimate = %d, want 4", got)
	}
}

func TestEstimateHRMessagesContextUsageIsStrictPartitionOfActualInputs(t *testing.T) {
	tools := commonsai.RecruitingTools()
	if len(tools) == 0 {
		t.Fatal("expected recruiting tool schemas")
	}
	messages := []*schema.Message{
		schema.SystemMessage("system and plan instruction 招聘"),
		schema.UserMessage("上一个问题"),
		schema.AssistantMessage("上一个回复", nil),
		schema.UserMessage("当前问题"),
	}
	usage := estimateHRMessagesContextUsage(
		RuntimeModelInfo{ID: 7, Name: "qwen", ContextWindowTokens: 8192, MaxOutputTokens: 1024},
		messages,
		tools[:1],
		"当前问题",
		nil,
		hrRuntimeGovernanceContext{},
	)
	breakdown := usage.GetBreakdown()
	if breakdown.GetToolSchemaTokens() <= 0 {
		t.Fatal("tool schema tokens should be included")
	}
	if breakdown.GetProtocolOverheadTokens() != 18 {
		t.Fatalf("protocol overhead = %d, want 18", breakdown.GetProtocolOverheadTokens())
	}
	if breakdown.GetSystemPromptTokens() <= 0 || breakdown.GetRecentMessageTokens() <= 0 || breakdown.GetCurrentMessageTokens() <= 0 {
		t.Fatalf("message breakdown missing: %+v", breakdown)
	}
	if got := contextUsageBreakdownTotal(breakdown); got != usage.GetPromptTokensEstimated() {
		t.Fatalf("breakdown total = %d, prompt estimate = %d", got, usage.GetPromptTokensEstimated())
	}
}

func TestEstimateHRCompletionContextUsageIsStrictPartition(t *testing.T) {
	usage := estimateHRCompletionContextUsage(RuntimeModelInfo{ContextWindowTokens: 8192, MaxOutputTokens: 1024}, "actual flattened input 招聘\nPLAN", "", nil, ChatMessageRow{}, nil, hrRuntimeGovernanceContext{})
	if got := contextUsageBreakdownTotal(usage.GetBreakdown()); got != usage.GetPromptTokensEstimated() {
		t.Fatalf("breakdown total = %d, prompt estimate = %d", got, usage.GetPromptTokensEstimated())
	}
	want := int32(estimateTokensConservative("actual flattened input 招聘\nPLAN") + 6)
	if usage.GetPromptTokensEstimated() != want {
		t.Fatalf("prompt estimate = %d, want exact completion input estimate %d", usage.GetPromptTokensEstimated(), want)
	}
}

func TestHRContextBreakdownClassifiesActiveSkillsAndChosenToolTraceFragments(t *testing.T) {
	governance := hrRuntimeGovernanceContext{SelectedAgentSkills: []hrRuntimeAgentSkill{{SkillMD: "SKILL_MARKER"}}}
	traces := []ToolTraceRow{
		{ResultContent: "SUCCESS_RESULT"},
		{ResultContent: "IGNORED_ERROR_RESULT", ErrorMsg: "CHOSEN_ERROR"},
	}
	wantSkill := int32(estimateTokensConservative("SKILL_MARKER"))
	wantTools := int32(estimateTokensConservative("SUCCESS_RESULT") + estimateTokensConservative("CHOSEN_ERROR"))

	messages := []*schema.Message{
		schema.SystemMessage("base system\nSKILL_MARKER\nSUCCESS_RESULT\nCHOSEN_ERROR"),
		schema.UserMessage("current"),
	}
	messageUsage := estimateHRMessagesContextUsage(RuntimeModelInfo{}, messages, nil, "current", traces, governance)
	if got := messageUsage.GetBreakdown(); got.GetSkillTokens() != wantSkill || got.GetToolResultTokens() != wantTools {
		t.Fatalf("message semantic breakdown = %+v, want skill/tools %d/%d", got, wantSkill, wantTools)
	}
	if contextUsageBreakdownTotal(messageUsage.GetBreakdown()) != messageUsage.GetPromptTokensEstimated() {
		t.Fatalf("message breakdown is not a strict partition: %+v", messageUsage)
	}

	prompt := "base system\nSKILL_MARKER\nSUCCESS_RESULT\nCHOSEN_ERROR\nUser:\ncurrent\nAssistant:"
	completionUsage := estimateHRCompletionContextUsage(RuntimeModelInfo{}, prompt, "current", nil, ChatMessageRow{}, traces, governance)
	if got := completionUsage.GetBreakdown(); got.GetSkillTokens() != wantSkill || got.GetToolResultTokens() != wantTools || got.GetCurrentMessageTokens() == 0 {
		t.Fatalf("completion semantic breakdown = %+v, want non-zero skill/tool/current", got)
	}
	if contextUsageBreakdownTotal(completionUsage.GetBreakdown()) != completionUsage.GetPromptTokensEstimated() {
		t.Fatalf("completion breakdown is not a strict partition: %+v", completionUsage)
	}
}

func TestApplyToolMetadataUsesLatestContextNotCumulativeBilling(t *testing.T) {
	usage := newHRContextUsageEnvelope(RuntimeModelInfo{ContextWindowTokens: 8192, MaxOutputTokens: 1024}, 500)
	metadata := commonsai.ToolMetadata{
		ContextTokenUsage: &schema.TokenUsage{PromptTokens: 700, CompletionTokens: 30, TotalTokens: 730},
		BillingTokenUsage: &schema.TokenUsage{PromptTokens: 2700, CompletionTokens: 130, TotalTokens: 2830},
	}
	if !applyToolMetadataContextUsage(usage, metadata) {
		t.Fatal("expected actual usage to be applied")
	}
	if usage.GetPromptTokensActual() != 700 || usage.GetTotalTokensActual() != 730 {
		t.Fatalf("actual snapshot = %d/%d, want latest 700/730", usage.GetPromptTokensActual(), usage.GetTotalTokensActual())
	}
	if usage.GetEstimated() || usage.GetSource() != "provider_actual" || usage.GetStage() != "final" {
		t.Fatalf("actual flags = estimated:%v source:%q stage:%q", usage.GetEstimated(), usage.GetSource(), usage.GetStage())
	}

	estimated := newHRContextUsageEnvelope(RuntimeModelInfo{ContextWindowTokens: 8192, MaxOutputTokens: 1024}, 500)
	estimated.PromptTokensEstimated = 500
	estimated.Estimated = true
	estimated.Source = "conservative_estimator"
	estimated.Stage = "pre_generation"
	if applyToolMetadataContextUsage(estimated, commonsai.ToolMetadata{BillingTokenUsage: metadata.BillingTokenUsage}) {
		t.Fatal("billing-only metadata must not replace current context estimate")
	}
	if !estimated.GetEstimated() || estimated.GetPromptTokensEstimated() != 500 {
		t.Fatalf("estimate should be retained without provider context usage: %+v", estimated)
	}
}

func TestApplyActualContextUsageValidationAndSaturation(t *testing.T) {
	tests := []struct {
		name       string
		actual     *schema.TokenUsage
		wantApply  bool
		wantPrompt int32
		wantDone   int32
		wantTotal  int32
	}{
		{name: "nil", actual: nil},
		{name: "all zero", actual: &schema.TokenUsage{}},
		{name: "negative prompt", actual: &schema.TokenUsage{PromptTokens: -1, CompletionTokens: 2, TotalTokens: 1}},
		{name: "negative completion", actual: &schema.TokenUsage{PromptTokens: 2, CompletionTokens: -1, TotalTokens: 1}},
		{name: "negative total", actual: &schema.TokenUsage{PromptTokens: 2, CompletionTokens: 1, TotalTokens: -1}},
		{name: "normal", actual: &schema.TokenUsage{PromptTokens: 120, CompletionTokens: 7, TotalTokens: 127}, wantApply: true, wantPrompt: 120, wantDone: 7, wantTotal: 127},
		{name: "missing total uses components", actual: &schema.TokenUsage{PromptTokens: 120, CompletionTokens: 7}, wantApply: true, wantPrompt: 120, wantDone: 7, wantTotal: 127},
		{name: "undersized total uses components", actual: &schema.TokenUsage{PromptTokens: 120, CompletionTokens: 7, TotalTokens: 100}, wantApply: true, wantPrompt: 120, wantDone: 7, wantTotal: 127},
		{name: "larger total retained", actual: &schema.TokenUsage{PromptTokens: 120, CompletionTokens: 7, TotalTokens: 140}, wantApply: true, wantPrompt: 120, wantDone: 7, wantTotal: 140},
		{name: "overflow saturates", actual: &schema.TokenUsage{PromptTokens: math.MaxInt32 + 50, CompletionTokens: math.MaxInt32 + 60, TotalTokens: math.MaxInt32 + 70}, wantApply: true, wantPrompt: math.MaxInt32, wantDone: math.MaxInt32, wantTotal: math.MaxInt32},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := estimateHRCompletionContextUsage(RuntimeModelInfo{ContextWindowTokens: 8192, MaxOutputTokens: 1024}, "input", "", nil, ChatMessageRow{}, nil, hrRuntimeGovernanceContext{})
			beforeEstimate := usage.GetPromptTokensEstimated()
			if got := applyActualContextUsage(usage, tt.actual); got != tt.wantApply {
				t.Fatalf("apply = %v, want %v", got, tt.wantApply)
			}
			if !tt.wantApply {
				if !usage.GetEstimated() || usage.GetSource() != "conservative_estimator" || usage.GetPromptTokensEstimated() != beforeEstimate {
					t.Fatalf("unavailable usage replaced estimate: %+v", usage)
				}
				return
			}
			if usage.GetPromptTokensActual() != tt.wantPrompt || usage.GetCompletionTokensActual() != tt.wantDone || usage.GetTotalTokensActual() != tt.wantTotal {
				t.Fatalf("actual usage = %d/%d/%d, want %d/%d/%d", usage.GetPromptTokensActual(), usage.GetCompletionTokensActual(), usage.GetTotalTokensActual(), tt.wantPrompt, tt.wantDone, tt.wantTotal)
			}
		})
	}
}

type usageAwareTestProvider struct {
	result   commonsai.GenerateResult
	modelIDs []int64
	prompts  []string
}

func (p *usageAwareTestProvider) Complete(context.Context, string) (string, error) {
	return "legacy", nil
}

func (p *usageAwareTestProvider) CompleteWithOptionsAndUsage(_ context.Context, prompt string, modelID int64, _ ChatCompletionOptions) (commonsai.GenerateResult, error) {
	p.modelIDs = append(p.modelIDs, modelID)
	p.prompts = append(p.prompts, prompt)
	return p.result, nil
}

func TestCompleteWithUsagePrefersUsageAwareProvider(t *testing.T) {
	provider := &usageAwareTestProvider{result: commonsai.GenerateResult{
		Content:    "actual reply",
		TokenUsage: &schema.TokenUsage{PromptTokens: 123, CompletionTokens: 7, TotalTokens: 130},
	}}
	service := &nativeAIService{provider: provider}
	result, err := service.completeWithUsage(context.Background(), "prompt", 9)
	if err != nil {
		t.Fatalf("completeWithUsage returned error: %v", err)
	}
	if strings.TrimSpace(result.Content) != "actual reply" || result.TokenUsage == nil || result.TokenUsage.PromptTokens != 123 {
		t.Fatalf("usage-aware result = %+v", result)
	}
}

func TestHRChatFinalContextUsageMatchesEventAndPersistedAssistant(t *testing.T) {
	store := newFakeAIStore()
	store.llmModels = []*pb.LlmModelInfo{{
		Id: 7, ModelName: "qwen-context", IsEnabled: true, IsDefault: true,
		ContextWindowTokens: 8192, MaxTokens: 1024,
	}}
	provider := &usageAwareTestProvider{result: commonsai.GenerateResult{
		Content:    "final answer",
		TokenUsage: &schema.TokenUsage{PromptTokens: 321, CompletionTokens: 21, TotalTokens: 342},
	}}
	service := &nativeAIService{store: store, provider: provider}
	stream := &captureChatStream{ctx: context.Background()}
	if err := service.ChatStream(&pb.ChatRequest{HrId: 77, Message: "draft a candidate outreach note", ModelId: 7}, stream); err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}
	if len(stream.responses) == 0 {
		t.Fatal("expected stream responses")
	}
	final := stream.responses[len(stream.responses)-1].GetContextUsage()
	if final.GetPromptTokensActual() != 321 || final.GetInputBudgetTokens() != 6758 || final.GetStage() != "final" {
		t.Fatalf("final stream context usage = %+v", final)
	}
	var actualEvent *pb.ContextUsageInfo
	for _, event := range stream.responses {
		if event.GetEventType() == "context_usage" && event.GetContextUsage().GetStage() == "final" {
			actualEvent = event.GetContextUsage()
		}
	}
	if actualEvent == nil || actualEvent.GetPromptTokensActual() != final.GetPromptTokensActual() {
		t.Fatalf("provider actual event = %+v, final = %+v", actualEvent, final)
	}
	if len(store.messages) < 2 {
		t.Fatalf("persisted messages = %d, want user and assistant", len(store.messages))
	}
	persisted := store.messages[len(store.messages)-1].ContextUsage
	if persisted == nil || persisted.GetPromptTokensActual() != final.GetPromptTokensActual() || persisted.GetInputBudgetTokens() != final.GetInputBudgetTokens() {
		t.Fatalf("persisted context usage = %+v, final = %+v", persisted, final)
	}
}

func TestHRCompletionWithoutProviderUsageKeepsEstimateOfExactSentPromptAndResolvedModel(t *testing.T) {
	tests := []struct {
		name        string
		requestedID int64
		wantID      int64
	}{
		{name: "default resolved once", requestedID: 0, wantID: 7},
		{name: "explicit unchanged", requestedID: 9, wantID: 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAIStore()
			store.llmModels = []*pb.LlmModelInfo{
				{Id: 7, ModelName: "default-model", IsEnabled: true, IsDefault: true, ContextWindowTokens: 8192, MaxTokens: 1024},
				{Id: 9, ModelName: "explicit-model", IsEnabled: true, ContextWindowTokens: 16384, MaxTokens: 2048},
			}
			provider := &usageAwareTestProvider{result: commonsai.GenerateResult{Content: "estimated reply", TokenUsage: &schema.TokenUsage{}}}
			service := &nativeAIService{store: store, provider: provider}
			resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "draft a candidate outreach note", ModelId: tt.requestedID})
			if err != nil {
				t.Fatalf("Chat returned error: %v", err)
			}
			if len(provider.modelIDs) != 1 || provider.modelIDs[0] != tt.wantID {
				t.Fatalf("provider model IDs = %v, want [%d]", provider.modelIDs, tt.wantID)
			}
			if len(provider.prompts) != 1 {
				t.Fatalf("provider prompts = %d, want 1", len(provider.prompts))
			}
			usage := resp.GetContextUsage()
			want := estimateHRCompletionContextUsage(RuntimeModelInfo{}, provider.prompts[0], "", nil, ChatMessageRow{}, nil, hrRuntimeGovernanceContext{}).GetPromptTokensEstimated()
			if !usage.GetEstimated() || usage.GetSource() != "conservative_estimator" || usage.GetPromptTokensEstimated() != want {
				t.Fatalf("final estimate = %+v, want prompt estimate %d", usage, want)
			}
			if got := contextUsageBreakdownTotal(usage.GetBreakdown()); got != usage.GetPromptTokensEstimated() {
				t.Fatalf("breakdown total = %d, prompt estimate = %d", got, usage.GetPromptTokensEstimated())
			}
		})
	}
}

func TestHRContextGuardRejectsBeforeProviderWithStableChatAndStreamErrors(t *testing.T) {
	for _, tt := range []struct {
		name  string
		model *pb.LlmModelInfo
		want  string
	}{
		{name: "invalid configuration", model: &pb.LlmModelInfo{Id: 7, ModelName: "invalid", IsEnabled: true, IsDefault: true, ContextWindowTokens: 256, MaxTokens: 1}, want: hrContextConfigInvalidCode},
		{name: "fixed envelope overflow", model: &pb.LlmModelInfo{Id: 7, ModelName: "small", IsEnabled: true, IsDefault: true, ContextWindowTokens: 520, MaxTokens: 100}, want: hrContextBudgetExceededCode},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAIStore()
			store.llmModels = []*pb.LlmModelInfo{tt.model}
			provider := &usageAwareTestProvider{result: commonsai.GenerateResult{Content: "must not run"}}
			service := &nativeAIService{store: store, provider: provider}
			resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "hello", ModelId: 7})
			if err != nil {
				t.Fatalf("Chat: %v", err)
			}
			if resp.GetMsg() != tt.want || len(provider.prompts) != 0 {
				t.Fatalf("chat msg=%q provider_calls=%d", resp.GetMsg(), len(provider.prompts))
			}

			streamStore := newFakeAIStore()
			streamStore.llmModels = []*pb.LlmModelInfo{tt.model}
			streamProvider := &usageAwareTestProvider{result: commonsai.GenerateResult{Content: "must not run"}}
			streamService := &nativeAIService{store: streamStore, provider: streamProvider}
			stream := &captureChatStream{ctx: context.Background()}
			if err := streamService.ChatStream(&pb.ChatRequest{HrId: 77, Message: "hello", ModelId: 7}, stream); err != nil {
				t.Fatalf("ChatStream: %v", err)
			}
			last := stream.responses[len(stream.responses)-1]
			if last.GetErrorType() != tt.want || last.GetEventMessage() != tt.want || !last.GetDone() || len(streamProvider.prompts) != 0 {
				t.Fatalf("stream last=%+v provider_calls=%d", last, len(streamProvider.prompts))
			}
		})
	}
}

func TestHRToolHandlerUsesActualMessagesResolvedModelAndLatestContextUsage(t *testing.T) {
	store := newFakeAIStore()
	store.llmModels = []*pb.LlmModelInfo{{
		Id: 7, ModelName: "default-tool-model", IsEnabled: true, IsDefault: true,
		ContextWindowTokens: 8192, MaxTokens: 1024,
	}}
	store.agentConfigs = []*pb.AgentConfigInfo{{
		Id: 11, Name: "hr-tool-agent", AgentType: hrRecruitingAgentType, IsDefault: true, IsEnabled: true,
		ToolBindings: []*pb.AgentToolBindingInfo{{ToolName: "get_job_list", IsEnabled: true}},
	}}
	provider := &fakeRecruitingToolProvider{
		reply: "tool-path answer",
		metadata: commonsai.ToolMetadata{
			ContextTokenUsage: &schema.TokenUsage{PromptTokens: 400, CompletionTokens: 20, TotalTokens: 420},
			BillingTokenUsage: &schema.TokenUsage{PromptTokens: 1400, CompletionTokens: 120, TotalTokens: 1520},
		},
	}
	service := newNativeAIService(store, provider, nil, &fakeHRJobClient{}, nil)
	stream := &captureChatStream{ctx: context.Background()}
	if err := service.ChatStream(&pb.ChatRequest{HrId: 77, Message: "draft a candidate outreach note"}, stream); err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}
	if provider.toolCalls != 1 || len(provider.modelIDs) != 1 || provider.modelIDs[0] != 7 {
		t.Fatalf("tool calls/model IDs = %d/%v, want 1/[7]", provider.toolCalls, provider.modelIDs)
	}
	if len(provider.messages) != 1 || len(provider.messages[0]) == 0 || !strings.Contains(provider.messages[0][0].Content, "Deterministic planner output JSON") {
		t.Fatalf("provider messages do not contain actual plan instruction: %#v", provider.messages)
	}
	var actualEvent *pb.ContextUsageInfo
	for _, event := range stream.responses {
		if event.GetEventType() != "context_usage" {
			continue
		}
		if !event.GetContextUsage().GetEstimated() {
			actualEvent = event.GetContextUsage()
		}
	}
	final := stream.responses[len(stream.responses)-1].GetContextUsage()
	if actualEvent == nil || final.GetPromptTokensActual() != 400 || actualEvent.GetPromptTokensActual() != 400 || final.GetTotalTokensActual() != 420 {
		t.Fatalf("actual/final usage = %+v / %+v, want latest 400/420", actualEvent, final)
	}
	persisted := store.messages[len(store.messages)-1].ContextUsage
	if persisted == nil || persisted.GetPromptTokensActual() != 400 || persisted.GetTotalTokensActual() != 420 {
		t.Fatalf("persisted usage = %+v, want latest context usage", persisted)
	}
	if len(store.usageAudits) != 1 || store.usageAudits[0].TokenUsageTotal != 1520 {
		t.Fatalf("usage audits = %#v, want cumulative billing total 1520", store.usageAudits)
	}
}

func TestHRToolHandlerWithoutProviderUsageKeepsExactMessageEstimate(t *testing.T) {
	store := newFakeAIStore()
	store.llmModels = []*pb.LlmModelInfo{{
		Id: 7, ModelName: "default-tool-model", IsEnabled: true, IsDefault: true,
		ContextWindowTokens: 8192, MaxTokens: 1024,
	}}
	store.agentConfigs = []*pb.AgentConfigInfo{{
		Id: 11, Name: "hr-tool-agent", AgentType: hrRecruitingAgentType, IsDefault: true, IsEnabled: true,
		ToolBindings: []*pb.AgentToolBindingInfo{{ToolName: "get_job_list", IsEnabled: true}},
	}}
	provider := &fakeRecruitingToolProvider{reply: "estimated tool-path answer", metadata: commonsai.ToolMetadata{ContextTokenUsage: &schema.TokenUsage{}}}
	service := newNativeAIService(store, provider, nil, &fakeHRJobClient{}, nil)
	resp, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "draft a candidate outreach note"})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if len(provider.messages) != 1 {
		t.Fatalf("provider message calls = %d, want 1", len(provider.messages))
	}
	want := estimateHRMessagesContextUsage(RuntimeModelInfo{}, provider.messages[0], hrRecruitingToolSchemas([]string{"get_job_list"}), "draft a candidate outreach note", nil, hrRuntimeGovernanceContext{}).GetPromptTokensEstimated()
	usage := resp.GetContextUsage()
	if !usage.GetEstimated() || usage.GetPromptTokensEstimated() != want || contextUsageBreakdownTotal(usage.GetBreakdown()) != want {
		t.Fatalf("final usage = %+v, want exact message estimate %d", usage, want)
	}
}

func TestHRADKHandlerReceivesResolvedDefaultModelID(t *testing.T) {
	store := newFakeAIStore()
	store.llmModels = []*pb.LlmModelInfo{{
		Id: 7, ModelName: "default-adk-model", IsEnabled: true, IsDefault: true,
		ContextWindowTokens: 8192, MaxTokens: 1024,
	}}
	store.agentConfigs = []*pb.AgentConfigInfo{{
		Id: 11, Name: "hr-adk-agent", AgentType: hrRecruitingAgentType, IsDefault: true, IsEnabled: true,
		ToolBindings: []*pb.AgentToolBindingInfo{{ToolName: "get_job_list", IsEnabled: true}},
	}}
	provider := &fakeCandidateADKProvider{reply: "adk-path answer"}
	service := newNativeAIService(store, provider, nil, &fakeHRJobClient{}, nil)
	service.agentRuntime = agentRuntimeADK
	if _, err := service.Chat(context.Background(), &pb.ChatRequest{HrId: 77, Message: "draft a candidate outreach note"}); err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if provider.adkCalls != 1 || provider.lastModelID != 7 {
		t.Fatalf("ADK calls/model ID = %d/%d, want 1/7", provider.adkCalls, provider.lastModelID)
	}
}

func TestContextUsagePayloadIncludesAllAdditiveFields(t *testing.T) {
	usage := &pb.ContextUsageInfo{
		PromptTokensActual: 44, InputBudgetTokens: 6758, SafetyMarginTokens: 410,
		BudgetUsageRatio: 0.2, BudgetStatus: "within_budget", IncludedMessageCount: 8,
		OmittedMessageCount: 2, SummaryApplied: true,
		Breakdown: &pb.ContextUsageBreakdown{ToolSchemaTokens: 11, ProtocolOverheadTokens: 22},
	}
	payload := contextUsagePayload(usage)
	for _, key := range []string{
		"prompt_tokens_actual", "input_budget_tokens", "safety_margin_tokens", "budget_usage_ratio",
		"budget_status", "included_message_count", "omitted_message_count", "summary_applied", "breakdown",
	} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("payload missing %q: %#v", key, payload)
		}
	}
	breakdown, ok := payload["breakdown"].(map[string]any)
	if !ok || breakdown["tool_schema_tokens"] != int32(11) || breakdown["protocol_overhead_tokens"] != int32(22) {
		t.Fatalf("payload breakdown = %#v", payload["breakdown"])
	}
}
