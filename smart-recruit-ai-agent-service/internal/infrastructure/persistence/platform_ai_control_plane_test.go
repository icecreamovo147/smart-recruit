package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-ai-agent-service/internal/domain/agentskill"
)

var (
	platformAITestEvaluationSuiteHash  = strings.Repeat("a", 64)
	platformAITestEvaluationResultHash = strings.Repeat("b", 64)
)

func TestPlatformAICapabilityPublishIsImmutableAndAudited(t *testing.T) {
	db := newPlatformAIControlPlaneTestDB(t)
	store := newPlatformAIControlPlaneTestStore(db)
	providerID, defaultModelID, _ := seedPlatformAIModels(t, db)
	_ = providerID
	seedPlatformAIAgentPrompt(t, db, 10, 20, "hr_recruiting_agent", "hr_agent")
	capability := seedPlatformAICapability(t, db, "ai.chat", PlatformAIAudienceTenantHR)

	snapshot := mustCapabilitySnapshotJSON(t, capability, []int64{defaultModelID}, defaultModelID)
	draft, err := store.CreatePlatformAICapabilityDraft(context.Background(), capability.ID, 91, snapshot, "first release", "req-draft")
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if draft.Status != PlatformAIReleaseDraft || draft.Version != 1 || draft.SnapshotHash == "" {
		t.Fatalf("unexpected draft: %+v", draft)
	}

	updatedDraft, err := store.UpdatePlatformAICapabilityDraft(context.Background(), draft.ID, 91, snapshot, "update draft", "req-edit")
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}

	published, err := store.PublishPlatformAICapabilityVersion(context.Background(), updatedDraft.ID, 91, "req-publish")
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if published.Status != PlatformAIReleasePublished || published.PublishedAt == nil {
		t.Fatalf("unexpected published release: %+v", published)
	}

	_, err = store.UpdatePlatformAICapabilityDraft(context.Background(), draft.ID, 91, snapshot, "illegal edit", "req-edit")
	if !errors.Is(err, ErrCapabilityVersionChanged) {
		t.Fatalf("update published error = %v, want ErrCapabilityVersionChanged", err)
	}

	var auditCount int64
	if err := db.Model(&platformAIConfigAuditRecord{}).Count(&auditCount).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if auditCount != 3 {
		t.Fatalf("audit count = %d, want 3", auditCount)
	}

	var audits []platformAIConfigAuditRecord
	if err := db.Order("id ASC").Find(&audits).Error; err != nil {
		t.Fatalf("load audit rows: %v", err)
	}
	if audits[0].BeforeSnapshot.Valid || audits[2].BeforeSnapshot.Valid {
		t.Fatalf("create and publish audit before snapshots must be SQL NULL: %+v", audits)
	}
	if !audits[1].BeforeSnapshot.Valid || !json.Valid([]byte(audits[1].BeforeSnapshot.String)) {
		t.Fatalf("draft update before snapshot must contain valid JSON: %+v", audits[1].BeforeSnapshot)
	}
	for _, audit := range audits {
		if !audit.AfterSnapshot.Valid || !json.Valid([]byte(audit.AfterSnapshot.String)) {
			t.Fatalf("audit %q after snapshot must contain valid JSON: %+v", audit.Action, audit.AfterSnapshot)
		}
	}
}

func TestNormalizeCapabilitySnapshotRequiresV2Contract(t *testing.T) {
	capability := platformAICapabilityRecord{CapabilityKey: "ai.chat", Audience: PlatformAIAudienceTenantHR}
	valid := mustCapabilitySnapshotObject(t, capability, []int64{1}, 1)
	tests := []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "schema v1",
			mutate: func(snapshot map[string]any) {
				snapshot["schema_version"] = 1
			},
			want: "unsupported capability snapshot schema_version 1",
		},
		{
			name: "missing schema",
			mutate: func(snapshot map[string]any) {
				delete(snapshot, "schema_version")
			},
			want: "schema_version is required",
		},
		{
			name: "missing capability key",
			mutate: func(snapshot map[string]any) {
				delete(snapshot, "capability_key")
			},
			want: "capability_key is required",
		},
		{
			name: "missing audience",
			mutate: func(snapshot map[string]any) {
				delete(snapshot, "audience")
			},
			want: "audience is required",
		},
		{
			name: "missing model policy",
			mutate: func(snapshot map[string]any) {
				delete(snapshot, "model_policy")
			},
			want: "model_policy is required",
		},
		{
			name: "missing configuration refs",
			mutate: func(snapshot map[string]any) {
				delete(snapshot, "configuration_refs")
			},
			want: "configuration_refs is required",
		},
		{
			name: "missing skill policy",
			mutate: func(snapshot map[string]any) {
				delete(snapshot, "skill_runtime_policy")
			},
			want: "skill_runtime_policy is required",
		},
		{
			name: "legacy skill IDs",
			mutate: func(snapshot map[string]any) {
				snapshot["configuration_refs"].(map[string]any)["ai_skill_version_ids"] = []int64{101}
			},
			want: "unknown field",
		},
		{
			name: "unknown top level",
			mutate: func(snapshot map[string]any) {
				snapshot["runtime_mode"] = "legacy"
			},
			want: "unknown field",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := cloneJSONMap(t, valid)
			tt.mutate(snapshot)
			raw, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatalf("encode snapshot: %v", err)
			}
			if _, _, _, err := normalizeCapabilitySnapshot(raw, capability); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("normalize error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestNormalizeCapabilitySnapshotPreservesCanonicalNormalization(t *testing.T) {
	capability := platformAICapabilityRecord{CapabilityKey: "ai.chat", Audience: PlatformAIAudienceTenantHR}
	snapshot := mustCapabilitySnapshotObject(t, capability, []int64{3, 1, 3}, 1)
	snapshot["configuration_refs"].(map[string]any)["agent_ids"] = []int64{10, 2, 10}
	policy := snapshot["skill_runtime_policy"].(map[string]any)
	policy["policy_version"] = " " + PlatformAISkillPolicyVersion + " "
	policy["evaluation_suite_hash"] = " " + strings.ToUpper(platformAITestEvaluationSuiteHash) + " "

	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("encode non-canonical snapshot: %v", err)
	}
	normalized, hash, decoded, err := normalizeCapabilitySnapshot(raw, capability)
	if err != nil {
		t.Fatalf("normalize snapshot: %v", err)
	}
	if got, want := decoded.ModelPolicy.AllowedLLMModelIDs, []int64{1, 3}; !equalInt64s(got, want) {
		t.Fatalf("allowed models = %#v, want %#v", got, want)
	}
	if got, want := decoded.ConfigurationRef.AgentIDs, []int64{2, 10}; !equalInt64s(got, want) {
		t.Fatalf("agent IDs = %#v, want %#v", got, want)
	}
	if decoded.SkillRuntimePolicy.PolicyVersion != PlatformAISkillPolicyVersion ||
		decoded.SkillRuntimePolicy.EvaluationSuiteHash != platformAITestEvaluationSuiteHash {
		t.Fatalf("normalized skill policy = %+v", decoded.SkillRuntimePolicy)
	}

	secondNormalized, secondHash, _, err := normalizeCapabilitySnapshot([]byte(normalized), capability)
	if err != nil {
		t.Fatalf("normalize canonical snapshot: %v", err)
	}
	if secondNormalized != normalized || secondHash != hash {
		t.Fatalf("canonical normalization is not stable: first=(%s,%s) second=(%s,%s)", normalized, hash, secondNormalized, secondHash)
	}
}

func TestNormalizeCapabilitySnapshotSkillPolicy(t *testing.T) {
	capability := platformAICapabilityRecord{CapabilityKey: "ai.chat", Audience: PlatformAIAudienceTenantHR}
	tests := []struct {
		name   string
		policy PlatformAISkillRuntimePolicy
		want   string
	}{
		{
			name:   "defaults",
			policy: PlatformAISkillRuntimePolicy{PolicyVersion: PlatformAISkillPolicyVersion},
		},
		{
			name:   "bounded custom budget",
			policy: validPlatformAITestSkillPolicy(),
		},
		{
			name:   "wrong policy version",
			policy: PlatformAISkillRuntimePolicy{PolicyVersion: "legacy"},
			want:   "unsupported skill runtime policy_version",
		},
		{
			name:   "token budget too large",
			policy: PlatformAISkillRuntimePolicy{PolicyVersion: PlatformAISkillPolicyVersion, MaxSkillTokens: 3001},
			want:   "max_skill_tokens",
		},
		{
			name:   "ratio too large",
			policy: PlatformAISkillRuntimePolicy{PolicyVersion: PlatformAISkillPolicyVersion, MaxInputRatio: 0.16},
			want:   "max_input_ratio",
		},
		{
			name:   "max skills not fixed",
			policy: PlatformAISkillRuntimePolicy{PolicyVersion: PlatformAISkillPolicyVersion, MaxSkills: 1},
			want:   "max_skills",
		},
		{
			name: "invalid evaluation hash",
			policy: PlatformAISkillRuntimePolicy{
				PolicyVersion:       PlatformAISkillPolicyVersion,
				EvaluationSuiteHash: "not-a-hash",
			},
			want: "evaluation_suite_hash",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := PlatformAICapabilitySnapshot{
				SchemaVersion:      PlatformAISnapshotSchemaVersion,
				CapabilityKey:      capability.CapabilityKey,
				Audience:           capability.Audience,
				ModelPolicy:        PlatformAIModelPolicy{AllowedLLMModelIDs: []int64{1}, DefaultLLMModelID: 1},
				ConfigurationRef:   PlatformAIConfigurationRefs{},
				SkillRuntimePolicy: tt.policy,
			}
			raw, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatalf("encode snapshot: %v", err)
			}
			_, _, normalized, err := normalizeCapabilitySnapshot(raw, capability)
			if tt.want != "" {
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("normalize error = %v, want %q", err, tt.want)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			if normalized.SkillRuntimePolicy.MaxSkillTokens != PlatformAIDefaultMaxSkillTokens ||
				normalized.SkillRuntimePolicy.MaxInputRatio != PlatformAIDefaultMaxSkillInputRatio ||
				normalized.SkillRuntimePolicy.MaxSkills != PlatformAIDefaultMaxSkills {
				t.Fatalf("normalized policy = %+v", normalized.SkillRuntimePolicy)
			}
		})
	}
}

func equalInt64s(left, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func TestPlatformAICapabilityDraftCanBeDeletedWithoutLosingAuditEvidence(t *testing.T) {
	db := newPlatformAIControlPlaneTestDB(t)
	store := newPlatformAIControlPlaneTestStore(db)
	_, defaultModelID, _ := seedPlatformAIModels(t, db)
	seedPlatformAIAgentPrompt(t, db, 10, 20, "hr_recruiting_agent", "hr_agent")
	capability := seedPlatformAICapability(t, db, "ai.chat", PlatformAIAudienceTenantHR)
	snapshot := mustCapabilitySnapshotJSON(t, capability, []int64{defaultModelID}, defaultModelID)

	draft, err := store.CreatePlatformAICapabilityDraft(context.Background(), capability.ID, 91, snapshot, "discard me", "req-draft")
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	if err := store.DeletePlatformAICapabilityDraft(context.Background(), draft.ID, 91, "req-delete"); err != nil {
		t.Fatalf("delete draft: %v", err)
	}

	var versionCount int64
	if err := db.Model(&platformAICapabilityVersionRecord{}).Where("id = ?", draft.ID).Count(&versionCount).Error; err != nil {
		t.Fatalf("count deleted draft: %v", err)
	}
	if versionCount != 0 {
		t.Fatalf("deleted draft still exists: count = %d", versionCount)
	}

	var audits []platformAIConfigAuditRecord
	if err := db.Where("capability_id = ?", capability.ID).Order("id ASC").Find(&audits).Error; err != nil {
		t.Fatalf("load audit rows: %v", err)
	}
	if len(audits) != 2 || audits[0].Action != "capability.release.draft.create" || audits[1].Action != "capability.release.draft.delete" {
		t.Fatalf("unexpected draft audit trail: %+v", audits)
	}
	for _, audit := range audits {
		if audit.CapabilityVersionID.Valid {
			t.Fatalf("deleted draft audit must not retain a foreign-key version reference: %+v", audit)
		}
	}
	if !audits[0].AfterSnapshot.Valid || !json.Valid([]byte(audits[0].AfterSnapshot.String)) || !audits[1].BeforeSnapshot.Valid || !json.Valid([]byte(audits[1].BeforeSnapshot.String)) {
		t.Fatalf("deleted draft audit must preserve valid JSON snapshots: %+v", audits)
	}

	draft, err = store.CreatePlatformAICapabilityDraft(context.Background(), capability.ID, 91, snapshot, "publish me", "req-draft-2")
	if err != nil {
		t.Fatalf("create second draft: %v", err)
	}
	if _, err := store.PublishPlatformAICapabilityVersion(context.Background(), draft.ID, 91, "req-publish"); err != nil {
		t.Fatalf("publish draft: %v", err)
	}
	if err := store.DeletePlatformAICapabilityDraft(context.Background(), draft.ID, 91, "req-delete-published"); !errors.Is(err, ErrCapabilityVersionChanged) {
		t.Fatalf("delete published version error = %v, want ErrCapabilityVersionChanged", err)
	}
}

func TestPlatformAICapabilityDraftRequiresAgentPromptClosure(t *testing.T) {
	for _, capabilityKey := range []string{"ai.chat", "ai.agent_run", "ai.application_analysis"} {
		t.Run(capabilityKey, func(t *testing.T) {
			db := newPlatformAIControlPlaneTestDB(t)
			store := newPlatformAIControlPlaneTestStore(db)
			_, defaultModelID, _ := seedPlatformAIModels(t, db)
			seedPlatformAIAgentPrompt(t, db, 10, 20, "hr_recruiting_agent", "hr_agent")
			seedPlatformAIAgentPrompt(t, db, 11, 21, "custom", "custom")
			capability := seedPlatformAICapability(t, db, capabilityKey, PlatformAIAudienceTenantHR)

			snapshot := mustCapabilitySnapshotJSON(t, capability, []int64{defaultModelID}, defaultModelID)
			var decoded PlatformAICapabilitySnapshot
			if err := json.Unmarshal(snapshot, &decoded); err != nil {
				t.Fatalf("decode snapshot: %v", err)
			}
			decoded.ConfigurationRef.PromptTemplateIDs = []int64{21}
			snapshot, err := json.Marshal(decoded)
			if err != nil {
				t.Fatalf("encode snapshot: %v", err)
			}
			_, err = store.CreatePlatformAICapabilityDraft(context.Background(), capability.ID, 91, snapshot, "invalid closure", "req-draft")
			if err == nil || !strings.Contains(err.Error(), "every released Agent") {
				t.Fatalf("create draft error = %v, want Agent/Prompt closure rejection", err)
			}
		})
	}
}

func TestResolveRuntimeModelHonorsPoolAndFallsBackWithinRelease(t *testing.T) {
	db := newPlatformAIControlPlaneTestDB(t)
	store := newPlatformAIControlPlaneTestStore(db)
	_, defaultModelID, alternateModelID := seedPlatformAIModels(t, db)
	capability := seedPlatformAICapability(t, db, "ai.chat", PlatformAIAudienceTenantHR)
	published := seedPublishedCapabilityVersion(t, db, capability, []int64{defaultModelID, alternateModelID}, defaultModelID)
	governed, err := store.ResolveCapabilityRuntimeModel(context.Background(), "ai.chat", PlatformAIAudienceTenantHR, published.ID, defaultModelID)
	if err != nil {
		t.Fatalf("resolve governed runtime: %v", err)
	}
	if len(governed.ConfigurationRefs.AgentIDs) != 1 || governed.ConfigurationRefs.AgentIDs[0] != 10 || len(governed.ConfigurationRefs.PromptTemplateIDs) != 1 || governed.ConfigurationRefs.PromptTemplateIDs[0] != 20 {
		t.Fatalf("released configuration refs = %+v", governed.ConfigurationRefs)
	}
	if err := store.assertNotReleasedConfiguration(context.Background(), "agent", 10); !errors.Is(err, ErrReleasedConfigurationImmutable) {
		t.Fatalf("released Agent mutation guard error = %v", err)
	}

	selected, err := store.ResolveRuntimeModel(context.Background(), "ai.chat", PlatformAIAudienceTenantHR, published.ID, alternateModelID)
	if err != nil {
		t.Fatalf("resolve selected model: %v", err)
	}
	if selected.EffectiveModelID != alternateModelID || selected.FallbackReason != "" {
		t.Fatalf("selected resolution = %+v", selected)
	}

	if err := db.Model(&llmModelRecord{}).Where("id = ?", alternateModelID).Update("is_enabled", false).Error; err != nil {
		t.Fatalf("disable alternate model: %v", err)
	}
	fallback, err := store.ResolveRuntimeModel(context.Background(), "ai.chat", PlatformAIAudienceTenantHR, published.ID, alternateModelID)
	if err != nil {
		t.Fatalf("resolve fallback model: %v", err)
	}
	if fallback.RequestedModelID != alternateModelID || fallback.EffectiveModelID != defaultModelID || fallback.FallbackReason != ModelFallbackUnavailable {
		t.Fatalf("fallback resolution = %+v", fallback)
	}

	_, err = store.ResolveRuntimeModel(context.Background(), "ai.chat", PlatformAIAudienceTenantHR, published.ID, alternateModelID+1000)
	if !errors.Is(err, ErrModelNotAllowed) {
		t.Fatalf("out-of-pool error = %v, want ErrModelNotAllowed", err)
	}
}

func TestRuntimeModelPoolsAreAudienceIsolated(t *testing.T) {
	db := newPlatformAIControlPlaneTestDB(t)
	store := newPlatformAIControlPlaneTestStore(db)
	_, hrModelID, candidateModelID := seedPlatformAIModels(t, db)
	hrCapability := seedPlatformAICapability(t, db, "ai.chat", PlatformAIAudienceTenantHR)
	candidateCapability := seedPlatformAICapability(t, db, "ai.chat", PlatformAIAudienceCandidate)
	hrVersion := seedPublishedCapabilityVersion(t, db, hrCapability, []int64{hrModelID}, hrModelID)
	seedPublishedCapabilityVersion(t, db, candidateCapability, []int64{candidateModelID}, candidateModelID)

	_, err := store.ResolveRuntimeModel(context.Background(), "ai.chat", PlatformAIAudienceTenantHR, hrVersion.ID, candidateModelID)
	if !errors.Is(err, ErrModelNotAllowed) {
		t.Fatalf("candidate model in HR pool error = %v, want ErrModelNotAllowed", err)
	}

	models, version, err := store.ListAllowedRuntimeModels(context.Background(), "ai.chat", PlatformAIAudienceCandidate, 0)
	if err != nil {
		t.Fatalf("list candidate models: %v", err)
	}
	if len(models) != 1 || models[0].ID != candidateModelID || !models[0].IsDefault || version.CapabilityID != candidateCapability.ID {
		t.Fatalf("candidate models = %+v, version = %+v", models, version)
	}
}

func newPlatformAIControlPlaneTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&platformAICapabilityRecord{},
		&platformAICapabilityVersionRecord{},
		&platformAIConfigAuditRecord{},
		&llmProviderRecord{},
		&llmModelRecord{},
		&agentConfigRecord{},
		&agentCapabilityBindingRecord{},
		&promptTemplateRecord{},
		&agentSkillRecord{},
		&agentSkillVersionRecord{},
		&agentSkillSectionRecord{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func seedPlatformAIAgentPrompt(t *testing.T, db *gorm.DB, agentID, promptID int64, agentType, promptAgentType string) {
	t.Helper()
	prompt := promptTemplateRecord{
		ID: promptID, Name: "prompt", Content: "system", Version: 1, IsActive: true,
		AgentType: promptAgentType, PromptRole: "system",
	}
	if err := db.Create(&prompt).Error; err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	agent := agentConfigRecord{
		ID: agentID, Name: "agent", DisplayName: "Agent", AgentType: agentType,
		PromptTemplateID: sql.NullInt64{Int64: promptID, Valid: true}, MaxIterations: 5, IsEnabled: true,
	}
	if err := db.Create(&agent).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
}

func seedPlatformAIModels(t *testing.T, db *gorm.DB) (int64, int64, int64) {
	t.Helper()
	provider := llmProviderRecord{Name: "provider", BaseURL: "https://example.invalid", APIKeyEncrypted: "encrypted", ProviderType: "openai_compatible", ProtocolType: "openai_chat_completions", AuthType: "bearer", IsEnabled: true}
	if err := db.Create(&provider).Error; err != nil {
		t.Fatalf("create provider: %v", err)
	}
	models := []llmModelRecord{
		{ProviderID: provider.ID, ModelName: "default-model", DisplayName: "Default", Temperature: 0.7, TopP: 1, MaxTokens: 4096, TemperatureEnabled: true, TopPEnabled: true, MaxConcurrency: 10, TimeoutSeconds: 90, IsEnabled: true, IsDefault: true},
		{ProviderID: provider.ID, ModelName: "alternate-model", DisplayName: "Alternate", Temperature: 0.7, TopP: 1, MaxTokens: 4096, TemperatureEnabled: true, TopPEnabled: true, MaxConcurrency: 10, TimeoutSeconds: 90, IsEnabled: true},
	}
	if err := db.Create(&models).Error; err != nil {
		t.Fatalf("create models: %v", err)
	}
	return provider.ID, models[0].ID, models[1].ID
}

func seedPlatformAICapability(t *testing.T, db *gorm.DB, key, audience string) platformAICapabilityRecord {
	t.Helper()
	row := platformAICapabilityRecord{CapabilityKey: key, Audience: audience, Name: key + " " + audience, Status: "active"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create capability: %v", err)
	}
	return row
}

func seedPublishedCapabilityVersion(t *testing.T, db *gorm.DB, capability platformAICapabilityRecord, allowed []int64, defaultID int64) platformAICapabilityVersionRecord {
	t.Helper()
	normalized, hash, _, err := normalizeCapabilitySnapshot(mustCapabilitySnapshotJSON(t, capability, allowed, defaultID), capability)
	if err != nil {
		t.Fatalf("normalize snapshot: %v", err)
	}
	row := platformAICapabilityVersionRecord{CapabilityID: capability.ID, Version: 1, Status: PlatformAIReleasePublished, SnapshotJSON: normalized, SnapshotHash: hash}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create published version: %v", err)
	}
	if err := db.Model(&platformAICapabilityRecord{}).Where("id = ?", capability.ID).Update("current_published_version_id", row.ID).Error; err != nil {
		t.Fatalf("set current version: %v", err)
	}
	return row
}

func mustCapabilitySnapshotJSON(t *testing.T, capability platformAICapabilityRecord, allowed []int64, defaultID int64) []byte {
	t.Helper()
	raw, err := json.Marshal(PlatformAICapabilitySnapshot{
		SchemaVersion: PlatformAISnapshotSchemaVersion,
		CapabilityKey: capability.CapabilityKey,
		Audience:      capability.Audience,
		ModelPolicy: PlatformAIModelPolicy{
			AllowedLLMModelIDs: allowed,
			DefaultLLMModelID:  defaultID,
		},
		ConfigurationRef:   PlatformAIConfigurationRefs{AgentIDs: []int64{10}, PromptTemplateIDs: []int64{20}},
		SkillRuntimePolicy: validPlatformAITestSkillPolicy(),
	})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	return raw
}

func mustCapabilitySnapshotObject(t *testing.T, capability platformAICapabilityRecord, allowed []int64, defaultID int64) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(mustCapabilitySnapshotJSON(t, capability, allowed, defaultID), &result); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	return result
}

func cloneJSONMap(t *testing.T, input map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("encode JSON clone: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode JSON clone: %v", err)
	}
	return result
}

func validPlatformAITestSkillPolicy() PlatformAISkillRuntimePolicy {
	return PlatformAISkillRuntimePolicy{
		PolicyVersion:        PlatformAISkillPolicyVersion,
		MaxSkillTokens:       PlatformAIDefaultMaxSkillTokens,
		MaxInputRatio:        PlatformAIDefaultMaxSkillInputRatio,
		MaxSkills:            PlatformAIDefaultMaxSkills,
		EvaluationSuiteHash:  platformAITestEvaluationSuiteHash,
		EvaluationResultHash: platformAITestEvaluationResultHash,
	}
}

type fixedPlatformAITestEvaluator struct {
	result PlatformAIAgentSkillReleaseEvaluationResult
	err    error
}

func (e fixedPlatformAITestEvaluator) EvaluateAgentSkillRelease(
	_ context.Context,
	_ PlatformAIAgentSkillReleaseEvaluationInput,
) (PlatformAIAgentSkillReleaseEvaluationResult, error) {
	return e.result, e.err
}

func newPlatformAIControlPlaneTestStore(db *gorm.DB) *NativeStore {
	store := NewNativeStore(db)
	store.SetAgentSkillReleaseEvaluator(fixedPlatformAITestEvaluator{
		result: PlatformAIAgentSkillReleaseEvaluationResult{
			Passed:     true,
			SuiteHash:  platformAITestEvaluationSuiteHash,
			ResultHash: platformAITestEvaluationResultHash,
		},
	})
	return store
}

func seedPlatformAIAgentSkillPackage(
	t *testing.T,
	db *gorm.DB,
	name, agentType, scenario string,
	role agentskill.CompositionRole,
	requiredCapabilities []string,
) agentSkillVersionRecord {
	t.Helper()
	compiled, err := agentskill.Compile(agentskill.PackageDraft{
		Manifest: agentskill.Manifest{
			SchemaVersion:        agentskill.SchemaVersion,
			SkillName:            name,
			DisplayName:          name,
			AgentType:            agentType,
			Scenario:             scenario,
			RiskLevel:            agentskill.RiskLevelMedium,
			Composition:          agentskill.Composition{Role: role},
			OutputContract:       agentskill.OutputContract{Mode: agentskill.OutputModeNone},
			RequiredCapabilities: requiredCapabilities,
		},
		Core: agentskill.Core{ContentMarkdown: fmt.Sprintf("Core instructions for %s.", name)},
		Sections: []agentskill.ReferenceSection{{
			SectionKey:      "details",
			Title:           "Details",
			ContentMarkdown: fmt.Sprintf("Reference details for %s.", name),
		}},
	})
	if err != nil {
		t.Fatalf("compile Agent Skill package: %v", err)
	}
	registry := agentSkillRecord{
		Name:              name,
		DisplayName:       name,
		IsEnabled:         true,
		IsManualInvocable: true,
	}
	if err := db.Create(&registry).Error; err != nil {
		t.Fatalf("create Agent Skill registry: %v", err)
	}
	version, err := persistCompiledAgentSkillVersion(db, registry.ID, "v1", "", 1, "", compiled)
	if err != nil {
		t.Fatalf("create Agent Skill version: %v", err)
	}
	if err := db.Model(&agentSkillRecord{}).Where("id = ?", registry.ID).
		Update("current_version_id", version.ID).Error; err != nil {
		t.Fatalf("activate Agent Skill version: %v", err)
	}
	return version
}
