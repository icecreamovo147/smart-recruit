package runtime

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	sharedmq "smart-recruit-domain-go/mq"

	"smart-recruit-proto/recruitment/pb"
)

const ServiceName = "notification-service"

type NotificationAPI interface {
	ListNotifications(context.Context, *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error)
	UnreadNotificationCount(context.Context, *pb.UnreadNotificationCountRequest) (*pb.UnreadNotificationCountResponse, error)
	NotificationSummary(context.Context, *pb.NotificationSummaryRequest) (*pb.NotificationSummaryResponse, error)
	MarkNotificationRead(context.Context, *pb.MarkNotificationReadRequest) (*pb.CommonResponse, error)
	MarkAllNotificationsRead(context.Context, *pb.MarkAllNotificationsReadRequest) (*pb.CommonResponse, error)
}

type Deps struct {
	Notification         NotificationAPI
	MQ                   *sharedmq.Conn
	OutboxPublisher      OutboxPublisher
	NotificationConsumer Consumer
	EmailConsumer        Consumer
}

type Runtime struct {
	Notification         pb.NotificationServiceServer
	Components           Components
	mq                   *sharedmq.Conn
	outboxPublisher      OutboxPublisher
	notificationConsumer Consumer
	emailConsumer        Consumer
}

type Components struct {
	Persistence       bool
	RealtimeDelivery  bool
	OutboxPublisher   bool
	NotificationInbox bool
	EmailCoordination bool
}

var IdempotencySemantics = []string{
	"notification persistence uses CreateOnce/CreateOnceWithResult for business-key idempotency",
	"notification consumer attaches Inbox storage before consuming RabbitMQ messages",
	"email consumer attaches Inbox storage before consuming RabbitMQ email coordination messages",
	"outbox publisher claims pending events and signals dispatch after transactional writes",
	"realtime delivery invalidates unread cache and publishes Redis notification events after persistence writes",
}

type OutboxPublisher interface {
	Start(context.Context)
}

type Consumer interface {
	Start(context.Context, *sharedmq.Conn) error
}

type StartError struct {
	Component string
	Err       error
}

func New(deps Deps) (*Runtime, error) {
	if deps.Notification == nil {
		return nil, fmt.Errorf("notification api is required")
	}
	components := Components{
		Persistence:       true,
		RealtimeDelivery:  true,
		OutboxPublisher:   deps.OutboxPublisher != nil,
		NotificationInbox: deps.NotificationConsumer != nil,
		EmailCoordination: deps.EmailConsumer != nil,
	}
	if deps.OutboxPublisher != nil || deps.NotificationConsumer != nil || deps.EmailConsumer != nil {
		if err := components.Validate(); err != nil {
			return nil, err
		}
	}
	return &Runtime{
		Notification:         notificationServer{api: deps.Notification},
		Components:           components,
		mq:                   deps.MQ,
		outboxPublisher:      deps.OutboxPublisher,
		notificationConsumer: deps.NotificationConsumer,
		emailConsumer:        deps.EmailConsumer,
	}, nil
}

func (c Components) Validate() error {
	switch {
	case !c.Persistence:
		return fmt.Errorf("notification persistence component is required")
	case !c.RealtimeDelivery:
		return fmt.Errorf("notification realtime delivery component is required")
	case !c.OutboxPublisher:
		return fmt.Errorf("notification outbox publisher is required")
	case !c.NotificationInbox:
		return fmt.Errorf("notification inbox consumer is required")
	case !c.EmailCoordination:
		return fmt.Errorf("notification email coordination consumer is required")
	default:
		return nil
	}
}

func (r *Runtime) Start(ctx context.Context) []StartError {
	if r == nil {
		return []StartError{{Component: "notification-runtime", Err: fmt.Errorf("notification runtime is nil")}}
	}
	if r.outboxPublisher != nil {
		r.outboxPublisher.Start(ctx)
	}
	if r.mq == nil {
		return []StartError{{Component: "notification-runtime-mq", Err: fmt.Errorf("notification runtime mq connection is nil")}}
	}
	var errs []StartError
	if r.notificationConsumer != nil {
		if err := r.notificationConsumer.Start(ctx, r.mq); err != nil {
			errs = append(errs, StartError{Component: "notification-consumer", Err: err})
		}
	}
	if r.emailConsumer != nil {
		if err := r.emailConsumer.Start(ctx, r.mq); err != nil {
			errs = append(errs, StartError{Component: "email-consumer", Err: err})
		}
	}
	return errs
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if r == nil || r.Notification == nil {
		return fmt.Errorf("notification runtime is not initialized")
	}
	pb.RegisterNotificationServiceServer(registrar, r.Notification)
	return nil
}

type notificationServer struct {
	pb.UnimplementedNotificationServiceServer
	api NotificationAPI
}

func (s notificationServer) ListNotifications(ctx context.Context, req *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	return s.api.ListNotifications(ctx, req)
}

func (s notificationServer) UnreadNotificationCount(ctx context.Context, req *pb.UnreadNotificationCountRequest) (*pb.UnreadNotificationCountResponse, error) {
	return s.api.UnreadNotificationCount(ctx, req)
}

func (s notificationServer) NotificationSummary(ctx context.Context, req *pb.NotificationSummaryRequest) (*pb.NotificationSummaryResponse, error) {
	return s.api.NotificationSummary(ctx, req)
}

func (s notificationServer) MarkNotificationRead(ctx context.Context, req *pb.MarkNotificationReadRequest) (*pb.CommonResponse, error) {
	return s.api.MarkNotificationRead(ctx, req)
}

func (s notificationServer) MarkAllNotificationsRead(ctx context.Context, req *pb.MarkAllNotificationsReadRequest) (*pb.CommonResponse, error) {
	return s.api.MarkAllNotificationsRead(ctx, req)
}
