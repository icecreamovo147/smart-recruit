package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/pagination"
)

type OfferRepo struct {
	db *gorm.DB
}

type OfferWithDetailsRow struct {
	ID               int64
	ApplicationID    int64
	CandidateUserID  int64
	JobID            int64
	Status           string
	Title            string
	SalaryRange      string
	Level            string
	WorkLocation     string
	StartDate        string
	ExpiresAt        *time.Time
	TermsJSON        string
	SentSnapshotJSON string
	CreatedBy        int64
	SentBy           *int64
	DecidedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Joined fields
	JobTitle            string
	CandidateName       string
	ApplicationStatusKey string
}

func NewOfferRepo(db *gorm.DB) *OfferRepo {
	return &OfferRepo{db: db}
}

// baseOfferSelect returns the common select expression for joined offer queries.
func baseOfferSelect() string {
	return `offers.id, offers.application_id, offers.candidate_user_id, offers.job_id,
		offers.status, offers.title, offers.salary_range, offers.level, offers.work_location,
		offers.start_date, offers.expires_at, offers.terms_json, offers.sent_snapshot_json,
		offers.created_by, offers.sent_by, offers.decided_at, offers.created_at, offers.updated_at,
		j.title AS job_title,
		COALESCE(cp.real_name, CONCAT('候选人', offers.candidate_user_id)) AS candidate_name,
		a.status_key AS application_status_key`
}

func (r *OfferRepo) baseJoins() *gorm.DB {
	return r.db.Table("offers").
		Select(baseOfferSelect()).
		Joins("JOIN jobs j ON j.id = offers.job_id").
		Joins("JOIN applications a ON a.id = offers.application_id").
		Joins("LEFT JOIN candidate_profiles cp ON cp.user_id = offers.candidate_user_id")
}

// ── Offer CRUD ──────────────────────────────────────────────────────────

func (r *OfferRepo) Create(ctx context.Context, o *model.Offer) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *OfferRepo) CreateWithTx(ctx context.Context, tx *gorm.DB, o *model.Offer) error {
	return tx.WithContext(ctx).Create(o).Error
}

func (r *OfferRepo) Update(ctx context.Context, o *model.Offer) error {
	return r.db.WithContext(ctx).Save(o).Error
}

func (r *OfferRepo) UpdateWithTx(ctx context.Context, tx *gorm.DB, o *model.Offer) error {
	return tx.WithContext(ctx).Save(o).Error
}

// UpdateStatus updates the offer status (and optionally sent_by/decided_at) by ID.
// Only updates the fields that are non-zero/non-empty in the provided map.
func (r *OfferRepo) UpdateStatus(ctx context.Context, offerID int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.Offer{}).
		Where("id = ?", offerID).
		Updates(updates).Error
}

func (r *OfferRepo) UpdateStatusWithTx(ctx context.Context, tx *gorm.DB, offerID int64, updates map[string]any) error {
	return tx.WithContext(ctx).Model(&model.Offer{}).
		Where("id = ?", offerID).
		Updates(updates).Error
}

func (r *OfferRepo) GetByID(ctx context.Context, id int64) (*OfferWithDetailsRow, error) {
	var row OfferWithDetailsRow
	err := r.baseJoins().
		Where("offers.id = ?", id).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (r *OfferRepo) GetModelByID(ctx context.Context, id int64) (*model.Offer, error) {
	var o model.Offer
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&o).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

// ListByApplication returns all offers for an application.
func (r *OfferRepo) ListByApplication(ctx context.Context, applicationID int64) ([]OfferWithDetailsRow, error) {
	var rows []OfferWithDetailsRow
	err := r.baseJoins().
		Where("offers.application_id = ?", applicationID).
		Order("offers.created_at DESC").
		Scan(&rows).Error
	return rows, err
}

// CountByCandidate returns the total number of offers for a candidate.
func (r *OfferRepo) CountByCandidate(ctx context.Context, candidateUserID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Offer{}).
		Where("candidate_user_id = ?", candidateUserID).
		Count(&count).Error
	return count, err
}

// ListByCandidate returns offers for a candidate using cursor-based pagination.
func (r *OfferRepo) ListByCandidate(ctx context.Context, candidateUserID int64, cursor string, limit int32) ([]OfferWithDetailsRow, string, bool, error) {
	t, id, err := pagination.DecodeCursor(cursor)
	if err != nil {
		return nil, "", false, err
	}
	query := r.baseJoins().
		Where("offers.candidate_user_id = ?", candidateUserID)
	if !t.IsZero() || id > 0 {
		query = query.Where("(offers.created_at, offers.id) < (?, ?)", t, id)
	}
	fetchLimit := int(limit) + 1
	var rows []OfferWithDetailsRow
	if err := query.Order("offers.created_at DESC, offers.id DESC").Limit(fetchLimit).Scan(&rows).Error; err != nil {
		return nil, "", false, err
	}
	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}
	var nextCursor string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	return rows, nextCursor, hasMore, nil
}

// ── Offer Events ────────────────────────────────────────────────────────

func (r *OfferRepo) CreateEvent(ctx context.Context, e *model.OfferEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *OfferRepo) CreateEventWithTx(ctx context.Context, tx *gorm.DB, e *model.OfferEvent) error {
	return tx.WithContext(ctx).Create(e).Error
}

func (r *OfferRepo) ListEventsByOfferID(ctx context.Context, offerID int64) ([]model.OfferEvent, error) {
	var rows []model.OfferEvent
	err := r.db.WithContext(ctx).
		Where("offer_id = ?", offerID).
		Order("created_at ASC").
		Find(&rows).Error
	return rows, err
}

// Transaction wraps a function in a DB transaction.
func (r *OfferRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
