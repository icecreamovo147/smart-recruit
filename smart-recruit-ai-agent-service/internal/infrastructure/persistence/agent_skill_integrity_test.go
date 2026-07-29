package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-ai-agent-service/internal/domain/agentskill"
	"smart-recruit-proto/recruitment/pb"
)

func TestCreateAgentSkillRollsBackWhenSectionPersistenceFails(t *testing.T) {
	store, db := newAgentSkillTestStore(t)
	injectedErr := errors.New("injected section persistence failure")
	registerAgentSkillSectionCreateFailure(t, db, injectedErr)

	created, err := store.CreateAgentSkill(context.Background(), &pb.CreateAgentSkillRequest{
		Package:              testAgentSkillPackage("rollback-create", "Rollback Create", "hr_recruiting_agent"),
		Version:              "v1",
		Activate:             true,
		ActorUserId:          42,
		IsEnabled:            true,
		IsEnabledSet:         true,
		IsManualInvocable:    true,
		IsManualInvocableSet: true,
	})
	if !errors.Is(err, injectedErr) {
		t.Fatalf("CreateAgentSkill error = %v, want %v", err, injectedErr)
	}
	if created != nil {
		t.Fatalf("CreateAgentSkill response = %#v, want nil", created)
	}

	assertAgentSkillRowCounts(t, db, 0, 0, 0)
}

func TestCreateAgentSkillVersionRollsBackActivationWhenSectionPersistenceFails(t *testing.T) {
	store, db := newAgentSkillTestStore(t)
	ctx := context.Background()
	existing := createAgentSkillForTest(t, store, "rollback-version", "Rollback Version")
	originalVersionID := existing.GetCurrentVersionId()

	injectedErr := errors.New("injected section persistence failure")
	registerAgentSkillSectionCreateFailure(t, db, injectedErr)

	draft := testAgentSkillPackage("rollback-version", "Rollback Version v2", "hr_recruiting_agent")
	draft.CoreMarkdown = "Use the updated policy only if persistence succeeds."
	created, err := store.CreateAgentSkillVersion(ctx, &pb.CreateAgentSkillVersionRequest{
		SkillId:     existing.GetId(),
		Version:     "v2",
		Package:     draft,
		ChangeNote:  "must roll back",
		Activate:    true,
		ActorUserId: 99,
	})
	if !errors.Is(err, injectedErr) {
		t.Fatalf("CreateAgentSkillVersion error = %v, want %v", err, injectedErr)
	}
	if created != nil {
		t.Fatalf("CreateAgentSkillVersion response = %#v, want nil", created)
	}

	var registry agentSkillRecord
	if err := db.First(&registry, existing.GetId()).Error; err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	if !registry.CurrentVersionID.Valid || registry.CurrentVersionID.Int64 != originalVersionID {
		t.Fatalf("current_version_id = %#v, want %d", registry.CurrentVersionID, originalVersionID)
	}

	var versions []agentSkillVersionRecord
	if err := db.Order("id ASC").Find(&versions).Error; err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 1 || versions[0].ID != originalVersionID || versions[0].Version != "v1" {
		t.Fatalf("versions after rollback = %#v, want only original v1", versions)
	}

	var sections []agentSkillSectionRecord
	if err := db.Order("id ASC").Find(&sections).Error; err != nil {
		t.Fatalf("list sections: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("sections after rollback = %#v, want original two sections", sections)
	}
	for _, section := range sections {
		if section.SkillVersionID != originalVersionID {
			t.Fatalf("section after rollback = %#v, want version_id %d", section, originalVersionID)
		}
	}
}

func TestAgentSkillPackageV2Lifecycle(t *testing.T) {
	store, db := newAgentSkillTestStore(t)
	ctx := context.Background()

	created, err := store.CreateAgentSkill(ctx, &pb.CreateAgentSkillRequest{
		Package:              testAgentSkillPackage("candidate-screen", "Candidate Screen", "hr_recruiting_agent"),
		Version:              "v1",
		ChangeNote:           "initial",
		Activate:             true,
		ActorUserId:          42,
		IsEnabled:            true,
		IsEnabledSet:         true,
		IsManualInvocable:    false,
		IsManualInvocableSet: true,
	})
	if err != nil {
		t.Fatalf("CreateAgentSkill returned error: %v", err)
	}
	if created.GetCode() != governanceOK || created.GetSkill() == nil {
		t.Fatalf("CreateAgentSkill response = %#v", created)
	}
	skill := created.GetSkill()
	if skill.GetName() != "candidate-screen" || skill.GetCurrentVersionId() <= 0 {
		t.Fatalf("created registry = %#v", skill)
	}
	if skill.GetCurrentVersion().GetAgentType() != "hr_recruiting_agent" ||
		skill.GetCurrentVersion().GetCompiledHash() == "" ||
		skill.GetCurrentVersion().GetRisk() != pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_MEDIUM {
		t.Fatalf("current version summary = %#v", skill.GetCurrentVersion())
	}

	var registries []agentSkillRecord
	if err := db.Find(&registries).Error; err != nil || len(registries) != 1 {
		t.Fatalf("registry rows = %#v, err = %v", registries, err)
	}
	var versions []agentSkillVersionRecord
	if err := db.Find(&versions).Error; err != nil || len(versions) != 1 {
		t.Fatalf("version rows = %#v, err = %v", versions, err)
	}
	if versions[0].CompiledHash == "" || versions[0].CoreEstimatedTokens <= 0 ||
		versions[0].ManifestJSON == "" || versions[0].CompiledMarkdown == "" {
		t.Fatalf("persisted version = %#v", versions[0])
	}
	var outboxRows []embeddingOutboxRecord
	if err := db.Order("id ASC").Find(&outboxRows).Error; err != nil || len(outboxRows) != 1 {
		t.Fatalf("embedding outbox rows = %#v, err = %v", outboxRows, err)
	}
	var queued embeddingUpsertEnvelope
	if err := json.Unmarshal([]byte(outboxRows[0].Payload), &queued); err != nil {
		t.Fatalf("decode embedding outbox payload: %v", err)
	}
	if queued.EventType != embeddingUpsertEventType ||
		queued.Payload.ObjectType != "agent_skill_version" ||
		queued.Payload.ObjectID != versions[0].ID {
		t.Fatalf("embedding outbox payload = %#v", queued)
	}
	var sections []agentSkillSectionRecord
	if err := db.Order("ordinal ASC").Find(&sections).Error; err != nil || len(sections) != 2 {
		t.Fatalf("section rows = %#v, err = %v", sections, err)
	}
	if sections[0].SectionKey != "requirements" || sections[1].SectionKey != "examples" {
		t.Fatalf("persisted section order = %#v", sections)
	}

	listed, err := store.ListAgentSkillVersions(ctx, &pb.ListAgentSkillVersionsRequest{SkillId: skill.GetId()})
	if err != nil || listed.GetCode() != governanceOK || len(listed.GetList()) != 1 {
		t.Fatalf("ListAgentSkillVersions response = %#v, err = %v", listed, err)
	}
	info := listed.GetList()[0]
	if info.GetPackage().GetCompiledHash() != versions[0].CompiledHash ||
		len(info.GetPackage().GetSections()) != 2 ||
		info.GetPackage().GetSections()[0].GetId() <= 0 {
		t.Fatalf("version package = %#v", info.GetPackage())
	}

	mismatched := testAgentSkillPackage("different-skill", "Different", "hr_recruiting_agent")
	rejected, err := store.CreateAgentSkillVersion(ctx, &pb.CreateAgentSkillVersionRequest{
		SkillId:  skill.GetId(),
		Version:  "v2",
		Package:  mismatched,
		Activate: true,
	})
	if err != nil || rejected.GetCode() != governanceBadRequest ||
		rejected.GetMsg() != "ai.agent_skill_package_invalid" {
		t.Fatalf("mismatched version response = %#v, err = %v", rejected, err)
	}
	var versionCount int64
	if err := db.Model(&agentSkillVersionRecord{}).Count(&versionCount).Error; err != nil {
		t.Fatalf("count versions after rejection: %v", err)
	}
	if versionCount != 1 {
		t.Fatalf("version count after rejected package = %d, want 1", versionCount)
	}

	before := versions[0]
	second := testAgentSkillPackage("candidate-screen", "Candidate Screen v2", "hr_recruiting_agent")
	second.CoreMarkdown = "Use the updated screening policy."
	createdVersion, err := store.CreateAgentSkillVersion(ctx, &pb.CreateAgentSkillVersionRequest{
		SkillId:  skill.GetId(),
		Version:  "v2",
		Package:  second,
		Activate: true,
	})
	if err != nil || createdVersion.GetCode() != governanceOK {
		t.Fatalf("CreateAgentSkillVersion response = %#v, err = %v", createdVersion, err)
	}
	if err := db.Order("id ASC").Find(&outboxRows).Error; err != nil || len(outboxRows) != 2 {
		t.Fatalf("embedding outbox rows after v2 = %#v, err = %v", outboxRows, err)
	}
	var unchanged agentSkillVersionRecord
	if err := db.First(&unchanged, before.ID).Error; err != nil {
		t.Fatalf("reload v1: %v", err)
	}
	if unchanged.CompiledHash != before.CompiledHash || unchanged.CompiledMarkdown != before.CompiledMarkdown {
		t.Fatalf("v1 mutated: before=%#v after=%#v", before, unchanged)
	}

	updated, err := store.UpdateAgentSkill(ctx, &pb.UpdateAgentSkillRequest{
		Id:                   skill.GetId(),
		DisplayName:          "Registry Label",
		DisplayNameSet:       true,
		Description:          "Registry description only",
		DescriptionSet:       true,
		IsManualInvocable:    true,
		IsManualInvocableSet: true,
		ActorUserId:          99,
	})
	if err != nil || updated.GetCode() != governanceOK {
		t.Fatalf("UpdateAgentSkill response = %#v, err = %v", updated, err)
	}
	if updated.GetSkill().GetName() != "candidate-screen" ||
		updated.GetSkill().GetDisplayName() != "Registry Label" ||
		updated.GetSkill().GetCurrentVersion().GetVersion() != "v2" {
		t.Fatalf("updated registry = %#v", updated.GetSkill())
	}
}

func TestAgentSkillVersionActivationValidatesOwnership(t *testing.T) {
	store, _ := newAgentSkillTestStore(t)
	ctx := context.Background()
	first := createAgentSkillForTest(t, store, "first-skill", "First")
	second := createAgentSkillForTest(t, store, "second-skill", "Second")

	foreign, err := store.ActivateAgentSkillVersion(ctx, &pb.ActivateAgentSkillVersionRequest{
		SkillId:   first.GetId(),
		VersionId: second.GetCurrentVersionId(),
	})
	if err != nil || foreign.GetCode() != governanceBadRequest {
		t.Fatalf("foreign activation response = %#v, err = %v", foreign, err)
	}
	reloaded, err := store.GetAgentSkill(ctx, &pb.GetAgentSkillRequest{Id: first.GetId()})
	if err != nil || reloaded.GetSkill().GetCurrentVersionId() != first.GetCurrentVersionId() {
		t.Fatalf("first skill after foreign activation = %#v, err = %v", reloaded, err)
	}

	repeated, err := store.ActivateAgentSkillVersion(ctx, &pb.ActivateAgentSkillVersionRequest{
		SkillId:   first.GetId(),
		VersionId: first.GetCurrentVersionId(),
	})
	if err != nil || repeated.GetCode() != governanceOK {
		t.Fatalf("repeated activation response = %#v, err = %v", repeated, err)
	}
}

func TestPreviewAgentSkillUsesCanonicalCompiler(t *testing.T) {
	store, _ := newAgentSkillTestStore(t)
	draft := testAgentSkillPackage("canonical-preview", "Canonical Preview", "hr_recruiting_agent")

	first, err := store.PreviewAgentSkill(context.Background(), &pb.PreviewAgentSkillRequest{Package: draft})
	if err != nil || first.GetCode() != governanceOK {
		t.Fatalf("first preview = %#v, err = %v", first, err)
	}
	second, err := store.PreviewAgentSkill(context.Background(), &pb.PreviewAgentSkillRequest{Package: draft})
	if err != nil || second.GetCode() != governanceOK {
		t.Fatalf("second preview = %#v, err = %v", second, err)
	}
	if first.GetPackage().GetCompiledHash() != second.GetPackage().GetCompiledHash() ||
		first.GetPackage().GetCompiledMarkdown() != second.GetPackage().GetCompiledMarkdown() {
		t.Fatalf("preview is not deterministic: first=%#v second=%#v", first.GetPackage(), second.GetPackage())
	}
}

func TestPreviewAgentSkillRejectsInvalidDrafts(t *testing.T) {
	tests := []struct {
		name           string
		mutate         func(*pb.AgentSkillPackageDraft) *pb.AgentSkillPackageDraft
		wantMessageKey string
	}{
		{
			name: "missing package",
			mutate: func(*pb.AgentSkillPackageDraft) *pb.AgentSkillPackageDraft {
				return nil
			},
			wantMessageKey: "ai.agent_skill_package_invalid",
		},
		{
			name: "invalid authoring json",
			mutate: func(draft *pb.AgentSkillPackageDraft) *pb.AgentSkillPackageDraft {
				draft.AuthoringJson = "{invalid"
				return draft
			},
			wantMessageKey: "ai.agent_skill_package_invalid",
		},
		{
			name: "client activation policy conflicts with risk",
			mutate: func(draft *pb.AgentSkillPackageDraft) *pb.AgentSkillPackageDraft {
				draft.Manifest.ActivationPolicy = pb.AgentSkillActivationPolicy_AGENT_SKILL_ACTIVATION_POLICY_CONFIRM
				return draft
			},
			wantMessageKey: "ai.agent_skill_package_invalid",
		},
		{
			name: "null reference section",
			mutate: func(draft *pb.AgentSkillPackageDraft) *pb.AgentSkillPackageDraft {
				draft.Sections = append(draft.Sections, nil)
				return draft
			},
			wantMessageKey: "ai.agent_skill_section_invalid",
		},
	}

	store, _ := newAgentSkillTestStore(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			draft := tt.mutate(testAgentSkillPackage("invalid-preview", "Invalid Preview", "hr_recruiting_agent"))
			resp, err := store.PreviewAgentSkill(context.Background(), &pb.PreviewAgentSkillRequest{Package: draft})
			if err != nil {
				t.Fatalf("PreviewAgentSkill returned error: %v", err)
			}
			if resp.GetCode() != governanceBadRequest || resp.GetMsg() != tt.wantMessageKey {
				t.Fatalf("response = %#v, want code=%d msg=%s", resp, governanceBadRequest, tt.wantMessageKey)
			}
		})
	}
}

func TestAgentSkillCompileErrorMessageKey(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "package invalid",
			err:  &agentskill.CompileError{Code: agentskill.CodePackageInvalid},
			want: "ai.agent_skill_package_invalid",
		},
		{
			name: "core budget exceeded",
			err:  &agentskill.CompileError{Code: agentskill.CodeCoreBudgetExceeded},
			want: "ai.agent_skill_core_budget_exceeded",
		},
		{
			name: "section invalid",
			err:  &agentskill.CompileError{Code: agentskill.CodeSectionInvalid},
			want: "ai.agent_skill_section_invalid",
		},
		{
			name: "composition conflict",
			err:  &agentskill.CompileError{Code: agentskill.CodeCompositionConflict},
			want: "ai.agent_skill_composition_conflict",
		},
		{
			name: "unknown error",
			err:  errors.New("unclassified"),
			want: "common.invalid_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := agentSkillCompileErrorMessageKey(tt.err); got != tt.want {
				t.Fatalf("agentSkillCompileErrorMessageKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListAvailableAgentSkillsUsesCurrentVersionManifest(t *testing.T) {
	store, _ := newAgentSkillTestStore(t)
	hr := createAgentSkillForTest(t, store, "hr-skill", "HR")
	createAgentSkillForTestWithAgent(t, store, "candidate-skill", "Candidate", "candidate_assistant")

	rows, total, err := store.ListAvailableAgentSkills(context.Background(), 1, 100, "hr_recruiting_agent")
	if err != nil {
		t.Fatalf("ListAvailableAgentSkills returned error: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].GetId() != hr.GetId() {
		t.Fatalf("rows=%#v total=%d, want only HR skill", rows, total)
	}
}

func newAgentSkillTestStore(t *testing.T) (*NativeStore, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&agentSkillRecord{}, &agentSkillVersionRecord{}, &agentSkillSectionRecord{}, &embeddingOutboxRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return &NativeStore{db: db}, db
}

func registerAgentSkillSectionCreateFailure(t *testing.T, db *gorm.DB, injectedErr error) {
	t.Helper()
	const callbackName = "test:fail_agent_skill_section_create"
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == (agentSkillSectionRecord{}).TableName() {
			tx.AddError(injectedErr)
		}
	}); err != nil {
		t.Fatalf("register section create failure: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Callback().Create().Remove(callbackName); err != nil {
			t.Errorf("remove section create failure: %v", err)
		}
	})
}

func assertAgentSkillRowCounts(t *testing.T, db *gorm.DB, wantRegistries, wantVersions, wantSections int64) {
	t.Helper()
	for name, count := range map[string]struct {
		model any
		want  int64
	}{
		"registries": {model: &agentSkillRecord{}, want: wantRegistries},
		"versions":   {model: &agentSkillVersionRecord{}, want: wantVersions},
		"sections":   {model: &agentSkillSectionRecord{}, want: wantSections},
	} {
		var got int64
		if err := db.Model(count.model).Count(&got).Error; err != nil {
			t.Fatalf("count %s: %v", name, err)
		}
		if got != count.want {
			t.Fatalf("%s count = %d, want %d", name, got, count.want)
		}
	}
}

func createAgentSkillForTest(t *testing.T, store *NativeStore, name, displayName string) *pb.AgentSkillInfo {
	t.Helper()
	return createAgentSkillForTestWithAgent(t, store, name, displayName, "hr_recruiting_agent")
}

func createAgentSkillForTestWithAgent(t *testing.T, store *NativeStore, name, displayName, agentType string) *pb.AgentSkillInfo {
	t.Helper()
	resp, err := store.CreateAgentSkill(context.Background(), &pb.CreateAgentSkillRequest{
		Package:              testAgentSkillPackage(name, displayName, agentType),
		Version:              "v1",
		Activate:             true,
		IsEnabled:            true,
		IsEnabledSet:         true,
		IsManualInvocable:    true,
		IsManualInvocableSet: true,
	})
	if err != nil || resp.GetCode() != governanceOK {
		t.Fatalf("create %s response = %#v, err = %v", name, resp, err)
	}
	return resp.GetSkill()
}

func testAgentSkillPackage(name, displayName, agentType string) *pb.AgentSkillPackageDraft {
	return &pb.AgentSkillPackageDraft{
		Manifest: &pb.AgentSkillManifest{
			SchemaVersion: 2,
			SkillName:     name,
			DisplayName:   displayName,
			Description:   "Package description",
			AgentType:     agentType,
			Category:      "screening",
			Scenario:      "candidate-screening",
			Priority:      100,
			Risk:          pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_MEDIUM,
			Composition: &pb.AgentSkillComposition{
				Role: pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY,
			},
			OutputContract: &pb.AgentSkillOutputContract{
				Mode: pb.AgentSkillOutputMode_AGENT_SKILL_OUTPUT_MODE_NONE,
			},
			RequiredCapabilities: []string{"search_candidates"},
			TriggerKeywords:      []string{"candidate", "screen"},
			SemanticTags:         []string{"screening"},
			EvaluationCriteria:   []string{"returns a useful shortlist"},
		},
		CoreMarkdown: "Screen candidates against the job requirements.",
		Sections: []*pb.AgentSkillSectionDraft{
			{
				SectionKey:      "examples",
				Title:           "Examples",
				ContentMarkdown: "Example screening response.",
				TriggerTerms:    []string{"example"},
				Priority:        10,
				Ordinal:         20,
			},
			{
				SectionKey:      "requirements",
				Title:           "Requirements",
				ContentMarkdown: "Check required and preferred qualifications.",
				TriggerTerms:    []string{"requirements"},
				Priority:        20,
				Ordinal:         10,
			},
		},
		AuthoringJson: `{"nodes":[]}`,
	}
}
