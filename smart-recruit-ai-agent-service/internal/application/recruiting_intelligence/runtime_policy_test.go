package recruiting_intelligence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRuntimePolicyDefaultsAreDeterministic(t *testing.T) {
	for name, policy := range map[string]RuntimePolicy{
		"zero":    {},
		"default": DefaultRuntimePolicy(),
		"invalid durations": NewRuntimePolicy(RuntimePolicyConfig{
			StructuredResumeParse:  true,
			CandidateMatch:         true,
			CandidateMatchSemantic: true,
			Fallbacks:              true,
		}),
	} {
		t.Run(name, func(t *testing.T) {
			if !policy.StructuredResumeParseEnabled() || !policy.CandidateMatchEnabled() || !policy.CandidateMatchSemanticEnabled() {
				t.Fatalf("default capabilities disabled: %#v", policy)
			}
			if policy.CandidateMatchShadowEnabled() || !policy.FallbacksEnabled() {
				t.Fatalf("default shadow/fallback policy unexpected: %#v", policy)
			}
			if policy.ResumeParseTimeout() != 30*time.Second || policy.CandidateMatchTimeout() != 15*time.Second {
				t.Fatalf("default timeouts = %v/%v", policy.ResumeParseTimeout(), policy.CandidateMatchTimeout())
			}
		})
	}
}

func TestRuntimePolicyPreservesDisabledAndFallbackCombinations(t *testing.T) {
	policy := NewRuntimePolicy(RuntimePolicyConfig{
		StructuredResumeParse:  false,
		CandidateMatch:         true,
		CandidateMatchSemantic: false,
		CandidateMatchShadow:   true,
		Fallbacks:              false,
		ResumeParseTimeout:     7 * time.Second,
		CandidateMatchTimeout:  11 * time.Second,
	})
	if policy.StructuredResumeParseEnabled() || !policy.CandidateMatchEnabled() || policy.CandidateMatchSemanticEnabled() || !policy.CandidateMatchShadowEnabled() || policy.FallbacksEnabled() {
		t.Fatalf("policy flags were not preserved: %#v", policy)
	}
	if policy.ResumeParseTimeout() != 7*time.Second || policy.CandidateMatchTimeout() != 11*time.Second {
		t.Fatalf("policy timeouts = %v/%v", policy.ResumeParseTimeout(), policy.CandidateMatchTimeout())
	}
}

func TestCandidateMatchExecutionPlan(t *testing.T) {
	tests := []struct {
		name                      string
		candidateMatch            bool
		semantic                  bool
		shadow                    bool
		wantEnabled               bool
		wantPrimary, wantShadow   CandidateMatchExecutionMode
		wantStructuredCompletions bool
	}{
		{name: "overall disabled overrides semantic and shadow", candidateMatch: false, semantic: true, shadow: true, wantPrimary: CandidateMatchExecutionDisabled, wantShadow: CandidateMatchExecutionDisabled},
		{name: "deterministic primary only", candidateMatch: true, semantic: false, shadow: false, wantEnabled: true, wantPrimary: CandidateMatchExecutionDeterministic, wantShadow: CandidateMatchExecutionDisabled},
		{name: "deterministic primary enhanced shadow", candidateMatch: true, semantic: false, shadow: true, wantEnabled: true, wantPrimary: CandidateMatchExecutionDeterministic, wantShadow: CandidateMatchExecutionEnhanced, wantStructuredCompletions: true},
		{name: "enhanced primary only", candidateMatch: true, semantic: true, shadow: false, wantEnabled: true, wantPrimary: CandidateMatchExecutionEnhanced, wantShadow: CandidateMatchExecutionDisabled, wantStructuredCompletions: true},
		{name: "enhanced primary deterministic shadow", candidateMatch: true, semantic: true, shadow: true, wantEnabled: true, wantPrimary: CandidateMatchExecutionEnhanced, wantShadow: CandidateMatchExecutionDeterministic, wantStructuredCompletions: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := NewRuntimePolicy(RuntimePolicyConfig{
				CandidateMatch:         tt.candidateMatch,
				CandidateMatchSemantic: tt.semantic,
				CandidateMatchShadow:   tt.shadow,
			})
			plan := policy.CandidateMatchExecutionPlan()
			if plan.Enabled() != tt.wantEnabled || plan.Primary() != tt.wantPrimary || plan.Shadow() != tt.wantShadow || plan.StructuredCompletionEnabled() != tt.wantStructuredCompletions {
				t.Fatalf("plan = enabled:%v primary:%s shadow:%s structured:%v", plan.Enabled(), plan.Primary(), plan.Shadow(), plan.StructuredCompletionEnabled())
			}
			if plan.EnhancedPrimaryEnabled() != (tt.wantPrimary == CandidateMatchExecutionEnhanced) || plan.EnhancedShadowEnabled() != (tt.wantShadow == CandidateMatchExecutionEnhanced) {
				t.Fatal("enhanced execution intent does not match primary/shadow modes")
			}
		})
	}
}

func TestRuntimePolicyContextsUseDistinctTimeoutsAndPropagateCancellation(t *testing.T) {
	policy := NewRuntimePolicy(RuntimePolicyConfig{
		ResumeParseTimeout:    time.Hour,
		CandidateMatchTimeout: 2 * time.Hour,
	})
	started := time.Now()
	resumeCtx, cancelResume := policy.ResumeContext(context.Background())
	defer cancelResume()
	matchCtx, cancelMatch := policy.CandidateMatchContext(context.Background())
	defer cancelMatch()
	resumeDeadline, resumeOK := resumeCtx.Deadline()
	matchDeadline, matchOK := matchCtx.Deadline()
	if !resumeOK || !matchOK || resumeDeadline.Before(started.Add(59*time.Minute)) || matchDeadline.Before(started.Add(119*time.Minute)) {
		t.Fatalf("unexpected deadlines resume=%v match=%v", resumeDeadline, matchDeadline)
	}

	parent, cancelParent := context.WithCancel(context.Background())
	child, cancelChild := policy.ResumeContext(parent)
	defer cancelChild()
	cancelParent()
	select {
	case <-child.Done():
		if !errors.Is(child.Err(), context.Canceled) {
			t.Fatalf("child error = %v", child.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("parent cancellation did not propagate")
	}
}

func TestRuntimePolicyResumeExecutionContextsReservePersistenceBudget(t *testing.T) {
	policy := NewRuntimePolicy(RuntimePolicyConfig{StructuredResumeParse: true, ResumeParseTimeout: 100 * time.Millisecond})
	totalCtx, parseCtx, cancel := policy.ResumeExecutionContexts(context.Background())
	defer cancel()
	totalDeadline, totalOK := totalCtx.Deadline()
	parseDeadline, parseOK := parseCtx.Deadline()
	if !totalOK || !parseOK {
		t.Fatal("resume execution contexts must both have deadlines")
	}
	reserve := totalDeadline.Sub(parseDeadline)
	if reserve < 15*time.Millisecond || reserve > 25*time.Millisecond {
		t.Fatalf("persistence reserve = %v, want approximately 20ms", reserve)
	}
}

func TestRuntimeRejectsOverallDisabledCapabilitiesBeforePromptLookup(t *testing.T) {
	tests := []struct {
		name      string
		policy    RuntimePolicy
		agentType string
	}{
		{name: "resume overall gate", policy: NewRuntimePolicy(RuntimePolicyConfig{StructuredResumeParse: false, CandidateMatch: true, CandidateMatchSemantic: true}), agentType: AgentTypeResumeProfileExtractor},
		{name: "candidate match overall gate blocks requirement extractor", policy: NewRuntimePolicy(RuntimePolicyConfig{CandidateMatch: false, CandidateMatchSemantic: true, CandidateMatchShadow: true}), agentType: AgentTypeJobRequirementExtractor},
		{name: "candidate match overall gate blocks evaluator", policy: NewRuntimePolicy(RuntimePolicyConfig{CandidateMatch: false, CandidateMatchSemantic: true, CandidateMatchShadow: true}), agentType: AgentTypeCandidateMatchEvaluator},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &runtimePromptStore{}
			runtime := NewRuntime(NewPromptLoader(store), &runtimeProvider{}, tt.policy)
			_, err := runtime.Complete(context.Background(), tt.agentType, "private input")
			if !errors.Is(err, ErrCapabilityDisabled) {
				t.Fatalf("Complete(%s) error = %v", tt.agentType, err)
			}
			if len(store.calls) != 0 {
				t.Fatalf("disabled capability performed prompt lookups: %#v", store.calls)
			}
		})
	}
}

func TestRuntimeCandidateMatchStructuredCompletionGateUsesEnhancedIntent(t *testing.T) {
	for _, tt := range []struct {
		name     string
		semantic bool
		shadow   bool
		wantCall bool
	}{
		{name: "deterministic only", semantic: false, shadow: false, wantCall: false},
		{name: "enhanced shadow", semantic: false, shadow: true, wantCall: true},
		{name: "enhanced primary", semantic: true, shadow: false, wantCall: true},
	} {
		for _, agentType := range []string{AgentTypeJobRequirementExtractor, AgentTypeCandidateMatchEvaluator} {
			t.Run(tt.name+"/"+agentType, func(t *testing.T) {
				store := &runtimePromptStore{prompt: PromptDescriptor{ID: 10, AgentType: agentType, Role: PromptRoleSystem, Content: "system"}}
				provider := &policyRuntimeProvider{}
				runtime := NewRuntime(NewPromptLoader(store), provider, NewRuntimePolicy(RuntimePolicyConfig{
					CandidateMatch: true, CandidateMatchSemantic: tt.semantic, CandidateMatchShadow: tt.shadow,
				}))
				_, err := runtime.Complete(context.Background(), agentType, "private input")
				if tt.wantCall {
					if err != nil || len(store.calls) != 1 || provider.calls != 1 {
						t.Fatalf("enhanced execution did not reach structured runtime: err=%v store=%d provider=%d", err, len(store.calls), provider.calls)
					}
					return
				}
				if !errors.Is(err, ErrCapabilityDisabled) || len(store.calls) != 0 || provider.calls != 0 {
					t.Fatalf("deterministic-only execution reached structured runtime: err=%v store=%d provider=%d", err, len(store.calls), provider.calls)
				}
			})
		}
	}
}

type policyRuntimeProvider struct {
	calls int
}

func (p *policyRuntimeProvider) CompleteStructured(_ context.Context, _, _ string) (StructuredCompletionResult, error) {
	p.calls++
	return StructuredCompletionResult{Content: `{}`, ModelName: "policy-test"}, nil
}

func TestRuntimeOperationCancellationReachesProvider(t *testing.T) {
	store := &runtimePromptStore{prompt: PromptDescriptor{ID: 10, AgentType: AgentTypeResumeProfileExtractor, Role: PromptRoleSystem, Content: "system"}}
	provider := cancelAwareRuntimeProvider{}
	runtime := NewRuntime(NewPromptLoader(store), provider)
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := runtime.Complete(parent, AgentTypeResumeProfileExtractor, "private input")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Complete error = %v, want context cancellation", err)
	}
}

type cancelAwareRuntimeProvider struct{}

func (cancelAwareRuntimeProvider) CompleteStructured(ctx context.Context, _, _ string) (StructuredCompletionResult, error) {
	<-ctx.Done()
	return StructuredCompletionResult{}, ctx.Err()
}
