package service

import (
	"context"
	"encoding/json"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

func TestBuildOutboxEventInjectsEventIDIntoPayload(t *testing.T) {
	event, err := buildOutboxEvent("notification.create", "application", 1, "notification.create", notificationPayload{
		ReceiverID:   1,
		ReceiverRole: 1,
		Type:         "application_approved",
		Title:        "投递进展更新",
		Content:      "通过筛选",
		BizType:      "application",
		BizID:        10,
	})
	if err != nil {
		t.Fatalf("build outbox event: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		t.Fatalf("payload should be json: %v", err)
	}
	if payload["event_id"] != event.EventID {
		t.Fatalf("payload event_id=%v, want %s", payload["event_id"], event.EventID)
	}
	if payload["schema_version"] != "1.0" {
		t.Fatalf("payload schema_version=%v, want 1.0", payload["schema_version"])
	}
	if payload["idempotency_key"] != event.IdempotencyKey {
		t.Fatalf("payload idempotency_key=%v, want %s", payload["idempotency_key"], event.IdempotencyKey)
	}
	if event.SchemaVersion != "1.0" || event.Producer == "" || event.IdempotencyKey == "" {
		t.Fatalf("event metadata was not standardized: %+v", event)
	}
	nestedPayload, ok := payload["payload"].(map[string]any)
	if !ok {
		t.Fatalf("payload envelope missing nested payload: %+v", payload)
	}
	if nestedPayload["title"] != "投递进展更新" {
		t.Fatalf("nested payload drifted: %+v", nestedPayload)
	}
}

func TestOutboxPublisherMarksDeadAfterRetryBudget(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.EventOutbox{}); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	repo := repository.NewOutboxRepo(db)
	event := &model.EventOutbox{
		EventID:       "evt-max-retry",
		EventType:     "notification.create",
		AggregateType: "application",
		RoutingKey:    "notification.create",
		Payload:       `{}`,
		Status:        model.EventOutboxStatusProcessing,
		RetryCount:    maxRetryCount - 1,
	}
	if err := repo.Create(context.Background(), event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	publisher := NewOutboxPublisher(repo, nil)
	publisher.markRetry(context.Background(), *event, "poison payload")

	var stored model.EventOutbox
	if err := db.First(&stored, event.ID).Error; err != nil {
		t.Fatalf("load event: %v", err)
	}
	if stored.Status != model.EventOutboxStatusDead || stored.DeadLetteredAt == nil {
		t.Fatalf("event not dead-lettered after retry budget: %+v", stored)
	}
}
