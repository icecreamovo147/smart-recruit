package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"logic-grpc-service/ai"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

const (
	llmRequirementExtractorVersion = "job-requirement-llm-v1"
	llmRequirementPromptAgent      = "job_requirement_extractor"
	llmRequirementPromptRole       = "system"
	llmRequirementSaveReserve      = 7 * time.Second
	llmRequirementMinCallWindow    = 1 * time.Second
	llmRequirementDefaultPrompt    = `You are a job requirement extractor. Given a job description, extract structured requirements.

You MUST output ONLY a JSON object — no explanation, no markdown, no code fences, no extra text.
The JSON object MUST conform to this exact schema:
{
  "profile_version": "job-requirement-profile-v1",
  "requirements": [
    {
      "id": "unique-kebab-case-id",
      "category": "core_skill|experience|education|certification|domain|language|soft_skill|other",
      "label": "Chinese label for the requirement",
      "description": "Brief description in Chinese",
      "priority": "must_have|nice_to_have|soft_skill",
      "weight": 0.0,
      "knockout": false,
      "aliases": ["alias1", "alias2"]
    }
  ]
}

Rules:
- IDs must be kebab-case English (e.g., "java-backend", "distributed-systems")
- Labels must be Chinese (e.g., "Java 后端开发经验", "分布式系统能力")
- priority must_have = required/core, nice_to_have = bonus, soft_skill = soft capability
- knockout = true only for hard disqualifiers (e.g., specific degree, certification)
- weights must sum to 1.0, reflecting relative importance
- aliases are alternate names/synonyms including Chinese variants
- Extract MUST-HAVE requirements first, then nice-to-have, then soft skills
- Minimum 2 requirements, maximum 20
- Output ONLY the JSON, no other text.`
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

	systemPrompt, promptKey, promptVersion := e.loadPrompt(ctx)

	fullText := strings.Join([]string{jobTitle, department, description, requirements}, "\n")
	userMsg := fmt.Sprintf("Job Information:\nTitle: %s\nDepartment: %s\nDescription:\n%s\nRequirements:\n%s\n\nExtract structured requirements as JSON.", jobTitle, department, description, requirements)

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

func (e *LLMJobRequirementExtractor) loadPrompt(ctx context.Context) (prompt string, key string, version int32) {
	if e.promptRepo == nil {
		logger.L().Info("llm requirement extractor: prompt repo is nil, using built-in fallback",
			zap.String("agent_type", llmRequirementPromptAgent),
		)
		return llmRequirementDefaultPrompt, "builtin", 0
	}
	tmpl, err := e.promptRepo.GetActiveByAgentType(ctx, llmRequirementPromptAgent, llmRequirementPromptRole)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.L().Info("llm requirement extractor: no DB prompt template found, using built-in",
				zap.String("agent_type", llmRequirementPromptAgent),
			)
			return llmRequirementDefaultPrompt, "builtin", 0
		}
		logger.L().Warn("llm requirement extractor: failed to load prompt from DB, using built-in",
			zap.String("agent_type", llmRequirementPromptAgent),
			zap.Error(err),
		)
		return llmRequirementDefaultPrompt, "builtin", 0
	}
	return tmpl.Content, tmpl.Name, tmpl.Version
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
