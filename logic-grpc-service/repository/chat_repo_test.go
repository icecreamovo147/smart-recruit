package repository

import (
	"context"
	"testing"

	"logic-grpc-service/model"
)

func TestCreateSession_FillsHROwnerFields(t *testing.T) {
	db := setupTestDB(t)
	repo := NewChatRepo(db)
	ctx := context.Background()

	session := &model.AIChatSession{HrID: 42, Title: "HR session"}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	var stored model.AIChatSession
	if err := db.First(&stored, session.ID).Error; err != nil {
		t.Fatalf("load session failed: %v", err)
	}
	if stored.OwnerRole != 2 {
		t.Fatalf("expected owner_role=2, got %d", stored.OwnerRole)
	}
	if stored.OwnerID != 42 {
		t.Fatalf("expected owner_id=42, got %d", stored.OwnerID)
	}
}

func TestAdd_FillsHROwnerFields(t *testing.T) {
	db := setupTestDB(t)
	repo := NewChatRepo(db)
	ctx := context.Background()

	session := &model.AIChatSession{HrID: 42, Title: "HR session"}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	msg := &model.AIChatHistory{
		SessionID: session.ID,
		HrID:      42,
		Role:      "user",
		Content:   "hello",
	}
	if err := repo.Add(ctx, msg); err != nil {
		t.Fatalf("add message failed: %v", err)
	}

	var stored model.AIChatHistory
	if err := db.First(&stored, msg.ID).Error; err != nil {
		t.Fatalf("load message failed: %v", err)
	}
	if stored.OwnerRole != 2 {
		t.Fatalf("expected owner_role=2, got %d", stored.OwnerRole)
	}
	if stored.OwnerID != 42 {
		t.Fatalf("expected owner_id=42, got %d", stored.OwnerID)
	}
}
