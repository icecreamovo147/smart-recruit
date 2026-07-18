package grpc

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
	"smart-recruit-proto/recruitment/pb"
)

const (
	hrContextBudgetExceededCode = "AI_CONTEXT_BUDGET_EXCEEDED"
	hrContextConfigInvalidCode  = "AI_CONTEXT_CONFIGURATION_INVALID"
	hrSummaryMessagePrefix      = "[Rolling conversation summary]\n"
	hrUnknownContextHistory     = 20
)

type hrContextGuardError struct{ code string }

func (e *hrContextGuardError) Error() string { return e.code }

func hrContextErrorCode(err error) string {
	var target *hrContextGuardError
	if errors.As(err, &target) {
		return target.code
	}
	return ""
}

type hrSessionSummaryStore interface {
	GetSessionSummaryState(ctx context.Context, ownerID, sessionID int64) (summary string, coveredMessageID int64, messageCount int, found bool, err error)
	UpsertSessionSummaryIfNewer(ctx context.Context, ownerID, sessionID int64, summary string, coveredMessageID int64, messageCount int) (bool, error)
}

type hrContextBudgetController struct {
	model      RuntimeModelInfo
	tools      []*schema.ToolInfo
	current    string
	traces     []ToolTraceRow
	governance hrRuntimeGovernanceContext
	history    []ChatMessageRow
	ownerID    int64
	sessionID  int64
	store      hrSessionSummaryStore
	generator  sessionSummaryGenerator
	usage      *pb.ContextUsageInfo

	mu          sync.Mutex
	summary     string
	coveredID   int64
	summaryDone bool
	asyncOnce   sync.Once
}

func newHRContextBudgetController(
	ctx context.Context,
	model RuntimeModelInfo,
	tools []*schema.ToolInfo,
	current string,
	traces []ToolTraceRow,
	governance hrRuntimeGovernanceContext,
	history []ChatMessageRow,
	ownerID, sessionID int64,
	store hrSessionSummaryStore,
	generator sessionSummaryGenerator,
) *hrContextBudgetController {
	c := &hrContextBudgetController{
		model: model, tools: tools, current: strings.TrimSpace(current), traces: traces,
		governance: governance, history: append([]ChatMessageRow(nil), history...),
		ownerID: ownerID, sessionID: sessionID, store: store, generator: generator,
		usage: newHRContextUsageEnvelope(model, 0),
	}
	if store != nil {
		loadCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if summary, covered, _, found, err := store.GetSessionSummaryState(loadCtx, ownerID, sessionID); err == nil && found {
			c.summary = strings.TrimSpace(summary)
			c.coveredID = covered
		} else if err != nil {
			logger.L().Warn("hr context summary load failed", zap.Int64("session_id", sessionID), zap.String("error_category", "summary_load"))
		}
	}
	return c
}

func (c *hrContextBudgetController) prepare(ctx context.Context, messages []*schema.Message, stage string) ([]*schema.Message, error) {
	if c == nil {
		return messages, nil
	}
	if c.model.MaxOutputTokens < 0 {
		c.usage = newHRContextUsageEnvelope(c.model, 0)
		return nil, &hrContextGuardError{code: hrContextConfigInvalidCode}
	}
	window := int64(c.model.ContextWindowTokens)
	budget := int64(newHRContextUsageEnvelope(c.model, 0).GetInputBudgetTokens())
	if window > 0 && budget <= 0 {
		c.usage = newHRContextUsageEnvelope(c.model, 0)
		return nil, &hrContextGuardError{code: hrContextConfigInvalidCode}
	}

	base, history, currentIndex := c.partition(messages)
	originalHistoryCount := len(history)
	allHistory := history
	if c.currentSummary() != "" {
		history = dropCoveredHistoryPrefix(allHistory, c.coveredHistoryCount())
	}
	if window <= 0 {
		if len(history) > hrUnknownContextHistory {
			history = history[len(history)-hrUnknownContextHistory:]
		}
		prepared := c.compose(base, history, c.currentSummary(), currentIndex)
		c.updateUsage(prepared, originalHistoryCount-len(history), c.currentSummary() != "")
		c.usage.BudgetStatus = "unknown_config"
		c.usage.BudgetUsageRatio = 0
		return prepared, nil
	}

	allPrepared := c.compose(base, history, c.currentSummary(), currentIndex)
	rawTokens := int64(estimateMessagesEnvelopeTokens(allPrepared, c.tools))
	if rawTokens > budget {
		// Refresh only the old portion. Failure is observable but trimming can
		// still produce a safe request.
		_ = c.refreshSummary(ctx, "sync_over_budget")
		if c.currentSummary() != "" {
			history = dropCoveredHistoryPrefix(allHistory, c.coveredHistoryCount())
		}
	}

	summary := c.currentSummary()
	prepared, omitted, summaryApplied, fixedOverflow := c.selectWithinBudget(base, history, summary, currentIndex, budget)
	omitted += originalHistoryCount - len(history)
	if fixedOverflow {
		c.updateUsage(base, len(history), false)
		c.usage.BudgetStatus = "over_budget"
		return nil, &hrContextGuardError{code: hrContextBudgetExceededCode}
	}
	c.updateUsage(prepared, omitted, summaryApplied)
	if rawTokens*100 >= budget*60 && rawTokens <= budget {
		c.triggerAsyncSummary()
	}
	_ = stage // retained for injectable/call-site diagnostics without logging content
	return prepared, nil
}

func dropCoveredHistoryPrefix(history []*schema.Message, count int) []*schema.Message {
	if count <= 0 {
		return history
	}
	if count >= len(history) {
		return nil
	}
	return history[count:]
}

func (c *hrContextBudgetController) coveredHistoryCount() int {
	c.mu.Lock()
	covered := c.coveredID
	c.mu.Unlock()
	count := 0
	for _, row := range c.history {
		if row.ID > 0 && row.ID <= covered && strings.TrimSpace(row.Content) != "" && strings.TrimSpace(row.Content) != c.current {
			count++
		}
	}
	return count
}

func (c *hrContextBudgetController) partition(messages []*schema.Message) (fixed []*schema.Message, history []*schema.Message, currentIndex int) {
	clean := make([]*schema.Message, 0, len(messages))
	for _, message := range messages {
		if message == nil || strings.HasPrefix(message.Content, hrSummaryMessagePrefix) {
			continue
		}
		clean = append(clean, message)
	}
	currentIndex = -1
	for i := len(clean) - 1; i >= 0; i-- {
		if clean[i].Role == schema.User && c.current != "" && strings.TrimSpace(clean[i].Content) == c.current {
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

func (c *hrContextBudgetController) compose(fixed, history []*schema.Message, summary string, _ int) []*schema.Message {
	out := make([]*schema.Message, 0, len(fixed)+len(history)+1)
	insertedSummary := false
	for _, message := range fixed {
		if message.Role == schema.System {
			out = append(out, message)
			if !insertedSummary && summary != "" {
				out = append(out, schema.SystemMessage(hrSummaryMessagePrefix+summary))
				insertedSummary = true
			}
		}
	}
	// Fixed system messages are emitted first, then chronological history, then
	// the current user/tool chain. This preserves role ordering within each part.
	out = append(out, history...)
	for _, message := range fixed {
		if message.Role != schema.System {
			out = append(out, message)
		}
	}
	if summary != "" && !insertedSummary {
		out = append([]*schema.Message{schema.SystemMessage(hrSummaryMessagePrefix + summary)}, out...)
	}
	return out
}

func (c *hrContextBudgetController) selectWithinBudget(fixed, history []*schema.Message, summary string, currentIndex int, budget int64) ([]*schema.Message, int, bool, bool) {
	fixedOnly := c.compose(fixed, nil, "", currentIndex)
	if int64(estimateMessagesEnvelopeTokens(fixedOnly, c.tools)) > budget {
		return fixedOnly, len(history), false, true
	}
	useSummary := summary != "" && int64(estimateMessagesEnvelopeTokens(c.compose(fixed, nil, summary, currentIndex), c.tools)) <= budget
	selected := make([]*schema.Message, 0, len(history))
	for i := len(history) - 1; i >= 0; i-- {
		candidate := append([]*schema.Message{history[i]}, selected...)
		if int64(estimateMessagesEnvelopeTokens(c.compose(fixed, candidate, conditionalString(useSummary, summary), currentIndex), c.tools)) <= budget {
			selected = candidate
		}
	}
	return c.compose(fixed, selected, conditionalString(useSummary, summary), currentIndex), len(history) - len(selected), useSummary, false
}

func conditionalString(ok bool, value string) string {
	if ok {
		return value
	}
	return ""
}

func estimateMessagesEnvelopeTokens(messages []*schema.Message, tools []*schema.ToolInfo) int {
	total := estimateToolSchemaTokens(tools) + len(messages)*4 + 2
	for _, message := range messages {
		if message != nil {
			total += estimateTokensConservative(message.Content)
		}
	}
	return total
}

func (c *hrContextBudgetController) updateUsage(prepared []*schema.Message, omitted int, summaryApplied bool) {
	next := estimateHRMessagesContextUsage(c.model, prepared, c.tools, c.current, c.traces, c.governance)
	next.OmittedMessageCount = saturatingInt32(int64(maxInt(omitted, 0)))
	next.SummaryApplied = summaryApplied
	if c.usage == nil {
		c.usage = next
	} else {
		*c.usage = *next
	}
}

func (c *hrContextBudgetController) currentSummary() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.summary
}

func (c *hrContextBudgetController) summaryRows() ([]ChatMessageRow, int64) {
	c.mu.Lock()
	covered := c.coveredID
	c.mu.Unlock()
	rows := make([]ChatMessageRow, 0, len(c.history))
	limit := len(c.history) - 10
	if limit <= 0 {
		return nil, 0
	}
	var maxID int64
	for _, row := range c.history[:limit] {
		if row.ID <= covered || row.ID == 0 || strings.TrimSpace(row.Content) == "" {
			continue
		}
		rows = append(rows, row)
		if row.ID > maxID {
			maxID = row.ID
		}
	}
	return rows, maxID
}

func (c *hrContextBudgetController) refreshSummary(ctx context.Context, category string) error {
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
	generated, err := c.generator.GenerateSessionSummary(ctx, oldSummary, texts)
	if err != nil || strings.TrimSpace(generated) == "" {
		logger.L().Warn("hr context summary generation failed", zap.Int64("session_id", c.sessionID), zap.String("error_category", category), zap.Int("message_count", len(rows)))
		if err != nil {
			return fmt.Errorf("summary generation: %w", err)
		}
		return errors.New("summary generation returned empty result")
	}
	generated = strings.TrimSpace(generated)
	written, err := c.store.UpsertSessionSummaryIfNewer(ctx, c.ownerID, c.sessionID, generated, maxID, len(rows))
	if err != nil {
		logger.L().Warn("hr context summary persistence failed", zap.Int64("session_id", c.sessionID), zap.String("error_category", "summary_persist"), zap.Int("message_count", len(rows)))
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

func (c *hrContextBudgetController) triggerAsyncSummary() {
	if c.store == nil || c.generator == nil {
		return
	}
	c.asyncOnce.Do(func() {
		go func() {
			defer func() {
				if recover() != nil {
					logger.L().Error("hr context async summary panic", zap.Int64("session_id", c.sessionID), zap.String("error_category", "summary_panic"))
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			_ = c.refreshSummary(ctx, "async_threshold")
		}()
	})
}

func preparedHRSummary(messages []*schema.Message) string {
	for _, message := range messages {
		if message != nil && strings.HasPrefix(message.Content, hrSummaryMessagePrefix) {
			return strings.TrimSpace(strings.TrimPrefix(message.Content, hrSummaryMessagePrefix))
		}
	}
	return ""
}

func selectedHRHistoryRows(messages []*schema.Message, history []ChatMessageRow, current ChatMessageRow) []ChatMessageRow {
	counts := make(map[string]int)
	for _, message := range messages {
		if message == nil || message.Role == schema.System || strings.HasPrefix(message.Content, hrSummaryMessagePrefix) {
			continue
		}
		role := "user"
		if message.Role == schema.Assistant {
			role = "assistant"
		}
		counts[role+"\x00"+strings.TrimSpace(message.Content)]++
	}
	out := make([]ChatMessageRow, 0, len(history))
	for _, row := range history {
		if current.ID != 0 && row.ID == current.ID {
			continue
		}
		key := row.Role + "\x00" + strings.TrimSpace(row.Content)
		if counts[key] <= 0 {
			continue
		}
		counts[key]--
		out = append(out, row)
	}
	return out
}
