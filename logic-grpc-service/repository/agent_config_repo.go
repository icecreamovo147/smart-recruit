package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

// AgentConfigRepo handles database operations for agent configs and capability bindings.
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

// ── Transactional Agent Config + Capability Bindings ─────────────────

// CreateWithCapabilityBindings creates an agent config and its capability
// bindings within a single transaction. If the bindings cannot be written,
// the entire operation is rolled back. If the agent is a default for its type,
// it clears existing defaults atomically.
func (r *AgentConfigRepo) CreateWithCapabilityBindings(ctx context.Context, cfg *model.AgentConfig, bindings []model.AgentCapabilityBinding) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear existing defaults for this type if creating a new default.
		if cfg.IsDefault == 1 {
			if err := tx.Model(&model.AgentConfig{}).
				Where("agent_type = ? AND is_default = 1", cfg.AgentType).
				Update("is_default", 0).Error; err != nil {
				return fmt.Errorf("clear existing defaults: %w", err)
			}
		}

		if err := tx.Create(cfg).Error; err != nil {
			return fmt.Errorf("create agent config: %w", err)
		}

		if len(bindings) == 0 {
			return nil
		}

		for i := range bindings {
			bindings[i].AgentID = cfg.ID
			if bindings[i].IsEnabled == 0 {
				bindings[i].IsEnabled = 1
			}
			if err := tx.Create(&bindings[i]).Error; err != nil {
				return fmt.Errorf("create capability binding %s:%s: %w", bindings[i].CapabilitySource, bindings[i].CapabilityKey, err)
			}
			if bindings[i].CapabilitySource == "builtin" {
				if err := tx.Create(&model.AgentToolBinding{
					AgentID:   cfg.ID,
					ToolName:  bindings[i].CapabilityKey,
					IsEnabled: bindings[i].IsEnabled,
				}).Error; err != nil {
					return fmt.Errorf("create legacy tool binding %s: %w", bindings[i].CapabilityKey, err)
				}
			}
		}

		return nil
	})
}

// UpdateWithCapabilityBindings updates an agent config and optionally replaces
// its capability bindings within a single transaction.
// If the updates include is_default=1, existing defaults for the same agent_type
// are cleared within the same transaction.
func (r *AgentConfigRepo) UpdateWithCapabilityBindings(ctx context.Context, id int64, updates map[string]any, bindings []model.AgentCapabilityBinding, replaceBindings bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			// Clear existing defaults if setting this one as default.
			if v, ok := updates["is_default"]; ok {
				if vi, ok2 := v.(int32); ok2 && vi == 1 {
					// Fetch the agent type to clear defaults for.
					var existing model.AgentConfig
					if err := tx.Select("agent_type").First(&existing, id).Error; err != nil {
						return fmt.Errorf("get existing agent type: %w", err)
					}
					if err := tx.Model(&model.AgentConfig{}).
						Where("agent_type = ? AND id <> ? AND is_default = 1", existing.AgentType, id).
						Update("is_default", 0).Error; err != nil {
						return fmt.Errorf("clear existing defaults: %w", err)
					}
				}
			}

			if err := tx.Model(&model.AgentConfig{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return fmt.Errorf("update agent config: %w", err)
			}
		}

		if !replaceBindings {
			return nil
		}

		// Delete existing bindings (both capability and legacy tool)
		if err := tx.Where("agent_id = ?", id).Delete(&model.AgentCapabilityBinding{}).Error; err != nil {
			return fmt.Errorf("delete existing capability bindings: %w", err)
		}
		if err := tx.Where("agent_id = ?", id).Delete(&model.AgentToolBinding{}).Error; err != nil {
			return fmt.Errorf("delete existing legacy tool bindings: %w", err)
		}

		for _, b := range bindings {
			b.AgentID = id
			if b.IsEnabled == 0 {
				b.IsEnabled = 1
			}
			if err := tx.Create(&b).Error; err != nil {
				return fmt.Errorf("create capability binding %s:%s: %w", b.CapabilitySource, b.CapabilityKey, err)
			}
			if b.CapabilitySource == "builtin" {
				if err := tx.Create(&model.AgentToolBinding{
					AgentID:   id,
					ToolName:  b.CapabilityKey,
					IsEnabled: b.IsEnabled,
				}).Error; err != nil {
					return fmt.Errorf("create legacy tool binding %s: %w", b.CapabilityKey, err)
				}
			}
		}

		return nil
	})
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
		if err := tx.Where("agent_id = ? AND capability_source = ?", agentID, "builtin").Delete(&model.AgentCapabilityBinding{}).Error; err != nil {
			return fmt.Errorf("delete existing builtin capability bindings: %w", err)
		}
		for _, name := range toolNames {
			if err := tx.Create(&model.AgentToolBinding{
				AgentID:   agentID,
				ToolName:  name,
				IsEnabled: 1,
			}).Error; err != nil {
				return fmt.Errorf("create binding for %s: %w", name, err)
			}
			if err := tx.Create(&model.AgentCapabilityBinding{
				AgentID:          agentID,
				CapabilitySource: "builtin",
				CapabilityKey:    name,
				IsEnabled:        1,
			}).Error; err != nil {
				return fmt.Errorf("create capability binding for %s: %w", name, err)
			}
		}
		return nil
	})
}

func (r *AgentConfigRepo) ListCapabilityBindings(ctx context.Context, agentID int64) ([]model.AgentCapabilityBinding, error) {
	var list []model.AgentCapabilityBinding
	err := r.db.WithContext(ctx).
		Where("agent_id = ? AND is_enabled = 1", agentID).
		Order("priority ASC, id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		return list, nil
	}

	legacy, err := r.ListToolBindings(ctx, agentID)
	if err != nil {
		return nil, err
	}
	list = make([]model.AgentCapabilityBinding, 0, len(legacy))
	for _, b := range legacy {
		list = append(list, model.AgentCapabilityBinding{
			AgentID:          b.AgentID,
			CapabilitySource: "builtin",
			CapabilityKey:    b.ToolName,
			IsEnabled:        b.IsEnabled,
			CreatedAt:        b.CreatedAt,
			UpdatedAt:        b.CreatedAt,
		})
	}
	return list, nil
}

func (r *AgentConfigRepo) ListEnabledCapabilityBindingsByAgentType(ctx context.Context, agentType string) ([]model.AgentCapabilityBinding, error) {
	cfg, err := r.GetByAgentType(ctx, agentType)
	if err != nil {
		return nil, err
	}
	return r.ListCapabilityBindings(ctx, cfg.ID)
}

func (r *AgentConfigRepo) ReplaceCapabilityBindings(ctx context.Context, agentID int64, bindings []model.AgentCapabilityBinding) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ?", agentID).Delete(&model.AgentCapabilityBinding{}).Error; err != nil {
			return fmt.Errorf("delete existing capability bindings: %w", err)
		}
		if err := tx.Where("agent_id = ?", agentID).Delete(&model.AgentToolBinding{}).Error; err != nil {
			return fmt.Errorf("delete existing legacy tool bindings: %w", err)
		}

		for _, b := range bindings {
			b.AgentID = agentID
			if b.IsEnabled == 0 {
				b.IsEnabled = 1
			}
			if err := tx.Create(&b).Error; err != nil {
				return fmt.Errorf("create capability binding %s:%s: %w", b.CapabilitySource, b.CapabilityKey, err)
			}
			if b.CapabilitySource == "builtin" {
				if err := tx.Create(&model.AgentToolBinding{
					AgentID:   agentID,
					ToolName:  b.CapabilityKey,
					IsEnabled: b.IsEnabled,
				}).Error; err != nil {
					return fmt.Errorf("create legacy tool binding %s: %w", b.CapabilityKey, err)
				}
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
