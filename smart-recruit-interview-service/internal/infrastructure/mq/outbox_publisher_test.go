package mq

import (
	"context"
	"encoding/json"
	"testing"

	"gorm.io/gorm"

	sharedmodel "smart-recruit-domain-go/model"
	"smart-recruit-interview-service/internal/application/port"
)

func TestOutboxPublisherWritesLegacyCompatibleEnvelope(t *testing.T) {
	store := &fakeOutboxStore{}
	publisher := NewOutboxPublisher(store)
	err := publisher.Publish(context.Background(), port.OutboxMessage{
		EventType:           "interview.email_requested",
		AggregateType:       "interview",
		AggregateID:         10,
		RoutingKey:          "email.send",
		ReceiverID:          20,
		ReceiverAccountType: "candidate",
		Type:                "interview_scheduled",
		Title:               "面试安排通知",
		Content:             "content",
		Link:                "/applications",
		BizType:             "interview",
		BizID:               10,
		JobTitle:            "Backend Engineer",
		RecipientName:       "Candidate A",
		InterviewDate:       "2026-07-13T10:00:00Z",
		InterviewMode:       "视频面试",
		InterviewLink:       "https://meet.example.com",
	})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if len(store.created) != 1 {
		t.Fatalf("created events=%d, want 1", len(store.created))
	}
	event := store.created[0]
	if event.SchemaVersion != envelopeSchemaVersion || event.Producer != legacyOutboxProducer || event.RoutingKey != "email.send" {
		t.Fatalf("unexpected outbox event: %+v", event)
	}
	if event.IdempotencyKey == "" {
		t.Fatal("expected idempotency key")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		t.Fatalf("payload json invalid: %v", err)
	}
	if payload["schema_version"] != envelopeSchemaVersion || payload["receiver_account_type"] != "candidate" || payload["interview_mode"] != "视频面试" {
		t.Fatalf("payload missing legacy/envelope fields: %#v", payload)
	}
	nested, ok := payload["payload"].(map[string]any)
	if !ok || nested["type"] != "interview_scheduled" {
		t.Fatalf("nested payload missing: %#v", payload["payload"])
	}
}

type fakeOutboxStore struct {
	created []*sharedmodel.EventOutbox
}

func (s *fakeOutboxStore) Create(_ context.Context, event *sharedmodel.EventOutbox) error {
	s.created = append(s.created, event)
	return nil
}

func (s *fakeOutboxStore) CreateWithTx(_ *gorm.DB, event *sharedmodel.EventOutbox) error {
	s.created = append(s.created, event)
	return nil
}
