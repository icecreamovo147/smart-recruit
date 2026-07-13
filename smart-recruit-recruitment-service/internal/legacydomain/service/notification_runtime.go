package service

import (
	"context"
	"errors"

	"smart-recruit-commons/email"
	"smart-recruit-commons/mq"
	"smart-recruit-commons/pkg/cache"
	"smart-recruit-recruitment-service/internal/legacydomain/repository"
)

const defaultNotificationWorkerConcurrency = 100

var errNilNotificationRuntime = errors.New("notification runtime is nil")
var errNilNotificationMQ = errors.New("notification runtime mq connection is nil")

type NotificationRuntime struct {
	Notification         *NotificationService
	Worker               *NotificationWorkerPool
	OutboxPublisher      *OutboxPublisher
	NotificationConsumer *NotificationConsumer
	EmailConsumer        *EmailConsumer
}

type NotificationRuntimeDeps struct {
	Users         *repository.UserRepo
	Notifications *repository.NotificationRepo
	Outbox        *repository.OutboxRepo
	Inbox         *repository.InboxRepo
	EmailLog      *repository.EmailLogRepo
	Cache         *cache.NotificationCache
	MQ            *mq.Conn
	Authz         *ServiceAuthorizer
	EmailRenderer *email.Renderer
	EmailSender   email.Sender
}

type NotificationRuntimeStartError struct {
	Component string
	Err       error
}

func NewNotificationRuntime(deps NotificationRuntimeDeps) *NotificationRuntime {
	outboxPublisher := NewOutboxPublisher(deps.Outbox, deps.MQ)
	return &NotificationRuntime{
		Notification:         NewNotificationService(deps.Notifications, deps.Cache, deps.Authz),
		Worker:               NewNotificationWorkerPool(deps.Notifications, deps.Cache, defaultNotificationWorkerConcurrency),
		OutboxPublisher:      outboxPublisher,
		NotificationConsumer: NewNotificationConsumer(deps.Notifications, deps.Cache).WithInbox(deps.Inbox),
		EmailConsumer:        NewEmailConsumer(deps.Users, deps.EmailLog, deps.EmailRenderer, deps.EmailSender).WithInbox(deps.Inbox),
	}
}

func (r *NotificationRuntime) Start(ctx context.Context, mqConn *mq.Conn) []NotificationRuntimeStartError {
	if r == nil {
		return []NotificationRuntimeStartError{{Component: "notification-runtime", Err: errNilNotificationRuntime}}
	}
	if r.OutboxPublisher != nil {
		r.OutboxPublisher.Start(ctx)
	}
	var errs []NotificationRuntimeStartError
	if mqConn == nil {
		return append(errs, NotificationRuntimeStartError{Component: "notification-runtime-mq", Err: errNilNotificationMQ})
	}
	if r.NotificationConsumer != nil {
		if err := r.NotificationConsumer.Start(ctx, mqConn); err != nil {
			errs = append(errs, NotificationRuntimeStartError{Component: "notification-consumer", Err: err})
		}
	}
	if r.EmailConsumer != nil {
		if err := r.EmailConsumer.Start(ctx, mqConn); err != nil {
			errs = append(errs, NotificationRuntimeStartError{Component: "email-consumer", Err: err})
		}
	}
	return errs
}
