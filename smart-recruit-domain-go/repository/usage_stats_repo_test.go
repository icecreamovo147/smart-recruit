package repository

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
)

func seedUsageTestData(t *testing.T, db *gorm.DB) {
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

func TestGetStatsByModel(t *testing.T) {
	db := setupTestDB(t)
	// Register ThirdPartyUsageLog if not already registered in setupTestDB
	if err := db.AutoMigrate(&model.ThirdPartyUsageLog{}); err != nil {
		t.Fatalf("migrate ThirdPartyUsageLog: %v", err)
	}
	seedUsageTestData(t, db)

	repo := NewUsageStatsRepo(db)
	ctx := context.Background()
	start := time.Now().Add(-72 * time.Hour)
	end := time.Now()

	rows, err := repo.GetStatsByModel(ctx, start, end)
	if err != nil {
		t.Fatalf("GetStatsByModel: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one row")
	}

	// gpt-4 should have 100+200+300 = 600 tokens, 3 calls
	// claude-3 should have 150+50 = 200 tokens, 2 calls
	for _, r := range rows {
		switch r.Name {
		case "gpt-4":
			if r.TotalTokens != 600 {
				t.Errorf("gpt-4 total_tokens: expected 600, got %d", r.TotalTokens)
			}
			if r.CallCount != 3 {
				t.Errorf("gpt-4 call_count: expected 3, got %d", r.CallCount)
			}
		case "claude-3":
			if r.TotalTokens != 200 {
				t.Errorf("claude-3 total_tokens: expected 200, got %d", r.TotalTokens)
			}
			if r.CallCount != 2 {
				t.Errorf("claude-3 call_count: expected 2, got %d", r.CallCount)
			}
		}
	}
}

func TestGetStatsByUser(t *testing.T) {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&model.ThirdPartyUsageLog{}); err != nil {
		t.Fatalf("migrate ThirdPartyUsageLog: %v", err)
	}
	seedUsageTestData(t, db)

	repo := NewUsageStatsRepo(db)
	ctx := context.Background()
	start := time.Now().Add(-72 * time.Hour)
	end := time.Now()

	rows, err := repo.GetStatsByUser(ctx, start, end)
	if err != nil {
		t.Fatalf("GetStatsByUser: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one row")
	}
	if len(rows) > 10 {
		t.Errorf("expected at most 10 rows (top 10), got %d", len(rows))
	}

	// Verify ordering: user 2 (450 tokens) should be first, user 1 (300 tokens) second, user 3 (50 tokens) third
	if len(rows) >= 1 {
		// Users may have different token totals
		for i := 1; i < len(rows); i++ {
			if rows[i-1].TotalTokens < rows[i].TotalTokens {
				t.Errorf("rows not sorted descending by total_tokens: %d (%d) < %d (%d)",
					i-1, rows[i-1].TotalTokens, i, rows[i].TotalTokens)
				break
			}
		}
	}
}

func TestGetStatsBySession(t *testing.T) {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&model.ThirdPartyUsageLog{}); err != nil {
		t.Fatalf("migrate ThirdPartyUsageLog: %v", err)
	}
	seedUsageTestData(t, db)

	repo := NewUsageStatsRepo(db)
	ctx := context.Background()
	start := time.Now().Add(-72 * time.Hour)
	end := time.Now()

	rows, err := repo.GetStatsBySession(ctx, start, end)
	if err != nil {
		t.Fatalf("GetStatsBySession: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one row")
	}
	if len(rows) > 10 {
		t.Errorf("expected at most 10 rows (top 10), got %d", len(rows))
	}
}

func TestGetTrend(t *testing.T) {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&model.ThirdPartyUsageLog{}); err != nil {
		t.Fatalf("migrate ThirdPartyUsageLog: %v", err)
	}
	seedUsageTestData(t, db)

	repo := NewUsageStatsRepo(db)
	ctx := context.Background()
	start := time.Now().Add(-72 * time.Hour)
	end := time.Now()

	rows, err := repo.GetTrend(ctx, start, end, "day")
	if err != nil {
		t.Fatalf("GetTrend: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one trend point")
	}
}

func TestGetSummary(t *testing.T) {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&model.ThirdPartyUsageLog{}); err != nil {
		t.Fatalf("migrate ThirdPartyUsageLog: %v", err)
	}
	seedUsageTestData(t, db)

	repo := NewUsageStatsRepo(db)
	ctx := context.Background()
	start := time.Now().Add(-72 * time.Hour)
	end := time.Now()

	row, err := repo.GetSummary(ctx, start, end)
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if row == nil {
		t.Fatal("expected a summary row")
	}
	// Total tokens: 100+200+150+300+50 = 800
	if row.TotalTokens != 800 {
		t.Errorf("total_tokens: expected 800, got %d", row.TotalTokens)
	}
	// Call count: 5
	if row.CallCount != 5 {
		t.Errorf("call_count: expected 5, got %d", row.CallCount)
	}
}

func TestGetStatsByModel_EmptyRange(t *testing.T) {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&model.ThirdPartyUsageLog{}); err != nil {
		t.Fatalf("migrate ThirdPartyUsageLog: %v", err)
	}

	repo := NewUsageStatsRepo(db)
	ctx := context.Background()
	start := time.Now().Add(-72 * time.Hour)
	end := time.Now()

	rows, err := repo.GetStatsByModel(ctx, start, end)
	if err != nil {
		t.Fatalf("GetStatsByModel empty: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for empty range, got %d", len(rows))
	}
}
