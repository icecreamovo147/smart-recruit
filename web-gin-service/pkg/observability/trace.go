package observability

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

type TraceContext struct {
	TraceID     string
	SpanID      string
	Traceparent string
}

func NewTraceContext(incoming string) TraceContext {
	traceID, _ := parseTraceparent(incoming)
	if traceID == "" {
		traceID = randomHex(16)
	}
	spanID := randomHex(8)
	traceparent := "00-" + traceID + "-" + spanID + "-01"
	return TraceContext{TraceID: traceID, SpanID: spanID, Traceparent: traceparent}
}

func parseTraceparent(value string) (string, string) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 4 || parts[0] != "00" {
		return "", ""
	}
	traceID := strings.ToLower(parts[1])
	spanID := strings.ToLower(parts[2])
	if len(traceID) != 32 || len(spanID) != 16 || !isHex(traceID) || !isHex(spanID) {
		return "", ""
	}
	if traceID == "00000000000000000000000000000000" || spanID == "0000000000000000" {
		return "", ""
	}
	return traceID, spanID
}

func randomHex(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return strings.Repeat("0", size*2-1) + "1"
	}
	return hex.EncodeToString(bytes)
}

func isHex(value string) bool {
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}
