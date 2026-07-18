package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func TestEscapeADKInstruction(t *testing.T) {
	input := "使用 tool {name} 查询数据"
	got := escapeADKInstruction(input)
	expected := "使用 tool {{name}} 查询数据"
	if got != expected {
		t.Errorf("escapeADKInstruction(%q) = %q, want %q", input, got, expected)
	}
}

func TestRecruitingAgentMiddlewarePreparesAuthoritativeState(t *testing.T) {
	middleware := &RecruitingAgentMiddleware{PrepareMessages: func(_ context.Context, messages []*schema.Message, stage string) ([]*schema.Message, error) {
		if stage != "adk_model_call" {
			t.Fatalf("stage=%q", stage)
		}
		return append(messages, schema.SystemMessage("budgeted")), nil
	}}
	state := &adk.ChatModelAgentState{Messages: []*schema.Message{schema.UserMessage("hello")}}
	_, rewritten, err := middleware.BeforeModelRewriteState(context.Background(), state, nil)
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if len(rewritten.Messages) != 2 || rewritten.Messages[1].Content != "budgeted" {
		t.Fatalf("messages=%v", rewritten.Messages)
	}
}

func TestEscapeADKInstruction_JSONExample(t *testing.T) {
	input := `{"key": "value"}`
	got := escapeADKInstruction(input)
	expected := `{{"key": "value"}}`
	if got != expected {
		t.Errorf("escapeADKInstruction(JSON) = %q, want %q", got, expected)
	}
}

func TestAgentRunState_Merge(t *testing.T) {
	state := &AgentRunState{}
	state.Merge(ToolMetadata{
		CandidateOptions: []ToolCandidateOption{
			{ApplicationID: 1, CandidateName: "张三"},
		},
	})
	state.Merge(ToolMetadata{
		Action: &ToolAction{Action: "approve_application", ApplicationID: 1},
	})

	if len(state.Metadata.CandidateOptions) != 1 {
		t.Errorf("expected 1 candidate option, got %d", len(state.Metadata.CandidateOptions))
	}
	if state.Metadata.Action == nil {
		t.Fatal("expected action to be set")
	}
	if state.Metadata.Action.Action != "approve_application" {
		t.Errorf("expected approve_application, got %s", state.Metadata.Action.Action)
	}
}

func TestAgentRunState_RecordTrace(t *testing.T) {
	state := &AgentRunState{}
	state.RecordTrace(ToolTrace{
		ToolName:  "query_today_applications",
		Arguments: map[string]any{"job_id": float64(1)},
		Result:    `{"today_applications": 5}`,
	})

	if len(state.Metadata.ToolTraces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(state.Metadata.ToolTraces))
	}
	tt := state.Metadata.ToolTraces[0]
	if tt.ToolName != "query_today_applications" {
		t.Errorf("expected query_today_applications, got %s", tt.ToolName)
	}
}

func TestAgentRunState_Concurrency(t *testing.T) {
	state := &AgentRunState{}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			state.RecordTrace(ToolTrace{ToolName: "test"})
		}(i)
	}
	wg.Wait()
	if len(state.Metadata.ToolTraces) != 100 {
		t.Errorf("expected 100 traces, got %d (race condition?)", len(state.Metadata.ToolTraces))
	}
}

func TestMarshalToolError(t *testing.T) {
	got := marshalToolError(nil)
	if got != `{"error":false}` {
		t.Errorf("nil error: got %s", got)
	}

	got = marshalToolError(json.Unmarshal([]byte("{"), &struct{}{})) // creates a real error
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("marshalToolError output is not valid JSON: %v", err)
	}
	if v, ok := m["error"]; !ok || v != true {
		t.Errorf("expected error:true, got %v", m)
	}
}

func TestRecruitingAgentMiddlewarePreservesStructuredPolicyErrorTrace(t *testing.T) {
	state := &AgentRunState{}
	middleware := &RecruitingAgentMiddleware{State: state}
	var gotArgsJSON, gotResult string
	var gotErr error
	middleware.OnToolExecuted = func(toolCallID, toolName, argsJSON, resultContent string, duration time.Duration, execErr error) {
		gotArgsJSON = argsJSON
		gotResult = resultContent
		gotErr = execErr
	}
	endpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		return `{"policy_decision":"confirmation_required","policy_reason":"policy_requires_confirmation","arguments_json":"{\"candidate_name\":\"***\",\"api_key\":\"***\"}","result_content":"{\"policy_decision\":\"confirmation_required\"}"}`, errors.New("MCP policy/tool call rejected: confirmation required")
	}
	wrapped, err := middleware.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{Name: "mcp_policy_server_search", CallID: "call-1"})
	if err != nil {
		t.Fatalf("WrapInvokableToolCall failed: %v", err)
	}
	result, err := wrapped(context.Background(), `{"candidate_name":"Alice","api_key":"sk_secret"}`)
	if err != nil {
		t.Fatalf("wrapped tool should return structured error content with nil error, got %v", err)
	}
	if !json.Valid([]byte(result)) || !strings.Contains(result, `"policy_decision":"confirmation_required"`) {
		t.Fatalf("expected structured policy output, got %s", result)
	}
	if strings.Contains(gotArgsJSON, "Alice") || strings.Contains(gotArgsJSON, "sk_secret") {
		t.Fatalf("expected OnToolExecuted args to be redacted, got %s", gotArgsJSON)
	}
	if gotResult != result || gotErr == nil {
		t.Fatalf("expected callback to receive structured result and original exec error")
	}
	if len(state.Metadata.ToolTraces) != 1 {
		t.Fatalf("expected one tool trace, got %d", len(state.Metadata.ToolTraces))
	}
	trace := state.Metadata.ToolTraces[0]
	if trace.Arguments["candidate_name"] != "***" || trace.Arguments["api_key"] != "***" {
		t.Fatalf("expected redacted trace arguments, got %#v", trace.Arguments)
	}
	if !strings.Contains(trace.Result, `"policy_decision":"confirmation_required"`) {
		t.Fatalf("expected trace result to preserve policy decision, got %s", trace.Result)
	}
}

func TestParseArgsForTrace_ValidJSON(t *testing.T) {
	args := parseArgsForTrace(`{"job_id": 42, "keyword": "test"}`)
	if v, ok := args["job_id"]; !ok || v != float64(42) {
		t.Errorf("expected job_id=42, got %v", args)
	}
}

func TestParseArgsForTrace_InvalidJSON(t *testing.T) {
	args := parseArgsForTrace(`not json`)
	if raw, ok := args["_raw"]; !ok || raw != "not json" {
		t.Errorf("expected _raw key, got %v", args)
	}
}

func TestAgentRunInput_Defaults(t *testing.T) {
	// Verify that MaxIterations=0 with toolMaxRounds=0 defaults to 5.
	input := AgentRunInput{
		AgentName: "test",
		OwnerID:   1,
	}
	if input.MaxIterations != 0 {
		t.Errorf("zero-value should be 0")
	}
}

func TestEmptyStateRecorder(t *testing.T) {
	var state *AgentRunState // nil
	state.RecordTrace(ToolTrace{ToolName: "test"})
	// Should not panic
	state.Merge(ToolMetadata{})
}
