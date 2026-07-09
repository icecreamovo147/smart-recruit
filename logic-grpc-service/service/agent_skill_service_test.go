package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/crypto"
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
	if err := db.AutoMigrate(&model.AgentSkill{}, &model.AgentSkillVersion{}, &model.AgentConfig{}, &model.AgentCapabilityBinding{}, &model.AgentToolBinding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewAgentSkillServiceWithAgentConfigRepo(repository.NewAgentSkillRepo(db), repository.NewAgentConfigRepo(db)), db
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

func TestAgentSkillServiceCreateStoresGovernanceMetadata(t *testing.T) {
	svc, db := newAgentSkillTestService(t)

	resp, err := svc.CreateAgentSkill(context.Background(), &pb.CreateAgentSkillRequest{
		Name:                 "match_governance",
		DisplayName:          "Match Governance",
		Description:          "Evaluate candidate match.",
		TriggerKeywords:      []string{"match", "match"},
		AgentType:            "hr_recruiting_agent",
		Category:             "candidate_match",
		Scenario:             "screening",
		Priority:             20,
		RiskLevel:            "high",
		RequiredCapabilities: []string{"builtin:evaluate_candidate_match", "search_jobs"},
		OutputSchema:         `{"type":"object","required":["score"]}`,
		EvaluationCriteria:   []string{"Evidence is cited", "Evidence is cited"},
		SemanticTags:         []string{"resume", "matching"},
	})
	if err != nil {
		t.Fatalf("CreateAgentSkill: %v", err)
	}
	skill := resp.GetSkill()
	if skill.GetAgentType() != "hr_recruiting_agent" || skill.GetCategory() != "candidate_match" || skill.GetScenario() != "screening" {
		t.Fatalf("metadata was not mapped to response: %+v", skill)
	}
	if skill.GetPriority() != 20 || skill.GetRiskLevel() != "high" {
		t.Fatalf("unexpected priority/risk: %+v", skill)
	}
	if got := skill.GetRequiredCapabilities(); len(got) != 2 || got[0] != "builtin:evaluate_candidate_match" || got[1] != "builtin:search_jobs" {
		t.Fatalf("unexpected required capabilities: %#v", got)
	}
	if len(skill.GetEvaluationCriteria()) != 1 || len(skill.GetSemanticTags()) != 2 {
		t.Fatalf("unexpected criteria/tags: criteria=%#v tags=%#v", skill.GetEvaluationCriteria(), skill.GetSemanticTags())
	}

	var stored model.AgentSkill
	if err := db.First(&stored, skill.GetId()).Error; err != nil {
		t.Fatalf("load stored skill: %v", err)
	}
	if stored.RequiredCapabilities == "" || stored.OutputSchema != `{"required":["score"],"type":"object"}` {
		t.Fatalf("metadata was not stored canonically: %+v", stored)
	}
}

func TestAgentSkillServiceDebugSemanticRetrievalFallback(t *testing.T) {
	svc, db := newAgentSkillTestService(t)
	if err := db.AutoMigrate(&model.AIMemory{}, &model.AIEmbedding{}); err != nil {
		t.Fatalf("auto migrate semantic debug tables: %v", err)
	}
	svc.WithSemanticDebugDependencies(repository.NewMemoryRepo(db), NewEmbeddingService(repository.NewAIEmbeddingRepo(db), nil, crypto.EncryptionKey{}))
	ctx := context.Background()
	skill := &model.AgentSkill{
		Name:              "candidate_match_debug",
		DisplayName:       "Candidate Match Debug",
		Description:       "candidate match",
		IsEnabled:         1,
		IsManualInvocable: 1,
		AgentType:         defaultAgentSkillAgentType,
		Category:          "candidate_match",
		SemanticTags:      `["candidate","match"]`,
	}
	version := &model.AgentSkillVersion{Version: "1.0.0", SkillMD: "candidate match", BodyMarkdown: "candidate match"}
	if err := repository.NewAgentSkillRepo(db).CreateSkillWithVersion(ctx, skill, version, true); err != nil {
		t.Fatalf("CreateSkillWithVersion: %v", err)
	}
	if err := repository.NewMemoryRepo(db).Create(ctx, &model.AIMemory{
		HrID:       20,
		ScopeType:  "hr",
		ScopeID:    0,
		MemoryType: "preference",
		Content:    "candidate match summaries should cite risks",
		Source:     "user",
		Confidence: 0.8,
		Importance: 0.9,
	}); err != nil {
		t.Fatalf("Create memory: %v", err)
	}

	resp, err := svc.DebugSemanticRetrieval(ctx, &pb.DebugSemanticRetrievalRequest{HrId: 20, Query: "candidate match", Limit: 5})
	if err != nil {
		t.Fatalf("DebugSemanticRetrieval: %v", err)
	}
	if resp.GetCode() != 0 || resp.GetEmbeddingAvailable() {
		t.Fatalf("unexpected response status: %#v", resp)
	}
	if resp.GetFallbackReason() == "" || len(resp.GetSkills()) != 1 || len(resp.GetMemories()) != 1 {
		t.Fatalf("fallback debug payload incomplete: %#v", resp)
	}
}

func TestAgentSkillServiceDebugSemanticRetrievalEmbeddingAvailableWhenServiceHealthy(t *testing.T) {
	svc, db := newAgentSkillTestService(t)
	if err := db.AutoMigrate(&model.AIMemory{}, &model.AIEmbedding{}); err != nil {
		t.Fatalf("auto migrate semantic debug tables: %v", err)
	}
	// Use a fake provider that returns a real vector (service is healthy),
	// but the ai_embeddings table is empty so Search returns 0 candidates.
	embeddingSvc := NewEmbeddingService(repository.NewAIEmbeddingRepo(db), nil, crypto.EncryptionKey{})
	embeddingSvc.SetProviderForTest(fakeEmbeddingProvider{
		vector: EmbeddingVector{Model: "fake-test", Vector: []float64{0.1, 0.2, 0.3}},
	})
	svc.WithSemanticDebugDependencies(repository.NewMemoryRepo(db), embeddingSvc)
	ctx := context.Background()
	skill := &model.AgentSkill{
		Name:              "candidate_match_debug",
		DisplayName:       "Candidate Match Debug",
		Description:       "candidate match",
		IsEnabled:         1,
		IsManualInvocable: 1,
		AgentType:         defaultAgentSkillAgentType,
		Category:          "candidate_match",
		SemanticTags:      `["candidate","match"]`,
	}
	version := &model.AgentSkillVersion{Version: "1.0.0", SkillMD: "candidate match", BodyMarkdown: "candidate match"}
	if err := repository.NewAgentSkillRepo(db).CreateSkillWithVersion(ctx, skill, version, true); err != nil {
		t.Fatalf("CreateSkillWithVersion: %v", err)
	}

	resp, err := svc.DebugSemanticRetrieval(ctx, &pb.DebugSemanticRetrievalRequest{HrId: 20, Query: "candidate match", Limit: 5})
	if err != nil {
		t.Fatalf("DebugSemanticRetrieval: %v", err)
	}
	if resp.GetCode() != 0 {
		t.Fatalf("unexpected response code: %#v", resp)
	}
	// The embedding service is healthy and the query vector was resolved
	// successfully, so embeddingAvailable should be true even though no
	// candidates exist in the ai_embeddings table yet (e.g. before backfill).
	if !resp.GetEmbeddingAvailable() {
		t.Fatalf("expected embedding_available=true when service is healthy, got false; fallback_reason=%q", resp.GetFallbackReason())
	}
	if resp.GetFallbackReason() != "" {
		t.Fatalf("expected empty fallback_reason when service is healthy, got %q", resp.GetFallbackReason())
	}
}

func TestAgentSkillServiceRejectsUnavailableRequiredCapability(t *testing.T) {
	svc, _ := newAgentSkillTestService(t)

	_, err := svc.CreateAgentSkill(context.Background(), &pb.CreateAgentSkillRequest{
		Name:                 "bad_capability",
		DisplayName:          "Bad Capability",
		Description:          "Invalid capability.",
		RequiredCapabilities: []string{"builtin:not_a_tool"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v (%v)", status.Code(err), err)
	}
	if !strings.Contains(err.Error(), "unavailable required capabilities: builtin:not_a_tool") {
		t.Fatalf("expected unavailable capability details, got %v", err)
	}
}

func TestAgentSkillServiceRejectsCapabilityUnavailableInAgentConfig(t *testing.T) {
	svc, db := newAgentSkillTestService(t)
	cfg := &model.AgentConfig{
		Name:        "limited_hr_agent",
		DisplayName: "Limited HR Agent",
		AgentType:   "hr_recruiting_agent",
		IsEnabled:   1,
		IsDefault:   1,
	}
	if err := db.Create(cfg).Error; err != nil {
		t.Fatalf("create agent config: %v", err)
	}
	if err := db.Create(&model.AgentCapabilityBinding{
		AgentID:          cfg.ID,
		CapabilitySource: "builtin",
		CapabilityKey:    "search_jobs",
		IsEnabled:        1,
	}).Error; err != nil {
		t.Fatalf("create capability binding: %v", err)
	}

	_, err := svc.CreateAgentSkill(context.Background(), &pb.CreateAgentSkillRequest{
		Name:                 "config_rejected_capability",
		DisplayName:          "Config Rejected Capability",
		Description:          "Requires a capability not bound to the default HR agent.",
		RequiredCapabilities: []string{"builtin:evaluate_candidate_match"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v (%v)", status.Code(err), err)
	}
	if !strings.Contains(err.Error(), "unavailable required capabilities: builtin:evaluate_candidate_match") {
		t.Fatalf("expected configured availability details, got %v", err)
	}
}

func TestAgentSkillServiceDoesNotUseNonDefaultConfigCapabilities(t *testing.T) {
	svc, db := newAgentSkillTestService(t)
	defaultCfg := &model.AgentConfig{
		Name:        "default_hr_agent",
		DisplayName: "Default HR Agent",
		AgentType:   "hr_recruiting_agent",
		IsEnabled:   1,
		IsDefault:   1,
	}
	secondaryCfg := &model.AgentConfig{
		Name:        "secondary_hr_agent",
		DisplayName: "Secondary HR Agent",
		AgentType:   "hr_recruiting_agent",
		IsEnabled:   1,
		IsDefault:   0,
	}
	if err := db.Create(defaultCfg).Error; err != nil {
		t.Fatalf("create default agent config: %v", err)
	}
	if err := db.Create(secondaryCfg).Error; err != nil {
		t.Fatalf("create secondary agent config: %v", err)
	}
	if err := db.Create(&model.AgentCapabilityBinding{
		AgentID:          defaultCfg.ID,
		CapabilitySource: "builtin",
		CapabilityKey:    "search_jobs",
		IsEnabled:        1,
	}).Error; err != nil {
		t.Fatalf("create default capability binding: %v", err)
	}
	if err := db.Create(&model.AgentCapabilityBinding{
		AgentID:          secondaryCfg.ID,
		CapabilitySource: "builtin",
		CapabilityKey:    "evaluate_candidate_match",
		IsEnabled:        1,
	}).Error; err != nil {
		t.Fatalf("create secondary capability binding: %v", err)
	}

	_, err := svc.CreateAgentSkill(context.Background(), &pb.CreateAgentSkillRequest{
		Name:                 "non_default_config_rejected",
		DisplayName:          "Non Default Config Rejected",
		Description:          "Requires capability only bound to a secondary config.",
		RequiredCapabilities: []string{"builtin:evaluate_candidate_match"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v (%v)", status.Code(err), err)
	}
}

func TestAgentSkillServiceResponseReportsConfigAwareUnavailableCapabilities(t *testing.T) {
	svc, db := newAgentSkillTestService(t)
	cfg := &model.AgentConfig{
		Name:        "default_hr_agent",
		DisplayName: "Default HR Agent",
		AgentType:   "hr_recruiting_agent",
		IsEnabled:   1,
		IsDefault:   1,
	}
	if err := db.Create(cfg).Error; err != nil {
		t.Fatalf("create agent config: %v", err)
	}
	if err := db.Create(&model.AgentCapabilityBinding{
		AgentID:          cfg.ID,
		CapabilitySource: "builtin",
		CapabilityKey:    "search_jobs",
		IsEnabled:        1,
	}).Error; err != nil {
		t.Fatalf("create capability binding: %v", err)
	}
	required, err := marshalStringList([]string{"builtin:evaluate_candidate_match"})
	if err != nil {
		t.Fatalf("marshal capabilities: %v", err)
	}
	skill := &model.AgentSkill{
		Name:                 "response_config_warning",
		DisplayName:          "Response Config Warning",
		Description:          "Stored before config changed.",
		IsEnabled:            1,
		IsManualInvocable:    1,
		TriggerKeywords:      "[]",
		AgentType:            "hr_recruiting_agent",
		Category:             "general",
		RiskLevel:            "medium",
		RequiredCapabilities: required,
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}

	resp, err := svc.GetAgentSkill(context.Background(), &pb.GetAgentSkillRequest{Id: skill.ID})
	if err != nil {
		t.Fatalf("GetAgentSkill: %v", err)
	}
	if got := resp.GetSkill().GetUnavailableCapabilities(); len(got) != 1 || got[0] != "builtin:evaluate_candidate_match" {
		t.Fatalf("expected config-aware unavailable capability, got %#v", got)
	}
}

func TestAgentSkillServiceUpdateAgentTypeRevalidatesCapabilities(t *testing.T) {
	svc, db := newAgentSkillTestService(t)
	required, err := marshalStringList([]string{"builtin:evaluate_candidate_match"})
	if err != nil {
		t.Fatalf("marshal capabilities: %v", err)
	}
	skill := &model.AgentSkill{
		Name:                 "agent_type_revalidation",
		DisplayName:          "Agent Type Revalidation",
		Description:          "Validate capabilities on agent type changes.",
		IsEnabled:            1,
		IsManualInvocable:    1,
		TriggerKeywords:      "[]",
		AgentType:            "hr_recruiting_agent",
		Category:             "general",
		RiskLevel:            "medium",
		RequiredCapabilities: required,
	}
	if err := db.Create(skill).Error; err != nil {
		t.Fatalf("create skill: %v", err)
	}

	_, err = svc.UpdateAgentSkill(context.Background(), &pb.UpdateAgentSkillRequest{
		Id:           skill.ID,
		AgentType:    "candidate_assistant",
		AgentTypeSet: true,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v (%v)", status.Code(err), err)
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

// TASK-006 验证：DebugSemanticRetrieval 响应填充 score breakdown 与 pool_confidence。

func TestAgentSkillServiceDebugSemanticRetrievalPopulatesScoreBreakdown(t *testing.T) {
	svc, db := newAgentSkillTestService(t)
	if err := db.AutoMigrate(&model.AIMemory{}, &model.AIEmbedding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	embeddingSvc := NewEmbeddingService(repository.NewAIEmbeddingRepo(db), nil, crypto.EncryptionKey{})
	embeddingSvc.SetProviderForTest(fakeEmbeddingProvider{
		vector: EmbeddingVector{Model: "fake-test", Vector: []float64{0.1, 0.2, 0.3}},
	})
	svc.WithSemanticDebugDependencies(repository.NewMemoryRepo(db), embeddingSvc)
	ctx := context.Background()
	skill := &model.AgentSkill{
		Name:              "candidate_match_breakdown",
		DisplayName:       "Candidate Match Breakdown",
		Description:       "candidate match",
		IsEnabled:         1,
		IsManualInvocable: 1,
		AgentType:         defaultAgentSkillAgentType,
		Category:          "candidate_match",
		SemanticTags:      `["candidate","match"]`,
	}
	version := &model.AgentSkillVersion{Version: "1.0.0", SkillMD: "candidate match", BodyMarkdown: "candidate match"}
	if err := repository.NewAgentSkillRepo(db).CreateSkillWithVersion(ctx, skill, version, true); err != nil {
		t.Fatalf("CreateSkillWithVersion: %v", err)
	}
	if err := repository.NewMemoryRepo(db).Create(ctx, &model.AIMemory{
		HrID:       30,
		ScopeType:  "hr",
		ScopeID:    0,
		MemoryType: "preference",
		Content:    "candidate match summaries should cite risks",
		Source:     "user",
		Confidence: 0.8,
		Importance: 0.9,
	}); err != nil {
		t.Fatalf("Create memory: %v", err)
	}

	resp, err := svc.DebugSemanticRetrieval(ctx, &pb.DebugSemanticRetrievalRequest{HrId: 30, Query: "candidate match", Limit: 5})
	if err != nil {
		t.Fatalf("DebugSemanticRetrieval: %v", err)
	}
	if resp.GetCode() != 0 {
		t.Fatalf("code = %d, want 0", resp.GetCode())
	}
	if !resp.GetEmbeddingAvailable() {
		t.Fatalf("expected embedding_available=true, got false; fallback_reason=%q", resp.GetFallbackReason())
	}
	// Skill item：breakdown 字段必须填充
	if len(resp.GetSkills()) != 1 {
		t.Fatalf("skills len = %d, want 1", len(resp.GetSkills()))
	}
	sk := resp.GetSkills()[0]
	if sk.GetScore() == 0 || sk.GetFinalRankScore() == 0 {
		t.Fatalf("Skill score fields should be non-zero: score=%v final_rank_score=%v", sk.GetScore(), sk.GetFinalRankScore())
	}
	if sk.GetScore() != sk.GetFinalRankScore() {
		t.Fatalf("compat Score (=%v) must equal FinalRankScore (=%v) per SDD §3.10", sk.GetScore(), sk.GetFinalRankScore())
	}
	if sk.GetRelevanceMode() == "" {
		t.Fatalf("Skill relevance_mode should be populated")
	}
	if sk.GetPoolRank() < 0 {
		t.Fatalf("Skill pool_rank should be >= 0, got %d", sk.GetPoolRank())
	}
	if sk.GetVectorScore() < 0 || sk.GetLexicalScore() < 0 || sk.GetMetadataScore() < 0 {
		t.Fatalf("breakdown scores should be non-negative: vector=%v lex=%v meta=%v", sk.GetVectorScore(), sk.GetLexicalScore(), sk.GetMetadataScore())
	}
	if sk.GetBusinessBoost() < 1.0 || sk.GetBusinessBoost() > 1.5 {
		t.Fatalf("business_boost out of [1.0, 1.5]: %v", sk.GetBusinessBoost())
	}
	// Memory item：breakdown 字段必须填充
	if len(resp.GetMemories()) != 1 {
		t.Fatalf("memories len = %d, want 1", len(resp.GetMemories()))
	}
	mem := resp.GetMemories()[0]
	if mem.GetScore() == 0 || mem.GetFinalRankScore() == 0 {
		t.Fatalf("Memory score fields should be non-zero: score=%v final_rank_score=%v", mem.GetScore(), mem.GetFinalRankScore())
	}
	if mem.GetScore() != mem.GetFinalRankScore() {
		t.Fatalf("Memory compat Score (=%v) must equal FinalRankScore (=%v)", mem.GetScore(), mem.GetFinalRankScore())
	}
	if mem.GetRelevanceMode() == "" {
		t.Fatalf("Memory relevance_mode should be populated")
	}
	if mem.GetPoolRank() < 1 {
		t.Fatalf("Memory pool_rank should be >= 1, got %d", mem.GetPoolRank())
	}
	// Top-level pool_confidence
	if resp.GetSkillPoolConfidence() == "" {
		t.Fatalf("SkillPoolConfidence should be populated")
	}
	if resp.GetMemoryPoolConfidence() == "" {
		t.Fatalf("MemoryPoolConfidence should be populated")
	}
	// 修复：5 个新 debug 字段（embedding_provider / embedding_model / embedding_dim /
	// candidate_count / query_embedding_latency_ms）必须填充。
	if resp.GetEmbeddingProvider() == "" {
		t.Fatalf("EmbeddingProvider should be populated, got %q", resp.GetEmbeddingProvider())
	}
	if resp.GetEmbeddingProvider() != "fake" {
		t.Fatalf("EmbeddingProvider should be 'fake' (test provider), got %q", resp.GetEmbeddingProvider())
	}
	if resp.GetEmbeddingModel() == "" {
		t.Fatalf("EmbeddingModel should be populated, got %q", resp.GetEmbeddingModel())
	}
	if resp.GetEmbeddingDim() != 3 {
		t.Fatalf("EmbeddingDim should be 3 (test vector has 3 dims), got %d", resp.GetEmbeddingDim())
	}
	if resp.GetCandidateCount() != 0 {
		// 没有 ai_embeddings 行，candidate_count 应为 0
		t.Fatalf("CandidateCount should be 0 (no embeddings in db), got %d", resp.GetCandidateCount())
	}
	if resp.GetQueryEmbeddingLatencyMs() < 0 {
		t.Fatalf("QueryEmbeddingLatencyMs should be >= 0, got %d", resp.GetQueryEmbeddingLatencyMs())
	}
}

func TestAgentSkillServiceDebugSemanticRetrievalFallbackPopulatesBreakdown(t *testing.T) {
	// Embedding 不可用时，降级路径也必须填充 breakdown 字段
	svc, db := newAgentSkillTestService(t)
	if err := db.AutoMigrate(&model.AIMemory{}, &model.AIEmbedding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	// embeddings 留 nil → embedding 不可用
	svc.WithSemanticDebugDependencies(repository.NewMemoryRepo(db), NewEmbeddingService(repository.NewAIEmbeddingRepo(db), nil, crypto.EncryptionKey{}))
	ctx := context.Background()
	skill := &model.AgentSkill{
		Name:              "fallback_skill",
		DisplayName:       "Fallback Skill",
		Description:       "candidate match",
		IsEnabled:         1,
		IsManualInvocable: 1,
		AgentType:         defaultAgentSkillAgentType,
		Category:          "candidate_match",
		SemanticTags:      `["candidate","match"]`,
	}
	version := &model.AgentSkillVersion{Version: "1.0.0", SkillMD: "candidate match", BodyMarkdown: "candidate match"}
	if err := repository.NewAgentSkillRepo(db).CreateSkillWithVersion(ctx, skill, version, true); err != nil {
		t.Fatalf("CreateSkillWithVersion: %v", err)
	}

	resp, err := svc.DebugSemanticRetrieval(ctx, &pb.DebugSemanticRetrievalRequest{HrId: 40, Query: "candidate match", Limit: 5})
	if err != nil {
		t.Fatalf("DebugSemanticRetrieval: %v", err)
	}
	if resp.GetEmbeddingAvailable() {
		t.Fatalf("expected embedding_available=false (no provider)")
	}
	if len(resp.GetSkills()) != 1 {
		t.Fatalf("skills len = %d, want 1", len(resp.GetSkills()))
	}
	sk := resp.GetSkills()[0]
	if sk.GetVectorScore() != 0 {
		t.Fatalf("fallback: vector_score should be 0, got %v", sk.GetVectorScore())
	}
	if sk.GetRelevanceMode() != "lexical_metadata" {
		t.Fatalf("fallback: relevance_mode should be lexical_metadata, got %q", sk.GetRelevanceMode())
	}
	if sk.GetBusinessBoost() < 1.0 || sk.GetBusinessBoost() > 1.5 {
		t.Fatalf("fallback: business_boost out of range: %v", sk.GetBusinessBoost())
	}
	if resp.GetSkillPoolConfidence() == "" {
		t.Fatalf("fallback: SkillPoolConfidence should still be populated")
	}
}

// TASK-FU-001 验证：context-based debug state 隔离。

func TestDebugContextStateIsolation(t *testing.T) {
	// 两个并发 ctx 互相不污染。
	t.Run("concurrent_contexts_isolated", func(t *testing.T) {
		ctx1 := withDebugPoolConfidence(context.Background(), debugPoolConfidenceView{
			SkillPoolConfidence:  RankConfidenceHigh,
			MemoryPoolConfidence: RankConfidenceLow,
		})
		ctx2 := withDebugPoolConfidence(context.Background(), debugPoolConfidenceView{
			SkillPoolConfidence:  RankConfidenceMedium,
			MemoryPoolConfidence: RankConfidenceNone,
		})

		got1 := getDebugPoolConfidenceFromContext(ctx1)
		got2 := getDebugPoolConfidenceFromContext(ctx2)

		if got1.SkillPoolConfidence != RankConfidenceHigh || got1.MemoryPoolConfidence != RankConfidenceLow {
			t.Fatalf("ctx1 got %+v, want high/low", got1)
		}
		if got2.SkillPoolConfidence != RankConfidenceMedium || got2.MemoryPoolConfidence != RankConfidenceNone {
			t.Fatalf("ctx2 got %+v, want medium/none", got2)
		}
		// 互相不污染
		if got1.SkillPoolConfidence == got2.SkillPoolConfidence {
			t.Fatalf("ctx1 and ctx2 should have different values")
		}
	})

	t.Run("memory_rankings_isolated", func(t *testing.T) {
		items1 := []RankedMemoryItem{{Memory: model.AIMemory{ID: 1}}}
		items2 := []RankedMemoryItem{{Memory: model.AIMemory{ID: 2}}, {Memory: model.AIMemory{ID: 3}}}

		ctx1 := withDebugMemoryRankings(context.Background(), items1)
		ctx2 := withDebugMemoryRankings(context.Background(), items2)

		got1 := getDebugMemoryRankingsFromContext(ctx1)
		got2 := getDebugMemoryRankingsFromContext(ctx2)
		if len(got1) != 1 || got1[0].Memory.ID != 1 {
			t.Fatalf("ctx1 got %+v, want 1 item id=1", got1)
		}
		if len(got2) != 2 {
			t.Fatalf("ctx2 got %+v, want 2 items", got2)
		}
	})

	t.Run("simulated_concurrent_debug_requests", func(t *testing.T) {
		// 模拟两个并发请求：ctx1 和 ctx2 各自走完 DebugSemanticRetrieval 的核心流程，
		// pool_confidence 不能互相覆盖。
		var wg sync.WaitGroup
		results := make([]debugPoolConfidenceView, 2)
		for i := 0; i < 2; i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx := context.Background()
				// 给 memory 一个非零 final_rank_score（> rankRelevanceGate = 0.15）
				// 这样 ComputePoolConfidence 会返回 low（top1 >= gate，1 个候选 → low）
				items := []RankedMemoryItem{{
					Memory:  model.AIMemory{ID: uint64(i + 1)},
					Signals: RankingSignals{FinalRankScore: 0.5},
				}}
				ctx = withDebugMemoryRankings(ctx, items)
				selected := []selectedAgentSkill{{FinalRankScore: 0.1, Manual: false}}
				poolView := computeDebugPoolConfidence(selected, getDebugMemoryRankingsFromContext(ctx))
				ctx = withDebugPoolConfidence(ctx, poolView)
				results[i] = getDebugPoolConfidenceFromContext(ctx)
			}()
		}
		wg.Wait()
		// 两个并发请求的 results 应该是独立的（不污染）
		if results[0].MemoryPoolConfidence == RankConfidenceNone {
			t.Fatalf("concurrent ctx should have low pool confidence (1 candidate, top1 > gate), got %v", results[0].MemoryPoolConfidence)
		}
		if results[1].MemoryPoolConfidence == RankConfidenceNone {
			t.Fatalf("concurrent ctx should have low pool confidence, got %v", results[1].MemoryPoolConfidence)
		}
	})
}

func TestDebugContextStateEmptyDefaults(t *testing.T) {
	t.Run("nil_ctx_returns_zero_pool", func(t *testing.T) {
		got := getDebugPoolConfidenceFromContext(context.Background())
		if got.SkillPoolConfidence != RankConfidenceNone || got.MemoryPoolConfidence != RankConfidenceNone {
			t.Fatalf("empty ctx should return zero value, got %+v", got)
		}
	})

	t.Run("nil_ctx_returns_nil_rankings", func(t *testing.T) {
		got := getDebugMemoryRankingsFromContext(context.Background())
		if got != nil {
			t.Fatalf("empty ctx should return nil rankings, got %+v", got)
		}
	})

	t.Run("write_nil_items_stores_empty_slice", func(t *testing.T) {
		// write nil should store empty slice (not nil) to keep type consistent
		ctx := withDebugMemoryRankings(context.Background(), nil)
		got := getDebugMemoryRankingsFromContext(ctx)
		if got == nil {
			t.Fatalf("write nil should store empty slice, got nil")
		}
		if len(got) != 0 {
			t.Fatalf("write nil should store empty slice, got len=%d", len(got))
		}
	})

	t.Run("child_ctx_inherits_parent", func(t *testing.T) {
		parent := withDebugPoolConfidence(context.Background(), debugPoolConfidenceView{
			SkillPoolConfidence: RankConfidenceHigh,
		})
		// 子 ctx 不显式覆盖时，WithValue 链向上查找
		child, cancel := context.WithCancel(parent)
		defer cancel()
		got := getDebugPoolConfidenceFromContext(child)
		if got.SkillPoolConfidence != RankConfidenceHigh {
			t.Fatalf("child ctx should inherit parent's value, got %+v", got)
		}
	})
}
