package application

import (
	"context"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

// InterviewRepository is the Interview-owned schedule, task, and feedback persistence port.
type InterviewRepository interface {
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	Create(ctx context.Context, s *model.InterviewSchedule) error
	CreateWithTx(ctx context.Context, tx *gorm.DB, s *model.InterviewSchedule) error
	Update(ctx context.Context, s *model.InterviewSchedule) error
	UpdateWithTx(ctx context.Context, tx *gorm.DB, s *model.InterviewSchedule) error
	GetByID(ctx context.Context, id int64) (*repository.InterviewWithDetailsRow, error)
	GetModelByID(ctx context.Context, id int64) (*model.InterviewSchedule, error)
	GetMaxRoundNo(ctx context.Context, applicationID int64) (int32, error)
	ListByApplication(ctx context.Context, applicationID int64) ([]repository.InterviewWithDetailsRow, error)
	ListByInterviewer(ctx context.Context, interviewerID int64, status string) ([]repository.InterviewWithDetailsRow, error)
	ListByCandidate(ctx context.Context, userID int64) ([]repository.InterviewWithDetailsRow, error)
	CancelPendingByApplication(ctx context.Context, tx *gorm.DB, applicationID int64, reason string) error
	CreateFeedback(ctx context.Context, f *model.InterviewFeedback) error
	CreateFeedbackWithTx(ctx context.Context, tx *gorm.DB, f *model.InterviewFeedback) error
	GetFeedbackByInterview(ctx context.Context, interviewID int64) (*model.InterviewFeedback, error)
	GetFeedbackByInterviewAndInterviewer(ctx context.Context, interviewID, interviewerID int64) (*model.InterviewFeedback, error)
	HasFeedback(ctx context.Context, interviewID int64) (bool, error)
	FeedbackExistsByInterviewer(ctx context.Context, interviewID, interviewerID int64) (bool, error)
	ListFeedbackByInterviews(ctx context.Context, interviewIDs []int64) ([]model.InterviewFeedback, error)
}
