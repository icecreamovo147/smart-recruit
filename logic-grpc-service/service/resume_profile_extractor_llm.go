package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	llmParserVersion          = "resume-profile-llm-v1"
	llmExtractorPromptAgent   = "resume_profile_extractor"
	llmExtractorPromptRole    = "system"
	llmExtractorSaveReserve   = 7 * time.Second
	llmExtractorMinCallWindow = 1 * time.Second
	llmExtractorDefaultPrompt = `You are a resume profile extractor. Extract structured data from the resume text below.

You MUST output ONLY a JSON object — no explanation, no markdown, no code fences, no extra text.
The JSON object MUST conform to this exact schema:
{
  "full_name": "string",
  "email": "string",
  "phone": "string",
  "location": "string",
  "headline": "string",
  "summary": "string",
  "total_experience_years": number,
  "highest_degree": "string",
  "educations": [
    {
      "school": "string",
      "degree": "string",
      "major": "string",
      "start_date": "string (YYYY, YYYY-MM, or YYYY-MM-DD)",
      "end_date": "string (YYYY, YYYY-MM, YYYY-MM-DD, or \"present\")",
      "description": "string"
    }
  ],
  "experiences": [
    {
      "company": "string",
      "title": "string",
      "location": "string",
      "start_date": "string",
      "end_date": "string",
      "is_current": bool,
      "description": "string",
      "achievements": ["string"]
    }
  ],
  "projects": [
    {
      "name": "string",
      "role": "string",
      "start_date": "string",
      "end_date": "string",
      "description": "string",
      "technologies": ["string"],
      "highlights": ["string"]
    }
  ],
  "skills": [
    {
      "name": "string",
      "category": "string",
      "level": "string",
      "years": number,
      "evidence": "string"
    }
  ]
}

Rules:
- full_name, email, phone, location, headline, summary can be empty strings if not found
- total_experience_years MUST be a number (0 if unknown)
- educations, experiences, projects, skills MUST be arrays (can be empty)
- Use "present" for end_date if the person currently works/studies there
- is_current must be true when end_date is "present" or "current"
- Extract skills with any evidence mentioned (project names, companies where used)
- Output ONLY the JSON, no other text.`
)

type LLMResumeProfileExtractor struct {
	llmConfigSvc *LlmConfigService
	promptRepo   *repository.PromptTemplateRepo
}

func NewLLMResumeProfileExtractor(llmConfigSvc *LlmConfigService, promptRepo *repository.PromptTemplateRepo) *LLMResumeProfileExtractor {
	return &LLMResumeProfileExtractor{
		llmConfigSvc: llmConfigSvc,
		promptRepo:   promptRepo,
	}
}

func (e *LLMResumeProfileExtractor) Extract(ctx context.Context, text string) (string, error) {
	result, err := e.ExtractWithMetadata(ctx, text)
	if err != nil {
		return "", err
	}
	return result.RawJSON, nil
}

func (e *LLMResumeProfileExtractor) ExtractWithMetadata(ctx context.Context, text string) (ResumeProfileExtractResult, error) {
	start := time.Now()

	if e == nil || e.llmConfigSvc == nil {
		return ResumeProfileExtractResult{}, fmt.Errorf("llm extractor: llm config service is nil (ENCRYPTION_KEY may not be set)")
	}

	cfg, err := e.llmConfigSvc.GetDefaultModelRuntimeConfig(ctx)
	if err != nil {
		return ResumeProfileExtractResult{}, fmt.Errorf("llm extractor: get default model config: %w", err)
	}
	if cfg.APIKey == "" || cfg.ModelName == "" {
		return ResumeProfileExtractResult{}, fmt.Errorf("llm extractor: default model not fully configured (api_key or model_name empty)")
	}

	systemPrompt, promptKey, promptVersion := e.loadPrompt(ctx)

	msgs := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(fmt.Sprintf("Resume text:\n%s\n\nExtract the structured profile as JSON.", text)),
	}

	timeout := e.resolveTimeout(ctx, cfg)
	logger.L().Info("llm resume profile extractor request started",
		zap.Int64("model_id", cfg.ModelID),
		zap.String("provider_type", cfg.ProviderType),
		zap.String("provider_name", cfg.ProviderName),
		zap.String("model_name", cfg.ModelName),
		zap.String("base_url", cfg.BaseURL),
		zap.Duration("timeout", timeout),
		zap.Int("input_chars", len(text)),
		zap.Int("system_prompt_chars", len(systemPrompt)),
		zap.String("prompt_key", promptKey),
		zap.Int32("prompt_version", promptVersion),
		zap.Int32("max_output_tokens", cfg.MaxOutputTokens),
	)
	cm, err := ai.NewChatModelWithParams(ctx, cfg.ProviderType, cfg.APIKey, cfg.ModelName, cfg.BaseURL, timeout, cfg.Params)
	if err != nil {
		return ResumeProfileExtractResult{}, fmt.Errorf("llm extractor: create chat model: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := cm.Generate(callCtx, msgs)
	if err != nil {
		logger.L().Warn("llm resume profile extractor request failed",
			zap.Int64("model_id", cfg.ModelID),
			zap.String("provider_type", cfg.ProviderType),
			zap.String("provider_name", cfg.ProviderName),
			zap.String("model_name", cfg.ModelName),
			zap.Duration("timeout", timeout),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
			zap.Error(callCtx.Err()),
		)
		return ResumeProfileExtractResult{}, fmt.Errorf("llm extractor: generate: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return ResumeProfileExtractResult{}, fmt.Errorf("llm extractor: empty response")
	}

	cleaned := cleanLLMJSONOutput(content)
	if cleaned == "" {
		return ResumeProfileExtractResult{}, fmt.Errorf("llm extractor: no valid JSON found in response: %s", truncateForLog(content, 200))
	}

	if err := quickValidateResumeProfileJSON(cleaned); err != nil {
		return ResumeProfileExtractResult{}, fmt.Errorf("llm extractor: invalid schema: %w", err)
	}
	logger.L().Debug("llm resume profile extractor cleaned output",
		zap.Int("output_chars", len(cleaned)),
		zap.String("output_preview", truncateForLog(cleaned, 1200)),
	)

	duration := time.Since(start)
	meta := ExtractorMetadata{
		ParserVersion: llmParserVersion,
		ExtractorType: "llm",
		ModelName:     cfg.ModelName,
		PromptKey:     promptKey,
		PromptVersion: promptVersion,
		FallbackUsed:  false,
		Duration:      duration,
	}

	logger.L().Info("llm resume profile extractor succeeded",
		zap.String("parser_version", meta.ParserVersion),
		zap.String("model_name", cfg.ModelName),
		zap.String("prompt_key", promptKey),
		zap.Int32("prompt_version", promptVersion),
		zap.Duration("duration", duration),
		zap.Int("input_chars", len(text)),
		zap.Int("output_chars", len(cleaned)),
	)

	return ResumeProfileExtractResult{RawJSON: cleaned, Metadata: meta}, nil
}

func (e *LLMResumeProfileExtractor) loadPrompt(ctx context.Context) (prompt string, key string, version int32) {
	if e.promptRepo == nil {
		logger.L().Info("llm extractor: prompt repo is nil, using built-in fallback",
			zap.String("agent_type", llmExtractorPromptAgent),
		)
		return llmExtractorDefaultPrompt, "builtin", 0
	}
	tmpl, err := e.promptRepo.GetActiveByAgentType(ctx, llmExtractorPromptAgent, llmExtractorPromptRole)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.L().Info("llm extractor: no DB prompt template found, using built-in fallback",
				zap.String("agent_type", llmExtractorPromptAgent),
			)
			return llmExtractorDefaultPrompt, "builtin", 0
		}
		logger.L().Warn("llm extractor: failed to load prompt template from DB, using built-in",
			zap.String("agent_type", llmExtractorPromptAgent),
			zap.Error(err),
		)
		return llmExtractorDefaultPrompt, "builtin", 0
	}
	return tmpl.Content, tmpl.Name, tmpl.Version
}

func (e *LLMResumeProfileExtractor) resolveTimeout(ctx context.Context, cfg *LlmRuntimeModelConfig) time.Duration {
	timeout := 60 * time.Second
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return timeout
	}
	remaining := time.Until(deadline)
	if remaining <= llmExtractorSaveReserve {
		return llmExtractorMinCallWindow
	}
	available := remaining - llmExtractorSaveReserve
	if available < llmExtractorMinCallWindow {
		return llmExtractorMinCallWindow
	}
	if available < timeout {
		return available
	}
	return timeout
}

func cleanLLMJSONOutput(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	braceStart := strings.Index(content, "{")
	if braceStart < 0 {
		return ""
	}
	braceEnd := strings.LastIndex(content, "}")
	if braceEnd < 0 || braceEnd < braceStart {
		return ""
	}
	return content[braceStart : braceEnd+1]
}

func quickValidateResumeProfileJSON(raw string) error {
	var doc struct {
		FullName             string `json:"full_name"`
		TotalExperienceYears any    `json:"total_experience_years"`
		Educations           any    `json:"educations"`
		Experiences          any    `json:"experiences"`
		Projects             any    `json:"projects"`
		Skills               any    `json:"skills"`
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	if err := decoder.Decode(&doc); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("multiple JSON values")
	}
	if doc.TotalExperienceYears == nil {
		return fmt.Errorf("total_experience_years is required")
	}
	if doc.Educations == nil {
		return fmt.Errorf("educations is required")
	}
	if doc.Experiences == nil {
		return fmt.Errorf("experiences is required")
	}
	if doc.Projects == nil {
		return fmt.Errorf("projects is required")
	}
	if doc.Skills == nil {
		return fmt.Errorf("skills is required")
	}
	return nil
}

func truncateForLog(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
