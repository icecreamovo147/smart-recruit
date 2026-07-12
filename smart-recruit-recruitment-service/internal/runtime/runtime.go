package runtime

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"logic-grpc-service/recruitment/pb"
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

type CandidateAPI interface {
	GetProfile(context.Context, *pb.GetProfileRequest) (*pb.GetProfileResponse, error)
	UpdateProfile(context.Context, *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error)
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

type Deps struct {
	Job         JobAPI
	JobTaxonomy JobTaxonomyAPI
	Candidate   CandidateAPI
	Application ApplicationAPI
}

type Runtime struct {
	Job         pb.JobServiceServer
	Candidate   pb.CandidateServiceServer
	Application pb.ApplicationServiceServer
}

func New(deps Deps) (*Runtime, error) {
	if deps.Job == nil {
		return nil, fmt.Errorf("recruitment job api is required")
	}
	if deps.JobTaxonomy == nil {
		return nil, fmt.Errorf("recruitment job taxonomy api is required")
	}
	if deps.Candidate == nil {
		return nil, fmt.Errorf("recruitment candidate api is required")
	}
	if deps.Application == nil {
		return nil, fmt.Errorf("recruitment application api is required")
	}
	return &Runtime{
		Job:         jobServer{api: deps.Job, taxonomy: deps.JobTaxonomy},
		Candidate:   candidateServer{api: deps.Candidate},
		Application: applicationServer{api: deps.Application},
	}, nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if r == nil || r.Job == nil || r.Candidate == nil || r.Application == nil {
		return fmt.Errorf("recruitment runtime is not initialized")
	}
	pb.RegisterJobServiceServer(registrar, r.Job)
	pb.RegisterCandidateServiceServer(registrar, r.Candidate)
	pb.RegisterApplicationServiceServer(registrar, r.Application)
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
