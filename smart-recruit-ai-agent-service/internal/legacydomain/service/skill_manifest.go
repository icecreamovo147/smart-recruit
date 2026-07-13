package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
)

var skillNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,127}$`)

type SkillManifest struct {
	Name         string               `json:"name"`
	DisplayName  string               `json:"display_name"`
	Description  string               `json:"description"`
	Version      string               `json:"version"`
	Instruction  string               `json:"instruction"`
	InputSchema  json.RawMessage      `json:"input_schema"`
	OutputSchema json.RawMessage      `json:"output_schema"`
	Runtime      SkillRuntimeManifest `json:"runtime"`
	Tools        []SkillToolManifest  `json:"tools"`
}

type SkillRuntimeManifest struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

type SkillToolManifest struct {
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	InputSchema   json.RawMessage      `json:"input_schema"`
	Runtime       SkillRuntimeManifest `json:"runtime"`
	RuntimeConfig json.RawMessage      `json:"runtime_config"`
}

type SkillHTTPRuntimeConfig struct {
	URL            string            `json:"url"`
	Method         string            `json:"method"`
	Headers        map[string]string `json:"headers"`
	TimeoutSeconds int               `json:"timeout_seconds"`
}

func ParseAndValidateSkillManifest(raw string) (*SkillManifest, error) {
	var manifest SkillManifest
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		return nil, fmt.Errorf("manifest_json is invalid JSON: %w", err)
	}
	if err := ValidateSkillManifest(&manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func ValidateSkillManifest(m *SkillManifest) error {
	if m == nil {
		return fmt.Errorf("manifest is required")
	}
	m.Name = strings.TrimSpace(m.Name)
	m.Version = strings.TrimSpace(m.Version)
	m.DisplayName = strings.TrimSpace(m.DisplayName)
	m.Runtime.Type = normalizeSkillRuntime(m.Runtime.Type)
	if !skillNamePattern.MatchString(m.Name) {
		return fmt.Errorf("manifest.name must match %s", skillNamePattern.String())
	}
	if m.Version == "" {
		return fmt.Errorf("manifest.version is required")
	}
	if m.DisplayName == "" {
		m.DisplayName = m.Name
	}
	if m.Runtime.Type == "" {
		m.Runtime.Type = "prompt"
	}
	if !isSupportedSkillRuntime(m.Runtime.Type) {
		return fmt.Errorf("manifest.runtime.type %q is unsupported", m.Runtime.Type)
	}
	if err := validateOptionalJSONSchema("manifest.input_schema", m.InputSchema); err != nil {
		return err
	}
	if err := validateOptionalJSONSchema("manifest.output_schema", m.OutputSchema); err != nil {
		return err
	}
	if len(m.Tools) == 0 && m.Runtime.Type != "prompt" {
		return fmt.Errorf("manifest.tools is required for %s runtime", m.Runtime.Type)
	}
	seen := map[string]bool{}
	for i := range m.Tools {
		t := &m.Tools[i]
		t.Name = strings.TrimSpace(t.Name)
		t.Runtime.Type = normalizeSkillRuntime(t.Runtime.Type)
		if t.Runtime.Type == "" {
			t.Runtime.Type = normalizeSkillRuntime(m.Runtime.Type)
		}
		if !skillNamePattern.MatchString(t.Name) {
			return fmt.Errorf("tools[%d].name must match %s", i, skillNamePattern.String())
		}
		if seen[t.Name] {
			return fmt.Errorf("duplicate tool name %q", t.Name)
		}
		seen[t.Name] = true
		if !isSupportedSkillRuntime(t.Runtime.Type) {
			return fmt.Errorf("tools[%d].runtime.type %q is unsupported", i, t.Runtime.Type)
		}
		if err := validateOptionalJSONSchema(fmt.Sprintf("tools[%d].input_schema", i), t.InputSchema); err != nil {
			return err
		}
		if (t.Runtime.Type == "http" || t.Runtime.Type == "tool") && isEmptyJSON(t.InputSchema) {
			return fmt.Errorf("tools[%d].input_schema is required for %s runtime", i, t.Runtime.Type)
		}
		if t.Runtime.Type == "http" {
			runtimeConfig := t.RuntimeConfig
			if isEmptyJSON(runtimeConfig) {
				runtimeConfig = t.Runtime.Config
			}
			if err := validateHTTPRuntimeConfig(runtimeConfig); err != nil {
				return fmt.Errorf("tools[%d].runtime_config: %w", i, err)
			}
		}
	}
	return nil
}

func normalizeSkillRuntime(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func isSupportedSkillRuntime(v string) bool {
	switch v {
	case "prompt", "tool", "workflow", "http":
		return true
	default:
		return false
	}
}

func validateOptionalJSONSchema(field string, raw json.RawMessage) error {
	if isEmptyJSON(raw) {
		return nil
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return fmt.Errorf("%s must be a JSON object schema: %w", field, err)
	}
	if len(schema) == 0 {
		return fmt.Errorf("%s must not be empty", field)
	}
	typ, ok := schema["type"].(string)
	if !ok || typ != "object" {
		return fmt.Errorf("%s.type must be object", field)
	}
	properties := map[string]any{}
	if v, ok := schema["properties"]; ok {
		props, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%s.properties must be an object", field)
		}
		properties = props
	}
	if v, ok := schema["required"]; ok {
		required, ok := v.([]any)
		if !ok {
			return fmt.Errorf("%s.required must be an array of strings", field)
		}
		for _, item := range required {
			name, ok := item.(string)
			if !ok || name == "" {
				return fmt.Errorf("%s.required must be an array of strings", field)
			}
			if _, exists := properties[name]; !exists {
				return fmt.Errorf("%s.required field %q must exist in properties", field, name)
			}
		}
	}
	return nil
}

func validateHTTPRuntimeConfig(raw json.RawMessage) error {
	if isEmptyJSON(raw) {
		return fmt.Errorf("http runtime_config is required")
	}
	var cfg SkillHTTPRuntimeConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	cfg.Method = strings.ToUpper(strings.TrimSpace(cfg.Method))
	if cfg.Method == "" {
		cfg.Method = "POST"
	}
	if cfg.Method != "POST" {
		return fmt.Errorf("method must be POST")
	}
	if err := validateSkillHTTPURL(context.Background(), cfg.URL); err != nil {
		return err
	}
	return nil
}

func validateSkillHTTPRuntimeConfigString(raw string) error {
	var msg json.RawMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return validateHTTPRuntimeConfig(msg)
}

func validateSkillHTTPURL(ctx context.Context, rawURL string) error {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("url must be absolute")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return fmt.Errorf("url host is required")
	}
	if isLocalhostName(host) {
		return fmt.Errorf("url host is not allowed")
	}
	if _, err := resolvePublicSkillHTTPAddrs(ctx, host); err != nil {
		return err
	}
	return nil
}

func resolvePublicSkillHTTPAddrs(ctx context.Context, host string) ([]netip.Addr, error) {
	if ip, err := netip.ParseAddr(host); err == nil {
		if !isPublicSkillHTTPAddr(ip) {
			return nil, fmt.Errorf("url host resolves to a disallowed IP")
		}
		return []netip.Addr{ip.Unmap()}, nil
	}
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("url host DNS lookup failed: %w", err)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("url host DNS lookup returned no addresses")
	}
	for _, addr := range addrs {
		if !isPublicSkillHTTPAddr(addr.Unmap()) {
			return nil, fmt.Errorf("url host resolves to a disallowed IP")
		}
	}
	return addrs, nil
}

func isPublicSkillHTTPAddr(addr netip.Addr) bool {
	return addr.IsValid() &&
		!addr.IsLoopback() &&
		!addr.IsPrivate() &&
		!addr.IsLinkLocalUnicast() &&
		!addr.IsLinkLocalMulticast() &&
		!addr.IsUnspecified() &&
		!addr.IsMulticast()
}

func isLocalhostName(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	return host == "localhost" || strings.HasSuffix(host, ".localhost")
}

func isEmptyJSON(raw json.RawMessage) bool {
	return len(raw) == 0 || strings.TrimSpace(string(raw)) == "null"
}
