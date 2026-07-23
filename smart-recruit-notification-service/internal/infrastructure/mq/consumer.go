package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	sharedmq "smart-recruit-commons/mq"
	"smart-recruit-notification-service/internal/application/command"
	appservice "smart-recruit-notification-service/internal/application/service"
	"smart-recruit-notification-service/internal/domain/repository"
	"smart-recruit-platform-go/logger"
	platformmetadata "smart-recruit-platform-go/metadata"
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
		normalized, err := normalizeMessagePayload(body)
		if err != nil {
			logger.L().Error("notification consumer: invalid payload", zap.Error(err))
			return fmt.Errorf("invalid payload: %w", err)
		}
		ctx = tenantContextFromPayload(ctx, normalized)
		return appservice.RunWithInbox(ctx, c.inbox, notificationConsumerName, normalized, func() error {
			return c.handleNormalized(ctx, normalized)
		})
	})
}

func (c *NotificationConsumer) handle(ctx context.Context, body []byte) error {
	normalized, err := normalizeMessagePayload(body)
	if err != nil {
		logger.L().Error("notification consumer: invalid payload", zap.Error(err))
		return fmt.Errorf("invalid payload: %w", err)
	}
	return c.handleNormalized(ctx, normalized)
}

func (c *NotificationConsumer) handleNormalized(ctx context.Context, body []byte) error {
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
		normalized, err := normalizeMessagePayload(body)
		if err != nil {
			logger.L().Error("email consumer: invalid payload", zap.Error(err))
			return fmt.Errorf("invalid payload: %w", err)
		}
		ctx = tenantContextFromPayload(ctx, normalized)
		return appservice.RunWithInbox(ctx, c.inbox, emailConsumerName, normalized, func() error {
			return c.handleNormalized(ctx, normalized)
		})
	})
}

func (c *EmailConsumer) handle(ctx context.Context, body []byte) error {
	normalized, err := normalizeMessagePayload(body)
	if err != nil {
		logger.L().Error("email consumer: invalid payload", zap.Error(err))
		return fmt.Errorf("invalid payload: %w", err)
	}
	return c.handleNormalized(ctx, normalized)
}

func (c *EmailConsumer) handleNormalized(ctx context.Context, body []byte) error {
	var payload command.EmailMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.L().Error("email consumer: invalid payload", zap.Error(err))
		return fmt.Errorf("invalid payload: %w", err)
	}
	return c.handler.HandleEmailMessage(ctx, payload)
}

func normalizeMessagePayload(body []byte) ([]byte, error) {
	payload, root, nested, err := payloadObject(body)
	if err != nil {
		return nil, err
	}
	if payload == nil {
		payload = root
	}
	if nested {
		promoteRootField(root, payload, "event_id")
		promoteRootField(root, payload, "idempotency_key")
		promoteRootField(root, payload, "event_type")
		promoteRootField(root, payload, "tenant_id")
		if _, ok := payload["event_type"]; !ok {
			if rawType, ok := root["type"]; ok {
				payload["event_type"] = rawType
			}
		}
	}
	applyLegacyAliases(payload)
	return json.Marshal(payload)
}

func tenantContextFromPayload(ctx context.Context, body []byte) context.Context {
	var envelope struct {
		TenantID int64 `json:"tenant_id"`
	}
	if json.Unmarshal(body, &envelope) == nil && envelope.TenantID > 0 {
		return platformmetadata.WithTenantActor(ctx, platformmetadata.TenantContext{TenantID: envelope.TenantID, AccountType: "service", ClientApp: "service"})
	}
	return ctx
}

func payloadObject(body []byte) (map[string]json.RawMessage, map[string]json.RawMessage, bool, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, nil, false, err
	}
	if len(root) == 0 {
		return nil, root, false, nil
	}
	raw, ok := root["payload"]
	if !ok || isJSONNull(raw) {
		return root, root, false, nil
	}
	payload, ok, err := rawObject(raw)
	if err != nil {
		return nil, nil, false, err
	}
	if !ok {
		return root, root, false, nil
	}
	return payload, root, true, nil
}

func rawObject(raw json.RawMessage) (map[string]json.RawMessage, bool, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err == nil {
		return payload, true, nil
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, false, nil
	}
	encoded = strings.TrimSpace(encoded)
	if encoded == "" || !strings.HasPrefix(encoded, "{") {
		return nil, false, nil
	}
	if err := json.Unmarshal([]byte(encoded), &payload); err != nil {
		return nil, false, err
	}
	return payload, true, nil
}

func promoteRootField(root, payload map[string]json.RawMessage, key string) {
	if _, ok := payload[key]; ok {
		return
	}
	if raw, ok := root[key]; ok && !isJSONNull(raw) {
		payload[key] = raw
	}
}

func applyLegacyAliases(payload map[string]json.RawMessage) {
	copyAlias(payload, "receiver_id", "user_id")
	copyAlias(payload, "receiver_account_type", "account_type")
	copyAlias(payload, "type", "category")
	copyAlias(payload, "biz_type", "related_type")
	copyAlias(payload, "biz_id", "related_id")
	copyAlias(payload, "job_title", "related_title")
}

func copyAlias(payload map[string]json.RawMessage, canonical, legacy string) {
	if _, ok := payload[canonical]; ok {
		return
	}
	if raw, ok := payload[legacy]; ok && !isJSONNull(raw) {
		payload[canonical] = raw
	}
}

func isJSONNull(raw json.RawMessage) bool {
	return strings.EqualFold(strings.TrimSpace(string(raw)), "null")
}
