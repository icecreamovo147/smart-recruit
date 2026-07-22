package persistence

import (
	"context"
	"time"

	appmemory "smart-recruit-ai-agent-service/internal/application/memory"
	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
)

type MemoryRepositoryAdapter struct {
	store *NativeStore
}

func NewMemoryRepositoryAdapter(store *NativeStore) *MemoryRepositoryAdapter {
	return &MemoryRepositoryAdapter{store: store}
}

func (r *MemoryRepositoryAdapter) CreateMemory(ctx context.Context, memory domainmemory.Memory) (domainmemory.Memory, error) {
	return r.store.CreateMemory(ctx, memory)
}

func (r *MemoryRepositoryAdapter) GetMemory(ctx context.Context, ownerRole domainmemory.OwnerRole, ownerID, id uint64) (domainmemory.Memory, bool, error) {
	return r.store.GetMemory(ctx, ownerRole, ownerID, id)
}

func (r *MemoryRepositoryAdapter) ListMemories(ctx context.Context, filter appmemory.ListFilter) ([]domainmemory.Memory, int64, error) {
	return r.store.ListMemories(ctx, MemoryListFilter{
		OwnerRole:  filter.OwnerRole,
		OwnerID:    filter.OwnerID,
		ScopeType:  filter.ScopeType,
		ScopeID:    filter.ScopeID,
		MemoryType: filter.MemoryType,
		Status:     filter.Status,
		PIILevel:   filter.PIILevel,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	})
}

func (r *MemoryRepositoryAdapter) UpdateMemory(ctx context.Context, memory domainmemory.Memory) (domainmemory.Memory, error) {
	return r.store.UpdateMemory(ctx, memory)
}

func (r *MemoryRepositoryAdapter) RevokeMemory(ctx context.Context, ownerRole domainmemory.OwnerRole, ownerID, id, revokedBy uint64, reason string) error {
	return r.store.RevokeMemory(ctx, ownerRole, ownerID, id, revokedBy, reason)
}

func (r *MemoryRepositoryAdapter) ListActiveForRecall(ctx context.Context, filter appmemory.RecallFilter) ([]domainmemory.Memory, error) {
	items, err := r.store.ListActiveForRecall(ctx, MemoryRecallFilter{
		OwnerRole:   filter.OwnerRole,
		OwnerID:     filter.OwnerID,
		Scopes:      filter.Scopes,
		Query:       filter.Query,
		Limit:       filter.Limit,
		MemoryTypes: filter.MemoryTypes,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domainmemory.Memory, 0, len(items))
	for _, item := range items {
		out = append(out, item.Memory)
	}
	return out, nil
}

func (r *MemoryRepositoryAdapter) ExpireAndCleanup(ctx context.Context, revokedRetention time.Duration) (appmemory.CleanupResult, error) {
	return r.store.ExpireAndCleanup(ctx, revokedRetention)
}

var _ appmemory.Repository = (*MemoryRepositoryAdapter)(nil)
