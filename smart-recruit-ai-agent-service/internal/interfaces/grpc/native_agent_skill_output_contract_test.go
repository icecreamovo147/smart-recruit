package grpc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domainagentskill "smart-recruit-ai-agent-service/internal/domain/agentskill"
	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
	commonsai "smart-recruit-commons/ai"
	"smart-recruit-platform-go/observability"
	"smart-recruit-proto/recruitment/pb"
)

const advisorySchemaID = "candidate-summary-v2"

var advisorySchema = json.RawMessage(`{
	"type": "object",
	"required": ["summary", "score"],
	"properties": {
		"summary": {"type": "string"},
		"score": {"maximum": 100, "type": "number"}
	}
}`)

func TestCompileHRRuntimeAdvisoryInstructionIsCanonicalBoundedAndHonest(t *testing.T) {
	first, err := compileHRRuntimeAdvisoryInstruction(domainagentskill.OutputContract{
		Mode:     domainagentskill.OutputModeAdvisory,
		SchemaID: advisorySchemaID,
		Schema:   advisorySchema,
	})
	if err != nil {
		t.Fatalf("compile first advisory instruction: %v", err)
	}
	second, err := compileHRRuntimeAdvisoryInstruction(domainagentskill.OutputContract{
		Mode:     domainagentskill.OutputModeAdvisory,
		SchemaID: advisorySchemaID,
		Schema:   json.RawMessage(`{"properties":{"score":{"type":"number","maximum":100},"summary":{"type":"string"}},"required":["summary","score"],"type":"object"}`),
	})
	if err != nil {
		t.Fatalf("compile second advisory instruction: %v", err)
	}
	if first != second {
		t.Fatalf("canonical advisory instructions differ:\nfirst=%s\nsecond=%s", first, second)
	}
	for _, want := range []string{
		"Natural-language output remains allowed",
		"Do not return raw JSON unless the user explicitly asks for JSON",
		"does not validate, repair, retry, or reject",
		`"schema_id":"candidate-summary-v2"`,
		`"properties":{"score":{"maximum":100,"type":"number"},"summary":{"type":"string"}}`,
	} {
		if !strings.Contains(first, want) {
			t.Fatalf("advisory instruction missing %q:\n%s", want, first)
		}
	}
	if len(first) > hrRuntimeMaxAdvisoryInstructionBytes {
		t.Fatalf("advisory instruction is not bounded: %d bytes", len(first))
	}
}

func TestCompileHRRuntimeAdvisoryInstructionMatchesCompilerSchemaIDBoundary(t *testing.T) {
	exactEscaped := "a" + strings.Repeat("\x00", domainagentskill.MaxOutputSchemaIDBytes-2) + "a"
	exactSchema := json.RawMessage(`{"x":"` +
		strings.Repeat("a", domainagentskill.MaxOutputSchemaBytes-len(`{"x":""}`)) +
		`"}`)
	compiled, err := domainagentskill.Compile(domainagentskill.PackageDraft{
		Manifest: domainagentskill.Manifest{
			SchemaVersion:  domainagentskill.SchemaVersion,
			SkillName:      "output-boundary",
			DisplayName:    "Output boundary",
			AgentType:      hrRecruitingAgentType,
			RiskLevel:      domainagentskill.RiskLevelLow,
			Composition:    domainagentskill.Composition{Role: domainagentskill.CompositionRolePrimary},
			OutputContract: domainagentskill.OutputContract{Mode: domainagentskill.OutputModeAdvisory, SchemaID: exactEscaped, Schema: exactSchema},
		},
		Core: domainagentskill.Core{ContentMarkdown: "Follow the output contract."},
	})
	if err != nil {
		t.Fatalf("compile exact-boundary package: %v", err)
	}
	if len(compiled.Manifest.OutputContract.Schema) != domainagentskill.MaxOutputSchemaBytes {
		t.Fatalf("compiled schema bytes=%d, want %d", len(compiled.Manifest.OutputContract.Schema), domainagentskill.MaxOutputSchemaBytes)
	}
	instruction, err := compileHRRuntimeAdvisoryInstruction(compiled.Manifest.OutputContract)
	if err != nil {
		t.Fatalf("compile exact-boundary advisory instruction: %v", err)
	}
	if len(instruction) > hrRuntimeMaxAdvisoryInstructionBytes ||
		!strings.Contains(instruction, `\u0000`) {
		t.Fatalf("exact-boundary advisory instruction bytes=%d, bound=%d", len(instruction), hrRuntimeMaxAdvisoryInstructionBytes)
	}

	_, err = compileHRRuntimeAdvisoryInstruction(domainagentskill.OutputContract{
		Mode:     domainagentskill.OutputModeAdvisory,
		SchemaID: strings.Repeat("界", domainagentskill.MaxOutputSchemaIDBytes/len("界")) + "aa",
		Schema:   json.RawMessage(`{"type":"object"}`),
	})
	if err == nil || !strings.Contains(err.Error(), "schema ID size") {
		t.Fatalf("over-boundary advisory instruction error = %v", err)
	}

	overSchema := json.RawMessage(`{"x":"` +
		strings.Repeat("a", domainagentskill.MaxOutputSchemaBytes-len(`{"x":""}`)+1) +
		`"}`)
	_, err = compileHRRuntimeAdvisoryInstruction(domainagentskill.OutputContract{
		Mode:   domainagentskill.OutputModeAdvisory,
		Schema: overSchema,
	})
	if err == nil || !strings.Contains(err.Error(), "advisory schema size") {
		t.Fatalf("over-boundary advisory schema error = %v", err)
	}
}

func TestHRChatAdvisoryContractReturnsFreeTextUnchangedAndRecordsOnce(t *testing.T) {
	service, store, provider := newOutputContractDirectRuntime(
		t,
		domainagentskill.OutputModeAdvisory,
		domainagentskill.RiskLevelLow,
	)
	const freeText = "候选人整体匹配，但我会用自然语言说明，而不是返回 JSON。"
	provider.reply = freeText

	response, err := service.Chat(context.Background(), &pb.ChatRequest{
		HrId:                 77,
		Message:              "resume screening advisory output",
		ModelId:              1,
		AgentSkillVersionIds: []int64{101},
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if response.GetCode() != 0 || response.GetReply() != freeText || provider.calls != 1 {
		t.Fatalf("response=%#v provider calls=%d", response, provider.calls)
	}
	if len(provider.prompts) != 1 {
		t.Fatalf("provider prompts=%d, want one", len(provider.prompts))
	}
	assertAdvisoryPrompt(t, provider.prompts[0])

	metrics := service.metrics.Prometheus()
	wantMetric := `smart_recruit_agent_skill_output_validation_total{runtime_mode="v2",execution_mode="direct",selection_mode="manual",role="primary",risk="low",result="applied",reason="advisory_not_enforced"} 1`
	if strings.Count(metrics, "smart_recruit_agent_skill_output_validation_total{") != 1 ||
		!strings.Contains(metrics, wantMetric) {
		t.Fatalf("advisory metric was not emitted exactly once:\n%s", metrics)
	}
	for _, message := range store.messages {
		if strings.Contains(message.ProcessContent, advisorySchemaID) ||
			strings.Contains(message.ProcessContent, `"required":["summary","score"]`) {
			t.Fatalf("runtime evidence leaked output contract: %s", message.ProcessContent)
		}
	}
}

func TestHRToolAndADKAdvisoryContractReachSharedPromptAndRecordOnce(t *testing.T) {
	const freeText = "The provider keeps this free-text answer unchanged."

	t.Run("tool provider", func(t *testing.T) {
		service, store, _ := newOutputContractDirectRuntime(
			t,
			domainagentskill.OutputModeAdvisory,
			domainagentskill.RiskLevelLow,
		)
		provider := &fakeRecruitingToolProvider{reply: freeText}
		service.provider = provider
		service.agentRuntime = agentRuntimeLegacy
		service.jobs = &fakeHRJobClient{}
		store.agentConfigs[0].ToolBindings = []*pb.AgentToolBindingInfo{{
			ToolName: "get_job_list", IsEnabled: true,
		}}

		response, err := service.Chat(context.Background(), &pb.ChatRequest{
			HrId: 77, Message: "draft a candidate outreach note", ModelId: 1,
			AgentSkillVersionIds: []int64{101},
		})
		if err != nil {
			t.Fatalf("tool Chat returned error: %v", err)
		}
		if response.GetReply() != freeText || provider.toolCalls != 1 ||
			len(provider.messages) != 1 || len(provider.messages[0]) == 0 {
			t.Fatalf("tool response=%#v calls=%d messages=%#v", response, provider.toolCalls, provider.messages)
		}
		assertAdvisoryPrompt(t, provider.messages[0][0].Content)
		assertSingleOutputContractMetric(t, service, "applied", "advisory_not_enforced")
	})

	t.Run("ADK provider", func(t *testing.T) {
		service, store, _ := newOutputContractDirectRuntime(
			t,
			domainagentskill.OutputModeAdvisory,
			domainagentskill.RiskLevelLow,
		)
		var instruction string
		provider := &fakeCandidateADKProvider{
			reply: freeText,
			onRun: func(input commonsai.AgentRunInput) {
				instruction = input.Instruction
			},
		}
		service.provider = provider
		service.agentRuntime = agentRuntimeADK
		service.jobs = &fakeHRJobClient{}
		store.agentConfigs[0].ToolBindings = []*pb.AgentToolBindingInfo{{
			ToolName: "get_job_list", IsEnabled: true,
		}}

		response, err := service.Chat(context.Background(), &pb.ChatRequest{
			HrId: 77, Message: "draft a candidate outreach note", ModelId: 1,
			AgentSkillVersionIds: []int64{101},
		})
		if err != nil {
			t.Fatalf("ADK Chat returned error: %v", err)
		}
		if response.GetReply() != freeText || provider.adkCalls != 1 {
			t.Fatalf("ADK response=%#v calls=%d", response, provider.adkCalls)
		}
		assertAdvisoryPrompt(t, instruction)
		assertSingleOutputContractMetric(t, service, "applied", "advisory_not_enforced")
	})
}

func TestHRAdvisoryPromptParityAcrossDirectDurableAndConfirmationResume(t *testing.T) {
	direct, _, directProvider := newOutputContractDirectRuntime(
		t,
		domainagentskill.OutputModeAdvisory,
		domainagentskill.RiskLevelLow,
	)
	if _, err := direct.Chat(context.Background(), &pb.ChatRequest{
		HrId: 77, Message: "resume screening advisory parity", ModelId: 1,
		AgentSkillVersionIds: []int64{101},
	}); err != nil {
		t.Fatalf("direct Chat returned error: %v", err)
	}
	directInstruction := advisoryInstructionFromPrompt(t, directProvider.prompts[0])

	durable, durableStore, durableProvider, actor := newAgentSkillConfirmationRuntime(
		t,
		domainagentskill.RiskLevelLow,
		[]int64{101},
	)
	setOutputContractDocument(
		&durableStore.agentSkillVersionDocs[0],
		domainagentskill.OutputModeAdvisory,
	)
	created, err := durable.CreateAgentRun(actor, &pb.CreateAgentRunRequest{
		HrId: 77, SessionId: 101, ClientRequestId: "advisory-durable-parity",
		Message: "resume screening advisory parity", ModelId: 1,
		AgentSkillVersionIds: []int64{101},
	})
	if err != nil || created.GetRun() == nil {
		t.Fatalf("CreateAgentRun response=%#v err=%v", created, err)
	}
	waitUntilAgentRunTest(t, time.Second, func() bool { return durableProvider.callCount() == 1 })
	durablePrompts := durableProvider.promptsSnapshot()
	if len(durablePrompts) != 1 {
		t.Fatalf("durable prompts=%#v", durablePrompts)
	}
	if got := advisoryInstructionFromPrompt(t, durablePrompts[0]); got != directInstruction {
		t.Fatalf("direct/durable advisory instruction mismatch:\ndirect=%s\ndurable=%s", directInstruction, got)
	}
	close(durableProvider.release)
	var durableFinal AgentRunRow
	waitUntilAgentRunTest(t, time.Second, func() bool {
		var found bool
		durableFinal, found = durableStore.runSnapshot(created.GetRun().GetRunId())
		return found && durableFinal.Status == agentRunStatusSucceeded
	})
	if durableFinal.AssistantText != "approved reply" || durableProvider.callCount() != 1 {
		t.Fatalf("durable advisory changed provider free text or retried: run=%#v calls=%d", durableFinal, durableProvider.callCount())
	}

	confirmed, confirmedStore, confirmedProvider, confirmedActor := newAgentSkillConfirmationRuntime(
		t,
		domainagentskill.RiskLevelHigh,
		[]int64{101},
	)
	setOutputContractDocument(
		&confirmedStore.agentSkillVersionDocs[0],
		domainagentskill.OutputModeAdvisory,
	)
	waiting := createWaitingAgentSkillRun(t, confirmed, confirmedActor, 101, []int64{101})
	confirmation := mapAgentRunSnapshot(waiting).GetConfirmationRequest()
	if confirmedProvider.callCount() != 0 {
		t.Fatalf("provider ran before approval: %d", confirmedProvider.callCount())
	}
	approved, err := confirmed.ConfirmAgentRun(confirmedActor, &pb.ConfirmAgentRunRequest{
		HrId: 77, RunId: waiting.ID, ClientRequestId: "advisory-confirmed",
		AgentSkillConfirmationId:       confirmation.GetAgentSkillConfirmationId(),
		AgentSkillConfirmationDecision: pb.AgentSkillConfirmationDecision_AGENT_SKILL_CONFIRMATION_DECISION_APPROVE,
		SelectedAgentSkillVersionIds:   []int64{101},
	})
	if err != nil || approved.GetCode() != 0 {
		t.Fatalf("ConfirmAgentRun response=%#v err=%v", approved, err)
	}
	waitUntilAgentRunTest(t, time.Second, func() bool { return confirmedProvider.callCount() == 1 })
	confirmedPrompts := confirmedProvider.promptsSnapshot()
	if len(confirmedPrompts) != 1 ||
		advisoryInstructionFromPrompt(t, confirmedPrompts[0]) != directInstruction ||
		strings.Count(confirmedPrompts[0], "### Advisory response structure") != 1 {
		t.Fatalf("confirmed prompt did not apply exact advisory contract once: %#v", confirmedPrompts)
	}
	close(confirmedProvider.release)
	var confirmedFinal AgentRunRow
	waitUntilAgentRunTest(t, time.Second, func() bool {
		var found bool
		confirmedFinal, found = confirmedStore.runSnapshot(waiting.ID)
		return found && confirmedFinal.Status == agentRunStatusSucceeded
	})
	if confirmedFinal.AssistantText != "approved reply" || confirmedProvider.callCount() != 1 {
		t.Fatalf("confirmed advisory changed provider free text or retried: run=%#v calls=%d", confirmedFinal, confirmedProvider.callCount())
	}
	metrics := confirmed.metrics.Prometheus()
	if strings.Count(metrics, `result="applied",reason="advisory_not_enforced"`) != 1 {
		t.Fatalf("confirmed advisory response metric was not emitted once:\n%s", metrics)
	}
}

func TestHRStrictOutputContractFailsClosedBeforeProviderOrTools(t *testing.T) {
	t.Run("direct", func(t *testing.T) {
		service, store, provider := newOutputContractDirectRuntime(
			t,
			domainagentskill.OutputModeStrict,
			domainagentskill.RiskLevelLow,
		)
		_, err := service.Chat(context.Background(), &pb.ChatRequest{
			HrId: 77, Message: "resume screening strict output", ModelId: 1,
			AgentSkillVersionIds: []int64{101},
		})
		if status.Code(err) != codes.FailedPrecondition ||
			status.Convert(err).Message() != "AGENT_SKILL_STRICT_OUTPUT_UNSUPPORTED" {
			t.Fatalf("strict direct error=%v", err)
		}
		if provider.calls != 0 || len(store.toolTraces) != 0 {
			t.Fatalf("strict direct provider calls=%d tool traces=%#v", provider.calls, store.toolTraces)
		}
		metrics := service.metrics.Prometheus()
		if strings.Count(metrics, `result="unsupported",reason="strict_unsupported"`) != 1 {
			t.Fatalf("strict direct metric was not emitted once:\n%s", metrics)
		}
	})

	t.Run("durable", func(t *testing.T) {
		service, store, provider, actor := newAgentSkillConfirmationRuntime(
			t,
			domainagentskill.RiskLevelLow,
			[]int64{101},
		)
		setOutputContractDocument(
			&store.agentSkillVersionDocs[0],
			domainagentskill.OutputModeStrict,
		)
		created, err := service.CreateAgentRun(actor, &pb.CreateAgentRunRequest{
			HrId: 77, SessionId: 101, ClientRequestId: "strict-durable",
			Message: "resume screening strict output", ModelId: 1,
			AgentSkillVersionIds: []int64{101},
		})
		if err != nil || created.GetRun() == nil {
			t.Fatalf("CreateAgentRun response=%#v err=%v", created, err)
		}
		var failed AgentRunRow
		waitUntilAgentRunTest(t, time.Second, func() bool {
			var found bool
			failed, found = store.runSnapshot(created.GetRun().GetRunId())
			return found && failed.Status == agentRunStatusFailed
		})
		if provider.callCount() != 0 || len(store.toolTraces) != 0 ||
			failed.ErrorType != "agent_skill_strict_output_unsupported" ||
			failed.ErrorMessage != "ai.agent_skill_strict_output_unsupported" {
			t.Fatalf("strict durable run=%#v provider calls=%d tool traces=%#v", failed, provider.callCount(), store.toolTraces)
		}
		metrics := service.metrics.Prometheus()
		if strings.Count(metrics, `result="unsupported",reason="strict_unsupported"`) != 1 {
			t.Fatalf("strict durable metric included preflight or duplicate emission:\n%s", metrics)
		}
	})

	t.Run("tool provider", func(t *testing.T) {
		service, store, _ := newOutputContractDirectRuntime(
			t,
			domainagentskill.OutputModeStrict,
			domainagentskill.RiskLevelLow,
		)
		provider := &fakeRecruitingToolProvider{reply: "must not run"}
		service.provider = provider
		service.agentRuntime = agentRuntimeLegacy
		service.jobs = &fakeHRJobClient{}
		store.agentConfigs[0].ToolBindings = []*pb.AgentToolBindingInfo{{
			ToolName: "get_job_list", IsEnabled: true,
		}}

		_, err := service.Chat(context.Background(), &pb.ChatRequest{
			HrId: 77, Message: "draft a candidate outreach note", ModelId: 1,
			AgentSkillVersionIds: []int64{101},
		})
		if status.Code(err) != codes.FailedPrecondition ||
			provider.toolCalls != 0 || len(store.toolTraces) != 0 {
			t.Fatalf("strict tool error=%v provider calls=%d tool traces=%#v", err, provider.toolCalls, store.toolTraces)
		}
		assertSingleOutputContractMetric(t, service, "unsupported", "strict_unsupported")
	})

	t.Run("ADK provider", func(t *testing.T) {
		service, store, _ := newOutputContractDirectRuntime(
			t,
			domainagentskill.OutputModeStrict,
			domainagentskill.RiskLevelLow,
		)
		provider := &fakeCandidateADKProvider{reply: "must not run"}
		service.provider = provider
		service.agentRuntime = agentRuntimeADK
		service.jobs = &fakeHRJobClient{}
		store.agentConfigs[0].ToolBindings = []*pb.AgentToolBindingInfo{{
			ToolName: "get_job_list", IsEnabled: true,
		}}

		_, err := service.Chat(context.Background(), &pb.ChatRequest{
			HrId: 77, Message: "draft a candidate outreach note", ModelId: 1,
			AgentSkillVersionIds: []int64{101},
		})
		if status.Code(err) != codes.FailedPrecondition ||
			provider.adkCalls != 0 || len(store.toolTraces) != 0 {
			t.Fatalf("strict ADK error=%v provider calls=%d tool traces=%#v", err, provider.adkCalls, store.toolTraces)
		}
		assertSingleOutputContractMetric(t, service, "unsupported", "strict_unsupported")
	})
}

func TestHRAdvisoryContractCannotComeFromSupportingAndNoneIsUnchanged(t *testing.T) {
	governance := hrRuntimeGovernanceContext{
		SelectedAgentSkills: []hrRuntimeAgentSkill{
			{
				Name: "primary", CoreMarkdown: "PRIMARY_CORE", Included: true,
				CompositionRole: domainagentskill.CompositionRolePrimary,
				OutputContract:  domainagentskill.OutputContract{Mode: domainagentskill.OutputModeNone},
			},
			{
				Name: "support", CoreMarkdown: "SUPPORT_CORE", Included: true,
				CompositionRole:     domainagentskill.CompositionRoleSupporting,
				OutputContract:      domainagentskill.OutputContract{Mode: domainagentskill.OutputModeAdvisory, Schema: json.RawMessage(`{}`)},
				AdvisoryInstruction: "FORBIDDEN_SUPPORTING_OUTPUT_CONTRACT",
			},
		},
	}
	prompt := renderHRProviderPrompt(&pb.ChatRequest{Message: "hello"}, nil, ChatMessageRow{}, nil, governance)
	if !strings.Contains(prompt, "PRIMARY_CORE") || !strings.Contains(prompt, "SUPPORT_CORE") ||
		strings.Contains(prompt, "FORBIDDEN_SUPPORTING_OUTPUT_CONTRACT") ||
		strings.Contains(prompt, "### Advisory response structure") {
		t.Fatalf("supporting/none output contract changed prompt:\n%s", prompt)
	}
}

func TestHRFeatureOffIgnoresAdvisoryOutputContract(t *testing.T) {
	service, _, provider := newOutputContractDirectRuntime(
		t,
		domainagentskill.OutputModeAdvisory,
		domainagentskill.RiskLevelLow,
	)
	service.skillPackageV2Enabled = false
	response, err := service.Chat(context.Background(), &pb.ChatRequest{
		HrId: 77, Message: "resume screening feature off", ModelId: 1,
		AgentSkillVersionIds: []int64{101},
	})
	if err != nil || response.GetCode() != 0 || provider.calls != 1 {
		t.Fatalf("feature-off response=%#v err=%v provider calls=%d", response, err, provider.calls)
	}
	if strings.Contains(provider.prompts[0], "### Advisory response structure") ||
		strings.Contains(provider.prompts[0], advisorySchemaID) {
		t.Fatalf("feature-off prompt loaded advisory contract:\n%s", provider.prompts[0])
	}
	if strings.Contains(service.metrics.Prometheus(), "advisory_not_enforced") {
		t.Fatalf("feature-off emitted advisory metric:\n%s", service.metrics.Prometheus())
	}
}

func newOutputContractDirectRuntime(
	t *testing.T,
	mode domainagentskill.OutputMode,
	risk domainagentskill.RiskLevel,
) (*nativeAIService, *fakeAIStore, *fakeChatProvider) {
	t.Helper()
	base := newFakeAIStore()
	base.agentConfigs = []*pb.AgentConfigInfo{{
		Id: 10, Name: "hr", AgentType: hrRecruitingAgentType, PromptTemplateId: 20,
		IsDefault: true, IsEnabled: true,
	}}
	base.promptByID[20] = &pb.PromptTemplateInfo{
		Id: 20, AgentType: hrRecruitingAgentType, PromptRole: hrRuntimePromptRoleSystem,
		IsActive: true, Content: "system",
	}
	document := runtimeSkillVersionDocument(
		101,
		1,
		domainagentskill.CompositionRolePrimary,
		risk,
		"resume screening core",
	)
	setOutputContractDocument(&document, mode)
	base.agentSkillPackages = []embeddinginfra.AgentSkillRuntimePackage{
		runtimeSkillPackage(t, document, nil),
	}
	store := &releasedSkillFakeStore{
		fakeAIStore: base,
		resolution:  runtimeSkillModel([]int64{101}, CapabilitySkillRuntimePolicy{}),
	}
	provider := &fakeChatProvider{reply: "free text"}
	service := newNativeAIService(store, provider, nil, nil, nil)
	service.skillPackageV2Enabled = true
	service.metrics = observability.NewRegistry("ai-agent")
	return service, base, provider
}

func setOutputContractDocument(
	document *embeddinginfra.AgentSkillVersionEmbeddingDocument,
	mode domainagentskill.OutputMode,
) {
	document.Manifest.OutputContract = domainagentskill.OutputContract{Mode: mode}
	if mode == domainagentskill.OutputModeAdvisory || mode == domainagentskill.OutputModeStrict {
		document.Manifest.OutputContract.SchemaID = advisorySchemaID
		document.Manifest.OutputContract.Schema = append(json.RawMessage(nil), advisorySchema...)
	}
}

func assertAdvisoryPrompt(t *testing.T, prompt string) {
	t.Helper()
	for _, want := range []string{
		"resume screening core",
		"### Advisory response structure",
		"Natural-language output remains allowed",
		`"schema_id":"candidate-summary-v2"`,
		`"required":["summary","score"]`,
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("provider prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Count(prompt, "### Advisory response structure") != 1 {
		t.Fatalf("advisory instruction count=%d, want one:\n%s", strings.Count(prompt, "### Advisory response structure"), prompt)
	}
}

func advisoryInstructionFromPrompt(t *testing.T, prompt string) string {
	t.Helper()
	const marker = "### Advisory response structure"
	start := strings.Index(prompt, marker)
	if start < 0 {
		t.Fatalf("prompt has no advisory instruction:\n%s", prompt)
	}
	end := strings.Index(prompt[start:], "\n\nSelected Agent Skills:")
	if end < 0 {
		t.Fatalf("prompt has no advisory instruction boundary:\n%s", prompt)
	}
	return strings.TrimSpace(prompt[start : start+end])
}

func assertSingleOutputContractMetric(
	t *testing.T,
	service *nativeAIService,
	result string,
	reason string,
) {
	t.Helper()
	metrics := service.metrics.Prometheus()
	if strings.Count(metrics, "smart_recruit_agent_skill_output_validation_total{") != 1 ||
		!strings.Contains(metrics, `result="`+result+`",reason="`+reason+`"} 1`) {
		t.Fatalf("output-contract metric result=%s reason=%s was not emitted exactly once:\n%s", result, reason, metrics)
	}
}
