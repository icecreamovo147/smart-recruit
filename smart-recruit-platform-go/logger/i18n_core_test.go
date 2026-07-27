package logger

import (
	"bytes"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"smart-recruit-platform-go/i18n"
)

func TestI18nCoreLocalizesMessageAndKeepsCause(t *testing.T) {
	if err := i18n.Configure("en-US"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i18n.Configure("zh-CN") })

	var output bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := localizeCore(zapcore.NewCore(encoder, zapcore.AddSync(&output), zap.DebugLevel))
	log := zap.New(core)
	log.Error("log.gateway.internal_error", zap.String("cause", "database unavailable"))

	text := output.String()
	for _, expected := range []string{
		`"msg":"An internal request error occurred"`,
		`"event_key":"log.gateway.internal_error"`,
		`"locale":"en-US"`,
		`"cause":"database unavailable"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("log %q does not contain %q", text, expected)
		}
	}
}

func TestI18nCoreKeepsLegacyEventsInOneConfiguredLanguage(t *testing.T) {
	t.Cleanup(func() { _ = i18n.Configure("zh-CN") })
	tests := []struct {
		locale  string
		message string
		reject  string
	}{
		{locale: "zh-CN", message: "服务事件", reject: "Service event"},
		{locale: "en-US", message: "Service event", reject: "服务事件"},
	}
	for _, test := range tests {
		if err := i18n.Configure(test.locale); err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		log := zap.New(localizeCore(zapcore.NewCore(encoder, zapcore.AddSync(&output), zap.DebugLevel)))
		log.Info("legacy natural-language event")

		text := output.String()
		if !strings.Contains(text, `"msg":"`+test.message+`"`) ||
			!strings.Contains(text, `"locale":"`+test.locale+`"`) ||
			strings.Contains(text, test.reject) {
			t.Fatalf("%s log is not single-language: %s", test.locale, text)
		}
	}
}
