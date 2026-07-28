package command

import "smart-recruit-ai-agent-service/internal/domain/model"

type EvaluateMCPToolPolicy struct {
	ServerID             uint64
	ToolName             string
	CallerRole           string
	CallerScope          string
	ConfirmationApproved bool
	Args                 map[string]any
}

type CreateSkillVersion struct {
	ActorID    uint64
	SkillID    uint64
	Current    int64
	Manifest   model.SkillManifest
	Content    string
	Changed    bool
	Activate   bool
	ChangeNote string
}

type ResolveEmbeddingRuntime struct{}

type AggregateCandidateMatch struct {
	ApplicationID uint64
	Profile       model.CandidateMatchProfile
	Results       []model.CandidateRequirementResult
	FallbackUsed  bool
	Persist       bool
}
