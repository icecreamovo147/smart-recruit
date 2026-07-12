package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"smart-recruit-domain-go/model"
)

const (
	InboxProcessedRetention  = 30 * 24 * time.Hour
	InboxDeadLetterRetention = 90 * 24 * time.Hour
)

type InboxRepo struct {
	db *gorm.DB
}

type InboxClaim struct {
	EventID        string
	EventType      string
	ConsumerName   string
	IdempotencyKey string
}

type InboxStats struct {
	Processing   int64
	Processed    int64
	Failed       int64
	DeadLettered int64
}

func NewInboxRepo(db *gorm.DB) *InboxRepo {
	return &InboxRepo{db: db}
}

func (r *InboxRepo) Claim(ctx context.Context, claim InboxClaim) (*model.EventInbox, bool, error) {
	now := time.Now()
	claimed := true
	record := &model.EventInbox{
		EventID:        claim.EventID,
		EventType:      claim.EventType,
		ConsumerName:   claim.ConsumerName,
		IdempotencyKey: claim.IdempotencyKey,
		Status:         model.EventInboxStatusProcessing,
		AttemptCount:   1,
		ReceivedAt:     now,
		ProcessingAt:   &now,
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "consumer_name"}, {Name: "event_id"}},
			DoNothing: true,
		}).Create(record)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 1 {
			return nil
		}
		var existing model.EventInbox
		if err := tx.Where("consumer_name = ? AND event_id = ?", claim.ConsumerName, claim.EventID).First(&existing).Error; err != nil {
			return err
		}
		*record = existing
		if existing.Status == model.EventInboxStatusProcessed || existing.Status == model.EventInboxStatusDead {
			claimed = false
			return nil
		}
		if err := tx.Model(&model.EventInbox{}).
			Where("id = ?", existing.ID).
			Updates(map[string]any{
				"status":           model.EventInboxStatusProcessing,
				"attempt_count":    gorm.Expr("attempt_count + 1"),
				"last_error":       "",
				"processing_at":    now,
				"dead_lettered_at": nil,
			}).Error; err != nil {
			return err
		}
		return tx.First(record, existing.ID).Error
	})
	if err != nil {
		return nil, false, err
	}
	return record, claimed, nil
}

func (r *InboxRepo) MarkProcessed(ctx context.Context, id uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.EventInbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       model.EventInboxStatusProcessed,
			"last_error":   "",
			"processed_at": now,
		}).Error
}

func (r *InboxRepo) MarkFailed(ctx context.Context, id uint64, errMsg string) error {
	return r.db.WithContext(ctx).Model(&model.EventInbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     model.EventInboxStatusFailed,
			"last_error": errMsg,
		}).Error
}

func (r *InboxRepo) MarkDead(ctx context.Context, id uint64, errMsg string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.EventInbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":           model.EventInboxStatusDead,
			"last_error":       errMsg,
			"dead_lettered_at": now,
		}).Error
}

func (r *InboxRepo) Stats(ctx context.Context) (*InboxStats, error) {
	stats := &InboxStats{}
	counts := []struct {
		status int32
		target *int64
	}{
		{model.EventInboxStatusProcessing, &stats.Processing},
		{model.EventInboxStatusProcessed, &stats.Processed},
		{model.EventInboxStatusFailed, &stats.Failed},
		{model.EventInboxStatusDead, &stats.DeadLettered},
	}
	for _, item := range counts {
		if err := r.db.WithContext(ctx).Model(&model.EventInbox{}).
			Where("status = ?", item.status).
			Count(item.target).Error; err != nil {
			return nil, err
		}
	}
	return stats, nil
}

func InboxRetentionCutoffs(now time.Time) (processedBefore time.Time, deadLetteredBefore time.Time) {
	return now.Add(-InboxProcessedRetention), now.Add(-InboxDeadLetterRetention)
}

func (r *InboxRepo) DeleteProcessedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("status = ? AND processed_at IS NOT NULL AND processed_at < ?", model.EventInboxStatusProcessed, cutoff).
		Delete(&model.EventInbox{})
	return result.RowsAffected, result.Error
}

func (r *InboxRepo) DeleteDeadLetteredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("status = ? AND dead_lettered_at IS NOT NULL AND dead_lettered_at < ?", model.EventInboxStatusDead, cutoff).
		Delete(&model.EventInbox{})
	return result.RowsAffected, result.Error
}

func (r *InboxRepo) GetByConsumerEvent(ctx context.Context, consumerName, eventID string) (*model.EventInbox, error) {
	var record model.EventInbox
	if err := r.db.WithContext(ctx).Where("consumer_name = ? AND event_id = ?", consumerName, eventID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}
