package service

import (
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"smart-recruit-ai-agent-service/internal/legacydomain/ai"
	"smart-recruit-ai-agent-service/internal/legacydomain/repository"
	"smart-recruit-commons/email"
	"smart-recruit-commons/mq"
	"smart-recruit-commons/oss"
	"smart-recruit-commons/pkg/cache"
	"smart-recruit-commons/pkg/crypto"
	"smart-recruit-platform-go/logger"
	"smart-recruit-platform-go/serviceconfig"
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
	EmbeddingConfig        *EmbeddingConfigService

	// Phase 6: Audit context repo for AI usage audit writes
	UsageAuditCtxRepo *repository.UsageAuditContextRepo

	// P1-003: AI usage statistics
	UsageStats *UsageStatsService
	// Notification runtime owns notification persistence, realtime delivery,
	// email coordination, outbox dispatch, and notification consumers.
	NotificationRuntime *NotificationRuntime
	// AI Agent runtime owns AI chat, candidate chat, provider fallback,
	// embedding workload execution, and durable agent-run execution.
	AIAgentRuntime *AIAgentRuntime
	// Background workers (caller must Start/Stop)
	OutboxPublisher      *OutboxPublisher
	NotificationConsumer *NotificationConsumer
	ResumeParseConsumer  *ResumeParseConsumer
	EmailConsumer        *EmailConsumer
	EmbeddingConsumer    *EmbeddingConsumer
	AgentRunConsumer     *AgentRunConsumer
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
	embeddingModelRepo := repository.NewEmbeddingModelRepo(db)
	embeddingProviderRepo := repository.NewEmbeddingProviderRepo(db)

	var embeddingProviderFactory *EmbeddingProviderFactory
	encKey, encErr := crypto.LoadEncryptionKey()
	if encErr == nil {
		embeddingProviderFactory = NewEmbeddingProviderFactory(cfg, embeddingModelRepo, embeddingProviderRepo)
	} else {
		logger.L().Warn("[embedding] encryption key not set, using unavailable provider", zap.Error(encErr))
	}
	embeddingSvc := NewEmbeddingService(repository.NewAIEmbeddingRepo(db), embeddingProviderFactory, encKey).WithRuntimePolicy(runtimePolicy)
	contextBuilder := NewAgentContextBuilder(chats, summaries, memories, aiClient, cfg, repository.NewPromptTemplateRepo(db)).WithEmbeddingService(embeddingSvc)
	agentRuntime := cfg.AI.AgentRuntime
	analyticsRepo := repository.NewAnalyticsRepo(db)
	usageAuditCtxRepo := repository.NewUsageAuditContextRepo(db)
	taxonomy := NewJobTaxonomyService(departments, locations, jobs, deptLocs)

	inboxRepo := repository.NewInboxRepo(db)
	resumeParseConsumer := NewResumeParseConsumer(resumes, ossClient).WithInbox(inboxRepo)
	embeddingConsumer := NewEmbeddingConsumer(embeddingSvc).WithInbox(inboxRepo)
	scopeEval := &scopeEvaluator{authzRepo: authzRepo}
	serviceAuth := NewServiceAuthorizer(authzRepo, scopeEval)
	notificationRuntime := NewNotificationRuntime(NotificationRuntimeDeps{
		Users:         users,
		Notifications: notifications,
		Outbox:        outbox,
		Inbox:         inboxRepo,
		EmailLog:      emailLogRepo,
		Cache:         notifCache,
		MQ:            mqConn,
		Authz:         serviceAuth,
		EmailRenderer: emailRenderer,
		EmailSender:   emailSender,
	})

	collaborationRepo := repository.NewCollaborationRepo(db)
	llmConfigSvc := newLlmConfigServiceWithFallback(db)
	agentCfgRepo := repository.NewAgentConfigRepo(db)
	promptTmplRepo := repository.NewPromptTemplateRepo(db)
	candidateAI := NewCandidateAIService(usageLogs, usageAuditCtxRepo, authzRepo, chats, applications, jobs, resumes, aiClient, candidateToolExecutor, agentRuntime, toolTraces, summaries, promptTmplRepo, agentCfgRepo)

	// Initialize MCP service before AI service for MCP tool injection
	mcpSvc := NewMCPService(repository.NewMCPRepo(db), cfg).WithRuntimePolicy(runtimePolicy)
	skillSvc := NewSkillService(repository.NewSkillRepo(db))
	embeddingEventPublisher := NewEmbeddingEventPublisher(mqConn)
	agentSkillSvc := NewAgentSkillServiceWithAgentConfigRepo(repository.NewAgentSkillRepo(db), agentCfgRepo).
		WithSemanticDebugDependencies(memories, embeddingSvc).
		WithEmbeddingEventPublisher(embeddingEventPublisher)
	agentSkillRepo := repository.NewAgentSkillRepo(db)
	backfillSvc := NewEmbeddingBackfillService(db, embeddingSvc, agentSkillRepo, memories)
	var embeddingConfigSvc *EmbeddingConfigService
	if encErr == nil {
		embeddingConfigSvc = NewEmbeddingConfigService(embeddingProviderRepo, embeddingModelRepo, encKey, backfillSvc, embeddingSvc)
	}
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
	aiSvc := NewAIService(chats, applications, jobs, resumes, summaries, toolTraces, agentRuns, memories, ossClient, aiClient, toolExecutor, contextBuilder, candidateAI, usageLogs, usageAuditCtxRepo, authzRepo, agentRuntime, serviceAuth, llmConfigSvc, agentCfgRepo, promptTmplRepo, mcpSvc, skillSvc, agentSkillRepo).
		WithAgentRunEventRepo(repository.NewAgentRunEventRepo(db)).
		WithAgentRunDispatcher(notificationRuntime.OutboxPublisher).
		WithEmbeddingService(embeddingSvc).
		WithEmbeddingEventPublisher(embeddingEventPublisher).
		WithRuntimePolicy(runtimePolicy)
	agentRunConsumer := NewAgentRunConsumer(aiSvc).WithInbox(inboxRepo)
	aiAgentRuntime := NewAIAgentRuntime(AIAgentRuntimeDeps{
		AI:                aiSvc,
		CandidateAI:       candidateAI,
		LlmConfig:         llmConfigSvc,
		Prompt:            NewPromptService(promptTmplRepo),
		AgentConfig:       NewAgentConfigService(agentCfgRepo, promptTmplRepo, mcpSvc, skillSvc),
		MCP:               mcpSvc,
		Skill:             skillSvc,
		AgentSkill:        agentSkillSvc,
		ResumeProfile:     resumeProfileSvc,
		CandidateMatch:    candidateMatchSvc,
		Intelligence:      recruitingIntelligenceSvc,
		Embedding:         embeddingSvc,
		EmbeddingConfig:   embeddingConfigSvc,
		EmbeddingConsumer: embeddingConsumer,
		AgentRunConsumer:  agentRunConsumer,
		RuntimePolicy:     runtimePolicy,
		RuntimeName:       agentRuntime,
	})

	return &Services{
		Auth:              NewAuthService(users, tokens, authzRepo, inviteCodes, jwtSecret),
		Analytics:         NewAnalyticsService(analyticsRepo, authzRepo, serviceAuth),
		Admin:             NewAdminService(inviteCodes, usageLogs, users, authzRepo, tokenCache, serviceAuth),
		UsageAuditCtxRepo: usageAuditCtxRepo,

		// P1-003: AI usage statistics
		UsageStats: NewUsageStatsService(repository.NewUsageStatsRepo(db), serviceAuth),

		Job:                    NewJobService(jobs, jobCache, authzRepo, taxonomy, scopeEval),
		Taxonomy:               taxonomy,
		Candidate:              NewCandidateService(profiles, resumes, ossClient, notificationRuntime.OutboxPublisher, usageLogs, serviceAuth),
		Application:            NewApplicationService(authzRepo, applications, profiles, resumes, jobs, interviews, notifications, notificationRuntime.OutboxPublisher, ossClient, jobCache, scopeEval),
		Interview:              NewInterviewService(authzRepo, interviews, users, applications, jobs, notifications, notificationRuntime.OutboxPublisher, ossClient, scopeEval, serviceAuth),
		Offer:                  NewOfferService(authzRepo, offers, applications, jobs, notifications, notificationRuntime.OutboxPublisher, scopeEval, serviceAuth),
		AI:                     aiSvc,
		CandidateAI:            candidateAI,
		Notification:           notificationRuntime.Notification,
		NotificationRuntime:    notificationRuntime,
		AIAgentRuntime:         aiAgentRuntime,
		LlmConfig:              aiAgentRuntime.LlmConfig,
		Prompt:                 aiAgentRuntime.Prompt,
		AgentConfig:            aiAgentRuntime.AgentConfig,
		MCP:                    aiAgentRuntime.MCP,
		Skill:                  aiAgentRuntime.Skill,
		AgentSkill:             aiAgentRuntime.AgentSkill,
		ResumeProfile:          aiAgentRuntime.ResumeProfile,
		CandidateMatch:         aiAgentRuntime.CandidateMatch,
		RecruitingIntelligence: aiAgentRuntime.Intelligence,
		Embedding:              aiAgentRuntime.Embedding,
		EmbeddingConfig:        aiAgentRuntime.EmbeddingConfig,

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

		OutboxPublisher:      notificationRuntime.OutboxPublisher,
		NotificationConsumer: notificationRuntime.NotificationConsumer,
		ResumeParseConsumer:  resumeParseConsumer,
		EmailConsumer:        notificationRuntime.EmailConsumer,
		EmbeddingConsumer:    aiAgentRuntime.EmbeddingConsumer,
		AgentRunConsumer:     aiAgentRuntime.AgentRunConsumer,
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
