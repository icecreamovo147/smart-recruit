package policy

import (
	"errors"
	"testing"

	"smart-recruit-ai-agent-service/internal/domain/model"
)

func TestEvaluateMCPToolPolicyPreservesDecisionOrder(t *testing.T) {
	min := 1.0
	policyModel := &model.MCPToolPolicy{
		ID:                     3,
		Enabled:                true,
		AllowedRoles:           []string{"hr_admin"},
		AllowedScopes:          []string{"tenant:1"},
		RequiredArgs:           []string{"query"},
		DeniedArgs:             []string{"token"},
		ArgRules:               map[string]model.MCPArgRule{"limit": {Min: &min}},
		RequireConfirmation:    true,
		RateLimitWindowSeconds: 60,
		RateLimitMaxCalls:      2,
		RedactFields:           []string{"query"},
	}

	denied, err := EvaluateMCPToolPolicy(policyModel, model.MCPPolicyContext{CallerRole: "candidate", CallerScope: "tenant:1", Args: map[string]any{"query": "hi"}})
	if err != nil {
		t.Fatalf("EvaluateMCPToolPolicy returned %v", err)
	}
	if denied.Decision != model.MCPPolicyDecisionDeny || denied.Reason != "caller_role_not_allowed" {
		t.Fatalf("denied = %+v", denied)
	}

	rateLimited, err := EvaluateMCPToolPolicy(policyModel, model.MCPPolicyContext{CallerRole: "hr_admin", CallerScope: "tenant:1", RecentCalls: 2, Args: map[string]any{"query": "hi", "limit": 2}})
	if err != nil {
		t.Fatalf("EvaluateMCPToolPolicy rate limit returned %v", err)
	}
	if rateLimited.Decision != model.MCPPolicyDecisionRateLimited {
		t.Fatalf("rateLimited = %+v", rateLimited)
	}

	confirmation, err := EvaluateMCPToolPolicy(policyModel, model.MCPPolicyContext{CallerRole: "hr_admin", CallerScope: "tenant:1", Args: map[string]any{"query": "hi", "limit": 2}})
	if err != nil {
		t.Fatalf("EvaluateMCPToolPolicy confirmation returned %v", err)
	}
	if confirmation.Decision != model.MCPPolicyDecisionConfirmationRequired || confirmation.Reason != "confirmation_required" {
		t.Fatalf("confirmation = %+v", confirmation)
	}
}

func TestValidateMCPServerConfigRejectsPrivateNetworkAndDisallowedCommand(t *testing.T) {
	if err := ValidateMCPServerConfig(model.MCPServerConfig{Transport: model.MCPTransportHTTP, URL: "http://127.0.0.1:3000"}); !errors.Is(err, ErrMCPServerInvalid) {
		t.Fatalf("private URL err = %v, want ErrMCPServerInvalid", err)
	}
	if err := ValidateMCPServerConfig(model.MCPServerConfig{Transport: model.MCPTransportStdio, Command: "bash -lc env", AllowedCommands: []string{"node"}}); !errors.Is(err, ErrMCPServerInvalid) {
		t.Fatalf("command err = %v, want ErrMCPServerInvalid", err)
	}
	if err := ValidateMCPServerConfig(model.MCPServerConfig{Transport: model.MCPTransportStdio, Command: "node server.js", AllowedCommands: []string{"node"}}); err != nil {
		t.Fatalf("allowed command returned %v", err)
	}
}

func TestLegacyExecutableSkillVersionPolicy(t *testing.T) {
	manifest := model.SkillManifest{
		Name:        "candidate_match",
		Version:     "1.0.0",
		RuntimeType: model.SkillRuntimeTool,
		Tools: []model.SkillToolManifest{{
			Name:        "match",
			RuntimeType: model.SkillRuntimeTool,
			InputSchema: `{"type":"object","properties":{"job_id":{"type":"number"}}}`,
		}},
	}
	if err := ValidateSkillManifest(manifest); err != nil {
		t.Fatalf("ValidateSkillManifest returned %v", err)
	}
	if next, changed := NextSkillVersion(2, true); next != 3 || !changed {
		t.Fatalf("NextSkillVersion changed = %d %v", next, changed)
	}
	if next, changed := NextSkillVersion(2, false); next != 2 || changed {
		t.Fatalf("NextSkillVersion unchanged = %d %v", next, changed)
	}
}

func TestEmbeddingRuntimeAndCandidateMatchPolicy(t *testing.T) {
	state := ResolveEmbeddingRuntime(nil, nil, "")
	if state.Status != model.EmbeddingStatusUnavailable || !state.FallbackUsed {
		t.Fatalf("state = %+v", state)
	}
	provider := model.EmbeddingProviderConfig{ID: 1, Name: "bailian", ProviderType: "bailian", Endpoint: "https://example.com", Enabled: true, HasEncryptedCredential: true}
	embeddingModel := model.EmbeddingModelConfig{ID: 2, ProviderID: 1, ModelName: "text-embedding", Dimension: 1536, Enabled: true, Default: true}
	state = ResolveEmbeddingRuntime(&provider, &embeddingModel, "")
	if state.Status != model.EmbeddingStatusAvailable || state.Dimension != 1536 {
		t.Fatalf("state = %+v", state)
	}

	aggregation, err := AggregateCandidateMatch(model.CandidateMatchProfile{Requirements: []model.CandidateRequirement{
		{ID: "r1", Label: "Go", Priority: model.RequirementMustHave, Category: model.RequirementCoreSkill, Weight: 1, Knockout: true},
		{ID: "r2", Label: "Project", Priority: model.RequirementNiceToHave, Category: model.RequirementExperience, Weight: 1},
	}}, []model.CandidateRequirementResult{
		{RequirementID: "r1", Status: model.MatchStatusMissing, Score: 0},
		{RequirementID: "r2", Status: model.MatchStatusMatch, Score: 80},
	}, false)
	if err != nil {
		t.Fatalf("AggregateCandidateMatch returned %v", err)
	}
	if aggregation.OverallScore > 40 || aggregation.Recommendation != model.RecommendationStrongNot {
		t.Fatalf("aggregation = %+v", aggregation)
	}
}
