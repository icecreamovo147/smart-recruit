package grpc

import (
	"context"
	"fmt"

	"smart-recruit-proto/recruitment/pb"
)

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

type JobAdapter struct{ delegate JobAPI }

func NewJobAdapter(delegate JobAPI) (*JobAdapter, error) {
	if delegate == nil {
		return nil, fmt.Errorf("job delegate is required")
	}
	return &JobAdapter{delegate: delegate}, nil
}

func (a *JobAdapter) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	return a.delegate.CreateJob(ctx, req)
}
func (a *JobAdapter) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.CommonResponse, error) {
	return a.delegate.UpdateJob(ctx, req)
}
func (a *JobAdapter) OfflineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return a.delegate.OfflineJob(ctx, req)
}
func (a *JobAdapter) OnlineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return a.delegate.OnlineJob(ctx, req)
}
func (a *JobAdapter) ListHRJobs(ctx context.Context, req *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	return a.delegate.ListHRJobs(ctx, req)
}
func (a *JobAdapter) ListPublicJobs(ctx context.Context, req *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error) {
	return a.delegate.ListPublicJobs(ctx, req)
}
func (a *JobAdapter) GetJobDetail(ctx context.Context, req *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	return a.delegate.GetJobDetail(ctx, req)
}

type JobTaxonomyAdapter struct{ delegate JobTaxonomyAPI }

func NewJobTaxonomyAdapter(delegate JobTaxonomyAPI) (*JobTaxonomyAdapter, error) {
	if delegate == nil {
		return nil, fmt.Errorf("job taxonomy delegate is required")
	}
	return &JobTaxonomyAdapter{delegate: delegate}, nil
}

func (a *JobTaxonomyAdapter) ListJobOptions(ctx context.Context, req *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error) {
	return a.delegate.ListJobOptions(ctx, req)
}
func (a *JobTaxonomyAdapter) ListDepartmentLocations(ctx context.Context, req *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error) {
	return a.delegate.ListDepartmentLocations(ctx, req)
}

type TaxonomyAdminAdapter struct{ delegate TaxonomyAdminAPI }

func NewTaxonomyAdminAdapter(delegate TaxonomyAdminAPI) (*TaxonomyAdminAdapter, error) {
	if delegate == nil {
		return nil, fmt.Errorf("taxonomy admin delegate is required")
	}
	return &TaxonomyAdminAdapter{delegate: delegate}, nil
}

func (a *TaxonomyAdminAdapter) ListDepartments(ctx context.Context, req *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error) {
	return a.delegate.ListDepartments(ctx, req)
}
func (a *TaxonomyAdminAdapter) CreateDepartment(ctx context.Context, req *pb.CreateDepartmentRequest) (*pb.DepartmentResponse, error) {
	return a.delegate.CreateDepartment(ctx, req)
}
func (a *TaxonomyAdminAdapter) UpdateDepartment(ctx context.Context, req *pb.UpdateDepartmentRequest) (*pb.DepartmentResponse, error) {
	return a.delegate.UpdateDepartment(ctx, req)
}
func (a *TaxonomyAdminAdapter) UpdateDepartmentStatus(ctx context.Context, req *pb.UpdateDepartmentStatusRequest) (*pb.CommonResponse, error) {
	return a.delegate.UpdateDepartmentStatus(ctx, req)
}
func (a *TaxonomyAdminAdapter) DeleteDepartment(ctx context.Context, req *pb.DeleteDepartmentRequest) (*pb.CommonResponse, error) {
	return a.delegate.DeleteDepartment(ctx, req)
}
func (a *TaxonomyAdminAdapter) ListJobLocations(ctx context.Context, req *pb.ListJobLocationsRequest) (*pb.ListJobLocationsResponse, error) {
	return a.delegate.ListJobLocations(ctx, req)
}
func (a *TaxonomyAdminAdapter) CreateJobLocation(ctx context.Context, req *pb.CreateJobLocationRequest) (*pb.JobLocationResponse, error) {
	return a.delegate.CreateJobLocation(ctx, req)
}
func (a *TaxonomyAdminAdapter) UpdateJobLocation(ctx context.Context, req *pb.UpdateJobLocationRequest) (*pb.JobLocationResponse, error) {
	return a.delegate.UpdateJobLocation(ctx, req)
}
func (a *TaxonomyAdminAdapter) UpdateJobLocationStatus(ctx context.Context, req *pb.UpdateJobLocationStatusRequest) (*pb.CommonResponse, error) {
	return a.delegate.UpdateJobLocationStatus(ctx, req)
}
func (a *TaxonomyAdminAdapter) DeleteJobLocation(ctx context.Context, req *pb.DeleteJobLocationRequest) (*pb.CommonResponse, error) {
	return a.delegate.DeleteJobLocation(ctx, req)
}
func (a *TaxonomyAdminAdapter) GetDepartmentLocationConfig(ctx context.Context, req *pb.GetDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	return a.delegate.GetDepartmentLocationConfig(ctx, req)
}
func (a *TaxonomyAdminAdapter) UpdateDepartmentLocationConfig(ctx context.Context, req *pb.UpdateDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	return a.delegate.UpdateDepartmentLocationConfig(ctx, req)
}
func (a *TaxonomyAdminAdapter) ListDepartmentsLocationMap(ctx context.Context, req *pb.ListDepartmentsLocationMapRequest) (*pb.ListDepartmentsLocationMapResponse, error) {
	return a.delegate.ListDepartmentsLocationMap(ctx, req)
}

type RecruitmentAdminAdapter struct{ delegate RecruitmentAdminAPI }

func NewRecruitmentAdminAdapter(delegate RecruitmentAdminAPI) (*RecruitmentAdminAdapter, error) {
	if delegate == nil {
		return nil, fmt.Errorf("recruitment admin delegate is required")
	}
	return &RecruitmentAdminAdapter{delegate: delegate}, nil
}

func (a *RecruitmentAdminAdapter) CreateInviteCode(ctx context.Context, req *pb.CreateInviteCodeRequest) (*pb.CreateInviteCodeResponse, error) {
	return a.delegate.CreateInviteCode(ctx, req)
}
func (a *RecruitmentAdminAdapter) ListInviteCodes(ctx context.Context, req *pb.ListInviteCodesRequest) (*pb.ListInviteCodesResponse, error) {
	return a.delegate.ListInviteCodes(ctx, req)
}
func (a *RecruitmentAdminAdapter) ExtendInviteCode(ctx context.Context, req *pb.ExtendInviteCodeRequest) (*pb.CommonResponse, error) {
	return a.delegate.ExtendInviteCode(ctx, req)
}
func (a *RecruitmentAdminAdapter) RevokeInviteCode(ctx context.Context, req *pb.RevokeInviteCodeRequest) (*pb.CommonResponse, error) {
	return a.delegate.RevokeInviteCode(ctx, req)
}
func (a *RecruitmentAdminAdapter) ReactivateInviteCode(ctx context.Context, req *pb.ReactivateInviteCodeRequest) (*pb.CommonResponse, error) {
	return a.delegate.ReactivateInviteCode(ctx, req)
}
func (a *RecruitmentAdminAdapter) ValidateInviteCode(ctx context.Context, req *pb.ValidateInviteCodeRequest) (*pb.ValidateInviteCodeResponse, error) {
	return a.delegate.ValidateInviteCode(ctx, req)
}
func (a *RecruitmentAdminAdapter) QueryUsageLogs(ctx context.Context, req *pb.QueryUsageLogsRequest) (*pb.QueryUsageLogsResponse, error) {
	return a.delegate.QueryUsageLogs(ctx, req)
}

type UsageStatsAdapter struct{ delegate UsageStatsAPI }

func NewUsageStatsAdapter(delegate UsageStatsAPI) (*UsageStatsAdapter, error) {
	if delegate == nil {
		return nil, fmt.Errorf("usage stats delegate is required")
	}
	return &UsageStatsAdapter{delegate: delegate}, nil
}

func (a *UsageStatsAdapter) GetUsageStats(ctx context.Context, req *pb.GetUsageStatsRequest) (*pb.GetUsageStatsResponse, error) {
	return a.delegate.GetUsageStats(ctx, req)
}
func (a *UsageStatsAdapter) GetUsageTrend(ctx context.Context, req *pb.GetUsageTrendRequest) (*pb.GetUsageTrendResponse, error) {
	return a.delegate.GetUsageTrend(ctx, req)
}

type CandidateAdapter struct{ delegate CandidateAPI }

func NewCandidateAdapter(delegate CandidateAPI) (*CandidateAdapter, error) {
	if delegate == nil {
		return nil, fmt.Errorf("candidate delegate is required")
	}
	return &CandidateAdapter{delegate: delegate}, nil
}

func (a *CandidateAdapter) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	return a.delegate.GetProfile(ctx, req)
}
func (a *CandidateAdapter) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error) {
	return a.delegate.UpdateProfile(ctx, req)
}
func (a *CandidateAdapter) FillProfileFromResume(ctx context.Context, req *pb.FillProfileFromResumeRequest) (*pb.FillProfileFromResumeResponse, error) {
	return a.delegate.FillProfileFromResume(ctx, req)
}
func (a *CandidateAdapter) ApplyProfileFill(ctx context.Context, req *pb.ApplyProfileFillRequest) (*pb.GetProfileResponse, error) {
	return a.delegate.ApplyProfileFill(ctx, req)
}
func (a *CandidateAdapter) GetResume(ctx context.Context, req *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	return a.delegate.GetResume(ctx, req)
}
func (a *CandidateAdapter) PresignResumeUpload(ctx context.Context, req *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error) {
	return a.delegate.PresignResumeUpload(ctx, req)
}
func (a *CandidateAdapter) ConfirmResumeUpload(ctx context.Context, req *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error) {
	return a.delegate.ConfirmResumeUpload(ctx, req)
}

type ApplicationAdapter struct{ delegate ApplicationAPI }

func NewApplicationAdapter(delegate ApplicationAPI) (*ApplicationAdapter, error) {
	if delegate == nil {
		return nil, fmt.Errorf("application delegate is required")
	}
	return &ApplicationAdapter{delegate: delegate}, nil
}

func (a *ApplicationAdapter) ApplyJob(ctx context.Context, req *pb.ApplyJobRequest) (*pb.CommonResponse, error) {
	return a.delegate.ApplyJob(ctx, req)
}
func (a *ApplicationAdapter) ListMyApplications(ctx context.Context, req *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	return a.delegate.ListMyApplications(ctx, req)
}
func (a *ApplicationAdapter) ListJobApplications(ctx context.Context, req *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	return a.delegate.ListJobApplications(ctx, req)
}
func (a *ApplicationAdapter) UpdateApplicationStatus(ctx context.Context, req *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	return a.delegate.UpdateApplicationStatus(ctx, req)
}
func (a *ApplicationAdapter) ListApplicationStatusTransitions(ctx context.Context, req *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error) {
	return a.delegate.ListApplicationStatusTransitions(ctx, req)
}

func NewCollaborationAdapter(delegate pb.CollaborationServiceServer) (pb.CollaborationServiceServer, error) {
	if delegate == nil {
		return nil, fmt.Errorf("collaboration delegate is required")
	}
	return delegate, nil
}
