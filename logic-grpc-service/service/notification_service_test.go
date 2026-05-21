package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

func TestListNotificationsCursorIncludesBusinessFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("migrate notifications: %v", err)
	}
	now := time.Now()
	rows := []model.Notification{
		{
			ReceiverID:   10,
			ReceiverRole: 1,
			Type:         "new_application",
			Title:        "新的岗位投递",
			Content:      "候选人已投递",
			BizType:      "application",
			BizID:        99,
			CreatedAt:    now,
		},
		{
			ReceiverID:   10,
			ReceiverRole: 1,
			Type:         "application_approved",
			Title:        "投递进展更新",
			Content:      "通过筛选",
			BizType:      "application",
			BizID:        100,
			CreatedAt:    now.Add(-time.Minute),
		},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("insert notifications: %v", err)
	}

	svc := NewNotificationService(repository.NewNotificationRepo(db), nil)
	resp, err := svc.ListNotifications(context.Background(), &pb.ListNotificationsRequest{
		UserId:   10,
		Role:     1,
		Page:     0,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(resp.List))
	}
	got := resp.List[0]
	if got.BizType != "application" || got.BizId != 99 {
		t.Fatalf("expected business fields to be populated, got biz_type=%q biz_id=%d", got.BizType, got.BizId)
	}
	if !resp.HasMore || resp.NextCursor == "" {
		t.Fatalf("expected cursor pagination metadata, has_more=%v next_cursor=%q", resp.HasMore, resp.NextCursor)
	}
}
