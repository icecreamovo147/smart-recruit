package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"logic-grpc-service/internal/platform/events"
)

type projectionStoreStub struct {
	records []ProjectionEventRecord
}

func (s *projectionStoreStub) SaveProjectionEvent(ctx context.Context, event ProjectionEventRecord) (bool, error) {
	s.records = append(s.records, event)
	return true, nil
}

type checkpointStoreStub struct {
	cursors map[string]string
}

func (s *checkpointStoreStub) GetCheckpoint(ctx context.Context, projectionName string) (string, error) {
	return s.cursors[projectionName], nil
}

func (s *checkpointStoreStub) SaveCheckpoint(ctx context.Context, projectionName string, cursor string) error {
	if s.cursors == nil {
		s.cursors = make(map[string]string)
	}
	s.cursors[projectionName] = cursor
	return nil
}

func TestEventProjectionIngestorStoresDomainEventProjection(t *testing.T) {
	store := &projectionStoreStub{}
	checkpoints := &checkpointStoreStub{}
	ingestor := NewEventProjectionIngestor(store, checkpoints)
	projectedAt := time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC)
	ingestor.now = func() time.Time { return projectedAt }
	occurredAt := time.Date(2026, 7, 11, 9, 0, 0, 0, time.UTC)

	envelope, err := events.NewEnvelope(events.NewEnvelopeInput{
		EventID:       "evt-offer-accepted",
		EventType:     "offer.accepted",
		AggregateType: "offer",
		AggregateID:   "88",
		OccurredAt:    occurredAt,
		Producer:      "offer.service",
		Payload:       json.RawMessage(`{"offer_id":88,"job_id":12}`),
		Metadata:      map[string]string{"source": "unit-test"},
	})
	if err != nil {
		t.Fatalf("new envelope: %v", err)
	}

	inserted, err := ingestor.IngestEnvelope(context.Background(), *envelope)
	if err != nil {
		t.Fatalf("ingest envelope: %v", err)
	}
	if !inserted {
		t.Fatal("expected inserted projection event")
	}
	if len(store.records) != 1 {
		t.Fatalf("records=%d, want 1", len(store.records))
	}
	record := store.records[0]
	if record.ProjectionName != "interview_offer" || record.Source != "domain_event" {
		t.Fatalf("projection classification drifted: %+v", record)
	}
	if record.EventID != "evt-offer-accepted" || record.AggregateID != "88" {
		t.Fatalf("event identity drifted: %+v", record)
	}
	if string(record.Payload) != `{"offer_id":88,"job_id":12}` {
		t.Fatalf("payload drifted: %s", record.Payload)
	}
	if record.ProjectedAt != projectedAt || checkpoints.cursors["interview_offer"] != "evt-offer-accepted" {
		t.Fatalf("projection checkpoint drifted: record=%+v checkpoints=%+v", record, checkpoints.cursors)
	}
}
