package projection

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-analytics-service/internal/domain/model"
)

func TestStoreSavesProjectionEventAndCheckpoint(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&projectionEventRow{}, &projectionCheckpointRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewStore(db)
	now := time.Date(2026, 7, 13, 10, 50, 0, 0, time.UTC)

	err = store.SaveProjectionEvent(context.Background(), model.ProjectionEvent{
		EventID:       "evt-1",
		EventType:     "application.status_changed",
		AggregateType: "application",
		AggregateID:   "42",
		OccurredAt:    now,
		Payload:       json.RawMessage(`{"status":"hired"}`),
	})
	if err != nil {
		t.Fatalf("SaveProjectionEvent returned %v", err)
	}
	if err := store.SaveProjectionEvent(context.Background(), model.ProjectionEvent{EventID: "evt-1"}); err != nil {
		t.Fatalf("duplicate SaveProjectionEvent returned %v", err)
	}
	var events int64
	if err := db.Table("analytics_projection_events").Count(&events).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if events != 1 {
		t.Fatalf("events = %d, want 1", events)
	}

	err = store.SaveCheckpoint(context.Background(), model.ProjectionCheckpoint{
		ProjectionName: "analytics-reporting",
		Cursor:         "cursor-1",
		UpdatedAt:      now,
	})
	if err != nil {
		t.Fatalf("SaveCheckpoint returned %v", err)
	}
	err = store.SaveCheckpoint(context.Background(), model.ProjectionCheckpoint{
		ProjectionName: "analytics-reporting",
		Cursor:         "cursor-2",
		UpdatedAt:      now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("second SaveCheckpoint returned %v", err)
	}
	var checkpoint projectionCheckpointRow
	if err := db.First(&checkpoint, "projection_name = ?", "analytics-reporting").Error; err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint.Cursor != "cursor-2" {
		t.Fatalf("cursor = %q, want cursor-2", checkpoint.Cursor)
	}
}
