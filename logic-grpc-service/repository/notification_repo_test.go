package repository

import (
	"context"
	"testing"

	"logic-grpc-service/model"
)

func TestNotificationCreateOnceByEventID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNotificationRepo(db)
	ctx := context.Background()
	eventID := "evt-notification-1"

	n := &model.Notification{
		EventID:      &eventID,
		ReceiverID:   1,
		ReceiverRole: 1,
		Type:         "application_approved",
		Title:        "投递进展更新",
		Content:      "通过筛选",
		BizType:      "application",
		BizID:        10,
	}
	created, err := repo.CreateOnceWithResult(ctx, n)
	if err != nil || !created {
		t.Fatalf("first notification should be created, created=%v err=%v", created, err)
	}

	dup := *n
	dup.ID = 0
	created, err = repo.CreateOnceWithResult(ctx, &dup)
	if err != nil {
		t.Fatalf("duplicate event id should be ignored without error: %v", err)
	}
	if created {
		t.Fatalf("duplicate event id should not create another row")
	}

	var count int64
	db.Model(&model.Notification{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly one notification, got %d", count)
	}
}

func TestNotificationCreateOnceByBusinessKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewNotificationRepo(db)
	ctx := context.Background()
	eventID1 := "evt-notification-2"
	eventID2 := "evt-notification-3"

	base := model.Notification{
		EventID:      &eventID1,
		ReceiverID:   2,
		ReceiverRole: 2,
		Type:         "new_application",
		Title:        "新的岗位投递",
		Content:      "候选人已投递",
		BizType:      "application",
		BizID:        20,
	}
	created, err := repo.CreateOnceWithResult(ctx, &base)
	if err != nil || !created {
		t.Fatalf("first notification should be created, created=%v err=%v", created, err)
	}

	dup := base
	dup.ID = 0
	dup.EventID = &eventID2
	created, err = repo.CreateOnceWithResult(ctx, &dup)
	if err != nil {
		t.Fatalf("duplicate business key should be ignored without error: %v", err)
	}
	if created {
		t.Fatalf("duplicate business key should not create another row")
	}
}
