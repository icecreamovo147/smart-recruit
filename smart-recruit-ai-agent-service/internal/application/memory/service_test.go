package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
)

type fakeMemoryRepo struct {
	recall        []domainmemory.Memory
	create        []domainmemory.Memory
	cleanupResult CleanupResult
}

func (f *fakeMemoryRepo) CreateMemory(_ context.Context, memory domainmemory.Memory) (domainmemory.Memory, error) {
	memory.ID = uint64(len(f.create) + 1)
	f.create = append(f.create, memory)
	return memory, nil
}

func (f *fakeMemoryRepo) GetMemory(_ context.Context, _ domainmemory.OwnerRole, _, _ uint64) (domainmemory.Memory, bool, error) {
	return domainmemory.Memory{}, false, nil
}

func (f *fakeMemoryRepo) ListMemories(_ context.Context, _ ListFilter) ([]domainmemory.Memory, int64, error) {
	return nil, 0, nil
}

func (f *fakeMemoryRepo) UpdateMemory(_ context.Context, memory domainmemory.Memory) (domainmemory.Memory, error) {
	return memory, nil
}

func (f *fakeMemoryRepo) RevokeMemory(_ context.Context, _ domainmemory.OwnerRole, _, _, _ uint64, _ string) error {
	return nil
}

func (f *fakeMemoryRepo) ListActiveForRecall(_ context.Context, filter RecallFilter) ([]domainmemory.Memory, error) {
	if filter.OwnerID == 0 {
		return nil, nil
	}
	out := make([]domainmemory.Memory, 0, len(f.recall))
	for _, memory := range f.recall {
		if memory.OwnerRole != filter.OwnerRole || memory.OwnerID != filter.OwnerID {
			continue
		}
		out = append(out, memory)
	}
	return out, nil
}

func (f *fakeMemoryRepo) ExpireAndCleanup(_ context.Context, _ time.Duration) (CleanupResult, error) {
	return f.cleanupResult, nil
}

func TestServiceRecallOwnerIsolation(t *testing.T) {
	repo := &fakeMemoryRepo{recall: []domainmemory.Memory{
		{ID: 1, OwnerRole: domainmemory.OwnerRoleHR, OwnerID: 7, Scope: domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0}, Content: "HR note", Importance: 0.8, Confidence: 0.8},
		{ID: 2, OwnerRole: domainmemory.OwnerRoleHR, OwnerID: 8, Scope: domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0}, Content: "other HR", Importance: 0.8, Confidence: 0.8},
	}}
	svc := NewService(repo, NewExtractor(nil), nil, Config{Enabled: true, InjectEnabled: true, MaxMemories: 5, MaxMemoryChars: 500, Ranking: domainmemory.DefaultRankingConfig()})
	result, err := svc.Recall(context.Background(), RecallRequest{
		OwnerRole: domainmemory.OwnerRoleHR,
		OwnerID:   7,
		Scopes:    []domainmemory.Scope{{Type: domainmemory.ScopeHR, ID: 0}},
		Query:     "note",
	})
	if err != nil {
		t.Fatalf("Recall() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Memory.ID != 1 {
		t.Fatalf("Recall() = %+v, want owner 7 memory only", result.Items)
	}
}

func TestServiceWriteDryRun(t *testing.T) {
	repo := &fakeMemoryRepo{}
	svc := NewService(repo, NewExtractor(nil), nil, Config{Enabled: true, WriteEnabled: false, Ranking: domainmemory.DefaultRankingConfig()})
	_, err := svc.Write(context.Background(), WriteRequest{
		OwnerRole:  domainmemory.OwnerRoleHR,
		OwnerID:    1,
		Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
		MemoryType: "preference",
		Content:    "偏好远程",
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if len(repo.create) != 0 {
		t.Fatalf("dry-run wrote %d memories", len(repo.create))
	}
}

func TestExtractorRememberRule(t *testing.T) {
	ext := NewExtractor(nil)
	items, err := ext.Extract(context.Background(), ExtractInput{
		OwnerRole: domainmemory.OwnerRoleHR,
		OwnerID:   3,
		UserText:  "请记住 更偏好线下面试",
	})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if len(items) == 0 || items[0].MemoryType != "preference" {
		t.Fatalf("Extract() = %+v, want preference memory", items)
	}
}

func TestServiceWriteRejectsHighPII(t *testing.T) {
	repo := &fakeMemoryRepo{}
	svc := NewService(repo, NewExtractor(nil), nil, Config{Enabled: true, WriteEnabled: true, Ranking: domainmemory.DefaultRankingConfig()})
	_, err := svc.Write(context.Background(), WriteRequest{
		OwnerRole:  domainmemory.OwnerRoleHR,
		OwnerID:    1,
		Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
		MemoryType: "fact",
		Content:    "call me at test@example.com",
	})
	if err == nil {
		t.Fatal("expected high PII rejection")
	}
	if len(repo.create) != 0 {
		t.Fatalf("persisted %d memories, want 0", len(repo.create))
	}
}

func TestServiceWriteAllowsHighPIIWithConfirmation(t *testing.T) {
	repo := &fakeMemoryRepo{}
	svc := NewService(repo, NewExtractor(nil), nil, Config{Enabled: true, WriteEnabled: true, Ranking: domainmemory.DefaultRankingConfig()})
	saved, err := svc.Write(context.Background(), WriteRequest{
		OwnerRole:      domainmemory.OwnerRoleHR,
		OwnerID:        1,
		Scope:          domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
		MemoryType:     "fact",
		Content:        "call me at test@example.com",
		ConfirmHighPII: true,
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if saved.ID == 0 || saved.PIILevel != domainmemory.PIILevelHigh {
		t.Fatalf("saved = %+v, want high PII memory", saved)
	}
	if len(repo.create) != 1 {
		t.Fatalf("persisted %d memories, want 1", len(repo.create))
	}
}

func TestServiceExpireAndCleanupDelegatesToRepo(t *testing.T) {
	repo := &fakeMemoryRepo{
		cleanupResult: CleanupResult{ExpiredArchived: 2, RevokedPurged: 1},
	}
	svc := NewService(repo, NewExtractor(nil), nil, Config{Enabled: true, Ranking: domainmemory.DefaultRankingConfig()})
	result, err := svc.ExpireAndCleanup(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatalf("ExpireAndCleanup() error = %v", err)
	}
	if result.ExpiredArchived != 2 || result.RevokedPurged != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestServiceRecallFiltersHighPIIForInject(t *testing.T) {
	repo := &fakeMemoryRepo{recall: []domainmemory.Memory{
		{ID: 1, OwnerRole: domainmemory.OwnerRoleHR, OwnerID: 7, Scope: domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0}, Content: "偏好远程", Importance: 0.8, Confidence: 0.8, PIILevel: domainmemory.PIILevelNone},
		{ID: 2, OwnerRole: domainmemory.OwnerRoleHR, OwnerID: 7, Scope: domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0}, Content: "phone 13800138000", Importance: 0.9, Confidence: 0.9, PIILevel: domainmemory.PIILevelHigh},
	}}
	svc := NewService(repo, NewExtractor(nil), nil, Config{Enabled: true, InjectEnabled: true, MaxMemories: 5, MaxMemoryChars: 500, Ranking: domainmemory.DefaultRankingConfig()})
	result, err := svc.Recall(context.Background(), RecallRequest{
		OwnerRole: domainmemory.OwnerRoleHR,
		OwnerID:   7,
		Scopes:    []domainmemory.Scope{{Type: domainmemory.ScopeHR, ID: 0}},
		Query:     "remote",
	})
	if err != nil {
		t.Fatalf("Recall() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Memory.ID != 1 {
		t.Fatalf("Recall() = %+v, want high PII filtered out", result.Items)
	}
	if strings.Contains(result.InjectText, "13800138000") {
		t.Fatalf("inject text leaked high PII: %s", result.InjectText)
	}
}
