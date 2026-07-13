package ai

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

type ToolResult struct {
	Content  string
	Metadata ToolMetadata
}

type ToolMetadata struct {
	CandidateOptions []ToolCandidateOption
	Action           *ToolAction

	// BillingTokenUsage accumulates every model call's token usage in a single
	// user action. Used for audit/cost tracking.
	BillingTokenUsage *schema.TokenUsage
	// ContextTokenUsage is overwritten by each model call and holds the latest
	// model call's token usage. Used for context window occupancy display.
	ContextTokenUsage *schema.TokenUsage
	// ToolTraces accumulates one entry per executed tool call within a single
	// ChatWithTools or ADK run.
	ToolTraces []ToolTrace
}

type ToolTrace struct {
	ToolName  string
	Arguments map[string]any
	Result    string
	Cost      time.Duration
	Error     error
}

type ToolCandidateOption struct {
	ApplicationID int64  `json:"application_id"`
	CandidateName string `json:"candidate_name"`
	MaskedPhone   string `json:"masked_phone"`
	JobTitle      string `json:"job_title"`
	StatusText    string `json:"status_text"`
	RoundNo       int32  `json:"round_no"`
	IsCurrent     int32  `json:"is_current"`
	AppliedAt     string `json:"applied_at"`
}

type ToolAction struct {
	Action        string
	ApplicationID int64
	ActionStatus  int32
	CandidateName string
	JobTitle      string
	Status        int32
}

func (m *ToolMetadata) merge(other ToolMetadata) {
	if len(other.CandidateOptions) > 0 {
		m.CandidateOptions = append(m.CandidateOptions, other.CandidateOptions...)
	}
	if other.Action != nil {
		m.Action = other.Action
	}
	m.addBillingTokenUsage(other.BillingTokenUsage)
	if other.ContextTokenUsage != nil {
		m.setContextTokenUsage(other.ContextTokenUsage)
	}
}

func (m *ToolMetadata) recordTrace(t ToolTrace) {
	m.ToolTraces = append(m.ToolTraces, t)
}

func (m *ToolMetadata) addBillingTokenUsage(usage *schema.TokenUsage) {
	if usage == nil {
		return
	}
	if m.BillingTokenUsage == nil {
		m.BillingTokenUsage = &schema.TokenUsage{}
	}
	m.BillingTokenUsage.PromptTokens += usage.PromptTokens
	m.BillingTokenUsage.PromptTokenDetails.CachedTokens += usage.PromptTokenDetails.CachedTokens
	m.BillingTokenUsage.CompletionTokens += usage.CompletionTokens
	m.BillingTokenUsage.CompletionTokensDetails.ReasoningTokens += usage.CompletionTokensDetails.ReasoningTokens
	m.BillingTokenUsage.TotalTokens += usage.TotalTokens
}

func (m *ToolMetadata) setContextTokenUsage(usage *schema.TokenUsage) {
	if usage == nil {
		return
	}
	if m.ContextTokenUsage == nil {
		m.ContextTokenUsage = &schema.TokenUsage{}
	}
	*m.ContextTokenUsage = *usage
}

func (m *ToolMetadata) recordModelUsage(usage *schema.TokenUsage) {
	m.addBillingTokenUsage(usage)
	m.setContextTokenUsage(usage)
}
