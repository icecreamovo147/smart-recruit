package service

import (
	"strings"
	"testing"
)

func TestRequiredAgentRunStatusesCoverFR004(t *testing.T) {
	want := map[string]bool{
		"queued": true, "planning": true, "running": true, "waiting_confirmation": true,
		"cancel_requested": true, "succeeded": true, "partial": true, "failed": true, "canceled": true,
	}
	if len(RequiredAgentRunStatuses) != len(want) {
		t.Fatalf("expected %d statuses, got %d", len(want), len(RequiredAgentRunStatuses))
	}
	for _, status := range RequiredAgentRunStatuses {
		if !want[status] {
			t.Fatalf("unexpected status %q", status)
		}
		if !IsKnownAgentRunStatus(status) {
			t.Fatalf("IsKnownAgentRunStatus(%q) = false", status)
		}
	}
}

func TestValidateAgentRunTransitionAllowsLegalPaths(t *testing.T) {
	cases := []struct {
		from, to string
	}{
		{AgentRunStatusQueued, AgentRunStatusPlanning},
		{AgentRunStatusQueued, AgentRunStatusCancelRequested},
		{AgentRunStatusPlanning, AgentRunStatusRunning},
		{AgentRunStatusPlanning, AgentRunStatusWaitingConfirmation},
		{AgentRunStatusRunning, AgentRunStatusWaitingConfirmation},
		{AgentRunStatusRunning, AgentRunStatusSucceeded},
		{AgentRunStatusRunning, AgentRunStatusPartial},
		{AgentRunStatusRunning, AgentRunStatusFailed},
		{AgentRunStatusRunning, AgentRunStatusCancelRequested},
		{AgentRunStatusWaitingConfirmation, AgentRunStatusRunning},
		{AgentRunStatusWaitingConfirmation, AgentRunStatusCancelRequested},
		{AgentRunStatusCancelRequested, AgentRunStatusCanceled},
		{AgentRunStatusCancelRequested, AgentRunStatusFailed},
		{AgentRunStatusCancelRequested, AgentRunStatusPartial},
		// idempotent same-status
		{AgentRunStatusRunning, AgentRunStatusRunning},
		{AgentRunStatusCancelRequested, AgentRunStatusCancelRequested},
		{AgentRunStatusSucceeded, AgentRunStatusSucceeded},
		{AgentRunStatusCanceled, AgentRunStatusCanceled},
	}
	for _, tc := range cases {
		if err := ValidateAgentRunTransition(tc.from, tc.to); err != nil {
			t.Fatalf("expected allow %s -> %s, got %v", tc.from, tc.to, err)
		}
	}
}

func TestValidateAgentRunTransitionRejectsIllegalPaths(t *testing.T) {
	cases := []struct {
		from, to string
	}{
		{AgentRunStatusSucceeded, AgentRunStatusRunning},
		{AgentRunStatusFailed, AgentRunStatusPlanning},
		{AgentRunStatusCanceled, AgentRunStatusQueued},
		{AgentRunStatusPartial, AgentRunStatusRunning},
		{AgentRunStatusQueued, AgentRunStatusSucceeded},
		{AgentRunStatusQueued, AgentRunStatusRunning},
		{AgentRunStatusWaitingConfirmation, AgentRunStatusSucceeded},
		{AgentRunStatusCancelRequested, AgentRunStatusRunning},
		{AgentRunStatusRunning, AgentRunStatusQueued},
		{AgentRunStatusPlanning, AgentRunStatusQueued},
	}
	for _, tc := range cases {
		err := ValidateAgentRunTransition(tc.from, tc.to)
		if err == nil {
			t.Fatalf("expected reject %s -> %s", tc.from, tc.to)
		}
		if !strings.Contains(err.Error(), "illegal agent run transition") && !strings.Contains(err.Error(), "unknown") {
			t.Fatalf("unexpected error for %s -> %s: %v", tc.from, tc.to, err)
		}
		if CanTransitionAgentRun(tc.from, tc.to) {
			t.Fatalf("CanTransitionAgentRun(%s,%s) should be false", tc.from, tc.to)
		}
	}
}

func TestValidateAgentRunTransitionRejectsUnknownStatus(t *testing.T) {
	if err := ValidateAgentRunTransition("running", "bogus"); err == nil {
		t.Fatal("expected unknown to-status error")
	}
	if err := ValidateAgentRunTransition("bogus", "running"); err == nil {
		t.Fatal("expected unknown from-status error")
	}
}

func TestIsTerminalAndActiveAgentRunStatus(t *testing.T) {
	for _, status := range []string{
		AgentRunStatusSucceeded, AgentRunStatusPartial, AgentRunStatusFailed, AgentRunStatusCanceled,
	} {
		if !IsTerminalAgentRunStatus(status) {
			t.Fatalf("expected terminal %q", status)
		}
		if IsActiveAgentRunStatus(status) {
			t.Fatalf("terminal status %q must not be active", status)
		}
	}
	for _, status := range []string{
		AgentRunStatusQueued, AgentRunStatusPlanning, AgentRunStatusRunning,
		AgentRunStatusWaitingConfirmation, AgentRunStatusCancelRequested,
	} {
		if IsTerminalAgentRunStatus(status) {
			t.Fatalf("expected non-terminal %q", status)
		}
		if !IsActiveAgentRunStatus(status) {
			t.Fatalf("expected active %q", status)
		}
	}
}

func TestIsStaleOrDuplicateEventSeq(t *testing.T) {
	cases := []struct {
		last, incoming int64
		stale          bool
	}{
		{0, 1, false},
		{1, 1, true},
		{5, 4, true},
		{5, 5, true},
		{5, 6, false},
		{0, 0, true},
	}
	for _, tc := range cases {
		got := IsStaleOrDuplicateEventSeq(tc.last, tc.incoming)
		if got != tc.stale {
			t.Fatalf("IsStaleOrDuplicateEventSeq(%d,%d)=%v want %v", tc.last, tc.incoming, got, tc.stale)
		}
	}
}
