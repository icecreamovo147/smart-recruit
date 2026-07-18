package dto

import "smart-recruit-ai-agent-service/internal/domain/model"

type AgentRunResult struct {
	Run        model.AgentRun
	Idempotent bool
	Dispatched bool
}

type PromptResult struct {
	Template       model.PromptTemplate
	VersionCreated bool
}
