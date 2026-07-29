package grpc

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"smart-recruit-platform-go/observability"
	"smart-recruit-proto/recruitment/pb"
)

const (
	agentSkillJudgeQueueSize         = 64
	agentSkillJudgeTimeout           = 10 * time.Second
	agentSkillJudgeMaxResponseRunes  = 4000
	agentSkillJudgeMaxCriteria       = 20
	agentSkillJudgeMaxCriterionRunes = 256
)

var (
	agentSkillJudgeEmailPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	agentSkillJudgePhonePattern = regexp.MustCompile(`(?:\+?86[-\s]?)?1[3-9]\d{9}`)
	agentSkillJudgeIDPattern    = regexp.MustCompile(`[0-9]{14,18}[0-9Xx]`)
)

type agentSkillExecutionModeContextKey struct{}
type agentSkillMetricsSuppressedContextKey struct{}

// AgentSkillJudgeInput intentionally contains no tenant/user/run identifiers,
// prompt, request, Tool result, resume, or job source body.
type AgentSkillJudgeInput struct {
	RedactedResponse   string
	EvaluationCriteria []string
}

type AgentSkillJudgeResult struct {
	Passed bool
}

// AgentSkillJudgeRunner is an optional production binding. The runtime remains
// non-blocking and observable when this binding is absent.
type AgentSkillJudgeRunner interface {
	Judge(context.Context, AgentSkillJudgeInput) (AgentSkillJudgeResult, error)
}

type agentSkillJudgeJob struct {
	Input  AgentSkillJudgeInput
	Labels observability.AgentSkillMetricLabels
}

type agentSkillJudgeDispatcher struct {
	enabled bool
	runner  AgentSkillJudgeRunner
	metrics *observability.Registry
	queue   chan agentSkillJudgeJob
	ctx     context.Context
	cancel  context.CancelFunc

	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
	worker    sync.WaitGroup
}

func newAgentSkillJudgeDispatcher(enabled bool, runner AgentSkillJudgeRunner, metrics *observability.Registry) *agentSkillJudgeDispatcher {
	dispatcher := &agentSkillJudgeDispatcher{
		enabled: enabled,
		runner:  runner,
		metrics: metrics,
	}
	if !enabled || runner == nil {
		return dispatcher
	}
	dispatcher.ctx, dispatcher.cancel = context.WithCancel(context.Background())
	dispatcher.queue = make(chan agentSkillJudgeJob, agentSkillJudgeQueueSize)
	dispatcher.worker.Add(1)
	go dispatcher.run()
	return dispatcher
}

func (d *agentSkillJudgeDispatcher) TrySubmit(input AgentSkillJudgeInput, labels observability.AgentSkillMetricLabels) {
	if d == nil || !d.enabled {
		return
	}
	if d.runner == nil {
		labels.Result = "unavailable"
		labels.Reason = "judge_runner_unavailable"
		d.metrics.RecordAgentSkillOutputValidation(labels)
		return
	}
	job := agentSkillJudgeJob{Input: cloneAgentSkillJudgeInput(input), Labels: labels}
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		labels.Result = "unavailable"
		labels.Reason = "judge_shutdown"
		d.metrics.RecordAgentSkillOutputValidation(labels)
		return
	}
	select {
	case d.queue <- job:
		d.mu.RUnlock()
		labels.Result = "queued"
		labels.Reason = "judge_queued"
		d.metrics.RecordAgentSkillOutputValidation(labels)
	default:
		d.mu.RUnlock()
		labels.Result = "unavailable"
		labels.Reason = "judge_queue_full"
		d.metrics.RecordAgentSkillOutputValidation(labels)
	}
}

func (d *agentSkillJudgeDispatcher) Close() {
	if d == nil {
		return
	}
	d.closeOnce.Do(func() {
		d.mu.Lock()
		d.closed = true
		cancel := d.cancel
		d.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	})
	d.worker.Wait()
}

func (d *agentSkillJudgeDispatcher) run() {
	defer d.worker.Done()
	for {
		select {
		case <-d.ctx.Done():
			return
		default:
		}
		select {
		case <-d.ctx.Done():
			return
		case job := <-d.queue:
			ctx, cancel := context.WithTimeout(d.ctx, agentSkillJudgeTimeout)
			result, err := d.runner.Judge(ctx, cloneAgentSkillJudgeInput(job.Input))
			cancel()
			labels := job.Labels
			switch {
			case err != nil:
				labels.Result = "failed"
				labels.Reason = "judge_error"
			case result.Passed:
				labels.Result = "passed"
				labels.Reason = "judge_passed"
			default:
				labels.Result = "failed"
				labels.Reason = "judge_failed"
			}
			d.metrics.RecordAgentSkillOutputValidation(labels)
		}
	}
}

func cloneAgentSkillJudgeInput(input AgentSkillJudgeInput) AgentSkillJudgeInput {
	return AgentSkillJudgeInput{
		RedactedResponse:   input.RedactedResponse,
		EvaluationCriteria: append([]string(nil), input.EvaluationCriteria...),
	}
}

func (s *nativeAIService) trySubmitAgentSkillJudge(
	ctx context.Context,
	response string,
	candidateName string,
	jobTitle string,
	governance hrRuntimeGovernanceContext,
) {
	if s == nil || s.agentSkillJudge == nil || !s.agentSkillJudge.enabled {
		return
	}
	criteria := make([]string, 0)
	for _, skill := range governance.SelectedAgentSkills {
		if !skill.Included {
			continue
		}
		criteria = append(criteria, skill.EvaluationCriteria...)
	}
	criteria = boundedAgentSkillJudgeCriteria(criteria)
	if len(criteria) == 0 {
		return
	}
	labels := observability.AgentSkillMetricLabels{
		RuntimeMode:   "v2",
		ExecutionMode: agentSkillExecutionMode(ctx),
		SelectionMode: governance.AgentSkillSelectionMode,
		Role:          "none",
		Risk:          "unknown",
	}
	s.agentSkillJudge.TrySubmit(AgentSkillJudgeInput{
		RedactedResponse:   redactAgentSkillJudgeResponse(response, candidateName, jobTitle),
		EvaluationCriteria: criteria,
	}, labels)
}

func boundedAgentSkillJudgeCriteria(values []string) []string {
	capacity := len(values)
	if capacity > agentSkillJudgeMaxCriteria {
		capacity = agentSkillJudgeMaxCriteria
	}
	out := make([]string, 0, capacity)
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = redactAndBoundAgentSkillJudgeText(value, agentSkillJudgeMaxCriterionRunes)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
		if len(out) == agentSkillJudgeMaxCriteria {
			break
		}
	}
	return out
}

func redactAgentSkillJudgeResponse(response string, sensitiveValues ...string) string {
	return redactAndBoundAgentSkillJudgeText(response, agentSkillJudgeMaxResponseRunes, sensitiveValues...)
}

func redactAndBoundAgentSkillJudgeText(value string, maxRunes int, sensitiveValues ...string) string {
	redacted := strings.TrimSpace(value)
	redacted = agentSkillJudgeEmailPattern.ReplaceAllString(redacted, "[EMAIL]")
	redacted = agentSkillJudgeIDPattern.ReplaceAllString(redacted, "[IDENTIFIER]")
	redacted = agentSkillJudgePhonePattern.ReplaceAllString(redacted, "[PHONE]")
	for _, value := range sensitiveValues {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		redacted = regexp.MustCompile(`(?i)`+regexp.QuoteMeta(value)).ReplaceAllString(redacted, "[ENTITY]")
	}
	return truncateAgentSkillJudgeText(redacted, maxRunes)
}

func truncateAgentSkillJudgeText(value string, maxRunes int) string {
	if maxRunes <= 0 || value == "" {
		return ""
	}
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	index := 0
	for offset := range value {
		if index == maxRunes {
			return value[:offset]
		}
		index++
	}
	return value
}

func withAgentSkillExecutionMode(ctx context.Context, durable bool) context.Context {
	mode := "direct"
	if durable {
		mode = "durable"
	}
	return context.WithValue(ctx, agentSkillExecutionModeContextKey{}, mode)
}

func agentSkillExecutionMode(ctx context.Context) string {
	if ctx != nil {
		if mode, ok := ctx.Value(agentSkillExecutionModeContextKey{}).(string); ok && mode == "durable" {
			return mode
		}
	}
	return "direct"
}

func withoutAgentSkillMetrics(ctx context.Context) context.Context {
	return context.WithValue(ctx, agentSkillMetricsSuppressedContextKey{}, true)
}

func agentSkillMetricsSuppressed(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	suppressed, _ := ctx.Value(agentSkillMetricsSuppressedContextKey{}).(bool)
	return suppressed
}

func disabledAgentSkillRuntimeEvidence(req *pb.ChatRequest, model RuntimeModelInfo) []*pb.AgentSkillRuntimeEvidence {
	versionIDs := positiveUniqueRuntimeIDs(req.GetAgentSkillVersionIds())
	if len(versionIDs) == 0 {
		versionIDs = positiveUniqueRuntimeIDs(model.ConfigurationRefs.AgentSkillVersionIDs)
	}
	if len(versionIDs) == 0 {
		return []*pb.AgentSkillRuntimeEvidence{{
			Included:       false,
			DecisionReason: "skill_v2_disabled",
		}}
	}
	evidence := make([]*pb.AgentSkillRuntimeEvidence, 0, len(versionIDs))
	for _, versionID := range versionIDs {
		evidence = append(evidence, &pb.AgentSkillRuntimeEvidence{
			VersionId:      versionID,
			Included:       false,
			DecisionReason: "skill_v2_disabled",
		})
	}
	return evidence
}

func (s *nativeAIService) recordAgentSkillRuntimeDecision(
	ctx context.Context,
	runtimeMode string,
	selectionMode string,
	evidence []*pb.AgentSkillRuntimeEvidence,
	governanceErrors []hrRuntimeGovernanceError,
	elapsed time.Duration,
) {
	if s == nil || s.metrics == nil || agentSkillMetricsSuppressed(ctx) {
		return
	}
	executionMode := agentSkillExecutionMode(ctx)
	summary := observability.AgentSkillMetricLabels{
		RuntimeMode:   runtimeMode,
		ExecutionMode: executionMode,
		SelectionMode: selectionMode,
		Role:          "none",
		Risk:          "unknown",
		Result:        "no_match",
		Reason:        "no_match",
	}
	switch {
	case runtimeMode == "disabled":
		summary.Result = "disabled"
		summary.Reason = "skill_v2_disabled"
	case len(governanceErrors) > 0:
		summary.Result = "error"
		summary.Reason = governanceErrors[0].Code
	case len(evidence) > 0:
		summary.Result, summary.Reason = agentSkillEvidenceResult(evidence[0])
		for _, item := range evidence {
			if item != nil && item.GetIncluded() {
				summary.Result, summary.Reason = agentSkillEvidenceResult(item)
				break
			}
		}
	}
	s.metrics.ObserveAgentSkillRetrieval(summary, elapsed)

	if len(evidence) == 0 {
		s.metrics.RecordAgentSkillSelection(summary)
		return
	}
	for _, item := range evidence {
		if item == nil {
			continue
		}
		labels := observability.AgentSkillMetricLabels{
			RuntimeMode:   runtimeMode,
			ExecutionMode: executionMode,
			SelectionMode: selectionMode,
			Role:          agentSkillMetricRole(item.GetCompositionRole()),
			Risk:          agentSkillMetricRisk(item.GetRisk()),
			Reason:        item.GetDecisionReason(),
		}
		labels.Result, labels.Reason = agentSkillEvidenceResult(item)
		s.metrics.RecordAgentSkillSelection(labels)
		if item.GetIncluded() {
			s.metrics.ObserveAgentSkillTokens(labels, int(item.GetLoadedTokens()))
		}
		if isAgentSkillBudgetDropReason(item.GetDecisionReason()) {
			s.metrics.RecordAgentSkillBudgetDrop(labels)
		}
		for _, section := range item.GetSections() {
			if section == nil || !isAgentSkillBudgetDropReason(section.GetDecisionReason()) {
				continue
			}
			sectionLabels := labels
			sectionLabels.Result = "dropped"
			sectionLabels.Reason = section.GetDecisionReason()
			s.metrics.RecordAgentSkillBudgetDrop(sectionLabels)
		}
	}
}

// recordAgentSkillConfirmation is the bounded metrics hook for the durable
// confirmation state machine. T13 owns when lifecycle outcomes are emitted.
func (s *nativeAIService) recordAgentSkillConfirmation(
	ctx context.Context,
	selectionMode string,
	role string,
	risk string,
	result string,
	reason string,
) {
	if s == nil || s.metrics == nil {
		return
	}
	s.metrics.RecordAgentSkillConfirmation(observability.AgentSkillMetricLabels{
		RuntimeMode:   agentSkillRuntimeMode(s.skillPackageV2Enabled),
		ExecutionMode: agentSkillExecutionMode(ctx),
		SelectionMode: selectionMode,
		Role:          role,
		Risk:          risk,
		Result:        result,
		Reason:        reason,
	})
}

// recordAgentSkillOutputValidation is the bounded metrics hook for advisory,
// strict, and judge validation. T15/T16 own the synchronous validation calls.
func (s *nativeAIService) recordAgentSkillOutputValidation(
	ctx context.Context,
	selectionMode string,
	role string,
	risk string,
	result string,
	reason string,
) {
	if s == nil || s.metrics == nil || agentSkillMetricsSuppressed(ctx) {
		return
	}
	s.metrics.RecordAgentSkillOutputValidation(observability.AgentSkillMetricLabels{
		RuntimeMode:   agentSkillRuntimeMode(s.skillPackageV2Enabled),
		ExecutionMode: agentSkillExecutionMode(ctx),
		SelectionMode: selectionMode,
		Role:          role,
		Risk:          risk,
		Result:        result,
		Reason:        reason,
	})
}

func (s *nativeAIService) recordAgentSkillAdvisoryResponse(
	ctx context.Context,
	governance hrRuntimeGovernanceContext,
) {
	primary, ok := primaryHRRuntimeAgentSkill(governance.SelectedAgentSkills)
	if !ok || primary.OutputContract.Mode != "advisory" || strings.TrimSpace(primary.AdvisoryInstruction) == "" {
		return
	}
	s.recordAgentSkillOutputValidation(
		ctx,
		governance.AgentSkillSelectionMode,
		"primary",
		string(primary.RiskLevel),
		"applied",
		"advisory_not_enforced",
	)
}

func (s *nativeAIService) recordAgentSkillStrictUnsupported(
	ctx context.Context,
	governance hrRuntimeGovernanceContext,
) {
	role, risk := "primary", "unknown"
	for _, item := range governance.AgentSkillRuntimeEvidence {
		if item == nil || item.GetDecisionReason() != "strict_output_contract_unsupported" {
			continue
		}
		role = agentSkillMetricRole(item.GetCompositionRole())
		risk = agentSkillMetricRisk(item.GetRisk())
		break
	}
	s.recordAgentSkillOutputValidation(
		ctx,
		governance.AgentSkillSelectionMode,
		role,
		risk,
		"unsupported",
		"strict_unsupported",
	)
}

func agentSkillRuntimeMode(enabled bool) string {
	if enabled {
		return "v2"
	}
	return "disabled"
}

func agentSkillEvidenceResult(item *pb.AgentSkillRuntimeEvidence) (string, string) {
	if item == nil {
		return "error", "other"
	}
	reason := strings.TrimSpace(item.GetDecisionReason())
	switch reason {
	case "skill_v2_disabled":
		return "disabled", reason
	case "confirmation_required", "blocked_by_confirmation":
		return "confirmation_required", reason
	}
	if item.GetIncluded() {
		return "included", reason
	}
	return "dropped", reason
}

func isAgentSkillBudgetDropReason(reason string) bool {
	switch strings.TrimSpace(reason) {
	case "core_budget_exceeded", "section_budget_exceeded", "blocked_by_primary_budget":
		return true
	default:
		return false
	}
}

func agentSkillMetricRole(role pb.AgentSkillCompositionRole) string {
	switch role {
	case pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_PRIMARY:
		return "primary"
	case pb.AgentSkillCompositionRole_AGENT_SKILL_COMPOSITION_ROLE_SUPPORTING:
		return "supporting"
	default:
		return "none"
	}
}

func agentSkillMetricRisk(risk pb.AgentSkillRiskLevel) string {
	switch risk {
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_LOW:
		return "low"
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_MEDIUM:
		return "medium"
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_HIGH:
		return "high"
	case pb.AgentSkillRiskLevel_AGENT_SKILL_RISK_LEVEL_CRITICAL:
		return "critical"
	default:
		return "unknown"
	}
}
