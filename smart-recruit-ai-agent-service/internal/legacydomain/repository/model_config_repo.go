package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"smart-recruit-ai-agent-service/internal/legacydomain/model"
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
		"model_name":            m.ModelName,
		"display_name":          m.DisplayName,
		"temperature":           m.Temperature,
		"top_p":                 m.TopP,
		"max_tokens":            m.MaxTokens,
		"context_window_tokens": m.ContextWindowTokens,
		"max_concurrency":       m.MaxConcurrency,
		"timeout_seconds":       m.TimeoutSeconds,
		"is_enabled":            m.IsEnabled,
		"is_default":            m.IsDefault,
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

// GetDefaultModel returns the default enabled model with an enabled provider,
// or the first enabled model on an enabled provider if no default is set.
func (r *ModelConfigRepo) GetDefaultModel(ctx context.Context) (*model.LlmModel, error) {
	var m model.LlmModel
	baseQuery := func() *gorm.DB {
		return r.db.WithContext(ctx).
			Model(&model.LlmModel{}).
			Joins("JOIN llm_providers ON llm_providers.id = llm_models.provider_id").
			Where("llm_models.is_enabled = 1 AND llm_providers.is_enabled = 1")
	}

	err := baseQuery().
		Where("llm_models.is_default = 1").
		Order("llm_models.updated_at DESC, llm_models.id DESC").
		First(&m).Error
	if err == nil {
		return &m, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	err = baseQuery().Order("llm_models.id ASC").First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetEnabledModelsByProvider returns all enabled models for a given provider.
func (r *ModelConfigRepo) GetEnabledModelsByProvider(ctx context.Context, providerID int64) ([]model.LlmModel, error) {
	var list []model.LlmModel
	err := r.db.WithContext(ctx).Where("provider_id = ? AND is_enabled = 1", providerID).Order("id ASC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ClearDefault clears the is_default flag for all models globally.
func (r *ModelConfigRepo) ClearDefault(ctx context.Context) error {
	return r.db.WithContext(ctx).Model(&model.LlmModel{}).
		Where("is_default = 1").
		Update("is_default", 0).Error
}

// CreateWithDefault creates a model and clears any existing global default atomically.
func (r *ModelConfigRepo) CreateWithDefault(ctx context.Context, m *model.LlmModel) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if m.IsDefault == 1 {
			if err := tx.Model(&model.LlmModel{}).Where("is_default = 1").Update("is_default", 0).Error; err != nil {
				return fmt.Errorf("clear existing defaults: %w", err)
			}
		}
		return tx.Create(m).Error
	})
}

// UpdatePartialWithDefault updates a model and clears any existing global default atomically when setting default.
func (r *ModelConfigRepo) UpdatePartialWithDefault(ctx context.Context, id int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if v, ok := updates["is_default"]; ok {
			if vi, ok2 := v.(int32); ok2 && vi == 1 {
				if err := tx.Model(&model.LlmModel{}).Where("is_default = 1").Update("is_default", 0).Error; err != nil {
					return fmt.Errorf("clear existing defaults: %w", err)
				}
			}
		}
		return tx.Model(&model.LlmModel{}).Where("id = ?", id).Updates(updates).Error
	})
}
