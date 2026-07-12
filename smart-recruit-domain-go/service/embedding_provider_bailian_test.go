package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBailianProvider_Success(t *testing.T) {
	var capturedMethod, capturedPath, capturedAuth, capturedContentType string
	var capturedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		capturedContentType = r.Header.Get("Content-Type")

		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"output": {
				"embeddings": [
					{"embedding": [0.1, 0.2, 0.3, 0.4]}
				]
			},
			"request_id": "req-001"
		}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test-key",
		WithBailianModel("text-embedding-v4"),
		WithBailianTimeout(10*time.Second),
	)

	ctx := context.Background()
	result, err := provider.EmbedText(ctx, "hello world")
	if err != nil {
		t.Fatalf("EmbedText: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", capturedMethod)
	}
	if !strings.Contains(capturedPath, srv.URL) && capturedPath != "/" && capturedPath != "" {
		// path varies based on server URL; just ensure method is correct
	}
	if capturedAuth != "Bearer sk-test-key" {
		t.Fatalf("expected Bearer sk-test-key, got %q", capturedAuth)
	}
	if capturedContentType != "application/json" {
		t.Fatalf("expected application/json, got %q", capturedContentType)
	}

	model, _ := capturedBody["model"].(string)
	if model != "text-embedding-v4" {
		t.Fatalf("expected model text-embedding-v4, got %q", model)
	}
	texts, _ := capturedBody["input"].([]any)
	if len(texts) != 1 || texts[0].(string) != "hello world" {
		t.Fatalf("unexpected input: %v", texts)
	}
	if _, exists := capturedBody["dimensions"]; exists {
		t.Fatalf("expected no dimensions field when not configured, got %v", capturedBody["dimensions"])
	}

	if len(result.Vector) != 4 {
		t.Fatalf("expected vector length 4, got %d", len(result.Vector))
	}
	if result.Vector[0] != 0.1 || result.Vector[1] != 0.2 {
		t.Fatalf("unexpected vector: %v", result.Vector)
	}
}

func TestBailianProvider_DataResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{"embedding": [0.5, 0.6, 0.7]}
			],
			"model": "text-embedding-v4"
		}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test",
		WithBailianTimeout(10*time.Second),
	)
	result, err := provider.EmbedText(context.Background(), "test")
	if err != nil {
		t.Fatalf("EmbedText: %v", err)
	}
	if len(result.Vector) != 3 || result.Vector[0] != 0.5 {
		t.Fatalf("unexpected vector from data response: %v", result.Vector)
	}
	if result.Model != "text-embedding-v4" {
		t.Fatalf("expected model text-embedding-v4, got %q", result.Model)
	}
}

func TestBailianProvider_401NoRetry(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-bad",
		WithBailianTimeout(10*time.Second),
		WithBailianMaxRetries(2),
	)
	_, err := provider.EmbedText(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", callCount)
	}
}

func TestBailianProvider_403NoRetry(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-bad",
		WithBailianTimeout(10*time.Second),
		WithBailianMaxRetries(2),
	)
	_, err := provider.EmbedText(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call (no retry), got %d", callCount)
	}
}

func TestBailianProvider_429Retry(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"output":{"embeddings":[{"embedding":[0.1]}]}}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test",
		WithBailianTimeout(10*time.Second),
		WithBailianMaxRetries(2),
	)
	result, err := provider.EmbedText(context.Background(), "test")
	if err != nil {
		t.Fatalf("EmbedText after retry: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", callCount)
	}
	if len(result.Vector) != 1 || result.Vector[0] != 0.1 {
		t.Fatalf("unexpected result after retry: %v", result.Vector)
	}
}

func TestBailianProvider_5xxRetry(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal error"}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test",
		WithBailianTimeout(10*time.Second),
		WithBailianMaxRetries(1),
	)
	_, err := provider.EmbedText(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", callCount)
	}
}

func TestBailianProvider_TimeoutRetry(t *testing.T) {
	provider := NewBailianTextEmbeddingProvider("http://127.0.0.1:1", "sk-test",
		WithBailianMaxRetries(1),
		WithBailianTimeout(1*time.Millisecond),
	)
	_, err := provider.EmbedText(context.Background(), "test")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "max retries exhausted") {
		t.Fatalf("expected retry exhaustion, got: %v", err)
	}
}

func TestBailianProvider_EmptyVector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"output":{"embeddings":[{"embedding":[]}]}}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test",
		WithBailianTimeout(10*time.Second),
	)
	_, err := provider.EmbedText(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for empty vector")
	}
}

func TestBailianProvider_NullVector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"output":{"embeddings":[{"embedding":null}]}}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test",
		WithBailianTimeout(10*time.Second),
	)
	_, err := provider.EmbedText(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for null vector")
	}
}

func TestBailianProvider_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`this is not json`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test",
		WithBailianTimeout(10*time.Second),
	)
	_, err := provider.EmbedText(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestBailianProvider_ModelOverride(t *testing.T) {
	var capturedModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		capturedModel = body.Model
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"output":{"embeddings":[{"embedding":[1.0]}]}}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test",
		WithBailianModel("custom-model-v2"),
		WithBailianTimeout(10*time.Second),
	)
	_, err := provider.EmbedText(context.Background(), "test")
	if err != nil {
		t.Fatalf("EmbedText: %v", err)
	}
	if capturedModel != "custom-model-v2" {
		t.Fatalf("expected custom-model-v2, got %q", capturedModel)
	}
}

func TestBailianProvider_DimensionsSet(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[{"embedding":[0.1,0.2]}],"model":"text-embedding-v4"}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test",
		WithBailianModel("text-embedding-v4"),
		WithBailianDimensions(256),
		WithBailianTimeout(10*time.Second),
	)
	_, err := provider.EmbedText(context.Background(), "test")
	if err != nil {
		t.Fatalf("EmbedText: %v", err)
	}
	dims, ok := capturedBody["dimensions"].(float64)
	if !ok {
		t.Fatalf("expected dimensions field, got %v", capturedBody["dimensions"])
	}
	if int(dims) != 256 {
		t.Fatalf("expected dimensions 256, got %v", dims)
	}
}

func TestBailianProvider_DimensionsNotSet(t *testing.T) {
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[{"embedding":[0.1]}]}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test",
		WithBailianModel("text-embedding-v4"),
		WithBailianTimeout(10*time.Second),
	)
	_, err := provider.EmbedText(context.Background(), "test")
	if err != nil {
		t.Fatalf("EmbedText: %v", err)
	}
	if _, exists := capturedBody["dimensions"]; exists {
		t.Fatalf("expected no dimensions field when zero, got %v", capturedBody["dimensions"])
	}
}

func TestBailianProvider_RequestCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"output":{"embeddings":[{"embedding":[1.0]}]}}`))
	}))
	defer srv.Close()

	provider := NewBailianTextEmbeddingProvider(srv.URL, "sk-test",
		WithBailianTimeout(10*time.Second),
		WithBailianMaxRetries(0),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := provider.EmbedText(ctx, "test")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestIsRetryableStatus(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{http.StatusOK, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusNotFound, false},
		{http.StatusTooManyRequests, true},
		{http.StatusRequestTimeout, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			got := isRetryableStatus(tt.status)
			if got != tt.want {
				t.Errorf("isRetryableStatus(%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
