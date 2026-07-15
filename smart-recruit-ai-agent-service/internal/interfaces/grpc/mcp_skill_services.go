package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"smart-recruit-ai-agent-service/internal/domain/model"
	"smart-recruit-ai-agent-service/internal/domain/policy"
	mcpinfra "smart-recruit-ai-agent-service/internal/infrastructure/mcp"
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
}

type mcpRuntimeStore interface {
	GetMCPRuntimeServer(ctx context.Context, serverID int64) (mcpinfra.ServerConfig, bool, error)
	UpdateMCPRuntimeStatus(ctx context.Context, serverID int64, status string, toolCount int, lastError string) error
	GetMCPRuntimeToolPolicy(ctx context.Context, serverID int64, toolName string) (*model.MCPToolPolicy, bool, error)
	CountRecentMCPToolCalls(ctx context.Context, serverID int64, toolName string, since time.Time) (int64, error)
	AppendMCPToolLog(ctx context.Context, log mcpinfra.ToolLog) error
}

func (s nativeMCPService) RedactedArgsForTool(ctx context.Context, serverID int64, toolName, argsJSON string) string {
	store, ok := s.store.(mcpRuntimeStore)
	if !ok {
		return redactMCPJSON(argsJSON, nil)
	}
	policyModel, _, err := store.GetMCPRuntimeToolPolicy(ctx, serverID, toolName)
	if err != nil || policyModel == nil {
		return redactMCPJSON(argsJSON, nil)
	}
	return redactMCPJSON(argsJSON, policyModel.RedactFields)
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
	store, ok := s.store.(mcpRuntimeStore)
	if !ok {
		return &pb.TestMCPConnectionResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured", Success: false}, nil
	}
	runner := s.runner
	if runner == nil {
		return &pb.TestMCPConnectionResponse{Code: configCodeUnavailable, Msg: "mcp runner is not configured", Success: false, Detail: "native MCP runner is not bound"}, nil
	}
	server, found, err := store.GetMCPRuntimeServer(ctx, req.GetServerId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.TestMCPConnectionResponse{Code: 404, Msg: "mcp server not found", Success: false}, nil
	}
	result, err := runner.Test(ctx, server)
	if err != nil {
		_ = store.UpdateMCPRuntimeStatus(ctx, server.ID, "error", 0, safeMCPText(err.Error()))
		return &pb.TestMCPConnectionResponse{Code: configCodeUnavailable, Msg: "mcp connection failed", Success: false, Detail: safeMCPText(result.Detail)}, nil
	}
	_ = store.UpdateMCPRuntimeStatus(ctx, server.ID, "connected", result.ToolsFound, "")
	return &pb.TestMCPConnectionResponse{Code: 0, Msg: "success", Success: result.Success, Detail: safeMCPText(result.Detail)}, nil
}

func (s nativeMCPService) ListMCPTools(ctx context.Context, req *pb.ListMCPToolsRequest) (*pb.ListMCPToolsResponse, error) {
	store, ok := s.store.(mcpRuntimeStore)
	if !ok {
		return &pb.ListMCPToolsResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	if s.runner == nil {
		return &pb.ListMCPToolsResponse{Code: configCodeUnavailable, Msg: "mcp runner is not configured"}, nil
	}
	server, found, err := store.GetMCPRuntimeServer(ctx, req.GetServerId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.ListMCPToolsResponse{Code: 404, Msg: "mcp server not found"}, nil
	}
	tools, err := s.runner.ListTools(ctx, server)
	if err != nil {
		_ = store.UpdateMCPRuntimeStatus(ctx, server.ID, "error", 0, safeMCPText(err.Error()))
		return &pb.ListMCPToolsResponse{Code: configCodeUnavailable, Msg: safeMCPText(err.Error())}, nil
	}
	_ = store.UpdateMCPRuntimeStatus(ctx, server.ID, "connected", len(tools), "")
	items := make([]*pb.MCPToolInfo, 0, len(tools))
	for _, tool := range tools {
		items = append(items, &pb.MCPToolInfo{Name: tool.Name, Description: safeMCPText(tool.Description), SchemaJson: redactMCPJSON(tool.SchemaJSON, nil)})
	}
	return &pb.ListMCPToolsResponse{Code: 0, Msg: "success", List: items}, nil
}

func (s nativeMCPService) CallMCPTool(ctx context.Context, req *pb.CallMCPToolRequest) (*pb.CallMCPToolResponse, error) {
	store, ok := s.store.(mcpRuntimeStore)
	if !ok {
		return &pb.CallMCPToolResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured", ErrorMsg: "store is not configured"}, nil
	}
	if s.runner == nil {
		return &pb.CallMCPToolResponse{Code: configCodeUnavailable, Msg: "mcp runner is not configured", ErrorMsg: "runner is not configured"}, nil
	}
	server, found, err := store.GetMCPRuntimeServer(ctx, req.GetServerId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.CallMCPToolResponse{Code: 404, Msg: "mcp server not found", ErrorMsg: "mcp server not found"}, nil
	}
	if !server.Enabled {
		resp := &pb.CallMCPToolResponse{Code: configCodeUnavailable, Msg: "mcp server is disabled", ErrorMsg: "mcp server is disabled", PolicyDecision: model.MCPPolicyDecisionDeny, PolicyReason: "server_disabled"}
		if err := store.AppendMCPToolLog(ctx, responseMCPLog(req, resp, 0, redactMCPJSON(req.GetArgsJson(), nil))); err != nil {
			return nil, err
		}
		return resp, nil
	}
	args, err := parseMCPArgs(req.GetArgsJson())
	if err != nil {
		resp := &pb.CallMCPToolResponse{Code: 400, Msg: err.Error(), ErrorMsg: err.Error(), PolicyDecision: model.MCPPolicyDecisionDeny, PolicyReason: "invalid_args_json"}
		if logErr := store.AppendMCPToolLog(ctx, responseMCPLog(req, resp, 0, redactMCPJSON(req.GetArgsJson(), nil))); logErr != nil {
			return nil, logErr
		}
		return resp, nil
	}
	policyModel, _, err := store.GetMCPRuntimeToolPolicy(ctx, req.GetServerId(), req.GetToolName())
	if err != nil {
		return nil, err
	}
	recentCalls := int64(0)
	if policyModel != nil && policyModel.RateLimitWindowSeconds > 0 {
		recentCalls, err = store.CountRecentMCPToolCalls(ctx, req.GetServerId(), req.GetToolName(), time.Now().Add(-time.Duration(policyModel.RateLimitWindowSeconds)*time.Second))
		if err != nil {
			return nil, err
		}
	}
	eval, err := policy.EvaluateMCPToolPolicy(policyModel, model.MCPPolicyContext{
		ServerID:             uint64(req.GetServerId()),
		ToolName:             req.GetToolName(),
		CallerRole:           req.GetCallerRole(),
		CallerScope:          req.GetCallerScope(),
		ConfirmationApproved: req.GetConfirmationApproved(),
		RecentCalls:          recentCalls,
		Args:                 args,
		Now:                  time.Now(),
	})
	if err != nil {
		return nil, err
	}
	redactedArgs := redactMCPJSON(req.GetArgsJson(), eval.RedactFields)
	if eval.Decision != model.MCPPolicyDecisionAllow {
		resp := policyDeniedMCPResponse(eval)
		if err := store.AppendMCPToolLog(ctx, responseMCPLog(req, resp, 0, redactedArgs)); err != nil {
			return nil, err
		}
		return resp, nil
	}
	result, err := s.runner.CallTool(ctx, server, req.GetToolName(), args)
	resp := &pb.CallMCPToolResponse{Code: 0, Msg: "success", ResultContent: truncateMCPText(redactMCPText(result.Content)), ErrorMsg: safeMCPText(result.Error), DurationMs: result.DurationMs, PolicyDecision: eval.Decision, PolicyReason: eval.Reason, PolicyId: int64(eval.PolicyID)}
	if err != nil || strings.TrimSpace(result.Error) != "" {
		resp.Code = configCodeUnavailable
		resp.Msg = "mcp tool execution failed"
		if resp.ErrorMsg == "" && err != nil {
			resp.ErrorMsg = safeMCPText(err.Error())
		}
	}
	if err := store.AppendMCPToolLog(ctx, responseMCPLog(req, resp, result.DurationMs, redactedArgs)); err != nil {
		return nil, err
	}
	return resp, nil
}

func parseMCPArgs(value string) (map[string]any, error) {
	if strings.TrimSpace(value) == "" {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(value), &args); err != nil {
		return nil, fmt.Errorf("args_json is invalid: %w", err)
	}
	return args, nil
}

func policyDeniedMCPResponse(eval model.MCPPolicyEvaluation) *pb.CallMCPToolResponse {
	code := int32(403)
	switch eval.Decision {
	case model.MCPPolicyDecisionConfirmationRequired:
		code = 409
	case model.MCPPolicyDecisionRateLimited:
		code = 429
	}
	reason := safeMCPText(eval.Reason)
	return &pb.CallMCPToolResponse{Code: code, Msg: reason, ErrorMsg: reason, PolicyDecision: eval.Decision, PolicyReason: reason, PolicyId: int64(eval.PolicyID)}
}

func responseMCPLog(req *pb.CallMCPToolRequest, resp *pb.CallMCPToolResponse, durationMs int64, redactedArgs string) mcpinfra.ToolLog {
	if durationMs == 0 && resp != nil {
		durationMs = resp.GetDurationMs()
	}
	return mcpinfra.ToolLog{
		ServerID:       req.GetServerId(),
		ToolName:       req.GetToolName(),
		ArgsJSON:       redactedArgs,
		ResultContent:  truncateMCPText(resp.GetResultContent()),
		DurationMs:     durationMs,
		ErrorMsg:       safeMCPText(resp.GetErrorMsg()),
		CalledByHRID:   req.GetCalledByHrId(),
		SessionID:      req.GetSessionId(),
		PolicyID:       resp.GetPolicyId(),
		PolicyDecision: resp.GetPolicyDecision(),
		PolicyReason:   safeMCPText(resp.GetPolicyReason()),
	}
}

func redactMCPJSON(value string, extraFields []string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	var parsed any
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return safeMCPText(value)
	}
	redacted := redactMCPJSONValue(parsed, extraFields)
	data, err := json.Marshal(redacted)
	if err != nil {
		return `{"redacted":true}`
	}
	return string(data)
}

func redactMCPJSONValue(value any, extraFields []string) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveMCPKey(key, extraFields) {
				result[key] = "[redacted]"
				continue
			}
			result[key] = redactMCPJSONValue(item, extraFields)
		}
		return result
	case []any:
		items := make([]any, len(typed))
		for i, item := range typed {
			items[i] = redactMCPJSONValue(item, extraFields)
		}
		return items
	default:
		return value
	}
}

func isSensitiveMCPKey(key string, extraFields []string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	for _, field := range extraFields {
		if lower == strings.ToLower(strings.TrimSpace(field)) {
			return true
		}
	}
	return strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "authorization") || strings.Contains(lower, "credential") || strings.Contains(lower, "cookie")
}

func safeMCPText(value string) string {
	return truncateMCPText(redactMCPText(value))
}

func redactMCPText(value string) string {
	lower := strings.ToLower(value)
	for _, marker := range []string{"api_key", "apikey", "token", "secret", "password", "authorization", "credential", "cookie"} {
		if strings.Contains(lower, marker) {
			return "[redacted]"
		}
	}
	return value
}

func truncateMCPText(value string) string {
	const maxLen = 4096
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "...[truncated]"
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
	resp, err := store.CreateAgentSkill(ctx, req)
	s.syncAgentSkillEmbedding(ctx, resp)
	return resp, err
}

func (s nativeAgentSkillService) UpdateAgentSkill(ctx context.Context, req *pb.UpdateAgentSkillRequest) (*pb.AgentSkillResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.AgentSkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	resp, err := store.UpdateAgentSkill(ctx, req)
	s.syncAgentSkillEmbedding(ctx, resp)
	return resp, err
}

func (s nativeAgentSkillService) CreateAgentSkillVersion(ctx context.Context, req *pb.CreateAgentSkillVersionRequest) (*pb.AgentSkillVersionResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.AgentSkillVersionResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	resp, err := store.CreateAgentSkillVersion(ctx, req)
	if err == nil && req.GetActivate() && resp != nil && resp.GetCode() == 0 {
		skillResp, getErr := store.GetAgentSkill(ctx, &pb.GetAgentSkillRequest{Id: req.GetSkillId()})
		if getErr == nil {
			s.syncAgentSkillEmbedding(ctx, skillResp)
		}
	}
	return resp, err
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
	resp, err := store.ActivateAgentSkillVersion(ctx, req)
	s.syncAgentSkillEmbedding(ctx, resp)
	return resp, err
}

func (s nativeAgentSkillService) UpdateAgentSkillStatus(ctx context.Context, req *pb.UpdateAgentSkillStatusRequest) (*pb.AgentSkillResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.AgentSkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	resp, err := store.UpdateAgentSkillStatus(ctx, req)
	s.syncAgentSkillEmbedding(ctx, resp)
	return resp, err
}

func (s nativeAgentSkillService) PreviewAgentSkill(ctx context.Context, req *pb.PreviewAgentSkillRequest) (*pb.PreviewAgentSkillResponse, error) {
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.PreviewAgentSkillResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured"}, nil
	}
	return store.PreviewAgentSkill(ctx, req)
}

func (s nativeAgentSkillService) DebugSemanticRetrieval(ctx context.Context, req *pb.DebugSemanticRetrievalRequest) (*pb.DebugSemanticRetrievalResponse, error) {
	if s.embedding != nil {
		return s.embedding.SearchAgentSkills(ctx, req.GetQuery(), int(req.GetLimit()))
	}
	store, ok := s.store.(agentSkillGovernanceStore)
	if !ok {
		return &pb.DebugSemanticRetrievalResponse{Code: configCodeUnavailable, Msg: "ai governance store is not configured", EmbeddingAvailable: false, FallbackReason: "store is not configured"}, nil
	}
	return store.DebugSemanticRetrieval(ctx, req)
}

func (s nativeAgentSkillService) syncAgentSkillEmbedding(ctx context.Context, resp *pb.AgentSkillResponse) {
	if s.embedding == nil || resp == nil || resp.GetCode() != 0 || resp.GetSkill() == nil {
		return
	}
	skill := resp.GetSkill()
	if skill.GetIsEnabled() && skill.GetCurrentVersionId() > 0 {
		_ = s.embedding.UpsertAgentSkill(ctx, skill.GetId())
		return
	}
	_ = s.embedding.InvalidateAgentSkill(ctx, skill.GetId())
}
