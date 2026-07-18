package outbox

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	sharedmq "smart-recruit-commons/mq"
)

const (
	defaultPollInterval = 5 * time.Second
	defaultBatchSize    = 50
	defaultLockTimeout  = 2 * time.Minute
	defaultMaxBackoff   = 10 * time.Minute
	defaultBackoffBase  = 5 * time.Second
	defaultPublishWait  = 10 * time.Second
	defaultMaxRetries   = 10

	StatusPending    int32 = 0
	StatusPublished  int32 = 1
	StatusDead       int32 = 2
	StatusProcessing int32 = 3
)

var (
	ErrStoreRequired     = errors.New("outbox store is required")
	ErrPublisherRequired = errors.New("outbox publisher is required")
)

type Event struct {
	ID             uint64     `gorm:"primaryKey"`
	EventID        string     `gorm:"column:event_id"`
	RoutingKey     string     `gorm:"column:routing_key"`
	Payload        string     `gorm:"column:payload;type:json"`
	Status         int32      `gorm:"column:status"`
	RetryCount     int32      `gorm:"column:retry_count"`
	NextRetryAt    *time.Time `gorm:"column:next_retry_at"`
	LastError      string     `gorm:"column:last_error"`
	LockedAt       *time.Time `gorm:"column:locked_at"`
	LockedBy       string     `gorm:"column:locked_by"`
	PublishedAt    *time.Time `gorm:"column:published_at"`
	DeadLetteredAt *time.Time `gorm:"column:dead_lettered_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

func (Event) TableName() string { return "event_outbox" }

type Store interface {
	ClaimPending(ctx context.Context, limit int, workerID string, lockTimeout time.Duration) ([]Event, error)
	MarkPublished(ctx context.Context, id uint64) error
	MarkRetryableFailure(ctx context.Context, id uint64, errMsg string, nextRetry time.Time) error
	MarkDead(ctx context.Context, id uint64, errMsg string) error
}

type Publisher interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
}

type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

func (s *GormStore) ClaimPending(ctx context.Context, limit int, workerID string, lockTimeout time.Duration) ([]Event, error) {
	if s == nil || s.db == nil {
		return nil, ErrStoreRequired
	}
	var events []Event
	if limit <= 0 {
		return events, nil
	}
	now := time.Now()
	staleBefore := now.Add(-lockTimeout)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"(status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND (locked_at IS NULL OR locked_at <= ?))",
				StatusPending, now,
				StatusProcessing, staleBefore,
			).
			Order("COALESCE(next_retry_at, created_at) ASC, id ASC").
			Limit(limit).
			Find(&events).Error; err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		ids := make([]uint64, 0, len(events))
		for _, event := range events {
			ids = append(ids, event.ID)
		}
		return tx.Model(&Event{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"status":    StatusProcessing,
				"locked_at": now,
				"locked_by": workerID,
			}).Error
	})
	return events, err
}

func (s *GormStore) MarkPublished(ctx context.Context, id uint64) error {
	if s == nil || s.db == nil {
		return ErrStoreRequired
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Event{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        StatusPublished,
			"next_retry_at": nil,
			"last_error":    "",
			"locked_at":     nil,
			"locked_by":     "",
			"published_at":  now,
		}).Error
}

func (s *GormStore) MarkRetryableFailure(ctx context.Context, id uint64, errMsg string, nextRetry time.Time) error {
	if s == nil || s.db == nil {
		return ErrStoreRequired
	}
	return s.db.WithContext(ctx).Model(&Event{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        StatusPending,
			"last_error":    truncateError(errMsg),
			"retry_count":   gorm.Expr("retry_count + 1"),
			"next_retry_at": nextRetry,
			"locked_at":     nil,
			"locked_by":     "",
		}).Error
}

func (s *GormStore) MarkDead(ctx context.Context, id uint64, errMsg string) error {
	if s == nil || s.db == nil {
		return ErrStoreRequired
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(&Event{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":           StatusDead,
			"last_error":       truncateError(errMsg),
			"retry_count":      gorm.Expr("retry_count + 1"),
			"next_retry_at":    nil,
			"locked_at":        nil,
			"locked_by":        "",
			"dead_lettered_at": now,
		}).Error
}

type MQPublisher struct {
	conn *sharedmq.Conn
}

func NewMQPublisher(conn *sharedmq.Conn) *MQPublisher {
	return &MQPublisher{conn: conn}
}

func (p *MQPublisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	if p == nil || p.conn == nil {
		return ErrPublisherRequired
	}
	return p.conn.Publish(ctx, routingKey, body)
}

type Options struct {
	PollInterval  time.Duration
	BatchSize     int
	LockTimeout   time.Duration
	PublishWait   time.Duration
	MaxRetryCount int32
	BackoffBase   time.Duration
	MaxBackoff    time.Duration
	WorkerID      string
	Logger        *zap.Logger
}

type Dispatcher struct {
	store     Store
	publisher Publisher
	options   Options
	notifyCh  chan struct{}

	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
	wg      sync.WaitGroup
}

func NewDispatcher(store Store, publisher Publisher, options Options) *Dispatcher {
	options = normalizeOptions(options)
	return &Dispatcher{
		store:     store,
		publisher: publisher,
		options:   options,
		notifyCh:  make(chan struct{}, 1),
	}
}

func (d *Dispatcher) Start(ctx context.Context) error {
	if err := d.validate(); err != nil {
		return err
	}
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("outbox dispatcher already started")
	}
	runCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	d.running = true
	d.wg.Add(1)
	d.mu.Unlock()

	go d.run(runCtx)
	return nil
}

func (d *Dispatcher) Stop(ctx context.Context) error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	cancel := d.cancel
	if !d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = false
	d.cancel = nil
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (d *Dispatcher) Signal() {
	if d == nil {
		return
	}
	select {
	case d.notifyCh <- struct{}{}:
	default:
	}
}

func (d *Dispatcher) DispatchOnce(ctx context.Context) (int, error) {
	if err := d.validate(); err != nil {
		return 0, err
	}
	events, err := d.store.ClaimPending(ctx, d.options.BatchSize, d.options.WorkerID, d.options.LockTimeout)
	if err != nil {
		return 0, fmt.Errorf("claim pending outbox events: %w", err)
	}
	var joined error
	for _, event := range events {
		if err := d.publishOne(ctx, event); err != nil {
			joined = errors.Join(joined, err)
		}
	}
	return len(events), joined
}

func (d *Dispatcher) run(ctx context.Context) {
	defer d.wg.Done()
	ticker := time.NewTicker(d.options.PollInterval)
	defer ticker.Stop()
	d.dispatchAndLog(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.dispatchAndLog(ctx)
		case <-d.notifyCh:
			d.dispatchAndLog(ctx)
		}
	}
}

func (d *Dispatcher) dispatchAndLog(ctx context.Context) {
	count, err := d.DispatchOnce(ctx)
	if err != nil {
		d.options.Logger.Error("worker outbox dispatch failed", zap.Error(err))
		return
	}
	if count > 0 {
		d.options.Logger.Info("worker outbox dispatched batch", zap.Int("count", count))
	}
}

func (d *Dispatcher) publishOne(ctx context.Context, event Event) error {
	if strings.TrimSpace(event.RoutingKey) == "" {
		return d.markRetry(ctx, event, "routing_key: empty")
	}
	publishCtx, cancel := context.WithTimeout(ctx, d.options.PublishWait)
	defer cancel()
	if err := d.publisher.Publish(publishCtx, event.RoutingKey, []byte(event.Payload)); err != nil {
		d.options.Logger.Warn("worker outbox publish failed",
			zap.String("event_id", event.EventID),
			zap.String("routing_key", event.RoutingKey),
			zap.Error(err),
		)
		return d.markRetry(ctx, event, err.Error())
	}
	if err := d.store.MarkPublished(ctx, event.ID); err != nil {
		return fmt.Errorf("mark outbox event %d published after MQ publish: %w", event.ID, err)
	}
	return nil
}

func (d *Dispatcher) markRetry(ctx context.Context, event Event, errMsg string) error {
	if event.RetryCount+1 >= d.options.MaxRetryCount {
		if err := d.store.MarkDead(ctx, event.ID, errMsg); err != nil {
			return fmt.Errorf("mark outbox event %d dead: %w", event.ID, err)
		}
		return nil
	}
	backoff := time.Duration(math.Min(
		float64(d.options.MaxBackoff),
		float64(d.options.BackoffBase)*math.Pow(2, float64(event.RetryCount)),
	))
	nextRetry := time.Now().Add(backoff)
	if err := d.store.MarkRetryableFailure(ctx, event.ID, errMsg, nextRetry); err != nil {
		return fmt.Errorf("mark outbox event %d retryable: %w", event.ID, err)
	}
	return nil
}

func (d *Dispatcher) validate() error {
	if d == nil {
		return fmt.Errorf("outbox dispatcher is required")
	}
	if d.store == nil {
		return ErrStoreRequired
	}
	if d.publisher == nil {
		return ErrPublisherRequired
	}
	return nil
}

func normalizeOptions(options Options) Options {
	if options.PollInterval <= 0 {
		options.PollInterval = defaultPollInterval
	}
	if options.BatchSize <= 0 {
		options.BatchSize = defaultBatchSize
	}
	if options.LockTimeout <= 0 {
		options.LockTimeout = defaultLockTimeout
	}
	if options.PublishWait <= 0 {
		options.PublishWait = defaultPublishWait
	}
	if options.MaxRetryCount <= 0 {
		options.MaxRetryCount = defaultMaxRetries
	}
	if options.BackoffBase <= 0 {
		options.BackoffBase = defaultBackoffBase
	}
	if options.MaxBackoff <= 0 {
		options.MaxBackoff = defaultMaxBackoff
	}
	if strings.TrimSpace(options.WorkerID) == "" {
		options.WorkerID = defaultWorkerID()
	}
	if options.Logger == nil {
		options.Logger = zap.NewNop()
	}
	return options
}

func defaultWorkerID() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		host = "unknown-host"
	}
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s-%d", host, time.Now().UnixNano())
	}
	return host + "-" + hex.EncodeToString(buf)
}

func truncateError(value string) string {
	const maxLen = 2048
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}
