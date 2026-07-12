package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestRuntimeRegistersRecruitmentGRPCServices(t *testing.T) {
	runtime, err := New(Deps{
		Job:         fakeJobAPI{},
		JobTaxonomy: fakeJobTaxonomyAPI{},
		Candidate:   fakeCandidateAPI{},
		Application: fakeApplicationAPI{},
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
		pb.CandidateService_ServiceDesc.ServiceName,
		pb.ApplicationService_ServiceDesc.ServiceName,
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
