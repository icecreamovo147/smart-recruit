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
	aiAgentConn            *grpc.ClientConn
	identityConn           *grpc.ClientConn
	recruitmentConn        *grpc.ClientConn
	NotificationRouteMode  string
	NotificationTargetAddr string
	AIAgentRouteMode       string
	AIAgentTargetAddr      string
	IdentityRouteMode      string
	IdentityTargetAddr     string
	RecruitmentRouteMode   string
	RecruitmentTargetAddr  string
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
	AIAgentHealth          healthpb.HealthClient
	IdentityHealth         healthpb.HealthClient
	RecruitmentHealth      healthpb.HealthClient
}

type ClientOptions struct {
	NotificationAddr      string
	NotificationRouteMode string
	AIAgentAddr           string
	AIAgentRouteMode      string
	IdentityAddr          string
	IdentityRouteMode     string
	RecruitmentAddr       string
	RecruitmentRouteMode  string
}

// NewClients creates a gRPC client connection with round-robin load balancing.
// Only read-only RPC methods get retry policies; write methods deliberately do
// not retry because they are not globally idempotent.
func NewClients(addr string) (*Clients, error) {
	return NewClientsWithOptions(addr, ClientOptions{
		NotificationAddr:      os.Getenv("NOTIFICATION_GRPC_ADDR"),
		NotificationRouteMode: os.Getenv("NOTIFICATION_ROUTE_MODE"),
		AIAgentAddr:           os.Getenv("AI_AGENT_GRPC_ADDR"),
		AIAgentRouteMode:      os.Getenv("AI_AGENT_ROUTE_MODE"),
		IdentityAddr:          os.Getenv("IDENTITY_GRPC_ADDR"),
		IdentityRouteMode:     os.Getenv("IDENTITY_ROUTE_MODE"),
		RecruitmentAddr:       os.Getenv("RECRUITMENT_GRPC_ADDR"),
		RecruitmentRouteMode:  os.Getenv("RECRUITMENT_ROUTE_MODE"),
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
	aiAgentMode := options.AIAgentRouteMode
	if aiAgentMode == "" {
		aiAgentMode = "logic"
	}
	if aiAgentMode != "logic" && aiAgentMode != "ai-agent" {
		return nil, fmt.Errorf("unsupported ai agent route mode %q", aiAgentMode)
	}
	identityMode := options.IdentityRouteMode
	if identityMode == "" {
		identityMode = "logic"
	}
	if identityMode != "logic" && identityMode != "identity" {
		return nil, fmt.Errorf("unsupported identity route mode %q", identityMode)
	}
	recruitmentMode := options.RecruitmentRouteMode
	if recruitmentMode == "" {
		recruitmentMode = "logic"
	}
	if recruitmentMode != "logic" && recruitmentMode != "recruitment" {
		return nil, fmt.Errorf("unsupported recruitment route mode %q", recruitmentMode)
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
	aiAgentConn := conn
	aiAgentTarget := addr
	identityConn := conn
	identityTarget := addr
	recruitmentConn := conn
	recruitmentTarget := addr
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
	if aiAgentMode == "ai-agent" {
		if options.AIAgentAddr == "" {
			closeClientConns(conn, notificationConn)
			return nil, fmt.Errorf("ai agent grpc addr is required when route mode is ai-agent")
		}
		aiAgentConn, err = grpc.NewClient(options.AIAgentAddr,
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
			closeClientConns(conn, notificationConn)
			return nil, err
		}
		aiAgentTarget = options.AIAgentAddr
	}
	if identityMode == "identity" {
		if options.IdentityAddr == "" {
			closeClientConns(conn, notificationConn, aiAgentConn)
			return nil, fmt.Errorf("identity grpc addr is required when route mode is identity")
		}
		identityConn, err = grpc.NewClient(options.IdentityAddr,
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
			closeClientConns(conn, notificationConn, aiAgentConn)
			return nil, err
		}
		identityTarget = options.IdentityAddr
	}
	if recruitmentMode == "recruitment" {
		if options.RecruitmentAddr == "" {
			closeClientConns(conn, notificationConn, aiAgentConn, identityConn)
			return nil, fmt.Errorf("recruitment grpc addr is required when route mode is recruitment")
		}
		recruitmentConn, err = grpc.NewClient(options.RecruitmentAddr,
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
			closeClientConns(conn, notificationConn, aiAgentConn, identityConn)
			return nil, err
		}
		recruitmentTarget = options.RecruitmentAddr
	}
	logicAdminClient := pb.NewAdminServiceClient(conn)
	identityAdminClient := logicAdminClient
	if identityMode == "identity" {
		identityAdminClient = newIdentityAdminClient(logicAdminClient, pb.NewAdminServiceClient(identityConn))
	}
	return &Clients{
		conn:                   conn,
		notificationConn:       notificationConn,
		aiAgentConn:            aiAgentConn,
		identityConn:           identityConn,
		recruitmentConn:        recruitmentConn,
		NotificationRouteMode:  notificationMode,
		NotificationTargetAddr: notificationTarget,
		AIAgentRouteMode:       aiAgentMode,
		AIAgentTargetAddr:      aiAgentTarget,
		IdentityRouteMode:      identityMode,
		IdentityTargetAddr:     identityTarget,
		RecruitmentRouteMode:   recruitmentMode,
		RecruitmentTargetAddr:  recruitmentTarget,
		Auth:                   pb.NewAuthServiceClient(identityConn),
		Job:                    pb.NewJobServiceClient(recruitmentConn),
		Candidate:              pb.NewCandidateServiceClient(recruitmentConn),
		Application:            pb.NewApplicationServiceClient(recruitmentConn),
		AI:                     pb.NewAIServiceClient(aiAgentConn),
		Notification:           pb.NewNotificationServiceClient(notificationConn),
		Interview:              pb.NewInterviewServiceClient(conn),
		Offer:                  pb.NewOfferServiceClient(conn),
		Admin:                  identityAdminClient,
		Collaboration:          pb.NewCollaborationServiceClient(conn),
		LlmConfig:              pb.NewLlmConfigServiceClient(aiAgentConn),
		Prompt:                 pb.NewPromptServiceClient(aiAgentConn),
		AgentConfig:            pb.NewAgentConfigServiceClient(aiAgentConn),
		MCP:                    pb.NewMCPServiceClient(aiAgentConn),
		Skill:                  pb.NewSkillServiceClient(aiAgentConn),
		AgentSkill:             pb.NewAgentSkillServiceClient(aiAgentConn),
		RecruitingIntelligence: pb.NewRecruitingIntelligenceServiceClient(aiAgentConn),
		EmbeddingConfig:        pb.NewEmbeddingConfigServiceClient(aiAgentConn),
		Health:                 healthpb.NewHealthClient(conn),
		NotificationHealth:     healthpb.NewHealthClient(notificationConn),
		AIAgentHealth:          healthpb.NewHealthClient(aiAgentConn),
		IdentityHealth:         healthpb.NewHealthClient(identityConn),
		RecruitmentHealth:      healthpb.NewHealthClient(recruitmentConn),
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
	if c.recruitmentConn != nil && c.recruitmentConn != c.conn && c.recruitmentConn != c.notificationConn && c.recruitmentConn != c.aiAgentConn && c.recruitmentConn != c.identityConn {
		if err := c.recruitmentConn.Close(); err != nil {
			closeClientConns(c.conn, c.notificationConn, c.aiAgentConn, c.identityConn)
			return err
		}
	}
	if c.identityConn != nil && c.identityConn != c.conn && c.identityConn != c.notificationConn && c.identityConn != c.aiAgentConn && c.identityConn != c.recruitmentConn {
		if err := c.identityConn.Close(); err != nil {
			closeClientConns(c.conn, c.notificationConn, c.aiAgentConn)
			return err
		}
	}
	if c.aiAgentConn != nil && c.aiAgentConn != c.conn && c.aiAgentConn != c.notificationConn && c.aiAgentConn != c.recruitmentConn {
		if err := c.aiAgentConn.Close(); err != nil {
			closeClientConns(c.conn, c.notificationConn)
			return err
		}
	}
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
	if c.aiAgentConn != nil && c.aiAgentConn != c.conn && c.aiAgentConn != c.notificationConn && c.AIAgentHealth != nil {
		if err := checkHealth(ctx, "ai-agent", c.AIAgentHealth); err != nil {
			return err
		}
	}
	if c.identityConn != nil && c.identityConn != c.conn && c.identityConn != c.notificationConn && c.identityConn != c.aiAgentConn && c.IdentityHealth != nil {
		if err := checkHealth(ctx, "identity", c.IdentityHealth); err != nil {
			return err
		}
	}
	if c.recruitmentConn != nil && c.recruitmentConn != c.conn && c.recruitmentConn != c.notificationConn && c.recruitmentConn != c.aiAgentConn && c.recruitmentConn != c.identityConn && c.RecruitmentHealth != nil {
		if err := checkHealth(ctx, "recruitment", c.RecruitmentHealth); err != nil {
			return err
		}
	}
	return nil
}

func closeClientConns(conn *grpc.ClientConn, extra ...*grpc.ClientConn) {
	seen := map[*grpc.ClientConn]struct{}{}
	for _, candidate := range append([]*grpc.ClientConn{conn}, extra...) {
		if candidate == nil {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		_ = candidate.Close()
	}
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
