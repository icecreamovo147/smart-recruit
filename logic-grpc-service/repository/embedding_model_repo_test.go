package repository

import (
	"context"
	"testing"

	"logic-grpc-service/model"
)

func setupEmbeddingModelTest(t *testing.T) (*EmbeddingModelRepo, *EmbeddingProviderRepo, context.Context) {
	t.Helper()
	db := setupTestDB(t)
	if err := db.AutoMigrate(
		&model.EmbeddingProviderConfig{},
		&model.EmbeddingModelConfig{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewEmbeddingModelRepo(db), NewEmbeddingProviderRepo(db), context.Background()
}

func createTestProvider(t *testing.T, repo *EmbeddingProviderRepo, ctx context.Context, name string) int64 {
	t.Helper()
	p := &model.EmbeddingProviderConfig{
		Name:            name,
		ProviderType:    "bailian_text_embedding",
		Endpoint:        "https://example.com/api",
		APIKeyEncrypted: "enc:key",
		IsEnabled:       1,
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create provider %s: %v", name, err)
	}
	return p.ID
}

func TestEmbeddingModelRepoCRUD(t *testing.T) {
	modelRepo, providerRepo, ctx := setupEmbeddingModelTest(t)
	providerID := createTestProvider(t, providerRepo, ctx, "test-provider")

	m := &model.EmbeddingModelConfig{
		ProviderID:      providerID,
		ModelName:       "text-embedding-v4",
		DisplayName:     "Text Embedding V4",
		EmbeddingDim:    1024,
		InputTokenLimit: 8192,
		BatchSize:       16,
		TimeoutSeconds:  30,
		MaxRetries:      2,
		IsEnabled:       1,
		IsDefault:       1,
	}
	if err := modelRepo.Create(ctx, m); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if m.ID == 0 {
		t.Fatal("expected ID after Create")
	}

	got, err := modelRepo.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ModelName != "text-embedding-v4" || got.EmbeddingDim != 1024 {
		t.Fatalf("unexpected model: %+v", got)
	}

	m.EmbeddingDim = 1536
	if err := modelRepo.Update(ctx, m); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = modelRepo.GetByID(ctx, m.ID)
	if got.EmbeddingDim != 1536 {
		t.Fatal("expected embedding_dim=1536 after update")
	}

	if err := modelRepo.Delete(ctx, m.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = modelRepo.GetByID(ctx, m.ID)
	if err == nil {
		t.Fatal("expected error after Delete")
	}
}

func TestEmbeddingModelRepoGetDefaultModel(t *testing.T) {
	modelRepo, providerRepo, ctx := setupEmbeddingModelTest(t)
	providerID := createTestProvider(t, providerRepo, ctx, "provider-default")

	for i, name := range []string{"model-a", "model-b", "model-c"} {
		isDefault := int32(0)
		if i == 1 {
			isDefault = 1
		}
		if err := modelRepo.Create(ctx, &model.EmbeddingModelConfig{
			ProviderID:      providerID,
			ModelName:       name,
			EmbeddingDim:    768,
			InputTokenLimit: 4096,
			IsEnabled:       1,
			IsDefault:       isDefault,
		}); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	defaultModel, err := modelRepo.GetDefaultModel(ctx)
	if err != nil {
		t.Fatalf("GetDefaultModel: %v", err)
	}
	if defaultModel.ModelName != "model-b" {
		t.Fatalf("expected model-b as default, got %s", defaultModel.ModelName)
	}
}

func TestEmbeddingModelRepoGetDefaultModelFallback(t *testing.T) {
	modelRepo, providerRepo, ctx := setupEmbeddingModelTest(t)
	providerID := createTestProvider(t, providerRepo, ctx, "provider-fallback")

	if err := modelRepo.Create(ctx, &model.EmbeddingModelConfig{
		ProviderID:      providerID,
		ModelName:       "only-model",
		EmbeddingDim:    768,
		InputTokenLimit: 4096,
		IsEnabled:       1,
		IsDefault:       0,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	m, err := modelRepo.GetDefaultModel(ctx)
	if err != nil {
		t.Fatalf("GetDefaultModel fallback: %v", err)
	}
	if m.ModelName != "only-model" {
		t.Fatalf("expected fallback to only-model, got %s", m.ModelName)
	}
}

func TestEmbeddingModelRepoGetEnabledModelsByProvider(t *testing.T) {
	modelRepo, providerRepo, ctx := setupEmbeddingModelTest(t)
	providerID := createTestProvider(t, providerRepo, ctx, "provider-gemp")

	for _, name := range []string{"m1", "m2", "m3"} {
		if err := modelRepo.Create(ctx, &model.EmbeddingModelConfig{
			ProviderID:      providerID,
			ModelName:       name,
			EmbeddingDim:    768,
			InputTokenLimit: 4096,
			IsEnabled:       1,
			IsDefault:       0,
		}); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}
	models, _, _ := modelRepo.List(ctx, 1, 10, providerID)
	for i := range models {
		if models[i].ModelName == "m3" {
			if err := modelRepo.UpdatePartial(ctx, models[i].ID, map[string]any{"is_enabled": 0}); err != nil {
				t.Fatalf("disable m3: %v", err)
			}
		}
	}

	enabled, err := modelRepo.GetEnabledModelsByProvider(ctx, providerID)
	if err != nil {
		t.Fatalf("GetEnabledModelsByProvider: %v", err)
	}
	if len(enabled) != 2 {
		t.Fatalf("expected 2 enabled models, got %d", len(enabled))
	}
}

func TestEmbeddingModelRepoClearDefault(t *testing.T) {
	modelRepo, providerRepo, ctx := setupEmbeddingModelTest(t)
	providerID := createTestProvider(t, providerRepo, ctx, "provider-clear")

	if err := modelRepo.Create(ctx, &model.EmbeddingModelConfig{
		ProviderID:      providerID,
		ModelName:       "default-model",
		EmbeddingDim:    768,
		InputTokenLimit: 4096,
		IsEnabled:       1,
		IsDefault:       1,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := modelRepo.ClearDefault(ctx); err != nil {
		t.Fatalf("ClearDefault: %v", err)
	}

	got, err := modelRepo.GetDefaultModel(ctx)
	if err != nil {
		t.Fatalf("GetDefaultModel: %v", err)
	}
	if got.IsDefault != 0 {
		t.Fatal("expected is_default=0 on fallback model after ClearDefault")
	}
}

func TestEmbeddingModelRepoList(t *testing.T) {
	modelRepo, providerRepo, ctx := setupEmbeddingModelTest(t)
	pid1 := createTestProvider(t, providerRepo, ctx, "p1")
	pid2 := createTestProvider(t, providerRepo, ctx, "p2")

	for _, name := range []string{"p1-a", "p1-b"} {
		if err := modelRepo.Create(ctx, &model.EmbeddingModelConfig{
			ProviderID:      pid1,
			ModelName:       name,
			EmbeddingDim:    768,
			InputTokenLimit: 4096,
			IsEnabled:       1,
		}); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}
	if err := modelRepo.Create(ctx, &model.EmbeddingModelConfig{
		ProviderID:      pid2,
		ModelName:       "p2-a",
		EmbeddingDim:    768,
		InputTokenLimit: 4096,
		IsEnabled:       1,
	}); err != nil {
		t.Fatalf("Create p2-a: %v", err)
	}

	all, total, err := modelRepo.List(ctx, 1, 10, 0)
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Fatalf("expected total=3, list=3; got total=%d, list=%d", total, len(all))
	}

	p1Models, total, err := modelRepo.List(ctx, 1, 10, pid1)
	if err != nil {
		t.Fatalf("List by provider: %v", err)
	}
	if total != 2 || len(p1Models) != 2 {
		t.Fatalf("expected total=2 for p1, got %d", total)
	}
}
