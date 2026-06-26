package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

// ModelConfigRepo handles database operations for llm_models.
type ModelConfigRepo struct {
	db *gorm.DB
}

func NewModelConfigRepo(db *gorm.DB) *ModelConfigRepo {
	return &ModelConfigRepo{db: db}
}

func (r *ModelConfigRepo) Create(ctx context.Context, m *model.LlmModel) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *ModelConfigRepo) Update(ctx context.Context, m *model.LlmModel) error {
	return r.db.WithContext(ctx).Model(m).Where("id = ?", m.ID).Updates(map[string]any{
		"model_name":      m.ModelName,
		"display_name":    m.DisplayName,
		"temperature":     m.Temperature,
		"top_p":           m.TopP,
		"max_tokens":      m.MaxTokens,
		"max_concurrency": m.MaxConcurrency,
		"timeout_seconds": m.TimeoutSeconds,
		"is_enabled":      m.IsEnabled,
		"is_default":      m.IsDefault,
	}).Error
}

func (r *ModelConfigRepo) UpdatePartial(ctx context.Context, id int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.LlmModel{}).Where("id = ?", id).Updates(updates).Error
}

func (r *ModelConfigRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.LlmModel{}).Error
}

func (r *ModelConfigRepo) GetByID(ctx context.Context, id int64) (*model.LlmModel, error) {
	var m model.LlmModel
	err := r.db.WithContext(ctx).First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ModelConfigRepo) List(ctx context.Context, page, pageSize int32, providerID int64) ([]model.LlmModel, int64, error) {
	var list []model.LlmModel
	var total int64
	query := r.db.WithContext(ctx).Model(&model.LlmModel{})
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

// GetDefaultModel returns the default enabled model, or the first enabled model if no default is set.
func (r *ModelConfigRepo) GetDefaultModel(ctx context.Context) (*model.LlmModel, error) {
	var m model.LlmModel
	err := r.db.WithContext(ctx).Where("is_enabled = 1 AND is_default = 1").Order("id ASC").First(&m).Error
	if err == nil {
		return &m, nil
	}
	// Fall back to first enabled model
	err = r.db.WithContext(ctx).Where("is_enabled = 1").Order("id ASC").First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetEnabledModelsByProvider returns all enabled models for a given provider.
func (r *ModelConfigRepo) GetEnabledModelsByProvider(ctx context.Context, providerID int64) ([]model.LlmModel, error) {
	var list []model.LlmModel
	err := r.db.WithContext(ctx).Where("provider_id = ? AND is_enabled = 1").Order("id ASC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ClearDefault clears the is_default flag for all models of a provider.
func (r *ModelConfigRepo) ClearDefault(ctx context.Context, providerID int64) error {
	return r.db.WithContext(ctx).Model(&model.LlmModel{}).
		Where("provider_id = ?", providerID).
		Update("is_default", 0).Error
}
