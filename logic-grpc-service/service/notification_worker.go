package service

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

// NotificationWorkerPool throttles concurrent async notification writes.
// Instead of unbounded goroutines, submissions block on a buffered channel.
type NotificationWorkerPool struct {
	repo  *repository.NotificationRepo
	cache *cache.NotificationCache
	sem   chan struct{}
}

func NewNotificationWorkerPool(repo *repository.NotificationRepo, c *cache.NotificationCache, maxConcurrent int) *NotificationWorkerPool {
	return &NotificationWorkerPool{
		repo:  repo,
		cache: c,
		sem:   make(chan struct{}, maxConcurrent),
	}
}

// Submit enqueues an idempotent notification write. If the pool is at capacity,
// it drops the notification and logs a warning (non-blocking fallback).
func (p *NotificationWorkerPool) Submit(n *model.Notification) {
	p.submit(n, true)
}

// SubmitAlways enqueues a notification write that always creates a new record.
func (p *NotificationWorkerPool) SubmitAlways(n *model.Notification) {
	p.submit(n, false)
}

// notifAccountType derives the account_type from the notification record.
// Requires ReceiverAccountType to be set by the caller (FR-19).
// Falls back to "candidate" only when the field is unexpectedly empty.
func notifAccountType(n *model.Notification) string {
	if n.ReceiverAccountType != "" {
		return n.ReceiverAccountType
	}
	logger.L().Warn("notification worker: missing receiver_account_type, defaulting to candidate",
		zap.Int64("receiver_id", n.ReceiverID))
	return "candidate"
}

func (p *NotificationWorkerPool) submit(n *model.Notification, createOnce bool) {
	logger.L().Info("notification submit start",
		zap.String("type", n.Type),
		zap.Int64("receiver_id", n.ReceiverID),
		zap.Int32("receiver_role", n.ReceiverRole),
		zap.Bool("create_once", createOnce),
	)
	select {
	case p.sem <- struct{}{}:
		go p.doWrite(n, createOnce)
	default:
		logger.L().Warn("notification worker pool full, dropping notification",
			zap.String("type", n.Type),
			zap.Int64("receiver_id", n.ReceiverID),
			zap.Int32("receiver_role", n.ReceiverRole),
		)
	}
}

func (p *NotificationWorkerPool) doWrite(n *model.Notification, createOnce bool) {
	defer func() {
		if r := recover(); r != nil {
			logger.L().Error("notification worker panic recovered", zap.Any("panic", r))
		}
		<-p.sem
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var err error
	created := true
	if createOnce {
		created, err = p.repo.CreateOnceWithResult(ctx, n)
	} else {
		err = p.repo.Create(ctx, n)
	}
	if err != nil {
		logger.L().Error("notification worker write FAILED",
			zap.String("type", n.Type),
			zap.Int64("receiver_id", n.ReceiverID),
			zap.String("title", n.Title),
			zap.Error(err),
		)
		return
	}
	if !created {
		return
	}
	logger.L().Info("notification created successfully",
		zap.String("type", n.Type),
		zap.Int64("receiver_id", n.ReceiverID),
		zap.String("title", n.Title),
		zap.Bool("create_once", createOnce),
	)
	// Invalidate cached unread count and publish event.
	if p.cache != nil {
		acctType := notifAccountType(n)
		p.cache.Invalidate(ctx, uint64(n.ReceiverID), acctType)
		p.publishCreatedEvent(ctx, n, acctType)
	}
}

func (p *NotificationWorkerPool) publishCreatedEvent(ctx context.Context, n *model.Notification, acctType string) {
	count, err := p.repo.UnreadCount(ctx, n.ReceiverID, acctType)
	if err != nil {
		logger.L().Warn("notification worker event unread count failed", zap.Error(err))
	}
	if p.cache != nil {
		p.cache.SetUnreadCount(ctx, uint64(n.ReceiverID), acctType, count)
	}
	payload, err := json.Marshal(notificationEvent{
		Type:             "notification_created",
		NotificationType: n.Type,
		NotificationID:   n.ID,
		Unread:           count,
		Title:            n.Title,
		Content:          n.Content,
		Link:             n.Link,
		CreatedAt:        n.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
	})
	if err != nil {
		logger.L().Warn("notification worker event marshal failed", zap.Error(err))
		return
	}
	p.cache.PublishNotificationEvent(ctx, uint64(n.ReceiverID), acctType, string(payload))
}
