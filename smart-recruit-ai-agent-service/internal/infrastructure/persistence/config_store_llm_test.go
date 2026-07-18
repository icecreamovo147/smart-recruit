package persistence

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-proto/recruitment/pb"
)

func TestCreateLlmModelDistinguishesExplicitZeroFromOmittedSamplingDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&llmProviderRecord{}, &llmModelRecord{}); err != nil {
		t.Fatal(err)
	}
	provider := llmProviderRecord{Name: "test", BaseURL: "https://example.com", APIKeyEncrypted: "test", ProviderType: "openai", ProtocolType: "openai_chat_completions", AuthType: "bearer", IsEnabled: true}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatal(err)
	}
	store := NewNativeStore(db)

	metadataSources := `{"temperature":{"source_type":"user"}}`
	explicit, err := store.CreateLlmModel(context.Background(), &pb.CreateModelRequest{ProviderId: provider.ID, ModelName: "explicit-zero", Temperature: 0, TemperatureSet: true, TopP: 0, TopPSet: true, MetadataSourcesJson: metadataSources})
	if err != nil {
		t.Fatal(err)
	}
	if explicit.Model.GetTemperature() != 0 || explicit.Model.GetTopP() != 0 {
		t.Fatalf("explicit model = %#v", explicit.Model)
	}
	if explicit.Model.GetMetadataSourcesJson() != metadataSources {
		t.Fatalf("metadata sources = %q", explicit.Model.GetMetadataSourcesJson())
	}

	omitted, err := store.CreateLlmModel(context.Background(), &pb.CreateModelRequest{ProviderId: provider.ID, ModelName: "omitted"})
	if err != nil {
		t.Fatal(err)
	}
	if omitted.Model.GetTemperature() != 0.7 || omitted.Model.GetTopP() != 1 {
		t.Fatalf("omitted model = %#v", omitted.Model)
	}
}

func TestCreateLlmModelRejectsProviderDuplicate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&llmProviderRecord{}, &llmModelRecord{}); err != nil {
		t.Fatal(err)
	}
	provider := llmProviderRecord{Name: "test", BaseURL: "https://example.com", APIKeyEncrypted: "test", ProviderType: "openai", ProtocolType: "openai_chat_completions", AuthType: "bearer", IsEnabled: true}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatal(err)
	}
	store := NewNativeStore(db)
	request := &pb.CreateModelRequest{ProviderId: provider.ID, ModelName: "same-model"}
	if response, err := store.CreateLlmModel(context.Background(), request); err != nil || response.Code != configOK {
		t.Fatalf("first create response=%#v err=%v", response, err)
	}
	if response, err := store.CreateLlmModel(context.Background(), request); err != nil || response.Code != configBadRequest {
		t.Fatalf("duplicate response=%#v err=%v", response, err)
	}
}

func TestCreateLlmModelRejectsNonObjectMetadataSources(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&llmProviderRecord{}, &llmModelRecord{}); err != nil {
		t.Fatal(err)
	}
	provider := llmProviderRecord{Name: "test", BaseURL: "https://example.com", APIKeyEncrypted: "test", ProviderType: "openai", ProtocolType: "openai_chat_completions", AuthType: "bearer", IsEnabled: true}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatal(err)
	}
	response, err := NewNativeStore(db).CreateLlmModel(context.Background(), &pb.CreateModelRequest{ProviderId: provider.ID, ModelName: "invalid-metadata", MetadataSourcesJson: `[]`})
	if err != nil {
		t.Fatal(err)
	}
	if response.Code != configBadRequest {
		t.Fatalf("response = %#v", response)
	}
}
