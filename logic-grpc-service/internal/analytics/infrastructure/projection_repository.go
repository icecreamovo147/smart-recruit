package infrastructure

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"logic-grpc-service/internal/analytics/application"
	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

type ProjectionRepository struct {
	repo *repository.AnalyticsProjectionRepo
}

func NewProjectionRepository(db *gorm.DB) *ProjectionRepository {
	return &ProjectionRepository{repo: repository.NewAnalyticsProjectionRepo(db)}
}

func (r *ProjectionRepository) SaveProjectionEvent(ctx context.Context, event application.ProjectionEventRecord) (bool, error) {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return false, err
	}
	return r.repo.SaveProjectedEvent(ctx, &model.AnalyticsProjectionEvent{
		ProjectionName: event.ProjectionName,
		Source:         event.Source,
		EventID:        event.EventID,
		EventType:      event.EventType,
		AggregateType:  event.AggregateType,
		AggregateID:    event.AggregateID,
		Producer:       event.Producer,
		IdempotencyKey: event.IdempotencyKey,
		CorrelationID:  event.CorrelationID,
		CausationID:    event.CausationID,
		TraceID:        event.TraceID,
		Payload:        string(event.Payload),
		Metadata:       string(metadata),
		OccurredAt:     event.OccurredAt,
		ProjectedAt:    event.ProjectedAt,
	})
}

func (r *ProjectionRepository) GetCheckpoint(ctx context.Context, projectionName string) (string, error) {
	return r.repo.GetCheckpoint(ctx, projectionName)
}

func (r *ProjectionRepository) SaveCheckpoint(ctx context.Context, projectionName string, cursor string) error {
	return r.repo.SaveCheckpoint(ctx, projectionName, cursor)
}
