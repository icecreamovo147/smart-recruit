package persistence

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
)

func setupMemoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	schema := `
CREATE TABLE ai_memories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id INTEGER,
  owner_role INTEGER NOT NULL DEFAULT 2,
  owner_id INTEGER NOT NULL DEFAULT 0,
  hr_id INTEGER NOT NULL DEFAULT 0,
  scope_type TEXT NOT NULL,
  scope_id INTEGER NOT NULL DEFAULT 0,
  memory_type TEXT NOT NULL,
  content TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'agent',
  confidence REAL NOT NULL DEFAULT 1.0,
  importance REAL NOT NULL DEFAULT 1.0,
  status TEXT NOT NULL DEFAULT 'active',
  deleted_at DATETIME,
  content_hash TEXT,
  pii_level TEXT NOT NULL DEFAULT 'none',
  source_session_id INTEGER,
  source_message_id INTEGER,
  source_run_id INTEGER,
  created_by INTEGER,
  revoked_by INTEGER,
  revoke_reason TEXT,
  expires_at DATETIME,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);`
	if err := db.Exec(schema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func memoryTestTenantID() *uint64 {
	value := uint64(11)
	return &value
}

func memoryTestHROwner(ownerID uint64) domainmemory.OwnerKey {
	return domainmemory.OwnerKey{TenantID: memoryTestTenantID(), Role: domainmemory.OwnerRoleHR, ID: ownerID}
}

func TestCreateMemoryDedupByHash(t *testing.T) {
	store := NewNativeStore(setupMemoryTestDB(t))
	ctx := context.Background()
	first, err := store.CreateMemory(ctx, domainmemory.Memory{
		TenantID:   memoryTestTenantID(),
		OwnerRole:  domainmemory.OwnerRoleHR,
		OwnerID:    9,
		Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
		MemoryType: "preference",
		Content:    "偏好线下面试",
		Confidence: 0.8,
		Importance: 0.7,
	})
	if err != nil {
		t.Fatalf("CreateMemory() first error = %v", err)
	}
	second, err := store.CreateMemory(ctx, domainmemory.Memory{
		TenantID:   memoryTestTenantID(),
		OwnerRole:  domainmemory.OwnerRoleHR,
		OwnerID:    9,
		Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
		MemoryType: "preference",
		Content:    "  偏好线下面试  ",
		Confidence: 0.95,
		Importance: 0.9,
	})
	if err != nil {
		t.Fatalf("CreateMemory() second error = %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("dedup ids = %d vs %d", first.ID, second.ID)
	}
	if second.Confidence != 0.95 {
		t.Fatalf("updated confidence = %v, want 0.95", second.Confidence)
	}
}

func TestListActiveForRecallOwnerIsolation(t *testing.T) {
	store := NewNativeStore(setupMemoryTestDB(t))
	ctx := context.Background()
	for _, ownerID := range []uint64{1, 2} {
		_, err := store.CreateMemory(ctx, domainmemory.Memory{
			TenantID:   memoryTestTenantID(),
			OwnerRole:  domainmemory.OwnerRoleHR,
			OwnerID:    ownerID,
			Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
			MemoryType: "fact",
			Content:    "memory for owner",
			Confidence: 0.8,
			Importance: 0.8,
		})
		if err != nil {
			t.Fatalf("CreateMemory() error = %v", err)
		}
	}
	items, err := store.ListActiveForRecall(ctx, MemoryRecallFilter{
		TenantID:  memoryTestTenantID(),
		OwnerRole: domainmemory.OwnerRoleHR,
		OwnerID:   1,
		Scopes:    []domainmemory.Scope{{Type: domainmemory.ScopeHR, ID: 0}},
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListActiveForRecall() error = %v", err)
	}
	if len(items) != 1 || items[0].Memory.OwnerID != 1 {
		t.Fatalf("recall = %+v, want single owner=1 memory", items)
	}
}

func TestMemoryTenantIsolationForSameHROwner(t *testing.T) {
	store := NewNativeStore(setupMemoryTestDB(t))
	ctx := context.Background()
	tenantOne := uint64(11)
	tenantTwo := uint64(12)
	create := func(tenantID *uint64, content string) domainmemory.Memory {
		memory, err := store.CreateMemory(ctx, domainmemory.Memory{
			TenantID: tenantID, OwnerRole: domainmemory.OwnerRoleHR, OwnerID: 7,
			Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
			MemoryType: "preference", Content: content,
		})
		if err != nil {
			t.Fatalf("CreateMemory() error = %v", err)
		}
		return memory
	}
	first := create(&tenantOne, "same preference")
	second := create(&tenantTwo, "same preference")
	if first.ID == second.ID {
		t.Fatal("different tenants unexpectedly deduplicated the same HR owner")
	}
	ownerOne := domainmemory.OwnerKey{TenantID: &tenantOne, Role: domainmemory.OwnerRoleHR, ID: 7}
	ownerTwo := domainmemory.OwnerKey{TenantID: &tenantTwo, Role: domainmemory.OwnerRoleHR, ID: 7}
	if _, found, err := store.GetMemory(ctx, ownerTwo, first.ID); err != nil || found {
		t.Fatalf("tenant two read tenant one memory: found=%v err=%v", found, err)
	}
	items, _, err := store.ListMemories(ctx, MemoryListFilter{
		TenantID: &tenantOne, OwnerRole: domainmemory.OwnerRoleHR, OwnerID: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != first.ID {
		t.Fatalf("tenant one list = %+v", items)
	}
	if err := store.RevokeMemory(ctx, ownerTwo, first.ID, 7, "cross tenant"); err == nil {
		t.Fatal("cross-tenant revoke unexpectedly succeeded")
	}
	if got, found, err := store.GetMemory(ctx, ownerOne, first.ID); err != nil || !found || got.Status != domainmemory.StatusActive {
		t.Fatalf("tenant one memory changed after cross-tenant revoke: %+v found=%v err=%v", got, found, err)
	}
}

func TestRevokeMemorySoftDelete(t *testing.T) {
	store := NewNativeStore(setupMemoryTestDB(t))
	ctx := context.Background()
	created, err := store.CreateMemory(ctx, domainmemory.Memory{
		OwnerRole:  domainmemory.OwnerRoleCandidate,
		OwnerID:    42,
		Scope:      domainmemory.Scope{Type: domainmemory.ScopeUser, ID: 42},
		MemoryType: "preference",
		Content:    "偏好远程",
	})
	if err != nil {
		t.Fatalf("CreateMemory() error = %v", err)
	}
	candidateOwner := domainmemory.OwnerKey{Role: domainmemory.OwnerRoleCandidate, ID: 42}
	if err := store.RevokeMemory(ctx, candidateOwner, created.ID, 42, "user request"); err != nil {
		t.Fatalf("RevokeMemory() error = %v", err)
	}
	got, found, err := store.GetMemory(ctx, candidateOwner, created.ID)
	if err != nil || !found {
		t.Fatalf("GetMemory() = %v found=%v err=%v", got, found, err)
	}
	if got.Status != domainmemory.StatusRevoked || got.DeletedAt == nil {
		t.Fatalf("revoked memory = %+v", got)
	}
	active, err := store.ListActiveForRecall(ctx, MemoryRecallFilter{
		OwnerRole: domainmemory.OwnerRoleCandidate,
		OwnerID:   42,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListActiveForRecall() error = %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("active recall after revoke = %d", len(active))
	}
}

func TestCreateMemoryRejectsHighPII(t *testing.T) {
	store := NewNativeStore(setupMemoryTestDB(t))
	_, err := store.CreateMemory(context.Background(), domainmemory.Memory{
		TenantID:   memoryTestTenantID(),
		OwnerRole:  domainmemory.OwnerRoleHR,
		OwnerID:    1,
		Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
		MemoryType: "fact",
		Content:    "call me at test@example.com",
	})
	if err == nil {
		t.Fatal("expected high PII rejection")
	}
}

func TestExpiredMemoryExcludedFromRecall(t *testing.T) {
	store := NewNativeStore(setupMemoryTestDB(t))
	ctx := context.Background()
	past := time.Now().Add(-time.Hour)
	_, err := store.CreateMemory(ctx, domainmemory.Memory{
		TenantID:   memoryTestTenantID(),
		OwnerRole:  domainmemory.OwnerRoleHR,
		OwnerID:    5,
		Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
		MemoryType: "fact",
		Content:    "expired fact",
		ExpiresAt:  &past,
	})
	if err != nil {
		t.Fatalf("CreateMemory() error = %v", err)
	}
	items, err := store.ListActiveForRecall(ctx, MemoryRecallFilter{TenantID: memoryTestTenantID(), OwnerRole: domainmemory.OwnerRoleHR, OwnerID: 5, Limit: 10})
	if err != nil {
		t.Fatalf("ListActiveForRecall() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expired memory recalled: %+v", items)
	}
}

func TestExpireAndCleanupArchivesExpiredAndPurgesRevoked(t *testing.T) {
	store := NewNativeStore(setupMemoryTestDB(t))
	ctx := context.Background()
	past := time.Now().Add(-time.Hour)
	oldRevoked := time.Now().Add(-40 * 24 * time.Hour)
	expired, err := store.CreateMemory(ctx, domainmemory.Memory{
		TenantID:   memoryTestTenantID(),
		OwnerRole:  domainmemory.OwnerRoleHR,
		OwnerID:    11,
		Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
		MemoryType: "fact",
		Content:    "expired cleanup",
		ExpiresAt:  &past,
	})
	if err != nil {
		t.Fatalf("CreateMemory() expired error = %v", err)
	}
	revoked, err := store.CreateMemory(ctx, domainmemory.Memory{
		TenantID:   memoryTestTenantID(),
		OwnerRole:  domainmemory.OwnerRoleHR,
		OwnerID:    11,
		Scope:      domainmemory.Scope{Type: domainmemory.ScopeHR, ID: 0},
		MemoryType: "fact",
		Content:    "revoked cleanup",
	})
	if err != nil {
		t.Fatalf("CreateMemory() revoked error = %v", err)
	}
	hrOwner := memoryTestHROwner(11)
	if err := store.RevokeMemory(ctx, hrOwner, revoked.ID, 11, "cleanup test"); err != nil {
		t.Fatalf("RevokeMemory() error = %v", err)
	}
	if err := store.db.WithContext(ctx).Model(&aiMemoryRecord{}).Where("id = ?", revoked.ID).
		Update("deleted_at", oldRevoked).Error; err != nil {
		t.Fatalf("backdate deleted_at: %v", err)
	}
	result, err := store.ExpireAndCleanup(ctx, 30*24*time.Hour)
	if err != nil {
		t.Fatalf("ExpireAndCleanup() error = %v", err)
	}
	if result.ExpiredArchived != 1 || result.RevokedPurged != 1 {
		t.Fatalf("cleanup result = %+v", result)
	}
	archived, found, err := store.GetMemory(ctx, hrOwner, expired.ID)
	if err != nil || !found || archived.Status != domainmemory.StatusArchived {
		t.Fatalf("archived memory = %+v found=%v err=%v", archived, found, err)
	}
	_, found, err = store.GetMemory(ctx, hrOwner, revoked.ID)
	if err != nil || found {
		t.Fatalf("purged memory should be gone: found=%v err=%v", found, err)
	}
}
