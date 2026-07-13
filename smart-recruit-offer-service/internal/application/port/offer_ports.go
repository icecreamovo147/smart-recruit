package port

import (
	"context"
	"time"

	"smart-recruit-offer-service/internal/domain/model"
)

type Permission string

const (
	PermissionOfferManage         Permission = "offer.manage"
	PermissionOfferRead           Permission = "offer.read"
	PermissionOfferSend           Permission = "offer.send"
	PermissionOfferDecisionManage Permission = "offer.decision.manage"
)

type Authorizer interface {
	VerifyActor(ctx context.Context, actorID int64) error
	Authorize(ctx context.Context, actorID int64, permission Permission) error
	CanManageApplication(ctx context.Context, actorID int64, applicationID int64) error
	CanReadApplication(ctx context.Context, actorID int64, applicationID int64) error
}

type ApplicationSnapshot struct {
	ApplicationID   int64
	CandidateUserID int64
	JobID           int64
	JobTitle        string
	StatusKey       model.ApplicationStatus
}

type ApplicationSnapshotReader interface {
	GetApplicationSnapshot(ctx context.Context, applicationID int64) (*ApplicationSnapshot, error)
}

type LifecycleTransitionCommand struct {
	ApplicationID     int64
	FromStatus        model.ApplicationStatus
	ToStatus          model.ApplicationStatus
	ActorUserID       int64
	ActorAccountType  string
	Reason            string
	CloseCurrentRound bool
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
