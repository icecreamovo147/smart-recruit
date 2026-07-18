package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	chatmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"smart-recruit-platform-go/logger"
)

type structuredTestModel struct {
	mu       sync.Mutex
	calls    int
	messages [][]*schema.Message
	generate func(context.Context, int) (*schema.Message, error)
}

func (m *structuredTestModel) Generate(ctx context.Context, input []*schema.Message, _ ...chatmodel.Option) (*schema.Message, error) {
	m.mu.Lock()
	m.calls++
	call := m.calls
	cloned := make([]*schema.Message, len(input))
	for i, message := range input {
		copyMessage := *message
		cloned[i] = &copyMessage
	}
	m.messages = append(m.messages, cloned)
	m.mu.Unlock()
	return m.generate(ctx, call)
}

func (m *structuredTestModel) Stream(context.Context, []*schema.Message, ...chatmodel.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("stream is not used by structured completion")
}

func (m *structuredTestModel) WithTools([]*schema.ToolInfo) (chatmodel.ToolCallingChatModel, error) {
	return m, nil
}

func newStructuredTestClient(model chatmodel.ToolCallingChatModel) *Client {
	return &Client{
		model:            "structured-test-model",
		cm:               model,
		timeout:          50 * time.Millisecond,
		retryMaxAttempts: 0,
		retryBaseDelay:   time.Millisecond,
		sem:              make(chan struct{}, 1),
		breaker:          NewCircuitBreaker(2, time.Minute, 1),
	}
}

func TestGenerateStructuredPreservesExactSystemUserRoles(t *testing.T) {
	model := &structuredTestModel{generate: func(context.Context, int) (*schema.Message, error) {
		return schema.AssistantMessage(`{"ok":true}`, nil), nil
	}}
	client := newStructuredTestClient(model)

	result, err := client.GenerateStructured(context.Background(), "database system prompt", "scoped user payload")
	if err != nil {
		t.Fatalf("GenerateStructured: %v", err)
	}
	if result.Content != `{"ok":true}` || result.ModelName != "structured-test-model" {
		t.Fatalf("result = %#v", result)
	}
	if model.calls != 1 || len(model.messages[0]) != 2 {
		t.Fatalf("calls/messages = %d/%d, want 1/2", model.calls, len(model.messages[0]))
	}
	if got := model.messages[0][0]; got.Role != schema.System || got.Content != "database system prompt" {
		t.Fatalf("system message = %#v", got)
	}
	if got := model.messages[0][1]; got.Role != schema.User || got.Content != "scoped user payload" {
		t.Fatalf("user message = %#v", got)
	}
	for _, message := range model.messages[0] {
		if strings.Contains(message.Content, "Markdown 输出硬性规范") || strings.Contains(message.Content, "智能招聘系统的数据分析助手") {
			t.Fatal("generic recruiting Markdown prompt contaminated structured messages")
		}
	}
}

func TestGenerateStructuredUsesRetryAndClassifiesFailure(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	previousLogger := logger.L()
	logger.Set(zap.New(core))
	t.Cleanup(func() { logger.Set(previousLogger) })

	const sensitiveMarker = "provider-secret-marker-7c91"
	model := &structuredTestModel{generate: func(_ context.Context, call int) (*schema.Message, error) {
		if call == 1 {
			return nil, errors.New("provider 503 unavailable " + sensitiveMarker)
		}
		return schema.AssistantMessage(`{"retried":true}`, nil), nil
	}}
	client := newStructuredTestClient(model)
	client.retryMaxAttempts = 1

	result, err := client.GenerateStructured(context.Background(), "system", "user")
	if err != nil {
		t.Fatalf("GenerateStructured retry: %v", err)
	}
	if result.Content != `{"retried":true}` || model.calls != 2 {
		t.Fatalf("result/calls = %#v/%d", result, model.calls)
	}

	failing := &structuredTestModel{generate: func(context.Context, int) (*schema.Message, error) {
		return nil, errors.New("provider 503 unavailable " + sensitiveMarker)
	}}
	failingClient := newStructuredTestClient(failing)
	_, err = failingClient.GenerateStructured(context.Background(), "private system body", "private user body")
	var aiErr *AIError
	if !errors.As(err, &aiErr) || aiErr.Type != AIUnavailable {
		t.Fatalf("error = %#v, want AIUnavailable", err)
	}
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message, sensitiveMarker) || strings.Contains(fmt.Sprint(entry.Context), sensitiveMarker) {
			t.Fatalf("structured log exposed raw provider error marker: %#v", entry)
		}
	}
}

func TestGenerateStructuredUsesTimeoutAndCircuitBreaker(t *testing.T) {
	timeoutModel := &structuredTestModel{generate: func(ctx context.Context, _ int) (*schema.Message, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	timeoutClient := newStructuredTestClient(timeoutModel)
	_, err := timeoutClient.GenerateStructured(context.Background(), "system", "user")
	var aiErr *AIError
	if !errors.As(err, &aiErr) || aiErr.Type != AITimeout {
		t.Fatalf("timeout error = %#v, want AITimeout", err)
	}

	circuitModel := &structuredTestModel{generate: func(context.Context, int) (*schema.Message, error) {
		return nil, errors.New("provider 503 unavailable")
	}}
	circuitClient := newStructuredTestClient(circuitModel)
	circuitClient.breaker = NewCircuitBreaker(1, time.Minute, 1)
	_, _ = circuitClient.GenerateStructured(context.Background(), "system", "first")
	_, err = circuitClient.GenerateStructured(context.Background(), "system", "second")
	if !errors.As(err, &aiErr) || aiErr.Type != AICircuitOpen {
		t.Fatalf("circuit error = %#v, want AICircuitOpen", err)
	}
	if circuitModel.calls != 1 {
		t.Fatalf("model calls = %d, want circuit to reject second call", circuitModel.calls)
	}
}
