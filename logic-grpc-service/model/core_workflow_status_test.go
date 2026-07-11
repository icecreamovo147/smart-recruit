package model

import "testing"

func TestCoreWorkflowStatusBaseline(t *testing.T) {
	if got := DefaultStatusKey(); got != StatusKeyApplied {
		t.Fatalf("DefaultStatusKey() = %q, want %q", got, StatusKeyApplied)
	}

	for _, key := range []string{
		StatusKeyRejected,
		StatusKeyWithdrawn,
		StatusKeyOfferRejected,
		StatusKeyHired,
	} {
		if !CanReapply(key) {
			t.Fatalf("CanReapply(%q) = false, want true", key)
		}
	}

	for _, key := range []string{
		StatusKeyApplied,
		StatusKeyViewed,
		StatusKeyScreenPassed,
		StatusKeyInterviewPending,
		StatusKeyOfferPending,
		StatusKeyOfferSent,
		StatusKeyOfferAccepted,
	} {
		if CanReapply(key) {
			t.Fatalf("CanReapply(%q) = true, want false for active workflow state", key)
		}
	}
}

func TestCandidateAndHRStatusLabelsStaySeparated(t *testing.T) {
	candidateRejected := CandidateStatusLabels[StatusKeyRejected]
	hrRejected := HRStatusLabels[StatusKeyRejected]
	if candidateRejected == "" || hrRejected == "" {
		t.Fatalf("missing rejected labels: candidate=%q hr=%q", candidateRejected, hrRejected)
	}
	if candidateRejected == hrRejected {
		t.Fatalf("candidate rejected label should remain less specific than HR label: %q", candidateRejected)
	}

	candidateWithdrawn := CandidateStatusLabels[StatusKeyWithdrawn]
	hrWithdrawn := HRStatusLabels[StatusKeyWithdrawn]
	if candidateWithdrawn == "" || hrWithdrawn == "" {
		t.Fatalf("missing withdrawn labels: candidate=%q hr=%q", candidateWithdrawn, hrWithdrawn)
	}
	if candidateWithdrawn == hrWithdrawn {
		t.Fatalf("withdrawn labels should remain audience-specific: candidate=%q hr=%q", candidateWithdrawn, hrWithdrawn)
	}
}
