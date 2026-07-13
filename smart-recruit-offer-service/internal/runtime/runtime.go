package runtime

import (
	"fmt"

	"google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

const ServiceName = "offer-service"

type Deps struct {
	Offer pb.OfferServiceServer
}

type Runtime struct {
	Offer pb.OfferServiceServer
}

func New(deps Deps) (*Runtime, error) {
	if deps.Offer == nil {
		return nil, fmt.Errorf("offer grpc server is required")
	}
	return &Runtime{
		Offer: deps.Offer,
	}, nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if r == nil || r.Offer == nil {
		return fmt.Errorf("offer runtime is not initialized")
	}
	pb.RegisterOfferServiceServer(registrar, r.Offer)
	return nil
}
