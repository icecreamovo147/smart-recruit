package service

import (
	"math"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

func almostEqual(a, b, eps float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return false
	}
	return math.Abs(a-b) <= eps
}

func TestComputeRelevanceScore(t *testing.T) {
	cases := []struct {
		name     string
		vector   float64
		lexical  float64
		metadata float64
		want     float64
	}{
		{"all_zero", 0, 0, 0, 0},
		{"pure_vector", 1, 0, 0, rankingConfig.WeightVector},
		{"pure_lexical", 0, 1, 0, rankingConfig.WeightLexical},
		{"pure_metadata", 0, 0, 1, rankingConfig.WeightMetadata},
		{"mixed_balanced", 0.5, 0.5, 0.5, 0.5},
		{"full_signals", 1, 1, 1, 1.0},
		{"vector_dominant", 0.9, 0.1, 0.1, 0.9*0.6 + 0.1*0.3 + 0.1*0.1},
		{"clamp_negative", -0.5, 0.5, 0.5, 0.6*0 + 0.3*0.5 + 0.1*0.5},
		{"clamp_overflow", 1.5, 0, 0, 0.6}, // clamp01(1.5)=1.0, then 1.0*0.6 = 0.6
		{"nan_vector", math.NaN(), 0.5, 0.5, 0.3*0.5 + 0.1*0.5},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computeRelevanceScore(c.vector, c.lexical, c.metadata)
			if got < 0 || got > 1 {
				t.Fatalf("relevance out of range: got %v", got)
			}
			if !almostEqual(got, c.want, 1e-9) {
				t.Fatalf("computeRelevanceScore(%v,%v,%v) = %v, want %v",
					c.vector, c.lexical, c.metadata, got, c.want)
			}
		})
	}
}

func TestComputeBusinessBoost(t *testing.T) {
	t.Run("skill_zero_priority", func(t *testing.T) {
		got := computeBusinessBoost(0, 0, 0, true)
		if !almostEqual(got, 1.0, 1e-9) {
			t.Fatalf("skill zero priority should be 1.0, got %v", got)
		}
	})

	t.Run("skill_priority_50", func(t *testing.T) {
		// priority_normalized = 1.0 → boost = 1 + 0.2 = 1.2
		got := computeBusinessBoost(50, 0, 0, true)
		if !almostEqual(got, 1.2, 1e-9) {
			t.Fatalf("skill priority 50 should be 1.2, got %v", got)
		}
	})

	t.Run("skill_priority_100_clamped", func(t *testing.T) {
		// priority_normalized = clamp01(100/50) = 1.0 → boost = 1.2
		got := computeBusinessBoost(100, 0, 0, true)
		if !almostEqual(got, 1.2, 1e-9) {
			t.Fatalf("skill priority 100 should be 1.2, got %v", got)
		}
	})

	t.Run("skill_priority_huge_clamped_to_max", func(t *testing.T) {
		// priority_normalized = 1.0; 1 + 0.2 = 1.2（boost max = 1.5）
		got := computeBusinessBoost(1000, 0, 0, true)
		if !almostEqual(got, 1.2, 1e-9) {
			t.Fatalf("skill huge priority should be 1.2, got %v", got)
		}
	})

	t.Run("skill_priority_negative_clamped", func(t *testing.T) {
		// priority_normalized = 0
		got := computeBusinessBoost(-10, 0, 0, true)
		if !almostEqual(got, 1.0, 1e-9) {
			t.Fatalf("skill negative priority should be 1.0, got %v", got)
		}
	})

	t.Run("memory_gated_zero_relevance", func(t *testing.T) {
		// importance / confidence 信号在相关性 gating 之前为 0；调用方传入 0 即可
		got := computeBusinessBoost(0, 0, 0, false)
		if !almostEqual(got, 1.0, 1e-9) {
			t.Fatalf("memory gated zero should be 1.0, got %v", got)
		}
	})

	t.Run("memory_active_signals", func(t *testing.T) {
		// imp=1, conf=1 → boost = 1 + 0.2 + 0.1 = 1.3
		got := computeBusinessBoost(0, 1, 1, false)
		if !almostEqual(got, 1.3, 1e-9) {
			t.Fatalf("memory active should be 1.3, got %v", got)
		}
	})

	t.Run("memory_clamped_to_max", func(t *testing.T) {
		// imp=1, conf=1 → 1.3 < 1.5
		got := computeBusinessBoost(0, 1, 1, false)
		if got > rankingConfig.BusinessBoostMax {
			t.Fatalf("memory boost should be <= 1.5, got %v", got)
		}
	})

	t.Run("memory_importance_out_of_range", func(t *testing.T) {
		// imp=2 → clamp01 → 1.0; 1 + 0.2 + 0.1*0 = 1.2
		got := computeBusinessBoost(0, 2, 0, false)
		if !almostEqual(got, 1.2, 1e-9) {
			t.Fatalf("memory importance=2 should clamp to 1, boost=1.2, got %v", got)
		}
	})

	t.Run("nan_priority", func(t *testing.T) {
		got := computeBusinessBoost(math.NaN(), 0, 0, true)
		if !almostEqual(got, 1.0, 1e-9) {
			t.Fatalf("nan priority should be 1.0, got %v", got)
		}
	})
}

func TestComputeFinalRankScore(t *testing.T) {
	t.Run("zero_relevance", func(t *testing.T) {
		got := computeFinalRankScore(0, 1.0)
		if got != 0 {
			t.Fatalf("zero relevance should be 0, got %v", got)
		}
	})

	t.Run("full_relevance_full_boost", func(t *testing.T) {
		got := computeFinalRankScore(1.0, 1.5)
		if !almostEqual(got, 1.5, 1e-9) {
			t.Fatalf("1.0 * 1.5 = 1.5 expected, got %v", got)
		}
	})

	t.Run("balanced", func(t *testing.T) {
		got := computeFinalRankScore(0.5, 1.2)
		if !almostEqual(got, 0.6, 1e-9) {
			t.Fatalf("0.5 * 1.2 = 0.6 expected, got %v", got)
		}
	})

	t.Run("nan_boost_uses_one", func(t *testing.T) {
		got := computeFinalRankScore(0.5, math.NaN())
		if !almostEqual(got, 0.5, 1e-9) {
			t.Fatalf("nan boost should fall back to 1.0, got %v", got)
		}
	})
}

func TestMemoryImportanceGatedByRelevance(t *testing.T) {
	t.Run("below_threshold", func(t *testing.T) {
		if got := memoryImportanceSignal(0.10, 1.0); got != 0 {
			t.Fatalf("below gate should return 0, got %v", got)
		}
		if got := memoryConfidenceSignal(0.10, 1.0); got != 0 {
			t.Fatalf("below gate should return 0, got %v", got)
		}
	})

	t.Run("at_threshold", func(t *testing.T) {
		if got := memoryImportanceSignal(rankingConfig.RelevanceGate, 1.0); !almostEqual(got, 1.0, 1e-9) {
			t.Fatalf("at gate should pass through, got %v", got)
		}
	})

	t.Run("above_threshold", func(t *testing.T) {
		if got := memoryImportanceSignal(0.5, 0.8); !almostEqual(got, 0.8, 1e-9) {
			t.Fatalf("above gate should pass through, got %v", got)
		}
		if got := memoryConfidenceSignal(0.5, 0.6); !almostEqual(got, 0.6, 1e-9) {
			t.Fatalf("above gate should pass through, got %v", got)
		}
	})

	t.Run("above_threshold_out_of_range", func(t *testing.T) {
		if got := memoryImportanceSignal(0.5, 1.5); !almostEqual(got, 1.0, 1e-9) {
			t.Fatalf("above gate out-of-range should clamp to 1, got %v", got)
		}
	})
}

func TestComputePoolConfidence(t *testing.T) {
	t.Run("empty_none", func(t *testing.T) {
		if got := ComputePoolConfidence(nil); got != RankConfidenceNone {
			t.Fatalf("empty pool should be none, got %v", got)
		}
	})

	t.Run("single_below_gate_none", func(t *testing.T) {
		rs := []RankingSignals{{FinalRankScore: 0.05}}
		if got := ComputePoolConfidence(rs); got != RankConfidenceNone {
			t.Fatalf("single below gate should be none, got %v", got)
		}
	})

	t.Run("single_above_gate_low", func(t *testing.T) {
		rs := []RankingSignals{{FinalRankScore: 0.5}}
		if got := ComputePoolConfidence(rs); got != RankConfidenceLow {
			t.Fatalf("single above gate should be low, got %v", got)
		}
	})

	t.Run("two_candidates_high_gap", func(t *testing.T) {
		rs := []RankingSignals{
			{FinalRankScore: 0.8},
			{FinalRankScore: 0.5},
		}
		if got := ComputePoolConfidence(rs); got != RankConfidenceHigh {
			t.Fatalf("gap=0.3 should be high, got %v", got)
		}
	})

	t.Run("two_candidates_medium_gap", func(t *testing.T) {
		rs := []RankingSignals{
			{FinalRankScore: 0.4},
			{FinalRankScore: 0.35},
		}
		if got := ComputePoolConfidence(rs); got != RankConfidenceMedium {
			t.Fatalf("gap=0.05 should be medium, got %v", got)
		}
	})

	t.Run("two_candidates_low_gap", func(t *testing.T) {
		rs := []RankingSignals{
			{FinalRankScore: 0.3},
			{FinalRankScore: 0.29},
		}
		if got := ComputePoolConfidence(rs); got != RankConfidenceLow {
			t.Fatalf("gap=0.01 should be low, got %v", got)
		}
	})

	t.Run("two_candidates_top1_below_gate_forces_low", func(t *testing.T) {
		rs := []RankingSignals{
			{FinalRankScore: 0.10},
			{FinalRankScore: 0.05},
		}
		if got := ComputePoolConfidence(rs); got != RankConfidenceLow {
			t.Fatalf("top1 below gate should force low, got %v", got)
		}
	})

	t.Run("nan_filtered_out", func(t *testing.T) {
		rs := []RankingSignals{
			{FinalRankScore: math.NaN()},
		}
		if got := ComputePoolConfidence(rs); got != RankConfidenceNone {
			t.Fatalf("all NaN should be none, got %v", got)
		}
	})

	t.Run("unsorted_input_handled", func(t *testing.T) {
		// 输入乱序也应当按 FinalRankScore desc 排序
		rs := []RankingSignals{
			{FinalRankScore: 0.3},
			{FinalRankScore: 0.8},
			{FinalRankScore: 0.5},
		}
		if got := ComputePoolConfidence(rs); got != RankConfidenceHigh {
			t.Fatalf("sorted top1=0.8, top2=0.5, gap=0.3 should be high, got %v", got)
		}
	})
}

// ------------------------------------------------------------------
// Skill 池：scoreSkillRankingSignals / RankSkillCandidates
// ------------------------------------------------------------------

func TestScoreSkillRankingSignalsAllModes(t *testing.T) {
	skill := repository.AgentSkillRuntimeRecord{
		ID:           1,
		Name:         "screening",
		DisplayName:  "Screening",
		Description:  "candidate screening",
		Category:     "sourcing",
		Scenario:     "resume review",
		BodyMarkdown: "review candidate",
		Priority:     20,
	}
	question := "candidate screening resume review"

	t.Run("vector_only_high", func(t *testing.T) {
		s := scoreSkillRankingSignals(question, skill, 0.95, true)
		if s.VectorScore < 0.9 {
			t.Fatalf("vector_score should be ~0.95, got %v", s.VectorScore)
		}
		if s.RelevanceMode != "vector_lexical_metadata" {
			t.Fatalf("expected vector_lexical_metadata, got %q", s.RelevanceMode)
		}
		if s.FinalRankScore <= 0 || s.FinalRankScore > 1.5 {
			t.Fatalf("final_rank_score out of range: %v", s.FinalRankScore)
		}
	})

	t.Run("lexical_only_fallback", func(t *testing.T) {
		s := scoreSkillRankingSignals(question, skill, 0, false)
		if s.VectorScore != 0 {
			t.Fatalf("vector_score should be 0 in fallback, got %v", s.VectorScore)
		}
		if s.RelevanceMode != "lexical_metadata" {
			t.Fatalf("expected lexical_metadata, got %q", s.RelevanceMode)
		}
		if s.LexicalScore <= 0 {
			t.Fatalf("lexical_score should be > 0 (tokens match), got %v", s.LexicalScore)
		}
		if s.FinalRankScore <= 0 {
			t.Fatalf("final_rank_score should be > 0, got %v", s.FinalRankScore)
		}
	})

	t.Run("hybrid", func(t *testing.T) {
		s := scoreSkillRankingSignals(question, skill, 0.7, true)
		if s.VectorScore < 0.6 || s.LexicalScore <= 0 {
			t.Fatalf("hybrid should have both vector and lexical, got vec=%v lex=%v", s.VectorScore, s.LexicalScore)
		}
		if s.RelevanceMode != "vector_lexical_metadata" {
			t.Fatalf("expected vector_lexical_metadata, got %q", s.RelevanceMode)
		}
	})

	t.Run("degraded_semantic_zero", func(t *testing.T) {
		// embedding service returns but this skill has no semantic score
		s := scoreSkillRankingSignals(question, skill, 0, true)
		if s.VectorScore != 0 {
			t.Fatalf("vector_score should be 0 when no semantic match, got %v", s.VectorScore)
		}
		if s.RelevanceMode != "lexical_metadata" {
			t.Fatalf("expected lexical_metadata, got %q", s.RelevanceMode)
		}
	})

	t.Run("no_match_returns_zero_final", func(t *testing.T) {
		empty := repository.AgentSkillRuntimeRecord{
			ID:           99,
			Name:         "unrelated",
			DisplayName:  "Unrelated",
			Description:  "totally different",
			BodyMarkdown: "xyz123",
		}
		s := scoreSkillRankingSignals("zzzqqq", empty, 0, false)
		if s.FinalRankScore != 0 {
			t.Fatalf("non-matching skill should have final=0, got %v", s.FinalRankScore)
		}
	})

	t.Run("nan_vector_handled", func(t *testing.T) {
		s := scoreSkillRankingSignals(question, skill, math.NaN(), true)
		if math.IsNaN(s.VectorScore) || s.VectorScore != 0 {
			t.Fatalf("NaN vector should become 0, got %v", s.VectorScore)
		}
	})
}

func TestRankSkillCandidatesAllModes(t *testing.T) {
	skills := []repository.AgentSkillRuntimeRecord{
		{ID: 1, Name: "screening", DisplayName: "Screening", Description: "candidate screening", BodyMarkdown: "review candidate", Category: "sourcing", Priority: 10, IsManualInvocable: 1},
		{ID: 2, Name: "offer", DisplayName: "Offer", Description: "candidate offer", BodyMarkdown: "prepare candidate offer", Category: "closing", Priority: 30, IsManualInvocable: 1},
		{ID: 3, Name: "unrelated", DisplayName: "Unrelated", Description: "薪资", BodyMarkdown: "沟通", Priority: 5, IsManualInvocable: 1},
	}

	t.Run("embedding_unavailable_lexical_metadata", func(t *testing.T) {
		// 全部 mode = lexical_metadata
		out := RankSkillCandidates("candidate", skills, nil, false)
		if len(out) == 0 {
			t.Fatalf("expected non-empty result")
		}
		for _, s := range out {
			if s.RelevanceMode != "lexical_metadata" {
				t.Fatalf("expected lexical_metadata, got %q for skill %d", s.RelevanceMode, s.ID)
			}
			if s.VectorScore != 0 {
				t.Fatalf("vector_score should be 0, got %v for skill %d", s.VectorScore, s.ID)
			}
		}
	})

	t.Run("embedding_available_vector_lexical_metadata", func(t *testing.T) {
		out := RankSkillCandidates("candidate", skills, map[int64]float64{2: 0.95}, true)
		hasVector := false
		for _, s := range out {
			if s.ID == 2 && s.VectorScore > 0.5 {
				hasVector = true
			}
		}
		if !hasVector {
			t.Fatalf("skill 2 should have vector_score > 0.5 with semanticScores={2: 0.95}")
		}
	})

	t.Run("mixed_signals_keep_separation", func(t *testing.T) {
		out := RankSkillCandidates("candidate", skills, map[int64]float64{1: 0.5, 2: 0.95, 3: 0.99}, true)
		// skill 3 (薪资/沟通) 与 "candidate" 不匹配，应当被过滤掉
		for _, s := range out {
			if s.ID == 3 {
				t.Fatalf("non-matching skill 3 should be filtered, got %+v", s)
			}
		}
	})

	t.Run("zero_score_candidates_dropped", func(t *testing.T) {
		// "abc" 与全部 skill 都不匹配
		out := RankSkillCandidates("abc", skills, nil, false)
		if len(out) != 0 {
			t.Fatalf("non-matching query should return empty, got %d candidates", len(out))
		}
	})

	t.Run("sort_by_final_rank_score_desc", func(t *testing.T) {
		// skill 2 priority=30、category=closing 都参与；skill 1 priority=10
		// 即便 semantic 给 skill 1 更高分，priority boost 仍能影响排序
		out := RankSkillCandidates("candidate", skills, map[int64]float64{1: 0.9, 2: 0.5}, true)
		if len(out) < 2 {
			t.Fatalf("expected >= 2 candidates, got %d", len(out))
		}
		for i := 1; i < len(out); i++ {
			if out[i-1].FinalRankScore < out[i].FinalRankScore {
				t.Fatalf("results not sorted by final_rank_score desc: %+v", out)
			}
		}
	})
}

func TestSkillPoolFallbackToLexicalMetadata(t *testing.T) {
	skill := repository.AgentSkillRuntimeRecord{
		ID:           7,
		Name:         "screening",
		DisplayName:  "Screening",
		Description:  "candidate screening resume review",
		BodyMarkdown: "review candidate resume carefully",
		Category:     "sourcing",
		Priority:     10,
	}
	out := RankSkillCandidates("candidate resume", []repository.AgentSkillRuntimeRecord{skill}, nil, false)
	if len(out) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(out))
	}
	got := out[0]
	if got.RelevanceMode != "lexical_metadata" {
		t.Fatalf("expected lexical_metadata, got %q", got.RelevanceMode)
	}
	if got.VectorScore != 0 {
		t.Fatalf("vector_score should be 0 in fallback, got %v", got.VectorScore)
	}
	if got.FinalRankScore <= 0 || got.FinalRankScore > rankingConfig.BusinessBoostMax {
		t.Fatalf("final_rank_score out of range: %v (max %v)", got.FinalRankScore, rankingConfig.BusinessBoostMax)
	}
	if got.BusinessBoost < 1.0 || got.BusinessBoost > rankingConfig.BusinessBoostMax {
		t.Fatalf("business_boost out of range: %v", got.BusinessBoost)
	}
	if !strings.Contains(got.Reason, "embedding unavailable") {
		t.Fatalf("reason should mention embedding unavailable, got %q", got.Reason)
	}
	// Score int 兼容映射
	expectedCompat := int(math.Round(got.FinalRankScore * 100))
	if got.Score != expectedCompat {
		t.Fatalf("compat Score = %d, want %d (= round(final*100))", got.Score, expectedCompat)
	}
}

// ------------------------------------------------------------------
// Memory 池：scoreMemoryRankingSignals / RankMemoryCandidates
// ------------------------------------------------------------------

func TestScoreMemoryRankingSignalsAllModes(t *testing.T) {
	mem := model.AIMemory{
		ID:         1,
		ScopeType:  "application",
		ScopeID:    99,
		MemoryType: "fact",
		Content:    "candidate lacks Go production experience",
		Confidence: 0.9,
		Importance: 1.0,
	}
	input := AgentContextInput{
		HrID:           7,
		JobID:          88,
		ApplicationID:  99,
		CurrentMessage: "does this candidate have enough Go experience",
	}

	t.Run("vector_high_with_semantic", func(t *testing.T) {
		s := scoreMemoryRankingSignals(mem, input, 0.95, true, 7)
		if s.VectorScore < 0.9 {
			t.Fatalf("vector_score should be ~0.95, got %v", s.VectorScore)
		}
		if s.RelevanceMode != "vector_lexical_metadata" {
			t.Fatalf("expected vector_lexical_metadata, got %q", s.RelevanceMode)
		}
		if s.FinalRankScore <= 0 || s.FinalRankScore > 1.5 {
			t.Fatalf("final_rank_score out of range: %v", s.FinalRankScore)
		}
		if s.BusinessBoost <= 1.0 {
			t.Fatalf("business_boost should be > 1.0 (importance+confidence gated), got %v", s.BusinessBoost)
		}
	})

	t.Run("lexical_only_fallback", func(t *testing.T) {
		s := scoreMemoryRankingSignals(mem, input, 0, false, 7)
		if s.VectorScore != 0 {
			t.Fatalf("vector_score should be 0 in fallback, got %v", s.VectorScore)
		}
		if s.RelevanceMode != "lexical_metadata" {
			t.Fatalf("expected lexical_metadata, got %q", s.RelevanceMode)
		}
		if s.LexicalScore <= 0 {
			t.Fatalf("lexical_score should be > 0 (Go token matches), got %v", s.LexicalScore)
		}
		if s.MetadataScore != 1.0 {
			t.Fatalf("metadata_score should be 1.0 (application match), got %v", s.MetadataScore)
		}
	})

	t.Run("hybrid", func(t *testing.T) {
		s := scoreMemoryRankingSignals(mem, input, 0.5, true, 7)
		if s.VectorScore < 0.4 || s.LexicalScore <= 0 {
			t.Fatalf("hybrid should have both vector and lexical, got vec=%v lex=%v", s.VectorScore, s.LexicalScore)
		}
	})

	t.Run("nan_vector_handled", func(t *testing.T) {
		s := scoreMemoryRankingSignals(mem, input, math.NaN(), true, 7)
		if s.VectorScore != 0 {
			t.Fatalf("NaN vector should become 0, got %v", s.VectorScore)
		}
	})
}

func TestRankMemoryCandidatesAllModes(t *testing.T) {
	mem1 := model.AIMemory{ID: 1, ScopeType: "application", ScopeID: 99, Content: "candidate lacks Go production experience", Confidence: 0.9, Importance: 1.0}
	mem2 := model.AIMemory{ID: 2, ScopeType: "job", ScopeID: 88, Content: "Job requires Go backend expertise", Confidence: 0.8, Importance: 0.6}
	mem3 := model.AIMemory{ID: 3, ScopeType: "hr", ScopeID: 0, Content: "HR prefers concise summaries", Confidence: 0.9, Importance: 0.9}
	mem4 := model.AIMemory{ID: 4, ScopeType: "application", ScopeID: 1, Content: "totally unrelated", Confidence: 0.5, Importance: 0.5}

	input := AgentContextInput{
		HrID:           7,
		JobID:          88,
		ApplicationID:  99,
		CurrentMessage: "does this candidate have enough Go experience",
	}
	all := []model.AIMemory{mem1, mem2, mem3, mem4}

	t.Run("embedding_unavailable_lexical_metadata", func(t *testing.T) {
		out := RankMemoryCandidates(input.CurrentMessage, all, input, nil, false)
		if len(out) == 0 {
			t.Fatalf("expected non-empty result")
		}
		for _, it := range out {
			if it.Signals.RelevanceMode != "lexical_metadata" {
				t.Fatalf("expected lexical_metadata, got %q for memory %d", it.Signals.RelevanceMode, it.Memory.ID)
			}
			if it.Signals.VectorScore != 0 {
				t.Fatalf("vector_score should be 0, got %v for memory %d", it.Signals.VectorScore, it.Memory.ID)
			}
			if it.Signals.FinalRankScore > rankingConfig.BusinessBoostMax {
				t.Fatalf("final_rank_score exceeds max: %v", it.Signals.FinalRankScore)
			}
		}
	})

	t.Run("application_memory_first_with_semantic", func(t *testing.T) {
		out := RankMemoryCandidates(input.CurrentMessage, all, input, map[uint64]float64{1: 0.95}, true)
		if len(out) == 0 {
			t.Fatalf("expected non-empty result")
		}
		if out[0].Memory.ID != 1 {
			t.Fatalf("application memory (ID=1) should be first, got ID=%d", out[0].Memory.ID)
		}
	})

	t.Run("zero_score_candidates_dropped", func(t *testing.T) {
		out := RankMemoryCandidates("zzz", all, input, nil, false)
		// "zzz" should not match any memory's content
		for _, it := range out {
			if it.Signals.FinalRankScore <= 0 {
				t.Fatalf("non-zero final_rank_score expected for kept items, got %v (id=%d)", it.Signals.FinalRankScore, it.Memory.ID)
			}
		}
	})

	t.Run("sort_by_final_rank_score_desc", func(t *testing.T) {
		out := RankMemoryCandidates(input.CurrentMessage, all, input, map[uint64]float64{1: 0.9, 2: 0.5}, true)
		if len(out) < 2 {
			t.Fatalf("expected >= 2 candidates, got %d", len(out))
		}
		for i := 1; i < len(out); i++ {
			if out[i-1].Signals.FinalRankScore < out[i].Signals.FinalRankScore {
				t.Fatalf("results not sorted by final_rank_score desc: %+v", out)
			}
		}
	})
}

func TestMemoryImportanceGatedByRelevance_RankMemoryCandidates(t *testing.T) {
	mem := model.AIMemory{
		ID:         1,
		ScopeType:  "application",
		ScopeID:    99,
		Content:    "completely unrelated content",
		Confidence: 1.0,
		Importance: 1.0,
	}
	input := AgentContextInput{
		HrID:           7,
		JobID:          88,
		ApplicationID:  99,
		CurrentMessage: "does this candidate have enough Go experience",
	}

	t.Run("low_relevance_gates_boost", func(t *testing.T) {
		// "completely unrelated content" 与 query 的 token 命中几乎为 0
		// vector=0, lexical≈0, metadata=1.0 → relevance = 0.1*1.0 = 0.1 < 0.15
		s := scoreMemoryRankingSignals(mem, input, 0, false, 7)
		if s.RelevanceScore >= rankingConfig.RelevanceGate {
			t.Fatalf("expected relevance_score < %v, got %v", rankingConfig.RelevanceGate, s.RelevanceScore)
		}
		if s.BusinessBoost != 1.0 {
			t.Fatalf("expected business_boost=1.0 (gated), got %v", s.BusinessBoost)
		}
	})

	t.Run("high_relevance_uses_boost", func(t *testing.T) {
		// mem2 with vector=0.9, lexical hits Go token, metadata=0.7 (job match)
		// relevance ≈ 0.6*0.9 + 0.3*0.29 + 0.1*0.7 ≈ 0.72 > 0.15
		// boost = 1.0 + 0.2*1.0 + 0.1*1.0 = 1.3
		mem2 := model.AIMemory{ID: 2, ScopeType: "job", ScopeID: 88, Content: "candidate lacks Go production experience", Confidence: 1.0, Importance: 1.0}
		s := scoreMemoryRankingSignals(mem2, input, 0.9, true, 7)
		if s.RelevanceScore < rankingConfig.RelevanceGate {
			t.Fatalf("expected relevance_score >= %v, got %v", rankingConfig.RelevanceGate, s.RelevanceScore)
		}
		if s.BusinessBoost <= 1.0 {
			t.Fatalf("expected business_boost > 1.0, got %v", s.BusinessBoost)
		}
	})
}

func TestMemoryPoolFallbackToLexicalMetadata(t *testing.T) {
	mem := model.AIMemory{ID: 1, ScopeType: "application", ScopeID: 99, Content: "candidate lacks Go production experience", Confidence: 0.9, Importance: 1.0}
	input := AgentContextInput{
		HrID:           7,
		JobID:          88,
		ApplicationID:  99,
		CurrentMessage: "candidate Go experience",
	}
	out := RankMemoryCandidates(input.CurrentMessage, []model.AIMemory{mem}, input, nil, false)
	if len(out) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out))
	}
	got := out[0]
	if got.Signals.RelevanceMode != "lexical_metadata" {
		t.Fatalf("expected lexical_metadata, got %q", got.Signals.RelevanceMode)
	}
	if got.Signals.VectorScore != 0 {
		t.Fatalf("vector_score should be 0 in fallback, got %v", got.Signals.VectorScore)
	}
	if got.Signals.FinalRankScore <= 0 || got.Signals.FinalRankScore > rankingConfig.BusinessBoostMax {
		t.Fatalf("final_rank_score out of range: %v (max %v)", got.Signals.FinalRankScore, rankingConfig.BusinessBoostMax)
	}
	if got.Signals.BusinessBoost < 1.0 || got.Signals.BusinessBoost > rankingConfig.BusinessBoostMax {
		t.Fatalf("business_boost out of range: %v", got.Signals.BusinessBoost)
	}
	if !strings.Contains(got.Memory.Content, "Go") {
		t.Fatalf("memory content lost: %q", got.Memory.Content)
	}
}

// ------------------------------------------------------------------
// 池置信度：Skill / Memory from rankings
// ------------------------------------------------------------------

func TestSkillPoolConfidenceFromRankings(t *testing.T) {
	t.Run("high_gap_three_candidates", func(t *testing.T) {
		signals := []RankingSignals{
			{FinalRankScore: 0.9},
			{FinalRankScore: 0.5},
			{FinalRankScore: 0.3},
		}
		if got := ComputePoolConfidence(signals); got != RankConfidenceHigh {
			t.Fatalf("gap=0.4 should be high, got %v", got)
		}
	})

	t.Run("low_gap_close_candidates", func(t *testing.T) {
		signals := []RankingSignals{
			{FinalRankScore: 0.4},
			{FinalRankScore: 0.39},
			{FinalRankScore: 0.38},
		}
		if got := ComputePoolConfidence(signals); got != RankConfidenceLow {
			t.Fatalf("gap=0.01 should be low, got %v", got)
		}
	})

	t.Run("none_for_empty", func(t *testing.T) {
		if got := ComputePoolConfidence(nil); got != RankConfidenceNone {
			t.Fatalf("empty should be none, got %v", got)
		}
	})
}

func TestMemoryPoolConfidenceFromRankings(t *testing.T) {
	t.Run("medium_gap_two_memories", func(t *testing.T) {
		items := []RankedMemoryItem{
			{Memory: model.AIMemory{ID: 1}, Signals: RankingSignals{FinalRankScore: 0.4}},
			{Memory: model.AIMemory{ID: 2}, Signals: RankingSignals{FinalRankScore: 0.36}},
		}
		signals := make([]RankingSignals, 0, len(items))
		for _, it := range items {
			signals = append(signals, it.Signals)
		}
		if got := ComputePoolConfidence(signals); got != RankConfidenceMedium {
			t.Fatalf("gap=0.04 should be medium, got %v", got)
		}
	})

	t.Run("all_zero_none", func(t *testing.T) {
		items := []RankedMemoryItem{
			{Memory: model.AIMemory{ID: 1}, Signals: RankingSignals{FinalRankScore: 0}},
			{Memory: model.AIMemory{ID: 2}, Signals: RankingSignals{FinalRankScore: 0}},
		}
		signals := make([]RankingSignals, 0, len(items))
		for _, it := range items {
			signals = append(signals, it.Signals)
		}
		if got := ComputePoolConfidence(signals); got != RankConfidenceNone {
			t.Fatalf("all zero should be none, got %v", got)
		}
	})
}

// ------------------------------------------------------------------
// TASK-008 补充：分池隔离、归一化、向量降级
// ------------------------------------------------------------------

// TestPoolIsolation 验证 Skill 池与 Memory 池的排序 / 信号独立，互不污染。
// 同一个 query 同时跑 Skill 与 Memory 排序；两侧的 RankingSignals 不能交叉影响。
func TestPoolIsolation(t *testing.T) {
	skills := []repository.AgentSkillRuntimeRecord{
		{ID: 1, Name: "screening", DisplayName: "Screening", Description: "candidate match", BodyMarkdown: "candidate match screening", Category: "sourcing", Priority: 10, IsManualInvocable: 1},
		{ID: 2, Name: "offer", DisplayName: "Offer", Description: "candidate offer", BodyMarkdown: "candidate offer preparation", Category: "closing", Priority: 30, IsManualInvocable: 1},
	}
	memories := []model.AIMemory{
		{ID: 100, ScopeType: "application", ScopeID: 99, Content: "candidate lacks Go production experience", Confidence: 0.9, Importance: 1.0},
		{ID: 200, ScopeType: "hr", ScopeID: 0, Content: "HR prefers concise summaries", Confidence: 0.5, Importance: 0.5},
	}
	input := AgentContextInput{
		HrID:           7,
		JobID:          88,
		ApplicationID:  99,
		CurrentMessage: "candidate",
	}

	skillOut := RankSkillCandidates("candidate", skills, map[int64]float64{1: 0.8, 2: 0.6}, true)
	memSignals := RankMemoryCandidates(input.CurrentMessage, memories, input, map[uint64]float64{100: 0.9}, true)

	// Skill 池应当只包含 2 个 skill IDs
	if len(skillOut) != 2 {
		t.Fatalf("skill pool should have 2 candidates, got %d", len(skillOut))
	}
	for _, sk := range skillOut {
		if sk.ID < 0 || sk.ID > 100 {
			t.Fatalf("skill pool contaminated: %+v", sk)
		}
	}
	// Memory 池应当只包含 2 个 memory IDs（100 / 200）
	if len(memSignals) != 2 {
		t.Fatalf("memory pool should have 2 candidates, got %d", len(memSignals))
	}
	for _, m := range memSignals {
		if m.Memory.ID != 100 && m.Memory.ID != 200 {
			t.Fatalf("memory pool contaminated: %+v", m.Memory)
		}
	}
	// 互相不污染：Skill 池的 pool_confidence 应当只反映 skill 的 gap；Memory 池同样独立
	skillConfs := make([]RankingSignals, 0, len(skillOut))
	for _, sk := range skillOut {
		skillConfs = append(skillConfs, RankingSignals{FinalRankScore: sk.FinalRankScore})
	}
	memConfs := make([]RankingSignals, 0, len(memSignals))
	for _, m := range memSignals {
		memConfs = append(memConfs, m.Signals)
	}
	// Skill 与 Memory 应当有独立的 confidence 推断（即使分数巧合相等也不应该"合并"）
	// 这里只检查两侧都不是 none 即可（如果各自有信号）
	if ComputePoolConfidence(skillConfs) == RankConfidenceNone {
		t.Fatalf("skill pool confidence should not be none when both have positive score")
	}
	if ComputePoolConfidence(memConfs) == RankConfidenceNone {
		t.Fatalf("memory pool confidence should not be none when both have positive score")
	}
}

// TestVectorScoreFallbackToZero 验证 embedding 不可用 / NaN / Inf / 越界时 vector_score 全部归零。
func TestVectorScoreFallbackToZero(t *testing.T) {
	skill := repository.AgentSkillRuntimeRecord{
		ID:           1,
		Name:         "screening",
		DisplayName:  "Screening",
		Description:  "candidate screening",
		BodyMarkdown: "review candidate",
		Category:     "sourcing",
		Priority:     10,
	}
	t.Run("embedding_unavailable", func(t *testing.T) {
		s := scoreSkillRankingSignals("candidate", skill, 0.9, false)
		if s.VectorScore != 0 {
			t.Fatalf("vector_score should be 0 when embedding unavailable, got %v", s.VectorScore)
		}
		if s.RelevanceMode != "lexical_metadata" {
			t.Fatalf("mode should be lexical_metadata, got %q", s.RelevanceMode)
		}
	})
	t.Run("nan_vector", func(t *testing.T) {
		s := scoreSkillRankingSignals("candidate", skill, math.NaN(), true)
		if s.VectorScore != 0 {
			t.Fatalf("NaN vector should become 0, got %v", s.VectorScore)
		}
	})
	t.Run("inf_vector", func(t *testing.T) {
		s := scoreSkillRankingSignals("candidate", skill, math.Inf(1), true)
		if s.VectorScore != 0 {
			t.Fatalf("Inf vector should become 0, got %v", s.VectorScore)
		}
	})
	t.Run("over_one_vector", func(t *testing.T) {
		s := scoreSkillRankingSignals("candidate", skill, 1.7, true)
		if s.VectorScore > 1.0 {
			t.Fatalf("over-1 vector should clip to <= 1, got %v", s.VectorScore)
		}
	})
}

// TestLexicalScoreNormalization 验证 keyword / token 命中按 token 数归一化到 [0, 1]。
func TestLexicalScoreNormalization(t *testing.T) {
	mem := model.AIMemory{
		ID:         1,
		ScopeType:  "hr",
		ScopeID:    0,
		Content:    "candidate Go experience",
		Confidence: 0.9,
		Importance: 0.9,
	}
	input := AgentContextInput{
		CurrentMessage: "candidate Go",
	}
	t.Run("all_match_normalized_to_one", func(t *testing.T) {
		// query tokens: "candidate" / "go" (nTokens=2)；content 同时有 2 个 token → lexical=1.0
		s := scoreMemoryRankingSignals(mem, input, 0, false, 2)
		if s.LexicalScore != 1.0 {
			t.Fatalf("lexical_score should be 1.0 when all query tokens match, got %v", s.LexicalScore)
		}
	})
	t.Run("partial_match", func(t *testing.T) {
		// query tokens: "candidate" / "go" (nTokens=2)；content 仅有 "candidate" → lexical=0.5
		mem2 := model.AIMemory{ID: 2, ScopeType: "hr", ScopeID: 0, Content: "candidate only", Confidence: 0.9, Importance: 0.9}
		s := scoreMemoryRankingSignals(mem2, input, 0, false, 2)
		if s.LexicalScore < 0.4 || s.LexicalScore > 0.6 {
			t.Fatalf("lexical_score should be ~0.5, got %v", s.LexicalScore)
		}
	})
}

// TestMetadataScoreNormalization 验证 Memory scope 命中映射到 [0, 1] 且 Memory 池无关 Skill。
func TestMetadataScoreNormalization(t *testing.T) {
	mem := model.AIMemory{ID: 1, ScopeType: "hr", ScopeID: 0, Content: "x", Confidence: 0.5, Importance: 0.5}
	input := AgentContextInput{HrID: 1, JobID: 88, ApplicationID: 99, CurrentMessage: "x"}

	t.Run("application_match", func(t *testing.T) {
		m := model.AIMemory{ID: 1, ScopeType: "application", ScopeID: 99, Content: "x"}
		if got := normalizeMemoryMetadataScore(m, input); got != 1.0 {
			t.Fatalf("application match should be 1.0, got %v", got)
		}
	})
	t.Run("job_match", func(t *testing.T) {
		m := model.AIMemory{ID: 1, ScopeType: "job", ScopeID: 88, Content: "x"}
		if got := normalizeMemoryMetadataScore(m, input); got != 0.7 {
			t.Fatalf("job match should be 0.7, got %v", got)
		}
	})
	t.Run("hr_match", func(t *testing.T) {
		m := model.AIMemory{ID: 1, ScopeType: "hr", ScopeID: 0, Content: "x"}
		if got := normalizeMemoryMetadataScore(m, input); got != 0.4 {
			t.Fatalf("hr match should be 0.4, got %v", got)
		}
	})
	t.Run("hr_alone", func(t *testing.T) {
		// 即使 input 没有 JobID / ApplicationID，hr 作用域仍然参与（始终 0.4）
		if got := normalizeMemoryMetadataScore(mem, AgentContextInput{}); got != 0.4 {
			t.Fatalf("hr alone should be 0.4, got %v", got)
		}
	})
}

// TestBoostBoundaryScenarios 覆盖 business_boost 在 0 / 1.0 / 1.5 边界的稳定性。
func TestBoostBoundaryScenarios(t *testing.T) {
	t.Run("zero_priority_zero_boost_signals", func(t *testing.T) {
		got := computeBusinessBoost(0, 0, 0, true)
		if got != 1.0 {
			t.Fatalf("expected 1.0, got %v", got)
		}
	})
	t.Run("max_priority_skill_clamps_to_1_2", func(t *testing.T) {
		// priority_norm = clamp01(100/50) = 1.0; boost = 1 + 0.2 = 1.2
		got := computeBusinessBoost(100, 0, 0, true)
		if got > 1.5 {
			t.Fatalf("skill boost should be <= 1.5, got %v", got)
		}
	})
	t.Run("memory_max_signals_clamp_to_1_3", func(t *testing.T) {
		got := computeBusinessBoost(0, 1, 1, false)
		if got != 1.3 {
			t.Fatalf("memory max signals should be 1.3, got %v", got)
		}
	})
}

// ------------------------------------------------------------------
// TASK-FU-004 验证：config-driven 权重 / 阈值
// ------------------------------------------------------------------

func TestLoadRankingConfigDefaults(t *testing.T) {
	// nil-config 不修改 var
	t.Run("empty_config_keeps_hardcode", func(t *testing.T) {
		ResetRankingConfigForTest()
		LoadRankingConfig(RankingConfig{}) // 全部 0 / 走默认
		if rankingConfig.WeightVector != rankingDefaults.WeightVector {
			t.Fatalf("rankingConfig.WeightVector = %v, want %v", rankingConfig.WeightVector, rankingDefaults.WeightVector)
		}
		if rankingConfig.BusinessBoostMax != rankingDefaults.BusinessBoostMax {
			t.Fatalf("rankingConfig.BusinessBoostMax = %v, want %v", rankingConfig.BusinessBoostMax, rankingDefaults.BusinessBoostMax)
		}
		if rankingConfig.RelevanceGate != rankingDefaults.RelevanceGate {
			t.Fatalf("rankingConfig.RelevanceGate = %v, want %v", rankingConfig.RelevanceGate, rankingDefaults.RelevanceGate)
		}
	})
}

func TestLoadRankingConfigOverride(t *testing.T) {
	t.Run("weight_vector_override", func(t *testing.T) {
		ResetRankingConfigForTest()
		LoadRankingConfig(RankingConfig{WeightVector: 0.7})
		if rankingConfig.WeightVector != 0.7 {
			t.Fatalf("rankingConfig.WeightVector = %v, want 0.7", rankingConfig.WeightVector)
		}
		// 其他 var 不变
		if rankingConfig.WeightLexical != rankingDefaults.WeightLexical {
			t.Fatalf("rankingConfig.WeightLexical should be unchanged")
		}
	})

	t.Run("multiple_overrides", func(t *testing.T) {
		ResetRankingConfigForTest()
		LoadRankingConfig(RankingConfig{
			WeightVector:     0.5,
			WeightLexical:    0.4,
			WeightMetadata:   0.1,
			BusinessBoostMax: 1.2,
		})
		if rankingConfig.WeightVector != 0.5 {
			t.Fatalf("rankingConfig.WeightVector = %v, want 0.5", rankingConfig.WeightVector)
		}
		if rankingConfig.WeightLexical != 0.4 {
			t.Fatalf("rankingConfig.WeightLexical = %v, want 0.4", rankingConfig.WeightLexical)
		}
		if rankingConfig.WeightMetadata != 0.1 {
			t.Fatalf("rankingConfig.WeightMetadata = %v, want 0.1", rankingConfig.WeightMetadata)
		}
		if rankingConfig.BusinessBoostMax != 1.2 {
			t.Fatalf("rankingConfig.BusinessBoostMax = %v, want 1.2", rankingConfig.BusinessBoostMax)
		}
	})
}

func TestLoadRankingConfigClamp(t *testing.T) {
	t.Run("clamp_weight_to_unit_interval", func(t *testing.T) {
		ResetRankingConfigForTest()
		LoadRankingConfig(RankingConfig{
			WeightVector:  1.5,  // 越界 → clamp 到 1
			WeightLexical: -0.5, // 越界 → clamp 到 0
		})
		if rankingConfig.WeightVector != 1.0 {
			t.Fatalf("over-1 should clamp to 1, got %v", rankingConfig.WeightVector)
		}
		if rankingConfig.WeightLexical != 0.0 {
			t.Fatalf("negative should clamp to 0, got %v", rankingConfig.WeightLexical)
		}
	})

	t.Run("clamp_business_boost_max", func(t *testing.T) {
		ResetRankingConfigForTest()
		LoadRankingConfig(RankingConfig{BusinessBoostMax: 2.5}) // 越界 → clamp 到 1.5
		if rankingConfig.BusinessBoostMax != rankingDefaults.BusinessBoostMax {
			t.Fatalf("over-max should clamp to default, got %v", rankingConfig.BusinessBoostMax)
		}
		LoadRankingConfig(RankingConfig{BusinessBoostMax: 0.5}) // 越界 → clamp 到 1.0
		if rankingConfig.BusinessBoostMax != 1.0 {
			t.Fatalf("under-min should clamp to 1.0, got %v", rankingConfig.BusinessBoostMax)
		}
	})
}

func TestResetRankingConfigForTest(t *testing.T) {
	t.Run("reset_restores_all_11_vars", func(t *testing.T) {
		// 故意污染
		rankingConfig.WeightVector = 0.1
		rankingConfig.BusinessBoostMax = 1.0
		rankingConfig.RelevanceGate = 0.5
		// reset
		ResetRankingConfigForTest()
		// 验证 11 个 var 都恢复到 hardcode
		if rankingConfig.WeightVector != rankingDefaults.WeightVector {
			t.Fatalf("rankingConfig.WeightVector not reset: %v", rankingConfig.WeightVector)
		}
		if rankingConfig.WeightLexical != rankingDefaults.WeightLexical {
			t.Fatalf("rankingConfig.WeightLexical not reset: %v", rankingConfig.WeightLexical)
		}
		if rankingConfig.WeightMetadata != rankingDefaults.WeightMetadata {
			t.Fatalf("rankingConfig.WeightMetadata not reset: %v", rankingConfig.WeightMetadata)
		}
		if rankingConfig.BusinessBoostMax != rankingDefaults.BusinessBoostMax {
			t.Fatalf("rankingConfig.BusinessBoostMax not reset: %v", rankingConfig.BusinessBoostMax)
		}
		if rankingConfig.PriorityNorm != rankingDefaults.PriorityNorm {
			t.Fatalf("rankingConfig.PriorityNorm not reset: %v", rankingConfig.PriorityNorm)
		}
		if rankingConfig.BoostAlpha != rankingDefaults.BoostAlpha {
			t.Fatalf("rankingConfig.BoostAlpha not reset: %v", rankingConfig.BoostAlpha)
		}
		if rankingConfig.BoostBeta != rankingDefaults.BoostBeta {
			t.Fatalf("rankingConfig.BoostBeta not reset: %v", rankingConfig.BoostBeta)
		}
		if rankingConfig.BoostGamma != rankingDefaults.BoostGamma {
			t.Fatalf("rankingConfig.BoostGamma not reset: %v", rankingConfig.BoostGamma)
		}
		if rankingConfig.RelevanceGate != rankingDefaults.RelevanceGate {
			t.Fatalf("rankingConfig.RelevanceGate not reset: %v", rankingConfig.RelevanceGate)
		}
		if rankingConfig.GapHigh != rankingDefaults.GapHigh {
			t.Fatalf("rankingConfig.GapHigh not reset: %v", rankingConfig.GapHigh)
		}
		if rankingConfig.GapMedium != rankingDefaults.GapMedium {
			t.Fatalf("rankingConfig.GapMedium not reset: %v", rankingConfig.GapMedium)
		}
	})
}

// TestRankingConfigIntegration 端到端验证：env / config 覆盖生效。
// 模拟 main.go 的转换路径：config.Ranking → service.RankingConfig → LoadRankingConfig。
func TestRankingConfigIntegration(t *testing.T) {
	t.Run("config_to_ranking_to_var", func(t *testing.T) {
		ResetRankingConfigForTest()
		// 模拟 config.Ranking 字段被 env 覆盖
		cfg := RankingConfig{
			WeightVector:   0.7,
			WeightLexical:  0.2,
			WeightMetadata: 0.1,
		}
		LoadRankingConfig(cfg)

		// 验证最终的相关性计算使用了新权重
		// relevance = 0.7*vec + 0.2*lex + 0.1*meta
		// 设 vec=1, lex=0, meta=0 → relevance = 0.7
		got := computeRelevanceScore(1.0, 0.0, 0.0)
		if got < 0.69 || got > 0.71 {
			t.Fatalf("relevance with overridden weights = %v, want ~0.7", got)
		}
	})
}

// ------------------------------------------------------------------
// TASK-FU-005 验证：applyRankingField warn 日志
// ------------------------------------------------------------------

// captureRankingLogs 临时把 zap global logger 替换为 observable logger 并返回恢复函数 + 收集到的 entry。
// 使用 zap/zaptest/observer：标准 zap 测试模式。
func captureRankingLogs(t *testing.T) (cleanup func(), entries *observer.ObservedLogs) {
	t.Helper()
	core, recorded := observer.New(zapcore.DebugLevel)
	oldLogger := logger.L()
	logger.Set(zap.New(core))
	return func() { logger.Set(oldLogger) }, recorded
}

func TestApplyRankingFieldWarnBelowMin(t *testing.T) {
	cleanup, entries := captureRankingLogs(t)
	defer cleanup()

	var target float64 = 0.5
	applyRankingField(&target, -0.5, 0, 1, "weight_vector")

	if target != 0 {
		t.Fatalf("target should clamp to min=0, got %v", target)
	}
	// 必须有 1 条 warn 日志
	filter := entries.FilterMessageSnippet("[ranking] env value out of range, clamped")
	if filter.Len() != 1 {
		t.Fatalf("expected 1 warn log, got %d (all logs: %+v)", filter.Len(), entries.All())
	}
	// 验证日志字段
	entry := filter.All()[0]
	if entry.Level != zapcore.WarnLevel {
		t.Fatalf("expected WarnLevel, got %v", entry.Level)
	}
	ctx := entry.ContextMap()
	if ctx["key"] != "weight_vector" {
		t.Fatalf("expected key=weight_vector, got %v", ctx["key"])
	}
	if ctx["value"] != -0.5 {
		t.Fatalf("expected value=-0.5, got %v", ctx["value"])
	}
	if ctx["clamped"] != 0.0 {
		t.Fatalf("expected clamped=0, got %v", ctx["clamped"])
	}
}

func TestApplyRankingFieldWarnAboveMax(t *testing.T) {
	cleanup, entries := captureRankingLogs(t)
	defer cleanup()

	var target float64 = 0.5
	applyRankingField(&target, 2.5, 0, 1, "weight_vector")

	if target != 1 {
		t.Fatalf("target should clamp to max=1, got %v", target)
	}
	filter := entries.FilterMessageSnippet("[ranking] env value out of range, clamped")
	if filter.Len() != 1 {
		t.Fatalf("expected 1 warn log, got %d", filter.Len())
	}
	ctx := filter.All()[0].ContextMap()
	if ctx["value"] != 2.5 {
		t.Fatalf("expected value=2.5, got %v", ctx["value"])
	}
	if ctx["clamped"] != 1.0 {
		t.Fatalf("expected clamped=1, got %v", ctx["clamped"])
	}
}

func TestApplyRankingFieldNoWarnOnValidValue(t *testing.T) {
	cleanup, entries := captureRankingLogs(t)
	defer cleanup()

	var target float64 = 0.5
	applyRankingField(&target, 0.7, 0, 1, "weight_vector")

	if target != 0.7 {
		t.Fatalf("target should be 0.7, got %v", target)
	}
	filter := entries.FilterMessageSnippet("[ranking] env value out of range, clamped")
	if filter.Len() != 0 {
		t.Fatalf("expected 0 warn logs, got %d (all: %+v)", filter.Len(), entries.All())
	}
}

func TestApplyRankingFieldNoWarnOnZero(t *testing.T) {
	cleanup, entries := captureRankingLogs(t)
	defer cleanup()

	var target float64 = 0.5
	applyRankingField(&target, 0, 0, 1, "weight_vector")

	if target != 0.5 {
		t.Fatalf("target should be unchanged, got %v", target)
	}
	filter := entries.FilterMessageSnippet("[ranking] env value out of range, clamped")
	if filter.Len() != 0 {
		t.Fatalf("expected 0 warn logs for zero value, got %d", filter.Len())
	}
}
