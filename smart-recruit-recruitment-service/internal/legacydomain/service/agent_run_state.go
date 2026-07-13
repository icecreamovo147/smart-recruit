package service

import (
	"fmt"
	"strings"
)

// Durable agent run statuses (SPEC FR-004).
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
)

// RequiredAgentRunStatuses is the complete FR-004 status set.
var RequiredAgentRunStatuses = []string{
	AgentRunStatusQueued,
	AgentRunStatusPlanning,
	AgentRunStatusRunning,
	AgentRunStatusWaitingConfirmation,
	AgentRunStatusCancelRequested,
	AgentRunStatusSucceeded,
	AgentRunStatusPartial,
	AgentRunStatusFailed,
	AgentRunStatusCanceled,
}

// agentRunTransitions defines legal from->to edges for durable runs.
// Same-status transitions are treated as idempotent no-ops by ValidateAgentRunTransition.
var agentRunTransitions = map[string]map[string]struct{}{
	AgentRunStatusQueued: {
		AgentRunStatusPlanning:        {},
		AgentRunStatusCancelRequested: {},
		AgentRunStatusFailed:          {},
		AgentRunStatusCanceled:        {},
	},
	AgentRunStatusPlanning: {
		AgentRunStatusRunning:             {},
		AgentRunStatusWaitingConfirmation: {},
		AgentRunStatusCancelRequested:     {},
		AgentRunStatusFailed:              {},
		AgentRunStatusPartial:             {},
		AgentRunStatusCanceled:            {},
	},
	AgentRunStatusRunning: {
		AgentRunStatusWaitingConfirmation: {},
		AgentRunStatusCancelRequested:     {},
		AgentRunStatusSucceeded:           {},
		AgentRunStatusPartial:             {},
		AgentRunStatusFailed:              {},
		AgentRunStatusCanceled:            {},
	},
	AgentRunStatusWaitingConfirmation: {
		AgentRunStatusRunning:         {},
		AgentRunStatusCancelRequested: {},
		AgentRunStatusFailed:          {},
		AgentRunStatusCanceled:        {},
	},
	AgentRunStatusCancelRequested: {
		AgentRunStatusCanceled: {},
		AgentRunStatusFailed:   {},
		AgentRunStatusPartial:  {},
	},
}

// IsKnownAgentRunStatus reports whether status is one of the required run states.
func IsKnownAgentRunStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case AgentRunStatusQueued,
		AgentRunStatusPlanning,
		AgentRunStatusRunning,
		AgentRunStatusWaitingConfirmation,
		AgentRunStatusCancelRequested,
		AgentRunStatusSucceeded,
		AgentRunStatusPartial,
		AgentRunStatusFailed,
		AgentRunStatusCanceled:
		return true
	default:
		return false
	}
}

// IsTerminalAgentRunStatus reports whether status is a terminal durable run state.
func IsTerminalAgentRunStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case AgentRunStatusSucceeded, AgentRunStatusPartial, AgentRunStatusFailed, AgentRunStatusCanceled:
		return true
	default:
		return false
	}
}

// IsActiveAgentRunStatus reports whether status keeps a run as the session's active run.
func IsActiveAgentRunStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case AgentRunStatusQueued,
		AgentRunStatusPlanning,
		AgentRunStatusRunning,
		AgentRunStatusWaitingConfirmation,
		AgentRunStatusCancelRequested:
		return true
	default:
		return false
	}
}

// ValidateAgentRunTransition checks whether from->to is a legal durable run transition.
// Identical from/to is allowed as an idempotent no-op (duplicate cancel/confirm/status writes).
func ValidateAgentRunTransition(from, to string) error {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if !IsKnownAgentRunStatus(from) {
		return fmt.Errorf("unknown agent run status %q", from)
	}
	if !IsKnownAgentRunStatus(to) {
		return fmt.Errorf("unknown agent run status %q", to)
	}
	if from == to {
		return nil
	}
	allowed, ok := agentRunTransitions[from]
	if !ok {
		return fmt.Errorf("illegal agent run transition %s -> %s", from, to)
	}
	if _, ok := allowed[to]; !ok {
		return fmt.Errorf("illegal agent run transition %s -> %s", from, to)
	}
	return nil
}

// CanTransitionAgentRun reports whether from->to is allowed (including idempotent same-status).
func CanTransitionAgentRun(from, to string) bool {
	return ValidateAgentRunTransition(from, to) == nil
}

// IsStaleOrDuplicateEventSeq reports whether incomingSeq should be ignored because it was
// already applied (duplicate) or is older than the last applied sequence (stale).
func IsStaleOrDuplicateEventSeq(lastAppliedSeq, incomingSeq int64) bool {
	return incomingSeq <= lastAppliedSeq
}
