package repository

import (
	"context"
	"testing"

	"logic-grpc-service/model"
)

func TestAIEmbeddingRepoUpsertGetAndQuery(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAIEmbeddingRepo(db)
	ctx := context.Background()

	row := &model.AIEmbedding{
		ObjectType:     "agent_skill",
		ObjectID:       10,
		ScopeType:      "hr",
		ScopeID:        7,
		TextHash:       "hash-v1",
		EmbeddingModel: "test-model",
		EmbeddingDim:   3,
		VectorJSON:     testStringPtr(`[1,0,0]`),
		MetadataJSON:   testStringPtr(`{"name":"screening"}`),
		Status:         "ready",
	}
	if err := repo.Upsert(ctx, row); err != nil {
		t.Fatalf("Upsert insert: %v", err)
	}
	if row.ID == 0 {
		t.Fatal("expected inserted ID")
	}

	row.VectorJSON = testStringPtr(`[0.5,0.5,0]`)
	row.MetadataJSON = testStringPtr(`{"name":"screening","version":2}`)
	if err := repo.Upsert(ctx, row); err != nil {
		t.Fatalf("Upsert update: %v", err)
	}

	got, err := repo.GetByObject(ctx, "agent_skill", 10, "test-model")
	if err != nil {
		t.Fatalf("GetByObject: %v", err)
	}
	if got == nil {
		t.Fatal("expected embedding")
	}
	if got.ID != row.ID || got.VectorJSON == nil || *got.VectorJSON != `[0.5,0.5,0]` {
		t.Fatalf("unexpected upserted row: id=%d vector=%v", got.ID, got.VectorJSON)
	}

	if err := repo.Upsert(ctx, &model.AIEmbedding{
		ObjectType:     "ai_memory",
		ObjectID:       99,
		ScopeType:      "job",
		ScopeID:        123,
		TextHash:       "hash-memory",
		EmbeddingModel: "test-model",
		EmbeddingDim:   3,
		VectorJSON:     testStringPtr(`[0,1,0]`),
		Status:         "ready",
	}); err != nil {
		t.Fatalf("Upsert second row: %v", err)
	}

	rows, err := repo.ListCandidates(ctx, AIEmbeddingQuery{
		ObjectTypes:    []string{"agent_skill"},
		ScopeType:      "hr",
		ScopeID:        7,
		EmbeddingModel: "test-model",
		Status:         "ready",
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("ListCandidates: %v", err)
	}
	if len(rows) != 1 || rows[0].ObjectType != "agent_skill" || rows[0].ObjectID != 10 {
		t.Fatalf("unexpected candidates: %#v", rows)
	}
}

func testStringPtr(value string) *string {
	return &value
}
