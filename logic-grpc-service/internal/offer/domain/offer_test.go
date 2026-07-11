package domain

import "testing"

func TestOfferStatusValuesRemainCompatible(t *testing.T) {
	t.Parallel()

	if OfferStatusDraft != "draft" {
		t.Fatalf("draft status drifted: %q", OfferStatusDraft)
	}
	if OfferStatusSent != "sent" {
		t.Fatalf("sent status drifted: %q", OfferStatusSent)
	}
	if OfferStatusAccepted != "accepted" {
		t.Fatalf("accepted status drifted: %q", OfferStatusAccepted)
	}
	if OfferStatusRejected != "rejected" {
		t.Fatalf("rejected status drifted: %q", OfferStatusRejected)
	}
}

func TestOfferEventTypeValuesRemainCompatible(t *testing.T) {
	t.Parallel()

	if OfferEventCreated != "created" {
		t.Fatalf("created event drifted: %q", OfferEventCreated)
	}
	if OfferEventSent != "sent" {
		t.Fatalf("sent event drifted: %q", OfferEventSent)
	}
	if OfferEventWithdrawn != "withdrawn" {
		t.Fatalf("withdrawn event drifted: %q", OfferEventWithdrawn)
	}
	if OfferEventRejected != "rejected" {
		t.Fatalf("rejected event drifted: %q", OfferEventRejected)
	}
}
