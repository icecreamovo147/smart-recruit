package interfaces

import (
	"context"
	"fmt"

	"logic-grpc-service/recruitment/pb"
)

type JobServer struct {
	pb.UnimplementedJobServiceServer
	api      JobAPI
	taxonomy JobTaxonomyAPI
}

func NewJobServer(api JobAPI, taxonomy JobTaxonomyAPI) (*JobServer, error) {
	if api == nil {
		return nil, fmt.Errorf("recruitment job api is required")
	}
	if taxonomy == nil {
		return nil, fmt.Errorf("recruitment job taxonomy api is required")
	}
	return &JobServer{api: api, taxonomy: taxonomy}, nil
}

func (s *JobServer) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	return s.api.CreateJob(ctx, req)
}

func (s *JobServer) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateJob(ctx, req)
}

func (s *JobServer) OfflineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return s.api.OfflineJob(ctx, req)
}

func (s *JobServer) OnlineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return s.api.OnlineJob(ctx, req)
}

func (s *JobServer) ListHRJobs(ctx context.Context, req *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	return s.api.ListHRJobs(ctx, req)
}

func (s *JobServer) ListPublicJobs(ctx context.Context, req *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error) {
	return s.api.ListPublicJobs(ctx, req)
}

func (s *JobServer) GetJobDetail(ctx context.Context, req *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	return s.api.GetJobDetail(ctx, req)
}

func (s *JobServer) ListJobOptions(ctx context.Context, req *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error) {
	return s.taxonomy.ListJobOptions(ctx, req)
}

func (s *JobServer) ListDepartmentLocations(ctx context.Context, req *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error) {
	return s.taxonomy.ListDepartmentLocations(ctx, req)
}
