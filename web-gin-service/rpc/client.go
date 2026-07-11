package rpc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"

	"web-gin-service/pkg/contextkeys"
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
		ctx = forwardMetadata(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func streamClientInterceptor(token string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, internalTokenHeader, token)
		}
		ctx = forwardMetadata(ctx)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

type Clients struct {
	conn                   *grpc.ClientConn
	notificationConn       *grpc.ClientConn
	NotificationRouteMode  string
	NotificationTargetAddr string
	Auth                   pb.AuthServiceClient
	Job                    pb.JobServiceClient
	Candidate              pb.CandidateServiceClient
	Application            pb.ApplicationServiceClient
	AI                     pb.AIServiceClient
	Notification           pb.NotificationServiceClient
	Interview              pb.InterviewServiceClient
	Offer                  pb.OfferServiceClient
	Admin                  pb.AdminServiceClient
	Collaboration          pb.CollaborationServiceClient
	LlmConfig              pb.LlmConfigServiceClient
	Prompt                 pb.PromptServiceClient
	AgentConfig            pb.AgentConfigServiceClient
	MCP                    pb.MCPServiceClient
	Skill                  pb.SkillServiceClient
	AgentSkill             pb.AgentSkillServiceClient
	RecruitingIntelligence pb.RecruitingIntelligenceServiceClient
	EmbeddingConfig        pb.EmbeddingConfigServiceClient
	Health                 healthpb.HealthClient
	NotificationHealth     healthpb.HealthClient
}

type ClientOptions struct {
	NotificationAddr      string
	NotificationRouteMode string
}

// NewClients creates a gRPC client connection with round-robin load balancing.
// Only read-only RPC methods get retry policies; write methods deliberately do
// not retry because they are not globally idempotent.
func NewClients(addr string) (*Clients, error) {
	return NewClientsWithOptions(addr, ClientOptions{
		NotificationAddr:      os.Getenv("NOTIFICATION_GRPC_ADDR"),
		NotificationRouteMode: os.Getenv("NOTIFICATION_ROUTE_MODE"),
	})
}

func NewClientsWithOptions(addr string, options ClientOptions) (*Clients, error) {
	notificationMode := options.NotificationRouteMode
	if notificationMode == "" {
		notificationMode = "logic"
	}
	if notificationMode != "logic" && notificationMode != "notification" {
		return nil, fmt.Errorf("unsupported notification route mode %q", notificationMode)
	}
	token := grpcInternalToken()
	opts := dialOptions(token)
	conn, err := grpc.NewClient(addr,
		append(opts,
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff: backoff.Config{
					MaxDelay: 5 * time.Second,
				},
				MinConnectTimeout: 3 * time.Second,
			}),
		)...,
	)
	if err != nil {
		return nil, err
	}
	notificationConn := conn
	notificationTarget := addr
	if notificationMode == "notification" {
		if options.NotificationAddr == "" {
			_ = conn.Close()
			return nil, fmt.Errorf("notification grpc addr is required when route mode is notification")
		}
		notificationConn, err = grpc.NewClient(options.NotificationAddr,
			append(opts,
				grpc.WithConnectParams(grpc.ConnectParams{
					Backoff: backoff.Config{
						MaxDelay: 5 * time.Second,
					},
					MinConnectTimeout: 3 * time.Second,
				}),
			)...,
		)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		notificationTarget = options.NotificationAddr
	}
	return &Clients{
		conn:                   conn,
		notificationConn:       notificationConn,
		NotificationRouteMode:  notificationMode,
		NotificationTargetAddr: notificationTarget,
		Auth:                   pb.NewAuthServiceClient(conn),
		Job:                    pb.NewJobServiceClient(conn),
		Candidate:              pb.NewCandidateServiceClient(conn),
		Application:            pb.NewApplicationServiceClient(conn),
		AI:                     pb.NewAIServiceClient(conn),
		Notification:           pb.NewNotificationServiceClient(notificationConn),
		Interview:              pb.NewInterviewServiceClient(conn),
		Offer:                  pb.NewOfferServiceClient(conn),
		Admin:                  pb.NewAdminServiceClient(conn),
		Collaboration:          pb.NewCollaborationServiceClient(conn),
		LlmConfig:              pb.NewLlmConfigServiceClient(conn),
		Prompt:                 pb.NewPromptServiceClient(conn),
		AgentConfig:            pb.NewAgentConfigServiceClient(conn),
		MCP:                    pb.NewMCPServiceClient(conn),
		Skill:                  pb.NewSkillServiceClient(conn),
		AgentSkill:             pb.NewAgentSkillServiceClient(conn),
		RecruitingIntelligence: pb.NewRecruitingIntelligenceServiceClient(conn),
		EmbeddingConfig:        pb.NewEmbeddingConfigServiceClient(conn),
		Health:                 healthpb.NewHealthClient(conn),
		NotificationHealth:     healthpb.NewHealthClient(notificationConn),
	}, nil
}

func dialOptions(token string) []grpc.DialOption {
	return []grpc.DialOption{
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
						{"service": "recruitment.AIService", "method": "GetAgentRun"},
						{"service": "recruitment.AIService", "method": "GetActiveAgentRun"},
						{"service": "recruitment.NotificationService", "method": "ListNotifications"},
						{"service": "recruitment.NotificationService", "method": "UnreadNotificationCount"},
						{"service": "recruitment.AdminService", "method": "GetDashboardReport"},
						{"service": "recruitment.AdminService", "method": "GetFunnelReport"},
						{"service": "recruitment.AdminService", "method": "GetTimeInStageReport"},
						{"service": "recruitment.AdminService", "method": "GetInterviewOfferMetrics"},
						{"service": "recruitment.AdminService", "method": "QueryAuthAuditLogs"},
						{"service": "recruitment.RecruitingIntelligenceService", "method": "GetResumeProfile"},
						{"service": "recruitment.RecruitingIntelligenceService", "method": "GetCandidateMatchEvaluation"},
						{"service": "recruitment.RecruitingIntelligenceService", "method": "CompareCandidatesForJob"},
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
}

func (c *Clients) Close() error {
	if c.notificationConn != nil && c.notificationConn != c.conn {
		if err := c.notificationConn.Close(); err != nil {
			_ = c.conn.Close()
			return err
		}
	}
	return c.conn.Close()
}

func (c *Clients) Ready(ctx context.Context) error {
	if err := checkHealth(ctx, "logic", c.Health); err != nil {
		return err
	}
	if c.notificationConn != nil && c.notificationConn != c.conn && c.NotificationHealth != nil {
		if err := checkHealth(ctx, "notification", c.NotificationHealth); err != nil {
			return err
		}
	}
	return nil
}

func checkHealth(ctx context.Context, target string, client healthpb.HealthClient) error {
	resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		return fmt.Errorf("%s grpc health check failed: %w", target, err)
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("%s grpc health status is %s", target, resp.GetStatus().String())
	}
	return nil
}

func forwardMetadata(ctx context.Context) context.Context {
	if rid, ok := ctx.Value(contextkeys.RequestID).(string); ok && rid != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", rid)
	}
	if ip, ok := ctx.Value(contextkeys.ClientIP).(string); ok && ip != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-client-ip", ip)
	}
	if uid, ok := ctx.Value(contextkeys.UserID).(int64); ok && uid > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-authenticated-user-id", strconv.FormatInt(uid, 10))
	}
	if at, ok := ctx.Value(contextkeys.AccountType).(string); ok && at != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-authenticated-account-type", at)
	}
	return ctx
}
