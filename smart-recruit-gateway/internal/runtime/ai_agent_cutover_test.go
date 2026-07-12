package runtime

import (
	"context"
	"testing"
)

func TestAIAgentCutoverPlanCoversChatAgentPromptConfigAndEmbeddingSmoke(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"aiAgent": "ai-agent"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildAIAgentCutoverPlan(context.Background(), table, "logic:50051", StaticTargetResolver{
		"ai-agent": "ai-agent:50066",
	})
	if err != nil {
		t.Fatalf("BuildAIAgentCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateCutover("ai-agent:50066"); err != nil {
		t.Fatalf("ValidateCutover returned error: %v", err)
	}
	required := map[string]bool{
		"AIService.Chat":                                false,
		"AIService.ChatStream":                          false,
		"AIService.CreateAgentRun":                      false,
		"AIService.GetAgentRun":                         false,
		"AIService.SubscribeAgentRunEvents":             false,
		"AIService.ConfirmAgentRun":                     false,
		"PromptService.ListPromptTemplates":             false,
		"PromptService.RenderPrompt":                    false,
		"AgentConfigService.ListAgents":                 false,
		"AgentConfigService.GetAgentConfig":             false,
		"EmbeddingConfigService.ListEmbeddingProviders": false,
		"EmbeddingConfigService.ListEmbeddingModels":    false,
		"EmbeddingConfigService.BackfillEmbeddings":     false,
	}
	for _, method := range plan.SmokeMethods {
		if _, ok := required[method]; ok {
			required[method] = true
		}
	}
	for method, seen := range required {
		if !seen {
			t.Fatalf("ai-agent smoke plan missing %s", method)
		}
	}
}

func TestAIAgentRollbackPlanUsesLogicOnly(t *testing.T) {
	table, err := NewRouteTable(map[string]string{"aiAgent": "logic"})
	if err != nil {
		t.Fatalf("NewRouteTable returned error: %v", err)
	}
	plan, err := BuildAIAgentCutoverPlan(context.Background(), table, "logic:50051", nil)
	if err != nil {
		t.Fatalf("BuildAIAgentCutoverPlan returned error: %v", err)
	}
	if err := plan.ValidateRollback("logic:50051"); err != nil {
		t.Fatalf("ValidateRollback returned error: %v", err)
	}
}
