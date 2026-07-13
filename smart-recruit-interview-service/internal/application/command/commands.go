package command

import "time"

type ScheduleInterview struct {
	HRID            int64
	ApplicationID   int64
	InterviewerID   int64
	RoundNo         int32
	Title           string
	Mode            string
	MeetingURL      string
	Location        string
	DurationMinutes int32
	CandidateNote   string
	InternalNote    string
	ScheduledAt     *time.Time
}

type UpdateInterview struct {
	HRID            int64
	InterviewID     int64
	Title           string
	Mode            string
	MeetingURL      string
	Location        string
	DurationMinutes int32
	CandidateNote   string
	InternalNote    string
	ScheduledAt     *time.Time
}

type CancelInterview struct {
	HRID         int64
	InterviewID  int64
	CancelReason string
}

type BatchCancelInterviews struct {
	HRID          int64
	ApplicationID int64
	CancelReason  string
}

type SubmitFeedback struct {
	InterviewerID       int64
	InterviewID         int64
	ApplicationID       int64
	Recommendation      string
	Score               int32
	DimensionScoresJSON string
	Comments            string
}
