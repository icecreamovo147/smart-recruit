package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"

	"smart-recruit-domain-go/ai"
	"smart-recruit-domain-go/repository"
	"smart-recruit-platform-go/logger"
)

const (
	llmRequirementExtractorVersion = "job-requirement-llm-v1"
	llmRequirementPromptAgent      = "job_requirement_extractor"
	llmRequirementPromptRole       = "system"
	llmRequirementSaveReserve      = 7 * time.Second
	llmRequirementMinCallWindow    = 1 * time.Second
)

type LLMJobRequirementExtractor struct {
	LlmConfigSvc *LlmConfigService
	promptRepo   *repository.PromptTemplateRepo
}

func NewLLMJobRequirementExtractor(llmConfigSvc *LlmConfigService, promptRepo *repository.PromptTemplateRepo) *LLMJobRequirementExtractor {
	return &LLMJobRequirementExtractor{
		LlmConfigSvc: llmConfigSvc,
		promptRepo:   promptRepo,
	}
}

func (e *LLMJobRequirementExtractor) Extract(ctx context.Context, jobTitle, department, description, requirements string) (*JobRequirementProfile, string, error) {
	result, err := e.ExtractWithMetadata(ctx, jobTitle, department, description, requirements)
	if err != nil {
		return nil, "", err
	}
	return result.Profile, result.InputHash, nil
}

func (e *LLMJobRequirementExtractor) ExtractWithMetadata(ctx context.Context, jobTitle, department, description, requirements string) (*JobRequirementExtractResult, error) {
	start := time.Now()

	if e == nil || e.LlmConfigSvc == nil {
		return nil, fmt.Errorf("llm requirement extractor: llm config service is nil")
	}

	cfg, err := e.LlmConfigSvc.GetDefaultModelRuntimeConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("llm requirement extractor: get default model config: %w", err)
	}
	if cfg.APIKey == "" || cfg.ModelName == "" {
		return nil, fmt.Errorf("llm requirement extractor: default model not fully configured")
	}

	systemPrompt, promptKey, promptVersion, err := e.loadPrompt(ctx)
	if err != nil {
		return nil, err
	}

	fullText := strings.Join([]string{jobTitle, department, description, requirements}, "\n")
	userMsg := fmt.Sprintf("岗位信息：\n岗位名称：%s\n所属部门：%s\n岗位描述：\n%s\n岗位要求：\n%s", jobTitle, department, description, requirements)

	msgs := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userMsg),
	}

	timeout := e.resolveTimeout(ctx, cfg)
	logger.L().Info("llm requirement extractor request started",
		zap.String("provider_type", cfg.ProviderType),
		zap.String("model_name", cfg.ModelName),
		zap.Duration("timeout", timeout),
		zap.Int("input_chars", len(fullText)),
		zap.String("prompt_key", promptKey),
		zap.Int32("prompt_version", promptVersion),
	)

	cm, err := ai.NewChatModelWithParams(ctx, cfg.ProviderType, cfg.APIKey, cfg.ModelName, cfg.BaseURL, timeout, cfg.Params)
	if err != nil {
		return nil, fmt.Errorf("llm requirement extractor: create chat model: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := cm.Generate(callCtx, msgs)
	if err != nil {
		logger.L().Warn("llm requirement extractor request failed",
			zap.String("model_name", cfg.ModelName),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return nil, fmt.Errorf("llm requirement extractor: generate: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return nil, fmt.Errorf("llm requirement extractor: empty response")
	}

	cleaned := cleanLLMJSONOutput(content)
	if cleaned == "" {
		return nil, fmt.Errorf("llm requirement extractor: no valid JSON found in response")
	}

	var profile JobRequirementProfile
	if err := json.Unmarshal([]byte(cleaned), &profile); err != nil {
		return nil, fmt.Errorf("llm requirement extractor: invalid JSON: %w", err)
	}

	if err := profile.Validate(); err != nil {
		return nil, fmt.Errorf("llm requirement extractor: schema validation failed: %w", err)
	}

	duration := time.Since(start)
	inputHash := computeInputHash(fullText, llmRequirementExtractorVersion)

	logger.L().Info("llm requirement extractor succeeded",
		zap.String("parser_version", llmRequirementExtractorVersion),
		zap.String("model_name", cfg.ModelName),
		zap.Duration("duration", duration),
		zap.Int("requirement_count", len(profile.Requirements)),
	)

	return &JobRequirementExtractResult{
		Profile:   &profile,
		InputHash: inputHash,
		Metadata: ExtractorMetadata{
			ParserVersion: llmRequirementExtractorVersion,
			ExtractorType: "llm",
			ModelName:     cfg.ModelName,
			PromptKey:     promptKey,
			PromptVersion: promptVersion,
			FallbackUsed:  false,
			Duration:      duration,
		},
	}, nil
}

func (e *LLMJobRequirementExtractor) loadPrompt(ctx context.Context) (prompt string, key string, version int32, err error) {
	if e.promptRepo == nil {
		return "", "", 0, fmt.Errorf("llm requirement extractor: DB prompt repo is not configured for agent_type=%s role=%s", llmRequirementPromptAgent, llmRequirementPromptRole)
	}
	tmpl, err := e.promptRepo.GetActiveByAgentType(ctx, llmRequirementPromptAgent, llmRequirementPromptRole)
	if err != nil {
		return "", "", 0, fmt.Errorf("llm requirement extractor: load DB prompt template for agent_type=%s role=%s: %w", llmRequirementPromptAgent, llmRequirementPromptRole, err)
	}
	if strings.TrimSpace(tmpl.Content) == "" {
		return "", "", 0, fmt.Errorf("llm requirement extractor: active DB prompt template is empty for agent_type=%s role=%s", llmRequirementPromptAgent, llmRequirementPromptRole)
	}
	return tmpl.Content, tmpl.Name, tmpl.Version, nil
}

func (e *LLMJobRequirementExtractor) resolveTimeout(ctx context.Context, cfg *LlmRuntimeModelConfig) time.Duration {
	timeout := 60 * time.Second
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return timeout
	}
	remaining := time.Until(deadline)
	if remaining <= llmRequirementSaveReserve {
		return llmRequirementMinCallWindow
	}
	available := remaining - llmRequirementSaveReserve
	if available < llmRequirementMinCallWindow {
		return llmRequirementMinCallWindow
	}
	if available < timeout {
		return available
	}
	return timeout
}
