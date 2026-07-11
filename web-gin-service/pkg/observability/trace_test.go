package observability

import "testing"

func TestNewTraceContextPreservesIncomingTraceID(t *testing.T) {
	ctx := NewTraceContext("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	if ctx.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("TraceID = %q", ctx.TraceID)
	}
	if len(ctx.SpanID) != 16 {
		t.Fatalf("SpanID length = %d", len(ctx.SpanID))
	}
	if ctx.Traceparent == "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		t.Fatal("expected a new child span id")
	}
}
