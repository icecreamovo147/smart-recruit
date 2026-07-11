package application

import (
	"context"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

// OfferRepository is the Offer-owned lifecycle and event persistence port.
type OfferRepository interface {
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	Create(ctx context.Context, o *model.Offer) error
	CreateWithTx(ctx context.Context, tx *gorm.DB, o *model.Offer) error
	Update(ctx context.Context, o *model.Offer) error
	UpdateWithTx(ctx context.Context, tx *gorm.DB, o *model.Offer) error
	UpdateStatus(ctx context.Context, offerID int64, updates map[string]any) error
	UpdateStatusWithTx(ctx context.Context, tx *gorm.DB, offerID int64, updates map[string]any) error
	GetByID(ctx context.Context, id int64) (*repository.OfferWithDetailsRow, error)
	GetModelByID(ctx context.Context, id int64) (*model.Offer, error)
	ListByApplication(ctx context.Context, applicationID int64) ([]repository.OfferWithDetailsRow, error)
	CountByCandidate(ctx context.Context, candidateUserID int64) (int64, error)
	ListByCandidate(ctx context.Context, candidateUserID int64, cursor string, limit int32) ([]repository.OfferWithDetailsRow, string, bool, error)
	CreateEvent(ctx context.Context, e *model.OfferEvent) error
	CreateEventWithTx(ctx context.Context, tx *gorm.DB, e *model.OfferEvent) error
	ListEventsByOfferID(ctx context.Context, offerID int64) ([]model.OfferEvent, error)
}
