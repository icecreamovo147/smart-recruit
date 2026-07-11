package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestRuntimeRegistersAnalyticsReportingGRPCService(t *testing.T) {
	t.Parallel()

	runtime, err := New(Deps{Reporting: &runtimeReportingAPI{}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)

	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC() error = %v", err)
	}

	services := server.GetServiceInfo()
	if _, ok := services[pb.AdminService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services missing %s: %v", pb.AdminService_ServiceDesc.ServiceName, services)
	}
}

func TestRuntimeRequiresReportingDependency(t *testing.T) {
	t.Parallel()

	if _, err := New(Deps{}); err == nil {
		t.Fatal("New(Deps{}) error = nil")
	}
}

func TestRuntimeRejectsNilRegistrar(t *testing.T) {
	t.Parallel()

	runtime, err := New(Deps{Reporting: &runtimeReportingAPI{}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := runtime.RegisterGRPC(nil); err == nil {
		t.Fatal("RegisterGRPC(nil) error = nil")
	}
}

type runtimeReportingAPI struct{}

func (runtimeReportingAPI) GetDashboardReport(context.Context, *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error) {
	return &pb.GetDashboardReportResponse{Code: errs.OK}, nil
}

func (runtimeReportingAPI) GetFunnelReport(context.Context, *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error) {
	return &pb.GetFunnelReportResponse{Code: errs.OK}, nil
}

func (runtimeReportingAPI) GetTimeInStageReport(context.Context, *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error) {
	return &pb.GetTimeInStageReportResponse{Code: errs.OK}, nil
}

func (runtimeReportingAPI) GetInterviewOfferMetrics(context.Context, *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error) {
	return &pb.GetInterviewOfferMetricsResponse{Code: errs.OK}, nil
}
