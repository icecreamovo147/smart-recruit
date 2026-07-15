package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultRunnerHTTPDiscoveryAndCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "tools/list":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"search","description":"Search","inputSchema":{"type":"object"}}]}}`))
		case "tools/call":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"tool result"}]}}`))
		default:
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"not found"}}`))
		}
	}))
	defer server.Close()

	runner := NewRunner()
	cfg := ServerConfig{ID: 1, Name: "http", Transport: "http", CommandOrURL: server.URL, Enabled: true, AllowPrivateNetwork: true}
	tools, err := runner.ListTools(context.Background(), cfg)
	if err != nil {
		t.Fatalf("ListTools error = %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "search" || tools[0].SchemaJSON == "" {
		t.Fatalf("tools = %#v", tools)
	}
	result, err := runner.CallTool(context.Background(), cfg, "search", map[string]any{"query": "alice"})
	if err != nil {
		t.Fatalf("CallTool error = %v", err)
	}
	if result.Content != "tool result" || result.Error != "" {
		t.Fatalf("result = %#v", result)
	}
}
