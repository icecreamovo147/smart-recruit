package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
	platformlogger "smart-recruit-platform-go/logger"
	"smart-recruit-proto/recruitment/pb"
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
	usage := &pb.ContextUsageInfo{ModelId: 8, ContextWindowTokens: 128000, PromptTokensEstimated: 321, Stage: "model_preview"}
	if err := store.UpdateChatSessionContextModel(ctx, chatOwnerRoleHR, 7, candidateSession.ID, 8, usage); err != nil {
		t.Fatalf("update candidate context model through HR scope: %v", err)
	}
	if err := db.First(&reloadedCandidate, candidateSession.ID).Error; err != nil {
		t.Fatalf("reload candidate session after context model update: %v", err)
	}
	if reloadedCandidate.SelectedModelID != 0 || reloadedCandidate.LatestContextUsageJSON != nil {
		t.Fatalf("HR context model update leaked into candidate session: %#v", reloadedCandidate)
	}
	if err := store.UpdateChatSessionContextModel(ctx, chatOwnerRoleHR, 7, hrSession.ID, 8, usage); err != nil {
		t.Fatalf("update HR context model: %v", err)
	}
	updatedHR, found, err := store.GetChatSession(ctx, chatOwnerRoleHR, 7, hrSession.ID)
	if err != nil || !found {
		t.Fatalf("reload HR session found=%v err=%v", found, err)
	}
	if updatedHR.SelectedModelID != 8 || updatedHR.LatestContextUsage.GetPromptTokensEstimated() != 321 {
		t.Fatalf("updated HR session = %#v", updatedHR)
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

func TestNativeStoreChatMessagePersistsAgentSkillMetadata(t *testing.T) {
	ctx := context.Background()
	db := newNativeStoreTestDB(t)
	store := NewNativeStore(db)
	now := time.Now().UTC()
	session := aiChatSessionRecord{HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, Title: "hr", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}

	saved, err := store.AppendChatMessage(ctx, aiagentgrpc.ChatMessageRow{
		OwnerRole:       chatOwnerRoleHR,
		OwnerID:         7,
		SessionID:       session.ID,
		Role:            "assistant",
		Content:         "reply",
		AgentSkillIDs:   []int64{7001, 7002},
		AgentSkillNames: []string{"candidate_screen", "resume_match"},
	})
	if err != nil {
		t.Fatalf("AppendChatMessage error = %v", err)
	}
	if !reflect.DeepEqual(saved.AgentSkillIDs, []int64{7001, 7002}) || !reflect.DeepEqual(saved.AgentSkillNames, []string{"candidate_screen", "resume_match"}) {
		t.Fatalf("saved skill metadata = ids %#v names %#v", saved.AgentSkillIDs, saved.AgentSkillNames)
	}

	messages, err := store.ListChatMessages(ctx, chatOwnerRoleHR, 7, session.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListChatMessages error = %v", err)
	}
	if len(messages) != 1 || !reflect.DeepEqual(messages[0].AgentSkillIDs, []int64{7001, 7002}) || !reflect.DeepEqual(messages[0].AgentSkillNames, []string{"candidate_screen", "resume_match"}) {
		t.Fatalf("listed messages = %#v, want skill metadata round-trip", messages)
	}
}

func TestNativeStoreChatContextUsagePersistsAndRestoresLatestSnapshot(t *testing.T) {
	ctx := context.Background()
	db := newNativeStoreTestDB(t)
	store := NewNativeStore(db)
	now := time.Now().UTC()
	session := aiChatSessionRecord{HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, Title: "hr", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	usage := &pb.ContextUsageInfo{
		ModelId:                  77,
		ModelName:                "qwen-test",
		ContextWindowTokens:      8192,
		PromptTokensEstimated:    321,
		RemainingTokensEstimated: 7871,
		UsageRatio:               0.039,
		Estimated:                true,
		Source:                   "hr-agent-runtime",
		Stage:                    "pre_generation",
		Breakdown: &pb.ContextUsageBreakdown{
			RecentMessageTokens:  99,
			CurrentMessageTokens: 12,
		},
	}

	saved, err := store.AppendChatMessage(ctx, aiagentgrpc.ChatMessageRow{
		OwnerRole: chatOwnerRoleHR, OwnerID: 7, SessionID: session.ID,
		Role: "assistant", Content: "reply", ContextUsage: usage,
	})
	if err != nil {
		t.Fatalf("AppendChatMessage error = %v", err)
	}
	if !proto.Equal(saved.ContextUsage, usage) {
		t.Fatalf("saved context usage = %#v, want %#v", saved.ContextUsage, usage)
	}

	var persistedMessage aiChatHistoryRecord
	if err := db.First(&persistedMessage, saved.ID).Error; err != nil {
		t.Fatalf("load persisted message: %v", err)
	}
	if persistedMessage.ContextUsageJSON == nil || !strings.Contains(*persistedMessage.ContextUsageJSON, `"prompt_tokens_estimated"`) || strings.Contains(*persistedMessage.ContextUsageJSON, `"promptTokensEstimated"`) {
		t.Fatalf("persisted context usage JSON = %v, want proto snake_case names", persistedMessage.ContextUsageJSON)
	}

	messages, err := store.ListChatMessages(ctx, chatOwnerRoleHR, 7, session.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListChatMessages error = %v", err)
	}
	if len(messages) != 1 || !proto.Equal(messages[0].ContextUsage, usage) {
		t.Fatalf("listed messages = %#v, want context usage round-trip", messages)
	}
	gotSession, found, err := store.GetChatSession(ctx, chatOwnerRoleHR, 7, session.ID)
	if err != nil || !found {
		t.Fatalf("GetChatSession = found %v err %v", found, err)
	}
	if !proto.Equal(gotSession.LatestContextUsage, usage) {
		t.Fatalf("latest context usage = %#v, want %#v", gotSession.LatestContextUsage, usage)
	}

	if _, err := store.AppendChatMessage(ctx, aiagentgrpc.ChatMessageRow{
		OwnerRole: chatOwnerRoleHR, OwnerID: 7, SessionID: session.ID,
		Role: "user", Content: "follow-up",
	}); err != nil {
		t.Fatalf("append user message: %v", err)
	}
	gotSession, found, err = store.GetChatSession(ctx, chatOwnerRoleHR, 7, session.ID)
	if err != nil || !found || !proto.Equal(gotSession.LatestContextUsage, usage) {
		t.Fatalf("latest context usage after user message = %#v, found %v err %v; want unchanged", gotSession.LatestContextUsage, found, err)
	}
}

func TestNativeStoreChatContextUsageHandlesNullLegacyAndCorruptJSON(t *testing.T) {
	core, captured := observer.New(zapcore.WarnLevel)
	previousLogger := platformlogger.L()
	platformlogger.Set(zap.New(core))
	t.Cleanup(func() { platformlogger.Set(previousLogger) })

	ctx := context.Background()
	db := newNativeStoreTestDB(t)
	store := NewNativeStore(db)
	now := time.Now().UTC()
	legacyJSON := `{"modelId":"88","promptTokensEstimated":42,"estimated":true}`
	session := aiChatSessionRecord{
		HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, Title: "legacy",
		LatestContextUsageJSON: &legacyJSON, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	emptyJSON := ""
	privacyMarker := "PRIVATE_CONTEXT_USAGE_MARKER_candidate@example.invalid"
	corruptJSON := `{"` + privacyMarker + `":"private context snapshot"}`
	privateMessage := "PRIVATE_MESSAGE_BODY_do-not-log"
	rows := []aiChatHistoryRecord{
		{HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, SessionID: session.ID, Role: "user", Content: "null usage", CreatedAt: now},
		{HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, SessionID: session.ID, Role: "assistant", Content: "empty usage", ContextUsageJSON: &emptyJSON, CreatedAt: now.Add(time.Second)},
		{HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, SessionID: session.ID, Role: "assistant", Content: privateMessage, ContextUsageJSON: &corruptJSON, CreatedAt: now.Add(2 * time.Second)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("create history: %v", err)
	}

	gotSession, found, err := store.GetChatSession(ctx, chatOwnerRoleHR, 7, session.ID)
	if err != nil || !found {
		t.Fatalf("GetChatSession = found %v err %v", found, err)
	}
	if gotSession.LatestContextUsage.GetModelId() != 88 || gotSession.LatestContextUsage.GetPromptTokensEstimated() != 42 {
		t.Fatalf("legacy camelCase latest usage = %#v", gotSession.LatestContextUsage)
	}
	messages, err := store.ListChatMessages(ctx, chatOwnerRoleHR, 7, session.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListChatMessages error = %v", err)
	}
	if len(messages) != 3 || messages[0].ContextUsage != nil || messages[1].ContextUsage != nil || messages[2].ContextUsage != nil {
		t.Fatalf("messages = %#v, want all invalid/empty snapshots degraded to nil", messages)
	}
	if messages[2].Content != privateMessage {
		t.Fatalf("corrupt snapshot hid message body: %#v", messages[2])
	}

	entries := captured.All()
	if len(entries) != 1 {
		t.Fatalf("warning entries = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	for key, want := range map[string]any{
		"record_type":    "chat_message",
		"record_id":      rows[2].ID,
		"session_id":     session.ID,
		"error_code":     "context_usage_parse_failed",
		"error_category": "invalid_json",
	} {
		if got := fields[key]; !reflect.DeepEqual(got, want) {
			t.Fatalf("warning field %q = %#v, want %#v", key, got, want)
		}
	}
	if len(fields) != 5 {
		t.Fatalf("warning fields = %#v, want only structured safe fields", fields)
	}
	encodedFields, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("marshal warning fields: %v", err)
	}
	logged := entries[0].Message + string(encodedFields)
	for _, forbidden := range []string{privateMessage, corruptJSON, privacyMarker} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("warning leaked forbidden content %q: %s", forbidden, logged)
		}
	}
}

func TestNativeStoreAppendChatMessageRollsBackWhenSessionUpdateFails(t *testing.T) {
	ctx := context.Background()
	db := newNativeStoreTestDB(t)
	store := NewNativeStore(db)
	now := time.Now().UTC().Add(-time.Hour)
	latestContextUsageJSON := `{"model_id":"41","prompt_tokens_estimated":120}`
	session := aiChatSessionRecord{
		HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, Title: "hr",
		LastMessagePreview: "existing preview", MessageCount: 3,
		LatestContextUsageJSON: &latestContextUsageJSON, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	var before aiChatSessionRecord
	if err := db.First(&before, session.ID).Error; err != nil {
		t.Fatalf("load session before append: %v", err)
	}
	if err := db.Exec(`CREATE TRIGGER fail_chat_session_update
		BEFORE UPDATE ON ai_chat_sessions
		BEGIN
			SELECT RAISE(ABORT, 'forced session update failure');
		END`).Error; err != nil {
		t.Fatalf("create session update failure trigger: %v", err)
	}

	_, err := store.AppendChatMessage(ctx, aiagentgrpc.ChatMessageRow{
		OwnerRole: chatOwnerRoleHR, OwnerID: 7, SessionID: session.ID,
		Role: "assistant", Content: "new preview must roll back",
		ContextUsage: &pb.ContextUsageInfo{ModelId: 99, PromptTokensEstimated: 456},
	})
	if err == nil {
		t.Fatal("AppendChatMessage error = nil, want session update failure")
	}

	var messageCount int64
	if err := db.Model(&aiChatHistoryRecord{}).Where("session_id = ?", session.ID).Count(&messageCount).Error; err != nil {
		t.Fatalf("count messages after failed append: %v", err)
	}
	if messageCount != 0 {
		t.Fatalf("messages after failed append = %d, want 0", messageCount)
	}
	var after aiChatSessionRecord
	if err := db.First(&after, session.ID).Error; err != nil {
		t.Fatalf("load session after failed append: %v", err)
	}
	if after.MessageCount != before.MessageCount || after.LastMessagePreview != before.LastMessagePreview ||
		!reflect.DeepEqual(after.LatestContextUsageJSON, before.LatestContextUsageJSON) || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("session changed after rolled-back append: before=%#v after=%#v", before, after)
	}
}

func TestNativeStoreListRecentChatMessagesReturnsLatestInChronologicalOrder(t *testing.T) {
	ctx := context.Background()
	db := newNativeStoreTestDB(t)
	store := NewNativeStore(db)
	now := time.Now().UTC()
	session := aiChatSessionRecord{HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, Title: "hr", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	for index := 1; index <= 25; index++ {
		row := aiChatHistoryRecord{
			HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, SessionID: session.ID,
			Role: "user", Content: fmt.Sprintf("message-%02d", index), CreatedAt: now.Add(time.Duration(index) * time.Second),
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create history %d: %v", index, err)
		}
	}

	messages, err := store.ListRecentChatMessages(ctx, chatOwnerRoleHR, 7, session.ID, 20)
	if err != nil {
		t.Fatalf("ListRecentChatMessages error = %v", err)
	}
	if len(messages) != 20 || messages[0].Content != "message-06" || messages[19].Content != "message-25" {
		t.Fatalf("recent messages = %#v, want message-06 through message-25", messageContents(messages))
	}
}

func TestNativeStoreGetChatSessionOwnerIsolation(t *testing.T) {
	ctx := context.Background()
	db := newNativeStoreTestDB(t)
	store := NewNativeStore(db)
	now := time.Now().UTC()

	hrSession := aiChatSessionRecord{HRID: 7, OwnerRole: chatOwnerRoleHR, OwnerID: 7, Title: "hr", CreatedAt: now, UpdatedAt: now}
	legacyHRSession := aiChatSessionRecord{HRID: 7, OwnerRole: chatOwnerRoleLegacyHR, OwnerID: 0, Title: "legacy-hr", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)}
	candidateCollisionSession := aiChatSessionRecord{HRID: 7, OwnerRole: chatOwnerRoleCandidate, OwnerID: 7, Title: "candidate-collision", CreatedAt: now.Add(2 * time.Second), UpdatedAt: now.Add(2 * time.Second)}
	otherCandidateSession := aiChatSessionRecord{HRID: 0, OwnerRole: chatOwnerRoleCandidate, OwnerID: 8, Title: "other-candidate", CreatedAt: now.Add(3 * time.Second), UpdatedAt: now.Add(3 * time.Second)}
	if err := db.Create(&hrSession).Error; err != nil {
		t.Fatalf("create hr session: %v", err)
	}
	if err := db.Create(&legacyHRSession).Error; err != nil {
		t.Fatalf("create legacy hr session: %v", err)
	}
	if err := db.Create(&candidateCollisionSession).Error; err != nil {
		t.Fatalf("create candidate collision session: %v", err)
	}
	if err := db.Create(&otherCandidateSession).Error; err != nil {
		t.Fatalf("create other candidate session: %v", err)
	}

	tests := []struct {
		name      string
		ownerRole int32
		ownerID   int64
		sessionID int64
		wantFound bool
		wantTitle string
	}{
		{name: "hr current session", ownerRole: chatOwnerRoleHR, ownerID: 7, sessionID: hrSession.ID, wantFound: true, wantTitle: "hr"},
		{name: "hr legacy session", ownerRole: chatOwnerRoleHR, ownerID: 7, sessionID: legacyHRSession.ID, wantFound: true, wantTitle: "legacy-hr"},
		{name: "hr blocked from candidate collision", ownerRole: chatOwnerRoleHR, ownerID: 7, sessionID: candidateCollisionSession.ID, wantFound: false},
		{name: "candidate own collision session", ownerRole: chatOwnerRoleCandidate, ownerID: 7, sessionID: candidateCollisionSession.ID, wantFound: true, wantTitle: "candidate-collision"},
		{name: "candidate blocked from hr same id", ownerRole: chatOwnerRoleCandidate, ownerID: 7, sessionID: hrSession.ID, wantFound: false},
		{name: "candidate blocked from legacy hr", ownerRole: chatOwnerRoleCandidate, ownerID: 7, sessionID: legacyHRSession.ID, wantFound: false},
		{name: "candidate blocked from other candidate", ownerRole: chatOwnerRoleCandidate, ownerID: 7, sessionID: otherCandidateSession.ID, wantFound: false},
		{name: "missing session", ownerRole: chatOwnerRoleHR, ownerID: 7, sessionID: 999999, wantFound: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, found, err := store.GetChatSession(ctx, tt.ownerRole, tt.ownerID, tt.sessionID)
			if err != nil {
				t.Fatalf("GetChatSession returned error: %v", err)
			}
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v; row = %#v", found, tt.wantFound, row)
			}
			if !tt.wantFound {
				return
			}
			if row.ID != tt.sessionID || row.Title != tt.wantTitle {
				t.Fatalf("row = %#v, want id %d title %q", row, tt.sessionID, tt.wantTitle)
			}
		})
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

func TestNativeStoreCompleteAgentRunCanceledSetsCanceledAt(t *testing.T) {
	ctx := context.Background()
	db := newNativeStoreTestDB(t)
	store := NewNativeStore(db)

	run, replay, err := store.CreateAgentRun(ctx, aiagentgrpc.AgentRunRow{
		SessionID:       101,
		OwnerID:         77,
		ClientRequestID: "cancel-timestamps",
		Status:          "queued",
		PlanJSON:        `{"durable_request":{"message":"user asks","model_id":123}}`,
		ModelID:         123,
		AgentType:       "hr_recruiting_agent",
		AgentID:         42,
		AgentName:       "hr-data-agent",
		StartedAt:       time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	})
	if err != nil {
		t.Fatalf("CreateAgentRun returned error: %v", err)
	}
	if replay {
		t.Fatal("CreateAgentRun returned idempotent replay for fresh run")
	}
	if run.AgentID != 42 || run.AgentType != "hr_recruiting_agent" || run.AgentName != "hr-data-agent" {
		t.Fatalf("created run identity = id:%d type:%q name:%q", run.AgentID, run.AgentType, run.AgentName)
	}

	completed, found, err := store.CompleteAgentRun(ctx, 77, run.ID, "", "canceled", "", "")
	if err != nil {
		t.Fatalf("CompleteAgentRun returned error: %v", err)
	}
	if !found {
		t.Fatalf("CompleteAgentRun did not find run %d", run.ID)
	}
	if completed.Status != "canceled" || completed.CompletedAt == nil || completed.CanceledAt == nil {
		t.Fatalf("completed run = %#v, want canceled with completed_at and canceled_at", completed)
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
	if err := db.AutoMigrate(&aiChatSessionRecord{}, &aiChatHistoryRecord{}, &agentRunRecord{}); err != nil {
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
