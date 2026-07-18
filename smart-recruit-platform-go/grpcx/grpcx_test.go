package grpcx

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"smart-recruit-platform-go/servicemeta"
)

func TestNewServerRequiresServiceName(t *testing.T) {
	if _, err := NewServer(ServerConfig{}); err == nil {
		t.Fatal("expected missing service name error")
	}
	server, err := NewServer(ServerConfig{ServiceName: "identity"})
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
	server.Stop()
}

func TestDialOptionsValidatesTarget(t *testing.T) {
	if _, err := DialOptions(ClientConfig{}); err == nil {
		t.Fatal("expected missing target error")
	}
	if _, err := DialOptions(ClientConfig{Target: "127.0.0.1:50051"}); err != nil {
		t.Fatalf("DialOptions returned error: %v", err)
	}
}

func TestInternalTokenUnaryClientInterceptor(t *testing.T) {
	interceptor := internalTokenUnaryClientInterceptor("secret")
	err := interceptor(context.Background(), "/test.Service/Call", nil, nil, nil, func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			t.Fatal("expected outgoing metadata")
		}
		values := md.Get(servicemeta.InternalTokenHeader)
		if len(values) != 1 || values[0] != "secret" {
			t.Fatalf("unexpected token metadata: %#v", values)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
}
