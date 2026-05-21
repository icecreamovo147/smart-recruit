package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"

	"logic-grpc-service/ai"
	"logic-grpc-service/config"
	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

const (
	defaultRecentMessageLimit     = 20
	defaultSummaryTriggerMessages = 30
	defaultMaxMemoryChars         = 1500
	defaultMaxPromptChars         = 20000
	defaultMaxMemories            = 10
)

// AgentContextInput is the input for building an agent context.
type AgentContextInput struct {
	HrID           int64
	SessionID      int64
	ApplicationID  int64
	JobID          int64
	CurrentMessage string
}

// AgentContext holds the assembled context for a single AI request.
type AgentContext struct {
	HrID               int64
	SessionID          int64
	ApplicationID      int64
	JobID              int64
	SessionSummary     string
	RecentMessages     []model.AIChatHistory
	LongTermMemories   []model.AIMemory
	PromptCharEstimate int
	MemoryCount        int
	MemoryCharCount    int
	SummaryCharCount   int
	MessageCount       int
}

// AgentContextBuilder assembles the prompt context from multiple memory layers.
type AgentContextBuilder struct {
	chats     *repository.ChatRepo
	summaries *repository.SessionSummaryRepo
	memories  *repository.MemoryRepo
	ai        *ai.Client
	cfg       config.Config
}

// NewAgentContextBuilder creates a new AgentContextBuilder.
func NewAgentContextBuilder(
	chats *repository.ChatRepo,
	summaries *repository.SessionSummaryRepo,
	memories *repository.MemoryRepo,
	aiClient *ai.Client,
	cfg config.Config,
) *AgentContextBuilder {
	return &AgentContextBuilder{
		chats:     chats,
		summaries: summaries,
		memories:  memories,
		ai:        aiClient,
		cfg:       cfg,
	}
}

func (b *AgentContextBuilder) recentLimit() int {
	if b.cfg.Agent.RecentMessageLimit > 0 {
		return b.cfg.Agent.RecentMessageLimit
	}
	return defaultRecentMessageLimit
}

func (b *AgentContextBuilder) summaryTrigger() int {
	if b.cfg.Agent.SummaryTriggerMessages > 0 {
		return b.cfg.Agent.SummaryTriggerMessages
	}
	return defaultSummaryTriggerMessages
}

func (b *AgentContextBuilder) maxMemoryChars() int {
	if b.cfg.Agent.MaxMemoryChars > 0 {
		return b.cfg.Agent.MaxMemoryChars
	}
	return defaultMaxMemoryChars
}

func (b *AgentContextBuilder) maxPromptChars() int {
	if b.cfg.Agent.MaxPromptChars > 0 {
		return b.cfg.Agent.MaxPromptChars
	}
	return defaultMaxPromptChars
}

func (b *AgentContextBuilder) maxMemories() int {
	if b.cfg.Agent.MaxMemories > 0 {
		return b.cfg.Agent.MaxMemories
	}
	return defaultMaxMemories
}

// Build assembles the full agent context by reading recent messages, session summary,
// and long-term memories.
func (b *AgentContextBuilder) Build(ctx context.Context, input AgentContextInput) (*AgentContext, error) {
	actx := &AgentContext{
		HrID:          input.HrID,
		SessionID:     input.SessionID,
		ApplicationID: input.ApplicationID,
		JobID:         input.JobID,
	}

	// Phase 1: Recent messages (last N, chronological order).
	recent, err := b.chats.ListRecentBySession(ctx, input.HrID, input.SessionID, b.recentLimit())
	if err != nil {
		return nil, err
	}
	actx.RecentMessages = recent
	actx.MessageCount = len(recent)

	// Phase 2: Session summary.
	summary, err := b.summaries.GetBySession(ctx, input.HrID, input.SessionID)
	if err != nil {
		return nil, err
	}
	if summary != nil {
		actx.SessionSummary = summary.Summary
		actx.SummaryCharCount = utf8.RuneCountInString(summary.Summary)
	}

	// Phase 4: Long-term memories.
	memories := b.retrieveMemories(ctx, input)
	actx.LongTermMemories = memories
	actx.MemoryCount = len(memories)
	for _, m := range memories {
		actx.MemoryCharCount += utf8.RuneCountInString(m.Content)
	}

	// Phase 5: Estimate prompt chars and trim if needed.
	actx.PromptCharEstimate = b.estimatePromptChars(actx, input.CurrentMessage)

	// Log context budget.
	logger.L().Info("[上下文预算]",
		zap.Int64("session_id", input.SessionID),
		zap.Int("recent_messages", actx.MessageCount),
		zap.Int("summary_chars", actx.SummaryCharCount),
		zap.Int("memory_count", actx.MemoryCount),
		zap.Int("memory_chars", actx.MemoryCharCount),
		zap.Int("prompt_estimate_chars", actx.PromptCharEstimate),
		zap.Int("max_prompt_chars", b.maxPromptChars()),
	)

	return actx, nil
}

// retrieveMemories fetches long-term memories relevant to the current context.
func (b *AgentContextBuilder) retrieveMemories(ctx context.Context, input AgentContextInput) []model.AIMemory {
	maxChars := b.maxMemoryChars()
	maxCount := b.maxMemories()
	var all []model.AIMemory

	// HR-level preferences (scope_type = hr, scope_id = 0).
	hrMemories, _ := b.memories.ListByHR(ctx, input.HrID, 5)
	all = append(all, hrMemories...)

	// Job-level memories.
	if input.JobID > 0 {
		jobMemories, _ := b.memories.ListRelevant(ctx, input.HrID, "job", input.JobID, nil, 3)
		all = append(all, jobMemories...)
	}

	// Application-level memories.
	if input.ApplicationID > 0 {
		appMemories, _ := b.memories.ListRelevant(ctx, input.HrID, "application", input.ApplicationID, nil, 3)
		all = append(all, appMemories...)
	}

	// Trim to budget: max count and max total chars.
	result := make([]model.AIMemory, 0, maxCount)
	charCount := 0
	for _, m := range all {
		if len(result) >= maxCount {
			break
		}
		mChars := utf8.RuneCountInString(m.Content)
		if charCount+mChars > maxChars {
			break
		}
		charCount += mChars
		result = append(result, m)
	}
	return result
}

// estimatePromptChars provides a rough estimate of the total prompt characters.
func (b *AgentContextBuilder) estimatePromptChars(actx *AgentContext, currentMsg string) int {
	n := 2000 // Base system prompt overhead (rules, identity, tool descriptions).
	n += utf8.RuneCountInString(actx.SessionSummary)
	n += utf8.RuneCountInString(currentMsg)
	for _, m := range actx.RecentMessages {
		n += utf8.RuneCountInString(m.Content)
	}
	for _, m := range actx.LongTermMemories {
		n += utf8.RuneCountInString(m.Content)
	}
	return n
}

// ShouldRefreshSummary checks if the session summary needs refreshing.
func (b *AgentContextBuilder) ShouldRefreshSummary(ctx context.Context, hrID, sessionID int64) (bool, error) {
	total, err := b.chats.CountBySession(ctx, hrID, sessionID)
	if err != nil {
		return false, err
	}
	if total < int64(b.summaryTrigger()) {
		return false, nil
	}
	summary, err := b.summaries.GetBySession(ctx, hrID, sessionID)
	if err != nil {
		return false, err
	}
	if summary == nil {
		return true, nil
	}
	// Refresh if summary lags behind by more than 20 messages.
	lag := total - int64(summary.MessageCount)
	return lag > 20, nil
}

// RefreshSessionSummary generates or updates the session summary asynchronously.
func (b *AgentContextBuilder) RefreshSessionSummary(sessionID, hrID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log := logger.With(zap.Int64("session_id", sessionID), zap.Int64("hr_id", hrID))

	// Get existing summary.
	oldSummary := ""
	existing, err := b.summaries.GetBySession(ctx, hrID, sessionID)
	if err == nil && existing != nil {
		oldSummary = existing.Summary
	}

	// Get recent messages to summarize (last 40 to give enough context).
	recent, err := b.chats.ListRecentBySession(ctx, hrID, sessionID, 40)
	if err != nil {
		log.Error("summary: failed to load recent messages", zap.Error(err))
		return
	}

	msgTexts := make([]string, 0, len(recent))
	var maxMsgID int64
	for _, m := range recent {
		prefix := "用户"
		if m.Role == "assistant" {
			prefix = "助手"
		}
		msgTexts = append(msgTexts, fmt.Sprintf("%s: %s", prefix, truncateForSummary(m.Content, 300)))
		if m.ID > maxMsgID {
			maxMsgID = m.ID
		}
	}

	newSummary, err := b.ai.GenerateSessionSummary(ctx, oldSummary, msgTexts)
	if err != nil {
		log.Error("summary: LLM generation failed", zap.Error(err))
		return
	}
	if strings.TrimSpace(newSummary) == "" {
		log.Warn("summary: LLM returned empty summary, skipping upsert")
		return
	}

	if err := b.summaries.Upsert(ctx, &model.AISessionSummary{
		SessionID:        uint64(sessionID),
		HrID:             uint64(hrID),
		Summary:          newSummary,
		CoveredMessageID: uint64(maxMsgID),
		MessageCount:     len(recent),
	}); err != nil {
		log.Error("summary: upsert failed", zap.Error(err))
		return
	}

	log.Info("summary refreshed",
		zap.Int("summary_chars", utf8.RuneCountInString(newSummary)),
		zap.Int64("covered_message_id", maxMsgID),
	)
}

// truncateForSummary truncates content for summary generation to avoid overly large prompts.
func truncateForSummary(content string, maxChars int) string {
	runes := []rune(content)
	if len(runes) <= maxChars {
		return content
	}
	return string(runes[:maxChars]) + "..."
}
