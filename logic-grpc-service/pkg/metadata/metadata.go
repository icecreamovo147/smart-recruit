package metadata

import "context"

type ctxKey string

const (
	KeyRequestID ctxKey = "x-request-id"
	KeyClientIP  ctxKey = "x-client-ip"
)

// GetRequestID extracts the HTTP request-id from context.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(KeyRequestID).(string); ok {
		return v
	}
	return ""
}

// GetClientIP extracts the client IP from context.
func GetClientIP(ctx context.Context) string {
	if v, ok := ctx.Value(KeyClientIP).(string); ok {
		return v
	}
	return ""
}
