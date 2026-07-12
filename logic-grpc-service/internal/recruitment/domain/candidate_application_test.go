package domain

import "testing"

func TestCandidateApplicationOwnershipBoundariesAreExplicit(t *testing.T) {
	t.Parallel()

	if RecruitmentOwnsCandidateFacts == "" {
		t.Fatal("recruitment candidate facts boundary must be explicit")
	}
	if AIAgentOwnsDerivedIntelligence == "" {
		t.Fatal("AI-derived intelligence boundary must be explicit")
	}
	if RecruitmentOwnsCandidateFacts == AIAgentOwnsDerivedIntelligence {
		t.Fatal("recruitment facts and AI-derived intelligence boundaries must remain distinct")
	}
}
