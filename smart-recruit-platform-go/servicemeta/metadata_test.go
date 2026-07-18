package servicemeta

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestServiceValidate(t *testing.T) {
	if err := (Service{Name: "identity", Env: "local", Version: "dev"}).Validate(); err != nil {
		t.Fatalf("expected valid service metadata: %v", err)
	}
	if err := (Service{Env: "local", Version: "dev"}).Validate(); err == nil {
		t.Fatal("expected missing name error")
	}
}

func TestAppendInternalToken(t *testing.T) {
	ctx, err := AppendInternalToken(context.Background(), " token ")
	if err != nil {
		t.Fatalf("AppendInternalToken returned error: %v", err)
	}
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	values := md.Get(InternalTokenHeader)
	if len(values) != 1 || values[0] != "token" {
		t.Fatalf("unexpected token metadata: %#v", values)
	}
}

func TestIncomingMetadataHelpers(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		InternalTokenHeader, "secret",
		RequestIDHeader, "req-1",
	))
	if got := InternalTokenFromIncoming(ctx); got != "secret" {
		t.Fatalf("unexpected internal token: %q", got)
	}
	if got := RequestIDFromIncoming(ctx); got != "req-1" {
		t.Fatalf("unexpected request id: %q", got)
	}
}
