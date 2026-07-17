package persistence

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-proto/recruitment/pb"
)

func TestUpdatePromptTemplateRejectsDisablingLastActiveInScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&promptTemplateRecord{}, &promptVersionRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := &NativeStore{db: db}
	ctx := context.Background()

	seeds := []promptTemplateRecord{
		{ID: 1, Name: "hr-only", Content: "hr", IsActive: true, AgentType: "hr_agent", PromptRole: "system"},
		{ID: 2, Name: "resume-a", Content: "a", IsActive: true, AgentType: "resume_profile_extractor", PromptRole: "system"},
		{ID: 3, Name: "resume-b", Content: "b", IsActive: true, AgentType: "resume_profile_extractor", PromptRole: "system"},
		{ID: 4, Name: "resume-inactive", Content: "x", IsActive: false, AgentType: "resume_profile_extractor", PromptRole: "system"},
		{ID: 5, Name: "legacy-hr", Content: "legacy", IsActive: true, AgentType: "hr_recruiting_agent", PromptRole: "system"},
	}
	if err := db.Create(&seeds).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Last active for resume after disabling one sibling still OK.
	if resp, err := store.UpdatePromptTemplate(ctx, &pb.UpdatePromptTemplateRequest{
		Id: 2, IsActive: false, IsActiveSet: true, UpdatedBy: 1, ChangeNote: "disable one",
	}); err != nil || resp.GetCode() != configOK || resp.GetTemplate() == nil || resp.GetTemplate().GetIsActive() {
		if err != nil {
			t.Fatalf("disable resume-a: %v", err)
		}
		if resp.GetCode() != configOK {
			t.Fatalf("disable resume-a code/msg = %d/%q", resp.GetCode(), resp.GetMsg())
		}
		if resp.GetTemplate().GetIsActive() {
			t.Fatalf("resume-a should be inactive")
		}
	}

	// Last remaining active resume cannot be disabled.
	if resp, err := store.UpdatePromptTemplate(ctx, &pb.UpdatePromptTemplateRequest{
		Id: 3, IsActive: false, IsActiveSet: true, UpdatedBy: 1,
	}); err != nil {
		t.Fatalf("disable last resume: %v", err)
	} else if resp.GetCode() != configBadRequest {
		t.Fatalf("disable last resume code = %d msg=%q, want %d", resp.GetCode(), resp.GetMsg(), configBadRequest)
	}

	// hr_agent and hr_recruiting_agent share scope: disabling one is OK while the other stays active.
	if resp, err := store.UpdatePromptTemplate(ctx, &pb.UpdatePromptTemplateRequest{
		Id: 1, IsActive: false, IsActiveSet: true, UpdatedBy: 1,
	}); err != nil || resp.GetCode() != configOK {
		if err != nil {
			t.Fatalf("disable hr_agent with sibling: %v", err)
		}
		t.Fatalf("disable hr_agent code/msg = %d/%q, want ok", resp.GetCode(), resp.GetMsg())
	}

	// After both HR prompts would become inactive, reject disabling the last one.
	if resp, err := store.UpdatePromptTemplate(ctx, &pb.UpdatePromptTemplateRequest{
		Id: 5, IsActive: false, IsActiveSet: true, UpdatedBy: 1,
	}); err != nil {
		t.Fatalf("disable last hr: %v", err)
	} else if resp.GetCode() != configBadRequest {
		t.Fatalf("disable last hr code = %d msg=%q, want %d", resp.GetCode(), resp.GetMsg(), configBadRequest)
	}

	// Re-enable always allowed.
	if resp, err := store.UpdatePromptTemplate(ctx, &pb.UpdatePromptTemplateRequest{
		Id: 1, IsActive: true, IsActiveSet: true, UpdatedBy: 1,
	}); err != nil || resp.GetCode() != configOK || !resp.GetTemplate().GetIsActive() {
		if err != nil {
			t.Fatalf("re-enable: %v", err)
		}
		t.Fatalf("re-enable response = %#v", resp)
	}
}
