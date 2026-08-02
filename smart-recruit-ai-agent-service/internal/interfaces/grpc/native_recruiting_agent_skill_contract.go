package grpc

import (
	"context"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	embeddinginfra "smart-recruit-ai-agent-service/internal/infrastructure/provider"
)

type recruitingAgentSkillPackageLoader struct {
	store hrRuntimeAgentSkillPackageStore
}

func (l recruitingAgentSkillPackageLoader) LoadReleasedAgentSkillPackages(
	ctx context.Context,
	versionIDs []int64,
) ([]recruitingruntime.ReleasedAgentSkillPackage, error) {
	packages, err := l.store.LoadAgentSkillRuntimePackages(ctx, versionIDs)
	if err != nil {
		return nil, err
	}
	output := make([]recruitingruntime.ReleasedAgentSkillPackage, 0, len(packages))
	for _, runtimePackage := range packages {
		output = append(output, mapReleasedAgentSkillPackage(runtimePackage))
	}
	return output, nil
}

func mapReleasedAgentSkillPackage(runtimePackage embeddinginfra.AgentSkillRuntimePackage) recruitingruntime.ReleasedAgentSkillPackage {
	output := recruitingruntime.ReleasedAgentSkillPackage{
		ID: runtimePackage.ID, SkillID: runtimePackage.SkillID, Version: runtimePackage.Version,
		ManifestJSON: runtimePackage.ManifestJSON, CoreMarkdown: runtimePackage.CoreMarkdown,
		CompiledMarkdown: runtimePackage.CompiledMarkdown, CompiledHash: runtimePackage.CompiledHash,
		CoreEstimatedTokens: runtimePackage.CoreEstimatedTokens,
		Sections:            make([]recruitingruntime.ReleasedAgentSkillSection, 0, len(runtimePackage.Sections)),
	}
	for _, section := range runtimePackage.Sections {
		output.Sections = append(output.Sections, recruitingruntime.ReleasedAgentSkillSection{
			ID: section.ID, SectionKey: section.SectionKey, Title: section.Title, Description: section.Description,
			ContentMarkdown: section.ContentMarkdown, TriggerTerms: append([]string(nil), section.TriggerTerms...),
			SemanticTags: append([]string(nil), section.SemanticTags...), PlannerIntents: append([]string(nil), section.PlannerIntents...),
			Priority: section.Priority, Ordinal: section.Ordinal, EstimatedTokens: section.EstimatedTokens, ContentHash: section.ContentHash,
		})
	}
	return output
}

func (s nativeRecruitingIntelligenceService) withStrictValidationMetrics(
	ctx context.Context,
) (context.Context, *recruitingruntime.StrictValidationCollector) {
	return recruitingruntime.WithStrictValidationCollector(ctx)
}

func (s nativeRecruitingIntelligenceService) finalizeStrictValidationMetrics(
	ctx context.Context,
	collector *recruitingruntime.StrictValidationCollector,
) {
	if s.meter == nil || collector == nil {
		return
	}
	for _, outcome := range collector.Outcomes() {
		s.meter.recordAgentSkillOutputValidation(
			ctx,
			"auto",
			"primary",
			outcome.Risk,
			outcome.Result,
			outcome.Reason,
		)
	}
}
