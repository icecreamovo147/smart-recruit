package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	gogrpc "google.golang.org/grpc"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	domainagentskill "smart-recruit-ai-agent-service/internal/domain/agentskill"
	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-platform-go/observability"
	"smart-recruit-proto/recruitment/pb"
)

type internalToolStrictStore struct {
	*fakeAIStore
	*fakeRecruitingGenerationStore
	resolution RuntimeModelInfo
}

// internalToolExactNoPackageStore intentionally embeds AIStore through its
// narrow interface so the concrete fake store's package-loading method is not
// promoted. It models a partially upgraded production adapter that resolves an
// exact capability release but cannot load its immutable Skill packages.
type internalToolExactNoPackageStore struct {
	AIStore
	billingMeterStore
	*fakeRecruitingGenerationStore
	resolution  RuntimeModelInfo
	promptCalls int
}

var (
	_ AIStore                                = (*internalToolExactNoPackageStore)(nil)
	_ capabilityRuntimeModelResolver         = (*internalToolExactNoPackageStore)(nil)
	_ recruitingReadStore                    = (*internalToolExactNoPackageStore)(nil)
	_ recruitingResumeProfileGenerationStore = (*internalToolExactNoPackageStore)(nil)
	_ recruitingruntime.PromptStore          = (*internalToolExactNoPackageStore)(nil)
)

func (s *internalToolExactNoPackageStore) ResolveCapabilityRuntimeModel(
	_ context.Context,
	_, _ string,
	_, requestedModelID int64,
) (CapabilityRuntimeModelResolution, error) {
	model := s.resolution
	if requestedModelID > 0 {
		model.ID = requestedModelID
		model.RequestedModelID = requestedModelID
	}
	return CapabilityRuntimeModelResolution{
		EffectiveModelID:       model.ID,
		ModelName:              model.Name,
		ProviderName:           model.ProviderName,
		RequestedModelID:       model.RequestedModelID,
		FallbackReason:         model.FallbackReason,
		CapabilityVersionID:    model.CapabilityVersionID,
		CapabilitySnapshotHash: model.CapabilitySnapshotHash,
		ContextWindowTokens:    model.ContextWindowTokens,
		MaxOutputTokens:        model.MaxOutputTokens,
		ConfigurationRefs:      model.ConfigurationRefs,
		SkillRuntimePolicy:     model.SkillRuntimePolicy,
	}, nil
}

func (s *internalToolExactNoPackageStore) LoadActiveRecruitingPrompt(
	_ context.Context,
	agentType, role string,
) (recruitingruntime.PromptDescriptor, error) {
	s.promptCalls++
	return recruitingruntime.PromptDescriptor{
		ID: 1, Name: "must-not-load", Version: 1, AgentType: agentType, Role: role, Content: "PRIVATE_PROMPT_BODY",
	}, nil
}

func (s *internalToolStrictStore) GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(
	ctx context.Context,
	applicationID int64,
) (RecruitingCandidateMatchSnapshot, bool, error) {
	return s.fakeRecruitingGenerationStore.GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(ctx, applicationID)
}

func (s *internalToolStrictStore) ResolveCapabilityRuntimeModel(
	_ context.Context,
	_, _ string,
	_, requestedModelID int64,
) (CapabilityRuntimeModelResolution, error) {
	model := s.resolution
	if requestedModelID > 0 {
		model.ID = requestedModelID
		model.RequestedModelID = requestedModelID
	}
	return CapabilityRuntimeModelResolution{
		EffectiveModelID:       model.ID,
		ModelName:              model.Name,
		ProviderName:           model.ProviderName,
		RequestedModelID:       model.RequestedModelID,
		FallbackReason:         model.FallbackReason,
		CapabilityVersionID:    model.CapabilityVersionID,
		CapabilitySnapshotHash: model.CapabilitySnapshotHash,
		ContextWindowTokens:    model.ContextWindowTokens,
		MaxOutputTokens:        model.MaxOutputTokens,
		ConfigurationRefs:      model.ConfigurationRefs,
		SkillRuntimePolicy:     model.SkillRuntimePolicy,
	}, nil
}

func (s *internalToolStrictStore) LoadActiveRecruitingPrompt(
	_ context.Context,
	agentType, role string,
) (recruitingruntime.PromptDescriptor, error) {
	if role != recruitingruntime.PromptRoleSystem {
		return recruitingruntime.PromptDescriptor{}, errors.New("unexpected prompt role")
	}
	switch agentType {
	case recruitingruntime.AgentTypeResumeProfileExtractor:
		return recruitingruntime.PromptDescriptor{ID: 71, Name: "resume", Version: 1, AgentType: agentType, Role: role, Content: "resume-system"}, nil
	case recruitingruntime.AgentTypeJobRequirementExtractor:
		return recruitingruntime.PromptDescriptor{ID: 72, Name: "job", Version: 1, AgentType: agentType, Role: role, Content: "job-system"}, nil
	case recruitingruntime.AgentTypeCandidateMatchEvaluator:
		return recruitingruntime.PromptDescriptor{ID: 73, Name: "match", Version: 1, AgentType: agentType, Role: role, Content: "match-system"}, nil
	default:
		return recruitingruntime.PromptDescriptor{}, errors.New("unexpected agent type")
	}
}

type internalToolBillingClient struct {
	meterBillingClient
	capabilityVersionID int64
}

func (c *internalToolBillingClient) CheckAIAccess(
	_ context.Context,
	_ *pb.CheckAIAccessRequest,
	_ ...gogrpc.CallOption,
) (*pb.CheckAIAccessResponse, error) {
	return &pb.CheckAIAccessResponse{
		Allowed:             true,
		CapabilityVersionId: c.capabilityVersionID,
		EnforcementMode:     pb.BillingEnforcementMode_BILLING_ENFORCEMENT_MODE_ENFORCE,
	}, nil
}

type internalToolStrictProvider struct {
	resume []recruitingruntime.StructuredCompletionResult
	job    []recruitingruntime.StructuredCompletionResult
	match  []recruitingruntime.StructuredCompletionResult
	calls  []string
}

func (p *internalToolStrictProvider) Complete(context.Context, string) (string, error) {
	return "", errors.New("generic completion must not be used")
}

func (p *internalToolStrictProvider) CompleteStructured(
	_ context.Context,
	systemPrompt, _ string,
) (recruitingruntime.StructuredCompletionResult, error) {
	var kind string
	var queue *[]recruitingruntime.StructuredCompletionResult
	switch {
	case strings.HasPrefix(systemPrompt, "resume-system"):
		kind, queue = "resume", &p.resume
	case strings.HasPrefix(systemPrompt, "job-system"):
		kind, queue = "job", &p.job
	case strings.HasPrefix(systemPrompt, "match-system"):
		kind, queue = "match", &p.match
	default:
		return recruitingruntime.StructuredCompletionResult{}, errors.New("unexpected system prompt")
	}
	p.calls = append(p.calls, kind)
	if len(*queue) == 0 {
		return recruitingruntime.StructuredCompletionResult{}, errors.New("missing provider result")
	}
	result := (*queue)[0]
	*queue = (*queue)[1:]
	return result, nil
}

func TestInternalRecruitingToolsApplyExactStrictContractRepairBillingAndMetrics(t *testing.T) {
	tests := []struct {
		name               string
		toolName           string
		agentType          string
		schema             json.RawMessage
		args               map[string]any
		configureStore     func(*internalToolStrictStore)
		configureProvider  func(*internalToolStrictProvider, bool)
		strictProviderKind string
		wantProviderCalls  int
	}{
		{
			name:      "resume",
			toolName:  "parse_resume_profile",
			agentType: recruitingruntime.AgentTypeResumeProfileExtractor,
			schema:    json.RawMessage(`{"type":"object","required":["full_name"]}`),
			args:      map[string]any{"application_id": 7001, "resume_id": 8001},
			configureStore: func(store *internalToolStrictStore) {
				store.applicationsByID[7001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
				store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
				store.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Ada Lovelace\nGo engineer"}
			},
			configureProvider: func(provider *internalToolStrictProvider, finalInvalid bool) {
				repaired := completeResumeProfileJSON()
				if finalInvalid {
					repaired = `{"headline":"still invalid"}`
				}
				provider.resume = strictToolResults(`{}`, repaired)
			},
			strictProviderKind: "resume",
			wantProviderCalls:  2,
		},
		{
			name:      "candidate match",
			toolName:  "evaluate_candidate_match",
			agentType: recruitingruntime.AgentTypeCandidateMatchEvaluator,
			schema:    json.RawMessage(`{"type":"object","required":["status"]}`),
			args:      map[string]any{"application_id": 7001},
			configureStore: func(store *internalToolStrictStore) {
				store.matchSources[7001] = sampleRecruitingMatchSource()
			},
			configureProvider: func(provider *internalToolStrictProvider, finalInvalid bool) {
				provider.job = strictToolResults(`{"profile_version":"job-requirement-profile-v1","requirements":[{"id":"rust","category":"core_skill","label":"Rust 开发能力","description":"岗位核心开发能力","priority":"must_have","weight":1,"knockout":false,"aliases":["Rust"]}]}`)
				repaired := `{"status":"missing","score":0,"confidence":0.8,"risk":"缺少相关项目证据","evidence":[]}`
				if finalInvalid {
					repaired = `{"score":0}`
				}
				provider.match = strictToolResults(`{}`, repaired)
			},
			strictProviderKind: "match",
			wantProviderCalls:  3,
		},
	}
	for _, test := range tests {
		for _, finalInvalid := range []bool{false, true} {
			suffix := "repair succeeds"
			if finalInvalid {
				suffix = "repair fails closed"
			}
			t.Run(test.name+"/"+suffix, func(t *testing.T) {
				store, provider, service, billing, metrics := newInternalToolStrictFixture(
					t, 101, test.agentType, domainagentskill.RiskLevelLow, test.schema,
				)
				test.configureStore(store)
				test.configureProvider(provider, finalInvalid)

				result, err := service.executeRecruitingIntelligenceTool(
					internalToolContext(), testRecruitingStaffUserID, test.toolName, test.args, 12001,
				)
				if finalInvalid {
					if err == nil || strings.Contains(result.Content, `"code":0`) {
						t.Fatalf("result=%#v error=%v, want fail closed", result, err)
					}
				} else if err != nil {
					t.Fatalf("executeRecruitingIntelligenceTool() error = %v; result=%#v", err, result)
				}
				if len(provider.calls) != test.wantProviderCalls {
					t.Fatalf("provider calls=%v, want %d", provider.calls, test.wantProviderCalls)
				}
				strictCalls := 0
				for _, kind := range provider.calls {
					if kind == test.strictProviderKind {
						strictCalls++
					}
				}
				if strictCalls != 2 {
					t.Fatalf("strict provider calls=%d in %v, want original plus one repair", strictCalls, provider.calls)
				}
				if billing.reserveCalls != 1 || billing.settleCalls != 1 || billing.cancelCalls != 0 {
					t.Fatalf("billing reserve=%d settle=%d cancel=%d", billing.reserveCalls, billing.settleCalls, billing.cancelCalls)
				}
				if auth := service.auth.(*fakeAuthClient); len(auth.authorizeRequests) != 1 {
					t.Fatalf("AI authorization calls=%d, want one", len(auth.authorizeRequests))
				}
				if usages := billing.settlementReq.GetProviderUsages(); len(usages) != test.wantProviderCalls {
					t.Fatalf("settled provider usages=%d, want %d", len(usages), test.wantProviderCalls)
				}
				metricText := metrics.Prometheus()
				wantResult, wantReason := `result="passed"`, `reason="strict_valid"`
				if finalInvalid {
					wantResult, wantReason = `result="failed"`, `reason="strict_invalid"`
				}
				if strings.Count(metricText, `smart_recruit_agent_skill_output_validation_total{`) != 1 ||
					!strings.Contains(metricText, wantResult) || !strings.Contains(metricText, wantReason) {
					t.Fatalf("strict metric was not emitted exactly once:\n%s", metricText)
				}
			})
		}
	}
}

func TestInternalRecruitingToolsRejectHighAndCriticalStrictContractsBeforeProvider(t *testing.T) {
	tests := []struct {
		name       string
		toolName   string
		agentType  string
		schema     json.RawMessage
		args       map[string]any
		configure  func(*internalToolStrictStore)
		riskLevels []domainagentskill.RiskLevel
	}{
		{
			name:      "resume",
			toolName:  "parse_resume_profile",
			agentType: recruitingruntime.AgentTypeResumeProfileExtractor,
			schema:    json.RawMessage(`{"type":"object","required":["full_name"]}`),
			args:      map[string]any{"application_id": 7001, "resume_id": 8001},
			configure: func(store *internalToolStrictStore) {
				store.applicationsByID[7001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
				store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
				store.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Ada Lovelace\nGo engineer"}
			},
			riskLevels: []domainagentskill.RiskLevel{domainagentskill.RiskLevelHigh, domainagentskill.RiskLevelCritical},
		},
		{
			name:      "candidate match",
			toolName:  "evaluate_candidate_match",
			agentType: recruitingruntime.AgentTypeJobRequirementExtractor,
			schema:    json.RawMessage(`{"type":"object","required":["profile_version"]}`),
			args:      map[string]any{"application_id": 7001},
			configure: func(store *internalToolStrictStore) {
				store.matchSources[7001] = sampleRecruitingMatchSource()
			},
			riskLevels: []domainagentskill.RiskLevel{domainagentskill.RiskLevelHigh, domainagentskill.RiskLevelCritical},
		},
	}
	for _, test := range tests {
		for _, risk := range test.riskLevels {
			t.Run(test.name+"/"+string(risk), func(t *testing.T) {
				store, provider, service, billing, metrics := newInternalToolStrictFixture(t, 101, test.agentType, risk, test.schema)
				test.configure(store)
				result, err := service.executeRecruitingIntelligenceTool(
					internalToolContext(), testRecruitingStaffUserID, test.toolName, test.args, 12001,
				)
				if err == nil || strings.Contains(result.Content, `"code":0`) {
					t.Fatalf("result=%#v error=%v, want fail closed", result, err)
				}
				if len(provider.calls) != 0 {
					t.Fatalf("provider calls=%v, want zero", provider.calls)
				}
				if billing.reserveCalls != 1 || billing.settleCalls != 0 || billing.cancelCalls != 1 {
					t.Fatalf("billing reserve=%d settle=%d cancel=%d", billing.reserveCalls, billing.settleCalls, billing.cancelCalls)
				}
				if auth := service.auth.(*fakeAuthClient); len(auth.authorizeRequests) != 1 {
					t.Fatalf("AI authorization calls=%d, want one", len(auth.authorizeRequests))
				}
				metricText := metrics.Prometheus()
				if strings.Count(metricText, `smart_recruit_agent_skill_output_validation_total{`) != 1 ||
					!strings.Contains(metricText, `result="failed"`) {
					t.Fatalf("strict metric was not emitted exactly once:\n%s", metricText)
				}
			})
		}
	}
}

func TestInternalRecruitingToolMeterRequiresExactCapabilityResolver(t *testing.T) {
	legacy := newNativeAIService(newFakeAIStore(), nil, nil, nil, nil)
	if legacy.recruitingIntelligenceMeter() != nil {
		t.Fatal("isolated legacy store received the v2 runtime meter")
	}
	released := &releasedSkillFakeStore{fakeAIStore: newFakeAIStore()}
	exact := newNativeAIService(released, nil, nil, nil, nil)
	if exact.recruitingIntelligenceMeter() != exact {
		t.Fatal("exact capability resolver did not receive the runtime meter")
	}
}

func TestInternalRecruitingToolFailsClosedWhenExactSkillPackageLoaderUnavailable(t *testing.T) {
	const (
		versionID         = int64(918273645)
		privateResumeBody = "PRIVATE_RESUME_BODY_48f35c"
	)
	base := newFakeAIStore()
	generation := newFakeRecruitingGenerationStore()
	generation.applicationsByID[7001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
	generation.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
	generation.resumeSources[8001] = RecruitingResumeSource{
		ResumeID: 8001, UserID: 3001, ParsedText: privateResumeBody,
	}
	store := &internalToolExactNoPackageStore{
		AIStore:                       base,
		billingMeterStore:             base,
		fakeRecruitingGenerationStore: generation,
		resolution: runtimeSkillModel(
			[]int64{versionID},
			CapabilitySkillRuntimePolicy{},
		),
	}
	if _, ok := any(store).(hrRuntimeAgentSkillPackageStore); ok {
		t.Fatal("test store unexpectedly implements the package loader")
	}

	provider := &internalToolStrictProvider{}
	billing := &internalToolBillingClient{
		meterBillingClient:  meterBillingClient{mode: pb.BillingEnforcementMode_BILLING_ENFORCEMENT_MODE_ENFORCE},
		capabilityVersionID: store.resolution.CapabilityVersionID,
	}
	metrics := observability.NewRegistry("ai-agent")
	service := newNativeAIService(store, provider, &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{
		Code: 0, ApplicationId: 7001, ResumeId: 8001, JobId: 9001,
	}}, nil, nil)
	service.recruitingPolicy = recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{
		StructuredResumeParse: true,
		Fallbacks:             true,
	})
	service.auth = &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: 0, Allowed: true}}
	service.billing = billing
	service.billingRequired = true
	service.skillPackageV2Enabled = true
	service.metrics = metrics

	result, err := service.executeRecruitingIntelligenceTool(
		internalToolContext(),
		testRecruitingStaffUserID,
		"parse_resume_profile",
		map[string]any{"application_id": 7001, "resume_id": 8001},
		12001,
	)
	if err == nil || strings.Contains(result.Content, `"code":0`) {
		t.Fatalf("result=%#v error=%v, want package-loader-unavailable failure", result, err)
	}
	if len(provider.calls) != 0 || store.promptCalls != 0 || len(generation.savedResumeDrafts) != 0 {
		t.Fatalf(
			"provider calls=%v prompt calls=%d saved drafts=%d, want fail closed before prompt/provider/persistence",
			provider.calls,
			store.promptCalls,
			len(generation.savedResumeDrafts),
		)
	}
	if billing.reserveCalls != 1 || billing.settleCalls != 0 || billing.cancelCalls != 1 {
		t.Fatalf(
			"billing reserve=%d settle=%d cancel=%d, want reserved then cancelled without settlement",
			billing.reserveCalls,
			billing.settleCalls,
			billing.cancelCalls,
		)
	}
	metricText := metrics.Prometheus()
	if strings.Count(metricText, `smart_recruit_agent_skill_output_validation_total{`) != 1 ||
		!strings.Contains(metricText, `result="failed"`) ||
		!strings.Contains(metricText, `reason="strict_invalid"`) {
		t.Fatalf("failed strict metric was not emitted exactly once; result=%#v error=%v:\n%s", result, err, metricText)
	}
	for _, forbidden := range []string{
		privateResumeBody,
		"PRIVATE_PROMPT_BODY",
		fmt.Sprint(versionID),
		"7001",
		"8001",
		"3001",
		"12001",
	} {
		if strings.Contains(metricText, forbidden) || strings.Contains(result.Content, forbidden) {
			t.Fatalf("failure surface leaked body or identifier %q; metric=%q result=%q", forbidden, metricText, result.Content)
		}
	}
}

func newInternalToolStrictFixture(
	t *testing.T,
	versionID int64,
	agentType string,
	risk domainagentskill.RiskLevel,
	outputSchema json.RawMessage,
) (*internalToolStrictStore, *internalToolStrictProvider, *nativeAIService, *internalToolBillingClient, *observability.Registry) {
	t.Helper()
	document := runtimeSkillVersionDocument(versionID, 1, domainagentskill.CompositionRolePrimary, risk, "strict structured core")
	document.Manifest.AgentType = agentType
	document.Manifest.OutputContract = domainagentskill.OutputContract{
		Mode: domainagentskill.OutputModeStrict, SchemaID: "internal-tool-v1", Schema: outputSchema,
	}
	aiStore := newFakeAIStore()
	aiStore.agentSkillPackages = []embeddinginfra.AgentSkillRuntimePackage{runtimeSkillPackage(t, document, nil)}
	store := &internalToolStrictStore{
		fakeAIStore:                   aiStore,
		fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore(),
		resolution:                    runtimeSkillModel([]int64{versionID}, CapabilitySkillRuntimePolicy{}),
	}
	provider := &internalToolStrictProvider{}
	billing := &internalToolBillingClient{
		meterBillingClient:  meterBillingClient{mode: pb.BillingEnforcementMode_BILLING_ENFORCEMENT_MODE_ENFORCE},
		capabilityVersionID: store.resolution.CapabilityVersionID,
	}
	metrics := observability.NewRegistry("ai-agent")
	service := newNativeAIService(store, provider, &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{
		Code: 0, ApplicationId: 7001, ResumeId: 8001, JobId: 9001,
	}}, nil, nil)
	service.recruitingPolicy = recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{
		StructuredResumeParse:  true,
		CandidateMatch:         true,
		CandidateMatchSemantic: true,
		Fallbacks:              false,
	})
	service.auth = &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: 0, Allowed: true}}
	service.billing = billing
	service.billingRequired = true
	service.skillPackageV2Enabled = true
	service.metrics = metrics
	return store, provider, service, billing, metrics
}

func strictToolResults(contents ...string) []recruitingruntime.StructuredCompletionResult {
	results := make([]recruitingruntime.StructuredCompletionResult, 0, len(contents))
	for _, content := range contents {
		results = append(results, recruitingruntime.StructuredCompletionResult{
			Content: content, ProviderKey: "provider-a", ModelName: "model-a",
			TokenUsage: &schema.TokenUsage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
		})
	}
	return results
}

func internalToolContext() context.Context {
	return platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{
		TenantID: 9, UserID: testRecruitingStaffUserID, AccountType: "staff",
	})
}
