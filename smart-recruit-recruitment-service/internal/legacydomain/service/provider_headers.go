package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"

	"smart-recruit-commons/pkg/crypto"
	"smart-recruit-platform-go/logger"
)

const maxExtraHeadersJSONBytes = 8192

var validHeaderName = regexp.MustCompile(`^[!#$%&'*+\-.^_` + "`" + `|~0-9A-Za-z]+$`)

var (
	ErrInvalidExtraHeadersJSON = errors.New("invalid extra_headers_json")
)

// validateAndCanonicalizeExtraHeaders parses and validates provider extra headers for persistence.
func validateAndCanonicalizeExtraHeaders(jsonStr string) (string, error) {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		return "", fmt.Errorf("%w: empty payload", ErrInvalidExtraHeadersJSON)
	}
	if len(trimmed) > maxExtraHeadersJSONBytes {
		return "", fmt.Errorf("%w: payload exceeds %d bytes", ErrInvalidExtraHeadersJSON, maxExtraHeadersJSONBytes)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return "", fmt.Errorf("%w: must be a JSON object", ErrInvalidExtraHeadersJSON)
	}
	if raw == nil {
		return "", fmt.Errorf("%w: must be a JSON object", ErrInvalidExtraHeadersJSON)
	}

	headers := make(map[string]string, len(raw))
	for key, value := range raw {
		name := strings.TrimSpace(key)
		if name == "" || !validHeaderName.MatchString(name) {
			return "", fmt.Errorf("%w: invalid header name %q", ErrInvalidExtraHeadersJSON, key)
		}
		var strValue string
		if err := json.Unmarshal(value, &strValue); err != nil {
			return "", fmt.Errorf("%w: header %q must have a string value", ErrInvalidExtraHeadersJSON, name)
		}
		if !utf8.ValidString(strValue) {
			return "", fmt.Errorf("%w: header %q contains invalid UTF-8", ErrInvalidExtraHeadersJSON, name)
		}
		headers[name] = strValue
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(headers); err != nil {
		return "", fmt.Errorf("%w: canonicalize failed", ErrInvalidExtraHeadersJSON)
	}
	canonical := strings.TrimSpace(buf.String())
	return canonical, nil
}

// maskExtraHeadersForResponse returns masked header JSON for API responses.
func maskExtraHeadersForResponse(stored string) string {
	if strings.TrimSpace(stored) == "" {
		return ""
	}
	headers, err := parseExtraHeadersForUse(stored)
	if err != nil {
		logger.L().Warn("invalid legacy extra_headers_json stored; returning empty masked value")
		return ""
	}
	masked := make(map[string]string, len(headers))
	for k, v := range headers {
		masked[k] = crypto.MaskAPIKey(v)
	}
	out, err := json.Marshal(masked)
	if err != nil {
		return ""
	}
	return string(out)
}

// parseExtraHeadersForUse parses stored extra headers for runtime provider calls.
func parseExtraHeadersForUse(stored string) (map[string]string, error) {
	trimmed := strings.TrimSpace(stored)
	if trimmed == "" {
		return nil, nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(trimmed), &headers); err != nil {
		return nil, fmt.Errorf("%w: stored value is not valid JSON object", ErrInvalidExtraHeadersJSON)
	}
	if headers == nil {
		return nil, fmt.Errorf("%w: stored value is not valid JSON object", ErrInvalidExtraHeadersJSON)
	}
	for key, value := range headers {
		name := strings.TrimSpace(key)
		if name == "" || !validHeaderName.MatchString(name) {
			return nil, fmt.Errorf("%w: invalid stored header name %q", ErrInvalidExtraHeadersJSON, key)
		}
		if !utf8.ValidString(value) {
			return nil, fmt.Errorf("%w: invalid stored header value for %q", ErrInvalidExtraHeadersJSON, name)
		}
	}
	return headers, nil
}

// applyExtraHeadersToRequest sets validated extra headers on an HTTP request.
func applyExtraHeadersToRequest(req *http.Request, stored string) error {
	headers, err := parseExtraHeadersForUse(stored)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return nil
}

// extraHeadersPtrFromRequest validates request extra headers and returns a DB pointer.
func extraHeadersPtrFromRequest(jsonStr string) (*string, error) {
	canonical, err := validateAndCanonicalizeExtraHeaders(jsonStr)
	if err != nil {
		return nil, err
	}
	return &canonical, nil
}

// logInvalidExtraHeaders logs a warning without exposing secret values.
func logInvalidExtraHeaders(providerID int64, providerKind string) {
	logger.L().Warn("invalid legacy extra_headers_json",
		zap.Int64("provider_id", providerID),
		zap.String("provider_kind", providerKind),
	)
}
