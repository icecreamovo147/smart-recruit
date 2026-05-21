package mq

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultExchange          = "recruitment.events"
	defaultDLXExchange       = "recruitment.events.dlx"
	defaultRetryExchange     = "recruitment.events.retry"
	defaultNotificationQueue = "recruitment.notification.create"
	defaultResumeParseQueue  = "recruitment.resume.parse"
	notificationRoutingKey   = "notification.create"
	resumeParseRoutingKey    = "resume.parse"
	retryHeader              = "x-retry-count"
)

type Config struct {
	URL               string
	Exchange          string
	DLXExchange       string
	RetryExchange     string
	NotificationQueue string
	ResumeParseQueue  string
	PrefetchCount     int
	MaxRetries        int
	RetryDelay        time.Duration
}

type queueBinding struct {
	name       string
	routingKey string
}

type consumerRegistration struct {
	ctx        context.Context
	queue      string
	routingKey string
	handler    Handler
}

type Conn struct {
	cfg       Config
	mu        sync.Mutex
	conn      *amqp.Connection
	channel   *amqp.Channel
	confirmCh chan amqp.Confirmation
	consumers map[string]consumerRegistration
	closed    bool
}

func DefaultConfig(url string) Config {
	return Config{URL: url}
}

func New(cfg Config) (*Conn, error) {
	cfg = cfg.withDefaults()
	c := &Conn{
		cfg:       cfg,
		consumers: make(map[string]consumerRegistration),
	}
	err := c.Reconnect()
	return c, err
}

func (cfg Config) withDefaults() Config {
	if cfg.URL == "" {
		cfg.URL = "amqp://guest:guest@127.0.0.1:5672/"
	}
	if cfg.Exchange == "" {
		cfg.Exchange = defaultExchange
	}
	if cfg.DLXExchange == "" {
		cfg.DLXExchange = defaultDLXExchange
	}
	if cfg.RetryExchange == "" {
		cfg.RetryExchange = defaultRetryExchange
	}
	if cfg.NotificationQueue == "" {
		cfg.NotificationQueue = defaultNotificationQueue
	}
	if cfg.ResumeParseQueue == "" {
		cfg.ResumeParseQueue = defaultResumeParseQueue
	}
	if cfg.PrefetchCount <= 0 {
		cfg.PrefetchCount = 10
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 5
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 5 * time.Second
	}
	return cfg
}

func (c *Conn) NotificationQueue() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg.NotificationQueue
}

func (c *Conn) ResumeParseQueue() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg.ResumeParseQueue
}

func (c *Conn) bindings() []queueBinding {
	return []queueBinding{
		{name: c.cfg.NotificationQueue, routingKey: notificationRoutingKey},
		{name: c.cfg.ResumeParseQueue, routingKey: resumeParseRoutingKey},
	}
}

func (c *Conn) routingKeyForQueue(queue string) string {
	for _, binding := range c.bindings() {
		if binding.name == queue {
			return binding.routingKey
		}
	}
	return queue
}

func (c *Conn) connectLocked() error {
	conn, err := amqp.Dial(c.cfg.URL)
	if err != nil {
		c.conn = nil
		c.channel = nil
		c.confirmCh = nil
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		c.conn = nil
		c.channel = nil
		c.confirmCh = nil
		return fmt.Errorf("rabbitmq channel: %w", err)
	}
	if err := c.declareTopology(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		c.conn = nil
		c.channel = nil
		c.confirmCh = nil
		return err
	}
	if err := ch.Qos(c.cfg.PrefetchCount, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		c.conn = nil
		c.channel = nil
		c.confirmCh = nil
		return fmt.Errorf("set qos: %w", err)
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		c.conn = nil
		c.channel = nil
		c.confirmCh = nil
		return fmt.Errorf("enable publisher confirms: %w", err)
	}

	c.conn = conn
	c.channel = ch
	c.confirmCh = ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	for _, consumer := range c.consumers {
		if consumer.ctx.Err() != nil {
			continue
		}
		if err := c.startConsumerLocked(consumer); err != nil {
			log.Printf("rabbitmq: restart consumer %s failed: %v", consumer.queue, err)
		}
	}
	return nil
}

func (c *Conn) declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(c.cfg.Exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(c.cfg.DLXExchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx: %w", err)
	}
	if err := ch.ExchangeDeclare(c.cfg.RetryExchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare retry exchange: %w", err)
	}

	retryTTL := int32(c.cfg.RetryDelay / time.Millisecond)
	if retryTTL <= 0 {
		retryTTL = 1000
	}
	for _, q := range c.bindings() {
		if _, err := ch.QueueDeclare(q.name, true, false, false, false, amqp.Table{
			"x-dead-letter-exchange": c.cfg.DLXExchange,
		}); err != nil {
			return fmt.Errorf("declare queue %s: %w", q.name, err)
		}
		if err := ch.QueueBind(q.name, q.routingKey, c.cfg.Exchange, false, nil); err != nil {
			return fmt.Errorf("bind queue %s: %w", q.name, err)
		}

		retryQueue := q.name + ".retry"
		if _, err := ch.QueueDeclare(retryQueue, true, false, false, false, amqp.Table{
			"x-message-ttl":             retryTTL,
			"x-dead-letter-exchange":    c.cfg.Exchange,
			"x-dead-letter-routing-key": q.routingKey,
		}); err != nil {
			return fmt.Errorf("declare retry queue %s: %w", retryQueue, err)
		}
		if err := ch.QueueBind(retryQueue, q.routingKey, c.cfg.RetryExchange, false, nil); err != nil {
			return fmt.Errorf("bind retry queue %s: %w", retryQueue, err)
		}

		dlqName := q.name + ".dlq"
		if _, err := ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare dlq %s: %w", dlqName, err)
		}
		if err := ch.QueueBind(dlqName, q.routingKey, c.cfg.DLXExchange, false, nil); err != nil {
			return fmt.Errorf("bind dlq %s: %w", dlqName, err)
		}
	}
	return nil
}

func (c *Conn) Channel() *amqp.Channel {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.channel
}

func (c *Conn) Reconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("connection closed")
	}
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.channel = nil
	c.conn = nil
	c.confirmCh = nil
	return c.connectLocked()
}

func (c *Conn) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.channel = nil
	c.conn = nil
	c.confirmCh = nil
}

func (c *Conn) NotifyClose() chan *amqp.Error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.NotifyClose(make(chan *amqp.Error, 1))
	}
	return nil
}

func (c *Conn) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *Conn) KeepAlive(ctx context.Context, reconnectInterval time.Duration) {
	if reconnectInterval <= 0 {
		reconnectInterval = 3 * time.Second
	}
	for {
		if c.IsClosed() {
			return
		}
		closeCh := c.NotifyClose()
		if closeCh == nil {
			if err := c.waitAndReconnect(ctx, reconnectInterval); err != nil {
				return
			}
			continue
		}
		select {
		case <-ctx.Done():
			return
		case _, ok := <-closeCh:
			if !ok || c.IsClosed() {
				return
			}
			if err := c.waitAndReconnect(ctx, reconnectInterval); err != nil {
				return
			}
		}
	}
}

func (c *Conn) waitAndReconnect(ctx context.Context, reconnectInterval time.Duration) error {
	timer := time.NewTimer(reconnectInterval)
	select {
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	case <-timer.C:
	}
	for {
		if c.IsClosed() {
			return fmt.Errorf("connection closed")
		}
		if err := c.Reconnect(); err == nil {
			return nil
		}
		timer := time.NewTimer(reconnectInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
