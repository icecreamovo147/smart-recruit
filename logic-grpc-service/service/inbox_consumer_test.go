package service

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

func TestConsumeWithInboxSkipsProcessedDuplicate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.EventInbox{}); err != nil {
		t.Fatalf("migrate inbox: %v", err)
	}
	inbox := repository.NewInboxRepo(db)
	ctx := context.Background()
	body := []byte(`{"event_id":"evt-service-inbox","event_type":"notification.create"}`)
	calls := 0

	for attempt := 0; attempt < 2; attempt++ {
		err := consumeWithInbox(ctx, inbox, "notification-consumer", body, func() error {
			calls++
			return nil
		})
		if err != nil {
			t.Fatalf("consume attempt %d: %v", attempt, err)
		}
	}
	if calls != 1 {
		t.Fatalf("handler calls=%d, want 1", calls)
	}
}

func TestInboxIdentityFallsBackToBodyHash(t *testing.T) {
	identity := inboxIdentity("embedding-consumer", []byte(`{"object_id":42}`))
	if identity.EventID == "" || identity.EventID[:12] != "body_sha256:" {
		t.Fatalf("unexpected fallback event id: %+v", identity)
	}
	if identity.IdempotencyKey == "" {
		t.Fatalf("missing idempotency key: %+v", identity)
	}
}
