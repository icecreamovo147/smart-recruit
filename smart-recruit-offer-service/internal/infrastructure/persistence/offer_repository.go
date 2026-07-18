package persistence

import (
	"context"
	"time"

	"gorm.io/gorm"

	"smart-recruit-commons/pkg/pagination"
	domainevent "smart-recruit-offer-service/internal/domain/event"
	"smart-recruit-offer-service/internal/domain/model"
	"smart-recruit-offer-service/internal/domain/repository"
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
	db *gorm.DB
}

func NewOfferRepository(db *gorm.DB) *OfferRepository {
	return &OfferRepository{db: db}
}

func (r *OfferRepository) Create(ctx context.Context, offer *model.Offer) error {
	record := toOfferRecord(offer)
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return err
	}
	copyOfferRecord(offer, record)
	return nil
}

func (r *OfferRepository) Save(ctx context.Context, offer *model.Offer) error {
	return r.db.WithContext(ctx).Save(toOfferRecord(offer)).Error
}

func (r *OfferRepository) AddEvent(ctx context.Context, event domainevent.OfferEvent) error {
	return r.db.WithContext(ctx).Create(toOfferEventRecord(event)).Error
}

func (r *OfferRepository) FindByID(ctx context.Context, offerID int64) (*model.Offer, error) {
	var record offerRecord
	err := r.db.WithContext(ctx).Where("id = ?", offerID).First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return fromOfferRecord(&record), nil
}

func (r *OfferRepository) FindDetailsByID(ctx context.Context, offerID int64) (*repository.OfferDetails, error) {
	var row offerDetailsRow
	result := r.baseJoins(ctx).Where("offers.id = ?", offerID).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 || row.ID == 0 {
		return nil, nil
	}
	return fromOfferDetailsRow(&row), nil
}

func (r *OfferRepository) ListByApplication(ctx context.Context, applicationID int64) ([]repository.OfferDetails, error) {
	var rows []offerDetailsRow
	err := r.baseJoins(ctx).
		Where("offers.application_id = ?", applicationID).
		Order("offers.created_at DESC").
		Scan(&rows).Error
	return fromOfferDetailsRows(rows), err
}

func (r *OfferRepository) ListByCandidate(ctx context.Context, candidateUserID int64, cursor string, limit int32) (repository.CandidateOfferPage, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&offerRecord{}).Where("candidate_user_id = ?", candidateUserID).Count(&total).Error; err != nil {
		return repository.CandidateOfferPage{}, err
	}
	t, id, err := pagination.DecodeCursor(cursor)
	if err != nil {
		return repository.CandidateOfferPage{}, err
	}
	query := r.baseJoins(ctx).
		Where("offers.candidate_user_id = ?", candidateUserID).
		Where("offers.status != ?", "draft")
	if !t.IsZero() || id > 0 {
		query = query.Where("(offers.created_at, offers.id) < (?, ?)", t, id)
	}
	fetchLimit := int(limit) + 1
	var rows []offerDetailsRow
	if err := query.Order("offers.created_at DESC, offers.id DESC").Limit(fetchLimit).Scan(&rows).Error; err != nil {
		return repository.CandidateOfferPage{}, err
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
	return repository.CandidateOfferPage{
		Offers:     fromOfferDetailsRows(rows),
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Total:      total,
	}, nil
}

func (r *OfferRepository) ListEvents(ctx context.Context, offerID int64) ([]domainevent.OfferEvent, error) {
	var records []offerEventRecord
	err := r.db.WithContext(ctx).Where("offer_id = ?", offerID).Order("created_at ASC").Find(&records).Error
	if err != nil {
		return nil, err
	}
	events := make([]domainevent.OfferEvent, 0, len(records))
	for _, record := range records {
		events = append(events, fromOfferEventRecord(record))
	}
	return events, nil
}

func (r *OfferRepository) Transaction(ctx context.Context, fn func(ctx context.Context, writer repository.OfferWriter) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := ContextWithTx(ctx, tx)
		return fn(txCtx, txWriter{tx: tx})
	})
}

func (r *OfferRepository) baseJoins(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Table("offers").
		Select(`offers.id, offers.application_id, offers.candidate_user_id, offers.job_id,
			offers.status, offers.title, offers.salary_range, offers.level, offers.work_location,
			offers.start_date, offers.expires_at, offers.terms_json, offers.sent_snapshot_json,
			offers.created_by, offers.sent_by, offers.decided_at, offers.created_at, offers.updated_at,
			j.title AS job_title,
			COALESCE(cp.real_name, CONCAT('候选人', offers.candidate_user_id)) AS candidate_name,
			a.status_key AS application_status_key,
			cu.username AS created_by_name,
			COALESCE(su.username, '') AS sent_by_name`).
		Joins("JOIN jobs j ON j.id = offers.job_id").
		Joins("JOIN applications a ON a.id = offers.application_id").
		Joins("LEFT JOIN candidate_profiles cp ON cp.user_id = offers.candidate_user_id").
		Joins("LEFT JOIN users cu ON cu.id = offers.created_by").
		Joins("LEFT JOIN users su ON su.id = offers.sent_by")
}

type txWriter struct {
	tx *gorm.DB
}

func (w txWriter) Create(ctx context.Context, offer *model.Offer) error {
	record := toOfferRecord(offer)
	if err := w.tx.WithContext(ctx).Create(record).Error; err != nil {
		return err
	}
	copyOfferRecord(offer, record)
	return nil
}

func (w txWriter) Save(ctx context.Context, offer *model.Offer) error {
	return w.tx.WithContext(ctx).Save(toOfferRecord(offer)).Error
}

func (w txWriter) AddEvent(ctx context.Context, event domainevent.OfferEvent) error {
	return w.tx.WithContext(ctx).Create(toOfferEventRecord(event)).Error
}

type offerRecord struct {
	ID               int64      `gorm:"primaryKey"`
	ApplicationID    int64      `gorm:"column:application_id"`
	CandidateUserID  int64      `gorm:"column:candidate_user_id"`
	JobID            int64      `gorm:"column:job_id"`
	Status           string     `gorm:"column:status"`
	Title            string     `gorm:"column:title"`
	SalaryRange      string     `gorm:"column:salary_range"`
	Level            string     `gorm:"column:level"`
	WorkLocation     string     `gorm:"column:work_location"`
	StartDate        string     `gorm:"column:start_date"`
	ExpiresAt        *time.Time `gorm:"column:expires_at"`
	TermsJSON        string     `gorm:"column:terms_json"`
	SentSnapshotJSON string     `gorm:"column:sent_snapshot_json"`
	CreatedBy        int64      `gorm:"column:created_by"`
	SentBy           *int64     `gorm:"column:sent_by"`
	DecidedAt        *time.Time `gorm:"column:decided_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (offerRecord) TableName() string { return "offers" }

type offerEventRecord struct {
	ID               uint64    `gorm:"primaryKey"`
	OfferID          int64     `gorm:"column:offer_id"`
	EventType        string    `gorm:"column:event_type"`
	ActorUserID      int64     `gorm:"column:actor_user_id"`
	ActorAccountType string    `gorm:"column:actor_account_type"`
	Reason           string    `gorm:"column:reason"`
	MetadataJSON     string    `gorm:"column:metadata_json"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

func (offerEventRecord) TableName() string { return "offer_events" }

type offerDetailsRow struct {
	ID                   int64
	ApplicationID        int64
	CandidateUserID      int64
	JobID                int64
	Status               string
	Title                string
	SalaryRange          string
	Level                string
	WorkLocation         string
	StartDate            string
	ExpiresAt            *time.Time
	TermsJSON            string
	SentSnapshotJSON     string
	CreatedBy            int64
	SentBy               *int64
	DecidedAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	JobTitle             string
	CandidateName        string
	ApplicationStatusKey string
	CreatedByName        string
	SentByName           string
}

func toOfferRecord(offer *model.Offer) *offerRecord {
	if offer == nil {
		return nil
	}
	return &offerRecord{
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

func copyOfferRecord(offer *model.Offer, record *offerRecord) {
	offer.ID = record.ID
	offer.CreatedAt = record.CreatedAt
	offer.UpdatedAt = record.UpdatedAt
}

func fromOfferRecord(record *offerRecord) *model.Offer {
	if record == nil {
		return nil
	}
	return &model.Offer{
		ID:               record.ID,
		ApplicationID:    record.ApplicationID,
		CandidateUserID:  record.CandidateUserID,
		JobID:            record.JobID,
		Status:           model.OfferStatus(record.Status),
		Title:            record.Title,
		SalaryRange:      record.SalaryRange,
		Level:            record.Level,
		WorkLocation:     record.WorkLocation,
		StartDate:        record.StartDate,
		ExpiresAt:        record.ExpiresAt,
		TermsJSON:        record.TermsJSON,
		SentSnapshotJSON: record.SentSnapshotJSON,
		CreatedBy:        record.CreatedBy,
		SentBy:           record.SentBy,
		DecidedAt:        record.DecidedAt,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func fromOfferDetailsRow(row *offerDetailsRow) *repository.OfferDetails {
	if row == nil {
		return nil
	}
	return &repository.OfferDetails{
		Offer:                *offerFromDetailsRow(row),
		JobTitle:             row.JobTitle,
		CandidateName:        row.CandidateName,
		ApplicationStatusKey: model.ApplicationStatus(row.ApplicationStatusKey),
		CreatedByName:        row.CreatedByName,
		SentByName:           row.SentByName,
	}
}

func fromOfferDetailsRows(rows []offerDetailsRow) []repository.OfferDetails {
	details := make([]repository.OfferDetails, 0, len(rows))
	for i := range rows {
		details = append(details, *fromOfferDetailsRow(&rows[i]))
	}
	return details
}

func offerFromDetailsRow(row *offerDetailsRow) *model.Offer {
	return &model.Offer{
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
	}
}

func toOfferEventRecord(event domainevent.OfferEvent) *offerEventRecord {
	return &offerEventRecord{
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

func fromOfferEventRecord(event offerEventRecord) domainevent.OfferEvent {
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
