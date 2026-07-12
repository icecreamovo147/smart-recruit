package runtime

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

const ServiceName = "offer-service"

type OfferAPI interface {
	CreateOffer(context.Context, *pb.CreateOfferRequest) (*pb.CreateOfferResponse, error)
	UpdateOffer(context.Context, *pb.UpdateOfferRequest) (*pb.CommonResponse, error)
	GetOffer(context.Context, *pb.GetOfferRequest) (*pb.GetOfferResponse, error)
	ListOffersByApplication(context.Context, *pb.ListOffersByApplicationRequest) (*pb.ListOffersByApplicationResponse, error)
	SendOffer(context.Context, *pb.SendOfferRequest) (*pb.CommonResponse, error)
	WithdrawOffer(context.Context, *pb.WithdrawOfferRequest) (*pb.CommonResponse, error)
	AcceptOffer(context.Context, *pb.AcceptOfferRequest) (*pb.CommonResponse, error)
	RejectOffer(context.Context, *pb.RejectOfferRequest) (*pb.CommonResponse, error)
	ListMyOffers(context.Context, *pb.ListMyOffersRequest) (*pb.ListMyOffersResponse, error)
	ListOfferEvents(context.Context, *pb.ListOfferEventsRequest) (*pb.ListOfferEventsResponse, error)
}

type Deps struct {
	Offer OfferAPI
}

type Runtime struct {
	Offer pb.OfferServiceServer
}

func New(deps Deps) (*Runtime, error) {
	if deps.Offer == nil {
		return nil, fmt.Errorf("offer api is required")
	}
	return &Runtime{
		Offer: offerServer{api: deps.Offer},
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

type offerServer struct {
	pb.UnimplementedOfferServiceServer
	api OfferAPI
}

func (s offerServer) CreateOffer(ctx context.Context, req *pb.CreateOfferRequest) (*pb.CreateOfferResponse, error) {
	return s.api.CreateOffer(ctx, req)
}

func (s offerServer) UpdateOffer(ctx context.Context, req *pb.UpdateOfferRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateOffer(ctx, req)
}

func (s offerServer) GetOffer(ctx context.Context, req *pb.GetOfferRequest) (*pb.GetOfferResponse, error) {
	return s.api.GetOffer(ctx, req)
}

func (s offerServer) ListOffersByApplication(ctx context.Context, req *pb.ListOffersByApplicationRequest) (*pb.ListOffersByApplicationResponse, error) {
	return s.api.ListOffersByApplication(ctx, req)
}

func (s offerServer) SendOffer(ctx context.Context, req *pb.SendOfferRequest) (*pb.CommonResponse, error) {
	return s.api.SendOffer(ctx, req)
}

func (s offerServer) WithdrawOffer(ctx context.Context, req *pb.WithdrawOfferRequest) (*pb.CommonResponse, error) {
	return s.api.WithdrawOffer(ctx, req)
}

func (s offerServer) AcceptOffer(ctx context.Context, req *pb.AcceptOfferRequest) (*pb.CommonResponse, error) {
	return s.api.AcceptOffer(ctx, req)
}

func (s offerServer) RejectOffer(ctx context.Context, req *pb.RejectOfferRequest) (*pb.CommonResponse, error) {
	return s.api.RejectOffer(ctx, req)
}

func (s offerServer) ListMyOffers(ctx context.Context, req *pb.ListMyOffersRequest) (*pb.ListMyOffersResponse, error) {
	return s.api.ListMyOffers(ctx, req)
}

func (s offerServer) ListOfferEvents(ctx context.Context, req *pb.ListOfferEventsRequest) (*pb.ListOfferEventsResponse, error) {
	return s.api.ListOfferEvents(ctx, req)
}
