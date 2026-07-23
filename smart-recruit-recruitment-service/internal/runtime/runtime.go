package runtime

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

const ServiceName = "recruitment-service"

type JobAPI interface {
	CreateJob(context.Context, *pb.CreateJobRequest) (*pb.CreateJobResponse, error)
	UpdateJob(context.Context, *pb.UpdateJobRequest) (*pb.CommonResponse, error)
	OfflineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error)
	OnlineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error)
	ListHRJobs(context.Context, *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error)
	ListPublicJobs(context.Context, *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error)
	GetJobDetail(context.Context, *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error)
}

type JobTaxonomyAPI interface {
	ListJobOptions(context.Context, *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error)
	ListDepartmentLocations(context.Context, *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error)
}

type TaxonomyAdminAPI interface {
	ListDepartments(context.Context, *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error)
	CreateDepartment(context.Context, *pb.CreateDepartmentRequest) (*pb.DepartmentResponse, error)
	UpdateDepartment(context.Context, *pb.UpdateDepartmentRequest) (*pb.DepartmentResponse, error)
	UpdateDepartmentStatus(context.Context, *pb.UpdateDepartmentStatusRequest) (*pb.CommonResponse, error)
	DeleteDepartment(context.Context, *pb.DeleteDepartmentRequest) (*pb.CommonResponse, error)
	ListJobLocations(context.Context, *pb.ListJobLocationsRequest) (*pb.ListJobLocationsResponse, error)
	CreateJobLocation(context.Context, *pb.CreateJobLocationRequest) (*pb.JobLocationResponse, error)
	UpdateJobLocation(context.Context, *pb.UpdateJobLocationRequest) (*pb.JobLocationResponse, error)
	UpdateJobLocationStatus(context.Context, *pb.UpdateJobLocationStatusRequest) (*pb.CommonResponse, error)
	DeleteJobLocation(context.Context, *pb.DeleteJobLocationRequest) (*pb.CommonResponse, error)
	GetDepartmentLocationConfig(context.Context, *pb.GetDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error)
	UpdateDepartmentLocationConfig(context.Context, *pb.UpdateDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error)
	ListDepartmentsLocationMap(context.Context, *pb.ListDepartmentsLocationMapRequest) (*pb.ListDepartmentsLocationMapResponse, error)
}

type RecruitmentAdminAPI interface {
	CreateInviteCode(context.Context, *pb.CreateInviteCodeRequest) (*pb.CreateInviteCodeResponse, error)
	ListInviteCodes(context.Context, *pb.ListInviteCodesRequest) (*pb.ListInviteCodesResponse, error)
	ExtendInviteCode(context.Context, *pb.ExtendInviteCodeRequest) (*pb.CommonResponse, error)
	RevokeInviteCode(context.Context, *pb.RevokeInviteCodeRequest) (*pb.CommonResponse, error)
	ReactivateInviteCode(context.Context, *pb.ReactivateInviteCodeRequest) (*pb.CommonResponse, error)
	ValidateInviteCode(context.Context, *pb.ValidateInviteCodeRequest) (*pb.ValidateInviteCodeResponse, error)
	QueryUsageLogs(context.Context, *pb.QueryUsageLogsRequest) (*pb.QueryUsageLogsResponse, error)
}

type UsageStatsAPI interface {
	GetUsageStats(context.Context, *pb.GetUsageStatsRequest) (*pb.GetUsageStatsResponse, error)
	GetUsageTrend(context.Context, *pb.GetUsageTrendRequest) (*pb.GetUsageTrendResponse, error)
}

type CandidateAPI interface {
	GetProfile(context.Context, *pb.GetProfileRequest) (*pb.GetProfileResponse, error)
	UpdateProfile(context.Context, *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error)
	FillProfileFromResume(context.Context, *pb.FillProfileFromResumeRequest) (*pb.FillProfileFromResumeResponse, error)
	ApplyProfileFill(context.Context, *pb.ApplyProfileFillRequest) (*pb.GetProfileResponse, error)
	GetResume(context.Context, *pb.GetResumeRequest) (*pb.GetResumeResponse, error)
	PresignResumeUpload(context.Context, *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error)
	ConfirmResumeUpload(context.Context, *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error)
}

type ApplicationAPI interface {
	ApplyJob(context.Context, *pb.ApplyJobRequest) (*pb.CommonResponse, error)
	ListMyApplications(context.Context, *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error)
	ListJobApplications(context.Context, *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error)
	UpdateApplicationStatus(context.Context, *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error)
	ListApplicationStatusTransitions(context.Context, *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error)
}

type ApplicationOwnerContractAPI interface {
	GetApplicationSnapshot(context.Context, *pb.GetApplicationSnapshotRequest) (*pb.GetApplicationSnapshotResponse, error)
	ApplyApplicationLifecycleTransition(context.Context, *pb.ApplyApplicationLifecycleTransitionRequest) (*pb.ApplyApplicationLifecycleTransitionResponse, error)
}

type Deps struct {
	Job                      JobAPI
	JobTaxonomy              JobTaxonomyAPI
	TaxonomyAdmin            TaxonomyAdminAPI
	Admin                    RecruitmentAdminAPI
	UsageStats               UsageStatsAPI
	Candidate                CandidateAPI
	Application              ApplicationAPI
	ApplicationOwnerContract ApplicationOwnerContractAPI
	Collaboration            pb.CollaborationServiceServer
}

type Runtime struct {
	Job              pb.JobServiceServer
	Admin            pb.AdminServiceServer
	Candidate        pb.CandidateServiceServer
	Application      pb.ApplicationServiceServer
	ApplicationOwner pb.ApplicationOwnerServiceServer
	Collaboration    pb.CollaborationServiceServer
}

func New(deps Deps) (*Runtime, error) {
	if deps.Job == nil {
		return nil, fmt.Errorf("recruitment job api is required")
	}
	if deps.JobTaxonomy == nil {
		return nil, fmt.Errorf("recruitment job taxonomy api is required")
	}
	if deps.TaxonomyAdmin == nil {
		return nil, fmt.Errorf("recruitment taxonomy admin api is required")
	}
	if deps.Admin == nil {
		return nil, fmt.Errorf("recruitment admin api is required")
	}
	if deps.UsageStats == nil {
		return nil, fmt.Errorf("recruitment usage stats api is required")
	}
	if deps.Candidate == nil {
		return nil, fmt.Errorf("recruitment candidate api is required")
	}
	if deps.Application == nil {
		return nil, fmt.Errorf("recruitment application api is required")
	}
	if deps.ApplicationOwnerContract == nil {
		return nil, fmt.Errorf("recruitment application owner contract api is required")
	}
	if deps.Collaboration == nil {
		return nil, fmt.Errorf("recruitment collaboration api is required")
	}
	return &Runtime{
		Job: jobServer{api: deps.Job, taxonomy: deps.JobTaxonomy},
		Admin: adminServer{
			taxonomy:   deps.TaxonomyAdmin,
			admin:      deps.Admin,
			usageStats: deps.UsageStats,
		},
		Candidate:        candidateServer{api: deps.Candidate},
		Application:      applicationServer{api: deps.Application},
		ApplicationOwner: applicationOwnerServer{owner: deps.ApplicationOwnerContract},
		Collaboration:    deps.Collaboration,
	}, nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if r == nil || r.Job == nil || r.Admin == nil || r.Candidate == nil || r.Application == nil || r.ApplicationOwner == nil || r.Collaboration == nil {
		return fmt.Errorf("recruitment runtime is not initialized")
	}
	pb.RegisterJobServiceServer(registrar, r.Job)
	pb.RegisterAdminServiceServer(registrar, r.Admin)
	pb.RegisterCandidateServiceServer(registrar, r.Candidate)
	pb.RegisterApplicationServiceServer(registrar, r.Application)
	pb.RegisterApplicationOwnerServiceServer(registrar, r.ApplicationOwner)
	pb.RegisterCollaborationServiceServer(registrar, r.Collaboration)
	return nil
}

type jobServer struct {
	pb.UnimplementedJobServiceServer
	api      JobAPI
	taxonomy JobTaxonomyAPI
}

func (s jobServer) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	return s.api.CreateJob(ctx, req)
}

func (s jobServer) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateJob(ctx, req)
}

func (s jobServer) OfflineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return s.api.OfflineJob(ctx, req)
}

func (s jobServer) OnlineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return s.api.OnlineJob(ctx, req)
}

func (s jobServer) ListHRJobs(ctx context.Context, req *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	return s.api.ListHRJobs(ctx, req)
}

func (s jobServer) ListPublicJobs(ctx context.Context, req *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error) {
	return s.api.ListPublicJobs(ctx, req)
}

func (s jobServer) GetJobDetail(ctx context.Context, req *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	return s.api.GetJobDetail(ctx, req)
}

func (s jobServer) ListJobOptions(ctx context.Context, req *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error) {
	return s.taxonomy.ListJobOptions(ctx, req)
}

func (s jobServer) ListDepartmentLocations(ctx context.Context, req *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error) {
	return s.taxonomy.ListDepartmentLocations(ctx, req)
}

type candidateServer struct {
	pb.UnimplementedCandidateServiceServer
	api CandidateAPI
}

func (s candidateServer) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	return s.api.GetProfile(ctx, req)
}

func (s candidateServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error) {
	return s.api.UpdateProfile(ctx, req)
}

func (s candidateServer) FillProfileFromResume(ctx context.Context, req *pb.FillProfileFromResumeRequest) (*pb.FillProfileFromResumeResponse, error) {
	return s.api.FillProfileFromResume(ctx, req)
}

func (s candidateServer) ApplyProfileFill(ctx context.Context, req *pb.ApplyProfileFillRequest) (*pb.GetProfileResponse, error) {
	return s.api.ApplyProfileFill(ctx, req)
}

func (s candidateServer) GetResume(ctx context.Context, req *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	return s.api.GetResume(ctx, req)
}

func (s candidateServer) PresignResumeUpload(ctx context.Context, req *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error) {
	return s.api.PresignResumeUpload(ctx, req)
}

func (s candidateServer) ConfirmResumeUpload(ctx context.Context, req *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error) {
	return s.api.ConfirmResumeUpload(ctx, req)
}

type applicationServer struct {
	pb.UnimplementedApplicationServiceServer
	api ApplicationAPI
}

func (s applicationServer) ApplyJob(ctx context.Context, req *pb.ApplyJobRequest) (*pb.CommonResponse, error) {
	return s.api.ApplyJob(ctx, req)
}

func (s applicationServer) ListMyApplications(ctx context.Context, req *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	return s.api.ListMyApplications(ctx, req)
}

func (s applicationServer) ListJobApplications(ctx context.Context, req *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	return s.api.ListJobApplications(ctx, req)
}

func (s applicationServer) UpdateApplicationStatus(ctx context.Context, req *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateApplicationStatus(ctx, req)
}

func (s applicationServer) ListApplicationStatusTransitions(ctx context.Context, req *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error) {
	return s.api.ListApplicationStatusTransitions(ctx, req)
}

type applicationOwnerServer struct {
	pb.UnimplementedApplicationOwnerServiceServer
	owner ApplicationOwnerContractAPI
}

func (s applicationOwnerServer) GetApplicationSnapshot(ctx context.Context, req *pb.GetApplicationSnapshotRequest) (*pb.GetApplicationSnapshotResponse, error) {
	if s.owner == nil {
		return &pb.GetApplicationSnapshotResponse{Code: errs.ErrInternal, Msg: "application owner contract not configured"}, nil
	}
	return s.owner.GetApplicationSnapshot(ctx, req)
}

func (s applicationOwnerServer) ApplyApplicationLifecycleTransition(ctx context.Context, req *pb.ApplyApplicationLifecycleTransitionRequest) (*pb.ApplyApplicationLifecycleTransitionResponse, error) {
	if s.owner == nil {
		return &pb.ApplyApplicationLifecycleTransitionResponse{Code: errs.ErrInternal, Msg: "application owner contract not configured"}, nil
	}
	return s.owner.ApplyApplicationLifecycleTransition(ctx, req)
}

type adminServer struct {
	pb.UnimplementedAdminServiceServer
	taxonomy   TaxonomyAdminAPI
	admin      RecruitmentAdminAPI
	usageStats UsageStatsAPI
}

func (s adminServer) CreateInviteCode(ctx context.Context, req *pb.CreateInviteCodeRequest) (*pb.CreateInviteCodeResponse, error) {
	return s.admin.CreateInviteCode(ctx, req)
}

func (s adminServer) ListInviteCodes(ctx context.Context, req *pb.ListInviteCodesRequest) (*pb.ListInviteCodesResponse, error) {
	return s.admin.ListInviteCodes(ctx, req)
}

func (s adminServer) ExtendInviteCode(ctx context.Context, req *pb.ExtendInviteCodeRequest) (*pb.CommonResponse, error) {
	return s.admin.ExtendInviteCode(ctx, req)
}

func (s adminServer) RevokeInviteCode(ctx context.Context, req *pb.RevokeInviteCodeRequest) (*pb.CommonResponse, error) {
	return s.admin.RevokeInviteCode(ctx, req)
}

func (s adminServer) ReactivateInviteCode(ctx context.Context, req *pb.ReactivateInviteCodeRequest) (*pb.CommonResponse, error) {
	return s.admin.ReactivateInviteCode(ctx, req)
}

func (s adminServer) ValidateInviteCode(ctx context.Context, req *pb.ValidateInviteCodeRequest) (*pb.ValidateInviteCodeResponse, error) {
	return s.admin.ValidateInviteCode(ctx, req)
}

func (s adminServer) ListDepartments(ctx context.Context, req *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error) {
	return s.taxonomy.ListDepartments(ctx, req)
}

func (s adminServer) CreateDepartment(ctx context.Context, req *pb.CreateDepartmentRequest) (*pb.DepartmentResponse, error) {
	return s.taxonomy.CreateDepartment(ctx, req)
}

func (s adminServer) UpdateDepartment(ctx context.Context, req *pb.UpdateDepartmentRequest) (*pb.DepartmentResponse, error) {
	return s.taxonomy.UpdateDepartment(ctx, req)
}

func (s adminServer) UpdateDepartmentStatus(ctx context.Context, req *pb.UpdateDepartmentStatusRequest) (*pb.CommonResponse, error) {
	return s.taxonomy.UpdateDepartmentStatus(ctx, req)
}

func (s adminServer) DeleteDepartment(ctx context.Context, req *pb.DeleteDepartmentRequest) (*pb.CommonResponse, error) {
	return s.taxonomy.DeleteDepartment(ctx, req)
}

func (s adminServer) ListJobLocations(ctx context.Context, req *pb.ListJobLocationsRequest) (*pb.ListJobLocationsResponse, error) {
	return s.taxonomy.ListJobLocations(ctx, req)
}

func (s adminServer) CreateJobLocation(ctx context.Context, req *pb.CreateJobLocationRequest) (*pb.JobLocationResponse, error) {
	return s.taxonomy.CreateJobLocation(ctx, req)
}

func (s adminServer) UpdateJobLocation(ctx context.Context, req *pb.UpdateJobLocationRequest) (*pb.JobLocationResponse, error) {
	return s.taxonomy.UpdateJobLocation(ctx, req)
}

func (s adminServer) UpdateJobLocationStatus(ctx context.Context, req *pb.UpdateJobLocationStatusRequest) (*pb.CommonResponse, error) {
	return s.taxonomy.UpdateJobLocationStatus(ctx, req)
}

func (s adminServer) DeleteJobLocation(ctx context.Context, req *pb.DeleteJobLocationRequest) (*pb.CommonResponse, error) {
	return s.taxonomy.DeleteJobLocation(ctx, req)
}

func (s adminServer) GetDepartmentLocationConfig(ctx context.Context, req *pb.GetDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	return s.taxonomy.GetDepartmentLocationConfig(ctx, req)
}

func (s adminServer) UpdateDepartmentLocationConfig(ctx context.Context, req *pb.UpdateDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	return s.taxonomy.UpdateDepartmentLocationConfig(ctx, req)
}

func (s adminServer) ListDepartmentsLocationMap(ctx context.Context, req *pb.ListDepartmentsLocationMapRequest) (*pb.ListDepartmentsLocationMapResponse, error) {
	return s.taxonomy.ListDepartmentsLocationMap(ctx, req)
}

func (s adminServer) QueryUsageLogs(ctx context.Context, req *pb.QueryUsageLogsRequest) (*pb.QueryUsageLogsResponse, error) {
	return s.admin.QueryUsageLogs(ctx, req)
}

func (s adminServer) GetUsageStats(ctx context.Context, req *pb.GetUsageStatsRequest) (*pb.GetUsageStatsResponse, error) {
	return s.usageStats.GetUsageStats(ctx, req)
}

func (s adminServer) GetUsageTrend(ctx context.Context, req *pb.GetUsageTrendRequest) (*pb.GetUsageTrendResponse, error) {
	return s.usageStats.GetUsageTrend(ctx, req)
}
