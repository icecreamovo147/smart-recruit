package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"logic-grpc-service/model"
)

type AnalyticsProjectionRepo struct {
	db *gorm.DB
}

func NewAnalyticsProjectionRepo(db *gorm.DB) *AnalyticsProjectionRepo {
	return &AnalyticsProjectionRepo{db: db}
}

func (r *AnalyticsProjectionRepo) SaveProjectedEvent(ctx context.Context, event *model.AnalyticsProjectionEvent) (bool, error) {
	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}},
			DoNothing: true,
		}).
		Create(event)
	return result.RowsAffected > 0, result.Error
}

func (r *AnalyticsProjectionRepo) GetProjectedEventByEventID(ctx context.Context, eventID string) (*model.AnalyticsProjectionEvent, error) {
	var event model.AnalyticsProjectionEvent
	err := r.db.WithContext(ctx).Where("event_id = ?", eventID).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *AnalyticsProjectionRepo) SaveCheckpoint(ctx context.Context, projectionName string, cursor string) error {
	checkpoint := &model.AnalyticsProjectionCheckpoint{
		ProjectionName: projectionName,
		Cursor:         cursor,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "projection_name"}},
			DoUpdates: clause.Assignments(map[string]any{
				"cursor":     cursor,
				"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
			}),
		}).
		Create(checkpoint).Error
}

func (r *AnalyticsProjectionRepo) GetCheckpoint(ctx context.Context, projectionName string) (string, error) {
	var checkpoint model.AnalyticsProjectionCheckpoint
	err := r.db.WithContext(ctx).Where("projection_name = ?", projectionName).First(&checkpoint).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return checkpoint.Cursor, nil
}
