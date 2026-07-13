package cache

import (
	"context"

	sharedcache "smart-recruit-commons/pkg/cache"
)

type NotificationCache struct {
	cache *sharedcache.NotificationCache
}

func NewNotificationCache(cache *sharedcache.NotificationCache) *NotificationCache {
	return &NotificationCache{cache: cache}
}

func (c *NotificationCache) GetUnreadCount(ctx context.Context, userID uint64, accountType string) (int64, bool) {
	if c == nil || c.cache == nil {
		return 0, false
	}
	return c.cache.GetUnreadCount(ctx, userID, accountType)
}

func (c *NotificationCache) SetUnreadCount(ctx context.Context, userID uint64, accountType string, count int64) {
	if c == nil || c.cache == nil {
		return
	}
	c.cache.SetUnreadCount(ctx, userID, accountType, count)
}

func (c *NotificationCache) Invalidate(ctx context.Context, userID uint64, accountType string) {
	if c == nil || c.cache == nil {
		return
	}
	c.cache.Invalidate(ctx, userID, accountType)
}

func (c *NotificationCache) PublishNotificationEvent(ctx context.Context, userID uint64, accountType string, payload string) error {
	if c == nil || c.cache == nil {
		return nil
	}
	c.cache.PublishNotificationEvent(ctx, userID, accountType, payload)
	return nil
}
