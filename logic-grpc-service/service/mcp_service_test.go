package service

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/config"
	"logic-grpc-service/model"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

// ── desensitizeJSON tests ─────────────────────────────────────────────

func TestDesensitizeJSON_Empty(t *testing.T) {
	if got := desensitizeJSON(""); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestDesensitizeJSON_NonJSON(t *testing.T) {
	input := "hello world 13800138000"
	got := desensitizeJSON(input)
	if strings.Contains(got, "13800138000") {
		t.Fatalf("expected phone number to be masked, got %q", got)
	}
}

func TestDesensitizeJSON_JSONWithSensitiveFields(t *testing.T) {
	input := `{"name":"John","phone":"13800138000","email":"john@example.com","api_key":"sk-123456"}`
	got := desensitizeJSON(input)
	if strings.Contains(got, "13800138000") {
		t.Fatalf("expected phone to be masked, got %q", got)
	}
	if strings.Contains(got, "john@example.com") {
		t.Fatalf("expected email to be masked, got %q", got)
	}
	if strings.Contains(got, "sk-123456") {
		t.Fatalf("expected api_key to be masked, got %q", got)
	}
	if !strings.Contains(got, "John") {
		t.Fatalf("expected name 'John' to remain, got %q", got)
	}
	if !strings.Contains(got, "***") {
		t.Fatalf("expected masked values '***' in output, got %q", got)
	}
}

func TestDesensitizeJSON_NestedJSON(t *testing.T) {
	input := `{"user":{"phone":"13912345678","name":"Alice"},"credentials":{"access_token":"tok_abc","password":"p@ss"}}`
	got := desensitizeJSON(input)
	if strings.Contains(got, "13912345678") {
		t.Fatalf("expected phone in nested JSON to be masked, got %q", got)
	}
	if strings.Contains(got, "tok_abc") {
		t.Fatalf("expected access_token to be masked, got %q", got)
	}
	if strings.Contains(got, "p@ss") {
		t.Fatalf("expected password to be masked, got %q", got)
	}
	if !strings.Contains(got, "Alice") {
		t.Fatalf("expected name 'Alice' to remain, got %q", got)
	}
}

func TestDesensitizeJSON_ArrayOfObjects(t *testing.T) {
	input := `[{"name":"Tool1","api_key":"key1"},{"name":"Tool2","api_key":"key2"}]`
	got := desensitizeJSON(input)
	if strings.Contains(got, "key1") || strings.Contains(got, "key2") {
		t.Fatalf("expected api_key values in arrays to be masked, got %q", got)
	}
}

func TestDesensitizeJSON_PlainTextEmailAndPhone(t *testing.T) {
	input := "请联系 13800138000 或 test@example.com"
	got := desensitizeJSON(input)
	if strings.Contains(got, "13800138000") {
		t.Fatalf("expected phone in plain text to be masked, got %q", got)
	}
	if strings.Contains(got, "test@example.com") {
		t.Fatalf("expected email in plain text to be masked, got %q", got)
	}
}

// ── desensitizeEnvVars tests ──────────────────────────────────────────

func TestDesensitizeEnvVars_Empty(t *testing.T) {
	if got := desensitizeEnvVars(""); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestDesensitizeEnvVars_AllValuesMasked(t *testing.T) {
	input := `{"API_KEY":"sk-xxx","DB_PASSWORD":"secret123","HOST":"localhost"}`
	got := desensitizeEnvVars(input)
	var parsed map[string]string
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v, got %q", err, got)
	}
	if parsed["API_KEY"] != "***" {
		t.Fatalf("expected API_KEY to be '***', got %q", parsed["API_KEY"])
	}
	if parsed["DB_PASSWORD"] != "***" {
		t.Fatalf("expected DB_PASSWORD to be '***', got %q", parsed["DB_PASSWORD"])
	}
	if parsed["HOST"] != "***" {
		t.Fatalf("expected HOST to be '***', got %q", parsed["HOST"])
	}
}

func TestDesensitizeEnvVars_InvalidJSON(t *testing.T) {
	input := `not-json`
	got := desensitizeEnvVars(input)
	if got != "{}" {
		t.Fatalf("expected '{}' for invalid JSON (not original), got %q", got)
	}
	if strings.Contains(got, "not-json") {
		t.Fatalf("expected no raw values leaked on parse failure, got %q", got)
	}
}

// ── serverToInfo tests ────────────────────────────────────────────────

func TestServerToInfo_DesensitizesEnvVars(t *testing.T) {
	envVars := `{"API_KEY":"sk-xxx","SECRET":"s3cr3t"}`
	srv := &model.MCPServer{
		ID:             1,
		Name:           "test",
		Transport:      "stdio",
		CommandOrURL:   "/usr/bin/test",
		EnvVars:        &envVars,
		TimeoutSeconds: 30,
		IsEnabled:      1,
	}
	info := serverToInfo(srv)
	if info.EnvVars == envVars {
		t.Fatal("expected env_vars to be desensitized in serverToInfo output")
	}
	if !strings.Contains(info.EnvVars, "***") {
		t.Fatalf("expected env_vars to contain masked values '***', got %q", info.EnvVars)
	}
	if strings.Contains(info.EnvVars, "sk-xxx") || strings.Contains(info.EnvVars, "s3cr3t") {
		t.Fatalf("expected env_vars original values to be masked, got %q", info.EnvVars)
	}
	if info.Name != "test" || info.Transport != "stdio" {
		t.Fatalf("expected non-sensitive fields to remain intact")
	}
}

func TestServerToInfo_NilPtrs(t *testing.T) {
	srv := &model.MCPServer{
		ID:             2,
		Name:           "no-env-server",
		Transport:      "sse",
		CommandOrURL:   "http://localhost:8080",
		TimeoutSeconds: 30,
	}
	info := serverToInfo(srv)
	if info.EnvVars != "" {
		t.Fatalf("expected empty env_vars, got %q", info.EnvVars)
	}
	if info.Args != "" {
		t.Fatalf("expected empty args, got %q", info.Args)
	}
	if info.Name != "no-env-server" {
		t.Fatalf("expected name to be preserved, got %q", info.Name)
	}
}

func TestServerToInfo_IsEnabled(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		srv := &model.MCPServer{
			ID: 1, Name: "e", Transport: "stdio", CommandOrURL: "/bin/test",
			IsEnabled: 1,
		}
		info := serverToInfo(srv)
		if !info.IsEnabled {
			t.Fatal("expected IsEnabled=true when model.IsEnabled=1")
		}
	})
	t.Run("disabled", func(t *testing.T) {
		srv := &model.MCPServer{
			ID: 2, Name: "d", Transport: "stdio", CommandOrURL: "/bin/test",
			IsEnabled: 0,
		}
		info := serverToInfo(srv)
		if info.IsEnabled {
			t.Fatal("expected IsEnabled=false when model.IsEnabled=0")
		}
	})
}

// ── extractResultText tests ───────────────────────────────────────────

func TestExtractResultText_Nil(t *testing.T) {
	if got := extractResultText(nil); got != "" {
		t.Fatalf("expected empty string for nil result, got %q", got)
	}
}

func TestExtractResultText_EmptyContent(t *testing.T) {
	result := &mcp.CallToolResult{}
	if got := extractResultText(result); got != "" {
		t.Fatalf("expected empty string for empty result, got %q", got)
	}
}

func TestExtractResultText_TextContent(t *testing.T) {
	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Text: "line1"},
			mcp.TextContent{Text: "line2"},
		},
	}
	got := extractResultText(result)
	if got != "line1\nline2" {
		t.Fatalf("expected 'line1\\nline2', got %q", got)
	}
}

// ── schemaTypeFromJSONSchema tests ────────────────────────────────────

func TestSchemaTypeFromJSONSchema(t *testing.T) {
	tests := []struct {
		input string
		want  schema.DataType
	}{
		{"string", schema.String},
		{"integer", schema.Integer},
		{"number", schema.Integer},
		{"boolean", schema.Boolean},
		{"array", schema.Array},
		{"object", schema.Object},
		{"unknown", schema.String},
		{"", schema.String},
	}
	for _, tt := range tests {
		got := schemaTypeFromJSONSchema(tt.input)
		if got != tt.want {
			t.Errorf("schemaTypeFromJSONSchema(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ── getStringField tests ──────────────────────────────────────────────

func TestGetStringField(t *testing.T) {
	m := map[string]any{
		"name": "test",
		"desc": 123,
	}
	if got := getStringField(m, "name"); got != "test" {
		t.Fatalf("expected 'test', got %q", got)
	}
	if got := getStringField(m, "desc"); got != "" {
		t.Fatalf("expected empty string for non-string value, got %q", got)
	}
	if got := getStringField(m, "missing"); got != "" {
		t.Fatalf("expected empty string for missing key, got %q", got)
	}
}

// ── mcpSchemaToEinoParams tests ───────────────────────────────────────

func TestMCPSchemaToEinoParams(t *testing.T) {
	inputSchema := mcp.ToolInputSchema{
		Properties: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "search query",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "max results",
			},
		},
		Required: []string{"query"},
	}
	params := mcpSchemaToEinoParams(inputSchema)
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	if params["query"] == nil || params["query"].Desc != "search query" {
		t.Fatalf("expected query param with description 'search query'")
	}
	if !params["query"].Required {
		t.Fatal("expected 'query' to be required")
	}
	if params["limit"].Required {
		t.Fatal("expected 'limit' to not be required")
	}
}

func TestMCPSchemaToEinoParams_Empty(t *testing.T) {
	params := mcpSchemaToEinoParams(mcp.ToolInputSchema{})
	if len(params) != 0 {
		t.Fatalf("expected 0 params for empty schema, got %d", len(params))
	}
}

func TestCollectBoundMCPToolsFiltersByCapabilityKey(t *testing.T) {
	ctx := context.Background()
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	cfg := config.Config{}
	cfg.MCP.DefaultTimeoutSeconds = 2
	cfg.MCP.MaxTimeoutSeconds = 120
	svc := NewMCPService(repo, cfg)

	alphaHTTP := newTestMCPHTTPServer(t, "alpha-upstream", []string{"search", "summarize"})
	defer alphaHTTP.Close()
	betaHTTP := newTestMCPHTTPServer(t, "beta-upstream", []string{"lookup"})
	defer betaHTTP.Close()

	alpha := &model.MCPServer{
		Name:           "alpha",
		Transport:      "http",
		CommandOrURL:   alphaHTTP.URL,
		TimeoutSeconds: 2,
		IsEnabled:      1,
	}
	if err := repo.CreateServer(ctx, alpha); err != nil {
		t.Fatalf("CreateServer alpha failed: %v", err)
	}
	beta := &model.MCPServer{
		Name:           "beta",
		Transport:      "http",
		CommandOrURL:   betaHTTP.URL,
		TimeoutSeconds: 2,
		IsEnabled:      1,
	}
	if err := repo.CreateServer(ctx, beta); err != nil {
		t.Fatalf("CreateServer beta failed: %v", err)
	}

	allowed := map[string]bool{
		MCPCapabilityKey(alpha.ID, "search"): true,
	}

	infos, err := svc.CollectBoundMCPToolInfos(ctx, allowed)
	if err != nil {
		t.Fatalf("CollectBoundMCPToolInfos failed: %v", err)
	}
	assertToolNames(t, toolInfoNames(infos), []string{"mcp_alpha_search"})

	callableTools, err := svc.CollectBoundMCPCallableTools(ctx, allowed)
	if err != nil {
		t.Fatalf("CollectBoundMCPCallableTools failed: %v", err)
	}
	callableNames := make([]string, 0, len(callableTools))
	for _, callableTool := range callableTools {
		info, err := callableTool.Info(ctx)
		if err != nil {
			t.Fatalf("callable tool Info failed: %v", err)
		}
		callableNames = append(callableNames, info.Name)
	}
	assertToolNames(t, callableNames, []string{"mcp_alpha_search"})
}

func TestCallMCPToolPolicyAllowAndAudit(t *testing.T) {
	ctx := context.Background()
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	svc := NewMCPService(repo, testMCPConfig())
	upstream := newTestMCPHTTPServer(t, "policy-upstream", []string{"search"})
	defer upstream.Close()
	server := createEnabledHTTPMCPServer(t, ctx, repo, upstream.URL)

	resp, err := svc.CallMCPTool(ctx, &pb.CallMCPToolRequest{
		ServerId:    server.ID,
		ToolName:    "search",
		ArgsJson:    `{"query":"golang","api_key":"secret"}`,
		CallerRole:  "hr_agent",
		CallerScope: "agent_runtime",
	})
	if err != nil {
		t.Fatalf("CallMCPTool failed: %v", err)
	}
	if resp.Code != 0 || resp.PolicyDecision != "allow" {
		t.Fatalf("expected allow response, got code=%d decision=%q err=%q", resp.Code, resp.PolicyDecision, resp.ErrorMsg)
	}
	logs, total, err := repo.ListToolLogsByServer(ctx, server.ID, 1, 10)
	if err != nil {
		t.Fatalf("ListToolLogsByServer failed: %v", err)
	}
	if total != 1 || logs[0].PolicyDecision != "allow" {
		t.Fatalf("expected allow audit log, total=%d log=%#v", total, logs[0])
	}
	if logs[0].ArgsJSON == nil || strings.Contains(*logs[0].ArgsJSON, "secret") {
		t.Fatalf("expected sensitive args to be redacted, got %#v", logs[0].ArgsJSON)
	}
}

func TestCallMCPToolPolicyDenyConfirmationPermissionScopeRateLimit(t *testing.T) {
	tests := []struct {
		name     string
		policy   model.MCPToolPolicy
		req      pb.CallMCPToolRequest
		seedLogs int
		want     string
	}{
		{
			name: "deny effect",
			policy: model.MCPToolPolicy{
				Effect:    "deny",
				RiskLevel: "critical",
			},
			req:  pb.CallMCPToolRequest{CallerRole: "hr_agent", CallerScope: "agent_runtime"},
			want: "deny",
		},
		{
			name: "confirmation required",
			policy: model.MCPToolPolicy{
				Effect:              "allow",
				RequireConfirmation: 1,
			},
			req:  pb.CallMCPToolRequest{CallerRole: "hr_agent", CallerScope: "agent_runtime"},
			want: "confirmation_required",
		},
		{
			name: "role denied",
			policy: model.MCPToolPolicy{
				Effect:           "allow",
				AllowedRolesJSON: strPtr(`["hr_admin"]`),
			},
			req:  pb.CallMCPToolRequest{CallerRole: "hr_agent", CallerScope: "agent_runtime"},
			want: "deny",
		},
		{
			name: "scope denied",
			policy: model.MCPToolPolicy{
				Effect:            "allow",
				AllowedScopesJSON: strPtr(`["admin_console"]`),
			},
			req:  pb.CallMCPToolRequest{CallerRole: "hr_agent", CallerScope: "agent_runtime"},
			want: "deny",
		},
		{
			name: "rate limited",
			policy: model.MCPToolPolicy{
				Effect:                 "allow",
				RateLimitWindowSeconds: 60,
				RateLimitMaxCalls:      1,
			},
			req:      pb.CallMCPToolRequest{CallerRole: "hr_agent", CallerScope: "agent_runtime"},
			seedLogs: 1,
			want:     "rate_limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			db := setupServiceTestDB(t)
			repo := repository.NewMCPRepo(db)
			svc := NewMCPService(repo, testMCPConfig())
			upstream := newTestMCPHTTPServer(t, "policy-upstream", []string{"search"})
			defer upstream.Close()
			server := createEnabledHTTPMCPServer(t, ctx, repo, upstream.URL)
			tt.policy.ServerID = server.ID
			tt.policy.ToolName = "search"
			if tt.policy.Effect == "" {
				tt.policy.Effect = "allow"
			}
			tt.policy.IsEnabled = 1
			if err := repo.CreateToolPolicy(ctx, &tt.policy); err != nil {
				t.Fatalf("CreateToolPolicy failed: %v", err)
			}
			for i := 0; i < tt.seedLogs; i++ {
				if err := repo.CreateToolLog(ctx, &model.MCPToolLog{ServerID: server.ID, ToolName: "search", PolicyDecision: "allow"}); err != nil {
					t.Fatalf("CreateToolLog failed: %v", err)
				}
			}
			tt.req.ServerId = server.ID
			tt.req.ToolName = "search"
			tt.req.ArgsJson = `{"query":"golang"}`
			resp, err := svc.CallMCPTool(ctx, &tt.req)
			if err != nil {
				t.Fatalf("CallMCPTool failed: %v", err)
			}
			if resp.Code != 1 || resp.PolicyDecision != tt.want {
				t.Fatalf("expected decision %q, got code=%d decision=%q reason=%q", tt.want, resp.Code, resp.PolicyDecision, resp.PolicyReason)
			}
			logs, _, err := repo.ListToolLogsByServer(ctx, server.ID, 1, 10)
			if err != nil {
				t.Fatalf("ListToolLogsByServer failed: %v", err)
			}
			if logs[0].PolicyDecision != tt.want {
				t.Fatalf("expected latest log decision %q, got %q", tt.want, logs[0].PolicyDecision)
			}
		})
	}
}

func TestCallMCPToolPolicyConfirmationApprovedAllows(t *testing.T) {
	ctx := context.Background()
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	svc := NewMCPService(repo, testMCPConfig())
	upstream := newTestMCPHTTPServer(t, "policy-upstream", []string{"search"})
	defer upstream.Close()
	server := createEnabledHTTPMCPServer(t, ctx, repo, upstream.URL)
	if err := repo.CreateToolPolicy(ctx, &model.MCPToolPolicy{
		ServerID:            server.ID,
		ToolName:            "search",
		Effect:              "allow",
		RequireConfirmation: 1,
		IsEnabled:           1,
	}); err != nil {
		t.Fatalf("CreateToolPolicy failed: %v", err)
	}
	resp, err := svc.CallMCPTool(ctx, &pb.CallMCPToolRequest{
		ServerId:             server.ID,
		ToolName:             "search",
		ArgsJson:             `{"query":"golang"}`,
		CallerRole:           "hr_agent",
		CallerScope:          "agent_runtime",
		ConfirmationApproved: true,
	})
	if err != nil {
		t.Fatalf("CallMCPTool failed: %v", err)
	}
	if resp.Code != 0 || resp.PolicyDecision != "allow" {
		t.Fatalf("expected approved confirmation to allow, got code=%d decision=%q", resp.Code, resp.PolicyDecision)
	}
}

func TestCallMCPToolPolicyDisabledAllowsWithAuditReason(t *testing.T) {
	ctx := context.Background()
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	svc := NewMCPService(repo, testMCPConfig()).WithRuntimePolicy(AgentRuntimePolicy{
		StructuredResumeParse: true,
		CandidateMatch:        true,
		SemanticRetrieval:     true,
		MCPPolicy:             false,
		Planner:               true,
		SkillGovernance:       true,
		Fallbacks:             true,
	})
	upstream := newTestMCPHTTPServer(t, "policy-upstream", []string{"search"})
	defer upstream.Close()
	server := createEnabledHTTPMCPServer(t, ctx, repo, upstream.URL)
	if err := repo.CreateToolPolicy(ctx, &model.MCPToolPolicy{
		ServerID:  server.ID,
		ToolName:  "search",
		Effect:    "deny",
		IsEnabled: 1,
	}); err != nil {
		t.Fatalf("CreateToolPolicy failed: %v", err)
	}

	resp, err := svc.CallMCPTool(ctx, &pb.CallMCPToolRequest{
		ServerId:    server.ID,
		ToolName:    "search",
		ArgsJson:    `{"query":"golang"}`,
		CallerRole:  "hr_agent",
		CallerScope: "agent_runtime",
	})
	if err != nil {
		t.Fatalf("CallMCPTool failed: %v", err)
	}
	if resp.Code != 0 || resp.PolicyDecision != "allow" || resp.PolicyReason != "policy_disabled" {
		t.Fatalf("expected disabled policy to allow with audit reason, got code=%d decision=%q reason=%q", resp.Code, resp.PolicyDecision, resp.PolicyReason)
	}
	logs, _, err := repo.ListToolLogsByServer(ctx, server.ID, 1, 10)
	if err != nil {
		t.Fatalf("ListToolLogsByServer failed: %v", err)
	}
	if len(logs) == 0 || logs[0].PolicyDecision != "allow" || logs[0].PolicyReason == nil || *logs[0].PolicyReason != "policy_disabled" {
		t.Fatalf("expected policy_disabled audit log, got %+v", logs)
	}
}

func TestMCPADKToolReturnsRedactedPolicyTracePayload(t *testing.T) {
	ctx := context.Background()
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	svc := NewMCPService(repo, testMCPConfig())
	upstream := newTestMCPHTTPServerWithResult(t, "policy-upstream", "search", `{"candidate_name":"Alice","token":"tok_secret","score":92}`)
	defer upstream.Close()
	server := createEnabledHTTPMCPServer(t, ctx, repo, upstream.URL)
	if err := repo.CreateToolPolicy(ctx, &model.MCPToolPolicy{
		ServerID:         server.ID,
		ToolName:         "search",
		Effect:           "allow",
		RedactFieldsJSON: strPtr(`["candidate_name"]`),
		IsEnabled:        1,
	}); err != nil {
		t.Fatalf("CreateToolPolicy failed: %v", err)
	}

	tools, err := svc.CollectBoundMCPCallableTools(ctx, map[string]bool{MCPCapabilityKey(server.ID, "search"): true})
	if err != nil {
		t.Fatalf("CollectBoundMCPCallableTools failed: %v", err)
	}
	result, err := tools[0].(tool.InvokableTool).InvokableRun(ctx, `{"query":"golang","candidate_name":"Alice","api_key":"sk_secret"}`)
	if err != nil {
		t.Fatalf("InvokableRun failed: %v", err)
	}
	if strings.Contains(result, "Alice") || strings.Contains(result, "sk_secret") || strings.Contains(result, "tok_secret") {
		t.Fatalf("expected ADK trace payload to be redacted, got %s", result)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result), &payload); err != nil {
		t.Fatalf("unmarshal payload failed: %v", err)
	}
	if payload["policy_decision"] != "allow" || payload["policy_reason"] == "" {
		t.Fatalf("expected structured allow policy fields, got %#v", payload)
	}
	argsJSON, ok := payload["arguments_json"].(string)
	if !ok || !strings.Contains(argsJSON, `"candidate_name":"***"`) || !strings.Contains(argsJSON, `"api_key":"***"`) {
		t.Fatalf("expected redacted arguments_json, got %#v", payload["arguments_json"])
	}
	resultContent, ok := payload["result_content"].(string)
	if !ok || !strings.Contains(resultContent, `"candidate_name":"***"`) || !strings.Contains(resultContent, `"token":"***"`) {
		t.Fatalf("expected redacted result_content, got %#v", payload["result_content"])
	}
}

func TestMCPADKToolRejectedPolicyReturnsStructuredDecision(t *testing.T) {
	ctx := context.Background()
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	svc := NewMCPService(repo, testMCPConfig())
	upstream := newTestMCPHTTPServer(t, "policy-upstream", []string{"search"})
	defer upstream.Close()
	server := createEnabledHTTPMCPServer(t, ctx, repo, upstream.URL)
	if err := repo.CreateToolPolicy(ctx, &model.MCPToolPolicy{
		ServerID:            server.ID,
		ToolName:            "search",
		Effect:              "allow",
		RequireConfirmation: 1,
		RedactFieldsJSON:    strPtr(`["candidate_name"]`),
		IsEnabled:           1,
	}); err != nil {
		t.Fatalf("CreateToolPolicy failed: %v", err)
	}

	tools, err := svc.CollectBoundMCPCallableTools(ctx, map[string]bool{MCPCapabilityKey(server.ID, "search"): true})
	if err != nil {
		t.Fatalf("CollectBoundMCPCallableTools failed: %v", err)
	}
	result, err := tools[0].(tool.InvokableTool).InvokableRun(ctx, `{"query":"golang","candidate_name":"Alice","api_key":"sk_secret"}`)
	if err != nil {
		t.Fatalf("InvokableRun should return structured policy content without transport error, got %v", err)
	}
	if strings.Contains(result, "Alice") || strings.Contains(result, "sk_secret") {
		t.Fatalf("expected rejected ADK payload to be redacted, got %s", result)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result), &payload); err != nil {
		t.Fatalf("unmarshal payload failed: %v", err)
	}
	if payload["policy_decision"] != "confirmation_required" || payload["policy_reason"] == "" {
		t.Fatalf("expected structured rejected policy fields, got %#v", payload)
	}
}

func TestCallMCPToolPolicyParameterValidationAndRedaction(t *testing.T) {
	ctx := context.Background()
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	svc := NewMCPService(repo, testMCPConfig())
	upstream := newTestMCPHTTPServer(t, "policy-upstream", []string{"search"})
	defer upstream.Close()
	server := createEnabledHTTPMCPServer(t, ctx, repo, upstream.URL)
	if err := repo.CreateToolPolicy(ctx, &model.MCPToolPolicy{
		ServerID:         server.ID,
		ToolName:         "search",
		Effect:           "allow",
		RequiredArgsJSON: strPtr(`["query"]`),
		DeniedArgsJSON:   strPtr(`["unsafe"]`),
		ArgRulesJSON:     strPtr(`{"limit":{"min":1,"max":10},"mode":{"enum":["basic","advanced"]},"query":{"regex":"^[a-z]+$"}}`),
		RedactFieldsJSON: strPtr(`["candidate_name"]`),
		IsEnabled:        1,
	}); err != nil {
		t.Fatalf("CreateToolPolicy failed: %v", err)
	}

	denied, err := svc.CallMCPTool(ctx, &pb.CallMCPToolRequest{
		ServerId:    server.ID,
		ToolName:    "search",
		ArgsJson:    `{"query":"GoLang","limit":20,"mode":"basic","unsafe":true,"candidate_name":"Alice"}`,
		CallerRole:  "hr_agent",
		CallerScope: "agent_runtime",
	})
	if err != nil {
		t.Fatalf("CallMCPTool denied path failed: %v", err)
	}
	if denied.Code != 1 || denied.PolicyDecision != "deny" {
		t.Fatalf("expected deny for invalid args, got code=%d decision=%q reason=%q", denied.Code, denied.PolicyDecision, denied.PolicyReason)
	}
	logs, _, err := repo.ListToolLogsByServer(ctx, server.ID, 1, 10)
	if err != nil {
		t.Fatalf("ListToolLogsByServer failed: %v", err)
	}
	if logs[0].ArgsJSON == nil || strings.Contains(*logs[0].ArgsJSON, "Alice") {
		t.Fatalf("expected configured field redaction in args log, got %#v", logs[0].ArgsJSON)
	}

	allowed, err := svc.CallMCPTool(ctx, &pb.CallMCPToolRequest{
		ServerId:    server.ID,
		ToolName:    "search",
		ArgsJson:    `{"query":"golang","limit":5,"mode":"basic","candidate_name":"Bob"}`,
		CallerRole:  "hr_agent",
		CallerScope: "agent_runtime",
	})
	if err != nil {
		t.Fatalf("CallMCPTool allowed path failed: %v", err)
	}
	if allowed.Code != 0 || allowed.PolicyDecision != "allow" {
		t.Fatalf("expected valid args to allow, got code=%d decision=%q reason=%q", allowed.Code, allowed.PolicyDecision, allowed.PolicyReason)
	}
}

// ── isSensitiveField tests ────────────────────────────────────────────

func TestIsSensitiveField(t *testing.T) {
	sensitive := []string{
		"phone", "mobile", "email", "api_key", "API_KEY",
		"password", "secret", "token", "access_token",
		"idcard", "id_card", "private_key", "credential",
	}
	notSensitive := []string{
		"name", "description", "id", "type", "value",
		"title", "content", "text", "url",
	}
	for _, s := range sensitive {
		if !isSensitiveField(s) {
			t.Errorf("expected '%s' to be sensitive", s)
		}
	}
	for _, s := range notSensitive {
		if isSensitiveField(s) {
			t.Errorf("expected '%s' to NOT be sensitive", s)
		}
	}
}

// ── CRUD validation tests ─────────────────────────────────────────────

func TestCreateMCPServer_Validation(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	cfg := config.Config{}
	cfg.MCP.DefaultTimeoutSeconds = 30
	cfg.MCP.AllowStdio = true
	cfg.MCP.AllowedStdioCommands = []string{"/usr/bin/test", "npx", "node"}
	cfg.MCP.MaxTimeoutSeconds = 120
	svc := NewMCPService(repo, cfg)
	ctx := context.Background()

	t.Run("empty name", func(t *testing.T) {
		_, err := svc.CreateMCPServer(ctx, &pb.CreateMCPServerRequest{Name: ""})
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("empty transport", func(t *testing.T) {
		_, err := svc.CreateMCPServer(ctx, &pb.CreateMCPServerRequest{
			Name: "test", Transport: "",
		})
		if err == nil {
			t.Fatal("expected error for empty transport")
		}
	})

	t.Run("invalid transport", func(t *testing.T) {
		_, err := svc.CreateMCPServer(ctx, &pb.CreateMCPServerRequest{
			Name: "test", Transport: "invalid",
		})
		if err == nil {
			t.Fatal("expected error for invalid transport")
		}
	})

	t.Run("empty command", func(t *testing.T) {
		_, err := svc.CreateMCPServer(ctx, &pb.CreateMCPServerRequest{
			Name: "test", Transport: "stdio", CommandOrUrl: "",
		})
		if err == nil {
			t.Fatal("expected error for empty command_or_url")
		}
	})

	t.Run("valid request", func(t *testing.T) {
		resp, err := svc.CreateMCPServer(ctx, &pb.CreateMCPServerRequest{
			Name:         "valid-server",
			Transport:    "stdio",
			CommandOrUrl: "/usr/bin/test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Server.Name != "valid-server" {
			t.Fatalf("expected name 'valid-server', got %q", resp.Server.Name)
		}
		// Default: disabled
		if resp.Server.IsEnabled {
			t.Fatal("expected new server to be disabled by default")
		}
	})
}

func TestListMCPServers_Empty(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	cfg := config.Config{}
	cfg.MCP.MaxTimeoutSeconds = 120
	svc := NewMCPService(repo, cfg)
	ctx := context.Background()

	resp, err := svc.ListMCPServers(ctx, &pb.ListMCPServersRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListMCPServers failed: %v", err)
	}
	if resp.Total != 0 {
		t.Fatalf("expected total 0, got %d", resp.Total)
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.List))
	}
}

func TestDeleteMCPServer_InvalidID(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	cfg := config.Config{}
	cfg.MCP.MaxTimeoutSeconds = 120
	svc := NewMCPService(repo, cfg)
	ctx := context.Background()

	t.Run("id zero", func(t *testing.T) {
		_, err := svc.DeleteMCPServer(ctx, &pb.DeleteMCPServerRequest{Id: 0})
		if err == nil {
			t.Fatal("expected error for id=0")
		}
	})

	t.Run("id negative", func(t *testing.T) {
		_, err := svc.DeleteMCPServer(ctx, &pb.DeleteMCPServerRequest{Id: -1})
		if err == nil {
			t.Fatal("expected error for id=-1")
		}
	})
}

func TestUpdateMCPServer_NotFound(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	cfg := config.Config{}
	cfg.MCP.MaxTimeoutSeconds = 120
	svc := NewMCPService(repo, cfg)
	ctx := context.Background()

	_, err := svc.UpdateMCPServer(ctx, &pb.UpdateMCPServerRequest{
		Id:   999,
		Name: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

func TestCreateAndGetMCPServer_Integration(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	cfg := config.Config{}
	cfg.MCP.MaxTimeoutSeconds = 120
	cfg.MCP.BlockPrivateNetwork = ptrBool(false)
	svc := NewMCPService(repo, cfg)
	ctx := context.Background()

	// Create
	createResp, err := svc.CreateMCPServer(ctx, &pb.CreateMCPServerRequest{
		Name:         "integration-test",
		Transport:    "sse",
		CommandOrUrl: "https://api.example.com/mcp",
		EnvVars:      `{"KEY":"val"}`,
	})
	if err != nil {
		t.Fatalf("CreateMCPServer failed: %v", err)
	}
	if createResp.Server.Id <= 0 {
		t.Fatal("expected positive server ID")
	}

	// Verify env_vars is desensitized in response
	if strings.Contains(createResp.Server.EnvVars, "val") {
		t.Fatalf("expected env_vars to be desensitized in create response, got %q", createResp.Server.EnvVars)
	}

	// Update name
	updateResp, err := svc.UpdateMCPServer(ctx, &pb.UpdateMCPServerRequest{
		Id:   createResp.Server.Id,
		Name: "updated-name",
	})
	if err != nil {
		t.Fatalf("UpdateMCPServer failed: %v", err)
	}
	if updateResp.Server.Name != "updated-name" {
		t.Fatalf("expected updated name 'updated-name', got %q", updateResp.Server.Name)
	}

	// List and verify
	listResp, err := svc.ListMCPServers(ctx, &pb.ListMCPServersRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListMCPServers failed: %v", err)
	}
	if listResp.Total != 1 {
		t.Fatalf("expected total 1, got %d", listResp.Total)
	}

	// Delete
	_, err = svc.DeleteMCPServer(ctx, &pb.DeleteMCPServerRequest{Id: createResp.Server.Id})
	if err != nil {
		t.Fatalf("DeleteMCPServer failed: %v", err)
	}

	// Verify deleted
	listResp2, _ := svc.ListMCPServers(ctx, &pb.ListMCPServersRequest{Page: 1, PageSize: 20})
	if listResp2.Total != 0 {
		t.Fatalf("expected total 0 after delete, got %d", listResp2.Total)
	}
}

// ── Test Helper ───────────────────────────────────────────────────────

func setupServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	if err := db.AutoMigrate(&model.MCPServer{}, &model.MCPToolLog{}, &model.MCPToolPolicy{}); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}

func testMCPConfig() config.Config {
	cfg := config.Config{}
	cfg.MCP.DefaultTimeoutSeconds = 2
	cfg.MCP.MaxTimeoutSeconds = 120
	cfg.MCP.BlockPrivateNetwork = ptrBool(false)
	return cfg
}

func createEnabledHTTPMCPServer(t *testing.T, ctx context.Context, repo *repository.MCPRepo, url string) *model.MCPServer {
	t.Helper()
	server := &model.MCPServer{
		Name:           "policy-server",
		Transport:      "http",
		CommandOrURL:   url,
		TimeoutSeconds: 2,
		IsEnabled:      1,
	}
	if err := repo.CreateServer(ctx, server); err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}
	return server
}

func strPtr(s string) *string {
	return &s
}

func newTestMCPHTTPServer(t *testing.T, name string, toolNames []string) *httptest.Server {
	t.Helper()
	srv := mcpserver.NewMCPServer(name, "1.0.0")
	for _, toolName := range toolNames {
		name := toolName
		srv.AddTool(
			mcp.NewTool(name,
				mcp.WithDescription("test tool "+name),
				mcp.WithString("query"),
				mcp.WithString("candidate_name"),
				mcp.WithString("api_key"),
			),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return mcp.NewToolResultText(name + " ok"), nil
			},
		)
	}
	return mcpserver.NewTestStreamableHTTPServer(srv)
}

func newTestMCPHTTPServerWithResult(t *testing.T, serverName, toolName, resultText string) *httptest.Server {
	t.Helper()
	srv := mcpserver.NewMCPServer(serverName, "1.0.0")
	srv.AddTool(
		mcp.NewTool(toolName,
			mcp.WithDescription("test tool "+toolName),
			mcp.WithString("query"),
			mcp.WithString("candidate_name"),
			mcp.WithString("api_key"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(resultText), nil
		},
	)
	return mcpserver.NewTestStreamableHTTPServer(srv)
}

func toolInfoNames(infos []*schema.ToolInfo) []string {
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Name)
	}
	return names
}

func assertToolNames(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected tool names %v, got %v", want, got)
	}
	seen := make(map[string]bool, len(got))
	for _, name := range got {
		seen[name] = true
	}
	for _, name := range want {
		if !seen[name] {
			t.Fatalf("expected tool names %v, got %v", want, got)
		}
	}
	for _, forbidden := range []string{"mcp_alpha_summarize", "mcp_beta_lookup"} {
		if seen[forbidden] {
			t.Fatalf("unexpected unbound tool %q in %v", forbidden, got)
		}
	}
}

// ── MCP Security Validation Tests ───────────────────────────────────

func TestValidateStdioConfig_Disabled(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{AllowStdio: false}}
	err := validateStdioConfig(cfg, "npx", "")
	if err == nil {
		t.Fatal("expected error when stdio is disabled")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected 'disabled' in error, got %v", err)
	}
}

func TestValidateStdioConfig_ShellCommand(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{AllowStdio: true, AllowedStdioCommands: []string{"npx"}}}
	shellCmds := []string{"sh", "bash", "zsh", "cmd", "powershell", "sh -c 'echo hi'", "bash -c 'echo hi'"}
	for _, cmd := range shellCmds {
		err := validateStdioConfig(cfg, cmd, "")
		if err == nil {
			t.Errorf("expected error for shell command %q", cmd)
		}
	}
}

func TestValidateStdioConfig_NotAllowlisted(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{AllowStdio: true, AllowedStdioCommands: []string{"npx"}}}
	err := validateStdioConfig(cfg, "uvx", "")
	if err == nil {
		t.Fatal("expected error for non-allowlisted command")
	}
}

func TestValidateStdioConfig_Allowlisted(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{AllowStdio: true, AllowedStdioCommands: []string{"npx", "node"}}}
	err := validateStdioConfig(cfg, "npx", `["mcp-server"]`)
	if err != nil {
		t.Fatalf("unexpected error for allowlisted command: %v", err)
	}
}

func TestValidateStdioConfig_InvalidArgs(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{AllowStdio: true, AllowedStdioCommands: []string{"npx"}}}
	err := validateStdioConfig(cfg, "npx", "not-json")
	if err == nil {
		t.Fatal("expected error for invalid args JSON")
	}
}

func TestValidateStdioConfig_ShellControlOperatorInArgs(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{AllowStdio: true, AllowedStdioCommands: []string{"npx"}}}
	err := validateStdioConfig(cfg, "npx", `["arg1;rm -rf /"]`)
	if err == nil {
		t.Fatal("expected error for shell control operator in args")
	}
}

func TestValidateURLConfig_PrivateNetwork(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{BlockPrivateNetwork: ptrBool(true)}}
	privateURLs := []string{
		"http://localhost:8080/mcp",
		"http://127.0.0.1:3000",
		"http://10.0.0.1:5000",
		"http://192.168.1.1:5000",
		"http://172.16.0.1:5000",
	}
	for _, u := range privateURLs {
		err := validateURLConfig(cfg, u)
		if err == nil {
			t.Errorf("expected error for private URL %q", u)
		}
	}
}

func TestValidateURLConfig_AllowedPublic(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{BlockPrivateNetwork: ptrBool(false)}}
	err := validateURLConfig(cfg, "https://api.example.com/mcp")
	if err != nil {
		t.Fatalf("unexpected error for public URL: %v", err)
	}
}

func TestValidateURLConfig_InvalidScheme(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{BlockPrivateNetwork: ptrBool(true)}}
	err := validateURLConfig(cfg, "ftp://example.com/mcp")
	if err == nil {
		t.Fatal("expected error for invalid scheme")
	}
}

func TestValidateURLConfig_MetadataBlockedEvenInAllowlist(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{AllowedURLHosts: []string{"169.254.169.254", "100.100.100.200"}}}

	for _, url := range []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://100.100.100.200/latest/meta-data/",
	} {
		err := validateURLConfig(cfg, url)
		if err == nil {
			t.Fatalf("expected metadata address %q to be blocked even if in allowlist", url)
		}
		if !strings.Contains(err.Error(), "blocked") {
			t.Fatalf("expected 'blocked' in error for metadata address %q, got: %v", url, err)
		}
	}
}

func TestValidateMCPConfig_CreateServer_Security(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repository.NewMCPRepo(db)
	cfg := config.Config{}
	cfg.MCP.DefaultTimeoutSeconds = 30
	cfg.MCP.AllowStdio = false
	cfg.MCP.MaxTimeoutSeconds = 120
	svc := NewMCPService(repo, cfg)
	ctx := context.Background()

	t.Run("stdio disabled", func(t *testing.T) {
		_, err := svc.CreateMCPServer(ctx, &pb.CreateMCPServerRequest{
			Name:         "stdio-disabled",
			Transport:    "stdio",
			CommandOrUrl: "npx",
		})
		if err == nil {
			t.Fatal("expected error when stdio is disabled")
		}
	})

	// Recreate with stdio allowed
	cfg.MCP.AllowStdio = true
	cfg.MCP.AllowedStdioCommands = []string{"npx"}
	svc2 := NewMCPService(repo, cfg)

	t.Run("stdio enabled with allowed command", func(t *testing.T) {
		resp, err := svc2.CreateMCPServer(ctx, &pb.CreateMCPServerRequest{
			Name:         "stdio-enabled",
			Transport:    "stdio",
			CommandOrUrl: "npx",
			Args:         `["-y","@modelcontextprotocol/server-filesystem"]`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Server.Name != "stdio-enabled" {
			t.Fatalf("expected name 'stdio-enabled', got %q", resp.Server.Name)
		}
	})

	t.Run("remote private URL blocked", func(t *testing.T) {
		cfg.MCP.BlockPrivateNetwork = ptrBool(true)
		svc3 := NewMCPService(repo, cfg)
		_, err := svc3.CreateMCPServer(ctx, &pb.CreateMCPServerRequest{
			Name:         "private-url",
			Transport:    "sse",
			CommandOrUrl: "http://localhost:8080/mcp",
		})
		if err == nil {
			t.Fatal("expected error for private URL")
		}
	})
}

func TestValidateURLConfig_AllowedURLHosts(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{AllowedURLHosts: []string{"api.example.com", "mcp.trusted.io"}}}

	t.Run("allowlisted host passes", func(t *testing.T) {
		err := validateURLConfig(cfg, "https://api.example.com/mcp")
		if err != nil {
			t.Fatalf("expected allowlisted URL to pass, got: %v", err)
		}
	})

	t.Run("non-allowlisted host rejected", func(t *testing.T) {
		err := validateURLConfig(cfg, "https://evil.com/mcp")
		if err == nil {
			t.Fatal("expected non-allowlisted host to be rejected")
		}
		if !strings.Contains(err.Error(), "not in the allowed hosts list") {
			t.Fatalf("expected 'not in the allowed hosts list' error, got: %v", err)
		}
	})

	t.Run("allowlisted host bypasses private network check", func(t *testing.T) {
		err := validateURLConfig(cfg, "http://localhost:8080/mcp")
		if err == nil {
			t.Fatal("expected localhost to be rejected (not in allowlist)")
		}
	})

	t.Run("empty allowlist falls through to private network check", func(t *testing.T) {
		cfg2 := config.Config{MCP: struct {
			DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
			DefaultMaxRetries     int      `yaml:"default_max_retries"`
			AllowStdio            bool     `yaml:"allow_stdio"`
			AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
			AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
			BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
			MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
		}{BlockPrivateNetwork: ptrBool(true)}}
		err := validateURLConfig(cfg2, "http://localhost:8080/mcp")
		if err == nil {
			t.Fatal("expected localhost blocked by private network check")
		}
	})
}

func TestValidateURLConfig_DNSFailureRejected(t *testing.T) {
	cfg := config.Config{MCP: struct {
		DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
		DefaultMaxRetries     int      `yaml:"default_max_retries"`
		AllowStdio            bool     `yaml:"allow_stdio"`
		AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
		AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
		BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
		MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
	}{BlockPrivateNetwork: ptrBool(true)}}

	t.Run("unresolvable host rejected", func(t *testing.T) {
		err := validateURLConfig(cfg, "http://this-domain-does-not-exist-12345.com/mcp")
		if err == nil {
			t.Fatal("expected unresolvable host to be rejected")
		}
		if !strings.Contains(err.Error(), "cannot be resolved") {
			t.Fatalf("expected 'cannot be resolved' error, got: %v", err)
		}
	})

	t.Run("unresolvable host passes if in allowlist", func(t *testing.T) {
		cfg2 := config.Config{MCP: struct {
			DefaultTimeoutSeconds int      `yaml:"default_timeout_seconds"`
			DefaultMaxRetries     int      `yaml:"default_max_retries"`
			AllowStdio            bool     `yaml:"allow_stdio"`
			AllowedStdioCommands  []string `yaml:"allowed_stdio_commands"`
			AllowedURLHosts       []string `yaml:"allowed_url_hosts"`
			BlockPrivateNetwork   *bool    `yaml:"block_private_network"`
			MaxTimeoutSeconds     int      `yaml:"max_timeout_seconds"`
		}{AllowedURLHosts: []string{"unresolvable.example.com"}}}
		err := validateURLConfig(cfg2, "http://unresolvable.example.com/mcp")
		if err != nil {
			t.Fatalf("expected unresolvable host in allowlist to pass, got: %v", err)
		}
	})
}

func ptrBool(v bool) *bool {
	return &v
}
