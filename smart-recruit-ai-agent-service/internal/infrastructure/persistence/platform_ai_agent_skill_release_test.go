package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"smart-recruit-ai-agent-service/internal/domain/agentskill"
)

type capturingPlatformAITestEvaluator struct {
	input PlatformAIAgentSkillReleaseEvaluationInput
}

func (e *capturingPlatformAITestEvaluator) EvaluateAgentSkillRelease(
	_ context.Context,
	input PlatformAIAgentSkillReleaseEvaluationInput,
) (PlatformAIAgentSkillReleaseEvaluationResult, error) {
	e.input = input
	return PlatformAIAgentSkillReleaseEvaluationResult{
		Passed:     true,
		SuiteHash:  strings.Repeat("a", 64),
		ResultHash: strings.Repeat("b", 64),
	}, nil
}

func TestEvaluateAgentSkillReleasePassesImmutableSnapshotPolicy(t *testing.T) {
	store := &NativeStore{}
	evaluator := &capturingPlatformAITestEvaluator{}
	store.SetAgentSkillReleaseEvaluator(evaluator)
	policy := PlatformAISkillRuntimePolicy{
		PolicyVersion:        PlatformAISkillPolicyVersion,
		MaxSkillTokens:       1777,
		MaxInputRatio:        0.123,
		MaxSkills:            2,
		EvaluationSuiteHash:  strings.Repeat("c", 64),
		EvaluationResultHash: strings.Repeat("d", 64),
	}
	snapshot := PlatformAICapabilitySnapshot{
		CapabilityKey:      "ai.agent_run",
		Audience:           PlatformAIAudienceTenantHR,
		SkillRuntimePolicy: policy,
	}
	packages := []PlatformAIAgentSkillReleasePackage{{
		SkillID:   41,
		VersionID: 73,
	}}

	if _, err := store.evaluateAgentSkillRelease(context.Background(), snapshot, packages); err != nil {
		t.Fatalf("evaluate release: %v", err)
	}
	if !reflect.DeepEqual(evaluator.input.Policy, policy) {
		t.Fatalf("evaluation policy = %+v, want immutable snapshot policy %+v", evaluator.input.Policy, policy)
	}
	if evaluator.input.CapabilityKey != snapshot.CapabilityKey ||
		evaluator.input.Audience != snapshot.Audience ||
		!reflect.DeepEqual(evaluator.input.Packages, packages) {
		t.Fatalf("evaluation input = %+v, want snapshot/packages preserved", evaluator.input)
	}
}

func TestValidatePublishedAgentSkillPackagesLoadsCanonicalPackageFromGORM(t *testing.T) {
	db, _, capability, defaultModelID := setupPlatformAIAgentSkillReleaseTest(t, "ai.chat")
	version := seedPlatformAIAgentSkillPackage(
		t,
		db,
		"gorm-release-package",
		"hr_recruiting_agent",
		"candidate-screening",
		agentskill.CompositionRolePrimary,
		nil,
	)
	if version.ID == 0 || version.SkillID == 0 {
		t.Fatalf("seeded version must have nonzero IDs: %+v", version)
	}
	distinctCreatedAt := time.Date(2026, time.July, 28, 9, 8, 7, 0, time.UTC)
	if err := db.Model(&agentSkillVersionRecord{}).
		Where("id = ?", version.ID).
		Updates(map[string]any{
			"version":        "v7-distinct",
			"authoring_json": `{"canvas":"distinct-authoring"}`,
			"change_note":    "distinct release note",
			"created_by":     int64(9087),
			"created_at":     distinctCreatedAt,
		}).Error; err != nil {
		t.Fatalf("seed distinctive version metadata: %v", err)
	}
	var persistedVersion agentSkillVersionRecord
	if err := db.First(&persistedVersion, version.ID).Error; err != nil {
		t.Fatalf("load persisted version: %v", err)
	}
	rows, err := loadPublishedAgentSkillVersionRows(db, []int64{version.ID})
	if err != nil {
		t.Fatalf("load published version rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("published version rows = %d, want 1", len(rows))
	}
	if !rows[0].RegistryEnabled {
		t.Fatal("published version registry must be enabled")
	}
	if got := rows[0].versionRecord(); !reflect.DeepEqual(got, persistedVersion) {
		t.Fatalf("scanned version = %+v, want full persisted record %+v", got, persistedVersion)
	}

	var snapshot PlatformAICapabilitySnapshot
	if err := json.Unmarshal(
		platformAITestSnapshotWithSkills(t, capability, defaultModelID, version.ID),
		&snapshot,
	); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	packages, err := validatePublishedAgentSkillPackages(db, snapshot)
	if err != nil {
		t.Fatalf("validate published package: %v", err)
	}
	if len(packages) != 1 {
		t.Fatalf("packages = %d, want 1", len(packages))
	}
	pkg := packages[0]
	if pkg.SkillID != version.SkillID || pkg.VersionID != version.ID {
		t.Fatalf(
			"package IDs = skill %d version %d, want skill %d version %d",
			pkg.SkillID,
			pkg.VersionID,
			version.SkillID,
			version.ID,
		)
	}
	if pkg.Package.Manifest.SkillName != "gorm-release-package" ||
		pkg.Package.Manifest.AgentType != "hr_recruiting_agent" ||
		pkg.Package.Manifest.Scenario != "candidate-screening" {
		t.Fatalf("manifest = %+v", pkg.Package.Manifest)
	}
	if pkg.Package.Core.ContentMarkdown != "Core instructions for gorm-release-package.\n" {
		t.Fatalf("core markdown = %q", pkg.Package.Core.ContentMarkdown)
	}
	if pkg.Package.CompiledHash != version.CompiledHash {
		t.Fatalf("compiled hash = %q, want %q", pkg.Package.CompiledHash, version.CompiledHash)
	}
	if len(pkg.Package.Sections) != 1 ||
		pkg.Package.Sections[0].SectionKey != "details" ||
		pkg.Package.Sections[0].ContentMarkdown != "Reference details for gorm-release-package.\n" {
		t.Fatalf("sections = %+v", pkg.Package.Sections)
	}
}

func TestValidatePublishedAgentSkillPackagesAcceptsSemanticallyEquivalentManifestJSON(t *testing.T) {
	db, _, capability, defaultModelID := setupPlatformAIAgentSkillReleaseTest(t, "ai.chat")
	version := seedPlatformAIAgentSkillPackage(
		t,
		db,
		"mysql-json-normalized-package",
		"hr_recruiting_agent",
		"candidate-screening",
		agentskill.CompositionRolePrimary,
		nil,
	)

	var manifest map[string]any
	if err := json.Unmarshal([]byte(version.ManifestJSON), &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	mysqlStyleJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("format semantically equivalent manifest: %v", err)
	}
	if string(mysqlStyleJSON) == version.ManifestJSON {
		t.Fatal("test fixture must differ from the compiler JSON representation")
	}
	if err := db.Model(&agentSkillVersionRecord{}).
		Where("id = ?", version.ID).
		Update("manifest_json", string(mysqlStyleJSON)).Error; err != nil {
		t.Fatalf("simulate MySQL JSON normalization: %v", err)
	}

	var snapshot PlatformAICapabilitySnapshot
	if err := json.Unmarshal(
		platformAITestSnapshotWithSkills(t, capability, defaultModelID, version.ID),
		&snapshot,
	); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	packages, err := validatePublishedAgentSkillPackages(db, snapshot)
	if err != nil {
		t.Fatalf("validate semantically equivalent manifest: %v", err)
	}
	if len(packages) != 1 || packages[0].Package.CompiledHash != version.CompiledHash {
		t.Fatalf("validated packages = %+v, want immutable package hash %q", packages, version.CompiledHash)
	}
}

func TestPlatformAIErrorMessageKeyIsSpecificAndSafe(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int32
		wantKey  string
	}{
		{
			name:     "invalid release configuration",
			err:      errors.New("sensitive internal validation detail"),
			wantCode: 400,
			wantKey:  "common.invalid_request",
		},
		{
			name:     "evaluation unavailable",
			err:      fmt.Errorf("wrapped: %w", ErrAgentSkillReleaseEvaluationUnavailable),
			wantCode: 503,
			wantKey:  "ai.unavailable",
		},
		{
			name:     "evaluation failed",
			err:      fmt.Errorf("wrapped: %w", ErrAgentSkillReleaseEvaluationFailed),
			wantCode: 400,
			wantKey:  "ai.agent_skill_package_invalid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _ := platformAIErrorCode(tt.err)
			if code != tt.wantCode {
				t.Fatalf("error code = %d, want %d", code, tt.wantCode)
			}
			if got := platformAIErrorMessageKey(tt.err, code); got != tt.wantKey {
				t.Fatalf("message key = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

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
			name: "missing version",
			prepare: func(db *gorm.DB, version agentSkillVersionRecord) {
				if err := db.Where("skill_version_id = ?", version.ID).
					Delete(&agentSkillSectionRecord{}).Error; err != nil {
					t.Fatalf("delete version sections: %v", err)
				}
				if err := db.Delete(&agentSkillVersionRecord{}, version.ID).Error; err != nil {
					t.Fatalf("delete version: %v", err)
				}
			},
			want: "all released agent skill versions must exist",
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
			if tt.prepare != nil {
				tt.prepare(db, version)
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

func TestPlatformAICapabilityDraftRequiresSkillAgentCapabilityFit(t *testing.T) {
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
			if tt.want != "" {
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("create draft error = %v, want %q", err, tt.want)
				}
				return
			}
			if err != nil {
				t.Fatalf("create draft: %v", err)
			}
			if _, err := store.PublishPlatformAICapabilityVersion(context.Background(), draft.ID, 91, "req-publish"); err != nil {
				t.Fatalf("publish: %v", err)
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
			var snapshot PlatformAICapabilitySnapshot
			if err := json.Unmarshal(mustCapabilitySnapshotJSON(t, capability, []int64{defaultModelID}, defaultModelID), &snapshot); err != nil {
				t.Fatalf("decode snapshot: %v", err)
			}
			raw, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatalf("encode snapshot: %v", err)
			}
			draft, err := store.CreatePlatformAICapabilityDraft(context.Background(), capability.ID, 91, raw, "evaluation", "req-draft")
			if err != nil {
				t.Fatalf("create draft: %v", err)
			}
			if tt.mutateStore != nil {
				tt.mutateStore(store)
			}
			_, err = store.PublishPlatformAICapabilityVersion(context.Background(), draft.ID, 91, "req-publish")
			if !errors.Is(err, tt.want) {
				t.Fatalf("publish error = %v, want %v", err, tt.want)
			}
			assertPlatformAIDraftUnpublished(t, db, capability.ID, draft.ID)
		})
	}
}

func TestPlatformAICapabilityDraftComputesEvaluationHashesAndPublishesWithProductionEvaluator(t *testing.T) {
	db, _, capability, defaultModelID := setupPlatformAIAgentSkillReleaseTest(t, "ai.chat")
	store := NewNativeStore(db)
	version := seedPlatformAIAgentSkillPackage(
		t,
		db,
		"candidate-screen-evaluation",
		"hr_recruiting_agent",
		"candidate-screening",
		agentskill.CompositionRolePrimary,
		nil,
	)
	var snapshot PlatformAICapabilitySnapshot
	if err := json.Unmarshal(platformAITestSnapshotWithSkills(t, capability, defaultModelID, version.ID), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	snapshot.SkillRuntimePolicy.EvaluationSuiteHash = ""
	snapshot.SkillRuntimePolicy.EvaluationResultHash = ""
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("encode snapshot: %v", err)
	}
	draft, err := store.CreatePlatformAICapabilityDraft(
		context.Background(),
		capability.ID,
		91,
		raw,
		"server evaluated draft",
		"req-evaluated-draft",
	)
	if err != nil {
		t.Fatalf("create evaluated draft: %v", err)
	}
	var stored PlatformAICapabilitySnapshot
	if err := json.Unmarshal([]byte(draft.SnapshotJSON), &stored); err != nil {
		t.Fatalf("decode stored snapshot: %v", err)
	}
	if len(stored.SkillRuntimePolicy.EvaluationSuiteHash) != 64 ||
		len(stored.SkillRuntimePolicy.EvaluationResultHash) != 64 {
		t.Fatalf("stored evaluation hashes = %+v", stored.SkillRuntimePolicy)
	}
	if _, err := store.PublishPlatformAICapabilityVersion(
		context.Background(),
		draft.ID,
		91,
		"req-evaluated-publish",
	); err != nil {
		t.Fatalf("publish evaluated draft: %v", err)
	}
}

func TestPlatformAICapabilityDraftFailsClosedWhenEvaluationUnavailable(t *testing.T) {
	_, store, capability, defaultModelID := setupPlatformAIAgentSkillReleaseTest(t, "ai.chat")
	store.SetAgentSkillReleaseEvaluator(nil)
	var snapshot PlatformAICapabilitySnapshot
	if err := json.Unmarshal(mustCapabilitySnapshotJSON(t, capability, []int64{defaultModelID}, defaultModelID), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	snapshot.SkillRuntimePolicy.EvaluationSuiteHash = ""
	snapshot.SkillRuntimePolicy.EvaluationResultHash = ""
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("encode snapshot: %v", err)
	}
	if _, err := store.CreatePlatformAICapabilityDraft(
		context.Background(),
		capability.ID,
		91,
		raw,
		"missing evaluator",
		"req-missing-evaluator",
	); !errors.Is(err, ErrAgentSkillReleaseEvaluationUnavailable) {
		t.Fatalf("create draft error = %v, want %v", err, ErrAgentSkillReleaseEvaluationUnavailable)
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
	t.Run("MySQL JSON normalization is accepted by publish and runtime", func(t *testing.T) {
		db, store, capability, defaultModelID := setupPlatformAIAgentSkillReleaseTest(t, "ai.chat")
		snapshot := mustCapabilitySnapshotJSON(t, capability, []int64{defaultModelID}, defaultModelID)
		draft, err := store.CreatePlatformAICapabilityDraft(
			context.Background(),
			capability.ID,
			91,
			snapshot,
			"MySQL JSON normalization",
			"req-draft",
		)
		if err != nil {
			t.Fatalf("create draft: %v", err)
		}
		var semanticSnapshot map[string]any
		if err := json.Unmarshal([]byte(draft.SnapshotJSON), &semanticSnapshot); err != nil {
			t.Fatalf("decode prepared snapshot: %v", err)
		}
		mysqlStyleJSON, err := json.MarshalIndent(semanticSnapshot, "", "  ")
		if err != nil {
			t.Fatalf("format semantically equivalent snapshot: %v", err)
		}
		if string(mysqlStyleJSON) == draft.SnapshotJSON {
			t.Fatal("test fixture must differ from the canonical snapshot representation")
		}
		if err := db.Model(&platformAICapabilityVersionRecord{}).Where("id = ?", draft.ID).
			Update("snapshot_json", string(mysqlStyleJSON)).Error; err != nil {
			t.Fatalf("simulate MySQL JSON normalization: %v", err)
		}

		published, err := store.PublishPlatformAICapabilityVersion(
			context.Background(),
			draft.ID,
			91,
			"req-publish",
		)
		if err != nil {
			t.Fatalf("publish normalized snapshot: %v", err)
		}
		resolution, err := store.ResolveRuntimeModel(
			context.Background(),
			capability.CapabilityKey,
			capability.Audience,
			published.ID,
			defaultModelID,
		)
		if err != nil {
			t.Fatalf("resolve normalized published snapshot: %v", err)
		}
		if resolution.SnapshotHash != draft.SnapshotHash {
			t.Fatalf(
				"runtime snapshot hash = %q, want %q",
				resolution.SnapshotHash,
				draft.SnapshotHash,
			)
		}
	})

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
