package persistence

import (
	"context"
	"fmt"

	"smart-recruit-ai-agent-service/internal/application/agentskilleval"
)

type deterministicAgentSkillReleaseEvaluator struct {
	evaluator *agentskilleval.Evaluator
	err       error
}

var _ PlatformAIAgentSkillReleaseEvaluator = deterministicAgentSkillReleaseEvaluator{}

func newDeterministicAgentSkillReleaseEvaluator() PlatformAIAgentSkillReleaseEvaluator {
	evaluator, err := agentskilleval.NewDefaultEvaluator()
	return deterministicAgentSkillReleaseEvaluator{evaluator: evaluator, err: err}
}

func (e deterministicAgentSkillReleaseEvaluator) EvaluateAgentSkillRelease(
	ctx context.Context,
	input PlatformAIAgentSkillReleaseEvaluationInput,
) (PlatformAIAgentSkillReleaseEvaluationResult, error) {
	if e.err != nil {
		return PlatformAIAgentSkillReleaseEvaluationResult{}, fmt.Errorf("load deterministic suite: %w", e.err)
	}
	packages := make([]agentskilleval.ReleasePackage, 0, len(input.Packages))
	for _, item := range input.Packages {
		packages = append(packages, agentskilleval.ReleasePackage{
			SkillID:   item.SkillID,
			VersionID: item.VersionID,
			Package:   item.Package,
		})
	}
	result, err := e.evaluator.Evaluate(ctx, agentskilleval.Input{
		CapabilityKey: input.CapabilityKey,
		Audience:      input.Audience,
		Policy: agentskilleval.RuntimePolicy{
			PolicyVersion:  input.Policy.PolicyVersion,
			MaxSkillTokens: input.Policy.MaxSkillTokens,
			MaxInputRatio:  input.Policy.MaxInputRatio,
			MaxSkills:      input.Policy.MaxSkills,
		},
		Packages: packages,
	})
	if err != nil {
		return PlatformAIAgentSkillReleaseEvaluationResult{}, err
	}
	return PlatformAIAgentSkillReleaseEvaluationResult{
		Passed:       result.Passed,
		SuiteVersion: result.SuiteVersion,
		SuiteHash:    result.SuiteHash,
		ResultHash:   result.ResultHash,
		Cases:        append([]agentskilleval.CaseResult(nil), result.Cases...),
	}, nil
}
