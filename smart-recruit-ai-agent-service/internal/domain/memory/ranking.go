package memory

import (
	"math"
	"strings"

	logicconfig "smart-recruit-platform-go/serviceconfig"
)

type RankingConfig struct {
	WeightVector     float64
	WeightLexical    float64
	WeightMetadata   float64
	BusinessBoostMax float64
	PriorityNorm     float64
	BoostAlpha       float64
	BoostBeta        float64
	BoostGamma       float64
	RelevanceGate    float64
	GapHigh          float64
	GapMedium        float64
}

type RankingSignals struct {
	VectorScore    float64
	LexicalScore   float64
	MetadataScore  float64
	RelevanceScore float64
	BusinessBoost  float64
	FinalRankScore float64
	RelevanceMode  string
}

type RecallQueryContext struct {
	Query              string
	TargetScopeType    string
	TargetScopeID      uint64
	QueryTokenCount    int
	EmbeddingAvailable bool
}

func DefaultRankingConfig() RankingConfig {
	return RankingConfig{
		WeightVector:     0.6,
		WeightLexical:    0.3,
		WeightMetadata:   0.1,
		BusinessBoostMax: 1.5,
		PriorityNorm:     50.0,
		BoostAlpha:       0.2,
		BoostBeta:        0.2,
		BoostGamma:       0.1,
		RelevanceGate:    0.15,
		GapHigh:          0.10,
		GapMedium:        0.03,
	}
}

func RankingConfigFromService(cfg logicconfig.Ranking) RankingConfig {
	defaults := DefaultRankingConfig()
	out := defaults
	applyRankingFloat(&out.WeightVector, cfg.WeightVector, defaults.WeightVector)
	applyRankingFloat(&out.WeightLexical, cfg.WeightLexical, defaults.WeightLexical)
	applyRankingFloat(&out.WeightMetadata, cfg.WeightMetadata, defaults.WeightMetadata)
	applyRankingFloat(&out.BusinessBoostMax, cfg.BusinessBoostMax, defaults.BusinessBoostMax)
	applyRankingFloat(&out.PriorityNorm, cfg.PriorityNorm, defaults.PriorityNorm)
	applyRankingFloat(&out.BoostAlpha, cfg.BoostAlpha, defaults.BoostAlpha)
	applyRankingFloat(&out.BoostBeta, cfg.BoostBeta, defaults.BoostBeta)
	applyRankingFloat(&out.BoostGamma, cfg.BoostGamma, defaults.BoostGamma)
	applyRankingFloat(&out.RelevanceGate, cfg.RelevanceGate, defaults.RelevanceGate)
	applyRankingFloat(&out.GapHigh, cfg.GapHigh, defaults.GapHigh)
	applyRankingFloat(&out.GapMedium, cfg.GapMedium, defaults.GapMedium)
	return out
}

func applyRankingFloat(target *float64, value, fallback float64) {
	if value > 0 {
		*target = value
	} else {
		*target = fallback
	}
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func ComputeRelevanceScore(cfg RankingConfig, vector, lexical, metadata float64) float64 {
	score := cfg.WeightVector*clamp01(vector) +
		cfg.WeightLexical*clamp01(lexical) +
		cfg.WeightMetadata*clamp01(metadata)
	return clamp01(score)
}

func memoryImportanceSignal(cfg RankingConfig, relevance, importance float64) float64 {
	if relevance < cfg.RelevanceGate {
		return 0
	}
	return clamp01(importance)
}

func memoryConfidenceSignal(cfg RankingConfig, relevance, confidence float64) float64 {
	if relevance < cfg.RelevanceGate {
		return 0
	}
	return clamp01(confidence)
}

func ComputeMemoryBusinessBoost(cfg RankingConfig, relevance, importance, confidence float64) float64 {
	importanceSignal := memoryImportanceSignal(cfg, relevance, importance)
	confidenceSignal := memoryConfidenceSignal(cfg, relevance, confidence)
	boost := 1.0 +
		cfg.BoostBeta*importanceSignal +
		cfg.BoostGamma*confidenceSignal
	if boost < 1.0 {
		return 1.0
	}
	if boost > cfg.BusinessBoostMax {
		return cfg.BusinessBoostMax
	}
	return boost
}

func ComputeFinalRankScore(relevance, boost float64) float64 {
	if relevance <= 0 || boost <= 0 {
		return 0
	}
	return relevance * boost
}

func ScopeMetadataScore(memory Scope, input RecallQueryContext) float64 {
	if input.TargetScopeType == "" {
		return 0.4
	}
	if memory.Type == input.TargetScopeType && memory.ID == input.TargetScopeID {
		return 1.0
	}
	switch input.TargetScopeType {
	case ScopeApplication:
		if memory.Type == ScopeJob {
			return 0.7
		}
		if memory.Type == ScopeHR || memory.Type == ScopeUser {
			return 0.4
		}
	case ScopeJob:
		if memory.Type == ScopeHR || memory.Type == ScopeUser {
			return 0.4
		}
	}
	return 0
}

func KeywordLexicalScore(query, content string, queryTokenCount int) float64 {
	query = strings.ToLower(strings.TrimSpace(query))
	content = strings.ToLower(strings.TrimSpace(content))
	if query == "" || content == "" {
		return 0
	}
	if strings.Contains(content, query) {
		return 1
	}
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return 0
	}
	hits := 0
	for _, token := range tokens {
		if strings.Contains(content, token) {
			hits++
		}
	}
	denom := queryTokenCount
	if denom <= 0 {
		denom = len(tokens)
	}
	score := float64(hits) / float64(denom)
	return clamp01(score)
}

func ScoreMemoryRankingSignals(cfg RankingConfig, memory Memory, input RecallQueryContext, vectorScore float64) RankingSignals {
	lexical := KeywordLexicalScore(input.Query, memory.Content, input.QueryTokenCount)
	metadata := ScopeMetadataScore(memory.Scope, input)
	relevance := ComputeRelevanceScore(cfg, vectorScore, lexical, metadata)
	boost := ComputeMemoryBusinessBoost(cfg, relevance, memory.Importance, memory.Confidence)
	final := ComputeFinalRankScore(relevance, boost)
	mode := "lexical_metadata"
	if input.EmbeddingAvailable && vectorScore > 0 {
		mode = "hybrid"
	} else if input.EmbeddingAvailable {
		mode = "lexical_metadata"
	} else {
		mode = "fallback"
	}
	return RankingSignals{
		VectorScore:    clamp01(vectorScore),
		LexicalScore:   lexical,
		MetadataScore:  metadata,
		RelevanceScore: relevance,
		BusinessBoost:  boost,
		FinalRankScore: final,
		RelevanceMode:  mode,
	}
}

type RankedMemory struct {
	Memory Memory
	Score  float64
	Reason string
	Signals RankingSignals
}

func RankMemories(cfg RankingConfig, memories []Memory, input RecallQueryContext, vectorScores map[uint64]float64) []RankedMemory {
	ranked := make([]RankedMemory, 0, len(memories))
	for _, memory := range memories {
		vector := 0.0
		if vectorScores != nil {
			vector = vectorScores[memory.ID]
		}
		signals := ScoreMemoryRankingSignals(cfg, memory, input, vector)
		if signals.FinalRankScore <= 0 {
			continue
		}
		ranked = append(ranked, RankedMemory{
			Memory:  memory,
			Score:   signals.FinalRankScore,
			Reason:  signals.RelevanceMode,
			Signals: signals,
		})
	}
	sortRankedMemories(ranked)
	return ranked
}

func sortRankedMemories(items []RankedMemory) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			a, b := items[i], items[j]
			swap := false
			switch {
			case b.Score > a.Score:
				swap = true
			case b.Score == a.Score && b.Memory.Importance > a.Memory.Importance:
				swap = true
			case b.Score == a.Score && b.Memory.Importance == a.Memory.Importance && b.Memory.ID < a.Memory.ID:
				swap = true
			}
			if swap {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func TruncateRankedMemories(items []RankedMemory, maxMemories, maxChars int) []RankedMemory {
	if maxMemories <= 0 {
		maxMemories = 10
	}
	if maxChars <= 0 {
		maxChars = 1500
	}
	out := make([]RankedMemory, 0, len(items))
	usedChars := 0
	for _, item := range items {
		if len(out) >= maxMemories {
			break
		}
		contentLen := len([]rune(strings.TrimSpace(item.Memory.Content)))
		if contentLen == 0 {
			continue
		}
		if usedChars+contentLen > maxChars && len(out) > 0 {
			break
		}
		out = append(out, item)
		usedChars += contentLen
	}
	return out
}
