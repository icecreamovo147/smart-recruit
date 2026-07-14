package persistence

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	commonsai "smart-recruit-commons/ai"
	"smart-recruit-proto/recruitment/pb"
)

type RuntimeLLMConfig struct {
	APIKey                  string
	Model                   string
	BaseURL                 string
	ProviderType            string
	Timeout                 time.Duration
	TotalTimeout            time.Duration
	ToolMaxRounds           int
	ToolTotalTimeout        time.Duration
	MaxConcurrency          int
	CircuitFailureThreshold int
	CircuitOpenTimeout      time.Duration
	HalfOpenMaxRequests     int
	RetryMaxAttempts        int
	RetryBaseDelay          time.Duration
	SlowResponseThreshold   time.Duration
}

type selectedLLMConfig struct {
	ProviderID          int64
	ModelID             int64
	APIKey              string
	Model               string
	BaseURL             string
	ProviderType        string
	Temperature         float64
	TopP                float64
	MaxTokens           int
	MaxConcurrency      int
	TimeoutSeconds      int
	ContextWindowTokens int
}

type llmRuntimeRow struct {
	ProviderID          int64   `gorm:"column:provider_id"`
	ModelID             int64   `gorm:"column:model_id"`
	APIKey              string  `gorm:"column:api_key"`
	Model               string  `gorm:"column:model"`
	BaseURL             string  `gorm:"column:base_url"`
	ProviderType        string  `gorm:"column:provider_type"`
	Temperature         float64 `gorm:"column:temperature"`
	TopP                float64 `gorm:"column:top_p"`
	MaxTokens           int     `gorm:"column:max_tokens"`
	MaxConcurrency      int     `gorm:"column:max_concurrency"`
	TimeoutSeconds      int     `gorm:"column:timeout_seconds"`
	ContextWindowTokens int     `gorm:"column:context_window_tokens"`
}

func (s *NativeStore) Complete(ctx context.Context, prompt string) (string, error) {
	return s.CompleteWithModel(ctx, prompt, 0)
}

func (s *NativeStore) CompleteWithModel(ctx context.Context, prompt string, modelID int64) (string, error) {
	cfg, err := s.selectLLMRuntimeConfig(ctx, modelID, 0)
	if err != nil {
		return "", err
	}
	client, err := s.newRuntimeClient(ctx, cfg)
	if err != nil {
		return "", err
	}
	return client.GenerateRecruitingReply(ctx, prompt, commonsai.RecruitingStats{}, nil)
}

func (s *NativeStore) validateLlmProviderConnection(ctx context.Context, provider llmProviderRecord) (*pb.TestProviderConnectionResponse, error) {
	cfg, err := s.selectLLMRuntimeConfig(ctx, 0, provider.ID)
	if err != nil {
		return &pb.TestProviderConnectionResponse{Code: configUnavailable, Msg: "provider runtime validation failed", Success: false, Detail: err.Error()}, nil
	}
	client, err := s.newRuntimeClient(ctx, cfg)
	if err != nil {
		return &pb.TestProviderConnectionResponse{Code: configUnavailable, Msg: "provider runtime validation failed", Success: false, Detail: err.Error()}, nil
	}
	testCtx := ctx
	if cfg.TimeoutSeconds <= 0 && s.runtimeLLM.Timeout <= 0 {
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
	}
	reply, err := client.GenerateRecruitingReply(testCtx, "请回复“连接正常”用于 Provider 连通性测试。", commonsai.RecruitingStats{}, nil)
	if err != nil {
		return &pb.TestProviderConnectionResponse{Code: configUnavailable, Msg: "provider connection failed", Success: false, Detail: err.Error()}, nil
	}
	if strings.TrimSpace(reply) == "" {
		return &pb.TestProviderConnectionResponse{Code: configUnavailable, Msg: "provider returned empty response", Success: false, Detail: "empty response from provider"}, nil
	}
	return &pb.TestProviderConnectionResponse{Code: configOK, Msg: "success", Success: true, Detail: fmt.Sprintf("validated with model %s", cfg.Model)}, nil
}

func (s *NativeStore) newRuntimeClient(ctx context.Context, cfg selectedLLMConfig) (*commonsai.Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("llm api_key is empty")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, fmt.Errorf("llm model is empty")
	}
	modelParams := commonsai.ModelParams{}
	if cfg.Temperature > 0 {
		modelParams.Temperature = &cfg.Temperature
	}
	if cfg.TopP > 0 {
		modelParams.TopP = &cfg.TopP
	}
	if cfg.MaxTokens > 0 {
		modelParams.MaxTokens = &cfg.MaxTokens
	}
	clientCfg := commonsai.ClientConfig{
		APIKey:                  cfg.APIKey,
		Model:                   cfg.Model,
		BaseURL:                 cfg.BaseURL,
		ProviderType:            cfg.ProviderType,
		ModelParams:             modelParams,
		Timeout:                 s.runtimeLLM.Timeout,
		TotalTimeout:            s.runtimeLLM.TotalTimeout,
		ToolMaxRounds:           s.runtimeLLM.ToolMaxRounds,
		ToolTotalTimeout:        s.runtimeLLM.ToolTotalTimeout,
		MaxConcurrency:          s.runtimeLLM.MaxConcurrency,
		CircuitFailureThreshold: s.runtimeLLM.CircuitFailureThreshold,
		CircuitOpenTimeout:      s.runtimeLLM.CircuitOpenTimeout,
		HalfOpenMaxRequests:     s.runtimeLLM.HalfOpenMaxRequests,
		RetryMaxAttempts:        s.runtimeLLM.RetryMaxAttempts,
		RetryBaseDelay:          s.runtimeLLM.RetryBaseDelay,
		SlowResponseThreshold:   s.runtimeLLM.SlowResponseThreshold,
	}
	if cfg.TimeoutSeconds > 0 {
		clientCfg.Timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	if cfg.MaxConcurrency > 0 {
		clientCfg.MaxConcurrency = cfg.MaxConcurrency
	}
	return commonsai.NewClientFromConfig(ctx, clientCfg)
}

func (s *NativeStore) selectLLMRuntimeConfig(ctx context.Context, modelID, providerID int64) (selectedLLMConfig, error) {
	var row llmRuntimeRow
	query := s.db.WithContext(ctx).Table("llm_models m").
		Select(`p.id AS provider_id,
			m.id AS model_id,
			p.api_key_encrypted AS api_key,
			m.model_name AS model,
			p.base_url AS base_url,
			p.provider_type AS provider_type,
			m.temperature AS temperature,
			m.top_p AS top_p,
			m.max_tokens AS max_tokens,
			m.max_concurrency AS max_concurrency,
			m.timeout_seconds AS timeout_seconds,
			m.context_window_tokens AS context_window_tokens`).
		Joins("JOIN llm_providers p ON p.id = m.provider_id").
		Where("m.is_enabled = ? AND p.is_enabled = ?", true, true)
	if modelID > 0 {
		query = query.Where("m.id = ?", modelID)
	}
	if providerID > 0 {
		query = query.Where("p.id = ?", providerID)
	}
	err := query.Order("m.is_default DESC, m.id ASC").Limit(1).Scan(&row).Error
	if err != nil {
		return selectedLLMConfig{}, err
	}
	if row.ModelID == 0 {
		if providerID > 0 {
			return s.defaultConfigForProvider(ctx, providerID)
		}
		if s.hasDefaultRuntimeConfig() {
			return s.defaultRuntimeSelection(), nil
		}
		return selectedLLMConfig{}, fmt.Errorf("no enabled llm model is configured")
	}
	apiKey, err := s.decryptAPIKey(row.APIKey)
	if err != nil {
		return selectedLLMConfig{}, fmt.Errorf("decrypt api key for provider %d: %w", row.ProviderID, err)
	}
	selected := selectedLLMConfig{
		ProviderID:          row.ProviderID,
		ModelID:             row.ModelID,
		APIKey:              apiKey,
		Model:               strings.TrimSpace(row.Model),
		BaseURL:             strings.TrimSpace(row.BaseURL),
		ProviderType:        strings.TrimSpace(row.ProviderType),
		Temperature:         row.Temperature,
		TopP:                row.TopP,
		MaxTokens:           row.MaxTokens,
		MaxConcurrency:      row.MaxConcurrency,
		TimeoutSeconds:      row.TimeoutSeconds,
		ContextWindowTokens: row.ContextWindowTokens,
	}
	selected.applyDefaults(s.runtimeLLM)
	return selected, nil
}

func (s *NativeStore) defaultConfigForProvider(ctx context.Context, providerID int64) (selectedLLMConfig, error) {
	var provider llmProviderRecord
	err := s.db.WithContext(ctx).Where("id = ? AND is_enabled = ?", providerID, true).First(&provider).Error
	if err == gorm.ErrRecordNotFound {
		return selectedLLMConfig{}, fmt.Errorf("provider not found or disabled")
	}
	if err != nil {
		return selectedLLMConfig{}, err
	}
	if strings.TrimSpace(s.runtimeLLM.Model) == "" {
		return selectedLLMConfig{}, fmt.Errorf("no enabled model exists for provider and AI_MODEL fallback is empty")
	}
	apiKey, err := s.decryptAPIKey(provider.APIKeyEncrypted)
	if err != nil {
		return selectedLLMConfig{}, fmt.Errorf("decrypt api key for provider %d: %w", provider.ID, err)
	}
	selected := selectedLLMConfig{
		ProviderID:   provider.ID,
		APIKey:       apiKey,
		Model:        strings.TrimSpace(s.runtimeLLM.Model),
		BaseURL:      strings.TrimSpace(provider.BaseURL),
		ProviderType: strings.TrimSpace(provider.ProviderType),
	}
	selected.applyDefaults(s.runtimeLLM)
	return selected, nil
}

func (s *NativeStore) hasDefaultRuntimeConfig() bool {
	return strings.TrimSpace(s.runtimeLLM.APIKey) != "" && strings.TrimSpace(s.runtimeLLM.Model) != ""
}

func (s *NativeStore) defaultRuntimeSelection() selectedLLMConfig {
	selected := selectedLLMConfig{
		APIKey:       strings.TrimSpace(s.runtimeLLM.APIKey),
		Model:        strings.TrimSpace(s.runtimeLLM.Model),
		BaseURL:      strings.TrimSpace(s.runtimeLLM.BaseURL),
		ProviderType: strings.TrimSpace(s.runtimeLLM.ProviderType),
	}
	selected.applyDefaults(s.runtimeLLM)
	return selected
}

func (c *selectedLLMConfig) applyDefaults(defaults RuntimeLLMConfig) {
	if strings.TrimSpace(c.APIKey) == "" {
		c.APIKey = strings.TrimSpace(defaults.APIKey)
	}
	if strings.TrimSpace(c.Model) == "" {
		c.Model = strings.TrimSpace(defaults.Model)
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		c.BaseURL = strings.TrimSpace(defaults.BaseURL)
	}
	if strings.TrimSpace(c.ProviderType) == "" {
		c.ProviderType = strings.TrimSpace(defaults.ProviderType)
	}
	if c.ProviderType == "" {
		c.ProviderType = "openai_compatible"
	}
}
