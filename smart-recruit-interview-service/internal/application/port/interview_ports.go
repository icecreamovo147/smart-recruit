package port

import (
	"context"
	"time"

	"smart-recruit-interview-service/internal/domain/model"
)

type Permission string

const (
	PermissionInterviewSchedule Permission = "interview.schedule"
	PermissionInterviewRead     Permission = "interview.read"
	PermissionFeedbackSubmit    Permission = "interview.feedback.submit"
)

type Authorizer interface {
	VerifyActor(ctx context.Context, actorID int64) error
	Authorize(ctx context.Context, actorID int64, permission Permission) error
	CanScheduleApplication(ctx context.Context, actorID int64, applicationID int64) error
	CanReadInterview(ctx context.Context, actorID int64, interviewID int64) error
}

type ApplicationSnapshot struct {
	ApplicationID    int64
	CandidateUserID  int64
	JobID            int64
	JobTitle         string
	CandidateName    string
	StatusKey        model.ApplicationStatus
	CurrentRoundNo   int32
	InterviewerScope bool
}

type ApplicationSnapshotReader interface {
	GetApplicationSnapshot(ctx context.Context, applicationID int64) (*ApplicationSnapshot, error)
}

type LifecycleTransitionCommand struct {
	ApplicationID    int64
	FromStatus       model.ApplicationStatus
	ToStatus         model.ApplicationStatus
	ActorUserID      int64
	ActorAccountType string
	Reason           string
}

type ApplicationLifecycle interface {
	ApplyTransition(ctx context.Context, command LifecycleTransitionCommand) (bool, error)
}

type OutboxMessage struct {
	EventType           string
	AggregateType       string
	AggregateID         int64
	RoutingKey          string
	ReceiverID          int64
	ReceiverRole        int32
	ReceiverAccountType string
	Type                string
	Title               string
	Content             string
	Link                string
	BizType             string
	BizID               int64
	JobTitle            string
	RecipientName       string
	InterviewDate       string
	InterviewMode       string
	InterviewLink       string
	InterviewLocation   string
}

type OutboxPublisher interface {
	Publish(ctx context.Context, message OutboxMessage) error
	Signal()
}

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}
