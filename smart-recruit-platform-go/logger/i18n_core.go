package logger

import (
	"crypto/sha256"
	"encoding/hex"

	"go.uber.org/zap/zapcore"

	"smart-recruit-platform-go/i18n"
)

type i18nCore struct {
	zapcore.Core
}

func localizeCore(core zapcore.Core) zapcore.Core {
	return i18nCore{Core: core}
}

func (core i18nCore) With(fields []zapcore.Field) zapcore.Core {
	return i18nCore{Core: core.Core.With(fields)}
}

func (core i18nCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if core.Enabled(entry.Level) {
		return checked.AddCore(entry, core)
	}
	return checked
}

func (core i18nCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	eventKey := entry.Message
	if i18n.Has(eventKey) {
		entry.Message = i18n.T(eventKey)
	} else {
		eventKey = legacyEventKey(eventKey)
		entry.Message = i18n.T("log.service.event")
	}
	fields = normalizeCauseFields(fields)
	fields = append(fields,
		zapcore.Field{Key: "event_key", Type: zapcore.StringType, String: eventKey},
		zapcore.Field{Key: "locale", Type: zapcore.StringType, String: string(i18n.Current())},
	)
	return core.Core.Write(entry, fields)
}

func legacyEventKey(message string) string {
	sum := sha256.Sum256([]byte(message))
	return "legacy." + hex.EncodeToString(sum[:6])
}

func normalizeCauseFields(fields []zapcore.Field) []zapcore.Field {
	for index := range fields {
		if fields[index].Key == "error" || fields[index].Key == "error_message" {
			fields[index].Key = "cause"
		}
	}
	return fields
}
