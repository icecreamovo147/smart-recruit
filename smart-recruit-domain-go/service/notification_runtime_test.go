package service

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-domain-go/email"
	"smart-recruit-domain-go/repository"
)

type testNotificationSender struct{}

func (testNotificationSender) Send(context.Context, string, string, string) error { return nil }
func (testNotificationSender) Close() error                                       { return nil }

func TestNewNotificationRuntimeWiresComponents(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	renderer, err := email.NewRenderer("http://localhost")
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}

	runtime := NewNotificationRuntime(NotificationRuntimeDeps{
		Users:         repository.NewUserRepo(db),
		Notifications: repository.NewNotificationRepo(db),
		Outbox:        repository.NewOutboxRepo(db),
		Inbox:         repository.NewInboxRepo(db),
		EmailLog:      repository.NewEmailLogRepo(db),
		Authz:         NewServiceAuthorizer(nil, nil),
		EmailRenderer: renderer,
		EmailSender:   testNotificationSender{},
	})

	if runtime.Notification == nil {
		t.Fatal("Notification service is nil")
	}
	if runtime.Worker == nil {
		t.Fatal("Notification worker is nil")
	}
	if runtime.OutboxPublisher == nil {
		t.Fatal("Outbox publisher is nil")
	}
	if runtime.NotificationConsumer == nil {
		t.Fatal("Notification consumer is nil")
	}
	if runtime.EmailConsumer == nil {
		t.Fatal("Email consumer is nil")
	}
}

func TestNotificationRuntimeStartRejectsNilRuntime(t *testing.T) {
	var runtime *NotificationRuntime
	errs := runtime.Start(context.Background(), nil)
	if len(errs) != 1 || errs[0].Component != "notification-runtime" {
		t.Fatalf("unexpected errors: %+v", errs)
	}
}

func TestNotificationRuntimeStartRejectsNilMQ(t *testing.T) {
	runtime := &NotificationRuntime{}
	errs := runtime.Start(context.Background(), nil)
	if len(errs) != 1 || errs[0].Component != "notification-runtime-mq" {
		t.Fatalf("unexpected errors: %+v", errs)
	}
}
