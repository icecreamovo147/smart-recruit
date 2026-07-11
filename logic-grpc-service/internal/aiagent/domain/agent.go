package domain

// SessionID identifies an AI Agent chat session.
type SessionID uint64

// AgentRunID identifies a durable AI Agent run.
type AgentRunID uint64

// AgentType identifies the configured AI Agent runtime.
type AgentType string

const (
	AgentTypeHRRecruiting       AgentType = "hr_recruiting_agent"
	AgentTypeCandidateAssistant AgentType = "candidate_assistant"
)

// AgentRunStatus is the current durable lifecycle state for an agent run.
type AgentRunStatus string

const (
	AgentRunStatusQueued              AgentRunStatus = "queued"
	AgentRunStatusPlanning            AgentRunStatus = "planning"
	AgentRunStatusRunning             AgentRunStatus = "running"
	AgentRunStatusWaitingConfirmation AgentRunStatus = "waiting_confirmation"
	AgentRunStatusCancelRequested     AgentRunStatus = "cancel_requested"
	AgentRunStatusSucceeded           AgentRunStatus = "succeeded"
	AgentRunStatusPartial             AgentRunStatus = "partial"
	AgentRunStatusFailed              AgentRunStatus = "failed"
	AgentRunStatusCanceled            AgentRunStatus = "canceled"
)

// EmbeddingStatus is the current semantic embedding availability state.
type EmbeddingStatus string

const (
	EmbeddingStatusReady       EmbeddingStatus = "ready"
	EmbeddingStatusUnavailable EmbeddingStatus = "unavailable"
	EmbeddingStatusInactive    EmbeddingStatus = "inactive"
)

// MCPPolicyDecision is the persisted governance decision for an MCP tool call.
type MCPPolicyDecision string

const (
	MCPPolicyDecisionAllow        MCPPolicyDecision = "allow"
	MCPPolicyDecisionDeny         MCPPolicyDecision = "deny"
	MCPPolicyDecisionConfirmation MCPPolicyDecision = "confirmation_required"
	MCPPolicyDecisionRateLimited  MCPPolicyDecision = "rate_limited"
)
