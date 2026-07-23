package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"smart-recruit-commons/oss"
	"smart-recruit-platform-go/config"
	logicconfig "smart-recruit-platform-go/serviceconfig"
)

func TestRecruitmentNacosStaticFallback(t *testing.T) {
	bootstrap := config.Bootstrap{ServiceName: "recruitment-service", ServiceEnv: "local", ServiceVersion: "test", StaticFallback: true}
	instance, err := instanceFromAddr("127.0.0.1:50062", bootstrap)
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	if instance.ServiceName != "recruitment" {
		t.Fatalf("ServiceName = %q, want recruitment", instance.ServiceName)
	}
	discovery, err := setupNacos(context.Background(), bootstrap, instance)
	if err != nil {
		t.Fatalf("setupNacos returned error: %v", err)
	}
	instances, err := discovery.Resolve(context.Background(), "recruitment")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 50062 {
		t.Fatalf("unexpected instances: %#v", instances)
	}
}

func TestAttachOSSPresignCache(t *testing.T) {
	storage := &recordingStorage{}
	cfg := logicconfig.Config{}
	cfg.Redis.Addr = "127.0.0.1:6379"
	cfg.Redis.PoolSize = 1

	cache, err := attachOSSPresignCache(storage, cfg)
	if err != nil {
		t.Fatalf("attachOSSPresignCache returned error: %v", err)
	}
	defer cache.Close()

	if cache == nil {
		t.Fatal("cache is nil")
	}
	if storage.cache != cache {
		t.Fatalf("storage cache = %#v, want returned cache %#v", storage.cache, cache)
	}
	if got := cache.Rdb.Options().Addr; got != cfg.Redis.Addr {
		t.Fatalf("cache redis addr = %q, want %q", got, cfg.Redis.Addr)
	}
}

func TestAttachOSSPresignCacheRequiresRedisAddr(t *testing.T) {
	_, err := attachOSSPresignCache(&recordingStorage{}, logicconfig.Config{})
	if err == nil || !strings.Contains(err.Error(), "redis addr is required") {
		t.Fatalf("attachOSSPresignCache error = %v, want redis addr required", err)
	}
}

type recordingStorage struct {
	cache *oss.PresignCache
}

func (s *recordingStorage) SetPresignCache(cache *oss.PresignCache) {
	s.cache = cache
}

func (s *recordingStorage) ProviderName() string { return "recording" }

func (s *recordingStorage) GeneratePresignedPutURL(string, string) (string, time.Time, error) {
	return "", time.Time{}, nil
}

func (s *recordingStorage) GeneratePresignedGetURL(string) (string, error) { return "", nil }

func (s *recordingStorage) VerifyObject(context.Context, string) error { return nil }

func (s *recordingStorage) VerifyObjectSize(context.Context, string, int64) error { return nil }

func (s *recordingStorage) DownloadObject(context.Context, string) ([]byte, error) { return nil, nil }

func (s *recordingStorage) CopyObject(context.Context, string, string) error { return nil }

func (s *recordingStorage) DeleteObject(context.Context, string) error { return nil }

func (s *recordingStorage) SavePresignSession(context.Context, oss.PresignSession) (string, error) {
	return "", nil
}

func (s *recordingStorage) SavePresignSessionWithID(context.Context, string, oss.PresignSession) error {
	return nil
}

func (s *recordingStorage) GetAndDeletePresignSession(context.Context, string) (*oss.PresignSession, error) {
	return nil, nil
}
