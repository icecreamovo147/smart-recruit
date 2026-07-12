package domain

// InterviewID identifies an Interview-owned schedule aggregate.
type InterviewID int64

// InterviewerID identifies the staff user assigned to an interview.
type InterviewerID int64

// ApplicationID identifies the recruitment application linked by an interview.
type ApplicationID int64

// InterviewStatus is the current persisted interview lifecycle state.
type InterviewStatus string

const (
	InterviewStatusPending   InterviewStatus = "pending"
	InterviewStatusScheduled InterviewStatus = "scheduled"
	InterviewStatusCanceled  InterviewStatus = "canceled"
	InterviewStatusCompleted InterviewStatus = "completed"
)

// FeedbackRecommendation is the current persisted interviewer recommendation state.
type FeedbackRecommendation string

const (
	FeedbackRecommendationPositive FeedbackRecommendation = "positive"
	FeedbackRecommendationNegative FeedbackRecommendation = "negative"
	FeedbackRecommendationPending  FeedbackRecommendation = "pending"
)
