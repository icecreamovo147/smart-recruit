package rpc

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"smart-recruit-gateway/pkg/contextkeys"
	"smart-recruit-gateway/pkg/observability"
	"smart-recruit-proto/recruitment/pb"
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
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		observability.DefaultMetrics.ObserveGRPCClient(method, status.Code(err).String(), time.Since(start))
		return err
	}
}

func streamClientInterceptor(token string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if token != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, internalTokenHeader, token)
		}
		ctx = forwardMetadata(ctx)
		start := time.Now()
		stream, err := streamer(ctx, desc, cc, method, opts...)
		observability.DefaultMetrics.ObserveGRPCClient(method, status.Code(err).String(), time.Since(start))
		return stream, err
	}
}

type Clients struct {
	conn                   *grpc.ClientConn
	notificationConn       *grpc.ClientConn
	aiAgentConn            *grpc.ClientConn
	identityConn           *grpc.ClientConn
	recruitmentConn        *grpc.ClientConn
	interviewConn          *grpc.ClientConn
	offerConn              *grpc.ClientConn
	analyticsConn          *grpc.ClientConn
	NotificationRouteMode  string
	NotificationTargetAddr string
	AIAgentRouteMode       string
	AIAgentTargetAddr      string
	IdentityRouteMode      string
	IdentityTargetAddr     string
	RecruitmentRouteMode   string
	RecruitmentTargetAddr  string
	InterviewRouteMode     string
	InterviewTargetAddr    string
	OfferRouteMode         string
	OfferTargetAddr        string
	AnalyticsRouteMode     string
	AnalyticsTargetAddr    string
	InternalTLSEnabled     bool
	Auth                   pb.AuthServiceClient
	Tenant                 pb.PlatformTenantServiceClient
	Job                    pb.JobServiceClient
	Candidate              pb.CandidateServiceClient
	Application            pb.ApplicationServiceClient
	ApplicationOwner       pb.ApplicationOwnerServiceClient
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
	InterviewHealth        healthpb.HealthClient
	OfferHealth            healthpb.HealthClient
	AnalyticsHealth        healthpb.HealthClient
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
	InterviewAddr         string
	InterviewRouteMode    string
	OfferAddr             string
	OfferRouteMode        string
	AnalyticsAddr         string
	AnalyticsRouteMode    string
	GRPCInternalTLS       string
	GRPCTLSCAFile         string
	GRPCTLSServerName     string
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
		InterviewAddr:         os.Getenv("INTERVIEW_GRPC_ADDR"),
		InterviewRouteMode:    os.Getenv("INTERVIEW_ROUTE_MODE"),
		OfferAddr:             os.Getenv("OFFER_GRPC_ADDR"),
		OfferRouteMode:        os.Getenv("OFFER_ROUTE_MODE"),
		AnalyticsAddr:         os.Getenv("ANALYTICS_GRPC_ADDR"),
		AnalyticsRouteMode:    os.Getenv("ANALYTICS_ROUTE_MODE"),
		GRPCInternalTLS:       os.Getenv("GRPC_INTERNAL_TLS"),
		GRPCTLSCAFile:         os.Getenv("GRPC_TLS_CA_FILE"),
		GRPCTLSServerName:     os.Getenv("GRPC_TLS_SERVER_NAME"),
	})
}

func NewClientsWithOptions(addr string, options ClientOptions) (*Clients, error) {
	notificationMode := options.NotificationRouteMode
	if notificationMode == "" {
		notificationMode = "notification"
	}
	if notificationMode != "notification" {
		return nil, fmt.Errorf("unsupported notification route mode %q", notificationMode)
	}
	if notificationMode == "notification" && strings.TrimSpace(options.NotificationAddr) == "" {
		options.NotificationAddr = "127.0.0.1:50065"
	}
	aiAgentMode := options.AIAgentRouteMode
	if aiAgentMode == "" {
		aiAgentMode = "ai-agent"
	}
	if aiAgentMode != "ai-agent" {
		return nil, fmt.Errorf("unsupported ai agent route mode %q", aiAgentMode)
	}
	if aiAgentMode == "ai-agent" && strings.TrimSpace(options.AIAgentAddr) == "" {
		options.AIAgentAddr = "127.0.0.1:50066"
	}
	identityMode := options.IdentityRouteMode
	if identityMode == "" {
		identityMode = "identity"
	}
	if identityMode != "identity" {
		return nil, fmt.Errorf("unsupported identity route mode %q", identityMode)
	}
	if identityMode == "identity" && strings.TrimSpace(options.IdentityAddr) == "" {
		options.IdentityAddr = "127.0.0.1:50061"
	}
	recruitmentMode := options.RecruitmentRouteMode
	if recruitmentMode == "" {
		recruitmentMode = "recruitment"
	}
	if recruitmentMode != "recruitment" {
		return nil, fmt.Errorf("unsupported recruitment route mode %q", recruitmentMode)
	}
	if recruitmentMode == "recruitment" && strings.TrimSpace(options.RecruitmentAddr) == "" {
		options.RecruitmentAddr = "127.0.0.1:50062"
	}
	interviewMode := options.InterviewRouteMode
	if interviewMode == "" {
		interviewMode = "interview"
	}
	if interviewMode != "interview" {
		return nil, fmt.Errorf("unsupported interview route mode %q", interviewMode)
	}
	if interviewMode == "interview" && strings.TrimSpace(options.InterviewAddr) == "" {
		options.InterviewAddr = "127.0.0.1:50063"
	}
	offerMode := options.OfferRouteMode
	if offerMode == "" {
		offerMode = "offer"
	}
	if offerMode != "offer" {
		return nil, fmt.Errorf("unsupported offer route mode %q", offerMode)
	}
	if offerMode == "offer" && strings.TrimSpace(options.OfferAddr) == "" {
		options.OfferAddr = "127.0.0.1:50064"
	}
	analyticsMode := options.AnalyticsRouteMode
	if analyticsMode == "" {
		analyticsMode = "analytics"
	}
	if analyticsMode != "analytics" {
		return nil, fmt.Errorf("unsupported analytics route mode %q", analyticsMode)
	}
	if analyticsMode == "analytics" && strings.TrimSpace(options.AnalyticsAddr) == "" {
		options.AnalyticsAddr = "127.0.0.1:50067"
	}
	token := grpcInternalToken()
	opts, tlsEnabled, err := dialOptions(token, options)
	if err != nil {
		return nil, err
	}
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
	interviewConn := conn
	interviewTarget := addr
	offerConn := conn
	offerTarget := addr
	analyticsConn := conn
	analyticsTarget := addr
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
	if interviewMode == "interview" {
		if options.InterviewAddr == "" {
			closeClientConns(conn, notificationConn, aiAgentConn, identityConn, recruitmentConn)
			return nil, fmt.Errorf("interview grpc addr is required when route mode is interview")
		}
		interviewConn, err = grpc.NewClient(options.InterviewAddr,
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
			closeClientConns(conn, notificationConn, aiAgentConn, identityConn, recruitmentConn)
			return nil, err
		}
		interviewTarget = options.InterviewAddr
	}
	if offerMode == "offer" {
		if options.OfferAddr == "" {
			closeClientConns(conn, notificationConn, aiAgentConn, identityConn, recruitmentConn, interviewConn)
			return nil, fmt.Errorf("offer grpc addr is required when route mode is offer")
		}
		offerConn, err = grpc.NewClient(options.OfferAddr,
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
			closeClientConns(conn, notificationConn, aiAgentConn, identityConn, recruitmentConn, interviewConn)
			return nil, err
		}
		offerTarget = options.OfferAddr
	}
	if analyticsMode == "analytics" {
		if options.AnalyticsAddr == "" {
			closeClientConns(conn, notificationConn, aiAgentConn, identityConn, recruitmentConn, interviewConn, offerConn)
			return nil, fmt.Errorf("analytics grpc addr is required when route mode is analytics")
		}
		analyticsConn, err = grpc.NewClient(options.AnalyticsAddr,
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
			closeClientConns(conn, notificationConn, aiAgentConn, identityConn, recruitmentConn, interviewConn, offerConn)
			return nil, err
		}
		analyticsTarget = options.AnalyticsAddr
	}
	baseAdminClient := pb.NewAdminServiceClient(conn)
	recruitmentAdminClient := baseAdminClient
	if recruitmentMode == "recruitment" {
		recruitmentAdminClient = newRecruitmentAdminClient(baseAdminClient, pb.NewAdminServiceClient(recruitmentConn))
	}
	identityAdminClient := recruitmentAdminClient
	if identityMode == "identity" {
		identityAdminClient = newIdentityAdminClient(recruitmentAdminClient, pb.NewAdminServiceClient(identityConn))
	}
	adminClient := identityAdminClient
	if analyticsMode == "analytics" {
		adminClient = newAnalyticsAdminClient(identityAdminClient, pb.NewAdminServiceClient(analyticsConn))
	}
	return &Clients{
		conn:                   conn,
		notificationConn:       notificationConn,
		aiAgentConn:            aiAgentConn,
		identityConn:           identityConn,
		recruitmentConn:        recruitmentConn,
		interviewConn:          interviewConn,
		offerConn:              offerConn,
		analyticsConn:          analyticsConn,
		NotificationRouteMode:  notificationMode,
		NotificationTargetAddr: notificationTarget,
		AIAgentRouteMode:       aiAgentMode,
		AIAgentTargetAddr:      aiAgentTarget,
		IdentityRouteMode:      identityMode,
		IdentityTargetAddr:     identityTarget,
		RecruitmentRouteMode:   recruitmentMode,
		RecruitmentTargetAddr:  recruitmentTarget,
		InterviewRouteMode:     interviewMode,
		InterviewTargetAddr:    interviewTarget,
		OfferRouteMode:         offerMode,
		OfferTargetAddr:        offerTarget,
		AnalyticsRouteMode:     analyticsMode,
		AnalyticsTargetAddr:    analyticsTarget,
		InternalTLSEnabled:     tlsEnabled,
		Auth:                   pb.NewAuthServiceClient(identityConn),
		Tenant:                 pb.NewPlatformTenantServiceClient(identityConn),
		Job:                    pb.NewJobServiceClient(recruitmentConn),
		Candidate:              pb.NewCandidateServiceClient(recruitmentConn),
		Application:            pb.NewApplicationServiceClient(recruitmentConn),
		ApplicationOwner:       pb.NewApplicationOwnerServiceClient(recruitmentConn),
		AI:                     pb.NewAIServiceClient(aiAgentConn),
		Notification:           pb.NewNotificationServiceClient(notificationConn),
		Interview:              pb.NewInterviewServiceClient(interviewConn),
		Offer:                  pb.NewOfferServiceClient(offerConn),
		Admin:                  adminClient,
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
		InterviewHealth:        healthpb.NewHealthClient(interviewConn),
		OfferHealth:            healthpb.NewHealthClient(offerConn),
		AnalyticsHealth:        healthpb.NewHealthClient(analyticsConn),
	}, nil
}

func dialOptions(token string, options ClientOptions) ([]grpc.DialOption, bool, error) {
	transportCredentials, tlsEnabled, err := clientTransportCredentials(options)
	if err != nil {
		return nil, false, err
	}
	return []grpc.DialOption{
		grpc.WithTransportCredentials(transportCredentials),
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
	}, tlsEnabled, nil
}

func clientTransportCredentials(options ClientOptions) (credentials.TransportCredentials, bool, error) {
	mode := strings.TrimSpace(strings.ToLower(options.GRPCInternalTLS))
	if mode == "" {
		mode = "optional"
	}
	caFile := strings.TrimSpace(options.GRPCTLSCAFile)
	switch mode {
	case "optional":
		if caFile == "" {
			return insecure.NewCredentials(), false, nil
		}
	case "required":
		if caFile == "" {
			return nil, false, fmt.Errorf("GRPC_TLS_CA_FILE is required when GRPC_INTERNAL_TLS=required")
		}
	default:
		return nil, false, fmt.Errorf("GRPC_INTERNAL_TLS must be optional or required")
	}
	creds, err := credentials.NewClientTLSFromFile(caFile, strings.TrimSpace(options.GRPCTLSServerName))
	if err != nil {
		return nil, false, fmt.Errorf("load gRPC TLS CA: %w", err)
	}
	return creds, true, nil
}

func (c *Clients) Close() error {
	if c.analyticsConn != nil && c.analyticsConn != c.conn && c.analyticsConn != c.notificationConn && c.analyticsConn != c.aiAgentConn && c.analyticsConn != c.identityConn && c.analyticsConn != c.recruitmentConn && c.analyticsConn != c.interviewConn && c.analyticsConn != c.offerConn {
		if err := c.analyticsConn.Close(); err != nil {
			closeClientConns(c.conn, c.notificationConn, c.aiAgentConn, c.identityConn, c.recruitmentConn, c.interviewConn, c.offerConn)
			return err
		}
	}
	if c.offerConn != nil && c.offerConn != c.conn && c.offerConn != c.notificationConn && c.offerConn != c.aiAgentConn && c.offerConn != c.identityConn && c.offerConn != c.recruitmentConn && c.offerConn != c.interviewConn {
		if err := c.offerConn.Close(); err != nil {
			closeClientConns(c.conn, c.notificationConn, c.aiAgentConn, c.identityConn, c.recruitmentConn, c.interviewConn)
			return err
		}
	}
	if c.interviewConn != nil && c.interviewConn != c.conn && c.interviewConn != c.notificationConn && c.interviewConn != c.aiAgentConn && c.interviewConn != c.identityConn && c.interviewConn != c.recruitmentConn {
		if err := c.interviewConn.Close(); err != nil {
			closeClientConns(c.conn, c.notificationConn, c.aiAgentConn, c.identityConn, c.recruitmentConn)
			return err
		}
	}
	if c.recruitmentConn != nil && c.recruitmentConn != c.conn && c.recruitmentConn != c.notificationConn && c.recruitmentConn != c.aiAgentConn && c.recruitmentConn != c.identityConn && c.recruitmentConn != c.interviewConn {
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
	if err := checkHealth(ctx, "primary", c.Health); err != nil {
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
	if c.interviewConn != nil && c.interviewConn != c.conn && c.interviewConn != c.notificationConn && c.interviewConn != c.aiAgentConn && c.interviewConn != c.identityConn && c.interviewConn != c.recruitmentConn && c.InterviewHealth != nil {
		if err := checkHealth(ctx, "interview", c.InterviewHealth); err != nil {
			return err
		}
	}
	if c.offerConn != nil && c.offerConn != c.conn && c.offerConn != c.notificationConn && c.offerConn != c.aiAgentConn && c.offerConn != c.identityConn && c.offerConn != c.recruitmentConn && c.offerConn != c.interviewConn && c.OfferHealth != nil {
		if err := checkHealth(ctx, "offer", c.OfferHealth); err != nil {
			return err
		}
	}
	if c.analyticsConn != nil && c.analyticsConn != c.conn && c.analyticsConn != c.notificationConn && c.analyticsConn != c.aiAgentConn && c.analyticsConn != c.identityConn && c.analyticsConn != c.recruitmentConn && c.analyticsConn != c.interviewConn && c.analyticsConn != c.offerConn && c.AnalyticsHealth != nil {
		if err := checkHealth(ctx, "analytics", c.AnalyticsHealth); err != nil {
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
	if tenantID, ok := ctx.Value(contextkeys.TenantID).(int64); ok && tenantID > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-authenticated-tenant-id", strconv.FormatInt(tenantID, 10))
	}
	if membershipID, ok := ctx.Value(contextkeys.MembershipID).(int64); ok && membershipID > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-authenticated-membership-id", strconv.FormatInt(membershipID, 10))
	}
	if clientApp, ok := ctx.Value(contextkeys.ClientApp).(string); ok && clientApp != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-authenticated-client-app", clientApp)
	}
	if traceparent, ok := ctx.Value(contextkeys.Traceparent).(string); ok && traceparent != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "traceparent", traceparent)
	}
	if traceID, ok := ctx.Value(contextkeys.TraceID).(string); ok && traceID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-trace-id", traceID)
	}
	if spanID, ok := ctx.Value(contextkeys.SpanID).(string); ok && spanID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-span-id", spanID)
	}
	return ctx
}
