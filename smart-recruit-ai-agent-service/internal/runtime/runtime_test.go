package runtime

import (
	"testing"

	"google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

func TestRuntimeRegistersAIAgentGRPCServices(t *testing.T) {
	runtime, err := New(fakeDeps())
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)
	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC returned error: %v", err)
	}
	services := server.GetServiceInfo()
	for _, serviceName := range []string{
		pb.AIService_ServiceDesc.ServiceName,
		pb.LlmConfigService_ServiceDesc.ServiceName,
		pb.PromptService_ServiceDesc.ServiceName,
		pb.AgentConfigService_ServiceDesc.ServiceName,
		pb.MCPService_ServiceDesc.ServiceName,
		pb.SkillService_ServiceDesc.ServiceName,
		pb.AgentSkillService_ServiceDesc.ServiceName,
		pb.RecruitingIntelligenceService_ServiceDesc.ServiceName,
		pb.EmbeddingConfigService_ServiceDesc.ServiceName,
	} {
		if _, ok := services[serviceName]; !ok {
			t.Fatalf("missing registered service %s", serviceName)
		}
	}
}

func TestRuntimeRequiresAIOwnedDependencies(t *testing.T) {
	if _, err := New(Deps{}); err == nil {
		t.Fatal("expected missing dependency error")
	}
}

func TestLongTaskControlsRequireRabbitMQAndWorkers(t *testing.T) {
	if err := (LongTaskControls{}).Validate(); err == nil {
		t.Fatal("expected missing long task controls error")
	}
	controls := LongTaskControls{RabbitMQRequired: true, EmbeddingWorker: true, AgentRunWorker: true}
	if err := controls.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestRuntimeUsesConfiguredLongTaskControls(t *testing.T) {
	deps := fakeDeps()
	deps.LongTasks = LongTaskControls{RabbitMQRequired: true, EmbeddingWorker: true, AgentRunWorker: true, RuntimeName: "ai-agent-service"}
	runtime, err := New(deps)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := runtime.LongTasks.Validate(); err != nil {
		t.Fatalf("long task validation failed: %v", err)
	}
	if runtime.LongTasks.RuntimeName != "ai-agent-service" {
		t.Fatalf("RuntimeName = %q", runtime.LongTasks.RuntimeName)
	}
}

func fakeDeps() Deps {
	return Deps{
		AI:                     fakeAIService{},
		LlmConfig:              fakeLlmConfigService{},
		Prompt:                 fakePromptService{},
		AgentConfig:            fakeAgentConfigService{},
		MCP:                    fakeMCPService{},
		Skill:                  fakeSkillService{},
		AgentSkill:             fakeAgentSkillService{},
		RecruitingIntelligence: fakeRecruitingIntelligenceService{},
		EmbeddingConfig:        fakeEmbeddingConfigService{},
	}
}

type fakeAIService struct {
	pb.UnimplementedAIServiceServer
}
type fakeLlmConfigService struct {
	pb.UnimplementedLlmConfigServiceServer
}
type fakePromptService struct {
	pb.UnimplementedPromptServiceServer
}
type fakeAgentConfigService struct {
	pb.UnimplementedAgentConfigServiceServer
}
type fakeMCPService struct {
	pb.UnimplementedMCPServiceServer
}
type fakeSkillService struct {
	pb.UnimplementedSkillServiceServer
}
type fakeAgentSkillService struct {
	pb.UnimplementedAgentSkillServiceServer
}
type fakeRecruitingIntelligenceService struct {
	pb.UnimplementedRecruitingIntelligenceServiceServer
}
type fakeEmbeddingConfigService struct {
	pb.UnimplementedEmbeddingConfigServiceServer
}
