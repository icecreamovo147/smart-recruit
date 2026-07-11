package interfaces

import (
	"context"
	"fmt"

	"logic-grpc-service/recruitment/pb"
)

type OfferServer struct {
	pb.UnimplementedOfferServiceServer
	api OfferAPI
}

func NewOfferServer(api OfferAPI) (*OfferServer, error) {
	if api == nil {
		return nil, fmt.Errorf("offer api is required")
	}
	return &OfferServer{api: api}, nil
}

func (s *OfferServer) CreateOffer(ctx context.Context, req *pb.CreateOfferRequest) (*pb.CreateOfferResponse, error) {
	return s.api.CreateOffer(ctx, req)
}

func (s *OfferServer) UpdateOffer(ctx context.Context, req *pb.UpdateOfferRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateOffer(ctx, req)
}

func (s *OfferServer) GetOffer(ctx context.Context, req *pb.GetOfferRequest) (*pb.GetOfferResponse, error) {
	return s.api.GetOffer(ctx, req)
}

func (s *OfferServer) ListOffersByApplication(ctx context.Context, req *pb.ListOffersByApplicationRequest) (*pb.ListOffersByApplicationResponse, error) {
	return s.api.ListOffersByApplication(ctx, req)
}

func (s *OfferServer) SendOffer(ctx context.Context, req *pb.SendOfferRequest) (*pb.CommonResponse, error) {
	return s.api.SendOffer(ctx, req)
}

func (s *OfferServer) WithdrawOffer(ctx context.Context, req *pb.WithdrawOfferRequest) (*pb.CommonResponse, error) {
	return s.api.WithdrawOffer(ctx, req)
}

func (s *OfferServer) AcceptOffer(ctx context.Context, req *pb.AcceptOfferRequest) (*pb.CommonResponse, error) {
	return s.api.AcceptOffer(ctx, req)
}

func (s *OfferServer) RejectOffer(ctx context.Context, req *pb.RejectOfferRequest) (*pb.CommonResponse, error) {
	return s.api.RejectOffer(ctx, req)
}

func (s *OfferServer) ListMyOffers(ctx context.Context, req *pb.ListMyOffersRequest) (*pb.ListMyOffersResponse, error) {
	return s.api.ListMyOffers(ctx, req)
}

func (s *OfferServer) ListOfferEvents(ctx context.Context, req *pb.ListOfferEventsRequest) (*pb.ListOfferEventsResponse, error) {
	return s.api.ListOfferEvents(ctx, req)
}
