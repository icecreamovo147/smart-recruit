package interfaces

import (
	"context"
	"testing"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/service"
)

func TestCurrentAnalyticsServiceSatisfiesReportingAPI(t *testing.T) {
	t.Parallel()

	var _ ReportingAPI = (*service.AnalyticsService)(nil)
}

func TestReportingAdminServerForwardsAnalyticsReports(t *testing.T) {
	t.Parallel()

	api := &recordingReportingAPI{}
	server, err := NewReportingAdminServer(api)
	if err != nil {
		t.Fatalf("NewReportingAdminServer() error = %v", err)
	}

	if _, err := server.GetDashboardReport(context.Background(), &pb.GetDashboardReportRequest{}); err != nil {
		t.Fatalf("GetDashboardReport() error = %v", err)
	}
	if _, err := server.GetInterviewOfferMetrics(context.Background(), &pb.GetInterviewOfferMetricsRequest{}); err != nil {
		t.Fatalf("GetInterviewOfferMetrics() error = %v", err)
	}
	if api.dashboardCalls != 1 {
		t.Fatalf("dashboardCalls = %d, want 1", api.dashboardCalls)
	}
	if api.metricsCalls != 1 {
		t.Fatalf("metricsCalls = %d, want 1", api.metricsCalls)
	}
}

func TestReportingAdminServerRequiresDependency(t *testing.T) {
	t.Parallel()

	if _, err := NewReportingAdminServer(nil); err == nil {
		t.Fatal("NewReportingAdminServer(nil) error = nil")
	}
}

type recordingReportingAPI struct {
	dashboardCalls int
	metricsCalls   int
}

func (a *recordingReportingAPI) GetDashboardReport(context.Context, *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error) {
	a.dashboardCalls++
	return &pb.GetDashboardReportResponse{Code: errs.OK}, nil
}

func (a *recordingReportingAPI) GetFunnelReport(context.Context, *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error) {
	return &pb.GetFunnelReportResponse{Code: errs.OK}, nil
}

func (a *recordingReportingAPI) GetTimeInStageReport(context.Context, *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error) {
	return &pb.GetTimeInStageReportResponse{Code: errs.OK}, nil
}

func (a *recordingReportingAPI) GetInterviewOfferMetrics(context.Context, *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error) {
	a.metricsCalls++
	return &pb.GetInterviewOfferMetricsResponse{Code: errs.OK}, nil
}
