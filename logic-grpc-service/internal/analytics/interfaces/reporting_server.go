package interfaces

import (
	"context"
	"fmt"

	"logic-grpc-service/recruitment/pb"
)

type ReportingAPI interface {
	GetDashboardReport(context.Context, *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error)
	GetFunnelReport(context.Context, *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error)
	GetTimeInStageReport(context.Context, *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error)
	GetInterviewOfferMetrics(context.Context, *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error)
}

type ReportingAdminServer struct {
	pb.UnimplementedAdminServiceServer
	api ReportingAPI
}

func NewReportingAdminServer(api ReportingAPI) (*ReportingAdminServer, error) {
	if api == nil {
		return nil, fmt.Errorf("analytics reporting api is required")
	}
	return &ReportingAdminServer{api: api}, nil
}

func (s *ReportingAdminServer) GetDashboardReport(ctx context.Context, req *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error) {
	return s.api.GetDashboardReport(ctx, req)
}

func (s *ReportingAdminServer) GetFunnelReport(ctx context.Context, req *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error) {
	return s.api.GetFunnelReport(ctx, req)
}

func (s *ReportingAdminServer) GetTimeInStageReport(ctx context.Context, req *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error) {
	return s.api.GetTimeInStageReport(ctx, req)
}

func (s *ReportingAdminServer) GetInterviewOfferMetrics(ctx context.Context, req *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error) {
	return s.api.GetInterviewOfferMetrics(ctx, req)
}
