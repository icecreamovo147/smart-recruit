package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

// AgentConfigRepo handles database operations for agent_configs and agent_tool_bindings.
type AgentConfigRepo struct {
	db *gorm.DB
}

func NewAgentConfigRepo(db *gorm.DB) *AgentConfigRepo {
	return &AgentConfigRepo{db: db}
}

// ── Agent Config CRUD ──────────────────────────────────────────────────

func (r *AgentConfigRepo) Create(ctx context.Context, cfg *model.AgentConfig) error {
	return r.db.WithContext(ctx).Create(cfg).Error
}

func (r *AgentConfigRepo) Update(ctx context.Context, cfg *model.AgentConfig) error {
	return r.db.WithContext(ctx).Model(cfg).Where("id = ?", cfg.ID).Updates(cfg).Error
}

func (r *AgentConfigRepo) UpdatePartial(ctx context.Context, id int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.AgentConfig{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AgentConfigRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.AgentConfig{}).Error
}

func (r *AgentConfigRepo) GetByID(ctx context.Context, id int64) (*model.AgentConfig, error) {
	var cfg model.AgentConfig
	err := r.db.WithContext(ctx).First(&cfg, id).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *AgentConfigRepo) GetByAgentType(ctx context.Context, agentType string) (*model.AgentConfig, error) {
	var cfg model.AgentConfig
	err := r.db.WithContext(ctx).Where("agent_type = ? AND is_enabled = 1", agentType).Order("is_default DESC, id ASC").First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *AgentConfigRepo) List(ctx context.Context, page, pageSize int32, agentType string) ([]model.AgentConfig, int64, error) {
	var list []model.AgentConfig
	var total int64
	query := r.db.WithContext(ctx).Model(&model.AgentConfig{})
	if agentType != "" {
		query = query.Where("agent_type = ?", agentType)
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

// ── Tool Bindings ──────────────────────────────────────────────────────

func (r *AgentConfigRepo) ListToolBindings(ctx context.Context, agentID int64) ([]model.AgentToolBinding, error) {
	var list []model.AgentToolBinding
	err := r.db.WithContext(ctx).Where("agent_id = ? AND is_enabled = 1", agentID).Order("id ASC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *AgentConfigRepo) ReplaceToolBindings(ctx context.Context, agentID int64, toolNames []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ?", agentID).Delete(&model.AgentToolBinding{}).Error; err != nil {
			return fmt.Errorf("delete existing bindings: %w", err)
		}
		for _, name := range toolNames {
			if err := tx.Create(&model.AgentToolBinding{
				AgentID:   agentID,
				ToolName:  name,
				IsEnabled: 1,
			}).Error; err != nil {
				return fmt.Errorf("create binding for %s: %w", name, err)
			}
		}
		return nil
	})
}

// GetAgentConfigWithBindings retrieves an agent config by agent_type with its tool bindings.
func (r *AgentConfigRepo) GetAgentConfigWithBindings(ctx context.Context, agentType string) (*model.AgentConfig, []model.AgentToolBinding, error) {
	cfg, err := r.GetByAgentType(ctx, agentType)
	if err != nil {
		return nil, nil, err
	}
	bindings, err := r.ListToolBindings(ctx, cfg.ID)
	if err != nil {
		return nil, nil, err
	}
	return cfg, bindings, nil
}

// ExistsByName checks if an agent config with the given name exists.
func (r *AgentConfigRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AgentConfig{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ClearDefaultsForType sets is_default = 0 for all agents of a given type.
func (r *AgentConfigRepo) ClearDefaultsForType(ctx context.Context, agentType string) error {
	return r.db.WithContext(ctx).Model(&model.AgentConfig{}).Where("agent_type = ?", agentType).Update("is_default", 0).Error
}
