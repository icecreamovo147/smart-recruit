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
		Ranking:          domainmemory.RankingConfigFromService(cfg.Ranking),
		CleanupInterval:  cfg.Agent.Memory.CleanupInterval.Duration,
		CleanupTimeout:   cfg.Agent.Memory.CleanupTimeout.Duration,
		RevokedRetention: cfg.Agent.Memory.RevokedRetention.Duration,
	}
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
