package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"logic-grpc-service/mq"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

type embeddingUpsertPayload struct {
	EventID      string `json:"event_id"`
	EventVersion string `json:"event_version"`
	EventTime    string `json:"event_time"`
	ObjectType   string `json:"object_type"`
	ObjectID     uint64 `json:"object_id"`
	ModelID      int64  `json:"model_id"`
	Text         string `json:"text"`
	TextHash     string `json:"text_hash"`
}

type EmbeddingConsumer struct {
	embedding *EmbeddingService
	inbox     *repository.InboxRepo
}

func NewEmbeddingConsumer(embedding *EmbeddingService) *EmbeddingConsumer {
	return &EmbeddingConsumer{embedding: embedding}
}

func (c *EmbeddingConsumer) WithInbox(inbox *repository.InboxRepo) *EmbeddingConsumer {
	c.inbox = inbox
	return c
}

func (c *EmbeddingConsumer) Start(ctx context.Context, mqConn *mq.Conn) error {
	return mqConn.Consume(ctx, mqConn.EmbeddingQueue(), func(ctx context.Context, body []byte) error {
		return consumeWithInbox(ctx, c.inbox, "embedding-consumer", body, func() error {
			return c.handle(ctx, body)
		})
	})
}

func (c *EmbeddingConsumer) handle(ctx context.Context, body []byte) error {
	var p embeddingUpsertPayload
	if err := json.Unmarshal(body, &p); err != nil {
		logger.L().Error("[embedding-consumer] invalid payload", zap.Error(err))
		return fmt.Errorf("invalid payload: %w", err)
	}

	if p.ObjectType == "" || p.ObjectID == 0 {
		logger.L().Warn("[embedding-consumer] missing object_type or object_id, skipping")
		return nil
	}

	scopeType, scopeID := resolveEmbeddingScope(p.ObjectType, p.ObjectID)

	_, err := c.embedding.EmbedObject(ctx, EmbedObjectInput{
		ObjectType: p.ObjectType,
		ObjectID:   p.ObjectID,
		ScopeType:  scopeType,
		ScopeID:    scopeID,
		Text:       p.Text,
	})
	if err != nil {
		logger.L().Error("[embedding-consumer] EmbedObject failed",
			zap.String("object_type", p.ObjectType),
			zap.Uint64("object_id", p.ObjectID),
			zap.String("text_hash", p.TextHash),
			zap.Error(err))
		return fmt.Errorf("embed object failed: %w", err)
	}

	logger.L().Info("[embedding-consumer] processed embedding upsert",
		zap.String("object_type", p.ObjectType),
		zap.Uint64("object_id", p.ObjectID),
		zap.String("text_hash", p.TextHash))

	return nil
}

func resolveEmbeddingScope(objectType string, objectID uint64) (string, uint64) {
	switch objectType {
	case "agent_skill":
		return "agent_skill", objectID
	case "ai_memory":
		return "ai_memory", objectID
	default:
		return "unknown", objectID
	}
}
