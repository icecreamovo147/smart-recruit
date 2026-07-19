package runtime

import (
	"errors"

	"google.golang.org/grpc"
	"smart-recruit-proto/recruitment/pb"
)

const ServiceName = "billing-service"

type Runtime struct{ Billing pb.BillingServiceServer }

func New(server pb.BillingServiceServer) (*Runtime, error) {
	if server == nil {
		return nil, errors.New("billing gRPC server is required")
	}
	return &Runtime{Billing: server}, nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return errors.New("gRPC registrar is required")
	}
	if r == nil || r.Billing == nil {
		return errors.New("billing runtime is not initialized")
	}
	pb.RegisterBillingServiceServer(registrar, r.Billing)
	return nil
}
