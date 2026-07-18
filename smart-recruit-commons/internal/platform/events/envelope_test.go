package events

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewEnvelopeRoundTrip(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, 7, 11, 12, 30, 0, 0, time.UTC)
	envelope, err := NewEnvelope(NewEnvelopeInput{
		EventID:       " evt-application-created ",
		EventType:     "application.created",
		AggregateType: "application",
		AggregateID:   "app-42",
		OccurredAt:    occurredAt,
		Producer:      "recruitment.application",
		CorrelationID: "request-123",
		CausationID:   "command-456",
		TraceID:       "trace-789",
		Actor: &Actor{
			UserID:      " 1001 ",
			AccountType: " staff ",
			RoleKeys:    []string{"hr_manager"},
		},
		Payload:  json.RawMessage(`{"application_id":42,"status":"submitted"}`),
		Metadata: map[string]string{" retry_hint ": " none "},
	})
	if err != nil {
		t.Fatalf("new envelope: %v", err)
	}

	if envelope.SchemaVersion != EnvelopeSchemaVersion {
		t.Fatalf("schema version=%q, want %q", envelope.SchemaVersion, EnvelopeSchemaVersion)
	}
	if envelope.EventID != "evt-application-created" {
		t.Fatalf("event id not trimmed: %q", envelope.EventID)
	}
	if envelope.Actor == nil || envelope.Actor.UserID != "1001" || envelope.Actor.AccountType != "staff" {
		t.Fatalf("actor not copied and normalized: %+v", envelope.Actor)
	}
	if envelope.Metadata["retry_hint"] != "none" {
		t.Fatalf("metadata not copied and normalized: %+v", envelope.Metadata)
	}

	encoded, err := MarshalJSONEnvelope(*envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	decoded, err := UnmarshalJSONEnvelope(encoded)
	if err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if decoded.EventID != envelope.EventID || decoded.IdempotencyKey != "application:app-42:application.created:evt-application-created" {
		t.Fatalf("decoded envelope drifted: %+v", decoded)
	}
	if string(decoded.Payload) != `{"application_id":42,"status":"submitted"}` {
		t.Fatalf("payload drifted: %s", decoded.Payload)
	}
}

func TestNewEnvelopeRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	_, err := NewEnvelope(NewEnvelopeInput{
		Payload: json.RawMessage(`{}`),
	})
	if err == nil {
		t.Fatal("expected missing required fields error")
	}

	message := err.Error()
	for _, field := range []string{"event_id", "event_type", "aggregate_type", "aggregate_id", "occurred_at", "producer"} {
		if !strings.Contains(message, field) {
			t.Fatalf("missing validation problem for %s in %q", field, message)
		}
	}
}

func TestNewEnvelopeRejectsInvalidPayloadJSON(t *testing.T) {
	t.Parallel()

	_, err := NewEnvelope(NewEnvelopeInput{
		EventID:       "evt-invalid-json",
		EventType:     "application.created",
		AggregateType: "application",
		AggregateID:   "app-42",
		OccurredAt:    time.Now(),
		Producer:      "recruitment.application",
		Payload:       json.RawMessage(`{"broken"`),
	})
	if err == nil || !strings.Contains(err.Error(), "payload must be valid JSON") {
		t.Fatalf("expected invalid payload error, got %v", err)
	}
}

func TestNewEnvelopeDefaultsEmptyPayloadAndIdempotencyKey(t *testing.T) {
	t.Parallel()

	envelope, err := NewEnvelope(NewEnvelopeInput{
		EventID:       "evt-defaults",
		EventType:     "interview.scheduled",
		AggregateType: "interview",
		AggregateID:   "interview-99",
		OccurredAt:    time.Now(),
		Producer:      "interview.scheduler",
	})
	if err != nil {
		t.Fatalf("new envelope: %v", err)
	}

	if string(envelope.Payload) != `{}` {
		t.Fatalf("payload default=%s, want {}", envelope.Payload)
	}
	if envelope.IdempotencyKey != "interview:interview-99:interview.scheduled:evt-defaults" {
		t.Fatalf("idempotency key=%q", envelope.IdempotencyKey)
	}
}

func TestNewEnvelopeDefensivelyCopiesInputs(t *testing.T) {
	t.Parallel()

	payload := json.RawMessage(`{"offer_id":7}`)
	metadata := map[string]string{"source": "offer"}
	actor := &Actor{RoleKeys: []string{"offer_manager"}}

	envelope, err := NewEnvelope(NewEnvelopeInput{
		EventID:       "evt-copy",
		EventType:     "offer.created",
		AggregateType: "offer",
		AggregateID:   "offer-7",
		OccurredAt:    time.Now(),
		Producer:      "offer.service",
		Actor:         actor,
		Payload:       payload,
		Metadata:      metadata,
	})
	if err != nil {
		t.Fatalf("new envelope: %v", err)
	}

	payload[1] = 'X'
	metadata["source"] = "mutated"
	actor.RoleKeys[0] = "mutated"

	if string(envelope.Payload) != `{"offer_id":7}` {
		t.Fatalf("payload was not defensively copied: %s", envelope.Payload)
	}
	if envelope.Metadata["source"] != "offer" {
		t.Fatalf("metadata was not defensively copied: %+v", envelope.Metadata)
	}
	if envelope.Actor.RoleKeys[0] != "offer_manager" {
		t.Fatalf("actor roles were not defensively copied: %+v", envelope.Actor.RoleKeys)
	}
}

func TestUnmarshalJSONEnvelopeRequiresIdempotencyKey(t *testing.T) {
	t.Parallel()

	_, err := UnmarshalJSONEnvelope([]byte(`{
		"schema_version":"1.0",
		"event_id":"evt-missing-idempotency",
		"event_type":"application.created",
		"aggregate_type":"application",
		"aggregate_id":"app-42",
		"occurred_at":"2026-07-11T12:30:00Z",
		"producer":"recruitment.application",
		"payload":{}
	}`))
	if err == nil || !strings.Contains(err.Error(), "idempotency_key") {
		t.Fatalf("expected idempotency validation error, got %v", err)
	}
}
