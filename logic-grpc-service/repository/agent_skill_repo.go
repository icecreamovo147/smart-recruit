package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"logic-grpc-service/model"
)

var (
	ErrAgentSkillDuplicateName    = errors.New("agent skill name already exists")
	ErrAgentSkillDuplicateVersion = errors.New("agent skill version already exists")
)

type AgentSkillRepo struct {
	db *gorm.DB
}

func NewAgentSkillRepo(db *gorm.DB) *AgentSkillRepo {
	return &AgentSkillRepo{db: db}
}

func (r *AgentSkillRepo) CreateSkill(ctx context.Context, skill *model.AgentSkill) error {
	if err := r.db.WithContext(ctx).Create(skill).Error; err != nil {
		return mapAgentSkillCreateError(err)
	}
	return nil
}

func (r *AgentSkillRepo) CreateSkillWithVersion(ctx context.Context, skill *model.AgentSkill, version *model.AgentSkillVersion, activate bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(skill).Error; err != nil {
			return fmt.Errorf("create agent skill: %w", mapAgentSkillCreateError(err))
		}
		version.SkillID = skill.ID
		if err := tx.Create(version).Error; err != nil {
			return fmt.Errorf("create agent skill version: %w", mapAgentSkillVersionCreateError(err))
		}
		if activate {
			if err := tx.Model(&model.AgentSkill{}).Where("id = ?", skill.ID).Update("current_version_id", version.ID).Error; err != nil {
				return fmt.Errorf("activate agent skill version: %w", err)
			}
			skill.CurrentVersionID = &version.ID
		}
		return nil
	})
}

func (r *AgentSkillRepo) UpdateSkillPartial(ctx context.Context, id int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.AgentSkill{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AgentSkillRepo) GetSkillByID(ctx context.Context, id int64) (*model.AgentSkill, error) {
	var skill model.AgentSkill
	if err := r.db.WithContext(ctx).First(&skill, id).Error; err != nil {
		return nil, err
	}
	return &skill, nil
}

func (r *AgentSkillRepo) ListSkills(ctx context.Context, page, pageSize int32, enabledOnly bool, keyword string) ([]model.AgentSkill, int64, error) {
	var list []model.AgentSkill
	var total int64
	query := r.db.WithContext(ctx).Model(&model.AgentSkill{})
	if enabledOnly {
		query = query.Where("is_enabled = 1")
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?", like, like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count agent skills: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Order("id ASC").Offset(offset).Limit(int(pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list agent skills: %w", err)
	}
	return list, total, nil
}

func (r *AgentSkillRepo) ListAvailable(ctx context.Context, page, pageSize int32) ([]model.AgentSkill, int64, error) {
	var list []model.AgentSkill
	var total int64
	query := r.db.WithContext(ctx).
		Model(&model.AgentSkill{}).
		Where("is_enabled = 1 AND is_manual_invocable = 1 AND current_version_id IS NOT NULL")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count available agent skills: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Order("id ASC").Offset(offset).Limit(int(pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list available agent skills: %w", err)
	}
	return list, total, nil
}

type AgentSkillRuntimeRecord struct {
	ID                   int64
	Name                 string
	DisplayName          string
	Description          string
	TriggerKeywords      string
	AgentType            string
	Category             string
	Scenario             string
	Priority             int32
	RiskLevel            string
	RequiredCapabilities string
	OutputSchema         string
	EvaluationCriteria   string
	SemanticTags         string
	IsManualInvocable    int32
	VersionID            int64
	Version              string
	SkillMD              string
	BodyMarkdown         string
}

func (r *AgentSkillRepo) ListEnabled(ctx context.Context) ([]AgentSkillRuntimeRecord, error) {
	var rows []AgentSkillRuntimeRecord
	err := r.db.WithContext(ctx).
		Table("agent_skills AS s").
		Select("s.id AS id, s.name AS name, s.display_name AS display_name, s.description AS description, s.trigger_keywords AS trigger_keywords, s.agent_type AS agent_type, s.category AS category, s.scenario AS scenario, s.priority AS priority, s.risk_level AS risk_level, s.required_capabilities AS required_capabilities, s.output_schema AS output_schema, s.evaluation_criteria AS evaluation_criteria, s.semantic_tags AS semantic_tags, s.is_manual_invocable AS is_manual_invocable, v.id AS version_id, v.version AS version, v.skill_md AS skill_md, v.body_markdown AS body_markdown").
		Joins("JOIN agent_skill_versions AS v ON v.id = s.current_version_id").
		Where("s.is_enabled = 1").
		Order("s.id ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *AgentSkillRepo) CreateVersion(ctx context.Context, version *model.AgentSkillVersion, activate bool, actorUserID *int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(version).Error; err != nil {
			return fmt.Errorf("create agent skill version: %w", mapAgentSkillVersionCreateError(err))
		}
		if activate {
			updates := map[string]any{
				"current_version_id": version.ID,
				"updated_at":         time.Now(),
			}
			if actorUserID != nil {
				updates["updated_by"] = actorUserID
			}
			if err := tx.Model(&model.AgentSkill{}).Where("id = ?", version.SkillID).Updates(updates).Error; err != nil {
				return fmt.Errorf("activate agent skill version: %w", err)
			}
		}
		return nil
	})
}

func (r *AgentSkillRepo) ListVersions(ctx context.Context, skillID int64) ([]model.AgentSkillVersion, error) {
	var list []model.AgentSkillVersion
	err := r.db.WithContext(ctx).Where("skill_id = ?", skillID).Order("id DESC").Find(&list).Error
	return list, err
}

func (r *AgentSkillRepo) GetVersionByID(ctx context.Context, versionID int64) (*model.AgentSkillVersion, error) {
	var version model.AgentSkillVersion
	if err := r.db.WithContext(ctx).First(&version, versionID).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *AgentSkillRepo) ActivateVersion(ctx context.Context, skillID, versionID int64, actorUserID *int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.AgentSkillVersion{}).Where("id = ? AND skill_id = ?", versionID, skillID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
		updates := map[string]any{
			"current_version_id": versionID,
			"updated_at":         time.Now(),
		}
		if actorUserID != nil {
			updates["updated_by"] = actorUserID
		}
		return tx.Model(&model.AgentSkill{}).Where("id = ?", skillID).Updates(updates).Error
	})
}

func mapAgentSkillCreateError(err error) error {
	if isDuplicateError(err) {
		return ErrAgentSkillDuplicateName
	}
	return err
}

func mapAgentSkillVersionCreateError(err error) error {
	if isDuplicateError(err) {
		return ErrAgentSkillDuplicateVersion
	}
	return err
}

func isDuplicateError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint") ||
		strings.Contains(strings.ToLower(err.Error()), "duplicate entry")
}
