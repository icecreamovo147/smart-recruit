package grpc

import (
	"context"

	"go.uber.org/zap"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	platformlogger "smart-recruit-platform-go/logger"
)

// recruitingRuntimeObserver intentionally maps only the recruiting runtime's
// fixed allow-list. It must never accept arbitrary fields or error strings.
type recruitingRuntimeObserver struct {
	log *zap.Logger
}

func newRecruitingRuntimeObserver() recruitingRuntimeObserver {
	return recruitingRuntimeObserver{log: platformlogger.L()}
}

func (o recruitingRuntimeObserver) ObserveRecruitingRuntime(_ context.Context, event recruitingruntime.Observation) {
	if o.log == nil {
		return
	}
	event = recruitingruntime.NormalizeObservation(event)
	o.log.Info("recruiting intelligence runtime stage",
		zap.String("operation", event.Operation),
		zap.String("request_id", event.RequestID),
		zap.String("resource_type", event.ResourceType),
		zap.Int64("resource_id", event.ResourceID),
		zap.String("stage", event.Stage),
		zap.Bool("terminal", event.Terminal),
		zap.String("category", event.Category),
		zap.String("agent_type", event.AgentType),
		zap.Int64("prompt_id", event.PromptID),
		zap.String("prompt_name", event.PromptName),
		zap.Int32("prompt_version", event.PromptVersion),
		zap.String("model", event.ModelName),
		zap.String("fallback", event.Fallback),
		zap.String("outcome", event.Outcome),
		zap.String("parser_version", event.ParserVersion),
		zap.String("scorer_version", event.ScorerVersion),
		zap.Int32("input_count", event.InputCount),
		zap.Int32("output_count", event.OutputCount),
		zap.Int32("evidence_count", event.EvidenceCount),
		zap.Int32("requirement_count", event.RequirementCount),
		zap.Int64("duration_ms", event.Duration.Milliseconds()),
	)
}
