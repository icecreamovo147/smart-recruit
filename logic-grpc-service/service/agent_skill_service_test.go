package service

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

const testAgentSkillFlowJSON = `{
	"nodes":[
		{"id":"trigger","type":"trigger","title":"Trigger","content":"Use for candidate updates.","order":1},
		{"id":"instruction","type":"instruction","title":"Instruction","content":"Review the update.","order":2},
		{"id":"output","type":"output","title":"Output","content":"Return a concise summary.","order":3},
		{"id":"constraint","type":"constraint","title":"Constraint","content":"Keep it factual.","order":4}
	]
}`

func newAgentSkillTestService(t *testing.T) (*AgentSkillService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentSkill{}, &model.AgentSkillVersion{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewAgentSkillService(repository.NewAgentSkillRepo(db)), db
}

func TestAgentSkillServiceUpdateClearsDisplayNameAndDescription(t *testing.T) {
	svc, db := newAgentSkillTestService(t)
	skill := &model.AgentSkill{
		Name:              "candidate_summary",
		DisplayName:       "Candidate Summary",
		Description:       "Initial description",
		IsEnabled:         1,
		IsManualInvocable: 1,
		TriggerKeywords:   "[]",
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}

	resp, err := svc.UpdateAgentSkill(context.Background(), &pb.UpdateAgentSkillRequest{
		Id:             skill.ID,
		DisplayName:    "",
		DisplayNameSet: true,
		Description:    "",
		DescriptionSet: true,
	})
	if err != nil {
		t.Fatalf("UpdateAgentSkill: %v", err)
	}
	if resp.GetSkill().GetDisplayName() != "" || resp.GetSkill().GetDescription() != "" {
		t.Fatalf("expected cleared fields, got %+v", resp.GetSkill())
	}
}

func TestAgentSkillServiceCreateVersionDuplicateReturnsAlreadyExists(t *testing.T) {
	svc, db := newAgentSkillTestService(t)
	skill := &model.AgentSkill{
		Name:              "candidate_summary",
		DisplayName:       "Candidate Summary",
		Description:       "Summarize candidate updates.",
		IsEnabled:         1,
		IsManualInvocable: 1,
		TriggerKeywords:   "[]",
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}

	req := &pb.CreateAgentSkillVersionRequest{
		SkillId:  skill.ID,
		Version:  "1.0.0",
		FlowJson: testAgentSkillFlowJSON,
	}
	if _, err := svc.CreateAgentSkillVersion(context.Background(), req); err != nil {
		t.Fatalf("first CreateAgentSkillVersion: %v", err)
	}
	if _, err := svc.CreateAgentSkillVersion(context.Background(), req); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists for duplicate version, got %v (%v)", status.Code(err), err)
	}
}

func TestAgentSkillServiceCreateVersionActivateUpdatesSkillAudit(t *testing.T) {
	svc, db := newAgentSkillTestService(t)
	oldUpdatedAt := time.Now().Add(-time.Hour)
	skill := &model.AgentSkill{
		Name:              "candidate_summary",
		DisplayName:       "Candidate Summary",
		Description:       "Summarize candidate updates.",
		IsEnabled:         1,
		IsManualInvocable: 1,
		TriggerKeywords:   "[]",
		UpdatedAt:         oldUpdatedAt,
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}

	resp, err := svc.CreateAgentSkillVersion(context.Background(), &pb.CreateAgentSkillVersionRequest{
		SkillId:     skill.ID,
		Version:     "1.0.1",
		FlowJson:    testAgentSkillFlowJSON,
		Activate:    true,
		ActorUserId: 42,
	})
	if err != nil {
		t.Fatalf("CreateAgentSkillVersion: %v", err)
	}

	var updated model.AgentSkill
	if err := db.First(&updated, skill.ID).Error; err != nil {
		t.Fatalf("reload skill: %v", err)
	}
	if updated.CurrentVersionID == nil || *updated.CurrentVersionID != resp.GetVersion().GetId() {
		t.Fatalf("expected current version %d, got %v", resp.GetVersion().GetId(), updated.CurrentVersionID)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != 42 {
		t.Fatalf("expected updated_by=42, got %v", updated.UpdatedBy)
	}
	if !updated.UpdatedAt.After(oldUpdatedAt) {
		t.Fatalf("expected updated_at after %s, got %s", oldUpdatedAt, updated.UpdatedAt)
	}
}
