package grpc

import (
	"context"
	"strings"
	"testing"

	gogrpc "google.golang.org/grpc"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestParseResumeProfileGeneratesAndPersistsFreshSnapshot(t *testing.T) {
	store := newFakeRecruitingGenerationStore()
	store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, ResumeID: 8001}
	store.resumeSources[8001] = RecruitingResumeSource{
		ResumeID:   8001,
		UserID:     3001,
		FileName:   "candidate.pdf",
		ParsedText: "Ada builds Go services and distributed systems.",
	}
	provider := &fakeChatProvider{reply: `{
		"full_name":"Ada Lovelace",
		"email":"ada@example.com",
		"headline":"Backend engineer",
		"summary":"Go backend and distributed systems",
		"total_experience_years":6,
		"highest_degree":"BS",
		"experiences":[{"company":"Engines","title":"Engineer","is_current":true,"achievements":["Go services"]}],
		"skills":[{"name":"Go","category":"language","level":"senior","years":5,"evidence":"services"}]
	}`}
	auth := &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}}
	service := nativeRecruitingIntelligenceService{
		store:        store,
		provider:     provider,
		auth:         auth,
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}

	resp, err := service.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
	if err != nil {
		t.Fatalf("ParseResumeProfile error = %v", err)
	}
	if resp.GetCode() != errs.OK {
		t.Fatalf("code = %d msg=%q, want OK", resp.GetCode(), resp.GetMsg())
	}
	if got := resp.GetProfile().GetProfile().GetFullName(); got != "Ada Lovelace" {
		t.Fatalf("full name = %q, want Ada Lovelace", got)
	}
	if len(store.savedResumeDrafts) != 1 {
		t.Fatalf("saved resume drafts = %d, want 1", len(store.savedResumeDrafts))
	}
	if store.savedResumeDrafts[0].InputHash == "" || store.savedResumeDrafts[0].ParserVersion == "" {
		t.Fatalf("draft missing parser metadata: %+v", store.savedResumeDrafts[0])
	}
	if provider.calls != 1 || len(provider.prompts) != 1 {
		t.Fatalf("provider calls = %d prompts=%d, want 1", provider.calls, len(provider.prompts))
	}
	if len(auth.authorizeRequests) != 1 || auth.authorizeRequests[0].GetPermissionKey() != "ai.hr.use" || auth.authorizeRequests[0].GetResourceType() != "resume" || auth.authorizeRequests[0].GetResourceId() != 8001 {
		t.Fatalf("auth requests = %+v, want ai.hr.use resume 8001", auth.authorizeRequests)
	}
}

func TestParseResumeProfileRejectsAIPermissionDenied(t *testing.T) {
	store := newFakeRecruitingGenerationStore()
	store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, ResumeID: 8001}
	store.resumeSources[8001] = RecruitingResumeSource{ResumeID: 8001, UserID: 3001, ParsedText: "private resume"}
	provider := &fakeChatProvider{reply: `{"summary":"must not run"}`}
	service := nativeRecruitingIntelligenceService{
		store:        store,
		provider:     provider,
		auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: false, Reason: "denied"}},
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}

	resp, err := service.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
	if err != nil {
		t.Fatalf("ParseResumeProfile error = %v", err)
	}
	if resp.GetCode() != errs.ErrForbidden {
		t.Fatalf("code = %d msg=%q, want forbidden", resp.GetCode(), resp.GetMsg())
	}
	if provider.calls != 0 || len(store.savedResumeDrafts) != 0 {
		t.Fatalf("provider calls=%d saved drafts=%d, want no generation", provider.calls, len(store.savedResumeDrafts))
	}
}

func TestEvaluateCandidateMatchGeneratesAndPreservesAgentRunID(t *testing.T) {
	store := newFakeRecruitingGenerationStore()
	source := sampleRecruitingMatchSource()
	store.matchSources[7001] = source
	agentRunID := uint64(12001)
	provider := &fakeChatProvider{reply: `{
		"overall_score":88.5,
		"recommendation":"strong_match",
		"summary":"Strong backend match",
		"strengths":["Go","distributed systems"],
		"risks":["limited frontend"],
		"missing_requirements":["frontend"],
		"dimensions":[{"name":"backend","score":92}],
		"evidence":[{"evidence_type":"skill","dimension":"backend","source_table":"resume_skills","source_id":6401,"snippet":"Go","weight":0.8,"score_impact":5.5,"metadata_json":"{\"source\":\"resume\"}"}]
	}`}
	auth := &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}}
	service := nativeRecruitingIntelligenceService{
		store:        store,
		provider:     provider,
		auth:         auth,
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}

	resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, AgentRunId: agentRunID})
	if err != nil {
		t.Fatalf("EvaluateCandidateMatch error = %v", err)
	}
	if resp.GetCode() != errs.OK {
		t.Fatalf("code = %d msg=%q, want OK", resp.GetCode(), resp.GetMsg())
	}
	evaluation := resp.GetEvaluation().GetEvaluation()
	if evaluation.GetAgentRunId() != agentRunID {
		t.Fatalf("agent run id = %d, want %d", evaluation.GetAgentRunId(), agentRunID)
	}
	if evaluation.GetOverallScore() <= 0 || evaluation.GetRecommendation() == "" ||
		!strings.Contains(evaluation.GetScoreBreakdownJson(), `"scorer_type":"legacy_deterministic"`) {
		t.Fatalf("evaluation = %+v, want deterministic score/recommendation", evaluation)
	}
	if len(resp.GetEvaluation().GetEvidence()) == 0 {
		t.Fatal("deterministic evaluation evidence is empty")
	}
	if len(store.savedMatchDrafts) != 1 || store.savedMatchDrafts[0].AgentRunID == nil || *store.savedMatchDrafts[0].AgentRunID != agentRunID {
		t.Fatalf("saved match drafts = %+v, want preserved agent run id", store.savedMatchDrafts)
	}
	if provider.calls != 0 {
		t.Fatalf("generic aggregate provider calls = %d, want zero", provider.calls)
	}
	if len(auth.authorizeRequests) != 1 || auth.authorizeRequests[0].GetResourceType() != "application" || auth.authorizeRequests[0].GetResourceId() != 7001 {
		t.Fatalf("auth requests = %+v, want application permission check", auth.authorizeRequests)
	}
}

func TestEvaluateCandidateMatchRejectsAIPermissionDenied(t *testing.T) {
	store := newFakeRecruitingGenerationStore()
	store.matchSources[7001] = sampleRecruitingMatchSource()
	provider := &fakeChatProvider{reply: `{"overall_score":90}`}
	service := nativeRecruitingIntelligenceService{
		store:        store,
		provider:     provider,
		auth:         &fakeAuthClient{authorizeResp: &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: false, Reason: "denied"}},
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}

	resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
	if err != nil {
		t.Fatalf("EvaluateCandidateMatch error = %v", err)
	}
	if resp.GetCode() != errs.ErrForbidden {
		t.Fatalf("code = %d msg=%q, want forbidden", resp.GetCode(), resp.GetMsg())
	}
	if provider.calls != 0 || len(store.savedMatchDrafts) != 0 {
		t.Fatalf("provider calls=%d saved drafts=%d, want no generation", provider.calls, len(store.savedMatchDrafts))
	}
}

type fakeRecruitingGenerationStore struct {
	*fakeRecruitingReadStore
	resumeSources     map[int64]RecruitingResumeSource
	matchSources      map[int64]RecruitingMatchSource
	savedResumeDrafts []RecruitingResumeProfileDraft
	savedMatchDrafts  []RecruitingCandidateMatchDraft
}

func newFakeRecruitingGenerationStore() *fakeRecruitingGenerationStore {
	return &fakeRecruitingGenerationStore{
		fakeRecruitingReadStore: newFakeRecruitingReadStore(),
		resumeSources:           make(map[int64]RecruitingResumeSource),
		matchSources:            make(map[int64]RecruitingMatchSource),
	}
}

func (f *fakeRecruitingGenerationStore) GetRecruitingResumeSource(_ context.Context, resumeID int64) (RecruitingResumeSource, bool, error) {
	if f.err != nil {
		return RecruitingResumeSource{}, false, f.err
	}
	row, ok := f.resumeSources[resumeID]
	return row, ok, nil
}

func (f *fakeRecruitingGenerationStore) SaveRecruitingResumeProfileDraft(_ context.Context, draft RecruitingResumeProfileDraft) (RecruitingResumeProfileSnapshot, error) {
	if f.err != nil {
		return RecruitingResumeProfileSnapshot{}, f.err
	}
	f.savedResumeDrafts = append(f.savedResumeDrafts, draft)
	profileID := uint64(6100 + len(f.savedResumeDrafts))
	started := sampleRecruitingResumeSnapshot(profileID, draft.ResumeID, draft.UserID).ParseRun.StartedAt
	snapshot := RecruitingResumeProfileSnapshot{
		ParseRun: RecruitingResumeParseRunRow{ID: 5100 + uint64(len(f.savedResumeDrafts)), ResumeID: draft.ResumeID, UserID: draft.UserID, Status: "succeeded", ParserVersion: draft.ParserVersion, InputHash: draft.InputHash, StartedAt: started, CompletedAt: &started, CreatedAt: started, UpdatedAt: started},
		Profile:  RecruitingResumeProfileRow{ID: profileID, ResumeID: draft.ResumeID, UserID: draft.UserID, ParseRunID: 5100 + uint64(len(f.savedResumeDrafts)), Version: int32(len(f.savedResumeDrafts)), IsCurrent: 1, FullName: draft.FullName, Email: draft.Email, Headline: draft.Headline, Summary: draft.Summary, RawJSON: draft.RawJSON, CreatedAt: started, UpdatedAt: started},
		Skills:   draft.Skills,
	}
	return snapshot, nil
}

func (f *fakeRecruitingGenerationStore) GetRecruitingMatchSource(_ context.Context, applicationID int64) (RecruitingMatchSource, bool, error) {
	if f.err != nil {
		return RecruitingMatchSource{}, false, f.err
	}
	row, ok := f.matchSources[applicationID]
	return row, ok, nil
}

func (f *fakeRecruitingGenerationStore) SaveRecruitingCandidateMatchDraft(_ context.Context, draft RecruitingCandidateMatchDraft) (RecruitingCandidateMatchSnapshot, error) {
	if f.err != nil {
		return RecruitingCandidateMatchSnapshot{}, f.err
	}
	f.savedMatchDrafts = append(f.savedMatchDrafts, draft)
	snapshot := sampleCandidateMatchSnapshot(9200+uint64(len(f.savedMatchDrafts)), draft.ApplicationID, int32(len(f.savedMatchDrafts)), draft.OverallScore)
	snapshot.Evaluation.JobID = draft.JobID
	snapshot.Evaluation.CandidateUserID = draft.CandidateUserID
	snapshot.Evaluation.ResumeProfileID = draft.ResumeProfileID
	snapshot.Evaluation.AgentRunID = draft.AgentRunID
	snapshot.Evaluation.Recommendation = draft.Recommendation
	snapshot.Evaluation.Summary = draft.Summary
	snapshot.Evaluation.StrengthsJSON = draft.StrengthsJSON
	snapshot.Evaluation.RisksJSON = draft.RisksJSON
	snapshot.Evaluation.ScoreBreakdownJSON = draft.ScoreBreakdownJSON
	snapshot.Evaluation.ModelName = draft.ModelName
	snapshot.Evidence = draft.Evidence
	return snapshot, nil
}

func sampleRecruitingMatchSource() RecruitingMatchSource {
	return RecruitingMatchSource{
		Application: RecruitingApplicationContext{ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, CandidateName: "Ada", ResumeID: 8001},
		Job:         RecruitingJobContext{JobID: 9001, Title: "Backend Engineer", Department: "Engineering", Location: "Shanghai", Description: "Build services", Requirements: "Go, distributed systems"},
		Profile:     sampleRecruitingResumeSnapshot(6001, 8001, 3001),
	}
}

type fakeAuthClient struct {
	authorizeResp     *pb.AuthorizeInternalResponse
	authorizeErr      error
	authorizeRequests []*pb.AuthorizeInternalRequest
}

func (f *fakeAuthClient) Register(context.Context, *pb.RegisterRequest, ...gogrpc.CallOption) (*pb.RegisterResponse, error) {
	panic("unexpected Register call")
}

func (f *fakeAuthClient) Login(context.Context, *pb.LoginRequest, ...gogrpc.CallOption) (*pb.LoginResponse, error) {
	panic("unexpected Login call")
}

func (f *fakeAuthClient) RefreshToken(context.Context, *pb.RefreshTokenRequest, ...gogrpc.CallOption) (*pb.RefreshTokenResponse, error) {
	panic("unexpected RefreshToken call")
}

func (f *fakeAuthClient) SwitchTenant(context.Context, *pb.SwitchTenantRequest, ...gogrpc.CallOption) (*pb.LoginResponse, error) {
	panic("unexpected SwitchTenant call")
}

func (f *fakeAuthClient) RevokeRefreshToken(context.Context, *pb.RevokeRefreshTokenRequest, ...gogrpc.CallOption) (*pb.CommonResponse, error) {
	panic("unexpected RevokeRefreshToken call")
}

func (f *fakeAuthClient) RecordAuthDecision(context.Context, *pb.AuthAuditRequest, ...gogrpc.CallOption) (*pb.CommonResponse, error) {
	panic("unexpected RecordAuthDecision call")
}

func (f *fakeAuthClient) GetPrincipal(context.Context, *pb.GetPrincipalRequest, ...gogrpc.CallOption) (*pb.GetPrincipalResponse, error) {
	panic("unexpected GetPrincipal call")
}

func (f *fakeAuthClient) AuthorizeInternal(_ context.Context, req *pb.AuthorizeInternalRequest, _ ...gogrpc.CallOption) (*pb.AuthorizeInternalResponse, error) {
	f.authorizeRequests = append(f.authorizeRequests, req)
	if f.authorizeErr != nil {
		return nil, f.authorizeErr
	}
	if f.authorizeResp != nil {
		return f.authorizeResp, nil
	}
	return &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}, nil
}

func (f *fakeAuthClient) UpdateEmail(context.Context, *pb.UpdateEmailRequest, ...gogrpc.CallOption) (*pb.CommonResponse, error) {
	panic("unexpected UpdateEmail call")
}
