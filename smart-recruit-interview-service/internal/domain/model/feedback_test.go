package model

import "testing"

func TestNewFeedbackValidatesRecommendationAndScore(t *testing.T) {
	if _, err := NewFeedback(FeedbackDetails{
		InterviewID:    1,
		ApplicationID:  2,
		InterviewerID:  3,
		Recommendation: "not-a-real-value",
		Score:          5,
	}); err != ErrFeedbackInvalidRecommendation {
		t.Fatalf("invalid recommendation error=%v, want ErrFeedbackInvalidRecommendation", err)
	}
	if _, err := NewFeedback(FeedbackDetails{
		InterviewID:    1,
		ApplicationID:  2,
		InterviewerID:  3,
		Recommendation: "recommend",
		Score:          11,
	}); err != ErrFeedbackScoreOutOfRange {
		t.Fatalf("invalid score error=%v, want ErrFeedbackScoreOutOfRange", err)
	}
}
