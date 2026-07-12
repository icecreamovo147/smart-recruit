package runtime

import (
	"fmt"

	"google.golang.org/grpc"

	analyticsinterfaces "logic-grpc-service/internal/analytics/interfaces"
	"logic-grpc-service/recruitment/pb"
)

type Deps struct {
	Reporting analyticsinterfaces.ReportingAPI
}

type Runtime struct {
	Descriptor Descriptor
	Reporting  pb.AdminServiceServer
}

func New(deps Deps) (*Runtime, error) {
	descriptor, err := NewDescriptor()
	if err != nil {
		return nil, err
	}
	reportingServer, err := analyticsinterfaces.NewReportingAdminServer(deps.Reporting)
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{
		Descriptor: descriptor,
		Reporting:  reportingServer,
	}
	if err := runtime.Validate(); err != nil {
		return nil, err
	}
	return runtime, nil
}

func (r *Runtime) Validate() error {
	if r == nil {
		return fmt.Errorf("analytics runtime is required")
	}
	if err := Validate(r.Descriptor); err != nil {
		return err
	}
	if r.Reporting == nil {
		return fmt.Errorf("analytics reporting server is required")
	}
	return nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if err := r.Validate(); err != nil {
		return err
	}
	pb.RegisterAdminServiceServer(registrar, r.Reporting)
	return nil
}
