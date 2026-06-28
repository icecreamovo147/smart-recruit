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
