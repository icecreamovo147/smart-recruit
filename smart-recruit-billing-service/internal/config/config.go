package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"smart-recruit-billing-service/internal/infrastructure/payment"
)

type Config struct {
	Billing BillingConfig `yaml:"billing"`
	Alipay  AlipayConfig  `yaml:"alipay"`
}

type BillingConfig struct {
	Mode string `yaml:"mode"`
}

type AlipayConfig struct {
	Environment         string `yaml:"environment"`
	Required            bool   `yaml:"required"`
	GatewayURL          string `yaml:"gateway_url"`
	AppID               string `yaml:"app_id"`
	SellerID            string `yaml:"seller_id"`
	PrivateKey          string `yaml:"private_key"`
	PrivateKeyFile      string `yaml:"private_key_file"`
	VerifyPublicKey     string `yaml:"verify_public_key"`
	VerifyPublicKeyFile string `yaml:"verify_public_key_file"`
	NotifyURL           string `yaml:"notify_url"`
	ReturnURL           string `yaml:"return_url"`
	DesktopEnabled      bool   `yaml:"desktop_enabled"`
	WAPEnabled          bool   `yaml:"wap_enabled"`
}

func Load(path string) (Config, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Config{}, errors.New("billing config path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read billing config %q: %w", path, err)
	}
	cfg := Config{
		Billing: BillingConfig{Mode: "shadow"},
		Alipay: AlipayConfig{
			Environment: "sandbox", GatewayURL: payment.DefaultSandboxGateway,
			DesktopEnabled: true, WAPEnabled: true,
		},
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode billing config %q: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Config{}, errors.New("billing config must contain exactly one YAML document")
		}
		return Config{}, fmt.Errorf("decode billing config %q: %w", path, err)
	}
	cfg.Billing.Mode = strings.TrimSpace(cfg.Billing.Mode)
	// Runtime environment is the deployment control plane. A checked-in or
	// machine-local YAML file may provide a safe default, but an explicit mode
	// must override it so Billing and AI Agent cannot silently run in different
	// enforcement modes.
	if mode := strings.TrimSpace(os.Getenv("AI_BILLING_MODE")); mode != "" {
		cfg.Billing.Mode = mode
	}
	if cfg.Billing.Mode != "shadow" && cfg.Billing.Mode != "enforce" {
		return Config{}, errors.New("billing.mode must be shadow or enforce")
	}
	baseDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return Config{}, fmt.Errorf("resolve billing config directory: %w", err)
	}
	if cfg.Alipay.PrivateKey, err = loadSecret(baseDir, "alipay.private_key", cfg.Alipay.PrivateKey, cfg.Alipay.PrivateKeyFile); err != nil {
		return Config{}, err
	}
	if cfg.Alipay.VerifyPublicKey, err = loadSecret(baseDir, "alipay.verify_public_key", cfg.Alipay.VerifyPublicKey, cfg.Alipay.VerifyPublicKeyFile); err != nil {
		return Config{}, err
	}
	cfg.Alipay.Environment = strings.TrimSpace(cfg.Alipay.Environment)
	cfg.Alipay.GatewayURL = strings.TrimSpace(cfg.Alipay.GatewayURL)
	cfg.Alipay.AppID = strings.TrimSpace(cfg.Alipay.AppID)
	cfg.Alipay.SellerID = strings.TrimSpace(cfg.Alipay.SellerID)
	cfg.Alipay.NotifyURL = strings.TrimSpace(cfg.Alipay.NotifyURL)
	cfg.Alipay.ReturnURL = strings.TrimSpace(cfg.Alipay.ReturnURL)
	return cfg, nil
}

func (c Config) PaymentConfig() payment.AlipayConfig {
	return payment.AlipayConfig{
		Environment: c.Alipay.Environment, GatewayURL: c.Alipay.GatewayURL,
		AppID: c.Alipay.AppID, PrivateKey: c.Alipay.PrivateKey, VerifyPublicKey: c.Alipay.VerifyPublicKey,
		SellerID: c.Alipay.SellerID, NotifyURL: c.Alipay.NotifyURL, ReturnURL: c.Alipay.ReturnURL,
		DesktopEnabled: c.Alipay.DesktopEnabled, WAPEnabled: c.Alipay.WAPEnabled,
	}
}

func loadSecret(baseDir, field, inlineValue, fileName string) (string, error) {
	inlineValue = strings.TrimSpace(inlineValue)
	fileName = strings.TrimSpace(fileName)
	if inlineValue != "" && fileName != "" {
		return "", fmt.Errorf("%s and %s_file cannot both be configured", field, field)
	}
	if fileName == "" {
		return inlineValue, nil
	}
	if !filepath.IsAbs(fileName) {
		fileName = filepath.Join(baseDir, fileName)
	}
	data, err := os.ReadFile(filepath.Clean(fileName))
	if err != nil {
		return "", fmt.Errorf("read %s file %q: %w", field, fileName, err)
	}
	if field == "alipay.private_key" {
		info, statErr := os.Stat(filepath.Clean(fileName))
		if statErr != nil {
			return "", fmt.Errorf("stat %s file %q: %w", field, fileName, statErr)
		}
		if info.Mode().Perm()&0o077 != 0 {
			return "", fmt.Errorf("%s file %q must not be group/world accessible (use mode 0600)", field, fileName)
		}
	}
	return strings.TrimSpace(string(data)), nil
}
