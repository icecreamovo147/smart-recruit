package ai

import "context"

// contextKey is used for per-request values stored in context.Context so ADK
// tool closures can retrieve owner/state without capturing them at tool
// creation time (enabling tool caching across requests).
type contextKey string

func (k contextKey) String() string { return "ai_context_" + string(k) }

const (
	contextKeyOwnerID contextKey = "owner_id"
	contextKeyState   contextKey = "state"
)

// WithOwnerID stores the user identifier (HR ID or candidate user ID) in the
// context for ADK tool handlers.
func WithOwnerID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, contextKeyOwnerID, id)
}

// WithAgentRunState stores the per-run AgentRunState in the context so tool
// handlers can propagate metadata without capturing the state pointer at
// tool-creation time.
func WithAgentRunState(ctx context.Context, state *AgentRunState) context.Context {
	return context.WithValue(ctx, contextKeyState, state)
}

func ownerIDFromContext(ctx context.Context) int64 {
	if v, ok := ctx.Value(contextKeyOwnerID).(int64); ok {
		return v
	}
	return 0
}

// OwnerIDFromContext exposes the request owner identifier for service-level
// tool wrappers that live outside the ai package.
func OwnerIDFromContext(ctx context.Context) int64 {
	return ownerIDFromContext(ctx)
}

func agentStateFromContext(ctx context.Context) *AgentRunState {
	if v, ok := ctx.Value(contextKeyState).(*AgentRunState); ok {
		return v
	}
	return nil
}

// AgentRunStateFromContext exposes the per-run state for service wrappers.
func AgentRunStateFromContext(ctx context.Context) *AgentRunState {
	return agentStateFromContext(ctx)
}
