package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/pkg/crypto"
	"smart-recruit-domain-go/repository"
)

func TestAgentContextBuilderMemoryRecallOrdersByScopeImportanceAndExpiry(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AIMemory{}, &model.AIEmbedding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	ctx := context.Background()
	memories := repository.NewMemoryRepo(db)
	embeddingSvc := NewEmbeddingService(repository.NewAIEmbeddingRepo(db), nil, crypto.EncryptionKey{})
	embeddingSvc.SetProviderForTest(fakeEmbeddingProvider{
		vector: EmbeddingVector{Model: "local-test", Vector: []float64{1, 0, 0}},
	})
	builder := (&AgentContextBuilder{memories: memories}).WithEmbeddingService(embeddingSvc)

	expiredAt := time.Now().Add(-time.Hour)
	fixtures := []model.AIMemory{
		{HrID: 7, ScopeType: "hr", ScopeID: 0, MemoryType: "preference", Content: "HR prefers concise summaries", Source: "user", Confidence: 0.9, Importance: 0.9},
		{HrID: 7, ScopeType: "job", ScopeID: 88, MemoryType: "fact", Content: "Job requires Go backend expertise", Source: "agent", Confidence: 0.8, Importance: 0.6},
		{HrID: 7, ScopeType: "application", ScopeID: 99, MemoryType: "warning", Content: "Candidate lacks Go production experience", Source: "tool", Confidence: 0.9, Importance: 1.0},
		{HrID: 7, ScopeType: "application", ScopeID: 99, MemoryType: "fact", Content: "Expired memory", Source: "tool", Confidence: 1.0, Importance: 1.0, ExpiresAt: &expiredAt},
	}
	for i := range fixtures {
		if err := memories.Create(ctx, &fixtures[i]); err != nil {
			t.Fatalf("create memory %d: %v", i, err)
		}
	}
	if _, err := embeddingSvc.EmbedObject(ctx, EmbedObjectInput{
		ObjectType: "ai_memory",
		ObjectID:   fixtures[0].ID,
		Text:       fixtures[0].Content,
		Model:      "local-test",
		Vector:     []float64{0, 1, 0},
	}); err != nil {
		t.Fatalf("embed hr memory: %v", err)
	}
	if _, err := embeddingSvc.EmbedObject(ctx, EmbedObjectInput{
		ObjectType: "ai_memory",
		ObjectID:   fixtures[2].ID,
		Text:       fixtures[2].Content,
		Model:      "local-test",
		Vector:     []float64{1, 0, 0},
	}); err != nil {
		t.Fatalf("embed application memory: %v", err)
	}
	for i := uint64(1000); i < 1025; i++ {
		if _, err := embeddingSvc.EmbedObject(ctx, EmbedObjectInput{
			ObjectType: "ai_memory",
			ObjectID:   i,
			Text:       "newer unrelated memory",
			Model:      "local-test",
			Vector:     []float64{1, 0, 0},
		}); err != nil {
			t.Fatalf("embed unrelated memory %d: %v", i, err)
		}
	}

	recalled := builder.retrieveMemories(ctx, AgentContextInput{
		HrID:           7,
		JobID:          88,
		ApplicationID:  99,
		CurrentMessage: "does this candidate have enough Go experience",
	})
	if len(recalled) != 3 {
		t.Fatalf("len(recalled) = %d, want 3: %#v", len(recalled), recalled)
	}
	if recalled[0].ID != fixtures[2].ID {
		t.Fatalf("first recalled = %#v, want application semantic/high-importance memory", recalled[0])
	}
	for _, memory := range recalled {
		if memory.Content == "Expired memory" {
			t.Fatalf("expired memory was recalled: %#v", recalled)
		}
	}
}
