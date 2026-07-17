package persistence

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-proto/recruitment/pb"
)

func TestEnsureDefaultCandidateAssistantSeedsOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&agentConfigRecord{}, &agentToolBindingRecord{}, &agentCapabilityBindingRecord{}, &promptTemplateRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewNativeStore(db)
	ctx := context.Background()

	if err := store.EnsureDefaultCandidateAssistant(ctx); err != nil {
		t.Fatalf("EnsureDefaultCandidateAssistant: %v", err)
	}
	resp, err := store.GetAgentConfig(ctx, &pb.GetAgentConfigRequest{AgentType: "candidate_assistant"})
	if err != nil {
		t.Fatalf("GetAgentConfig: %v", err)
	}
	if resp.GetCode() != 0 || resp.GetAgent() == nil || resp.GetAgent().GetAgentType() != "candidate_assistant" {
		t.Fatalf("agent response = %#v", resp)
	}
	if resp.GetAgent().GetMaxIterations() != 5 {
		t.Fatalf("max_iterations = %d, want 5", resp.GetAgent().GetMaxIterations())
	}
	if len(resp.GetAgent().GetToolBindings()) != 6 {
		t.Fatalf("tool bindings = %d, want 6", len(resp.GetAgent().GetToolBindings()))
	}

	var count int64
	if err := db.Model(&agentConfigRecord{}).Where("agent_type = ?", "candidate_assistant").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("agents = %d, want 1", count)
	}

	// Second call is no-op
	if err := store.EnsureDefaultCandidateAssistant(ctx); err != nil {
		t.Fatalf("second Ensure: %v", err)
	}
	if err := db.Model(&agentConfigRecord{}).Where("agent_type = ?", "candidate_assistant").Count(&count).Error; err != nil {
		t.Fatalf("recount: %v", err)
	}
	if count != 1 {
		t.Fatalf("after second seed agents = %d, want 1", count)
	}
}

func TestSessionSummaryUpsertAndGet(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&aiSessionSummaryRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewNativeStore(db)
	ctx := context.Background()

	if err := store.UpsertSessionSummary(ctx, 55, 1001, "first summary", 10, 15); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, found, err := store.GetSessionSummary(ctx, 55, 1001)
	if err != nil || !found || got != "first summary" {
		t.Fatalf("get = (%q, %v, %v)", got, found, err)
	}
	if err := store.UpsertSessionSummary(ctx, 55, 1001, "updated summary", 20, 16); err != nil {
		t.Fatalf("upsert2: %v", err)
	}
	got, found, err = store.GetSessionSummary(ctx, 55, 1001)
	if err != nil || !found || got != "updated summary" {
		t.Fatalf("get2 = (%q, %v, %v)", got, found, err)
	}
	_, found, err = store.GetSessionSummary(ctx, 99, 1001)
	if err != nil || found {
		t.Fatalf("wrong owner found=%v err=%v", found, err)
	}
}
