package repository

import (
	"context"

	"smart-recruit-interview-service/internal/domain/model"
)

type InterviewDetails struct {
	Interview             model.Interview
	ApplicationStatusKey  model.ApplicationStatus
	JobTitle              string
	CandidateUserID       int64
	CandidateName         string
	CandidatePhone        string
	InterviewerName       string
	ResumeOssKey          string
	ResumeURL             string
	HasFeedbackForRequest bool
}

type InterviewWriter interface {
	Create(ctx context.Context, interview *model.Interview) error
	Save(ctx context.Context, interview *model.Interview) error
	CreateFeedback(ctx context.Context, feedback *model.Feedback) error
	CancelActiveByApplication(ctx context.Context, applicationID int64, reason string) error
}

type InterviewRepository interface {
	InterviewWriter
	FindByID(ctx context.Context, interviewID int64) (*model.Interview, error)
	FindDetailsByID(ctx context.Context, interviewID int64) (*InterviewDetails, error)
	MaxRoundNo(ctx context.Context, applicationID int64) (int32, error)
	ListByApplication(ctx context.Context, applicationID int64) ([]InterviewDetails, error)
	ListByInterviewer(ctx context.Context, interviewerID int64, status model.InterviewStatus) ([]InterviewDetails, error)
	ListByCandidate(ctx context.Context, candidateUserID int64) ([]InterviewDetails, error)
	ListFeedbackByInterviews(ctx context.Context, interviewIDs []int64) ([]model.Feedback, error)
	FeedbackExistsByInterviewer(ctx context.Context, interviewID int64, interviewerID int64) (bool, error)
	FindFeedbackByInterviewAndInterviewer(ctx context.Context, interviewID int64, interviewerID int64) (*model.Feedback, error)
	Transaction(ctx context.Context, fn func(context.Context, InterviewWriter) error) error
}
