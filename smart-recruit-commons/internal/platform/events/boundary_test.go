package events

import (
	"strings"
	"testing"
)

func TestDefaultConsumerBoundariesRequireInboxRetryDLQAndReplay(t *testing.T) {
	boundaries := DefaultConsumerBoundaries()
	if err := ValidateConsumerBoundaries(boundaries); err != nil {
		t.Fatalf("ValidateConsumerBoundaries returned error: %v", err)
	}
	seen := map[string]ConsumerBoundary{}
	for _, boundary := range boundaries {
		seen[boundary.Name] = boundary
	}
	for _, required := range []string{
		"notification-consumer",
		"email-consumer",
		"resume-parse-consumer",
		"embedding-consumer",
		"agent-run-consumer",
		"analytics-projection-consumer",
	} {
		boundary, ok := seen[required]
		if !ok {
			t.Fatalf("missing consumer boundary %q", required)
		}
		if !boundary.InboxRequired || !boundary.RetryRequired || !boundary.DLQRequired || !boundary.ReplayRequired {
			t.Fatalf("boundary %s lacks required safety flags: %+v", required, boundary)
		}
		if !strings.Contains(boundary.IdempotencyKeySource, "envelope.") {
			t.Fatalf("boundary %s idempotency must reference envelope fields: %+v", required, boundary)
		}
	}
}

func TestValidateConsumerBoundariesRejectsUnsafeBoundary(t *testing.T) {
	err := ValidateConsumerBoundaries([]ConsumerBoundary{{
		Name:                 "unsafe-consumer",
		Owner:                "notification",
		Queue:                "queue",
		SideEffect:           "write",
		IdempotencyKeySource: "business key",
		InboxRequired:        false,
		RetryRequired:        true,
		DLQRequired:          true,
		ReplayRequired:       true,
	}})
	if err == nil || !strings.Contains(err.Error(), "inbox") {
		t.Fatalf("expected inbox validation error, got %v", err)
	}
}

func TestDefaultReplayRuleRequiresEnvelopeInboxDLQAndOperator(t *testing.T) {
	rule := DefaultReplayRule()
	if err := rule.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	rule.RequiresOperator = false
	if err := rule.Validate(); err == nil {
		t.Fatal("Validate accepted replay without operator action")
	}
}
