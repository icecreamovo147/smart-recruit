package i18n

import (
	"errors"
	"testing"
)

func TestParseLocaleDefaultsToChinese(t *testing.T) {
	locale, err := ParseLocale("")
	if err != nil || locale != LocaleZhCN {
		t.Fatalf("ParseLocale = %q, %v", locale, err)
	}
}

func TestParseLocaleRejectsUnsupportedLocale(t *testing.T) {
	if _, err := ParseLocale("fr-FR"); err == nil {
		t.Fatal("expected unsupported locale error")
	}
}

func TestCatalogParity(t *testing.T) {
	if err := ValidateCatalogs(); err != nil {
		t.Fatal(err)
	}
}

func TestRenderAndUnknownFallbackStayInRequestedLocale(t *testing.T) {
	if got := Render(LocaleEnUS, "common.success", nil); got != "Success" {
		t.Fatalf("got %q", got)
	}
	if got := Render(LocaleZhCN, "missing.key", nil); got != "服务暂时不可用，请稍后重试" {
		t.Fatalf("got %q", got)
	}
}

func TestKeyErrorRetainsCauseAndKey(t *testing.T) {
	cause := errors.New("database unavailable")
	err := Wrap("common.operation_failed", cause)
	key, _, ok := ErrorKey(err)
	if !ok || key != "common.operation_failed" {
		t.Fatalf("key = %q, ok = %v", key, ok)
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected cause to be retained")
	}
}

func TestKeyForCode(t *testing.T) {
	tests := map[int32]string{
		0:   "common.success",
		400: "common.invalid_request",
		401: "common.unauthenticated",
		403: "common.forbidden",
		404: "common.not_found",
		500: "common.operation_failed",
	}
	for code, want := range tests {
		if got := KeyForCode(code); got != want {
			t.Errorf("KeyForCode(%d) = %q, want %q", code, got, want)
		}
	}
}
