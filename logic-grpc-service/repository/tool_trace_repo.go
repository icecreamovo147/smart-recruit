package repository

import (
	"context"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type ToolTraceRepo struct {
	db *gorm.DB
}

func NewToolTraceRepo(db *gorm.DB) *ToolTraceRepo {
	return &ToolTraceRepo{db: db}
}

func (r *ToolTraceRepo) Create(ctx context.Context, trace *model.AIToolTrace) error {
	return r.db.WithContext(ctx).Create(trace).Error
}

func (r *ToolTraceRepo) ListBySession(ctx context.Context, hrID, sessionID int64, limit int) ([]model.AIToolTrace, error) {
	var rows []model.AIToolTrace
	err := r.db.WithContext(ctx).
		Where("hr_id = ? AND session_id = ?", hrID, sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}
