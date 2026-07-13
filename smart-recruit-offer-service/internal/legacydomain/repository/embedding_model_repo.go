package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"smart-recruit-offer-service/internal/legacydomain/model"
)

type EmbeddingModelRepo struct {
	db *gorm.DB
}

func NewEmbeddingModelRepo(db *gorm.DB) *EmbeddingModelRepo {
	return &EmbeddingModelRepo{db: db}
}

func (r *EmbeddingModelRepo) Create(ctx context.Context, m *model.EmbeddingModelConfig) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *EmbeddingModelRepo) Update(ctx context.Context, m *model.EmbeddingModelConfig) error {
	return r.db.WithContext(ctx).Model(m).Where("id = ?", m.ID).Updates(map[string]any{
		"model_name":        m.ModelName,
		"display_name":      m.DisplayName,
		"embedding_dim":     m.EmbeddingDim,
		"input_token_limit": m.InputTokenLimit,
		"batch_size":        m.BatchSize,
		"timeout_seconds":   m.TimeoutSeconds,
		"max_retries":       m.MaxRetries,
		"is_enabled":        m.IsEnabled,
		"is_default":        m.IsDefault,
	}).Error
}

func (r *EmbeddingModelRepo) UpdatePartial(ctx context.Context, id int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.EmbeddingModelConfig{}).Where("id = ?", id).Updates(updates).Error
}

func (r *EmbeddingModelRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.EmbeddingModelConfig{}).Error
}

func (r *EmbeddingModelRepo) GetByID(ctx context.Context, id int64) (*model.EmbeddingModelConfig, error) {
	var m model.EmbeddingModelConfig
	err := r.db.WithContext(ctx).First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *EmbeddingModelRepo) List(ctx context.Context, page, pageSize int32, providerID int64) ([]model.EmbeddingModelConfig, int64, error) {
	var list []model.EmbeddingModelConfig
	var total int64
	query := r.db.WithContext(ctx).Model(&model.EmbeddingModelConfig{})
	if providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Order("id ASC").Offset(offset).Limit(int(pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}
	return list, total, nil
}

func (r *EmbeddingModelRepo) GetDefaultModel(ctx context.Context) (*model.EmbeddingModelConfig, error) {
	var m model.EmbeddingModelConfig
	err := r.db.WithContext(ctx).Where("is_enabled = 1 AND is_default = 1").Order("id ASC").First(&m).Error
	if err == nil {
		return &m, nil
	}
	err = r.db.WithContext(ctx).Where("is_enabled = 1").Order("id ASC").First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *EmbeddingModelRepo) GetEnabledModelsByProvider(ctx context.Context, providerID int64) ([]model.EmbeddingModelConfig, error) {
	var list []model.EmbeddingModelConfig
	err := r.db.WithContext(ctx).Where("provider_id = ? AND is_enabled = 1", providerID).Order("id ASC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *EmbeddingModelRepo) ClearDefault(ctx context.Context) error {
	return r.db.WithContext(ctx).Model(&model.EmbeddingModelConfig{}).
		Where("is_default = 1").
		Update("is_default", 0).Error
}
