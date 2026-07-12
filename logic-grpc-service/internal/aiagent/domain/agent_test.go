package domain

import (
	"testing"

	"logic-grpc-service/service"
)

func TestAgentTypeValuesRemainCompatible(t *testing.T) {
	t.Parallel()

	if AgentTypeHRRecruiting != "hr_recruiting_agent" {
		t.Fatalf("hr recruiting agent type drifted: %q", AgentTypeHRRecruiting)
	}
	if AgentTypeCandidateAssistant != "candidate_assistant" {
		t.Fatalf("candidate assistant agent type drifted: %q", AgentTypeCandidateAssistant)
	}
}

func TestAgentRunStatusValuesRemainCompatible(t *testing.T) {
	t.Parallel()

	if string(AgentRunStatusQueued) != service.AgentRunStatusQueued {
		t.Fatalf("queued status drifted: %q", AgentRunStatusQueued)
	}
	if string(AgentRunStatusWaitingConfirmation) != service.AgentRunStatusWaitingConfirmation {
		t.Fatalf("waiting confirmation status drifted: %q", AgentRunStatusWaitingConfirmation)
	}
	if string(AgentRunStatusSucceeded) != service.AgentRunStatusSucceeded {
		t.Fatalf("succeeded status drifted: %q", AgentRunStatusSucceeded)
	}
	if string(AgentRunStatusCanceled) != service.AgentRunStatusCanceled {
		t.Fatalf("canceled status drifted: %q", AgentRunStatusCanceled)
	}
}

func TestEmbeddingStatusValuesRemainCompatible(t *testing.T) {
	t.Parallel()

	if string(EmbeddingStatusReady) != service.EmbeddingStatusReady {
		t.Fatalf("ready embedding status drifted: %q", EmbeddingStatusReady)
	}
	if string(EmbeddingStatusUnavailable) != service.EmbeddingStatusUnavailable {
		t.Fatalf("unavailable embedding status drifted: %q", EmbeddingStatusUnavailable)
	}
	if string(EmbeddingStatusInactive) != service.EmbeddingStatusInactive {
		t.Fatalf("inactive embedding status drifted: %q", EmbeddingStatusInactive)
	}
}

func TestMCPPolicyDecisionValuesRemainCompatible(t *testing.T) {
	t.Parallel()

	if MCPPolicyDecisionAllow != "allow" {
		t.Fatalf("allow MCP policy decision drifted: %q", MCPPolicyDecisionAllow)
	}
	if MCPPolicyDecisionConfirmation != "confirmation_required" {
		t.Fatalf("confirmation MCP policy decision drifted: %q", MCPPolicyDecisionConfirmation)
	}
	if MCPPolicyDecisionRateLimited != "rate_limited" {
		t.Fatalf("rate limited MCP policy decision drifted: %q", MCPPolicyDecisionRateLimited)
	}
}
