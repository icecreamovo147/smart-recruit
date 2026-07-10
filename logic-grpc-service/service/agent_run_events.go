package service

// Durable run event type constants (SDD §6).
const (
	AgentRunEventCreated              = "run.created"
	AgentRunEventStatusChanged        = "run.status_changed"
	AgentRunEventAssistantDelta       = "assistant.delta"
	AgentRunEventAssistantSnapshot    = "assistant.snapshot"
	AgentRunEventProcessDelta         = "process.delta"
	AgentRunEventProcessSnapshot      = "process.snapshot"
	AgentRunEventToolStarted          = "tool.started"
	AgentRunEventToolFinished         = "tool.finished"
	AgentRunEventConfirmationRequired = "confirmation.required"
	AgentRunEventConfirmationAccepted = "confirmation.accepted"
	AgentRunEventResult               = "run.result"
	AgentRunEventError                = "run.error"
	AgentRunEventCompleted            = "run.completed"
	AgentRunEventCanceled             = "run.canceled"
	AgentRunEventHeartbeat            = "run.heartbeat"
)

// planRequestKey is the plan_json object key holding the original create-run request.
const planRequestKey = "durable_request"
