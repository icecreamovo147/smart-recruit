package service

import (
	"testing"

	"smart-recruit-interview-service/internal/domain/model"
)

func TestScheduleTransitionMatrix(t *testing.T) {
	tests := []struct {
		name           string
		current        model.ApplicationStatus
		wantTransition bool
		wantFrom       model.ApplicationStatus
		wantTo         model.ApplicationStatus
	}{
		{name: "viewed transitions", current: model.ApplicationStatusViewed, wantTransition: true, wantFrom: model.ApplicationStatusViewed, wantTo: model.ApplicationStatusInterviewPending},
		{name: "screen passed transitions", current: model.ApplicationStatusScreenPassed, wantTransition: true, wantFrom: model.ApplicationStatusScreenPassed, wantTo: model.ApplicationStatusInterviewPending},
		{name: "cancelled transitions", current: model.ApplicationStatusInterviewCancelled, wantTransition: true, wantFrom: model.ApplicationStatusInterviewCancelled, wantTo: model.ApplicationStatusInterviewPending},
		{name: "passed transitions", current: model.ApplicationStatusInterviewPassed, wantTransition: true, wantFrom: model.ApplicationStatusInterviewPassed, wantTo: model.ApplicationStatusInterviewPending},
		{name: "pending noops", current: model.ApplicationStatusInterviewPending},
		{name: "applied noops", current: model.ApplicationStatusApplied},
		{name: "screening noops", current: model.ApplicationStatusScreening},
		{name: "interviewing noops", current: model.ApplicationStatusInterviewing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transition, ok, err := ScheduleTransition(tt.current)
			if err != nil {
				t.Fatalf("ScheduleTransition returned error: %v", err)
			}
			if ok != tt.wantTransition {
				t.Fatalf("transition=%v, want %v", ok, tt.wantTransition)
			}
			if ok && (transition.From != tt.wantFrom || transition.To != tt.wantTo) {
				t.Fatalf("transition=%+v, want %s -> %s", transition, tt.wantFrom, tt.wantTo)
			}
		})
	}
}

func TestCancelAndFeedbackTransitionNoops(t *testing.T) {
	if _, ok := CancelTransition(model.ApplicationStatusOfferSent); ok {
		t.Fatal("expected cancel transition to noop outside interview states")
	}
	if transition, ok := CancelTransition(model.ApplicationStatusInterviewing); !ok || transition.To != model.ApplicationStatusInterviewCancelled {
		t.Fatalf("cancel transition=%+v ok=%v, want interview_cancelled", transition, ok)
	}
	if _, ok := FeedbackTransition(model.ApplicationStatusInterviewing); ok {
		t.Fatal("expected feedback transition to noop unless interview_pending")
	}
	if transition, ok := FeedbackTransition(model.ApplicationStatusInterviewPending); !ok || transition.To != model.ApplicationStatusInterviewing {
		t.Fatalf("feedback transition=%+v ok=%v, want interviewing", transition, ok)
	}
}
