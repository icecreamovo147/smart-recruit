package oss

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// PresignCache wraps Redis to cache OSS presigned URLs and upload sessions.
// Presigned URLs are valid for 15 min; we cache for 10 min to be safe.
type PresignCache struct {
	Rdb *redis.Client
	ttl time.Duration
}

func NewPresignCache(addr, password string, db int) *PresignCache {
	return NewPresignCacheWithOptions(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func NewPresignCacheWithOptions(opts *redis.Options) *PresignCache {
	rdb := redis.NewClient(opts)
	return &PresignCache{Rdb: rdb, ttl: 10 * time.Minute}
}

func (c *PresignCache) Get(ctx context.Context, ossKey string) (string, bool) {
	val, err := c.Rdb.Get(ctx, "oss:presign:"+ossKey).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

func (c *PresignCache) Set(ctx context.Context, ossKey, url string) {
	_ = c.Rdb.Set(ctx, "oss:presign:"+ossKey, url, c.ttl).Err()
}

func (c *PresignCache) Close() error {
	return c.Rdb.Close()
}
