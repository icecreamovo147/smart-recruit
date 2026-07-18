package model

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOfferDraftLifecycle(t *testing.T) {
	offer, err := NewDraft(DraftDetails{
		ApplicationID:   10,
		CandidateUserID: 20,
		JobID:           30,
		Title:           "Backend Engineer",
		SalaryRange:     "30k-40k",
		CreatedBy:       40,
	})
	if err != nil {
		t.Fatalf("NewDraft returned error: %v", err)
	}
	if offer.Status != OfferStatusDraft {
		t.Fatalf("status=%s, want %s", offer.Status, OfferStatusDraft)
	}

	if err := offer.ApplyDraftPatch(DraftPatch{Title: "Senior Backend Engineer", Level: "P6"}); err != nil {
		t.Fatalf("ApplyDraftPatch returned error: %v", err)
	}
	if offer.Title != "Senior Backend Engineer" || offer.Level != "P6" {
		t.Fatalf("patch not applied: title=%q level=%q", offer.Title, offer.Level)
	}
}

func TestOfferSendSnapshotsCurrentTerms(t *testing.T) {
	offer := mustDraftOffer(t)
	if err := offer.MarkSent(77); err != nil {
		t.Fatalf("MarkSent returned error: %v", err)
	}
	if offer.Status != OfferStatusSent {
		t.Fatalf("status=%s, want %s", offer.Status, OfferStatusSent)
	}
	if offer.SentBy == nil || *offer.SentBy != 77 {
		t.Fatalf("sent_by=%v, want 77", offer.SentBy)
	}
	if !strings.Contains(offer.SentSnapshotJSON, `"title":"Offer Title"`) {
		t.Fatalf("snapshot missing title: %s", offer.SentSnapshotJSON)
	}
	if err := offer.MarkSent(77); !errors.Is(err, ErrOfferNotSendable) {
		t.Fatalf("second MarkSent error=%v, want ErrOfferNotSendable", err)
	}
}

func TestOfferWithdrawRules(t *testing.T) {
	draft := mustDraftOffer(t)
	revert, err := draft.MarkWithdrawn()
	if err != nil {
		t.Fatalf("draft MarkWithdrawn returned error: %v", err)
	}
	if revert {
		t.Fatal("draft withdrawal should not require application revert")
	}

	sent := mustDraftOffer(t)
	if err := sent.MarkSent(77); err != nil {
		t.Fatalf("MarkSent returned error: %v", err)
	}
	revert, err = sent.MarkWithdrawn()
	if err != nil {
		t.Fatalf("sent MarkWithdrawn returned error: %v", err)
	}
	if !revert {
		t.Fatal("sent withdrawal should require application revert")
	}

	if _, err := sent.MarkWithdrawn(); !errors.Is(err, ErrOfferNotWithdrawable) {
		t.Fatalf("second withdrawal error=%v, want ErrOfferNotWithdrawable", err)
	}
}

func TestOfferDecisionRules(t *testing.T) {
	now := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)
	offer := mustDraftOffer(t)
	offer.ExpiresAt = &expires
	if err := offer.MarkSent(77); err != nil {
		t.Fatalf("MarkSent returned error: %v", err)
	}
	if err := offer.MarkAccepted(999, now); !errors.Is(err, ErrCandidateMismatch) {
		t.Fatalf("wrong candidate error=%v, want ErrCandidateMismatch", err)
	}
	if err := offer.MarkAccepted(20, now); err != nil {
		t.Fatalf("MarkAccepted returned error: %v", err)
	}
	if offer.Status != OfferStatusAccepted || offer.DecidedAt == nil {
		t.Fatalf("accepted state not applied: status=%s decided_at=%v", offer.Status, offer.DecidedAt)
	}

	expired := mustDraftOffer(t)
	past := now.Add(-time.Minute)
	expired.ExpiresAt = &past
	if err := expired.MarkSent(77); err != nil {
		t.Fatalf("MarkSent returned error: %v", err)
	}
	if err := expired.MarkAccepted(20, now); !errors.Is(err, ErrOfferExpired) {
		t.Fatalf("expired accept error=%v, want ErrOfferExpired", err)
	}
}

func TestValidateApplicationTransition(t *testing.T) {
	if err := ValidateApplicationTransition(ApplicationStatusInterviewPassed, ApplicationStatusOfferPending); err != nil {
		t.Fatalf("expected interview_passed -> offer_pending to be valid: %v", err)
	}
	if err := ValidateApplicationTransition(ApplicationStatusOfferSent, ApplicationStatusOfferAccepted); err != nil {
		t.Fatalf("expected offer_sent -> offer_accepted to be valid: %v", err)
	}
	if err := ValidateApplicationTransition(ApplicationStatusOfferRejected, ApplicationStatusOfferPending); err == nil {
		t.Fatal("expected terminal offer_rejected -> offer_pending to fail")
	}
}

func mustDraftOffer(t *testing.T) *Offer {
	t.Helper()
	offer, err := NewDraft(DraftDetails{
		ApplicationID:   10,
		CandidateUserID: 20,
		JobID:           30,
		Title:           "Offer Title",
		SalaryRange:     "30k-40k",
		Level:           "P5",
		WorkLocation:    "Shanghai",
		StartDate:       "2026-08-01",
		TermsJSON:       `{"probation":"3 months"}`,
		CreatedBy:       40,
	})
	if err != nil {
		t.Fatalf("NewDraft returned error: %v", err)
	}
	return offer
}
