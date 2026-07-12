package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
)

type EmbeddingProviderRepo struct {
	db *gorm.DB
}

func NewEmbeddingProviderRepo(db *gorm.DB) *EmbeddingProviderRepo {
	return &EmbeddingProviderRepo{db: db}
}

func (r *EmbeddingProviderRepo) Create(ctx context.Context, p *model.EmbeddingProviderConfig) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *EmbeddingProviderRepo) Update(ctx context.Context, p *model.EmbeddingProviderConfig) error {
	return r.db.WithContext(ctx).Model(p).Where("id = ?", p.ID).Updates(map[string]any{
		"name":              p.Name,
		"provider_type":     p.ProviderType,
		"endpoint":          p.Endpoint,
		"api_key_encrypted": p.APIKeyEncrypted,
		"extra_headers":     p.ExtraHeaders,
		"is_enabled":        p.IsEnabled,
	}).Error
}

func (r *EmbeddingProviderRepo) UpdatePartial(ctx context.Context, id int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.EmbeddingProviderConfig{}).Where("id = ?", id).Updates(updates).Error
}

func (r *EmbeddingProviderRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.EmbeddingProviderConfig{}).Error
}

func (r *EmbeddingProviderRepo) GetByID(ctx context.Context, id int64) (*model.EmbeddingProviderConfig, error) {
	var p model.EmbeddingProviderConfig
	err := r.db.WithContext(ctx).First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *EmbeddingProviderRepo) List(ctx context.Context, page, pageSize int32) ([]model.EmbeddingProviderConfig, int64, error) {
	var list []model.EmbeddingProviderConfig
	var total int64
	query := r.db.WithContext(ctx).Model(&model.EmbeddingProviderConfig{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Order("id ASC").Offset(offset).Limit(int(pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}
	return list, total, nil
}

func (r *EmbeddingProviderRepo) ListEnabled(ctx context.Context) ([]model.EmbeddingProviderConfig, error) {
	var list []model.EmbeddingProviderConfig
	err := r.db.WithContext(ctx).Where("is_enabled = 1").Order("id ASC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *EmbeddingProviderRepo) FindByIDs(ctx context.Context, ids []int64) ([]model.EmbeddingProviderConfig, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []model.EmbeddingProviderConfig
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}
