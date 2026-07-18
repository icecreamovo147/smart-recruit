package repository

import (
	"context"

	"smart-recruit-notification-service/internal/domain/model"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *model.Notification) error
	CreateOnceWithResult(ctx context.Context, n *model.Notification) (bool, error)
	List(ctx context.Context, receiverID int64, accountType string, page, pageSize int32) ([]model.Notification, int64, error)
	ListCursor(ctx context.Context, receiverID int64, accountType string, cursor string, limit int32) ([]model.Notification, string, bool, error)
	UnreadCount(ctx context.Context, receiverID int64, accountType string) (int64, error)
	Latest(ctx context.Context, receiverID int64, accountType string) (*model.Notification, error)
	MarkRead(ctx context.Context, receiverID int64, accountType string, notificationID int64) (int64, error)
	MarkAllReadBatch(ctx context.Context, receiverID int64, accountType string, limit int) (int64, error)
}

type EmailLogRepository interface {
	Create(ctx context.Context, log *model.EmailLog) error
	ExistsByEventID(ctx context.Context, eventID string) (bool, error)
}

type InboxClaim struct {
	EventID        string
	EventType      string
	ConsumerName   string
	IdempotencyKey string
}

type InboxRecord struct {
	ID uint64
}

type InboxRepository interface {
	Claim(ctx context.Context, claim InboxClaim) (*InboxRecord, bool, error)
	MarkProcessed(ctx context.Context, id uint64) error
	MarkFailed(ctx context.Context, id uint64, errMsg string) error
}
