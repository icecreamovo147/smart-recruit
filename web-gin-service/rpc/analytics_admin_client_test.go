package rpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"web-gin-service/recruitment/pb"
)

func TestAnalyticsAdminClientRoutesReportingOnly(t *testing.T) {
	logic := &recordingAdminClient{}
	analytics := &recordingAdminClient{}
	client := newAnalyticsAdminClient(logic, analytics)

	if _, err := client.GetDashboardReport(context.Background(), &pb.GetDashboardReportRequest{}); err != nil {
		t.Fatalf("GetDashboardReport returned error: %v", err)
	}
	if _, err := client.GetFunnelReport(context.Background(), &pb.GetFunnelReportRequest{}); err != nil {
		t.Fatalf("GetFunnelReport returned error: %v", err)
	}
	if _, err := client.GetTimeInStageReport(context.Background(), &pb.GetTimeInStageReportRequest{}); err != nil {
		t.Fatalf("GetTimeInStageReport returned error: %v", err)
	}
	if _, err := client.GetInterviewOfferMetrics(context.Background(), &pb.GetInterviewOfferMetricsRequest{}); err != nil {
		t.Fatalf("GetInterviewOfferMetrics returned error: %v", err)
	}
	if _, err := client.QueryAuthAuditLogs(context.Background(), &pb.QueryAuthAuditLogsRequest{}); err != nil {
		t.Fatalf("QueryAuthAuditLogs returned error: %v", err)
	}

	for _, method := range []string{"GetDashboardReport", "GetFunnelReport", "GetTimeInStageReport", "GetInterviewOfferMetrics"} {
		if !analytics.called(method) {
			t.Fatalf("analytics client did not receive %s: %v", method, analytics.calls)
		}
	}
	if analytics.called("QueryAuthAuditLogs") {
		t.Fatalf("analytics client must not receive QueryAuthAuditLogs: %v", analytics.calls)
	}
	if !logic.called("QueryAuthAuditLogs") {
		t.Fatalf("base client should receive QueryAuthAuditLogs: %v", logic.calls)
	}
}

func (c *recordingAdminClient) GetDashboardReport(context.Context, *pb.GetDashboardReportRequest, ...grpc.CallOption) (*pb.GetDashboardReportResponse, error) {
	c.calls = append(c.calls, "GetDashboardReport")
	return &pb.GetDashboardReportResponse{}, nil
}

func (c *recordingAdminClient) GetFunnelReport(context.Context, *pb.GetFunnelReportRequest, ...grpc.CallOption) (*pb.GetFunnelReportResponse, error) {
	c.calls = append(c.calls, "GetFunnelReport")
	return &pb.GetFunnelReportResponse{}, nil
}

func (c *recordingAdminClient) GetTimeInStageReport(context.Context, *pb.GetTimeInStageReportRequest, ...grpc.CallOption) (*pb.GetTimeInStageReportResponse, error) {
	c.calls = append(c.calls, "GetTimeInStageReport")
	return &pb.GetTimeInStageReportResponse{}, nil
}

func (c *recordingAdminClient) GetInterviewOfferMetrics(context.Context, *pb.GetInterviewOfferMetricsRequest, ...grpc.CallOption) (*pb.GetInterviewOfferMetricsResponse, error) {
	c.calls = append(c.calls, "GetInterviewOfferMetrics")
	return &pb.GetInterviewOfferMetricsResponse{}, nil
}
