package logger

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// GormLogger adapts zap to GORM's logger interface.
type GormLogger struct {
	LogLevel                  gormLogger.LogLevel
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
}

// NewGormLogger creates a GORM logger wired to the global zap logger.
func NewGormLogger(cfg *GormLogConfig) *GormLogger {
	level := gormLogger.Warn
	if cfg.EnableSQLLog {
		level = gormLogger.Info
	}
	return &GormLogger{
		LogLevel:                  level,
		SlowThreshold:             time.Duration(cfg.SlowThresholdMS) * time.Millisecond,
		IgnoreRecordNotFoundError: cfg.IgnoreRecordNotFoundError,
	}
}

func (l *GormLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	nl := *l
	nl.LogLevel = level
	return &nl
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= gormLogger.Info {
		GetRequestLogger(ctx).Sugar().Infof(msg, data...)
	}
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= gormLogger.Warn {
		GetRequestLogger(ctx).Sugar().Warnf(msg, data...)
	}
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= gormLogger.Error {
		GetRequestLogger(ctx).Sugar().Errorf(msg, data...)
	}
}

// Trace is called by GORM for every SQL statement.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormLogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	log := GetRequestLogger(ctx)

	switch {
	case err != nil && !(errors.Is(err, gorm.ErrRecordNotFound) && l.IgnoreRecordNotFoundError):
		log.Error("sql",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
	case elapsed > l.SlowThreshold && l.LogLevel >= gormLogger.Warn:
		log.Warn("sql slow",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
		)
	case l.LogLevel >= gormLogger.Info:
		log.Debug("sql",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
		)
	}
}
