package repository

import (
	"context"

	"gorm.io/gorm"

	"logic-grpc-service/model"
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

// ListRelevant retrieves long-term memories matching the given scope and types, ordered by recency.
func (r *MemoryRepo) ListRelevant(ctx context.Context, hrID int64, scopeType string, scopeID int64, memoryTypes []string, limit int) ([]model.AIMemory, error) {
	query := r.db.WithContext(ctx).Model(&model.AIMemory{}).
		Where("hr_id = ? AND scope_type = ? AND scope_id = ?", hrID, scopeType, scopeID)
	if len(memoryTypes) > 0 {
		query = query.Where("memory_type IN ?", memoryTypes)
	}
	var rows []model.AIMemory
	err := query.Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

// ListByHR retrieves all HR-level memories (scope_type = 'hr') for the given HR.
func (r *MemoryRepo) ListByHR(ctx context.Context, hrID int64, limit int) ([]model.AIMemory, error) {
	var rows []model.AIMemory
	err := r.db.WithContext(ctx).Model(&model.AIMemory{}).
		Where("hr_id = ? AND scope_type = ?", hrID, "hr").
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}
