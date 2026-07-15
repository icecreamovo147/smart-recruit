package recruiting_intelligence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type promptLoadCall struct {
	agentType string
	role      string
}

func TestResumeFallbackObservationContainsClassificationNotProviderBody(t *testing.T) {
	store := &perRequestPromptStore{}
	provider := &failingObservationProvider{err: errors.New("private provider response with candidate@example.invalid")}
	observations := &observationCapture{}
	policy := NewRuntimePolicy(RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: true})
	runtime := NewRuntimeWithObserver(NewPromptLoader(store), provider, observations, policy)
	result, err := NewResumeProfileExtractor(runtime, policy).Extract(context.Background(), ResumeSource{ResumeID: 1, UserID: 2, ParsedText: "Synthetic Person\nGo engineer"})
	if err != nil || !result.FallbackUsed || result.FallbackKind != ResumeExtractionProvider {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(observations.events) != 3 {
		t.Fatalf("events=%+v, want prompt, provider error, fallback outcome", observations.events)
	}
	fallback := observations.events[2]
	if fallback.Stage != "resume_profile_extraction" || fallback.Outcome != "success" || fallback.Fallback != string(ResumeExtractionProvider) {
		t.Fatalf("fallback event=%+v", fallback)
	}
	for _, event := range observations.events {
		if event.Terminal {
			t.Fatalf("component stage must not claim operation terminal success: %+v", event)
		}
		if event.PromptName == "private provider response with candidate@example.invalid" || event.ModelName == "private provider response with candidate@example.invalid" {
			t.Fatalf("provider body leaked into event: %+v", event)
		}
	}
}

type failingObservationProvider struct {
	err error
}

func (p *failingObservationProvider) CompleteStructured(context.Context, string, string) (StructuredCompletionResult, error) {
	return StructuredCompletionResult{}, p.err
}

type perRequestPromptStore struct {
	calls []promptLoadCall
}

func (s *perRequestPromptStore) LoadActiveRecruitingPrompt(_ context.Context, agentType, role string) (PromptDescriptor, error) {
	s.calls = append(s.calls, promptLoadCall{agentType: agentType, role: role})
	return PromptDescriptor{
		ID: int64(len(s.calls)), Name: agentType + "-active", Version: int32(len(s.calls)),
		AgentType: agentType, Role: role, Content: "system-body-" + agentType,
	}, nil
}

type roleCaptureProvider struct {
	systemPrompts []string
	userPrompts   []string
}

func (p *roleCaptureProvider) CompleteStructured(_ context.Context, systemPrompt, userPrompt string) (StructuredCompletionResult, error) {
	p.systemPrompts = append(p.systemPrompts, systemPrompt)
	p.userPrompts = append(p.userPrompts, userPrompt)
	return StructuredCompletionResult{Content: `{}`, ModelName: "safe-model"}, nil
}

type observationCapture struct {
	events []Observation
}

func (c *observationCapture) ObserveRecruitingRuntime(_ context.Context, event Observation) {
	c.events = append(c.events, event)
}

func TestRuntimeLoadsEveryRecruitingSystemPromptOnEveryRequest(t *testing.T) {
	store := &perRequestPromptStore{}
	provider := &roleCaptureProvider{}
	observations := &observationCapture{}
	policy := NewRuntimePolicy(RuntimePolicyConfig{
		StructuredResumeParse: true, CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: true,
	})
	runtime := NewRuntimeWithObserver(NewPromptLoader(store), provider, observations, policy)
	agentTypes := []string{AgentTypeResumeProfileExtractor, AgentTypeJobRequirementExtractor, AgentTypeCandidateMatchEvaluator}
	for request := 1; request <= 2; request++ {
		for _, agentType := range agentTypes {
			ctx := WithObservationMetadata(context.Background(), "opaque-request-1", "resume", 8001)
			if _, err := runtime.Complete(ctx, agentType, fmt.Sprintf("private-user-%d", request)); err != nil {
				t.Fatalf("Complete(%s, request=%d): %v", agentType, request, err)
			}
		}
	}
	if len(store.calls) != 6 || len(provider.systemPrompts) != 6 || len(provider.userPrompts) != 6 {
		t.Fatalf("calls = prompt:%d system:%d user:%d, want 6 each", len(store.calls), len(provider.systemPrompts), len(provider.userPrompts))
	}
	for index, call := range store.calls {
		wantAgentType := agentTypes[index%len(agentTypes)]
		if call.agentType != wantAgentType || call.role != PromptRoleSystem {
			t.Fatalf("prompt call %d = agent_type:%q role:%q, want %q/system", index, call.agentType, call.role, wantAgentType)
		}
		if provider.systemPrompts[index] != "system-body-"+wantAgentType {
			t.Fatalf("system prompt %d was not loaded prompt content", index)
		}
		if provider.userPrompts[index] == provider.systemPrompts[index] {
			t.Fatalf("request %d mixed System and User messages", index)
		}
	}
	if len(observations.events) != 12 {
		t.Fatalf("observations = %d, want prompt_load + structured_completion for each request", len(observations.events))
	}
	for index := 0; index < len(observations.events); index += 2 {
		load, completion := observations.events[index], observations.events[index+1]
		if load.Stage != "prompt_load" || completion.Stage != "structured_completion" || load.Outcome != "success" || completion.Outcome != "success" {
			t.Fatalf("events %d/%d = %#v / %#v", index, index+1, load, completion)
		}
		if load.Terminal || completion.Terminal {
			t.Fatalf("stage events must remain terminal=false: %#v / %#v", load, completion)
		}
		if load.PromptID <= 0 || load.PromptName == "" || load.PromptVersion <= 0 || completion.ModelName != observationCorrelation("model-name", "model", "safe-model", maxObservationModelNameBytes) {
			t.Fatalf("safe identity/model metadata missing: %#v / %#v", load, completion)
		}
		wantRequestID := observationCorrelation("request-id", "rid", "opaque-request-1", 0)
		if load.RequestID != wantRequestID || load.ResourceType != "resume" || load.ResourceID != 8001 || completion.RequestID != wantRequestID {
			t.Fatalf("safe request/resource metadata missing: %#v / %#v", load, completion)
		}
	}
}

func TestObservationCountsAreBoundedBeforeObserver(t *testing.T) {
	capture := &observationCapture{}
	runtime := NewRuntimeWithObserver(nil, nil, capture)
	runtime.Observe(context.Background(), Observation{InputCount: 5001, OutputCount: -1, EvidenceCount: 1001, RequirementCount: 1000})
	if len(capture.events) != 1 {
		t.Fatalf("events=%d, want 1", len(capture.events))
	}
	event := capture.events[0]
	if event.InputCount != 1000 || event.OutputCount != 0 || event.EvidenceCount != 1000 || event.RequirementCount != 1000 {
		t.Fatalf("bounded counts=%+v", event)
	}
}

func TestObservationBoundaryRejectsSensitiveAndUnboundedStringValues(t *testing.T) {
	marker := "candidate@example.invalid\nresume-body raw-response evidence-body apikey-marker"
	capture := &observationCapture{}
	runtime := NewRuntimeWithObserver(nil, nil, capture)
	ctx := WithObservationMetadata(context.Background(), marker, marker, -99)
	runtime.Observe(ctx, Observation{
		Operation: marker, RequestID: marker, ResourceType: marker, ResourceID: -1,
		Stage: marker, Category: marker, AgentType: marker, PromptID: -2, PromptName: marker,
		PromptVersion: -3, ModelName: marker, Fallback: marker, Outcome: marker,
		ParserVersion: marker, ScorerVersion: strings.Repeat("v", maxObservationVersionBytes+1),
		InputCount: -1, OutputCount: 1001, EvidenceCount: 5000, RequirementCount: -5,
		Duration: 48 * time.Hour,
	})
	if len(capture.events) != 1 {
		t.Fatalf("events=%d, want 1", len(capture.events))
	}
	event := capture.events[0]
	if event.Operation != "unknown" || event.RequestID == "" || strings.Contains(event.RequestID, marker) || event.ResourceType != "unknown" || event.ResourceID != 0 ||
		event.Stage != "unknown" || event.Category != "unknown" || event.AgentType != "unknown" || event.PromptID != 0 ||
		event.PromptName != "" || event.PromptVersion != 0 || event.ModelName != "" || event.Fallback != "unknown" ||
		event.Outcome != "unknown" || event.ParserVersion != "" || event.ScorerVersion == "" || strings.Contains(event.ScorerVersion, marker) {
		t.Fatalf("unsafe strings or classifications survived normalization: %+v", event)
	}
	if event.InputCount != 0 || event.OutputCount != 1000 || event.EvidenceCount != 1000 || event.RequirementCount != 0 || event.Duration != maxObservationDuration {
		t.Fatalf("unsafe numeric values survived normalization: %+v", event)
	}
	if strings.Contains(fmt.Sprintf("%+v", event), marker) {
		t.Fatalf("privacy marker survived observation boundary: %+v", event)
	}
}

func TestObservationBoundaryPreservesSafeCorrelationsAndClassifications(t *testing.T) {
	want := Observation{
		Operation: AgentTypeResumeProfileExtractor, RequestID: "req_01234567-89ab", ResourceType: "resume", ResourceID: 8001,
		Stage: "structured_completion", Category: "provider_failure", AgentType: AgentTypeResumeProfileExtractor,
		PromptID: 47, PromptName: "Resume Profile Extractor System Prompt zh-CN v2", PromptVersion: 2,
		ModelName: "openai/gpt-4.1-mini", Fallback: "provider", Outcome: "error",
		ParserVersion: ResumeLLMParserVersion, ScorerVersion: CandidateMatchScorerVersion,
		InputCount: 1, OutputCount: 2, EvidenceCount: 3, RequirementCount: 4, Duration: 23 * time.Millisecond,
	}
	got := NormalizeObservation(want)
	if got.Operation != want.Operation || got.ResourceType != want.ResourceType || got.ResourceID != want.ResourceID ||
		got.Stage != want.Stage || got.Category != want.Category || got.AgentType != want.AgentType || got.PromptID != want.PromptID ||
		got.PromptVersion != want.PromptVersion || got.Fallback != want.Fallback || got.Outcome != want.Outcome ||
		got.ParserVersion != want.ParserVersion || got.ScorerVersion != want.ScorerVersion {
		t.Fatalf("valid classifications changed:\n got=%+v\nwant=%+v", got, want)
	}
	for field, value := range map[string]string{"request": got.RequestID, "prompt": got.PromptName, "model": got.ModelName} {
		if value == "" || len(value) > 40 {
			t.Fatalf("%s correlation=%q is empty or unbounded", field, value)
		}
	}
	if got.RequestID == want.RequestID || got.PromptName == want.PromptName || got.ModelName == want.ModelName {
		t.Fatalf("reversible externally controlled identity survived: %+v", got)
	}
}

func TestObservationRequestIDCorrelationIsStableIdempotentAndNonReversible(t *testing.T) {
	values := []string{
		"13800138000", "secret-resume-body", "sk-abcdefghijklmnopqrstuvwxyz", "550e8400-e29b-41d4-a716-446655440000",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV", "normal-request-id", "line\ncontrol", "请求编号-甲", strings.Repeat("x", 4096),
	}
	seen := map[string]string{}
	for _, value := range values {
		first := NormalizeObservation(Observation{RequestID: value})
		second := NormalizeObservation(first)
		third := NormalizeObservation(Observation{RequestID: value})
		if first != second || first.RequestID != third.RequestID {
			t.Fatalf("request correlation is not stable/idempotent for input length %d", len(value))
		}
		if first.RequestID == "" || first.RequestID == value || strings.Contains(first.RequestID, value) || len(first.RequestID) > 40 {
			t.Fatalf("request correlation is reversible or unbounded for input length %d: %q", len(value), first.RequestID)
		}
		if prior, exists := seen[first.RequestID]; exists && prior != value {
			t.Fatalf("distinct request IDs collided: %q and %q", prior, value)
		}
		seen[first.RequestID] = value
	}
}

func TestObservationIdentityHonorsPersistedContractsWithoutRawLeak(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		value  string
		maxLen int
	}{
		{name: "prompt max bytes", field: "prompt", value: strings.Repeat("界", 84) + "v2.1", maxLen: maxObservationPromptNameBytes},
		{name: "model max bytes", field: "model", value: strings.Repeat("模", 40) + "/v2.1", maxLen: maxObservationModelNameBytes},
		{name: "prompt punctuation", field: "prompt", value: "简历画像｜系统提示（中文）—v2.1", maxLen: maxObservationPromptNameBytes},
		{name: "malicious safe alphabet", field: "model", value: "secret-resume-body", maxLen: maxObservationModelNameBytes},
		{name: "overlong prompt", field: "prompt", value: strings.Repeat("p", maxObservationPromptNameBytes+1), maxLen: maxObservationPromptNameBytes},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := Observation{}
			if tc.field == "prompt" {
				input.PromptName = tc.value
			} else {
				input.ModelName = tc.value
			}
			got := NormalizeObservation(input)
			value := got.PromptName
			if tc.field == "model" {
				value = got.ModelName
			}
			if value == "" || value == tc.value || strings.Contains(value, tc.value) || len(value) > 40 {
				t.Fatalf("identity correlation=%q for input bytes=%d max=%d", value, len(tc.value), tc.maxLen)
			}
		})
	}
	for _, value := range []string{"bad\ncontrol", string([]byte{0xff, 0xfe})} {
		got := NormalizeObservation(Observation{PromptName: value, ModelName: value, ParserVersion: value, ScorerVersion: value})
		if got.PromptName != "" || got.ModelName != "" || got.ParserVersion != "" || got.ScorerVersion != "" {
			t.Fatalf("control/invalid UTF-8 identity did not fail closed: %+v", got)
		}
	}
}
