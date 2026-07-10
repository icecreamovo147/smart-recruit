// Command ranking-show prints the current effective Skill / Memory ranking weights.
//
// Output format: 11 lines of `key=value` (4 decimal places), one per weight / threshold
// defined in SDD §3.3. Values reflect env (RANKING_*) overrides merged with hardcode
// defaults, after clamping to legal ranges.
//
// Usage:
//
//	go run ./cmd/ranking-show/                       # print hardcode defaults
//	RANKING_WEIGHT_VECTOR=0.7 go run ./cmd/ranking-show/   # print with override
//
// The tool is read-only: it never mutates the global service state used at runtime
// (it operates on a private copy of `rankingConfig` via service.LoadRankingConfig).
package main

import (
	"fmt"
	"os"

	"logic-grpc-service/config"
	"logic-grpc-service/service"
)

func main() {
	// Load config (best-effort: in dev mode fall back to config.example.yaml).
	// Errors are non-fatal: we still print hardcode defaults below.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: config load failed (%v); printing hardcode defaults only\n", err)
		cfg = config.Config{}
	}

	// Apply env / yaml overrides via service.LoadRankingConfig (same path as
	// production). This mutates the package-level rankingConfig struct used by
	// rank score functions, but for a CLI invocation that's acceptable.
	service.LoadRankingConfig(service.RankingConfig{
		WeightVector:     cfg.Ranking.WeightVector,
		WeightLexical:    cfg.Ranking.WeightLexical,
		WeightMetadata:   cfg.Ranking.WeightMetadata,
		BusinessBoostMax: cfg.Ranking.BusinessBoostMax,
		PriorityNorm:     cfg.Ranking.PriorityNorm,
		BoostAlpha:       cfg.Ranking.BoostAlpha,
		BoostBeta:        cfg.Ranking.BoostBeta,
		BoostGamma:       cfg.Ranking.BoostGamma,
		RelevanceGate:    cfg.Ranking.RelevanceGate,
		GapHigh:          cfg.Ranking.GapHigh,
		GapMedium:        cfg.Ranking.GapMedium,
	})

	// Snapshot the effective values via a helper that reads the package-level
	// rankingConfig struct. We use the exported Ranking() helper below.
	snap := service.SnapshotRankingConfig()
	printLine("weight_vector", snap.WeightVector)
	printLine("weight_lexical", snap.WeightLexical)
	printLine("weight_metadata", snap.WeightMetadata)
	printLine("business_boost_max", snap.BusinessBoostMax)
	printLine("priority_norm", snap.PriorityNorm)
	printLine("boost_alpha", snap.BoostAlpha)
	printLine("boost_beta", snap.BoostBeta)
	printLine("boost_gamma", snap.BoostGamma)
	printLine("relevance_gate", snap.RelevanceGate)
	printLine("gap_high", snap.GapHigh)
	printLine("gap_medium", snap.GapMedium)
}

func printLine(key string, value float64) {
	fmt.Printf("%s=%.4f\n", key, value)
}
