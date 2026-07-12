package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestRuntimeRegistersRecruitmentGRPCServices(t *testing.T) {
	t.Parallel()

	runtime, err := New(Deps{
		Job:         &runtimeJobAPI{},
		JobTaxonomy: &runtimeJobTaxonomyAPI{},
		Candidate:   &runtimeCandidateAPI{},
		Application: &runtimeApplicationAPI{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)

	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC() error = %v", err)
	}

	services := server.GetServiceInfo()
	for _, serviceName := range []string{
		pb.JobService_ServiceDesc.ServiceName,
		pb.CandidateService_ServiceDesc.ServiceName,
		pb.ApplicationService_ServiceDesc.ServiceName,
	} {
		if _, ok := services[serviceName]; !ok {
			t.Fatalf("registered services missing %s: %v", serviceName, services)
		}
	}
}

func TestRuntimeRequiresRecruitmentDependencies(t *testing.T) {
	t.Parallel()

	if _, err := New(Deps{}); err == nil {
		t.Fatal("New(Deps{}) error = nil")
	}
}

func TestRuntimeRejectsNilRegistrar(t *testing.T) {
	t.Parallel()

	runtime, err := New(Deps{
		Job:         &runtimeJobAPI{},
		JobTaxonomy: &runtimeJobTaxonomyAPI{},
		Candidate:   &runtimeCandidateAPI{},
		Application: &runtimeApplicationAPI{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := runtime.RegisterGRPC(nil); err == nil {
		t.Fatal("RegisterGRPC(nil) error = nil")
	}
}

type runtimeJobAPI struct{}

func (runtimeJobAPI) CreateJob(context.Context, *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	return &pb.CreateJobResponse{Code: errs.OK}, nil
}

func (runtimeJobAPI) UpdateJob(context.Context, *pb.UpdateJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeJobAPI) OfflineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeJobAPI) OnlineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeJobAPI) ListHRJobs(context.Context, *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}

func (runtimeJobAPI) ListPublicJobs(context.Context, *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error) {
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}

func (runtimeJobAPI) GetJobDetail(context.Context, *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	return &pb.GetJobDetailResponse{Code: errs.OK}, nil
}

type runtimeJobTaxonomyAPI struct{}

func (runtimeJobTaxonomyAPI) ListJobOptions(context.Context, *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error) {
	return &pb.ListJobOptionsResponse{Code: errs.OK}, nil
}

func (runtimeJobTaxonomyAPI) ListDepartmentLocations(context.Context, *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error) {
	return &pb.ListDepartmentLocationsResponse{Code: errs.OK}, nil
}

type runtimeCandidateAPI struct{}

func (runtimeCandidateAPI) GetProfile(context.Context, *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	return &pb.GetProfileResponse{Code: errs.OK}, nil
}

func (runtimeCandidateAPI) UpdateProfile(context.Context, *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error) {
	return &pb.GetProfileResponse{Code: errs.OK}, nil
}

func (runtimeCandidateAPI) GetResume(context.Context, *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	return &pb.GetResumeResponse{Code: errs.OK}, nil
}

func (runtimeCandidateAPI) PresignResumeUpload(context.Context, *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error) {
	return &pb.PresignResumeUploadResponse{Code: errs.OK}, nil
}

func (runtimeCandidateAPI) ConfirmResumeUpload(context.Context, *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error) {
	return &pb.ConfirmResumeUploadResponse{Code: errs.OK}, nil
}

type runtimeApplicationAPI struct{}

func (runtimeApplicationAPI) ApplyJob(context.Context, *pb.ApplyJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeApplicationAPI) ListMyApplications(context.Context, *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	return &pb.ListMyApplicationsResponse{Code: errs.OK}, nil
}

func (runtimeApplicationAPI) ListJobApplications(context.Context, *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	return &pb.ListJobApplicationsResponse{Code: errs.OK}, nil
}

func (runtimeApplicationAPI) UpdateApplicationStatus(context.Context, *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeApplicationAPI) ListApplicationStatusTransitions(context.Context, *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error) {
	return &pb.ListApplicationStatusTransitionsResponse{Code: errs.OK}, nil
}
