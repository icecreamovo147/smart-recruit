package grpc

import (
	"context"
	"errors"
	"testing"

	gogrpc "google.golang.org/grpc"

	"smart-recruit-platform-go/errs"
	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

const testRecruitingStaffUserID int64 = 100

func TestVerifyRecruitingStaffActorRejectsMissingStaffUser(t *testing.T) {
	authErr := nativeRecruitingIntelligenceService{}.verifyRecruitingStaffActor(recruitingAuthContext(), 0)
	assertRecruitingAuthCode(t, authErr, errs.ErrBadRequest)
}

func TestVerifyRecruitingStaffActorRejectsMissingAuthenticatedMetadata(t *testing.T) {
	authErr := nativeRecruitingIntelligenceService{}.verifyRecruitingStaffActor(context.Background(), testRecruitingStaffUserID)
	assertRecruitingAuthCode(t, authErr, errs.ErrForbidden)
}

func TestVerifyRecruitingStaffActorRejectsMetadataUserMismatch(t *testing.T) {
	ctx := platformmetadata.WithAuthActor(context.Background(), testRecruitingStaffUserID+1, "staff")
	authErr := nativeRecruitingIntelligenceService{}.verifyRecruitingStaffActor(ctx, testRecruitingStaffUserID)
	assertRecruitingAuthCode(t, authErr, errs.ErrForbidden)
}

func TestAuthorizeRecruitingApplicationRejectsMissingDependency(t *testing.T) {
	_, authErr := nativeRecruitingIntelligenceService{}.authorizeRecruitingApplication(recruitingAuthContext(), testRecruitingStaffUserID, 7001)
	assertRecruitingAuthCode(t, authErr, errs.ErrInternal)
}

func TestAuthorizeRecruitingApplicationReturnsSnapshot(t *testing.T) {
	snapshot := &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 7001, JobId: 9001}
	applications := &fakeApplicationOwnerClient{snapshot: snapshot}
	service := nativeRecruitingIntelligenceService{applications: applications}

	got, authErr := service.authorizeRecruitingApplication(recruitingAuthContext(), testRecruitingStaffUserID, 7001)
	if authErr != nil {
		t.Fatalf("authorizeRecruitingApplication authErr = %v", authErr)
	}
	if got != snapshot {
		t.Fatalf("snapshot = %+v, want same response %+v", got, snapshot)
	}
	if len(applications.requests) != 1 || applications.requests[0].GetApplicationId() != 7001 {
		t.Fatalf("GetApplicationSnapshot requests = %+v, want application 7001", applications.requests)
	}
}

func TestAuthorizeRecruitingApplicationPreservesForbidden(t *testing.T) {
	service := nativeRecruitingIntelligenceService{
		applications: &fakeApplicationOwnerClient{
			snapshot: &pb.GetApplicationSnapshotResponse{Code: errs.ErrForbidden, Msg: "forbidden"},
		},
	}

	_, authErr := service.authorizeRecruitingApplication(recruitingAuthContext(), testRecruitingStaffUserID, 7001)
	assertRecruitingAuthCode(t, authErr, errs.ErrForbidden)
}

func TestAuthorizeRecruitingJobRejectsMissingDependency(t *testing.T) {
	_, authErr := nativeRecruitingIntelligenceService{}.authorizeRecruitingJob(recruitingAuthContext(), testRecruitingStaffUserID, 9001)
	assertRecruitingAuthCode(t, authErr, errs.ErrInternal)
}

func TestAuthorizeRecruitingJobFindsJobOnPageOne(t *testing.T) {
	job := &pb.Job{JobId: 9001, HrId: testRecruitingStaffUserID, Title: "Backend Engineer"}
	jobs := &fakeJobClient{responses: map[int32]*pb.ListJobsResponse{
		1: {Code: errs.OK, Total: 1, List: []*pb.Job{job}},
	}}
	service := nativeRecruitingIntelligenceService{jobs: jobs}

	got, authErr := service.authorizeRecruitingJob(recruitingAuthContext(), testRecruitingStaffUserID, 9001)
	if authErr != nil {
		t.Fatalf("authorizeRecruitingJob authErr = %v", authErr)
	}
	if got != job {
		t.Fatalf("job = %+v, want same job %+v", got, job)
	}
	assertListHRJobsRequest(t, jobs.requests, 0, testRecruitingStaffUserID, 1, 100)
}

func TestAuthorizeRecruitingJobFindsJobOnLaterPage(t *testing.T) {
	job := &pb.Job{JobId: 9500, HrId: testRecruitingStaffUserID, Title: "Data Engineer"}
	jobs := &fakeJobClient{responses: map[int32]*pb.ListJobsResponse{
		1: {Code: errs.OK, Total: 101, List: makeRecruitingJobs(1, 100)},
		2: {Code: errs.OK, Total: 101, List: []*pb.Job{job}},
	}}
	service := nativeRecruitingIntelligenceService{jobs: jobs}

	got, authErr := service.authorizeRecruitingJob(recruitingAuthContext(), testRecruitingStaffUserID, 9500)
	if authErr != nil {
		t.Fatalf("authorizeRecruitingJob authErr = %v", authErr)
	}
	if got != job {
		t.Fatalf("job = %+v, want same job %+v", got, job)
	}
	assertListHRJobsRequest(t, jobs.requests, 0, testRecruitingStaffUserID, 1, 100)
	assertListHRJobsRequest(t, jobs.requests, 1, testRecruitingStaffUserID, 2, 100)
}

func TestAuthorizeRecruitingJobDeniesWhenScopedListDoesNotContainJob(t *testing.T) {
	jobs := &fakeJobClient{responses: map[int32]*pb.ListJobsResponse{
		1: {Code: errs.OK, Total: 1, List: []*pb.Job{{JobId: 9002, HrId: testRecruitingStaffUserID}}},
	}}
	service := nativeRecruitingIntelligenceService{jobs: jobs}

	_, authErr := service.authorizeRecruitingJob(recruitingAuthContext(), testRecruitingStaffUserID, 9001)
	assertRecruitingAuthCode(t, authErr, errs.ErrForbidden)
}

func TestAuthorizeRecruitingJobFailClosedForListErrors(t *testing.T) {
	t.Run("forbidden response", func(t *testing.T) {
		service := nativeRecruitingIntelligenceService{jobs: &fakeJobClient{responses: map[int32]*pb.ListJobsResponse{
			1: {Code: errs.ErrForbidden, Msg: "forbidden"},
		}}}

		_, authErr := service.authorizeRecruitingJob(recruitingAuthContext(), testRecruitingStaffUserID, 9001)
		assertRecruitingAuthCode(t, authErr, errs.ErrForbidden)
	})

	t.Run("grpc error", func(t *testing.T) {
		service := nativeRecruitingIntelligenceService{jobs: &fakeJobClient{errByPage: map[int32]error{
			1: errors.New("recruitment unavailable"),
		}}}

		_, authErr := service.authorizeRecruitingJob(recruitingAuthContext(), testRecruitingStaffUserID, 9001)
		assertRecruitingAuthCode(t, authErr, errs.ErrInternal)
	})
}

func recruitingAuthContext() context.Context {
	return platformmetadata.WithAuthActor(context.Background(), testRecruitingStaffUserID, "staff")
}

func assertRecruitingAuthCode(t *testing.T, authErr *recruitingAuthError, want int32) {
	t.Helper()
	if authErr == nil {
		t.Fatalf("authErr = nil, want code %d", want)
	}
	if authErr.code != want {
		t.Fatalf("authErr.code = %d, want %d; err=%v", authErr.code, want, authErr)
	}
}

func assertListHRJobsRequest(t *testing.T, requests []*pb.ListHRJobsRequest, index int, staffUserID int64, page, pageSize int32) {
	t.Helper()
	if len(requests) <= index {
		t.Fatalf("ListHRJobs requests len = %d, want index %d", len(requests), index)
	}
	req := requests[index]
	if req.GetHrId() != staffUserID || req.GetPage() != page || req.GetPageSize() != pageSize {
		t.Fatalf("ListHRJobs request[%d] = %+v, want hr=%d page=%d pageSize=%d", index, req, staffUserID, page, pageSize)
	}
}

func makeRecruitingJobs(startID int64, count int) []*pb.Job {
	jobs := make([]*pb.Job, 0, count)
	for i := 0; i < count; i++ {
		jobs = append(jobs, &pb.Job{JobId: startID + int64(i), HrId: testRecruitingStaffUserID})
	}
	return jobs
}

type fakeApplicationOwnerClient struct {
	snapshot *pb.GetApplicationSnapshotResponse
	err      error
	requests []*pb.GetApplicationSnapshotRequest
}

func (f *fakeApplicationOwnerClient) GetApplicationSnapshot(ctx context.Context, req *pb.GetApplicationSnapshotRequest, opts ...gogrpc.CallOption) (*pb.GetApplicationSnapshotResponse, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	return f.snapshot, nil
}

func (f *fakeApplicationOwnerClient) ApplyApplicationLifecycleTransition(context.Context, *pb.ApplyApplicationLifecycleTransitionRequest, ...gogrpc.CallOption) (*pb.ApplyApplicationLifecycleTransitionResponse, error) {
	panic("unexpected ApplyApplicationLifecycleTransition call")
}

type fakeJobClient struct {
	responses map[int32]*pb.ListJobsResponse
	errByPage map[int32]error
	requests  []*pb.ListHRJobsRequest
}

func (f *fakeJobClient) CreateJob(context.Context, *pb.CreateJobRequest, ...gogrpc.CallOption) (*pb.CreateJobResponse, error) {
	panic("unexpected CreateJob call")
}

func (f *fakeJobClient) UpdateJob(context.Context, *pb.UpdateJobRequest, ...gogrpc.CallOption) (*pb.CommonResponse, error) {
	panic("unexpected UpdateJob call")
}

func (f *fakeJobClient) OfflineJob(context.Context, *pb.OfflineJobRequest, ...gogrpc.CallOption) (*pb.CommonResponse, error) {
	panic("unexpected OfflineJob call")
}

func (f *fakeJobClient) OnlineJob(context.Context, *pb.OfflineJobRequest, ...gogrpc.CallOption) (*pb.CommonResponse, error) {
	panic("unexpected OnlineJob call")
}

func (f *fakeJobClient) ListHRJobs(ctx context.Context, req *pb.ListHRJobsRequest, opts ...gogrpc.CallOption) (*pb.ListJobsResponse, error) {
	f.requests = append(f.requests, req)
	if err := f.errByPage[req.GetPage()]; err != nil {
		return nil, err
	}
	if resp := f.responses[req.GetPage()]; resp != nil {
		return resp, nil
	}
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}

func (f *fakeJobClient) ListPublicJobs(context.Context, *pb.ListPublicJobsRequest, ...gogrpc.CallOption) (*pb.ListJobsResponse, error) {
	panic("unexpected ListPublicJobs call")
}

func (f *fakeJobClient) GetJobDetail(context.Context, *pb.GetJobDetailRequest, ...gogrpc.CallOption) (*pb.GetJobDetailResponse, error) {
	panic("unexpected GetJobDetail call")
}

func (f *fakeJobClient) ListJobOptions(context.Context, *pb.ListJobOptionsRequest, ...gogrpc.CallOption) (*pb.ListJobOptionsResponse, error) {
	panic("unexpected ListJobOptions call")
}

func (f *fakeJobClient) ListDepartmentLocations(context.Context, *pb.ListDepartmentLocationsRequest, ...gogrpc.CallOption) (*pb.ListDepartmentLocationsResponse, error) {
	panic("unexpected ListDepartmentLocations call")
}
