package repository

import (
	"context"

	"smart-recruit-ai-agent-service/internal/domain/model"
)

type ChatSessionRepository interface {
	GetSessionForOwner(ctx context.Context, sessionID, ownerID uint64) (*model.ChatSession, error)
	AppendMessage(ctx context.Context, message model.ChatMessage) error
}

type AgentRunRepository interface {
	FindByClientRequestID(ctx context.Context, actorID uint64, clientRequestID string) (*model.AgentRun, error)
	CreateRun(ctx context.Context, run model.AgentRun) (*model.AgentRun, error)
	GetOwnedRun(ctx context.Context, runID, actorID uint64) (*model.AgentRun, error)
	UpdateRunStatus(ctx context.Context, runID uint64, from, to string) (*model.AgentRun, error)
	AppendEvent(ctx context.Context, event model.AgentRunEvent) error
}

type PromptRepository interface {
	GetTemplate(ctx context.Context, templateID uint64) (*model.PromptTemplate, error)
	CreateTemplate(ctx context.Context, template model.PromptTemplate, version model.PromptVersion) (*model.PromptTemplate, error)
	UpdateTemplate(ctx context.Context, template model.PromptTemplate, version *model.PromptVersion) (*model.PromptTemplate, error)
}

type AuditSink interface {
	RecordAudit(ctx context.Context, event model.AuditEvent) error
}
