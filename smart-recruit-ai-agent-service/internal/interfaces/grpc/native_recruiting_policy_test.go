package grpc

import (
	"testing"
	"time"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
)

func TestNewNativeRuntimeDepsInjectsRecruitingPolicy(t *testing.T) {
	store := &structuredRuntimeStore{fakeAIStore: newFakeAIStore()}
	policy := recruitingruntime.NewRuntimePolicy(recruitingruntime.RuntimePolicyConfig{
		CandidateMatch:        true,
		CandidateMatchShadow:  true,
		Fallbacks:             false,
		ResumeParseTimeout:    7 * time.Second,
		CandidateMatchTimeout: 11 * time.Second,
	})
	deps := NewNativeRuntimeDeps(RuntimeDeps{Store: store, Provider: store, RecruitingPolicy: policy})
	service, ok := deps.RecruitingIntelligence.(nativeRecruitingIntelligenceService)
	if !ok {
		t.Fatalf("recruiting service type = %T", deps.RecruitingIntelligence)
	}
	if service.structured == nil {
		t.Fatal("structured recruiting runtime was not wired")
	}
	if !service.policy.CandidateMatchShadowEnabled() || service.policy.FallbacksEnabled() {
		t.Fatal("native service did not receive exact policy")
	}
	if service.structured.Policy().ResumeParseTimeout() != 7*time.Second || service.structured.Policy().CandidateMatchTimeout() != 11*time.Second {
		t.Fatal("structured runtime did not receive policy timeouts")
	}
}
