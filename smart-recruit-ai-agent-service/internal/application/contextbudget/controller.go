package contextbudget

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"

	"smart-recruit-platform-go/logger"
)

type Controller struct {
	model           ModelInfo
	tools           []*schema.ToolInfo
	current         string
	history         []MessageRow
	ownerID         int64
	sessionID       int64
	memorySection   string
	summaryAudience string
	store           SessionSummaryStore
	generator       SummaryGenerator

	mu          sync.Mutex
	summary     string
	coveredID   int64
	summaryDone bool
	asyncOnce   sync.Once

	lastResult PrepareResult
}

type Config struct {
	Model           ModelInfo
	Tools           []*schema.ToolInfo
	Current         string
	History         []MessageRow
	OwnerID         int64
	SessionID       int64
	MemorySection   string
	SummaryAudience string
	Store           SessionSummaryStore
	Generator       SummaryGenerator
}

func New(ctx context.Context, cfg Config) *Controller {
	c := &Controller{
		model:           cfg.Model,
		tools:           cfg.Tools,
		current:         strings.TrimSpace(cfg.Current),
		history:         append([]MessageRow(nil), cfg.History...),
		ownerID:         cfg.OwnerID,
		sessionID:       cfg.SessionID,
		memorySection:   strings.TrimSpace(cfg.MemorySection),
		summaryAudience: strings.TrimSpace(cfg.SummaryAudience),
		store:           cfg.Store,
		generator:       cfg.Generator,
	}
	if c.summaryAudience == "" {
		c.summaryAudience = AudienceHR
	}
	if c.store != nil {
		loadCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if summary, covered, _, found, err := c.store.GetSessionSummaryState(loadCtx, cfg.OwnerID, cfg.SessionID); err == nil && found {
			c.summary = strings.TrimSpace(summary)
			c.coveredID = covered
		} else if err != nil {
			logger.L().Warn("context summary load failed", zap.Int64("session_id", cfg.SessionID), zap.String("error_category", "summary_load"))
		}
	}
	return c
}

func (c *Controller) LastPrepareResult() PrepareResult {
	if c == nil {
		return PrepareResult{}
	}
	return c.lastResult
}

func (c *Controller) Prepare(ctx context.Context, messages []*schema.Message, stage string) ([]*schema.Message, error) {
	if c == nil {
		return messages, nil
	}
	if c.model.MaxOutputTokens < 0 {
		return nil, &GuardError{Code: ConfigInvalidCode}
	}
	window := int64(c.model.ContextWindowTokens)
	budget := int64(InputBudgetTokens(c.model))
	if window > 0 && budget <= 0 {
		return nil, &GuardError{Code: ConfigInvalidCode}
	}

	base, history, currentIndex := PartitionMessages(messages, c.current)
	originalHistoryCount := len(history)
	allHistory := history
	if c.currentSummary() != "" {
		history = DropCoveredHistoryPrefix(allHistory, c.coveredHistoryCount())
	}
	if window <= 0 {
		if len(history) > UnknownContextHistory {
			history = history[len(history)-UnknownContextHistory:]
		}
		prepared := ComposeMessages(base, history, c.currentSummary(), c.memorySection, currentIndex)
		omitted := originalHistoryCount - len(history)
		c.lastResult = PrepareResult{
			SummaryApplied: c.currentSummary() != "",
			MemoryApplied:  c.memorySection != "",
			OmittedCount:   omitted,
		}
		_ = stage
		return prepared, nil
	}

	allPrepared := ComposeMessages(base, history, c.currentSummary(), c.memorySection, currentIndex)
	rawTokens := int64(EstimateMessagesEnvelopeTokens(allPrepared, c.tools))
	if rawTokens > budget {
		_ = c.refreshSummary(ctx, "sync_over_budget")
		if c.currentSummary() != "" {
			history = DropCoveredHistoryPrefix(allHistory, c.coveredHistoryCount())
		}
	}

	summary := c.currentSummary()
	prepared, omitted, summaryApplied, memoryApplied, fixedOverflow := SelectWithinBudget(
		c.tools, base, history, summary, c.memorySection, currentIndex, budget,
	)
	omitted += originalHistoryCount - len(history)
	c.lastResult = PrepareResult{
		SummaryApplied: summaryApplied,
		MemoryApplied:  memoryApplied,
		OmittedCount:   omitted,
	}
	if fixedOverflow {
		return nil, &GuardError{Code: BudgetExceededCode}
	}
	if rawTokens*100 >= budget*60 && rawTokens <= budget {
		c.triggerAsyncSummary()
	}
	_ = stage
	return prepared, nil
}

func InputBudgetTokens(model ModelInfo) int32 {
	window := int64(model.ContextWindowTokens)
	maxOutput := int64(model.MaxOutputTokens)
	if maxOutput < 0 || window <= 0 {
		return 0
	}
	safety := (window + 19) / 20
	if safety < 256 {
		safety = 256
	}
	if safety > 2048 {
		safety = 2048
	}
	inputBudget := window - maxOutput - safety
	if inputBudget <= 0 {
		return 0
	}
	if inputBudget > int64(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(inputBudget)
}

func DropCoveredHistoryPrefix(history []*schema.Message, count int) []*schema.Message {
	if count <= 0 {
		return history
	}
	if count >= len(history) {
		return nil
	}
	return history[count:]
}

func IsInjectedContextMessage(content string) bool {
	return strings.HasPrefix(content, SummaryMessagePrefix) || strings.HasPrefix(content, MemoryMessagePrefix)
}

func PartitionMessages(messages []*schema.Message, current string) (fixed []*schema.Message, history []*schema.Message, currentIndex int) {
	clean := make([]*schema.Message, 0, len(messages))
	for _, message := range messages {
		if message == nil || IsInjectedContextMessage(message.Content) {
			continue
		}
		clean = append(clean, message)
	}
	current = strings.TrimSpace(current)
	currentIndex = -1
	for i := len(clean) - 1; i >= 0; i-- {
		if clean[i].Role == schema.User && current != "" && strings.TrimSpace(clean[i].Content) == current {
			currentIndex = i
			break
		}
	}
	for i, message := range clean {
		if message.Role == schema.System || i == currentIndex || (currentIndex >= 0 && i > currentIndex) {
			fixed = append(fixed, message)
		} else {
			history = append(history, message)
		}
	}
	return fixed, history, currentIndex
}

func ComposeMessages(fixed, history []*schema.Message, summary, memorySection string, _ int) []*schema.Message {
	out := make([]*schema.Message, 0, len(fixed)+len(history)+2)
	insertedSummary := false
	insertedMemory := false
	for _, message := range fixed {
		if message.Role == schema.System {
			out = append(out, message)
			if !insertedSummary && summary != "" {
				out = append(out, schema.SystemMessage(SummaryMessagePrefix+summary))
				insertedSummary = true
			}
			if !insertedMemory && strings.TrimSpace(memorySection) != "" {
				out = append(out, schema.SystemMessage(MemoryMessagePrefix+memorySection))
				insertedMemory = true
			}
		}
	}
	out = append(out, history...)
	for _, message := range fixed {
		if message.Role != schema.System {
			out = append(out, message)
		}
	}
	if summary != "" && !insertedSummary {
		out = append([]*schema.Message{schema.SystemMessage(SummaryMessagePrefix + summary)}, out...)
		insertedSummary = true
	}
	if strings.TrimSpace(memorySection) != "" && !insertedMemory {
		prefix := []*schema.Message{schema.SystemMessage(MemoryMessagePrefix + memorySection)}
		if insertedSummary && len(out) > 0 && strings.HasPrefix(out[0].Content, SummaryMessagePrefix) {
			out = append(out[:1], append(prefix, out[1:]...)...)
		} else {
			out = append(prefix, out...)
		}
	}
	return out
}

func SelectWithinBudget(
	tools []*schema.ToolInfo,
	fixed, history []*schema.Message,
	summary, memorySection string,
	currentIndex int,
	budget int64,
) (prepared []*schema.Message, omitted int, summaryApplied, memoryApplied, fixedOverflow bool) {
	fixedOnly := ComposeMessages(fixed, nil, "", "", currentIndex)
	if int64(EstimateMessagesEnvelopeTokens(fixedOnly, tools)) > budget {
		return fixedOnly, len(history), false, false, true
	}
	effectiveMemory := strings.TrimSpace(memorySection)
	memoryApplied = effectiveMemory != ""
	if memoryApplied {
		withMemory := ComposeMessages(fixed, nil, "", effectiveMemory, currentIndex)
		if int64(EstimateMessagesEnvelopeTokens(withMemory, tools)) > budget {
			effectiveMemory = ""
			memoryApplied = false
		}
	}
	useSummary := summary != "" && int64(EstimateMessagesEnvelopeTokens(ComposeMessages(fixed, nil, summary, effectiveMemory, currentIndex), tools)) <= budget
	selected := make([]*schema.Message, 0, len(history))
	for i := len(history) - 1; i >= 0; i-- {
		candidate := append([]*schema.Message{history[i]}, selected...)
		summaryValue := conditionalString(useSummary, summary)
		if int64(EstimateMessagesEnvelopeTokens(ComposeMessages(fixed, candidate, summaryValue, effectiveMemory, currentIndex), tools)) <= budget {
			selected = candidate
		}
	}
	return ComposeMessages(fixed, selected, conditionalString(useSummary, summary), effectiveMemory, currentIndex),
		len(history) - len(selected), useSummary, memoryApplied, false
}

func conditionalString(ok bool, value string) string {
	if ok {
		return value
	}
	return ""
}

func PreparedSummaryText(messages []*schema.Message) string {
	for _, message := range messages {
		if message != nil && strings.HasPrefix(message.Content, SummaryMessagePrefix) {
			return strings.TrimSpace(strings.TrimPrefix(message.Content, SummaryMessagePrefix))
		}
	}
	return ""
}

func PreparedMemoryText(messages []*schema.Message) string {
	for _, message := range messages {
		if message != nil && strings.HasPrefix(message.Content, MemoryMessagePrefix) {
			return strings.TrimSpace(strings.TrimPrefix(message.Content, MemoryMessagePrefix))
		}
	}
	return ""
}

func (c *Controller) coveredHistoryCount() int {
	c.mu.Lock()
	covered := c.coveredID
	current := c.current
	c.mu.Unlock()
	count := 0
	for _, row := range c.history {
		if row.ID > 0 && row.ID <= covered && strings.TrimSpace(row.Content) != "" && strings.TrimSpace(row.Content) != current {
			count++
		}
	}
	return count
}

func (c *Controller) currentSummary() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.summary
}

func (c *Controller) summaryRows() ([]MessageRow, int64) {
	c.mu.Lock()
	covered := c.coveredID
	current := c.current
	c.mu.Unlock()
	rows := make([]MessageRow, 0, len(c.history))
	limit := len(c.history) - 10
	if limit <= 0 {
		return nil, 0
	}
	var maxID int64
	for _, row := range c.history[:limit] {
		if row.ID <= covered || row.ID == 0 || strings.TrimSpace(row.Content) == "" {
			continue
		}
		if strings.TrimSpace(row.Content) == current {
			continue
		}
		rows = append(rows, row)
		if row.ID > maxID {
			maxID = row.ID
		}
	}
	return rows, maxID
}

func (c *Controller) refreshSummary(ctx context.Context, category string) error {
	if c.store == nil || c.generator == nil {
		return nil
	}
	c.mu.Lock()
	if c.summaryDone && category == "sync_over_budget" {
		c.mu.Unlock()
		return nil
	}
	if category == "sync_over_budget" {
		c.summaryDone = true
	}
	oldSummary := c.summary
	c.mu.Unlock()
	rows, maxID := c.summaryRows()
	if len(rows) == 0 || maxID == 0 {
		return nil
	}
	texts := make([]string, 0, len(rows))
	for _, row := range rows {
		content := []rune(strings.TrimSpace(row.Content))
		if len(content) > 300 {
			content = content[:300]
		}
		texts = append(texts, fmt.Sprintf("%s: %s", row.Role, string(content)))
	}
	generated, err := c.generateSummary(ctx, oldSummary, texts)
	if err != nil || strings.TrimSpace(generated) == "" {
		logger.L().Warn("context summary generation failed", zap.Int64("session_id", c.sessionID), zap.String("error_category", category), zap.Int("message_count", len(rows)))
		if err != nil {
			return fmt.Errorf("summary generation: %w", err)
		}
		return errors.New("summary generation returned empty result")
	}
	generated = strings.TrimSpace(generated)
	written, err := c.store.UpsertSessionSummaryIfNewer(ctx, c.ownerID, c.sessionID, generated, maxID, len(rows))
	if err != nil {
		logger.L().Warn("context summary persistence failed", zap.Int64("session_id", c.sessionID), zap.String("error_category", "summary_persist"), zap.Int("message_count", len(rows)))
		return fmt.Errorf("summary persistence: %w", err)
	}
	if written {
		c.mu.Lock()
		if maxID > c.coveredID {
			c.summary, c.coveredID = generated, maxID
		}
		c.mu.Unlock()
	}
	return nil
}

func (c *Controller) generateSummary(ctx context.Context, oldSummary string, texts []string) (string, error) {
	if audienceGen, ok := c.generator.(AudienceSummaryGenerator); ok {
		return audienceGen.GenerateSessionSummaryForAudience(ctx, c.summaryAudience, oldSummary, texts)
	}
	return c.generator.GenerateSessionSummary(ctx, oldSummary, texts)
}

func (c *Controller) triggerAsyncSummary() {
	if c.store == nil || c.generator == nil {
		return
	}
	c.asyncOnce.Do(func() {
		go func() {
			defer func() {
				if recover() != nil {
					logger.L().Error("context async summary panic", zap.Int64("session_id", c.sessionID), zap.String("error_category", "summary_panic"))
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			_ = c.refreshSummary(ctx, "async_threshold")
		}()
	})
}
