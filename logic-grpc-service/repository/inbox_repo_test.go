package repository

import (
	"context"
	"testing"
	"time"

	"logic-grpc-service/model"
)

func TestInboxClaimSkipsProcessedDuplicate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboxRepo(db)
	ctx := context.Background()
	claim := InboxClaim{
		EventID:        "evt-inbox-1",
		EventType:      "notification.create",
		ConsumerName:   "notification-consumer",
		IdempotencyKey: "notification:evt-inbox-1",
	}

	record, claimed, err := repo.Claim(ctx, claim)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if !claimed || record.AttemptCount != 1 {
		t.Fatalf("first claim not recorded: claimed=%v record=%+v", claimed, record)
	}
	if err := repo.MarkProcessed(ctx, record.ID); err != nil {
		t.Fatalf("mark processed: %v", err)
	}

	duplicate, claimed, err := repo.Claim(ctx, claim)
	if err != nil {
		t.Fatalf("claim duplicate: %v", err)
	}
	if claimed {
		t.Fatalf("processed duplicate should be skipped: %+v", duplicate)
	}
}

func TestInboxClaimRetriesFailedRecord(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboxRepo(db)
	ctx := context.Background()
	claim := InboxClaim{
		EventID:      "evt-inbox-retry",
		EventType:    "email.send",
		ConsumerName: "email-consumer",
	}

	record, claimed, err := repo.Claim(ctx, claim)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if !claimed {
		t.Fatal("first claim should be claimed")
	}
	if err := repo.MarkFailed(ctx, record.ID, "temporary failure"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	record, claimed, err = repo.Claim(ctx, claim)
	if err != nil {
		t.Fatalf("reclaim failed record: %v", err)
	}
	if !claimed || record.AttemptCount != 2 || record.Status != model.EventInboxStatusProcessing {
		t.Fatalf("failed record not reclaimed: claimed=%v record=%+v", claimed, record)
	}
}

func TestInboxClaimSkipsDeadLetterDuplicate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboxRepo(db)
	ctx := context.Background()
	claim := InboxClaim{
		EventID:      "evt-inbox-dead",
		EventType:    "email.send",
		ConsumerName: "email-consumer",
	}

	record, claimed, err := repo.Claim(ctx, claim)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if !claimed {
		t.Fatal("first claim should be claimed")
	}
	if err := repo.MarkDead(ctx, record.ID, "poison payload"); err != nil {
		t.Fatalf("mark dead: %v", err)
	}

	record, claimed, err = repo.Claim(ctx, claim)
	if err != nil {
		t.Fatalf("reclaim dead record: %v", err)
	}
	if claimed || record.Status != model.EventInboxStatusDead {
		t.Fatalf("dead record should not be reclaimed: claimed=%v record=%+v", claimed, record)
	}
}

func TestInboxStatsAndRetention(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboxRepo(db)
	ctx := context.Background()
	now := time.Now()
	processedCutoff, deadCutoff := InboxRetentionCutoffs(now)
	oldProcessed := processedCutoff.Add(-time.Second)
	freshProcessed := processedCutoff.Add(time.Second)
	oldDead := deadCutoff.Add(-time.Second)
	freshDead := deadCutoff.Add(time.Second)

	records := []model.EventInbox{
		{EventID: "evt-processing", EventType: "notification.create", ConsumerName: "notification-consumer", Status: model.EventInboxStatusProcessing, ReceivedAt: now},
		{EventID: "evt-failed", EventType: "notification.create", ConsumerName: "notification-consumer", Status: model.EventInboxStatusFailed, ReceivedAt: now},
		{EventID: "evt-old-processed", EventType: "notification.create", ConsumerName: "notification-consumer", Status: model.EventInboxStatusProcessed, ReceivedAt: now, ProcessedAt: &oldProcessed},
		{EventID: "evt-fresh-processed", EventType: "notification.create", ConsumerName: "notification-consumer", Status: model.EventInboxStatusProcessed, ReceivedAt: now, ProcessedAt: &freshProcessed},
		{EventID: "evt-old-dead", EventType: "notification.create", ConsumerName: "notification-consumer", Status: model.EventInboxStatusDead, ReceivedAt: now, DeadLetteredAt: &oldDead},
		{EventID: "evt-fresh-dead", EventType: "notification.create", ConsumerName: "notification-consumer", Status: model.EventInboxStatusDead, ReceivedAt: now, DeadLetteredAt: &freshDead},
	}
	for index := range records {
		if err := db.Create(&records[index]).Error; err != nil {
			t.Fatalf("create inbox record %d: %v", index, err)
		}
	}

	stats, err := repo.Stats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Processing != 1 || stats.Failed != 1 || stats.Processed != 2 || stats.DeadLettered != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	deletedProcessed, err := repo.DeleteProcessedBefore(ctx, processedCutoff)
	if err != nil {
		t.Fatalf("delete processed: %v", err)
	}
	deletedDead, err := repo.DeleteDeadLetteredBefore(ctx, deadCutoff)
	if err != nil {
		t.Fatalf("delete dead: %v", err)
	}
	if deletedProcessed != 1 || deletedDead != 1 {
		t.Fatalf("deleted processed=%d dead=%d, want 1/1", deletedProcessed, deletedDead)
	}
}
