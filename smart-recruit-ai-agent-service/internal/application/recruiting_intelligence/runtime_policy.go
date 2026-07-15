package recruiting_intelligence

import (
	"context"
	"time"
)

const (
	defaultResumeParseTimeout    = 30 * time.Second
	defaultCandidateMatchTimeout = 15 * time.Second
	maxResumePersistenceReserve  = 2 * time.Second
	maxMatchPersistenceReserve   = 2 * time.Second
)

// RuntimePolicyConfig is copied into RuntimePolicy at service startup. The
// resulting policy exposes no mutation surface and is safe to share between
// concurrent requests.
type RuntimePolicyConfig struct {
	StructuredResumeParse  bool
	CandidateMatch         bool
	CandidateMatchSemantic bool
	CandidateMatchShadow   bool
	Fallbacks              bool
	ResumeParseTimeout     time.Duration
	CandidateMatchTimeout  time.Duration
}

// RuntimePolicy is an immutable, request-safe snapshot of recruiting feature
// policy. Its zero value intentionally resolves to the service defaults so
// internal tests and callers that omit policy retain deterministic behavior.
type RuntimePolicy struct {
	configured             bool
	structuredResumeParse  bool
	candidateMatch         bool
	candidateMatchSemantic bool
	candidateMatchShadow   bool
	fallbacks              bool
	resumeParseTimeout     time.Duration
	candidateMatchTimeout  time.Duration
}

// CandidateMatchExecutionMode identifies which scorer owns a primary or
// shadow execution. It is deliberately separate from the overall candidate
// match switch: disabling the enhanced/semantic primary must not disable the
// deterministic primary path.
type CandidateMatchExecutionMode string

const (
	CandidateMatchExecutionDisabled      CandidateMatchExecutionMode = "disabled"
	CandidateMatchExecutionDeterministic CandidateMatchExecutionMode = "deterministic"
	CandidateMatchExecutionEnhanced      CandidateMatchExecutionMode = "enhanced"
)

// CandidateMatchExecutionPlan is an immutable, request-safe description of
// candidate-match execution intent for the extractor, evaluator, and scorer
// tasks. Private fields prevent downstream code from mutating the policy
// snapshot for a request or a shadow run.
type CandidateMatchExecutionPlan struct {
	enabled bool
	primary CandidateMatchExecutionMode
	shadow  CandidateMatchExecutionMode
}

func (p CandidateMatchExecutionPlan) Enabled() bool {
	return p.enabled
}

func (p CandidateMatchExecutionPlan) Primary() CandidateMatchExecutionMode {
	return p.primary
}

func (p CandidateMatchExecutionPlan) Shadow() CandidateMatchExecutionMode {
	return p.shadow
}

func (p CandidateMatchExecutionPlan) EnhancedPrimaryEnabled() bool {
	return p.enabled && p.primary == CandidateMatchExecutionEnhanced
}

func (p CandidateMatchExecutionPlan) EnhancedShadowEnabled() bool {
	return p.enabled && p.shadow == CandidateMatchExecutionEnhanced
}

// StructuredCompletionEnabled reports whether either the primary or shadow
// execution may enter the enhanced pipeline. A deterministic-only primary
// does not need recruiting LLM calls.
func (p CandidateMatchExecutionPlan) StructuredCompletionEnabled() bool {
	return p.EnhancedPrimaryEnabled() || p.EnhancedShadowEnabled()
}

func DefaultRuntimePolicy() RuntimePolicy {
	return NewRuntimePolicy(RuntimePolicyConfig{
		StructuredResumeParse:  true,
		CandidateMatch:         true,
		CandidateMatchSemantic: true,
		CandidateMatchShadow:   false,
		Fallbacks:              true,
		ResumeParseTimeout:     defaultResumeParseTimeout,
		CandidateMatchTimeout:  defaultCandidateMatchTimeout,
	})
}

func NewRuntimePolicy(config RuntimePolicyConfig) RuntimePolicy {
	resumeTimeout := config.ResumeParseTimeout
	if resumeTimeout <= 0 {
		resumeTimeout = defaultResumeParseTimeout
	}
	matchTimeout := config.CandidateMatchTimeout
	if matchTimeout <= 0 {
		matchTimeout = defaultCandidateMatchTimeout
	}
	return RuntimePolicy{
		configured:             true,
		structuredResumeParse:  config.StructuredResumeParse,
		candidateMatch:         config.CandidateMatch,
		candidateMatchSemantic: config.CandidateMatchSemantic,
		candidateMatchShadow:   config.CandidateMatchShadow,
		fallbacks:              config.Fallbacks,
		resumeParseTimeout:     resumeTimeout,
		candidateMatchTimeout:  matchTimeout,
	}
}

func (p RuntimePolicy) effective() RuntimePolicy {
	if !p.configured {
		return DefaultRuntimePolicy()
	}
	return p
}

func (p RuntimePolicy) StructuredResumeParseEnabled() bool {
	return p.effective().structuredResumeParse
}

func (p RuntimePolicy) CandidateMatchEnabled() bool {
	return p.effective().candidateMatch
}

func (p RuntimePolicy) CandidateMatchSemanticEnabled() bool {
	return p.effective().candidateMatchSemantic
}

func (p RuntimePolicy) CandidateMatchShadowEnabled() bool {
	return p.effective().candidateMatchShadow
}

// CandidateMatchExecutionPlan preserves the dev runtime semantics:
// candidate_match is the overall gate, semantic selects the enhanced primary
// instead of the deterministic primary, and shadow runs the opposite scorer.
func (p RuntimePolicy) CandidateMatchExecutionPlan() CandidateMatchExecutionPlan {
	effective := p.effective()
	if !effective.candidateMatch {
		return CandidateMatchExecutionPlan{primary: CandidateMatchExecutionDisabled, shadow: CandidateMatchExecutionDisabled}
	}
	plan := CandidateMatchExecutionPlan{enabled: true, primary: CandidateMatchExecutionDeterministic, shadow: CandidateMatchExecutionDisabled}
	if effective.candidateMatchSemantic {
		plan.primary = CandidateMatchExecutionEnhanced
	}
	if effective.candidateMatchShadow {
		if plan.primary == CandidateMatchExecutionEnhanced {
			plan.shadow = CandidateMatchExecutionDeterministic
		} else {
			plan.shadow = CandidateMatchExecutionEnhanced
		}
	}
	return plan
}

func (p RuntimePolicy) FallbacksEnabled() bool {
	return p.effective().fallbacks
}

func (p RuntimePolicy) ResumeParseTimeout() time.Duration {
	return p.effective().resumeParseTimeout
}

func (p RuntimePolicy) CandidateMatchTimeout() time.Duration {
	return p.effective().candidateMatchTimeout
}

func (p RuntimePolicy) ResumeContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, p.ResumeParseTimeout())
}

// ResumeExecutionContexts places parsing and persistence under one feature
// deadline while ending the parsing window early enough to leave a
// deterministic persistence reserve. A shorter caller deadline reduces both
// windows proportionally instead of eliminating the save budget.
func (p RuntimePolicy) ResumeExecutionContexts(parent context.Context) (context.Context, context.Context, context.CancelFunc) {
	available := p.ResumeParseTimeout()
	if parentDeadline, ok := parent.Deadline(); ok {
		if remaining := time.Until(parentDeadline); remaining < available {
			available = remaining
		}
	}
	if available <= 0 {
		available = time.Nanosecond
	}
	totalCtx, cancelTotal := context.WithTimeout(parent, available)
	reserve := available / 5
	if reserve > maxResumePersistenceReserve {
		reserve = maxResumePersistenceReserve
	}
	if reserve <= 0 {
		reserve = time.Nanosecond
	}
	parseBudget := available - reserve
	if parseBudget <= 0 {
		parseBudget = time.Nanosecond
	}
	parseCtx, cancelParse := context.WithTimeout(totalCtx, parseBudget)
	return totalCtx, parseCtx, func() {
		cancelParse()
		cancelTotal()
	}
}

func (p RuntimePolicy) CandidateMatchContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, p.CandidateMatchTimeout())
}

// CandidateMatchExecutionContexts places source loading, scoring, and
// persistence under one feature deadline. Scoring ends early enough to leave
// a bounded transaction budget, matching the resume-profile execution model.
func (p RuntimePolicy) CandidateMatchExecutionContexts(parent context.Context) (context.Context, context.Context, context.CancelFunc) {
	available := p.CandidateMatchTimeout()
	if parentDeadline, ok := parent.Deadline(); ok {
		if remaining := time.Until(parentDeadline); remaining < available {
			available = remaining
		}
	}
	if available <= 0 {
		available = time.Nanosecond
	}
	totalCtx, cancelTotal := context.WithTimeout(parent, available)
	reserve := available / 5
	if reserve > maxMatchPersistenceReserve {
		reserve = maxMatchPersistenceReserve
	}
	if reserve <= 0 {
		reserve = time.Nanosecond
	}
	generationBudget := available - reserve
	if generationBudget <= 0 {
		generationBudget = time.Nanosecond
	}
	generationCtx, cancelGeneration := context.WithTimeout(totalCtx, generationBudget)
	return totalCtx, generationCtx, func() {
		cancelGeneration()
		cancelTotal()
	}
}
