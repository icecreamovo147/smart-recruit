package persistence

import (
	"context"
	"reflect"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
)

func TestNativeStoreChatOwnerRoleIsolationWithHRIDCollision(t *testing.T) {
	ctx := context.Background()
	db := newNativeStoreTestDB(t)
	store := NewNativeStore(db)
	now := time.Now().UTC()

	hrSession := aiChatSessionRecord{HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, Title: "hr", CreatedAt: now, UpdatedAt: now}
	legacyHRSession := aiChatSessionRecord{HRID: 7, OwnerRole: chatOwnerRoleLegacyHR, OwnerID: 0, Title: "legacy-hr", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)}
	candidateSession := aiChatSessionRecord{HRID: 7, OwnerRole: chatOwnerRoleCandidate, OwnerID: 7, Title: "candidate", CreatedAt: now.Add(2 * time.Second), UpdatedAt: now.Add(2 * time.Second)}
	if err := db.Create(&hrSession).Error; err != nil {
		t.Fatalf("create hr session: %v", err)
	}
	if err := db.Create(&legacyHRSession).Error; err != nil {
		t.Fatalf("create legacy hr session: %v", err)
	}
	if err := db.Create(&candidateSession).Error; err != nil {
		t.Fatalf("create candidate session: %v", err)
	}

	messages := []aiChatHistoryRecord{
		{HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, SessionID: hrSession.ID, Role: "assistant", Content: "hr-message", CreatedAt: now},
		{HRID: 7, OwnerRole: chatOwnerRoleLegacyHR, OwnerID: 0, SessionID: legacyHRSession.ID, Role: "assistant", Content: "legacy-message", CreatedAt: now.Add(time.Second)},
		{HRID: 7, OwnerRole: chatOwnerRoleCandidate, OwnerID: 7, SessionID: candidateSession.ID, Role: "assistant", Content: "candidate-message", CreatedAt: now.Add(2 * time.Second)},
	}
	if err := db.Create(&messages).Error; err != nil {
		t.Fatalf("create messages: %v", err)
	}

	hrSessions, _, err := store.ListChatSessions(ctx, chatOwnerRoleHR, 7, 1, 20)
	if err != nil {
		t.Fatalf("list hr sessions: %v", err)
	}
	if got, want := sessionTitles(hrSessions), []string{"legacy-hr", "hr"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hr session titles = %#v, want %#v", got, want)
	}

	candidateSessions, _, err := store.ListChatSessions(ctx, chatOwnerRoleCandidate, 7, 1, 20)
	if err != nil {
		t.Fatalf("list candidate sessions: %v", err)
	}
	if got, want := sessionTitles(candidateSessions), []string{"candidate"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate session titles = %#v, want %#v", got, want)
	}

	hrMessages, err := store.ListChatMessages(ctx, chatOwnerRoleHR, 7, 0, 1, 20)
	if err != nil {
		t.Fatalf("list hr messages: %v", err)
	}
	if got, want := messageContents(hrMessages), []string{"hr-message", "legacy-message"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("hr messages = %#v, want %#v", got, want)
	}

	candidateMessages, err := store.ListChatMessages(ctx, chatOwnerRoleCandidate, 7, 0, 1, 20)
	if err != nil {
		t.Fatalf("list candidate messages: %v", err)
	}
	if got, want := messageContents(candidateMessages), []string{"candidate-message"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate messages = %#v, want %#v", got, want)
	}

	hrMessagesInCandidateSession, err := store.ListChatMessages(ctx, chatOwnerRoleHR, 7, candidateSession.ID, 1, 20)
	if err != nil {
		t.Fatalf("list hr messages in candidate session: %v", err)
	}
	if len(hrMessagesInCandidateSession) != 0 {
		t.Fatalf("hr saw candidate session messages: %#v", hrMessagesInCandidateSession)
	}

	if err := store.UpdateChatSessionTitle(ctx, chatOwnerRoleHR, 7, candidateSession.ID, "leaked"); err != nil {
		t.Fatalf("hr update candidate session: %v", err)
	}
	var reloadedCandidate aiChatSessionRecord
	if err := db.First(&reloadedCandidate, candidateSession.ID).Error; err != nil {
		t.Fatalf("reload candidate session: %v", err)
	}
	if reloadedCandidate.Title != "candidate" {
		t.Fatalf("hr update changed candidate title to %q", reloadedCandidate.Title)
	}

	if err := store.DeleteChatSession(ctx, chatOwnerRoleHR, 7, candidateSession.ID); err != nil {
		t.Fatalf("hr delete candidate session: %v", err)
	}
	if err := db.First(&reloadedCandidate, candidateSession.ID).Error; err != nil {
		t.Fatalf("reload candidate session after delete attempt: %v", err)
	}
	if reloadedCandidate.DeletedAt != nil {
		t.Fatalf("hr delete marked candidate session deleted at %v", reloadedCandidate.DeletedAt)
	}
}

func TestNativeStoreWritesCompatibilityHRIDOnlyForHR(t *testing.T) {
	ctx := context.Background()
	db := newNativeStoreTestDB(t)
	store := NewNativeStore(db)

	hrSession, err := store.EnsureChatSession(ctx, chatOwnerRoleHR, 42, "hr", 0)
	if err != nil {
		t.Fatalf("ensure hr session: %v", err)
	}
	candidateSession, err := store.EnsureChatSession(ctx, chatOwnerRoleCandidate, 42, "candidate", 0)
	if err != nil {
		t.Fatalf("ensure candidate session: %v", err)
	}

	var hrSessionRecord aiChatSessionRecord
	if err := db.First(&hrSessionRecord, hrSession.ID).Error; err != nil {
		t.Fatalf("load hr session: %v", err)
	}
	if hrSessionRecord.HRID != 42 {
		t.Fatalf("hr session hr_id = %d, want 42", hrSessionRecord.HRID)
	}

	var candidateSessionRecord aiChatSessionRecord
	if err := db.First(&candidateSessionRecord, candidateSession.ID).Error; err != nil {
		t.Fatalf("load candidate session: %v", err)
	}
	if candidateSessionRecord.HRID != 0 {
		t.Fatalf("candidate session hr_id = %d, want 0", candidateSessionRecord.HRID)
	}

	hrMessage, err := store.AppendChatMessage(ctx, aiagentgrpc.ChatMessageRow{OwnerRole: chatOwnerRoleHR, OwnerID: 42, SessionID: hrSession.ID, Role: "user", Content: "hr"})
	if err != nil {
		t.Fatalf("append hr message: %v", err)
	}
	candidateMessage, err := store.AppendChatMessage(ctx, aiagentgrpc.ChatMessageRow{OwnerRole: chatOwnerRoleCandidate, OwnerID: 42, SessionID: candidateSession.ID, Role: "user", Content: "candidate"})
	if err != nil {
		t.Fatalf("append candidate message: %v", err)
	}

	var hrMessageRecord aiChatHistoryRecord
	if err := db.First(&hrMessageRecord, hrMessage.ID).Error; err != nil {
		t.Fatalf("load hr message: %v", err)
	}
	if hrMessageRecord.HRID != 42 {
		t.Fatalf("hr message hr_id = %d, want 42", hrMessageRecord.HRID)
	}

	var candidateMessageRecord aiChatHistoryRecord
	if err := db.First(&candidateMessageRecord, candidateMessage.ID).Error; err != nil {
		t.Fatalf("load candidate message: %v", err)
	}
	if candidateMessageRecord.HRID != 0 {
		t.Fatalf("candidate message hr_id = %d, want 0", candidateMessageRecord.HRID)
	}
}

func newNativeStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&aiChatSessionRecord{}, &aiChatHistoryRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func sessionTitles(rows []aiagentgrpc.ChatSessionRow) []string {
	titles := make([]string, 0, len(rows))
	for _, row := range rows {
		titles = append(titles, row.Title)
	}
	return titles
}

func messageContents(rows []aiagentgrpc.ChatMessageRow) []string {
	contents := make([]string, 0, len(rows))
	for _, row := range rows {
		contents = append(contents, row.Content)
	}
	return contents
}
