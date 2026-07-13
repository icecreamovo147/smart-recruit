package mq

import (
	"context"
	"math"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	sharedmq "smart-recruit-domain-go/mq"
	"smart-recruit-platform-go/logger"
)

const (
	defaultPollInterval = 5 * time.Second
	defaultBatchSize    = 50
	defaultLockTimeout  = 2 * time.Minute
	maxBackoff          = 10 * time.Minute
	backoffBase         = 5 * time.Second
	publishTimeout      = 10 * time.Second
	maxRetryCount       = 10

	eventOutboxStatusPending    int32 = 0
	eventOutboxStatusPublished  int32 = 1
	eventOutboxStatusDead       int32 = 2
	eventOutboxStatusProcessing int32 = 3
)

type eventOutbox struct {
	ID             uint64     `gorm:"primaryKey"`
	EventID        string     `gorm:"column:event_id"`
	RoutingKey     string     `gorm:"column:routing_key"`
	Payload        string     `gorm:"column:payload;type:json"`
	Status         int32      `gorm:"column:status"`
	RetryCount     int32      `gorm:"column:retry_count"`
	NextRetryAt    *time.Time `gorm:"column:next_retry_at"`
	LastError      string     `gorm:"column:last_error"`
	LockedAt       *time.Time `gorm:"column:locked_at"`
	LockedBy       string     `gorm:"column:locked_by"`
	PublishedAt    *time.Time `gorm:"column:published_at"`
	DeadLetteredAt *time.Time `gorm:"column:dead_lettered_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

func (eventOutbox) TableName() string { return "event_outbox" }

type OutboxPublisher struct {
	db           *gorm.DB
	mqConn       *sharedmq.Conn
	pollInterval time.Duration
	batchSize    int
	lockTimeout  time.Duration
	workerID     string
	notifyCh     chan struct{}
}

func NewOutboxPublisher(db *gorm.DB, mqConn *sharedmq.Conn) *OutboxPublisher {
	return &OutboxPublisher{
		db:           db,
		mqConn:       mqConn,
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
		lockTimeout:  defaultLockTimeout,
		workerID:     "notification-outbox",
		notifyCh:     make(chan struct{}, 1),
	}
}

func (p *OutboxPublisher) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(p.pollInterval)
		defer ticker.Stop()
		p.poll(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.poll(ctx)
			case <-p.notifyCh:
				p.poll(ctx)
			}
		}
	}()
}

func (p *OutboxPublisher) Signal() {
	select {
	case p.notifyCh <- struct{}{}:
	default:
	}
}

func (p *OutboxPublisher) poll(ctx context.Context) {
	events, err := p.claimPending(ctx)
	if err != nil {
		logger.L().Error("notification outbox claim pending failed", zap.Error(err))
		return
	}
	for _, event := range events {
		p.publishOne(ctx, event)
	}
}

func (p *OutboxPublisher) claimPending(ctx context.Context) ([]eventOutbox, error) {
	var events []eventOutbox
	now := time.Now()
	staleBefore := now.Add(-p.lockTimeout)
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"(status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND (locked_at IS NULL OR locked_at <= ?))",
				eventOutboxStatusPending, now,
				eventOutboxStatusProcessing, staleBefore,
			).
			Order("COALESCE(next_retry_at, created_at) ASC, id ASC").
			Limit(p.batchSize).
			Find(&events).Error; err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		ids := make([]uint64, 0, len(events))
		for _, event := range events {
			ids = append(ids, event.ID)
		}
		return tx.Model(&eventOutbox{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"status":    eventOutboxStatusProcessing,
				"locked_at": now,
				"locked_by": p.workerID,
			}).Error
	})
	return events, err
}

func (p *OutboxPublisher) publishOne(ctx context.Context, event eventOutbox) {
	if p.mqConn == nil {
		p.markRetry(ctx, event, "rabbitmq: not configured")
		return
	}
	publishCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	if err := p.mqConn.Publish(publishCtx, event.RoutingKey, []byte(event.Payload)); err != nil {
		logger.L().Warn("notification outbox publish failed",
			zap.String("event_id", event.EventID),
			zap.String("routing_key", event.RoutingKey),
			zap.Error(err),
		)
		p.markRetry(ctx, event, err.Error())
		return
	}
	if err := p.markPublished(ctx, event.ID); err != nil {
		logger.L().Error("notification outbox mark published failed after MQ publish",
			zap.String("event_id", event.EventID),
			zap.Error(err),
		)
	}
}

func (p *OutboxPublisher) markPublished(ctx context.Context, id uint64) error {
	now := time.Now()
	return p.db.WithContext(ctx).Model(&eventOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        eventOutboxStatusPublished,
			"next_retry_at": nil,
			"last_error":    "",
			"locked_at":     nil,
			"locked_by":     "",
			"published_at":  now,
		}).Error
}

func (p *OutboxPublisher) markRetry(ctx context.Context, event eventOutbox, errMsg string) {
	if event.RetryCount+1 >= maxRetryCount {
		if err := p.markDead(ctx, event.ID, errMsg); err != nil {
			logger.L().Error("notification outbox mark dead-letter error", zap.Error(err))
		}
		return
	}
	backoff := time.Duration(math.Min(
		float64(maxBackoff),
		float64(backoffBase)*math.Pow(2, float64(event.RetryCount)),
	))
	if err := p.markRetryableFailure(ctx, event.ID, errMsg, time.Now().Add(backoff)); err != nil {
		logger.L().Error("notification outbox mark retryable error", zap.Error(err))
	}
}

func (p *OutboxPublisher) markRetryableFailure(ctx context.Context, id uint64, errMsg string, nextRetry time.Time) error {
	return p.db.WithContext(ctx).Model(&eventOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        eventOutboxStatusPending,
			"last_error":    errMsg,
			"retry_count":   gorm.Expr("retry_count + 1"),
			"next_retry_at": nextRetry,
			"locked_at":     nil,
			"locked_by":     "",
		}).Error
}

func (p *OutboxPublisher) markDead(ctx context.Context, id uint64, errMsg string) error {
	now := time.Now()
	return p.db.WithContext(ctx).Model(&eventOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":           eventOutboxStatusDead,
			"last_error":       errMsg,
			"retry_count":      gorm.Expr("retry_count + 1"),
			"next_retry_at":    nil,
			"locked_at":        nil,
			"locked_by":        "",
			"dead_lettered_at": now,
		}).Error
}
