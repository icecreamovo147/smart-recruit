package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/repository"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

// setupUsageStatsTestDB creates an in-memory SQLite DB with tables needed for usage stats tests.
func setupUsageStatsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	tables := []interface{}{
		&model.ThirdPartyUsageLog{},
	}
	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			t.Fatalf("migrate %T: %v", table, err)
		}
	}
	return db
}

// seedUsageStatsData inserts sample usage log entries.
func seedUsageStatsData(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()
	logs := []model.ThirdPartyUsageLog{
		{UserID: 1, Model: "gpt-4", EstimatedTokens: 100, CostMs: 500, RequestID: "req-001", CreatedAt: now.Add(-48 * time.Hour)},
		{UserID: 1, Model: "gpt-4", EstimatedTokens: 200, CostMs: 600, RequestID: "req-001", CreatedAt: now.Add(-47 * time.Hour)},
		{UserID: 2, Model: "claude-3", EstimatedTokens: 150, CostMs: 400, RequestID: "req-002", CreatedAt: now.Add(-24 * time.Hour)},
		{UserID: 2, Model: "gpt-4", EstimatedTokens: 300, CostMs: 700, RequestID: "req-003", CreatedAt: now.Add(-23 * time.Hour)},
		{UserID: 3, Model: "claude-3", EstimatedTokens: 50, CostMs: 200, RequestID: "req-004", CreatedAt: now.Add(-1 * time.Hour)},
	}
	for i := range logs {
		if err := db.Create(&logs[i]).Error; err != nil {
			t.Fatalf("insert usage log %d: %v", i, err)
		}
	}
}

func TestUsageStatsService_GetUsageStats_ByModel(t *testing.T) {
	db := setupUsageStatsTestDB(t)
	seedUsageStatsData(t, db)

	svc := NewUsageStatsService(repository.NewUsageStatsRepo(db), NewServiceAuthorizer(nil, nil))
	ctx := metadata.WithAuthActor(context.Background(), 1, "staff")

	resp, err := svc.GetUsageStats(ctx, &pb.GetUsageStatsRequest{
		StartTime: time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		EndTime:   time.Now().Format(time.RFC3339),
		Dimension: "model",
	})
	if err != nil {
		t.Fatalf("GetUsageStats: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK code, got %d: %s", resp.Code, resp.Msg)
	}
	if len(resp.List) == 0 {
		t.Fatal("expected at least one item")
	}

	for _, item := range resp.List {
		switch item.Name {
		case "gpt-4":
			if item.TotalTokens != 600 {
				t.Errorf("gpt-4 total_tokens: expected 600, got %d", item.TotalTokens)
			}
			if item.CallCount != 3 {
				t.Errorf("gpt-4 call_count: expected 3, got %d", item.CallCount)
			}
		case "claude-3":
			if item.TotalTokens != 200 {
				t.Errorf("claude-3 total_tokens: expected 200, got %d", item.TotalTokens)
			}
			if item.CallCount != 2 {
				t.Errorf("claude-3 call_count: expected 2, got %d", item.CallCount)
			}
		}
	}
}

func TestUsageStatsService_GetUsageStats_ByUser(t *testing.T) {
	db := setupUsageStatsTestDB(t)
	seedUsageStatsData(t, db)

	svc := NewUsageStatsService(repository.NewUsageStatsRepo(db), NewServiceAuthorizer(nil, nil))
	ctx := metadata.WithAuthActor(context.Background(), 1, "staff")

	resp, err := svc.GetUsageStats(ctx, &pb.GetUsageStatsRequest{
		StartTime: time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		EndTime:   time.Now().Format(time.RFC3339),
		Dimension: "user",
	})
	if err != nil {
		t.Fatalf("GetUsageStats(user): %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK code, got %d: %s", resp.Code, resp.Msg)
	}
	if len(resp.List) == 0 {
		t.Fatal("expected at least one item")
	}
}

func TestUsageStatsService_GetUsageStats_BySession(t *testing.T) {
	db := setupUsageStatsTestDB(t)
	seedUsageStatsData(t, db)

	svc := NewUsageStatsService(repository.NewUsageStatsRepo(db), NewServiceAuthorizer(nil, nil))
	ctx := metadata.WithAuthActor(context.Background(), 1, "staff")

	resp, err := svc.GetUsageStats(ctx, &pb.GetUsageStatsRequest{
		StartTime: time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		EndTime:   time.Now().Format(time.RFC3339),
		Dimension: "session",
	})
	if err != nil {
		t.Fatalf("GetUsageStats(session): %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK code, got %d: %s", resp.Code, resp.Msg)
	}
	if len(resp.List) == 0 {
		t.Fatal("expected at least one item")
	}
}

func TestUsageStatsService_GetUsageStats_DefaultDimension(t *testing.T) {
	db := setupUsageStatsTestDB(t)
	seedUsageStatsData(t, db)

	svc := NewUsageStatsService(repository.NewUsageStatsRepo(db), NewServiceAuthorizer(nil, nil))
	ctx := metadata.WithAuthActor(context.Background(), 1, "staff")

	// Empty dimension should default to "model"
	resp, err := svc.GetUsageStats(ctx, &pb.GetUsageStatsRequest{
		StartTime: time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		EndTime:   time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("GetUsageStats(default): %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK code, got %d: %s", resp.Code, resp.Msg)
	}
	if len(resp.List) == 0 {
		t.Fatal("expected at least one item")
	}
}

func TestUsageStatsService_GetUsageTrend_Day(t *testing.T) {
	db := setupUsageStatsTestDB(t)
	seedUsageStatsData(t, db)

	svc := NewUsageStatsService(repository.NewUsageStatsRepo(db), NewServiceAuthorizer(nil, nil))
	ctx := metadata.WithAuthActor(context.Background(), 1, "staff")

	resp, err := svc.GetUsageTrend(ctx, &pb.GetUsageTrendRequest{
		StartTime:   time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		EndTime:     time.Now().Format(time.RFC3339),
		Granularity: "day",
	})
	if err != nil {
		t.Fatalf("GetUsageTrend: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK code, got %d: %s", resp.Code, resp.Msg)
	}
	if len(resp.List) == 0 {
		t.Fatal("expected at least one trend point")
	}
}

func TestUsageStatsService_GetUsageTrend_DefaultGranularity(t *testing.T) {
	db := setupUsageStatsTestDB(t)
	seedUsageStatsData(t, db)

	svc := NewUsageStatsService(repository.NewUsageStatsRepo(db), NewServiceAuthorizer(nil, nil))
	ctx := metadata.WithAuthActor(context.Background(), 1, "staff")

	resp, err := svc.GetUsageTrend(ctx, &pb.GetUsageTrendRequest{
		StartTime: time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		EndTime:   time.Now().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("GetUsageTrend(default): %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK code, got %d: %s", resp.Code, resp.Msg)
	}
	if len(resp.List) == 0 {
		t.Fatal("expected at least one trend point")
	}
}

func TestUsageStatsService_GetUsageStats_EmptyRange(t *testing.T) {
	db := setupUsageStatsTestDB(t)
	// No seed data

	svc := NewUsageStatsService(repository.NewUsageStatsRepo(db), NewServiceAuthorizer(nil, nil))
	ctx := metadata.WithAuthActor(context.Background(), 1, "staff")

	resp, err := svc.GetUsageStats(ctx, &pb.GetUsageStatsRequest{
		StartTime: time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
		EndTime:   time.Now().Format(time.RFC3339),
		Dimension: "model",
	})
	if err != nil {
		t.Fatalf("GetUsageStats(empty): %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK code, got %d: %s", resp.Code, resp.Msg)
	}
	if len(resp.List) != 0 {
		t.Errorf("expected 0 items for empty range, got %d", len(resp.List))
	}
}

func TestUsageStatsService_GetUsageStats_BadTimeRange(t *testing.T) {
	svc := NewUsageStatsService(nil, NewServiceAuthorizer(nil, nil))
	ctx := metadata.WithAuthActor(context.Background(), 1, "staff")

	resp, err := svc.GetUsageStats(ctx, &pb.GetUsageStatsRequest{
		StartTime: "invalid-date",
		EndTime:   "also-invalid",
	})
	if err != nil {
		t.Fatalf("GetUsageStats(bad time): %v", err)
	}
	if resp.Code != errs.ErrBadRequest {
		t.Errorf("expected ErrBadRequest code, got %d: %s", resp.Code, resp.Msg)
	}
}

func TestParseTimeRange(t *testing.T) {
	now := time.Now()

	// Default look-back
	start, end, err := parseTimeRange("", "", 30)
	if err != nil {
		t.Fatalf("parseTimeRange empty: %v", err)
	}
	if !end.After(start) {
		t.Error("expected end after start for default range")
	}
	if end.Sub(start) > 31*24*time.Hour {
		t.Errorf("expected ~30 day range, got %v", end.Sub(start))
	}

	// Custom range
	customStart := now.Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	customEnd := now.Format(time.RFC3339)
	start, end, err = parseTimeRange(customStart, customEnd, 30)
	if err != nil {
		t.Fatalf("parseTimeRange custom: %v", err)
	}
	if !end.After(start) {
		t.Error("expected end after start for custom range")
	}

	// Invalid start time
	_, _, err = parseTimeRange("bad-date", "", 30)
	if err == nil {
		t.Error("expected error for invalid start time")
	}

	// Invalid end time
	_, _, err = parseTimeRange("", "bad-date", 30)
	if err == nil {
		t.Error("expected error for invalid end time")
	}
}

func TestUsageStatsService_NoAuthContext(t *testing.T) {
	svc := NewUsageStatsService(nil, NewServiceAuthorizer(nil, nil))
	ctx := context.Background() // No auth metadata

	resp, err := svc.GetUsageStats(ctx, &pb.GetUsageStatsRequest{
		Dimension: "model",
	})
	if err != nil {
		t.Fatalf("GetUsageStats(no auth): %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden when no auth context, got %d: %s", resp.Code, resp.Msg)
	}
}
