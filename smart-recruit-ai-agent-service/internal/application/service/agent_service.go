package service

import (
	"context"
	"errors"
	"time"

	"smart-recruit-ai-agent-service/internal/application/command"
	"smart-recruit-ai-agent-service/internal/application/dto"
	"smart-recruit-ai-agent-service/internal/application/port"
	"smart-recruit-ai-agent-service/internal/domain/model"
	"smart-recruit-ai-agent-service/internal/domain/policy"
	"smart-recruit-ai-agent-service/internal/domain/repository"
)

var (
	ErrChatSessionRepositoryRequired = errors.New("chat session repository is required")
	ErrAgentRunRepositoryRequired    = errors.New("agent run repository is required")
	ErrPromptRepositoryRequired      = errors.New("prompt repository is required")
	ErrDispatcherRequired            = errors.New("durable run dispatcher is required")
	ErrAuditSinkRequired             = errors.New("audit sink is required")
	ErrSessionForbidden              = errors.New("chat session not found or forbidden")
)

type AgentRunDeps struct {
	Sessions   repository.ChatSessionRepository
	Runs       repository.AgentRunRepository
	Dispatcher port.DurableRunDispatcher
	Audit      repository.AuditSink
	Now        func() time.Time
}

type AgentRunService struct {
	sessions   repository.ChatSessionRepository
	runs       repository.AgentRunRepository
	dispatcher port.DurableRunDispatcher
	audit      repository.AuditSink
	now        func() time.Time
}

func NewAgentRunService(deps AgentRunDeps) (*AgentRunService, error) {
	if deps.Sessions == nil {
		return nil, ErrChatSessionRepositoryRequired
	}
	if deps.Runs == nil {
		return nil, ErrAgentRunRepositoryRequired
	}
	if deps.Dispatcher == nil {
		return nil, ErrDispatcherRequired
	}
	if deps.Audit == nil {
		return nil, ErrAuditSinkRequired
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &AgentRunService{sessions: deps.Sessions, runs: deps.Runs, dispatcher: deps.Dispatcher, audit: deps.Audit, now: now}, nil
}

func (s *AgentRunService) Create(ctx context.Context, cmd command.CreateAgentRun) (dto.AgentRunResult, error) {
	if err := policy.ValidateAgentRunRequest(cmd.SessionID, cmd.ActorID, cmd.ClientRequestID, cmd.Message, cmd.ActionType); err != nil {
		return dto.AgentRunResult{}, err
	}
	if existing, err := s.runs.FindByClientRequestID(ctx, cmd.ActorID, cmd.ClientRequestID); err != nil {
		return dto.AgentRunResult{}, err
	} else if existing != nil {
		return dto.AgentRunResult{Run: *existing, Idempotent: true}, nil
	}
	session, err := s.sessions.GetSessionForOwner(ctx, cmd.SessionID, cmd.ActorID)
	if err != nil {
		return dto.AgentRunResult{}, err
	}
	if session == nil {
		return dto.AgentRunResult{}, ErrSessionForbidden
	}
	now := s.now()
	run, err := s.runs.CreateRun(ctx, model.AgentRun{
		SessionID:       cmd.SessionID,
		ActorID:         cmd.ActorID,
		AgentType:       defaultString(cmd.AgentType, "hr"),
		AgentName:       defaultString(cmd.AgentName, "hr_recruiting_agent"),
		ModelID:         cmd.ModelID,
		ModelName:       cmd.ModelName,
		Status:          model.AgentRunStatusQueued,
		ClientRequestID: cmd.ClientRequestID,
		Message:         cmd.Message,
		ActionType:      cmd.ActionType,
		StartedAt:       now,
	})
	if err != nil {
		return dto.AgentRunResult{}, err
	}
	if err := s.runs.AppendEvent(ctx, model.AgentRunEvent{RunID: run.ID, EventType: "run.created", Payload: `{"status":"queued"}`, CreatedAt: now}); err != nil {
		return dto.AgentRunResult{}, err
	}
	planning, err := s.runs.UpdateRunStatus(ctx, run.ID, model.AgentRunStatusQueued, model.AgentRunStatusPlanning)
	if err != nil {
		return dto.AgentRunResult{}, err
	}
	if err := s.dispatcher.DispatchAgentRun(ctx, planning.ID); err != nil {
		return dto.AgentRunResult{}, err
	}
	if err := s.audit.RecordAudit(ctx, model.AuditEvent{ActorID: cmd.ActorID, Operation: "agent_run.create", ResourceType: "agent_run", ResourceID: planning.ID, Decision: "allow", OccurredAt: now}); err != nil {
		return dto.AgentRunResult{}, err
	}
	return dto.AgentRunResult{Run: *planning, Dispatched: true}, nil
}

func (s *AgentRunService) Cancel(ctx context.Context, cmd command.CancelAgentRun) (dto.AgentRunResult, error) {
	run, err := s.runs.GetOwnedRun(ctx, cmd.RunID, cmd.ActorID)
	if err != nil {
		return dto.AgentRunResult{}, err
	}
	if run.IsTerminal() || run.Status == model.AgentRunStatusCancelRequested {
		return dto.AgentRunResult{Run: *run, Idempotent: true}, nil
	}
	if err := policy.ValidateAgentRunTransition(run.Status, model.AgentRunStatusCancelRequested); err != nil {
		return dto.AgentRunResult{}, err
	}
	next, err := s.runs.UpdateRunStatus(ctx, run.ID, run.Status, model.AgentRunStatusCancelRequested)
	if err != nil {
		return dto.AgentRunResult{}, err
	}
	if err := s.audit.RecordAudit(ctx, model.AuditEvent{ActorID: cmd.ActorID, Operation: "agent_run.cancel", ResourceType: "agent_run", ResourceID: run.ID, Decision: "allow", OccurredAt: s.now()}); err != nil {
		return dto.AgentRunResult{}, err
	}
	return dto.AgentRunResult{Run: *next}, nil
}

func (s *AgentRunService) Confirm(ctx context.Context, cmd command.ConfirmAgentRun) (dto.AgentRunResult, error) {
	run, err := s.runs.GetOwnedRun(ctx, cmd.RunID, cmd.ActorID)
	if err != nil {
		return dto.AgentRunResult{}, err
	}
	if run.Status != model.AgentRunStatusWaitingConfirmation {
		if run.IsTerminal() || run.IsActive() {
			return dto.AgentRunResult{Run: *run, Idempotent: true}, nil
		}
		return dto.AgentRunResult{}, policy.ErrIllegalAgentRunTransition
	}
	next, err := s.runs.UpdateRunStatus(ctx, run.ID, model.AgentRunStatusWaitingConfirmation, model.AgentRunStatusRunning)
	if err != nil {
		return dto.AgentRunResult{}, err
	}
	if err := s.runs.AppendEvent(ctx, model.AgentRunEvent{RunID: run.ID, EventType: "confirmation.accepted", Payload: `{}`, CreatedAt: s.now()}); err != nil {
		return dto.AgentRunResult{}, err
	}
	if err := s.dispatcher.DispatchAgentRun(ctx, next.ID); err != nil {
		return dto.AgentRunResult{}, err
	}
	if err := s.audit.RecordAudit(ctx, model.AuditEvent{ActorID: cmd.ActorID, Operation: "agent_run.confirm", ResourceType: "agent_run", ResourceID: run.ID, Decision: "allow", OccurredAt: s.now()}); err != nil {
		return dto.AgentRunResult{}, err
	}
	return dto.AgentRunResult{Run: *next, Dispatched: true}, nil
}

type PromptDeps struct {
	Prompts repository.PromptRepository
	Audit   repository.AuditSink
	Now     func() time.Time
}

type PromptService struct {
	prompts repository.PromptRepository
	audit   repository.AuditSink
	now     func() time.Time
}

func NewPromptService(deps PromptDeps) (*PromptService, error) {
	if deps.Prompts == nil {
		return nil, ErrPromptRepositoryRequired
	}
	if deps.Audit == nil {
		return nil, ErrAuditSinkRequired
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &PromptService{prompts: deps.Prompts, audit: deps.Audit, now: now}, nil
}

func (s *PromptService) CreateTemplate(ctx context.Context, cmd command.CreatePromptTemplate) (dto.PromptResult, error) {
	role := defaultString(cmd.Role, model.PromptRoleSystem)
	template := model.PromptTemplate{Name: cmd.Name, Content: cmd.Content, Variables: cmd.Variables, Version: 1, IsActive: true, AgentType: cmd.AgentType, Role: role, CreatedBy: cmd.ActorID, UpdatedBy: cmd.ActorID}
	if err := policy.ValidatePrompt(template); err != nil {
		return dto.PromptResult{}, err
	}
	created, err := s.prompts.CreateTemplate(ctx, template, model.PromptVersion{Version: 1, Content: template.Content, ChangedBy: cmd.ActorID, ChangeNote: "initial version"})
	if err != nil {
		return dto.PromptResult{}, err
	}
	if err := s.audit.RecordAudit(ctx, model.AuditEvent{ActorID: cmd.ActorID, Operation: "prompt.create", ResourceType: "prompt_template", ResourceID: created.ID, Decision: "allow", OccurredAt: s.now()}); err != nil {
		return dto.PromptResult{}, err
	}
	return dto.PromptResult{Template: *created, VersionCreated: true}, nil
}

func (s *PromptService) UpdateTemplate(ctx context.Context, cmd command.UpdatePromptTemplate) (dto.PromptResult, error) {
	existing, err := s.prompts.GetTemplate(ctx, cmd.ID)
	if err != nil {
		return dto.PromptResult{}, err
	}
	updated := *existing
	if cmd.Name != "" {
		updated.Name = cmd.Name
	}
	if cmd.Variables != "" {
		updated.Variables = cmd.Variables
	}
	if cmd.IsActive != nil {
		updated.IsActive = *cmd.IsActive
	}
	updated.UpdatedBy = cmd.ActorID
	nextVersion, contentChanged := policy.NextPromptVersion(*existing, cmd.Content)
	var version *model.PromptVersion
	if contentChanged {
		updated.Content = cmd.Content
		updated.Version = nextVersion
		version = &model.PromptVersion{TemplateID: cmd.ID, Version: nextVersion, Content: cmd.Content, ChangedBy: cmd.ActorID, ChangeNote: "content updated"}
	}
	if err := policy.ValidatePrompt(updated); err != nil {
		return dto.PromptResult{}, err
	}
	saved, err := s.prompts.UpdateTemplate(ctx, updated, version)
	if err != nil {
		return dto.PromptResult{}, err
	}
	if err := s.audit.RecordAudit(ctx, model.AuditEvent{ActorID: cmd.ActorID, Operation: "prompt.update", ResourceType: "prompt_template", ResourceID: saved.ID, Decision: "allow", OccurredAt: s.now()}); err != nil {
		return dto.PromptResult{}, err
	}
	return dto.PromptResult{Template: *saved, VersionCreated: contentChanged}, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
