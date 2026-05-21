package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NotificationCache wraps Redis to cache notification unread counts.
// Invalidation happens on Create / MarkRead / MarkAllRead.
type NotificationCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewNotificationCache(addr, password string, db int) *NotificationCache {
	return NewNotificationCacheWithOptions(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func NewNotificationCacheWithOptions(opts *redis.Options) *NotificationCache {
	rdb := redis.NewClient(opts)
	return &NotificationCache{rdb: rdb, ttl: 120 * time.Second}
}

func (c *NotificationCache) unreadKey(receiverID uint64, receiverRole int32) string {
	return fmt.Sprintf("notif:unread:%d:%d", receiverID, receiverRole)
}

func NotificationEventChannel(receiverID uint64, receiverRole int32) string {
	return fmt.Sprintf("notif:event:%d:%d", receiverRole, receiverID)
}

func (c *NotificationCache) GetUnreadCount(ctx context.Context, receiverID uint64, receiverRole int32) (int64, bool) {
	val, err := c.rdb.Get(ctx, c.unreadKey(receiverID, receiverRole)).Int64()
	if err != nil {
		return 0, false
	}
	return val, true
}

func (c *NotificationCache) SetUnreadCount(ctx context.Context, receiverID uint64, receiverRole int32, count int64) {
	_ = c.rdb.Set(ctx, c.unreadKey(receiverID, receiverRole), count, c.ttl).Err()
}

func (c *NotificationCache) Invalidate(ctx context.Context, receiverID uint64, receiverRole int32) {
	_ = c.rdb.Del(ctx, c.unreadKey(receiverID, receiverRole)).Err()
}

func (c *NotificationCache) PublishNotificationEvent(ctx context.Context, receiverID uint64, receiverRole int32, payload string) {
	_ = c.rdb.Publish(ctx, NotificationEventChannel(receiverID, receiverRole), payload).Err()
}

func (c *NotificationCache) Close() error {
	return c.rdb.Close()
}
