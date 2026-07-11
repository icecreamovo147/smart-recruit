package interfaces

import (
	"context"

	"logic-grpc-service/recruitment/pb"
)

// OfferAPI is the Offer-owned gRPC contract implemented by the current OfferService.
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
