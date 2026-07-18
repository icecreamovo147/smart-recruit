package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"smart-recruit-notification-service/internal/domain/repository"
)

type inboxPayload struct {
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	Type           string `json:"type"`
	IdempotencyKey string `json:"idempotency_key"`
}

type InboxIdentity struct {
	EventID        string
	EventType      string
	IdempotencyKey string
}

func RunWithInbox(ctx context.Context, inbox repository.InboxRepository, consumerName string, body []byte, handle func() error) error {
	if inbox == nil {
		return handle()
	}
	identity := DeriveInboxIdentity(consumerName, body)
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
		return nil
	}
	if err := handle(); err != nil {
		if record != nil {
			_ = inbox.MarkFailed(ctx, record.ID, err.Error())
		}
		return err
	}
	if record == nil {
		return nil
	}
	if err := inbox.MarkProcessed(ctx, record.ID); err != nil {
		return fmt.Errorf("inbox mark processed: %w", err)
	}
	return nil
}

func DeriveInboxIdentity(consumerName string, body []byte) InboxIdentity {
	var payload inboxPayload
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
	return InboxIdentity{
		EventID:        eventID,
		EventType:      eventType,
		IdempotencyKey: idempotencyKey,
	}
}
