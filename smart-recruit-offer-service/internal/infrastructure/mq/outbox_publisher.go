package mq

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"smart-recruit-offer-service/internal/application/port"
	"smart-recruit-offer-service/internal/infrastructure/persistence"
	"smart-recruit-platform-go/businessclock"
)

const (
	envelopeSchemaVersion = "1.0"
	legacyOutboxProducer  = "smart-recruit-commons.outbox"
)

type outboxStore interface {
	Create(ctx context.Context, event *EventOutboxRecord) error
	CreateWithTx(tx *gorm.DB, event *EventOutboxRecord) error
}

type OutboxPublisher struct {
	outbox outboxStore
}

type GormOutboxStore struct {
	db *gorm.DB
}

func NewGormOutboxStore(db *gorm.DB) *GormOutboxStore {
	return &GormOutboxStore{db: db}
}

func (s *GormOutboxStore) Create(ctx context.Context, event *EventOutboxRecord) error {
	return s.db.WithContext(ctx).Create(event).Error
}

func (s *GormOutboxStore) CreateWithTx(tx *gorm.DB, event *EventOutboxRecord) error {
	return tx.Create(event).Error
}

func NewOutboxPublisher(outbox outboxStore) *OutboxPublisher {
	return &OutboxPublisher{outbox: outbox}
}

func (p *OutboxPublisher) Publish(ctx context.Context, message port.OutboxMessage) error {
	event, err := p.buildEvent(message)
	if err != nil {
		return err
	}
	if tx, ok := persistence.TxFromContext(ctx); ok {
		return p.outbox.CreateWithTx(tx, event)
	}
	return p.outbox.Create(ctx, event)
}

func (p *OutboxPublisher) Signal() {
}

func (p *OutboxPublisher) buildEvent(message port.OutboxMessage) (*EventOutboxRecord, error) {
	eventID := newEventID()
	payload := notificationEmailPayload{
		TenantID:            message.TenantID,
		EventID:             eventID,
		ReceiverID:          message.ReceiverID,
		ReceiverRole:        message.ReceiverRole,
		ReceiverAccountType: message.ReceiverAccountType,
		Type:                message.Type,
		Title:               message.Title,
		Content:             message.Content,
		Link:                message.Link,
		BizType:             message.BizType,
		BizID:               message.BizID,
		JobTitle:            message.JobTitle,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	envelope, err := newEnvelope(envelopeInput{
		EventID:       eventID,
		EventType:     message.EventType,
		AggregateType: message.AggregateType,
		AggregateID:   strconv.FormatInt(message.AggregateID, 10),
		OccurredAt:    businessclock.Now(),
		Producer:      legacyOutboxProducer,
		Payload:       payloadJSON,
		Metadata:      map[string]string{"routing_key": message.RoutingKey},
	})
	if err != nil {
		return nil, err
	}
	outboxPayloadJSON, err := marshalEventPayload(*envelope, payloadJSON)
	if err != nil {
		return nil, err
	}
	metadataJSON, err := json.Marshal(envelope.Metadata)
	if err != nil {
		return nil, err
	}
	record := &EventOutboxRecord{
		EventID:        eventID,
		SchemaVersion:  envelope.SchemaVersion,
		EventType:      message.EventType,
		AggregateType:  message.AggregateType,
		AggregateID:    uint64(message.AggregateID),
		RoutingKey:     message.RoutingKey,
		Producer:       envelope.Producer,
		IdempotencyKey: envelope.IdempotencyKey,
		Payload:        string(outboxPayloadJSON),
		Metadata:       string(metadataJSON),
		Status:         EventOutboxStatusPending,
	}
	if message.TenantID > 0 {
		record.TenantID = &message.TenantID
	}
	return record, nil
}

const EventOutboxStatusPending int32 = 0

type EventOutboxRecord struct {
	ID             uint64     `gorm:"primaryKey"`
	TenantID       *int64     `gorm:"column:tenant_id"`
	EventID        string     `gorm:"column:event_id"`
	SchemaVersion  string     `gorm:"column:schema_version"`
	EventType      string     `gorm:"column:event_type"`
	AggregateType  string     `gorm:"column:aggregate_type"`
	AggregateID    uint64     `gorm:"column:aggregate_id"`
	RoutingKey     string     `gorm:"column:routing_key"`
	Producer       string     `gorm:"column:producer"`
	IdempotencyKey string     `gorm:"column:idempotency_key"`
	CorrelationID  string     `gorm:"column:correlation_id"`
	CausationID    string     `gorm:"column:causation_id"`
	TraceID        string     `gorm:"column:trace_id"`
	Payload        string     `gorm:"column:payload"`
	Metadata       string     `gorm:"column:metadata"`
	Status         int32      `gorm:"column:status"`
	RetryCount     int32      `gorm:"column:retry_count"`
	NextRetryAt    *time.Time `gorm:"column:next_retry_at"`
	LastError      string     `gorm:"column:last_error"`
	LockedAt       *time.Time `gorm:"column:locked_at"`
	LockedBy       string     `gorm:"column:locked_by"`
	PublishedAt    *time.Time `gorm:"column:published_at"`
	DeadLetteredAt *time.Time `gorm:"column:dead_lettered_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (EventOutboxRecord) TableName() string { return "event_outbox" }

type envelopeInput struct {
	EventID        string
	EventType      string
	AggregateType  string
	AggregateID    string
	OccurredAt     time.Time
	Producer       string
	IdempotencyKey string
	Payload        json.RawMessage
	Metadata       map[string]string
}

type envelope struct {
	SchemaVersion  string            `json:"schema_version"`
	EventID        string            `json:"event_id"`
	EventType      string            `json:"event_type"`
	AggregateType  string            `json:"aggregate_type"`
	AggregateID    string            `json:"aggregate_id"`
	OccurredAt     time.Time         `json:"occurred_at"`
	Producer       string            `json:"producer"`
	IdempotencyKey string            `json:"idempotency_key"`
	Payload        json.RawMessage   `json:"payload"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func newEnvelope(input envelopeInput) (*envelope, error) {
	result := &envelope{
		SchemaVersion:  envelopeSchemaVersion,
		EventID:        strings.TrimSpace(input.EventID),
		EventType:      strings.TrimSpace(input.EventType),
		AggregateType:  strings.TrimSpace(input.AggregateType),
		AggregateID:    strings.TrimSpace(input.AggregateID),
		OccurredAt:     input.OccurredAt,
		Producer:       strings.TrimSpace(input.Producer),
		IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
		Payload:        append(json.RawMessage(nil), input.Payload...),
		Metadata:       cloneMetadata(input.Metadata),
	}
	if len(result.Payload) == 0 {
		result.Payload = json.RawMessage(`{}`)
	}
	if result.IdempotencyKey == "" {
		result.IdempotencyKey = strings.Join([]string{result.AggregateType, result.AggregateID, result.EventType, result.EventID}, ":")
	}
	return result, result.validate()
}

func (e envelope) validate() error {
	var missing []string
	if e.SchemaVersion != envelopeSchemaVersion {
		missing = append(missing, "schema_version")
	}
	if e.EventID == "" {
		missing = append(missing, "event_id")
	}
	if e.EventType == "" {
		missing = append(missing, "event_type")
	}
	if e.AggregateType == "" {
		missing = append(missing, "aggregate_type")
	}
	if e.AggregateID == "" {
		missing = append(missing, "aggregate_id")
	}
	if e.OccurredAt.IsZero() {
		missing = append(missing, "occurred_at")
	}
	if e.Producer == "" {
		missing = append(missing, "producer")
	}
	if e.IdempotencyKey == "" {
		missing = append(missing, "idempotency_key")
	}
	if !json.Valid(e.Payload) {
		missing = append(missing, "payload")
	}
	if len(missing) > 0 {
		return &envelopeError{fields: missing}
	}
	return nil
}

type envelopeError struct {
	fields []string
}

func (e *envelopeError) Error() string {
	return "invalid outbox envelope: " + strings.Join(e.fields, ", ")
}

func marshalEventPayload(envelope envelope, payloadJSON []byte) ([]byte, error) {
	if err := envelope.validate(); err != nil {
		return nil, err
	}
	envelopeJSON, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	var envelopeObject map[string]json.RawMessage
	if err := json.Unmarshal(envelopeJSON, &envelopeObject); err != nil {
		return nil, err
	}
	var payloadObject map[string]json.RawMessage
	if err := json.Unmarshal(payloadJSON, &payloadObject); err != nil || payloadObject == nil {
		return envelopeJSON, nil
	}
	merged := make(map[string]json.RawMessage, len(payloadObject)+len(envelopeObject))
	for key, value := range payloadObject {
		merged[key] = value
	}
	for key, value := range envelopeObject {
		merged[key] = value
	}
	return json.Marshal(merged)
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	result := make(map[string]string, len(metadata))
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey != "" {
			result[trimmedKey] = strings.TrimSpace(value)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

type notificationEmailPayload struct {
	TenantID            int64  `json:"tenant_id,omitempty"`
	EventID             string `json:"event_id"`
	ReceiverID          int64  `json:"receiver_id"`
	ReceiverRole        int32  `json:"receiver_role,omitempty"`
	ReceiverAccountType string `json:"receiver_account_type"`
	Type                string `json:"type"`
	Title               string `json:"title"`
	Content             string `json:"content"`
	Link                string `json:"link"`
	BizType             string `json:"biz_type"`
	BizID               int64  `json:"biz_id"`
	JobTitle            string `json:"job_title,omitempty"`
}

func newEventID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
