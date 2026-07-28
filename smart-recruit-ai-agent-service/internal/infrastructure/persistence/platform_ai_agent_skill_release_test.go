package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"

	"smart-recruit-ai-agent-service/internal/domain/agentskill"
)

func TestPlatformAICapabilityPublishRevalidatesAgentSkillPackage(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*gorm.DB, agentSkillVersionRecord)
		want    string
	}{
		{name: "valid immutable package"},
		{
			name: "disabled registry",
			prepare: func(db *gorm.DB, version agentSkillVersionRecord) {
				if err := db.Model(&agentSkillRecord{}).Where("id = ?", version.SkillID).
					Update("is_enabled", false).Error; err != nil {
					t.Fatalf("disable registry: %v", err)
				}
			},
			want: "disabled registry",
		},
		{
			name: "compiled hash mismatch",
			prepare: func(db *gorm.DB, version agentSkillVersionRecord) {
				if err := db.Model(&agentSkillVersionRecord{}).Where("id = ?", version.ID).
					Update("compiled_hash", strings.Repeat("f", 64)).Error; err != nil {
					t.Fatalf("corrupt hash: %v", err)
				}
			},
			want: "package hash or canonical content",
		},
		{
			name: "section content mismatch",
			prepare: func(db *gorm.DB, version agentSkillVersionRecord) {
				if err := db.Model(&agentSkillSectionRecord{}).Where("skill_version_id = ?", version.ID).
					Update("content_markdown", "mutated after compilation").Error; err != nil {
					t.Fatalf("corrupt section: %v", err)
				}
			},
			want: "package hash or canonical content",
		},
		{
			name: "section trigger terms corrupt JSON",
			prepare: func(db *gorm.DB, version agentSkillVersionRecord) {
				corruptReleaseAgentSkillSectionJSON(t, db, version.ID, "trigger_terms_json", `{"term":"screen"}`)
			},
			want: "trigger_terms_json must be an array of strings",
		},
		{
			name: "section semantic tags corrupt JSON",
			prepare: func(db *gorm.DB, version agentSkillVersionRecord) {
				corruptReleaseAgentSkillSectionJSON(t, db, version.ID, "semantic_tags_json", `"screen"`)
			},
			want: "semantic_tags_json must be an array of strings",
		},
		{
			name: "section planner intents corrupt JSON",
			prepare: func(db *gorm.DB, version agentSkillVersionRecord) {
				corruptReleaseAgentSkillSectionJSON(t, db, version.ID, "planner_intents_json", `null`)
			},
			want: "planner_intents_json must be an array of strings",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, store, capability, defaultModelID := setupPlatformAIAgentSkillReleaseTest(t, "ai.chat")
			version := seedPlatformAIAgentSkillPackage(
				t,
				db,
				"candidate-screen",
				"hr_recruiting_agent",
				"candidate-screening",
				agentskill.CompositionRolePrimary,
				nil,
			)
			if tt.prepare != nil {
				tt.prepare(db, version)
			}
			snapshot := platformAITestSnapshotWithSkills(t, capability, defaultModelID, version.ID)
			draft, err := store.CreatePlatformAICapabilityDraft(
				context.Background(),
				capability.ID,
				91,
				snapshot,
				"package release",
				"req-draft",
			)
			if err != nil {
				t.Fatalf("create draft: %v", err)
			}
			_, err = store.PublishPlatformAICapabilityVersion(context.Background(), draft.ID, 91, "req-publish")
			if tt.want == "" {
				if err != nil {
					t.Fatalf("publish: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("publish error = %v, want %q", err, tt.want)
			}
			assertPlatformAIDraftUnpublished(t, db, capability.ID, draft.ID)
		})
	}
}

func TestPlatformAICapabilityPublishRequiresSkillAgentCapabilityFit(t *testing.T) {
	tests := []struct {
		name         string
		agentType    string
		requiredCaps []string
		bindCaps     []string
		want         string
	}{
		{
			name:      "wrong Agent type",
			agentType: "candidate_assistant",
			want:      "no released Agent of type",
		},
		{
			name:         "missing required capability",
			agentType:    "hr_recruiting_agent",
			requiredCaps: []string{"search_candidates"},
			want:         "satisfies required capabilities",
		},
		{
			name:         "matching Agent and capability",
			agentType:    "hr_recruiting_agent",
			requiredCaps: []string{"search_candidates"},
			bindCaps:     []string{"search_candidates"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, store, capability, defaultModelID := setupPlatformAIAgentSkillReleaseTest(t, "ai.chat")
			for _, capabilityKey := range tt.bindCaps {
				if err := db.Create(&agentCapabilityBindingRecord{
					AgentID:       10,
					CapabilityKey: capabilityKey,
					IsEnabled:     true,
				}).Error; err != nil {
					t.Fatalf("create capability binding: %v", err)
				}
			}
			version := seedPlatformAIAgentSkillPackage(
				t,
				db,
				"capability-fit",
				tt.agentType,
				"candidate-screening",
				agentskill.CompositionRolePrimary,
				tt.requiredCaps,
			)
			draft, err := store.CreatePlatformAICapabilityDraft(
				context.Background(),
				capability.ID,
				91,
				platformAITestSnapshotWithSkills(t, capability, defaultModelID, version.ID),
				"capability fit",
				"req-draft",
			)
			if err != nil {
				t.Fatalf("create draft: %v", err)
			}
			_, err = store.PublishPlatformAICapabilityVersion(context.Background(), draft.ID, 91, "req-publish")
			if tt.want == "" {
				if err != nil {
					t.Fatalf("publish: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("publish error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidatePublishedAgentSkillComposition(t *testing.T) {
	tests := []struct {
		name     string
		packages []PlatformAIAgentSkillReleasePackage
		want     string
	}{
		{
			name: "one Primary and one Supporting",
			packages: []PlatformAIAgentSkillReleasePackage{
				releasePackage(1, "hr_recruiting_agent", "screening", agentskill.CompositionRolePrimary),
				releasePackage(2, "hr_recruiting_agent", "screening", agentskill.CompositionRoleSupporting),
			},
		},
		{
			name: "two Primary in same composition scope",
			packages: []PlatformAIAgentSkillReleasePackage{
				releasePackage(1, "hr_recruiting_agent", "screening", agentskill.CompositionRolePrimary),
				releasePackage(2, "hr_recruiting_agent", "screening", agentskill.CompositionRolePrimary),
			},
			want: "exceeds one Primary plus one Supporting",
		},
		{
			name: "Supporting without Primary in same composition scope",
			packages: []PlatformAIAgentSkillReleasePackage{
				releasePackage(1, "hr_recruiting_agent", "screening", agentskill.CompositionRoleSupporting),
			},
			want: "requires exactly one Primary",
		},
		{
			name: "Supporting cannot use Primary from another scenario",
			packages: []PlatformAIAgentSkillReleasePackage{
				releasePackage(1, "hr_recruiting_agent", "ranking", agentskill.CompositionRolePrimary),
				releasePackage(2, "hr_recruiting_agent", "screening", agentskill.CompositionRoleSupporting),
			},
			want: "requires exactly one Primary",
		},
		{
			name: "two structured Primary across scenarios",
			packages: []PlatformAIAgentSkillReleasePackage{
				releasePackage(1, "candidate_match_evaluator", "screening", agentskill.CompositionRolePrimary),
				releasePackage(2, "candidate_match_evaluator", "ranking", agentskill.CompositionRolePrimary),
			},
			want: "cannot publish more than one Primary",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgentSkillReleaseComposition(tt.packages)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("validate composition: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("composition error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestDecodeReleaseAgentSkillSectionStringList(t *testing.T) {
	tests := []struct {
		name  string
		value sql.NullString
		want  []string
	}{
		{
			name:  "SQL NULL represents canonical empty list",
			value: sql.NullString{},
			want:  []string{},
		},
		{
			name:  "empty array",
			value: sql.NullString{String: `[]`, Valid: true},
			want:  []string{},
		},
		{
			name:  "string array",
			value: sql.NullString{String: `["screen","rank"]`, Valid: true},
			want:  []string{"screen", "rank"},
		},
		{
			name:  "JSON null",
			value: sql.NullString{String: `null`, Valid: true},
		},
		{
			name:  "object",
			value: sql.NullString{String: `{"term":"screen"}`, Valid: true},
		},
		{
			name:  "string",
			value: sql.NullString{String: `"screen"`, Valid: true},
		},
		{
			name:  "scalar",
			value: sql.NullString{String: `42`, Valid: true},
		},
		{
			name:  "mixed array",
			value: sql.NullString{String: `["screen",null]`, Valid: true},
		},
		{
			name:  "malformed",
			value: sql.NullString{String: `["screen"`, Valid: true},
		},
		{
			name:  "empty stored value",
			value: sql.NullString{String: ` `, Valid: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeReleaseAgentSkillSectionStringList(tt.value)
			if tt.want == nil {
				if err == nil {
					t.Fatalf("decode result = %#v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !equalStrings(got, tt.want) {
				t.Fatalf("decode result = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestValidateStructuredAgentSkillReleaseCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		capabilityKey string
		agentType     string
		required      []string
		want          string
	}{
		{
			name:          "resume extractor accepted",
			capabilityKey: "ai.resume_parse",
			agentType:     "resume_profile_extractor",
		},
		{
			name:          "job requirement extractor accepted",
			capabilityKey: "ai.match_evaluation",
			agentType:     "job_requirement_extractor",
		},
		{
			name:          "candidate evaluator accepted",
			capabilityKey: "ai.match_evaluation",
			agentType:     "candidate_match_evaluator",
		},
		{
			name:          "conversation agent rejected",
			capabilityKey: "ai.resume_parse",
			agentType:     "hr_recruiting_agent",
			want:          "incompatible",
		},
		{
			name:          "structured capability binding rejected",
			capabilityKey: "ai.resume_parse",
			agentType:     "resume_profile_extractor",
			required:      []string{"search_candidates"},
			want:          "cannot require Agent capability bindings",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgentSkillReleaseCompatibility(
				PlatformAICapabilitySnapshot{CapabilityKey: tt.capabilityKey},
				agentskill.Manifest{
					AgentType:            tt.agentType,
					RequiredCapabilities: tt.required,
				},
				nil,
				nil,
			)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("validate structured compatibility: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("compatibility error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestPlatformAICapabilityPublishFailsClosedWithoutMatchingEvaluation(t *testing.T) {
	tests := []struct {
		name        string
		mutateStore func(*NativeStore)
		mutate      func(*PlatformAICapabilitySnapshot)
		want        error
	}{
		{
			name: "missing evaluator",
			mutateStore: func(store *NativeStore) {
				store.SetAgentSkillReleaseEvaluator(nil)
			},
			want: ErrAgentSkillReleaseEvaluationUnavailable,
		},
		{
			name: "missing snapshot hashes",
			mutate: func(snapshot *PlatformAICapabilitySnapshot) {
				snapshot.SkillRuntimePolicy.EvaluationSuiteHash = ""
				snapshot.SkillRuntimePolicy.EvaluationResultHash = ""
			},
			want: ErrAgentSkillReleaseEvaluationUnavailable,
		},
		{
			name: "evaluator rejects",
			mutateStore: func(store *NativeStore) {
				store.SetAgentSkillReleaseEvaluator(fixedPlatformAITestEvaluator{
					result: PlatformAIAgentSkillReleaseEvaluationResult{
						Passed:     false,
						SuiteHash:  platformAITestEvaluationSuiteHash,
						ResultHash: platformAITestEvaluationResultHash,
					},
				})
			},
			want: ErrAgentSkillReleaseEvaluationFailed,
		},
		{
			name: "hash mismatch",
			mutateStore: func(store *NativeStore) {
				store.SetAgentSkillReleaseEvaluator(fixedPlatformAITestEvaluator{
					result: PlatformAIAgentSkillReleaseEvaluationResult{
						Passed:     true,
						SuiteHash:  strings.Repeat("c", 64),
						ResultHash: strings.Repeat("d", 64),
					},
				})
			},
			want: ErrAgentSkillReleaseEvaluationFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, store, capability, defaultModelID := setupPlatformAIAgentSkillReleaseTest(t, "ai.chat")
			if tt.mutateStore != nil {
				tt.mutateStore(store)
			}
			var snapshot PlatformAICapabilitySnapshot
			if err := json.Unmarshal(mustCapabilitySnapshotJSON(t, capability, []int64{defaultModelID}, defaultModelID), &snapshot); err != nil {
				t.Fatalf("decode snapshot: %v", err)
			}
			if tt.mutate != nil {
				tt.mutate(&snapshot)
			}
			raw, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatalf("encode snapshot: %v", err)
			}
			draft, err := store.CreatePlatformAICapabilityDraft(context.Background(), capability.ID, 91, raw, "evaluation", "req-draft")
			if err != nil {
				t.Fatalf("create draft: %v", err)
			}
			_, err = store.PublishPlatformAICapabilityVersion(context.Background(), draft.ID, 91, "req-publish")
			if !errors.Is(err, tt.want) {
				t.Fatalf("publish error = %v, want %v", err, tt.want)
			}
			assertPlatformAIDraftUnpublished(t, db, capability.ID, draft.ID)
		})
	}
}

func TestRetiredCapabilitySnapshotIsHistoricalOnly(t *testing.T) {
	db := newPlatformAIControlPlaneTestDB(t)
	store := newPlatformAIControlPlaneTestStore(db)
	_, defaultModelID, _ := seedPlatformAIModels(t, db)
	capability := seedPlatformAICapability(t, db, "ai.chat", PlatformAIAudienceTenantHR)
	normalized, hash, _, err := normalizeCapabilitySnapshot(
		mustCapabilitySnapshotJSON(t, capability, []int64{defaultModelID}, defaultModelID),
		capability,
	)
	if err != nil {
		t.Fatalf("normalize snapshot: %v", err)
	}
	retired := platformAICapabilityVersionRecord{
		CapabilityID: capability.ID,
		Version:      1,
		Status:       PlatformAIReleaseRetired,
		SnapshotJSON: normalized,
		SnapshotHash: hash,
	}
	if err := db.Create(&retired).Error; err != nil {
		t.Fatalf("create retired release: %v", err)
	}
	if err := db.Model(&platformAICapabilityRecord{}).Where("id = ?", capability.ID).
		Update("current_published_version_id", retired.ID).Error; err != nil {
		t.Fatalf("point capability at retired release: %v", err)
	}
	if _, err := store.ResolveRuntimeModel(
		context.Background(),
		capability.CapabilityKey,
		capability.Audience,
		retired.ID,
		defaultModelID,
	); !errors.Is(err, ErrCapabilityUnavailable) {
		t.Fatalf("resolve retired snapshot error = %v, want ErrCapabilityUnavailable", err)
	}
}

func TestPlatformAICapabilitySnapshotHashIsRevalidated(t *testing.T) {
	t.Run("draft cannot publish after snapshot tampering", func(t *testing.T) {
		db, store, capability, defaultModelID := setupPlatformAIAgentSkillReleaseTest(t, "ai.chat")
		snapshot := mustCapabilitySnapshotJSON(t, capability, []int64{defaultModelID}, defaultModelID)
		draft, err := store.CreatePlatformAICapabilityDraft(
			context.Background(),
			capability.ID,
			91,
			snapshot,
			"immutable snapshot",
			"req-draft",
		)
		if err != nil {
			t.Fatalf("create draft: %v", err)
		}
		var decoded PlatformAICapabilitySnapshot
		if err := json.Unmarshal(snapshot, &decoded); err != nil {
			t.Fatalf("decode snapshot: %v", err)
		}
		decoded.SkillRuntimePolicy.MaxSkillTokens = 2000
		mutated, err := json.Marshal(decoded)
		if err != nil {
			t.Fatalf("encode mutated snapshot: %v", err)
		}
		if err := db.Model(&platformAICapabilityVersionRecord{}).Where("id = ?", draft.ID).
			Update("snapshot_json", string(mutated)).Error; err != nil {
			t.Fatalf("tamper draft snapshot: %v", err)
		}
		if _, err := store.PublishPlatformAICapabilityVersion(
			context.Background(),
			draft.ID,
			91,
			"req-publish",
		); !errors.Is(err, ErrCapabilitySnapshotHash) {
			t.Fatalf("publish error = %v, want ErrCapabilitySnapshotHash", err)
		}
		assertPlatformAIDraftUnpublished(t, db, capability.ID, draft.ID)
	})

	t.Run("runtime rejects published snapshot after hash tampering", func(t *testing.T) {
		db := newPlatformAIControlPlaneTestDB(t)
		store := newPlatformAIControlPlaneTestStore(db)
		_, defaultModelID, _ := seedPlatformAIModels(t, db)
		capability := seedPlatformAICapability(t, db, "ai.chat", PlatformAIAudienceTenantHR)
		published := seedPublishedCapabilityVersion(t, db, capability, []int64{defaultModelID}, defaultModelID)
		if err := db.Model(&platformAICapabilityVersionRecord{}).Where("id = ?", published.ID).
			Update("snapshot_hash", strings.Repeat("f", 64)).Error; err != nil {
			t.Fatalf("tamper published snapshot hash: %v", err)
		}
		if _, err := store.ResolveRuntimeModel(
			context.Background(),
			capability.CapabilityKey,
			capability.Audience,
			published.ID,
			defaultModelID,
		); !errors.Is(err, ErrCapabilitySnapshotHash) {
			t.Fatalf("resolve error = %v, want ErrCapabilitySnapshotHash", err)
		}
	})
}

func setupPlatformAIAgentSkillReleaseTest(
	t *testing.T,
	capabilityKey string,
) (*gorm.DB, *NativeStore, platformAICapabilityRecord, int64) {
	t.Helper()
	db := newPlatformAIControlPlaneTestDB(t)
	store := newPlatformAIControlPlaneTestStore(db)
	_, defaultModelID, _ := seedPlatformAIModels(t, db)
	seedPlatformAIAgentPrompt(t, db, 10, 20, "hr_recruiting_agent", "hr_agent")
	capability := seedPlatformAICapability(t, db, capabilityKey, PlatformAIAudienceTenantHR)
	return db, store, capability, defaultModelID
}

func platformAITestSnapshotWithSkills(
	t *testing.T,
	capability platformAICapabilityRecord,
	defaultModelID int64,
	versionIDs ...int64,
) []byte {
	t.Helper()
	var snapshot PlatformAICapabilitySnapshot
	if err := json.Unmarshal(mustCapabilitySnapshotJSON(t, capability, []int64{defaultModelID}, defaultModelID), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	snapshot.ConfigurationRef.AgentSkillVersionIDs = append([]int64(nil), versionIDs...)
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("encode snapshot: %v", err)
	}
	return raw
}

func assertPlatformAIDraftUnpublished(t *testing.T, db *gorm.DB, capabilityID, draftID int64) {
	t.Helper()
	var version platformAICapabilityVersionRecord
	if err := db.First(&version, draftID).Error; err != nil {
		t.Fatalf("load draft: %v", err)
	}
	if version.Status != PlatformAIReleaseDraft || version.PublishedAt != nil {
		t.Fatalf("failed publication mutated draft: %+v", version)
	}
	var capability platformAICapabilityRecord
	if err := db.First(&capability, capabilityID).Error; err != nil {
		t.Fatalf("load capability: %v", err)
	}
	if capability.CurrentPublishedVersionID.Valid {
		t.Fatalf("failed publication advanced capability pointer: %+v", capability.CurrentPublishedVersionID)
	}
}

func releasePackage(id int64, agentType, scenario string, role agentskill.CompositionRole) PlatformAIAgentSkillReleasePackage {
	return PlatformAIAgentSkillReleasePackage{
		VersionID: id,
		Package: agentskill.CompiledPackage{
			Manifest: agentskill.Manifest{
				AgentType:   agentType,
				Scenario:    scenario,
				Composition: agentskill.Composition{Role: role},
			},
		},
	}
}

func corruptReleaseAgentSkillSectionJSON(
	t *testing.T,
	db *gorm.DB,
	versionID int64,
	column string,
	value string,
) {
	t.Helper()
	switch column {
	case "trigger_terms_json", "semantic_tags_json", "planner_intents_json":
	default:
		t.Fatalf("unsupported Agent Skill Section JSON column %q", column)
	}
	if err := db.Model(&agentSkillSectionRecord{}).
		Where("skill_version_id = ?", versionID).
		Update(column, value).Error; err != nil {
		t.Fatalf("corrupt Agent Skill Section %s: %v", column, err)
	}
}
