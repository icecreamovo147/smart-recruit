package runtime

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"logic-grpc-service/recruitment/pb"
)

const ServiceName = "analytics-service"

type ReportingAPI interface {
	GetDashboardReport(context.Context, *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error)
	GetFunnelReport(context.Context, *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error)
	GetTimeInStageReport(context.Context, *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error)
	GetInterviewOfferMetrics(context.Context, *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error)
}

type ProjectionReadModel struct {
	Source              string
	ProjectionMode      string
	TransactionalWrites bool
	ReportingAPIs       []string
}

func DefaultProjectionReadModel() ProjectionReadModel {
	return ProjectionReadModel{
		Source:         "analytics-owned projection/read-model",
		ProjectionMode: "domain-event-projection-read-models",
		ReportingAPIs: []string{
			"AdminService.GetDashboardReport",
			"AdminService.GetFunnelReport",
			"AdminService.GetTimeInStageReport",
			"AdminService.GetInterviewOfferMetrics",
		},
	}
}

func (model ProjectionReadModel) Validate() error {
	if model.Source == "" {
		return fmt.Errorf("analytics read model source is required")
	}
	if model.ProjectionMode != "domain-event-projection-read-models" {
		return fmt.Errorf("analytics projection mode = %q, want domain-event-projection-read-models", model.ProjectionMode)
	}
	if model.TransactionalWrites {
		return fmt.Errorf("analytics runtime must not write transactional domain state")
	}
	if len(model.ReportingAPIs) == 0 {
		return fmt.Errorf("analytics reporting APIs are required")
	}
	return nil
}

type Deps struct {
	Reporting  ReportingAPI
	Projection ProjectionReadModel
}

type Runtime struct {
	Reporting  pb.AdminServiceServer
	Projection ProjectionReadModel
}

func New(deps Deps) (*Runtime, error) {
	if deps.Reporting == nil {
		return nil, fmt.Errorf("analytics reporting api is required")
	}
	projection := deps.Projection
	if projection.Source == "" && projection.ProjectionMode == "" && len(projection.ReportingAPIs) == 0 {
		projection = DefaultProjectionReadModel()
	}
	if err := projection.Validate(); err != nil {
		return nil, err
	}
	return &Runtime{
		Reporting:  reportingServer{api: deps.Reporting},
		Projection: projection,
	}, nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if r == nil || r.Reporting == nil {
		return fmt.Errorf("analytics runtime is not initialized")
	}
	if err := r.Projection.Validate(); err != nil {
		return err
	}
	pb.RegisterAdminServiceServer(registrar, r.Reporting)
	return nil
}

type reportingServer struct {
	pb.UnimplementedAdminServiceServer
	api ReportingAPI
}

func (s reportingServer) GetDashboardReport(ctx context.Context, req *pb.GetDashboardReportRequest) (*pb.GetDashboardReportResponse, error) {
	return s.api.GetDashboardReport(ctx, req)
}

func (s reportingServer) GetFunnelReport(ctx context.Context, req *pb.GetFunnelReportRequest) (*pb.GetFunnelReportResponse, error) {
	return s.api.GetFunnelReport(ctx, req)
}

func (s reportingServer) GetTimeInStageReport(ctx context.Context, req *pb.GetTimeInStageReportRequest) (*pb.GetTimeInStageReportResponse, error) {
	return s.api.GetTimeInStageReport(ctx, req)
}

func (s reportingServer) GetInterviewOfferMetrics(ctx context.Context, req *pb.GetInterviewOfferMetricsRequest) (*pb.GetInterviewOfferMetricsResponse, error) {
	return s.api.GetInterviewOfferMetrics(ctx, req)
}
