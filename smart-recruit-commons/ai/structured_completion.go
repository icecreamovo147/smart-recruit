package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"

	"smart-recruit-platform-go/logger"
)

// StructuredCompletionResult contains the model output and safe provenance for
// callers that require an exact System/User message boundary.
type StructuredCompletionResult struct {
	Content    string
	ModelName  string
	TokenUsage *schema.TokenUsage
}

// GenerateStructured completes an internal structured-generation request using
// exactly one System message and one User message. It intentionally bypasses
// the generic recruiting/Markdown prompt builder while retaining Client call
// controls such as timeout, retry, concurrency limiting, and circuit breaking.
func (c *Client) GenerateStructured(ctx context.Context, systemPrompt, userPrompt string) (StructuredCompletionResult, error) {
	if c == nil || c.cm == nil {
		return StructuredCompletionResult{}, fmt.Errorf("ai chat model is nil")
	}

	start := time.Now()
	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}
	var response *schema.Message
	err := c.callPrivacySafe(ctx, func(callCtx context.Context) error {
		var generateErr error
		response, generateErr = c.cm.Generate(callCtx, messages)
		return generateErr
	})
	if err != nil {
		classified := ClassifyAIError(err)
		logger.L().Warn("ai structured completion failed",
			zap.String("model", c.model),
			zap.Int("system_chars", len([]rune(systemPrompt))),
			zap.Int("user_chars", len([]rune(userPrompt))),
			zap.Duration("cost", time.Since(start)),
			zap.String("error_type", string(classified.Type)),
		)
		return StructuredCompletionResult{}, classified
	}
	if response == nil || strings.TrimSpace(response.Content) == "" {
		return StructuredCompletionResult{}, NewAIError(AIEmptyReply, "", fmt.Errorf("ai returned empty reply"))
	}

	usage := tokenUsageFromMessage(response)
	logger.L().Info("ai structured completion done",
		append(TokenUsageLogFields(usage),
			zap.String("model", c.model),
			zap.Int("system_chars", len([]rune(systemPrompt))),
			zap.Int("user_chars", len([]rune(userPrompt))),
			zap.Int("reply_chars", len([]rune(response.Content))),
			zap.Duration("cost", time.Since(start)),
		)...,
	)
	return StructuredCompletionResult{Content: response.Content, ModelName: c.model, TokenUsage: usage}, nil
}
