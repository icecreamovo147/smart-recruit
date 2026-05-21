package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger

func init() {
	global, _ = zap.NewProduction()
}

// New creates a production logger (JSON format, info level and above).
func New(level string) *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	cfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	l, _ := cfg.Build()
	return l
}

// NewConsole creates a human-readable console logger for development.
func NewConsole() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	l, _ := cfg.Build()
	return l
}

// L returns the global logger.
func L() *zap.Logger { return global }

// Set replaces the global logger.
func Set(l *zap.Logger) { global = l }

// With creates a child logger with the given fields.
func With(fields ...zap.Field) *zap.Logger { return global.With(fields...) }
