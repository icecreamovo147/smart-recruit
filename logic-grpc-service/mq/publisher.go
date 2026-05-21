package mq

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrNotConnected = errors.New("rabbitmq: not connected")

func IsNotConnected(err error) bool {
	return errors.Is(err, ErrNotConnected)
}

func (c *Conn) Publish(ctx context.Context, routingKey string, body []byte) error {
	return c.publish(ctx, c.cfg.Exchange, routingKey, body, nil)
}

func (c *Conn) publish(ctx context.Context, exchange, routingKey string, body []byte, headers amqp.Table) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.publishLocked(ctx, exchange, routingKey, body, headers)
}

func (c *Conn) publishLocked(ctx context.Context, exchange, routingKey string, body []byte, headers amqp.Table) error {
	if c.closed || c.channel == nil || c.confirmCh == nil {
		return ErrNotConnected
	}
	if headers == nil {
		headers = amqp.Table{}
	}
	if err := c.channel.PublishWithContext(ctx,
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Headers:      headers,
			Body:         body,
		},
	); err != nil {
		return err
	}
	select {
	case confirm, ok := <-c.confirmCh:
		if !ok {
			return ErrNotConnected
		}
		if !confirm.Ack {
			return fmt.Errorf("rabbitmq: publish nacked by broker")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
