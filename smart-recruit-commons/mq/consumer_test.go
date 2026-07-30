package mq

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestRetryCountFromHeaders(t *testing.T) {
	headers := amqp.Table{retryHeader: int32(3)}
	if got := retryCountFromHeaders(headers); got != 3 {
		t.Fatalf("retry count = %d, want 3", got)
	}
	if got := retryCountFromHeaders(nil); got != 0 {
		t.Fatalf("nil headers retry count = %d, want 0", got)
	}
}

func TestConsumeRetainsEmbeddingRegistrationWhileDisconnected(t *testing.T) {
	ctx := context.Background()
	conn := &Conn{
		cfg:       DefaultConfig("amqp://example").withDefaults(),
		consumers: make(map[string]consumerRegistration),
	}
	handler := func(context.Context, []byte) error { return nil }

	if err := conn.Consume(ctx, conn.cfg.EmbeddingQueue, handler); err != nil {
		t.Fatalf("Consume() returned error while disconnected: %v", err)
	}
	registration, ok := conn.consumers[conn.cfg.EmbeddingQueue]
	if !ok {
		t.Fatal("embedding consumer registration was not retained for reconnect")
	}
	if registration.ctx != ctx ||
		registration.queue != conn.cfg.EmbeddingQueue ||
		registration.routingKey != embeddingUpsertRoutingKey ||
		registration.handler == nil {
		t.Fatalf("unexpected embedding consumer registration: %#v", registration)
	}
}
