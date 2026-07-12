package repository

import (
	"context"
	"testing"

	"smart-recruit-domain-go/model"
)

func TestEmbeddingProviderRepoCRUD(t *testing.T) {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&model.EmbeddingProviderConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := NewEmbeddingProviderRepo(db)
	ctx := context.Background()

	p := &model.EmbeddingProviderConfig{
		Name:            "my-bailian",
		ProviderType:    "bailian_text_embedding",
		Endpoint:        "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding",
		APIKeyEncrypted: "enc:sk-test123",
		ExtraHeaders:    nil,
		IsEnabled:       1,
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == 0 {
		t.Fatal("expected ID after Create")
	}

	got, err := repo.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "my-bailian" || got.ProviderType != "bailian_text_embedding" {
		t.Fatalf("unexpected provider: %+v", got)
	}

	p.IsEnabled = 0
	if err := repo.Update(ctx, p); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = repo.GetByID(ctx, p.ID)
	if got.IsEnabled != 0 {
		t.Fatal("expected is_enabled=0 after update")
	}

	if err := repo.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, p.ID)
	if err == nil {
		t.Fatal("expected error after Delete")
	}
}

func TestEmbeddingProviderRepoListEnabled(t *testing.T) {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&model.EmbeddingProviderConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := NewEmbeddingProviderRepo(db)
	ctx := context.Background()

	for _, name := range []string{"provider-a", "provider-b", "provider-c"} {
		if err := repo.Create(ctx, &model.EmbeddingProviderConfig{
			Name:            name,
			ProviderType:    "bailian_text_embedding",
			Endpoint:        "https://example.com/api",
			APIKeyEncrypted: "enc:key",
			IsEnabled:       1,
		}); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}
	var p3id int64
	list, _, _ := repo.List(ctx, 1, 10)
	for i := range list {
		if list[i].Name == "provider-c" {
			p3id = list[i].ID
		}
	}
	if p3id == 0 {
		t.Fatal("provider-c not found")
	}
	if err := repo.UpdatePartial(ctx, p3id, map[string]any{"is_enabled": 0}); err != nil {
		t.Fatalf("disable provider-c: %v", err)
	}

	enabled, err := repo.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("ListEnabled: %v", err)
	}
	if len(enabled) != 2 {
		t.Fatalf("expected 2 enabled, got %d", len(enabled))
	}

	list, total, err := repo.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 || len(list) != 3 {
		t.Fatalf("expected total=3, list=3; got total=%d, list=%d", total, len(list))
	}
}

func TestEmbeddingProviderRepoFindByIDs(t *testing.T) {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&model.EmbeddingProviderConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := NewEmbeddingProviderRepo(db)
	ctx := context.Background()

	var ids []int64
	for _, name := range []string{"alpha", "beta"} {
		p := &model.EmbeddingProviderConfig{
			Name:            name,
			ProviderType:    "bailian_text_embedding",
			Endpoint:        "https://example.com/api",
			APIKeyEncrypted: "enc:key",
			IsEnabled:       1,
		}
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
		ids = append(ids, p.ID)
	}

	found, err := repo.FindByIDs(ctx, ids)
	if err != nil {
		t.Fatalf("FindByIDs: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("expected 2, got %d", len(found))
	}

	empty, err := repo.FindByIDs(ctx, nil)
	if err != nil {
		t.Fatalf("FindByIDs(nil): %v", err)
	}
	if empty != nil {
		t.Fatal("expected nil for empty ids")
	}
}
