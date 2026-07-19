package projection

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"smart-recruit-analytics-service/internal/domain/model"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) SaveProjectionEvent(ctx context.Context, event model.ProjectionEvent) error {
	payload := string(event.Payload)
	if payload == "" {
		payload = "{}"
	}
	row := projectionEventRow{
		TenantID:       event.TenantID,
		ProjectionName: "analytics-reporting",
		Source:         "domain_event",
		EventID:        event.EventID,
		EventType:      event.EventType,
		AggregateType:  event.AggregateType,
		AggregateID:    event.AggregateID,
		Payload:        payload,
		OccurredAt:     event.OccurredAt,
		ProjectedAt:    time.Now(),
	}
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}},
			DoNothing: true,
		}).
		Create(&row).Error
}

func (s *Store) SaveCheckpoint(ctx context.Context, checkpoint model.ProjectionCheckpoint) error {
	row := projectionCheckpointRow{
		ProjectionName: checkpoint.ProjectionName,
		Cursor:         checkpoint.Cursor,
		UpdatedAt:      checkpoint.UpdatedAt,
	}
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "projection_name"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"cursor":     checkpoint.Cursor,
				"updated_at": checkpoint.UpdatedAt,
			}),
		}).
		Create(&row).Error
}

type projectionEventRow struct {
	ID             uint64    `gorm:"primaryKey"`
	TenantID       *int64    `gorm:"column:tenant_id"`
	ProjectionName string    `gorm:"column:projection_name"`
	Source         string    `gorm:"column:source"`
	EventID        string    `gorm:"column:event_id;uniqueIndex:uk_analytics_projection_event"`
	EventType      string    `gorm:"column:event_type"`
	AggregateType  string    `gorm:"column:aggregate_type"`
	AggregateID    string    `gorm:"column:aggregate_id"`
	Payload        string    `gorm:"column:payload"`
	OccurredAt     time.Time `gorm:"column:occurred_at"`
	ProjectedAt    time.Time `gorm:"column:projected_at"`
}

func (projectionEventRow) TableName() string { return "analytics_projection_events" }

type projectionCheckpointRow struct {
	ProjectionName string    `gorm:"column:projection_name;primaryKey"`
	Cursor         string    `gorm:"column:cursor"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (projectionCheckpointRow) TableName() string { return "analytics_projection_checkpoints" }
