package observability

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	agentSkillSelectionMetric        = "smart_recruit_agent_skill_selection_total"
	agentSkillConfirmationMetric     = "smart_recruit_agent_skill_confirmation_total"
	agentSkillBudgetDropMetric       = "smart_recruit_agent_skill_budget_drop_total"
	agentSkillOutputValidationMetric = "smart_recruit_agent_skill_output_validation_total"
)

var (
	agentSkillTokenBuckets   = []float64{1, 10, 50, 100, 250, 500, 800, 1200, 2000, 3000, 5000, 10000}
	agentSkillRuntimeModes   = labelSet("v2", "disabled")
	agentSkillExecutionModes = labelSet("direct", "durable")
	agentSkillSelectionModes = labelSet("auto", "manual", "none")
	agentSkillRoles          = labelSet("primary", "supporting", "none")
	agentSkillRisks          = labelSet("low", "medium", "high", "critical", "unknown")
	agentSkillResults        = labelSet("included", "dropped", "confirmation_required", "disabled", "no_match", "error", "queued", "passed", "failed", "unavailable")
	agentSkillReasons        = labelSet(
		"none",
		"no_match",
		"skill_v2_disabled",
		"core_included",
		"section_included",
		"core_budget_exceeded",
		"section_budget_exceeded",
		"blocked_by_primary_budget",
		"skill_limit_exceeded",
		"confirmation_required",
		"blocked_by_confirmation",
		"critical_auto_excluded",
		"primary_limit_exceeded",
		"supporting_limit_exceeded",
		"supporting_requires_primary",
		"composition_conflict",
		"composition_role_invalid",
		"manual_selection_invalid",
		"manual_selection_limit_exceeded",
		"manual_invocation_disabled",
		"outside_capability_release",
		"required_capability_missing",
		"agent_type_mismatch",
		"skill_disabled",
		"package_invalid",
		"package_integrity_failed",
		"package_store_unavailable",
		"package_lookup_failed",
		"version_unavailable",
		"section_integrity_failed",
		"section_not_relevant",
		"judge_queued",
		"judge_queue_full",
		"judge_runner_unavailable",
		"judge_shutdown",
		"judge_passed",
		"judge_failed",
		"judge_error",
		"output_valid",
		"output_invalid",
		"approved",
		"rejected",
		"expired",
		"canceled",
		"consumed",
		"replayed",
		"binding_mismatch",
		"advisory_valid",
		"advisory_invalid",
		"strict_valid",
		"strict_invalid",
		"schema_mismatch",
	)
)

// AgentSkillMetricLabels is deliberately bounded. Callers may provide arbitrary
// strings, but each field is reduced to one of the finite allowlisted values
// before it can become a Prometheus label.
type AgentSkillMetricLabels struct {
	RuntimeMode   string
	ExecutionMode string
	SelectionMode string
	Role          string
	Risk          string
	Result        string
	Reason        string
}

type agentSkillMetricKey struct {
	Name          string
	RuntimeMode   string
	ExecutionMode string
	SelectionMode string
	Role          string
	Risk          string
	Result        string
	Reason        string
}

type agentSkillMetricsSnapshot struct {
	Counters         []agentSkillCounterSnapshot
	TokenHistograms  []agentSkillHistogramSnapshot
	RetrievalLatency []agentSkillHistogramSnapshot
}

type agentSkillCounterSnapshot struct {
	Key   agentSkillMetricKey
	Count uint64
}

type agentSkillHistogramSnapshot struct {
	Key     agentSkillMetricKey
	Count   uint64
	Sum     float64
	Buckets []uint64
}

func (r *Registry) RecordAgentSkillSelection(labels AgentSkillMetricLabels) {
	r.recordAgentSkillCounter(agentSkillSelectionMetric, labels)
}

func (r *Registry) RecordAgentSkillConfirmation(labels AgentSkillMetricLabels) {
	r.recordAgentSkillCounter(agentSkillConfirmationMetric, labels)
}

func (r *Registry) RecordAgentSkillBudgetDrop(labels AgentSkillMetricLabels) {
	r.recordAgentSkillCounter(agentSkillBudgetDropMetric, labels)
}

func (r *Registry) RecordAgentSkillOutputValidation(labels AgentSkillMetricLabels) {
	r.recordAgentSkillCounter(agentSkillOutputValidationMetric, labels)
}

func (r *Registry) ObserveAgentSkillTokens(labels AgentSkillMetricLabels, tokens int) {
	if r == nil {
		return
	}
	if tokens < 0 {
		tokens = 0
	}
	r.observeAgentSkillHistogram(
		normalizeAgentSkillMetricKey("smart_recruit_agent_skill_tokens", labels),
		float64(tokens),
		agentSkillTokenBuckets,
		r.agentSkillTokenHistograms,
	)
}

func (r *Registry) ObserveAgentSkillRetrieval(labels AgentSkillMetricLabels, elapsed time.Duration) {
	if r == nil {
		return
	}
	if elapsed < 0 {
		elapsed = 0
	}
	r.observeAgentSkillHistogram(
		normalizeAgentSkillMetricKey("smart_recruit_agent_skill_retrieval_duration_seconds", labels),
		elapsed.Seconds(),
		defaultBuckets,
		r.agentSkillRetrievalLatency,
	)
}

func (r *Registry) recordAgentSkillCounter(name string, labels AgentSkillMetricLabels) {
	if r == nil {
		return
	}
	key := normalizeAgentSkillMetricKey(name, labels)
	r.mu.Lock()
	r.agentSkillCounters[key]++
	r.mu.Unlock()
}

func (r *Registry) observeAgentSkillHistogram(key agentSkillMetricKey, value float64, buckets []float64, destination map[agentSkillMetricKey]*histogram) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := destination[key]
	if item == nil {
		item = &histogram{Buckets: make([]uint64, len(buckets))}
		destination[key] = item
	}
	item.Count++
	item.Sum += value
	for index, bucket := range buckets {
		if value <= bucket {
			item.Buckets[index]++
		}
	}
}

func (r *Registry) agentSkillMetricsSnapshotLocked() agentSkillMetricsSnapshot {
	snapshot := agentSkillMetricsSnapshot{
		Counters:         make([]agentSkillCounterSnapshot, 0, len(r.agentSkillCounters)),
		TokenHistograms:  make([]agentSkillHistogramSnapshot, 0, len(r.agentSkillTokenHistograms)),
		RetrievalLatency: make([]agentSkillHistogramSnapshot, 0, len(r.agentSkillRetrievalLatency)),
	}
	for key, count := range r.agentSkillCounters {
		snapshot.Counters = append(snapshot.Counters, agentSkillCounterSnapshot{Key: key, Count: count})
	}
	for key, item := range r.agentSkillTokenHistograms {
		snapshot.TokenHistograms = append(snapshot.TokenHistograms, copyAgentSkillHistogram(key, item))
	}
	for key, item := range r.agentSkillRetrievalLatency {
		snapshot.RetrievalLatency = append(snapshot.RetrievalLatency, copyAgentSkillHistogram(key, item))
	}
	sort.Slice(snapshot.Counters, func(i, j int) bool {
		return agentSkillMetricKeyLess(snapshot.Counters[i].Key, snapshot.Counters[j].Key)
	})
	sort.Slice(snapshot.TokenHistograms, func(i, j int) bool {
		return agentSkillMetricKeyLess(snapshot.TokenHistograms[i].Key, snapshot.TokenHistograms[j].Key)
	})
	sort.Slice(snapshot.RetrievalLatency, func(i, j int) bool {
		return agentSkillMetricKeyLess(snapshot.RetrievalLatency[i].Key, snapshot.RetrievalLatency[j].Key)
	})
	return snapshot
}

func copyAgentSkillHistogram(key agentSkillMetricKey, item *histogram) agentSkillHistogramSnapshot {
	if item == nil {
		return agentSkillHistogramSnapshot{Key: key}
	}
	return agentSkillHistogramSnapshot{
		Key:     key,
		Count:   item.Count,
		Sum:     item.Sum,
		Buckets: append([]uint64(nil), item.Buckets...),
	}
}

func writeAgentSkillMetrics(builder *strings.Builder, snapshot agentSkillMetricsSnapshot) {
	counterHelp := map[string]string{
		agentSkillSelectionMetric:        "Agent Skill Package selection decisions.",
		agentSkillConfirmationMetric:     "Agent Skill confirmation lifecycle decisions.",
		agentSkillBudgetDropMetric:       "Agent Skill Package content dropped by the runtime budget.",
		agentSkillOutputValidationMetric: "Agent Skill output validation and optional judge results.",
	}
	for _, name := range []string{
		agentSkillSelectionMetric,
		agentSkillConfirmationMetric,
		agentSkillBudgetDropMetric,
		agentSkillOutputValidationMetric,
	} {
		fmt.Fprintf(builder, "# HELP %s %s\n", name, counterHelp[name])
		fmt.Fprintf(builder, "# TYPE %s counter\n", name)
		for _, item := range snapshot.Counters {
			if item.Key.Name == name {
				fmt.Fprintf(builder, "%s%s %d\n", name, agentSkillLabels(item.Key, ""), item.Count)
			}
		}
	}
	writeAgentSkillHistogram(builder, "smart_recruit_agent_skill_tokens", "Agent Skill tokens loaded per decision.", agentSkillTokenBuckets, snapshot.TokenHistograms)
	writeAgentSkillHistogram(builder, "smart_recruit_agent_skill_retrieval_duration_seconds", "Agent Skill retrieval and composition latency.", defaultBuckets, snapshot.RetrievalLatency)
}

func writeAgentSkillHistogram(builder *strings.Builder, name, help string, buckets []float64, snapshots []agentSkillHistogramSnapshot) {
	fmt.Fprintf(builder, "# HELP %s %s\n", name, help)
	fmt.Fprintf(builder, "# TYPE %s histogram\n", name)
	for _, item := range snapshots {
		for index, bucket := range buckets {
			fmt.Fprintf(builder, "%s_bucket%s %d\n", name, agentSkillLabels(item.Key, fmt.Sprintf("%g", bucket)), item.Buckets[index])
		}
		fmt.Fprintf(builder, "%s_bucket%s %d\n", name, agentSkillLabels(item.Key, "+Inf"), item.Count)
		fmt.Fprintf(builder, "%s_sum%s %.9f\n", name, agentSkillLabels(item.Key, ""), item.Sum)
		fmt.Fprintf(builder, "%s_count%s %d\n", name, agentSkillLabels(item.Key, ""), item.Count)
	}
}

func normalizeAgentSkillMetricKey(name string, labels AgentSkillMetricLabels) agentSkillMetricKey {
	return agentSkillMetricKey{
		Name:          name,
		RuntimeMode:   boundedAgentSkillLabel(labels.RuntimeMode, agentSkillRuntimeModes, "v2"),
		ExecutionMode: boundedAgentSkillLabel(labels.ExecutionMode, agentSkillExecutionModes, "direct"),
		SelectionMode: boundedAgentSkillLabel(labels.SelectionMode, agentSkillSelectionModes, "none"),
		Role:          boundedAgentSkillLabel(labels.Role, agentSkillRoles, "none"),
		Risk:          boundedAgentSkillLabel(labels.Risk, agentSkillRisks, "unknown"),
		Result:        boundedAgentSkillLabel(labels.Result, agentSkillResults, "error"),
		Reason:        boundedAgentSkillLabel(labels.Reason, agentSkillReasons, "other"),
	}
}

func boundedAgentSkillLabel(value string, allowed map[string]struct{}, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if _, exists := allowed[value]; exists {
		return value
	}
	return fallback
}

func labelSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func agentSkillLabels(key agentSkillMetricKey, le string) string {
	labels := []string{
		fmt.Sprintf(`runtime_mode="%s"`, escapeLabel(key.RuntimeMode)),
		fmt.Sprintf(`execution_mode="%s"`, escapeLabel(key.ExecutionMode)),
		fmt.Sprintf(`selection_mode="%s"`, escapeLabel(key.SelectionMode)),
		fmt.Sprintf(`role="%s"`, escapeLabel(key.Role)),
		fmt.Sprintf(`risk="%s"`, escapeLabel(key.Risk)),
		fmt.Sprintf(`result="%s"`, escapeLabel(key.Result)),
		fmt.Sprintf(`reason="%s"`, escapeLabel(key.Reason)),
	}
	if le != "" {
		labels = append(labels, fmt.Sprintf(`le="%s"`, escapeLabel(le)))
	}
	return "{" + strings.Join(labels, ",") + "}"
}

func agentSkillMetricKeyLess(left, right agentSkillMetricKey) bool {
	leftValues := []string{left.Name, left.RuntimeMode, left.ExecutionMode, left.SelectionMode, left.Role, left.Risk, left.Result, left.Reason}
	rightValues := []string{right.Name, right.RuntimeMode, right.ExecutionMode, right.SelectionMode, right.Role, right.Risk, right.Result, right.Reason}
	for index := range leftValues {
		if leftValues[index] != rightValues[index] {
			return leftValues[index] < rightValues[index]
		}
	}
	return false
}
