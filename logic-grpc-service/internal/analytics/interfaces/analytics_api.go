package interfaces

import (
	"context"

	"logic-grpc-service/recruitment/pb"
)

// AnalyticsQueryAPI is the Analytics-owned reporting query gRPC contract.
type AnalyticsQueryAPI interface {
	GetDashboardReport(context.Context, *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error)
	GetFunnelReport(context.Context, *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error)
	GetTimeInStageReport(context.Context, *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error)
	GetInterviewOfferMetrics(context.Context, *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error)
	QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error)
}
