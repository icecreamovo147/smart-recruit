package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestRuntimeRegistersRecruitmentGRPCServices(t *testing.T) {
	runtime, err := New(Deps{
		Job:           fakeJobAPI{},
		JobTaxonomy:   fakeJobTaxonomyAPI{},
		TaxonomyAdmin: fakeTaxonomyAdminAPI{},
		Admin:         fakeRecruitmentAdminAPI{},
		UsageStats:    fakeUsageStatsAPI{},
		Candidate:     fakeCandidateAPI{},
		Application:   fakeApplicationAPI{},
		Collaboration: fakeCollaborationService{},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)
	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC returned error: %v", err)
	}
	services := server.GetServiceInfo()
	for _, serviceName := range []string{
		pb.JobService_ServiceDesc.ServiceName,
		pb.AdminService_ServiceDesc.ServiceName,
		pb.CandidateService_ServiceDesc.ServiceName,
		pb.ApplicationService_ServiceDesc.ServiceName,
		pb.CollaborationService_ServiceDesc.ServiceName,
	} {
		if _, ok := services[serviceName]; !ok {
			t.Fatalf("missing registered service %s", serviceName)
		}
	}
}

func TestRuntimeRequiresDependencies(t *testing.T) {
	if _, err := New(Deps{}); err == nil {
		t.Fatal("expected missing deps error")
	}
}

type fakeJobAPI struct{}

func (fakeJobAPI) CreateJob(context.Context, *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	return &pb.CreateJobResponse{Code: errs.OK}, nil
}

func (fakeJobAPI) UpdateJob(context.Context, *pb.UpdateJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeJobAPI) OfflineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeJobAPI) OnlineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeJobAPI) ListHRJobs(context.Context, *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}

func (fakeJobAPI) ListPublicJobs(context.Context, *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error) {
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}

func (fakeJobAPI) GetJobDetail(context.Context, *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	return &pb.GetJobDetailResponse{Code: errs.OK}, nil
}

type fakeJobTaxonomyAPI struct{}

func (fakeJobTaxonomyAPI) ListJobOptions(context.Context, *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error) {
	return &pb.ListJobOptionsResponse{Code: errs.OK}, nil
}

func (fakeJobTaxonomyAPI) ListDepartmentLocations(context.Context, *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error) {
	return &pb.ListDepartmentLocationsResponse{Code: errs.OK}, nil
}

type fakeTaxonomyAdminAPI struct{}

func (fakeTaxonomyAdminAPI) ListDepartments(context.Context, *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error) {
	return &pb.ListDepartmentsResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) CreateDepartment(context.Context, *pb.CreateDepartmentRequest) (*pb.DepartmentResponse, error) {
	return &pb.DepartmentResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) UpdateDepartment(context.Context, *pb.UpdateDepartmentRequest) (*pb.DepartmentResponse, error) {
	return &pb.DepartmentResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) UpdateDepartmentStatus(context.Context, *pb.UpdateDepartmentStatusRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) DeleteDepartment(context.Context, *pb.DeleteDepartmentRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) ListJobLocations(context.Context, *pb.ListJobLocationsRequest) (*pb.ListJobLocationsResponse, error) {
	return &pb.ListJobLocationsResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) CreateJobLocation(context.Context, *pb.CreateJobLocationRequest) (*pb.JobLocationResponse, error) {
	return &pb.JobLocationResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) UpdateJobLocation(context.Context, *pb.UpdateJobLocationRequest) (*pb.JobLocationResponse, error) {
	return &pb.JobLocationResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) UpdateJobLocationStatus(context.Context, *pb.UpdateJobLocationStatusRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) DeleteJobLocation(context.Context, *pb.DeleteJobLocationRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) GetDepartmentLocationConfig(context.Context, *pb.GetDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	return &pb.DepartmentLocationConfigResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) UpdateDepartmentLocationConfig(context.Context, *pb.UpdateDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	return &pb.DepartmentLocationConfigResponse{Code: errs.OK}, nil
}

func (fakeTaxonomyAdminAPI) ListDepartmentsLocationMap(context.Context, *pb.ListDepartmentsLocationMapRequest) (*pb.ListDepartmentsLocationMapResponse, error) {
	return &pb.ListDepartmentsLocationMapResponse{Code: errs.OK}, nil
}

type fakeRecruitmentAdminAPI struct{}

func (fakeRecruitmentAdminAPI) CreateInviteCode(context.Context, *pb.CreateInviteCodeRequest) (*pb.CreateInviteCodeResponse, error) {
	return &pb.CreateInviteCodeResponse{Code: errs.OK}, nil
}

func (fakeRecruitmentAdminAPI) ListInviteCodes(context.Context, *pb.ListInviteCodesRequest) (*pb.ListInviteCodesResponse, error) {
	return &pb.ListInviteCodesResponse{Code: errs.OK}, nil
}

func (fakeRecruitmentAdminAPI) ExtendInviteCode(context.Context, *pb.ExtendInviteCodeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeRecruitmentAdminAPI) RevokeInviteCode(context.Context, *pb.RevokeInviteCodeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeRecruitmentAdminAPI) ReactivateInviteCode(context.Context, *pb.ReactivateInviteCodeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeRecruitmentAdminAPI) ValidateInviteCode(context.Context, *pb.ValidateInviteCodeRequest) (*pb.ValidateInviteCodeResponse, error) {
	return &pb.ValidateInviteCodeResponse{Code: errs.OK}, nil
}

func (fakeRecruitmentAdminAPI) QueryUsageLogs(context.Context, *pb.QueryUsageLogsRequest) (*pb.QueryUsageLogsResponse, error) {
	return &pb.QueryUsageLogsResponse{Code: errs.OK}, nil
}

type fakeUsageStatsAPI struct{}

func (fakeUsageStatsAPI) GetUsageStats(context.Context, *pb.GetUsageStatsRequest) (*pb.GetUsageStatsResponse, error) {
	return &pb.GetUsageStatsResponse{Code: errs.OK}, nil
}

func (fakeUsageStatsAPI) GetUsageTrend(context.Context, *pb.GetUsageTrendRequest) (*pb.GetUsageTrendResponse, error) {
	return &pb.GetUsageTrendResponse{Code: errs.OK}, nil
}

type fakeCandidateAPI struct{}

func (fakeCandidateAPI) GetProfile(context.Context, *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	return &pb.GetProfileResponse{Code: errs.OK}, nil
}

func (fakeCandidateAPI) UpdateProfile(context.Context, *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error) {
	return &pb.GetProfileResponse{Code: errs.OK}, nil
}

func (fakeCandidateAPI) GetResume(context.Context, *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	return &pb.GetResumeResponse{Code: errs.OK}, nil
}

func (fakeCandidateAPI) PresignResumeUpload(context.Context, *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error) {
	return &pb.PresignResumeUploadResponse{Code: errs.OK}, nil
}

func (fakeCandidateAPI) ConfirmResumeUpload(context.Context, *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error) {
	return &pb.ConfirmResumeUploadResponse{Code: errs.OK}, nil
}

type fakeApplicationAPI struct{}

func (fakeApplicationAPI) ApplyJob(context.Context, *pb.ApplyJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeApplicationAPI) ListMyApplications(context.Context, *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	return &pb.ListMyApplicationsResponse{Code: errs.OK}, nil
}

func (fakeApplicationAPI) ListJobApplications(context.Context, *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	return &pb.ListJobApplicationsResponse{Code: errs.OK}, nil
}

func (fakeApplicationAPI) UpdateApplicationStatus(context.Context, *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeApplicationAPI) ListApplicationStatusTransitions(context.Context, *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error) {
	return &pb.ListApplicationStatusTransitionsResponse{Code: errs.OK}, nil
}

type fakeCollaborationService struct {
	pb.UnimplementedCollaborationServiceServer
}
