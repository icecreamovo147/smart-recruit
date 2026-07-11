package domain

import "testing"

func TestInterviewStatusValuesRemainCompatible(t *testing.T) {
	t.Parallel()

	if InterviewStatusPending != "pending" {
		t.Fatalf("pending status drifted: %q", InterviewStatusPending)
	}
	if InterviewStatusCanceled != "canceled" {
		t.Fatalf("canceled status drifted: %q", InterviewStatusCanceled)
	}
	if FeedbackRecommendationPositive != "positive" {
		t.Fatalf("positive recommendation drifted: %q", FeedbackRecommendationPositive)
	}
}
