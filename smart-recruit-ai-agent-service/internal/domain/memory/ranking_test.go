package memory

import "testing"

func TestComputeMemoryBusinessBoostGating(t *testing.T) {
	cfg := DefaultRankingConfig()
	relevanceLow := cfg.RelevanceGate - 0.01
	boostLow := ComputeMemoryBusinessBoost(cfg, relevanceLow, 1, 1)
	if boostLow != 1.0 {
		t.Fatalf("gated boost = %v, want 1.0", boostLow)
	}
	boostHigh := ComputeMemoryBusinessBoost(cfg, cfg.RelevanceGate+0.2, 1, 1)
	if boostHigh <= 1.0 {
		t.Fatalf("active boost = %v, want > 1.0", boostHigh)
	}
	if boostHigh > cfg.BusinessBoostMax {
		t.Fatalf("boost %v exceeds max %v", boostHigh, cfg.BusinessBoostMax)
	}
}

func TestRankMemoriesOrdering(t *testing.T) {
	cfg := DefaultRankingConfig()
	memories := []Memory{
		{ID: 1, Scope: Scope{Type: ScopeApplication, ID: 10}, Content: "prefer remote work", Importance: 0.5, Confidence: 0.8},
		{ID: 2, Scope: Scope{Type: ScopeApplication, ID: 10}, Content: "remote work preference", Importance: 0.9, Confidence: 0.9},
	}
	input := RecallQueryContext{
		Query:           "remote work",
		TargetScopeType: ScopeApplication,
		TargetScopeID:   10,
		QueryTokenCount: 2,
	}
	ranked := RankMemories(cfg, memories, input, nil)
	if len(ranked) != 2 {
		t.Fatalf("ranked len = %d, want 2", len(ranked))
	}
	if ranked[0].Memory.ID != 2 {
		t.Fatalf("top memory id = %d, want 2", ranked[0].Memory.ID)
	}
}

func TestRankMemoriesDropsZeroScore(t *testing.T) {
	cfg := DefaultRankingConfig()
	memories := []Memory{
		{ID: 1, Scope: Scope{Type: ScopeCandidate, ID: 999}, Content: "unrelated topic", Importance: 0.1, Confidence: 0.1},
	}
	input := RecallQueryContext{Query: "xyzzyplugh", TargetScopeType: ScopeApplication, TargetScopeID: 99, QueryTokenCount: 1}
	ranked := RankMemories(cfg, memories, input, nil)
	if len(ranked) != 0 {
		t.Fatalf("expected zero-score memories to drop, got %d", len(ranked))
	}
}

func TestScoreMemoryRankingLexicalFallbackMode(t *testing.T) {
	cfg := DefaultRankingConfig()
	memory := Memory{
		ID: 1, Scope: Scope{Type: ScopeHR, ID: 0},
		Content: "prefer remote work", Importance: 0.8, Confidence: 0.8,
	}
	input := RecallQueryContext{
		Query: "remote work", TargetScopeType: ScopeHR, TargetScopeID: 0,
		QueryTokenCount: 2, EmbeddingAvailable: false,
	}
	signals := ScoreMemoryRankingSignals(cfg, memory, input, 0)
	if signals.RelevanceMode != "fallback" {
		t.Fatalf("mode = %q, want fallback", signals.RelevanceMode)
	}
	if signals.LexicalScore <= 0 {
		t.Fatalf("lexical score = %v, want > 0", signals.LexicalScore)
	}
	if signals.FinalRankScore <= 0 {
		t.Fatalf("final rank score = %v, want > 0", signals.FinalRankScore)
	}
}

func TestRankMemoriesUsesLexicalWhenEmbeddingsUnavailable(t *testing.T) {
	cfg := DefaultRankingConfig()
	memories := []Memory{
		{ID: 1, Scope: Scope{Type: ScopeHR, ID: 0}, Content: "unrelated topic", Importance: 0.9, Confidence: 0.9},
		{ID: 2, Scope: Scope{Type: ScopeHR, ID: 0}, Content: "prefers remote interviews", Importance: 0.5, Confidence: 0.5},
	}
	input := RecallQueryContext{
		Query: "remote interview", TargetScopeType: ScopeHR, TargetScopeID: 0,
		QueryTokenCount: 2, EmbeddingAvailable: false,
	}
	ranked := RankMemories(cfg, memories, input, nil)
	if len(ranked) < 2 {
		t.Fatalf("ranked len = %d, want both memories with lexical ordering", len(ranked))
	}
	if ranked[0].Memory.ID != 2 {
		t.Fatalf("top memory id = %d, want 2", ranked[0].Memory.ID)
	}
	if ranked[0].Reason != "fallback" {
		t.Fatalf("reason = %q, want fallback", ranked[0].Reason)
	}
}

func TestTruncateRankedMemories(t *testing.T) {
	items := []RankedMemory{
		{Memory: Memory{Content: "1234567890"}},
		{Memory: Memory{Content: "1234567890"}},
	}
	got := TruncateRankedMemories(items, 1, 100)
	if len(got) != 1 {
		t.Fatalf("max memories truncate = %d, want 1", len(got))
	}
	got = TruncateRankedMemories(items, 10, 12)
	if len(got) != 1 {
		t.Fatalf("max chars truncate = %d, want 1", len(got))
	}
}
