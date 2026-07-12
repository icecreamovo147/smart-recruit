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
	var published model.EventOutbox
	if err := db.First(&published, event.ID).Error; err != nil {
		t.Fatalf("load published event: %v", err)
	}
	if published.PublishedAt == nil {
		t.Fatalf("published event missing published_at: %+v", published)
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

func TestOutboxMarkDeadRecordsDeadLetterTime(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOutboxRepo(db)
	ctx := context.Background()

	event := &model.EventOutbox{
		EventID:       "evt-dead",
		EventType:     "notification.create",
		AggregateType: "application",
		AggregateID:   1,
		RoutingKey:    "notification.create",
		Payload:       `{"event_id":"evt-dead"}`,
		Status:        model.EventOutboxStatusProcessing,
		RetryCount:    9,
	}
	if err := repo.Create(ctx, event); err != nil {
		t.Fatalf("create outbox event: %v", err)
	}

	if err := repo.MarkDead(ctx, event.ID, "poison payload"); err != nil {
		t.Fatalf("mark dead: %v", err)
	}
	var stored model.EventOutbox
	if err := db.First(&stored, event.ID).Error; err != nil {
		t.Fatalf("load dead event: %v", err)
	}
	if stored.Status != model.EventOutboxStatusDead || stored.DeadLetteredAt == nil || stored.NextRetryAt != nil {
		t.Fatalf("dead-letter metadata not recorded: %+v", stored)
	}
	if stored.RetryCount != 10 {
		t.Fatalf("retry_count=%d, want 10", stored.RetryCount)
	}
}

func TestOutboxStats(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOutboxRepo(db)
	ctx := context.Background()
	nextRetry := time.Now().Add(time.Minute)

	events := []model.EventOutbox{
		{EventID: "evt-pending", EventType: "notification.create", AggregateType: "application", RoutingKey: "notification.create", Payload: `{}`, Status: model.EventOutboxStatusPending},
		{EventID: "evt-retrying", EventType: "notification.create", AggregateType: "application", RoutingKey: "notification.create", Payload: `{}`, Status: model.EventOutboxStatusPending, RetryCount: 1, NextRetryAt: &nextRetry},
		{EventID: "evt-processing", EventType: "notification.create", AggregateType: "application", RoutingKey: "notification.create", Payload: `{}`, Status: model.EventOutboxStatusProcessing},
		{EventID: "evt-published", EventType: "notification.create", AggregateType: "application", RoutingKey: "notification.create", Payload: `{}`, Status: model.EventOutboxStatusPublished},
		{EventID: "evt-dead-stats", EventType: "notification.create", AggregateType: "application", RoutingKey: "notification.create", Payload: `{}`, Status: model.EventOutboxStatusDead},
	}
	for index := range events {
		if err := repo.Create(ctx, &events[index]); err != nil {
			t.Fatalf("create event %d: %v", index, err)
		}
	}

	stats, err := repo.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Pending != 1 || stats.Retrying != 1 || stats.Processing != 1 || stats.Published != 1 || stats.DeadLettered != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if stats.OldestPendingAt == nil || stats.OldestRetryAt == nil {
		t.Fatalf("expected backlog timestamps: %+v", stats)
	}
}

func TestOutboxRetentionDeletesOnlyExpiredTerminalEvents(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOutboxRepo(db)
	ctx := context.Background()
	now := time.Now()
	publishedCutoff, deadCutoff := OutboxRetentionCutoffs(now)
	oldPublished := publishedCutoff.Add(-time.Second)
	freshPublished := publishedCutoff.Add(time.Second)
	oldDead := deadCutoff.Add(-time.Second)
	freshDead := deadCutoff.Add(time.Second)

	events := []model.EventOutbox{
		{EventID: "evt-old-published", EventType: "notification.create", AggregateType: "application", RoutingKey: "notification.create", Payload: `{}`, Status: model.EventOutboxStatusPublished, PublishedAt: &oldPublished},
		{EventID: "evt-fresh-published", EventType: "notification.create", AggregateType: "application", RoutingKey: "notification.create", Payload: `{}`, Status: model.EventOutboxStatusPublished, PublishedAt: &freshPublished},
		{EventID: "evt-old-dead", EventType: "notification.create", AggregateType: "application", RoutingKey: "notification.create", Payload: `{}`, Status: model.EventOutboxStatusDead, DeadLetteredAt: &oldDead},
		{EventID: "evt-fresh-dead", EventType: "notification.create", AggregateType: "application", RoutingKey: "notification.create", Payload: `{}`, Status: model.EventOutboxStatusDead, DeadLetteredAt: &freshDead},
		{EventID: "evt-pending-retained", EventType: "notification.create", AggregateType: "application", RoutingKey: "notification.create", Payload: `{}`, Status: model.EventOutboxStatusPending},
	}
	for index := range events {
		if err := repo.Create(ctx, &events[index]); err != nil {
			t.Fatalf("create event %d: %v", index, err)
		}
	}

	deletedPublished, err := repo.DeletePublishedBefore(ctx, publishedCutoff)
	if err != nil {
		t.Fatalf("delete published: %v", err)
	}
	deletedDead, err := repo.DeleteDeadLetteredBefore(ctx, deadCutoff)
	if err != nil {
		t.Fatalf("delete dead-lettered: %v", err)
	}
	if deletedPublished != 1 || deletedDead != 1 {
		t.Fatalf("deleted published=%d dead=%d, want 1/1", deletedPublished, deletedDead)
	}

	var remaining []model.EventOutbox
	if err := db.Order("event_id").Find(&remaining).Error; err != nil {
		t.Fatalf("list remaining: %v", err)
	}
	got := make([]string, 0, len(remaining))
	for _, event := range remaining {
		got = append(got, event.EventID)
	}
	want := []string{"evt-fresh-dead", "evt-fresh-published", "evt-pending-retained"}
	if len(got) != len(want) {
		t.Fatalf("remaining=%v, want %v", got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("remaining=%v, want %v", got, want)
		}
	}
}
