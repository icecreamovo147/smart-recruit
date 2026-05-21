package rpc

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"web-gin-service/recruitment/pb"
)

const internalTokenHeader = "x-internal-token"

// grpcInternalToken returns the shared secret for gRPC internal auth.
func grpcInternalToken() string {
	return os.Getenv("GRPC_INTERNAL_TOKEN")
}

func unaryClientInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, internalTokenHeader, token)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func streamClientInterceptor(token string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, internalTokenHeader, token)
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}

type Clients struct {
	conn         *grpc.ClientConn
	Auth         pb.AuthServiceClient
	Job          pb.JobServiceClient
	Candidate    pb.CandidateServiceClient
	Application  pb.ApplicationServiceClient
	AI           pb.AIServiceClient
	Notification pb.NotificationServiceClient
		Admin        pb.AdminServiceClient
	Health       healthpb.HealthClient
}

// NewClients creates a gRPC client connection with round-robin load balancing.
// Only read-only RPC methods get retry policies; write methods deliberately do
// not retry because they are not globally idempotent.
func NewClients(addr string) (*Clients, error) {
	token := grpcInternalToken()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(unaryClientInterceptor(token)),
		grpc.WithStreamInterceptor(streamClientInterceptor(token)),
		grpc.WithDefaultServiceConfig(`{
			"loadBalancingPolicy": "round_robin",
			"methodConfig": [
				{
					"name": [
						{"service": "recruitment.JobService", "method": "ListHRJobs"},
						{"service": "recruitment.JobService", "method": "ListPublicJobs"},
						{"service": "recruitment.JobService", "method": "GetJobDetail"},
						{"service": "recruitment.CandidateService", "method": "GetProfile"},
						{"service": "recruitment.CandidateService", "method": "GetResume"},
						{"service": "recruitment.ApplicationService", "method": "ListMyApplications"},
						{"service": "recruitment.ApplicationService", "method": "ListJobApplications"},
						{"service": "recruitment.AIService", "method": "History"},
						{"service": "recruitment.AIService", "method": "ListChatSessions"},
						{"service": "recruitment.AIService", "method": "SessionMessages"},
						{"service": "recruitment.NotificationService", "method": "ListNotifications"},
						{"service": "recruitment.NotificationService", "method": "UnreadNotificationCount"},
						{"service": "grpc.health.v1.Health", "method": "Check"}
					],
					"retryPolicy": {
						"maxAttempts": 3,
						"initialBackoff": "0.1s",
						"maxBackoff": "1s",
						"backoffMultiplier": 2,
						"retryableStatusCodes": ["UNAVAILABLE"]
					}
				}
			]
		}`),
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}
	return &Clients{
		conn:         conn,
		Auth:         pb.NewAuthServiceClient(conn),
		Job:          pb.NewJobServiceClient(conn),
		Candidate:    pb.NewCandidateServiceClient(conn),
		Application:  pb.NewApplicationServiceClient(conn),
		AI:           pb.NewAIServiceClient(conn),
		Notification: pb.NewNotificationServiceClient(conn),
		Admin:        pb.NewAdminServiceClient(conn),
		Health:       healthpb.NewHealthClient(conn),
	}, nil
}

func (c *Clients) Close() error {
	return c.conn.Close()
}

func (c *Clients) Ready(ctx context.Context) error {
	resp, err := c.Health.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return err
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("grpc health status is %s", resp.GetStatus().String())
	}
	return nil
}
