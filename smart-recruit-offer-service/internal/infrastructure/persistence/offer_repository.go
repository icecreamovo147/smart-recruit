package persistence

import (
	"context"

	"gorm.io/gorm"

	domainevent "smart-recruit-offer-service/internal/domain/event"
	"smart-recruit-offer-service/internal/domain/model"
	"smart-recruit-offer-service/internal/domain/repository"
	sharedmodel "smart-recruit-offer-service/internal/legacydomain/model"
	sharedrepo "smart-recruit-offer-service/internal/legacydomain/repository"
)

type txContextKey struct{}

func ContextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func TxFromContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txContextKey{}).(*gorm.DB)
	return tx, ok && tx != nil
}

type OfferRepository struct {
	offers *sharedrepo.OfferRepo
}

func NewOfferRepository(offers *sharedrepo.OfferRepo) *OfferRepository {
	return &OfferRepository{offers: offers}
}

func (r *OfferRepository) Create(ctx context.Context, offer *model.Offer) error {
	return r.offers.Create(ctx, toSharedOffer(offer))
}

func (r *OfferRepository) Save(ctx context.Context, offer *model.Offer) error {
	return r.offers.Update(ctx, toSharedOffer(offer))
}

func (r *OfferRepository) AddEvent(ctx context.Context, event domainevent.OfferEvent) error {
	return r.offers.CreateEvent(ctx, toSharedOfferEvent(event))
}

func (r *OfferRepository) FindByID(ctx context.Context, offerID int64) (*model.Offer, error) {
	offer, err := r.offers.GetModelByID(ctx, offerID)
	if err != nil || offer == nil {
		return nil, err
	}
	return fromSharedOffer(offer), nil
}

func (r *OfferRepository) FindDetailsByID(ctx context.Context, offerID int64) (*repository.OfferDetails, error) {
	row, err := r.offers.GetByID(ctx, offerID)
	if err != nil || row == nil {
		return nil, err
	}
	return fromSharedOfferDetails(row), nil
}

func (r *OfferRepository) ListByApplication(ctx context.Context, applicationID int64) ([]repository.OfferDetails, error) {
	rows, err := r.offers.ListByApplication(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	return fromSharedOfferDetailsRows(rows), nil
}

func (r *OfferRepository) ListByCandidate(ctx context.Context, candidateUserID int64, cursor string, limit int32) (repository.CandidateOfferPage, error) {
	total, err := r.offers.CountByCandidate(ctx, candidateUserID)
	if err != nil {
		return repository.CandidateOfferPage{}, err
	}
	rows, nextCursor, hasMore, err := r.offers.ListByCandidate(ctx, candidateUserID, cursor, limit)
	if err != nil {
		return repository.CandidateOfferPage{}, err
	}
	return repository.CandidateOfferPage{
		Offers:     fromSharedOfferDetailsRows(rows),
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Total:      total,
	}, nil
}

func (r *OfferRepository) ListEvents(ctx context.Context, offerID int64) ([]domainevent.OfferEvent, error) {
	rows, err := r.offers.ListEventsByOfferID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	events := make([]domainevent.OfferEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, fromSharedOfferEvent(row))
	}
	return events, nil
}

func (r *OfferRepository) Transaction(ctx context.Context, fn func(ctx context.Context, writer repository.OfferWriter) error) error {
	return r.offers.Transaction(ctx, func(tx *gorm.DB) error {
		txCtx := ContextWithTx(ctx, tx)
		return fn(txCtx, txWriter{offers: r.offers, tx: tx})
	})
}

type txWriter struct {
	offers *sharedrepo.OfferRepo
	tx     *gorm.DB
}

func (w txWriter) Create(ctx context.Context, offer *model.Offer) error {
	shared := toSharedOffer(offer)
	if err := w.offers.CreateWithTx(ctx, w.tx, shared); err != nil {
		return err
	}
	offer.ID = shared.ID
	offer.CreatedAt = shared.CreatedAt
	offer.UpdatedAt = shared.UpdatedAt
	return nil
}

func (w txWriter) Save(ctx context.Context, offer *model.Offer) error {
	return w.offers.UpdateWithTx(ctx, w.tx, toSharedOffer(offer))
}

func (w txWriter) AddEvent(ctx context.Context, event domainevent.OfferEvent) error {
	return w.offers.CreateEventWithTx(ctx, w.tx, toSharedOfferEvent(event))
}

func toSharedOffer(offer *model.Offer) *sharedmodel.Offer {
	if offer == nil {
		return nil
	}
	return &sharedmodel.Offer{
		ID:               offer.ID,
		ApplicationID:    offer.ApplicationID,
		CandidateUserID:  offer.CandidateUserID,
		JobID:            offer.JobID,
		Status:           string(offer.Status),
		Title:            offer.Title,
		SalaryRange:      offer.SalaryRange,
		Level:            offer.Level,
		WorkLocation:     offer.WorkLocation,
		StartDate:        offer.StartDate,
		ExpiresAt:        offer.ExpiresAt,
		TermsJSON:        offer.TermsJSON,
		SentSnapshotJSON: offer.SentSnapshotJSON,
		CreatedBy:        offer.CreatedBy,
		SentBy:           offer.SentBy,
		DecidedAt:        offer.DecidedAt,
		CreatedAt:        offer.CreatedAt,
		UpdatedAt:        offer.UpdatedAt,
	}
}

func fromSharedOffer(offer *sharedmodel.Offer) *model.Offer {
	if offer == nil {
		return nil
	}
	return &model.Offer{
		ID:               offer.ID,
		ApplicationID:    offer.ApplicationID,
		CandidateUserID:  offer.CandidateUserID,
		JobID:            offer.JobID,
		Status:           model.OfferStatus(offer.Status),
		Title:            offer.Title,
		SalaryRange:      offer.SalaryRange,
		Level:            offer.Level,
		WorkLocation:     offer.WorkLocation,
		StartDate:        offer.StartDate,
		ExpiresAt:        offer.ExpiresAt,
		TermsJSON:        offer.TermsJSON,
		SentSnapshotJSON: offer.SentSnapshotJSON,
		CreatedBy:        offer.CreatedBy,
		SentBy:           offer.SentBy,
		DecidedAt:        offer.DecidedAt,
		CreatedAt:        offer.CreatedAt,
		UpdatedAt:        offer.UpdatedAt,
	}
}

func fromSharedOfferDetails(row *sharedrepo.OfferWithDetailsRow) *repository.OfferDetails {
	if row == nil {
		return nil
	}
	return &repository.OfferDetails{
		Offer: model.Offer{
			ID:               row.ID,
			ApplicationID:    row.ApplicationID,
			CandidateUserID:  row.CandidateUserID,
			JobID:            row.JobID,
			Status:           model.OfferStatus(row.Status),
			Title:            row.Title,
			SalaryRange:      row.SalaryRange,
			Level:            row.Level,
			WorkLocation:     row.WorkLocation,
			StartDate:        row.StartDate,
			ExpiresAt:        row.ExpiresAt,
			TermsJSON:        row.TermsJSON,
			SentSnapshotJSON: row.SentSnapshotJSON,
			CreatedBy:        row.CreatedBy,
			SentBy:           row.SentBy,
			DecidedAt:        row.DecidedAt,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		},
		JobTitle:             row.JobTitle,
		CandidateName:        row.CandidateName,
		ApplicationStatusKey: model.ApplicationStatus(row.ApplicationStatusKey),
		CreatedByName:        row.CreatedByName,
		SentByName:           row.SentByName,
	}
}

func fromSharedOfferDetailsRows(rows []sharedrepo.OfferWithDetailsRow) []repository.OfferDetails {
	details := make([]repository.OfferDetails, 0, len(rows))
	for i := range rows {
		details = append(details, *fromSharedOfferDetails(&rows[i]))
	}
	return details
}

func toSharedOfferEvent(event domainevent.OfferEvent) *sharedmodel.OfferEvent {
	return &sharedmodel.OfferEvent{
		ID:               event.ID,
		OfferID:          event.OfferID,
		EventType:        string(event.EventType),
		ActorUserID:      event.ActorUserID,
		ActorAccountType: event.ActorAccountType,
		Reason:           event.Reason,
		MetadataJSON:     event.MetadataJSON,
		CreatedAt:        event.CreatedAt,
	}
}

func fromSharedOfferEvent(event sharedmodel.OfferEvent) domainevent.OfferEvent {
	return domainevent.OfferEvent{
		ID:               event.ID,
		OfferID:          event.OfferID,
		EventType:        domainevent.Type(event.EventType),
		ActorUserID:      event.ActorUserID,
		ActorAccountType: event.ActorAccountType,
		Reason:           event.Reason,
		MetadataJSON:     event.MetadataJSON,
		CreatedAt:        event.CreatedAt,
	}
}
