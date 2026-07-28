package persistence

import (
	"context"
	"encoding/json"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-proto/recruitment/pb"
)

func TestAgentSkillVersionActivationEnforcesOwnershipAndReleaseImmutability(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&agentSkillRecord{}, &agentSkillVersionRecord{}, &platformAICapabilityVersionRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	skills := []agentSkillRecord{
		{ID: 1, Name: "skill-one", DisplayName: "Skill One", AgentType: "hr_recruiting_agent", IsEnabled: true, IsManualInvocable: true, CurrentVersionID: nullInt64From(11)},
		{ID: 2, Name: "skill-two", DisplayName: "Skill Two", AgentType: "hr_recruiting_agent", IsEnabled: true, IsManualInvocable: true, CurrentVersionID: nullInt64From(22)},
	}
	versions := []agentSkillVersionRecord{
		{ID: 11, SkillID: 1, Version: "v1", SkillMD: "one"},
		{ID: 22, SkillID: 2, Version: "v1", SkillMD: "two"},
	}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("seed skills: %v", err)
	}
	if err := db.Create(&versions).Error; err != nil {
		t.Fatalf("seed versions: %v", err)
	}
	store := &NativeStore{db: db}
	ctx := context.Background()

	foreign, err := store.ActivateAgentSkillVersion(ctx, &pb.ActivateAgentSkillVersionRequest{SkillId: 1, VersionId: 22})
	if err != nil || foreign.GetCode() != governanceBadRequest {
		t.Fatalf("foreign activation response=%#v err=%v", foreign, err)
	}
	var unchanged agentSkillRecord
	if err := db.First(&unchanged, 1).Error; err != nil || !unchanged.CurrentVersionID.Valid || unchanged.CurrentVersionID.Int64 != 11 {
		t.Fatalf("skill after foreign activation=%#v err=%v", unchanged, err)
	}

	snapshot, err := json.Marshal(PlatformAICapabilitySnapshot{
		SchemaVersion: 1,
		CapabilityKey: "ai.agent_run",
		Audience:      "enterprise",
		ConfigurationRef: PlatformAIConfigurationRefs{
			AgentSkillVersionIDs: []int64{11},
		},
	})
	if err != nil {
		t.Fatalf("marshal release: %v", err)
	}
	if err := db.Create(&platformAICapabilityVersionRecord{
		ID: 101, CapabilityID: 10, Version: 1, Status: PlatformAIReleasePublished,
		SnapshotJSON: string(snapshot), SnapshotHash: "hash",
	}).Error; err != nil {
		t.Fatalf("seed release: %v", err)
	}
	blocked, err := store.CreateAgentSkillVersion(ctx, &pb.CreateAgentSkillVersionRequest{
		SkillId: 1, Version: "v2", SkillMd: "two", Activate: true,
	})
	if err != nil || blocked.GetCode() != governanceBadRequest {
		t.Fatalf("released activation response=%#v err=%v", blocked, err)
	}
}

func TestListAvailableAgentSkillsUsesExecutableCurrentVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&agentSkillRecord{}, &agentSkillVersionRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	skills := []agentSkillRecord{
		{ID: 1, Name: "available", DisplayName: "Available", AgentType: "hr_recruiting_agent", IsEnabled: true, IsManualInvocable: true, CurrentVersionID: nullInt64From(11)},
		{ID: 2, Name: "automatic", DisplayName: "Automatic", AgentType: "hr_recruiting_agent", IsEnabled: true, IsManualInvocable: false, CurrentVersionID: nullInt64From(22)},
		{ID: 3, Name: "foreign-current", DisplayName: "Foreign", AgentType: "hr_recruiting_agent", IsEnabled: true, IsManualInvocable: true, CurrentVersionID: nullInt64From(22)},
	}
	versions := []agentSkillVersionRecord{
		{ID: 11, SkillID: 1, Version: "v1", SkillMD: "usable"},
		{ID: 22, SkillID: 2, Version: "v1", SkillMD: "automatic"},
	}
	if err := db.Create(&skills).Error; err != nil {
		t.Fatalf("seed skills: %v", err)
	}
	if err := db.Create(&versions).Error; err != nil {
		t.Fatalf("seed versions: %v", err)
	}
	store := &NativeStore{db: db}

	rows, total, err := store.ListAvailableAgentSkills(context.Background(), 1, 100, "hr_recruiting_agent")
	if err != nil {
		t.Fatalf("ListAvailableAgentSkills returned error: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].GetId() != 1 {
		t.Fatalf("rows=%#v total=%d, want only skill 1", rows, total)
	}
}
