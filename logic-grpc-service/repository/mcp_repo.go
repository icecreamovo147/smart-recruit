package repository

import (
	"context"
	"fmt"

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

// ── MCPToolLog ──────────────────────────────────────────────────────

func (r *MCPRepo) CreateToolLog(ctx context.Context, log *model.MCPToolLog) error {
	return r.db.WithContext(ctx).Create(log).Error
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
