package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const jobCacheKeyPrefix = "jobs:public:first_page"
const jobCacheTTL = 60 * time.Second

type JobCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewJobCache(addr, password string, db int) *JobCache {
	return NewJobCacheWithOptions(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

func NewJobCacheWithOptions(opts *redis.Options) *JobCache {
	return &JobCache{rdb: redis.NewClient(opts), ttl: jobCacheTTL}
}

func publicFirstPageKey(pageSize int32) string {
	if pageSize <= 0 {
		pageSize = 10
	}
	return fmt.Sprintf("%s:%d", jobCacheKeyPrefix, pageSize)
}

func (c *JobCache) GetPublicFirstPage(ctx context.Context, pageSize int32) ([]byte, bool) {
	b, err := c.rdb.Get(ctx, publicFirstPageKey(pageSize)).Bytes()
	if err != nil {
		return nil, false
	}
	return b, true
}

func (c *JobCache) SetPublicFirstPage(ctx context.Context, pageSize int32, data []byte) {
	_ = c.rdb.Set(ctx, publicFirstPageKey(pageSize), data, c.ttl).Err()
}

func (c *JobCache) InvalidatePublicFirstPage(ctx context.Context) {
	_ = c.rdb.Del(ctx, jobCacheKeyPrefix).Err()
	iter := c.rdb.Scan(ctx, 0, jobCacheKeyPrefix+":*", 100).Iterator()
	keys := make([]string, 0)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		_ = c.rdb.Del(ctx, keys...).Err()
	}
}

func (c *JobCache) Close() error {
	return c.rdb.Close()
}
