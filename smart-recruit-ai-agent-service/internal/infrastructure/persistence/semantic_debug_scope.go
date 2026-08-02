package persistence

import (
	"context"
	"fmt"
	"strings"
)

func (s *NativeStore) ResolveSemanticDebugAgentSkillVersionIDs(
	ctx context.Context,
	agentType string,
) ([]int64, int64, error) {
	audience := ""
	switch strings.TrimSpace(agentType) {
	case "", "hr_recruiting_agent":
		audience = PlatformAIAudienceTenantHR
	case "candidate_assistant":
		audience = PlatformAIAudienceCandidate
	default:
		return nil, 0, fmt.Errorf("unsupported semantic debug agent_type %q", agentType)
	}
	_, version, snapshot, err := s.loadPublishedCapability(ctx, "ai.chat", audience, 0)
	if err != nil {
		return nil, 0, err
	}
	return append([]int64(nil), snapshot.ConfigurationRef.AgentSkillVersionIDs...), version.ID, nil
}
