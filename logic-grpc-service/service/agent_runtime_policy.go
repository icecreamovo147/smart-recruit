package service

import (
	"context"
	"errors"
	"time"

	"logic-grpc-service/config"
)

var ErrAgentCapabilityDisabled = errors.New("agent capability disabled")

type AgentRuntimePolicy struct {
	Planner                  bool
	StructuredResumeParse    bool
	CandidateMatch           bool
	SkillGovernance          bool
	SemanticRetrieval        bool
	MCPPolicy                bool
	Fallbacks                bool
	ResumeParseTimeout       time.Duration
	CandidateMatchTimeout    time.Duration
	SemanticRetrievalTimeout time.Duration
}

func NewAgentRuntimePolicy(cfg config.Config) AgentRuntimePolicy {
	return AgentRuntimePolicy{
		Planner:                  boolValue(cfg.Agent.Features.Planner, true),
		StructuredResumeParse:    boolValue(cfg.Agent.Features.StructuredResumeParse, true),
		CandidateMatch:           boolValue(cfg.Agent.Features.CandidateMatch, true),
		SkillGovernance:          boolValue(cfg.Agent.Features.SkillGovernance, true),
		SemanticRetrieval:        boolValue(cfg.Agent.Features.SemanticRetrieval, true),
		MCPPolicy:                boolValue(cfg.Agent.Features.MCPPolicy, true),
		Fallbacks:                boolValue(cfg.Agent.Features.Fallbacks, true),
		ResumeParseTimeout:       cfg.Agent.Features.ResumeParseTimeout.Duration,
		CandidateMatchTimeout:    cfg.Agent.Features.CandidateMatchTimeout.Duration,
		SemanticRetrievalTimeout: cfg.Agent.Features.SemanticRetrievalTimeout.Duration,
	}
}

func DefaultAgentRuntimePolicy() AgentRuntimePolicy {
	return AgentRuntimePolicy{
		Planner:                  true,
		StructuredResumeParse:    true,
		CandidateMatch:           true,
		SkillGovernance:          true,
		SemanticRetrieval:        true,
		MCPPolicy:                true,
		Fallbacks:                true,
		ResumeParseTimeout:       30 * time.Second,
		CandidateMatchTimeout:    15 * time.Second,
		SemanticRetrievalTimeout: 5 * time.Second,
	}
}

func (p AgentRuntimePolicy) withDefaults() AgentRuntimePolicy {
	defaults := DefaultAgentRuntimePolicy()
	if p.ResumeParseTimeout <= 0 {
		p.ResumeParseTimeout = defaults.ResumeParseTimeout
	}
	if p.CandidateMatchTimeout <= 0 {
		p.CandidateMatchTimeout = defaults.CandidateMatchTimeout
	}
	if p.SemanticRetrievalTimeout <= 0 {
		p.SemanticRetrievalTimeout = defaults.SemanticRetrievalTimeout
	}
	return p
}

func contextWithPolicyTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return parent, func() {}
	}
	if _, ok := parent.Deadline(); ok {
		return parent, func() {}
	}
	return context.WithTimeout(parent, timeout)
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
