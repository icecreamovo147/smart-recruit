package query

import "smart-recruit-interview-service/internal/domain/model"

type GetInterview struct {
	UserID      int64
	InterviewID int64
}

type ListInterviewers struct {
	HRID     int64
	Page     int32
	PageSize int32
	Keyword  string
}

type ListApplicationInterviews struct {
	HRID          int64
	ApplicationID int64
}

type ListMyInterviews struct {
	InterviewerID int64
	Status        model.InterviewStatus
}

type ListCandidateInterviews struct {
	UserID int64
}

type GetFeedback struct {
	InterviewerID int64
	InterviewID   int64
}
