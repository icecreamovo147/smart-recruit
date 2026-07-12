package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"logic-grpc-service/internal/analytics/domain"
	"logic-grpc-service/internal/platform/events"
)

type EventProjectionIngestor struct {
	events      ProjectionEventStore
	checkpoints ProjectionCheckpointStore
	now         func() time.Time
}

func NewEventProjectionIngestor(events ProjectionEventStore, checkpoints ProjectionCheckpointStore) *EventProjectionIngestor {
	return &EventProjectionIngestor{
		events:      events,
		checkpoints: checkpoints,
		now:         time.Now,
	}
}

func (i *EventProjectionIngestor) IngestEnvelope(ctx context.Context, envelope events.Envelope) (bool, error) {
	if i == nil || i.events == nil || i.checkpoints == nil {
		return false, fmt.Errorf("analytics event projection ingestor is not configured")
	}
	if err := envelope.Validate(); err != nil {
		return false, err
	}

	projectionName := string(domain.ProjectionForEventType(envelope.EventType))
	inserted, err := i.events.SaveProjectionEvent(ctx, ProjectionEventRecord{
		ProjectionName: projectionName,
		Source:         string(domain.ProjectionSourceDomainEvent),
		EventID:        envelope.EventID,
		EventType:      envelope.EventType,
		AggregateType:  envelope.AggregateType,
		AggregateID:    envelope.AggregateID,
		Producer:       envelope.Producer,
		IdempotencyKey: envelope.IdempotencyKey,
		CorrelationID:  envelope.CorrelationID,
		CausationID:    envelope.CausationID,
		TraceID:        envelope.TraceID,
		Payload:        json.RawMessage(append([]byte(nil), envelope.Payload...)),
		Metadata:       cloneMetadata(envelope.Metadata),
		OccurredAt:     envelope.OccurredAt,
		ProjectedAt:    i.now(),
	})
	if err != nil {
		return false, err
	}
	if err := i.checkpoints.SaveCheckpoint(ctx, projectionName, envelope.EventID); err != nil {
		return false, err
	}
	return inserted, nil
}

func (i *EventProjectionIngestor) IngestPayload(ctx context.Context, body []byte) (bool, error) {
	envelope, err := events.UnmarshalJSONEnvelope(body)
	if err != nil {
		return false, err
	}
	return i.IngestEnvelope(ctx, *envelope)
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}
