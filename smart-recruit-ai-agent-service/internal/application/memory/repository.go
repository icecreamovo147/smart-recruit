package memory

import (
	"context"
	"time"

	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
)

type Repository interface {
	CreateMemory(ctx context.Context, memory domainmemory.Memory) (domainmemory.Memory, error)
	GetMemory(ctx context.Context, ownerRole domainmemory.OwnerRole, ownerID, id uint64) (domainmemory.Memory, bool, error)
	ListMemories(ctx context.Context, filter ListFilter) ([]domainmemory.Memory, int64, error)
	UpdateMemory(ctx context.Context, memory domainmemory.Memory) (domainmemory.Memory, error)
	RevokeMemory(ctx context.Context, ownerRole domainmemory.OwnerRole, ownerID, id, revokedBy uint64, reason string) error
	ListActiveForRecall(ctx context.Context, filter RecallFilter) ([]domainmemory.Memory, error)
	ExpireAndCleanup(ctx context.Context, revokedRetention time.Duration) (CleanupResult, error)
}

type CleanupResult struct {
	ExpiredArchived       int64
	RevokedPurged         int64
	EmbeddingsInvalidated int64
	InvalidatedMemoryIDs  []uint64
}

type ListFilter struct {
	OwnerRole  domainmemory.OwnerRole
	OwnerID    uint64
	ScopeType  string
	ScopeID    uint64
	MemoryType string
	Status     domainmemory.Status
	PIILevel   domainmemory.PIILevel
	Page       int
	PageSize   int
}

type RecallFilter struct {
	OwnerRole   domainmemory.OwnerRole
	OwnerID     uint64
	Scopes      []domainmemory.Scope
	Query       string
	Limit       int
	MemoryTypes []string
}

type EmbeddingUpserter interface {
	UpsertMemoryEmbedding(ctx context.Context, memory domainmemory.Memory) error
}

type MemoryEmbeddingInvalidator interface {
	InvalidateMemoryEmbedding(ctx context.Context, memoryID uint64) error
}

type ModelExtractor interface {
	Extract(ctx context.Context, input ExtractInput) ([]ExtractCandidate, error)
}

type ExtractInput struct {
	OwnerRole domainmemory.OwnerRole
	OwnerID   uint64
	UserText  string
	ReplyText string
}

type ExtractCandidate struct {
	Scope      domainmemory.Scope
	MemoryType string
	Content    string
	Confidence float64
	Importance float64
	Source     string
}
