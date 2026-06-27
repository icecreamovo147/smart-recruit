package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/mcp"
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
	if got != input {
		t.Fatalf("expected original value for invalid JSON, got %q", got)
	}
}

// ── serverToInfo tests ────────────────────────────────────────────────

func TestServerToInfo_DesensitizesEnvVars(t *testing.T) {
	envVars := `{"API_KEY":"sk-xxx","SECRET":"s3cr3t"}`
	srv := &model.MCPServer{
		ID:            1,
		Name:          "test",
		Transport:     "stdio",
		CommandOrURL:  "/usr/bin/test",
		EnvVars:       &envVars,
		TimeoutSeconds: 30,
		IsEnabled:     1,
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
		ID:            2,
		Name:          "no-env-server",
		Transport:     "sse",
		CommandOrURL:  "http://localhost:8080",
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
	var cfg config.Config
	cfg.MCP.DefaultTimeoutSeconds = 30
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
	var cfg config.Config
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
	var cfg config.Config
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
	var cfg config.Config
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
	var cfg config.Config
	svc := NewMCPService(repo, cfg)
	ctx := context.Background()

	// Create
	createResp, err := svc.CreateMCPServer(ctx, &pb.CreateMCPServerRequest{
		Name:         "integration-test",
		Transport:    "sse",
		CommandOrUrl: "http://localhost:8080/mcp",
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
	if err := db.AutoMigrate(&model.MCPServer{}, &model.MCPToolLog{}); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}
