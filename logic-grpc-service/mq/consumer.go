package mq

import (
	"context"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Handler func(ctx context.Context, body []byte) error

func (c *Conn) Consume(ctx context.Context, queue string, handler Handler) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("rabbitmq: connection closed")
	}
	consumer := consumerRegistration{
		ctx:        ctx,
		queue:      queue,
		routingKey: c.routingKeyForQueue(queue),
		handler:    handler,
	}
	c.consumers[queue] = consumer
	if c.channel == nil {
		return nil
	}
	return c.startConsumerLocked(consumer)
}

func (c *Conn) startConsumerLocked(consumer consumerRegistration) error {
	msgs, err := c.channel.Consume(consumer.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", consumer.queue, err)
	}
	go c.consumeLoop(consumer, msgs)
	return nil
}

func (c *Conn) consumeLoop(consumer consumerRegistration, msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-consumer.ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			if err := consumer.handler(consumer.ctx, msg.Body); err != nil {
				c.handleFailure(consumer.ctx, consumer, msg, err)
				continue
			}
			if err := msg.Ack(false); err != nil {
				log.Printf("mq: ack failed for %s: %v", consumer.queue, err)
			}
		}
	}
}

func (c *Conn) handleFailure(ctx context.Context, consumer consumerRegistration, msg amqp.Delivery, handlerErr error) {
	retryCount := retryCountFromHeaders(msg.Headers)
	if retryCount < c.cfg.MaxRetries {
		headers := cloneHeaders(msg.Headers)
		headers[retryHeader] = retryCount + 1
		retryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := c.publish(retryCtx, c.cfg.RetryExchange, consumer.routingKey, msg.Body, headers); err != nil {
			log.Printf("mq: retry publish failed for %s: %v", consumer.queue, err)
			_ = msg.Nack(false, true)
			return
		}
		log.Printf("mq: handler error for %s, retry %d/%d: %v", consumer.queue, retryCount+1, c.cfg.MaxRetries, handlerErr)
		_ = msg.Ack(false)
		return
	}
	log.Printf("mq: handler error for %s after %d retries, dead-lettering: %v", consumer.queue, retryCount, handlerErr)
	_ = msg.Nack(false, false)
}

func cloneHeaders(headers amqp.Table) amqp.Table {
	cloned := amqp.Table{}
	for k, v := range headers {
		cloned[k] = v
	}
	return cloned
}

func retryCountFromHeaders(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	switch v := headers[retryHeader].(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}
