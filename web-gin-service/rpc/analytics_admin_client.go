package rpc

import (
	"context"

	"google.golang.org/grpc"

	"web-gin-service/recruitment/pb"
)

type analyticsAdminClient struct {
	pb.AdminServiceClient
	analytics pb.AdminServiceClient
}

func newAnalyticsAdminClient(base, analytics pb.AdminServiceClient) pb.AdminServiceClient {
	if analytics == nil {
		return base
	}
	return &analyticsAdminClient{
		AdminServiceClient: base,
		analytics:          analytics,
	}
}

func (c *analyticsAdminClient) GetDashboardReport(ctx context.Context, in *pb.GetDashboardReportRequest, opts ...grpc.CallOption) (*pb.GetDashboardReportResponse, error) {
	return c.analytics.GetDashboardReport(ctx, in, opts...)
}

func (c *analyticsAdminClient) GetFunnelReport(ctx context.Context, in *pb.GetFunnelReportRequest, opts ...grpc.CallOption) (*pb.GetFunnelReportResponse, error) {
	return c.analytics.GetFunnelReport(ctx, in, opts...)
}

func (c *analyticsAdminClient) GetTimeInStageReport(ctx context.Context, in *pb.GetTimeInStageReportRequest, opts ...grpc.CallOption) (*pb.GetTimeInStageReportResponse, error) {
	return c.analytics.GetTimeInStageReport(ctx, in, opts...)
}

func (c *analyticsAdminClient) GetInterviewOfferMetrics(ctx context.Context, in *pb.GetInterviewOfferMetricsRequest, opts ...grpc.CallOption) (*pb.GetInterviewOfferMetricsResponse, error) {
	return c.analytics.GetInterviewOfferMetrics(ctx, in, opts...)
}
