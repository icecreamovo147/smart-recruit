package metadata

import (
	"context"
	"testing"
)

func TestWithTraceContextPreservesIncomingTraceID(t *testing.T) {
	ctx := WithTraceContext(context.Background(), "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", "")
	if got := GetTraceID(ctx); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("TraceID = %q", got)
	}
	if got := GetSpanID(ctx); len(got) != 16 {
		t.Fatalf("SpanID length = %d", len(got))
	}
	if got := GetTraceparent(ctx); got == "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		t.Fatal("expected a new child span id")
	}
}

func TestWithTenantActorRoundTripsTrustedContext(t *testing.T) {
	want := TenantContext{
		TenantID: 12, MembershipID: 34, UserID: 56,
		AccountType: "staff", ClientApp: "staff",
	}
	got := GetTenantContext(WithTenantActor(context.Background(), want))
	if got != want {
		t.Fatalf("tenant context = %#v, want %#v", got, want)
	}
}

func TestWithAuthActorRemainsGlobalByDefault(t *testing.T) {
	got := GetTenantContext(WithAuthActor(context.Background(), 7, "candidate"))
	if got.UserID != 7 || got.AccountType != "candidate" {
		t.Fatalf("actor = %#v", got)
	}
	if got.TenantID != 0 || got.MembershipID != 0 || got.ClientApp != "" {
		t.Fatalf("global actor unexpectedly tenant-bound: %#v", got)
	}
}
