package service

import (
	"context"
	"errors"
	"time"

	"smart-recruit-ai-agent-service/internal/application/command"
	"smart-recruit-ai-agent-service/internal/application/dto"
	"smart-recruit-ai-agent-service/internal/domain/model"
	"smart-recruit-ai-agent-service/internal/domain/policy"
	"smart-recruit-ai-agent-service/internal/domain/repository"
)

var ErrSkillRepositoryRequired = errors.New("skill repository is required")

type MCPPolicyDeps struct {
	Policies repository.MCPPolicyRepository
	Now      func() time.Time
}

type MCPPolicyService struct {
	policies repository.MCPPolicyRepository
	now      func() time.Time
}

func NewMCPPolicyService(deps MCPPolicyDeps) *MCPPolicyService {
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &MCPPolicyService{policies: deps.Policies, now: now}
}

func (s *MCPPolicyService) Evaluate(ctx context.Context, cmd command.EvaluateMCPToolPolicy) (dto.MCPPolicyResult, error) {
	var toolPolicy *model.MCPToolPolicy
	var recentCalls int64
	if s.policies != nil {
		p, err := s.policies.GetEnabledToolPolicy(ctx, cmd.ServerID, cmd.ToolName)
		if err != nil {
			return dto.MCPPolicyResult{}, err
		}
		toolPolicy = p
		if p != nil && p.RateLimitWindowSeconds > 0 && p.RateLimitMaxCalls > 0 {
			since := s.now().Add(-time.Duration(p.RateLimitWindowSeconds) * time.Second)
			count, err := s.policies.CountToolCallsSince(ctx, cmd.ServerID, cmd.ToolName, since)
			if err != nil {
				return dto.MCPPolicyResult{}, err
			}
			recentCalls = count
		}
	}
	evaluation, err := policy.EvaluateMCPToolPolicy(toolPolicy, model.MCPPolicyContext{
		ServerID:             cmd.ServerID,
		ToolName:             cmd.ToolName,
		CallerRole:           cmd.CallerRole,
		CallerScope:          cmd.CallerScope,
		ConfirmationApproved: cmd.ConfirmationApproved,
		RecentCalls:          recentCalls,
		Args:                 cmd.Args,
		Now:                  s.now(),
	})
	if err != nil {
		return dto.MCPPolicyResult{}, err
	}
	return dto.MCPPolicyResult{Evaluation: evaluation}, nil
}

type SkillDeps struct {
	Skills repository.SkillRepository
	Audit  repository.AuditSink
	Now    func() time.Time
}

type SkillService struct {
	skills repository.SkillRepository
	audit  repository.AuditSink
	now    func() time.Time
}

func NewSkillService(deps SkillDeps) (*SkillService, error) {
	if deps.Skills == nil {
		return nil, ErrSkillRepositoryRequired
	}
	if deps.Audit == nil {
		return nil, ErrAuditSinkRequired
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &SkillService{skills: deps.Skills, audit: deps.Audit, now: now}, nil
}

func (s *SkillService) Select(ctx context.Context, cmd command.SelectAgentSkills) (dto.AgentSkillSelectionResult, error) {
	candidates, err := s.skills.ListEnabledAgentSkills(ctx)
	if err != nil {
		return dto.AgentSkillSelectionResult{}, err
	}
	selected := policy.SelectAgentSkills(candidates, model.AgentSkillSelectionRequest{
		AgentType:             cmd.AgentType,
		Question:              cmd.Question,
		ManualIDs:             cmd.ManualIDs,
		AvailableCapabilities: cmd.AvailableCapabilities,
		SemanticScores:        cmd.SemanticScores,
		MaxSkills:             cmd.MaxSkills,
	})
	return dto.AgentSkillSelectionResult{Selected: selected}, nil
}

func (s *SkillService) CreateVersion(ctx context.Context, cmd command.CreateSkillVersion) (dto.SkillVersionResult, error) {
	if err := policy.ValidateSkillManifest(cmd.Manifest); err != nil {
		return dto.SkillVersionResult{}, err
	}
	next, changed := policy.NextSkillVersion(cmd.Current, cmd.Changed)
	version := model.SkillVersion{
		SkillID:    cmd.SkillID,
		Version:    next,
		Manifest:   cmd.Manifest,
		Content:    cmd.Content,
		Active:     cmd.Activate,
		ChangedBy:  cmd.ActorID,
		ChangeNote: cmd.ChangeNote,
		CreatedAt:  s.now(),
	}
	if changed {
		created, err := s.skills.CreateSkillVersion(ctx, version)
		if err != nil {
			return dto.SkillVersionResult{}, err
		}
		version = *created
	}
	if cmd.Activate {
		if err := s.skills.ActivateSkillVersion(ctx, cmd.SkillID, version.Version); err != nil {
			return dto.SkillVersionResult{}, err
		}
	}
	if err := s.audit.RecordAudit(ctx, model.AuditEvent{ActorID: cmd.ActorID, Operation: "skill.version.create", ResourceType: "skill", ResourceID: cmd.SkillID, Decision: "allow", OccurredAt: s.now()}); err != nil {
		return dto.SkillVersionResult{}, err
	}
	return dto.SkillVersionResult{Version: version, VersionCreated: changed, Activated: cmd.Activate}, nil
}

type EmbeddingRuntimeService struct {
	config repository.EmbeddingConfigRepository
}

func NewEmbeddingRuntimeService(config repository.EmbeddingConfigRepository) *EmbeddingRuntimeService {
	return &EmbeddingRuntimeService{config: config}
}

func (s *EmbeddingRuntimeService) Resolve(ctx context.Context, _ command.ResolveEmbeddingRuntime) (dto.EmbeddingRuntimeResult, error) {
	if s == nil || s.config == nil {
		return dto.EmbeddingRuntimeResult{State: policy.ResolveEmbeddingRuntime(nil, nil, "embedding_config_missing")}, nil
	}
	provider, err := s.config.GetDefaultEmbeddingProvider(ctx)
	if err != nil {
		return dto.EmbeddingRuntimeResult{}, err
	}
	if provider == nil {
		return dto.EmbeddingRuntimeResult{State: policy.ResolveEmbeddingRuntime(nil, nil, "embedding_provider_missing")}, nil
	}
	embeddingModel, err := s.config.GetDefaultEmbeddingModel(ctx, provider.ID)
	if err != nil {
		return dto.EmbeddingRuntimeResult{}, err
	}
	return dto.EmbeddingRuntimeResult{State: policy.ResolveEmbeddingRuntime(provider, embeddingModel, "embedding_model_missing")}, nil
}

type CandidateMatchService struct {
	matches repository.CandidateMatchRepository
}

func NewCandidateMatchService(matches repository.CandidateMatchRepository) *CandidateMatchService {
	return &CandidateMatchService{matches: matches}
}

func (s *CandidateMatchService) Aggregate(ctx context.Context, cmd command.AggregateCandidateMatch) (dto.CandidateMatchResult, error) {
	aggregation, err := policy.AggregateCandidateMatch(cmd.Profile, cmd.Results, cmd.FallbackUsed)
	if err != nil {
		return dto.CandidateMatchResult{}, err
	}
	if cmd.Persist && s != nil && s.matches != nil {
		if err := s.matches.SaveCandidateMatchAggregation(ctx, cmd.ApplicationID, aggregation); err != nil {
			return dto.CandidateMatchResult{}, err
		}
		return dto.CandidateMatchResult{Aggregation: aggregation, Persisted: true}, nil
	}
	return dto.CandidateMatchResult{Aggregation: aggregation}, nil
}
