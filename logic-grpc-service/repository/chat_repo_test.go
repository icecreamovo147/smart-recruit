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

func TestUpdateUserMessageAgentSkills(t *testing.T) {
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

	if err := repo.UpdateUserMessageAgentSkills(ctx, 42, session.ID, msg.ID, `[1]`, `["候选人岗位匹配复核"]`); err != nil {
		t.Fatalf("update message agent skills failed: %v", err)
	}

	var stored model.AIChatHistory
	if err := db.First(&stored, msg.ID).Error; err != nil {
		t.Fatalf("load message failed: %v", err)
	}
	if stored.AgentSkillIDsJSON != `[1]` {
		t.Fatalf("AgentSkillIDsJSON = %q, want [1]", stored.AgentSkillIDsJSON)
	}
	if stored.AgentSkillNamesJSON != `["候选人岗位匹配复核"]` {
		t.Fatalf("AgentSkillNamesJSON = %q", stored.AgentSkillNamesJSON)
	}
}

func TestUpdateUserMessageAgentSkills_AllowsNoopUpdate(t *testing.T) {
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

	if err := repo.UpdateUserMessageAgentSkills(ctx, 42, session.ID, msg.ID, "", ""); err != nil {
		t.Fatalf("noop update message agent skills failed: %v", err)
	}
}

func TestUpdateUserMessageAgentSkills_MissingMessage(t *testing.T) {
	db := setupTestDB(t)
	repo := NewChatRepo(db)
	ctx := context.Background()

	session := &model.AIChatSession{HrID: 42, Title: "HR session"}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if err := repo.UpdateUserMessageAgentSkills(ctx, 42, session.ID, 999, "", ""); err == nil {
		t.Fatalf("expected missing message update to fail")
	}
}
