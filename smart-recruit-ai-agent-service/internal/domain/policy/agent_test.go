package policy

import (
	"errors"
	"testing"

	"smart-recruit-ai-agent-service/internal/domain/model"
)

func TestValidateAgentRunTransitionMatchesDurableStateMachine(t *testing.T) {
	allowed := [][2]string{
		{model.AgentRunStatusQueued, model.AgentRunStatusPlanning},
		{model.AgentRunStatusPlanning, model.AgentRunStatusRunning},
		{model.AgentRunStatusRunning, model.AgentRunStatusWaitingConfirmation},
		{model.AgentRunStatusWaitingConfirmation, model.AgentRunStatusRunning},
		{model.AgentRunStatusCancelRequested, model.AgentRunStatusCanceled},
		{model.AgentRunStatusSucceeded, model.AgentRunStatusSucceeded},
	}
	for _, tc := range allowed {
		if err := ValidateAgentRunTransition(tc[0], tc[1]); err != nil {
			t.Fatalf("ValidateAgentRunTransition(%q,%q) returned %v", tc[0], tc[1], err)
		}
	}

	err := ValidateAgentRunTransition(model.AgentRunStatusSucceeded, model.AgentRunStatusRunning)
	if !errors.Is(err, ErrIllegalAgentRunTransition) {
		t.Fatalf("err = %v, want ErrIllegalAgentRunTransition", err)
	}
}

func TestPromptVersionAndProviderFallback(t *testing.T) {
	template := model.PromptTemplate{Content: "old", Version: 2}
	version, changed := NextPromptVersion(template, "new")
	if !changed || version != 3 {
		t.Fatalf("version=%d changed=%v, want version 3 changed", version, changed)
	}
	version, changed = NextPromptVersion(template, "old")
	if changed || version != 2 {
		t.Fatalf("version=%d changed=%v, want unchanged version 2", version, changed)
	}

	selected, err := SelectProvider(
		model.ProviderCandidate{Enabled: false},
		[]model.ProviderCandidate{{ID: 2, Enabled: true, FallbackAllowed: true, ModelName: "fallback"}},
		true,
	)
	if err != nil {
		t.Fatalf("SelectProvider returned %v", err)
	}
	if selected.ID != 2 {
		t.Fatalf("selected provider = %+v, want fallback id 2", selected)
	}
}

func TestValidateAgentRunRequestRequiresMessageOrAction(t *testing.T) {
	err := ValidateAgentRunRequest(1, 2, "req-1", "", "")
	if !errors.Is(err, ErrAgentRunRequestInvalid) {
		t.Fatalf("err = %v, want ErrAgentRunRequestInvalid", err)
	}
}
