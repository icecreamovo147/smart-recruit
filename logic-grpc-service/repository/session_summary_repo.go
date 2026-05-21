package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"logic-grpc-service/model"
)

type SessionSummaryRepo struct {
	db *gorm.DB
}

func NewSessionSummaryRepo(db *gorm.DB) *SessionSummaryRepo {
	return &SessionSummaryRepo{db: db}
}

func (r *SessionSummaryRepo) GetBySession(ctx context.Context, hrID, sessionID int64) (*model.AISessionSummary, error) {
	var summary model.AISessionSummary
	err := r.db.WithContext(ctx).
		Where("hr_id = ? AND session_id = ?", hrID, sessionID).
		First(&summary).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &summary, err
}

func (r *SessionSummaryRepo) Upsert(ctx context.Context, summary *model.AISessionSummary) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"summary", "covered_message_id", "message_count", "updated_at"}),
	}).Create(summary).Error
}
