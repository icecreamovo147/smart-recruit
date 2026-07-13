package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	sharedmq "smart-recruit-commons/mq"
	"smart-recruit-notification-service/internal/application/command"
	appservice "smart-recruit-notification-service/internal/application/service"
	"smart-recruit-notification-service/internal/domain/repository"
	"smart-recruit-platform-go/logger"
)

const (
	notificationConsumerName = "notification-consumer"
	emailConsumerName        = "email-consumer"
)

type NotificationHandler interface {
	HandleNotificationMessage(ctx context.Context, msg command.NotificationMessage) error
}

type EmailHandler interface {
	HandleEmailMessage(ctx context.Context, msg command.EmailMessage) error
}

type NotificationConsumer struct {
	handler NotificationHandler
	inbox   repository.InboxRepository
}

func NewNotificationConsumer(handler NotificationHandler, inbox repository.InboxRepository) *NotificationConsumer {
	return &NotificationConsumer{handler: handler, inbox: inbox}
}

func (c *NotificationConsumer) Start(ctx context.Context, mqConn *sharedmq.Conn) error {
	if c == nil || c.handler == nil {
		return fmt.Errorf("notification handler is required")
	}
	return mqConn.Consume(ctx, mqConn.NotificationQueue(), func(ctx context.Context, body []byte) error {
		return appservice.RunWithInbox(ctx, c.inbox, notificationConsumerName, body, func() error {
			return c.handle(ctx, body)
		})
	})
}

func (c *NotificationConsumer) handle(ctx context.Context, body []byte) error {
	var payload command.NotificationMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.L().Error("notification consumer: invalid payload", zap.Error(err))
		return fmt.Errorf("invalid payload: %w", err)
	}
	return c.handler.HandleNotificationMessage(ctx, payload)
}

type EmailConsumer struct {
	handler EmailHandler
	inbox   repository.InboxRepository
}

func NewEmailConsumer(handler EmailHandler, inbox repository.InboxRepository) *EmailConsumer {
	return &EmailConsumer{handler: handler, inbox: inbox}
}

func (c *EmailConsumer) Start(ctx context.Context, mqConn *sharedmq.Conn) error {
	if c == nil || c.handler == nil {
		return fmt.Errorf("email handler is required")
	}
	return mqConn.Consume(ctx, mqConn.EmailQueue(), func(ctx context.Context, body []byte) error {
		return appservice.RunWithInbox(ctx, c.inbox, emailConsumerName, body, func() error {
			return c.handle(ctx, body)
		})
	})
}

func (c *EmailConsumer) handle(ctx context.Context, body []byte) error {
	var payload command.EmailMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.L().Error("email consumer: invalid payload", zap.Error(err))
		return fmt.Errorf("invalid payload: %w", err)
	}
	return c.handler.HandleEmailMessage(ctx, payload)
}
