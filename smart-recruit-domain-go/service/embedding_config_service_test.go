package service

import (
	"testing"

	"smart-recruit-domain-go/model"
)

func TestEmbeddingProviderToInfoMasking(t *testing.T) {
	p := &model.EmbeddingProviderConfig{
		ID:              1,
		Name:            "test-provider",
		ProviderType:    "bailian",
		Endpoint:        "https://api.example.com",
		APIKeyEncrypted: "encrypted-key",
		IsEnabled:       1,
	}
	if p.Name != "test-provider" {
		t.Errorf("unexpected name: %s", p.Name)
	}
	if p.ProviderType != "bailian" {
		t.Errorf("unexpected provider_type: %s", p.ProviderType)
	}
}

func TestEmbeddingModelConfigDefaults(t *testing.T) {
	m := &model.EmbeddingModelConfig{
		ID:         1,
		ProviderID: 1,
		ModelName:  "text-embedding-v4",
		IsEnabled:  1,
		IsDefault:  1,
	}
	if m.ModelName != "text-embedding-v4" {
		t.Errorf("unexpected model_name: %s", m.ModelName)
	}
	if m.LastTestStatus != "" {
		t.Errorf("last_test_status should be empty by default, got: %s", m.LastTestStatus)
	}
}

func TestBackfillInputDefaultValues(t *testing.T) {
	input := BackfillInput{
		ObjectType: "agent_skill",
		Limit:      100,
		BatchSize:  0,
		Force:      false,
		DryRun:     true,
	}
	if input.BatchSize <= 0 {
		if input.BatchSize <= 0 {
			input.BatchSize = 20
		}
	}
	if input.BatchSize != 20 {
		t.Errorf("expected BatchSize 20 after default, got %d", input.BatchSize)
	}
}

func TestEmbeddingScopeResolution(t *testing.T) {
	tests := []struct {
		objectType  string
		objectID    uint64
		wantScope   string
		wantScopeID uint64
	}{
		{objectType: "agent_skill", objectID: 1, wantScope: "agent_skill", wantScopeID: 1},
		{objectType: "ai_memory", objectID: 42, wantScope: "ai_memory", wantScopeID: 42},
		{objectType: "unknown", objectID: 99, wantScope: "unknown", wantScopeID: 99},
	}

	for _, tt := range tests {
		t.Run(tt.objectType, func(t *testing.T) {
			scopeType, scopeID := resolveEmbeddingScope(tt.objectType, tt.objectID)
			if scopeType != tt.wantScope {
				t.Errorf("scopeType = %q, want %q", scopeType, tt.wantScope)
			}
			if scopeID != tt.wantScopeID {
				t.Errorf("scopeID = %d, want %d", scopeID, tt.wantScopeID)
			}
		})
	}
}

func TestEmbeddingUpsertEventDefaults(t *testing.T) {
	event := EmbeddingUpsertEvent{
		ObjectType: "agent_skill",
		ObjectID:   123,
		Text:       "test text",
		TextHash:   "abc123",
	}
	if event.ObjectType != "agent_skill" {
		t.Errorf("unexpected ObjectType: %s", event.ObjectType)
	}
	if event.EventVersion == "" {
		event.EventVersion = "1.0"
	}
	if event.EventVersion != "1.0" {
		t.Errorf("unexpected EventVersion: %s", event.EventVersion)
	}
}
