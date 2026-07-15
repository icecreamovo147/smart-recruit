package mcp_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"smart-recruit-ai-agent-service/internal/domain/model"
	mcpinfra "smart-recruit-ai-agent-service/internal/infrastructure/mcp"
	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
	"smart-recruit-proto/recruitment/pb"
)

func TestNativeMCPRuntimeConnectionDiscoveryPolicyAuditAndHRTool(t *testing.T) {
	store := newRuntimeStore()
	store.mcpServers[7] = mcpinfra.ServerConfig{ID: 7, Name: "safe", Transport: model.MCPTransportHTTP, CommandOrURL: "https://mcp.example.test", Enabled: true}
	store.mcpPolicies["7:search"] = &model.MCPToolPolicy{
		ID:                  9,
		ServerID:            7,
		ToolName:            "search",
		Enabled:             true,
		Effect:              model.MCPPolicyDecisionAllow,
		AllowedRoles:        []string{"hr_agent"},
		AllowedScopes:       []string{"agent_runtime"},
		RequiredArgs:        []string{"query"},
		RedactFields:        []string{"query"},
		RequireConfirmation: true,
	}
	store.agentConfigs = []*pb.AgentConfigInfo{{
		Id:        11,
		AgentType: "hr_recruiting_agent",
		IsDefault: true,
		IsEnabled: true,
		CapabilityBindings: []*pb.AgentCapabilityBindingInfo{{
			CapabilitySource: "mcp",
			CapabilityKey:    "7:search",
			IsEnabled:        true,
		}},
	}}
	runner := mcpinfra.RunnerFunc{
		TestFunc: func(context.Context, mcpinfra.ServerConfig) (mcpinfra.TestResult, error) {
			return mcpinfra.TestResult{Success: true, Detail: "ok", ToolsFound: 1, DurationMs: 12}, nil
		},
		ListToolsFunc: func(context.Context, mcpinfra.ServerConfig) ([]mcpinfra.Tool, error) {
			return []mcpinfra.Tool{{Name: "search", Description: "Search safely", SchemaJSON: `{"type":"object","properties":{"query":{"type":"string"}}}`}}, nil
		},
		CallToolFunc: func(_ context.Context, _ mcpinfra.ServerConfig, tool string, args map[string]any) (mcpinfra.CallResult, error) {
			if tool != "search" || strings.TrimSpace(fmt.Sprint(args["query"])) == "" {
				t.Fatalf("runner call = %s %#v", tool, args)
			}
			return mcpinfra.CallResult{Content: `{"candidate":"Alice","token":"secret-token"}`, DurationMs: 5}, nil
		},
	}
	deps := aiagentgrpc.NewNativeRuntimeDeps(aiagentgrpc.RuntimeDeps{Store: store, Provider: runtimeProvider{reply: "MCP-informed reply"}, MCPRunner: runner})

	testResp, err := deps.MCP.TestMCPConnection(context.Background(), &pb.TestMCPConnectionRequest{ServerId: 7})
	if err != nil || testResp.GetCode() != 0 || !testResp.GetSuccess() || store.statuses[7] != "connected" || store.toolCounts[7] != 1 {
		t.Fatalf("test response=%#v err=%v status=%s tools=%d", testResp, err, store.statuses[7], store.toolCounts[7])
	}
	toolsResp, err := deps.MCP.ListMCPTools(context.Background(), &pb.ListMCPToolsRequest{ServerId: 7})
	if err != nil || toolsResp.GetCode() != 0 || len(toolsResp.GetList()) != 1 || toolsResp.GetList()[0].GetName() != "search" {
		t.Fatalf("tools response=%#v err=%v", toolsResp, err)
	}
	denied, err := deps.MCP.CallMCPTool(context.Background(), &pb.CallMCPToolRequest{ServerId: 7, ToolName: "search", ArgsJson: `{"query":"Alice"}`, CalledByHrId: 88, SessionId: 99, CallerRole: "hr_agent", CallerScope: "agent_runtime"})
	if err != nil || denied.GetPolicyDecision() != model.MCPPolicyDecisionConfirmationRequired || len(store.mcpLogs) != 1 {
		t.Fatalf("denied=%#v err=%v logs=%d", denied, err, len(store.mcpLogs))
	}
	if !strings.Contains(store.mcpLogs[0].ArgsJSON, "[redacted]") {
		t.Fatalf("denied args not redacted: %s", store.mcpLogs[0].ArgsJSON)
	}
	chatResp, err := deps.AI.Chat(context.Background(), &pb.ChatRequest{HrId: 88, Message: "find Alice", SkillCapabilityKeys: []string{"7:search"}, AgentSkillSelectionConfirmed: true})
	if err != nil || chatResp.GetCode() != 0 || chatResp.GetReply() != "MCP-informed reply" {
		t.Fatalf("chat response=%#v err=%v", chatResp, err)
	}
	if len(store.toolTraces) != 1 || store.toolTraces[0].ToolName != "mcp_7_search" || strings.Contains(store.toolTraces[0].ResultContent, "secret-token") {
		t.Fatalf("tool traces=%#v", store.toolTraces)
	}
	if strings.Contains(store.toolTraces[0].ArgsJSON, "find Alice") || !strings.Contains(store.toolTraces[0].ArgsJSON, "[redacted]") {
		t.Fatalf("tool trace args not policy-redacted: %s", store.toolTraces[0].ArgsJSON)
	}
	if len(store.mcpLogs) != 2 || store.mcpLogs[1].PolicyDecision != model.MCPPolicyDecisionAllow {
		t.Fatalf("mcp logs=%#v", store.mcpLogs)
	}
	invalid, err := deps.MCP.CallMCPTool(context.Background(), &pb.CallMCPToolRequest{ServerId: 7, ToolName: "search", ArgsJson: `{"query":`, CallerRole: "hr_agent", CallerScope: "agent_runtime"})
	if err != nil || invalid.GetCode() != 400 || len(store.mcpLogs) != 3 || store.mcpLogs[2].PolicyReason != "invalid_args_json" {
		t.Fatalf("invalid args response=%#v err=%v logs=%#v", invalid, err, store.mcpLogs)
	}
}

type runtimeProvider struct {
	reply string
}

func (p runtimeProvider) Complete(context.Context, string) (string, error) {
	return p.reply, nil
}

type runtimeStore struct {
	nextSessionID int64
	nextMessageID int64
	sessions      []aiagentgrpc.ChatSessionRow
	messages      []aiagentgrpc.ChatMessageRow
	toolTraces    []aiagentgrpc.ToolTraceRow
	agentConfigs  []*pb.AgentConfigInfo
	mcpServers    map[int64]mcpinfra.ServerConfig
	mcpPolicies   map[string]*model.MCPToolPolicy
	mcpLogs       []mcpinfra.ToolLog
	statuses      map[int64]string
	toolCounts    map[int64]int
}

func newRuntimeStore() *runtimeStore {
	return &runtimeStore{
		nextSessionID: 100,
		nextMessageID: 200,
		mcpServers:    make(map[int64]mcpinfra.ServerConfig),
		mcpPolicies:   make(map[string]*model.MCPToolPolicy),
		statuses:      make(map[int64]string),
		toolCounts:    make(map[int64]int),
	}
}

func (s *runtimeStore) EnsureChatSession(_ context.Context, _ int32, _ int64, title string, applicationID int64) (aiagentgrpc.ChatSessionRow, error) {
	s.nextSessionID++
	row := aiagentgrpc.ChatSessionRow{ID: s.nextSessionID, Title: title, ApplicationID: applicationID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	s.sessions = append(s.sessions, row)
	return row, nil
}

func (s *runtimeStore) GetChatSession(context.Context, int32, int64, int64) (aiagentgrpc.ChatSessionRow, bool, error) {
	return aiagentgrpc.ChatSessionRow{}, false, nil
}

func (s *runtimeStore) ListChatSessions(context.Context, int32, int64, int32, int32) ([]aiagentgrpc.ChatSessionRow, int64, error) {
	return s.sessions, int64(len(s.sessions)), nil
}

func (s *runtimeStore) UpdateChatSessionTitle(context.Context, int32, int64, int64, string) error {
	return nil
}

func (s *runtimeStore) DeleteChatSession(context.Context, int32, int64, int64) error {
	return nil
}

func (s *runtimeStore) AppendChatMessage(_ context.Context, row aiagentgrpc.ChatMessageRow) (aiagentgrpc.ChatMessageRow, error) {
	s.nextMessageID++
	row.ID = s.nextMessageID
	row.CreatedAt = time.Now()
	s.messages = append(s.messages, row)
	return row, nil
}

func (s *runtimeStore) ListChatMessages(context.Context, int32, int64, int64, int32, int32) ([]aiagentgrpc.ChatMessageRow, error) {
	return s.messages, nil
}

func (s *runtimeStore) ListToolTraces(context.Context, int64, int64) ([]aiagentgrpc.ToolTraceRow, error) {
	return s.toolTraces, nil
}

func (s *runtimeStore) AppendToolTrace(_ context.Context, _ int64, row aiagentgrpc.ToolTraceRow) (aiagentgrpc.ToolTraceRow, error) {
	row.ID = int64(len(s.toolTraces) + 1)
	s.toolTraces = append(s.toolTraces, row)
	return row, nil
}

func (s *runtimeStore) CreateAgentRun(context.Context, aiagentgrpc.AgentRunRow) (aiagentgrpc.AgentRunRow, bool, error) {
	return aiagentgrpc.AgentRunRow{}, false, nil
}
func (s *runtimeStore) ListAgentRuns(context.Context, int64, int64) ([]aiagentgrpc.AgentRunRow, error) {
	return nil, nil
}
func (s *runtimeStore) GetAgentRun(context.Context, int64, int64) (aiagentgrpc.AgentRunRow, bool, error) {
	return aiagentgrpc.AgentRunRow{}, false, nil
}
func (s *runtimeStore) GetActiveAgentRun(context.Context, int64, int64) (aiagentgrpc.AgentRunRow, bool, error) {
	return aiagentgrpc.AgentRunRow{}, false, nil
}
func (s *runtimeStore) UpdateAgentRunStatus(context.Context, int64, int64, string) (aiagentgrpc.AgentRunRow, bool, error) {
	return aiagentgrpc.AgentRunRow{}, false, nil
}
func (s *runtimeStore) UpdateAgentRunPlan(context.Context, int64, int64, string, string) (aiagentgrpc.AgentRunRow, bool, error) {
	return aiagentgrpc.AgentRunRow{}, false, nil
}
func (s *runtimeStore) CompleteAgentRun(context.Context, int64, int64, string, string, string, string) (aiagentgrpc.AgentRunRow, bool, error) {
	return aiagentgrpc.AgentRunRow{}, false, nil
}
func (s *runtimeStore) AppendAgentRunEvent(context.Context, int64, string, string) (aiagentgrpc.AgentRunEventRow, error) {
	return aiagentgrpc.AgentRunEventRow{}, nil
}
func (s *runtimeStore) ListAgentRunEvents(context.Context, int64, int64, int64) ([]aiagentgrpc.AgentRunEventRow, error) {
	return nil, nil
}
func (s *runtimeStore) ListLlmProviders(context.Context, int32, int32) ([]*pb.LlmProviderInfo, int64, error) {
	return nil, 0, nil
}
func (s *runtimeStore) ListLlmModels(context.Context, int32, int32, int64) ([]*pb.LlmModelInfo, int64, error) {
	return nil, 0, nil
}
func (s *runtimeStore) ListPromptTemplates(context.Context, int32, int32, string) ([]*pb.PromptTemplateInfo, int64, error) {
	return nil, 0, nil
}
func (s *runtimeStore) ListAgentConfigs(_ context.Context, _ int32, _ int32, agentType string) ([]*pb.AgentConfigInfo, int64, error) {
	items := make([]*pb.AgentConfigInfo, 0, len(s.agentConfigs))
	for _, config := range s.agentConfigs {
		if config.GetAgentType() == agentType {
			items = append(items, config)
		}
	}
	return items, int64(len(items)), nil
}
func (s *runtimeStore) ListMCPServers(context.Context, int32, int32) ([]*pb.MCPServerInfo, int64, error) {
	return nil, 0, nil
}
func (s *runtimeStore) ListAgentSkills(context.Context, int32, int32, string, bool) ([]*pb.AgentSkillInfo, int64, error) {
	return nil, 0, nil
}
func (s *runtimeStore) ListEmbeddingProviders(context.Context, int32, int32) ([]*pb.EmbeddingProviderInfo, int64, error) {
	return nil, 0, nil
}
func (s *runtimeStore) ListEmbeddingModels(context.Context, int32, int32, int64) ([]*pb.EmbeddingModelInfo, int64, error) {
	return nil, 0, nil
}

func (s *runtimeStore) GetMCPRuntimeServer(_ context.Context, serverID int64) (mcpinfra.ServerConfig, bool, error) {
	server, ok := s.mcpServers[serverID]
	return server, ok, nil
}
func (s *runtimeStore) UpdateMCPRuntimeStatus(_ context.Context, serverID int64, status string, toolCount int, _ string) error {
	s.statuses[serverID] = status
	s.toolCounts[serverID] = toolCount
	return nil
}
func (s *runtimeStore) GetMCPRuntimeToolPolicy(_ context.Context, serverID int64, toolName string) (*model.MCPToolPolicy, bool, error) {
	policyModel, ok := s.mcpPolicies[fmt.Sprintf("%d:%s", serverID, toolName)]
	return policyModel, ok, nil
}
func (s *runtimeStore) CountRecentMCPToolCalls(_ context.Context, serverID int64, toolName string, _ time.Time) (int64, error) {
	var count int64
	for _, log := range s.mcpLogs {
		if log.ServerID == serverID && log.ToolName == toolName {
			count++
		}
	}
	return count, nil
}
func (s *runtimeStore) AppendMCPToolLog(_ context.Context, log mcpinfra.ToolLog) error {
	s.mcpLogs = append(s.mcpLogs, log)
	return nil
}
