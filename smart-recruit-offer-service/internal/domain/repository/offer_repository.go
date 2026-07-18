package repository

import (
	"context"

	"smart-recruit-offer-service/internal/domain/event"
	"smart-recruit-offer-service/internal/domain/model"
)

type OfferDetails struct {
	model.Offer
	JobTitle             string
	CandidateName        string
	ApplicationStatusKey model.ApplicationStatus
	CreatedByName        string
	SentByName           string
}

type CandidateOfferPage struct {
	Offers     []OfferDetails
	NextCursor string
	HasMore    bool
	Total      int64
}

type OfferWriter interface {
	Create(ctx context.Context, offer *model.Offer) error
	Save(ctx context.Context, offer *model.Offer) error
	AddEvent(ctx context.Context, event event.OfferEvent) error
}

type OfferRepository interface {
	OfferWriter
	FindByID(ctx context.Context, offerID int64) (*model.Offer, error)
	FindDetailsByID(ctx context.Context, offerID int64) (*OfferDetails, error)
	ListByApplication(ctx context.Context, applicationID int64) ([]OfferDetails, error)
	ListByCandidate(ctx context.Context, candidateUserID int64, cursor string, limit int32) (CandidateOfferPage, error)
	ListEvents(ctx context.Context, offerID int64) ([]event.OfferEvent, error)
	Transaction(ctx context.Context, fn func(ctx context.Context, writer OfferWriter) error) error
}
