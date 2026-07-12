package infrastructure

import (
	"gorm.io/gorm"

	"logic-grpc-service/repository"
)

// OfferRepository adapts the current GORM offer repository to Offer application ports.
type OfferRepository = repository.OfferRepo

// NewOfferRepository creates an Offer-owned repository adapter.
func NewOfferRepository(db *gorm.DB) *OfferRepository {
	return repository.NewOfferRepo(db)
}
