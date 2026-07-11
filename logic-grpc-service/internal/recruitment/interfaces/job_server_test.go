package interfaces

import (
	"context"
	"testing"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestJobServerForwardsRecruitmentOwnedMethods(t *testing.T) {
	t.Parallel()

	api := &recordingJobAPI{}
	taxonomy := &recordingJobTaxonomyAPI{}
	server, err := NewJobServer(api, taxonomy)
	if err != nil {
		t.Fatalf("NewJobServer() error = %v", err)
	}

	if _, err := server.CreateJob(context.Background(), &pb.CreateJobRequest{}); err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if _, err := server.ListJobOptions(context.Background(), &pb.ListJobOptionsRequest{}); err != nil {
		t.Fatalf("ListJobOptions() error = %v", err)
	}
	if api.createJobCalls != 1 {
		t.Fatalf("createJobCalls = %d, want 1", api.createJobCalls)
	}
	if taxonomy.listJobOptionsCalls != 1 {
		t.Fatalf("listJobOptionsCalls = %d, want 1", taxonomy.listJobOptionsCalls)
	}
}

func TestJobServerRequiresDependencies(t *testing.T) {
	t.Parallel()

	if _, err := NewJobServer(nil, &recordingJobTaxonomyAPI{}); err == nil {
		t.Fatal("NewJobServer(nil, taxonomy) error = nil")
	}
	if _, err := NewJobServer(&recordingJobAPI{}, nil); err == nil {
		t.Fatal("NewJobServer(api, nil) error = nil")
	}
}

type recordingJobAPI struct {
	createJobCalls int
}

func (a *recordingJobAPI) CreateJob(context.Context, *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	a.createJobCalls++
	return &pb.CreateJobResponse{Code: errs.OK}, nil
}

func (a *recordingJobAPI) UpdateJob(context.Context, *pb.UpdateJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingJobAPI) OfflineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingJobAPI) OnlineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingJobAPI) ListHRJobs(context.Context, *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}

func (a *recordingJobAPI) ListPublicJobs(context.Context, *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error) {
	return &pb.ListJobsResponse{Code: errs.OK}, nil
}

func (a *recordingJobAPI) GetJobDetail(context.Context, *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	return &pb.GetJobDetailResponse{Code: errs.OK}, nil
}

type recordingJobTaxonomyAPI struct {
	listJobOptionsCalls int
}

func (a *recordingJobTaxonomyAPI) ListJobOptions(context.Context, *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error) {
	a.listJobOptionsCalls++
	return &pb.ListJobOptionsResponse{Code: errs.OK}, nil
}

func (a *recordingJobTaxonomyAPI) ListDepartmentLocations(context.Context, *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error) {
	return &pb.ListDepartmentLocationsResponse{Code: errs.OK}, nil
}
