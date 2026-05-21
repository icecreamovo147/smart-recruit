package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type InviteCodeRepo struct {
	db *gorm.DB
}

func NewInviteCodeRepo(db *gorm.DB) *InviteCodeRepo {
	return &InviteCodeRepo{db: db}
}

func (r *InviteCodeRepo) Create(ctx context.Context, ic *model.InviteCode) error {
	return r.db.WithContext(ctx).Create(ic).Error
}

// GetByCode returns the invite code if it is active and not expired.
func (r *InviteCodeRepo) GetByCode(ctx context.Context, code string) (*model.InviteCode, error) {
	var ic model.InviteCode
	err := r.db.WithContext(ctx).
		Where("code = ? AND is_active = 1 AND (expires_at IS NULL OR expires_at > ?)", code, time.Now()).
		First(&ic).Error
	if err != nil {
		return nil, err
	}
	return &ic, nil
}

func (r *InviteCodeRepo) ListByCreator(ctx context.Context, createdBy int64, page, pageSize int32) ([]model.InviteCode, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&model.InviteCode{}).Where("created_by = ?", createdBy)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.InviteCode
	offset := int((page - 1) * pageSize)
	err := query.Order("created_at DESC").Offset(offset).Limit(int(pageSize)).Find(&rows).Error
	return rows, total, err
}

func (r *InviteCodeRepo) GetByID(ctx context.Context, id int64) (*model.InviteCode, error) {
	var ic model.InviteCode
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&ic).Error
	if err != nil {
		return nil, err
	}
	return &ic, nil
}

func (r *InviteCodeRepo) Extend(ctx context.Context, id int64, newExpiresAt *time.Time) error {
	return r.db.WithContext(ctx).Model(&model.InviteCode{}).Where("id = ?", id).Update("expires_at", newExpiresAt).Error
}

func (r *InviteCodeRepo) Revoke(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.InviteCode{}).Where("id = ?", id).Update("is_active", 0).Error
}

func (r *InviteCodeRepo) Reactivate(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.InviteCode{}).Where("id = ?", id).Update("is_active", 1).Error
}
