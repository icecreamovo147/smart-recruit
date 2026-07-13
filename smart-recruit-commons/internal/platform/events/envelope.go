// Package events defines shared contracts for domain events that cross
// bounded-context boundaries through Outbox, Inbox, consumers, and projections.
package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// EnvelopeSchemaVersion is the current domain event envelope schema.
	EnvelopeSchemaVersion = "1.0"
)

// Actor identifies the authenticated actor that caused a domain event when the
// source operation has user context.
type Actor struct {
	UserID      string   `json:"user_id,omitempty"`
	AccountType string   `json:"account_type,omitempty"`
	RoleKeys    []string `json:"role_keys,omitempty"`
}

// Envelope is the standard JSON domain-event envelope shared by the Outbox,
// future Inbox records, asynchronous consumers, and Analytics projections.
type Envelope struct {
	SchemaVersion  string            `json:"schema_version"`
	EventID        string            `json:"event_id"`
	EventType      string            `json:"event_type"`
	AggregateType  string            `json:"aggregate_type"`
	AggregateID    string            `json:"aggregate_id"`
	OccurredAt     time.Time         `json:"occurred_at"`
	Producer       string            `json:"producer"`
	CorrelationID  string            `json:"correlation_id,omitempty"`
	CausationID    string            `json:"causation_id,omitempty"`
	IdempotencyKey string            `json:"idempotency_key"`
	TraceID        string            `json:"trace_id,omitempty"`
	Actor          *Actor            `json:"actor,omitempty"`
	Payload        json.RawMessage   `json:"payload"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// NewEnvelopeInput contains the source-owned values needed to build a standard
// domain-event envelope.
type NewEnvelopeInput struct {
	EventID        string
	EventType      string
	AggregateType  string
	AggregateID    string
	OccurredAt     time.Time
	Producer       string
	CorrelationID  string
	CausationID    string
	IdempotencyKey string
	TraceID        string
	Actor          *Actor
	Payload        json.RawMessage
	Metadata       map[string]string
}

// NewEnvelope builds and validates a domain-event envelope. If IdempotencyKey
// is omitted, it derives a stable key from the aggregate and event identity.
func NewEnvelope(input NewEnvelopeInput) (*Envelope, error) {
	envelope := &Envelope{
		SchemaVersion:  EnvelopeSchemaVersion,
		EventID:        strings.TrimSpace(input.EventID),
		EventType:      strings.TrimSpace(input.EventType),
		AggregateType:  strings.TrimSpace(input.AggregateType),
		AggregateID:    strings.TrimSpace(input.AggregateID),
		OccurredAt:     input.OccurredAt,
		Producer:       strings.TrimSpace(input.Producer),
		CorrelationID:  strings.TrimSpace(input.CorrelationID),
		CausationID:    strings.TrimSpace(input.CausationID),
		IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
		TraceID:        strings.TrimSpace(input.TraceID),
		Actor:          cloneActor(input.Actor),
		Payload:        clonePayload(input.Payload),
		Metadata:       cloneMetadata(input.Metadata),
	}
	if len(envelope.Payload) == 0 {
		envelope.Payload = json.RawMessage(`{}`)
	}
	if envelope.IdempotencyKey == "" {
		envelope.IdempotencyKey = defaultIdempotencyKey(envelope.AggregateType, envelope.AggregateID, envelope.EventType, envelope.EventID)
	}
	if err := envelope.Validate(); err != nil {
		return nil, err
	}
	return envelope, nil
}

// Validate checks the envelope fields required for idempotent consumption,
// retry diagnostics, and replayable projections.
func (e Envelope) Validate() error {
	var problems []error
	if strings.TrimSpace(e.SchemaVersion) != EnvelopeSchemaVersion {
		problems = append(problems, fmt.Errorf("schema_version must be %q", EnvelopeSchemaVersion))
	}
	if strings.TrimSpace(e.EventID) == "" {
		problems = append(problems, errors.New("event_id is required"))
	}
	if strings.TrimSpace(e.EventType) == "" {
		problems = append(problems, errors.New("event_type is required"))
	}
	if strings.TrimSpace(e.AggregateType) == "" {
		problems = append(problems, errors.New("aggregate_type is required"))
	}
	if strings.TrimSpace(e.AggregateID) == "" {
		problems = append(problems, errors.New("aggregate_id is required"))
	}
	if e.OccurredAt.IsZero() {
		problems = append(problems, errors.New("occurred_at is required"))
	}
	if strings.TrimSpace(e.Producer) == "" {
		problems = append(problems, errors.New("producer is required"))
	}
	if strings.TrimSpace(e.IdempotencyKey) == "" {
		problems = append(problems, errors.New("idempotency_key is required"))
	}
	if len(e.Payload) == 0 {
		problems = append(problems, errors.New("payload is required"))
	} else if !json.Valid(e.Payload) {
		problems = append(problems, errors.New("payload must be valid JSON"))
	}
	if len(problems) > 0 {
		return errors.Join(problems...)
	}
	return nil
}

// MarshalJSONEnvelope validates and encodes an envelope for Outbox payloads or
// broker messages.
func MarshalJSONEnvelope(envelope Envelope) ([]byte, error) {
	if err := envelope.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(envelope)
}

// UnmarshalJSONEnvelope decodes and validates a broker or Outbox payload.
func UnmarshalJSONEnvelope(data []byte) (*Envelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	if err := envelope.Validate(); err != nil {
		return nil, err
	}
	envelope.Actor = cloneActor(envelope.Actor)
	envelope.Payload = clonePayload(envelope.Payload)
	envelope.Metadata = cloneMetadata(envelope.Metadata)
	return &envelope, nil
}

func defaultIdempotencyKey(aggregateType, aggregateID, eventType, eventID string) string {
	parts := []string{
		strings.TrimSpace(aggregateType),
		strings.TrimSpace(aggregateID),
		strings.TrimSpace(eventType),
		strings.TrimSpace(eventID),
	}
	return strings.Join(parts, ":")
}

func cloneActor(actor *Actor) *Actor {
	if actor == nil {
		return nil
	}
	return &Actor{
		UserID:      strings.TrimSpace(actor.UserID),
		AccountType: strings.TrimSpace(actor.AccountType),
		RoleKeys:    append([]string(nil), actor.RoleKeys...),
	}
}

func clonePayload(payload json.RawMessage) json.RawMessage {
	if len(payload) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), payload...)
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	copied := make(map[string]string, len(metadata))
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		copied[trimmedKey] = strings.TrimSpace(value)
	}
	if len(copied) == 0 {
		return nil
	}
	return copied
}
