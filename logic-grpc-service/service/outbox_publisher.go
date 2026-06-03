package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/mq"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

const (
	defaultPollInterval = 5 * time.Second
	defaultBatchSize    = 50
	maxBackoff          = 10 * time.Minute
	backoffBase         = 5 * time.Second
	defaultLockTimeout  = 2 * time.Minute
	publishTimeout      = 10 * time.Second
)

type OutboxPublisher struct {
	repo         *repository.OutboxRepo
	mqConn       *mq.Conn
	pollInterval time.Duration
	batchSize    int
	lockTimeout  time.Duration
	workerID     string
	notifyCh     chan struct{}
}

func NewOutboxPublisher(repo *repository.OutboxRepo, mqConn *mq.Conn) *OutboxPublisher {
	return &OutboxPublisher{
		repo:         repo,
		mqConn:       mqConn,
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
		lockTimeout:  defaultLockTimeout,
		workerID:     newWorkerID(),
		notifyCh:     make(chan struct{}, 1),
	}
}

func NewUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (p *OutboxPublisher) WriteEvent(ctx context.Context, eventType, aggregateType string, aggregateID uint64, routingKey string, payload any) error {
	event, err := buildOutboxEvent(eventType, aggregateType, aggregateID, routingKey, payload)
	if err != nil {
		return err
	}
	return p.repo.Create(ctx, event)
}

func (p *OutboxPublisher) WriteEventTx(tx *gorm.DB, eventType, aggregateType string, aggregateID uint64, routingKey string, payload any) error {
	event, err := buildOutboxEvent(eventType, aggregateType, aggregateID, routingKey, payload)
	if err != nil {
		return err
	}
	return p.repo.CreateWithTx(tx, event)
}

func buildOutboxEvent(eventType, aggregateType string, aggregateID uint64, routingKey string, payload any) (*model.EventOutbox, error) {
	eventID := NewUUID()
	payloadJSON, err := marshalEventPayload(eventID, payload)
	if err != nil {
		return nil, err
	}
	return &model.EventOutbox{
		EventID:       eventID,
		EventType:     eventType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		RoutingKey:    routingKey,
		Payload:       string(payloadJSON),
		Status:        model.EventOutboxStatusPending,
	}, nil
}

func marshalEventPayload(eventID string, payload any) ([]byte, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	if err := json.Unmarshal(payloadJSON, &object); err != nil {
		return payloadJSON, nil
	}
	object["event_id"] = eventID
	return json.Marshal(object)
}

// Signal wakes the publish loop immediately so that events written to the
// outbox table are published to RabbitMQ without waiting for the next tick.
// It is a no-op when the notify channel is already full (one signal covers
// all pending events — the poll loop drains everything with ClaimPending).
func (p *OutboxPublisher) Signal() {
	select {
	case p.notifyCh <- struct{}{}:
	default:
	}
}

// Start begins the publish loop. Runs until ctx is cancelled.
func (p *OutboxPublisher) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(p.pollInterval)
		defer ticker.Stop()
		// Poll immediately on start in case events were written before
		// the publisher was started.
		p.poll(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.poll(ctx)
			case <-p.notifyCh:
				p.poll(ctx)
			}
		}
	}()
}

func (p *OutboxPublisher) poll(ctx context.Context) {
	events, err := p.repo.ClaimPending(ctx, p.batchSize, p.workerID, p.lockTimeout)
	if err != nil {
		logger.L().Error("outbox claim pending failed", zap.Error(err))
		return
	}
	for _, ev := range events {
		p.publishOne(ctx, ev)
	}
}

func (p *OutboxPublisher) publishOne(ctx context.Context, ev model.EventOutbox) {
	if p.mqConn == nil {
		p.markRetry(ctx, ev, "rabbitmq: not configured")
		return
	}
	publishCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	err := p.mqConn.Publish(publishCtx, ev.RoutingKey, []byte(ev.Payload))
	if err != nil {
		logger.L().Warn("outbox publish failed",
			zap.String("event_id", ev.EventID),
			zap.String("routing_key", ev.RoutingKey),
			zap.Error(err),
		)
		p.markRetry(ctx, ev, err.Error())
		return
	}
	if err := p.repo.MarkPublished(ctx, ev.ID); err != nil {
		// Message was already published to MQ — a duplicate delivery is possible.
		// The consumer must be idempotent (e.g. dedup by event_id).
		logger.L().Error("outbox mark published failed after MQ publish, duplicate delivery possible",
			zap.String("event_id", ev.EventID), zap.Error(err))
	}
}

func (p *OutboxPublisher) markRetry(ctx context.Context, ev model.EventOutbox, errMsg string) {
	backoff := time.Duration(math.Min(
		float64(backoffBase)*math.Pow(2, float64(ev.RetryCount)),
		float64(maxBackoff),
	))
	nextRetry := time.Now().Add(backoff)
	if mErr := p.repo.MarkRetryableFailure(ctx, ev.ID, errMsg, nextRetry); mErr != nil {
		logger.L().Error("outbox mark retryable failure error", zap.Error(mErr))
	}
}

func newWorkerID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown-host"
	}
	return host + "-" + NewUUID()
}

// Stop gracefully waits for in-flight publishes (no-op in this simple version).
func (p *OutboxPublisher) Stop() {
	// Graceful stop: the context cancellation in Start() handles cleanup.
	// In-flight publish state updates will complete naturally.
}
