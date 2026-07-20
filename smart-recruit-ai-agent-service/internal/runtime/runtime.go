package runtime

import (
	"fmt"

	"google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

const ServiceName = "ai-agent-service"

type Deps struct {
	AI                     pb.AIServiceServer
	LlmConfig              pb.LlmConfigServiceServer
	Prompt                 pb.PromptServiceServer
	AgentConfig            pb.AgentConfigServiceServer
	MCP                    pb.MCPServiceServer
	Skill                  pb.SkillServiceServer
	AgentSkill             pb.AgentSkillServiceServer
	RecruitingIntelligence pb.RecruitingIntelligenceServiceServer
	EmbeddingConfig        pb.EmbeddingConfigServiceServer
	PlatformAIControlPlane pb.PlatformAIControlPlaneServiceServer
	LongTasks              LongTaskControls
}

type Runtime struct {
	AI                     pb.AIServiceServer
	LlmConfig              pb.LlmConfigServiceServer
	Prompt                 pb.PromptServiceServer
	AgentConfig            pb.AgentConfigServiceServer
	MCP                    pb.MCPServiceServer
	Skill                  pb.SkillServiceServer
	AgentSkill             pb.AgentSkillServiceServer
	RecruitingIntelligence pb.RecruitingIntelligenceServiceServer
	EmbeddingConfig        pb.EmbeddingConfigServiceServer
	PlatformAIControlPlane pb.PlatformAIControlPlaneServiceServer
	LongTasks              LongTaskControls
}

type LongTaskControls struct {
	RabbitMQRequired bool
	EmbeddingWorker  bool
	AgentRunWorker   bool
	RuntimeName      string
}

func New(deps Deps) (*Runtime, error) {
	if deps.AI == nil {
		return nil, fmt.Errorf("ai service is required")
	}
	if deps.LlmConfig == nil {
		return nil, fmt.Errorf("llm config service is required")
	}
	if deps.Prompt == nil {
		return nil, fmt.Errorf("prompt service is required")
	}
	if deps.AgentConfig == nil {
		return nil, fmt.Errorf("agent config service is required")
	}
	if deps.MCP == nil {
		return nil, fmt.Errorf("mcp service is required")
	}
	if deps.Skill == nil {
		return nil, fmt.Errorf("skill service is required")
	}
	if deps.AgentSkill == nil {
		return nil, fmt.Errorf("agent skill service is required")
	}
	if deps.RecruitingIntelligence == nil {
		return nil, fmt.Errorf("recruiting intelligence service is required")
	}
	if deps.EmbeddingConfig == nil {
		return nil, fmt.Errorf("embedding config service is required")
	}
	if deps.PlatformAIControlPlane == nil {
		return nil, fmt.Errorf("platform AI control plane service is required")
	}
	longTasks := deps.LongTasks
	if !longTasks.Configured() {
		longTasks = LongTaskControls{RabbitMQRequired: true}
	} else {
		if err := longTasks.Validate(); err != nil {
			return nil, err
		}
	}
	return &Runtime{
		AI:                     deps.AI,
		LlmConfig:              deps.LlmConfig,
		Prompt:                 deps.Prompt,
		AgentConfig:            deps.AgentConfig,
		MCP:                    deps.MCP,
		Skill:                  deps.Skill,
		AgentSkill:             deps.AgentSkill,
		RecruitingIntelligence: deps.RecruitingIntelligence,
		EmbeddingConfig:        deps.EmbeddingConfig,
		PlatformAIControlPlane: deps.PlatformAIControlPlane,
		LongTasks:              longTasks,
	}, nil
}

func (controls LongTaskControls) Configured() bool {
	return controls.RabbitMQRequired || controls.EmbeddingWorker || controls.AgentRunWorker || controls.RuntimeName != ""
}

func (controls LongTaskControls) Validate() error {
	if !controls.RabbitMQRequired {
		return fmt.Errorf("ai agent long tasks must require RabbitMQ")
	}
	if !controls.EmbeddingWorker {
		return fmt.Errorf("embedding worker control is required")
	}
	if !controls.AgentRunWorker {
		return fmt.Errorf("agent run worker control is required")
	}
	return nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if r == nil {
		return fmt.Errorf("ai agent runtime is not initialized")
	}
	pb.RegisterAIServiceServer(registrar, r.AI)
	pb.RegisterLlmConfigServiceServer(registrar, r.LlmConfig)
	pb.RegisterPromptServiceServer(registrar, r.Prompt)
	pb.RegisterAgentConfigServiceServer(registrar, r.AgentConfig)
	pb.RegisterMCPServiceServer(registrar, r.MCP)
	pb.RegisterSkillServiceServer(registrar, r.Skill)
	pb.RegisterAgentSkillServiceServer(registrar, r.AgentSkill)
	pb.RegisterRecruitingIntelligenceServiceServer(registrar, r.RecruitingIntelligence)
	pb.RegisterEmbeddingConfigServiceServer(registrar, r.EmbeddingConfig)
	pb.RegisterPlatformAIControlPlaneServiceServer(registrar, r.PlatformAIControlPlane)
	return nil
}
