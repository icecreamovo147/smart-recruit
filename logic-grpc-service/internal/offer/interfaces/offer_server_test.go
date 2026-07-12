package interfaces

import (
	"context"
	"testing"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestOfferServerForwardsOfferOwnedMethods(t *testing.T) {
	t.Parallel()

	api := &recordingOfferAPI{}
	server, err := NewOfferServer(api)
	if err != nil {
		t.Fatalf("NewOfferServer() error = %v", err)
	}

	if _, err := server.CreateOffer(context.Background(), &pb.CreateOfferRequest{}); err != nil {
		t.Fatalf("CreateOffer() error = %v", err)
	}
	if _, err := server.AcceptOffer(context.Background(), &pb.AcceptOfferRequest{}); err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}
	if _, err := server.ListOfferEvents(context.Background(), &pb.ListOfferEventsRequest{}); err != nil {
		t.Fatalf("ListOfferEvents() error = %v", err)
	}
	if api.createCalls != 1 {
		t.Fatalf("createCalls = %d, want 1", api.createCalls)
	}
	if api.acceptCalls != 1 {
		t.Fatalf("acceptCalls = %d, want 1", api.acceptCalls)
	}
	if api.listEventsCalls != 1 {
		t.Fatalf("listEventsCalls = %d, want 1", api.listEventsCalls)
	}
}

func TestOfferServerRequiresDependency(t *testing.T) {
	t.Parallel()

	if _, err := NewOfferServer(nil); err == nil {
		t.Fatal("NewOfferServer(nil) error = nil")
	}
}

type recordingOfferAPI struct {
	createCalls     int
	acceptCalls     int
	listEventsCalls int
}

func (a *recordingOfferAPI) CreateOffer(context.Context, *pb.CreateOfferRequest) (*pb.CreateOfferResponse, error) {
	a.createCalls++
	return &pb.CreateOfferResponse{Code: errs.OK}, nil
}

func (a *recordingOfferAPI) UpdateOffer(context.Context, *pb.UpdateOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingOfferAPI) GetOffer(context.Context, *pb.GetOfferRequest) (*pb.GetOfferResponse, error) {
	return &pb.GetOfferResponse{Code: errs.OK}, nil
}

func (a *recordingOfferAPI) ListOffersByApplication(context.Context, *pb.ListOffersByApplicationRequest) (*pb.ListOffersByApplicationResponse, error) {
	return &pb.ListOffersByApplicationResponse{Code: errs.OK}, nil
}

func (a *recordingOfferAPI) SendOffer(context.Context, *pb.SendOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingOfferAPI) WithdrawOffer(context.Context, *pb.WithdrawOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingOfferAPI) AcceptOffer(context.Context, *pb.AcceptOfferRequest) (*pb.CommonResponse, error) {
	a.acceptCalls++
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingOfferAPI) RejectOffer(context.Context, *pb.RejectOfferRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (a *recordingOfferAPI) ListMyOffers(context.Context, *pb.ListMyOffersRequest) (*pb.ListMyOffersResponse, error) {
	return &pb.ListMyOffersResponse{Code: errs.OK}, nil
}

func (a *recordingOfferAPI) ListOfferEvents(context.Context, *pb.ListOfferEventsRequest) (*pb.ListOfferEventsResponse, error) {
	a.listEventsCalls++
	return &pb.ListOfferEventsResponse{Code: errs.OK}, nil
}
