package persistence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	commonsai "smart-recruit-commons/ai"
	"smart-recruit-proto/recruitment/pb"
)

const (
	discoveryTimeout    = 15 * time.Second
	discoveryBodyLimit  = 2 << 20
	discoveryModelLimit = 2000
	discoveryCacheTTL   = 5 * time.Minute
)

type discoveredModel struct {
	ModelName                string
	DisplayName              string
	OwnedBy                  string
	ContextWindowTokens      int32
	ContextWindowTokensKnown bool
	MaxInputTokens           int32
	MaxInputTokensKnown      bool
	MaxOutputTokens          int32
	MaxOutputTokensKnown     bool
	Temperature              float64
	TemperatureKnown         bool
	TopP                     float64
	TopPKnown                bool
	CapabilitiesJSON         string
	FieldSources             map[string]metadataFieldSource
}

type modelDiscoveryStrategy interface {
	Discover(context.Context, discoveryRequest) ([]discoveredModel, error)
}

type discoveryRequest struct {
	Provider llmProviderRecord
	APIKey   string
	Client   *http.Client
}

type discoveryCacheEntry struct {
	models    []discoveredModel
	fetchedAt time.Time
}

var modelDiscoveryCache = struct {
	sync.Mutex
	entries map[string]discoveryCacheEntry
}{entries: make(map[string]discoveryCacheEntry)}

var discoveryStrategies = map[string]modelDiscoveryStrategy{
	"openai":            openAIModelDiscovery{},
	"openai_compatible": openAIModelDiscovery{},
	"anthropic":         anthropicModelDiscovery{},
	"azure_openai":      azureModelDiscovery{},
	"google_gemini":     geminiModelDiscovery{},
	"ollama":            ollamaModelDiscovery{},
}

// DiscoverLlmProviderModels retrieves provider catalog data without exposing
// credentials to the browser. Results are cached briefly unless refresh=true.
func (s *NativeStore) DiscoverLlmProviderModels(ctx context.Context, req *pb.DiscoverProviderModelsRequest) (*pb.DiscoverProviderModelsResponse, error) {
	if req.GetProviderId() <= 0 {
		return &pb.DiscoverProviderModelsResponse{Code: configBadRequest, Msg: "common.invalid_request"}, nil
	}
	var provider llmProviderRecord
	if err := s.db.WithContext(ctx).First(&provider, req.GetProviderId()).Error; err != nil {
		return &pb.DiscoverProviderModelsResponse{Code: configNotFound, Msg: "common.not_found"}, nil
	}
	if !provider.IsEnabled {
		return &pb.DiscoverProviderModelsResponse{Code: configUnavailable, Msg: "common.operation_failed"}, nil
	}
	profile, err := commonsai.ResolveProviderProfile(provider.ProviderType, provider.ProtocolType, provider.AuthType)
	if err != nil {
		return &pb.DiscoverProviderModelsResponse{Code: configBadRequest, Msg: "common.invalid_request"}, nil
	}
	strategy := discoveryStrategies[profile.DiscoveryType]
	if strategy == nil {
		return &pb.DiscoverProviderModelsResponse{Code: configUnsupported, Msg: "common.operation_failed"}, nil
	}
	apiKey := ""
	if profile.AuthType != commonsai.AuthNone {
		apiKey, err = s.decryptAPIKey(provider.APIKeyEncrypted)
		if err != nil {
			return &pb.DiscoverProviderModelsResponse{Code: configUnavailable, Msg: "common.operation_failed"}, nil
		}
	}
	cacheKey := fmt.Sprintf("%d|%s|%s|%s", provider.ID, provider.UpdatedAt.UTC().Format(time.RFC3339Nano), provider.BaseURL, nullString(provider.DiscoveryURL))
	models, fetchedAt, cached := loadDiscoveryCache(cacheKey, req.GetRefresh())
	if !cached {
		allowPrivate := profile.DiscoveryType == "ollama"
		client := discoveryHTTPClient(provider, profile.AuthType, apiKey, allowPrivate)
		discoveryCtx, cancel := context.WithTimeout(ctx, discoveryTimeout)
		defer cancel()
		models, err = strategy.Discover(discoveryCtx, discoveryRequest{Provider: provider, APIKey: apiKey, Client: client})
		if err != nil {
			return &pb.DiscoverProviderModelsResponse{Code: configUnavailable, Msg: "common.operation_failed"}, nil
		}
		fetchedAt = time.Now()
		models = annotateProviderModels(models, metadataSourceProviderAPI, discoverySourceRef(provider), fetchedAt)
		models, err = s.enrichModelsFromCatalog(discoveryCtx, provider, models, fetchedAt)
		if err != nil {
			return nil, err
		}
		if err := s.storeModelMetadataObservations(discoveryCtx, provider.ID, models, fetchedAt.Add(24*time.Hour)); err != nil {
			return nil, err
		}
		storeDiscoveryCache(cacheKey, models, fetchedAt)
	}
	configured := make(map[string]bool)
	var configuredNames []string
	if err := s.db.WithContext(ctx).Model(&llmModelRecord{}).Where("provider_id = ?", provider.ID).Pluck("model_name", &configuredNames).Error; err != nil {
		return nil, err
	}
	for _, name := range configuredNames {
		configured[name] = true
	}
	items := make([]*pb.DiscoveredLlmModel, 0, len(models))
	for _, model := range models {
		item := discoveredModelToPB(model)
		item.AlreadyConfigured = configured[model.ModelName]
		items = append(items, item)
	}
	return &pb.DiscoverProviderModelsResponse{Code: configOK, Msg: "common.success", List: items, Source: profile.DiscoveryType, FetchedAt: formatTime(fetchedAt)}, nil
}

// GetLlmProviderModelPreset enriches one catalog item. Ollama exposes detailed
// model information through /api/show, so this is intentionally fetched only
// after the user selects a model instead of multiplying list requests.
func (s *NativeStore) GetLlmProviderModelPreset(ctx context.Context, req *pb.GetProviderModelPresetRequest) (*pb.GetProviderModelPresetResponse, error) {
	if req.GetProviderId() <= 0 || strings.TrimSpace(req.GetModelName()) == "" {
		return &pb.GetProviderModelPresetResponse{Code: configBadRequest, Msg: "common.invalid_request"}, nil
	}
	var provider llmProviderRecord
	if err := s.db.WithContext(ctx).First(&provider, req.GetProviderId()).Error; err != nil {
		return &pb.GetProviderModelPresetResponse{Code: configNotFound, Msg: "common.not_found"}, nil
	}
	profile, err := commonsai.ResolveProviderProfile(provider.ProviderType, provider.ProtocolType, provider.AuthType)
	if err != nil {
		return &pb.GetProviderModelPresetResponse{Code: configBadRequest, Msg: "common.invalid_request"}, nil
	}
	if profile.DiscoveryType != "ollama" {
		catalog, err := s.DiscoverLlmProviderModels(ctx, &pb.DiscoverProviderModelsRequest{ProviderId: provider.ID})
		if err != nil {
			return nil, err
		}
		if catalog.Code != configOK {
			return &pb.GetProviderModelPresetResponse{Code: catalog.Code, Msg: "common.operation_failed"}, nil
		}
		for _, item := range catalog.List {
			if item.GetModelName() == req.GetModelName() {
				return &pb.GetProviderModelPresetResponse{Code: configOK, Msg: "common.success", Model: item, Source: catalog.Source, FetchedAt: catalog.FetchedAt}, nil
			}
		}
		return &pb.GetProviderModelPresetResponse{Code: configNotFound, Msg: "common.not_found"}, nil
	}
	client := discoveryHTTPClient(provider, profile.AuthType, "", true)
	endpoint := strings.TrimSpace(nullString(provider.DiscoveryURL))
	if endpoint != "" && strings.HasSuffix(endpoint, "/api/tags") {
		endpoint = strings.TrimSuffix(endpoint, "/api/tags") + "/api/show"
	}
	if endpoint == "" {
		endpoint = strings.TrimRight(provider.BaseURL, "/") + "/api/show"
	}
	model, err := inspectOllamaModel(ctx, client, endpoint, strings.TrimSpace(req.GetModelName()))
	if err != nil {
		return &pb.GetProviderModelPresetResponse{Code: configUnavailable, Msg: "common.operation_failed"}, nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&llmModelRecord{}).Where("provider_id = ? AND model_name = ?", provider.ID, model.ModelName).Count(&count).Error; err != nil {
		return nil, err
	}
	fetchedAt := time.Now()
	enriched := annotateProviderModels([]discoveredModel{model}, metadataSourceProviderDetail, sanitizeSourceURL(endpoint), fetchedAt)
	enriched, err = s.enrichModelsFromCatalog(ctx, provider, enriched, fetchedAt)
	if err != nil {
		return nil, err
	}
	if err := s.storeModelMetadataObservations(ctx, provider.ID, enriched, fetchedAt.Add(24*time.Hour)); err != nil {
		return nil, err
	}
	item := discoveredModelToPB(enriched[0])
	item.AlreadyConfigured = count > 0
	return &pb.GetProviderModelPresetResponse{Code: configOK, Msg: "common.success", Model: item, Source: "merged", FetchedAt: formatTime(fetchedAt)}, nil
}

func inspectOllamaModel(ctx context.Context, client *http.Client, endpoint, modelName string) (discoveredModel, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return discoveredModel{}, err
	}
	if err := validateDiscoveryURL(ctx, parsed, true); err != nil {
		return discoveredModel{}, err
	}
	body, _ := json.Marshal(map[string]string{"model": modelName})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsed.String(), bytes.NewReader(body))
	if err != nil {
		return discoveredModel{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return discoveredModel{}, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, discoveryBodyLimit+1))
	if err != nil {
		return discoveredModel{}, err
	}
	if len(responseBody) > discoveryBodyLimit {
		return discoveredModel{}, fmt.Errorf("model preset response exceeds 2 MiB")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return discoveredModel{}, fmt.Errorf("model preset returned HTTP %d", resp.StatusCode)
	}
	var payload struct {
		ModelInfo    map[string]any `json:"model_info"`
		Capabilities []string       `json:"capabilities"`
		Details      map[string]any `json:"details"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return discoveredModel{}, fmt.Errorf("decode model preset response: %w", err)
	}
	contextLength := int32(0)
	for key, value := range payload.ModelInfo {
		if !strings.HasSuffix(key, ".context_length") {
			continue
		}
		if number, ok := value.(float64); ok && number > float64(contextLength) {
			contextLength = int32(number)
		}
	}
	metadata := map[string]any{"capabilities": payload.Capabilities, "details": payload.Details}
	caps, _ := json.Marshal(metadata)
	return discoveredModel{ModelName: modelName, DisplayName: modelName, OwnedBy: "local", ContextWindowTokens: contextLength, ContextWindowTokensKnown: contextLength > 0, CapabilitiesJSON: emptyJSONObject(caps)}, nil
}

func discoveredModelToPB(model discoveredModel) *pb.DiscoveredLlmModel {
	return &pb.DiscoveredLlmModel{ModelName: model.ModelName, DisplayName: model.DisplayName, OwnedBy: model.OwnedBy, ContextWindowTokens: model.ContextWindowTokens, ContextWindowTokensKnown: model.ContextWindowTokensKnown, MaxInputTokens: model.MaxInputTokens, MaxInputTokensKnown: model.MaxInputTokensKnown, MaxOutputTokens: model.MaxOutputTokens, MaxOutputTokensKnown: model.MaxOutputTokensKnown, Temperature: model.Temperature, TemperatureKnown: model.TemperatureKnown, TopP: model.TopP, TopPKnown: model.TopPKnown, CapabilitiesJson: model.CapabilitiesJSON, FieldSources: metadataFieldSourcesToPB(model.FieldSources)}
}

func discoverySourceRef(provider llmProviderRecord) string {
	if explicit := strings.TrimSpace(nullString(provider.DiscoveryURL)); explicit != "" {
		return sanitizeSourceURL(explicit)
	}
	return sanitizeSourceURL(provider.BaseURL)
}

func sanitizeSourceURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" {
		return ""
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func loadDiscoveryCache(key string, refresh bool) ([]discoveredModel, time.Time, bool) {
	if refresh {
		return nil, time.Time{}, false
	}
	modelDiscoveryCache.Lock()
	defer modelDiscoveryCache.Unlock()
	entry, ok := modelDiscoveryCache.entries[key]
	if !ok || time.Since(entry.fetchedAt) >= discoveryCacheTTL {
		return nil, time.Time{}, false
	}
	return append([]discoveredModel(nil), entry.models...), entry.fetchedAt, true
}

func storeDiscoveryCache(key string, models []discoveredModel, fetchedAt time.Time) {
	modelDiscoveryCache.Lock()
	defer modelDiscoveryCache.Unlock()
	for existingKey, entry := range modelDiscoveryCache.entries {
		if fetchedAt.Sub(entry.fetchedAt) >= discoveryCacheTTL {
			delete(modelDiscoveryCache.entries, existingKey)
		}
	}
	modelDiscoveryCache.entries[key] = discoveryCacheEntry{models: append([]discoveredModel(nil), models...), fetchedAt: fetchedAt}
}

func discoveryHTTPClient(provider llmProviderRecord, authType, apiKey string, allowPrivate bool) *http.Client {
	headers := parseRuntimeHeaders(nullString(provider.ExtraHeaders))
	transport := http.RoundTripper(discoveryBaseTransport(allowPrivate))
	transport = discoveryAuthTransport{base: transport, authType: authType, apiKey: apiKey, extraHeaders: headers}
	client := &http.Client{Timeout: discoveryTimeout, Transport: transport}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		return validateDiscoveryURL(req.Context(), req.URL, allowPrivate)
	}
	return client
}

func discoveryBaseTransport(allowPrivate bool) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	if allowPrivate {
		return transport
	}
	dialer := &net.Dialer{Timeout: discoveryTimeout, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid discovery address: %w", err)
		}
		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve discovery host: %w", err)
		}
		for _, resolved := range addresses {
			ip := resolved.IP
			if isPrivateOrLocalIP(ip) {
				continue
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		}
		return nil, fmt.Errorf("discovery host has no permitted public address")
	}
	return transport
}

type discoveryAuthTransport struct {
	base         http.RoundTripper
	authType     string
	apiKey       string
	extraHeaders map[string]string
}

func (t discoveryAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	switch t.authType {
	case commonsai.AuthBearer:
		clone.Header.Set("Authorization", "Bearer "+t.apiKey)
	case commonsai.AuthXAPIKey:
		clone.Header.Set("x-api-key", t.apiKey)
		if clone.Header.Get("anthropic-version") == "" {
			clone.Header.Set("anthropic-version", "2023-06-01")
		}
	case commonsai.AuthAzureAPIKey:
		clone.Header.Set("api-key", t.apiKey)
	case commonsai.AuthGoogleKey:
		clone.Header.Set("x-goog-api-key", t.apiKey)
	}
	for key, value := range t.extraHeaders {
		if safeDiscoveryHeader(key) {
			clone.Header.Set(key, value)
		}
	}
	clone.Header.Set("User-Agent", "smart-recruit-model-discovery/1.0")
	return t.base.RoundTrip(clone)
}

func safeDiscoveryHeader(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "", "host", "content-length", "connection", "authorization", "x-api-key", "api-key", "x-goog-api-key":
		return false
	default:
		return true
	}
}

func validateDiscoveryURL(ctx context.Context, target *url.URL, allowPrivate bool) error {
	if target == nil || target.Hostname() == "" {
		return fmt.Errorf("invalid discovery url")
	}
	if target.Scheme != "https" && !(allowPrivate && target.Scheme == "http") {
		return fmt.Errorf("discovery url must use https")
	}
	if allowPrivate {
		return nil
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, target.Hostname())
	if err != nil {
		return fmt.Errorf("resolve discovery host: %w", err)
	}
	for _, address := range addresses {
		if isPrivateOrLocalIP(address.IP) {
			return fmt.Errorf("discovery host resolves to a private or local address")
		}
	}
	return nil
}

func isPrivateOrLocalIP(ip net.IP) bool {
	_, carrierGradeNAT, _ := net.ParseCIDR("100.64.0.0/10")
	return !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || carrierGradeNAT.Contains(ip)
}

func fetchDiscoveryJSON(ctx context.Context, client *http.Client, rawURL string, allowPrivate bool, target any) (int, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return 0, fmt.Errorf("invalid discovery url: %w", err)
	}
	if err := validateDiscoveryURL(ctx, parsed, allowPrivate); err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, discoveryBodyLimit+1))
	if err != nil {
		return resp.StatusCode, err
	}
	if len(body) > discoveryBodyLimit {
		return resp.StatusCode, fmt.Errorf("model discovery response exceeds 2 MiB")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := strings.TrimSpace(string(body))
		if len(detail) > 512 {
			detail = detail[:512]
		}
		return resp.StatusCode, fmt.Errorf("model discovery returned HTTP %d: %s", resp.StatusCode, detail)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return resp.StatusCode, fmt.Errorf("decode model discovery response: %w", err)
	}
	return resp.StatusCode, nil
}

type openAIModelDiscovery struct{}

func (openAIModelDiscovery) Discover(ctx context.Context, req discoveryRequest) ([]discoveredModel, error) {
	candidates := openAIModelURLs(req.Provider)
	var lastErr error
	for _, candidate := range candidates {
		var payload struct {
			Data []struct {
				ID      string `json:"id"`
				OwnedBy string `json:"owned_by"`
			} `json:"data"`
		}
		status, err := fetchDiscoveryJSON(ctx, req.Client, candidate, false, &payload)
		if err != nil {
			lastErr = err
			if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
				continue
			}
			return nil, err
		}
		models := make([]discoveredModel, 0, len(payload.Data))
		for _, item := range payload.Data {
			if strings.TrimSpace(item.ID) != "" {
				models = append(models, discoveredModel{ModelName: item.ID, DisplayName: item.ID, OwnedBy: item.OwnedBy})
			}
		}
		return normalizeDiscoveredModels(models), nil
	}
	return nil, lastErr
}

func openAIModelURLs(provider llmProviderRecord) []string {
	if explicit := strings.TrimSpace(nullString(provider.DiscoveryURL)); explicit != "" {
		return []string{explicit}
	}
	base := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	candidates := make([]string, 0, 3)
	if strings.HasSuffix(base, "/v1") || strings.Contains(base, "/compatible-mode/v1") {
		candidates = append(candidates, base+"/models")
	}
	candidates = append(candidates, base+"/v1/models", base+"/models")
	return uniqueStrings(candidates)
}

type anthropicModelDiscovery struct{}

func (anthropicModelDiscovery) Discover(ctx context.Context, req discoveryRequest) ([]discoveredModel, error) {
	endpoint := strings.TrimSpace(nullString(req.Provider.DiscoveryURL))
	if endpoint == "" {
		endpoint = appendVersionedPath(req.Provider.BaseURL, "v1", "models")
	}
	models := make([]discoveredModel, 0)
	for page := 0; page < 10; page++ {
		var payload struct {
			Data []struct {
				ID             string         `json:"id"`
				DisplayName    string         `json:"display_name"`
				MaxInputTokens int32          `json:"max_input_tokens"`
				MaxTokens      int32          `json:"max_tokens"`
				Capabilities   map[string]any `json:"capabilities"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		if _, err := fetchDiscoveryJSON(ctx, req.Client, endpoint, false, &payload); err != nil {
			return nil, err
		}
		for _, item := range payload.Data {
			caps, _ := json.Marshal(item.Capabilities)
			models = append(models, discoveredModel{ModelName: item.ID, DisplayName: defaultString(item.DisplayName, item.ID), MaxInputTokens: item.MaxInputTokens, MaxInputTokensKnown: item.MaxInputTokens > 0, MaxOutputTokens: item.MaxTokens, MaxOutputTokensKnown: item.MaxTokens > 0, CapabilitiesJSON: emptyJSONObject(caps)})
		}
		if !payload.HasMore || payload.LastID == "" {
			break
		}
		parsed, _ := url.Parse(endpoint)
		query := parsed.Query()
		query.Set("after_id", payload.LastID)
		parsed.RawQuery = query.Encode()
		endpoint = parsed.String()
	}
	return normalizeDiscoveredModels(models), nil
}

type azureModelDiscovery struct{}

func (azureModelDiscovery) Discover(ctx context.Context, req discoveryRequest) ([]discoveredModel, error) {
	endpoint := strings.TrimSpace(nullString(req.Provider.DiscoveryURL))
	if endpoint == "" {
		endpoint = strings.TrimRight(req.Provider.BaseURL, "/") + "/openai/models"
		parsed, _ := url.Parse(endpoint)
		query := parsed.Query()
		query.Set("api-version", defaultString(nullString(req.Provider.APIVersion), "2024-10-21"))
		parsed.RawQuery = query.Encode()
		endpoint = parsed.String()
	}
	var payload struct {
		Data []struct {
			ID           string         `json:"id"`
			OwnedBy      string         `json:"owned_by"`
			Capabilities map[string]any `json:"capabilities"`
		} `json:"data"`
	}
	if _, err := fetchDiscoveryJSON(ctx, req.Client, endpoint, false, &payload); err != nil {
		return nil, err
	}
	models := make([]discoveredModel, 0, len(payload.Data))
	for _, item := range payload.Data {
		caps, _ := json.Marshal(item.Capabilities)
		models = append(models, discoveredModel{ModelName: item.ID, DisplayName: item.ID, OwnedBy: item.OwnedBy, CapabilitiesJSON: emptyJSONObject(caps)})
	}
	return normalizeDiscoveredModels(models), nil
}

type geminiModelDiscovery struct{}

func (geminiModelDiscovery) Discover(ctx context.Context, req discoveryRequest) ([]discoveredModel, error) {
	endpoint := strings.TrimSpace(nullString(req.Provider.DiscoveryURL))
	if endpoint == "" {
		endpoint = appendVersionedPath(req.Provider.BaseURL, defaultString(nullString(req.Provider.APIVersion), "v1beta"), "models")
	}
	models := make([]discoveredModel, 0)
	for page := 0; page < 10; page++ {
		var payload struct {
			Models []struct {
				Name, DisplayName                 string
				InputTokenLimit, OutputTokenLimit int32
				SupportedGenerationMethods        []string
				Temperature, TopP                 *float64
			} `json:"models"`
			NextPageToken string `json:"nextPageToken"`
		}
		if _, err := fetchDiscoveryJSON(ctx, req.Client, endpoint, false, &payload); err != nil {
			return nil, err
		}
		for _, item := range payload.Models {
			if !containsString(item.SupportedGenerationMethods, "generateContent") {
				continue
			}
			name := strings.TrimPrefix(item.Name, "models/")
			caps, _ := json.Marshal(item.SupportedGenerationMethods)
			model := discoveredModel{ModelName: name, DisplayName: defaultString(item.DisplayName, name), MaxInputTokens: item.InputTokenLimit, MaxInputTokensKnown: item.InputTokenLimit > 0, MaxOutputTokens: item.OutputTokenLimit, MaxOutputTokensKnown: item.OutputTokenLimit > 0, CapabilitiesJSON: string(caps)}
			if item.Temperature != nil {
				model.Temperature = *item.Temperature
				model.TemperatureKnown = true
			}
			if item.TopP != nil {
				model.TopP = *item.TopP
				model.TopPKnown = true
			}
			models = append(models, model)
		}
		if payload.NextPageToken == "" {
			break
		}
		parsed, _ := url.Parse(endpoint)
		query := parsed.Query()
		query.Set("pageToken", payload.NextPageToken)
		parsed.RawQuery = query.Encode()
		endpoint = parsed.String()
	}
	return normalizeDiscoveredModels(models), nil
}

type ollamaModelDiscovery struct{}

func (ollamaModelDiscovery) Discover(ctx context.Context, req discoveryRequest) ([]discoveredModel, error) {
	endpoint := strings.TrimSpace(nullString(req.Provider.DiscoveryURL))
	if endpoint == "" {
		endpoint = strings.TrimRight(req.Provider.BaseURL, "/") + "/api/tags"
	}
	var payload struct {
		Models []struct {
			Name, Model string
			Details     map[string]any `json:"details"`
		} `json:"models"`
	}
	if _, err := fetchDiscoveryJSON(ctx, req.Client, endpoint, true, &payload); err != nil {
		return nil, err
	}
	models := make([]discoveredModel, 0, len(payload.Models))
	for _, item := range payload.Models {
		name := defaultString(item.Name, item.Model)
		caps, _ := json.Marshal(item.Details)
		models = append(models, discoveredModel{ModelName: name, DisplayName: name, OwnedBy: "local", CapabilitiesJSON: emptyJSONObject(caps)})
	}
	return normalizeDiscoveredModels(models), nil
}

func appendVersionedPath(baseURL, version, resource string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	version = strings.Trim(version, "/")
	if strings.HasSuffix(base, "/"+version) {
		return base + "/" + resource
	}
	return base + "/" + version + "/" + resource
}

func normalizeDiscoveredModels(models []discoveredModel) []discoveredModel {
	seen := make(map[string]bool)
	out := make([]discoveredModel, 0, len(models))
	for _, model := range models {
		model.ModelName = strings.TrimSpace(model.ModelName)
		if model.ModelName == "" || seen[model.ModelName] || len(out) >= discoveryModelLimit {
			continue
		}
		seen[model.ModelName] = true
		if strings.TrimSpace(model.DisplayName) == "" {
			model.DisplayName = model.ModelName
		}
		out = append(out, model)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModelName < out[j].ModelName })
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func emptyJSONObject(value []byte) string {
	raw := strings.TrimSpace(string(value))
	if raw == "" || raw == "null" || raw == "{}" {
		return ""
	}
	return raw
}
func sanitizeDiscoveryError(err error) string {
	if err == nil {
		return "model discovery failed"
	}
	message := strings.ReplaceAll(err.Error(), "\n", " ")
	if len(message) > 512 {
		message = message[:512]
	}
	return message
}
