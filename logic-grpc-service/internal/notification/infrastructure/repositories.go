package infrastructure

import (
	"gorm.io/gorm"

	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/repository"
)

// NotificationRepository adapts the current GORM notification repository to Notification application ports.
type NotificationRepository = repository.NotificationRepo

// NotificationCache adapts the current Redis unread-count and realtime stream cache.
type NotificationCache = cache.NotificationCache

// NewNotificationRepository creates a Notification-owned repository adapter.
func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return repository.NewNotificationRepo(db)
}
