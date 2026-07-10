package repository

import (
	"context"
	"testing"
	"time"

	"logic-grpc-service/model"
)

func TestMCPServerCreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMCPRepo(db)
	ctx := context.Background()

	envVars := `{"API_KEY":"sk-test"}`
	server := &model.MCPServer{
		Name:           "test-server",
		Transport:      "stdio",
		CommandOrURL:   "/usr/bin/test",
		Args:           strPtr(`["--flag"]`),
		EnvVars:        &envVars,
		TimeoutSeconds: 30,
		IsEnabled:      0,
	}
	if err := repo.CreateServer(ctx, server); err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}
	if server.ID <= 0 {
		t.Fatal("expected non-zero server ID after CreateServer")
	}

	// Verify via GetServerByID
	loaded, err := repo.GetServerByID(ctx, server.ID)
	if err != nil {
		t.Fatalf("GetServerByID failed: %v", err)
	}
	if loaded.Name != "test-server" {
		t.Fatalf("expected name 'test-server', got '%s'", loaded.Name)
	}
	if loaded.Transport != "stdio" {
		t.Fatalf("expected transport 'stdio', got '%s'", loaded.Transport)
	}
	if loaded.CommandOrURL != "/usr/bin/test" {
		t.Fatalf("expected command '/usr/bin/test', got '%s'", loaded.CommandOrURL)
	}
	if loaded.IsEnabled != 0 {
		t.Fatalf("expected IsEnabled=0 (default disabled), got %d", loaded.IsEnabled)
	}
}

func TestMCPServerUpdatePartial(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMCPRepo(db)
	ctx := context.Background()

	server := &model.MCPServer{
		Name:           "initial",
		Transport:      "stdio",
		CommandOrURL:   "/usr/bin/initial",
		TimeoutSeconds: 30,
		IsEnabled:      0,
	}
	if err := repo.CreateServer(ctx, server); err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}

	// Update name and enable
	updates := map[string]any{
		"name":       "updated",
		"is_enabled": 1,
	}
	if err := repo.UpdateServerPartial(ctx, server.ID, updates); err != nil {
		t.Fatalf("UpdateServerPartial failed: %v", err)
	}

	loaded, err := repo.GetServerByID(ctx, server.ID)
	if err != nil {
		t.Fatalf("GetServerByID failed: %v", err)
	}
	if loaded.Name != "updated" {
		t.Fatalf("expected name 'updated', got '%s'", loaded.Name)
	}
	if loaded.IsEnabled != 1 {
		t.Fatalf("expected IsEnabled=1, got %d", loaded.IsEnabled)
	}
}

func TestMCPServerDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMCPRepo(db)
	ctx := context.Background()

	server := &model.MCPServer{
		Name:           "to-delete",
		Transport:      "stdio",
		CommandOrURL:   "/usr/bin/delete",
		TimeoutSeconds: 30,
	}
	if err := repo.CreateServer(ctx, server); err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}

	if err := repo.DeleteServer(ctx, server.ID); err != nil {
		t.Fatalf("DeleteServer failed: %v", err)
	}

	// Verify deletion
	_, err := repo.GetServerByID(ctx, server.ID)
	if err == nil {
		t.Fatal("expected error after deleting server, got nil")
	}
}

func TestMCPServerListPagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMCPRepo(db)
	ctx := context.Background()

	// Create 3 servers
	for i := 0; i < 3; i++ {
		name := "server"
		if i == 0 {
			name = "alpha"
		} else if i == 1 {
			name = "beta"
		} else {
			name = "gamma"
		}
		server := &model.MCPServer{
			Name:           name,
			Transport:      "stdio",
			CommandOrURL:   "/usr/bin/" + name,
			TimeoutSeconds: 30,
		}
		if err := repo.CreateServer(ctx, server); err != nil {
			t.Fatalf("CreateServer failed: %v", err)
		}
	}

	// List with page 1, pageSize 2
	list, total, err := repo.ListServers(ctx, 1, 2)
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total=3, got %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 items on page 1, got %d", len(list))
	}

	// Page 2
	list2, total2, err := repo.ListServers(ctx, 2, 2)
	if err != nil {
		t.Fatalf("ListServers page 2 failed: %v", err)
	}
	if total2 != 3 {
		t.Fatalf("expected total=3 on page 2, got %d", total2)
	}
	if len(list2) != 1 {
		t.Fatalf("expected 1 item on page 2, got %d", len(list2))
	}
}

func TestMCPServerListEnabled(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMCPRepo(db)
	ctx := context.Background()

	// Create one enabled and one disabled server
	enabled := &model.MCPServer{
		Name:           "enabled-server",
		Transport:      "stdio",
		CommandOrURL:   "/usr/bin/enabled",
		TimeoutSeconds: 30,
		IsEnabled:      1,
	}
	if err := repo.CreateServer(ctx, enabled); err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}

	disabled := &model.MCPServer{
		Name:           "disabled-server",
		Transport:      "stdio",
		CommandOrURL:   "/usr/bin/disabled",
		TimeoutSeconds: 30,
		IsEnabled:      0,
	}
	if err := repo.CreateServer(ctx, disabled); err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}

	enabledList, err := repo.ListEnabledServers(ctx)
	if err != nil {
		t.Fatalf("ListEnabledServers failed: %v", err)
	}
	if len(enabledList) != 1 {
		t.Fatalf("expected 1 enabled server, got %d", len(enabledList))
	}
	if enabledList[0].Name != "enabled-server" {
		t.Fatalf("expected 'enabled-server', got '%s'", enabledList[0].Name)
	}
}

func TestMCPToolLogCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMCPRepo(db)
	ctx := context.Background()

	// Create a server first
	server := &model.MCPServer{
		Name:           "log-test-server",
		Transport:      "stdio",
		CommandOrURL:   "/usr/bin/test",
		TimeoutSeconds: 30,
	}
	if err := repo.CreateServer(ctx, server); err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}

	args := `{"key":"value"}`
	result := "tool result"
	hrID := int64(42)
	sessionID := int64(100)
	toolLog := &model.MCPToolLog{
		ServerID:      server.ID,
		ToolName:      "test_tool",
		ArgsJSON:      &args,
		ResultContent: &result,
		DurationMs:    150,
		CalledByHRID:  &hrID,
		SessionID:     &sessionID,
	}
	if err := repo.CreateToolLog(ctx, toolLog); err != nil {
		t.Fatalf("CreateToolLog failed: %v", err)
	}
	if toolLog.ID <= 0 {
		t.Fatal("expected non-zero tool log ID")
	}

	// List tool logs by server
	logs, total, err := repo.ListToolLogsByServer(ctx, server.ID, 1, 10)
	if err != nil {
		t.Fatalf("ListToolLogsByServer failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total=1, got %d", total)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].ToolName != "test_tool" {
		t.Fatalf("expected tool_name 'test_tool', got '%s'", logs[0].ToolName)
	}
}

func TestMCPToolLogListBySession(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMCPRepo(db)
	ctx := context.Background()

	server := &model.MCPServer{
		Name:           "session-log-server",
		Transport:      "stdio",
		CommandOrURL:   "/usr/bin/test",
		TimeoutSeconds: 30,
	}
	if err := repo.CreateServer(ctx, server); err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}

	sessionID := int64(200)
	for i := 0; i < 3; i++ {
		toolName := "tool_0"
		if i == 1 {
			toolName = "tool_1"
		} else if i == 2 {
			toolName = "tool_2"
		}
		args := `{"idx":` + toolName[len(toolName)-1:] + `}`
		result := "result for " + toolName
		log := &model.MCPToolLog{
			ServerID:      server.ID,
			ToolName:      toolName,
			ArgsJSON:      &args,
			ResultContent: &result,
			DurationMs:    100,
			SessionID:     &sessionID,
		}
		if err := repo.CreateToolLog(ctx, log); err != nil {
			t.Fatalf("CreateToolLog failed: %v", err)
		}
	}

	// List by session
	logs, total, err := repo.ListToolLogsBySession(ctx, sessionID, 1, 10)
	if err != nil {
		t.Fatalf("ListToolLogsBySession failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total=3, got %d", total)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}
}

func TestMCPToolPolicyCRUDAndLookup(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMCPRepo(db)
	ctx := context.Background()

	server := &model.MCPServer{Name: "policy-server", Transport: "stdio", CommandOrURL: "/usr/bin/test"}
	if err := repo.CreateServer(ctx, server); err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}

	required := `["query"]`
	policy := &model.MCPToolPolicy{
		ServerID:         server.ID,
		ToolName:         "search",
		Effect:           "allow",
		RiskLevel:        "high",
		RequiredArgsJSON: &required,
		IsEnabled:        1,
	}
	if err := repo.CreateToolPolicy(ctx, policy); err != nil {
		t.Fatalf("CreateToolPolicy failed: %v", err)
	}
	if policy.ID <= 0 {
		t.Fatal("expected policy ID")
	}

	loaded, err := repo.GetEnabledToolPolicy(ctx, server.ID, "search")
	if err != nil {
		t.Fatalf("GetEnabledToolPolicy failed: %v", err)
	}
	if loaded.RiskLevel != "high" {
		t.Fatalf("expected risk high, got %q", loaded.RiskLevel)
	}

	loaded.Effect = "deny"
	if err := repo.UpdateToolPolicy(ctx, loaded); err != nil {
		t.Fatalf("UpdateToolPolicy failed: %v", err)
	}
	updated, err := repo.GetToolPolicyByID(ctx, policy.ID)
	if err != nil {
		t.Fatalf("GetToolPolicyByID failed: %v", err)
	}
	if updated.Effect != "deny" {
		t.Fatalf("expected effect deny, got %q", updated.Effect)
	}

	list, total, err := repo.ListToolPolicies(ctx, server.ID, 1, 10)
	if err != nil {
		t.Fatalf("ListToolPolicies failed: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected one policy, total=%d len=%d", total, len(list))
	}

	if err := repo.DeleteToolPolicy(ctx, policy.ID); err != nil {
		t.Fatalf("DeleteToolPolicy failed: %v", err)
	}
	if _, err := repo.GetToolPolicyByID(ctx, policy.ID); err == nil {
		t.Fatal("expected deleted policy lookup to fail")
	}
}

func TestMCPRepoCountToolLogsSince(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMCPRepo(db)
	ctx := context.Background()

	server := &model.MCPServer{Name: "rate-server", Transport: "stdio", CommandOrURL: "/usr/bin/test"}
	if err := repo.CreateServer(ctx, server); err != nil {
		t.Fatalf("CreateServer failed: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := repo.CreateToolLog(ctx, &model.MCPToolLog{ServerID: server.ID, ToolName: "search", PolicyDecision: "allow"}); err != nil {
			t.Fatalf("CreateToolLog failed: %v", err)
		}
	}
	count, err := repo.CountToolLogsSince(ctx, server.ID, "search", logsSincePast())
	if err != nil {
		t.Fatalf("CountToolLogsSince failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 logs, got %d", count)
	}
}

func logsSincePast() time.Time {
	return time.Now().Add(-time.Hour)
}

// strPtr is a helper to create a *string.
func strPtr(s string) *string {
	return &s
}
