package repository

import (
	"context"
	"testing"

	"logic-grpc-service/model"
)

func TestAgentCapabilityBindingsReplaceAndLegacyMirror(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentConfigRepo(db)
	ctx := context.Background()

	agent := &model.AgentConfig{
		Name:          "hr_agent",
		DisplayName:   "HR Agent",
		AgentType:     "hr_recruiting_agent",
		MaxIterations: 5,
		IsEnabled:     1,
	}
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	err := repo.ReplaceCapabilityBindings(ctx, agent.ID, []model.AgentCapabilityBinding{
		{CapabilitySource: "builtin", CapabilityKey: "search_jobs", IsEnabled: 1},
		{CapabilitySource: "mcp", CapabilityKey: "12:lookup_profile", IsEnabled: 1},
	})
	if err != nil {
		t.Fatalf("replace capability bindings: %v", err)
	}

	caps, err := repo.ListCapabilityBindings(ctx, agent.ID)
	if err != nil {
		t.Fatalf("list capability bindings: %v", err)
	}
	if len(caps) != 2 {
		t.Fatalf("expected 2 capability bindings, got %d", len(caps))
	}

	legacy, err := repo.ListToolBindings(ctx, agent.ID)
	if err != nil {
		t.Fatalf("list legacy tool bindings: %v", err)
	}
	if len(legacy) != 1 || legacy[0].ToolName != "search_jobs" {
		t.Fatalf("expected only builtin binding mirrored to legacy table, got %#v", legacy)
	}
}

func TestCreateWithCapabilityBindings_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentConfigRepo(db)
	ctx := context.Background()

	cfg := &model.AgentConfig{
		Name:          "tx-agent",
		DisplayName:   "TX Agent",
		AgentType:     "hr_recruiting_agent",
		MaxIterations: 5,
		IsEnabled:     1,
	}
	bindings := []model.AgentCapabilityBinding{
		{CapabilitySource: "builtin", CapabilityKey: "search_jobs", IsEnabled: 1},
		{CapabilitySource: "mcp", CapabilityKey: "1:lookup", IsEnabled: 1},
	}

	if err := repo.CreateWithCapabilityBindings(ctx, cfg, bindings); err != nil {
		t.Fatalf("CreateWithCapabilityBindings failed: %v", err)
	}
	if cfg.ID <= 0 {
		t.Fatal("expected non-zero agent ID after creation")
	}

	// Verify bindings were written
	caps, err := repo.ListCapabilityBindings(ctx, cfg.ID)
	if err != nil {
		t.Fatalf("ListCapabilityBindings failed: %v", err)
	}
	if len(caps) != 2 {
		t.Fatalf("expected 2 capability bindings, got %d", len(caps))
	}

	// Verify legacy mirror
	legacy, err := repo.ListToolBindings(ctx, cfg.ID)
	if err != nil {
		t.Fatalf("ListToolBindings failed: %v", err)
	}
	if len(legacy) != 1 || legacy[0].ToolName != "search_jobs" {
		t.Fatalf("expected 1 builtin legacy binding for 'search_jobs', got %#v", legacy)
	}
}

func TestCreateWithCapabilityBindings_ClearsDefaults(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentConfigRepo(db)
	ctx := context.Background()

	// Create first default
	first := &model.AgentConfig{
		Name:          "first-default",
		DisplayName:   "First Default",
		AgentType:     "test_agent",
		MaxIterations: 5,
		IsDefault:     1,
		IsEnabled:     1,
	}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first default: %v", err)
	}

	// Create second default for same type — should clear the first
	second := &model.AgentConfig{
		Name:          "second-default",
		DisplayName:   "Second Default",
		AgentType:     "test_agent",
		MaxIterations: 5,
		IsDefault:     1,
		IsEnabled:     1,
	}
	if err := repo.CreateWithCapabilityBindings(ctx, second, nil); err != nil {
		t.Fatalf("CreateWithCapabilityBindings failed: %v", err)
	}

	// Verify first is no longer default
	firstReloaded, err := repo.GetByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("get first default: %v", err)
	}
	if firstReloaded.IsDefault != 0 {
		t.Fatal("expected first default to be cleared")
	}
}

func TestUpdateWithCapabilityBindings_ClearsDefaults(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentConfigRepo(db)
	ctx := context.Background()

	// Create two agents of same type
	a1 := &model.AgentConfig{
		Name:          "agent-one",
		DisplayName:   "Agent One",
		AgentType:     "test_agent",
		MaxIterations: 5,
		IsDefault:     1,
		IsEnabled:     1,
	}
	if err := repo.Create(ctx, a1); err != nil {
		t.Fatalf("create agent one: %v", err)
	}

	a2 := &model.AgentConfig{
		Name:          "agent-two",
		DisplayName:   "Agent Two",
		AgentType:     "test_agent",
		MaxIterations: 5,
		IsEnabled:     1,
	}
	if err := repo.Create(ctx, a2); err != nil {
		t.Fatalf("create agent two: %v", err)
	}

	// Set agent-two as default — should clear agent-one's default
	updates := map[string]any{"is_default": int32(1)}
	if err := repo.UpdateWithCapabilityBindings(ctx, a2.ID, updates, nil, false); err != nil {
		t.Fatalf("UpdateWithCapabilityBindings failed: %v", err)
	}

	// Verify a1 is no longer default
	a1Reloaded, err := repo.GetByID(ctx, a1.ID)
	if err != nil {
		t.Fatalf("get agent one: %v", err)
	}
	if a1Reloaded.IsDefault != 0 {
		t.Fatal("expected agent one default to be cleared")
	}

	// Verify a2 is now default
	a2Reloaded, err := repo.GetByID(ctx, a2.ID)
	if err != nil {
		t.Fatalf("get agent two: %v", err)
	}
	if a2Reloaded.IsDefault != 1 {
		t.Fatal("expected agent two to be default")
	}
}

func TestCreateWithCapabilityBindings_RollsBackOnFailure(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentConfigRepo(db)
	ctx := context.Background()

	// Create an agent with a binding that has a duplicate unique key.
	// Two identical (capability_source, capability_key) pairs trigger
	// the uk_agent_capability unique constraint, causing the transaction
	// to roll back. Verify no agent_config or bindings remain.
	cfg := &model.AgentConfig{
		Name:          "rollback-agent",
		DisplayName:   "Rollback Agent",
		AgentType:     "hr_recruiting_agent",
		MaxIterations: 5,
		IsEnabled:     1,
	}
	bindings := []model.AgentCapabilityBinding{
		{CapabilitySource: "builtin", CapabilityKey: "search_jobs", IsEnabled: 1},
		{CapabilitySource: "builtin", CapabilityKey: "search_jobs", IsEnabled: 1}, // duplicate — triggers UK violation
	}

	err := repo.CreateWithCapabilityBindings(ctx, cfg, bindings)
	if err == nil {
		t.Fatal("expected error due to duplicate capability binding, but got nil")
	}

	// Verify no agent was persisted
	var agentCount int64
	if err := db.Model(&model.AgentConfig{}).Where("name = ?", "rollback-agent").Count(&agentCount).Error; err != nil {
		t.Fatalf("count agent configs: %v", err)
	}
	if agentCount != 0 {
		t.Fatalf("expected 0 agent configs after failed creation, got %d", agentCount)
	}

	// Verify no capability bindings were persisted
	var bindingCount int64
	if err := db.Model(&model.AgentCapabilityBinding{}).Count(&bindingCount).Error; err != nil {
		t.Fatalf("count capability bindings: %v", err)
	}
	if bindingCount != 0 {
		t.Fatalf("expected 0 capability bindings after failed creation, got %d", bindingCount)
	}

	// Verify no legacy tool bindings were persisted
	var legacyCount int64
	if err := db.Model(&model.AgentToolBinding{}).Count(&legacyCount).Error; err != nil {
		t.Fatalf("count legacy bindings: %v", err)
	}
	if legacyCount != 0 {
		t.Fatalf("expected 0 legacy tool bindings after failed creation, got %d", legacyCount)
	}
}

func TestUpdateWithCapabilityBindings_ReplacesBindings(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentConfigRepo(db)
	ctx := context.Background()

	cfg := &model.AgentConfig{
		Name:          "replace-agent",
		DisplayName:   "Replace Agent",
		AgentType:     "hr_recruiting_agent",
		MaxIterations: 5,
		IsEnabled:     1,
	}
	if err := repo.Create(ctx, cfg); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	// Set initial bindings
	initial := []model.AgentCapabilityBinding{
		{CapabilitySource: "builtin", CapabilityKey: "search_jobs", IsEnabled: 1},
	}
	if err := repo.UpdateWithCapabilityBindings(ctx, cfg.ID, nil, initial, true); err != nil {
		t.Fatalf("set initial bindings: %v", err)
	}

	// Replace with different bindings
	replacement := []model.AgentCapabilityBinding{
		{CapabilitySource: "mcp", CapabilityKey: "2:tool1", IsEnabled: 1},
	}
	if err := repo.UpdateWithCapabilityBindings(ctx, cfg.ID, nil, replacement, true); err != nil {
		t.Fatalf("replace bindings: %v", err)
	}

	caps, err := repo.ListCapabilityBindings(ctx, cfg.ID)
	if err != nil {
		t.Fatalf("ListCapabilityBindings: %v", err)
	}
	if len(caps) != 1 || caps[0].CapabilitySource != "mcp" || caps[0].CapabilityKey != "2:tool1" {
		t.Fatalf("expected only mcp binding, got %#v", caps)
	}
}

func TestAgentCapabilityBindingsFallbackToLegacyToolBindings(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentConfigRepo(db)
	ctx := context.Background()

	agent := &model.AgentConfig{
		Name:          "legacy_agent",
		DisplayName:   "Legacy Agent",
		AgentType:     "hr_recruiting_agent",
		MaxIterations: 5,
		IsEnabled:     1,
	}
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := db.WithContext(ctx).Create(&model.AgentToolBinding{
		AgentID:   agent.ID,
		ToolName:  "get_job_detail",
		IsEnabled: 1,
	}).Error; err != nil {
		t.Fatalf("create legacy binding: %v", err)
	}

	caps, err := repo.ListCapabilityBindings(ctx, agent.ID)
	if err != nil {
		t.Fatalf("list capability bindings: %v", err)
	}
	if len(caps) != 1 {
		t.Fatalf("expected 1 fallback capability, got %d", len(caps))
	}
	if caps[0].CapabilitySource != "builtin" || caps[0].CapabilityKey != "get_job_detail" {
		t.Fatalf("unexpected fallback capability: %#v", caps[0])
	}
}
