package service

import (
	"strings"
	"testing"
)

func TestValidateAndCanonicalizeExtraHeaders(t *testing.T) {
	canonical, err := validateAndCanonicalizeExtraHeaders(`{"X-Custom-Auth":"secret-value","Accept":"application/json"}`)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !strings.Contains(canonical, `"X-Custom-Auth":"secret-value"`) {
		t.Fatalf("unexpected canonical: %s", canonical)
	}

	if _, err := validateAndCanonicalizeExtraHeaders(`["not","object"]`); err == nil {
		t.Fatal("expected array rejection")
	}
	if _, err := validateAndCanonicalizeExtraHeaders(`{"X-Key":123}`); err == nil {
		t.Fatal("expected non-string rejection")
	}
	if _, err := validateAndCanonicalizeExtraHeaders(`{"" :"value"}`); err == nil {
		t.Fatal("expected invalid header name rejection")
	}
}

func TestMaskExtraHeadersForResponseNeverReturnsRaw(t *testing.T) {
	masked := maskExtraHeadersForResponse(`{"Authorization":"super-secret-token"}`)
	if masked == "" || strings.Contains(masked, "super-secret-token") {
		t.Fatalf("expected masked value, got %q", masked)
	}
	if masked == `{"Authorization":"super-secret-token"}` {
		t.Fatal("raw secret leaked in masked response")
	}
}

func TestMaskExtraHeadersForResponseInvalidLegacy(t *testing.T) {
	masked := maskExtraHeadersForResponse(`not-json`)
	if masked != "" {
		t.Fatalf("expected empty masked legacy value, got %q", masked)
	}
}

func TestParseExtraHeadersForUseRejectsInvalidStored(t *testing.T) {
	if _, err := parseExtraHeadersForUse(`{bad`); err == nil {
		t.Fatal("expected invalid stored headers error")
	}
}
