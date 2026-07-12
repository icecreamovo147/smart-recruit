package repository

import (
	"context"
	"testing"
	"time"

	"logic-grpc-service/model"
)

func TestAnalyticsProjectionRepoIdempotentEventAndCheckpoint(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAnalyticsProjectionRepo(db)
	ctx := context.Background()
	event := &model.AnalyticsProjectionEvent{
		ProjectionName: "funnel",
		Source:         "domain_event",
		EventID:        "evt-analytics-1",
		EventType:      "application.status_changed",
		AggregateType:  "application",
		AggregateID:    "42",
		Producer:       "recruitment.application",
		IdempotencyKey: "application:42:application.status_changed:evt-analytics-1",
		Payload:        `{"application_id":42}`,
		Metadata:       `{"source":"test"}`,
		OccurredAt:     time.Date(2026, 7, 11, 8, 0, 0, 0, time.UTC),
		ProjectedAt:    time.Date(2026, 7, 11, 8, 1, 0, 0, time.UTC),
	}

	inserted, err := repo.SaveProjectedEvent(ctx, event)
	if err != nil {
		t.Fatalf("save projected event: %v", err)
	}
	if !inserted {
		t.Fatal("first event insert should report inserted")
	}
	inserted, err = repo.SaveProjectedEvent(ctx, event)
	if err != nil {
		t.Fatalf("save duplicate projected event: %v", err)
	}
	if inserted {
		t.Fatal("duplicate event insert should be idempotently skipped")
	}

	stored, err := repo.GetProjectedEventByEventID(ctx, "evt-analytics-1")
	if err != nil {
		t.Fatalf("get projected event: %v", err)
	}
	if stored == nil || stored.ProjectionName != "funnel" || stored.AggregateID != "42" {
		t.Fatalf("stored event drifted: %+v", stored)
	}

	if err := repo.SaveCheckpoint(ctx, "funnel", "evt-analytics-1"); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	if err := repo.SaveCheckpoint(ctx, "funnel", "evt-analytics-2"); err != nil {
		t.Fatalf("update checkpoint: %v", err)
	}
	cursor, err := repo.GetCheckpoint(ctx, "funnel")
	if err != nil {
		t.Fatalf("get checkpoint: %v", err)
	}
	if cursor != "evt-analytics-2" {
		t.Fatalf("checkpoint cursor=%q, want evt-analytics-2", cursor)
	}
}
