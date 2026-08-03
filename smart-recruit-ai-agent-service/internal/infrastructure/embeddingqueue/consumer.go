package embeddingqueue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"smart-recruit-commons/mq"
)

const consumerName = "ai-agent-embedding-consumer"

type AgentSkillEmbeddingProcessor interface {
	UpsertAgentSkillVersion(ctx context.Context, versionID int64) error
}

type InboxStore interface {
	ClaimInbox(ctx context.Context, eventID, eventType, consumerName, idempotencyKey string) (recordID uint64, claimed bool, err error)
	MarkInboxProcessed(ctx context.Context, id uint64) error
	MarkInboxFailed(ctx context.Context, id uint64, errMsg string) error
}

type Consumer struct {
	mq        *mq.Conn
	store     InboxStore
	processor AgentSkillEmbeddingProcessor
	logger    *zap.Logger
}

type Options struct {
	Logger *zap.Logger
}

type envelope struct {
	EventID        string          `json:"event_id"`
	EventType      string          `json:"event_type"`
	AggregateType  string          `json:"aggregate_type"`
	AggregateID    string          `json:"aggregate_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	ObjectType     string          `json:"object_type"`
	ObjectID       int64           `json:"object_id"`
	Payload        json.RawMessage `json:"payload"`
}

type upsertPayload struct {
	ObjectType string `json:"object_type"`
	ObjectID   int64  `json:"object_id"`
}

func NewConsumer(mqConn *mq.Conn, store InboxStore, processor AgentSkillEmbeddingProcessor, opts Options) *Consumer {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Consumer{mq: mqConn, store: store, processor: processor, logger: logger}
}

func (c *Consumer) Start(ctx context.Context) error {
	if c == nil || c.mq == nil {
		return fmt.Errorf("embedding consumer requires rabbitmq connection")
	}
	if c.store == nil {
		return fmt.Errorf("embedding consumer requires inbox store")
	}
	if c.processor == nil {
		return fmt.Errorf("embedding consumer requires processor")
	}
	queue := c.mq.EmbeddingQueue()
	c.logger.Info("embedding consumer starting", zap.String("queue", queue))
	return c.mq.Consume(ctx, queue, c.handleMessage)
}

func (c *Consumer) handleMessage(ctx context.Context, body []byte) error {
	event, payload, err := decodeMessage(body)
	if err != nil {
		return err
	}
	eventID := strings.TrimSpace(event.EventID)
	if eventID == "" {
		sum := sha256.Sum256(body)
		eventID = hex.EncodeToString(sum[:])
	}
	eventType := strings.TrimSpace(event.EventType)
	if eventType == "" {
		eventType = "embedding.upsert"
	}
	idempotencyKey := strings.TrimSpace(event.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("%s:%d:%s", payload.ObjectType, payload.ObjectID, eventType)
	}
	recordID, claimed, err := c.store.ClaimInbox(ctx, eventID, eventType, consumerName, idempotencyKey)
	if err != nil {
		return fmt.Errorf("claim embedding inbox: %w", err)
	}
	if !claimed {
		return nil
	}
	if err := c.process(ctx, payload); err != nil {
		_ = c.store.MarkInboxFailed(ctx, recordID, err.Error())
		return err
	}
	if err := c.store.MarkInboxProcessed(ctx, recordID); err != nil {
		return fmt.Errorf("mark embedding inbox processed: %w", err)
	}
	return nil
}

func (c *Consumer) process(ctx context.Context, payload upsertPayload) error {
	switch strings.TrimSpace(payload.ObjectType) {
	case "agent_skill_version":
		if payload.ObjectID <= 0 {
			return fmt.Errorf("agent skill version object_id is required")
		}
		if err := c.processor.UpsertAgentSkillVersion(ctx, payload.ObjectID); err != nil {
			return fmt.Errorf("upsert agent skill version embedding: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported embedding object_type %q", payload.ObjectType)
	}
}

func decodeMessage(body []byte) (envelope, upsertPayload, error) {
	var event envelope
	if err := json.Unmarshal(body, &event); err != nil {
		return event, upsertPayload{}, fmt.Errorf("decode embedding event: %w", err)
	}
	payload := upsertPayload{ObjectType: event.ObjectType, ObjectID: event.ObjectID}
	if len(event.Payload) > 0 && string(event.Payload) != "null" {
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return event, upsertPayload{}, fmt.Errorf("decode embedding payload: %w", err)
		}
	}
	if payload.ObjectType == "" {
		payload.ObjectType = strings.TrimSpace(event.AggregateType)
	}
	if payload.ObjectID <= 0 {
		payload.ObjectID, _ = strconv.ParseInt(strings.TrimSpace(event.AggregateID), 10, 64)
	}
	if strings.TrimSpace(event.EventType) != "" && strings.TrimSpace(event.EventType) != "embedding.upsert" {
		return event, payload, fmt.Errorf("unsupported embedding event_type %q", event.EventType)
	}
	return event, payload, nil
}
