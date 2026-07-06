package service

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"logic-grpc-service/ai"
	"logic-grpc-service/config"
	"logic-grpc-service/email"
	"logic-grpc-service/mq"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/pkg/crypto"
	"logic-grpc-service/pkg/logger"
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
	Auth                   *AuthService
	Job                    *JobService
	Candidate              *CandidateService
	Application            *ApplicationService
	Interview              *InterviewService
	Offer                  *OfferService
	AI                     *AIService
	CandidateAI            *CandidateAIService
	Notification           *NotificationService
	Admin                  *AdminService
	Taxonomy               *JobTaxonomyService
	Collaboration          *CollaborationService
	Analytics              *AnalyticsService
	LlmConfig              *LlmConfigService
	Prompt                 *PromptService
	AgentConfig            *AgentConfigService
	MCP                    *MCPService
	Skill                  *SkillService
	AgentSkill             *AgentSkillService
	ResumeProfile          *ResumeProfileService
	CandidateMatch         *CandidateMatchService
	RecruitingIntelligence *RecruitingIntelligenceService
	Embedding              *EmbeddingService

	// Phase 6: Audit context repo for AI usage audit writes
	UsageAuditCtxRepo *repository.UsageAuditContextRepo

	// P1-003: AI usage statistics
	UsageStats *UsageStatsService
	// Background workers (caller must Start/Stop)
	OutboxPublisher      *OutboxPublisher
	NotificationConsumer *NotificationConsumer
	ResumeParseConsumer  *ResumeParseConsumer
	EmailConsumer        *EmailConsumer
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
	agentRuns *repository.AgentRunRepo,
	memories *repository.MemoryRepo,
	notifications *repository.NotificationRepo,
	outbox *repository.OutboxRepo,
	inviteCodes *repository.InviteCodeRepo,
	departments *repository.DepartmentRepo,
	locations *repository.JobLocationRepo,
	deptLocs *repository.DepartmentLocationRepo,
	usageLogs *repository.UsageLogRepo,
	authzRepo *repository.AuthzRepo,
	emailLogRepo *repository.EmailLogRepo,
	notifCache *cache.NotificationCache,
	jobCache *cache.JobCache,
	ossClient oss.Storage,
	aiClient *ai.Client,
	mqConn *mq.Conn,
	cfg config.Config,
	jwtSecret string,
	emailSender email.Sender,
	emailRenderer *email.Renderer,
) *Services {
	resumeProfileRepo := repository.NewResumeProfileRepo(db)
	candidateMatchRepo := repository.NewCandidateMatchRepo(db)
	runtimePolicy := NewAgentRuntimePolicy(cfg)
	toolExecutor := ai.NewToolExecutor(applications, jobs, resumes, ossClient, authzRepo, profiles, resumeProfileRepo, candidateMatchRepo)
	candidateToolExecutor := ai.NewCandidateToolExecutor(applications, jobs, resumes)
	embeddingSvc := NewEmbeddingService(repository.NewAIEmbeddingRepo(db), UnavailableEmbeddingProvider{}).WithRuntimePolicy(runtimePolicy)
	contextBuilder := NewAgentContextBuilder(chats, summaries, memories, aiClient, cfg, repository.NewPromptTemplateRepo(db)).WithEmbeddingService(embeddingSvc)
	agentRuntime := cfg.AI.AgentRuntime
	analyticsRepo := repository.NewAnalyticsRepo(db)
	usageAuditCtxRepo := repository.NewUsageAuditContextRepo(db)
	taxonomy := NewJobTaxonomyService(departments, locations, jobs, deptLocs)

	outboxPublisher := NewOutboxPublisher(outbox, mqConn)
	notificationConsumer := NewNotificationConsumer(notifications, notifCache)
	resumeParseConsumer := NewResumeParseConsumer(resumes, ossClient)
	emailConsumer := NewEmailConsumer(users, emailLogRepo, emailRenderer, emailSender)
	scopeEval := &scopeEvaluator{authzRepo: authzRepo}
	serviceAuth := NewServiceAuthorizer(authzRepo, scopeEval)

	collaborationRepo := repository.NewCollaborationRepo(db)
	llmConfigSvc := newLlmConfigServiceWithFallback(db)
	agentCfgRepo := repository.NewAgentConfigRepo(db)
	promptTmplRepo := repository.NewPromptTemplateRepo(db)
	candidateAI := NewCandidateAIService(usageLogs, usageAuditCtxRepo, authzRepo, chats, applications, jobs, resumes, aiClient, candidateToolExecutor, agentRuntime, toolTraces, summaries, promptTmplRepo, agentCfgRepo)

	// Initialize MCP service before AI service for MCP tool injection
	mcpSvc := NewMCPService(repository.NewMCPRepo(db), cfg).WithRuntimePolicy(runtimePolicy)
	skillSvc := NewSkillService(repository.NewSkillRepo(db))
	agentSkillSvc := NewAgentSkillServiceWithAgentConfigRepo(repository.NewAgentSkillRepo(db), agentCfgRepo).WithSemanticDebugDependencies(memories, embeddingSvc)
	agentSkillRepo := repository.NewAgentSkillRepo(db)
	heuristicExtractor := NewHeuristicResumeProfileExtractor()
	llmExtractor := NewLLMResumeProfileExtractor(llmConfigSvc, promptTmplRepo)
	resumeProfileExtractor := NewFallbackResumeProfileExtractor(llmExtractor, heuristicExtractor,
		WithFallbackEnabled(runtimePolicy.Fallbacks))
	if llmConfigSvc == nil {
		logger.L().Warn("resume profile extractor: LLM not configured (ENCRYPTION_KEY not set); " +
			"fallback behavior controlled by agent.features.fallbacks config")
	} else {
		logger.L().Info("resume profile extractor: LLM + heuristic fallback chain initialized")
	}
	resumeProfileSvc := NewResumeProfileService(resumes, resumeProfileRepo, resumeProfileExtractor).WithRuntimePolicy(runtimePolicy)
	var reqExtractor RequirementExtractorV2 = NewJobRequirementExtractorHeuristic()
	if llmConfigSvc != nil {
		reqExtractor = NewFallbackRequirementExtractor(
			NewLLMJobRequirementExtractor(llmConfigSvc, promptTmplRepo),
			NewJobRequirementExtractorHeuristic(),
			WithReqFallbackEnabled(runtimePolicy.Fallbacks),
		)
	} else {
		logger.L().Info("requirement extractor: LLM not configured, using heuristic only")
	}
	candidateMatchSvc := NewCandidateMatchService(applications, jobs, profiles, resumes, resumeProfileRepo, candidateMatchRepo).
		WithRuntimePolicy(runtimePolicy).
		WithRequirementExtractor(reqExtractor).
		WithPromptRepo(promptTmplRepo)
	recruitingIntelligenceSvc := NewRecruitingIntelligenceService(applications, jobs, resumes, resumeProfileRepo, candidateMatchRepo, resumeProfileSvc, candidateMatchSvc, serviceAuth)

	return &Services{
		Auth:              NewAuthService(users, tokens, authzRepo, inviteCodes, jwtSecret),
		Analytics:         NewAnalyticsService(analyticsRepo, authzRepo, serviceAuth),
		Admin:             NewAdminService(inviteCodes, usageLogs, users, authzRepo, tokenCache, serviceAuth),
		UsageAuditCtxRepo: usageAuditCtxRepo,

		// P1-003: AI usage statistics
		UsageStats: NewUsageStatsService(repository.NewUsageStatsRepo(db), serviceAuth),

		Job:                    NewJobService(jobs, jobCache, authzRepo, taxonomy, scopeEval),
		Taxonomy:               taxonomy,
		Candidate:              NewCandidateService(profiles, resumes, ossClient, outboxPublisher, usageLogs, serviceAuth),
		Application:            NewApplicationService(authzRepo, applications, profiles, resumes, jobs, interviews, notifications, outboxPublisher, ossClient, jobCache, scopeEval),
		Interview:              NewInterviewService(authzRepo, interviews, users, applications, jobs, notifications, outboxPublisher, ossClient, scopeEval, serviceAuth),
		Offer:                  NewOfferService(authzRepo, offers, applications, jobs, notifications, outboxPublisher, scopeEval, serviceAuth),
		AI:                     NewAIService(chats, applications, jobs, resumes, summaries, toolTraces, agentRuns, memories, ossClient, aiClient, toolExecutor, contextBuilder, candidateAI, usageLogs, usageAuditCtxRepo, authzRepo, agentRuntime, serviceAuth, llmConfigSvc, agentCfgRepo, promptTmplRepo, mcpSvc, skillSvc, agentSkillRepo).WithEmbeddingService(embeddingSvc).WithRuntimePolicy(runtimePolicy),
		CandidateAI:            candidateAI,
		Notification:           NewNotificationService(notifications, notifCache, serviceAuth),
		LlmConfig:              llmConfigSvc,
		Prompt:                 NewPromptService(promptTmplRepo),
		AgentConfig:            NewAgentConfigService(agentCfgRepo, promptTmplRepo, mcpSvc, skillSvc),
		MCP:                    mcpSvc,
		Skill:                  skillSvc,
		AgentSkill:             agentSkillSvc,
		ResumeProfile:          resumeProfileSvc,
		CandidateMatch:         candidateMatchSvc,
		RecruitingIntelligence: recruitingIntelligenceSvc,
		Embedding:              embeddingSvc,

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
		EmailConsumer:        emailConsumer,
	}
}

// newLlmConfigServiceWithFallback creates a LlmConfigService if ENCRYPTION_KEY is available,
// or returns nil if not set, allowing the rest of the app to function without the encryption key.
func newLlmConfigServiceWithFallback(db *gorm.DB) *LlmConfigService {
	encKey, err := crypto.LoadEncryptionKey()
	if err != nil {
		logger.L().Warn("llm config service disabled: ENCRYPTION_KEY not set, "+
			"provider config and model config APIs will return errors", zap.Error(err))
		return nil
	}
	return NewLlmConfigService(repository.NewProviderRepo(db), repository.NewModelConfigRepo(db), encKey)
}
