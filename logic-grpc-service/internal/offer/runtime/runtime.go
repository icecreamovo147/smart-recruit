package runtime

import (
	"fmt"

	"google.golang.org/grpc"

	offerinterfaces "logic-grpc-service/internal/offer/interfaces"
	"logic-grpc-service/recruitment/pb"
)

type Deps struct {
	Offer offerinterfaces.OfferAPI
}

type Runtime struct {
	Descriptor Descriptor
	Offer      pb.OfferServiceServer
}

func New(deps Deps) (*Runtime, error) {
	descriptor, err := NewDescriptor()
	if err != nil {
		return nil, err
	}
	offerServer, err := offerinterfaces.NewOfferServer(deps.Offer)
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{
		Descriptor: descriptor,
		Offer:      offerServer,
	}
	if err := runtime.Validate(); err != nil {
		return nil, err
	}
	return runtime, nil
}

func (r *Runtime) Validate() error {
	if r == nil {
		return fmt.Errorf("offer runtime is required")
	}
	if err := Validate(r.Descriptor); err != nil {
		return err
	}
	if r.Offer == nil {
		return fmt.Errorf("offer server is required")
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
	pb.RegisterOfferServiceServer(registrar, r.Offer)
	return nil
}
