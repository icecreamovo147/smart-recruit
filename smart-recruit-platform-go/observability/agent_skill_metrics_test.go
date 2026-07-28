package observability

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAgentSkillMetricsAreConcurrentAndBounded(t *testing.T) {
	registry := NewRegistry("ai-agent")
	const workers = 32
	var wait sync.WaitGroup
	wait.Add(workers)
	for index := 0; index < workers; index++ {
		go func() {
			defer wait.Done()
			registry.RecordAgentSkillSelection(AgentSkillMetricLabels{
				RuntimeMode:   "tenant-secret-runtime",
				ExecutionMode: "user-secret-execution",
				SelectionMode: "skill-123456",
				Role:          "private-role",
				Risk:          "candidate@example.invalid",
				Result:        "version-987654",
				Reason:        "PRIVATE_CANDIDATE_NAME",
			})
			registry.ObserveAgentSkillTokens(AgentSkillMetricLabels{
				RuntimeMode: "v2", ExecutionMode: "durable", SelectionMode: "auto",
				Role: "primary", Risk: "low", Result: "included", Reason: "core_included",
			}, 321)
			registry.ObserveAgentSkillRetrieval(AgentSkillMetricLabels{
				RuntimeMode: "v2", ExecutionMode: "durable", SelectionMode: "auto",
				Role: "none", Risk: "unknown", Result: "included", Reason: "core_included",
			}, 25*time.Millisecond)
		}()
	}
	wait.Wait()

	output := registry.Prometheus()
	for _, forbidden := range []string{
		"tenant-secret-runtime",
		"user-secret-execution",
		"skill-123456",
		"candidate@example.invalid",
		"version-987654",
		"PRIVATE_CANDIDATE_NAME",
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("metrics leaked unbounded/private label %q:\n%s", forbidden, output)
		}
	}
	if !strings.Contains(output, `smart_recruit_agent_skill_selection_total{runtime_mode="v2",execution_mode="direct",selection_mode="none",role="none",risk="unknown",result="error",reason="other"} 32`) {
		t.Fatalf("bounded selection counter missing:\n%s", output)
	}
	if !strings.Contains(output, `smart_recruit_agent_skill_tokens_count{runtime_mode="v2",execution_mode="durable",selection_mode="auto",role="primary",risk="low",result="included",reason="core_included"} 32`) {
		t.Fatalf("token histogram count missing:\n%s", output)
	}
	if !strings.Contains(output, `smart_recruit_agent_skill_retrieval_duration_seconds_count{runtime_mode="v2",execution_mode="durable",selection_mode="auto",role="none",risk="unknown",result="included",reason="core_included"} 32`) {
		t.Fatalf("retrieval histogram count missing:\n%s", output)
	}
}

func TestAgentSkillMetricsExposeReservedLifecycleHooksWithoutRegistration(t *testing.T) {
	registry := NewRegistry("ai-agent")
	labels := AgentSkillMetricLabels{
		RuntimeMode: "v2", ExecutionMode: "durable", SelectionMode: "manual",
		Role: "primary", Risk: "high", Result: "confirmation_required", Reason: "confirmation_required",
	}
	registry.RecordAgentSkillConfirmation(labels)
	labels.Result = "failed"
	labels.Reason = "output_invalid"
	registry.RecordAgentSkillOutputValidation(labels)
	registry.RecordAgentSkillBudgetDrop(AgentSkillMetricLabels{
		RuntimeMode: "v2", ExecutionMode: "direct", SelectionMode: "auto",
		Role: "supporting", Risk: "medium", Result: "dropped", Reason: "section_budget_exceeded",
	})

	output := registry.Prometheus()
	for _, name := range []string{
		"smart_recruit_agent_skill_confirmation_total",
		"smart_recruit_agent_skill_output_validation_total",
		"smart_recruit_agent_skill_budget_drop_total",
	} {
		if !strings.Contains(output, name) {
			t.Fatalf("%s missing from metrics output:\n%s", name, output)
		}
	}
}
