package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"logic-grpc-service/config"
	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

// MCPService implements the pb.MCPServiceServer interface.
type MCPService struct {
	pb.UnimplementedMCPServiceServer
	mcpRepo *repository.MCPRepo
	cfg     config.Config
}

// NewMCPService creates a new MCPService.
func NewMCPService(mcpRepo *repository.MCPRepo, cfg config.Config) *MCPService {
	return &MCPService{
		mcpRepo: mcpRepo,
		cfg:     cfg,
	}
}

// ── Server CRUD ─────────────────────────────────────────────────────

func (s *MCPService) ListMCPServers(ctx context.Context, req *pb.ListMCPServersRequest) (*pb.ListMCPServersResponse, error) {
	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}

	servers, total, err := s.mcpRepo.ListServers(ctx, page, pageSize)
	if err != nil {
		logger.L().Error("list MCP servers failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list MCP servers failed")
	}

	list := make([]*pb.MCPServerInfo, 0, len(servers))
	for i := range servers {
		list = append(list, serverToInfo(&servers[i]))
	}

	return &pb.ListMCPServersResponse{
		Code:  0,
		Msg:   "ok",
		Total: total,
		List:  list,
	}, nil
}

func (s *MCPService) CreateMCPServer(ctx context.Context, req *pb.CreateMCPServerRequest) (*pb.MCPServerResponse, error) {
	if strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if strings.TrimSpace(req.GetTransport()) == "" {
		return nil, status.Error(codes.InvalidArgument, "transport is required")
	}
	transportType := strings.ToLower(strings.TrimSpace(req.GetTransport()))
	if transportType != "stdio" && transportType != "sse" && transportType != "http" {
		return nil, status.Error(codes.InvalidArgument, "transport must be one of: stdio, sse, http")
	}
	if strings.TrimSpace(req.GetCommandOrUrl()) == "" {
		return nil, status.Error(codes.InvalidArgument, "command_or_url is required")
	}

	timeout := req.GetTimeoutSeconds()
	if timeout <= 0 {
		timeout = int32(s.cfg.MCP.DefaultTimeoutSeconds)
	}

	var argsStr *string
	if req.GetArgs() != "" {
		a := req.GetArgs()
		argsStr = &a
	}
	var envStr *string
	if req.GetEnvVars() != "" {
		e := req.GetEnvVars()
		envStr = &e
	}

	server := &model.MCPServer{
		Name:           strings.TrimSpace(req.GetName()),
		Transport:      transportType,
		CommandOrURL:   strings.TrimSpace(req.GetCommandOrUrl()),
		Args:           argsStr,
		EnvVars:        envStr,
		TimeoutSeconds: timeout,
		IsEnabled:      0, // Default disabled for safety
	}

	if err := s.mcpRepo.CreateServer(ctx, server); err != nil {
		logger.L().Error("create MCP server failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create MCP server failed")
	}

	return &pb.MCPServerResponse{
		Code:   0,
		Msg:    "ok",
		Server: serverToInfo(server),
	}, nil
}

func (s *MCPService) UpdateMCPServer(ctx context.Context, req *pb.UpdateMCPServerRequest) (*pb.MCPServerResponse, error) {
	id := req.GetId()
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	existing, err := s.mcpRepo.GetServerByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "MCP server not found")
		}
		logger.L().Error("get MCP server failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get MCP server failed")
	}

	updates := map[string]any{}
	if req.GetName() != "" {
		updates["name"] = strings.TrimSpace(req.GetName())
	}
	if req.GetTransport() != "" {
		transportType := strings.ToLower(strings.TrimSpace(req.GetTransport()))
		if transportType != "stdio" && transportType != "sse" && transportType != "http" {
			return nil, status.Error(codes.InvalidArgument, "transport must be one of: stdio, sse, http")
		}
		updates["transport"] = transportType
	}
	if req.GetCommandOrUrl() != "" {
		updates["command_or_url"] = strings.TrimSpace(req.GetCommandOrUrl())
	}
	if req.GetArgs() != "" {
		a := req.GetArgs()
		updates["args"] = &a
	}
	if req.GetEnvVars() != "" {
		e := req.GetEnvVars()
		updates["env_vars"] = &e
	}
	if req.GetTimeoutSecondsSet() {
		updates["timeout_seconds"] = req.GetTimeoutSeconds()
	}
	if req.GetIsEnabledSet() {
		v := int32(0)
		if req.GetIsEnabled() {
			v = 1
		}
		updates["is_enabled"] = v
	}

	if len(updates) > 0 {
		if err := s.mcpRepo.UpdateServerPartial(ctx, id, updates); err != nil {
			logger.L().Error("update MCP server failed", zap.Error(err))
			return nil, status.Error(codes.Internal, "update MCP server failed")
		}
	}

	// Re-fetch updated server
	updated, err := s.mcpRepo.GetServerByID(ctx, id)
	if err != nil {
		logger.L().Error("get updated MCP server failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get updated MCP server failed")
	}

	_ = existing

	return &pb.MCPServerResponse{
		Code:   0,
		Msg:    "ok",
		Server: serverToInfo(updated),
	}, nil
}

func (s *MCPService) DeleteMCPServer(ctx context.Context, req *pb.DeleteMCPServerRequest) (*pb.CommonResponse, error) {
	id := req.GetId()
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.mcpRepo.DeleteServer(ctx, id); err != nil {
		logger.L().Error("delete MCP server failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "delete MCP server failed")
	}

	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}

// ── Connection Test ─────────────────────────────────────────────────

func (s *MCPService) TestMCPConnection(ctx context.Context, req *pb.TestMCPConnectionRequest) (*pb.TestMCPConnectionResponse, error) {
	serverID := req.GetServerId()
	if serverID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "server_id is required")
	}

	server, err := s.mcpRepo.GetServerByID(ctx, serverID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.TestMCPConnectionResponse{
				Code:    1,
				Msg:     "MCP server not found",
				Success: false,
				Detail:  "MCP server not found",
			}, nil
		}
		logger.L().Error("get MCP server failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get MCP server failed")
	}

	// Use a longer timeout for connection testing (60s) to allow npx cold start
	timeout := server.TimeoutSeconds
	if timeout <= 0 || timeout < 60 {
		timeout = 60
	}
	testCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	mcpClient, err := s.createClient(server)
	if err != nil {
		return &pb.TestMCPConnectionResponse{
			Code:    1,
			Msg:     "create client failed",
			Success: false,
			Detail:  fmt.Sprintf("failed to create MCP client: %v", err),
		}, nil
	}
	defer mcpClient.Close()

	// Initialize the connection
	initResult, err := mcpClient.Initialize(testCtx, mcp.InitializeRequest{})
	if err != nil {
		return &pb.TestMCPConnectionResponse{
			Code:    1,
			Msg:     "initialize failed",
			Success: false,
			Detail:  fmt.Sprintf("initialization failed: %v", err),
		}, nil
	}

	detail := fmt.Sprintf("connected successfully, server version: %s", initResult.ServerInfo.Name)
	return &pb.TestMCPConnectionResponse{
		Code:    0,
		Msg:     "ok",
		Success: true,
		Detail:  detail,
	}, nil
}

// ── Tool Listing ────────────────────────────────────────────────────

func (s *MCPService) ListMCPTools(ctx context.Context, req *pb.ListMCPToolsRequest) (*pb.ListMCPToolsResponse, error) {
	serverID := req.GetServerId()
	if serverID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "server_id is required")
	}

	server, err := s.mcpRepo.GetServerByID(ctx, serverID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "MCP server not found")
		}
		logger.L().Error("get MCP server failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get MCP server failed")
	}

	mcpClient, err := s.createClient(server)
	if err != nil {
		logger.L().Error("create MCP client failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create MCP client failed")
	}
	defer mcpClient.Close()

	listCtx, cancel := s.mcpTimeoutCtx(ctx, server)
	defer cancel()

	initResult, err := mcpClient.Initialize(listCtx, mcp.InitializeRequest{})
	if err != nil {
		logger.L().Error("MCP initialize failed", zap.Error(err))
		return nil, status.Error(codes.Internal, fmt.Sprintf("MCP initialize failed: %v", err))
	}
	_ = initResult

	toolsResult, err := mcpClient.ListTools(listCtx, mcp.ListToolsRequest{})
	if err != nil {
		logger.L().Error("list MCP tools failed", zap.Error(err))
		return nil, status.Error(codes.Internal, fmt.Sprintf("list MCP tools failed: %v", err))
	}

	tools := make([]*pb.MCPToolInfo, 0, len(toolsResult.Tools))
	for _, t := range toolsResult.Tools {
		schemaJSON := ""
		if schemaBytes, err := json.Marshal(t.InputSchema); err == nil {
			schemaJSON = string(schemaBytes)
		}
		tools = append(tools, &pb.MCPToolInfo{
			Name:        t.Name,
			Description: t.Description,
			SchemaJson:  schemaJSON,
		})
	}

	return &pb.ListMCPToolsResponse{
		Code: 0,
		Msg:  "ok",
		List: tools,
	}, nil
}

func (s *MCPService) ListEnabledMCPCapabilities(ctx context.Context) ([]*pb.CapabilityInfo, error) {
	servers, err := s.mcpRepo.ListEnabledServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled servers: %w", err)
	}

	var capabilities []*pb.CapabilityInfo
	for i := range servers {
		server := &servers[i]
		tools, err := s.listToolsForServer(ctx, server)
		if err != nil {
			logger.L().Warn("list MCP capabilities failed",
				zap.Int64("server_id", server.ID),
				zap.String("server_name", server.Name),
				zap.Error(err))
			continue
		}
		for _, t := range tools {
			key := MCPCapabilityKey(server.ID, t.Name)
			capabilities = append(capabilities, &pb.CapabilityInfo{
				Source:        "mcp",
				Key:           key,
				Name:          MCPRuntimeToolName(server.Name, t.Name),
				DisplayName:   fmt.Sprintf("%s / %s", server.Name, t.Name),
				Description:   t.Description,
				McpServerId:   server.ID,
				McpServerName: server.Name,
				IsAvailable:   true,
			})
		}
	}
	return capabilities, nil
}

func (s *MCPService) listToolsForServer(ctx context.Context, server *model.MCPServer) ([]mcp.Tool, error) {
	mcpClient, err := s.createClient(server)
	if err != nil {
		return nil, fmt.Errorf("create MCP client: %w", err)
	}
	defer mcpClient.Close()

	listCtx, cancel := s.mcpTimeoutCtx(ctx, server)
	defer cancel()

	if _, err := mcpClient.Initialize(listCtx, mcp.InitializeRequest{}); err != nil {
		return nil, fmt.Errorf("initialize MCP client: %w", err)
	}
	toolsResult, err := mcpClient.ListTools(listCtx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list MCP tools: %w", err)
	}
	return toolsResult.Tools, nil
}

// ── Tool Call ───────────────────────────────────────────────────────

func (s *MCPService) CallMCPTool(ctx context.Context, req *pb.CallMCPToolRequest) (*pb.CallMCPToolResponse, error) {
	serverID := req.GetServerId()
	if serverID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "server_id is required")
	}
	if strings.TrimSpace(req.GetToolName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "tool_name is required")
	}

	server, err := s.mcpRepo.GetServerByID(ctx, serverID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "MCP server not found")
		}
		logger.L().Error("get MCP server failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "get MCP server failed")
	}

	if server.IsEnabled != 1 {
		return nil, status.Error(codes.PermissionDenied, "MCP server is disabled")
	}

	startTime := time.Now()

	mcpClient, err := s.createClient(server)
	if err != nil {
		logger.L().Error("create MCP client failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create MCP client failed")
	}
	defer mcpClient.Close()

	callCtx, cancel := s.mcpTimeoutCtx(ctx, server)
	defer cancel()

	initResult, err := mcpClient.Initialize(callCtx, mcp.InitializeRequest{})
	if err != nil {
		logger.L().Error("MCP initialize failed", zap.Error(err))
		return nil, status.Error(codes.Internal, fmt.Sprintf("MCP initialize failed: %v", err))
	}
	_ = initResult

	// Parse arguments JSON
	var args any
	argsStr := req.GetArgsJson()
	if argsStr != "" {
		if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid args_json: %v", err))
		}
	}

	callResult, err := mcpClient.CallTool(callCtx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      req.GetToolName(),
			Arguments: args,
		},
	})
	durationMs := time.Since(startTime).Milliseconds()

	// Extract result content
	resultText := extractResultText(callResult)
	var errorMsg string
	if callResult != nil && callResult.IsError {
		errorMsg = resultText
	}

	if err != nil {
		errorMsg = err.Error()
		durationMs = time.Since(startTime).Milliseconds()
	}

	// Desensitize args and result for logging
	desensitizedArgs := desensitizeJSON(argsStr)
	desensitizedResult := truncateString(desensitizeJSON(resultText), 4096)

	// Log the tool call
	toolLog := &model.MCPToolLog{
		ServerID:      serverID,
		ToolName:      req.GetToolName(),
		ArgsJSON:      &desensitizedArgs,
		ResultContent: &desensitizedResult,
		DurationMs:    int32(durationMs),
		ErrorMsg:      strPtrOrNil(errorMsg),
		CalledByHRID:  &req.CalledByHrId,
		SessionID:     &req.SessionId,
	}
	if req.GetCalledByHrId() <= 0 {
		toolLog.CalledByHRID = nil
	}
	if req.GetSessionId() <= 0 {
		toolLog.SessionID = nil
	}
	if logErr := s.mcpRepo.CreateToolLog(ctx, toolLog); logErr != nil {
		logger.L().Warn("create MCP tool log failed", zap.Error(logErr))
	}

	if err != nil {
		return &pb.CallMCPToolResponse{
			Code:          1,
			Msg:           "tool call failed",
			ResultContent: "",
			ErrorMsg:      errorMsg,
			DurationMs:    durationMs,
		}, nil
	}

	return &pb.CallMCPToolResponse{
		Code:          0,
		Msg:           "ok",
		ResultContent: resultText,
		ErrorMsg:      errorMsg,
		DurationMs:    durationMs,
	}, nil
}

// ── Internal Methods ────────────────────────────────────────────────

// createClient creates an MCP client based on the server's transport configuration.
func (s *MCPService) createClient(server *model.MCPServer) (client.MCPClient, error) {
	switch server.Transport {
	case "stdio":
		return s.createStdioClient(server)
	case "sse":
		return s.createSSEClient(server)
	case "http":
		return s.createHTTPClient(server)
	default:
		return nil, fmt.Errorf("unsupported transport: %s", server.Transport)
	}
}

func (s *MCPService) createStdioClient(server *model.MCPServer) (client.MCPClient, error) {
	var args []string
	if server.Args != nil && *server.Args != "" {
		if err := json.Unmarshal([]byte(*server.Args), &args); err != nil {
			return nil, fmt.Errorf("parse args: %w", err)
		}
	}

	var env []string
	if server.EnvVars != nil && *server.EnvVars != "" {
		var envMap map[string]string
		if err := json.Unmarshal([]byte(*server.EnvVars), &envMap); err != nil {
			return nil, fmt.Errorf("parse env_vars: %w", err)
		}
		for k, v := range envMap {
			env = append(env, k+"="+v)
		}
	}

	return client.NewStdioMCPClient(server.CommandOrURL, env, args...)
}

func (s *MCPService) createSSEClient(server *model.MCPServer) (client.MCPClient, error) {
	return client.NewSSEMCPClient(server.CommandOrURL)
}

func (s *MCPService) createHTTPClient(server *model.MCPServer) (client.MCPClient, error) {
	return client.NewStreamableHttpClient(server.CommandOrURL)
}

// mcpTimeoutCtx returns a context with a timeout derived from the server's configured timeout.
// Ensures the timeout is at least 60s for stdio transports (npx cold start can be slow).
func (s *MCPService) mcpTimeoutCtx(parent context.Context, server *model.MCPServer) (context.Context, context.CancelFunc) {
	timeout := server.TimeoutSeconds
	if timeout <= 0 {
		timeout = int32(s.cfg.MCP.DefaultTimeoutSeconds)
	}
	if timeout < 60 && server.Transport == "stdio" {
		timeout = 60
	}
	return context.WithTimeout(parent, time.Duration(timeout)*time.Second)
}

// serverToInfo converts a model.MCPServer to a pb.MCPServerInfo.
// env_vars values are desensitized to prevent credential leakage through API responses.
func serverToInfo(s *model.MCPServer) *pb.MCPServerInfo {
	args := ""
	if s.Args != nil {
		args = *s.Args
	}
	envVars := ""
	if s.EnvVars != nil {
		envVars = desensitizeEnvVars(*s.EnvVars)
	}
	return &pb.MCPServerInfo{
		Id:             s.ID,
		Name:           s.Name,
		Transport:      s.Transport,
		CommandOrUrl:   s.CommandOrURL,
		Args:           args,
		EnvVars:        envVars,
		TimeoutSeconds: s.TimeoutSeconds,
		IsEnabled:      s.IsEnabled == 1,
		CreatedAt:      s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      s.UpdatedAt.Format(time.RFC3339),
	}
}

// desensitizeEnvVars masks all values in a JSON env_vars string.
// It parses the JSON object and replaces every value with "***",
// then returns the re-marshaled JSON. On any parse/marshal error,
// it returns the original string unchanged.
func desensitizeEnvVars(envVarsJSON string) string {
	if envVarsJSON == "" {
		return ""
	}
	var envMap map[string]any
	if err := json.Unmarshal([]byte(envVarsJSON), &envMap); err != nil {
		return envVarsJSON
	}
	for k := range envMap {
		envMap[k] = "***"
	}
	result, err := json.Marshal(envMap)
	if err != nil {
		return envVarsJSON
	}
	return string(result)
}

// extractResultText extracts text content from a CallToolResult.
func extractResultText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	var texts []string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			texts = append(texts, textContent.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// desensitizeJSON desensitizes potential sensitive data in JSON strings.
// It parses the input as JSON, recursively walks all fields, and masks values
// of known sensitive field names (phone, email, API key, password, ID card, etc.)
// with "***". If the input is not valid JSON, it falls back to regex-based PII masking.
func desensitizeJSON(s string) string {
	if s == "" {
		return ""
	}

	var data any
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		// Not valid JSON — fall back to regex-based PII masking
		return desensitizePlainText(s)
	}

	desensitized := desensitizeValue(data)
	result, err := json.Marshal(desensitized)
	if err != nil {
		return s // fallback to original on marshal error
	}
	return string(result)
}

// desensitizePlainText applies regex-based PII masking to plain text.
func desensitizePlainText(s string) string {
	// Phone numbers: 1xx-xxxx-xxxx or 1xxxxxxxxx
	rePhone := regexp.MustCompile(`1[3-9]\d{1}[\s\-]?\d{4}[\s\-]?\d{4}`)
	s = rePhone.ReplaceAllString(s, "***")

	// Email addresses
	reEmail := regexp.MustCompile(`[\w.+-]+@[\w-]+\.[\w.]+`)
	s = reEmail.ReplaceAllString(s, "***")

	// ID card numbers (18 digits, possibly with X suffix)
	reIDCard := regexp.MustCompile(`\d{6}[\s\-]?\d{8}[\s\-]?[\dXx]{4}`)
	s = reIDCard.ReplaceAllString(s, "***")

	return s
}

// desensitizeValue recursively walks an arbitrary value and masks sensitive fields.
func desensitizeValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for key, subVal := range val {
			if isSensitiveField(key) {
				val[key] = "***"
			} else {
				val[key] = desensitizeValue(subVal)
			}
		}
		return val
	case []any:
		for i, item := range val {
			val[i] = desensitizeValue(item)
		}
		return val
	default:
		return v
	}
}

// isSensitiveField checks whether a JSON field name is considered sensitive.
func isSensitiveField(name string) bool {
	lower := strings.ToLower(name)
	sensitiveKeys := []string{
		"phone", "mobile", "tel", "telephone",
		"email", "mail",
		"api_key", "apikey", "api-key", "api key",
		"apisecret", "api_secret", "api-secret",
		"password", "passwd", "pwd", "pass",
		"secret", "secretkey", "secret_key", "secret-key",
		"token", "accesstoken", "access_token", "access-token",
		"idcard", "id_card", "id-card", "idnumber", "id_number", "id-number",
		"identity", "identitynumber", "identity_number",
		"ssn", "socialsecurity",
		"private_key", "privatekey", "private-key",
		"credential", "credentials",
		"auth", "authorization", "auth_token", "auth-token",
		"appkey", "app_key", "app-key",
		"appsecret", "app_secret", "app-secret",
		"sessionkey", "session_key", "session-key",
		"refresh_token", "refreshtoken", "refresh-token",
	}
	for _, key := range sensitiveKeys {
		if lower == key {
			return true
		}
		// Also check if the key contains common sensitive substrings
		if strings.Contains(lower, "password") || strings.Contains(lower, "secret") ||
			strings.Contains(lower, "private_key") || strings.Contains(lower, "credential") {
			return true
		}
	}
	return false
}

// strPtrOrNil returns a pointer to the string, or nil if empty.
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ── Agent Tool Injection ───────────────────────────────────────────

func MCPCapabilityKey(serverID int64, toolName string) string {
	return fmt.Sprintf("%d:%s", serverID, toolName)
}

func MCPRuntimeToolName(serverName, toolName string) string {
	return fmt.Sprintf("mcp_%s_%s", serverName, toolName)
}

// CollectEnabledMCPToolInfos collects tool definitions from enabled MCP servers
// and returns them as Eino schema.ToolInfo slice, suitable for injection into the
// ADK Agent tool list.
func (s *MCPService) CollectEnabledMCPToolInfos(ctx context.Context) ([]*schema.ToolInfo, error) {
	return s.CollectBoundMCPToolInfos(ctx, nil)
}

func (s *MCPService) CollectBoundMCPToolInfos(ctx context.Context, allowedKeys map[string]bool) ([]*schema.ToolInfo, error) {
	servers, err := s.mcpRepo.ListEnabledServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled servers: %w", err)
	}

	var allTools []*schema.ToolInfo
	for i := range servers {
		server := &servers[i]
		mcpClient, err := s.createClient(server)
		if err != nil {
			logger.L().Warn("failed to create MCP client for server",
				zap.Int64("server_id", server.ID),
				zap.String("server_name", server.Name),
				zap.Error(err))
			continue
		}

		func() {
			defer mcpClient.Close()
			listCtx, cancel := s.mcpTimeoutCtx(ctx, server)
			defer cancel()

			initResult, err := mcpClient.Initialize(listCtx, mcp.InitializeRequest{})
			if err != nil {
				logger.L().Warn("MCP initialize failed for server",
					zap.Int64("server_id", server.ID),
					zap.Error(err))
				return
			}
			_ = initResult

			toolsResult, err := mcpClient.ListTools(listCtx, mcp.ListToolsRequest{})
			if err != nil {
				logger.L().Warn("list MCP tools failed for server",
					zap.Int64("server_id", server.ID),
					zap.Error(err))
				return
			}

			for _, t := range toolsResult.Tools {
				if allowedKeys != nil && !allowedKeys[MCPCapabilityKey(server.ID, t.Name)] {
					continue
				}
				qualifiedName := MCPRuntimeToolName(server.Name, t.Name)
				toolInfo := &schema.ToolInfo{
					Name: qualifiedName,
					Desc: t.Description,
				}

				// Convert input schema to Eino ParameterInfo map
				if params := mcpSchemaToEinoParams(t.InputSchema); len(params) > 0 {
					toolInfo.ParamsOneOf = schema.NewParamsOneOfByParams(params)
				}

				allTools = append(allTools, toolInfo)
			}
		}()
	}

	return allTools, nil
}

// mcpSchemaToEinoParams converts MCP ToolInputSchema to Eino ParameterInfo map.
func mcpSchemaToEinoParams(inputSchema mcp.ToolInputSchema) map[string]*schema.ParameterInfo {
	params := make(map[string]*schema.ParameterInfo)
	requiredSet := make(map[string]bool, len(inputSchema.Required))
	for _, r := range inputSchema.Required {
		requiredSet[r] = true
	}
	for name, propAny := range inputSchema.Properties {
		prop, ok := propAny.(map[string]any)
		if !ok {
			continue
		}
		param := &schema.ParameterInfo{
			Desc:     getStringField(prop, "description"),
			Required: requiredSet[name],
		}
		param.Type = schemaTypeFromJSONSchema(getStringField(prop, "type"))
		params[name] = param
	}
	return params
}

// schemaTypeFromJSONSchema maps JSON Schema types to Eino schema types.
func schemaTypeFromJSONSchema(t string) schema.DataType {
	switch t {
	case "string":
		return schema.String
	case "integer", "number":
		return schema.Integer
	case "boolean":
		return schema.Boolean
	case "array":
		return schema.Array
	case "object":
		return schema.Object
	default:
		return schema.String
	}
}

// getStringField safely extracts a string field from a map.
func getStringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// CollectEnabledMCPCallableTools collects callable tool wrappers from all enabled
// MCP servers. Each wrapper implements tool.BaseTool and tool.InvokableTool so it
// can be used with the ADK runtime (ChatWithADKAgent). When invoked, the wrapper
// connects to the MCP server and calls the tool.
func (s *MCPService) CollectEnabledMCPCallableTools(ctx context.Context) ([]tool.BaseTool, error) {
	return s.CollectBoundMCPCallableTools(ctx, nil)
}

func (s *MCPService) CollectBoundMCPCallableTools(ctx context.Context, allowedKeys map[string]bool) ([]tool.BaseTool, error) {
	servers, err := s.mcpRepo.ListEnabledServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled servers: %w", err)
	}

	var allTools []tool.BaseTool
	for i := range servers {
		server := &servers[i]
		mcpClient, err := s.createClient(server)
		if err != nil {
			logger.L().Warn("failed to create MCP client for ADK tool",
				zap.Int64("server_id", server.ID),
				zap.String("server_name", server.Name),
				zap.Error(err))
			continue
		}

		func() {
			defer mcpClient.Close()
			listCtx, cancel := s.mcpTimeoutCtx(ctx, server)
			defer cancel()

			initResult, err := mcpClient.Initialize(listCtx, mcp.InitializeRequest{})
			if err != nil {
				logger.L().Warn("MCP initialize failed for ADK tool",
					zap.Int64("server_id", server.ID),
					zap.Error(err))
				return
			}
			_ = initResult

			toolsResult, err := mcpClient.ListTools(listCtx, mcp.ListToolsRequest{})
			if err != nil {
				logger.L().Warn("list MCP tools failed for ADK tool",
					zap.Int64("server_id", server.ID),
					zap.Error(err))
				return
			}

			for _, t := range toolsResult.Tools {
				if allowedKeys != nil && !allowedKeys[MCPCapabilityKey(server.ID, t.Name)] {
					continue
				}
				qualifiedName := MCPRuntimeToolName(server.Name, t.Name)
				toolInfo := &schema.ToolInfo{
					Name: qualifiedName,
					Desc: t.Description,
				}
				if params := mcpSchemaToEinoParams(t.InputSchema); len(params) > 0 {
					toolInfo.ParamsOneOf = schema.NewParamsOneOfByParams(params)
				}

				sid := server.ID
				tName := t.Name
				wrapper := utils.NewTool(toolInfo, func(ctx context.Context, argsJSON string) (string, error) {
					resp, err := s.CallMCPTool(ctx, &pb.CallMCPToolRequest{
						ServerId: sid,
						ToolName: tName,
						ArgsJson: argsJSON,
					})
					if err != nil {
						return "", err
					}
					return resp.ResultContent, nil
				})
				allTools = append(allTools, wrapper)
			}
		}()
	}

	return allTools, nil
}
