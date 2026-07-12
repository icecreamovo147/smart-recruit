package service

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/repository"
	"smart-recruit-proto/recruitment/pb"
)

func setupLlmConfigServiceTest(t *testing.T) (*LlmConfigService, *repository.ModelConfigRepo, *repository.ProviderRepo, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.LlmProvider{}, &model.LlmModel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	providerRepo := repository.NewProviderRepo(db)
	modelRepo := repository.NewModelConfigRepo(db)
	svc := NewLlmConfigService(providerRepo, modelRepo, testEncryptionKey(t))
	return svc, modelRepo, providerRepo, context.Background()
}

func TestLlmConfigServiceSetDefaultClearsOtherProvider(t *testing.T) {
	svc, modelRepo, providerRepo, ctx := setupLlmConfigServiceTest(t)

	pidA := createLlmProviderForTest(t, providerRepo, ctx, "provider-a")
	pidB := createLlmProviderForTest(t, providerRepo, ctx, "provider-b")

	modelA := &model.LlmModel{
		ProviderID: pidA,
		ModelName:  "model-a",
		MaxTokens:  4096,
		IsEnabled:  1,
		IsDefault:  1,
	}
	if err := modelRepo.CreateWithDefault(ctx, modelA); err != nil {
		t.Fatalf("CreateWithDefault model-a: %v", err)
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

	defaultModel, err := modelRepo.GetDefaultModel(ctx)
	if err != nil {
		t.Fatalf("GetDefaultModel: %v", err)
	}
	if defaultModel.ProviderID != pidB {
		t.Fatalf("expected provider B default, got provider_id=%d", defaultModel.ProviderID)
	}

	gotA, err := modelRepo.GetByID(ctx, modelA.ID)
	if err != nil {
		t.Fatalf("GetByID model-a: %v", err)
	}
	if gotA.IsDefault != 0 {
		t.Fatal("expected provider A default cleared")
	}

	_ = svc
}

func TestLlmConfigServiceRejectsDefaultOnDisabledProvider(t *testing.T) {
	svc, _, providerRepo, ctx := setupLlmConfigServiceTest(t)

	pid := createLlmProviderForTest(t, providerRepo, ctx, "disabled-provider")
	if err := providerRepo.UpdatePartial(ctx, pid, map[string]any{"is_enabled": 0}); err != nil {
		t.Fatalf("disable provider: %v", err)
	}

	_, err := svc.CreateModel(ctx, &pb.CreateModelRequest{
		ProviderId: pid,
		ModelName:  "blocked-default",
		IsDefault:  true,
	})
	if err == nil {
		t.Fatal("expected error creating default model on disabled provider")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}

func createLlmProviderForTest(t *testing.T, repo *repository.ProviderRepo, ctx context.Context, name string) int64 {
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
