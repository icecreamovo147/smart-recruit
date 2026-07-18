package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenVersionCache struct {
	client *redis.Client
}

func NewTokenVersionCache(client *redis.Client) TokenVersionCache {
	return TokenVersionCache{client: client}
}

func (c TokenVersionCache) SetTokenVersion(ctx context.Context, userID uint64, version int32, ttl time.Duration) error {
	if c.client == nil {
		return nil
	}
	return c.client.Set(ctx, key(userID), version, ttl).Err()
}

func (c TokenVersionCache) DeleteTokenVersion(ctx context.Context, userID uint64) error {
	if c.client == nil {
		return nil
	}
	return c.client.Del(ctx, key(userID)).Err()
}

func key(userID uint64) string {
	return fmt.Sprintf("token_version:%d", userID)
}
