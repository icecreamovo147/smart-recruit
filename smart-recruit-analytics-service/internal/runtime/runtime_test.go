package runtime

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestRuntimeRegistersReportingAdminService(t *testing.T) {
	runtime, err := New(Deps{Reporting: fakeReportingAPI{}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)
	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC returned error: %v", err)
	}
	if _, ok := server.GetServiceInfo()[pb.AdminService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services missing %s", pb.AdminService_ServiceDesc.ServiceName)
	}
	if runtime.Projection.TransactionalWrites {
		t.Fatal("analytics runtime must not write transactional domain state")
	}
}

func TestRuntimeRequiresReportingAPI(t *testing.T) {
	if _, err := New(Deps{}); err == nil {
		t.Fatal("New accepted missing reporting api")
	}
}

func TestRuntimeRejectsTransactionalDomainWrites(t *testing.T) {
	projection := DefaultProjectionReadModel()
	projection.TransactionalWrites = true
	if _, err := New(Deps{Reporting: fakeReportingAPI{}, Projection: projection}); err == nil {
		t.Fatal("New accepted transactional writes")
	}
}

func TestDefaultProjectionReadModelListsReportingAPIsOnly(t *testing.T) {
	model := DefaultProjectionReadModel()
	if err := model.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	apis := strings.Join(model.ReportingAPIs, "\n")
	for _, required := range []string{
		"AdminService.GetDashboardReport",
		"AdminService.GetFunnelReport",
		"AdminService.GetTimeInStageReport",
		"AdminService.GetInterviewOfferMetrics",
	} {
		if !strings.Contains(apis, required) {
			t.Fatalf("reporting APIs missing %q: %s", required, apis)
		}
	}
	if strings.Contains(apis, "QueryAuthAuditLogs") {
		t.Fatalf("analytics reporting runtime must not claim Identity-owned QueryAuthAuditLogs: %s", apis)
	}
}

type fakeReportingAPI struct{}

func (fakeReportingAPI) GetDashboardReport(context.Context, *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error) {
	return &pb.GetDashboardReportResponse{Code: errs.OK}, nil
}

func (fakeReportingAPI) GetFunnelReport(context.Context, *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error) {
	return &pb.GetFunnelReportResponse{Code: errs.OK}, nil
}

func (fakeReportingAPI) GetTimeInStageReport(context.Context, *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error) {
	return &pb.GetTimeInStageReportResponse{Code: errs.OK}, nil
}

func (fakeReportingAPI) GetInterviewOfferMetrics(context.Context, *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error) {
	return &pb.GetInterviewOfferMetricsResponse{Code: errs.OK}, nil
}
