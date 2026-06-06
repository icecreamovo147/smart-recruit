package service

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"logic-grpc-service/ai"
	"logic-grpc-service/config"
	"logic-grpc-service/mq"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/repository"
)

// TransactionPolicy documents the service-layer transaction strategy.
//
// Single-write operations (JobService, AuthService read-only methods) use
// direct repository calls without explicit transactions — the database handles
// atomicity at the statement level.
//
// Multi-write operations must be wrapped in (*Repo).Transaction(ctx, fn):
//   - ApplicationService.ApplyJob         — insert application + outbox event
//   - ApplicationService.UpdateApplicationStatus — update status + outbox event + is_current flags
//   - CandidateService.ConfirmResumeUpload — insert/update resume + outbox event
//   - Any future operation that combines 2+ writes or a write + outbox event
//
// The outbox pattern guarantees that domain events are only committed when
// their parent transaction commits, preventing partial event emission.

// Services aggregates all domain services and background workers.
type Services struct {
	Auth          *AuthService
	Job           *JobService
	Candidate     *CandidateService
	Application   *ApplicationService
	Interview     *InterviewService
	Offer         *OfferService
	AI            *AIService
	CandidateAI   *CandidateAIService
	Notification  *NotificationService
	Admin         *AdminService
	Taxonomy      *JobTaxonomyService
	Collaboration *CollaborationService
	Analytics     *AnalyticsService

	// Phase 6: Audit context repo for AI usage audit writes
	UsageAuditCtxRepo *repository.UsageAuditContextRepo

	// Background workers (caller must Start/Stop)
	OutboxPublisher      *OutboxPublisher
	NotificationConsumer *NotificationConsumer
	ResumeParseConsumer  *ResumeParseConsumer
}

func NewServices(
	tokenCache *redis.Client,
	db *gorm.DB,
	users *repository.UserRepo,
	tokens *repository.RefreshTokenRepo,
	jobs *repository.JobRepo,
	profiles *repository.ProfileRepo,
	resumes *repository.ResumeRepo,
	applications *repository.ApplicationRepo,
	interviews *repository.InterviewRepo,
	offers *repository.OfferRepo,
	chats *repository.ChatRepo,
	summaries *repository.SessionSummaryRepo,
	toolTraces *repository.ToolTraceRepo,
	memories *repository.MemoryRepo,
	notifications *repository.NotificationRepo,
	outbox *repository.OutboxRepo,
	inviteCodes *repository.InviteCodeRepo,
	departments *repository.DepartmentRepo,
	locations *repository.JobLocationRepo,
	deptLocs *repository.DepartmentLocationRepo,
	usageLogs *repository.UsageLogRepo,
	authzRepo *repository.AuthzRepo,
	notifCache *cache.NotificationCache,
	jobCache *cache.JobCache,
	ossClient oss.Storage,
	aiClient *ai.Client,
	mqConn *mq.Conn,
	cfg config.Config,
	jwtSecret string,
) *Services {
	toolExecutor := ai.NewToolExecutor(applications, jobs, resumes, ossClient, authzRepo)
	candidateToolExecutor := ai.NewCandidateToolExecutor(applications, jobs, resumes)
	contextBuilder := NewAgentContextBuilder(chats, summaries, memories, aiClient, cfg)
	agentRuntime := cfg.AI.AgentRuntime
	analyticsRepo := repository.NewAnalyticsRepo(db)
	usageAuditCtxRepo := repository.NewUsageAuditContextRepo(db)
	candidateAI := NewCandidateAIService(usageLogs, usageAuditCtxRepo, authzRepo, chats, applications, jobs, resumes, aiClient, candidateToolExecutor, agentRuntime, toolTraces, summaries)
	taxonomy := NewJobTaxonomyService(departments, locations, jobs, deptLocs)

	outboxPublisher := NewOutboxPublisher(outbox, mqConn)
	notificationConsumer := NewNotificationConsumer(notifications, notifCache)
	resumeParseConsumer := NewResumeParseConsumer(resumes, ossClient)
	scopeEval := &scopeEvaluator{authzRepo: authzRepo}
	serviceAuth := NewServiceAuthorizer(authzRepo, scopeEval)

	collaborationRepo := repository.NewCollaborationRepo(db)

	return &Services{
		Auth:              NewAuthService(users, tokens, authzRepo, inviteCodes, jwtSecret),
		Analytics:         NewAnalyticsService(analyticsRepo, authzRepo, serviceAuth),
		Admin:             NewAdminService(inviteCodes, usageLogs, users, authzRepo, tokenCache, serviceAuth),
		UsageAuditCtxRepo: usageAuditCtxRepo,
		Job:               NewJobService(jobs, jobCache, authzRepo, taxonomy, scopeEval),
		Taxonomy:          taxonomy,
		Candidate:         NewCandidateService(profiles, resumes, ossClient, outboxPublisher, usageLogs, serviceAuth),
		Application:       NewApplicationService(authzRepo, applications, profiles, resumes, jobs, interviews, notifications, outboxPublisher, ossClient, jobCache, scopeEval),
		Interview:         NewInterviewService(authzRepo, interviews, users, applications, jobs, notifications, outboxPublisher, ossClient, scopeEval, serviceAuth),
		Offer:             NewOfferService(authzRepo, offers, applications, jobs, notifications, outboxPublisher, scopeEval, serviceAuth),
		AI:                NewAIService(chats, applications, jobs, resumes, summaries, toolTraces, memories, ossClient, aiClient, toolExecutor, contextBuilder, candidateAI, usageLogs, usageAuditCtxRepo, authzRepo, agentRuntime, serviceAuth),
		CandidateAI:       candidateAI,
		Notification:      NewNotificationService(notifications, notifCache, serviceAuth),
		Collaboration: NewCollaborationService(
			authzRepo,
			collaborationRepo,
			applications,
			profiles,
			jobs,
			users,
			interviews,
			offers,
			resumes,
			ossClient,
			serviceAuth,
			scopeEval,
		),

		OutboxPublisher:      outboxPublisher,
		NotificationConsumer: notificationConsumer,
		ResumeParseConsumer:  resumeParseConsumer,
	}
}
