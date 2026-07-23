package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
	commonsai "smart-recruit-commons/ai"
)

func TestLoadActiveRecruitingPromptSelectsLatestAndRefreshes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&promptTemplateRecord{}); err != nil {
		t.Fatalf("migrate prompt templates: %v", err)
	}
	now := time.Now().UTC()
	rows := []promptTemplateRecord{
		{ID: 1, Name: "resume-old", Content: "system-old", Version: 1, IsActive: true, AgentType: recruitingruntime.AgentTypeResumeProfileExtractor, PromptRole: recruitingruntime.PromptRoleSystem, UpdatedAt: now.Add(-time.Minute)},
		{ID: 2, Name: "resume-user", Content: "wrong-role", Version: 9, IsActive: true, AgentType: recruitingruntime.AgentTypeResumeProfileExtractor, PromptRole: "user", UpdatedAt: now},
		{ID: 3, Name: "other-agent", Content: "wrong-agent", Version: 9, IsActive: true, AgentType: recruitingruntime.AgentTypeCandidateMatchEvaluator, PromptRole: recruitingruntime.PromptRoleSystem, UpdatedAt: now},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed prompt templates: %v", err)
	}
	store := NewNativeStore(db)

	first, err := store.LoadActiveRecruitingPrompt(context.Background(), recruitingruntime.AgentTypeResumeProfileExtractor, recruitingruntime.PromptRoleSystem)
	if err != nil {
		t.Fatalf("load first active prompt: %v", err)
	}
	if first.ID != 1 || first.Name != "resume-old" || first.Version != 1 || first.Content != "system-old" {
		t.Fatalf("first prompt = %#v", first)
	}

	newPrompt := promptTemplateRecord{ID: 4, Name: "resume-new", Content: "system-new", Version: 2, IsActive: true, AgentType: recruitingruntime.AgentTypeResumeProfileExtractor, PromptRole: recruitingruntime.PromptRoleSystem, UpdatedAt: now.Add(time.Minute)}
	if err := db.Create(&newPrompt).Error; err != nil {
		t.Fatalf("activate new prompt: %v", err)
	}
	second, err := store.LoadActiveRecruitingPrompt(context.Background(), recruitingruntime.AgentTypeResumeProfileExtractor, recruitingruntime.PromptRoleSystem)
	if err != nil {
		t.Fatalf("load refreshed active prompt: %v", err)
	}
	if second.ID != 4 || second.Name != "resume-new" || second.Version != 2 || second.Content != "system-new" {
		t.Fatalf("second prompt = %#v", second)
	}
}

func TestNativeStoreCompleteStructuredSharesConcurrencyAcrossRequests(t *testing.T) {
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		current := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			observed := maxInFlight.Load()
			if current <= observed || maxInFlight.CompareAndSwap(observed, current) {
				break
			}
		}
		time.Sleep(75 * time.Millisecond)
		writeStructuredTestResponse(t, w, `{"ok":true}`)
	}))
	t.Cleanup(server.Close)

	store := newStructuredRuntimeTestStore(t, server.URL+"/v1", "concurrency-model", 1, 5)
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := store.CompleteStructured(context.Background(), "system", "user")
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("CompleteStructured: %v", err)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("provider calls = %d, want 2", got)
	}
	if got := maxInFlight.Load(); got != 1 {
		t.Fatalf("max concurrent provider calls = %d, want shared limit 1", got)
	}
}

func TestNativeStoreCompleteStructuredSharesCircuitAndRefreshesChangedConfig(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var request struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if request.Model == "failing-model" {
			http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
			return
		}
		writeStructuredTestResponse(t, w, `{"refreshed":true}`)
	}))
	t.Cleanup(server.Close)

	store := newStructuredRuntimeTestStore(t, server.URL+"/v1", "failing-model", 2, 1)
	_, firstErr := store.CompleteStructured(context.Background(), "system", "first")
	if firstErr == nil {
		t.Fatal("first provider call unexpectedly succeeded")
	}
	_, secondErr := store.CompleteStructured(context.Background(), "system", "second")
	var aiErr *commonsai.AIError
	if !errors.As(secondErr, &aiErr) || aiErr.Type != commonsai.AICircuitOpen {
		t.Fatalf("second error = %#v, want shared circuit open", secondErr)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("provider calls before refresh = %d, want 1", got)
	}

	if err := store.db.Model(&llmModelRecord{}).Where("id = ?", 1).Update("model_name", "healthy-model").Error; err != nil {
		t.Fatalf("refresh model config: %v", err)
	}
	result, err := store.CompleteStructured(context.Background(), "system", "after-refresh")
	if err != nil {
		t.Fatalf("CompleteStructured after config refresh: %v", err)
	}
	if result.Content != `{"refreshed":true}` || result.ModelName != "healthy-model" {
		t.Fatalf("refreshed result = %#v", result)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("provider calls after refresh = %d, want 2", got)
	}
}

func TestNativeStoreCompleteStructuredLateOldConfigCannotReplaceNewClient(t *testing.T) {
	var oldCalls atomic.Int32
	var newCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		switch request.Model {
		case "old-model":
			oldCalls.Add(1)
			writeStructuredTestResponse(t, w, `{"old":true}`)
		case "new-model":
			newCalls.Add(1)
			http.Error(w, "new provider unavailable", http.StatusServiceUnavailable)
		default:
			t.Errorf("unexpected model %q", request.Model)
			http.Error(w, "unexpected model", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	store := newStructuredRuntimeTestStore(t, server.URL+"/v1", "old-model", 2, 1)
	oldSelected := make(chan struct{})
	releaseOld := make(chan struct{})
	cache := structuredRuntimeClientCacheFor(store)
	cache.beforeClientLookupForTest = func(cfg selectedLLMConfig) {
		if cfg.Model == "old-model" {
			close(oldSelected)
			<-releaseOld
		}
	}

	oldDone := make(chan error, 1)
	go func() {
		_, err := store.CompleteStructured(context.Background(), "system", "old-request")
		oldDone <- err
	}()
	<-oldSelected

	if err := store.db.Model(&llmModelRecord{}).Where("id = ?", 1).Update("model_name", "new-model").Error; err != nil {
		close(releaseOld)
		t.Fatalf("refresh model config: %v", err)
	}
	_, refreshedErr := store.CompleteStructured(context.Background(), "system", "new-request")
	if refreshedErr == nil {
		close(releaseOld)
		t.Fatal("new configuration provider call unexpectedly succeeded")
	}
	close(releaseOld)
	if err := <-oldDone; err != nil {
		t.Fatalf("late old configuration request: %v", err)
	}

	_, afterLateOldErr := store.CompleteStructured(context.Background(), "system", "new-request-after-old")
	var aiErr *commonsai.AIError
	if !errors.As(afterLateOldErr, &aiErr) || aiErr.Type != commonsai.AICircuitOpen {
		t.Fatalf("request after late old config error = %#v, want preserved new-config circuit", afterLateOldErr)
	}
	if got := oldCalls.Load(); got != 1 {
		t.Fatalf("old-model provider calls = %d, want 1", got)
	}
	if got := newCalls.Load(); got != 1 {
		t.Fatalf("new-model provider calls = %d, want 1 because refreshed client circuit remains shared", got)
	}
}

func TestNativeStoreResolvesContextConfigAndCompletionUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "usage-completion", "object": "chat.completion", "created": time.Now().Unix(),
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": "usage reply"}, "finish_reason": "stop"}},
			"usage":   map[string]any{"prompt_tokens": 91, "completion_tokens": 9, "total_tokens": 100},
		})
	}))
	t.Cleanup(server.Close)
	store := newStructuredRuntimeTestStore(t, server.URL+"/v1", "usage-model", 1, 5)
	if err := store.db.Model(&llmModelRecord{}).Where("id = ?", 1).Updates(map[string]any{
		"context_window_tokens": 32768,
		"max_tokens":            2048,
	}).Error; err != nil {
		t.Fatalf("update context config: %v", err)
	}

	info, found, err := store.ResolveLLMRuntimeModelInfo(context.Background(), 1)
	if err != nil || !found {
		t.Fatalf("ResolveLLMRuntimeModelInfo found=%v err=%v", found, err)
	}
	if info.ContextWindowTokens != 32768 || info.MaxOutputTokens != 2048 {
		t.Fatalf("resolved context config = %d/%d, want 32768/2048", info.ContextWindowTokens, info.MaxOutputTokens)
	}

	result, err := store.CompleteWithOptionsAndUsage(context.Background(), "prompt", 1, aiagentgrpc.ChatCompletionOptions{})
	if err != nil {
		t.Fatalf("CompleteWithOptionsAndUsage: %v", err)
	}
	if result.Content != "usage reply" || result.TokenUsage == nil || result.TokenUsage.PromptTokens != 91 || result.TokenUsage.TotalTokens != 100 {
		t.Fatalf("completion result = %+v", result)
	}
}

func newStructuredRuntimeTestStore(t *testing.T, baseURL, model string, maxConcurrency, circuitThreshold int) *NativeStore {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "llm-runtime.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&llmProviderRecord{}, &llmModelRecord{}); err != nil {
		t.Fatalf("migrate llm runtime: %v", err)
	}
	now := time.Now().UTC()
	provider := llmProviderRecord{ID: 1, Name: "test", BaseURL: baseURL, APIKeyEncrypted: "test-api-key", ProviderType: "openai_compatible", IsEnabled: true, CreatedAt: now, UpdatedAt: now}
	modelRow := llmModelRecord{ID: 1, ProviderID: 1, ModelName: model, MaxConcurrency: maxConcurrency, TimeoutSeconds: 2, IsEnabled: true, IsDefault: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("seed llm provider: %v", err)
	}
	if err := db.Create(&modelRow).Error; err != nil {
		t.Fatalf("seed llm model: %v", err)
	}
	store := NewNativeStore(db)
	store.SetRuntimeLLMConfig(RuntimeLLMConfig{
		MaxConcurrency:          maxConcurrency,
		CircuitFailureThreshold: circuitThreshold,
		CircuitOpenTimeout:      time.Minute,
		HalfOpenMaxRequests:     1,
	})
	t.Cleanup(func() { structuredRuntimeClientCaches.Delete(store) })
	return store
}

func writeStructuredTestResponse(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":      "test-completion",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]any{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": "stop",
		}},
	}); err != nil {
		t.Errorf("encode response: %v", err)
	}
}
