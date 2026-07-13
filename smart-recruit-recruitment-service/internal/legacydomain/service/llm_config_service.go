package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"smart-recruit-commons/pkg/crypto"
	"smart-recruit-platform-go/logger"
	"smart-recruit-platform-go/serviceconfig"
	"smart-recruit-proto/recruitment/pb"
	"smart-recruit-recruitment-service/internal/legacydomain/ai"
	"smart-recruit-recruitment-service/internal/legacydomain/model"
	"smart-recruit-recruitment-service/internal/legacydomain/repository"
)

// LlmConfigService implements the pb.LlmConfigServiceServer interface.
type LlmConfigService struct {
	pb.UnimplementedLlmConfigServiceServer
	providerRepo *repository.ProviderRepo
	modelRepo    *repository.ModelConfigRepo
	encKey       crypto.EncryptionKey
	httpClient   *http.Client
}

// LlmRuntimeModelConfig is the complete model configuration needed to create
// a runtime AI client for one request.
type LlmRuntimeModelConfig struct {
	ModelID             int64
	ProviderType        string
	ProviderName        string
	APIKey              string
	ModelName           string
	BaseURL             string
	Params              ai.ModelParams
	Concurrency         int32
	Timeout             time.Duration
	ContextWindowTokens int32
	MaxOutputTokens     int32
}

// NewLlmConfigService creates a new LlmConfigService.
func NewLlmConfigService(providerRepo *repository.ProviderRepo, modelRepo *repository.ModelConfigRepo, encKey crypto.EncryptionKey) *LlmConfigService {
	return &LlmConfigService{
		providerRepo: providerRepo,
		modelRepo:    modelRepo,
		encKey:       encKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ── Provider CRUD ───────────────────────────────────────────────────────

func (s *LlmConfigService) ListProviders(ctx context.Context, req *pb.ListProvidersRequest) (*pb.ListProvidersResponse, error) {
	page, pageSize := normalizeManagementPage(req.GetPage(), req.GetPageSize())

	providers, total, err := s.providerRepo.List(ctx, page, pageSize)
	if err != nil {
		logger.L().Error("list providers failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list providers failed")
	}

	list := make([]*pb.LlmProviderInfo, 0, len(providers))
	for _, p := range providers {
		list = append(list, s.providerToInfo(&p))
	}
	return &pb.ListProvidersResponse{
		Code:  0,
		Msg:   "ok",
		Total: total,
		List:  list,
	}, nil
}

func (s *LlmConfigService) CreateProvider(ctx context.Context, req *pb.CreateProviderRequest) (*pb.ProviderResponse, error) {
	if strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if strings.TrimSpace(req.GetBaseUrl()) == "" {
		return nil, status.Error(codes.InvalidArgument, "base_url is required")
	}
	if strings.TrimSpace(req.GetApiKey()) == "" {
		return nil, status.Error(codes.InvalidArgument, "api_key is required")
	}
	if strings.TrimSpace(req.GetProviderType()) == "" {
		return nil, status.Error(codes.InvalidArgument, "provider_type is required")
	}

	// Encrypt the API key
	encrypted, err := crypto.Encrypt(s.encKey, []byte(req.GetApiKey()))
	if err != nil {
		logger.L().Error("encrypt api key failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "encrypt api key failed")
	}

	var extraHeaders *string
	if req.GetExtraHeadersJson() != "" {
		eh, err := extraHeadersPtrFromRequest(req.GetExtraHeadersJson())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		extraHeaders = eh
	}

	provider := &model.LlmProvider{
		Name:            strings.TrimSpace(req.GetName()),
		BaseURL:         strings.TrimSpace(req.GetBaseUrl()),
		APIKeyEncrypted: encrypted,
		ProviderType:    strings.TrimSpace(req.GetProviderType()),
		ExtraHeaders:    extraHeaders,
		IsEnabled:       1,
	}

	if err := s.providerRepo.Create(ctx, provider); err != nil {
		logger.L().Error("create provider failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create provider failed")
	}

	return &pb.ProviderResponse{
		Code:     0,
		Msg:      "ok",
		Provider: s.providerToInfo(provider),
	}, nil
}

func (s *LlmConfigService) UpdateProvider(ctx context.Context, req *pb.UpdateProviderRequest) (*pb.ProviderResponse, error) {
	id := req.GetId()
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	existing, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "provider not found")
		}
		logger.L().Error("get provider failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get provider failed")
	}

	updates := map[string]any{}
	if req.GetName() != "" {
		updates["name"] = strings.TrimSpace(req.GetName())
	}
	if req.GetBaseUrl() != "" {
		updates["base_url"] = strings.TrimSpace(req.GetBaseUrl())
	}
	if req.GetApiKey() != "" {
		encrypted, err := crypto.Encrypt(s.encKey, []byte(req.GetApiKey()))
		if err != nil {
			logger.L().Error("encrypt api key failed", zap.Error(err))
			return nil, status.Error(codes.Internal, "encrypt api key failed")
		}
		updates["api_key_encrypted"] = encrypted
	}
	if req.GetProviderType() != "" {
		updates["provider_type"] = strings.TrimSpace(req.GetProviderType())
	}
	if req.GetExtraHeadersJson() != "" || req.GetExtraHeadersSet() {
		if req.GetExtraHeadersJson() != "" {
			eh, err := extraHeadersPtrFromRequest(req.GetExtraHeadersJson())
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
			updates["extra_headers"] = eh
		} else {
			updates["extra_headers"] = nil
		}
	}
	if req.GetIsEnabledSet() {
		v := int32(0)
		if req.GetIsEnabled() {
			v = 1
		}
		updates["is_enabled"] = v
	}

	if len(updates) > 0 {
		if err := s.providerRepo.UpdatePartial(ctx, id, updates); err != nil {
			logger.L().Error("update provider failed", zap.Error(err))
			return nil, status.Error(codes.Internal, "update provider failed")
		}
	}

	// Re-fetch updated provider
	updated, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		logger.L().Error("get updated provider failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get updated provider failed")
	}

	_ = existing // used to check before

	return &pb.ProviderResponse{
		Code:     0,
		Msg:      "ok",
		Provider: s.providerToInfo(updated),
	}, nil
}

func (s *LlmConfigService) DeleteProvider(ctx context.Context, req *pb.DeleteProviderRequest) (*pb.CommonResponse, error) {
	id := req.GetId()
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.providerRepo.Delete(ctx, id); err != nil {
		logger.L().Error("delete provider failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "delete provider failed")
	}

	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}

func (s *LlmConfigService) TestProviderConnection(ctx context.Context, req *pb.TestProviderConnectionRequest) (*pb.TestProviderConnectionResponse, error) {
	id := req.GetProviderId()
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "provider_id is required")
	}

	provider, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.TestProviderConnectionResponse{
				Code:    1,
				Msg:     "provider not found",
				Success: false,
				Detail:  "provider not found",
			}, nil
		}
		logger.L().Error("get provider failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get provider failed")
	}

	// Decrypt API key
	apiKeyBytes, err := crypto.Decrypt(s.encKey, provider.APIKeyEncrypted)
	if err != nil {
		logger.L().Error("decrypt api key for test failed", zap.Error(err))
		return &pb.TestProviderConnectionResponse{
			Code:    1,
			Msg:     "decrypt api key failed",
			Success: false,
			Detail:  "failed to decrypt api key",
		}, nil
	}
	apiKey := string(apiKeyBytes)

	// Pick a model name from configured models for this provider
	models, err := s.modelRepo.GetEnabledModelsByProvider(ctx, id)
	if err != nil || len(models) == 0 {
		return &pb.TestProviderConnectionResponse{
			Code:    1,
			Msg:     "no model configured",
			Success: false,
			Detail:  "no enabled model found for this provider, please add a model first",
		}, nil
	}
	modelName := models[0].ModelName

	// Build a minimal chat request based on provider protocol
	baseURL := strings.TrimRight(provider.BaseURL, "/")
	var (
		httpReq *http.Request
		body    string
	)
	switch provider.ProviderType {
	case "anthropic":
		body = fmt.Sprintf(`{"model":"%s","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`, modelName)
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/messages",
			strings.NewReader(body))
		if err == nil {
			httpReq.Header.Set("x-api-key", apiKey)
			httpReq.Header.Set("anthropic-version", "2023-06-01")
		}
	case "ollama":
		body = fmt.Sprintf(`{"model":"%s","stream":false,"messages":[{"role":"user","content":"hi"}]}`, modelName)
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/chat",
			strings.NewReader(body))
	default: // openai_compatible, deepseek, etc.
		body = fmt.Sprintf(`{"model":"%s","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`, modelName)
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions",
			strings.NewReader(body))
		if err == nil {
			httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}
	if err != nil {
		return &pb.TestProviderConnectionResponse{
			Code:    1,
			Msg:     "create request failed",
			Success: false,
			Detail:  err.Error(),
		}, nil
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Add extra headers if present (may override the default auth header)
	if provider.ExtraHeaders != nil && *provider.ExtraHeaders != "" {
		extra, err := parseExtraHeadersForUse(*provider.ExtraHeaders)
		if err != nil {
			return &pb.TestProviderConnectionResponse{
				Code:    1,
				Msg:     "invalid stored extra headers",
				Success: false,
				Detail:  err.Error(),
			}, nil
		}
		for k, v := range extra {
			httpReq.Header.Set(k, v)
		}
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return &pb.TestProviderConnectionResponse{
			Code:    1,
			Msg:     "connection failed",
			Success: false,
			Detail:  fmt.Sprintf("request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return &pb.TestProviderConnectionResponse{
			Code:    0,
			Msg:     "ok",
			Success: true,
			Detail:  fmt.Sprintf("connected successfully, HTTP %d: %s", resp.StatusCode, string(bodyBytes)),
		}, nil
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return &pb.TestProviderConnectionResponse{
		Code:    1,
		Msg:     "connection test failed",
		Success: false,
		Detail:  fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)),
	}, nil
}

// ── Model CRUD ──────────────────────────────────────────────────────────

func (s *LlmConfigService) ListModels(ctx context.Context, req *pb.ListModelsRequest) (*pb.ListModelsResponse, error) {
	page, pageSize := normalizeManagementPage(req.GetPage(), req.GetPageSize())

	models, total, err := s.modelRepo.List(ctx, page, pageSize, req.GetProviderId())
	if err != nil {
		logger.L().Error("list models failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list models failed")
	}

	// Batch-fetch provider names to avoid N+1 queries
	providerIDs := make([]int64, 0, len(models))
	providerIDSet := map[int64]struct{}{}
	for _, m := range models {
		if _, ok := providerIDSet[m.ProviderID]; !ok {
			providerIDSet[m.ProviderID] = struct{}{}
			providerIDs = append(providerIDs, m.ProviderID)
		}
	}
	providerNameMap := map[int64]string{}
	if len(providerIDs) > 0 {
		if providers, err := s.providerRepo.FindByIDs(ctx, providerIDs); err == nil {
			for i := range providers {
				providerNameMap[providers[i].ID] = providers[i].Name
			}
		}
	}

	list := make([]*pb.LlmModelInfo, 0, len(models))
	for _, m := range models {
		list = append(list, s.modelToInfo(&m, providerNameMap[m.ProviderID]))
	}

	return &pb.ListModelsResponse{
		Code:  0,
		Msg:   "ok",
		Total: total,
		List:  list,
	}, nil
}

func (s *LlmConfigService) CreateModel(ctx context.Context, req *pb.CreateModelRequest) (*pb.ModelResponse, error) {
	if req.GetProviderId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "provider_id is required")
	}
	if strings.TrimSpace(req.GetModelName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "model_name is required")
	}

	// Verify provider exists and get provider name
	providerName := ""
	provider, err := s.providerRepo.GetByID(ctx, req.GetProviderId())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "provider not found")
		}
		return nil, status.Error(codes.Internal, "get provider failed")
	}
	providerName = provider.Name
	if req.GetIsDefault() {
		if err := validateRunnableDefaultModel(provider, 1); err != nil {
			return nil, err
		}
	}

	temperature := req.GetTemperature()
	if temperature == 0 {
		temperature = 0.7
	}
	topP := req.GetTopP()
	if topP == 0 {
		topP = 1.0
	}
	maxTokens := req.GetMaxTokens()
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	maxConcurrency := req.GetMaxConcurrency()
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}
	timeoutSeconds := req.GetTimeoutSeconds()
	if timeoutSeconds <= 0 {
		timeoutSeconds = 90
	}

	model := &model.LlmModel{
		ProviderID:     req.GetProviderId(),
		ModelName:      strings.TrimSpace(req.GetModelName()),
		DisplayName:    strings.TrimSpace(req.GetDisplayName()),
		Temperature:    temperature,
		TopP:           topP,
		MaxTokens:      maxTokens,
		MaxConcurrency: maxConcurrency,
		TimeoutSeconds: timeoutSeconds,
		IsEnabled:      1,
		IsDefault:      0,
	}

	contextWindowTokens := req.GetContextWindowTokens()
	if contextWindowTokens < 0 {
		return nil, status.Error(codes.InvalidArgument, "context_window_tokens must be >= 0")
	}
	model.ContextWindowTokens = contextWindowTokens

	if contextWindowTokens > 0 && maxTokens >= contextWindowTokens {
		return nil, status.Error(codes.InvalidArgument, "max_tokens must be less than context_window_tokens when context_window_tokens is configured")
	}

	if req.GetIsDefault() {
		model.IsDefault = 1
	}

	createFn := s.modelRepo.Create
	if req.GetIsDefault() {
		createFn = s.modelRepo.CreateWithDefault
	}
	if err := createFn(ctx, model); err != nil {
		logger.L().Error("create model failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create model failed")
	}

	return &pb.ModelResponse{
		Code:  0,
		Msg:   "ok",
		Model: s.modelToInfo(model, providerName),
	}, nil
}

func (s *LlmConfigService) UpdateModel(ctx context.Context, req *pb.UpdateModelRequest) (*pb.ModelResponse, error) {
	id := req.GetId()
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	existing, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "model not found")
		}
		return nil, status.Error(codes.Internal, "get model failed")
	}

	updates := map[string]any{}
	finalMaxTokens := existing.MaxTokens
	finalContextWindowTokens := existing.ContextWindowTokens
	if req.GetModelName() != "" {
		updates["model_name"] = strings.TrimSpace(req.GetModelName())
	}
	if req.GetDisplayName() != "" {
		updates["display_name"] = strings.TrimSpace(req.GetDisplayName())
	}
	if req.GetTemperatureSet() {
		updates["temperature"] = req.GetTemperature()
	}
	if req.GetTopPSet() {
		updates["top_p"] = req.GetTopP()
	}
	if req.GetMaxTokensSet() {
		maxTokens := req.GetMaxTokens()
		if maxTokens <= 0 {
			return nil, status.Error(codes.InvalidArgument, "max_tokens must be > 0")
		}
		finalMaxTokens = maxTokens
		updates["max_tokens"] = maxTokens
	}
	if req.GetContextWindowTokensSet() {
		ct := req.GetContextWindowTokens()
		if ct < 0 {
			return nil, status.Error(codes.InvalidArgument, "context_window_tokens must be >= 0")
		}
		finalContextWindowTokens = ct
		updates["context_window_tokens"] = ct
	}
	if req.GetMaxConcurrencySet() {
		updates["max_concurrency"] = req.GetMaxConcurrency()
	}
	if req.GetTimeoutSecondsSet() {
		updates["timeout_seconds"] = req.GetTimeoutSeconds()
	}
	if req.GetIsEnabledSet() {
		v := int32(0)
		if req.GetIsEnabled() {
			v = 1
		}
		updates["is_enabled"] = v
	}
	if req.GetIsDefaultSet() && req.GetIsDefault() {
		finalIsEnabled := existing.IsEnabled
		if req.GetIsEnabledSet() {
			if req.GetIsEnabled() {
				finalIsEnabled = 1
			} else {
				finalIsEnabled = 0
			}
		}
		provider, err := s.providerRepo.GetByID(ctx, existing.ProviderID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, status.Error(codes.NotFound, "provider not found")
			}
			return nil, status.Error(codes.Internal, "get provider failed")
		}
		if err := validateRunnableDefaultModel(provider, finalIsEnabled); err != nil {
			return nil, err
		}
		updates["is_default"] = 1
	} else if req.GetIsDefaultSet() && !req.GetIsDefault() {
		updates["is_default"] = 0
	}

	if finalContextWindowTokens > 0 && finalMaxTokens >= finalContextWindowTokens {
		return nil, status.Error(codes.InvalidArgument, "max_tokens must be less than context_window_tokens when context_window_tokens is configured")
	}

	if len(updates) > 0 {
		updateFn := s.modelRepo.UpdatePartial
		if req.GetIsDefaultSet() && req.GetIsDefault() {
			updateFn = s.modelRepo.UpdatePartialWithDefault
		}
		if err := updateFn(ctx, id, updates); err != nil {
			logger.L().Error("update model failed", zap.Error(err))
			return nil, status.Error(codes.Internal, "update model failed")
		}
	}

	// Re-fetch updated model
	updated, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, "get updated model failed")
	}

	providerName := ""
	if provider, err := s.providerRepo.GetByID(ctx, updated.ProviderID); err == nil {
		providerName = provider.Name
	}

	return &pb.ModelResponse{
		Code:  0,
		Msg:   "ok",
		Model: s.modelToInfo(updated, providerName),
	}, nil
}

func (s *LlmConfigService) DeleteModel(ctx context.Context, req *pb.DeleteModelRequest) (*pb.CommonResponse, error) {
	id := req.GetId()
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.modelRepo.Delete(ctx, id); err != nil {
		logger.L().Error("delete model failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "delete model failed")
	}

	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}

// ── Helper methods ──────────────────────────────────────────────────────

func (s *LlmConfigService) providerToInfo(p *model.LlmProvider) *pb.LlmProviderInfo {
	// Decrypt to get the raw key for masking
	apiKeyMasked := ""
	if keyBytes, err := crypto.Decrypt(s.encKey, p.APIKeyEncrypted); err == nil {
		apiKeyMasked = crypto.MaskAPIKey(string(keyBytes))
	}

	// Mask extra_headers values
	extraHeaders := ""
	if p.ExtraHeaders != nil {
		extraHeaders = maskExtraHeadersForResponse(*p.ExtraHeaders)
	}

	return &pb.LlmProviderInfo{
		Id:               p.ID,
		Name:             p.Name,
		BaseUrl:          p.BaseURL,
		ApiKeyMasked:     apiKeyMasked,
		ProviderType:     p.ProviderType,
		ExtraHeadersJson: extraHeaders,
		IsEnabled:        p.IsEnabled == 1,
		CreatedAt:        p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        p.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *LlmConfigService) modelToInfo(m *model.LlmModel, providerName string) *pb.LlmModelInfo {
	return &pb.LlmModelInfo{
		Id:                  m.ID,
		ProviderId:          m.ProviderID,
		ModelName:           m.ModelName,
		DisplayName:         m.DisplayName,
		Temperature:         m.Temperature,
		TopP:                m.TopP,
		MaxTokens:           m.MaxTokens,
		ContextWindowTokens: m.ContextWindowTokens,
		MaxConcurrency:      m.MaxConcurrency,
		TimeoutSeconds:      m.TimeoutSeconds,
		IsEnabled:           m.IsEnabled == 1,
		IsDefault:           m.IsDefault == 1,
		CreatedAt:           m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           m.UpdatedAt.Format(time.RFC3339),
		ProviderName:        providerName,
	}
}

func validateRunnableDefaultModel(provider *model.LlmProvider, modelEnabled int32) error {
	if modelEnabled != 1 {
		return status.Error(codes.FailedPrecondition, "default model must be enabled")
	}
	if provider == nil || provider.IsEnabled != 1 {
		return status.Error(codes.FailedPrecondition, "default model provider must be enabled")
	}
	return nil
}

// GetModelRuntimeConfig looks up the complete model config for runtime model
// selection, including generation parameters from llm_models.
func (s *LlmConfigService) GetModelRuntimeConfig(ctx context.Context, modelID int64) (*LlmRuntimeModelConfig, error) {
	llmModel, err := s.modelRepo.GetByID(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("model %d not found: %w", modelID, err)
	}
	if llmModel.IsEnabled != 1 {
		return nil, fmt.Errorf("model %d is disabled", modelID)
	}
	provider, err := s.providerRepo.GetByID(ctx, llmModel.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("provider %d not found: %w", llmModel.ProviderID, err)
	}
	if provider.IsEnabled != 1 {
		return nil, fmt.Errorf("provider %d is disabled", provider.ID)
	}
	keyBytes, err := crypto.Decrypt(s.encKey, provider.APIKeyEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt api key for provider %d: %w", provider.ID, err)
	}
	params := modelParamsFromConfig(llmModel)
	return &LlmRuntimeModelConfig{
		ModelID:             llmModel.ID,
		ProviderType:        provider.ProviderType,
		ProviderName:        provider.Name,
		APIKey:              string(keyBytes),
		ModelName:           llmModel.ModelName,
		BaseURL:             provider.BaseURL,
		Params:              params,
		Concurrency:         llmModel.MaxConcurrency,
		Timeout:             time.Duration(llmModel.TimeoutSeconds) * time.Second,
		ContextWindowTokens: llmModel.ContextWindowTokens,
		MaxOutputTokens:     llmModel.MaxTokens,
	}, nil
}

// GetModelDetails looks up model config by ID for runtime model selection.
// Returns provider_type, provider_name, api_key (decrypted), model_name, and base_url.
func (s *LlmConfigService) GetModelDetails(ctx context.Context, modelID int64) (providerType, providerName, apiKey, modelName, baseURL string, err error) {
	runtimeCfg, err := s.GetModelRuntimeConfig(ctx, modelID)
	if err != nil {
		return "", "", "", "", "", err
	}
	return runtimeCfg.ProviderType, runtimeCfg.ProviderName, runtimeCfg.APIKey, runtimeCfg.ModelName, runtimeCfg.BaseURL, nil
}

// GetDefaultModelRuntimeConfig returns the enabled default model with provider
// details and model generation parameters.
func (s *LlmConfigService) GetDefaultModelRuntimeConfig(ctx context.Context) (*LlmRuntimeModelConfig, error) {
	llmModel, err := s.modelRepo.GetDefaultModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("default model not found: %w", err)
	}
	return s.GetModelRuntimeConfig(ctx, llmModel.ID)
}

// GetDefaultModelDetails returns the enabled default model plus provider details.
func (s *LlmConfigService) GetDefaultModelDetails(ctx context.Context) (modelID int64, providerType, providerName, apiKey, modelName, baseURL string, err error) {
	runtimeCfg, err := s.GetDefaultModelRuntimeConfig(ctx)
	if err != nil {
		return 0, "", "", "", "", "", fmt.Errorf("default model not found: %w", err)
	}
	return runtimeCfg.ModelID, runtimeCfg.ProviderType, runtimeCfg.ProviderName, runtimeCfg.APIKey, runtimeCfg.ModelName, runtimeCfg.BaseURL, nil
}

// GetDefaultModelConfig retrieves the default model config (Provider + Model) for AI client initialization.
// Returns provider name, base_url, api_key, model name, and model params.
func GetDefaultModelConfig(ctx context.Context, providerRepo *repository.ProviderRepo, modelRepo *repository.ModelConfigRepo, encKey crypto.EncryptionKey, cfg config.Config) (baseURL, apiKey, model, providerType string, params ai.ModelParams, err error) {
	// Try to find default model from DB
	llmModel, dbErr := modelRepo.GetDefaultModel(ctx)
	if dbErr == nil && llmModel != nil {
		provider, pErr := providerRepo.GetByID(ctx, llmModel.ProviderID)
		if pErr == nil && provider != nil && provider.IsEnabled == 1 {
			keyBytes, dErr := crypto.Decrypt(encKey, provider.APIKeyEncrypted)
			if dErr == nil {
				return provider.BaseURL, string(keyBytes), llmModel.ModelName, provider.ProviderType, modelParamsFromConfig(llmModel), nil
			}
		}
	}

	// Fall back to environment variable configuration
	if cfg.AI.APIKey != "" {
		return cfg.AI.BaseURL, cfg.AI.APIKey, cfg.AI.Model, "", ai.ModelParams{}, nil
	}

	return "", "", "", "", ai.ModelParams{}, fmt.Errorf("no default model config found and no AI env config")
}

func modelParamsFromConfig(llmModel *model.LlmModel) ai.ModelParams {
	params := ai.ModelParams{}
	temperature := llmModel.Temperature
	params.Temperature = &temperature
	topP := llmModel.TopP
	params.TopP = &topP
	if llmModel.MaxTokens > 0 {
		maxTokens := int(llmModel.MaxTokens)
		params.MaxTokens = &maxTokens
	}
	return params
}
