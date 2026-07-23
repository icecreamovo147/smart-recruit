package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// RecruitingToolExecutor is the business execution interface used by ADK
// BaseTool wrappers. Microservice runtimes implement this with gRPC-backed
// executors; the monorepo historically used a DB-backed ToolExecutor.
type RecruitingToolExecutor interface {
	Execute(ctx context.Context, ownerID int64, toolName string, args map[string]any) (ToolResult, error)
}

// tools excluded from the model-facing ADK surface. Status proposals need a
// dedicated confirmation transport and must not auto-execute.
var adkExcludedModelTools = map[string]bool{
	"propose_application_status_update": true,
}

// NewRecruitingADKTools builds Eino BaseTool wrappers for HR recruiting tools.
// Per-request owner ID and AgentRunState are read from context (WithOwnerID /
// WithAgentRunState). The executor is captured at creation time so tools may
// be cached across requests.
//
// allowlist, when non-empty, restricts tools to the named set (agent bindings).
// When allowlist is empty, no model tools are returned (fail-closed for
// configured agents that disabled everything).
func NewRecruitingADKTools(executor RecruitingToolExecutor, allowlist []string) ([]tool.BaseTool, error) {
	if executor == nil {
		return nil, fmt.Errorf("recruiting tool executor must not be nil")
	}
	allowed := map[string]bool{}
	for _, name := range allowlist {
		name = strings.TrimSpace(name)
		if name == "" || adkExcludedModelTools[name] {
			continue
		}
		allowed[name] = true
	}
	if len(allowed) == 0 {
		return nil, nil
	}

	errorHandler := func(_ context.Context, err error) string {
		data, _ := json.Marshal(map[string]any{
			"error":   true,
			"message": err.Error(),
		})
		return string(data)
	}

	out := make([]tool.BaseTool, 0, len(allowed))
	for _, info := range RecruitingTools() {
		if info == nil || !allowed[info.Name] {
			continue
		}
		toolName := info.Name
		// Clone ToolInfo pointer content so each tool keeps its own desc.
		desc := *info
		base := &executorBackedTool{
			info:     &desc,
			toolName: toolName,
			executor: executor,
		}
		out = append(out, utils.WrapToolWithErrorHandler(base, errorHandler))
	}
	return out, nil
}

// executorBackedTool adapts RecruitingToolExecutor to eino InvokableTool without
// requiring per-tool typed structs. Empty / null argument payloads are treated
// as {}.
type executorBackedTool struct {
	info     *schema.ToolInfo
	toolName string
	executor RecruitingToolExecutor
}

func (t *executorBackedTool) Info(context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *executorBackedTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	args := map[string]any{}
	raw := strings.TrimSpace(argumentsInJSON)
	if raw != "" && raw != "null" {
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			return "", fmt.Errorf("invalid tool arguments for %s: %w", t.toolName, err)
		}
		if args == nil {
			args = map[string]any{}
		}
	}
	ownerID := ownerIDFromContext(ctx)
	if ownerID <= 0 {
		return "", fmt.Errorf("owner_id is required for tool %s", t.toolName)
	}
	state := agentStateFromContext(ctx)
	result, err := t.executor.Execute(ctx, ownerID, t.toolName, args)
	if state != nil && err == nil {
		state.Merge(result.Metadata)
	}
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// FilterADKToolsByName keeps only tools whose Info.Name is in allowlist.
func FilterADKToolsByName(tools []tool.BaseTool, allowlist []string) []tool.BaseTool {
	if len(tools) == 0 || len(allowlist) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, name := range allowlist {
		name = strings.TrimSpace(name)
		if name != "" {
			allowed[name] = true
		}
	}
	out := make([]tool.BaseTool, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		info, err := t.Info(context.Background())
		if err != nil || info == nil || !allowed[info.Name] {
			continue
		}
		out = append(out, t)
	}
	return out
}

// ADKToolNames returns tool names from a BaseTool list.
func ADKToolNames(tools []tool.BaseTool) []string {
	out := make([]string, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		info, err := t.Info(context.Background())
		if err != nil || info == nil || strings.TrimSpace(info.Name) == "" {
			continue
		}
		out = append(out, info.Name)
	}
	return out
}
