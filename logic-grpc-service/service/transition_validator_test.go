package service

import (
	"errors"
	"testing"

	"logic-grpc-service/model"
)

// TestForbiddenTransitions verifies that at least five forbidden transitions
// return a TransitionError (AC-2 requirement).
func TestForbiddenTransitions(t *testing.T) {
	forbidden := []struct {
		from, to string
		desc     string
	}{
		{model.StatusKeyApplied, model.StatusKeyHired, "applied -> hired: skips too many steps"},
		{model.StatusKeyRejected, model.StatusKeyApplied, "rejected -> applied: terminal to non-terminal"},
		{model.StatusKeyHired, model.StatusKeyRejected, "hired -> rejected: terminal to anything"},
		{model.StatusKeyWithdrawn, model.StatusKeyViewed, "withdrawn -> viewed: terminal to non-terminal"},
		{model.StatusKeyOfferRejected, model.StatusKeyOfferPending, "offer_rejected -> offer_pending: terminal to non-terminal"},
		{model.StatusKeyApplied, model.StatusKeyApplied, "applied -> applied: same-state transition"},
	}

	for _, tt := range forbidden {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if err == nil {
				t.Errorf("expected error for transition %q -> %q, got nil", tt.from, tt.to)
				return
			}
			var te *TransitionError
			if !errors.As(err, &te) {
				t.Errorf("expected TransitionError for %q -> %q, got %T: %v", tt.from, tt.to, err, err)
			}
			if te.From != tt.from {
				t.Errorf("expected TransitionError.From=%q, got %q", tt.from, te.From)
			}
			if te.To != tt.to {
				t.Errorf("expected TransitionError.To=%q, got %q", tt.to, te.To)
			}
		})
	}
}

// TestAllowedTransitions verifies that at least three allowed transitions return nil.
func TestAllowedTransitions(t *testing.T) {
	allowed := []struct {
		from, to string
		desc     string
	}{
		{model.StatusKeyApplied, model.StatusKeyViewed, "applied -> viewed"},
		{model.StatusKeyApplied, model.StatusKeyScreenPassed, "applied -> screen_passed"},
		{model.StatusKeyViewed, model.StatusKeyScreenPassed, "viewed -> screen_passed"},
		{model.StatusKeyScreening, model.StatusKeyScreenPassed, "screening -> screen_passed"},
		{model.StatusKeyOfferSent, model.StatusKeyOfferAccepted, "offer_sent -> offer_accepted"},
	}

	for _, tt := range allowed {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if err != nil {
				t.Errorf("unexpected error for transition %q -> %q: %v", tt.from, tt.to, err)
			}
		})
	}
}

// TestValidateStatusKey verifies that an unknown key returns an error.
func TestValidateStatusKey(t *testing.T) {
	t.Run("unknown key returns error", func(t *testing.T) {
		err := ValidateStatusKey("nonexistent_status")
		if err == nil {
			t.Fatal("expected error for unknown status key, got nil")
		}
		var te *TransitionError
		if !errors.As(err, &te) {
			t.Fatalf("expected TransitionError, got %T: %v", err, err)
		}
		if te.From != "nonexistent_status" {
			t.Errorf("expected From=\"nonexistent_status\", got %q", te.From)
		}
	})

	t.Run("known key returns nil", func(t *testing.T) {
		err := ValidateStatusKey(model.StatusKeyApplied)
		if err != nil {
			t.Errorf("unexpected error for known key %q: %v", model.StatusKeyApplied, err)
		}
		err = ValidateStatusKey(model.StatusKeyHired)
		if err != nil {
			t.Errorf("unexpected error for known key %q: %v", model.StatusKeyHired, err)
		}
	})
}

// TestAllowedNextStatuses verifies that AllowedNextStatuses returns the correct set.
func TestAllowedNextStatuses(t *testing.T) {
	t.Run("applied returns correct next statuses", func(t *testing.T) {
		next := AllowedNextStatuses(model.StatusKeyApplied)
		if len(next) == 0 {
			t.Fatal("expected non-empty set for applied")
		}
		if !next[model.StatusKeyViewed] {
			t.Error("expected viewed to be an allowed next status from applied")
		}
		if !next[model.StatusKeyScreenPassed] {
			t.Error("expected screen_passed to be an allowed next status from applied")
		}
		if !next[model.StatusKeyRejected] {
			t.Error("expected rejected to be an allowed next status from applied")
		}
		if !next[model.StatusKeyWithdrawn] {
			t.Error("expected withdrawn to be an allowed next status from applied")
		}
		if next[model.StatusKeyHired] {
			t.Error("did not expect hired to be an allowed next status from applied")
		}
	})

	t.Run("terminal state returns empty set", func(t *testing.T) {
		next := AllowedNextStatuses(model.StatusKeyHired)
		if len(next) != 0 {
			t.Errorf("expected empty set for terminal state hired, got %d entries", len(next))
		}

		next = AllowedNextStatuses(model.StatusKeyRejected)
		if len(next) != 0 {
			t.Errorf("expected empty set for terminal state rejected, got %d entries", len(next))
		}

		next = AllowedNextStatuses(model.StatusKeyWithdrawn)
		if len(next) != 0 {
			t.Errorf("expected empty set for terminal state withdrawn, got %d entries", len(next))
		}

		next = AllowedNextStatuses(model.StatusKeyOfferRejected)
		if len(next) != 0 {
			t.Errorf("expected empty set for terminal state offer_rejected, got %d entries", len(next))
		}
	})

	t.Run("unknown state returns empty set", func(t *testing.T) {
		next := AllowedNextStatuses("nonexistent")
		if len(next) != 0 {
			t.Errorf("expected empty set for unknown state, got %d entries", len(next))
		}
	})
}
