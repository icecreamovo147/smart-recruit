package rpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestNewClientsDefaultsNotificationToLogic(t *testing.T) {
	clients, err := NewClientsWithOptions("passthrough:///logic:50051", ClientOptions{})
	if err != nil {
		t.Fatalf("NewClientsWithOptions: %v", err)
	}
	defer clients.Close()

	if clients.NotificationRouteMode != "logic" {
		t.Fatalf("NotificationRouteMode = %q, want logic", clients.NotificationRouteMode)
	}
	if clients.NotificationTargetAddr != "passthrough:///logic:50051" {
		t.Fatalf("NotificationTargetAddr = %q", clients.NotificationTargetAddr)
	}
	if clients.notificationConn != clients.conn {
		t.Fatal("default notification client should reuse the logic connection")
	}
}

func TestNewClientsRoutesNotificationToExtractedService(t *testing.T) {
	clients, err := NewClientsWithOptions("passthrough:///logic:50051", ClientOptions{
		NotificationRouteMode: "notification",
		NotificationAddr:      "passthrough:///notification:50051",
	})
	if err != nil {
		t.Fatalf("NewClientsWithOptions: %v", err)
	}
	defer clients.Close()

	if clients.NotificationRouteMode != "notification" {
		t.Fatalf("NotificationRouteMode = %q, want notification", clients.NotificationRouteMode)
	}
	if clients.NotificationTargetAddr != "passthrough:///notification:50051" {
		t.Fatalf("NotificationTargetAddr = %q", clients.NotificationTargetAddr)
	}
	if clients.notificationConn == clients.conn {
		t.Fatal("notification cutover should use a separate gRPC connection")
	}
}

func TestNewClientsNotificationCutoverRequiresAddress(t *testing.T) {
	_, err := NewClientsWithOptions("passthrough:///logic:50051", ClientOptions{
		NotificationRouteMode: "notification",
	})
	if err == nil {
		t.Fatal("expected missing notification address error")
	}
}

func TestNewClientsRejectsInvalidNotificationRouteMode(t *testing.T) {
	_, err := NewClientsWithOptions("passthrough:///logic:50051", ClientOptions{
		NotificationRouteMode: "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid route mode error")
	}
}

func TestReadyChecksNotificationHealthWhenCutoverUsesSeparateConnection(t *testing.T) {
	clients := &Clients{
		conn:               &grpc.ClientConn{},
		notificationConn:   &grpc.ClientConn{},
		Health:             fakeHealthClient{status: healthpb.HealthCheckResponse_SERVING},
		NotificationHealth: fakeHealthClient{err: errors.New("notification down")},
	}

	err := clients.Ready(context.Background())
	if err == nil || !strings.Contains(err.Error(), "notification grpc health check failed") {
		t.Fatalf("expected notification health failure, got %v", err)
	}
}

func TestReadySkipsNotificationHealthWhenRouteUsesLogicConnection(t *testing.T) {
	conn := &grpc.ClientConn{}
	clients := &Clients{
		conn:               conn,
		notificationConn:   conn,
		Health:             fakeHealthClient{status: healthpb.HealthCheckResponse_SERVING},
		NotificationHealth: fakeHealthClient{err: errors.New("should not be called")},
	}

	if err := clients.Ready(context.Background()); err != nil {
		t.Fatalf("Ready: %v", err)
	}
}

type fakeHealthClient struct {
	status healthpb.HealthCheckResponse_ServingStatus
	err    error
}

func (f fakeHealthClient) Check(context.Context, *healthpb.HealthCheckRequest, ...grpc.CallOption) (*healthpb.HealthCheckResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &healthpb.HealthCheckResponse{Status: f.status}, nil
}

func (f fakeHealthClient) List(context.Context, *healthpb.HealthListRequest, ...grpc.CallOption) (*healthpb.HealthListResponse, error) {
	return nil, nil
}

func (f fakeHealthClient) Watch(context.Context, *healthpb.HealthCheckRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[healthpb.HealthCheckResponse], error) {
	return nil, nil
}
