package port

import (
	"context"

	"smart-recruit-ai-agent-service/internal/domain/model"
)

type DurableRunDispatcher interface {
	DispatchAgentRun(ctx context.Context, runID uint64) error
}

type ProviderSelector interface {
	SelectProvider(ctx context.Context, primary model.ProviderCandidate, fallbacks []model.ProviderCandidate) (model.ProviderCandidate, error)
}
