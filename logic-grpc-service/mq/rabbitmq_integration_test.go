package mq

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRabbitMQPublishConsumeIntegration(t *testing.T) {
	url := os.Getenv("RABBITMQ_TEST_URL")
	if url == "" {
		t.Skip("set RABBITMQ_TEST_URL to run RabbitMQ integration test")
	}
	suffix := time.Now().Format("20060102150405.000000000")
	cfg := Config{
		URL:               url,
		Exchange:          "test.recruitment.events." + suffix,
		DLXExchange:       "test.recruitment.events.dlx." + suffix,
		RetryExchange:     "test.recruitment.events.retry." + suffix,
		NotificationQueue: "test.recruitment.notification." + suffix,
		ResumeParseQueue:  "test.recruitment.resume." + suffix,
		PrefetchCount:     1,
		MaxRetries:        1,
		RetryDelay:        100 * time.Millisecond,
	}
	conn, err := New(cfg)
	if err != nil {
		t.Fatalf("connect rabbitmq: %v", err)
	}
	defer conn.Close()
	defer cleanupTestTopology(t, conn, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	delivered := make(chan string, 1)
	if err := conn.Consume(ctx, conn.NotificationQueue(), func(ctx context.Context, body []byte) error {
		delivered <- string(body)
		return nil
	}); err != nil {
		t.Fatalf("start consumer: %v", err)
	}

	const body = `{"ok":true}`
	if err := conn.Publish(ctx, notificationRoutingKey, []byte(body)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case got := <-delivered:
		if got != body {
			t.Fatalf("delivered body = %s, want %s", got, body)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for delivery")
	}
}

func cleanupTestTopology(t *testing.T, conn *Conn, cfg Config) {
	t.Helper()
	ch := conn.Channel()
	if ch == nil {
		return
	}
	_, _ = ch.QueueDelete(cfg.NotificationQueue, false, false, false)
	_, _ = ch.QueueDelete(cfg.NotificationQueue+".retry", false, false, false)
	_, _ = ch.QueueDelete(cfg.NotificationQueue+".dlq", false, false, false)
	_, _ = ch.QueueDelete(cfg.ResumeParseQueue, false, false, false)
	_, _ = ch.QueueDelete(cfg.ResumeParseQueue+".retry", false, false, false)
	_, _ = ch.QueueDelete(cfg.ResumeParseQueue+".dlq", false, false, false)
	_ = ch.ExchangeDelete(cfg.Exchange, false, false)
	_ = ch.ExchangeDelete(cfg.RetryExchange, false, false)
	_ = ch.ExchangeDelete(cfg.DLXExchange, false, false)
}
