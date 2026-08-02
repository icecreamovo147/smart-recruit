package persistence

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

const (
	embeddingUpsertEventType  = "embedding.upsert"
	embeddingUpsertRoutingKey = "embedding.upsert"
	embeddingOutboxProducer   = "smart-recruit-ai-agent-service"
)

type embeddingOutboxRecord struct {
	ID             uint64     `gorm:"column:id;primaryKey"`
	EventID        string     `gorm:"column:event_id"`
	SchemaVersion  string     `gorm:"column:schema_version"`
	EventType      string     `gorm:"column:event_type"`
	AggregateType  string     `gorm:"column:aggregate_type"`
	AggregateID    uint64     `gorm:"column:aggregate_id"`
	RoutingKey     string     `gorm:"column:routing_key"`
	Producer       string     `gorm:"column:producer"`
	IdempotencyKey string     `gorm:"column:idempotency_key"`
	Payload        string     `gorm:"column:payload"`
	Metadata       *string    `gorm:"column:metadata"`
	Status         int32      `gorm:"column:status"`
	NextRetryAt    *time.Time `gorm:"column:next_retry_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (embeddingOutboxRecord) TableName() string { return "event_outbox" }

type embeddingUpsertEnvelope struct {
	SchemaVersion  string                 `json:"schema_version"`
	EventID        string                 `json:"event_id"`
	EventType      string                 `json:"event_type"`
	AggregateType  string                 `json:"aggregate_type"`
	AggregateID    string                 `json:"aggregate_id"`
	OccurredAt     time.Time              `json:"occurred_at"`
	Producer       string                 `json:"producer"`
	IdempotencyKey string                 `json:"idempotency_key"`
	ObjectType     string                 `json:"object_type"`
	ObjectID       int64                  `json:"object_id"`
	Payload        embeddingUpsertPayload `json:"payload"`
}

type embeddingUpsertPayload struct {
	ObjectType string `json:"object_type"`
	ObjectID   int64  `json:"object_id"`
}

func queueAgentSkillEmbeddingUpsert(tx *gorm.DB, versionID int64, compiledHash string) error {
	if tx == nil || versionID <= 0 {
		return fmt.Errorf("agent skill embedding outbox request is invalid")
	}
	eventID, err := newEmbeddingEventID()
	if err != nil {
		return err
	}
	now := time.Now()
	idempotencyKey := fmt.Sprintf("agent_skill_version:%d:embedding.upsert:%s", versionID, compiledHash)
	payload := embeddingUpsertPayload{ObjectType: "agent_skill_version", ObjectID: versionID}
	envelope := embeddingUpsertEnvelope{
		SchemaVersion:  "1.0",
		EventID:        eventID,
		EventType:      embeddingUpsertEventType,
		AggregateType:  "agent_skill_version",
		AggregateID:    strconv.FormatInt(versionID, 10),
		OccurredAt:     now,
		Producer:       embeddingOutboxProducer,
		IdempotencyKey: idempotencyKey,
		ObjectType:     payload.ObjectType,
		ObjectID:       payload.ObjectID,
		Payload:        payload,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal agent skill embedding event: %w", err)
	}
	record := embeddingOutboxRecord{
		EventID:        eventID,
		SchemaVersion:  envelope.SchemaVersion,
		EventType:      envelope.EventType,
		AggregateType:  envelope.AggregateType,
		AggregateID:    uint64(versionID),
		RoutingKey:     embeddingUpsertRoutingKey,
		Producer:       envelope.Producer,
		IdempotencyKey: envelope.IdempotencyKey,
		Payload:        string(raw),
		Status:         0,
		NextRetryAt:    &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := tx.Create(&record).Error; err != nil {
		return fmt.Errorf("queue agent skill embedding event: %w", err)
	}
	return nil
}

func newEmbeddingEventID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate embedding event id: %w", err)
	}
	return hex.EncodeToString(raw), nil
}
