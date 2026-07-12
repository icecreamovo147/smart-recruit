package runtime

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"smart-recruit-domain-go/service"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestRuntimeRegistersNotificationGRPCService(t *testing.T) {
	runtime, err := New(Deps{Notification: fakeNotificationAPI{}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)
	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC returned error: %v", err)
	}
	services := server.GetServiceInfo()
	if _, ok := services[pb.NotificationService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("missing registered service %s", pb.NotificationService_ServiceDesc.ServiceName)
	}
}

func TestRuntimeRequiresNotificationDependency(t *testing.T) {
	if _, err := New(Deps{}); err == nil {
		t.Fatal("expected missing notification dependency error")
	}
}

func TestComponentsRequirePersistenceRealtimeOutboxInboxAndEmail(t *testing.T) {
	if err := (Components{}).Validate(); err == nil {
		t.Fatal("expected missing components error")
	}
	components := Components{
		Persistence:       true,
		RealtimeDelivery:  true,
		OutboxPublisher:   true,
		NotificationInbox: true,
		EmailCoordination: true,
	}
	if err := components.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestRuntimeInspectsNotificationRuntimeComponents(t *testing.T) {
	runtime, err := New(Deps{
		Notification: fakeNotificationAPI{},
		Runtime: &service.NotificationRuntime{
			Notification:         &service.NotificationService{},
			Worker:               &service.NotificationWorkerPool{},
			OutboxPublisher:      &service.OutboxPublisher{},
			NotificationConsumer: &service.NotificationConsumer{},
			EmailConsumer:        &service.EmailConsumer{},
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := runtime.Components.Validate(); err != nil {
		t.Fatalf("component validation failed: %v", err)
	}
}

func TestIdempotencySemanticsDocumentOutboxInboxAndRealtime(t *testing.T) {
	joined := strings.Join(IdempotencySemantics, "\n")
	for _, required := range []string{"CreateOnceWithResult", "Inbox", "email consumer", "outbox publisher", "realtime delivery"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("idempotency semantics missing %q: %s", required, joined)
		}
	}
}

type fakeNotificationAPI struct{}

func (fakeNotificationAPI) ListNotifications(context.Context, *pb.ListNotificationsRequest) (*pb.ListNotificationsResponse, error) {
	return &pb.ListNotificationsResponse{Code: errs.OK}, nil
}

func (fakeNotificationAPI) UnreadNotificationCount(context.Context, *pb.UnreadNotificationCountRequest) (*pb.UnreadNotificationCountResponse, error) {
	return &pb.UnreadNotificationCountResponse{}, nil
}

func (fakeNotificationAPI) NotificationSummary(context.Context, *pb.NotificationSummaryRequest) (*pb.NotificationSummaryResponse, error) {
	return &pb.NotificationSummaryResponse{}, nil
}

func (fakeNotificationAPI) MarkNotificationRead(context.Context, *pb.MarkNotificationReadRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeNotificationAPI) MarkAllNotificationsRead(context.Context, *pb.MarkAllNotificationsReadRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}
