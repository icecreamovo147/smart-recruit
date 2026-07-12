package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
)

type EmailLogRepo struct {
	db *gorm.DB
}

func NewEmailLogRepo(db *gorm.DB) *EmailLogRepo {
	return &EmailLogRepo{db: db}
}

// Create inserts an email log record.
func (r *EmailLogRepo) Create(ctx context.Context, log *model.EmailLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// ExistsByEventID checks if an email has already been sent for this event.
func (r *EmailLogRepo) ExistsByEventID(ctx context.Context, eventID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.EmailLog{}).Where("event_id = ?", eventID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetByEventID retrieves an email log by event_id (nil if not found).
func (r *EmailLogRepo) GetByEventID(ctx context.Context, eventID string) (*model.EmailLog, error) {
	var log model.EmailLog
	err := r.db.WithContext(ctx).Where("event_id = ?", eventID).First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &log, err
}
