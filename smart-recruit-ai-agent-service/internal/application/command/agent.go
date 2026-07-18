package command

type CreateAgentRun struct {
	ActorID         uint64
	SessionID       uint64
	ClientRequestID string
	Message         string
	ActionType      string
	AgentType       string
	AgentName       string
	ModelID         uint64
	ModelName       string
}

type CancelAgentRun struct {
	ActorID uint64
	RunID   uint64
}

type ConfirmAgentRun struct {
	ActorID uint64
	RunID   uint64
}

type CreatePromptTemplate struct {
	ActorID   uint64
	Name      string
	Content   string
	Variables string
	AgentType string
	Role      string
}

type UpdatePromptTemplate struct {
	ActorID   uint64
	ID        uint64
	Name      string
	Content   string
	Variables string
	IsActive  *bool
}
