package ai

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestResolveProviderProfileCanonicalizesProviderAndProtocol(t *testing.T) {
	tests := []struct {
		name, provider, protocol, auth, wantProvider, wantProtocol, wantAuth, wantDiscovery string
	}{
		{name: "legacy azure alias", provider: "azure", wantProvider: "azure_openai", wantProtocol: ProtocolOpenAIChat, wantAuth: AuthAzureAPIKey, wantDiscovery: "azure_openai"},
		{name: "anthropic ignores mismatched builtin override", provider: "anthropic", protocol: ProtocolOllamaChat, auth: AuthNone, wantProvider: "anthropic", wantProtocol: ProtocolAnthropicMessage, wantAuth: AuthXAPIKey, wantDiscovery: "anthropic"},
		{name: "custom ollama follows inference protocol", provider: "custom", protocol: ProtocolOllamaChat, auth: AuthNone, wantProvider: "custom", wantProtocol: ProtocolOllamaChat, wantAuth: AuthNone, wantDiscovery: "ollama"},
		{name: "custom openai follows inference protocol", provider: "custom", protocol: ProtocolOpenAIChat, auth: AuthBearer, wantProvider: "custom", wantProtocol: ProtocolOpenAIChat, wantAuth: AuthBearer, wantDiscovery: "openai_compatible"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, err := ResolveProviderProfile(tt.provider, tt.protocol, tt.auth)
			if err != nil {
				t.Fatalf("ResolveProviderProfile() error = %v", err)
			}
			if profile.ProviderType != tt.wantProvider || profile.ProtocolType != tt.wantProtocol || profile.AuthType != tt.wantAuth {
				t.Fatalf("profile = %#v, want provider=%s protocol=%s auth=%s", profile, tt.wantProvider, tt.wantProtocol, tt.wantAuth)
			}
			if profile.DiscoveryType != tt.wantDiscovery {
				t.Fatalf("discovery = %s, want %s", profile.DiscoveryType, tt.wantDiscovery)
			}
		})
	}
}

func TestNewProtocolChatModelAuthenticationPolicy(t *testing.T) {
	if _, err := NewProtocolChatModel(context.Background(), ChatModelRequest{ProviderType: "openai", Model: "gpt-test", BaseURL: "https://example.com", Timeout: time.Second}); err == nil {
		t.Fatal("expected missing API key to be rejected")
	}
	if _, err := NewProtocolChatModel(context.Background(), ChatModelRequest{ProviderType: "ollama", Model: "qwen3", BaseURL: "http://127.0.0.1:11434", Timeout: time.Second}); err != nil {
		t.Fatalf("Ollama with no API key should be constructible: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestHeaderRoundTripperBlocksCredentialOverrides(t *testing.T) {
	transport := headerRoundTripper{base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer runtime" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := req.Header.Get("X-Tenant"); got != "acme" {
			t.Fatalf("X-Tenant = %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
	}), headers: map[string]string{"Authorization": "Bearer attacker", "X-Tenant": "acme"}}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set("Authorization", "Bearer runtime")
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
}

func TestAuthRoundTripperAppliesSelectedStrategy(t *testing.T) {
	transport := authRoundTripper{base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "" {
			t.Fatal("bearer header must be removed")
		}
		if got := req.Header.Get("x-api-key"); got != "secret" {
			t.Fatalf("x-api-key = %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
	}), authType: AuthXAPIKey, apiKey: "secret"}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set("Authorization", "Bearer legacy")
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
}
