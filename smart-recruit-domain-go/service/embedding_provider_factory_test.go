package service

import (
	"context"
	"crypto/rand"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/pkg/crypto"
	"smart-recruit-domain-go/repository"
	"smart-recruit-platform-go/serviceconfig"
	"smart-recruit-proto/recruitment/pb"
)

func setupFactoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	if err := db.AutoMigrate(&model.EmbeddingProviderConfig{}, &model.EmbeddingModelConfig{}); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}

func testEncryptionKey(t *testing.T) crypto.EncryptionKey {
	t.Helper()
	var key crypto.EncryptionKey
	if _, err := rand.Read(key[:]); err != nil {
		t.Fatalf("failed to generate encryption key: %v", err)
	}
	return key
}

func encryptAPIKey(t *testing.T, key crypto.EncryptionKey, plaintext string) string {
	t.Helper()
	result, err := crypto.Encrypt(key, []byte(plaintext))
	if err != nil {
		t.Fatalf("failed to encrypt API key: %v", err)
	}
	return result
}

func seedDefaultBailianProvider(t *testing.T, db *gorm.DB, apiKeyEncrypted string) (int64, int64) {
	t.Helper()
	provider := model.EmbeddingProviderConfig{
		Name:            "test-bailian",
		ProviderType:    "bailian",
		Endpoint:        "http://localhost:9999/v1/embeddings",
		APIKeyEncrypted: apiKeyEncrypted,
		IsEnabled:       1,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}

	modelConfig := model.EmbeddingModelConfig{
		ProviderID:     provider.ID,
		ModelName:      "text-embedding-v4",
		DisplayName:    "text-embedding-v4",
		IsEnabled:      1,
		IsDefault:      1,
		TimeoutSeconds: 30,
		MaxRetries:     2,
	}
	if err := db.Create(&modelConfig).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	return provider.ID, modelConfig.ID
}

func factoryWithDB(db *gorm.DB, enabled bool) *EmbeddingProviderFactory {
	cfg := config.Config{}
	if !enabled {
		v := false
		cfg.Embedding.Enabled = &v
	}
	return NewEmbeddingProviderFactory(
		cfg,
		repository.NewEmbeddingModelRepo(db),
		repository.NewEmbeddingProviderRepo(db),
	)
}

func TestEmbeddingProviderFactory_ConfigDisabled(t *testing.T) {
	db := setupFactoryTestDB(t)
	factory := factoryWithDB(db, false)
	provider := factory.Build(context.Background(), crypto.EncryptionKey{})
	if _, ok := provider.(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider, got %T", provider)
	}
}

func TestEmbeddingProviderFactory_NoDefaultModel(t *testing.T) {
	db := setupFactoryTestDB(t)
	factory := factoryWithDB(db, true)
	provider := factory.Build(context.Background(), crypto.EncryptionKey{})
	if _, ok := provider.(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider, got %T", provider)
	}
}

func TestEmbeddingProviderFactory_ModelDisabled(t *testing.T) {
	db := setupFactoryTestDB(t)

	provider := model.EmbeddingProviderConfig{
		Name:            "test-bailian",
		ProviderType:    "bailian",
		Endpoint:        "http://localhost:9999",
		APIKeyEncrypted: "dummy",
		IsEnabled:       1,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}

	modelConfig := model.EmbeddingModelConfig{
		ProviderID: provider.ID,
		ModelName:  "text-embedding-v4",
		IsEnabled:  0,
		IsDefault:  1,
	}
	if err := db.Create(&modelConfig).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	factory := factoryWithDB(db, true)
	providerResult := factory.Build(context.Background(), crypto.EncryptionKey{})
	if _, ok := providerResult.(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider, got %T", providerResult)
	}
}

func TestEmbeddingProviderFactory_ProviderDisabled(t *testing.T) {
	db := setupFactoryTestDB(t)

	providerConfig := model.EmbeddingProviderConfig{
		Name:            "test-bailian",
		ProviderType:    "bailian",
		Endpoint:        "http://localhost:9999",
		APIKeyEncrypted: "dummy",
		IsEnabled:       0,
	}
	if err := db.Create(&providerConfig).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}

	modelConfig := model.EmbeddingModelConfig{
		ProviderID: providerConfig.ID,
		ModelName:  "text-embedding-v4",
		IsEnabled:  1,
		IsDefault:  1,
	}
	if err := db.Create(&modelConfig).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	factory := factoryWithDB(db, true)
	providerResult := factory.Build(context.Background(), crypto.EncryptionKey{})
	if _, ok := providerResult.(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider, got %T", providerResult)
	}
}

func TestEmbeddingProviderFactory_UnsupportedProviderType(t *testing.T) {
	db := setupFactoryTestDB(t)

	providerConfig := model.EmbeddingProviderConfig{
		Name:            "test-openai",
		ProviderType:    "openai",
		Endpoint:        "http://localhost:9999",
		APIKeyEncrypted: "dummy",
		IsEnabled:       1,
	}
	if err := db.Create(&providerConfig).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}

	modelConfig := model.EmbeddingModelConfig{
		ProviderID: providerConfig.ID,
		ModelName:  "text-embedding-3-small",
		IsEnabled:  1,
		IsDefault:  1,
	}
	if err := db.Create(&modelConfig).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	factory := factoryWithDB(db, true)
	providerResult := factory.Build(context.Background(), crypto.EncryptionKey{})
	if _, ok := providerResult.(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider, got %T", providerResult)
	}
}

func TestEmbeddingProviderFactory_DecryptFails(t *testing.T) {
	db := setupFactoryTestDB(t)
	key := testEncryptionKey(t)
	encrypted := encryptAPIKey(t, key, "sk-real-key")

	providerConfig := model.EmbeddingProviderConfig{
		Name:            "test-bailian",
		ProviderType:    "bailian",
		Endpoint:        "http://localhost:9999",
		APIKeyEncrypted: encrypted,
		IsEnabled:       1,
	}
	if err := db.Create(&providerConfig).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}

	modelConfig := model.EmbeddingModelConfig{
		ProviderID: providerConfig.ID,
		ModelName:  "text-embedding-v4",
		IsEnabled:  1,
		IsDefault:  1,
	}
	if err := db.Create(&modelConfig).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	var wrongKey crypto.EncryptionKey
	copy(wrongKey[:], []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	factory := factoryWithDB(db, true)
	providerResult := factory.Build(context.Background(), wrongKey)
	if _, ok := providerResult.(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider on decrypt failure, got %T", providerResult)
	}
}

func TestEmbeddingProviderFactory_DecryptInvalidData(t *testing.T) {
	db := setupFactoryTestDB(t)

	providerConfig := model.EmbeddingProviderConfig{
		Name:            "test-bailian",
		ProviderType:    "bailian",
		Endpoint:        "http://localhost:9999",
		APIKeyEncrypted: "invalid-hex-data",
		IsEnabled:       1,
	}
	if err := db.Create(&providerConfig).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}

	modelConfig := model.EmbeddingModelConfig{
		ProviderID: providerConfig.ID,
		ModelName:  "text-embedding-v4",
		IsEnabled:  1,
		IsDefault:  1,
	}
	if err := db.Create(&modelConfig).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	key := testEncryptionKey(t)

	factory := factoryWithDB(db, true)
	providerResult := factory.Build(context.Background(), key)
	if _, ok := providerResult.(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider (not panic) on decrypt failure, got %T", providerResult)
	}
}

func TestEmbeddingProviderFactory_SuccessBailian(t *testing.T) {
	db := setupFactoryTestDB(t)
	key := testEncryptionKey(t)
	encrypted := encryptAPIKey(t, key, "sk-real-key")

	seedDefaultBailianProvider(t, db, encrypted)

	factory := factoryWithDB(db, true)
	providerResult := factory.Build(context.Background(), key)

	if _, ok := providerResult.(*BailianTextEmbeddingProvider); !ok {
		t.Fatalf("expected *BailianTextEmbeddingProvider, got %T", providerResult)
	}
}

func TestEmbeddingService_RebuildProviderSwitchesActiveProvider(t *testing.T) {
	db := setupFactoryTestDB(t)
	key := testEncryptionKey(t)
	encrypted := encryptAPIKey(t, key, "sk-real-key")
	seedDefaultBailianProvider(t, db, encrypted)

	factory := factoryWithDB(db, true)
	svc := NewEmbeddingService(repository.NewAIEmbeddingRepo(db), factory, key)

	if _, ok := svc.currentProvider().(UnavailableEmbeddingProvider); ok {
		t.Fatalf("expected real provider after initial build, got UnavailableEmbeddingProvider")
	}
	if _, ok := svc.currentProvider().(*BailianTextEmbeddingProvider); !ok {
		t.Fatalf("expected *BailianTextEmbeddingProvider after initial build, got %T", svc.currentProvider())
	}

	// Simulate a config change: delete the default model and rebuild.
	// The factory should now return UnavailableEmbeddingProvider.
	if err := db.Where("is_default = ?", 1).Delete(&model.EmbeddingModelConfig{}).Error; err != nil {
		t.Fatalf("delete default model: %v", err)
	}
	svc.RebuildProvider(context.Background())

	if _, ok := svc.currentProvider().(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider after rebuild, got %T", svc.currentProvider())
	}
}

func TestEmbeddingService_RebuildProviderHandlesNilFactory(t *testing.T) {
	svc := NewEmbeddingService(nil, nil, crypto.EncryptionKey{})
	if _, ok := svc.currentProvider().(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider with nil factory, got %T", svc.currentProvider())
	}
	// Rebuild on nil factory should be a no-op (not panic).
	svc.RebuildProvider(context.Background())
	if _, ok := svc.currentProvider().(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider after rebuild with nil factory, got %T", svc.currentProvider())
	}
}

func TestEmbeddingProviderFactory_ZeroEncryptionKey(t *testing.T) {
	db := setupFactoryTestDB(t)

	providerConfig := model.EmbeddingProviderConfig{
		Name:            "test-bailian",
		ProviderType:    "bailian",
		Endpoint:        "http://localhost:9999",
		APIKeyEncrypted: "abc",
		IsEnabled:       1,
	}
	if err := db.Create(&providerConfig).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}

	modelConfig := model.EmbeddingModelConfig{
		ProviderID: providerConfig.ID,
		ModelName:  "text-embedding-v4",
		IsEnabled:  1,
		IsDefault:  1,
	}
	if err := db.Create(&modelConfig).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}

	factory := factoryWithDB(db, true)
	providerResult := factory.Build(context.Background(), crypto.EncryptionKey{})
	if _, ok := providerResult.(UnavailableEmbeddingProvider); !ok {
		t.Fatalf("expected UnavailableEmbeddingProvider with zero encryption key, got %T", providerResult)
	}
}

func TestEmbeddingConfigService_UpdateProviderClearsEmptyExtraHeaders(t *testing.T) {
	db := setupFactoryTestDB(t)
	key := testEncryptionKey(t)
	encrypted := encryptAPIKey(t, key, "sk-real-key")

	provider := model.EmbeddingProviderConfig{
		Name:            "test-bailian",
		ProviderType:    "bailian",
		Endpoint:        "https://api.example.com/v1/embeddings",
		APIKeyEncrypted: encrypted,
		ExtraHeaders:    stringPtr(`{"X-Foo":"bar"}`),
		IsEnabled:       1,
	}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}

	providerRepo := repository.NewEmbeddingProviderRepo(db)
	modelRepo := repository.NewEmbeddingModelRepo(db)
	svc := NewEmbeddingConfigService(providerRepo, modelRepo, key, nil, nil)

	// Update with ExtraHeadersSet=true but empty JSON. This must NOT
	// produce `extra_headers = ''` in the SQL (which MySQL JSON column
	// rejects with "Invalid JSON text" 3140). It should clear the field.
	resp, err := svc.UpdateEmbeddingProvider(context.Background(), &pb.UpdateEmbeddingProviderRequest{
		Id:               provider.ID,
		Endpoint:         "https://api.example.com/v2/embeddings",
		ExtraHeadersSet:  true,
		ExtraHeadersJson: "",
	})
	if err != nil {
		t.Fatalf("UpdateEmbeddingProvider with empty extra headers: %v", err)
	}
	if resp.GetCode() != 0 {
		t.Fatalf("expected code 0, got %d (%s)", resp.GetCode(), resp.GetMsg())
	}

	var stored model.EmbeddingProviderConfig
	if err := db.First(&stored, provider.ID).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored.ExtraHeaders != nil {
		t.Fatalf("expected ExtraHeaders to be cleared, got %q", *stored.ExtraHeaders)
	}
	if stored.Endpoint != "https://api.example.com/v2/embeddings" {
		t.Fatalf("endpoint not updated, got %q", stored.Endpoint)
	}

	// Now set a non-empty extra headers value — should be stored.
	ehJSON := `{"X-Trace":"on"}`
	resp, err = svc.UpdateEmbeddingProvider(context.Background(), &pb.UpdateEmbeddingProviderRequest{
		Id:               provider.ID,
		ExtraHeadersSet:  true,
		ExtraHeadersJson: ehJSON,
	})
	if err != nil {
		t.Fatalf("UpdateEmbeddingProvider with new extra headers: %v", err)
	}
	if resp.GetCode() != 0 {
		t.Fatalf("expected code 0, got %d (%s)", resp.GetCode(), resp.GetMsg())
	}
	if err := db.First(&stored, provider.ID).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored.ExtraHeaders == nil || *stored.ExtraHeaders != ehJSON {
		t.Fatalf("expected ExtraHeaders=%q, got %v", ehJSON, stored.ExtraHeaders)
	}

	// Update without ExtraHeadersSet — existing value should be preserved.
	resp, err = svc.UpdateEmbeddingProvider(context.Background(), &pb.UpdateEmbeddingProviderRequest{
		Id:       provider.ID,
		Endpoint: "https://api.example.com/v3/embeddings",
	})
	if err != nil {
		t.Fatalf("UpdateEmbeddingProvider without extra headers: %v", err)
	}
	if err := db.First(&stored, provider.ID).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored.ExtraHeaders == nil || *stored.ExtraHeaders != ehJSON {
		t.Fatalf("ExtraHeaders should be preserved when not set, got %v", stored.ExtraHeaders)
	}
}

func stringPtr(s string) *string {
	return &s
}
