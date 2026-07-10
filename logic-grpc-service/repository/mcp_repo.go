package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

// MCPRepo handles database operations for MCP servers and tool logs.
type MCPRepo struct {
	db *gorm.DB
}

func NewMCPRepo(db *gorm.DB) *MCPRepo {
	return &MCPRepo{db: db}
}

// ── MCPServer CRUD ──────────────────────────────────────────────────

func (r *MCPRepo) CreateServer(ctx context.Context, s *model.MCPServer) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *MCPRepo) UpdateServer(ctx context.Context, s *model.MCPServer) error {
	return r.db.WithContext(ctx).Model(s).Where("id = ?", s.ID).Updates(map[string]any{
		"name":            s.Name,
		"transport":       s.Transport,
		"command_or_url":  s.CommandOrURL,
		"args":            s.Args,
		"env_vars":        s.EnvVars,
		"timeout_seconds": s.TimeoutSeconds,
		"is_enabled":      s.IsEnabled,
	}).Error
}

func (r *MCPRepo) UpdateServerPartial(ctx context.Context, id int64, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.MCPServer{}).Where("id = ?", id).Updates(updates).Error
}

func (r *MCPRepo) DeleteServer(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.MCPServer{}).Error
}

func (r *MCPRepo) GetServerByID(ctx context.Context, id int64) (*model.MCPServer, error) {
	var s model.MCPServer
	err := r.db.WithContext(ctx).First(&s, id).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *MCPRepo) ListServers(ctx context.Context, page, pageSize int32) ([]model.MCPServer, int64, error) {
	var list []model.MCPServer
	var total int64
	query := r.db.WithContext(ctx).Model(&model.MCPServer{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Order("id ASC").Offset(offset).Limit(int(pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}
	return list, total, nil
}

// ListEnabledServers returns all enabled servers.
func (r *MCPRepo) ListEnabledServers(ctx context.Context) ([]model.MCPServer, error) {
	var list []model.MCPServer
	err := r.db.WithContext(ctx).Where("is_enabled = 1").Order("id ASC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ── MCPToolPolicy ───────────────────────────────────────────────────

func (r *MCPRepo) CreateToolPolicy(ctx context.Context, policy *model.MCPToolPolicy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}

func (r *MCPRepo) UpdateToolPolicy(ctx context.Context, policy *model.MCPToolPolicy) error {
	return r.db.WithContext(ctx).Model(policy).Where("id = ?", policy.ID).Updates(map[string]any{
		"server_id":                 policy.ServerID,
		"tool_name":                 policy.ToolName,
		"effect":                    policy.Effect,
		"risk_level":                policy.RiskLevel,
		"require_confirmation":      policy.RequireConfirmation,
		"allowed_roles_json":        policy.AllowedRolesJSON,
		"allowed_scopes_json":       policy.AllowedScopesJSON,
		"required_args_json":        policy.RequiredArgsJSON,
		"denied_args_json":          policy.DeniedArgsJSON,
		"arg_rules_json":            policy.ArgRulesJSON,
		"redact_fields_json":        policy.RedactFieldsJSON,
		"rate_limit_window_seconds": policy.RateLimitWindowSeconds,
		"rate_limit_max_calls":      policy.RateLimitMaxCalls,
		"is_enabled":                policy.IsEnabled,
		"updated_by_hr_id":          policy.UpdatedByHRID,
	}).Error
}

func (r *MCPRepo) DeleteToolPolicy(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.MCPToolPolicy{}).Error
}

func (r *MCPRepo) GetToolPolicyByID(ctx context.Context, id int64) (*model.MCPToolPolicy, error) {
	var policy model.MCPToolPolicy
	if err := r.db.WithContext(ctx).First(&policy, id).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *MCPRepo) GetEnabledToolPolicy(ctx context.Context, serverID int64, toolName string) (*model.MCPToolPolicy, error) {
	var policy model.MCPToolPolicy
	err := r.db.WithContext(ctx).
		Where("server_id = ? AND tool_name = ? AND is_enabled = 1", serverID, toolName).
		First(&policy).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *MCPRepo) ListToolPolicies(ctx context.Context, serverID int64, page, pageSize int32) ([]model.MCPToolPolicy, int64, error) {
	var list []model.MCPToolPolicy
	var total int64
	query := r.db.WithContext(ctx).Model(&model.MCPToolPolicy{})
	if serverID > 0 {
		query = query.Where("server_id = ?", serverID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Order("server_id ASC, tool_name ASC, id ASC").Offset(offset).Limit(int(pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}
	return list, total, nil
}

// ── MCPToolLog ──────────────────────────────────────────────────────

func (r *MCPRepo) CreateToolLog(ctx context.Context, log *model.MCPToolLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *MCPRepo) CountToolLogsSince(ctx context.Context, serverID int64, toolName string, since time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&model.MCPToolLog{}).
		Where("server_id = ? AND tool_name = ? AND created_at >= ?", serverID, toolName, since).
		Count(&total).Error
	return total, err
}

func (r *MCPRepo) ListToolLogsByServer(ctx context.Context, serverID int64, page, pageSize int32) ([]model.MCPToolLog, int64, error) {
	var list []model.MCPToolLog
	var total int64
	query := r.db.WithContext(ctx).Model(&model.MCPToolLog{}).Where("server_id = ?", serverID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Order("id DESC").Offset(offset).Limit(int(pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}
	return list, total, nil
}

func (r *MCPRepo) ListToolLogsBySession(ctx context.Context, sessionID int64, page, pageSize int32) ([]model.MCPToolLog, int64, error) {
	var list []model.MCPToolLog
	var total int64
	query := r.db.WithContext(ctx).Model(&model.MCPToolLog{}).Where("session_id = ?", sessionID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}
	offset := int((page - 1) * pageSize)
	if err := query.Order("id DESC").Offset(offset).Limit(int(pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}
	return list, total, nil
}
