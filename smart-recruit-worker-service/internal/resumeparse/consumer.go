package resumeparse

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"smart-recruit-commons/mq"
	"smart-recruit-commons/resumeparser"
)

const consumerName = "resume-parse-consumer"

// Payload is the resume.parse outbox / MQ body written by recruitment-service.
type Payload struct {
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	IdempotencyKey string `json:"idempotency_key"`
	ResumeID       int64  `json:"resume_id"`
	FileType       string `json:"file_type"`
	OSSKey         string `json:"oss_key"`
}

// ObjectDownloader is the subset of oss.Storage needed for resume parse.
type ObjectDownloader interface {
	DownloadObject(ctx context.Context, ossKey string) ([]byte, error)
}

// ResumeStore persists parse results and inbox claims.
type ResumeStore interface {
	ClaimInbox(ctx context.Context, eventID, eventType, consumerName, idempotencyKey string) (recordID uint64, claimed bool, err error)
	MarkInboxProcessed(ctx context.Context, id uint64) error
	MarkInboxFailed(ctx context.Context, id uint64, errMsg string) error
	UpdateParsedText(ctx context.Context, resumeID int64, text string) error
	HasParsedText(ctx context.Context, resumeID int64) (bool, error)
}

// Consumer consumes resume.parse messages, extracts text, and updates resumes.parsed_text.
type Consumer struct {
	mq       *mq.Conn
	store    ResumeStore
	storage  ObjectDownloader
	registry *resumeparser.Registry
	logger   *zap.Logger
}

type Options struct {
	Logger   *zap.Logger
	Registry *resumeparser.Registry
}

func NewConsumer(mqConn *mq.Conn, store ResumeStore, storage ObjectDownloader, opts Options) *Consumer {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	registry := opts.Registry
	if registry == nil {
		registry = resumeparser.DefaultRegistry
	}
	return &Consumer{
		mq:       mqConn,
		store:    store,
		storage:  storage,
		registry: registry,
		logger:   logger,
	}
}

// Start registers the MQ consumer. It returns after registration; processing runs until ctx cancel.
func (c *Consumer) Start(ctx context.Context) error {
	if c == nil || c.mq == nil {
		return fmt.Errorf("resume parse consumer requires rabbitmq connection")
	}
	if c.store == nil {
		return fmt.Errorf("resume parse consumer requires store")
	}
	if c.storage == nil {
		return fmt.Errorf("resume parse consumer requires object storage")
	}
	queue := c.mq.ResumeParseQueue()
	c.logger.Info("resume parse consumer starting", zap.String("queue", queue))
	return c.mq.Consume(ctx, queue, func(ctx context.Context, body []byte) error {
		return c.handleMessage(ctx, body)
	})
}

func (c *Consumer) handleMessage(ctx context.Context, body []byte) error {
	payload, err := decodePayload(body)
	if err != nil {
		// Permanent bad payload: ack via returning error still retries; invalid JSON
		// should not infinite-retry. Treat as non-retryable by logging and returning nil
		// only if we can classify; for safety return error so MQ DLQ path can engage after max retries.
		c.logger.Error("resume parse consumer: invalid payload", zap.Error(err))
		return err
	}
	identity := deriveIdentity(payload, body)
	recordID, claimed, err := c.store.ClaimInbox(ctx, identity.eventID, identity.eventType, consumerName, identity.idempotencyKey)
	if err != nil {
		return fmt.Errorf("claim inbox: %w", err)
	}
	if !claimed {
		c.logger.Info("resume parse consumer: skip already processed event",
			zap.String("event_id", identity.eventID),
			zap.Int64("resume_id", payload.ResumeID),
		)
		return nil
	}

	if err := c.process(ctx, payload); err != nil {
		_ = c.store.MarkInboxFailed(ctx, recordID, err.Error())
		return err
	}
	if err := c.store.MarkInboxProcessed(ctx, recordID); err != nil {
		return fmt.Errorf("mark inbox processed: %w", err)
	}
	return nil
}

func (c *Consumer) process(ctx context.Context, payload Payload) error {
	if payload.ResumeID <= 0 {
		return fmt.Errorf("resume_id is required")
	}
	if strings.TrimSpace(payload.OSSKey) == "" {
		return fmt.Errorf("oss_key is required")
	}
	fileType := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(payload.FileType)), ".")
	if fileType == "" {
		return fmt.Errorf("file_type is required")
	}

	// Idempotent success path: already has text (e.g. concurrent worker or redelivery).
	if has, err := c.store.HasParsedText(ctx, payload.ResumeID); err == nil && has {
		c.logger.Info("resume parse consumer: resume already has parsed text",
			zap.Int64("resume_id", payload.ResumeID),
		)
		return nil
	}

	start := time.Now()
	data, err := c.storage.DownloadObject(ctx, payload.OSSKey)
	if err != nil {
		return fmt.Errorf("download resume object: %w", err)
	}
	c.logger.Info("resume parse consumer: object downloaded",
		zap.Int64("resume_id", payload.ResumeID),
		zap.String("file_type", fileType),
		zap.Int("size_bytes", len(data)),
		zap.Duration("cost", time.Since(start)),
	)

	parser, err := c.registry.GetParser(fileType)
	if err != nil {
		return fmt.Errorf("unsupported resume format %s: use PDF or DOCX", strings.ToUpper(fileType))
	}
	if len(data) >= 4 && !resumeparser.ValidateMagicBytes(fileType, data[:min(len(data), 8)]) {
		return fmt.Errorf("file magic bytes do not match declared format %s", strings.ToUpper(fileType))
	}

	parseStart := time.Now()
	text, err := parser.ExtractText(ctx, data)
	if err != nil {
		return fmt.Errorf("extract text: %w", err)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("no usable text extracted from %s file", strings.ToUpper(fileType))
	}

	// Bound stored text size (parser already limits; keep defense in depth).
	if runes := []rune(text); len(runes) > 200_000 {
		text = string(runes[:200_000])
	}

	if err := c.store.UpdateParsedText(ctx, payload.ResumeID, text); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("resume %d not found", payload.ResumeID)
		}
		return fmt.Errorf("update parsed_text: %w", err)
	}

	c.logger.Info("resume parse consumer: extraction completed",
		zap.Int64("resume_id", payload.ResumeID),
		zap.String("file_type", fileType),
		zap.Int("text_chars", len([]rune(text))),
		zap.Duration("parse_cost", time.Since(parseStart)),
		zap.Duration("total_cost", time.Since(start)),
	)
	return nil
}

type inboxIdentity struct {
	eventID        string
	eventType      string
	idempotencyKey string
}

func deriveIdentity(payload Payload, body []byte) inboxIdentity {
	eventID := strings.TrimSpace(payload.EventID)
	if eventID == "" {
		sum := sha256.Sum256(body)
		eventID = "body_sha256:" + hex.EncodeToString(sum[:])
	}
	eventType := strings.TrimSpace(payload.EventType)
	if eventType == "" {
		eventType = "resume.parse"
	}
	idempotencyKey := strings.TrimSpace(payload.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = consumerName + ":" + eventID
	}
	return inboxIdentity{eventID: eventID, eventType: eventType, idempotencyKey: idempotencyKey}
}

func decodePayload(body []byte) (Payload, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return Payload{}, fmt.Errorf("invalid json: %w", err)
	}
	payload := Payload{
		EventID:        stringField(raw, "event_id"),
		EventType:      stringField(raw, "event_type"),
		IdempotencyKey: stringField(raw, "idempotency_key"),
		ResumeID:       anyToInt64(raw["resume_id"]),
		FileType:       stringField(raw, "file_type"),
		OSSKey:         stringField(raw, "oss_key"),
	}
	if payload.ResumeID <= 0 {
		return Payload{}, fmt.Errorf("resume_id missing or invalid")
	}
	return payload, nil
}

func stringField(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	v, ok := raw[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func anyToInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	case string:
		var i int64
		_, _ = fmt.Sscan(strings.TrimSpace(n), &i)
		return i
	default:
		return 0
	}
}
