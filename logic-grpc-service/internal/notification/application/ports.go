package application

import (
	"context"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

// NotificationRepository is the Notification-owned persistence and unread-count port.
type NotificationRepository interface {
	Create(ctx context.Context, n *model.Notification) error
	Exists(ctx context.Context, receiverID int64, accountType string, bizType string, bizID int64, notificationType string) (bool, error)
	CreateOnce(ctx context.Context, n *model.Notification) error
	CreateOnceWithResult(ctx context.Context, n *model.Notification) (bool, error)
	CreateOrIgnore(ctx context.Context, n *model.Notification) error
	List(ctx context.Context, receiverID int64, accountType string, page, pageSize int32) ([]model.Notification, int64, error)
	ListCursor(ctx context.Context, receiverID int64, accountType string, cursor string, limit int32) ([]model.Notification, string, bool, error)
	UnreadCount(ctx context.Context, receiverID int64, accountType string) (int64, error)
	Latest(ctx context.Context, receiverID int64, accountType string) (*model.Notification, error)
	MarkRead(ctx context.Context, receiverID int64, accountType string, notificationID int64) (int64, error)
	MarkAllReadBatch(ctx context.Context, receiverID int64, accountType string, limit int) (int64, error)
}

// UnreadAndRealtimeCache is the Notification-owned cache and stream publication port.
type UnreadAndRealtimeCache interface {
	GetUnreadCount(ctx context.Context, receiverID uint64, accountType string) (int64, bool)
	SetUnreadCount(ctx context.Context, receiverID uint64, accountType string, count int64)
	Invalidate(ctx context.Context, receiverID uint64, accountType string)
	PublishNotificationEvent(ctx context.Context, receiverID uint64, accountType string, payload string)
	Close() error
}

// AsyncNotificationWriter is the Notification-owned asynchronous write port.
type AsyncNotificationWriter interface {
	Submit(n *model.Notification)
	SubmitAlways(n *model.Notification)
}

// OutboxEventPublisher is the event and email-coordination publication port used by notification workflows.
type OutboxEventPublisher interface {
	WriteEvent(ctx context.Context, eventType, aggregateType string, aggregateID uint64, routingKey string, payload any) error
	WriteEventTx(tx *gorm.DB, eventType, aggregateType string, aggregateID uint64, routingKey string, payload any) error
	Signal()
}
