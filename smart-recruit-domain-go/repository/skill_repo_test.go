package repository

import (
	"context"
	"testing"

	"smart-recruit-domain-go/model"
)

func TestSkillRepoCreateVersionAndListEnabledCurrentTools(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSkillRepo(db)
	ctx := context.Background()

	skill := &model.Skill{Name: "resume_ops", DisplayName: "Resume Ops", SourceType: "local", IsEnabled: 1}
	if err := repo.CreateSkill(ctx, skill); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	version := &model.SkillVersion{
		SkillID:      skill.ID,
		Version:      "1.0.0",
		ManifestJSON: `{"name":"resume_ops","version":"1.0.0","runtime":{"type":"http"},"tools":[{"name":"parse"}]}`,
		RuntimeType:  "http",
	}
	if err := repo.CreateVersionWithTools(ctx, version, []model.SkillTool{{ToolName: "parse", IsEnabled: 1}}, true); err != nil {
		t.Fatalf("CreateVersionWithTools: %v", err)
	}

	rows, err := repo.ListEnabledCurrentSkillTools(ctx)
	if err != nil {
		t.Fatalf("ListEnabledCurrentSkillTools: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 enabled tool, got %d", len(rows))
	}
	if rows[0].SkillName != "resume_ops" || rows[0].ToolName != "parse" || rows[0].RuntimeType != "http" {
		t.Fatalf("unexpected row: %+v", rows[0])
	}
}

func TestSkillRepoDisabledSkillOrToolExcluded(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSkillRepo(db)
	ctx := context.Background()

	skill := &model.Skill{Name: "screening", DisplayName: "Screening", SourceType: "local", IsEnabled: 1}
	if err := repo.CreateSkill(ctx, skill); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	version := &model.SkillVersion{SkillID: skill.ID, Version: "1", ManifestJSON: `{}`, RuntimeType: "http"}
	if err := repo.CreateVersionWithTools(ctx, version, []model.SkillTool{{ToolName: "rank", IsEnabled: 1}}, true); err != nil {
		t.Fatalf("CreateVersionWithTools: %v", err)
	}
	tools, err := repo.ListTools(ctx, version.ID, false)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if err := repo.UpdateToolEnabled(ctx, tools[0].ID, false); err != nil {
		t.Fatalf("UpdateToolEnabled: %v", err)
	}
	rows, err := repo.ListEnabledCurrentSkillTools(ctx)
	if err != nil {
		t.Fatalf("ListEnabledCurrentSkillTools: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("disabled tool should be excluded, got %d rows", len(rows))
	}

	if err := repo.UpdateSkillPartial(ctx, skill.ID, map[string]any{"is_enabled": int32(0)}); err != nil {
		t.Fatalf("UpdateSkillPartial: %v", err)
	}
	if err := db.Model(&model.SkillTool{}).Where("skill_version_id = ?", version.ID).Update("is_enabled", 1).Error; err != nil {
		t.Fatalf("enable tool: %v", err)
	}
	rows, err = repo.ListEnabledCurrentSkillTools(ctx)
	if err != nil {
		t.Fatalf("ListEnabledCurrentSkillTools: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("disabled skill should be excluded, got %d rows", len(rows))
	}
}
