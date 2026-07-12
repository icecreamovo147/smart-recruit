package rpc

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestNewClientsDefaultsNotificationToService(t *testing.T) {
	clients, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{})
	if err != nil {
		t.Fatalf("NewClientsWithOptions: %v", err)
	}
	defer clients.Close()

	if clients.NotificationRouteMode != "notification" {
		t.Fatalf("NotificationRouteMode = %q, want notification", clients.NotificationRouteMode)
	}
	if clients.NotificationTargetAddr != "127.0.0.1:50065" {
		t.Fatalf("NotificationTargetAddr = %q", clients.NotificationTargetAddr)
	}
	if clients.notificationConn == clients.conn {
		t.Fatal("default notification client should use the notification service connection")
	}
}

func TestNewClientsRoutesNotificationToExtractedService(t *testing.T) {
	clients, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
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

func TestNewClientsRoutesAIAgentToExtractedService(t *testing.T) {
	clients, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		AIAgentRouteMode: "ai-agent",
		AIAgentAddr:      "passthrough:///ai-agent:50051",
	})
	if err != nil {
		t.Fatalf("NewClientsWithOptions: %v", err)
	}
	defer clients.Close()

	if clients.AIAgentRouteMode != "ai-agent" {
		t.Fatalf("AIAgentRouteMode = %q, want ai-agent", clients.AIAgentRouteMode)
	}
	if clients.AIAgentTargetAddr != "passthrough:///ai-agent:50051" {
		t.Fatalf("AIAgentTargetAddr = %q", clients.AIAgentTargetAddr)
	}
	if clients.aiAgentConn == clients.conn {
		t.Fatal("AI Agent cutover should use a separate gRPC connection")
	}
}

func TestNewClientsRoutesAnalyticsToExtractedService(t *testing.T) {
	clients, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		AnalyticsRouteMode: "analytics",
		AnalyticsAddr:      "passthrough:///analytics:50051",
	})
	if err != nil {
		t.Fatalf("NewClientsWithOptions: %v", err)
	}
	defer clients.Close()

	if clients.AnalyticsRouteMode != "analytics" {
		t.Fatalf("AnalyticsRouteMode = %q, want analytics", clients.AnalyticsRouteMode)
	}
	if clients.AnalyticsTargetAddr != "passthrough:///analytics:50051" {
		t.Fatalf("AnalyticsTargetAddr = %q", clients.AnalyticsTargetAddr)
	}
	if clients.analyticsConn == clients.conn {
		t.Fatal("Analytics cutover should use a separate gRPC connection")
	}
	if _, ok := clients.Admin.(*analyticsAdminClient); !ok {
		t.Fatalf("Admin client type = %T, want *analyticsAdminClient", clients.Admin)
	}
}

func TestNewClientsRoutesIdentityToExtractedService(t *testing.T) {
	clients, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		IdentityRouteMode: "identity",
		IdentityAddr:      "passthrough:///identity:50051",
	})
	if err != nil {
		t.Fatalf("NewClientsWithOptions: %v", err)
	}
	defer clients.Close()

	if clients.IdentityRouteMode != "identity" {
		t.Fatalf("IdentityRouteMode = %q, want identity", clients.IdentityRouteMode)
	}
	if clients.IdentityTargetAddr != "passthrough:///identity:50051" {
		t.Fatalf("IdentityTargetAddr = %q", clients.IdentityTargetAddr)
	}
	if clients.identityConn == clients.conn {
		t.Fatal("Identity cutover should use a separate gRPC connection")
	}
	if _, ok := clients.Admin.(*analyticsAdminClient); !ok {
		t.Fatalf("Admin client type = %T, want *analyticsAdminClient", clients.Admin)
	}
}

func TestNewClientsRoutesRecruitmentToExtractedService(t *testing.T) {
	clients, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		RecruitmentRouteMode: "recruitment",
		RecruitmentAddr:      "passthrough:///recruitment:50051",
	})
	if err != nil {
		t.Fatalf("NewClientsWithOptions: %v", err)
	}
	defer clients.Close()

	if clients.RecruitmentRouteMode != "recruitment" {
		t.Fatalf("RecruitmentRouteMode = %q, want recruitment", clients.RecruitmentRouteMode)
	}
	if clients.RecruitmentTargetAddr != "passthrough:///recruitment:50051" {
		t.Fatalf("RecruitmentTargetAddr = %q", clients.RecruitmentTargetAddr)
	}
	if clients.recruitmentConn == clients.conn {
		t.Fatal("Recruitment cutover should use a separate gRPC connection")
	}
}

func TestNewClientsRoutesInterviewToExtractedService(t *testing.T) {
	clients, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		InterviewRouteMode: "interview",
		InterviewAddr:      "passthrough:///interview:50051",
	})
	if err != nil {
		t.Fatalf("NewClientsWithOptions: %v", err)
	}
	defer clients.Close()

	if clients.InterviewRouteMode != "interview" {
		t.Fatalf("InterviewRouteMode = %q, want interview", clients.InterviewRouteMode)
	}
	if clients.InterviewTargetAddr != "passthrough:///interview:50051" {
		t.Fatalf("InterviewTargetAddr = %q", clients.InterviewTargetAddr)
	}
	if clients.interviewConn == clients.conn {
		t.Fatal("Interview cutover should use a separate gRPC connection")
	}
}

func TestNewClientsRoutesOfferToExtractedService(t *testing.T) {
	clients, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		OfferRouteMode: "offer",
		OfferAddr:      "passthrough:///offer:50051",
	})
	if err != nil {
		t.Fatalf("NewClientsWithOptions: %v", err)
	}
	defer clients.Close()

	if clients.OfferRouteMode != "offer" {
		t.Fatalf("OfferRouteMode = %q, want offer", clients.OfferRouteMode)
	}
	if clients.OfferTargetAddr != "passthrough:///offer:50051" {
		t.Fatalf("OfferTargetAddr = %q", clients.OfferTargetAddr)
	}
	if clients.offerConn == clients.conn {
		t.Fatal("Offer cutover should use a separate gRPC connection")
	}
}

func TestNewClientsRejectsInvalidRecruitmentRouteMode(t *testing.T) {
	_, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		RecruitmentRouteMode: "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid recruitment route mode error")
	}
}

func TestNewClientsRejectsInvalidInterviewRouteMode(t *testing.T) {
	_, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		InterviewRouteMode: "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid interview route mode error")
	}
}

func TestNewClientsRejectsInvalidOfferRouteMode(t *testing.T) {
	_, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		OfferRouteMode: "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid offer route mode error")
	}
}

func TestNewClientsRejectsInvalidAnalyticsRouteMode(t *testing.T) {
	_, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		AnalyticsRouteMode: "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid analytics route mode error")
	}
}

func TestNewClientsRejectsInvalidIdentityRouteMode(t *testing.T) {
	_, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		IdentityRouteMode: "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid identity route mode error")
	}
}

func TestNewClientsRejectsInvalidAIAgentRouteMode(t *testing.T) {
	_, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		AIAgentRouteMode: "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid ai agent route mode error")
	}
}

func TestNewClientsRejectsInvalidNotificationRouteMode(t *testing.T) {
	_, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		NotificationRouteMode: "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid route mode error")
	}
}

func TestNewClientsRequiresCAFileWhenInternalTLSRequired(t *testing.T) {
	_, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		GRPCInternalTLS: "required",
	})
	if err == nil || !strings.Contains(err.Error(), "GRPC_TLS_CA_FILE") {
		t.Fatalf("expected GRPC_TLS_CA_FILE error, got %v", err)
	}
}

func TestNewClientsLoadsInternalTLSCA(t *testing.T) {
	caFile := writeTestCertificate(t)
	clients, err := NewClientsWithOptions("passthrough:///recruitment:50062", ClientOptions{
		GRPCInternalTLS:   "required",
		GRPCTLSCAFile:     caFile,
		GRPCTLSServerName: "smart-recruit-identity-service.recruitment.svc.cluster.local",
	})
	if err != nil {
		t.Fatalf("NewClientsWithOptions: %v", err)
	}
	defer clients.Close()
	if !clients.InternalTLSEnabled {
		t.Fatal("InternalTLSEnabled = false, want true")
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

func TestReadySkipsNotificationHealthWhenRouteUsesPrimaryConnection(t *testing.T) {
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

func TestReadyChecksAIAgentHealthWhenCutoverUsesSeparateConnection(t *testing.T) {
	clients := &Clients{
		conn:          &grpc.ClientConn{},
		aiAgentConn:   &grpc.ClientConn{},
		Health:        fakeHealthClient{status: healthpb.HealthCheckResponse_SERVING},
		AIAgentHealth: fakeHealthClient{err: errors.New("ai agent down")},
	}

	err := clients.Ready(context.Background())
	if err == nil || !strings.Contains(err.Error(), "ai-agent grpc health check failed") {
		t.Fatalf("expected ai agent health failure, got %v", err)
	}
}

func TestReadyChecksIdentityHealthWhenCutoverUsesSeparateConnection(t *testing.T) {
	clients := &Clients{
		conn:           &grpc.ClientConn{},
		identityConn:   &grpc.ClientConn{},
		Health:         fakeHealthClient{status: healthpb.HealthCheckResponse_SERVING},
		IdentityHealth: fakeHealthClient{err: errors.New("identity down")},
	}

	err := clients.Ready(context.Background())
	if err == nil || !strings.Contains(err.Error(), "identity grpc health check failed") {
		t.Fatalf("expected identity health failure, got %v", err)
	}
}

func TestReadyChecksRecruitmentHealthWhenCutoverUsesSeparateConnection(t *testing.T) {
	clients := &Clients{
		conn:              &grpc.ClientConn{},
		recruitmentConn:   &grpc.ClientConn{},
		Health:            fakeHealthClient{status: healthpb.HealthCheckResponse_SERVING},
		RecruitmentHealth: fakeHealthClient{err: errors.New("recruitment down")},
	}

	err := clients.Ready(context.Background())
	if err == nil || !strings.Contains(err.Error(), "recruitment grpc health check failed") {
		t.Fatalf("expected recruitment health failure, got %v", err)
	}
}

func TestReadyChecksInterviewHealthWhenCutoverUsesSeparateConnection(t *testing.T) {
	clients := &Clients{
		conn:            &grpc.ClientConn{},
		interviewConn:   &grpc.ClientConn{},
		Health:          fakeHealthClient{status: healthpb.HealthCheckResponse_SERVING},
		InterviewHealth: fakeHealthClient{err: errors.New("interview down")},
	}

	err := clients.Ready(context.Background())
	if err == nil || !strings.Contains(err.Error(), "interview grpc health check failed") {
		t.Fatalf("expected interview health failure, got %v", err)
	}
}

func TestReadyChecksOfferHealthWhenCutoverUsesSeparateConnection(t *testing.T) {
	clients := &Clients{
		conn:        &grpc.ClientConn{},
		offerConn:   &grpc.ClientConn{},
		Health:      fakeHealthClient{status: healthpb.HealthCheckResponse_SERVING},
		OfferHealth: fakeHealthClient{err: errors.New("offer down")},
	}

	err := clients.Ready(context.Background())
	if err == nil || !strings.Contains(err.Error(), "offer grpc health check failed") {
		t.Fatalf("expected offer health failure, got %v", err)
	}
}

func TestReadyChecksAnalyticsHealthWhenCutoverUsesSeparateConnection(t *testing.T) {
	clients := &Clients{
		conn:            &grpc.ClientConn{},
		analyticsConn:   &grpc.ClientConn{},
		Health:          fakeHealthClient{status: healthpb.HealthCheckResponse_SERVING},
		AnalyticsHealth: fakeHealthClient{err: errors.New("analytics down")},
	}

	err := clients.Ready(context.Background())
	if err == nil || !strings.Contains(err.Error(), "analytics grpc health check failed") {
		t.Fatalf("expected analytics health failure, got %v", err)
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

func writeTestCertificate(t *testing.T) string {
	t.Helper()
	const cert = `-----BEGIN CERTIFICATE-----
MIIBODCB36ADAgECAhQ+r5/1OEvz/cAuCOpVBMNZBuT85TAKBggqhkjOPQQDAjAS
MRAwDgYDVQQDDAd0ZXN0LWNhMB4XDTI2MDcxMDE4NTAxM1oXDTM2MDcwODE4NTAx
M1owEjEQMA4GA1UEAwwHdGVzdC1jYTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IA
BOGBDDutMrmJV/Ik3HMDqQHetQXJW+XX2wC7JIrR0Hmml+PhJqCHrmehIvI7tIc4
gChWmTI8NhHX+YCH7QJCul2jEzARMA8GA1UdEwEB/wQFMAMBAf8wCgYIKoZIzj0E
AwIDSAAwRQIgLJmnxdkKmii70e47ESLi1D0R/XzreLokhSmUepKgzCMCIQCz8rDI
fGNwAACec8twMMOFx6oNlOD5U8qc2yUI2qK93g==
-----END CERTIFICATE-----
`
	file, err := os.CreateTemp(t.TempDir(), "ca-*.crt")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := file.WriteString(cert); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return file.Name()
}
