package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestParseResumeProfileStrictFailureDoesNotPersistOrSwitchCurrent(t *testing.T) {
	store := newFakeRecruitingGenerationStore()
	store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
	store.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Ada\nGo engineer"}
	store.currentProfileByResumeID[8001] = RecruitingResumeProfileRow{ID: 6001, ResumeID: 8001, Version: 4, IsCurrent: 1}
	provider := &capturingResumeProvider{content: `{"full_name":"Ada","total_experience_years":1,"educations":[],"experiences":[],"projects":[],"skills":["Go"]}`}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: false})
	service := resumeProfileTestService(store, provider, policy)

	resp, err := service.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
	if err != nil {
		t.Fatalf("ParseResumeProfile error = %v", err)
	}
	if resp.GetCode() == errs.OK || resp.GetProfile() != nil {
		t.Fatalf("response = %+v, want schema failure", resp)
	}
	if len(store.savedResumeDrafts) != 0 {
		t.Fatalf("saved drafts = %d, want none", len(store.savedResumeDrafts))
	}
	current := store.currentProfileByResumeID[8001]
	if current.ID != 6001 || current.Version != 4 || current.IsCurrent != 1 {
		t.Fatalf("current profile changed = %+v", current)
	}
}

func TestParseResumeProfileUsesSystemPromptAndPolicyFallback(t *testing.T) {
	store := newFakeRecruitingGenerationStore()
	store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
	store.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, FileName: "candidate.pdf", ParsedText: "Ada Lovelace\nGo engineer"}
	provider := &capturingResumeProvider{content: `{"full_name":"Ada","total_experience_years":1,"educations":[],"experiences":[],"projects":[],"skills":["Go"]}`}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: true})
	service := resumeProfileTestService(store, provider, policy)

	resp, err := service.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
	if err != nil || resp.GetCode() != errs.OK {
		t.Fatalf("ParseResumeProfile response=%+v error=%v", resp, err)
	}
	if provider.system != "database resume system" || !strings.Contains(provider.user, "Resume parsed text:\nAda Lovelace") {
		t.Fatalf("structured messages system=%q user=%q", provider.system, provider.user)
	}
	if len(store.savedResumeDrafts) != 1 || store.savedResumeDrafts[0].ParserVersion != recruitingruntime.ResumeHeuristicParserVersion {
		t.Fatalf("saved drafts = %+v, want heuristic provenance", store.savedResumeDrafts)
	}
	if store.savedResumeDrafts[0].InputHash == "" || store.savedResumeDrafts[0].FullName != "Ada Lovelace" {
		t.Fatalf("saved draft = %+v", store.savedResumeDrafts[0])
	}
}

func TestParseResumeProfileSaveFailureReturnsNonSuccessWithoutCommittedDraft(t *testing.T) {
	base := newFakeRecruitingGenerationStore()
	base.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
	base.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Ada Lovelace\nGo engineer"}
	store := &failingResumeSaveStore{fakeRecruitingGenerationStore: base, saveErr: errors.New("transaction rolled back")}
	provider := &capturingResumeProvider{content: completeResumeProfileJSON()}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: false})
	service := resumeProfileTestService(store, provider, policy)

	resp, err := service.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
	if err != nil {
		t.Fatalf("ParseResumeProfile error = %v", err)
	}
	if resp.GetCode() == errs.OK || resp.GetProfile() != nil || len(base.savedResumeDrafts) != 0 {
		t.Fatalf("response=%+v saved=%d, want rollback/non-success", resp, len(base.savedResumeDrafts))
	}
}

func TestParseResumeProfileModelWindowExpiryLeavesPersistenceBudget(t *testing.T) {
	base := newFakeRecruitingGenerationStore()
	base.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
	base.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Ada Lovelace\nGo engineer"}
	store := &observingResumeSaveStore{fakeRecruitingGenerationStore: base}
	provider := &deadlineResumeProvider{}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: true, ResumeParseTimeout: 100 * time.Millisecond})
	service := resumeProfileTestService(store, provider, policy)

	started := time.Now()
	resp, err := service.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
	if err != nil || resp.GetCode() != errs.OK {
		t.Fatalf("ParseResumeProfile response=%+v error=%v", resp, err)
	}
	if !provider.deadlineObserved || store.saveContextErr != nil || store.saveRemaining <= 0 {
		t.Fatalf("provider deadline=%v save err=%v remaining=%v", provider.deadlineObserved, store.saveContextErr, store.saveRemaining)
	}
	if elapsed := time.Since(started); elapsed < 70*time.Millisecond || elapsed >= 100*time.Millisecond {
		t.Fatalf("elapsed = %v, want parsing window expiry before total deadline", elapsed)
	}
	if len(base.savedResumeDrafts) != 1 || base.savedResumeDrafts[0].ParserVersion != recruitingruntime.ResumeHeuristicParserVersion {
		t.Fatalf("saved drafts = %+v, want persisted timeout fallback", base.savedResumeDrafts)
	}
}

func TestParseResumeProfileParentCancellationDoesNotSaveFallback(t *testing.T) {
	base := newFakeRecruitingGenerationStore()
	base.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, ResumeID: 8001}
	base.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "Ada Lovelace\nGo engineer"}
	store := &observingResumeSaveStore{fakeRecruitingGenerationStore: base}
	provider := &deadlineResumeProvider{}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: true, ResumeParseTimeout: time.Second})
	service := resumeProfileTestService(store, provider, policy)
	ctx, cancel := context.WithCancel(recruitingAuthContext())
	cancel()

	resp, err := service.ParseResumeProfile(ctx, &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
	if err != nil {
		t.Fatalf("ParseResumeProfile error = %v", err)
	}
	if resp.GetCode() == errs.OK || store.saveCalled || len(base.savedResumeDrafts) != 0 {
		t.Fatalf("response=%+v saveCalled=%v saved=%d", resp, store.saveCalled, len(base.savedResumeDrafts))
	}
}

func resumeProfileTestService(store recruitingReadStore, provider recruitingruntime.StructuredCompletionProvider, policy recruitingruntime.RuntimePolicy) nativeRecruitingIntelligenceService {
	runtime := recruitingruntime.NewRuntime(recruitingruntime.NewPromptLoader(resumeProfilePromptStore{}), provider, policy)
	return nativeRecruitingIntelligenceService{
		store:        store,
		structured:   runtime,
		policy:       policy,
		auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}},
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001}},
	}
}

type resumeProfilePromptStore struct{}

func (resumeProfilePromptStore) LoadActiveRecruitingPrompt(context.Context, string, string) (recruitingruntime.PromptDescriptor, error) {
	return recruitingruntime.PromptDescriptor{ID: 71, Name: "resume-active", Version: 2, AgentType: recruitingruntime.AgentTypeResumeProfileExtractor, Role: recruitingruntime.PromptRoleSystem, Content: "database resume system"}, nil
}

type capturingResumeProvider struct {
	content string
	system  string
	user    string
}

func (p *capturingResumeProvider) CompleteStructured(_ context.Context, systemPrompt, userPrompt string) (recruitingruntime.StructuredCompletionResult, error) {
	p.system = systemPrompt
	p.user = userPrompt
	return recruitingruntime.StructuredCompletionResult{Content: p.content, ModelName: "model-a"}, nil
}

type failingResumeSaveStore struct {
	*fakeRecruitingGenerationStore
	saveErr error
}

type observingResumeSaveStore struct {
	*fakeRecruitingGenerationStore
	saveCalled     bool
	saveContextErr error
	saveRemaining  time.Duration
}

func (s *observingResumeSaveStore) SaveRecruitingResumeProfileDraft(ctx context.Context, draft RecruitingResumeProfileDraft) (RecruitingResumeProfileSnapshot, error) {
	s.saveCalled = true
	s.saveContextErr = ctx.Err()
	if deadline, ok := ctx.Deadline(); ok {
		s.saveRemaining = time.Until(deadline)
	}
	return s.fakeRecruitingGenerationStore.SaveRecruitingResumeProfileDraft(ctx, draft)
}

type deadlineResumeProvider struct {
	deadlineObserved bool
}

func (p *deadlineResumeProvider) CompleteStructured(ctx context.Context, _, _ string) (recruitingruntime.StructuredCompletionResult, error) {
	<-ctx.Done()
	p.deadlineObserved = errors.Is(ctx.Err(), context.DeadlineExceeded)
	return recruitingruntime.StructuredCompletionResult{}, ctx.Err()
}

func (s *failingResumeSaveStore) SaveRecruitingResumeProfileDraft(context.Context, RecruitingResumeProfileDraft) (RecruitingResumeProfileSnapshot, error) {
	return RecruitingResumeProfileSnapshot{}, s.saveErr
}

// These methods upgrade the pre-existing generation fakes to the structured
// runtime without weakening production validation. The legacy fixture omitted
// two required empty arrays, so the test adapter supplies those schema defaults.
func (p *fakeChatProvider) CompleteStructured(ctx context.Context, systemPrompt, userPrompt string) (recruitingruntime.StructuredCompletionResult, error) {
	reply, err := p.Complete(ctx, userPrompt)
	if err != nil {
		return recruitingruntime.StructuredCompletionResult{}, err
	}
	var object map[string]any
	if json.Unmarshal([]byte(reply), &object) == nil {
		for field, fallback := range map[string]any{
			"full_name": "", "email": "", "phone": "", "location": "", "headline": "", "summary": "",
			"total_experience_years": float64(0), "highest_degree": "",
		} {
			if _, exists := object[field]; !exists {
				object[field] = fallback
			}
		}
		for _, field := range []string{"educations", "experiences", "projects", "skills"} {
			if _, exists := object[field]; !exists {
				object[field] = []any{}
			}
		}
		if educations, ok := object["educations"].([]any); ok {
			for _, raw := range educations {
				if item, itemOK := raw.(map[string]any); itemOK {
					addResumeFixtureDefaults(item, map[string]any{"school": "", "degree": "", "major": "", "start_date": "", "end_date": "", "description": ""})
				}
			}
		}
		if experiences, ok := object["experiences"].([]any); ok {
			for _, raw := range experiences {
				if item, itemOK := raw.(map[string]any); itemOK {
					addResumeFixtureDefaults(item, map[string]any{"company": "", "title": "", "location": "", "start_date": "", "end_date": "", "is_current": false, "description": "", "achievements": []any{}})
				}
			}
		}
		if projects, ok := object["projects"].([]any); ok {
			for _, raw := range projects {
				if item, itemOK := raw.(map[string]any); itemOK {
					addResumeFixtureDefaults(item, map[string]any{"name": "", "role": "", "start_date": "", "end_date": "", "description": "", "technologies": []any{}, "highlights": []any{}})
				}
			}
		}
		if skills, ok := object["skills"].([]any); ok {
			for _, raw := range skills {
				if item, itemOK := raw.(map[string]any); itemOK {
					addResumeFixtureDefaults(item, map[string]any{"name": "", "category": "", "level": "", "years": float64(0), "evidence": ""})
				}
			}
		}
		if normalized, marshalErr := json.Marshal(object); marshalErr == nil {
			reply = string(normalized)
		}
	}
	_ = systemPrompt
	return recruitingruntime.StructuredCompletionResult{Content: reply, ModelName: "fake-model"}, nil
}

func addResumeFixtureDefaults(item map[string]any, defaults map[string]any) {
	for field, fallback := range defaults {
		if _, exists := item[field]; !exists {
			item[field] = fallback
		}
	}
}

func (f *fakeRecruitingGenerationStore) LoadActiveRecruitingPrompt(context.Context, string, string) (recruitingruntime.PromptDescriptor, error) {
	return recruitingruntime.PromptDescriptor{ID: 1, Name: "legacy-test-resume", Version: 1, AgentType: recruitingruntime.AgentTypeResumeProfileExtractor, Role: recruitingruntime.PromptRoleSystem, Content: "database resume system"}, nil
}

func completeResumeProfileJSON() string {
	return `{"full_name":"Ada Lovelace","email":"","phone":"","location":"","headline":"Engineer","summary":"Go engineer","total_experience_years":5,"highest_degree":"","educations":[],"experiences":[],"projects":[],"skills":[{"name":"Go","category":"language","level":"senior","years":5,"evidence":"services"}]}`
}
