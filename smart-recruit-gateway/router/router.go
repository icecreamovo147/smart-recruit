package router

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"smart-recruit-gateway/config"
	_ "smart-recruit-gateway/docs"
	"smart-recruit-gateway/handler"
	"smart-recruit-gateway/handler/candidate"
	"smart-recruit-gateway/handler/hr"
	"smart-recruit-gateway/middleware"
	"smart-recruit-gateway/pkg/authz"
	"smart-recruit-gateway/pkg/contextkeys"
	"smart-recruit-gateway/pkg/logger"
	"smart-recruit-gateway/pkg/observability"
	"smart-recruit-gateway/pkg/redisclient"
	"smart-recruit-gateway/rpc"
	pb "smart-recruit-proto/recruitment/pb"
)

func Setup(cfg config.Config, clients *rpc.Clients, rdb *redis.Client) (*gin.Engine, *middleware.LimiterRegistry) {
	// ── Wire authorization audit logger ──────────────────────────────────
	// Audit events are buffered and flushed asynchronously with fail-safe
	// timeout to avoid blocking the request path while minimizing data loss.
	var auditDropped int64
	auditCh := make(chan middleware.AuthAuditEntry, 4096)
	middleware.SetAuditLogger(func(entry middleware.AuthAuditEntry) {
		select {
		case auditCh <- entry:
		default:
			// Buffer full: blocking send with 50ms timeout
			select {
			case auditCh <- entry:
			case <-time.After(50 * time.Millisecond):
				dropped := atomic.AddInt64(&auditDropped, 1)
				logger.L().Warn("audit event dropped (buffer full)",
					zap.Int64("dropped_total", dropped),
					zap.String("decision", entry.Decision),
					zap.String("permission", entry.PermissionKey),
					zap.String("request_id", entry.RequestID),
				)
			}
		}
	})
	go func() {
		for entry := range auditCh {
			resp, err := clients.Auth.RecordAuthDecision(context.Background(), &pb.AuthAuditRequest{
				ActorUserId:   entry.ActorUserID,
				ActorRoles:    entry.ActorRoles,
				PermissionKey: entry.PermissionKey,
				ResourceType:  entry.ResourceType,
				ResourceId:    entry.ResourceID,
				Decision:      entry.Decision,
				Reason:        entry.Reason,
				RequestId:     entry.RequestID,
				ClientIp:      entry.ClientIP,
			})
			if err != nil {
				logger.L().Error("audit record gRPC call failed",
					zap.Error(err),
					zap.Int64("actor", entry.ActorUserID),
					zap.String("permission", entry.PermissionKey),
				)
			} else if resp != nil && resp.Code != 0 {
				logger.L().Warn("audit record returned non-zero code",
					zap.String("msg", resp.Msg),
					zap.Int64("actor", entry.ActorUserID),
				)
			}
		}
	}()

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Metrics(observability.DefaultMetrics), middleware.AccessLog(), middleware.Recovery(), middleware.SecurityHeaders(), middleware.CSP(), middleware.CORSWrapper())
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/livez", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		deps := gin.H{"grpc": "ok", "redis": "ok"}
		ready := true
		if err := clients.Ready(ctx); err != nil {
			deps["grpc"] = err.Error()
			ready = false
		}
		if err := redisclient.Ping(ctx, rdb); err != nil {
			deps["redis"] = err.Error()
			ready = false
		}
		status := "ok"
		code := 200
		if !ready {
			status = "not_ready"
			code = 503
		}
		c.JSON(code, gin.H{"status": status, "dependencies": deps})
	})
	r.GET("/metrics", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(200, observability.DefaultMetrics.Prometheus())
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authHandler := handler.NewAuthHandler(clients, cfg.AuthCookieName, cfg.CandidateCookie, cfg.HRCookie, cfg.InterviewerCookie, cfg.AuthCookieSecure, cfg.JWTSecret, rdb)
	platformTenantHandler := handler.NewPlatformTenantHandler(clients)
	platformUserHandler := handler.NewPlatformUserHandler(clients)
	platformAIHandler := handler.NewPlatformAIHandler(clients)
	publicHandler := handler.NewPublicHandler(clients)
	hrJobHandler := hr.NewJobHandler(clients)
	hrApplicationHandler := hr.NewApplicationHandler(clients)
	hrAIHandler := hr.NewAIHandler(clients)
	llmConfigHandler := hr.NewLlmConfigHandler(clients)
	hrInterviewHandler := hr.NewInterviewHandler(clients)
	hrOfferHandler := hr.NewOfferHandler(clients)
	hrCapabilityHandler := hr.NewCapabilityHandler(clients)
	embeddingConfigHandler := hr.NewEmbeddingConfigHandler(clients)
	agentSkillHandler := hr.NewAgentSkillHandler(clients)
	candidateOfferHandler := candidate.NewOfferHandler(clients)
	candidateInterviewHandler := candidate.NewInterviewHandler(clients)
	profileHandler := candidate.NewProfileHandler(clients)
	resumeHandler := candidate.NewResumeHandler(clients)
	applyHandler := candidate.NewApplyHandler(clients)
	candidateAIHandler := candidate.NewAIHandler(clients)
	notificationHandler := handler.NewNotificationHandler(clients, rdb)
	dashboardHandler := hr.NewDashboardHandler(clients)
	analyticsHandler := hr.NewAnalyticsHandler(clients)
	collaborationHandler := hr.NewCollaborationHandler(clients)
	recruitingIntelligenceHandler := hr.NewRecruitingIntelligenceHandler(clients)
	candidateBillingHandler := handler.NewBillingHandler(clients, pb.BillingOwnerType_BILLING_OWNER_TYPE_USER)
	tenantBillingHandler := handler.NewBillingHandler(clients, pb.BillingOwnerType_BILLING_OWNER_TYPE_TENANT)
	alipayWebhookHandler := handler.NewAlipayWebhookHandler(clients)

	normalTimeout := middleware.Timeout(10 * time.Second)
	uploadTimeout := middleware.Timeout(20 * time.Second)
	aiTimeout := middleware.Timeout(45 * time.Second)
	mcpTimeout := middleware.Timeout(120 * time.Second)

	limiters := middleware.NewLimiterRegistry(
		cfg.RateLimit.AuthRPS, cfg.RateLimit.AuthBurst,
		cfg.RateLimit.AIRPS, cfg.RateLimit.AIBurst,
		cfg.RateLimit.GeneralRPS, cfg.RateLimit.GeneralBurst,
	)

	authLimit := middleware.ResilientRateLimit(rdb, "auth", cfg.RateLimit.AuthRPS, cfg.RateLimit.AuthBurst, limiters.Auth)
	aiLimit := middleware.ResilientRateLimit(rdb, "ai", cfg.RateLimit.AIRPS, cfg.RateLimit.AIBurst, limiters.AI)
	generalLimit := middleware.ResilientRateLimit(rdb, "general", cfg.RateLimit.GeneralRPS, cfg.RateLimit.GeneralBurst, limiters.General)
	candidateAIQuota := middleware.AIDailyQuota(rdb, "candidate", cfg.RateLimit.AIQuotaCandidateDaily)
	hrAIQuota := middleware.AIDailyQuota(rdb, "hr", cfg.RateLimit.AIQuotaHRDaily)
	resumePresignQuota := middleware.ResumePresignQuota(rdb, cfg.RateLimit.ResumePresignHourlyLimit, cfg.RateLimit.ResumePresignDailyLimit)
	resumeConfirmQuota := middleware.ResumeConfirmQuota(rdb, cfg.RateLimit.ResumeConfirmHourlyLimit, cfg.RateLimit.ResumeConfirmDailyLimit)
	riskBlock := middleware.RiskBlock(rdb)

	v1 := r.Group("/api/v1", generalLimit)
	bodyAuth := middleware.MaxBodyBytes(4 << 10)
	bodyProfile := middleware.MaxBodyBytes(16 << 10)
	bodyAI := middleware.MaxBodyBytes(64 << 10)
	bodyJob := middleware.MaxBodyBytes(64 << 10)
	bodyAdmin := middleware.MaxBodyBytes(128 << 10) // 128KB for admin config (prompts, agents, etc.)

	// ── Public auth endpoints ──────────────────────────────────────────
	v1.POST("/auth/register", normalTimeout, authLimit, bodyAuth, authHandler.Register)
	v1.POST("/auth/register/validate-invite-code", normalTimeout, authHandler.ValidateInviteCode)
	v1.POST("/auth/login", normalTimeout, authLimit, bodyAuth, authHandler.Login)
	v1.POST("/auth/logout", normalTimeout, middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.InterviewerCookie, cfg.AuthCookieName, rdb), authHandler.Logout)
	v1.POST("/auth/refresh", normalTimeout, authHandler.RefreshToken)
	v1.POST("/auth/tenant/switch", normalTimeout, authLimit, bodyAuth, authHandler.SwitchTenant)
	v1.GET("/auth/me", normalTimeout, middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.InterviewerCookie, cfg.AuthCookieName, rdb), authHandler.Me)
	v1.PUT("/auth/email", normalTimeout, bodyAuth, middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.InterviewerCookie, cfg.AuthCookieName, rdb), authHandler.UpdateEmail)
	v1.GET("/jobs", normalTimeout, publicHandler.ListJobs)
	v1.GET("/jobs/:job_id", normalTimeout, publicHandler.JobDetail)
	// Public taxonomy for candidate job-board filters (active departments/locations).
	v1.GET("/job-options", normalTimeout, publicHandler.JobOptions)
	v1.POST("/public/billing/webhooks/alipay", normalTimeout, middleware.MaxBodyBytes(64<<10), alipayWebhookHandler.Notify)

	// ── Authenticated middleware (with token_version validation via Redis) ─
	jwtAuth := middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.InterviewerCookie, cfg.AuthCookieName, rdb)
	currentPrincipal := middleware.ValidateCurrentPrincipal(func(ctx context.Context, userID int64) (*middleware.CurrentPrincipal, error) {
		tenantID, _ := ctx.Value(contextkeys.TenantID).(int64)
		membershipID, _ := ctx.Value(contextkeys.MembershipID).(int64)
		clientApp, _ := ctx.Value(contextkeys.ClientApp).(string)
		resp, err := clients.Auth.GetPrincipal(ctx, &pb.GetPrincipalRequest{
			UserId: userID, TenantId: tenantID, MembershipId: membershipID, ClientApp: clientApp,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Code != 0 {
			return nil, nil
		}
		return &middleware.CurrentPrincipal{
			UserID:       resp.UserId,
			Username:     resp.Username,
			Role:         resp.Role,
			AccountType:  resp.AccountType,
			Roles:        resp.Roles,
			Permissions:  resp.Permissions,
			TokenVersion: resp.TokenVersion,
			TenantID:     resp.TenantId,
			MembershipID: resp.MembershipId,
			ClientApp:    resp.ClientApp,
		}, nil
	})

	platformGroup := v1.Group("/platform", jwtAuth, currentPrincipal, middleware.RequirePlatformApp())
	platformGroup.GET("/dashboard", normalTimeout, middleware.RequirePermission(authz.PermPlatformDashboardRead), platformTenantHandler.Dashboard)
	platformGroup.GET("/tenants", normalTimeout, middleware.RequirePermission(authz.PermPlatformTenantRead), platformTenantHandler.List)
	platformGroup.POST("/tenants", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformTenantManage), platformTenantHandler.Create)
	platformGroup.GET("/tenants/:tenant_id", normalTimeout, middleware.RequirePermission(authz.PermPlatformTenantRead), platformTenantHandler.Get)
	platformGroup.PATCH("/tenants/:tenant_id/status", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformTenantManage), platformTenantHandler.UpdateStatus)
	platformGroup.GET("/tenants/:tenant_id/memberships", normalTimeout, middleware.RequirePermission(authz.PermPlatformTenantRead), platformTenantHandler.ListMemberships)
	platformGroup.PATCH("/tenants/:tenant_id/memberships/:membership_id/status", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformMemberManage), platformTenantHandler.UpdateMembershipStatus)
	platformGroup.GET("/audit-logs", normalTimeout, middleware.RequirePermission(authz.PermPlatformAuditRead), platformTenantHandler.AuditLogs)
	platformGroup.GET("/plans", normalTimeout, middleware.RequirePermission(authz.PermPlatformPlanRead), platformTenantHandler.ListPlans)
	platformGroup.POST("/plans/:plan_id/versions", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformPlanManage), platformTenantHandler.SavePlanVersion)
	platformGroup.POST("/plans/:plan_id/versions/:version_id/publish", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformPlanPublish), platformTenantHandler.PublishPlanVersion)
	platformGroup.GET("/billing/products", normalTimeout, middleware.RequirePermission(authz.PermPlatformPlanRead), tenantBillingHandler.AdminCatalog)
	platformGroup.POST("/billing/prices", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformPlanManage), tenantBillingHandler.SavePrice)
	platformGroup.GET("/billing/rates", normalTimeout, middleware.RequirePermission(authz.PermPlatformPlanRead), tenantBillingHandler.AdminRateCards)
	platformGroup.POST("/billing/rates", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformPlanManage), tenantBillingHandler.SaveRateCard)
	platformGroup.GET("/tenants/:tenant_id/subscription", normalTimeout, middleware.RequirePermission(authz.PermPlatformTenantRead), platformTenantHandler.GetSubscription)
	platformGroup.PUT("/tenants/:tenant_id/subscription", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformSubscriptionManage), platformTenantHandler.UpdateSubscription)
	platformGroup.PUT("/tenants/:tenant_id/entitlement-override", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformPlanManage), platformTenantHandler.UpdateEntitlementOverride)
	platformGroup.GET("/tenants/:tenant_id/usage", normalTimeout, middleware.RequirePermission(authz.PermPlatformUsageRead), platformTenantHandler.GetUsage)
	platformGroup.GET("/quota-alerts", normalTimeout, middleware.RequirePermission(authz.PermPlatformAlertRead), platformTenantHandler.ListAlerts)
	platformGroup.PATCH("/quota-alerts/:alert_id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAlertManage), platformTenantHandler.UpdateAlert)
	platformGroup.GET("/users", normalTimeout, middleware.RequirePermission(authz.PermPlatformUserManage), platformUserHandler.List)
	platformGroup.POST("/users", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformUserManage), platformUserHandler.Create)
	platformGroup.PATCH("/users/:user_id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformUserManage), platformUserHandler.Update)
	platformAIGroup := platformGroup.Group("/ai")
	platformAIGroup.GET("/capabilities", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIReleaseRead), platformAIHandler.ListCapabilities)
	platformAIGroup.GET("/capabilities/:capability_id/versions", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIReleaseRead), platformAIHandler.ListCapabilityVersions)
	platformAIGroup.POST("/capabilities/:capability_id/versions", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIReleaseManage), platformAIHandler.CreateCapabilityDraft)
	platformAIGroup.PUT("/capability-versions/:version_id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIReleaseManage), platformAIHandler.UpdateCapabilityDraft)
	platformAIGroup.POST("/capability-versions/:version_id/publish", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIReleasePublish), platformAIHandler.PublishCapabilityVersion)
	platformAIGroup.GET("/audit-logs", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsRead), platformAIHandler.AuditLogs)

	// ── Candidate routes ───────────────────────────────────────────────
	// Each candidate route declares the required permission explicitly.
	candidateGroup := v1.Group("/candidate", jwtAuth, currentPrincipal, middleware.RequireAnyRole(authz.RoleCandidate))
	candidateGroup.GET("/profile", normalTimeout, middleware.RequirePermission(authz.PermCandidateProfileManage), profileHandler.Get)
	candidateGroup.PUT("/profile", normalTimeout, bodyProfile, middleware.RequirePermission(authz.PermCandidateProfileManage), profileHandler.Update)
	candidateGroup.GET("/resume", normalTimeout, middleware.RequirePermission(authz.PermCandidateResumeManage), resumeHandler.Get)
	candidateGroup.POST("/resume/presign", riskBlock, resumePresignQuota, normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermCandidateResumeManage), resumeHandler.Presign)
	candidateGroup.POST("/resume/confirm", riskBlock, resumeConfirmQuota, uploadTimeout, bodyProfile, middleware.RequirePermission(authz.PermCandidateResumeManage), resumeHandler.Confirm)
	candidateGroup.POST("/applications", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermCandidateApplicationManage), applyHandler.Apply)
	candidateGroup.GET("/applications", normalTimeout, middleware.RequirePermission(authz.PermCandidateApplicationManage), applyHandler.Mine)
	candidateGroup.GET("/interviews", normalTimeout, middleware.RequirePermission(authz.PermCandidateApplicationManage), candidateInterviewHandler.List)
	candidateGroup.GET("/offers", normalTimeout, middleware.RequirePermission(authz.PermOfferDecisionManage), candidateOfferHandler.ListMyOffers)
	candidateGroup.GET("/offers/:offer_id", normalTimeout, middleware.RequirePermission(authz.PermOfferDecisionManage), candidateOfferHandler.Get)
	candidateGroup.POST("/offers/:offer_id/accept", normalTimeout, middleware.RequirePermission(authz.PermOfferDecisionManage), candidateOfferHandler.Accept)
	candidateGroup.POST("/offers/:offer_id/reject", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermOfferDecisionManage), candidateOfferHandler.Reject)
	candidateGroup.GET("/notifications", normalTimeout, middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.List)
	candidateGroup.GET("/notifications/unread-count", normalTimeout, middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.UnreadCount)
	candidateGroup.GET("/notifications/summary", normalTimeout, middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.Summary)
	candidateGroup.GET("/notifications/stream", middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.Stream)
	candidateGroup.PATCH("/notifications/:notification_id/read", normalTimeout, middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.MarkRead)
	candidateGroup.PATCH("/notifications/read-all", normalTimeout, middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.MarkAllRead)
	candidateGroup.GET("/ai/models", normalTimeout, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.ListAvailableModels)
	candidateGroup.GET("/ai/sessions", normalTimeout, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.ListSessions)
	candidateGroup.POST("/ai/sessions", normalTimeout, bodyProfile, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.CreateSession)
	candidateGroup.GET("/ai/sessions/:session_id/messages", normalTimeout, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.SessionMessages)
	candidateGroup.PUT("/ai/sessions/:session_id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.UpdateSession)
	candidateGroup.DELETE("/ai/sessions/:session_id", normalTimeout, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.DeleteSession)
	candidateGroup.POST("/ai/chat", riskBlock, aiLimit, candidateAIQuota, aiTimeout, bodyAI, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.Chat)
	candidateGroup.POST("/ai/chat/stream", riskBlock, aiLimit, candidateAIQuota, bodyAI, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.ChatStream)
	candidateGroup.GET("/billing/catalog", normalTimeout, candidateBillingHandler.Catalog)
	candidateGroup.GET("/billing/subscription", normalTimeout, candidateBillingHandler.Account)
	candidateGroup.GET("/billing/credits", normalTimeout, candidateBillingHandler.Account)
	candidateGroup.GET("/billing/orders", normalTimeout, candidateBillingHandler.Orders)
	candidateGroup.POST("/billing/orders", normalTimeout, bodyAuth, candidateBillingHandler.CreateOrder)
	candidateGroup.POST("/billing/orders/:order_no/pay", normalTimeout, bodyAuth, candidateBillingHandler.Pay)
	candidateGroup.POST("/billing/orders/:order_no/refund", normalTimeout, bodyAuth, candidateBillingHandler.Refund)

	// ── Staff routes (formerly /hr) ────────────────────────────────────
	// Base group: any staff role (recruiter, recruiting_admin, system_admin, interviewer).
	staffGroup := v1.Group("/hr", jwtAuth, currentPrincipal, middleware.RequireActiveTenant(), middleware.RequireAnyRole(authz.StaffRoles()...))
	staffGroup.GET("/billing/catalog", normalTimeout, tenantBillingHandler.Catalog)
	staffGroup.GET("/billing/subscription", normalTimeout, tenantBillingHandler.Account)
	staffGroup.GET("/billing/credits", normalTimeout, tenantBillingHandler.Account)
	staffGroup.GET("/billing/orders", normalTimeout, middleware.RequirePermission(authz.PermBillingManage), tenantBillingHandler.Orders)
	staffGroup.POST("/billing/orders", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermBillingManage), tenantBillingHandler.CreateOrder)
	staffGroup.POST("/billing/orders/:order_no/pay", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermBillingManage), tenantBillingHandler.Pay)
	staffGroup.POST("/billing/orders/:order_no/refund", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermBillingManage), tenantBillingHandler.Refund)

	// Job management — requires explicit job permissions
	staffGroup.GET("/job-options", normalTimeout, middleware.RequirePermission(authz.PermJobRead), hrJobHandler.JobOptions)
	staffGroup.POST("/jobs", normalTimeout, bodyJob, middleware.RequirePermission(authz.PermJobCreate), hrJobHandler.Create)
	staffGroup.PUT("/jobs/:job_id", normalTimeout, bodyJob, middleware.RequirePermission(authz.PermJobUpdate), hrJobHandler.Update)
	staffGroup.PATCH("/jobs/:job_id/offline", normalTimeout, middleware.RequirePermission(authz.PermJobPublish), hrJobHandler.Offline)
	staffGroup.PATCH("/jobs/:job_id/online", normalTimeout, middleware.RequirePermission(authz.PermJobPublish), hrJobHandler.Online)
	staffGroup.GET("/jobs", normalTimeout, middleware.RequirePermission(authz.PermJobRead), hrJobHandler.List)
	staffGroup.GET("/jobs/:job_id/applications", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), hrApplicationHandler.ListByJob)
	staffGroup.PATCH("/applications/:application_id/status", normalTimeout, middleware.RequirePermission(authz.PermApplicationStatusUpdate), hrApplicationHandler.UpdateStatus)
	staffGroup.GET("/applications/:id/transitions", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), hrApplicationHandler.ListTransitions)
	staffGroup.GET("/applications/:id/resume-profile", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), recruitingIntelligenceHandler.GetResumeProfileByApplication)
	staffGroup.GET("/resume-profiles", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), recruitingIntelligenceHandler.GetResumeProfile)
	staffGroup.POST("/resume-profiles/parse", riskBlock, aiLimit, hrAIQuota, aiTimeout, bodyAuth, middleware.RequirePermission(authz.PermAIHRUse), recruitingIntelligenceHandler.ParseResumeProfile)
	staffGroup.POST("/applications/:application_id/match-evaluations", riskBlock, aiLimit, hrAIQuota, aiTimeout, bodyAuth, middleware.RequirePermission(authz.PermAIHRUse), recruitingIntelligenceHandler.EvaluateCandidateMatch)
	staffGroup.GET("/applications/:id/match-evaluation", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), recruitingIntelligenceHandler.GetCandidateMatchEvaluation)
	staffGroup.GET("/jobs/:job_id/candidate-comparison", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), recruitingIntelligenceHandler.CompareCandidatesForJob)

	// Interview management — requires interview permissions
	staffGroup.GET("/applications/:id/interviews", normalTimeout, middleware.RequirePermission(authz.PermInterviewRead), hrInterviewHandler.ListByApplication)
	staffGroup.POST("/interviews", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermInterviewSchedule), hrInterviewHandler.Schedule)
	staffGroup.PUT("/interviews/:interview_id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermInterviewSchedule), hrInterviewHandler.Update)
	staffGroup.PATCH("/interviews/:interview_id/cancel", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermInterviewSchedule), hrInterviewHandler.Cancel)
	staffGroup.POST("/applications/:application_id/cancel-interviews", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermInterviewSchedule), hrInterviewHandler.BatchCancelInterviews)
	staffGroup.GET("/interviews/:interview_id", normalTimeout, middleware.RequireAnyPermission(authz.PermInterviewRead, authz.PermInterviewSchedule), hrInterviewHandler.Get)
	staffGroup.GET("/interviewers", normalTimeout, middleware.RequirePermission(authz.PermInterviewSchedule), hrInterviewHandler.ListInterviewers)

	// Interviewer task routes (also under /hr for staff access)
	staffGroup.GET("/my-interviews", normalTimeout, middleware.RequirePermission(authz.PermInterviewRead), hrInterviewHandler.ListMy)
	staffGroup.POST("/interviews/:interview_id/feedback", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermInterviewFeedback), hrInterviewHandler.SubmitFeedback)
	staffGroup.GET("/interviews/:interview_id/feedback", normalTimeout, middleware.RequirePermission(authz.PermInterviewFeedback), hrInterviewHandler.GetFeedback)

	// Offer management — requires offer permissions
	staffGroup.POST("/offers", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermOfferManage), hrOfferHandler.Create)
	staffGroup.PUT("/offers/:offer_id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermOfferManage), hrOfferHandler.Update)
	staffGroup.GET("/offers/:offer_id", normalTimeout, middleware.RequirePermission(authz.PermOfferRead), hrOfferHandler.Get)
	staffGroup.GET("/applications/:id/offers", normalTimeout, middleware.RequirePermission(authz.PermOfferRead), hrOfferHandler.ListByApplication)
	staffGroup.POST("/offers/:offer_id/send", normalTimeout, middleware.RequirePermission(authz.PermOfferSend), hrOfferHandler.Send)
	staffGroup.POST("/offers/:offer_id/withdraw", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermOfferManage), hrOfferHandler.Withdraw)
	staffGroup.GET("/offers/:offer_id/events", normalTimeout, middleware.RequirePermission(authz.PermOfferRead), hrOfferHandler.ListEvents)

	// AI — requires HR AI permission
	staffGroup.GET("/ai/sessions", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.ListSessions)
	staffGroup.POST("/ai/sessions", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.CreateSession)
	staffGroup.GET("/ai/sessions/:session_id/messages", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.SessionMessages)
	staffGroup.PUT("/ai/sessions/:session_id/context-model", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.PreviewChatContext)
	staffGroup.GET("/ai/sessions/:session_id/tool-traces", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.GetToolTraces)
	staffGroup.GET("/ai/sessions/:session_id/agent-runs", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.GetAgentRuns)
	staffGroup.GET("/ai/sessions/:session_id/active-run", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.GetActiveAgentRun)
	staffGroup.PUT("/ai/sessions/:session_id", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.UpdateSession)
	staffGroup.DELETE("/ai/sessions/:session_id", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.DeleteSession)
	staffGroup.GET("/ai/models", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), llmConfigHandler.ListAvailableModels)
	staffGroup.GET("/ai/skill-capabilities", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.ListSkillCapabilities)
	staffGroup.GET("/agent-skills/available", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), agentSkillHandler.ListAvailable)
	staffGroup.GET("/capabilities", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrCapabilityHandler.List)
	staffGroup.GET("/capabilities/:id", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrCapabilityHandler.Get)
	staffGroup.POST("/capabilities/from-template", normalTimeout, bodyAdmin, middleware.RequireAnyRole(authz.RoleRecruitingAdmin, authz.RoleSystemAdmin), hrCapabilityHandler.CreateFromTemplate)
	staffGroup.POST("/ai/application-analysis-sessions", riskBlock, aiLimit, hrAIQuota, aiTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.CreateApplicationAnalysisSession)
	staffGroup.POST("/ai/chat", riskBlock, aiLimit, hrAIQuota, aiTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.Chat)
	staffGroup.POST("/ai/chat/stream", riskBlock, aiLimit, hrAIQuota, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.ChatStream)
	// Durable resumable HR Agent runs: thin wrappers over AI Agent lifecycle RPCs.
	// Create/confirm use AI quota/risk middleware; SSE has no short timeout; cancel is explicit.
	staffGroup.POST("/ai/runs", riskBlock, aiLimit, hrAIQuota, aiTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.CreateAgentRun)
	staffGroup.GET("/ai/runs/:run_id", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.GetAgentRun)
	staffGroup.GET("/ai/runs/:run_id/events", middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.SubscribeAgentRunEvents)
	staffGroup.POST("/ai/runs/:run_id/cancel", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.CancelAgentRun)
	staffGroup.POST("/ai/runs/:run_id/confirm", riskBlock, aiLimit, hrAIQuota, aiTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.ConfirmAgentRun)
	staffGroup.POST("/ai/analyze-application", riskBlock, aiLimit, hrAIQuota, aiTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.AnalyzeApplication)
	staffGroup.GET("/ai/history", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.History)

	// Notifications — requires notification read permission
	staffGroup.GET("/notifications", normalTimeout, middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.List)
	staffGroup.GET("/notifications/unread-count", normalTimeout, middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.UnreadCount)
	staffGroup.GET("/notifications/summary", normalTimeout, middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.Summary)
	staffGroup.GET("/notifications/stream", middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.Stream)
	staffGroup.PATCH("/notifications/:notification_id/read", normalTimeout, middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.MarkRead)
	staffGroup.PATCH("/notifications/read-all", normalTimeout, middleware.RequirePermission(authz.PermNotificationRead), notificationHandler.MarkAllRead)

	// Dashboard — requires job.read or application.read
	staffGroup.GET("/dashboard/summary", normalTimeout, middleware.RequireAnyPermission(authz.PermJobRead, authz.PermApplicationRead), dashboardHandler.Summary)

	// ── Phase 6: Analytics & Reporting routes ─────────────────────
	// Dashboard report (scoped KPI) — requires job.read or application.read
	staffGroup.GET("/analytics/dashboard", normalTimeout, middleware.RequireAnyPermission(authz.PermJobRead, authz.PermApplicationRead), analyticsHandler.DashboardReport)

	// Funnel report — requires application.read
	staffGroup.GET("/analytics/funnel", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), analyticsHandler.FunnelReport)

	// Time-in-stage report — requires application.read
	staffGroup.GET("/analytics/time-in-stage", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), analyticsHandler.TimeInStageReport)

	// Interview & Offer metrics — requires application.read
	staffGroup.GET("/analytics/metrics", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), analyticsHandler.InterviewOfferMetrics)

	// Auth audit logs (security audit) — requires audit.security.read
	staffGroup.GET("/admin/auth-audit-logs", normalTimeout, middleware.RequirePermission(authz.PermAuditSecurityRead), analyticsHandler.AuthAuditLogs)

	// ── Collaboration routes ──────────────────────────────────────────
	// Candidate workspace
	staffGroup.GET("/candidates/:candidate_user_id/workspace", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), collaborationHandler.GetCandidateWorkspace)

	// Notes
	staffGroup.POST("/notes", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermCollaborationNoteCreate), collaborationHandler.CreateNote)
	staffGroup.GET("/notes", normalTimeout, middleware.RequirePermission(authz.PermCollaborationNoteRead), collaborationHandler.ListNotes)

	// Tags
	staffGroup.GET("/tags", normalTimeout, middleware.RequirePermission(authz.PermCollaborationTagManage), collaborationHandler.ListTags)
	staffGroup.POST("/tags", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermCollaborationTagManage), collaborationHandler.CreateTag)
	staffGroup.POST("/tags/assign", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermCollaborationTagManage), collaborationHandler.AssignTag)
	staffGroup.POST("/tags/unassign", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermCollaborationTagManage), collaborationHandler.UnassignTag)
	staffGroup.GET("/candidates/:candidate_user_id/tags", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), collaborationHandler.ListCandidateTags)

	// Follow-up tasks
	staffGroup.POST("/follow-up-tasks", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermCollaborationTaskManage), collaborationHandler.CreateFollowUpTask)
	staffGroup.GET("/follow-up-tasks", normalTimeout, middleware.RequirePermission(authz.PermCollaborationTaskManage), collaborationHandler.ListFollowUpTasks)
	staffGroup.PATCH("/follow-up-tasks/:task_id/complete", normalTimeout, middleware.RequirePermission(authz.PermCollaborationTaskManage), collaborationHandler.CompleteFollowUpTask)
	// Timeline
	staffGroup.GET("/candidates/:candidate_user_id/timeline", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), collaborationHandler.ListTimelineEvents)

	// ── Admin routes (/hr/admin) ───────────────────────────────────────
	// Admin routes require explicit admin permissions (not role hierarchy).
	adminHandler := hr.NewAdminHandler(clients)
	promptHandler := hr.NewPromptHandler(clients)
	adminGroup := staffGroup.Group("/admin")

	// Invite codes
	adminGroup.POST("/invite-codes", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminInviteManage), adminHandler.CreateInviteCode)
	adminGroup.GET("/invite-codes", normalTimeout, middleware.RequirePermission(authz.PermAdminInviteManage), adminHandler.ListInviteCodes)
	adminGroup.PATCH("/invite-codes/:id/extend", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminInviteManage), adminHandler.ExtendInviteCode)
	adminGroup.PATCH("/invite-codes/:id/revoke", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminInviteManage), adminHandler.RevokeInviteCode)
	adminGroup.PATCH("/invite-codes/:id/reactivate", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminInviteManage), adminHandler.ReactivateInviteCode)

	// Department taxonomy
	adminGroup.GET("/departments", normalTimeout, middleware.RequirePermission(authz.PermAdminDepartmentManage), adminHandler.ListDepartments)
	adminGroup.POST("/departments", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminDepartmentManage), adminHandler.CreateDepartment)
	adminGroup.PUT("/departments/:id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminDepartmentManage), adminHandler.UpdateDepartment)
	adminGroup.PATCH("/departments/:id/status", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminDepartmentManage), adminHandler.UpdateDepartmentStatus)
	adminGroup.DELETE("/departments/:id", normalTimeout, middleware.RequirePermission(authz.PermAdminDepartmentManage), adminHandler.DeleteDepartment)

	// Job location taxonomy
	adminGroup.GET("/locations", normalTimeout, middleware.RequirePermission(authz.PermAdminLocationManage), adminHandler.ListJobLocations)
	adminGroup.POST("/locations", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminLocationManage), adminHandler.CreateJobLocation)
	adminGroup.PUT("/locations/:id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminLocationManage), adminHandler.UpdateJobLocation)
	adminGroup.PATCH("/locations/:id/status", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminLocationManage), adminHandler.UpdateJobLocationStatus)
	adminGroup.DELETE("/locations/:id", normalTimeout, middleware.RequirePermission(authz.PermAdminLocationManage), adminHandler.DeleteJobLocation)

	// Department location config
	adminGroup.GET("/departments/location-map", normalTimeout, middleware.RequirePermission(authz.PermAdminDepartmentManage), adminHandler.ListDepartmentsLocationMap)
	adminGroup.GET("/departments/:id/locations", normalTimeout, middleware.RequirePermission(authz.PermAdminDepartmentManage), adminHandler.GetDepartmentLocationConfig)
	adminGroup.PUT("/departments/:id/locations", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminDepartmentManage), adminHandler.UpdateDepartmentLocationConfig)

	// Usage audit
	adminGroup.GET("/third-party-usage-logs", normalTimeout, middleware.RequirePermission(authz.PermAuditUsageRead), adminHandler.ListUsageLogs)

	// P1-003: AI usage statistics & trend
	adminGroup.GET("/usage-stats", normalTimeout, middleware.RequirePermission(authz.PermAuditUsageRead), adminHandler.GetUsageStats)
	adminGroup.GET("/usage-trend", normalTimeout, middleware.RequirePermission(authz.PermAuditUsageRead), adminHandler.GetUsageTrend)
	// RBAC role & permission management
	adminGroup.GET("/roles", normalTimeout, middleware.RequirePermission(authz.PermAdminRoleManage), adminHandler.ListRoles)
	adminGroup.GET("/permissions", normalTimeout, middleware.RequirePermission(authz.PermAdminRoleManage), adminHandler.ListPermissions)
	adminGroup.GET("/users/:user_id/roles", normalTimeout, middleware.RequirePermission(authz.PermAdminRoleManage), adminHandler.GetUserRoles)
	adminGroup.POST("/users/:user_id/roles/assign", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminRoleManage), adminHandler.AssignUserRole)
	adminGroup.POST("/users/:user_id/roles/revoke", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminRoleManage), adminHandler.RevokeUserRole)
	adminGroup.POST("/users/:user_id/data-scopes", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAdminUserManage), adminHandler.AssignDataScope)
	adminGroup.DELETE("/data-scopes/:scope_id", normalTimeout, middleware.RequirePermission(authz.PermAdminUserManage), adminHandler.RevokeDataScope)
	adminGroup.GET("/staff-users", normalTimeout, middleware.RequirePermission(authz.PermAdminUserManage), adminHandler.ListStaffUsers)
	adminGroup.POST("/staff-users", normalTimeout, bodyProfile, middleware.RequirePermission(authz.PermAdminUserManage), adminHandler.CreateStaffUser)

	// Platform-global LLM Provider & Model configuration.
	platformAIGroup.GET("/llm-providers", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), llmConfigHandler.ListProviders)
	platformAIGroup.POST("/llm-providers", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIConfigManage), llmConfigHandler.CreateProvider)
	platformAIGroup.PUT("/llm-providers/:id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIConfigManage), llmConfigHandler.UpdateProvider)
	platformAIGroup.DELETE("/llm-providers/:id", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigManage), llmConfigHandler.DeleteProvider)
	platformAIGroup.POST("/llm-providers/:id/test", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), llmConfigHandler.TestProviderConnection)
	platformAIGroup.POST("/llm-providers/:id/models/discover", mcpTimeout, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), llmConfigHandler.DiscoverProviderModels)
	platformAIGroup.POST("/llm-providers/:id/models/preset", mcpTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), llmConfigHandler.GetProviderModelPreset)

	platformAIGroup.GET("/llm-models", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), llmConfigHandler.ListModels)
	platformAIGroup.POST("/llm-models", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIConfigManage), llmConfigHandler.CreateModel)
	platformAIGroup.PUT("/llm-models/:id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIConfigManage), llmConfigHandler.UpdateModel)
	platformAIGroup.DELETE("/llm-models/:id", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigManage), llmConfigHandler.DeleteModel)
	platformAIGroup.POST("/llm-models/:id/test", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), llmConfigHandler.TestModelConnection)

	// Prompt template management — requires AI business permission
	platformAIGroup.GET("/prompt-templates", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), promptHandler.List)
	platformAIGroup.POST("/prompt-templates", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), promptHandler.Create)
	platformAIGroup.PUT("/prompt-templates/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), promptHandler.Update)
	platformAIGroup.DELETE("/prompt-templates/:id", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigManage), promptHandler.Delete)
	platformAIGroup.GET("/prompt-templates/:id/versions", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), promptHandler.ListVersions)
	platformAIGroup.POST("/prompt-templates/:id/rollback", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), promptHandler.Rollback)

	// Agent configuration management — requires AI business permission
	agentConfigHandler := hr.NewAgentConfigHandler(clients)
	platformAIGroup.GET("/agent-configs", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), agentConfigHandler.ListAgents)
	platformAIGroup.GET("/agent-configs/capabilities", mcpTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), agentConfigHandler.ListCapabilities)
	platformAIGroup.POST("/agent-configs", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), agentConfigHandler.CreateAgent)
	platformAIGroup.PUT("/agent-configs/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), agentConfigHandler.UpdateAgent)
	platformAIGroup.DELETE("/agent-configs/:id", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigManage), agentConfigHandler.DeleteAgent)

	// MCP server management
	mcpHandler := hr.NewMCPHandler(clients)
	platformAIGroup.GET("/mcp-servers", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), mcpHandler.ListMCPServers)
	platformAIGroup.POST("/mcp-servers", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), mcpHandler.CreateMCPServer)
	platformAIGroup.PUT("/mcp-servers/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), mcpHandler.UpdateMCPServer)
	platformAIGroup.DELETE("/mcp-servers/:id", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigManage), mcpHandler.DeleteMCPServer)
	platformAIGroup.GET("/mcp-tool-policies", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), mcpHandler.ListMCPToolPolicies)
	platformAIGroup.POST("/mcp-tool-policies", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), mcpHandler.CreateMCPToolPolicy)
	platformAIGroup.PUT("/mcp-tool-policies/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), mcpHandler.UpdateMCPToolPolicy)
	platformAIGroup.DELETE("/mcp-tool-policies/:id", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigManage), mcpHandler.DeleteMCPToolPolicy)
	platformAIGroup.POST("/mcp-servers/:id/test", mcpTimeout, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), mcpHandler.TestMCPConnection)
	platformAIGroup.GET("/mcp-servers/:id/tools", mcpTimeout, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), mcpHandler.ListMCPTools)
	platformAIGroup.GET("/mcp-servers/:id/logs", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsRead), mcpHandler.ListMCPToolLogs)
	platformAIGroup.POST("/mcp-servers/:id/call-tool", mcpTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), mcpHandler.CallMCPTool)

	// SKILL registry management
	skillHandler := hr.NewSkillHandler(clients)
	platformAIGroup.GET("/skills", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), skillHandler.ListSkills)
	platformAIGroup.POST("/skills", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), skillHandler.CreateSkill)
	platformAIGroup.PUT("/skills/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), skillHandler.UpdateSkill)
	platformAIGroup.GET("/skills/:id/versions", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), skillHandler.ListSkillVersions)
	platformAIGroup.POST("/skills/:id/versions", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), skillHandler.CreateSkillVersion)
	platformAIGroup.POST("/skills/:id/versions/:version_id/activate", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigManage), skillHandler.ActivateSkillVersion)
	platformAIGroup.GET("/skills/:id/tools", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), skillHandler.ListSkillTools)
	platformAIGroup.PUT("/skills/:id/tools/:tool_id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), skillHandler.UpdateSkillTool)

	// Agent SKILL.md management — requires AI business permission
	platformAIGroup.GET("/agent-skills", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), agentSkillHandler.List)
	platformAIGroup.GET("/agent-skills/semantic-debug", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsRead), agentSkillHandler.DebugSemanticRetrieval)
	platformAIGroup.POST("/agent-skills", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), agentSkillHandler.Create)
	platformAIGroup.POST("/agent-skills/preview", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), agentSkillHandler.Preview)
	platformAIGroup.GET("/agent-skills/:id", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), agentSkillHandler.Get)
	platformAIGroup.PUT("/agent-skills/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), agentSkillHandler.Update)
	platformAIGroup.PATCH("/agent-skills/:id/status", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIConfigManage), agentSkillHandler.UpdateStatus)
	platformAIGroup.POST("/agent-skills/:id/embedding/regenerate", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), agentSkillHandler.RegenerateEmbedding)
	platformAIGroup.GET("/agent-skills/:id/versions", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), agentSkillHandler.ListVersions)
	platformAIGroup.POST("/agent-skills/:id/versions", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermPlatformAIConfigManage), agentSkillHandler.CreateVersion)
	platformAIGroup.POST("/agent-skills/:id/versions/:version_id/activate", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigManage), agentSkillHandler.ActivateVersion)

	// Embedding Provider & Model configuration — requires SYSTEM_CONFIG_MANAGE
	platformAIGroup.GET("/embedding-providers", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), embeddingConfigHandler.ListProviders)
	platformAIGroup.POST("/embedding-providers", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIConfigManage), embeddingConfigHandler.CreateProvider)
	platformAIGroup.PUT("/embedding-providers/:id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIConfigManage), embeddingConfigHandler.UpdateProvider)
	platformAIGroup.DELETE("/embedding-providers/:id", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigManage), embeddingConfigHandler.DeleteProvider)

	platformAIGroup.GET("/embedding-models", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigRead), embeddingConfigHandler.ListModels)
	platformAIGroup.POST("/embedding-models", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIConfigManage), embeddingConfigHandler.CreateModel)
	platformAIGroup.PUT("/embedding-models/:id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIConfigManage), embeddingConfigHandler.UpdateModel)
	platformAIGroup.POST("/embedding-models/:id/set-default", normalTimeout, middleware.RequirePermission(authz.PermPlatformAIConfigManage), embeddingConfigHandler.SetDefaultModel)
	platformAIGroup.POST("/embedding-models/test", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), embeddingConfigHandler.TestModel)

	platformAIGroup.POST("/embedding-backfill", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermPlatformAIDiagnosticsExec), embeddingConfigHandler.BackfillEmbeddings)

	return r, limiters
}
