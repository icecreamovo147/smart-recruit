package metadata

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

type ctxKey string

const (
	KeyRequestID ctxKey = "x-request-id"
	KeyClientIP  ctxKey = "x-client-ip"

	// KeyAuthUserID and KeyAuthAccountType carry the authenticated actor's identity
	// from the HTTP gateway through gRPC metadata into backend service context.
	// Set by the gRPC server interceptor (injectMetadataIntoContext).
	KeyAuthUserID       ctxKey = "x-authenticated-user-id"
	KeyAuthAccountType  ctxKey = "x-authenticated-account-type"
	KeyAuthTenantID     ctxKey = "x-authenticated-tenant-id"
	KeyAuthMembershipID ctxKey = "x-authenticated-membership-id"
	KeyAuthClientApp    ctxKey = "x-authenticated-client-app"
	KeyTraceID          ctxKey = "x-trace-id"
	KeySpanID           ctxKey = "x-span-id"
	KeyTraceparent      ctxKey = "traceparent"
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

func GetAuthTenantID(ctx context.Context) int64 {
	return getInt64(ctx, KeyAuthTenantID)
}

func GetAuthMembershipID(ctx context.Context) int64 {
	return getInt64(ctx, KeyAuthMembershipID)
}

func GetAuthClientApp(ctx context.Context) string {
	if value, ok := ctx.Value(KeyAuthClientApp).(string); ok {
		return value
	}
	return ""
}

type TenantContext struct {
	TenantID     int64
	MembershipID int64
	UserID       int64
	AccountType  string
	ClientApp    string
}

func GetTenantContext(ctx context.Context) TenantContext {
	return TenantContext{
		TenantID: GetAuthTenantID(ctx), MembershipID: GetAuthMembershipID(ctx),
		UserID: GetAuthUserID(ctx), AccountType: GetAuthAccountType(ctx), ClientApp: GetAuthClientApp(ctx),
	}
}

func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(KeyTraceID).(string); ok {
		return v
	}
	return ""
}

func GetSpanID(ctx context.Context) string {
	if v, ok := ctx.Value(KeySpanID).(string); ok {
		return v
	}
	return ""
}

func GetTraceparent(ctx context.Context) string {
	if v, ok := ctx.Value(KeyTraceparent).(string); ok {
		return v
	}
	return ""
}

func WithTraceContext(ctx context.Context, incomingTraceparent, incomingTraceID string) context.Context {
	traceID, _ := parseTraceparent(incomingTraceparent)
	if traceID == "" && validTraceID(incomingTraceID) {
		traceID = strings.ToLower(incomingTraceID)
	}
	if traceID == "" {
		traceID = randomHex(16)
	}
	spanID := randomHex(8)
	traceparent := "00-" + traceID + "-" + spanID + "-01"
	ctx = context.WithValue(ctx, KeyTraceID, traceID)
	ctx = context.WithValue(ctx, KeySpanID, spanID)
	return context.WithValue(ctx, KeyTraceparent, traceparent)
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

func WithTenantActor(ctx context.Context, actor TenantContext) context.Context {
	ctx = WithAuthActor(ctx, actor.UserID, actor.AccountType)
	if actor.TenantID > 0 {
		ctx = context.WithValue(ctx, KeyAuthTenantID, actor.TenantID)
	}
	if actor.MembershipID > 0 {
		ctx = context.WithValue(ctx, KeyAuthMembershipID, actor.MembershipID)
	}
	if actor.ClientApp != "" {
		ctx = context.WithValue(ctx, KeyAuthClientApp, actor.ClientApp)
	}
	return ctx
}

func getInt64(ctx context.Context, key ctxKey) int64 {
	if value, ok := ctx.Value(key).(int64); ok {
		return value
	}
	if value, ok := ctx.Value(key).(string); ok {
		var id int64
		if _, err := fmt.Sscanf(value, "%d", &id); err == nil {
			return id
		}
	}
	return 0
}

func parseTraceparent(value string) (string, string) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 4 || parts[0] != "00" {
		return "", ""
	}
	traceID := strings.ToLower(parts[1])
	spanID := strings.ToLower(parts[2])
	if !validTraceID(traceID) || !validSpanID(spanID) {
		return "", ""
	}
	return traceID, spanID
}

func validTraceID(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return len(value) == 32 && value != "00000000000000000000000000000000" && isHex(value)
}

func validSpanID(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return len(value) == 16 && value != "0000000000000000" && isHex(value)
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
