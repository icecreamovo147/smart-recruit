package ai

import (
	"context"
	"sync"
	"testing"
	"time"

	chatmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type loopOptionsModel struct {
	mu    sync.Mutex
	calls int
}

func (m *loopOptionsModel) Generate(context.Context, []*schema.Message, ...chatmodel.Option) (*schema.Message, error) {
	return schema.AssistantMessage("final", nil), nil
}

func (m *loopOptionsModel) Stream(_ context.Context, messages []*schema.Message, _ ...chatmodel.Option) (*schema.StreamReader[*schema.Message], error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	final := false
	for _, message := range messages {
		if message.Role == schema.Tool || message.Role == schema.System && containsAny(message.Content, "工具调用轮次已达到上限") {
			final = true
			break
		}
	}
	var response *schema.Message
	if final {
		response = schema.AssistantMessage("final", nil)
	} else {
		response = &schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{
			ID: "call-1", Type: "function", Function: schema.FunctionCall{Name: "get_job_list", Arguments: `{}`},
		}}}
	}
	reader, writer := schema.Pipe[*schema.Message](1)
	go func() {
		defer writer.Close()
		writer.Send(response, nil)
	}()
	return reader, nil
}

func (m *loopOptionsModel) WithTools([]*schema.ToolInfo) (chatmodel.ToolCallingChatModel, error) {
	return m, nil
}

type countingToolRunner struct{ calls int }

func (r *countingToolRunner) Execute(context.Context, int64, string, map[string]any) (ToolResult, error) {
	r.calls++
	return ToolResult{Content: `{"jobs":[]}`}, nil
}

func TestChatWithToolsPerCallMaxRoundsOverridesSharedDefault(t *testing.T) {
	model := &loopOptionsModel{}
	client := &Client{
		model:            "loop-options-test",
		cm:               model,
		timeout:          time.Second,
		toolMaxRounds:    5,
		retryMaxAttempts: 0,
		sem:              make(chan struct{}, 1),
		breaker:          NewCircuitBreaker(2, time.Minute, 1),
	}
	runner := &countingToolRunner{}
	reply, _, err := client.ChatWithToolsWithOptions(
		context.Background(),
		[]*schema.Message{schema.UserMessage("jobs")},
		[]*schema.ToolInfo{{Name: "get_job_list"}},
		runner,
		1,
		nil,
		nil,
		nil,
		ToolLoopOptions{MaxRounds: 1},
	)
	if err != nil || reply != "final" {
		t.Fatalf("reply=%q err=%v", reply, err)
	}
	if runner.calls != 0 {
		t.Fatalf("tool calls=%d, want 0 when round 1 is the limit", runner.calls)
	}
}

func TestChatWithToolsCompatibilityWrapperUsesClientDefault(t *testing.T) {
	model := &loopOptionsModel{}
	client := &Client{
		model:            "loop-options-test",
		cm:               model,
		timeout:          time.Second,
		toolMaxRounds:    5,
		retryMaxAttempts: 0,
		sem:              make(chan struct{}, 1),
		breaker:          NewCircuitBreaker(2, time.Minute, 1),
	}
	runner := &countingToolRunner{}
	_, _, err := client.ChatWithTools(
		context.Background(),
		[]*schema.Message{schema.UserMessage("jobs")},
		[]*schema.ToolInfo{{Name: "get_job_list"}},
		runner,
		1,
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("ChatWithTools: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("tool calls=%d, want compatibility default to execute one tool", runner.calls)
	}
}

func TestChatWithToolsPreparesEveryProviderCallIncludingFallback(t *testing.T) {
	model := &loopOptionsModel{}
	client := &Client{model: "prepare-test", cm: model, timeout: time.Second, toolMaxRounds: 5, sem: make(chan struct{}, 1), breaker: NewCircuitBreaker(2, time.Minute, 1)}
	calls := 0
	reply, _, err := client.ChatWithToolsWithOptions(
		context.Background(), []*schema.Message{schema.UserMessage("jobs")}, []*schema.ToolInfo{{Name: "get_job_list"}}, &countingToolRunner{}, 1,
		nil, nil, nil,
		ToolLoopOptions{MaxRounds: 1, PrepareMessages: func(_ context.Context, messages []*schema.Message, _ string) ([]*schema.Message, error) {
			calls++
			return messages, nil
		}},
	)
	if err != nil || reply != "final" {
		t.Fatalf("reply=%q err=%v", reply, err)
	}
	if calls != 2 {
		t.Fatalf("prepare calls=%d, want first model call and fallback call", calls)
	}
}
