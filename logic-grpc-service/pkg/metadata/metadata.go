package metadata

import (
	"context"
	"fmt"
)

type ctxKey string

const (
	KeyRequestID ctxKey = "x-request-id"
	KeyClientIP  ctxKey = "x-client-ip"

	// KeyAuthUserID and KeyAuthAccountType carry the authenticated actor's identity
	// from web-gin through gRPC metadata into the logic service context.
	// Set by the gRPC server interceptor (injectMetadataIntoContext).
	KeyAuthUserID      ctxKey = "x-authenticated-user-id"
	KeyAuthAccountType ctxKey = "x-authenticated-account-type"
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

// GetAuthUserID extracts the authenticated user ID from the gRPC context.
// Returns 0 if not set (e.g. unauthenticated path or test without metadata).
func GetAuthUserID(ctx context.Context) int64 {
	if v, ok := ctx.Value(KeyAuthUserID).(int64); ok {
		return v
	}
	// Fallback: the metadata comes as a string from gRPC headers.
	if v, ok := ctx.Value(KeyAuthUserID).(string); ok {
		// Best-effort parse — the interceptor stores it as int64.
		var id int64
		if _, err := fmt.Sscanf(v, "%d", &id); err == nil {
			return id
		}
	}
	return 0
}

// GetAuthAccountType extracts the authenticated account type from the gRPC context.
func GetAuthAccountType(ctx context.Context) string {
	if v, ok := ctx.Value(KeyAuthAccountType).(string); ok {
		return v
	}
	return ""
}

// WithAuthActor returns a child context with the given user ID and account type
// injected as the authenticated actor. Use this in tests to simulate gRPC metadata
// that would normally be set by the server interceptor.
func WithAuthActor(ctx context.Context, userID int64, accountType string) context.Context {
	ctx = context.WithValue(ctx, KeyAuthUserID, userID)
	if accountType != "" {
		ctx = context.WithValue(ctx, KeyAuthAccountType, accountType)
	}
	return ctx
}
