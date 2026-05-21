package redisclient

import (
	"context"

	"github.com/redis/go-redis/v9"

	"web-gin-service/config"
)

func New(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
}

func Ping(ctx context.Context, rdb *redis.Client) error {
	if rdb == nil {
		return redis.Nil
	}
	return rdb.Ping(ctx).Err()
}
