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

func (c *NotificationCache) unreadKey(receiverID uint64, accountType string) string {
	return fmt.Sprintf("notif:unread:%d:%s", receiverID, accountType)
}

func NotificationEventChannel(receiverID uint64, accountType string) string {
	return fmt.Sprintf("notif:event:%s:%d", accountType, receiverID)
}

func (c *NotificationCache) GetUnreadCount(ctx context.Context, receiverID uint64, accountType string) (int64, bool) {
	val, err := c.rdb.Get(ctx, c.unreadKey(receiverID, accountType)).Int64()
	if err != nil {
		return 0, false
	}
	return val, true
}

func (c *NotificationCache) SetUnreadCount(ctx context.Context, receiverID uint64, accountType string, count int64) {
	_ = c.rdb.Set(ctx, c.unreadKey(receiverID, accountType), count, c.ttl).Err()
}

func (c *NotificationCache) Invalidate(ctx context.Context, receiverID uint64, accountType string) {
	_ = c.rdb.Del(ctx, c.unreadKey(receiverID, accountType)).Err()
}

func (c *NotificationCache) PublishNotificationEvent(ctx context.Context, receiverID uint64, accountType string, payload string) {
	_ = c.rdb.Publish(ctx, NotificationEventChannel(receiverID, accountType), payload).Err()
}

func (c *NotificationCache) Close() error {
	return c.rdb.Close()
}
