package runtime

import (
	"context"
	"fmt"
)

var AIAgentSmokeMethods = []string{
	"AIService.Chat",
	"AIService.ChatStream",
	"AIService.CreateAgentRun",
	"AIService.GetAgentRun",
	"AIService.SubscribeAgentRunEvents",
	"AIService.ConfirmAgentRun",
	"PromptService.ListPromptTemplates",
	"PromptService.RenderPrompt",
	"AgentConfigService.ListAgents",
	"AgentConfigService.GetAgentConfig",
	"EmbeddingConfigService.ListEmbeddingProviders",
	"EmbeddingConfigService.ListEmbeddingModels",
	"EmbeddingConfigService.BackfillEmbeddings",
}

type AIAgentCutoverPlan struct {
	Mode         string
	Target       string
	ReadyTargets []RouteEntry
	SmokeMethods []string
}

func BuildAIAgentCutoverPlan(ctx context.Context, table RouteTable, logicTarget string, resolver TargetResolver) (AIAgentCutoverPlan, error) {
	resolved, err := table.ResolveTargets(ctx, logicTarget, resolver)
	if err != nil {
		return AIAgentCutoverPlan{}, err
	}
	entry, ok := resolved.Entry("ai-agent")
	if !ok {
		return AIAgentCutoverPlan{}, fmt.Errorf("ai-agent route entry is required")
	}
	return AIAgentCutoverPlan{
		Mode:         entry.Mode,
		Target:       entry.Target,
		ReadyTargets: resolved.ReadyTargets(logicTarget),
		SmokeMethods: append([]string(nil), AIAgentSmokeMethods...),
	}, nil
}

func (plan AIAgentCutoverPlan) ValidateCutover(aiAgentTarget string) error {
	if plan.Mode != "ai-agent" {
		return fmt.Errorf("ai-agent route mode = %q, want ai-agent", plan.Mode)
	}
	if plan.Target != aiAgentTarget {
		return fmt.Errorf("ai-agent route target = %q, want %q", plan.Target, aiAgentTarget)
	}
	if len(plan.SmokeMethods) == 0 {
		return fmt.Errorf("ai-agent smoke methods are required")
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "ai-agent" && target.Target == aiAgentTarget {
			return nil
		}
	}
	return fmt.Errorf("ai-agent ready target %q is missing", aiAgentTarget)
}

func (plan AIAgentCutoverPlan) ValidateRollback(logicTarget string) error {
	if plan.Mode != LogicService {
		return fmt.Errorf("ai-agent rollback mode = %q, want logic", plan.Mode)
	}
	if plan.Target != logicTarget {
		return fmt.Errorf("ai-agent rollback target = %q, want %q", plan.Target, logicTarget)
	}
	for _, target := range plan.ReadyTargets {
		if target.Service == "ai-agent" {
			return fmt.Errorf("ai-agent should not be a separate ready target during logic rollback")
		}
	}
	return nil
}
