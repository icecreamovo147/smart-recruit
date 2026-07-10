package repository

import (
	"context"
	"testing"

	"logic-grpc-service/model"
)

func setupLlmModelTest(t *testing.T) (*ModelConfigRepo, *ProviderRepo, context.Context) {
	t.Helper()
	db := setupTestDB(t)
	if err := db.AutoMigrate(&model.LlmProvider{}, &model.LlmModel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewModelConfigRepo(db), NewProviderRepo(db), context.Background()
}

func createTestLlmProvider(t *testing.T, repo *ProviderRepo, ctx context.Context, name string) int64 {
	t.Helper()
	p := &model.LlmProvider{
		Name:            name,
		BaseURL:         "https://example.com/v1",
		APIKeyEncrypted: "enc:key",
		ProviderType:    "openai_compatible",
		IsEnabled:       1,
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create provider %s: %v", name, err)
	}
	return p.ID
}

func TestModelConfigRepoClearDefaultGlobal(t *testing.T) {
	modelRepo, providerRepo, ctx := setupLlmModelTest(t)
	pidA := createTestLlmProvider(t, providerRepo, ctx, "provider-a")
	pidB := createTestLlmProvider(t, providerRepo, ctx, "provider-b")

	modelA := &model.LlmModel{
		ProviderID: pidA,
		ModelName:  "model-a",
		MaxTokens:  4096,
		IsEnabled:  1,
		IsDefault:  1,
	}
	modelB := &model.LlmModel{
		ProviderID: pidB,
		ModelName:  "model-b",
		MaxTokens:  4096,
		IsEnabled:  1,
		IsDefault:  1,
	}
	if err := modelRepo.Create(ctx, modelA); err != nil {
		t.Fatalf("Create model-a: %v", err)
	}
	if err := modelRepo.Create(ctx, modelB); err != nil {
		t.Fatalf("Create model-b: %v", err)
	}

	if err := modelRepo.ClearDefault(ctx); err != nil {
		t.Fatalf("ClearDefault: %v", err)
	}

	gotA, err := modelRepo.GetByID(ctx, modelA.ID)
	if err != nil {
		t.Fatalf("GetByID model-a: %v", err)
	}
	gotB, err := modelRepo.GetByID(ctx, modelB.ID)
	if err != nil {
		t.Fatalf("GetByID model-b: %v", err)
	}
	if gotA.IsDefault != 0 || gotB.IsDefault != 0 {
		t.Fatalf("expected all defaults cleared, got model-a=%d model-b=%d", gotA.IsDefault, gotB.IsDefault)
	}
}

func TestModelConfigRepoCreateWithDefaultClearsOtherProvider(t *testing.T) {
	modelRepo, providerRepo, ctx := setupLlmModelTest(t)
	pidA := createTestLlmProvider(t, providerRepo, ctx, "provider-a")
	pidB := createTestLlmProvider(t, providerRepo, ctx, "provider-b")

	modelA := &model.LlmModel{
		ProviderID: pidA,
		ModelName:  "model-a",
		MaxTokens:  4096,
		IsEnabled:  1,
		IsDefault:  1,
	}
	if err := modelRepo.Create(ctx, modelA); err != nil {
		t.Fatalf("Create model-a: %v", err)
	}

	modelB := &model.LlmModel{
		ProviderID: pidB,
		ModelName:  "model-b",
		MaxTokens:  4096,
		IsEnabled:  1,
		IsDefault:  1,
	}
	if err := modelRepo.CreateWithDefault(ctx, modelB); err != nil {
		t.Fatalf("CreateWithDefault model-b: %v", err)
	}

	gotA, err := modelRepo.GetByID(ctx, modelA.ID)
	if err != nil {
		t.Fatalf("GetByID model-a: %v", err)
	}
	if gotA.IsDefault != 0 {
		t.Fatal("expected provider A default cleared when provider B set default")
	}

	defaultModel, err := modelRepo.GetDefaultModel(ctx)
	if err != nil {
		t.Fatalf("GetDefaultModel: %v", err)
	}
	if defaultModel.ID != modelB.ID {
		t.Fatalf("expected model-b as default, got id=%d", defaultModel.ID)
	}
}

func TestModelConfigRepoGetDefaultModelFallback(t *testing.T) {
	modelRepo, providerRepo, ctx := setupLlmModelTest(t)
	pid := createTestLlmProvider(t, providerRepo, ctx, "provider-fallback")

	if err := modelRepo.Create(ctx, &model.LlmModel{
		ProviderID: pid,
		ModelName:  "only-model",
		MaxTokens:  4096,
		IsEnabled:  1,
		IsDefault:  0,
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

func TestModelConfigRepoGetDefaultModelSkipsDisabledProvider(t *testing.T) {
	modelRepo, providerRepo, ctx := setupLlmModelTest(t)
	pidDisabled := createTestLlmProvider(t, providerRepo, ctx, "provider-disabled")
	pidEnabled := createTestLlmProvider(t, providerRepo, ctx, "provider-enabled")
	if err := providerRepo.UpdatePartial(ctx, pidDisabled, map[string]any{"is_enabled": 0}); err != nil {
		t.Fatalf("disable provider: %v", err)
	}

	if err := modelRepo.Create(ctx, &model.LlmModel{
		ProviderID: pidDisabled,
		ModelName:  "stale-default",
		MaxTokens:  4096,
		IsEnabled:  1,
		IsDefault:  1,
	}); err != nil {
		t.Fatalf("Create stale default: %v", err)
	}
	fallback := &model.LlmModel{
		ProviderID: pidEnabled,
		ModelName:  "runnable-fallback",
		MaxTokens:  4096,
		IsEnabled:  1,
		IsDefault:  0,
	}
	if err := modelRepo.Create(ctx, fallback); err != nil {
		t.Fatalf("Create fallback: %v", err)
	}

	got, err := modelRepo.GetDefaultModel(ctx)
	if err != nil {
		t.Fatalf("GetDefaultModel: %v", err)
	}
	if got.ID != fallback.ID {
		t.Fatalf("expected runnable fallback model, got id=%d model=%s", got.ID, got.ModelName)
	}
}
