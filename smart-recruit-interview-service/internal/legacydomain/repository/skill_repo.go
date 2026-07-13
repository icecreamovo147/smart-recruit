package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"smart-recruit-interview-service/internal/legacydomain/model"
)

type SkillRepo struct {
	db *gorm.DB
}

func NewSkillRepo(db *gorm.DB) *SkillRepo {
	return &SkillRepo{db: db}
}

func (r *SkillRepo) CreateSkill(ctx context.Context, skill *model.Skill) error {
	return r.db.WithContext(ctx).Create(skill).Error
}

func (r *SkillRepo) UpdateSkillPartial(ctx context.Context, id int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.Skill{}).Where("id = ?", id).Updates(updates).Error
}

func (r *SkillRepo) GetSkillByID(ctx context.Context, id int64) (*model.Skill, error) {
	var skill model.Skill
	if err := r.db.WithContext(ctx).First(&skill, id).Error; err != nil {
		return nil, err
	}
	return &skill, nil
}

func (r *SkillRepo) GetSkillByName(ctx context.Context, name string) (*model.Skill, error) {
	var skill model.Skill
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&skill).Error; err != nil {
		return nil, err
	}
	return &skill, nil
}

func (r *SkillRepo) ListSkills(ctx context.Context, page, pageSize int32) ([]model.Skill, int64, error) {
	var list []model.Skill
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Skill{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Order("id ASC").Offset(offset).Limit(int(pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}
	return list, total, nil
}

func (r *SkillRepo) CreateVersionWithTools(ctx context.Context, version *model.SkillVersion, tools []model.SkillTool, activate bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(version).Error; err != nil {
			return fmt.Errorf("create skill version: %w", err)
		}
		for i := range tools {
			tools[i].SkillVersionID = version.ID
			if err := tx.Create(&tools[i]).Error; err != nil {
				return fmt.Errorf("create skill tool %s: %w", tools[i].ToolName, err)
			}
		}
		if activate {
			if err := tx.Model(&model.Skill{}).Where("id = ?", version.SkillID).Update("current_version_id", version.ID).Error; err != nil {
				return fmt.Errorf("activate skill version: %w", err)
			}
		}
		return nil
	})
}

func (r *SkillRepo) ListVersions(ctx context.Context, skillID int64) ([]model.SkillVersion, error) {
	var list []model.SkillVersion
	err := r.db.WithContext(ctx).Where("skill_id = ?", skillID).Order("id DESC").Find(&list).Error
	return list, err
}

func (r *SkillRepo) GetVersionByID(ctx context.Context, versionID int64) (*model.SkillVersion, error) {
	var version model.SkillVersion
	if err := r.db.WithContext(ctx).First(&version, versionID).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *SkillRepo) GetToolByID(ctx context.Context, toolID int64) (*model.SkillTool, error) {
	var item model.SkillTool
	if err := r.db.WithContext(ctx).First(&item, toolID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *SkillRepo) ActivateVersion(ctx context.Context, skillID, versionID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.SkillVersion{}).Where("id = ? AND skill_id = ?", versionID, skillID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&model.Skill{}).Where("id = ?", skillID).Update("current_version_id", versionID).Error
	})
}

func (r *SkillRepo) ListTools(ctx context.Context, skillVersionID int64, enabledOnly bool) ([]model.SkillTool, error) {
	var list []model.SkillTool
	query := r.db.WithContext(ctx).Where("skill_version_id = ?", skillVersionID)
	if enabledOnly {
		query = query.Where("is_enabled = 1")
	}
	err := query.Order("id ASC").Find(&list).Error
	return list, err
}

func (r *SkillRepo) UpdateToolEnabled(ctx context.Context, toolID int64, enabled bool) error {
	value := int32(0)
	if enabled {
		value = 1
	}
	return r.db.WithContext(ctx).Model(&model.SkillTool{}).Where("id = ?", toolID).Update("is_enabled", value).Error
}

func (r *SkillRepo) UpdateToolPartial(ctx context.Context, toolID int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.SkillTool{}).Where("id = ?", toolID).Updates(updates).Error
}

func (r *SkillRepo) ListEnabledCurrentSkillTools(ctx context.Context) ([]SkillToolWithSkill, error) {
	var rows []SkillToolWithSkill
	err := r.db.WithContext(ctx).
		Table("ai_skill_tools AS t").
		Select("s.id AS skill_id, s.name AS skill_name, s.display_name AS skill_display_name, s.description AS skill_description, v.id AS skill_version_id, v.version AS version, v.instruction AS instruction, v.runtime_type AS runtime_type, t.id AS tool_id, t.tool_name AS tool_name, t.description AS tool_description, t.input_schema_json AS input_schema_json, t.runtime_config_json AS runtime_config_json").
		Joins("JOIN ai_skill_versions AS v ON v.id = t.skill_version_id").
		Joins("JOIN ai_skills AS s ON s.current_version_id = v.id").
		Where("s.is_enabled = 1 AND t.is_enabled = 1").
		Order("s.id ASC, t.id ASC").
		Scan(&rows).Error
	return rows, err
}

type SkillToolWithSkill struct {
	SkillID           int64
	SkillName         string
	SkillDisplayName  string
	SkillDescription  string
	SkillVersionID    int64
	Version           string
	Instruction       string
	RuntimeType       string
	ToolID            int64
	ToolName          string
	ToolDescription   string
	InputSchemaJSON   *string
	RuntimeConfigJSON *string
}
