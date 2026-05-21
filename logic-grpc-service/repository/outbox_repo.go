package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"logic-grpc-service/model"
)

type OutboxRepo struct {
	db *gorm.DB
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
	return r.db.WithContext(ctx).Model(&model.EventOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        model.EventOutboxStatusPublished,
			"next_retry_at": nil,
			"last_error":    "",
			"locked_at":     nil,
			"locked_by":     "",
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
	return r.db.WithContext(ctx).Model(&model.EventOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      model.EventOutboxStatusDead,
			"last_error":  errMsg,
			"retry_count": gorm.Expr("retry_count + 1"),
			"locked_at":   nil,
			"locked_by":   "",
		}).Error
}
