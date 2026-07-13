package model

import "time"

const (
	AgentRunStatusQueued              = "queued"
	AgentRunStatusPlanning            = "planning"
	AgentRunStatusRunning             = "running"
	AgentRunStatusWaitingConfirmation = "waiting_confirmation"
	AgentRunStatusCancelRequested     = "cancel_requested"
	AgentRunStatusSucceeded           = "succeeded"
	AgentRunStatusPartial             = "partial"
	AgentRunStatusFailed              = "failed"
	AgentRunStatusCanceled            = "canceled"

	PromptRoleSystem = "system"
)

type ChatSession struct {
	ID        uint64
	OwnerID   uint64
	OwnerRole string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ChatMessage struct {
	ID        uint64
	SessionID uint64
	RunID     uint64
	Role      string
	Content   string
	CreatedAt time.Time
}

type AgentRun struct {
	ID              uint64
	SessionID       uint64
	ActorID         uint64
	AgentType       string
	AgentName       string
	ModelID         uint64
	ModelName       string
	Status          string
	ClientRequestID string
	Message         string
	ActionType      string
	FinalAnswer     string
	ErrorType       string
	ErrorMessage    string
	StartedAt       time.Time
	CompletedAt     *time.Time
	CancelRequested *time.Time
	CanceledAt      *time.Time
	LastEventSeq    int64
}

func (r AgentRun) IsTerminal() bool {
	switch r.Status {
	case AgentRunStatusSucceeded, AgentRunStatusPartial, AgentRunStatusFailed, AgentRunStatusCanceled:
		return true
	default:
		return false
	}
}

func (r AgentRun) IsActive() bool {
	switch r.Status {
	case AgentRunStatusQueued, AgentRunStatusPlanning, AgentRunStatusRunning, AgentRunStatusWaitingConfirmation, AgentRunStatusCancelRequested:
		return true
	default:
		return false
	}
}

type AgentRunEvent struct {
	RunID     uint64
	Seq       int64
	EventType string
	Payload   string
	CreatedAt time.Time
}

type PromptTemplate struct {
	ID        uint64
	Name      string
	Content   string
	Variables string
	Version   int64
	IsActive  bool
	AgentType string
	Role      string
	CreatedBy uint64
	UpdatedBy uint64
}

type PromptVersion struct {
	TemplateID uint64
	Version    int64
	Content    string
	ChangedBy  uint64
	ChangeNote string
}

type ModelConfig struct {
	ID           uint64
	ProviderID   uint64
	ModelName    string
	ProviderType string
	ProviderName string
	Enabled      bool
	Default      bool
}

type ProviderCandidate struct {
	ID              uint64
	Name            string
	Type            string
	ModelName       string
	Enabled         bool
	Default         bool
	FallbackAllowed bool
}

type AuditEvent struct {
	ActorID       uint64
	Operation     string
	ResourceType  string
	ResourceID    uint64
	Decision      string
	Reason        string
	CorrelationID string
	OccurredAt    time.Time
}
