package service

import (
	"logic-grpc-service/ai"
	"logic-grpc-service/config"
	"logic-grpc-service/mq"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/repository"
)

// Services aggregates all domain services and background workers.
type Services struct {
	Auth         *AuthService
	Job          *JobService
	Candidate    *CandidateService
	Application  *ApplicationService
	AI           *AIService
	CandidateAI  *CandidateAIService
	Notification *NotificationService
	Admin        *AdminService
	Taxonomy     *JobTaxonomyService

	// Background workers (caller must Start/Stop)
	OutboxPublisher      *OutboxPublisher
	NotificationConsumer *NotificationConsumer
	ResumeParseConsumer  *ResumeParseConsumer
}

func NewServices(
	users *repository.UserRepo,
	jobs *repository.JobRepo,
	profiles *repository.ProfileRepo,
	resumes *repository.ResumeRepo,
	applications *repository.ApplicationRepo,
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
	notifCache *cache.NotificationCache,
	jobCache *cache.JobCache,
	ossClient *oss.Client,
	aiClient *ai.Client,
	mqConn *mq.Conn,
	cfg config.Config,
	jwtSecret string,
) *Services {
	toolExecutor := ai.NewToolExecutor(applications, jobs, resumes, ossClient)
	candidateToolExecutor := ai.NewCandidateToolExecutor(applications, jobs, resumes)
	contextBuilder := NewAgentContextBuilder(chats, summaries, memories, aiClient, cfg)
	candidateAI := NewCandidateAIService(chats, applications, jobs, resumes, aiClient, candidateToolExecutor)
	taxonomy := NewJobTaxonomyService(departments, locations, jobs, deptLocs)

	outboxPublisher := NewOutboxPublisher(outbox, mqConn)
	notificationConsumer := NewNotificationConsumer(notifications, notifCache)
	resumeParseConsumer := NewResumeParseConsumer(resumes, ossClient)

	return &Services{
		Auth:         NewAuthService(users, jwtSecret),
		Admin:        NewAdminService(inviteCodes),
		Job:          NewJobService(jobs, jobCache, taxonomy),
		Taxonomy:     taxonomy,
		Candidate:    NewCandidateService(profiles, resumes, ossClient, outboxPublisher),
		Application:  NewApplicationService(applications, profiles, resumes, jobs, notifications, outboxPublisher, ossClient, jobCache),
		AI:           NewAIService(chats, applications, jobs, resumes, summaries, toolTraces, memories, ossClient, aiClient, toolExecutor, contextBuilder, candidateAI),
		CandidateAI:  candidateAI,
		Notification: NewNotificationService(notifications, notifCache),

		OutboxPublisher:      outboxPublisher,
		NotificationConsumer: notificationConsumer,
		ResumeParseConsumer:  resumeParseConsumer,
	}
}
