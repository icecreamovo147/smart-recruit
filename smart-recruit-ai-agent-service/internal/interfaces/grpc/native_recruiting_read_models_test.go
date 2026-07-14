package grpc

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestGetResumeProfileReadModelValidationAndStore(t *testing.T) {
	service := nativeRecruitingIntelligenceService{}

	missingID, err := service.GetResumeProfile(recruitingAuthContext(), &pb.GetResumeProfileRequest{StaffUserId: testRecruitingStaffUserID})
	if err != nil {
		t.Fatalf("GetResumeProfile missing identifier error = %v", err)
	}
	if missingID.GetCode() != errs.ErrBadRequest {
		t.Fatalf("missing identifier code = %d, want %d", missingID.GetCode(), errs.ErrBadRequest)
	}

	missingStore, err := service.GetResumeProfile(recruitingAuthContext(), &pb.GetResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
	if err != nil {
		t.Fatalf("GetResumeProfile missing store error = %v", err)
	}
	if missingStore.GetCode() != errs.ErrInternal {
		t.Fatalf("missing store code = %d, want %d", missingStore.GetCode(), errs.ErrInternal)
	}
}

func TestGetResumeProfileReadModelSuccessByResumeIDAuthorizesResolvedApplication(t *testing.T) {
	store := newFakeRecruitingReadStore()
	store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, CandidateName: "Ada", ResumeID: 8001}
	store.currentProfileByResumeID[8001] = RecruitingResumeProfileRow{ID: 6001, ResumeID: 8001, UserID: 3001, ParseRunID: 5001, Version: 2, IsCurrent: 1}
	store.resumeSnapshots[6001] = sampleRecruitingResumeSnapshot(6001, 8001, 3001)
	applications := &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}}
	service := nativeRecruitingIntelligenceService{store: store, applications: applications}

	resp, err := service.GetResumeProfile(recruitingAuthContext(), &pb.GetResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
	if err != nil {
		t.Fatalf("GetResumeProfile error = %v", err)
	}
	if resp.GetCode() != errs.OK {
		t.Fatalf("GetResumeProfile code = %d msg=%q, want OK", resp.GetCode(), resp.GetMsg())
	}
	if got := resp.GetProfile().GetProfile().GetFullName(); got != "Ada Lovelace" {
		t.Fatalf("profile full name = %q, want Ada Lovelace", got)
	}
	if len(resp.GetProfile().GetEducations()) != 1 || len(resp.GetProfile().GetExperiences()) != 1 || len(resp.GetProfile().GetProjects()) != 1 || len(resp.GetProfile().GetSkills()) != 1 {
		t.Fatalf("snapshot missing nested rows: %+v", resp.GetProfile())
	}
	if len(applications.requests) != 1 || applications.requests[0].GetApplicationId() != 7001 {
		t.Fatalf("application auth requests = %+v, want resolved application 7001", applications.requests)
	}
}

func TestGetResumeProfileReadModelRejectsApplicationResumeMismatch(t *testing.T) {
	store := newFakeRecruitingReadStore()
	store.applicationsByID[7001] = RecruitingApplicationContext{ApplicationID: 7001, JobID: 9001, ResumeID: 8001}
	service := nativeRecruitingIntelligenceService{
		store:        store,
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}

	resp, err := service.GetResumeProfile(recruitingAuthContext(), &pb.GetResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, ResumeId: 9999})
	if err != nil {
		t.Fatalf("GetResumeProfile error = %v", err)
	}
	if resp.GetCode() != errs.ErrBadRequest {
		t.Fatalf("mismatch code = %d msg=%q, want bad request", resp.GetCode(), resp.GetMsg())
	}
}

func TestGetResumeProfileReadModelNotFound(t *testing.T) {
	store := newFakeRecruitingReadStore()
	store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, JobID: 9001, ResumeID: 8001}
	service := nativeRecruitingIntelligenceService{
		store:        store,
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}

	resp, err := service.GetResumeProfile(recruitingAuthContext(), &pb.GetResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
	if err != nil {
		t.Fatalf("GetResumeProfile error = %v", err)
	}
	if resp.GetCode() != 404 {
		t.Fatalf("not found code = %d msg=%q, want 404", resp.GetCode(), resp.GetMsg())
	}
}

func TestParseResumeProfileReadThroughPath(t *testing.T) {
	t.Run("missing ids", func(t *testing.T) {
		resp, err := nativeRecruitingIntelligenceService{}.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID})
		if err != nil {
			t.Fatalf("ParseResumeProfile error = %v", err)
		}
		if resp.GetCode() != errs.ErrBadRequest {
			t.Fatalf("code = %d msg=%q, want bad request", resp.GetCode(), resp.GetMsg())
		}
	})

	t.Run("store nil", func(t *testing.T) {
		resp, err := nativeRecruitingIntelligenceService{}.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		if err != nil {
			t.Fatalf("ParseResumeProfile error = %v", err)
		}
		if resp.GetCode() != errs.ErrInternal {
			t.Fatalf("code = %d msg=%q, want internal", resp.GetCode(), resp.GetMsg())
		}
	})

	t.Run("existing profile success", func(t *testing.T) {
		store := newFakeRecruitingReadStore()
		store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, ResumeID: 8001}
		store.currentProfileByResumeID[8001] = RecruitingResumeProfileRow{ID: 6001, ResumeID: 8001, UserID: 3001, ParseRunID: 5001, Version: 2, IsCurrent: 1}
		store.resumeSnapshots[6001] = sampleRecruitingResumeSnapshot(6001, 8001, 3001)
		applications := &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}}
		service := nativeRecruitingIntelligenceService{store: store, applications: applications}

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
		if len(applications.requests) != 1 || applications.requests[0].GetApplicationId() != 7001 {
			t.Fatalf("application auth requests = %+v, want resolved application 7001", applications.requests)
		}
	})

	t.Run("no current profile does not fabricate", func(t *testing.T) {
		store := newFakeRecruitingReadStore()
		store.latestApplicationByResumeID[8001] = RecruitingApplicationContext{ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, ResumeID: 8001}
		service := nativeRecruitingIntelligenceService{
			store:        store,
			applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
		}

		resp, err := service.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ResumeId: 8001})
		if err != nil {
			t.Fatalf("ParseResumeProfile error = %v", err)
		}
		if resp.GetCode() == errs.OK || resp.GetProfile() != nil {
			t.Fatalf("response = %+v, want non-success without profile", resp)
		}
		if !strings.Contains(resp.GetMsg(), "current resume profile not found") || !strings.Contains(resp.GetMsg(), "parser") {
			t.Fatalf("msg = %q, want not found/parser unavailable", resp.GetMsg())
		}
		if len(store.resumeSnapshots) != 0 || len(store.currentProfileByResumeID) != 0 {
			t.Fatalf("store fabricated profile data: current=%+v snapshots=%+v", store.currentProfileByResumeID, store.resumeSnapshots)
		}
	})

	t.Run("application resume mismatch", func(t *testing.T) {
		store := newFakeRecruitingReadStore()
		store.applicationsByID[7001] = RecruitingApplicationContext{ApplicationID: 7001, JobID: 9001, ResumeID: 8001}
		service := nativeRecruitingIntelligenceService{
			store:        store,
			applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
		}

		resp, err := service.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, ResumeId: 9999})
		if err != nil {
			t.Fatalf("ParseResumeProfile error = %v", err)
		}
		if resp.GetCode() != errs.ErrBadRequest {
			t.Fatalf("code = %d msg=%q, want bad request", resp.GetCode(), resp.GetMsg())
		}
	})

	t.Run("unauthorized application", func(t *testing.T) {
		store := newFakeRecruitingReadStore()
		service := nativeRecruitingIntelligenceService{
			store:        store,
			applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.ErrForbidden, Msg: "denied"}},
		}

		resp, err := service.ParseResumeProfile(recruitingAuthContext(), &pb.ParseResumeProfileRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		if err != nil {
			t.Fatalf("ParseResumeProfile error = %v", err)
		}
		if resp.GetCode() != errs.ErrForbidden {
			t.Fatalf("code = %d msg=%q, want forbidden", resp.GetCode(), resp.GetMsg())
		}
	})
}

func TestGetCandidateMatchEvaluationReadModelResolution(t *testing.T) {
	store := newFakeRecruitingReadStore()
	latest := sampleCandidateMatchSnapshot(9103, 7001, 3, 91)
	version := sampleCandidateMatchSnapshot(9102, 7001, 2, 86)
	byID := sampleCandidateMatchSnapshot(9101, 7001, 1, 80)
	store.latestMatchByApp[7001] = latest
	store.matchByAppVersion[matchVersionKey(7001, 2)] = version
	store.matchSnapshots[9101] = byID
	service := nativeRecruitingIntelligenceService{
		store:        store,
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}

	tests := []struct {
		name      string
		req       *pb.GetCandidateMatchEvaluationRequest
		wantID    uint64
		wantScore float64
	}{
		{name: "latest", req: &pb.GetCandidateMatchEvaluationRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001}, wantID: 9103, wantScore: 91},
		{name: "version", req: &pb.GetCandidateMatchEvaluationRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, EvaluationVersion: 2}, wantID: 9102, wantScore: 86},
		{name: "id", req: &pb.GetCandidateMatchEvaluationRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, EvaluationId: 9101}, wantID: 9101, wantScore: 80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.GetCandidateMatchEvaluation(recruitingAuthContext(), tt.req)
			if err != nil {
				t.Fatalf("GetCandidateMatchEvaluation error = %v", err)
			}
			if resp.GetCode() != errs.OK {
				t.Fatalf("code = %d msg=%q, want OK", resp.GetCode(), resp.GetMsg())
			}
			evaluation := resp.GetEvaluation().GetEvaluation()
			if evaluation.GetId() != tt.wantID || evaluation.GetOverallScore() != tt.wantScore {
				t.Fatalf("evaluation = %+v, want id=%d score=%v", evaluation, tt.wantID, tt.wantScore)
			}
			if evaluation.GetMissingRequirementsJson() != `["r1","r2"]` {
				t.Fatalf("missing requirements = %s, want derived JSON", evaluation.GetMissingRequirementsJson())
			}
			if evaluation.GetDimensionsJson() == "[]" {
				t.Fatalf("dimensions json was not derived from score breakdown")
			}
			if len(resp.GetEvaluation().GetEvidence()) != 1 {
				t.Fatalf("evidence len = %d, want 1", len(resp.GetEvaluation().GetEvidence()))
			}
		})
	}
}

func TestGetCandidateMatchEvaluationReadModelHidesApplicationMismatch(t *testing.T) {
	store := newFakeRecruitingReadStore()
	store.matchSnapshots[9101] = sampleCandidateMatchSnapshot(9101, 7002, 1, 80)
	service := nativeRecruitingIntelligenceService{
		store:        store,
		applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
	}

	resp, err := service.GetCandidateMatchEvaluation(recruitingAuthContext(), &pb.GetCandidateMatchEvaluationRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, EvaluationId: 9101})
	if err != nil {
		t.Fatalf("GetCandidateMatchEvaluation error = %v", err)
	}
	if resp.GetCode() != 404 {
		t.Fatalf("application mismatch code = %d msg=%q, want 404", resp.GetCode(), resp.GetMsg())
	}
}

func TestEvaluateCandidateMatchReadThroughPath(t *testing.T) {
	t.Run("bad request", func(t *testing.T) {
		resp, err := nativeRecruitingIntelligenceService{}.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID})
		if err != nil {
			t.Fatalf("EvaluateCandidateMatch error = %v", err)
		}
		if resp.GetCode() != errs.ErrBadRequest {
			t.Fatalf("code = %d msg=%q, want bad request", resp.GetCode(), resp.GetMsg())
		}
	})

	t.Run("store nil", func(t *testing.T) {
		resp, err := nativeRecruitingIntelligenceService{}.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		if err != nil {
			t.Fatalf("EvaluateCandidateMatch error = %v", err)
		}
		if resp.GetCode() != errs.ErrInternal {
			t.Fatalf("code = %d msg=%q, want internal", resp.GetCode(), resp.GetMsg())
		}
	})

	t.Run("latest success", func(t *testing.T) {
		store := newFakeRecruitingReadStore()
		store.latestMatchByApp[7001] = sampleCandidateMatchSnapshot(9103, 7001, 3, 91)
		service := nativeRecruitingIntelligenceService{
			store:        store,
			applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
		}

		resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		if err != nil {
			t.Fatalf("EvaluateCandidateMatch error = %v", err)
		}
		if resp.GetCode() != errs.OK {
			t.Fatalf("code = %d msg=%q, want OK", resp.GetCode(), resp.GetMsg())
		}
		if got := resp.GetEvaluation().GetEvaluation().GetId(); got != 9103 {
			t.Fatalf("evaluation id = %d, want 9103", got)
		}
	})

	t.Run("agent run success", func(t *testing.T) {
		store := newFakeRecruitingReadStore()
		agentRunID := uint64(12001)
		snapshot := sampleCandidateMatchSnapshot(9104, 7001, 4, 93)
		snapshot.Evaluation.AgentRunID = &agentRunID
		store.matchByAppAgentRun[matchAgentRunKey(7001, agentRunID)] = snapshot
		service := nativeRecruitingIntelligenceService{
			store:        store,
			applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
		}

		resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, AgentRunId: agentRunID})
		if err != nil {
			t.Fatalf("EvaluateCandidateMatch error = %v", err)
		}
		if resp.GetCode() != errs.OK {
			t.Fatalf("code = %d msg=%q, want OK", resp.GetCode(), resp.GetMsg())
		}
		if got := resp.GetEvaluation().GetEvaluation().GetAgentRunId(); got != agentRunID {
			t.Fatalf("agent run id = %d, want %d", got, agentRunID)
		}
	})

	t.Run("agent run application mismatch hidden", func(t *testing.T) {
		store := newFakeRecruitingReadStore()
		agentRunID := uint64(12001)
		snapshot := sampleCandidateMatchSnapshot(9104, 7002, 1, 93)
		snapshot.Evaluation.AgentRunID = &agentRunID
		store.matchByAppAgentRun[matchAgentRunKey(7002, agentRunID)] = snapshot
		service := nativeRecruitingIntelligenceService{
			store:        store,
			applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
		}

		resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001, AgentRunId: agentRunID})
		if err != nil {
			t.Fatalf("EvaluateCandidateMatch error = %v", err)
		}
		if resp.GetCode() != 404 || resp.GetEvaluation() != nil {
			t.Fatalf("response = %+v, want 404 without leaked evaluation", resp)
		}
	})

	t.Run("no result does not fabricate", func(t *testing.T) {
		store := newFakeRecruitingReadStore()
		service := nativeRecruitingIntelligenceService{
			store:        store,
			applications: &fakeApplicationOwnerClient{snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, ResumeId: 8001, JobId: 9001}},
		}

		resp, err := service.EvaluateCandidateMatch(recruitingAuthContext(), &pb.EvaluateCandidateMatchRequest{StaffUserId: testRecruitingStaffUserID, ApplicationId: 7001})
		if err != nil {
			t.Fatalf("EvaluateCandidateMatch error = %v", err)
		}
		if resp.GetCode() == errs.OK || resp.GetEvaluation() != nil {
			t.Fatalf("response = %+v, want non-success without evaluation", resp)
		}
		if !strings.Contains(resp.GetMsg(), "candidate match evaluation not found") || !strings.Contains(resp.GetMsg(), "matcher") {
			t.Fatalf("msg = %q, want not found/matcher unavailable", resp.GetMsg())
		}
		if len(store.latestMatchByApp) != 0 || len(store.matchByAppAgentRun) != 0 {
			t.Fatalf("store fabricated evaluation data: latest=%+v agent=%+v", store.latestMatchByApp, store.matchByAppAgentRun)
		}
	})
}

func TestCompareCandidatesForJobReadModelOrderingMissingAndAuthDeny(t *testing.T) {
	store := newFakeRecruitingReadStore()
	store.applicationsByJob[9001] = []RecruitingApplicationContext{
		{ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, CandidateName: "Ada", ResumeID: 8001},
		{ApplicationID: 7002, JobID: 9001, CandidateUserID: 3002, CandidateName: "Grace", ResumeID: 8002},
		{ApplicationID: 7003, JobID: 9001, CandidateUserID: 3003, CandidateName: "Linus", ResumeID: 8003},
	}
	store.latestEvaluationsByApplicationID[7001] = sampleCandidateMatchEvaluation(9101, 7001, 1, 88)
	store.latestEvaluationsByApplicationID[7003] = sampleCandidateMatchEvaluation(9103, 7003, 1, 95)
	service := nativeRecruitingIntelligenceService{
		store: store,
		jobs: &fakeJobClient{responses: map[int32]*pb.ListJobsResponse{
			1: {Code: errs.OK, Total: 1, List: []*pb.Job{{JobId: 9001, HrId: testRecruitingStaffUserID}}},
		}},
	}

	resp, err := service.CompareCandidatesForJob(recruitingAuthContext(), &pb.CompareCandidatesForJobRequest{StaffUserId: testRecruitingStaffUserID, JobId: 9001})
	if err != nil {
		t.Fatalf("CompareCandidatesForJob error = %v", err)
	}
	if resp.GetCode() != errs.OK {
		t.Fatalf("code = %d msg=%q, want OK", resp.GetCode(), resp.GetMsg())
	}
	gotOrder := comparisonApplicationIDs(resp.GetCandidates())
	if want := []int64{7003, 7001, 7002}; !reflect.DeepEqual(gotOrder, want) {
		t.Fatalf("candidate order = %#v, want %#v", gotOrder, want)
	}
	if want := []int64{7002}; !reflect.DeepEqual(resp.GetMissingApplicationIds(), want) {
		t.Fatalf("missing application ids = %#v, want %#v", resp.GetMissingApplicationIds(), want)
	}

	denied := nativeRecruitingIntelligenceService{store: store, jobs: &fakeJobClient{responses: map[int32]*pb.ListJobsResponse{
		1: {Code: errs.OK, Total: 1, List: []*pb.Job{{JobId: 9999, HrId: testRecruitingStaffUserID}}},
	}}}
	deniedResp, err := denied.CompareCandidatesForJob(recruitingAuthContext(), &pb.CompareCandidatesForJobRequest{StaffUserId: testRecruitingStaffUserID, JobId: 9001})
	if err != nil {
		t.Fatalf("CompareCandidatesForJob denied error = %v", err)
	}
	if deniedResp.GetCode() != errs.ErrForbidden {
		t.Fatalf("denied code = %d msg=%q, want forbidden", deniedResp.GetCode(), deniedResp.GetMsg())
	}
}

func sampleRecruitingResumeSnapshot(profileID uint64, resumeID, userID int64) RecruitingResumeProfileSnapshot {
	started := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	return RecruitingResumeProfileSnapshot{
		ParseRun: RecruitingResumeParseRunRow{ID: 5001, ResumeID: resumeID, UserID: userID, Status: "succeeded", ParserVersion: "v2", InputHash: "hash", StartedAt: started, CompletedAt: &started, CreatedAt: started, UpdatedAt: started},
		Profile:  RecruitingResumeProfileRow{ID: profileID, ResumeID: resumeID, UserID: userID, ParseRunID: 5001, Version: 2, IsCurrent: 1, FullName: "Ada Lovelace", Email: "ada@example.com", Phone: "123", Location: "London", Headline: "Mathematician", Summary: "Analytical engine", TotalExperienceYears: 5.5, HighestDegree: "BS", RawJSON: `{"name":"Ada"}`, CreatedAt: started, UpdatedAt: started},
		Educations: []RecruitingResumeEducationRow{
			{ID: 6101, School: "University", Degree: "BS", Major: "Math", SortOrder: 1},
		},
		Experiences: []RecruitingResumeExperienceRow{
			{ID: 6201, Company: "Engines", Title: "Engineer", IsCurrent: 1, AchievementsJSON: `["analysis"]`, SortOrder: 1},
		},
		Projects: []RecruitingResumeProjectRow{
			{ID: 6301, Name: "Compiler", Role: "Lead", TechnologiesJSON: `["go"]`, HighlightsJSON: `["first"]`, SortOrder: 1},
		},
		Skills: []RecruitingResumeSkillRow{
			{ID: 6401, Name: "Go", Category: "language", Level: "senior", Years: 4, Evidence: "projects", SortOrder: 1},
		},
	}
}

func sampleCandidateMatchSnapshot(evaluationID uint64, applicationID int64, version int32, score float64) RecruitingCandidateMatchSnapshot {
	evaluatedAt := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	return RecruitingCandidateMatchSnapshot{
		Evaluation: sampleCandidateMatchEvaluation(evaluationID, applicationID, version, score),
		Evidence: []RecruitingCandidateMatchEvidenceRow{
			{ID: evaluationID + 100, EvidenceType: "skill", Dimension: "backend", SourceTable: "resume_skills", Snippet: "Go", Weight: 0.8, ScoreImpact: 4.5, MetadataJSON: `{"source":"resume"}`, CreatedAt: evaluatedAt},
		},
	}
}

func sampleCandidateMatchEvaluation(evaluationID uint64, applicationID int64, version int32, score float64) RecruitingCandidateMatchEvaluationRow {
	evaluatedAt := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	return RecruitingCandidateMatchEvaluationRow{
		ID:                 evaluationID,
		ApplicationID:      applicationID,
		JobID:              9001,
		CandidateUserID:    3001,
		ResumeProfileID:    6001,
		EvaluationVersion:  version,
		IsLatest:           1,
		OverallScore:       score,
		Recommendation:     "strong_match",
		Summary:            "Excellent",
		StrengthsJSON:      `["go"]`,
		RisksJSON:          `[]`,
		ScoreBreakdownJSON: `{"requirement_results":[{"requirement_id":"r1","status":"missing"},{"requirement_id":"r2","status":"conflict"}],"dimensions":[{"name":"backend","score":90}]}`,
		ModelName:          "test-model",
		EvaluatedAt:        evaluatedAt,
		CreatedAt:          evaluatedAt,
		UpdatedAt:          evaluatedAt,
	}
}

func comparisonApplicationIDs(items []*pb.CandidateComparisonItem) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.GetApplicationId())
	}
	return ids
}

type fakeRecruitingReadStore struct {
	applicationsByID                 map[int64]RecruitingApplicationContext
	latestApplicationByResumeID      map[int64]RecruitingApplicationContext
	profilesByID                     map[uint64]RecruitingResumeProfileRow
	currentProfileByResumeID         map[int64]RecruitingResumeProfileRow
	resumeSnapshots                  map[uint64]RecruitingResumeProfileSnapshot
	matchSnapshots                   map[uint64]RecruitingCandidateMatchSnapshot
	matchByAppVersion                map[string]RecruitingCandidateMatchSnapshot
	matchByAppAgentRun               map[string]RecruitingCandidateMatchSnapshot
	latestMatchByApp                 map[int64]RecruitingCandidateMatchSnapshot
	applicationsByJob                map[int64][]RecruitingApplicationContext
	latestEvaluationsByApplicationID map[int64]RecruitingCandidateMatchEvaluationRow
	err                              error
}

func newFakeRecruitingReadStore() *fakeRecruitingReadStore {
	return &fakeRecruitingReadStore{
		applicationsByID:                 make(map[int64]RecruitingApplicationContext),
		latestApplicationByResumeID:      make(map[int64]RecruitingApplicationContext),
		profilesByID:                     make(map[uint64]RecruitingResumeProfileRow),
		currentProfileByResumeID:         make(map[int64]RecruitingResumeProfileRow),
		resumeSnapshots:                  make(map[uint64]RecruitingResumeProfileSnapshot),
		matchSnapshots:                   make(map[uint64]RecruitingCandidateMatchSnapshot),
		matchByAppVersion:                make(map[string]RecruitingCandidateMatchSnapshot),
		matchByAppAgentRun:               make(map[string]RecruitingCandidateMatchSnapshot),
		latestMatchByApp:                 make(map[int64]RecruitingCandidateMatchSnapshot),
		applicationsByJob:                make(map[int64][]RecruitingApplicationContext),
		latestEvaluationsByApplicationID: make(map[int64]RecruitingCandidateMatchEvaluationRow),
	}
}

func (f *fakeRecruitingReadStore) GetRecruitingApplicationByID(_ context.Context, applicationID int64) (RecruitingApplicationContext, bool, error) {
	if f.err != nil {
		return RecruitingApplicationContext{}, false, f.err
	}
	row, ok := f.applicationsByID[applicationID]
	return row, ok, nil
}

func (f *fakeRecruitingReadStore) GetLatestRecruitingApplicationByResumeID(_ context.Context, resumeID int64) (RecruitingApplicationContext, bool, error) {
	if f.err != nil {
		return RecruitingApplicationContext{}, false, f.err
	}
	row, ok := f.latestApplicationByResumeID[resumeID]
	return row, ok, nil
}

func (f *fakeRecruitingReadStore) GetRecruitingResumeProfileByID(_ context.Context, profileID uint64) (RecruitingResumeProfileRow, bool, error) {
	if f.err != nil {
		return RecruitingResumeProfileRow{}, false, f.err
	}
	row, ok := f.profilesByID[profileID]
	return row, ok, nil
}

func (f *fakeRecruitingReadStore) GetCurrentRecruitingResumeProfileByResumeID(_ context.Context, resumeID int64) (RecruitingResumeProfileRow, bool, error) {
	if f.err != nil {
		return RecruitingResumeProfileRow{}, false, f.err
	}
	row, ok := f.currentProfileByResumeID[resumeID]
	return row, ok, nil
}

func (f *fakeRecruitingReadStore) GetRecruitingResumeProfileSnapshot(_ context.Context, profileID uint64) (RecruitingResumeProfileSnapshot, bool, error) {
	if f.err != nil {
		return RecruitingResumeProfileSnapshot{}, false, f.err
	}
	row, ok := f.resumeSnapshots[profileID]
	return row, ok, nil
}

func (f *fakeRecruitingReadStore) GetRecruitingCandidateMatchEvaluationSnapshot(_ context.Context, evaluationID uint64) (RecruitingCandidateMatchSnapshot, bool, error) {
	if f.err != nil {
		return RecruitingCandidateMatchSnapshot{}, false, f.err
	}
	row, ok := f.matchSnapshots[evaluationID]
	return row, ok, nil
}

func (f *fakeRecruitingReadStore) GetRecruitingCandidateMatchEvaluationSnapshotByApplicationVersion(_ context.Context, applicationID int64, version int32) (RecruitingCandidateMatchSnapshot, bool, error) {
	if f.err != nil {
		return RecruitingCandidateMatchSnapshot{}, false, f.err
	}
	row, ok := f.matchByAppVersion[matchVersionKey(applicationID, version)]
	return row, ok, nil
}

func (f *fakeRecruitingReadStore) GetRecruitingCandidateMatchEvaluationSnapshotByApplicationAgentRunID(_ context.Context, applicationID int64, agentRunID uint64) (RecruitingCandidateMatchSnapshot, bool, error) {
	if f.err != nil {
		return RecruitingCandidateMatchSnapshot{}, false, f.err
	}
	row, ok := f.matchByAppAgentRun[matchAgentRunKey(applicationID, agentRunID)]
	return row, ok, nil
}

func (f *fakeRecruitingReadStore) GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(_ context.Context, applicationID int64) (RecruitingCandidateMatchSnapshot, bool, error) {
	if f.err != nil {
		return RecruitingCandidateMatchSnapshot{}, false, f.err
	}
	row, ok := f.latestMatchByApp[applicationID]
	return row, ok, nil
}

func (f *fakeRecruitingReadStore) ListCurrentRecruitingApplicationsByJobID(_ context.Context, jobID int64) ([]RecruitingApplicationContext, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]RecruitingApplicationContext(nil), f.applicationsByJob[jobID]...), nil
}

func (f *fakeRecruitingReadStore) ListLatestRecruitingCandidateMatchEvaluationsByApplicationIDs(_ context.Context, applicationIDs []int64) ([]RecruitingCandidateMatchEvaluationRow, error) {
	if f.err != nil {
		return nil, f.err
	}
	rows := make([]RecruitingCandidateMatchEvaluationRow, 0, len(applicationIDs))
	for _, applicationID := range applicationIDs {
		if row, ok := f.latestEvaluationsByApplicationID[applicationID]; ok {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func matchVersionKey(applicationID int64, version int32) string {
	return fmt.Sprintf("%d:%d", applicationID, version)
}

func matchAgentRunKey(applicationID int64, agentRunID uint64) string {
	return fmt.Sprintf("%d:%d", applicationID, agentRunID)
}
