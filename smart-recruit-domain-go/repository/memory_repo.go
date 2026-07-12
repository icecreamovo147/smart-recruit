package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
)

type MemoryRepo struct {
	db *gorm.DB
}

func NewMemoryRepo(db *gorm.DB) *MemoryRepo {
	return &MemoryRepo{db: db}
}

func (r *MemoryRepo) Create(ctx context.Context, memory *model.AIMemory) error {
	return r.db.WithContext(ctx).Create(memory).Error
}

// ListRelevant retrieves unexpired long-term memories matching the given scope and types.
func (r *MemoryRepo) ListRelevant(ctx context.Context, hrID int64, scopeType string, scopeID int64, memoryTypes []string, limit int) ([]model.AIMemory, error) {
	query := r.db.WithContext(ctx).Model(&model.AIMemory{}).
		Where("hr_id = ? AND scope_type = ? AND scope_id = ?", hrID, scopeType, scopeID).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now())
	if len(memoryTypes) > 0 {
		query = query.Where("memory_type IN ?", memoryTypes)
	}
	var rows []model.AIMemory
	err := query.Order("importance DESC, confidence DESC, created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

// ListByHR retrieves all HR-level memories (scope_type = 'hr') for the given HR.
func (r *MemoryRepo) ListByHR(ctx context.Context, hrID int64, limit int) ([]model.AIMemory, error) {
	var rows []model.AIMemory
	err := r.db.WithContext(ctx).Model(&model.AIMemory{}).
		Where("hr_id = ? AND scope_type = ?", hrID, "hr").
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Order("importance DESC, confidence DESC, created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

type MemoryRecallScope struct {
	ScopeType string
	ScopeID   uint64
}

// ListRecallCandidates retrieves unexpired memories for multiple scopes.
func (r *MemoryRepo) ListRecallCandidates(ctx context.Context, hrID int64, scopes []MemoryRecallScope, memoryTypes []string, limit int) ([]model.AIMemory, error) {
	query := r.db.WithContext(ctx).Model(&model.AIMemory{}).
		Where("hr_id = ?", hrID).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now())
	if len(scopes) > 0 {
		query = query.Where(memoryScopePredicate(scopes), memoryScopeValues(scopes)...)
	}
	if len(memoryTypes) > 0 {
		query = query.Where("memory_type IN ?", memoryTypes)
	}
	if limit <= 0 {
		limit = 20
	}
	var rows []model.AIMemory
	err := query.Order("importance DESC, confidence DESC, created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func memoryScopePredicate(scopes []MemoryRecallScope) string {
	predicate := ""
	for range scopes {
		if predicate != "" {
			predicate += " OR "
		}
		predicate += "(scope_type = ? AND scope_id = ?)"
	}
	if predicate == "" {
		return "1 = 1"
	}
	return "(" + predicate + ")"
}

func memoryScopeValues(scopes []MemoryRecallScope) []any {
	values := make([]any, 0, len(scopes)*2)
	for _, scope := range scopes {
		values = append(values, scope.ScopeType, scope.ScopeID)
	}
	return values
}
