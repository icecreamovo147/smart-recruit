package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"logic-grpc-service/model"
)

const (
	OutboxPublishedRetention  = 30 * 24 * time.Hour
	OutboxDeadLetterRetention = 90 * 24 * time.Hour
)

type OutboxRepo struct {
	db *gorm.DB
}

type OutboxStats struct {
	Pending         int64
	Retrying        int64
	Processing      int64
	Published       int64
	DeadLettered    int64
	OldestPendingAt *time.Time
	OldestRetryAt   *time.Time
}

func NewOutboxRepo(db *gorm.DB) *OutboxRepo {
	return &OutboxRepo{db: db}
}

func (r *OutboxRepo) Create(ctx context.Context, event *model.EventOutbox) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *OutboxRepo) CreateWithTx(tx *gorm.DB, event *model.EventOutbox) error {
	return tx.Create(event).Error
}

func (r *OutboxRepo) ClaimPending(ctx context.Context, limit int, workerID string, lockTimeout time.Duration) ([]model.EventOutbox, error) {
	var events []model.EventOutbox
	if limit <= 0 {
		return events, nil
	}
	now := time.Now()
	staleBefore := now.Add(-lockTimeout)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"(status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND (locked_at IS NULL OR locked_at <= ?))",
				model.EventOutboxStatusPending, now,
				model.EventOutboxStatusProcessing, staleBefore,
			).
			Order("COALESCE(next_retry_at, created_at) ASC, id ASC").
			Limit(limit).
			Find(&events).Error; err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		ids := make([]uint64, 0, len(events))
		for _, ev := range events {
			ids = append(ids, ev.ID)
		}
		return tx.Model(&model.EventOutbox{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"status":    model.EventOutboxStatusProcessing,
				"locked_at": now,
				"locked_by": workerID,
			}).Error
	})
	return events, err
}

func (r *OutboxRepo) MarkPublished(ctx context.Context, id uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.EventOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        model.EventOutboxStatusPublished,
			"next_retry_at": nil,
			"last_error":    "",
			"locked_at":     nil,
			"locked_by":     "",
			"published_at":  now,
		}).Error
}

func (r *OutboxRepo) MarkRetryableFailure(ctx context.Context, id uint64, errMsg string, nextRetry time.Time) error {
	return r.db.WithContext(ctx).Model(&model.EventOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        model.EventOutboxStatusPending,
			"last_error":    errMsg,
			"retry_count":   gorm.Expr("retry_count + 1"),
			"next_retry_at": nextRetry,
			"locked_at":     nil,
			"locked_by":     "",
		}).Error
}

func (r *OutboxRepo) MarkDead(ctx context.Context, id uint64, errMsg string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.EventOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":           model.EventOutboxStatusDead,
			"last_error":       errMsg,
			"retry_count":      gorm.Expr("retry_count + 1"),
			"next_retry_at":    nil,
			"locked_at":        nil,
			"locked_by":        "",
			"dead_lettered_at": now,
		}).Error
}

func (r *OutboxRepo) Stats(ctx context.Context) (*OutboxStats, error) {
	stats := &OutboxStats{}
	if err := r.db.WithContext(ctx).Model(&model.EventOutbox{}).
		Where("status = ? AND retry_count = 0", model.EventOutboxStatusPending).
		Count(&stats.Pending).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&model.EventOutbox{}).
		Where("status = ? AND retry_count > 0", model.EventOutboxStatusPending).
		Count(&stats.Retrying).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&model.EventOutbox{}).
		Where("status = ?", model.EventOutboxStatusProcessing).
		Count(&stats.Processing).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&model.EventOutbox{}).
		Where("status = ?", model.EventOutboxStatusPublished).
		Count(&stats.Published).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&model.EventOutbox{}).
		Where("status = ?", model.EventOutboxStatusDead).
		Count(&stats.DeadLettered).Error; err != nil {
		return nil, err
	}
	var oldestPending model.EventOutbox
	err := r.db.WithContext(ctx).Where("status = ?", model.EventOutboxStatusPending).
		Order("created_at ASC, id ASC").
		First(&oldestPending).Error
	if err == nil {
		stats.OldestPendingAt = &oldestPending.CreatedAt
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	var oldestRetry model.EventOutbox
	err = r.db.WithContext(ctx).Where("status = ? AND retry_count > 0 AND next_retry_at IS NOT NULL", model.EventOutboxStatusPending).
		Order("next_retry_at ASC, id ASC").
		First(&oldestRetry).Error
	if err == nil {
		stats.OldestRetryAt = oldestRetry.NextRetryAt
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return stats, nil
}

func OutboxRetentionCutoffs(now time.Time) (publishedBefore time.Time, deadLetteredBefore time.Time) {
	return now.Add(-OutboxPublishedRetention), now.Add(-OutboxDeadLetterRetention)
}

func (r *OutboxRepo) DeletePublishedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("status = ? AND published_at IS NOT NULL AND published_at < ?", model.EventOutboxStatusPublished, cutoff).
		Delete(&model.EventOutbox{})
	return result.RowsAffected, result.Error
}

func (r *OutboxRepo) DeleteDeadLetteredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("status = ? AND dead_lettered_at IS NOT NULL AND dead_lettered_at < ?", model.EventOutboxStatusDead, cutoff).
		Delete(&model.EventOutbox{})
	return result.RowsAffected, result.Error
}
