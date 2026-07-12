package service

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-domain-go/model"
	"smart-recruit-domain-go/pkg/crypto"
	"smart-recruit-domain-go/repository"
)

func newEmbeddingBackfillTestService(t *testing.T) (*EmbeddingBackfillService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentSkill{}, &model.AgentSkillVersion{}, &model.AIMemory{}, &model.AIEmbedding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	embeddingRepo := repository.NewAIEmbeddingRepo(db)
	embeddingSvc := NewEmbeddingService(embeddingRepo, nil, crypto.EncryptionKey{})
	embeddingSvc.SetProviderForTest(fakeEmbeddingProvider{
		vector: EmbeddingVector{Model: "fake-test", Vector: []float64{0.1, 0.2, 0.3}},
	})
	svc := NewEmbeddingBackfillService(
		db,
		embeddingSvc,
		repository.NewAgentSkillRepo(db),
		repository.NewMemoryRepo(db),
	)
	return svc, db
}

func TestEmbeddingBackfillServiceSingleSkillNotFound(t *testing.T) {
	svc, _ := newEmbeddingBackfillTestService(t)
	ctx := context.Background()

	result, err := svc.Run(ctx, BackfillInput{
		ObjectType: "agent_skill",
		ObjectID:   999,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.SuccessCount != 0 || result.FailedCount != 0 || result.SkippedCount != 1 {
		t.Fatalf("result = %+v, want skipped single-object no-match", result)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected error reason for missing skill")
	}
}

func TestEmbeddingBackfillServiceBatchBackfillStillCompatible(t *testing.T) {
	svc, db := newEmbeddingBackfillTestService(t)
	ctx := context.Background()
	skill := &model.AgentSkill{
		Name:              "batch_skill",
		DisplayName:       "Batch Skill",
		Description:       "batch backfill",
		IsEnabled:         1,
		IsManualInvocable: 1,
		AgentType:         defaultAgentSkillAgentType,
	}
	version := &model.AgentSkillVersion{Version: "1.0.0", SkillMD: "batch", BodyMarkdown: "batch backfill content"}
	if err := repository.NewAgentSkillRepo(db).CreateSkillWithVersion(ctx, skill, version, true); err != nil {
		t.Fatalf("CreateSkillWithVersion: %v", err)
	}

	result, err := svc.Run(ctx, BackfillInput{ObjectType: "agent_skill"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.SuccessCount != 1 {
		t.Fatalf("result = %+v, want one successful backfill", result)
	}
}
