package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	domainagentskill "smart-recruit-ai-agent-service/internal/domain/agentskill"
	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
)

func TestAgentSkillEmbeddingDocumentsUseImmutableVersionsAndScopedSections(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&agentSkillRecord{}, &agentSkillVersionRecord{}, &agentSkillSectionRecord{}, &aiEmbeddingRecord{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewNativeStore(db)
	ctx := context.Background()

	manifestOne := domainagentskill.Manifest{
		SchemaVersion: 2,
		SkillName:     "resume-screen",
		DisplayName:   "Resume Screen",
		AgentType:     "hr_recruiting_agent",
		Scenario:      "resume",
		RiskLevel:     domainagentskill.RiskLevelMedium,
		Composition:   domainagentskill.Composition{Role: domainagentskill.CompositionRolePrimary},
	}
	manifestTwo := manifestOne
	manifestTwo.SkillName = "interview-plan"
	manifestTwo.DisplayName = "Interview Plan"
	manifestTwo.Scenario = "interview"
	rawOne, _ := json.Marshal(manifestOne)
	rawTwo, _ := json.Marshal(manifestTwo)

	if err := db.Create(&[]agentSkillRecord{
		{ID: 1, Name: "resume-screen", DisplayName: "Resume Screen", IsEnabled: true},
		{ID: 2, Name: "interview-plan", DisplayName: "Interview Plan", IsEnabled: true},
		{ID: 3, Name: "disabled-skill", DisplayName: "Disabled", IsEnabled: false},
	}).Error; err != nil {
		t.Fatalf("seed skills: %v", err)
	}
	if err := db.Create(&[]agentSkillVersionRecord{
		{ID: 101, SkillID: 1, Version: "2.0.0", ManifestJSON: string(rawOne), CoreMarkdown: "resume core", CompiledMarkdown: "resume", CompiledHash: "hash-101", CoreEstimatedTokens: 10},
		{ID: 102, SkillID: 2, Version: "2.0.0", ManifestJSON: string(rawTwo), CoreMarkdown: "interview core", CompiledMarkdown: "interview", CompiledHash: "hash-102", CoreEstimatedTokens: 10},
		{ID: 103, SkillID: 3, Version: "2.0.0", ManifestJSON: string(rawOne), CoreMarkdown: "disabled core", CompiledMarkdown: "disabled", CompiledHash: "hash-103", CoreEstimatedTokens: 10},
	}).Error; err != nil {
		t.Fatalf("seed versions: %v", err)
	}
	if err := db.Create(&[]agentSkillSectionRecord{
		{ID: 1001, SkillVersionID: 101, SectionKey: "resume-rubric", Title: "Resume Rubric", ContentMarkdown: "resume section", TriggerTermsJSON: jsonStringList([]string{"resume"}), SemanticTagsJSON: jsonStringList([]string{"screen"}), PlannerIntentsJSON: jsonStringList([]string{"candidate_review"}), Priority: 10, Ordinal: 1, EstimatedTokens: 20, ContentHash: "section-1001"},
		{ID: 1002, SkillVersionID: 102, SectionKey: "interview-rubric", Title: "Interview Rubric", ContentMarkdown: "interview section", Priority: 5, Ordinal: 1, EstimatedTokens: 20, ContentHash: "section-1002"},
		{ID: 1003, SkillVersionID: 103, SectionKey: "disabled-rubric", Title: "Disabled Rubric", ContentMarkdown: "disabled section", Priority: 5, Ordinal: 1, EstimatedTokens: 20, ContentHash: "section-1003"},
	}).Error; err != nil {
		t.Fatalf("seed sections: %v", err)
	}

	versions, err := store.ListAgentSkillVersionEmbeddingDocuments(ctx, 0, 10)
	if err != nil {
		t.Fatalf("list version documents: %v", err)
	}
	if len(versions) != 2 || versions[0].ID != 101 || versions[0].SkillID != 1 || versions[0].Manifest.SkillName != "resume-screen" || versions[1].ID != 102 {
		t.Fatalf("version documents = %+v", versions)
	}

	sections, err := store.ListAgentSkillSectionEmbeddingDocuments(ctx, 0, []int64{101}, 10)
	if err != nil {
		t.Fatalf("list scoped section documents: %v", err)
	}
	if len(sections) != 1 || sections[0].ID != 1001 || sections[0].VersionID != 101 || len(sections[0].TriggerTerms) != 1 {
		t.Fatalf("scoped section documents = %+v", sections)
	}

	for _, row := range []embeddinginfra.AIEmbeddingRecord{
		{ObjectType: "agent_skill_version", ObjectID: 101, ScopeType: "agent_skill_version", ScopeID: 101, TextHash: "version-global", EmbeddingModel: "model", Vector: []float64{1}, Status: "ready"},
		{ObjectType: "agent_skill_section", ObjectID: 1001, ScopeType: "agent_skill_version", ScopeID: 101, TextHash: "a", EmbeddingModel: "model", Vector: []float64{1}, Status: "ready"},
		{ObjectType: "agent_skill_section", ObjectID: 1002, ScopeType: "agent_skill_version", ScopeID: 102, TextHash: "b", EmbeddingModel: "model", Vector: []float64{1}, Status: "ready"},
	} {
		if err := store.UpsertAIEmbedding(ctx, row); err != nil {
			t.Fatalf("upsert embedding: %v", err)
		}
	}
	tenantID := int64(77)
	if err := db.Create(&[]aiEmbeddingRecord{
		{
			TenantID:       &tenantID,
			ObjectType:     "agent_skill_version",
			ObjectID:       102,
			ScopeType:      "agent_skill_version",
			ScopeID:        102,
			TextHash:       "version-tenant",
			EmbeddingModel: "model",
			EmbeddingDim:   1,
			VectorJSON:     sql.NullString{String: "[1]", Valid: true},
			Status:         "ready",
		},
		{
			TenantID:       &tenantID,
			ObjectType:     "agent_skill_section",
			ObjectID:       1004,
			ScopeType:      "agent_skill_version",
			ScopeID:        101,
			TextHash:       "section-tenant",
			EmbeddingModel: "model",
			EmbeddingDim:   1,
			VectorJSON:     sql.NullString{String: "[1]", Valid: true},
			Status:         "ready",
		},
	}).Error; err != nil {
		t.Fatalf("seed tenant embeddings: %v", err)
	}

	catalogEmbeddings, err := store.ListAIEmbeddings(ctx, "agent_skill_version", "model", 10)
	if err != nil {
		t.Fatalf("list catalog embeddings: %v", err)
	}
	if len(catalogEmbeddings) != 1 || catalogEmbeddings[0].ObjectID != 101 {
		t.Fatalf("catalog embeddings must exclude tenant-owned rows: %+v", catalogEmbeddings)
	}

	embeddings, err := store.ListAIEmbeddingsByScopeIDs(ctx, "agent_skill_section", "model", "agent_skill_version", []int64{101}, 10)
	if err != nil {
		t.Fatalf("list scoped embeddings: %v", err)
	}
	if len(embeddings) != 1 || embeddings[0].ObjectID != 1001 || embeddings[0].ScopeID != 101 {
		t.Fatalf("scoped embeddings must exclude tenant-owned rows: %+v", embeddings)
	}
}

func jsonStringList(values []string) sql.NullString {
	raw, _ := json.Marshal(values)
	return sql.NullString{String: string(raw), Valid: true}
}
