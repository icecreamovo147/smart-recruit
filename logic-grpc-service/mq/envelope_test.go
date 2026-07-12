package mq

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"logic-grpc-service/internal/platform/events"
)

func TestConsumeEnvelopeRejectsInvalidEnvelopeBeforeHandler(t *testing.T) {
	conn := &Conn{consumers: map[string]consumerRegistration{}}
	var called bool
	if err := conn.ConsumeEnvelope(context.Background(), "queue", func(context.Context, *events.Envelope) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("ConsumeEnvelope returned error: %v", err)
	}
	registered := conn.consumers["queue"]
	if registered.handler == nil {
		t.Fatal("expected registered handler")
	}
	if err := registered.handler(context.Background(), []byte(`{"not":"an envelope"}`)); err == nil {
		t.Fatal("expected invalid envelope error")
	}
	if called {
		t.Fatal("handler should not run for invalid envelope")
	}
}

func TestConsumeEnvelopePassesValidatedEnvelope(t *testing.T) {
	conn := &Conn{consumers: map[string]consumerRegistration{}}
	want, err := events.NewEnvelope(events.NewEnvelopeInput{
		EventID:       "evt-1",
		EventType:     "notification.create",
		AggregateType: "notification",
		AggregateID:   "1",
		OccurredAt:    time.Now(),
		Producer:      "test",
		Payload:       json.RawMessage(`{"ok":true}`),
	})
	if err != nil {
		t.Fatalf("NewEnvelope returned error: %v", err)
	}
	body, err := events.MarshalJSONEnvelope(*want)
	if err != nil {
		t.Fatalf("MarshalJSONEnvelope returned error: %v", err)
	}
	if err := conn.ConsumeEnvelope(context.Background(), "queue", func(_ context.Context, got *events.Envelope) error {
		if got.EventID != want.EventID || got.IdempotencyKey != want.IdempotencyKey {
			t.Fatalf("unexpected envelope: %+v", got)
		}
		return nil
	}); err != nil {
		t.Fatalf("ConsumeEnvelope returned error: %v", err)
	}
	if err := conn.consumers["queue"].handler(context.Background(), body); err != nil {
		t.Fatalf("registered handler returned error: %v", err)
	}
}

func TestQueuePlansExposeRetryDLQAndReplayRules(t *testing.T) {
	conn := &Conn{cfg: Config{
		NotificationQueue: "notification",
		ResumeParseQueue:  "resume",
		EmailQueue:        "email",
		EmbeddingQueue:    "embedding",
		AgentRunQueue:     "agent",
	}}
	plans := conn.QueuePlans()
	if len(plans) != 5 {
		t.Fatalf("queue plans len = %d, want 5", len(plans))
	}
	for _, plan := range ReplayPlans(plans) {
		if err := plan.Validate(); err != nil {
			t.Fatalf("ReplayPlan for %s invalid: %v", plan.Queue, err)
		}
	}
}
