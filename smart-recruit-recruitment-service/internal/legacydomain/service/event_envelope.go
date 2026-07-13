package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const envelopeSchemaVersion = "1.0"

type outboxEnvelope struct {
	SchemaVersion  string            `json:"schema_version"`
	EventID        string            `json:"event_id"`
	EventType      string            `json:"event_type"`
	AggregateType  string            `json:"aggregate_type"`
	AggregateID    string            `json:"aggregate_id"`
	OccurredAt     time.Time         `json:"occurred_at"`
	Producer       string            `json:"producer"`
	IdempotencyKey string            `json:"idempotency_key"`
	Payload        json.RawMessage   `json:"payload"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type outboxEnvelopeInput struct {
	EventID       string
	EventType     string
	AggregateType string
	AggregateID   string
	OccurredAt    time.Time
	Producer      string
	Payload       json.RawMessage
	Metadata      map[string]string
}

func newOutboxEnvelope(input outboxEnvelopeInput) (*outboxEnvelope, error) {
	envelope := &outboxEnvelope{
		SchemaVersion:  envelopeSchemaVersion,
		EventID:        strings.TrimSpace(input.EventID),
		EventType:      strings.TrimSpace(input.EventType),
		AggregateType:  strings.TrimSpace(input.AggregateType),
		AggregateID:    strings.TrimSpace(input.AggregateID),
		OccurredAt:     input.OccurredAt,
		Producer:       strings.TrimSpace(input.Producer),
		IdempotencyKey: strings.Join([]string{strings.TrimSpace(input.AggregateType), strings.TrimSpace(input.AggregateID), strings.TrimSpace(input.EventType), strings.TrimSpace(input.EventID)}, ":"),
		Payload:        append(json.RawMessage(nil), input.Payload...),
		Metadata:       cloneOutboxMetadata(input.Metadata),
	}
	if len(envelope.Payload) == 0 {
		envelope.Payload = json.RawMessage(`{}`)
	}
	return envelope, envelope.validate()
}

func (e outboxEnvelope) validate() error {
	var problems []error
	if e.SchemaVersion != envelopeSchemaVersion {
		problems = append(problems, fmt.Errorf("schema_version must be %q", envelopeSchemaVersion))
	}
	if e.EventID == "" {
		problems = append(problems, errors.New("event_id is required"))
	}
	if e.EventType == "" {
		problems = append(problems, errors.New("event_type is required"))
	}
	if e.AggregateType == "" {
		problems = append(problems, errors.New("aggregate_type is required"))
	}
	if e.AggregateID == "" {
		problems = append(problems, errors.New("aggregate_id is required"))
	}
	if e.OccurredAt.IsZero() {
		problems = append(problems, errors.New("occurred_at is required"))
	}
	if e.Producer == "" {
		problems = append(problems, errors.New("producer is required"))
	}
	if e.IdempotencyKey == "" {
		problems = append(problems, errors.New("idempotency_key is required"))
	}
	if len(e.Payload) == 0 || !json.Valid(e.Payload) {
		problems = append(problems, errors.New("payload must be valid JSON"))
	}
	return errors.Join(problems...)
}

func marshalJSONOutboxEnvelope(envelope outboxEnvelope) ([]byte, error) {
	if err := envelope.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(envelope)
}

func cloneOutboxMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	copied := make(map[string]string, len(metadata))
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey != "" {
			copied[trimmedKey] = strings.TrimSpace(value)
		}
	}
	return copied
}
