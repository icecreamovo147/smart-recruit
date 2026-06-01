package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type UsageLogRepo struct {
	db *gorm.DB
}

func NewUsageLogRepo(db *gorm.DB) *UsageLogRepo {
	return &UsageLogRepo{db: db}
}

func (r *UsageLogRepo) Create(ctx context.Context, log *model.ThirdPartyUsageLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// CountRecentByUser counts usage logs for a user within the given time window.
func (r *UsageLogRepo) CountRecentByUser(ctx context.Context, userID int64, serviceType string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Where("user_id = ? AND service_type = ? AND created_at >= ?", userID, serviceType, since).
		Count(&count).Error
	return count, err
}

// CountRecentByIP counts usage logs for an IP within the given time window.
func (r *UsageLogRepo) CountRecentByIP(ctx context.Context, ip, serviceType string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Where("ip = ? AND service_type = ? AND created_at >= ?", ip, serviceType, since).
		Count(&count).Error
	return count, err
}

// CountPresignWithoutConfirm counts presign logs without corresponding confirm logs for a user.
func (r *UsageLogRepo) CountPresignWithoutConfirm(ctx context.Context, userID int64, since time.Time) (int64, error) {
	var presignCount int64
	if err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Where("user_id = ? AND service_type = ? AND created_at >= ?", userID, "oss_presign", since).
		Count(&presignCount).Error; err != nil {
		return 0, err
	}
	var confirmCount int64
	if err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Where("user_id = ? AND service_type = ? AND created_at >= ?", userID, "oss_confirm", since).
		Count(&confirmCount).Error; err != nil {
		return 0, err
	}
	return presignCount - confirmCount, nil
}

// CountLargeFileConfirms counts recent confirm logs with large files for a user.
func (r *UsageLogRepo) CountLargeFileConfirms(ctx context.Context, userID int64, minSize int64, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Where("user_id = ? AND service_type = ? AND object_size >= ? AND created_at >= ?", userID, "oss_confirm", minSize, since).
		Count(&count).Error
	return count, err
}

// CountRecentErrors counts recent error/timeout logs for a user.
func (r *UsageLogRepo) CountRecentErrors(ctx context.Context, userID int64, serviceType string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ThirdPartyUsageLog{}).
		Where("user_id = ? AND service_type = ? AND status != ? AND created_at >= ?", userID, serviceType, "ok", since).
		Count(&count).Error
	return count, err
}

// UsageLogFilter holds optional filter criteria for listing usage logs.
type UsageLogFilter struct {
	ServiceType string
	Provider    string
	Status      string
	UserID      int64
	RequestID   string
	StartTime   *time.Time
	EndTime     *time.Time
}

// List returns a paginated list of usage logs matching the filter.
func (r *UsageLogRepo) List(ctx context.Context, filter UsageLogFilter, page, pageSize int) ([]model.ThirdPartyUsageLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	q := r.db.WithContext(ctx).Model(&model.ThirdPartyUsageLog{})
	if filter.ServiceType != "" {
		q = q.Where("service_type = ?", filter.ServiceType)
	}
	if filter.Provider != "" {
		q = q.Where("provider = ?", filter.Provider)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.UserID != 0 {
		q = q.Where("user_id = ?", filter.UserID)
	}
	if filter.RequestID != "" {
		q = q.Where("request_id = ?", filter.RequestID)
	}
	if filter.StartTime != nil {
		q = q.Where("created_at >= ?", filter.StartTime)
	}
	if filter.EndTime != nil {
		q = q.Where("created_at <= ?", filter.EndTime)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.ThirdPartyUsageLog
	offset := (page - 1) * pageSize
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
