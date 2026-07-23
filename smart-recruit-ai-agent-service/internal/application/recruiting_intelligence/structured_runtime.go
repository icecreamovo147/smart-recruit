package recruiting_intelligence

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

const (
	PromptRoleSystem                 = "system"
	AgentTypeResumeProfileExtractor  = "resume_profile_extractor"
	AgentTypeJobRequirementExtractor = "job_requirement_extractor"
	AgentTypeCandidateMatchEvaluator = "candidate_match_evaluator"
)

var (
	ErrPromptNotFound     = errors.New("active recruiting prompt not found")
	ErrPromptInvalid      = errors.New("active recruiting prompt is invalid")
	ErrPromptStore        = errors.New("recruiting prompt store failed")
	ErrProvider           = errors.New("structured completion provider failed")
	ErrCapabilityDisabled = errors.New("recruiting capability is disabled")
)

type ErrorKind string

const (
	ErrorKindPrompt   ErrorKind = "prompt"
	ErrorKindProvider ErrorKind = "provider"
	ErrorKindPolicy   ErrorKind = "policy"
)

// RuntimeError deliberately omits prompt/message/model-output bodies from its
// string form while preserving the underlying error for classification.
type RuntimeError struct {
	Kind      ErrorKind
	Operation string
	AgentType string
	Cause     error
}

func (e *RuntimeError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("recruiting structured runtime %s failed for agent_type=%s (%s)", e.Operation, e.AgentType, e.Kind)
}

func (e *RuntimeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type PromptDescriptor struct {
	ID        int64
	Name      string
	Version   int32
	AgentType string
	Role      string
	Content   string
}

type PromptStore interface {
	LoadActiveRecruitingPrompt(ctx context.Context, agentType, role string) (PromptDescriptor, error)
}

type PromptLoader struct {
	store PromptStore
}

func NewPromptLoader(store PromptStore) *PromptLoader {
	return &PromptLoader{store: store}
}

func (l *PromptLoader) LoadSystemPrompt(ctx context.Context, agentType string) (PromptDescriptor, error) {
	if !supportedAgentType(agentType) {
		return PromptDescriptor{}, &RuntimeError{Kind: ErrorKindPrompt, Operation: "load_prompt", AgentType: agentType, Cause: ErrPromptInvalid}
	}
	if l == nil || l.store == nil {
		return PromptDescriptor{}, &RuntimeError{Kind: ErrorKindPrompt, Operation: "load_prompt", AgentType: agentType, Cause: ErrPromptStore}
	}
	prompt, err := l.store.LoadActiveRecruitingPrompt(ctx, agentType, PromptRoleSystem)
	if err != nil {
		return PromptDescriptor{}, &RuntimeError{Kind: ErrorKindPrompt, Operation: "load_prompt", AgentType: agentType, Cause: err}
	}
	if prompt.AgentType != agentType || prompt.Role != PromptRoleSystem || strings.TrimSpace(prompt.Content) == "" || prompt.ID <= 0 {
		return PromptDescriptor{}, &RuntimeError{Kind: ErrorKindPrompt, Operation: "validate_prompt", AgentType: agentType, Cause: ErrPromptInvalid}
	}
	return prompt, nil
}

type StructuredCompletionResult struct {
	Content     string
	ProviderKey string
	ModelName   string
	TokenUsage  *schema.TokenUsage
}

type StructuredCompletionProvider interface {
	CompleteStructured(ctx context.Context, systemPrompt, userPrompt string) (StructuredCompletionResult, error)
}

type CompletionResult struct {
	Content     string
	ProviderKey string
	ModelName   string
	TokenUsage  *schema.TokenUsage
	Prompt      PromptDescriptor
}

type BillingProviderUsage struct {
	ProviderKey           string
	ModelName             string
	TokenUsage            *schema.TokenUsage
	EstimatedInputTokens  int
	EstimatedOutputTokens int
}

type BillingUsageCollector struct {
	mu     sync.Mutex
	usages []BillingProviderUsage
}

type billingUsageCollectorContextKey struct{}

func WithBillingUsageCollector(ctx context.Context) (context.Context, *BillingUsageCollector) {
	collector := &BillingUsageCollector{}
	return context.WithValue(ctx, billingUsageCollectorContextKey{}, collector), collector
}

func (c *BillingUsageCollector) add(usage BillingProviderUsage) {
	if c == nil {
		return
	}
	if usage.TokenUsage != nil {
		copyUsage := *usage.TokenUsage
		usage.TokenUsage = &copyUsage
	}
	c.mu.Lock()
	c.usages = append(c.usages, usage)
	c.mu.Unlock()
}

func (c *BillingUsageCollector) Usages() []BillingProviderUsage {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]BillingProviderUsage, len(c.usages))
	copy(result, c.usages)
	return result
}

// Observation is the complete allow-list for recruiting runtime diagnostics.
// It deliberately has no request body, prompt content, model response, evidence,
// resource text, error string, or candidate identifier field.
type Observation struct {
	Operation        string
	RequestID        string
	ResourceType     string
	ResourceID       int64
	Stage            string
	Terminal         bool
	Category         string
	AgentType        string
	PromptID         int64
	PromptName       string
	PromptVersion    int32
	ModelName        string
	Fallback         string
	Outcome          string
	ParserVersion    string
	ScorerVersion    string
	InputCount       int32
	OutputCount      int32
	EvidenceCount    int32
	RequirementCount int32
	Duration         time.Duration
	normalized       bool
}

const (
	maxObservationPromptNameBytes = 256
	maxObservationModelNameBytes  = 128
	maxObservationVersionBytes    = 64
	maxObservationDuration        = 24 * time.Hour
	observationDigestHexLength    = 24
)

var observationCorrelationKey = newObservationCorrelationKey()

// NormalizeObservation is the fail-closed boundary for recruiting diagnostics.
// It accepts only fixed classifications and bounded identity values so an
// externally supplied request ID or a future caller cannot turn logs into a
// body/PII/error sink.
func NormalizeObservation(observation Observation) Observation {
	if observation.normalized {
		return observation
	}
	observation.Operation = allowObservationValue(observation.Operation,
		AgentTypeResumeProfileExtractor, AgentTypeJobRequirementExtractor, AgentTypeCandidateMatchEvaluator,
		"resume_profile", "candidate_match")
	observation.RequestID = observationCorrelation("request-id", "rid", observation.RequestID, 0)
	observation.ResourceType = allowObservationValue(observation.ResourceType, "resume", "application")
	if observation.ResourceID < 0 {
		observation.ResourceID = 0
	}
	observation.Stage = allowObservationValue(observation.Stage,
		"runtime_policy", "prompt_load", "structured_completion", "resume_profile_extraction",
		"job_requirement_extraction", "candidate_requirement_evaluation", "source", "generation",
		"aggregation", "persistence", "operation")
	observation.Category = allowObservationValue(observation.Category,
		"success", "failure", "fallback_success", "prompt_failure", "provider_failure",
		"empty_response_failure", "json_failure", "schema_failure", "policy_failure", "timeout",
		"fallback_failure", "domain_validation_failure", "aggregation_failure", "source_failure",
		"persistence_failure", "configuration_failure", "authorization_failure", "not_found")
	observation.AgentType = allowObservationValue(observation.AgentType,
		AgentTypeResumeProfileExtractor, AgentTypeJobRequirementExtractor, AgentTypeCandidateMatchEvaluator)
	if observation.PromptID < 0 {
		observation.PromptID = 0
	}
	observation.PromptName = observationCorrelation("prompt-name", "prompt", observation.PromptName, maxObservationPromptNameBytes)
	if observation.PromptVersion < 0 {
		observation.PromptVersion = 0
	}
	observation.ModelName = observationCorrelation("model-name", "model", observation.ModelName, maxObservationModelNameBytes)
	observation.Fallback = allowObservationValueWithEmpty(observation.Fallback, "none",
		"none", "disabled", "deterministic", "deterministic_missing", "heuristic", "legacy_deterministic",
		"prompt", "provider", "timeout", "empty_response", "json", "schema", "policy", "fallback")
	observation.Outcome = allowObservationValue(observation.Outcome, "success", "error")
	observation.ParserVersion = trustedObservationVersion(observation.ParserVersion, "parser",
		ResumeLLMParserVersion, ResumeHeuristicParserVersion,
		JobRequirementLLMVersion, JobRequirementHeuristicVersion)
	observation.ScorerVersion = trustedObservationVersion(observation.ScorerVersion, "scorer",
		CandidateMatchScorerVersion, LegacyCandidateMatchScorerVersion)
	observation.InputCount = boundedObservationCount(observation.InputCount)
	observation.OutputCount = boundedObservationCount(observation.OutputCount)
	observation.EvidenceCount = boundedObservationCount(observation.EvidenceCount)
	observation.RequirementCount = boundedObservationCount(observation.RequirementCount)
	if observation.Duration < 0 {
		observation.Duration = 0
	} else if observation.Duration > maxObservationDuration {
		observation.Duration = maxObservationDuration
	}
	observation.normalized = true
	return observation
}

func allowObservationValue(value string, allowed ...string) string {
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return "unknown"
}

func allowObservationValueWithEmpty(value, emptyValue string, allowed ...string) string {
	if value == "" {
		return emptyValue
	}
	return allowObservationValue(value, allowed...)
}

func newObservationCorrelationKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil
	}
	return key
}

// observationCorrelation retains only a process-local, domain-separated HMAC
// token. A random key prevents dictionary reversal of low-entropy values such
// as phone numbers. The unexported normalized marker makes repeated boundaries
// idempotent without trusting an externally supplied token-shaped string.
func observationCorrelation(domain, label, value string, schemaMaxBytes int) string {
	if value == "" || len(observationCorrelationKey) == 0 {
		return ""
	}
	if schemaMaxBytes > 0 && !safeObservationIdentity(value) {
		return ""
	}
	if schemaMaxBytes > 0 && len(value) > schemaMaxBytes {
		domain += ":outside-schema"
	}
	mac := hmac.New(sha256.New, observationCorrelationKey)
	_, _ = mac.Write([]byte("smart-recruit:recruiting-observation:v1:" + domain + "\x00" + value))
	return label + ":v1:" + hex.EncodeToString(mac.Sum(nil))[:observationDigestHexLength]
}

func safeObservationIdentity(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.Is(unicode.C, r) {
			return false
		}
	}
	return true
}

func trustedObservationVersion(value, label string, allowed ...string) string {
	if value == "" {
		return ""
	}
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return observationCorrelation(label+"-version", label, value, maxObservationVersionBytes)
}

type Observer interface {
	ObserveRecruitingRuntime(context.Context, Observation)
}

type observationMetadata struct {
	RequestID    string
	ResourceType string
	ResourceID   int64
}

type observationMetadataKey struct{}

// WithObservationMetadata attaches opaque/internal identifiers without
// coupling the application package to transport metadata implementations.
func WithObservationMetadata(ctx context.Context, requestID, resourceType string, resourceID int64) context.Context {
	return context.WithValue(ctx, observationMetadataKey{}, observationMetadata{RequestID: requestID, ResourceType: resourceType, ResourceID: resourceID})
}

type Runtime struct {
	prompts  *PromptLoader
	provider StructuredCompletionProvider
	policy   RuntimePolicy
	observer Observer
}

func NewRuntime(prompts *PromptLoader, provider StructuredCompletionProvider, policies ...RuntimePolicy) *Runtime {
	return NewRuntimeWithObserver(prompts, provider, nil, policies...)
}

func NewRuntimeWithObserver(prompts *PromptLoader, provider StructuredCompletionProvider, observer Observer, policies ...RuntimePolicy) *Runtime {
	policy := DefaultRuntimePolicy()
	if len(policies) > 0 {
		policy = policies[0].effective()
	}
	return &Runtime{prompts: prompts, provider: provider, policy: policy, observer: observer}
}

func (r *Runtime) Policy() RuntimePolicy {
	if r == nil {
		return DefaultRuntimePolicy()
	}
	return r.policy.effective()
}

// Complete reloads the active system prompt on every request so prompt version
// activation is observed without a process restart or cache invalidation.
func (r *Runtime) Complete(ctx context.Context, agentType, userPrompt string) (CompletionResult, error) {
	promptStarted := time.Now()
	if r == nil || r.prompts == nil {
		if r != nil {
			r.observe(ctx, Observation{Operation: agentType, Stage: "prompt_load", Category: "prompt_failure", AgentType: agentType, Outcome: "error", Duration: time.Since(promptStarted)})
		}
		return CompletionResult{}, &RuntimeError{Kind: ErrorKindPrompt, Operation: "load_prompt", AgentType: agentType, Cause: ErrPromptStore}
	}
	operationCtx, cancel, err := r.operationContext(ctx, agentType)
	if err != nil {
		r.observe(ctx, Observation{Operation: agentType, Stage: "runtime_policy", Category: "policy_failure", AgentType: agentType, Outcome: "error", Fallback: "disabled", Duration: time.Since(promptStarted)})
		return CompletionResult{}, err
	}
	defer cancel()
	prompt, err := r.prompts.LoadSystemPrompt(operationCtx, agentType)
	if err != nil {
		r.observe(operationCtx, Observation{Operation: agentType, Stage: "prompt_load", Category: "prompt_failure", AgentType: agentType, Outcome: "error", Duration: time.Since(promptStarted)})
		return CompletionResult{}, err
	}
	r.observe(operationCtx, observationForPrompt("prompt_load", "success", prompt, "", "", time.Since(promptStarted)))
	if r.provider == nil {
		event := observationForPrompt("structured_completion", "error", prompt, "", "none", 0)
		event.Category = "provider_failure"
		r.observe(operationCtx, event)
		return CompletionResult{}, &RuntimeError{Kind: ErrorKindProvider, Operation: "complete", AgentType: agentType, Cause: ErrProvider}
	}
	completionStarted := time.Now()
	result, err := r.provider.CompleteStructured(operationCtx, prompt.Content, userPrompt)
	if err != nil {
		event := observationForPrompt("structured_completion", "error", prompt, "", "none", time.Since(completionStarted))
		event.Category = "provider_failure"
		r.observe(operationCtx, event)
		return CompletionResult{}, &RuntimeError{Kind: ErrorKindProvider, Operation: "complete", AgentType: agentType, Cause: err}
	}
	r.observe(operationCtx, observationForPrompt("structured_completion", "success", prompt, result.ModelName, "none", time.Since(completionStarted)))
	if collector, _ := operationCtx.Value(billingUsageCollectorContextKey{}).(*BillingUsageCollector); collector != nil {
		collector.add(BillingProviderUsage{
			ProviderKey: result.ProviderKey, ModelName: result.ModelName, TokenUsage: result.TokenUsage,
			EstimatedInputTokens:  estimateStructuredTokens(prompt.Content + "\n" + userPrompt),
			EstimatedOutputTokens: estimateStructuredTokens(result.Content),
		})
	}
	return CompletionResult{Content: result.Content, ProviderKey: result.ProviderKey, ModelName: result.ModelName, TokenUsage: result.TokenUsage, Prompt: prompt}, nil
}

func estimateStructuredTokens(value string) int {
	runes := utf8.RuneCountInString(value)
	if runes <= 0 {
		return 0
	}
	return (runes + 3) / 4
}

func observationForPrompt(stage, outcome string, prompt PromptDescriptor, modelName, fallback string, duration time.Duration) Observation {
	return Observation{
		Operation: prompt.AgentType, Stage: stage, Terminal: false, Category: observationCategory(outcome),
		AgentType: prompt.AgentType, PromptID: prompt.ID, PromptName: prompt.Name,
		PromptVersion: prompt.Version, ModelName: modelName, Fallback: fallback, Outcome: outcome, Duration: duration,
	}
}

func observationCategory(outcome string) string {
	if outcome == "success" {
		return "success"
	}
	return "failure"
}

// ObservationCategoryForError maps errors to a fixed, privacy-safe category;
// it never returns err.Error() or any provider/body-derived string.
func ObservationCategoryForError(err error) string {
	if err == nil {
		return "success"
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var resumeErr *ResumeExtractionError
	if errors.As(err, &resumeErr) {
		return observationCategoryForKind(string(resumeErr.Kind))
	}
	var jobErr *JobRequirementExtractionError
	if errors.As(err, &jobErr) {
		return observationCategoryForKind(string(jobErr.Kind))
	}
	var runtimeErr *RuntimeError
	if errors.As(err, &runtimeErr) {
		return observationCategoryForKind(string(runtimeErr.Kind))
	}
	switch {
	case errors.Is(err, ErrCandidateMatchSchema), errors.Is(err, ErrCandidateMatchProvenance):
		return "schema_failure"
	case errors.Is(err, ErrCandidateMatchPolicy):
		return "policy_failure"
	default:
		return "aggregation_failure"
	}
}

func observationCategoryForKind(kind string) string {
	switch kind {
	case "prompt":
		return "prompt_failure"
	case "provider":
		return "provider_failure"
	case "empty_response":
		return "empty_response_failure"
	case "json":
		return "json_failure"
	case "schema":
		return "schema_failure"
	case "policy":
		return "policy_failure"
	case "timeout":
		return "timeout"
	case "fallback":
		return "fallback_failure"
	default:
		return "domain_validation_failure"
	}
}

// Observe emits a pre-built, metadata-only event. Callers own operation
// terminality; Runtime component events are always intermediate.
func (r *Runtime) Observe(ctx context.Context, observation Observation) {
	if !observation.Terminal {
		observation.Terminal = false
	}
	r.observe(ctx, observation)
}

func (r *Runtime) observe(ctx context.Context, observation Observation) {
	if r == nil || r.observer == nil {
		return
	}
	if metadata, ok := ctx.Value(observationMetadataKey{}).(observationMetadata); ok {
		if observation.RequestID == "" {
			observation.RequestID = metadata.RequestID
		}
		if observation.ResourceType == "" {
			observation.ResourceType = metadata.ResourceType
		}
		if observation.ResourceID == 0 {
			observation.ResourceID = metadata.ResourceID
		}
	}
	r.observer.ObserveRecruitingRuntime(ctx, NormalizeObservation(observation))
}

func boundedObservationCount(value int32) int32 {
	if value < 0 {
		return 0
	}
	if value > 1000 {
		return 1000
	}
	return value
}

func (r *Runtime) observeOutcome(ctx context.Context, stage, agentType, outcome, fallback string, started time.Time) {
	category := observationCategory(outcome)
	if outcome == "success" && fallback != "none" && fallback != "deterministic" {
		category = "fallback_success"
	}
	r.observe(ctx, Observation{Operation: agentType, Stage: stage, Terminal: false, Category: category, AgentType: agentType, Outcome: outcome, Fallback: fallback, Duration: time.Since(started)})
}

func observeRuntimeOutcome(runtime *Runtime, ctx context.Context, stage, agentType, outcome, fallback string, started time.Time) {
	if runtime == nil {
		return
	}
	runtime.observeOutcome(ctx, stage, agentType, outcome, fallback, started)
}

func (r *Runtime) operationContext(ctx context.Context, agentType string) (context.Context, context.CancelFunc, error) {
	policy := r.Policy()
	switch agentType {
	case AgentTypeResumeProfileExtractor:
		if !policy.StructuredResumeParseEnabled() {
			return nil, nil, &RuntimeError{Kind: ErrorKindPolicy, Operation: "check_capability", AgentType: agentType, Cause: ErrCapabilityDisabled}
		}
		operationCtx, cancel := policy.ResumeContext(ctx)
		return operationCtx, cancel, nil
	case AgentTypeJobRequirementExtractor:
		if !policy.CandidateMatchExecutionPlan().StructuredCompletionEnabled() {
			return nil, nil, &RuntimeError{Kind: ErrorKindPolicy, Operation: "check_capability", AgentType: agentType, Cause: ErrCapabilityDisabled}
		}
		operationCtx, cancel := policy.CandidateMatchContext(ctx)
		return operationCtx, cancel, nil
	case AgentTypeCandidateMatchEvaluator:
		if !policy.CandidateMatchExecutionPlan().StructuredCompletionEnabled() {
			return nil, nil, &RuntimeError{Kind: ErrorKindPolicy, Operation: "check_capability", AgentType: agentType, Cause: ErrCapabilityDisabled}
		}
		operationCtx, cancel := policy.CandidateMatchContext(ctx)
		return operationCtx, cancel, nil
	default:
		return nil, nil, &RuntimeError{Kind: ErrorKindPrompt, Operation: "load_prompt", AgentType: agentType, Cause: ErrPromptInvalid}
	}
}

func supportedAgentType(agentType string) bool {
	switch agentType {
	case AgentTypeResumeProfileExtractor, AgentTypeJobRequirementExtractor, AgentTypeCandidateMatchEvaluator:
		return true
	default:
		return false
	}
}
