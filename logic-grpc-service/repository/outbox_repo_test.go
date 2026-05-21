package repository

import (
	"context"
	"testing"
	"time"

	"logic-grpc-service/model"
)

func TestOutboxClaimRetryAndPublish(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOutboxRepo(db)
	ctx := context.Background()

	event := &model.EventOutbox{
		EventID:       "evt-claim-retry",
		EventType:     "notification.create",
		AggregateType: "application",
		AggregateID:   1,
		RoutingKey:    "notification.create",
		Payload:       `{"event_id":"evt-claim-retry"}`,
		Status:        model.EventOutboxStatusPending,
	}
	if err := repo.Create(ctx, event); err != nil {
		t.Fatalf("create outbox event: %v", err)
	}

	claimed, err := repo.ClaimPending(ctx, 10, "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("claim pending: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != event.ID {
		t.Fatalf("expected to claim one event %d, got %+v", event.ID, claimed)
	}

	var stored model.EventOutbox
	if err := db.First(&stored, event.ID).Error; err != nil {
		t.Fatalf("load claimed event: %v", err)
	}
	if stored.Status != model.EventOutboxStatusProcessing || stored.LockedBy != "worker-a" || stored.LockedAt == nil {
		t.Fatalf("event was not marked processing: %+v", stored)
	}

	if err := repo.MarkRetryableFailure(ctx, event.ID, "temporary failure", time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("mark retryable failure: %v", err)
	}
	claimed, err = repo.ClaimPending(ctx, 10, "worker-b", time.Minute)
	if err != nil {
		t.Fatalf("reclaim pending: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != event.ID {
		t.Fatalf("expected retryable event to be claimed again, got %+v", claimed)
	}

	if err := repo.MarkPublished(ctx, event.ID); err != nil {
		t.Fatalf("mark published: %v", err)
	}
	claimed, err = repo.ClaimPending(ctx, 10, "worker-c", time.Minute)
	if err != nil {
		t.Fatalf("claim after publish: %v", err)
	}
	if len(claimed) != 0 {
		t.Fatalf("published event should not be claimed again: %+v", claimed)
	}
}

func TestOutboxClaimStaleProcessing(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOutboxRepo(db)
	ctx := context.Background()
	lockedAt := time.Now().Add(-time.Hour)

	event := &model.EventOutbox{
		EventID:       "evt-stale-processing",
		EventType:     "resume.parse",
		AggregateType: "resume",
		AggregateID:   1,
		RoutingKey:    "resume.parse",
		Payload:       `{"event_id":"evt-stale-processing"}`,
		Status:        model.EventOutboxStatusProcessing,
		LockedAt:      &lockedAt,
		LockedBy:      "dead-worker",
	}
	if err := repo.Create(ctx, event); err != nil {
		t.Fatalf("create outbox event: %v", err)
	}

	claimed, err := repo.ClaimPending(ctx, 10, "worker-new", time.Minute)
	if err != nil {
		t.Fatalf("claim stale processing: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != event.ID {
		t.Fatalf("expected stale processing event to be claimed, got %+v", claimed)
	}
}
