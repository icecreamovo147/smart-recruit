package memory

import (
	logicconfig "smart-recruit-platform-go/serviceconfig"
	"time"

	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
)

type Config struct {
	Enabled          bool
	WriteEnabled     bool
	InjectEnabled    bool
	MaxMemories      int
	MaxMemoryChars   int
	Ranking          domainmemory.RankingConfig
	CleanupInterval  time.Duration
	CleanupTimeout   time.Duration
	RevokedRetention time.Duration
}

func ConfigFromService(cfg logicconfig.Config) Config {
	return Config{
		Enabled:          boolOrDefault(cfg.Agent.Memory.Enabled, true),
		WriteEnabled:     boolOrDefault(cfg.Agent.Memory.WriteEnabled, true),
		InjectEnabled:    boolOrDefault(cfg.Agent.Memory.InjectEnabled, true),
		MaxMemories:      cfg.Agent.MaxMemories,
		MaxMemoryChars:   cfg.Agent.MaxMemoryChars,
		Ranking:          rankingConfigFromService(cfg.Ranking),
		CleanupInterval:  cfg.Agent.Memory.CleanupInterval.Duration,
		CleanupTimeout:   cfg.Agent.Memory.CleanupTimeout.Duration,
		RevokedRetention: cfg.Agent.Memory.RevokedRetention.Duration,
	}
}

func rankingConfigFromService(cfg logicconfig.Ranking) domainmemory.RankingConfig {
	out := domainmemory.DefaultRankingConfig()
	applyPositiveRankingFloat(&out.WeightVector, cfg.WeightVector)
	applyPositiveRankingFloat(&out.WeightLexical, cfg.WeightLexical)
	applyPositiveRankingFloat(&out.WeightMetadata, cfg.WeightMetadata)
	applyPositiveRankingFloat(&out.BusinessBoostMax, cfg.BusinessBoostMax)
	applyPositiveRankingFloat(&out.PriorityNorm, cfg.PriorityNorm)
	applyPositiveRankingFloat(&out.BoostAlpha, cfg.BoostAlpha)
	applyPositiveRankingFloat(&out.BoostBeta, cfg.BoostBeta)
	applyPositiveRankingFloat(&out.BoostGamma, cfg.BoostGamma)
	applyPositiveRankingFloat(&out.RelevanceGate, cfg.RelevanceGate)
	applyPositiveRankingFloat(&out.GapHigh, cfg.GapHigh)
	applyPositiveRankingFloat(&out.GapMedium, cfg.GapMedium)
	return out
}

func applyPositiveRankingFloat(target *float64, value float64) {
	if value > 0 {
		*target = value
	}
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
