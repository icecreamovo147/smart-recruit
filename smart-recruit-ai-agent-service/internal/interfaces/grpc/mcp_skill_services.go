package grpc

import (
	"context"

	"smart-recruit-proto/recruitment/pb"
)

type mcpGovernanceStore interface {
	CreateMCPServer(context.Context, *pb.CreateMCPServerRequest) (*pb.MCPServerResponse, error)
	UpdateMCPServer(context.Context, *pb.UpdateMCPServerRequest) (*pb.MCPServerResponse, error)
	DeleteMCPServer(context.Context, *pb.DeleteMCPServerRequest) (*pb.CommonResponse, error)
	ListMCPToolPolicies(context.Context, *pb.ListMCPToolPoliciesRequest) (*pb.ListMCPToolPoliciesResponse, error)
	CreateMCPToolPolicy(context.Context, *pb.CreateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error)
	UpdateMCPToolPolicy(context.Context, *pb.UpdateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error)
	DeleteMCPToolPolicy(context.Context, *pb.DeleteMCPToolPolicyRequest) (*pb.CommonResponse, error)
	ListMCPToolLogs(context.Context, *pb.ListMCPToolLogsRequest) (*pb.ListMCPToolLogsResponse, error)
	TestMCPConnection(context.Context, *pb.TestMCPConnectionRequest) (*pb.TestMCPConnectionResponse, error)
	ListMCPTools(context.Context, *pb.ListMCPToolsRequest) (*pb.ListMCPToolsResponse, error)
	CallMCPTool(context.Context, *pb.CallMCPToolRequest) (*pb.CallMCPToolResponse, error)
}

type skillGovernanceStore interface {
	ListSkills(context.Context, *pb.ListSkillsRequest) (*pb.ListSkillsResponse, error)
	CreateSkill(context.Context, *pb.CreateSkillRequest) (*pb.SkillResponse, error)
	UpdateSkill(context.Context, *pb.UpdateSkillRequest) (*pb.SkillResponse, error)
	CreateSkillVersion(context.Context, *pb.CreateSkillVersionRequest) (*pb.SkillVersionResponse, error)
	ListSkillVersions(context.Context, *pb.ListSkillVersionsRequest) (*pb.ListSkillVersionsResponse, error)
	ActivateSkillVersion(context.Context, *pb.ActivateSkillVersionRequest) (*pb.SkillResponse, error)
	ListSkillTools(context.Context, *pb.ListSkillToolsRequest) (*pb.ListSkillToolsResponse, error)
	UpdateSkillTool(context.Context, *pb.UpdateSkillToolRequest) (*pb.SkillToolResponse, error)
}

type agentSkillGovernanceStore interface {
	GetAgentSkill(context.Context, *pb.GetAgentSkillRequest) (*pb.AgentSkillResponse, error)
	CreateAgentSkill(context.Context, *pb.CreateAgentSkillRequest) (*pb.AgentSkillResponse, error)
	UpdateAgentSkill(context.Context, *pb.UpdateAgentSkillRequest) (*pb.AgentSkillResponse, error)
	CreateAgentSkillVersion(context.Context, *pb.CreateAgentSkillVersionRequest) (*pb.AgentSkillVersionResponse, error)
	ListAgentSkillVersions(context.Context, *pb.ListAgentSkillVersionsRequest) (*pb.ListAgentSkillVersionsResponse, error)
	ActivateAgentSkillVersion(context.Context, *pb.ActivateAgentSkillVersionRequest) (*pb.AgentSkillResponse, error)
	UpdateAgentSkillStatus(context.Context, *pb.UpdateAgentSkillStatusRequest) (*pb.AgentSkillResponse, error)
	PreviewAgentSkill(context.Context, *pb.PreviewAgentSkillRequest) (*pb.PreviewAgentSkillResponse, error)
	DebugSemanticRetrieval(context.Context, *pb.DebugSemanticRetrievalRequest) (*pb.DebugSemanticRetrievalResponse, error)
}

func (s nativeMCPService) CreateMCPServer(ctx context.Context, req *pb.CreateMCPServerRequest) (*pb.MCPServerResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.MCPServerResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.CreateMCPServer(ctx, req)
}

func (s nativeMCPService) UpdateMCPServer(ctx context.Context, req *pb.UpdateMCPServerRequest) (*pb.MCPServerResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.MCPServerResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.UpdateMCPServer(ctx, req)
}

func (s nativeMCPService) DeleteMCPServer(ctx context.Context, req *pb.DeleteMCPServerRequest) (*pb.CommonResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.DeleteMCPServer(ctx, req)
}

func (s nativeMCPService) CreateMCPToolPolicy(ctx context.Context, req *pb.CreateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.MCPToolPolicyResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.CreateMCPToolPolicy(ctx, req)
}

func (s nativeMCPService) UpdateMCPToolPolicy(ctx context.Context, req *pb.UpdateMCPToolPolicyRequest) (*pb.MCPToolPolicyResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.MCPToolPolicyResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.UpdateMCPToolPolicy(ctx, req)
}

func (s nativeMCPService) DeleteMCPToolPolicy(ctx context.Context, req *pb.DeleteMCPToolPolicyRequest) (*pb.CommonResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.CommonResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.DeleteMCPToolPolicy(ctx, req)
}

func (s nativeMCPService) TestMCPConnection(ctx context.Context, req *pb.TestMCPConnectionRequest) (*pb.TestMCPConnectionResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.TestMCPConnectionResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured", Success: false}, nil
	}
	return store.TestMCPConnection(ctx, req)
}

func (s nativeMCPService) ListMCPTools(ctx context.Context, req *pb.ListMCPToolsRequest) (*pb.ListMCPToolsResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.ListMCPToolsResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.ListMCPTools(ctx, req)
}

func (s nativeMCPService) CallMCPTool(ctx context.Context, req *pb.CallMCPToolRequest) (*pb.CallMCPToolResponse, error) {
	store, ok := s.store.(mcpGovernanceStore)
	if !ok {
		return &pb.CallMCPToolResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured", ErrorMsg: "store is not configured"}, nil
	}
	return store.CallMCPTool(ctx, req)
}

func (s nativeSkillService) CreateSkill(ctx context.Context, req *pb.CreateSkillRequest) (*pb.SkillResponse, error) {
	store, ok := s.store.(skillGovernanceStore)
	if !ok {
		return &pb.SkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.CreateSkill(ctx, req)
}

func (s nativeSkillService) UpdateSkill(ctx context.Context, req *pb.UpdateSkillRequest) (*pb.SkillResponse, error) {
	store, ok := s.store.(skillGovernanceStore)
	if !ok {
		return &pb.SkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.UpdateSkill(ctx, req)
}

func (s nativeSkillService) CreateSkillVersion(ctx context.Context, req *pb.CreateSkillVersionRequest) (*pb.SkillVersionResponse, error) {
	store, ok := s.store.(skillGovernanceStore)
	if !ok {
		return &pb.SkillVersionResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.CreateSkillVersion(ctx, req)
}

func (s nativeSkillService) ListSkillVersions(ctx context.Context, req *pb.ListSkillVersionsRequest) (*pb.ListSkillVersionsResponse, error) {
	store, ok := s.store.(skillGovernanceStore)
	if !ok {
		return &pb.ListSkillVersionsResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.ListSkillVersions(ctx, req)
}

func (s nativeSkillService) ActivateSkillVersion(ctx context.Context, req *pb.ActivateSkillVersionRequest) (*pb.SkillResponse, error) {
	store, ok := s.store.(skillGovernanceStore)
	if !ok {
		return &pb.SkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.ActivateSkillVersion(ctx, req)
}

func (s nativeSkillService) ListSkillTools(ctx context.Context, req *pb.ListSkillToolsRequest) (*pb.ListSkillToolsResponse, error) {
	store, ok := s.store.(skillGovernanceStore)
	if !ok {
		return &pb.ListSkillToolsResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.ListSkillTools(ctx, req)
}

func (s nativeSkillService) UpdateSkillTool(ctx context.Context, req *pb.UpdateSkillToolRequest) (*pb.SkillToolResponse, error) {
	store, ok := s.store.(skillGovernanceStore)
	if !ok {
		return &pb.SkillToolResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.UpdateSkillTool(ctx, req)
}

func (s nativeAgentSkillService) GetAgentSkill(ctx context.Context, req *pb.GetAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.AgentSkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.GetAgentSkill(ctx, req)
}

func (s nativeAgentSkillService) CreateAgentSkill(ctx context.Context, req *pb.CreateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.AgentSkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.CreateAgentSkill(ctx, req)
}

func (s nativeAgentSkillService) UpdateAgentSkill(ctx context.Context, req *pb.UpdateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.AgentSkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.UpdateAgentSkill(ctx, req)
}

func (s nativeAgentSkillService) CreateAgentSkillVersion(ctx context.Context, req *pb.CreateAgentSkillVersionRequest) (*pb.AgentSkillVersionResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.AgentSkillVersionResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.CreateAgentSkillVersion(ctx, req)
}

func (s nativeAgentSkillService) ListAgentSkillVersions(ctx context.Context, req *pb.ListAgentSkillVersionsRequest) (*pb.ListAgentSkillVersionsResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.ListAgentSkillVersionsResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.ListAgentSkillVersions(ctx, req)
}

func (s nativeAgentSkillService) ActivateAgentSkillVersion(ctx context.Context, req *pb.ActivateAgentSkillVersionRequest) (*pb.AgentSkillResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.AgentSkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.ActivateAgentSkillVersion(ctx, req)
}

func (s nativeAgentSkillService) UpdateAgentSkillStatus(ctx context.Context, req *pb.UpdateAgentSkillStatusRequest) (*pb.AgentSkillResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.AgentSkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.UpdateAgentSkillStatus(ctx, req)
}

func (s nativeAgentSkillService) PreviewAgentSkill(ctx context.Context, req *pb.PreviewAgentSkillRequest) (*pb.PreviewAgentSkillResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.PreviewAgentSkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.PreviewAgentSkill(ctx, req)
}

func (s nativeAgentSkillService) DebugSemanticRetrieval(ctx context.Context, req *pb.DebugSemanticRetrievalRequest) (*pb.DebugSemanticRetrievalResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.DebugSemanticRetrievalResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured", EmbeddingAvailable: false, FallbackReason: "store is not configured"}, nil
	}
	return store.DebugSemanticRetrieval(ctx, req)
}
