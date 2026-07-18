package observability

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerConfig struct {
	ServiceName string
	Env         string
	Level       string
}

func NewLogger(cfg LoggerConfig) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()
	if isLocal(cfg.Env) {
		zapCfg = zap.NewDevelopmentConfig()
	}
	if strings.TrimSpace(cfg.Level) != "" {
		level := zap.NewAtomicLevel()
		if err := level.UnmarshalText([]byte(strings.TrimSpace(cfg.Level))); err != nil {
			return nil, err
		}
		zapCfg.Level = level
	}
	zapCfg.EncoderConfig.TimeKey = "ts"
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := zapCfg.Build(zap.Fields(zap.String("service", strings.TrimSpace(cfg.ServiceName))))
	if err != nil {
		return nil, err
	}
	return logger, nil
}

func RedactSecrets(fields map[string]string) map[string]string {
	redacted := make(map[string]string, len(fields))
	for key, value := range fields {
		if isSecretKey(key) {
			redacted[key] = "[REDACTED]"
			continue
		}
		redacted[key] = value
	}
	return redacted
}

func isSecretKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "credential")
}

func isLocal(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "", "local", "dev", "development", "test":
		return true
	default:
		return false
	}
}
