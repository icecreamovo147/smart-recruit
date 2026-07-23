package service

import (
	"context"
	"testing"
	"time"

	"smart-recruit-ai-agent-service/internal/application/command"
	"smart-recruit-ai-agent-service/internal/domain/model"
)

func TestMCPPolicyServiceEvaluatesRepositoryPolicy(t *testing.T) {
	repo := &fakeMCPPolicyRepo{
		policy: &model.MCPToolPolicy{
			ID:                     9,
			Enabled:                true,
			AllowedRoles:           []string{"hr_admin"},
			RequireConfirmation:    true,
			RateLimitWindowSeconds: 60,
			RateLimitMaxCalls:      2,
		},
		count: 2,
	}
	service := NewMCPPolicyService(MCPPolicyDeps{Policies: repo, Now: fixedCapabilityNow})
	result, err := service.Evaluate(context.Background(), command.EvaluateMCPToolPolicy{ServerID: 1, ToolName: "search", CallerRole: "hr_admin"})
	if err != nil {
		t.Fatalf("Evaluate returned %v", err)
	}
	if result.Evaluation.Decision != model.MCPPolicyDecisionRateLimited {
		t.Fatalf("result = %+v", result)
	}
	if repo.since.IsZero() {
		t.Fatal("expected rate limit lookup window")
	}
}

func TestSkillServiceCreatesVersionSelectsAndAudits(t *testing.T) {
	repo := &fakeSkillRepo{skills: []model.AgentSkill{{ID: 7, Name: "match", AgentType: "hr", Enabled: true, Priority: 9, Category: "candidate"}}}
	service, err := NewSkillService(SkillDeps{Skills: repo, Audit: fakeAudit{}, Now: fixedCapabilityNow})
	if err != nil {
		t.Fatalf("NewSkillService returned %v", err)
	}
	selected, err := service.Select(context.Background(), command.SelectAgentSkills{
		AgentType: "hr",
		Question:  "请分析候选人和岗位匹配度",
	})
	if err != nil {
		t.Fatalf("Select returned %v", err)
	}
	if len(selected.Selected) != 1 || selected.Selected[0].ID != 7 {
		t.Fatalf("selected = %+v", selected)
	}

	version, err := service.CreateVersion(context.Background(), command.CreateSkillVersion{
		ActorID:  1,
		SkillID:  7,
		Current:  1,
		Changed:  true,
		Activate: true,
		Manifest: model.SkillManifest{
			Name:        "candidate_match",
			Version:     "1.0.1",
			RuntimeType: model.SkillRuntimePrompt,
		},
		Content: "match candidates",
	})
	if err != nil {
		t.Fatalf("CreateVersion returned %v", err)
	}
	if !version.VersionCreated || !version.Activated || repo.activatedVersion != 2 {
		t.Fatalf("version = %+v activated=%d", version, repo.activatedVersion)
	}
}

func TestEmbeddingAndCandidateMatchServices(t *testing.T) {
	embeddingSvc := NewEmbeddingRuntimeService(fakeEmbeddingConfig{
		provider: &model.EmbeddingProviderConfig{ID: 1, Name: "bailian", ProviderType: "bailian", Endpoint: "https://example.com", Enabled: true, HasEncryptedCredential: true},
		model:    &model.EmbeddingModelConfig{ID: 2, ProviderID: 1, ModelName: "embedding", Dimension: 1024, Enabled: true},
	})
	state, err := embeddingSvc.Resolve(context.Background(), command.ResolveEmbeddingRuntime{})
	if err != nil {
		t.Fatalf("Resolve returned %v", err)
	}
	if state.State.Status != model.EmbeddingStatusAvailable {
		t.Fatalf("state = %+v", state)
	}

	repo := &fakeCandidateMatchRepo{}
	matchSvc := NewCandidateMatchService(repo)
	result, err := matchSvc.Aggregate(context.Background(), command.AggregateCandidateMatch{
		ApplicationID: 5,
		Persist:       true,
		Profile: model.CandidateMatchProfile{Requirements: []model.CandidateRequirement{
			{ID: "go", Label: "Go", Priority: model.RequirementMustHave, Category: model.RequirementCoreSkill, Weight: 1},
		}},
		Results: []model.CandidateRequirementResult{{RequirementID: "go", Status: model.MatchStatusStrongMatch, Score: 95}},
	})
	if err != nil {
		t.Fatalf("Aggregate returned %v", err)
	}
	if !result.Persisted || repo.applicationID != 5 || result.Aggregation.OverallScore <= 0 {
		t.Fatalf("result = %+v repo=%+v", result, repo)
	}
}

func fixedCapabilityNow() time.Time {
	return time.Date(2026, 7, 13, 12, 30, 0, 0, time.UTC)
}

type fakeMCPPolicyRepo struct {
	policy *model.MCPToolPolicy
	count  int64
	since  time.Time
}

func (f *fakeMCPPolicyRepo) GetEnabledToolPolicy(context.Context, uint64, string) (*model.MCPToolPolicy, error) {
	return f.policy, nil
}

func (f *fakeMCPPolicyRepo) CountToolCallsSince(_ context.Context, _ uint64, _ string, since time.Time) (int64, error) {
	f.since = since
	return f.count, nil
}

type fakeSkillRepo struct {
	skills           []model.AgentSkill
	versions         []model.SkillVersion
	activatedSkill   uint64
	activatedVersion int64
}

func (f *fakeSkillRepo) ListEnabledAgentSkills(context.Context) ([]model.AgentSkill, error) {
	return f.skills, nil
}

func (f *fakeSkillRepo) CreateSkillVersion(_ context.Context, version model.SkillVersion) (*model.SkillVersion, error) {
	f.versions = append(f.versions, version)
	return &version, nil
}

func (f *fakeSkillRepo) ActivateSkillVersion(_ context.Context, skillID uint64, version int64) error {
	f.activatedSkill = skillID
	f.activatedVersion = version
	return nil
}

type fakeEmbeddingConfig struct {
	provider *model.EmbeddingProviderConfig
	model    *model.EmbeddingModelConfig
}

func (f fakeEmbeddingConfig) GetDefaultEmbeddingProvider(context.Context) (*model.EmbeddingProviderConfig, error) {
	return f.provider, nil
}

func (f fakeEmbeddingConfig) GetDefaultEmbeddingModel(context.Context, uint64) (*model.EmbeddingModelConfig, error) {
	return f.model, nil
}

type fakeCandidateMatchRepo struct {
	applicationID uint64
	aggregation   model.CandidateMatchAggregation
}

func (f *fakeCandidateMatchRepo) SaveCandidateMatchAggregation(_ context.Context, applicationID uint64, aggregation model.CandidateMatchAggregation) error {
	f.applicationID = applicationID
	f.aggregation = aggregation
	return nil
}
