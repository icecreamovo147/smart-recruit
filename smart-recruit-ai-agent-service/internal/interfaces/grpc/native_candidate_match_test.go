package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

type structuredCandidateMatchStore struct {
	*fakeRecruitingGenerationStore
}

func (s *structuredCandidateMatchStore) LoadActiveRecruitingPrompt(_ context.Context, agentType, role string) (recruitingruntime.PromptDescriptor, error) {
	if role != recruitingruntime.PromptRoleSystem {
		return recruitingruntime.PromptDescriptor{}, errors.New("unexpected role")
	}
	switch agentType {
	case recruitingruntime.AgentTypeJobRequirementExtractor:
		return recruitingruntime.PromptDescriptor{ID: 81, Name: "job-active", Version: 3, AgentType: agentType, Role: role, Content: "job-system"}, nil
	case recruitingruntime.AgentTypeCandidateMatchEvaluator:
		return recruitingruntime.PromptDescriptor{ID: 82, Name: "matcher-active", Version: 4, AgentType: agentType, Role: role, Content: "matcher-system"}, nil
	default:
		return recruitingruntime.PromptDescriptor{}, errors.New("unexpected agent type")
	}
}

type candidateMatchStructuredProvider struct {
	jobReply     string
	matcherReply string
	jobErr       error
	matcherErr   error
	block        bool
	systems      []string
	users        []string
}

func (p *candidateMatchStructuredProvider) Complete(context.Context, string) (string, error) {
	return "", errors.New("generic completion must not be used")
}

func (p *candidateMatchStructuredProvider) CompleteStructured(ctx context.Context, systemPrompt, userPrompt string) (recruitingruntime.StructuredCompletionResult, error) {
	p.systems = append(p.systems, systemPrompt)
	p.users = append(p.users, userPrompt)
	if p.block {
		<-ctx.Done()
		return recruitingruntime.StructuredCompletionResult{}, ctx.Err()
	}
	if systemPrompt == "job-system" {
		if p.jobErr != nil {
			return recruitingruntime.StructuredCompletionResult{}, p.jobErr
		}
		return recruitingruntime.StructuredCompletionResult{Content: p.jobReply, ModelName: "job-model"}, nil
	}
	if p.matcherErr != nil {
		return recruitingruntime.StructuredCompletionResult{}, p.matcherErr
	}
	return recruitingruntime.StructuredCompletionResult{Content: p.matcherReply, ModelName: "matcher-model"}, nil
}

type shadowFailureCandidateMatchStore struct {
	*structuredCandidateMatchStore
	promptErr error
}

func (s *shadowFailureCandidateMatchStore) LoadActiveRecruitingPrompt(ctx context.Context, agentType, role string) (recruitingruntime.PromptDescriptor, error) {
	if s.promptErr != nil {
		return recruitingruntime.PromptDescriptor{}, s.promptErr
	}
	return s.structuredCandidateMatchStore.LoadActiveRecruitingPrompt(ctx, agentType, role)
}

func TestEvaluateCandidateMatchStructuredPipelineAggregatesAndPreservesEvidenceAndAgentRun(t *testing.T) {
	store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
	source := sampleRecruitingMatchSource()
	source.Profile.Skills[0].Evidence = strings.Repeat("项目经验", 100)
	store.matchSources[7001] = source
	provider := &candidateMatchStructuredProvider{
		jobReply:     `{"profile_version":"job-requirement-profile-v1","requirements":[{"id":"go","category":"core_skill","label":"Go 开发能力","description":"岗位核心开发能力","priority":"must_have","weight":0.6,"knockout":false,"aliases":["Go"]},{"id":"communication","category":"soft_skill","label":"沟通协作能力","description":"跨团队沟通协作能力","priority":"soft_skill","weight":0.4,"knockout":false,"aliases":["沟通"]}]}`,
		matcherReply: `{"status":"missing","score":100,"confidence":0.9,"risk":"未找到沟通协作证据","evidence":[]}`,
	}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: false})
	service := nativeRecruitingIntelligenceService{
		store: store, provider: provider,
		structured: recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(store), provider, policy), policy: policy,
		auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}},
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}
	agentRunID := uint64(12001)
	resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, AgentRunId: agentRunID})
	if err != nil || resp.GetCode() != errs.OK {
		t.Fatalf("EvaluateCandidateMatch resp=%+v err=%v", resp, err)
	}
	if len(store.savedMatchDrafts) != 1 {
		t.Fatalf("saved drafts = %d, want 1", len(store.savedMatchDrafts))
	}
	draft := store.savedMatchDrafts[0]
	if draft.OverallScore != 83.8 || draft.Recommendation != recruitingruntime.RecommendationRecommend {
		t.Fatalf("score/recommendation = %.1f/%q, want deterministic 83.8/recommend", draft.OverallScore, draft.Recommendation)
	}
	if draft.AgentRunID == nil || *draft.AgentRunID != agentRunID {
		t.Fatalf("agent_run_id = %v, want %d", draft.AgentRunID, agentRunID)
	}
	if len(draft.Evidence) != 1 || draft.Evidence[0].SourceID == nil || *draft.Evidence[0].SourceID != 6401 || draft.Evidence[0].SourceTable != "resume_skills" {
		t.Fatalf("evidence = %+v, want real resume skill source ID", draft.Evidence)
	}
	if utf8.RuneCountInString(draft.Evidence[0].Snippet) > recruitingruntime.MaxCandidateEvidenceRunes {
		t.Fatalf("evidence snippet has %d runes, want <= %d", utf8.RuneCountInString(draft.Evidence[0].Snippet), recruitingruntime.MaxCandidateEvidenceRunes)
	}
	if len(provider.systems) != 2 || provider.systems[0] != "job-system" || provider.systems[1] != "matcher-system" {
		t.Fatalf("structured systems = %+v, want job then per-requirement matcher", provider.systems)
	}
	if strings.Contains(draft.ScoreBreakdownJSON, `"overall_score":100`) || !strings.Contains(draft.ScoreBreakdownJSON, `"risk_penalty":5`) {
		t.Fatalf("breakdown = %s, model score must not become an aggregate total", draft.ScoreBreakdownJSON)
	}
}

func TestEvaluateCandidateMatchFallbackDisabledInvalidMainChainDoesNotSave(t *testing.T) {
	store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
	store.matchSources[7001] = sampleRecruitingMatchSource()
	provider := &candidateMatchStructuredProvider{jobReply: `{"overall_score":100}`, matcherReply: `{}`}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: false})
	service := nativeRecruitingIntelligenceService{
		store: store, provider: provider,
		structured: recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(store), provider, policy), policy: policy,
		auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}},
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}
	resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
	if err != nil {
		t.Fatalf("EvaluateCandidateMatch transport error = %v", err)
	}
	if resp.GetCode() == errs.OK || len(store.savedMatchDrafts) != 0 {
		t.Fatalf("resp=%+v saved=%d, want failed main chain and no new evaluation", resp, len(store.savedMatchDrafts))
	}
}

func TestEvaluateCandidateMatchDeterministicPrimaryMakesNoModelCall(t *testing.T) {
	store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
	store.matchSources[7001] = sampleRecruitingMatchSource()
	provider := &candidateMatchStructuredProvider{jobReply: `{}`, matcherReply: `{}`}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: false, Fallbacks: false})
	service := nativeRecruitingIntelligenceService{
		store: store, provider: provider,
		structured: recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(store), provider, policy), policy: policy,
		auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}},
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}
	resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
	if err != nil || resp.GetCode() != errs.OK {
		t.Fatalf("EvaluateCandidateMatch resp=%+v err=%v", resp, err)
	}
	if len(provider.systems) != 0 || len(store.savedMatchDrafts) != 1 {
		t.Fatalf("structured calls=%d saved=%d, want deterministic zero-call save", len(provider.systems), len(store.savedMatchDrafts))
	}
	if !strings.Contains(store.savedMatchDrafts[0].ScoreBreakdownJSON, `"scorer_type":"legacy_deterministic"`) ||
		!strings.Contains(store.savedMatchDrafts[0].ScoreBreakdownJSON, `"name":"education"`) ||
		!strings.Contains(store.savedMatchDrafts[0].ScoreBreakdownJSON, `"name":"profile"`) {
		t.Fatalf("breakdown = %s, want restored legacy five-dimension scorer", store.savedMatchDrafts[0].ScoreBreakdownJSON)
	}
}

func TestDeterministicCandidateMatchUsesCompleteDevScoringInputButPersistsBoundedEvidence(t *testing.T) {
	source := RecruitingMatchSource{
		Application: RecruitingApplicationContext{ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, ResumeID: 8001},
		Job:         RecruitingJobContext{JobID: 9001, Requirements: "Docker Kubernetes Tailtoken"},
		Profile: RecruitingResumeProfileSnapshot{Profile: RecruitingResumeProfileRow{
			ID: 6001, FullName: "profile-name", Headline: "profile-headline", Summary: "profile-summary", HighestDegree: "profile-degree", TotalExperienceYears: 3,
		}},
		CandidateProfile: &RecruitingCandidateProfileRow{
			ID: 1, RealName: "candidate-name", Education: "candidate-education", School: "candidate-school", WorkExperience: "candidate-work", Skills: "candidate-skills",
		},
		ResumeParsedText: "parsed-text",
	}
	source.Profile.Educations = []RecruitingResumeEducationRow{{ID: 6101, School: "education-school", Degree: "education-degree", Major: "education-major", Description: "education-description"}}
	source.Profile.Projects = []RecruitingResumeProjectRow{{ID: 6301, Name: "project-name", Role: "project-role", Description: "project-description", TechnologiesJSON: `["project-technology"]`, HighlightsJSON: `["project-highlight"]`}}
	source.Profile.Skills = []RecruitingResumeSkillRow{{ID: 6401, Name: "Go", Category: "skill-category", Level: "skill-level", Evidence: "Docker"}}
	for i := 1; i <= 101; i++ {
		description := "experience-description"
		if i == 1 {
			description = strings.Repeat("x", recruitingruntime.MaxCandidateEvidenceRunes+1) + " Kubernetes"
		}
		if i == 101 {
			description = "Tailtoken"
		}
		source.Profile.Experiences = append(source.Profile.Experiences, RecruitingResumeExperienceRow{
			ID: uint64(6200 + i), Company: "experience-company", Title: "experience-title", Description: description, AchievementsJSON: `["experience-achievement"]`,
		})
	}

	resumeText, skillNames := recruitingLegacyScoringSource(source)
	wantParts := []string{
		"parsed-text", "profile-name", "profile-headline", "profile-summary", "profile-degree",
		"candidate-name", "candidate-education", "candidate-school", "candidate-work", "candidate-skills",
		"education-school", "education-degree", "education-major", "education-description",
	}
	for _, experience := range source.Profile.Experiences {
		wantParts = append(wantParts, experience.Company, experience.Title, experience.Description, experience.AchievementsJSON)
	}
	wantParts = append(wantParts,
		"project-name", "project-role", "project-description", `["project-technology"]`, `["project-highlight"]`,
		"Go", "skill-category", "skill-level", "Docker",
	)
	wantResumeText := strings.Join(wantParts, " ")
	if resumeText != wantResumeText || !reflect.DeepEqual(skillNames, []string{"Go"}) {
		t.Fatalf("legacy scoring source did not preserve complete dev field order: text_equal=%v skills=%v", resumeText == wantResumeText, skillNames)
	}

	service := nativeRecruitingIntelligenceService{policy: recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true})}
	draft := service.generateDeterministicCandidateMatchDraft(source, 0, false)
	var breakdown struct {
		InputHash  string                                   `json:"input_hash"`
		Dimensions []recruitingruntime.LegacyScoreDimension `json:"dimensions"`
	}
	if err := json.Unmarshal([]byte(draft.ScoreBreakdownJSON), &breakdown); err != nil {
		t.Fatalf("decode breakdown: %v", err)
	}
	if skills := breakdown.Dimensions[0]; skills.Score != 0 || !reflect.DeepEqual(skills.Missing, []string{"docker", "kubernetes"}) {
		t.Fatalf("skills = %+v, want only ResumeSkill.Name tokens and no Evidence-derived match", skills)
	}
	if requirements := breakdown.Dimensions[1]; requirements.Score != 100 || !reflect.DeepEqual(requirements.Matched, []string{"docker", "kubernetes", "tailtoken"}) {
		t.Fatalf("requirements = %+v, want tokens after rune 240 and record 101 from complete source", requirements)
	}
	jobText := "    Docker Kubernetes Tailtoken"
	sum := sha256.Sum256([]byte(strings.Join([]string{jobText, wantResumeText, recruitingruntime.LegacyCandidateMatchScorerVersion}, "\n")))
	if wantHash := hex.EncodeToString(sum[:]); breakdown.InputHash != wantHash {
		t.Fatalf("input hash = %q, want complete dev input hash %q", breakdown.InputHash, wantHash)
	}
	if len(draft.Evidence) != 1 || draft.Evidence[0].SourceTable != "resume_skills" || utf8.RuneCountInString(draft.Evidence[0].Snippet) > recruitingruntime.MaxCandidateEvidenceRunes {
		t.Fatalf("persisted evidence = %+v, want independently bounded/redacted source evidence", draft.Evidence)
	}
}

func TestEvaluateCandidateMatchDeterministicPrimaryPersistsWithoutProvider(t *testing.T) {
	store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
	store.matchSources[7001] = sampleRecruitingMatchSource()
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: false, Fallbacks: false})
	service := nativeRecruitingIntelligenceService{
		store: store, policy: policy,
		auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}},
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}
	resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
	if err != nil || resp.GetCode() != errs.OK || len(store.savedMatchDrafts) != 1 {
		t.Fatalf("resp=%+v err=%v saved=%d, want nil-provider deterministic save", resp, err, len(store.savedMatchDrafts))
	}
}

func TestEvaluateCandidateMatchEnhancedFailureFallsBackToLegacyDeterministic(t *testing.T) {
	store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
	store.matchSources[7001] = sampleRecruitingMatchSource()
	provider := &candidateMatchStructuredProvider{jobReply: `{"overall_score":100}`}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: true})
	service := nativeRecruitingIntelligenceService{
		store: store, provider: provider,
		structured: recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(store), provider, policy), policy: policy,
		auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}},
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}
	resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
	if err != nil || resp.GetCode() != errs.OK || len(store.savedMatchDrafts) != 1 {
		t.Fatalf("resp=%+v err=%v saved=%d, want legacy fallback save", resp, err, len(store.savedMatchDrafts))
	}
	if draft := store.savedMatchDrafts[0]; draft.ModelName != recruitingruntime.LegacyCandidateMatchScorerVersion || !strings.Contains(draft.ScoreBreakdownJSON, `"fallback_used":true`) {
		t.Fatalf("draft=%+v, want observable legacy fallback", draft)
	}
}

func TestCandidateMatchShadowRunsOppositeScorerWithoutReplacingPrimary(t *testing.T) {
	jobReply := `{"profile_version":"job-requirement-profile-v1","requirements":[{"id":"go","category":"core_skill","label":"Go 开发能力","description":"岗位核心开发能力","priority":"must_have","weight":1,"knockout":false,"aliases":["Go"]}]}`
	for _, test := range []struct {
		name        string
		semantic    bool
		wantPrimary recruitingruntime.CandidateMatchExecutionMode
		wantShadow  recruitingruntime.CandidateMatchExecutionMode
	}{
		{name: "enhanced primary deterministic shadow", semantic: true, wantPrimary: recruitingruntime.CandidateMatchExecutionEnhanced, wantShadow: recruitingruntime.CandidateMatchExecutionDeterministic},
		{name: "deterministic primary enhanced shadow", semantic: false, wantPrimary: recruitingruntime.CandidateMatchExecutionDeterministic, wantShadow: recruitingruntime.CandidateMatchExecutionEnhanced},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
			provider := &candidateMatchStructuredProvider{jobReply: jobReply}
			policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: test.semantic, CandidateMatchShadow: true, Fallbacks: false})
			service := nativeRecruitingIntelligenceService{store: store, provider: provider, structured: recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(store), provider, policy), policy: policy}
			primary, comparison, err := service.generateCandidateMatchDraftWithShadow(context.Background(), sampleRecruitingMatchSource(), 12001)
			if err != nil || comparison == nil || !comparison.ShadowSucceeded {
				t.Fatalf("primary=%+v comparison=%+v err=%v", primary, comparison, err)
			}
			if comparison.PrimaryMode != test.wantPrimary || comparison.ShadowMode != test.wantShadow || primary.AgentRunID == nil || *primary.AgentRunID != 12001 {
				t.Fatalf("primary=%+v comparison=%+v, want modes %s/%s and preserved run", primary, comparison, test.wantPrimary, test.wantShadow)
			}
			if len(store.savedMatchDrafts) != 0 || len(provider.systems) != 1 || provider.systems[0] != "job-system" {
				t.Fatalf("saved=%d systems=%+v, shadow must be read-only and enhanced scorer must use structured job prompt", len(store.savedMatchDrafts), provider.systems)
			}
			if test.semantic && primary.ModelName != "job-model" {
				t.Fatalf("primary model=%q, want enhanced primary preserved", primary.ModelName)
			}
			if !test.semantic && primary.ModelName != recruitingruntime.LegacyCandidateMatchScorerVersion {
				t.Fatalf("primary model=%q, want deterministic primary preserved", primary.ModelName)
			}
		})
	}
}

func TestEvaluateCandidateMatchEnhancedShadowFailuresAreIsolatedFromDeterministicPrimary(t *testing.T) {
	tests := []struct {
		name      string
		promptErr error
		provider  *candidateMatchStructuredProvider
	}{
		{name: "prompt", promptErr: errors.New("prompt unavailable"), provider: &candidateMatchStructuredProvider{}},
		{name: "provider", provider: &candidateMatchStructuredProvider{jobErr: errors.New("provider unavailable")}},
		{name: "schema", provider: &candidateMatchStructuredProvider{jobReply: `{"overall_score":100}`}},
		{name: "timeout", provider: &candidateMatchStructuredProvider{block: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
			base.matchSources[7001] = sampleRecruitingMatchSource()
			store := &shadowFailureCandidateMatchStore{structuredCandidateMatchStore: base, promptErr: test.promptErr}
			policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{
				CandidateMatch: true, CandidateMatchSemantic: false, CandidateMatchShadow: true,
				Fallbacks: false, CandidateMatchTimeout: 80 * time.Millisecond,
			})
			service := nativeRecruitingIntelligenceService{
				store: store, provider: test.provider,
				structured: recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(store), test.provider, policy), policy: policy,
				auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}},
				applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
			}
			resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, AgentRunId: 12001})
			if err != nil || resp.GetCode() != errs.OK {
				t.Fatalf("EvaluateCandidateMatch resp=%+v err=%v, want isolated shadow failure", resp, err)
			}
			if len(base.savedMatchDrafts) != 1 {
				t.Fatalf("saved drafts = %d, want exactly one primary evaluation", len(base.savedMatchDrafts))
			}
			draft := base.savedMatchDrafts[0]
			if draft.ModelName != recruitingruntime.LegacyCandidateMatchScorerVersion || draft.AgentRunID == nil || *draft.AgentRunID != 12001 {
				t.Fatalf("saved primary = %+v, want unchanged deterministic primary", draft)
			}
		})
	}
}

func TestEvaluateCandidateMatchMissingStructuredDependencyFollowsFallbackPolicyWithoutGenericAggregate(t *testing.T) {
	for _, test := range []struct {
		name      string
		fallbacks bool
		wantOK    bool
	}{
		{name: "fallback enabled uses deterministic", fallbacks: true, wantOK: true},
		{name: "fallback disabled fails closed", fallbacks: false, wantOK: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := newFakeRecruitingGenerationStore()
			store.matchSources[7001] = sampleRecruitingMatchSource()
			generic := &fakeChatProvider{reply: `{"overall_score":100,"recommendation":"strong_match"}`}
			policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: test.fallbacks})
			service := nativeRecruitingIntelligenceService{
				store: store, provider: generic, policy: policy,
				auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}},
				applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
			}
			resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
			if err != nil || (resp.GetCode() == errs.OK) != test.wantOK {
				t.Fatalf("resp=%+v err=%v wantOK=%v", resp, err, test.wantOK)
			}
			if generic.calls != 0 {
				t.Fatalf("generic aggregate calls = %d, want zero", generic.calls)
			}
			if test.wantOK && (len(store.savedMatchDrafts) != 1 || store.savedMatchDrafts[0].ModelName != recruitingruntime.LegacyCandidateMatchScorerVersion) {
				t.Fatalf("saved drafts = %+v, want one deterministic fallback", store.savedMatchDrafts)
			}
			if !test.wantOK && len(store.savedMatchDrafts) != 0 {
				t.Fatalf("saved drafts = %+v, want no evaluation", store.savedMatchDrafts)
			}
		})
	}
}

type deadlineCandidateMatchStore struct {
	*structuredCandidateMatchStore
	readWaitForContext bool
	saveCalled         bool
	saveContextErr     error
	saveRemaining      time.Duration
}

func (s *deadlineCandidateMatchStore) GetRecruitingMatchSource(ctx context.Context, applicationID int64) (RecruitingMatchSource, bool, error) {
	if s.readWaitForContext {
		<-ctx.Done()
		return RecruitingMatchSource{}, false, ctx.Err()
	}
	return s.structuredCandidateMatchStore.GetRecruitingMatchSource(ctx, applicationID)
}

func (s *deadlineCandidateMatchStore) SaveRecruitingCandidateMatchDraft(ctx context.Context, draft RecruitingCandidateMatchDraft) (RecruitingCandidateMatchSnapshot, error) {
	s.saveCalled = true
	s.saveContextErr = ctx.Err()
	if deadline, ok := ctx.Deadline(); ok {
		s.saveRemaining = time.Until(deadline)
	}
	return s.structuredCandidateMatchStore.SaveRecruitingCandidateMatchDraft(ctx, draft)
}

type reserveCandidateMatchProvider struct {
	jobReply string
	block    bool
}

func (p *reserveCandidateMatchProvider) Complete(context.Context, string) (string, error) {
	return "", errors.New("generic completion must not be used")
}

func (p *reserveCandidateMatchProvider) CompleteStructured(ctx context.Context, _, _ string) (recruitingruntime.StructuredCompletionResult, error) {
	if p.block {
		<-ctx.Done()
		return recruitingruntime.StructuredCompletionResult{}, ctx.Err()
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return recruitingruntime.StructuredCompletionResult{}, errors.New("generation deadline missing")
	}
	wait := time.Until(deadline) - 20*time.Millisecond
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return recruitingruntime.StructuredCompletionResult{}, ctx.Err()
		}
	}
	return recruitingruntime.StructuredCompletionResult{Content: p.jobReply, ModelName: "job-model"}, nil
}

func TestEvaluateCandidateMatchDeadlineCoversSourceReadAndParentCancellationPreventsSave(t *testing.T) {
	base := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
	base.matchSources[7001] = sampleRecruitingMatchSource()
	for _, test := range []struct {
		name            string
		cancel          bool
		readBlock       bool
		generationBlock bool
	}{
		{name: "source read timeout", readBlock: true},
		{name: "generation timeout", generationBlock: true},
		{name: "parent cancellation", cancel: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &deadlineCandidateMatchStore{structuredCandidateMatchStore: base, readWaitForContext: test.readBlock}
			policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: test.generationBlock, Fallbacks: false, CandidateMatchTimeout: 40 * time.Millisecond})
			service := nativeRecruitingIntelligenceService{
				store: store, policy: policy,
				auth: &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}}, applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001}},
			}
			if test.generationBlock {
				provider := &reserveCandidateMatchProvider{block: true}
				service.provider = provider
				service.structured = recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(store), provider, policy)
			}
			ctx := recruitingAuthContext()
			if test.cancel {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			resp, err := service.EvaluateCandidateMatch(ctx, &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
			if err != nil || resp.GetCode() == errs.OK || store.saveCalled {
				t.Fatalf("resp=%+v err=%v save=%v, want deadline/cancel non-success without save", resp, err, store.saveCalled)
			}
		})
	}
}

func TestEvaluateCandidateMatchReservesPersistenceBudget(t *testing.T) {
	base := &structuredCandidateMatchStore{fakeRecruitingGenerationStore: newFakeRecruitingGenerationStore()}
	base.matchSources[7001] = sampleRecruitingMatchSource()
	store := &deadlineCandidateMatchStore{structuredCandidateMatchStore: base}
	provider := &reserveCandidateMatchProvider{jobReply: `{"profile_version":"job-requirement-profile-v1","requirements":[{"id":"go","category":"core_skill","label":"Go 开发能力","description":"岗位核心开发能力","priority":"must_have","weight":1,"knockout":false,"aliases":["Go"]}]}`}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{CandidateMatch: true, CandidateMatchSemantic: true, Fallbacks: false, CandidateMatchTimeout: 250 * time.Millisecond})
	service := nativeRecruitingIntelligenceService{
		store: store, provider: provider, structured: recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(store), provider, policy), policy: policy,
		auth: &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}}, applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001}},
	}
	resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
	if err != nil || resp.GetCode() != errs.OK || !store.saveCalled || store.saveContextErr != nil || store.saveRemaining <= 20*time.Millisecond {
		t.Fatalf("resp=%+v err=%v save=%v saveErr=%v remaining=%v, want bounded persistence reserve", resp, err, store.saveCalled, store.saveContextErr, store.saveRemaining)
	}
}
