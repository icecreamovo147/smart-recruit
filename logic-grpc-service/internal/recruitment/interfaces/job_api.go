package interfaces

import (
	"context"

	"logic-grpc-service/recruitment/pb"
)

// JobAPI is the Recruitment-owned gRPC job contract implemented by the current JobService.
type JobAPI interface {
	CreateJob(context.Context, *pb.CreateJobRequest) (*pb.CreateJobResponse, error)
	UpdateJob(context.Context, *pb.UpdateJobRequest) (*pb.CommonResponse, error)
	OfflineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error)
	OnlineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error)
	ListHRJobs(context.Context, *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error)
	ListPublicJobs(context.Context, *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error)
	GetJobDetail(context.Context, *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error)
}

// JobTaxonomyAPI is the Recruitment-owned job taxonomy read contract required by the generated JobService.
type JobTaxonomyAPI interface {
	ListJobOptions(context.Context, *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error)
	ListDepartmentLocations(context.Context, *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error)
}
