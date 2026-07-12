package interfaces

import (
	"context"
	"fmt"

	"logic-grpc-service/recruitment/pb"
)

type ApplicationServer struct {
	pb.UnimplementedApplicationServiceServer
	api ApplicationAPI
}

func NewApplicationServer(api ApplicationAPI) (*ApplicationServer, error) {
	if api == nil {
		return nil, fmt.Errorf("recruitment application api is required")
	}
	return &ApplicationServer{api: api}, nil
}

func (s *ApplicationServer) ApplyJob(ctx context.Context, req *pb.ApplyJobRequest) (*pb.CommonResponse, error) {
	return s.api.ApplyJob(ctx, req)
}

func (s *ApplicationServer) ListMyApplications(ctx context.Context, req *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	return s.api.ListMyApplications(ctx, req)
}

func (s *ApplicationServer) ListJobApplications(ctx context.Context, req *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	return s.api.ListJobApplications(ctx, req)
}

func (s *ApplicationServer) UpdateApplicationStatus(ctx context.Context, req *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateApplicationStatus(ctx, req)
}

func (s *ApplicationServer) ListApplicationStatusTransitions(ctx context.Context, req *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error) {
	return s.api.ListApplicationStatusTransitions(ctx, req)
}
