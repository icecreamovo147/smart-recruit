package persistence

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-proto/recruitment/pb"
)

func TestAgentPromptBindingValidationOnCreateAndUpdate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&promptTemplateRecord{},
		&agentConfigRecord{},
		&agentToolBindingRecord{},
		&agentCapabilityBindingRecord{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	prompts := []promptTemplateRecord{
		{ID: 1, Name: "hr-system", Content: "ok", IsActive: true, AgentType: "hr_recruiting_agent", PromptRole: "system"},
		{ID: 2, Name: "legacy-hr-system", Content: "legacy", IsActive: true, AgentType: "hr_agent", PromptRole: "system"},
		{ID: 3, Name: "inactive", Content: "inactive", IsActive: false, AgentType: "hr_recruiting_agent", PromptRole: "system"},
		{ID: 4, Name: "wrong-role", Content: "user", IsActive: true, AgentType: "hr_recruiting_agent", PromptRole: "user"},
		{ID: 5, Name: "candidate", Content: "candidate", IsActive: true, AgentType: "candidate_assistant", PromptRole: "system"},
	}
	if err := db.Create(&prompts).Error; err != nil {
		t.Fatalf("seed prompts: %v", err)
	}
	store := &NativeStore{db: db}
	ctx := context.Background()

	for _, tt := range []struct {
		name     string
		promptID int64
		wantCode int32
	}{
		{name: "exact active system", promptID: 1, wantCode: configOK},
		{name: "read-only legacy HR alias", promptID: 2, wantCode: configOK},
		{name: "inactive", promptID: 3, wantCode: configBadRequest},
		{name: "wrong role", promptID: 4, wantCode: configBadRequest},
		{name: "incompatible type", promptID: 5, wantCode: configBadRequest},
		{name: "missing", promptID: 999, wantCode: configBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp, createErr := store.CreateAgent(ctx, &pb.CreateAgentRequest{
				Name: "agent-" + tt.name, DisplayName: tt.name, AgentType: "hr_recruiting_agent", PromptTemplateId: tt.promptID,
			})
			if createErr != nil {
				t.Fatalf("CreateAgent returned error: %v", createErr)
			}
			if resp.GetCode() != tt.wantCode {
				t.Fatalf("CreateAgent code/msg = %d/%q, want %d", resp.GetCode(), resp.GetMsg(), tt.wantCode)
			}
		})
	}

	existing := agentConfigRecord{ID: 100, Name: "existing", DisplayName: "Existing", AgentType: "hr_recruiting_agent", PromptTemplateID: nullInt64From(1), IsEnabled: true}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing agent: %v", err)
	}
	resp, updateErr := store.UpdateAgent(ctx, &pb.UpdateAgentRequest{Id: 100, PromptTemplateId: 5, PromptTemplateIdSet: true})
	if updateErr != nil {
		t.Fatalf("UpdateAgent returned error: %v", updateErr)
	}
	if resp.GetCode() != configBadRequest {
		t.Fatalf("UpdateAgent response = %#v, want incompatible binding rejection", resp)
	}
	var saved agentConfigRecord
	if err := db.First(&saved, 100).Error; err != nil {
		t.Fatalf("reload existing agent: %v", err)
	}
	if !saved.PromptTemplateID.Valid || saved.PromptTemplateID.Int64 != 1 {
		t.Fatalf("saved prompt binding = %#v, want unchanged prompt 1", saved.PromptTemplateID)
	}
	if err := db.Create(&agentToolBindingRecord{AgentID: 100, ToolName: "search_candidates", IsEnabled: true}).Error; err != nil {
		t.Fatalf("seed Agent Tool binding: %v", err)
	}
	pinned, found, getErr := store.GetRuntimeAgentConfigByID(ctx, 100)
	if getErr != nil {
		t.Fatalf("GetRuntimeAgentConfigByID returned error: %v", getErr)
	}
	if !found || pinned.GetId() != 100 || pinned.GetName() != "existing" || len(pinned.GetToolBindings()) != 1 || pinned.GetToolBindings()[0].GetToolName() != "search_candidates" {
		t.Fatalf("pinned runtime Agent = %#v, found=%v", pinned, found)
	}
}

func TestReplaceAgentBindingsRejectsRetiredSkillSource(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&agentCapabilityBindingRecord{}); err != nil {
		t.Fatalf("migrate Agent Capability bindings: %v", err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		return replaceAgentBindings(tx, 7, nil, []*pb.AgentCapabilityBindingInfo{{
			AgentId:          7,
			CapabilitySource: "skill",
			CapabilityKey:    "legacy:tool",
			IsEnabled:        true,
			Priority:         1,
		}}, false, true)
	})
	if err == nil {
		t.Fatal("replaceAgentBindings accepted retired skill capability source")
	}

	var count int64
	if err := db.Model(&agentCapabilityBindingRecord{}).Count(&count).Error; err != nil {
		t.Fatalf("count capability bindings: %v", err)
	}
	if count != 0 {
		t.Fatalf("retired skill binding persisted: %d", count)
	}
}
