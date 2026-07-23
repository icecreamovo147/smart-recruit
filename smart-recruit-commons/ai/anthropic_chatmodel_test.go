package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestAnthropicBuildRequestRejectsSystemOnlyMessages(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calls.Add(1)
	}))
	defer server.Close()
	model := newAnthropicChatModel(AnthropicChatModelConfig{BaseURL: server.URL, Model: "test-model"})
	stream, err := model.Stream(t.Context(), []*schema.Message{schema.SystemMessage("system instructions")})
	if err == nil {
		t.Fatalf("Stream error = nil, stream=%v", stream)
	}
	if !strings.Contains(err.Error(), "non-system message") {
		t.Fatalf("error = %q, want non-system validation", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("provider calls = %d, want 0", calls.Load())
	}
}

func TestAnthropicBuildRequestSerializesNonEmptyMessageArray(t *testing.T) {
	model := newAnthropicChatModel(AnthropicChatModelConfig{Model: "test-model"})
	body, err := model.buildRequest([]*schema.Message{
		schema.SystemMessage("system instructions"),
		schema.UserMessage("analyze this application"),
	}, true, nil)
	if err != nil {
		t.Fatalf("buildRequest returned error: %v", err)
	}
	var request struct {
		Messages []anthropicMsg `json:"messages"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if len(request.Messages) != 1 || request.Messages[0].Role != "user" {
		t.Fatalf("messages = %#v, want one user message", request.Messages)
	}
}
