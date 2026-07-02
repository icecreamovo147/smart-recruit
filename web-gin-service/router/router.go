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

	"web-gin-service/config"
	_ "web-gin-service/docs"
	"web-gin-service/handler"
	"web-gin-service/handler/candidate"
	"web-gin-service/handler/hr"
	"web-gin-service/middleware"
	"web-gin-service/pkg/authz"
	"web-gin-service/pkg/logger"
	"web-gin-service/pkg/redisclient"
	pb "web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
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
	// Background audit consumer: forwards events to logic-grpc-service.
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
	r.Use(middleware.RequestID(), middleware.AccessLog(), middleware.Recovery(), middleware.SecurityHeaders(), middleware.CSP(), middleware.CORSWrapper())
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
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authHandler := handler.NewAuthHandler(clients, cfg.AuthCookieName, cfg.CandidateCookie, cfg.HRCookie, cfg.InterviewerCookie, cfg.AuthCookieSecure, cfg.JWTSecret, rdb)
	publicHandler := handler.NewPublicHandler(clients)
	hrJobHandler := hr.NewJobHandler(clients)
	hrApplicationHandler := hr.NewApplicationHandler(clients)
	hrAIHandler := hr.NewAIHandler(clients)
	llmConfigHandler := hr.NewLlmConfigHandler(clients)
	hrInterviewHandler := hr.NewInterviewHandler(clients)
	hrOfferHandler := hr.NewOfferHandler(clients)
	hrCapabilityHandler := hr.NewCapabilityHandler(clients)
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
	v1.GET("/auth/me", normalTimeout, middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.InterviewerCookie, cfg.AuthCookieName, rdb), authHandler.Me)
	v1.PUT("/auth/email", normalTimeout, bodyAuth, middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.InterviewerCookie, cfg.AuthCookieName, rdb), authHandler.UpdateEmail)
	v1.GET("/jobs", normalTimeout, publicHandler.ListJobs)
	v1.GET("/jobs/:job_id", normalTimeout, publicHandler.JobDetail)

	// ── Authenticated middleware (with token_version validation via Redis) ─
	jwtAuth := middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.InterviewerCookie, cfg.AuthCookieName, rdb)
	currentPrincipal := middleware.ValidateCurrentPrincipal(func(ctx context.Context, userID int64) (*middleware.CurrentPrincipal, error) {
		resp, err := clients.Auth.GetPrincipal(ctx, &pb.GetPrincipalRequest{UserId: userID})
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
		}, nil
	})

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
	candidateGroup.GET("/ai/sessions", normalTimeout, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.ListSessions)
	candidateGroup.POST("/ai/sessions", normalTimeout, bodyProfile, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.CreateSession)
	candidateGroup.GET("/ai/sessions/:session_id/messages", normalTimeout, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.SessionMessages)
	candidateGroup.PUT("/ai/sessions/:session_id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.UpdateSession)
	candidateGroup.DELETE("/ai/sessions/:session_id", normalTimeout, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.DeleteSession)
	candidateGroup.POST("/ai/chat/stream", riskBlock, aiLimit, candidateAIQuota, bodyAI, middleware.RequirePermission(authz.PermAICandidateUse), candidateAIHandler.ChatStream)

	// ── Staff routes (formerly /hr) ────────────────────────────────────
	// Base group: any staff role (recruiter, recruiting_admin, system_admin, interviewer).
	staffGroup := v1.Group("/hr", jwtAuth, currentPrincipal, middleware.RequireAnyRole(authz.StaffRoles()...))

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
	staffGroup.GET("/applications/:application_id/resume-profile", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), recruitingIntelligenceHandler.GetResumeProfileByApplication)
	staffGroup.GET("/resume-profiles", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), recruitingIntelligenceHandler.GetResumeProfile)
	staffGroup.POST("/resume-profiles/parse", riskBlock, aiLimit, hrAIQuota, aiTimeout, bodyAuth, middleware.RequirePermission(authz.PermAIHRUse), recruitingIntelligenceHandler.ParseResumeProfile)
	staffGroup.POST("/applications/:application_id/match-evaluations", riskBlock, aiLimit, hrAIQuota, aiTimeout, bodyAuth, middleware.RequirePermission(authz.PermAIHRUse), recruitingIntelligenceHandler.EvaluateCandidateMatch)
	staffGroup.GET("/applications/:application_id/match-evaluation", normalTimeout, middleware.RequirePermission(authz.PermApplicationRead), recruitingIntelligenceHandler.GetCandidateMatchEvaluation)
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
	staffGroup.GET("/ai/sessions/:session_id/tool-traces", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.GetToolTraces)
	staffGroup.GET("/ai/sessions/:session_id/agent-runs", normalTimeout, middleware.RequirePermission(authz.PermAIHRUse), hrAIHandler.GetAgentRuns)
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

	// LLM Provider & Model configuration — requires SYSTEM_CONFIG_MANAGE
	adminGroup.GET("/llm-providers", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), llmConfigHandler.ListProviders)
	adminGroup.POST("/llm-providers", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermSystemConfigManage), llmConfigHandler.CreateProvider)
	adminGroup.PUT("/llm-providers/:id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermSystemConfigManage), llmConfigHandler.UpdateProvider)
	adminGroup.DELETE("/llm-providers/:id", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), llmConfigHandler.DeleteProvider)
	adminGroup.POST("/llm-providers/:id/test", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), llmConfigHandler.TestProviderConnection)

	adminGroup.GET("/llm-models", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), llmConfigHandler.ListModels)
	adminGroup.POST("/llm-models", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermSystemConfigManage), llmConfigHandler.CreateModel)
	adminGroup.PUT("/llm-models/:id", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermSystemConfigManage), llmConfigHandler.UpdateModel)
	adminGroup.DELETE("/llm-models/:id", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), llmConfigHandler.DeleteModel)

	// Prompt template management — requires AI business permission
	adminGroup.GET("/prompt-templates", normalTimeout, middleware.RequirePermission(authz.PermAIPromptManage), promptHandler.List)
	adminGroup.POST("/prompt-templates", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermAIPromptManage), promptHandler.Create)
	adminGroup.PUT("/prompt-templates/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermAIPromptManage), promptHandler.Update)
	adminGroup.DELETE("/prompt-templates/:id", normalTimeout, middleware.RequirePermission(authz.PermAIPromptManage), promptHandler.Delete)
	adminGroup.GET("/prompt-templates/:id/versions", normalTimeout, middleware.RequirePermission(authz.PermAIPromptManage), promptHandler.ListVersions)
	adminGroup.POST("/prompt-templates/:id/rollback", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermAIPromptManage), promptHandler.Rollback)

	// Agent configuration management — requires AI business permission
	agentConfigHandler := hr.NewAgentConfigHandler(clients)
	adminGroup.GET("/agent-configs", normalTimeout, middleware.RequirePermission(authz.PermAIAgentManage), agentConfigHandler.ListAgents)
	adminGroup.GET("/agent-configs/capabilities", mcpTimeout, middleware.RequirePermission(authz.PermAIAgentManage), agentConfigHandler.ListCapabilities)
	adminGroup.POST("/agent-configs", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermAIAgentManage), agentConfigHandler.CreateAgent)
	adminGroup.PUT("/agent-configs/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermAIAgentManage), agentConfigHandler.UpdateAgent)
	adminGroup.DELETE("/agent-configs/:id", normalTimeout, middleware.RequirePermission(authz.PermAIAgentManage), agentConfigHandler.DeleteAgent)

	// MCP server management
	mcpHandler := hr.NewMCPHandler(clients)
	adminGroup.GET("/mcp-servers", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.ListMCPServers)
	adminGroup.POST("/mcp-servers", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.CreateMCPServer)
	adminGroup.PUT("/mcp-servers/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.UpdateMCPServer)
	adminGroup.DELETE("/mcp-servers/:id", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.DeleteMCPServer)
	adminGroup.GET("/mcp-tool-policies", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.ListMCPToolPolicies)
	adminGroup.POST("/mcp-tool-policies", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.CreateMCPToolPolicy)
	adminGroup.PUT("/mcp-tool-policies/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.UpdateMCPToolPolicy)
	adminGroup.DELETE("/mcp-tool-policies/:id", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.DeleteMCPToolPolicy)
	adminGroup.POST("/mcp-servers/:id/test", mcpTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.TestMCPConnection)
	adminGroup.GET("/mcp-servers/:id/tools", mcpTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.ListMCPTools)
	adminGroup.POST("/mcp-servers/:id/call-tool", mcpTimeout, bodyAdmin, middleware.RequirePermission(authz.PermSystemConfigManage), mcpHandler.CallMCPTool)

	// SKILL registry management
	skillHandler := hr.NewSkillHandler(clients)
	adminGroup.GET("/skills", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), skillHandler.ListSkills)
	adminGroup.POST("/skills", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermSystemConfigManage), skillHandler.CreateSkill)
	adminGroup.PUT("/skills/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermSystemConfigManage), skillHandler.UpdateSkill)
	adminGroup.GET("/skills/:id/versions", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), skillHandler.ListSkillVersions)
	adminGroup.POST("/skills/:id/versions", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermSystemConfigManage), skillHandler.CreateSkillVersion)
	adminGroup.POST("/skills/:id/versions/:version_id/activate", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), skillHandler.ActivateSkillVersion)
	adminGroup.GET("/skills/:id/tools", normalTimeout, middleware.RequirePermission(authz.PermSystemConfigManage), skillHandler.ListSkillTools)
	adminGroup.PUT("/skills/:id/tools/:tool_id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermSystemConfigManage), skillHandler.UpdateSkillTool)

	// Agent SKILL.md management — requires AI business permission
	adminGroup.GET("/agent-skills", normalTimeout, middleware.RequirePermission(authz.PermAIAgentSkillManage), agentSkillHandler.List)
	adminGroup.GET("/agent-skills/semantic-debug", normalTimeout, middleware.RequirePermission(authz.PermAIAgentSkillManage), agentSkillHandler.DebugSemanticRetrieval)
	adminGroup.POST("/agent-skills", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermAIAgentSkillManage), agentSkillHandler.Create)
	adminGroup.POST("/agent-skills/preview", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermAIAgentSkillManage), agentSkillHandler.Preview)
	adminGroup.GET("/agent-skills/:id", normalTimeout, middleware.RequirePermission(authz.PermAIAgentSkillManage), agentSkillHandler.Get)
	adminGroup.PUT("/agent-skills/:id", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermAIAgentSkillManage), agentSkillHandler.Update)
	adminGroup.PATCH("/agent-skills/:id/status", normalTimeout, bodyAuth, middleware.RequirePermission(authz.PermAIAgentSkillManage), agentSkillHandler.UpdateStatus)
	adminGroup.GET("/agent-skills/:id/versions", normalTimeout, middleware.RequirePermission(authz.PermAIAgentSkillManage), agentSkillHandler.ListVersions)
	adminGroup.POST("/agent-skills/:id/versions", normalTimeout, bodyAdmin, middleware.RequirePermission(authz.PermAIAgentSkillManage), agentSkillHandler.CreateVersion)
	adminGroup.POST("/agent-skills/:id/versions/:version_id/activate", normalTimeout, middleware.RequirePermission(authz.PermAIAgentSkillManage), agentSkillHandler.ActivateVersion)

	return r, limiters
}
