package repository

import (
	"context"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

// UsageAuditContextRepo manages AI usage RBAC context records.
type UsageAuditContextRepo struct {
	db *gorm.DB
}

func NewUsageAuditContextRepo(db *gorm.DB) *UsageAuditContextRepo {
	return &UsageAuditContextRepo{db: db}
}

// Create writes an AI usage auth context record.
func (r *UsageAuditContextRepo) Create(ctx context.Context, c *model.AIUsageAuthContext) error {
	return r.db.WithContext(ctx).Create(c).Error
}

// ListByActor returns paginated auth context records for a given actor.
func (r *UsageAuditContextRepo) ListByActor(ctx context.Context, actorUserID uint64, page, pageSize int) ([]model.AIUsageAuthContext, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	q := r.db.WithContext(ctx).Model(&model.AIUsageAuthContext{}).
		Where("actor_user_id = ?", actorUserID)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.AIUsageAuthContext
	offset := (page - 1) * pageSize
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListByPermission returns paginated auth context records for a given permission.
func (r *UsageAuditContextRepo) ListByPermission(ctx context.Context, permissionKey string, page, pageSize int) ([]model.AIUsageAuthContext, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	q := r.db.WithContext(ctx).Model(&model.AIUsageAuthContext{}).
		Where("permission_key = ?", permissionKey)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.AIUsageAuthContext
	offset := (page - 1) * pageSize
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// -- CreateWithTx writes an AI usage auth context record within a transaction.

func (r *UsageAuditContextRepo) CreateWithTx(ctx context.Context, tx *gorm.DB, c *model.AIUsageAuthContext) error {
	return tx.WithContext(ctx).Create(c).Error
}
