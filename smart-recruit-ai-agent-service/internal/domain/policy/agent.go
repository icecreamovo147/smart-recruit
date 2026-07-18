package policy

import (
	"errors"
	"fmt"
	"strings"

	"smart-recruit-ai-agent-service/internal/domain/model"
)

var (
	ErrUnknownAgentRunStatus      = errors.New("unknown agent run status")
	ErrIllegalAgentRunTransition  = errors.New("illegal agent run transition")
	ErrAgentRunRequestInvalid     = errors.New("agent run request invalid")
	ErrPromptInvalid              = errors.New("prompt template invalid")
	ErrProviderUnavailable        = errors.New("provider unavailable")
	ErrProviderFallbackDisallowed = errors.New("provider fallback disabled")
)

var RequiredAgentRunStatuses = []string{
	model.AgentRunStatusQueued,
	model.AgentRunStatusPlanning,
	model.AgentRunStatusRunning,
	model.AgentRunStatusWaitingConfirmation,
	model.AgentRunStatusCancelRequested,
	model.AgentRunStatusSucceeded,
	model.AgentRunStatusPartial,
	model.AgentRunStatusFailed,
	model.AgentRunStatusCanceled,
}

var allowedTransitions = map[string]map[string]struct{}{
	model.AgentRunStatusQueued: {
		model.AgentRunStatusPlanning:        {},
		model.AgentRunStatusCancelRequested: {},
		model.AgentRunStatusFailed:          {},
		model.AgentRunStatusCanceled:        {},
	},
	model.AgentRunStatusPlanning: {
		model.AgentRunStatusRunning:             {},
		model.AgentRunStatusWaitingConfirmation: {},
		model.AgentRunStatusCancelRequested:     {},
		model.AgentRunStatusFailed:              {},
		model.AgentRunStatusPartial:             {},
		model.AgentRunStatusCanceled:            {},
	},
	model.AgentRunStatusRunning: {
		model.AgentRunStatusWaitingConfirmation: {},
		model.AgentRunStatusCancelRequested:     {},
		model.AgentRunStatusSucceeded:           {},
		model.AgentRunStatusPartial:             {},
		model.AgentRunStatusFailed:              {},
		model.AgentRunStatusCanceled:            {},
	},
	model.AgentRunStatusWaitingConfirmation: {
		model.AgentRunStatusRunning:         {},
		model.AgentRunStatusCancelRequested: {},
		model.AgentRunStatusFailed:          {},
		model.AgentRunStatusCanceled:        {},
	},
	model.AgentRunStatusCancelRequested: {
		model.AgentRunStatusCanceled: {},
		model.AgentRunStatusFailed:   {},
		model.AgentRunStatusPartial:  {},
	},
}

func IsKnownAgentRunStatus(status string) bool {
	for _, candidate := range RequiredAgentRunStatuses {
		if status == candidate {
			return true
		}
	}
	return false
}

func ValidateAgentRunTransition(from, to string) error {
	if !IsKnownAgentRunStatus(from) {
		return fmt.Errorf("%w: from=%s", ErrUnknownAgentRunStatus, from)
	}
	if !IsKnownAgentRunStatus(to) {
		return fmt.Errorf("%w: to=%s", ErrUnknownAgentRunStatus, to)
	}
	if from == to {
		return nil
	}
	if allowed, ok := allowedTransitions[from]; ok {
		if _, ok := allowed[to]; ok {
			return nil
		}
	}
	return fmt.Errorf("%w: %s -> %s", ErrIllegalAgentRunTransition, from, to)
}

func IsStaleOrDuplicateEventSeq(lastAppliedSeq, incomingSeq int64) bool {
	return incomingSeq <= lastAppliedSeq
}

func ValidateAgentRunRequest(sessionID, actorID uint64, clientRequestID, message, actionType string) error {
	if sessionID == 0 {
		return fmt.Errorf("%w: session_id is required", ErrAgentRunRequestInvalid)
	}
	if actorID == 0 {
		return fmt.Errorf("%w: actor_id is required", ErrAgentRunRequestInvalid)
	}
	if strings.TrimSpace(clientRequestID) == "" {
		return fmt.Errorf("%w: client_request_id is required", ErrAgentRunRequestInvalid)
	}
	if strings.TrimSpace(message) == "" && strings.TrimSpace(actionType) == "" {
		return fmt.Errorf("%w: message or action_type is required", ErrAgentRunRequestInvalid)
	}
	return nil
}

func ValidatePrompt(template model.PromptTemplate) error {
	if strings.TrimSpace(template.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrPromptInvalid)
	}
	if strings.TrimSpace(template.Content) == "" {
		return fmt.Errorf("%w: content is required", ErrPromptInvalid)
	}
	if strings.TrimSpace(template.AgentType) == "" {
		return fmt.Errorf("%w: agent_type is required", ErrPromptInvalid)
	}
	return nil
}

func NextPromptVersion(existing model.PromptTemplate, newContent string) (int64, bool) {
	if strings.TrimSpace(newContent) == "" || newContent == existing.Content {
		return existing.Version, false
	}
	if existing.Version <= 0 {
		return 1, true
	}
	return existing.Version + 1, true
}

func SelectProvider(primary model.ProviderCandidate, fallbacks []model.ProviderCandidate, fallbackEnabled bool) (model.ProviderCandidate, error) {
	if primary.Enabled {
		return primary, nil
	}
	if !fallbackEnabled {
		return model.ProviderCandidate{}, ErrProviderFallbackDisallowed
	}
	for _, candidate := range fallbacks {
		if candidate.Enabled && candidate.FallbackAllowed {
			return candidate, nil
		}
	}
	return model.ProviderCandidate{}, ErrProviderUnavailable
}
