package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"web-gin-service/config"
	_ "web-gin-service/docs"
	"web-gin-service/handler"
	"web-gin-service/handler/candidate"
	"web-gin-service/handler/hr"
	"web-gin-service/middleware"
	"web-gin-service/pkg/redisclient"
	"web-gin-service/rpc"
)

func Setup(cfg config.Config, clients *rpc.Clients, rdb *redis.Client) (*gin.Engine, *middleware.LimiterRegistry) {
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

	authHandler := handler.NewAuthHandler(clients, cfg.AuthCookieName, cfg.CandidateCookie, cfg.HRCookie, cfg.AuthCookieSecure, cfg.JWTSecret)
	publicHandler := handler.NewPublicHandler(clients)
	hrJobHandler := hr.NewJobHandler(clients)
	hrApplicationHandler := hr.NewApplicationHandler(clients)
	hrAIHandler := hr.NewAIHandler(clients)
	profileHandler := candidate.NewProfileHandler(clients)
	resumeHandler := candidate.NewResumeHandler(clients)
	applyHandler := candidate.NewApplyHandler(clients)
	candidateAIHandler := candidate.NewAIHandler(clients)
	notificationHandler := handler.NewNotificationHandler(clients, rdb)
	dashboardHandler := hr.NewDashboardHandler(clients)

	normalTimeout := middleware.Timeout(10 * time.Second)
	uploadTimeout := middleware.Timeout(20 * time.Second)
	aiTimeout := middleware.Timeout(45 * time.Second)

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

	v1.POST("/auth/register", normalTimeout, authLimit, bodyAuth, authHandler.Register)
	v1.POST("/auth/register/validate-invite-code", normalTimeout, authHandler.ValidateInviteCode)
	v1.POST("/auth/login", normalTimeout, authLimit, bodyAuth, authHandler.Login)
	v1.POST("/auth/logout", normalTimeout, middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.AuthCookieName), authHandler.Logout)
	v1.POST("/auth/refresh", normalTimeout, authHandler.RefreshToken)
	v1.GET("/auth/me", normalTimeout, middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.AuthCookieName), authHandler.Me)
	v1.GET("/jobs", normalTimeout, publicHandler.ListJobs)
	v1.GET("/jobs/:job_id", normalTimeout, publicHandler.JobDetail)

	candidateGroup := v1.Group("/candidate", middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.AuthCookieName), middleware.RequireRole(1))
	candidateGroup.GET("/profile", normalTimeout, profileHandler.Get)
	candidateGroup.PUT("/profile", normalTimeout, bodyProfile, profileHandler.Update)
	candidateGroup.GET("/resume", normalTimeout, resumeHandler.Get)
	candidateGroup.POST("/resume/presign", riskBlock, resumePresignQuota, normalTimeout, bodyAuth, resumeHandler.Presign)
	candidateGroup.POST("/resume/confirm", riskBlock, resumeConfirmQuota, uploadTimeout, bodyProfile, resumeHandler.Confirm)
	candidateGroup.POST("/applications", normalTimeout, bodyAuth, applyHandler.Apply)
	candidateGroup.GET("/applications", normalTimeout, applyHandler.Mine)
	candidateGroup.GET("/notifications", normalTimeout, notificationHandler.List)
	candidateGroup.GET("/notifications/unread-count", normalTimeout, notificationHandler.UnreadCount)
	candidateGroup.GET("/notifications/summary", normalTimeout, notificationHandler.Summary)
	candidateGroup.GET("/notifications/stream", notificationHandler.Stream)
	candidateGroup.PATCH("/notifications/:notification_id/read", normalTimeout, notificationHandler.MarkRead)
	candidateGroup.PATCH("/notifications/read-all", normalTimeout, notificationHandler.MarkAllRead)
	candidateGroup.GET("/ai/sessions", normalTimeout, candidateAIHandler.ListSessions)
	candidateGroup.POST("/ai/sessions", normalTimeout, bodyProfile, candidateAIHandler.CreateSession)
	candidateGroup.GET("/ai/sessions/:session_id/messages", normalTimeout, candidateAIHandler.SessionMessages)
	candidateGroup.PUT("/ai/sessions/:session_id", normalTimeout, bodyAuth, candidateAIHandler.UpdateSession)
	candidateGroup.DELETE("/ai/sessions/:session_id", normalTimeout, candidateAIHandler.DeleteSession)
	candidateGroup.POST("/ai/chat/stream", riskBlock, aiLimit, candidateAIQuota, bodyAI, candidateAIHandler.ChatStream)

	hrGroup := v1.Group("/hr", middleware.JWTAuthByClient(cfg.JWTSecret, cfg.CandidateCookie, cfg.HRCookie, cfg.AuthCookieName), middleware.RequireMinRole(2))
	hrGroup.GET("/job-options", normalTimeout, hrJobHandler.JobOptions)
	hrGroup.POST("/jobs", normalTimeout, bodyJob, hrJobHandler.Create)
	hrGroup.PUT("/jobs/:job_id", normalTimeout, bodyJob, hrJobHandler.Update)
	hrGroup.PATCH("/jobs/:job_id/offline", normalTimeout, hrJobHandler.Offline)
	hrGroup.PATCH("/jobs/:job_id/online", normalTimeout, hrJobHandler.Online)
	hrGroup.GET("/jobs", normalTimeout, hrJobHandler.List)
	hrGroup.GET("/jobs/:job_id/applications", normalTimeout, hrApplicationHandler.ListByJob)
	hrGroup.PATCH("/applications/:application_id/status", normalTimeout, hrApplicationHandler.UpdateStatus)
	hrGroup.GET("/ai/sessions", normalTimeout, hrAIHandler.ListSessions)
	hrGroup.POST("/ai/sessions", normalTimeout, hrAIHandler.CreateSession)
	hrGroup.GET("/ai/sessions/:session_id/messages", normalTimeout, hrAIHandler.SessionMessages)
	hrGroup.PUT("/ai/sessions/:session_id", normalTimeout, hrAIHandler.UpdateSession)
	hrGroup.DELETE("/ai/sessions/:session_id", normalTimeout, hrAIHandler.DeleteSession)
	hrGroup.POST("/ai/application-analysis-sessions", riskBlock, aiLimit, hrAIQuota, aiTimeout, hrAIHandler.CreateApplicationAnalysisSession)
	hrGroup.POST("/ai/chat", riskBlock, aiLimit, hrAIQuota, aiTimeout, hrAIHandler.Chat)
	hrGroup.POST("/ai/chat/stream", riskBlock, aiLimit, hrAIQuota, hrAIHandler.ChatStream)
	hrGroup.POST("/ai/analyze-application", riskBlock, aiLimit, hrAIQuota, aiTimeout, hrAIHandler.AnalyzeApplication)
	hrGroup.GET("/ai/history", normalTimeout, hrAIHandler.History)
	hrGroup.GET("/notifications", normalTimeout, notificationHandler.List)
	hrGroup.GET("/notifications/unread-count", normalTimeout, notificationHandler.UnreadCount)
	hrGroup.GET("/notifications/summary", normalTimeout, notificationHandler.Summary)
	hrGroup.GET("/notifications/stream", notificationHandler.Stream)
	hrGroup.PATCH("/notifications/:notification_id/read", normalTimeout, notificationHandler.MarkRead)
	hrGroup.PATCH("/notifications/read-all", normalTimeout, notificationHandler.MarkAllRead)
	hrGroup.GET("/dashboard/summary", normalTimeout, dashboardHandler.Summary)

	adminHandler := hr.NewAdminHandler(clients)
	adminGroup := hrGroup.Group("/admin", middleware.RequireRole(3))
	adminGroup.POST("/invite-codes", normalTimeout, bodyAuth, adminHandler.CreateInviteCode)
	adminGroup.GET("/invite-codes", normalTimeout, adminHandler.ListInviteCodes)
	adminGroup.PATCH("/invite-codes/:id/extend", normalTimeout, bodyAuth, adminHandler.ExtendInviteCode)
	adminGroup.PATCH("/invite-codes/:id/revoke", normalTimeout, bodyAuth, adminHandler.RevokeInviteCode)
	adminGroup.PATCH("/invite-codes/:id/reactivate", normalTimeout, bodyAuth, adminHandler.ReactivateInviteCode)

	// Department taxonomy
	adminGroup.GET("/departments", normalTimeout, adminHandler.ListDepartments)
	adminGroup.POST("/departments", normalTimeout, bodyAuth, adminHandler.CreateDepartment)
	adminGroup.PUT("/departments/:id", normalTimeout, bodyAuth, adminHandler.UpdateDepartment)
	adminGroup.PATCH("/departments/:id/status", normalTimeout, bodyAuth, adminHandler.UpdateDepartmentStatus)
	adminGroup.DELETE("/departments/:id", normalTimeout, adminHandler.DeleteDepartment)

	// Job location taxonomy
	adminGroup.GET("/locations", normalTimeout, adminHandler.ListJobLocations)
	adminGroup.POST("/locations", normalTimeout, bodyAuth, adminHandler.CreateJobLocation)
	adminGroup.PUT("/locations/:id", normalTimeout, bodyAuth, adminHandler.UpdateJobLocation)
	adminGroup.PATCH("/locations/:id/status", normalTimeout, bodyAuth, adminHandler.UpdateJobLocationStatus)
	adminGroup.DELETE("/locations/:id", normalTimeout, adminHandler.DeleteJobLocation)

	// Department location config (admin)
	adminGroup.GET("/departments/location-map", normalTimeout, adminHandler.ListDepartmentsLocationMap)
	adminGroup.GET("/departments/:id/locations", normalTimeout, adminHandler.GetDepartmentLocationConfig)
	adminGroup.PUT("/departments/:id/locations", normalTimeout, bodyAuth, adminHandler.UpdateDepartmentLocationConfig)

	// Usage audit (admin only)
	adminGroup.GET("/third-party-usage-logs", normalTimeout, adminHandler.ListUsageLogs)

	return r, limiters
}
