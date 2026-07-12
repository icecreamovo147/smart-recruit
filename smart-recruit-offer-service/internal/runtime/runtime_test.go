package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestRuntimeRegistersOfferGRPCService(t *testing.T) {
	runtime, err := New(Deps{Offer: fakeOfferAPI{}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)
	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC returned error: %v", err)
	}
	services := server.GetServiceInfo()
	if _, ok := services[pb.OfferService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("missing registered service %s", pb.OfferService_ServiceDesc.ServiceName)
	}
}

func TestRuntimeRequiresOfferDependency(t *testing.T) {
	if _, err := New(Deps{}); err == nil {
		t.Fatal("expected missing offer dependency error")
	}
}

type fakeOfferAPI struct{}

func (fakeOfferAPI) CreateOffer(context.Context, *pb.CreateOfferRequest) (*pb.CreateOfferResponse, error) {
	return &pb.CreateOfferResponse{Code: errs.OK}, nil
}

func (fakeOfferAPI) UpdateOffer(context.Context, *pb.UpdateOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeOfferAPI) GetOffer(context.Context, *pb.GetOfferRequest) (*pb.GetOfferResponse, error) {
	return &pb.GetOfferResponse{Code: errs.OK}, nil
}

func (fakeOfferAPI) ListOffersByApplication(context.Context, *pb.ListOffersByApplicationRequest) (*pb.ListOffersByApplicationResponse, error) {
	return &pb.ListOffersByApplicationResponse{Code: errs.OK}, nil
}

func (fakeOfferAPI) SendOffer(context.Context, *pb.SendOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeOfferAPI) WithdrawOffer(context.Context, *pb.WithdrawOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeOfferAPI) AcceptOffer(context.Context, *pb.AcceptOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeOfferAPI) RejectOffer(context.Context, *pb.RejectOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeOfferAPI) ListMyOffers(context.Context, *pb.ListMyOffersRequest) (*pb.ListMyOffersResponse, error) {
	return &pb.ListMyOffersResponse{Code: errs.OK}, nil
}

func (fakeOfferAPI) ListOfferEvents(context.Context, *pb.ListOfferEventsRequest) (*pb.ListOfferEventsResponse, error) {
	return &pb.ListOfferEventsResponse{Code: errs.OK}, nil
}
