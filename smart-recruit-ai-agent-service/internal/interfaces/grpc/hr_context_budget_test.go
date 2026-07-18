package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

type budgetSummaryStoreFake struct {
	mu      sync.Mutex
	summary string
	covered int64
	writes  int
	done    chan struct{}
}

func (f *budgetSummaryStoreFake) GetSessionSummaryState(context.Context, int64, int64) (string, int64, int, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.summary, f.covered, 0, f.summary != "", nil
}

func (f *budgetSummaryStoreFake) UpsertSessionSummaryIfNewer(_ context.Context, _, _ int64, summary string, covered int64, _ int) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if covered <= f.covered {
		return false, nil
	}
	f.summary, f.covered = summary, covered
	f.writes++
	if f.done != nil {
		select {
		case <-f.done:
		default:
			close(f.done)
		}
	}
	return true, nil
}

type budgetSummaryGeneratorFake struct {
	err   error
	value string
	calls int
}

func (f *budgetSummaryGeneratorFake) GenerateSessionSummary(context.Context, string, []string) (string, error) {
	f.calls++
	return f.value, f.err
}

func TestHRContextBudgetKnownSelectsRecentHistoryInChronologicalOrder(t *testing.T) {
	model := RuntimeModelInfo{ContextWindowTokens: 520, MaxOutputTokens: 100} // B=164
	history := make([]ChatMessageRow, 0, 8)
	messages := []*schema.Message{schema.SystemMessage("fixed governance")}
	for i := 1; i <= 8; i++ {
		content := fmt.Sprintf("history-%02d %s", i, strings.Repeat("x", 64))
		history = append(history, ChatMessageRow{ID: int64(i), Role: "assistant", Content: content})
		messages = append(messages, schema.AssistantMessage(content, nil))
	}
	messages = append(messages, schema.UserMessage("current question"))
	c := newHRContextBudgetController(context.Background(), model, nil, "current question", nil, hrRuntimeGovernanceContext{}, history, 1, 2, nil, nil)
	prepared, err := c.prepare(context.Background(), messages, "test")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := estimateMessagesEnvelopeTokens(prepared, nil); int64(got) > int64(c.usage.GetInputBudgetTokens()) {
		t.Fatalf("prepared tokens=%d exceed budget=%d", got, c.usage.GetInputBudgetTokens())
	}
	var selected []string
	for _, message := range prepared {
		if strings.HasPrefix(message.Content, "history-") {
			selected = append(selected, strings.Fields(message.Content)[0])
		}
	}
	if len(selected) == 0 || selected[len(selected)-1] != "history-08" {
		t.Fatalf("recent-first selection=%v", selected)
	}
	for i := 1; i < len(selected); i++ {
		if selected[i-1] >= selected[i] {
			t.Fatalf("not chronological: %v", selected)
		}
	}
	if c.usage.GetOmittedMessageCount() == 0 || c.usage.GetIncludedMessageCount() != int32(len(prepared)) {
		t.Fatalf("usage counts=%+v", c.usage)
	}
}

func TestHRContextBudgetUnknownUsesSummaryAndRecentTwenty(t *testing.T) {
	store := &budgetSummaryStoreFake{summary: "older facts", covered: 5}
	history := make([]ChatMessageRow, 0, 30)
	messages := []*schema.Message{schema.SystemMessage("system")}
	for i := 1; i <= 30; i++ {
		content := fmt.Sprintf("m-%02d", i)
		history = append(history, ChatMessageRow{ID: int64(i), Role: "assistant", Content: content})
		messages = append(messages, schema.AssistantMessage(content, nil))
	}
	messages = append(messages, schema.UserMessage("now"))
	c := newHRContextBudgetController(context.Background(), RuntimeModelInfo{}, nil, "now", nil, hrRuntimeGovernanceContext{}, history, 1, 2, store, nil)
	prepared, err := c.prepare(context.Background(), messages, "test")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if c.usage.GetBudgetStatus() != "unknown_config" || c.usage.GetBudgetUsageRatio() != 0 || !c.usage.GetSummaryApplied() {
		t.Fatalf("usage=%+v", c.usage)
	}
	if len(prepared) != 23 { // system + summary + recent20 + current
		t.Fatalf("prepared count=%d", len(prepared))
	}
	if !strings.Contains(prepared[2].Content, "m-11") || !strings.Contains(prepared[len(prepared)-2].Content, "m-30") {
		t.Fatalf("unexpected recent tail: first=%q last=%q", prepared[2].Content, prepared[len(prepared)-2].Content)
	}
}

func TestHRContextBudgetRejectsInvalidAndFixedOnlyOverflowBeforeCall(t *testing.T) {
	invalid := newHRContextBudgetController(context.Background(), RuntimeModelInfo{ContextWindowTokens: 256, MaxOutputTokens: 10}, nil, "q", nil, hrRuntimeGovernanceContext{}, nil, 1, 2, nil, nil)
	if _, err := invalid.prepare(context.Background(), []*schema.Message{schema.SystemMessage("s"), schema.UserMessage("q")}, "test"); hrContextErrorCode(err) != hrContextConfigInvalidCode {
		t.Fatalf("invalid config error=%v", err)
	}
	overflow := newHRContextBudgetController(context.Background(), RuntimeModelInfo{ContextWindowTokens: 520, MaxOutputTokens: 100}, nil, "q", nil, hrRuntimeGovernanceContext{}, nil, 1, 2, nil, nil)
	_, err := overflow.prepare(context.Background(), []*schema.Message{schema.SystemMessage(strings.Repeat("界", 200)), schema.UserMessage("q")}, "test")
	if hrContextErrorCode(err) != hrContextBudgetExceededCode {
		t.Fatalf("overflow error=%v", err)
	}
}

func TestHRContextBudgetSyncSummaryFailureFallsBackToTrimming(t *testing.T) {
	store := &budgetSummaryStoreFake{}
	gen := &budgetSummaryGeneratorFake{err: errors.New("synthetic")}
	history := make([]ChatMessageRow, 0, 15)
	messages := []*schema.Message{schema.SystemMessage("system")}
	for i := 1; i <= 15; i++ {
		content := fmt.Sprintf("old-%d %s", i, strings.Repeat("x", 80))
		history = append(history, ChatMessageRow{ID: int64(i), Role: "assistant", Content: content})
		messages = append(messages, schema.AssistantMessage(content, nil))
	}
	messages = append(messages, schema.UserMessage("q"))
	c := newHRContextBudgetController(context.Background(), RuntimeModelInfo{ContextWindowTokens: 600, MaxOutputTokens: 100}, nil, "q", nil, hrRuntimeGovernanceContext{}, history, 1, 2, store, gen)
	prepared, err := c.prepare(context.Background(), messages, "test")
	if err != nil {
		t.Fatalf("prepare should trim after summary failure: %v", err)
	}
	if gen.calls != 1 || c.usage.GetOmittedMessageCount() == 0 || c.usage.GetSummaryApplied() {
		t.Fatalf("calls=%d usage=%+v", gen.calls, c.usage)
	}
	if int64(estimateMessagesEnvelopeTokens(prepared, nil)) > int64(c.usage.GetInputBudgetTokens()) {
		t.Fatal("trimmed request exceeds budget")
	}
}

func TestHRContextBudgetAsyncSummaryAtSixtyPercent(t *testing.T) {
	store := &budgetSummaryStoreFake{done: make(chan struct{})}
	gen := &budgetSummaryGeneratorFake{value: "generated"}
	history := make([]ChatMessageRow, 0, 12)
	messages := []*schema.Message{schema.SystemMessage(strings.Repeat("a", 1000))}
	for i := 1; i <= 12; i++ {
		content := fmt.Sprintf("h-%d-%s", i, strings.Repeat("x", 20))
		history = append(history, ChatMessageRow{ID: int64(i), Role: "assistant", Content: content})
		messages = append(messages, schema.AssistantMessage(content, nil))
	}
	messages = append(messages, schema.UserMessage("q"))
	c := newHRContextBudgetController(context.Background(), RuntimeModelInfo{ContextWindowTokens: 900, MaxOutputTokens: 100}, nil, "q", nil, hrRuntimeGovernanceContext{}, history, 1, 2, store, gen)
	if _, err := c.prepare(context.Background(), messages, "test"); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	select {
	case <-store.done:
	case <-time.After(2 * time.Second):
		t.Fatal("async summary was not triggered")
	}
}
