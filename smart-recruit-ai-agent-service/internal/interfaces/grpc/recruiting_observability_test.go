package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	"smart-recruit-platform-go/errs"
	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

func TestRecruitingRuntimeObserverLogsOnlyPrivacyAllowList(t *testing.T) {
	core, captured := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	marker := "candidate@example.invalid secret-resume raw-model-output evidence-body api-key"
	event := recruitingruntime.Observation{
		Operation: "resume_profile", RequestID: "opaque-request-1", ResourceType: "resume", ResourceID: 8001,
		Stage: "structured_completion", Terminal: false, Category: "success", AgentType: recruitingruntime.AgentTypeResumeProfileExtractor,
		PromptID: 47, PromptName: "resume_profile_extractor", PromptVersion: 2,
		ModelName: "safe-model", Fallback: "none", Outcome: "success", ParserVersion: "parser-v1", ScorerVersion: "scorer-v1",
		InputCount: 1, OutputCount: 2, EvidenceCount: 3, RequirementCount: 4, Duration: 23 * time.Millisecond,
	}
	recruitingRuntimeObserver{log: logger}.ObserveRecruitingRuntime(context.WithValue(context.Background(), privacyMarkerContextKey{}, marker), event)
	entries := captured.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	encoded, err := json.Marshal(entries[0].ContextMap())
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded) + entries[0].Message
	if strings.Contains(text, marker) || strings.Contains(text, "candidate@example.invalid") || strings.Contains(text, "secret-resume") || strings.Contains(text, "raw-model-output") || strings.Contains(text, "evidence-body") || strings.Contains(text, "api-key") {
		t.Fatalf("privacy marker leaked into log: %s", text)
	}
	wantKeys := map[string]bool{
		"operation": true, "request_id": true, "resource_type": true, "resource_id": true,
		"terminal": true, "category": true,
		"stage": true, "agent_type": true, "prompt_id": true, "prompt_name": true, "prompt_version": true,
		"model": true, "fallback": true, "outcome": true, "duration_ms": true,
		"parser_version": true, "scorer_version": true, "input_count": true, "output_count": true,
		"evidence_count": true, "requirement_count": true,
	}
	for key := range entries[0].ContextMap() {
		if !wantKeys[key] {
			t.Fatalf("unexpected recruiting log field %q", key)
		}
		delete(wantKeys, key)
	}
	if len(wantKeys) != 0 {
		t.Fatalf("missing recruiting log fields: %v", wantKeys)
	}
}

func TestRecruitingRuntimeObserverFailClosedNormalizesEveryStringField(t *testing.T) {
	core, captured := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	marker := "candidate@example.invalid\nresume-body raw-model-output evidence-body apikey-marker"
	event := recruitingruntime.Observation{
		Operation: marker, RequestID: marker, ResourceType: marker, ResourceID: -1,
		Stage: marker, Category: marker, AgentType: marker, PromptID: -2, PromptName: marker,
		PromptVersion: -3, ModelName: marker, Fallback: marker, Outcome: marker,
		ParserVersion: marker, ScorerVersion: marker,
		InputCount: -1, OutputCount: 1001, EvidenceCount: 1002, RequirementCount: -2,
		Duration: -time.Second,
	}
	recruitingRuntimeObserver{log: logger}.ObserveRecruitingRuntime(context.Background(), event)
	entries := captured.All()
	if len(entries) != 1 {
		t.Fatalf("log entries=%d, want 1", len(entries))
	}
	encoded, err := json.Marshal(entries[0].ContextMap())
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded) + entries[0].Message
	if strings.Contains(text, marker) || strings.Contains(text, "candidate@example.invalid") || strings.Contains(text, "resume-body") || strings.Contains(text, "raw-model-output") || strings.Contains(text, "evidence-body") || strings.Contains(text, "apikey-marker") {
		t.Fatalf("privacy marker survived log boundary: %s", text)
	}
	fields := entries[0].ContextMap()
	for _, key := range []string{"operation", "resource_type", "stage", "category", "agent_type", "fallback", "outcome"} {
		if fields[key] != "unknown" {
			t.Fatalf("%s=%v, want fail-closed unknown", key, fields[key])
		}
	}
	for _, key := range []string{"prompt_name", "model", "parser_version", "scorer_version"} {
		if fields[key] != "" {
			t.Fatalf("%s=%v, want rejected empty identity", key, fields[key])
		}
	}
	if requestID := fmt.Sprint(fields["request_id"]); requestID == "" || strings.Contains(requestID, "candidate@example.invalid") || len(requestID) > 40 {
		t.Fatalf("request ID correlation is empty, reversible, or unbounded: %q", requestID)
	}
	for key, want := range map[string]string{
		"resource_id": "0", "prompt_id": "0", "prompt_version": "0", "input_count": "0",
		"output_count": "1000", "evidence_count": "1000", "requirement_count": "0", "duration_ms": "0",
	} {
		if fmt.Sprint(fields[key]) != want {
			t.Fatalf("%s=%v, want %s", key, fields[key], want)
		}
	}
	if fields["terminal"] != false {
		t.Fatalf("numeric normalization failed: %+v", fields)
	}
}

func TestRecruitingRuntimeObserverNeverLogsSafeAlphabetSecretsReversibly(t *testing.T) {
	values := []string{
		"13800138000",
		"secret-resume-body",
		"sk-abcdefghijklmnopqrstuvwxyz",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.signature",
		"550e8400-e29b-41d4-a716-446655440000",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"normal-request-id",
	}
	for _, value := range values {
		t.Run(value[:min(len(value), 12)], func(t *testing.T) {
			core, captured := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)
			event := recruitingruntime.Observation{
				Operation: "resume_profile", RequestID: value, ResourceType: "resume", ResourceID: 8001,
				Stage: "operation", Terminal: true, Category: "success", Outcome: "success",
				PromptName: value, ModelName: value, ParserVersion: value, ScorerVersion: value,
			}
			recruitingRuntimeObserver{log: logger}.ObserveRecruitingRuntime(context.Background(), event)
			entries := captured.All()
			if len(entries) != 1 {
				t.Fatalf("entries=%d, want 1", len(entries))
			}
			encoded, err := json.Marshal(entries[0].ContextMap())
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), value) {
				t.Fatalf("raw safe-alphabet value leaked into log: %s", encoded)
			}
			requestID := fmt.Sprint(entries[0].ContextMap()["request_id"])
			if requestID == "" || len(requestID) > 40 {
				t.Fatalf("bounded stable request correlation missing: %q", requestID)
			}
		})
	}
}

func TestRecruitingNativeObservationRejectsPropagatedExternalRequestID(t *testing.T) {
	marker := "candidate@example.invalid\nresume-body raw-model-output evidence-body apikey-marker"
	store := newFakeRecruitingGenerationStore()
	store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
	store.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Synthetic Go engineer"}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true})
	service := resumeProfileTestService(store, &capturingResumeProvider{content: completeResumeProfileJSON()}, policy)
	capture := &recruitingObservationCapture{}
	service.observer = capture
	ctx := context.WithValue(recruitingAuthContext(), platformmetadata.KeyRequestID, marker)
	resp, err := service.ParseResumeProfile(ctx, &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
	if err != nil || resp.GetCode() != errs.OK {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
	if len(capture.events) == 0 {
		t.Fatal("expected observation events")
	}
	var correlation string
	for _, event := range capture.events {
		if event.RequestID == "" || event.RequestID == marker || strings.Contains(event.RequestID, marker) || len(event.RequestID) > 40 {
			t.Fatalf("external request ID survived native observation boundary: %+v", event)
		}
		if correlation == "" {
			correlation = event.RequestID
		} else if event.RequestID != correlation {
			t.Fatalf("request correlation changed within operation: first=%q event=%+v", correlation, event)
		}
	}
}

type privacyMarkerContextKey struct{}

type recruitingObservationCapture struct {
	events []recruitingruntime.Observation
}

func recruitingObservationContext() context.Context {
	return context.WithValue(recruitingAuthContext(), platformmetadata.KeyRequestID, "opaque-request-1")
}

func (c *recruitingObservationCapture) ObserveRecruitingRuntime(_ context.Context, event recruitingruntime.Observation) {
	c.events = append(c.events, event)
}

func assertSingleTerminal(t *testing.T, capture *recruitingObservationCapture, operation, category string) {
	t.Helper()
	var terminals []recruitingruntime.Observation
	for _, event := range capture.events {
		if event.Terminal {
			terminals = append(terminals, event)
		}
	}
	if len(terminals) != 1 {
		t.Fatalf("terminal events=%+v, want exactly one (all=%+v)", terminals, capture.events)
	}
	terminal := terminals[0]
	if terminal.Operation != operation || terminal.Stage != "operation" || terminal.Category != category {
		t.Fatalf("terminal=%+v, want operation=%s category=%s", terminal, operation, category)
	}
	wantRequestID := recruitingruntime.NormalizeObservation(recruitingruntime.Observation{RequestID: "opaque-request-1"}).RequestID
	if terminal.RequestID != wantRequestID || terminal.ResourceID <= 0 || terminal.ResourceType == "" || terminal.Duration < 0 {
		t.Fatalf("terminal safe metadata incomplete: %+v", terminal)
	}
	if category == "persistence_failure" && terminal.ParserVersion == "" && terminal.ScorerVersion == "" {
		t.Fatalf("persistence terminal lost completed parser/scorer provenance: %+v", terminal)
	}
}

func TestRecruitingNativeOperationsEmitExactlyOneTerminalOutcome(t *testing.T) {
	t.Run("resume primary success", func(t *testing.T) {
		store := newFakeRecruitingGenerationStore()
		store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
		store.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Synthetic Go engineer"}
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true})
		service := resumeProfileTestService(store, &capturingResumeProvider{content: completeResumeProfileJSON()}, policy)
		capture := &recruitingObservationCapture{}
		service.observer = capture
		resp, err := service.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
		if err != nil || resp.GetCode() != errs.OK {
			t.Fatalf("resp=%+v err=%v", resp, err)
		}
		assertSingleTerminal(t, capture, "resume_profile", "success")
		if capture.events[len(capture.events)-1].ParserVersion == "" || capture.events[len(capture.events)-1].OutputCount <= 0 {
			t.Fatalf("terminal version/counts missing: %+v", capture.events[len(capture.events)-1])
		}
	})

	t.Run("resume fallback success", func(t *testing.T) {
		store := newFakeRecruitingGenerationStore()
		store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
		store.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Synthetic Go engineer"}
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: true})
		service := resumeProfileTestService(store, &capturingResumeProvider{content: `{"skills":["Go"]}`}, policy)
		capture := &recruitingObservationCapture{}
		service.observer = capture
		resp, err := service.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
		if err != nil || resp.GetCode() != errs.OK {
			t.Fatalf("resp=%+v err=%v", resp, err)
		}
		assertSingleTerminal(t, capture, "resume_profile", "fallback_success")
	})

	t.Run("resume persistence failure", func(t *testing.T) {
		base := newFakeRecruitingGenerationStore()
		base.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
		base.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Synthetic Go engineer"}
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true})
		service := resumeProfileTestService(&failingResumeSaveStore{fakeRecruitingGenerationStore: base, saveErr: errors.New("private database body")}, &capturingResumeProvider{content: completeResumeProfileJSON()}, policy)
		capture := &recruitingObservationCapture{}
		service.observer = capture
		_, _ = service.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
		assertSingleTerminal(t, capture, "resume_profile", "persistence_failure")
	})

	t.Run("candidate enhanced success", func(t *testing.T) {
		store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
		store.matchSources[7001] = sampleRecruitingMatchSource()
		provider := &candidateMatchStructuredProvider{jobReply: `{"profile_version":"job-requirement-profile-v1","requirements":[{"id":"go","category":"core_skill","label":"Go 开发能力","description":"岗位核心开发能力","priority":"must_have","weight":1,"knockout":false,"aliases":["Go"]}]}`}
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true})
		service := candidateObservationService(store, provider, policy)
		capture := &recruitingObservationCapture{}
		service.observer = capture
		resp, err := service.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		if err != nil || resp.GetCode() != errs.OK {
			t.Fatalf("resp=%+v err=%v", resp, err)
		}
		assertSingleTerminal(t, capture, "candidate_match", "success")
		terminal := capture.events[len(capture.events)-1]
		if terminal.ScorerVersion == "" || terminal.RequirementCount <= 0 || terminal.EvidenceCount < 0 {
			t.Fatalf("terminal scorer/counts missing: %+v", terminal)
		}
	})

	t.Run("candidate fallback success", func(t *testing.T) {
		store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
		store.matchSources[7001] = sampleRecruitingMatchSource()
		provider := &candidateMatchStructuredProvider{jobReply: `{"overall_score":100}`}
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: true})
		service := candidateObservationService(store, provider, policy)
		capture := &recruitingObservationCapture{}
		service.observer = capture
		_, _ = service.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		assertSingleTerminal(t, capture, "candidate_match", "fallback_success")
	})

	t.Run("candidate schema failure", func(t *testing.T) {
		store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
		store.matchSources[7001] = sampleRecruitingMatchSource()
		provider := &candidateMatchStructuredProvider{jobReply: `{"overall_score":100}`}
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true})
		service := candidateObservationService(store, provider, policy)
		capture := &recruitingObservationCapture{}
		service.observer = capture
		_, _ = service.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		assertSingleTerminal(t, capture, "candidate_match", "schema_failure")
	})

	t.Run("candidate provider failure", func(t *testing.T) {
		store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
		store.matchSources[7001] = sampleRecruitingMatchSource()
		provider := &candidateMatchStructuredProvider{jobErr: errors.New("private provider response")}
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true})
		service := candidateObservationService(store, provider, policy)
		capture := &recruitingObservationCapture{}
		service.observer = capture
		_, _ = service.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		assertSingleTerminal(t, capture, "candidate_match", "provider_failure")
	})

	t.Run("candidate aggregation dependency failure", func(t *testing.T) {
		store := newFakeRecruitingGenerationStore()
		store.matchSources[7001] = sampleRecruitingMatchSource()
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true})
		capture := &recruitingObservationCapture{}
		service := nativeRecruitingIntelligenceService{
			store: store, policy: policy, observer: capture,
			auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}},
			applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
		}
		_, _ = service.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		assertSingleTerminal(t, capture, "candidate_match", "configuration_failure")
	})

	t.Run("candidate timeout", func(t *testing.T) {
		store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
		store.matchSources[7001] = sampleRecruitingMatchSource()
		provider := &candidateMatchStructuredProvider{block: true}
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, CandidateMatchTimeout: 30 * time.Millisecond})
		service := candidateObservationService(store, provider, policy)
		capture := &recruitingObservationCapture{}
		service.observer = capture
		_, _ = service.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		assertSingleTerminal(t, capture, "candidate_match", "timeout")
	})

	t.Run("candidate source failure", func(t *testing.T) {
		store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
		provider := &candidateMatchStructuredProvider{}
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true})
		service := candidateObservationService(store, provider, policy)
		capture := &recruitingObservationCapture{}
		service.observer = capture
		_, _ = service.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		assertSingleTerminal(t, capture, "candidate_match", "not_found")
	})

	t.Run("candidate persistence failure", func(t *testing.T) {
		base := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
		base.matchSources[7001] = sampleRecruitingMatchSource()
		store := &failingCandidateObservationStore{structuredCandidateMatchStore: base}
		provider := &candidateMatchStructuredProvider{}
		policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: false})
		service := candidateObservationService(store, provider, policy)
		capture := &recruitingObservationCapture{}
		service.observer = capture
		_, _ = service.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		assertSingleTerminal(t, capture, "candidate_match", "persistence_failure")
	})
}

func TestRecruitingNativeMethodBoundaryFinalizesEveryEarlyReturn(t *testing.T) {
	allowedApplication := func() *fakeApplicationOwnerClient {
		return &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}}
	}
	allowedAuth := func() *fakeAuthClient {
		return &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}}
	}
	resumeCompatibility := func() *fakeRecruitingReadStore {
		store := newFakeRecruitingReadStore()
		store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
		return store
	}
	candidateCompatibility := func() *fakeRecruitingReadStore { return newFakeRecruitingReadStore() }

	tests := []struct {
		name      string
		operation string
		category  string
		invoke    func(nativeRecruitingIntelligenceService)
		service   func() nativeRecruitingIntelligenceService
	}{
		{name: "resume nil request", operation: "resume_profile", category: "domain_validation_failure", service: func() nativeRecruitingIntelligenceService { return nativeRecruitingIntelligenceService{} }, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), nil)
		}},
		{name: "resume invalid identifiers", operation: "resume_profile", category: "domain_validation_failure", service: func() nativeRecruitingIntelligenceService { return nativeRecruitingIntelligenceService{} }, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{})
		}},
		{name: "resume missing store", operation: "resume_profile", category: "configuration_failure", service: func() nativeRecruitingIntelligenceService { return nativeRecruitingIntelligenceService{} }, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{ResumeId: 8001})
		}},
		{name: "resume application owner missing", operation: "resume_profile", category: "configuration_failure", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: newFakeRecruitingGenerationStore()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "resume application owner nil response", operation: "resume_profile", category: "configuration_failure", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: newFakeRecruitingGenerationStore(), applications: &fakeApplicationOwnerClient{}}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "resume application authorization forbidden", operation: "resume_profile", category: "authorization_failure", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: newFakeRecruitingGenerationStore(), applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.ErrForbidden}}}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "resume application not found", operation: "resume_profile", category: "not_found", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: newFakeRecruitingGenerationStore(), applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: 404}}}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "resume application owner error", operation: "resume_profile", category: "source_failure", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: newFakeRecruitingGenerationStore(), applications: &fakeApplicationOwnerClient{err: errors.New("private owner body")}}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "resume AI authorization dependency missing", operation: "resume_profile", category: "configuration_failure", service: func() nativeRecruitingIntelligenceService {
			store := newFakeRecruitingGenerationStore()
			store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
			return nativeRecruitingIntelligenceService{store: store, applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
		}},
		{name: "resume structured runtime missing", operation: "resume_profile", category: "configuration_failure", service: func() nativeRecruitingIntelligenceService {
			store := newFakeRecruitingGenerationStore()
			store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
			store.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, ParsedText: "Synthetic"}
			return nativeRecruitingIntelligenceService{store: store, policy: recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true}), auth: allowedAuth(), applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
		}},
		{name: "resume capability disabled", operation: "resume_profile", category: "policy_failure", service: func() nativeRecruitingIntelligenceService {
			store := newFakeRecruitingGenerationStore()
			store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
			store.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, ParsedText: "Synthetic"}
			return nativeRecruitingIntelligenceService{store: store, policy: recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{}), auth: allowedAuth(), applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
		}},
		{name: "resume compatibility success", operation: "resume_profile", category: "success", service: func() nativeRecruitingIntelligenceService {
			store := resumeCompatibility()
			store.currentProfileByResumeID[8001] = RecruitingResumeProfileRow{ID: 6001, ResumeID: 8001}
			store.resumeSnapshots[6001] = sampleRecruitingResumeSnapshot(6001, 8001, 3001)
			return nativeRecruitingIntelligenceService{store: store, applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
		}},
		{name: "resume compatibility not found", operation: "resume_profile", category: "not_found", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: resumeCompatibility(), applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
		}},
		{name: "resume compatibility read error", operation: "resume_profile", category: "source_failure", service: func() nativeRecruitingIntelligenceService {
			store := resumeCompatibility()
			store.err = errors.New("private read body")
			return nativeRecruitingIntelligenceService{store: store, applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.ParseResumeProfile(recruitingObservationContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
		}},
		{name: "candidate nil request", operation: "candidate_match", category: "domain_validation_failure", service: func() nativeRecruitingIntelligenceService { return nativeRecruitingIntelligenceService{} }, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), nil)
		}},
		{name: "candidate missing store", operation: "candidate_match", category: "configuration_failure", service: func() nativeRecruitingIntelligenceService { return nativeRecruitingIntelligenceService{} }, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{ApplicationId: 7001})
		}},
		{name: "candidate application owner missing", operation: "candidate_match", category: "configuration_failure", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: candidateCompatibility()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "candidate authorization forbidden", operation: "candidate_match", category: "authorization_failure", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: candidateCompatibility(), applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.ErrForbidden}}}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "candidate application not found", operation: "candidate_match", category: "not_found", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: candidateCompatibility(), applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: 404}}}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "candidate application owner error", operation: "candidate_match", category: "source_failure", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: candidateCompatibility(), applications: &fakeApplicationOwnerClient{err: errors.New("private owner body")}}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "candidate AI authorization dependency missing", operation: "candidate_match", category: "configuration_failure", service: func() nativeRecruitingIntelligenceService {
			store := newFakeRecruitingGenerationStore()
			return nativeRecruitingIntelligenceService{store: store, policy: recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true}), applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "candidate structured runtime missing", operation: "candidate_match", category: "configuration_failure", service: func() nativeRecruitingIntelligenceService {
			store := newFakeRecruitingGenerationStore()
			store.matchSources[7001] = sampleRecruitingMatchSource()
			return nativeRecruitingIntelligenceService{store: store, policy: recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true}), auth: allowedAuth(), applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "candidate structured path disabled deterministic success", operation: "candidate_match", category: "success", service: func() nativeRecruitingIntelligenceService {
			store := newFakeRecruitingGenerationStore()
			store.matchSources[7001] = sampleRecruitingMatchSource()
			return nativeRecruitingIntelligenceService{store: store, policy: recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: false}), auth: allowedAuth(), applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "candidate capability disabled compatibility success", operation: "candidate_match", category: "success", service: func() nativeRecruitingIntelligenceService {
			store := candidateCompatibility()
			store.latestMatchByApp[7001] = sampleCandidateMatchSnapshot(9101, 7001, 1, 80)
			return nativeRecruitingIntelligenceService{store: store, applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
		{name: "candidate compatibility not found by run", operation: "candidate_match", category: "not_found", service: func() nativeRecruitingIntelligenceService {
			return nativeRecruitingIntelligenceService{store: candidateCompatibility(), applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, AgentRunId: 42})
		}},
		{name: "candidate compatibility read error", operation: "candidate_match", category: "source_failure", service: func() nativeRecruitingIntelligenceService {
			store := candidateCompatibility()
			store.err = errors.New("private read body")
			return nativeRecruitingIntelligenceService{store: store, applications: allowedApplication()}
		}, invoke: func(s nativeRecruitingIntelligenceService) {
			_, _ = s.EvaluateCandidateMatch(recruitingObservationContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			capture := &recruitingObservationCapture{}
			service := tc.service()
			service.observer = capture
			tc.invoke(service)
			var terminals []recruitingruntime.Observation
			for _, event := range capture.events {
				if event.Terminal {
					terminals = append(terminals, event)
				}
			}
			wantOutcome := "error"
			if tc.category == "success" || tc.category == "fallback_success" {
				wantOutcome = "success"
			}
			if len(terminals) != 1 || terminals[0].Operation != tc.operation || terminals[0].Category != tc.category || terminals[0].Outcome != wantOutcome {
				t.Fatalf("terminal events=%+v, want exactly one %s/%s outcome=%s (all=%+v)", terminals, tc.operation, tc.category, wantOutcome, capture.events)
			}
		})
	}
}

type candidateObservationStore interface {
	recruitingReadStore
	recruitingruntime.PromptStore
}

type failingCandidateObservationStore struct {
	*structuredCandidateMatchStore
}

func (s *failingCandidateObservationStore) SaveRecruitingCandidateMatchDraft(context.Context, RecruitingCandidateMatchDraft) (RecruitingCandidateMatchSnapshot, error) {
	return RecruitingCandidateMatchSnapshot{}, errors.New("private database response")
}

func candidateObservationService(store candidateObservationStore, provider *candidateMatchStructuredProvider, policy recruitingruntime.RuntimePolicy) nativeRecruitingIntelligenceService {
	return nativeRecruitingIntelligenceService{
		store: store, provider: provider, structured: recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(store), provider, policy), policy: policy,
		auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}},
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}
}
