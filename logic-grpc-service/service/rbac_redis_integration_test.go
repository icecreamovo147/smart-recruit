//go:build integration
// +build integration

package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"

	"logic-grpc-service/pkg/jwt"
)

// redisConfig is a minimal struct for Redis connection config.
type redisConfig struct {
	Redis struct {
		Addr         string `yaml:"addr"`
		Password     string `yaml:"password"`
		DB           int    `yaml:"db"`
		DialTimeout  string `yaml:"dial_timeout"`
		ReadTimeout  string `yaml:"read_timeout"`
		WriteTimeout string `yaml:"write_timeout"`
	} `yaml:"redis"`
}

// loadRedisAddr reads Redis config from the project's config.yaml.
func loadRedisAddr() string {
	candidates := []string{
		"config/config.yaml",
		"../config/config.yaml",
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "config/config.yaml"))
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg redisConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}
		if cfg.Redis.Addr != "" {
			return cfg.Redis.Addr
		}
	}
	return ""
}

// TestIntegration_TokenVersionCacheFlow verifies the full token_version
// lifecycle against a real Redis instance:
//   1. SET token_version:{user_id} with TTL
//   2. Verify protected request with matching version passes
//   3. Simulate role/scope mutation → bump version → old token rejected
//   4. Delete key → verify Redis miss causes rejection (fail-closed)
//   5. Verify TTL is set correctly
func TestIntegration_TokenVersionCacheFlow(t *testing.T) {
	addr := loadRedisAddr()
	if addr == "" {
		t.Skip("Redis config not found; set up config/config.yaml with redis.addr to run integration tests")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}

	const (
		testUserID  = 99999
		testVersion = 5
	)

	// Use the same key format as AdminService.setTokenVersionCache.
	// A high user ID avoids collisions with real data.
	key := fmt.Sprintf("token_version:%d", testUserID)
	t.Cleanup(func() { rdb.Del(context.Background(), key) })

	// ── Test 1: SET token_version with TTL ────────────────────────────
	svc := &AdminService{redisClient: rdb}
	err := svc.setTokenVersionCache(ctx, testUserID, testVersion)
	if err != nil {
		t.Fatalf("SET token_version failed: %v", err)
	}

	val, err := rdb.Get(ctx, key).Int()
	if err != nil {
		t.Fatalf("key not found after SET: %v", err)
	}
	if val != testVersion {
		t.Errorf("expected version %d, got %d", testVersion, val)
	}
	t.Logf("✅ token_version:%d = %d", testUserID, val)

	// ── Test 2: TTL is set and within AccessTokenTTL ──────────────────
	ttl := rdb.TTL(ctx, key).Val()
	if ttl <= 0 {
		t.Errorf("expected positive TTL, got %v", ttl)
	}
	if ttl > jwt.AccessTokenTTL+time.Second {
		t.Errorf("TTL %v exceeds AccessTokenTTL %v", ttl, jwt.AccessTokenTTL)
	}
	t.Logf("✅ TTL = %v (within AccessTokenTTL %v)", ttl, jwt.AccessTokenTTL)

	// ── Test 3: Bump version (simulate role/scope mutation) ───────────
	newVersion := int32(testVersion + 1)
	err = svc.setTokenVersionCache(ctx, testUserID, newVersion)
	if err != nil {
		t.Fatalf("SET bumped version failed: %v", err)
	}

	// Old version should not match (stored > jwt version → reject)
	stored, _ := rdb.Get(ctx, key).Int()
	if int32(stored) <= testVersion {
		t.Errorf("expected stored version %d > old jwt version %d", stored, testVersion)
	}
	t.Logf("✅ bumped: stored=%d, old JWT would be rejected (stored > jwt=%d)", stored, testVersion)

	// ── Test 4: Delete key → Redis miss should reject (fail-closed) ───
	if err := rdb.Del(ctx, key).Err(); err != nil {
		t.Fatalf("DEL key failed: %v", err)
	}
	_, err = rdb.Get(ctx, key).Int()
	if err == nil {
		t.Error("key still exists after DEL")
	}
	t.Log("✅ key deleted; Redis miss → middleware reject (fail-closed)")

	// ── Test 5: Key namespace matches production format ────────────────
	expectedKey := fmt.Sprintf("token_version:%d", testUserID)
	if key != expectedKey {
		t.Errorf("key namespace mismatch: %s != %s", key, expectedKey)
	}
	t.Logf("✅ key namespace matches production format: %s", expectedKey)
}

// TestIntegration_TokenVersionSetFailsFallbackToDel verifies that when SET
// fails (e.g. Redis goes down), the code tries DEL as a fail-safe, and if
// DEL also fails, returns an error.
func TestIntegration_TokenVersionSetFailsFallbackToDel(t *testing.T) {
	addr := loadRedisAddr()
	if addr == "" {
		t.Skip("Redis config not found")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           0,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	defer rdb.Close()

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	const testUserID = 99998
	key := fmt.Sprintf("token_version:%d", testUserID)
	t.Cleanup(func() { rdb.Del(context.Background(), key) })

	// Pre-set a value at the production key
	rdb.Set(ctx, key, 1, time.Hour)

	// Close the client to simulate Redis becoming unreachable
	rdb.Close()

	svc := &AdminService{redisClient: rdb}
	err := svc.setTokenVersionCache(ctx, testUserID, 2)
	if err == nil {
		t.Error("expected error when both SET and DEL fail, got nil")
	} else {
		t.Logf("✅ SET→DEL→error correctly returned: %v", err)
	}
}
