package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"smart-recruit-interview-service/internal/legacydomain/model"
)

// ProviderRepo handles database operations for llm_providers.
type ProviderRepo struct {
	db *gorm.DB
}

func NewProviderRepo(db *gorm.DB) *ProviderRepo {
	return &ProviderRepo{db: db}
}

func (r *ProviderRepo) Create(ctx context.Context, p *model.LlmProvider) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *ProviderRepo) Update(ctx context.Context, p *model.LlmProvider) error {
	return r.db.WithContext(ctx).Model(p).Where("id = ?", p.ID).Updates(map[string]any{
		"name":              p.Name,
		"base_url":          p.BaseURL,
		"api_key_encrypted": p.APIKeyEncrypted,
		"provider_type":     p.ProviderType,
		"extra_headers":     p.ExtraHeaders,
		"is_enabled":        p.IsEnabled,
	}).Error
}

func (r *ProviderRepo) UpdatePartial(ctx context.Context, id int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.LlmProvider{}).Where("id = ?", id).Updates(updates).Error
}

func (r *ProviderRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.LlmProvider{}).Error
}

func (r *ProviderRepo) GetByID(ctx context.Context, id int64) (*model.LlmProvider, error) {
	var p model.LlmProvider
	err := r.db.WithContext(ctx).First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProviderRepo) List(ctx context.Context, page, pageSize int32) ([]model.LlmProvider, int64, error) {
	var list []model.LlmProvider
	var total int64
	query := r.db.WithContext(ctx).Model(&model.LlmProvider{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Order("id ASC").Offset(offset).Limit(int(pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}
	return list, total, nil
}

// ListEnabled returns all enabled providers.
func (r *ProviderRepo) ListEnabled(ctx context.Context) ([]model.LlmProvider, error) {
	var list []model.LlmProvider
	err := r.db.WithContext(ctx).Where("is_enabled = 1").Order("id ASC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// FindByIDs returns providers matching the given IDs.
func (r *ProviderRepo) FindByIDs(ctx context.Context, ids []int64) ([]model.LlmProvider, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.LlmProvider
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}
