package mq

import (
	"context"
	"encoding/json"
	"testing"

	"gorm.io/gorm"

	"smart-recruit-offer-service/internal/application/port"
	"smart-recruit-offer-service/internal/infrastructure/persistence"
)

func TestBuildEventKeepsLegacyEnvelopeCompatibility(t *testing.T) {
	publisher := NewOutboxPublisher(&fakeOutboxStore{})
	event, err := publisher.buildEvent(port.OutboxMessage{
		EventType:           "offer.notification_requested",
		AggregateType:       "offer",
		AggregateID:         10,
		RoutingKey:          "notification.create",
		ReceiverID:          20,
		ReceiverRole:        1,
		ReceiverAccountType: "candidate",
		Type:                "offer_sent",
		Title:               "Offer 已发送",
		Content:             "请及时查看",
		Link:                "/applications",
		BizType:             "offer",
		BizID:               10,
		JobTitle:            "Backend Engineer",
	})
	if err != nil {
		t.Fatalf("buildEvent returned error: %v", err)
	}
	if event.SchemaVersion != "1.0" || event.Producer == "" || event.IdempotencyKey == "" {
		t.Fatalf("event metadata not standardized: %+v", event)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		t.Fatalf("payload should be json: %v", err)
	}
	if payload["event_id"] != event.EventID {
		t.Fatalf("payload event_id=%v, want %s", payload["event_id"], event.EventID)
	}
	if payload["schema_version"] != "1.0" {
		t.Fatalf("schema_version=%v, want 1.0", payload["schema_version"])
	}
	if payload["idempotency_key"] != event.IdempotencyKey {
		t.Fatalf("payload idempotency_key=%v, want %s", payload["idempotency_key"], event.IdempotencyKey)
	}
	if payload["receiver_id"].(float64) != 20 {
		t.Fatalf("legacy top-level receiver_id missing: %+v", payload)
	}
	nestedPayload, ok := payload["payload"].(map[string]any)
	if !ok {
		t.Fatalf("missing nested payload: %+v", payload)
	}
	if nestedPayload["title"] != "Offer 已发送" || nestedPayload["job_title"] != "Backend Engineer" {
		t.Fatalf("nested payload drifted: %+v", nestedPayload)
	}
}

func TestPublishUsesTransactionFromContext(t *testing.T) {
	store := &fakeOutboxStore{}
	publisher := NewOutboxPublisher(store)
	ctx := persistence.ContextWithTx(context.Background(), &gorm.DB{})
	if err := publisher.Publish(ctx, port.OutboxMessage{
		EventType:     "offer.notification_requested",
		AggregateType: "offer",
		AggregateID:   10,
		RoutingKey:    "notification.create",
		Type:          "offer_created",
	}); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if !store.usedTx {
		t.Fatal("expected Publish to use CreateWithTx when tx is in context")
	}
	if store.created != nil {
		t.Fatalf("non-transactional create should not be used: %+v", store.created)
	}
}

type fakeOutboxStore struct {
	created   *EventOutboxRecord
	createdTx *EventOutboxRecord
	usedTx    bool
}

func (s *fakeOutboxStore) Create(_ context.Context, event *EventOutboxRecord) error {
	s.created = event
	return nil
}

func (s *fakeOutboxStore) CreateWithTx(_ *gorm.DB, event *EventOutboxRecord) error {
	s.usedTx = true
	s.createdTx = event
	return nil
}
