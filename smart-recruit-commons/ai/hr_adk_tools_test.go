package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

type fakeRecruitingExecutor struct {
	calls []string
	err   error
}

func (f *fakeRecruitingExecutor) Execute(_ context.Context, ownerID int64, toolName string, args map[string]any) (ToolResult, error) {
	f.calls = append(f.calls, toolName)
	if ownerID <= 0 {
		return ToolResult{}, errors.New("owner missing")
	}
	if f.err != nil {
		return ToolResult{}, f.err
	}
	payload, _ := json.Marshal(map[string]any{"tool": toolName, "args": args, "ok": true})
	return ToolResult{Content: string(payload)}, nil
}

func TestNewRecruitingADKToolsRespectsAllowlistAndExclusion(t *testing.T) {
	exec := &fakeRecruitingExecutor{}
	tools, err := NewRecruitingADKTools(exec, []string{"get_job_list", "query_today_applications", "propose_application_status_update", "not_a_tool"})
	if err != nil {
		t.Fatalf("NewRecruitingADKTools: %v", err)
	}
	names := ADKToolNames(tools)
	if len(names) != 2 {
		t.Fatalf("tool names = %v, want get_job_list and query_today_applications", names)
	}
	for _, name := range names {
		if name == "propose_application_status_update" {
			t.Fatal("propose_application_status_update must not be model-facing")
		}
	}
}

func TestNewRecruitingADKToolsEmptyAllowlistFailClosed(t *testing.T) {
	tools, err := NewRecruitingADKTools(&fakeRecruitingExecutor{}, nil)
	if err != nil {
		t.Fatalf("NewRecruitingADKTools: %v", err)
	}
	if len(tools) != 0 {
		t.Fatalf("tools = %d, want 0 for empty allowlist", len(tools))
	}
}

func TestNewRecruitingADKToolsInvokesExecutorWithOwnerContext(t *testing.T) {
	exec := &fakeRecruitingExecutor{}
	tools, err := NewRecruitingADKTools(exec, []string{"get_job_list"})
	if err != nil || len(tools) != 1 {
		t.Fatalf("tools err=%v len=%d", err, len(tools))
	}
	inv, ok := tools[0].(tool.InvokableTool)
	if !ok {
		t.Fatalf("tool type %T does not implement InvokableTool", tools[0])
	}
	ctx := WithOwnerID(context.Background(), 77)
	ctx = WithAgentRunState(ctx, &AgentRunState{})
	out, err := inv.InvokableRun(ctx, `{}`)
	if err != nil {
		t.Fatalf("InvokableRun: %v", err)
	}
	if !strings.Contains(out, "get_job_list") {
		t.Fatalf("output = %s, want get_job_list result", out)
	}
	if len(exec.calls) != 1 || exec.calls[0] != "get_job_list" {
		t.Fatalf("calls = %v", exec.calls)
	}
}

func TestNewRecruitingADKToolsRequiresOwner(t *testing.T) {
	exec := &fakeRecruitingExecutor{}
	tools, err := NewRecruitingADKTools(exec, []string{"get_job_list"})
	if err != nil || len(tools) != 1 {
		t.Fatalf("tools err=%v len=%d", err, len(tools))
	}
	inv := tools[0].(tool.InvokableTool)
	// Error handler converts error to JSON content with nil err.
	out, err := inv.InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("expected error handler to swallow error, got %v", err)
	}
	if !strings.Contains(out, "owner_id") && !strings.Contains(out, "error") {
		t.Fatalf("output = %s, want owner error payload", out)
	}
}

func TestFilterADKToolsByName(t *testing.T) {
	exec := &fakeRecruitingExecutor{}
	tools, err := NewRecruitingADKTools(exec, []string{"get_job_list", "query_total_applications"})
	if err != nil {
		t.Fatal(err)
	}
	filtered := FilterADKToolsByName(tools, []string{"get_job_list"})
	names := ADKToolNames(filtered)
	if len(names) != 1 || names[0] != "get_job_list" {
		t.Fatalf("filtered = %v", names)
	}
}
