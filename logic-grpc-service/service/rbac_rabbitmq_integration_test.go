//go:build integration
// +build integration

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"gopkg.in/yaml.v3"

	"logic-grpc-service/model"
)

// rabbitMQConfig is a minimal struct for RabbitMQ connection config.
type rabbitMQConfig struct {
	RabbitMQ struct {
		URL               string `yaml:"url"`
		Exchange          string `yaml:"exchange"`
		NotificationQueue string `yaml:"notification_queue"`
	} `yaml:"rabbitmq"`
}

// loadRabbitMQURL reads RabbitMQ config from the project's config.yaml.
func loadRabbitMQURL() string {
	candidates := []string{
		"config/config.yaml",
		"../config/config.yaml",
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "config/config.yaml"))
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg rabbitMQConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}
		if cfg.RabbitMQ.URL != "" {
			return cfg.RabbitMQ.URL
		}
	}
	return ""
}

// TestIntegration_NotificationPayloadUsesAccountType verifies that:
//  1. A notification payload with receiver_account_type is serializable
//  2. The consumer path uses receiver_account_type (not numeric receiver_role)
func TestIntegration_NotificationPayloadUsesAccountType(t *testing.T) {
	url := loadRabbitMQURL()
	if url == "" {
		t.Skip("RabbitMQ config not found; set up config/config.yaml with rabbitmq.url to run integration tests")
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		t.Skipf("RabbitMQ not available: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()

	ctx := context.Background()

	// Create a test-only queue with a unique suffix
	testSuffix := fmt.Sprintf(".integration.%d", time.Now().UnixNano())
	testQueue := "test.notifications" + testSuffix

	_, err = ch.QueueDeclare(testQueue, true, false, false, false, nil)
	if err != nil {
		t.Fatalf("declare queue: %v", err)
	}
	t.Cleanup(func() {
		ch.QueueDelete(testQueue, false, false, false)
	})
	t.Logf("✅ test queue created: %s", testQueue)

	// ── Test 1: Publish notification with receiver_account_type ────────
	payload := notificationPayload{
		ReceiverID:          123,
		ReceiverAccountType: "staff",
		ReceiverRole:        2, // legacy numeric — must NOT be relied upon
		Type:                "test_notification",
		Title:               "Test",
		Content:             "Integration test notification",
		BizType:             "test",
		BizID:               1,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	err = ch.PublishWithContext(ctx, "", testQueue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	t.Log("✅ notification published with receiver_account_type='staff'")

	// ── Test 2: Consume and verify account_type is used ────────────────
	msgs, err := ch.ConsumeWithContext(ctx, testQueue, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}

	select {
	case msg := <-msgs:
		var received notificationPayload
		if err := json.Unmarshal(msg.Body, &received); err != nil {
			t.Fatalf("unmarshal consumed message: %v", err)
		}
		if received.ReceiverAccountType != "staff" {
			t.Errorf("expected receiver_account_type='staff', got %q", received.ReceiverAccountType)
		}

		// Verify consumer would use account_type (not role)
		acctType := received.ReceiverAccountType
		if acctType == "" {
			// This would be the old fallback — should NOT happen
			t.Error("receiver_account_type is empty — old numeric role fallback would trigger")
		}
		t.Logf("✅ consumer uses receiver_account_type=%q", acctType)

		// ── Verify no consumer path derives channel from numeric role ──
		n := &model.Notification{
			ReceiverID:          received.ReceiverID,
			ReceiverAccountType: received.ReceiverAccountType,
			ReceiverRole:        received.ReceiverRole,
			Type:                received.Type,
		}
		got := notifAccountType(n)
		if got != "staff" {
			t.Errorf("notifAccountType returned %q, expected 'staff'", got)
		}
		// Even with empty account_type, should default to "candidate" (not derive from role=2)
		n2 := &model.Notification{
			ReceiverID:          received.ReceiverID,
			ReceiverAccountType: "",
			ReceiverRole:        2, // should NOT be used as staff indicator
		}
		got2 := notifAccountType(n2)
		if got2 != "candidate" {
			t.Errorf("empty account_type with role=2 returned %q, expected fallback 'candidate' (numeric role MUST NOT be used)", got2)
		}
		t.Log("✅ consumer does not derive channel from numeric receiver_role")

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for consumed message")
	}

	t.Log("✅ RabbitMQ integration test passed")
}
