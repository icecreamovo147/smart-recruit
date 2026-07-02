package service

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

type fakeEmbeddingProvider struct {
	vector EmbeddingVector
	err    error
}

func (f fakeEmbeddingProvider) EmbedText(context.Context, string) (EmbeddingVector, error) {
	return f.vector, f.err
}

func newEmbeddingTestService(t *testing.T, provider EmbeddingProvider) (*EmbeddingService, *repository.AIEmbeddingRepo) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AIEmbedding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := repository.NewAIEmbeddingRepo(db)
	return NewEmbeddingService(repo, provider), repo
}

func TestEmbeddingServiceUnavailableProviderRecordsFallbackState(t *testing.T) {
	svc, repo := newEmbeddingTestService(t, UnavailableEmbeddingProvider{})
	ctx := context.Background()

	row, err := svc.EmbedObject(ctx, EmbedObjectInput{
		ObjectType: "agent_skill",
		ObjectID:   42,
		Text:       "screen candidates",
	})
	if err != nil {
		t.Fatalf("EmbedObject: %v", err)
	}
	if row.Status != EmbeddingStatusUnavailable {
		t.Fatalf("expected unavailable status, got %q", row.Status)
	}
	if row.VectorJSON != nil || row.LastError == "" {
		t.Fatalf("expected no vector and explicit error, vector=%v error=%q", row.VectorJSON, row.LastError)
	}

	stored, err := repo.GetByObject(ctx, "agent_skill", 42, DefaultEmbeddingModel)
	if err != nil {
		t.Fatalf("GetByObject: %v", err)
	}
	if stored == nil || stored.Status != EmbeddingStatusUnavailable {
		t.Fatalf("expected stored unavailable row, got %#v", stored)
	}

	_, err = svc.Search(ctx, EmbeddingSearchInput{QueryText: "screen"})
	if !errors.Is(err, ErrEmbeddingProviderUnavailable) {
		t.Fatalf("expected provider unavailable search error, got %v", err)
	}
}

func TestEmbeddingServiceStoresExplicitVectorAndSearchesByCosineSimilarity(t *testing.T) {
	svc, _ := newEmbeddingTestService(t, UnavailableEmbeddingProvider{})
	ctx := context.Background()

	fixtures := []EmbedObjectInput{
		{ObjectType: "agent_skill", ObjectID: 1, ScopeType: "hr", ScopeID: 7, Text: "resume screening", Model: "local-test", Vector: []float64{1, 0, 0}},
		{ObjectType: "agent_skill", ObjectID: 2, ScopeType: "hr", ScopeID: 7, Text: "interview scheduling", Model: "local-test", Vector: []float64{0, 1, 0}},
		{ObjectType: "ai_memory", ObjectID: 3, ScopeType: "hr", ScopeID: 7, Text: "resume parsing", Model: "local-test", Vector: []float64{0.8, 0.2, 0}},
		{ObjectType: "agent_skill", ObjectID: 4, ScopeType: "job", ScopeID: 99, Text: "other scope", Model: "local-test", Vector: []float64{1, 0, 0}},
	}
	for _, fixture := range fixtures {
		if _, err := svc.EmbedObject(ctx, fixture); err != nil {
			t.Fatalf("EmbedObject fixture %d: %v", fixture.ObjectID, err)
		}
	}

	results, err := svc.Search(ctx, EmbeddingSearchInput{
		QueryVector: []float64{1, 0, 0},
		Model:       "local-test",
		ObjectTypes: []string{"agent_skill", "ai_memory"},
		ScopeType:   "hr",
		ScopeID:     7,
		Limit:       2,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Embedding.ObjectID != 1 || results[1].Embedding.ObjectID != 3 {
		t.Fatalf("unexpected order: %#v", results)
	}
	if results[0].Score <= results[1].Score {
		t.Fatalf("expected descending scores, got %.4f then %.4f", results[0].Score, results[1].Score)
	}
}

func TestEmbeddingServiceProviderVectorUpsertsByObjectModelAndHash(t *testing.T) {
	svc, repo := newEmbeddingTestService(t, fakeEmbeddingProvider{
		vector: EmbeddingVector{Model: "provider-test", Vector: []float64{0.25, 0.75}},
	})
	ctx := context.Background()

	first, err := svc.EmbedObject(ctx, EmbedObjectInput{
		ObjectType: "ai_memory",
		ObjectID:   11,
		Text:       "prefers concise summaries",
	})
	if err != nil {
		t.Fatalf("first EmbedObject: %v", err)
	}
	second, err := svc.EmbedObject(ctx, EmbedObjectInput{
		ObjectType:   "ai_memory",
		ObjectID:     11,
		Text:         "prefers concise summaries",
		MetadataJSON: `{"source":"test"}`,
	})
	if err != nil {
		t.Fatalf("second EmbedObject: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same row to be updated, got %d and %d", first.ID, second.ID)
	}
	stored, err := repo.GetByObject(ctx, "ai_memory", 11, "provider-test")
	if err != nil {
		t.Fatalf("GetByObject: %v", err)
	}
	if stored == nil || stored.Status != EmbeddingStatusReady || stored.MetadataJSON == nil || *stored.MetadataJSON != `{"source":"test"}` {
		t.Fatalf("unexpected stored row: %#v", stored)
	}
}
