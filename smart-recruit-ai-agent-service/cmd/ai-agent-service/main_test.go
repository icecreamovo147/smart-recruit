package main

import (
	"context"
	"slices"
	"testing"
	"time"

	recruitingruntime "smart-recruit-ai-agent-service/internal/application/recruiting_intelligence"
	aiagentruntime "smart-recruit-ai-agent-service/internal/runtime"
	platformconfig "smart-recruit-platform-go/config"
	logicconfig "smart-recruit-platform-go/serviceconfig"
)

func TestTenantBoundaryExcludesOwnerScopedBillingOutbox(t *testing.T) {
	const table = "ai_billing_settlement_outbox"
	if slices.Contains(aiAgentTenantOwnedTables, table) || slices.Contains(aiAgentMixedScopeTables, table) {
		t.Fatalf("%s must not use tenant_id scoping; ownership is represented by owner_type and owner_id", table)
	}
}

func TestInstanceFromAddrUsesAIAgentDiscoveryName(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50066", platformconfig.Bootstrap{
		ServiceName:    "ai-agent-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	if instance.ServiceName != "ai-agent" {
		t.Fatalf("ServiceName = %q, want ai-agent", instance.ServiceName)
	}
	if instance.Port != 50066 {
		t.Fatalf("Port = %d, want 50066", instance.Port)
	}
	if instance.Metadata["service"] != "ai-agent-service" {
		t.Fatalf("metadata service = %q", instance.Metadata["service"])
	}
}

func TestRecruitingRuntimePolicyMapsAllExistingSettings(t *testing.T) {
	var cfg logicconfig.Config
	structuredResume := false
	candidateMatch := true
	semantic := false
	shadow := true
	fallbacks := false
	cfg.Agent.Features.StructuredResumeParse = &structuredResume
	cfg.Agent.Features.CandidateMatch = &candidateMatch
	cfg.Agent.Features.CandidateMatchSemantic = &semantic
	cfg.Agent.Features.CandidateMatchShadow = &shadow
	cfg.Agent.Features.Fallbacks = &fallbacks
	cfg.Agent.Features.ResumeParseTimeout.Duration = 7 * time.Second
	cfg.Agent.Features.CandidateMatchTimeout.Duration = 11 * time.Second

	policy := recruitingRuntimePolicy(cfg)
	if policy.StructuredResumeParseEnabled() || !policy.CandidateMatchEnabled() || policy.CandidateMatchSemanticEnabled() || !policy.CandidateMatchShadowEnabled() || policy.FallbacksEnabled() {
		t.Fatal("feature flags were not mapped exactly")
	}
	if policy.ResumeParseTimeout() != 7*time.Second || policy.CandidateMatchTimeout() != 11*time.Second {
		t.Fatalf("timeouts = %v/%v", policy.ResumeParseTimeout(), policy.CandidateMatchTimeout())
	}
}

func TestRecruitingRuntimePolicyUsesDeterministicDefaultsForUnsetSettings(t *testing.T) {
	policy := recruitingRuntimePolicy(logicconfig.Config{})
	if !policy.StructuredResumeParseEnabled() || !policy.CandidateMatchEnabled() || !policy.CandidateMatchSemanticEnabled() || policy.CandidateMatchShadowEnabled() || !policy.FallbacksEnabled() {
		t.Fatal("unset flags did not resolve to service defaults")
	}
	if policy.ResumeParseTimeout() != 30*time.Second || policy.CandidateMatchTimeout() != 15*time.Second {
		t.Fatalf("timeouts = %v/%v", policy.ResumeParseTimeout(), policy.CandidateMatchTimeout())
	}
}

func TestAgentSkillPackageFeaturesUseFailClosedDefaults(t *testing.T) {
	var cfg logicconfig.Config
	if boolSetting(cfg.Agent.Features.SkillPackageV2, false) {
		t.Fatal("unset Skill Package v2 feature must be disabled")
	}
	if boolSetting(cfg.Agent.Features.AgentSkillJudge, false) {
		t.Fatal("unset Agent Skill judge feature must be disabled")
	}
	enabled := true
	cfg.Agent.Features.SkillPackageV2 = &enabled
	cfg.Agent.Features.AgentSkillJudge = &enabled
	if !boolSetting(cfg.Agent.Features.SkillPackageV2, false) || !boolSetting(cfg.Agent.Features.AgentSkillJudge, false) {
		t.Fatal("explicit Agent Skill features were not enabled")
	}
}

func TestCloseAIRuntimeClosesNativeAIServiceLifecycle(t *testing.T) {
	service := &closeTrackingAIService{}
	closeAIRuntime(nil)
	closeAIRuntime(&aiagentruntime.Runtime{})
	closeAIRuntime(&aiagentruntime.Runtime{AI: service})
	if service.closeCalls != 1 {
		t.Fatalf("AI service Close calls = %d, want 1", service.closeCalls)
	}
}

func TestRecruitingRuntimePolicyMapsEachBooleanNilDefaultAndExplicitFalse(t *testing.T) {
	falseValue := false
	tests := []struct {
		name           string
		set            func(*logicconfig.Config, *bool)
		defaultEnabled bool
	}{
		{name: "structured_resume_parse", set: func(cfg *logicconfig.Config, value *bool) { cfg.Agent.Features.StructuredResumeParse = value }, defaultEnabled: true},
		{name: "candidate_match", set: func(cfg *logicconfig.Config, value *bool) { cfg.Agent.Features.CandidateMatch = value }, defaultEnabled: true},
		{name: "candidate_match_semantic", set: func(cfg *logicconfig.Config, value *bool) { cfg.Agent.Features.CandidateMatchSemantic = value }, defaultEnabled: true},
		{name: "candidate_match_shadow", set: func(cfg *logicconfig.Config, value *bool) { cfg.Agent.Features.CandidateMatchShadow = value }, defaultEnabled: false},
		{name: "fallbacks", set: func(cfg *logicconfig.Config, value *bool) { cfg.Agent.Features.Fallbacks = value }, defaultEnabled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/nil default", func(t *testing.T) {
			var cfg logicconfig.Config
			tt.set(&cfg, nil)
			policy := recruitingRuntimePolicy(cfg)
			got := runtimePolicyBoolean(policy, tt.name)
			if got != tt.defaultEnabled {
				t.Fatalf("nil mapping = %v, want %v", got, tt.defaultEnabled)
			}
		})
		t.Run(tt.name+"/explicit false", func(t *testing.T) {
			var cfg logicconfig.Config
			tt.set(&cfg, &falseValue)
			if runtimePolicyBoolean(recruitingRuntimePolicy(cfg), tt.name) {
				t.Fatal("explicit false was replaced by the default")
			}
		})
	}
}

func TestRecruitingRuntimePolicyMapsEachTimeoutDefaultAndExplicitValue(t *testing.T) {
	for _, tt := range []struct {
		name        string
		set         func(*logicconfig.Config, time.Duration)
		read        func(recruitingruntime.RuntimePolicy) time.Duration
		wantDefault time.Duration
	}{
		{name: "resume_parse_timeout", set: func(cfg *logicconfig.Config, value time.Duration) {
			cfg.Agent.Features.ResumeParseTimeout.Duration = value
		}, read: func(policy recruitingruntime.RuntimePolicy) time.Duration { return policy.ResumeParseTimeout() }, wantDefault: 30 * time.Second},
		{name: "candidate_match_timeout", set: func(cfg *logicconfig.Config, value time.Duration) {
			cfg.Agent.Features.CandidateMatchTimeout.Duration = value
		}, read: func(policy recruitingruntime.RuntimePolicy) time.Duration { return policy.CandidateMatchTimeout() }, wantDefault: 15 * time.Second},
	} {
		t.Run(tt.name+"/zero default", func(t *testing.T) {
			var cfg logicconfig.Config
			if got := tt.read(recruitingRuntimePolicy(cfg)); got != tt.wantDefault {
				t.Fatalf("zero mapping = %v, want %v", got, tt.wantDefault)
			}
		})
		t.Run(tt.name+"/explicit value", func(t *testing.T) {
			var cfg logicconfig.Config
			tt.set(&cfg, 19*time.Second)
			if got := tt.read(recruitingRuntimePolicy(cfg)); got != 19*time.Second {
				t.Fatalf("explicit mapping = %v", got)
			}
		})
	}
}

func runtimePolicyBoolean(policy recruitingruntime.RuntimePolicy, name string) bool {
	switch name {
	case "structured_resume_parse":
		return policy.StructuredResumeParseEnabled()
	case "candidate_match":
		return policy.CandidateMatchEnabled()
	case "candidate_match_semantic":
		return policy.CandidateMatchSemanticEnabled()
	case "candidate_match_shadow":
		return policy.CandidateMatchShadowEnabled()
	case "fallbacks":
		return policy.FallbacksEnabled()
	default:
		panic("unknown runtime policy boolean: " + name)
	}
}

type closeTrackingAIService struct {
	noopAIService
	closeCalls int
}

func (s *closeTrackingAIService) Close() error {
	s.closeCalls++
	return nil
}

func TestSetupNacosSupportsLocalStaticFallback(t *testing.T) {
	instance, err := instanceFromAddr("127.0.0.1:50066", platformconfig.Bootstrap{
		ServiceName:    "ai-agent-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
	})
	if err != nil {
		t.Fatalf("instanceFromAddr returned error: %v", err)
	}
	discovery, err := setupNacos(context.Background(), platformconfig.Bootstrap{
		ServiceName:    "ai-agent-service",
		ServiceEnv:     "local",
		ServiceVersion: "test",
		NacosGroup:     "DEFAULT_GROUP",
		StaticFallback: true,
	}, instance)
	if err != nil {
		t.Fatalf("setupNacos returned error: %v", err)
	}
	instances, err := discovery.Resolve(context.Background(), "ai-agent")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if len(instances) != 1 || instances[0].Port != 50066 {
		t.Fatalf("unexpected instances: %#v", instances)
	}
}

func TestMQConfigMapsAIAgentWorkerQueues(t *testing.T) {
	var cfg logicconfig.Config
	cfg.RabbitMQ.URL = "amqp://example"
	cfg.RabbitMQ.Exchange = "events"
	cfg.RabbitMQ.EmbeddingQueue = "embedding.upsert"
	cfg.RabbitMQ.AgentRunQueue = "agent.run"
	cfg.RabbitMQ.PrefetchCount = 8
	cfg.RabbitMQ.MaxRetries = 4

	mapped := mqConfig(cfg)
	if mapped.URL != "amqp://example" || mapped.Exchange != "events" {
		t.Fatalf("unexpected mq target mapping: %#v", mapped)
	}
	if mapped.EmbeddingQueue != "embedding.upsert" {
		t.Fatalf("EmbeddingQueue = %q", mapped.EmbeddingQueue)
	}
	if mapped.AgentRunQueue != "agent.run" {
		t.Fatalf("AgentRunQueue = %q", mapped.AgentRunQueue)
	}
	if mapped.PrefetchCount != 8 || mapped.MaxRetries != 4 {
		t.Fatalf("unexpected retry settings: %#v", mapped)
	}
}

func TestMQKeepAliveUsesConfiguredIntervalAndStopsWithServiceContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	conn := &trackingMQKeepAliveConnection{
		started: make(chan time.Duration, 1),
		stopped: make(chan struct{}),
	}
	const reconnectInterval = 17 * time.Second

	done := startMQKeepAlive(ctx, conn, reconnectInterval)
	select {
	case got := <-conn.started:
		if got != reconnectInterval {
			t.Fatalf("reconnect interval = %v, want %v", got, reconnectInterval)
		}
	case <-time.After(time.Second):
		t.Fatal("MQ KeepAlive did not start")
	}

	stopped := make(chan struct{})
	go func() {
		stopMQKeepAlive(cancel, done)
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("MQ KeepAlive did not stop after service context cancellation")
	}
	select {
	case <-conn.stopped:
	default:
		t.Fatal("MQ KeepAlive connection did not observe context cancellation")
	}
}

type trackingMQKeepAliveConnection struct {
	started chan time.Duration
	stopped chan struct{}
}

func (c *trackingMQKeepAliveConnection) KeepAlive(ctx context.Context, reconnectInterval time.Duration) {
	c.started <- reconnectInterval
	<-ctx.Done()
	close(c.stopped)
}
