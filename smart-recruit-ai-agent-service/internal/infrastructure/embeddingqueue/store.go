package embeddingqueue

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

type inboxRecord struct {
	ID             uint64     `gorm:"column:id;primaryKey"`
	EventID        string     `gorm:"column:event_id;uniqueIndex:uk_embedding_inbox_consumer_event,priority:2"`
	EventType      string     `gorm:"column:event_type"`
	ConsumerName   string     `gorm:"column:consumer_name;uniqueIndex:uk_embedding_inbox_consumer_event,priority:1"`
	IdempotencyKey string     `gorm:"column:idempotency_key"`
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

func (inboxRecord) TableName() string { return "event_inbox" }

func (s *Store) ClaimInbox(ctx context.Context, eventID, eventType, name, idempotencyKey string) (recordID uint64, claimed bool, err error) {
	if s == nil || s.db == nil {
		return 0, false, gorm.ErrInvalidDB
	}
	now := time.Now()
	claimed = true
	row := &inboxRecord{
		EventID: eventID, EventType: eventType, ConsumerName: name, IdempotencyKey: idempotencyKey,
		Status: inboxStatusProcessing, AttemptCount: 1, ReceivedAt: now, ProcessingAt: &now,
		CreatedAt: now, UpdatedAt: now,
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
		var existing inboxRecord
		if err := tx.Where("consumer_name = ? AND event_id = ?", name, eventID).First(&existing).Error; err != nil {
			return err
		}
		*row = existing
		if existing.Status == inboxStatusProcessed || existing.Status == inboxStatusDead {
			claimed = false
			return nil
		}
		if err := tx.Model(&inboxRecord{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"status": inboxStatusProcessing, "attempt_count": gorm.Expr("attempt_count + 1"),
			"last_error": "", "processing_at": now, "dead_lettered_at": nil, "updated_at": now,
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
		return gorm.ErrInvalidDB
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(&inboxRecord{}).Where("id = ?", id).Updates(map[string]any{
		"status": inboxStatusProcessed, "last_error": "", "processed_at": now, "updated_at": now,
	}).Error
}

func (s *Store) MarkInboxFailed(ctx context.Context, id uint64, errMsg string) error {
	if s == nil || s.db == nil || id == 0 {
		return gorm.ErrInvalidDB
	}
	if len(errMsg) > 2000 {
		errMsg = errMsg[:2000]
	}
	return s.db.WithContext(ctx).Model(&inboxRecord{}).Where("id = ?", id).Updates(map[string]any{
		"status": inboxStatusFailed, "last_error": errMsg, "updated_at": time.Now(),
	}).Error
}
