package ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	chatmodel "github.com/cloudwego/eino/components/model"
	"google.golang.org/genai"
)

const (
	ProtocolOpenAIChat       = "openai_chat_completions"
	ProtocolAnthropicMessage = "anthropic_messages"
	ProtocolGeminiContent    = "gemini_generate_content"
	ProtocolOllamaChat       = "ollama_chat"

	AuthBearer      = "bearer"
	AuthXAPIKey     = "x_api_key"
	AuthAzureAPIKey = "azure_api_key"
	AuthGoogleKey   = "google_api_key"
	AuthNone        = "none"
)

// ProviderProfile is the canonical mapping between a product-facing provider
// type and the protocol/auth strategies used at runtime and during discovery.
type ProviderProfile struct {
	ProviderType  string
	ProtocolType  string
	AuthType      string
	DiscoveryType string
}

var providerProfiles = map[string]ProviderProfile{
	"openai":            {ProviderType: "openai", ProtocolType: ProtocolOpenAIChat, AuthType: AuthBearer, DiscoveryType: "openai"},
	"anthropic":         {ProviderType: "anthropic", ProtocolType: ProtocolAnthropicMessage, AuthType: AuthXAPIKey, DiscoveryType: "anthropic"},
	"azure_openai":      {ProviderType: "azure_openai", ProtocolType: ProtocolOpenAIChat, AuthType: AuthAzureAPIKey, DiscoveryType: "azure_openai"},
	"google_gemini":     {ProviderType: "google_gemini", ProtocolType: ProtocolGeminiContent, AuthType: AuthGoogleKey, DiscoveryType: "google_gemini"},
	"ollama":            {ProviderType: "ollama", ProtocolType: ProtocolOllamaChat, AuthType: AuthNone, DiscoveryType: "ollama"},
	"openai_compatible": {ProviderType: "openai_compatible", ProtocolType: ProtocolOpenAIChat, AuthType: AuthBearer, DiscoveryType: "openai_compatible"},
	"custom":            {ProviderType: "custom", ProtocolType: ProtocolOpenAIChat, AuthType: AuthBearer, DiscoveryType: "openai_compatible"},
}

// ResolveProviderProfile normalizes historical aliases and applies optional
// explicit protocol/auth overrides for custom providers.
func ResolveProviderProfile(providerType, protocolType, authType string) (ProviderProfile, error) {
	providerType = normalizeProviderType(providerType)
	profile, ok := providerProfiles[providerType]
	if !ok {
		return ProviderProfile{}, fmt.Errorf("unsupported llm provider type %q", providerType)
	}
	if providerType == "custom" {
		if strings.TrimSpace(protocolType) != "" {
			profile.ProtocolType = strings.TrimSpace(protocolType)
		}
		if strings.TrimSpace(authType) != "" {
			profile.AuthType = strings.TrimSpace(authType)
		}
	}
	if _, ok := protocolAdapters[profile.ProtocolType]; !ok {
		return ProviderProfile{}, fmt.Errorf("unsupported llm protocol type %q", profile.ProtocolType)
	}
	// Discovery follows the inference protocol. A provider that exposes only
	// one of the two surfaces must be configured with the protocol it actually
	// implements; there is intentionally no independent discovery protocol.
	profile.DiscoveryType = discoveryTypeForProtocol(profile.ProtocolType, profile.AuthType, profile.ProviderType)
	return profile, nil
}

func discoveryTypeForProtocol(protocolType, authType, providerType string) string {
	switch protocolType {
	case ProtocolAnthropicMessage:
		return "anthropic"
	case ProtocolGeminiContent:
		return "google_gemini"
	case ProtocolOllamaChat:
		return "ollama"
	case ProtocolOpenAIChat:
		if authType == AuthAzureAPIKey || providerType == "azure_openai" {
			return "azure_openai"
		}
		if providerType == "openai" {
			return "openai"
		}
		return "openai_compatible"
	default:
		return ""
	}
}

func normalizeProviderType(providerType string) string {
	switch strings.TrimSpace(providerType) {
	case "", "deepseek":
		return "openai_compatible"
	case "azure":
		return "azure_openai"
	case "google":
		return "google_gemini"
	case "other":
		return "custom"
	default:
		return strings.TrimSpace(providerType)
	}
}

// ChatModelRequest is protocol-neutral input to a ToolCallingChatModel adapter.
type ChatModelRequest struct {
	ProviderType string
	ProtocolType string
	AuthType     string
	APIKey       string
	Model        string
	BaseURL      string
	APIVersion   string
	ExtraHeaders map[string]string
	Timeout      time.Duration
	Params       ModelParams
}

type protocolAdapter interface {
	New(context.Context, ChatModelRequest) (chatmodel.ToolCallingChatModel, error)
}

var protocolAdapters = map[string]protocolAdapter{
	ProtocolOpenAIChat:       openAIChatAdapter{},
	ProtocolAnthropicMessage: anthropicMessageAdapter{},
	ProtocolGeminiContent:    geminiContentAdapter{},
	ProtocolOllamaChat:       ollamaChatAdapter{},
}

// NewProtocolChatModel resolves the Provider Profile and delegates model
// creation to the registered protocol strategy.
func NewProtocolChatModel(ctx context.Context, req ChatModelRequest) (chatmodel.ToolCallingChatModel, error) {
	profile, err := ResolveProviderProfile(req.ProviderType, req.ProtocolType, req.AuthType)
	if err != nil {
		return nil, err
	}
	if profile.AuthType != AuthNone && strings.TrimSpace(req.APIKey) == "" {
		return nil, fmt.Errorf("llm api_key is required for auth type %s", profile.AuthType)
	}
	req.ProviderType = profile.ProviderType
	req.ProtocolType = profile.ProtocolType
	req.AuthType = profile.AuthType
	adapter := protocolAdapters[profile.ProtocolType]
	return adapter.New(ctx, req)
}

type openAIChatAdapter struct{}

func (openAIChatAdapter) New(ctx context.Context, req ChatModelRequest) (chatmodel.ToolCallingChatModel, error) {
	cfg := &openai.ChatModelConfig{
		APIKey: req.APIKey, Model: req.Model, BaseURL: req.BaseURL, Timeout: req.Timeout,
		HTTPClient: decoratedHTTPClient(req.Timeout, req.ExtraHeaders, req.AuthType, req.APIKey),
	}
	if normalizeProviderType(req.ProviderType) == "azure_openai" || req.AuthType == AuthAzureAPIKey {
		cfg.ByAzure = true
		cfg.APIVersion = strings.TrimSpace(req.APIVersion)
		if cfg.APIVersion == "" {
			cfg.APIVersion = "2024-10-21"
		}
		cfg.AzureModelMapperFunc = func(model string) string { return model }
	}
	applyOpenAIParams(cfg, req.Params)
	return openai.NewChatModel(ctx, cfg)
}

func applyOpenAIParams(cfg *openai.ChatModelConfig, params ModelParams) {
	if params.Temperature != nil {
		value := float32(*params.Temperature)
		cfg.Temperature = &value
	}
	if params.TopP != nil {
		value := float32(*params.TopP)
		cfg.TopP = &value
	}
	if params.MaxTokens != nil && *params.MaxTokens > 0 {
		value := *params.MaxTokens
		cfg.MaxTokens = &value
	}
}

type anthropicMessageAdapter struct{}

func (anthropicMessageAdapter) New(_ context.Context, req ChatModelRequest) (chatmodel.ToolCallingChatModel, error) {
	return newAnthropicChatModel(AnthropicChatModelConfig{
		APIKey: req.APIKey, BaseURL: req.BaseURL, Model: req.Model,
		Timeout: int(req.Timeout.Seconds()), MaxTokens: req.Params.MaxTokens,
		Temperature: req.Params.Temperature, HTTPClient: decoratedHTTPClient(req.Timeout, req.ExtraHeaders, req.AuthType, req.APIKey),
	}), nil
}

type geminiContentAdapter struct{}

func (geminiContentAdapter) New(ctx context.Context, req ChatModelRequest) (chatmodel.ToolCallingChatModel, error) {
	timeout := req.Timeout
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:      req.APIKey,
		HTTPClient:  decoratedHTTPClient(req.Timeout, req.ExtraHeaders, req.AuthType, req.APIKey),
		HTTPOptions: genai.HTTPOptions{BaseURL: req.BaseURL, APIVersion: req.APIVersion, Timeout: &timeout},
	})
	if err != nil {
		return nil, err
	}
	cfg := &gemini.Config{Client: client, Model: req.Model, MaxTokens: req.Params.MaxTokens}
	if req.Params.Temperature != nil {
		value := float32(*req.Params.Temperature)
		cfg.Temperature = &value
	}
	if req.Params.TopP != nil {
		value := float32(*req.Params.TopP)
		cfg.TopP = &value
	}
	return gemini.NewChatModel(ctx, cfg)
}

type ollamaChatAdapter struct{}

func (ollamaChatAdapter) New(ctx context.Context, req ChatModelRequest) (chatmodel.ToolCallingChatModel, error) {
	options := &ollama.Options{}
	if req.Params.Temperature != nil {
		options.Temperature = float32(*req.Params.Temperature)
	}
	if req.Params.TopP != nil {
		options.TopP = float32(*req.Params.TopP)
	}
	if req.Params.MaxTokens != nil {
		options.NumPredict = *req.Params.MaxTokens
	}
	return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: req.BaseURL, Timeout: req.Timeout, HTTPClient: decoratedHTTPClient(req.Timeout, req.ExtraHeaders, req.AuthType, req.APIKey),
		Model: req.Model, Options: options,
	})
}

type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	for key, value := range t.headers {
		if isSafeExtraHeader(key) {
			clone.Header.Set(key, value)
		}
	}
	return t.base.RoundTrip(clone)
}

type authRoundTripper struct {
	base     http.RoundTripper
	authType string
	apiKey   string
}

func (t authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	clone.Header.Del("Authorization")
	clone.Header.Del("x-api-key")
	clone.Header.Del("api-key")
	clone.Header.Del("x-goog-api-key")
	switch t.authType {
	case AuthBearer, "":
		clone.Header.Set("Authorization", "Bearer "+t.apiKey)
	case AuthXAPIKey:
		clone.Header.Set("x-api-key", t.apiKey)
	case AuthAzureAPIKey:
		clone.Header.Set("api-key", t.apiKey)
	case AuthGoogleKey:
		clone.Header.Set("x-goog-api-key", t.apiKey)
	case AuthNone:
	}
	return t.base.RoundTrip(clone)
}

func decoratedHTTPClient(timeout time.Duration, headers map[string]string, authType, apiKey string) *http.Client {
	var transport http.RoundTripper = http.DefaultTransport
	if len(headers) > 0 {
		transport = headerRoundTripper{base: transport, headers: headers}
	}
	transport = authRoundTripper{base: transport, authType: authType, apiKey: apiKey}
	return &http.Client{Timeout: timeout, Transport: transport}
}

func isSafeExtraHeader(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "host", "content-length", "connection", "authorization", "x-api-key", "api-key", "x-goog-api-key":
		return false
	default:
		return strings.TrimSpace(key) != ""
	}
}
