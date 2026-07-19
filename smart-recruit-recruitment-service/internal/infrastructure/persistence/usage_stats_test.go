package persistence

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-proto/recruitment/pb"
)

func TestUsageDimensionIncludesProviderAndServiceType(t *testing.T) {
	if usageDimension("provider") != "provider" {
		t.Fatalf("provider dimension = %q", usageDimension("provider"))
	}
	if usageDimension("service_type") != "service_type" {
		t.Fatalf("service_type dimension = %q", usageDimension("service_type"))
	}
	if usageDimension("model") != "model" {
		t.Fatalf("model dimension = %q", usageDimension("model"))
	}
}

func TestGetUsageStatsSummaryAndProviderNames(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:usage_stats?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&usageLogRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec(`CREATE TABLE ai_usage_events (
		id INTEGER PRIMARY KEY, owner_type TEXT, owner_id INTEGER, user_id INTEGER,
		operation TEXT, provider_key TEXT, model_key TEXT, provider_request_id TEXT,
		supplier_cost_micros INTEGER NOT NULL DEFAULT 0, occurred_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("migrate billing usage: %v", err)
	}
	now := time.Now()
	seeds := []usageLogRecord{
		{UserID: 1, ServiceType: "ai_chat", Provider: "dashscope", Model: "qwen", EstimatedTokens: 100, Status: "ok", CostMs: 10, CreatedAt: now},
		{UserID: 1, ServiceType: "ai_chat", Provider: "dashscope", Model: "qwen", EstimatedTokens: 50, Status: "error", CostMs: 20, CreatedAt: now},
		{UserID: 2, ServiceType: "oss_presign", Provider: "tencent_cos", Model: "", EstimatedTokens: 0, Status: "ok", CostMs: 5, CreatedAt: now},
	}
	if err := db.Create(&seeds).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	adapter := &usageStatsAdapter{nativeStore: &nativeStore{db: db}}
	resp, err := adapter.GetUsageStats(context.Background(), &pb.GetUsageStatsRequest{Dimension: "provider"})
	if err != nil {
		t.Fatalf("GetUsageStats: %v", err)
	}
	if resp.GetSummary() == nil || resp.GetSummary().GetCallCount() != 3 {
		t.Fatalf("summary = %#v, want call_count=3", resp.GetSummary())
	}
	if resp.GetSummary().GetSuccessCount() != 2 || resp.GetSummary().GetFailedCount() != 1 {
		t.Fatalf("success/failed = %d/%d", resp.GetSummary().GetSuccessCount(), resp.GetSummary().GetFailedCount())
	}
	if len(resp.GetList()) != 2 {
		t.Fatalf("list len = %d, want 2 providers", len(resp.GetList()))
	}
	names := map[string]bool{}
	for _, item := range resp.GetList() {
		names[item.GetName()] = true
	}
	if !names["dashscope"] || !names["tencent_cos"] {
		t.Fatalf("provider names = %#v, want dashscope and tencent_cos", names)
	}
}

func TestParseOptionalTimeAcceptsISOWithMillis(t *testing.T) {
	ts, err := parseOptionalTime("2026-06-17T10:00:00.000Z")
	if err != nil || ts == nil {
		t.Fatalf("parse millis ISO: ts=%v err=%v", ts, err)
	}
}
