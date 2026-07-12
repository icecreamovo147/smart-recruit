package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestRuntimeRegistersOfferGRPCService(t *testing.T) {
	t.Parallel()

	runtime, err := New(Deps{Offer: &runtimeOfferAPI{}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)

	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC() error = %v", err)
	}

	services := server.GetServiceInfo()
	if _, ok := services[pb.OfferService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services missing %s: %v", pb.OfferService_ServiceDesc.ServiceName, services)
	}
}

func TestRuntimeRequiresOfferDependency(t *testing.T) {
	t.Parallel()

	if _, err := New(Deps{}); err == nil {
		t.Fatal("New(Deps{}) error = nil")
	}
}

func TestRuntimeRejectsNilRegistrar(t *testing.T) {
	t.Parallel()

	runtime, err := New(Deps{Offer: &runtimeOfferAPI{}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := runtime.RegisterGRPC(nil); err == nil {
		t.Fatal("RegisterGRPC(nil) error = nil")
	}
}

type runtimeOfferAPI struct{}

func (runtimeOfferAPI) CreateOffer(context.Context, *pb.CreateOfferRequest) (*pb.CreateOfferResponse, error) {
	return &pb.CreateOfferResponse{Code: errs.OK}, nil
}

func (runtimeOfferAPI) UpdateOffer(context.Context, *pb.UpdateOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeOfferAPI) GetOffer(context.Context, *pb.GetOfferRequest) (*pb.GetOfferResponse, error) {
	return &pb.GetOfferResponse{Code: errs.OK}, nil
}

func (runtimeOfferAPI) ListOffersByApplication(context.Context, *pb.ListOffersByApplicationRequest) (*pb.ListOffersByApplicationResponse, error) {
	return &pb.ListOffersByApplicationResponse{Code: errs.OK}, nil
}

func (runtimeOfferAPI) SendOffer(context.Context, *pb.SendOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeOfferAPI) WithdrawOffer(context.Context, *pb.WithdrawOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeOfferAPI) AcceptOffer(context.Context, *pb.AcceptOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeOfferAPI) RejectOffer(context.Context, *pb.RejectOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeOfferAPI) ListMyOffers(context.Context, *pb.ListMyOffersRequest) (*pb.ListMyOffersResponse, error) {
	return &pb.ListMyOffersResponse{Code: errs.OK}, nil
}

func (runtimeOfferAPI) ListOfferEvents(context.Context, *pb.ListOfferEventsRequest) (*pb.ListOfferEventsResponse, error) {
	return &pb.ListOfferEventsResponse{Code: errs.OK}, nil
}
