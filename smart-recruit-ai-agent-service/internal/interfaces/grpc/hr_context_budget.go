package grpc

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/schema"

	"smart-recruit-ai-agent-service/internal/application/contextbudget"
	"smart-recruit-proto/recruitment/pb"
)

const (
	hrContextBudgetExceededCode = contextbudget.BudgetExceededCode
	hrContextConfigInvalidCode  = contextbudget.ConfigInvalidCode
	hrSummaryMessagePrefix      = contextbudget.SummaryMessagePrefix
	hrUnknownContextHistory     = contextbudget.UnknownContextHistory
)

type hrSessionSummaryStore = contextbudget.SessionSummaryStore

type hrContextGuardError = contextbudget.GuardError

type hrContextBudgetController struct {
	inner         *contextbudget.Controller
	model         RuntimeModelInfo
	tools         []*schema.ToolInfo
	current       string
	traces        []ToolTraceRow
	governance    hrRuntimeGovernanceContext
	memorySection string

	usage *pb.ContextUsageInfo
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
	inner := contextbudget.New(ctx, contextbudget.Config{
		Model: contextbudget.ModelInfo{
			ContextWindowTokens: model.ContextWindowTokens,
			MaxOutputTokens:     model.MaxOutputTokens,
		},
		Tools:           tools,
		Current:         current,
		History:         chatRowsToBudgetRows(history),
		OwnerID:         ownerID,
		SessionID:       sessionID,
		MemorySection:   hrBudgetMemorySection(governance.MemorySection),
		SummaryAudience: contextbudget.AudienceHR,
		Store:           store,
		Generator:       generator,
	})
	return &hrContextBudgetController{
		inner: inner, model: model, tools: tools, current: current,
		traces: traces, governance: governance,
		memorySection: hrBudgetMemorySection(governance.MemorySection),
		usage:         newHRContextUsageEnvelope(model, 0),
	}
}

func hrBudgetMemorySection(section string) string {
	section = strings.TrimSpace(section)
	if section == "" || section == emptyMemorySection {
		return ""
	}
	return section
}

func (c *hrContextBudgetController) prepare(ctx context.Context, messages []*schema.Message, stage string) ([]*schema.Message, error) {
	if c == nil || c.inner == nil {
		return messages, nil
	}
	prepared, err := c.inner.Prepare(ctx, messages, stage)
	if err != nil {
		if contextbudget.ErrorCode(err) == contextbudget.ConfigInvalidCode {
			c.usage = newHRContextUsageEnvelope(c.model, 0)
		}
		return nil, err
	}
	result := c.inner.LastPrepareResult()
	c.updateUsage(prepared, result.OmittedCount, result.SummaryApplied, result.MemoryApplied)
	window := int64(c.model.ContextWindowTokens)
	budget := int64(newHRContextUsageEnvelope(c.model, 0).GetInputBudgetTokens())
	if window <= 0 {
		c.usage.BudgetStatus = "unknown_config"
		c.usage.BudgetUsageRatio = 0
	} else if err == nil && budget > 0 {
		// Budget status already set by updateUsage via newHRContextUsageEnvelope ratios.
	}
	return prepared, nil
}

func (c *hrContextBudgetController) updateUsage(prepared []*schema.Message, omitted int, summaryApplied, memoryApplied bool) {
	next := estimateHRMessagesContextUsage(c.model, prepared, c.tools, c.current, c.traces, c.governance)
	next.OmittedMessageCount = saturatingInt32(int64(maxInt(omitted, 0)))
	next.SummaryApplied = summaryApplied
	next.MemoryApplied = memoryApplied
	if c.usage == nil {
		c.usage = next
	} else {
		*c.usage = *next
	}
}

func hrContextErrorCode(err error) string {
	return contextbudget.ErrorCode(err)
}

func dropCoveredHistoryPrefix(history []*schema.Message, count int) []*schema.Message {
	return contextbudget.DropCoveredHistoryPrefix(history, count)
}

func estimateMessagesEnvelopeTokens(messages []*schema.Message, tools []*schema.ToolInfo) int {
	return contextbudget.EstimateMessagesEnvelopeTokens(messages, tools)
}

func preparedHRSummary(messages []*schema.Message) string {
	return contextbudget.PreparedSummaryText(messages)
}

func selectedHRHistoryRows(messages []*schema.Message, history []ChatMessageRow, current ChatMessageRow) []ChatMessageRow {
	counts := make(map[string]int)
	for _, message := range messages {
		if message == nil || message.Role == schema.System || contextbudget.IsInjectedContextMessage(message.Content) {
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

func chatRowsToBudgetRows(rows []ChatMessageRow) []contextbudget.MessageRow {
	out := make([]contextbudget.MessageRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, contextbudget.MessageRow{ID: row.ID, Role: row.Role, Content: row.Content})
	}
	return out
}
