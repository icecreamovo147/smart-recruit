package service

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"smart-recruit-domain-go/mq"
	"smart-recruit-platform-go/logger"
)

const (
	EmbeddingUpsertRoutingKey = "embedding.upsert"
	EmbeddingDeleteRoutingKey = "embedding.delete"
)

type EmbeddingUpsertEvent struct {
	EventID      string `json:"event_id"`
	EventVersion string `json:"event_version"`
	EventTime    string `json:"event_time"`
	ObjectType   string `json:"object_type"`
	ObjectID     uint64 `json:"object_id"`
	ModelID      int64  `json:"model_id"`
	Text         string `json:"text"`
	TextHash     string `json:"text_hash"`
}

type EmbeddingEventPublisher struct {
	mqConn *mq.Conn
}

func NewEmbeddingEventPublisher(mqConn *mq.Conn) *EmbeddingEventPublisher {
	return &EmbeddingEventPublisher{mqConn: mqConn}
}

func (p *EmbeddingEventPublisher) PublishUpsert(ctx context.Context, event EmbeddingUpsertEvent) error {
	if p == nil || p.mqConn == nil {
		logger.L().Debug("[embedding-event] publisher not available, skipping event",
			zap.String("object_type", event.ObjectType),
			zap.Uint64("object_id", event.ObjectID))
		return nil
	}

	if event.EventID == "" {
		event.EventID = NewUUID()
	}
	if event.EventVersion == "" {
		event.EventVersion = "1.0"
	}
	if event.EventTime == "" {
		event.EventTime = time.Now().UTC().Format(time.RFC3339Nano)
	}

	body, err := json.Marshal(event)
	if err != nil {
		logger.L().Error("[embedding-event] failed to marshal event", zap.Error(err))
		return err
	}

	if err := p.mqConn.Publish(ctx, EmbeddingUpsertRoutingKey, body); err != nil {
		logger.L().Error("[embedding-event] failed to publish upsert event",
			zap.String("object_type", event.ObjectType),
			zap.Uint64("object_id", event.ObjectID),
			zap.Error(err))
		return err
	}

	logger.L().Debug("[embedding-event] published upsert event",
		zap.String("event_id", event.EventID),
		zap.String("object_type", event.ObjectType),
		zap.Uint64("object_id", event.ObjectID))

	return nil
}

func (p *EmbeddingEventPublisher) PublishUpsertBestEffort(ctx context.Context, event EmbeddingUpsertEvent) {
	if err := p.PublishUpsert(ctx, event); err != nil {
		logger.L().Warn("[embedding-event] best-effort publish failed",
			zap.String("object_type", event.ObjectType),
			zap.Uint64("object_id", event.ObjectID),
			zap.Error(err))
	}
}
