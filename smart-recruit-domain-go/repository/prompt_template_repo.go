package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
)

// PromptTemplateRepo handles database operations for prompt_templates and prompt_versions.
type PromptTemplateRepo struct {
	db *gorm.DB
}

// NewPromptTemplateRepo creates a new PromptTemplateRepo.
func NewPromptTemplateRepo(db *gorm.DB) *PromptTemplateRepo {
	return &PromptTemplateRepo{db: db}
}

// ── PromptTemplate CRUD ──────────────────────────────────────────────────

// Create inserts a new prompt template.
func (r *PromptTemplateRepo) Create(ctx context.Context, t *model.PromptTemplate) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// GetByID retrieves a prompt template by ID.
func (r *PromptTemplateRepo) GetByID(ctx context.Context, id int64) (*model.PromptTemplate, error) {
	var t model.PromptTemplate
	err := r.db.WithContext(ctx).First(&t, id).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Update updates a prompt template.
func (r *PromptTemplateRepo) Update(ctx context.Context, t *model.PromptTemplate) error {
	return r.db.WithContext(ctx).Model(t).Where("id = ?", t.ID).Updates(map[string]any{
		"name":        t.Name,
		"content":     t.Content,
		"variables":   t.Variables,
		"version":     t.Version,
		"is_active":   t.IsActive,
		"agent_type":  t.AgentType,
		"prompt_role": t.PromptRole,
		"updated_by":  t.UpdatedBy,
	}).Error
}

// UpdatePartial updates specific fields of a prompt template.
func (r *PromptTemplateRepo) UpdatePartial(ctx context.Context, id int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.PromptTemplate{}).Where("id = ?", id).Updates(updates).Error
}

// Delete soft-deletes a prompt template by setting is_active = 0.
func (r *PromptTemplateRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&model.PromptTemplate{}).Where("id = ?", id).Update("is_active", 0).Error
}

// List retrieves paginated prompt templates with optional agent_type filter.
func (r *PromptTemplateRepo) List(ctx context.Context, page, pageSize int32, agentType string) ([]model.PromptTemplate, int64, error) {
	var list []model.PromptTemplate
	var total int64
	query := r.db.WithContext(ctx).Model(&model.PromptTemplate{})
	if agentType != "" {
		query = query.Where("agent_type = ?", agentType)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Offset(offset).Limit(int(pageSize)).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}
	return list, total, nil
}

// GetActiveByAgentType retrieves the active template for a given agent_type and prompt_role.
func (r *PromptTemplateRepo) GetActiveByAgentType(ctx context.Context, agentType, promptRole string) (*model.PromptTemplate, error) {
	var t model.PromptTemplate
	err := r.db.WithContext(ctx).
		Where("agent_type = ? AND prompt_role = ? AND is_active = 1", agentType, promptRole).
		Order("version DESC").
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ── PromptVersion CRUD ───────────────────────────────────────────────────

// CreateVersion inserts a new version record.
func (r *PromptTemplateRepo) CreateVersion(ctx context.Context, v *model.PromptVersion) error {
	return r.db.WithContext(ctx).Create(v).Error
}

// ListVersions retrieves paginated version history for a template.
func (r *PromptTemplateRepo) ListVersions(ctx context.Context, templateID int64, page, pageSize int32) ([]model.PromptVersion, int64, error) {
	var list []model.PromptVersion
	var total int64
	query := r.db.WithContext(ctx).Model(&model.PromptVersion{}).Where("template_id = ?", templateID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Offset(offset).Limit(int(pageSize)).Order("version DESC").Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list versions: %w", err)
	}
	return list, total, nil
}

// GetVersion retrieves a specific version record.
func (r *PromptTemplateRepo) GetVersion(ctx context.Context, templateID int64, version int32) (*model.PromptVersion, error) {
	var v model.PromptVersion
	err := r.db.WithContext(ctx).
		Where("template_id = ? AND version = ?", templateID, version).
		First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}
