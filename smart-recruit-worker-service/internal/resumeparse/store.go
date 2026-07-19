package resumeparse

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	inboxStatusProcessing int32 = 0
	inboxStatusProcessed  int32 = 1
	inboxStatusFailed     int32 = 2
	inboxStatusDead       int32 = 3
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

type inboxRow struct {
	ID             uint64     `gorm:"primaryKey"`
	TenantID       *int64     `gorm:"column:tenant_id"`
	EventID        string     `gorm:"column:event_id;size:128"`
	EventType      string     `gorm:"column:event_type;size:128"`
	ConsumerName   string     `gorm:"column:consumer_name;size:128"`
	IdempotencyKey string     `gorm:"column:idempotency_key;size:255"`
	Status         int32      `gorm:"column:status"`
	AttemptCount   int32      `gorm:"column:attempt_count"`
	LastError      string     `gorm:"column:last_error"`
	ReceivedAt     time.Time  `gorm:"column:received_at"`
	ProcessingAt   *time.Time `gorm:"column:processing_at"`
	ProcessedAt    *time.Time `gorm:"column:processed_at"`
	DeadLetteredAt *time.Time `gorm:"column:dead_lettered_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (inboxRow) TableName() string { return "event_inbox" }

// ClaimInbox inserts or reclaims an inbox row for (consumer, event_id).
// claimed=false means the event was already processed/dead and should be skipped.
func (s *Store) ClaimInbox(ctx context.Context, eventID, eventType, consumerName, idempotencyKey string) (recordID uint64, claimed bool, err error) {
	if s == nil || s.db == nil {
		return 0, true, nil
	}
	now := time.Now()
	claimed = true
	row := &inboxRow{
		EventID:        eventID,
		EventType:      eventType,
		ConsumerName:   consumerName,
		IdempotencyKey: idempotencyKey,
		Status:         inboxStatusProcessing,
		AttemptCount:   1,
		ReceivedAt:     now,
		ProcessingAt:   &now,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "consumer_name"}, {Name: "event_id"}},
			DoNothing: true,
		}).Create(row)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 1 {
			return nil
		}
		var existing inboxRow
		if err := tx.Where("consumer_name = ? AND event_id = ?", consumerName, eventID).First(&existing).Error; err != nil {
			return err
		}
		*row = existing
		if existing.Status == inboxStatusProcessed || existing.Status == inboxStatusDead {
			claimed = false
			return nil
		}
		if err := tx.Model(&inboxRow{}).
			Where("id = ?", existing.ID).
			Updates(map[string]any{
				"status":           inboxStatusProcessing,
				"attempt_count":    gorm.Expr("attempt_count + 1"),
				"last_error":       "",
				"processing_at":    now,
				"dead_lettered_at": nil,
			}).Error; err != nil {
			return err
		}
		return tx.First(row, existing.ID).Error
	})
	if err != nil {
		return 0, false, err
	}
	return row.ID, claimed, nil
}

func (s *Store) MarkInboxProcessed(ctx context.Context, id uint64) error {
	if s == nil || s.db == nil || id == 0 {
		return nil
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(&inboxRow{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": inboxStatusProcessed, "last_error": "", "processed_at": now}).Error
}

func (s *Store) MarkInboxFailed(ctx context.Context, id uint64, errMsg string) error {
	if s == nil || s.db == nil || id == 0 {
		return nil
	}
	if len(errMsg) > 2000 {
		errMsg = errMsg[:2000]
	}
	return s.db.WithContext(ctx).Model(&inboxRow{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": inboxStatusFailed, "last_error": errMsg}).Error
}

// UpdateParsedText writes extracted resume text and parsed_at for the resume row.
func (s *Store) UpdateParsedText(ctx context.Context, resumeID int64, text string) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	now := time.Now()
	result := s.db.WithContext(ctx).Table("resumes").
		Where("id = ?", resumeID).
		Updates(map[string]any{
			"parsed_text": text,
			"parsed_at":   now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// HasParsedText reports whether the resume already has non-empty parsed text.
func (s *Store) HasParsedText(ctx context.Context, resumeID int64) (bool, error) {
	if s == nil || s.db == nil {
		return false, gorm.ErrInvalidDB
	}
	var count int64
	err := s.db.WithContext(ctx).Table("resumes").
		Where("id = ? AND parsed_text IS NOT NULL AND TRIM(parsed_text) <> ''", resumeID).
		Count(&count).Error
	return count > 0, err
}
