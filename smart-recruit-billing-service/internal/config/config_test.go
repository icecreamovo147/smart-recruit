package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadUsesRelativeKeyFilesAndDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "private.pem"), []byte("private-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "public.pem"), []byte("public-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	data := `
alipay:
  required: true
  app_id: sandbox-app
  seller_id: sandbox-seller
  private_key_file: private.pem
  public_key_file: public.pem
  notify_url: https://example.test/api/v1/public/billing/webhooks/alipay
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Billing.Mode != "shadow" || cfg.Alipay.Environment != "sandbox" || !cfg.Alipay.DesktopEnabled || !cfg.Alipay.WAPEnabled {
		t.Fatalf("defaults not applied: %+v", cfg)
	}
	if cfg.Alipay.PrivateKey != "private-value" || cfg.Alipay.PublicKey != "public-value" {
		t.Fatalf("key files not loaded: private=%q public=%q", cfg.Alipay.PrivateKey, cfg.Alipay.PublicKey)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("alipay:\n  appid: typo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "field appid not found") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRejectsInlineAndFileKeyTogether(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := "alipay:\n  private_key: inline\n  private_key_file: private.pem\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "cannot both be configured") {
		t.Fatalf("Load() error = %v", err)
	}
}
