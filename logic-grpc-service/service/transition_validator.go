package service

import (
	"fmt"

	"logic-grpc-service/model"
)

// ── Transition Matrix ──────────────────────────────────────────────────
// allowedTransitions defines the valid status key state machine.
// Only transitions listed here (or their reverse, for withdrawal) are permitted.
// Adding a new transition here automatically enables it across the service layer.

var allowedTransitions = map[string]map[string]bool{
	model.StatusKeyApplied: {
		model.StatusKeyViewed:       true,
		model.StatusKeyScreenPassed: true,
		model.StatusKeyRejected:     true,
		model.StatusKeyWithdrawn:    true,
	},
	model.StatusKeyViewed: {
		model.StatusKeyScreening:        true,
		model.StatusKeyScreenPassed:     true,
		model.StatusKeyInterviewPending: true, // skip screening, go directly to interview
		model.StatusKeyInterviewPassed:  true, // skip intermediate stages
		model.StatusKeyOfferPending:     true, // skip intermediate stages
		model.StatusKeyRejected:         true,
		model.StatusKeyWithdrawn:        true,
	},
	model.StatusKeyScreening: {
		model.StatusKeyScreenPassed: true,
		model.StatusKeyRejected:     true,
		model.StatusKeyWithdrawn:    true,
	},
	model.StatusKeyScreenPassed: {
		model.StatusKeyInterviewPending: true,
		model.StatusKeyRejected:         true,
		model.StatusKeyWithdrawn:        true,
	},
	model.StatusKeyInterviewPending: {
		model.StatusKeyInterviewing:    true,
		model.StatusKeyInterviewPassed: true, // HR can directly pass after all rounds complete
		model.StatusKeyRejected:        true,
		model.StatusKeyWithdrawn:       true,
	},
	model.StatusKeyInterviewing: {
		model.StatusKeyInterviewPending: true, // reschedule after cancellation within same round
		model.StatusKeyInterviewPassed:  true,
		model.StatusKeyRejected:         true,
		model.StatusKeyWithdrawn:        true,
	},
	model.StatusKeyInterviewPassed: {
		model.StatusKeyInterviewPending: true, // multi-round interview loop (2nd, 3rd, ...)
		model.StatusKeyOfferPending:     true,
		model.StatusKeyRejected:         true,
		model.StatusKeyWithdrawn:        true,
	},
	model.StatusKeyOfferPending: {
		model.StatusKeyOfferSent: true,
		model.StatusKeyRejected:  true,
		model.StatusKeyWithdrawn: true,
	},
	model.StatusKeyOfferSent: {
		model.StatusKeyOfferAccepted: true,
		model.StatusKeyOfferRejected: true,
		model.StatusKeyOfferPending:  true, // HR withdraws sent offer → back to pending for re-issuance
		model.StatusKeyWithdrawn:     true,
	},
	model.StatusKeyOfferAccepted: {
		model.StatusKeyHired: true,
	},
	// Terminal states: no outgoing transitions (except Rejected → ScreenPassed for HR re-pass).
	model.StatusKeyHired:         {},
	model.StatusKeyRejected: {
		model.StatusKeyScreenPassed: true, // HR re-pass creates a new application round.
	},
	model.StatusKeyOfferRejected: {},
	model.StatusKeyWithdrawn:     {},
}

// ── Validation ─────────────────────────────────────────────────────────

// TransitionError is returned when an invalid status transition is attempted.
type TransitionError struct {
	From string
	To   string
	Msg  string
}

func (e *TransitionError) Error() string {
	return e.Msg
}

// ValidateTransition checks if a transition from `from` to `to` is allowed.
// Returns nil if the transition is valid, or a TransitionError if not.
func ValidateTransition(from, to string) error {
	if from == to {
		return &TransitionError{
			From: from,
			To:   to,
			Msg:  fmt.Sprintf("状态未变更：%s", model.HRStatusLabels[from]),
		}
	}
	if targets, ok := allowedTransitions[from]; ok {
		if targets[to] {
			return nil
		}
	}
	return &TransitionError{
		From: from,
		To:   to,
		Msg:  fmt.Sprintf("不允许从「%s」变更为「%s」", model.HRStatusLabels[from], model.HRStatusLabels[to]),
	}
}

// AllowedNextStatuses returns the set of status keys that can transition from the given status.
func AllowedNextStatuses(from string) map[string]bool {
	if targets, ok := allowedTransitions[from]; ok {
		result := make(map[string]bool, len(targets))
		for k, v := range targets {
			result[k] = v
		}
		return result
	}
	return map[string]bool{}
}

// ValidateStatusKey returns nil if the key is a known application status key.
func ValidateStatusKey(key string) error {
	if _, ok := model.HRStatusLabels[key]; !ok {
		return &TransitionError{
			From: key,
			Msg:  fmt.Sprintf("未知的投递状态：%s", key),
		}
	}
	return nil
}
