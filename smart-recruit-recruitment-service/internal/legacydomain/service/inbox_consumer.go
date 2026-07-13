package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"smart-recruit-platform-go/logger"
	"smart-recruit-recruitment-service/internal/legacydomain/repository"
)

type inboxConsumerPayload struct {
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	Type           string `json:"type"`
	IdempotencyKey string `json:"idempotency_key"`
}

func consumeWithInbox(ctx context.Context, inbox *repository.InboxRepo, consumerName string, body []byte, handle func() error) error {
	if inbox == nil {
		return handle()
	}
	identity := inboxIdentity(consumerName, body)
	record, claimed, err := inbox.Claim(ctx, repository.InboxClaim{
		EventID:        identity.EventID,
		EventType:      identity.EventType,
		ConsumerName:   consumerName,
		IdempotencyKey: identity.IdempotencyKey,
	})
	if err != nil {
		return err
	}
	if !claimed {
		logger.L().Info("inbox duplicate event skipped",
			zap.String("consumer", consumerName),
			zap.String("event_id", identity.EventID))
		return nil
	}
	if err := handle(); err != nil {
		if markErr := inbox.MarkFailed(ctx, record.ID, err.Error()); markErr != nil {
			logger.L().Warn("inbox mark failed error",
				zap.String("consumer", consumerName),
				zap.String("event_id", identity.EventID),
				zap.Error(markErr))
		}
		return err
	}
	if err := inbox.MarkProcessed(ctx, record.ID); err != nil {
		return fmt.Errorf("inbox mark processed: %w", err)
	}
	return nil
}

type inboxEventIdentity struct {
	EventID        string
	EventType      string
	IdempotencyKey string
}

func inboxIdentity(consumerName string, body []byte) inboxEventIdentity {
	var payload inboxConsumerPayload
	_ = json.Unmarshal(body, &payload)
	eventID := strings.TrimSpace(payload.EventID)
	if eventID == "" {
		sum := sha256.Sum256(body)
		eventID = "body_sha256:" + hex.EncodeToString(sum[:])
	}
	eventType := strings.TrimSpace(payload.EventType)
	if eventType == "" {
		eventType = strings.TrimSpace(payload.Type)
	}
	idempotencyKey := strings.TrimSpace(payload.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = consumerName + ":" + eventID
	}
	return inboxEventIdentity{
		EventID:        eventID,
		EventType:      eventType,
		IdempotencyKey: idempotencyKey,
	}
}
