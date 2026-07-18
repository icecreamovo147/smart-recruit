package persistence

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOpenAIModelURLsUsesVersionedBaseAndExplicitOverride(t *testing.T) {
	provider := llmProviderRecord{BaseURL: "https://api.example.com/v1"}
	urls := openAIModelURLs(provider)
	if len(urls) == 0 || urls[0] != "https://api.example.com/v1/models" {
		t.Fatalf("urls = %#v", urls)
	}
	provider.DiscoveryURL = sql.NullString{String: "https://catalog.example.com/models", Valid: true}
	urls = openAIModelURLs(provider)
	if len(urls) != 1 || urls[0] != provider.DiscoveryURL.String {
		t.Fatalf("explicit urls = %#v", urls)
	}
}

func TestValidateDiscoveryURLRejectsPrivateCloudEndpoint(t *testing.T) {
	target, _ := url.Parse("http://127.0.0.1:11434/api/tags")
	if err := validateDiscoveryURL(context.Background(), target, false); err == nil {
		t.Fatal("expected private HTTP endpoint to be rejected")
	}
	if err := validateDiscoveryURL(context.Background(), target, true); err != nil {
		t.Fatalf("Ollama private endpoint rejected: %v", err)
	}
}

func TestOllamaModelDiscoveryParsesAndSortsCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:8b","details":{"family":"qwen3"}},{"name":"llama3.2:latest"}]}`))
	}))
	defer server.Close()
	models, err := (ollamaModelDiscovery{}).Discover(context.Background(), discoveryRequest{Provider: llmProviderRecord{BaseURL: server.URL}, Client: &http.Client{Transport: http.DefaultTransport}})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ModelName != "llama3.2:latest" || models[1].ModelName != "qwen3:8b" {
		t.Fatalf("models = %#v", models)
	}
}

func TestInspectOllamaModelReadsContextAndCapabilities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/show" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"model_info":{"qwen3.context_length":32768},"capabilities":["completion","tools"],"details":{"family":"qwen3"}}`))
	}))
	defer server.Close()
	model, err := inspectOllamaModel(context.Background(), server.Client(), server.URL+"/api/show", "qwen3:8b")
	if err != nil {
		t.Fatal(err)
	}
	if !model.ContextWindowTokensKnown || model.ContextWindowTokens != 32768 {
		t.Fatalf("model = %#v", model)
	}
	if model.CapabilitiesJSON == "" {
		t.Fatal("expected capabilities metadata")
	}
}

func TestBundledModelCatalogSyncAndMergeAreIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&llmModelCatalogRecord{}, &llmModelMetadataObservationRecord{}); err != nil {
		t.Fatal(err)
	}
	store := NewNativeStore(db)
	for range 2 {
		if err := store.SyncBundledLlmModelCatalog(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Model(&llmModelCatalogRecord{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	catalog, err := parseBundledModelCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if count != int64(len(catalog.Entries)) {
		t.Fatalf("catalog count = %d, want %d", count, len(catalog.Entries))
	}
	observedAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	models := annotateProviderModels([]discoveredModel{{ModelName: "deepseek-v4-pro", DisplayName: "deepseek-v4-pro", OwnedBy: "deepseek"}}, metadataSourceProviderAPI, "https://api.deepseek.com", observedAt)
	models, err = store.enrichModelsFromCatalog(context.Background(), llmProviderRecord{ProviderType: "openai_compatible", BaseURL: "https://proxy.example.com/v1"}, models, observedAt)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || !models[0].ContextWindowTokensKnown || models[0].ContextWindowTokens != 1000000 || !models[0].MaxOutputTokensKnown || models[0].MaxOutputTokens != 384000 {
		t.Fatalf("enriched model = %#v", models)
	}
	if models[0].FieldSources["model_name"].SourceType != metadataSourceProviderAPI {
		t.Fatalf("model name source = %#v", models[0].FieldSources["model_name"])
	}
	if models[0].FieldSources["context_window_tokens"].SourceType != metadataSourceOfficialDocument {
		t.Fatalf("context source = %#v", models[0].FieldSources["context_window_tokens"])
	}

	models = annotateProviderModels([]discoveredModel{{ModelName: "mimo-v2.5-pro", DisplayName: "mimo-v2.5-pro", OwnedBy: "xiaomi"}}, metadataSourceProviderAPI, "https://api.xiaomimimo.com/v1", observedAt)
	models, err = store.enrichModelsFromCatalog(context.Background(), llmProviderRecord{ProviderType: "openai_compatible", BaseURL: "https://api.xiaomimimo.com/v1"}, models, observedAt)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || !models[0].ContextWindowTokensKnown || models[0].ContextWindowTokens != 1000000 || !models[0].MaxOutputTokensKnown || models[0].MaxOutputTokens != 128000 {
		t.Fatalf("enriched MiMo model = %#v", models)
	}
	if !models[0].TemperatureKnown || models[0].Temperature != 1 || !models[0].TopPKnown || models[0].TopP != 0.95 {
		t.Fatalf("MiMo sampling defaults = %#v", models[0])
	}
}

func TestProviderCatalogFamiliesRecognizesMainstreamProviderHosts(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{baseURL: "https://api.openai.com/v1", want: "openai"},
		{baseURL: "https://api.anthropic.com", want: "anthropic"},
		{baseURL: "https://generativelanguage.googleapis.com", want: "google_gemini"},
		{baseURL: "https://api.x.ai/v1", want: "xai"},
		{baseURL: "https://api.mistral.ai/v1", want: "mistral"},
		{baseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", want: "qwen"},
		{baseURL: "https://open.bigmodel.cn/api/paas/v4", want: "zhipu"},
		{baseURL: "https://api.minimax.io/v1", want: "minimax"},
		{baseURL: "https://api.xiaomimimo.com/v1", want: "xiaomi"},
		{baseURL: "https://api.moonshot.ai/v1", want: "moonshot"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			families := providerCatalogFamilies(llmProviderRecord{ProviderType: "openai_compatible", BaseURL: tt.baseURL})
			found := false
			for _, family := range families {
				if family == tt.want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("families = %#v, want %q", families, tt.want)
			}
		})
	}
}

func TestModelMetadataObservationsDeduplicateUnchangedValues(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&llmProviderRecord{}, &llmModelMetadataObservationRecord{}); err != nil {
		t.Fatal(err)
	}
	provider := llmProviderRecord{Name: "DeepSeek", BaseURL: "https://api.deepseek.com", APIKeyEncrypted: "test", ProviderType: "openai_compatible", ProtocolType: "openai_chat_completions", AuthType: "bearer", IsEnabled: true}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatal(err)
	}
	observedAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	models := annotateProviderModels([]discoveredModel{{ModelName: "deepseek-v4-flash", DisplayName: "deepseek-v4-flash", OwnedBy: "deepseek"}}, metadataSourceProviderAPI, provider.BaseURL, observedAt)
	store := NewNativeStore(db)
	for range 2 {
		if err := store.storeModelMetadataObservations(context.Background(), provider.ID, models, observedAt.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Model(&llmModelMetadataObservationRecord{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("observation count = %d, want model_name and owned_by only", count)
	}
}
