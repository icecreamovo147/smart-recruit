package logger

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var global *zap.Logger

func init() {
	global = New("info")
}

// Init configures the global logger with dual (console + file) output via zap tee cores.
func Init(cfg LogConfig) error {
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
	}

	var cores []zapcore.Core

	if cfg.EnableConsole {
		cores = append(cores, zapcore.NewCore(
			newConsoleEncoder(cfg.Format),
			zapcore.AddSync(os.Stdout),
			level,
		))
	}

	if cfg.EnableFile {
		lumberjackWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		}
		cores = append(cores, zapcore.NewCore(
			newFileEncoder(),
			zapcore.AddSync(lumberjackWriter),
			level,
		))
	}

	if len(cores) == 0 {
		cores = append(cores, zapcore.NewCore(
			newConsoleEncoder(cfg.Format),
			zapcore.AddSync(os.Stdout),
			level,
		))
	}

	core := localizeCore(zapcore.NewTee(cores...))
	global = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return nil
}

func newConsoleEncoder(format string) zapcore.Encoder {
	cfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	if format == "json" {
		cfg.EncodeLevel = zapcore.LowercaseLevelEncoder
		return zapcore.NewJSONEncoder(cfg)
	}
	return zapcore.NewConsoleEncoder(cfg)
}

func newFileEncoder() zapcore.Encoder {
	cfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	return zapcore.NewJSONEncoder(cfg)
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
	l, _ := cfg.Build(zap.WrapCore(localizeCore))
	return l
}

// NewConsole creates a human-readable console logger for development.
func NewConsole() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	l, _ := cfg.Build(zap.WrapCore(localizeCore))
	return l
}

// L returns the global logger.
func L() *zap.Logger { return global }

// Set replaces the global logger.
func Set(l *zap.Logger) { global = l }

// With creates a child logger with the given fields.
func With(fields ...zap.Field) *zap.Logger { return global.With(fields...) }

// ── Request-scoped logger helpers ──────────────────────────────────────

type ctxKey string

const ctxKeyRequestLogger ctxKey = "request-logger"

// WithRequestLogger stores a child logger with request-scoped fields in the context.
func WithRequestLogger(ctx context.Context, fields ...zap.Field) context.Context {
	return context.WithValue(ctx, ctxKeyRequestLogger, global.With(fields...))
}

// GetRequestLogger retrieves the request-scoped logger from context, falling back to global.
func GetRequestLogger(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKeyRequestLogger).(*zap.Logger); ok {
		return l
	}
	return global
}
