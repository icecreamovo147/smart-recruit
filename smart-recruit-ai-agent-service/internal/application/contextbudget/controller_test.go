package contextbudget

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

func TestKnownBudgetSelectsRecentHistoryInChronologicalOrder(t *testing.T) {
	model := ModelInfo{ContextWindowTokens: 520, MaxOutputTokens: 100}
	history := make([]MessageRow, 0, 8)
	messages := []*schema.Message{schema.SystemMessage("fixed governance")}
	for i := 1; i <= 8; i++ {
		content := fmt.Sprintf("history-%02d %s", i, strings.Repeat("x", 64))
		history = append(history, MessageRow{ID: int64(i), Role: "assistant", Content: content})
		messages = append(messages, schema.AssistantMessage(content, nil))
	}
	messages = append(messages, schema.UserMessage("current question"))
	c := New(context.Background(), Config{
		Model: model, Current: "current question", History: history,
	})
	prepared, err := c.Prepare(context.Background(), messages, "test")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if got := EstimateMessagesEnvelopeTokens(prepared, nil); int64(got) > int64(InputBudgetTokens(model)) {
		t.Fatalf("prepared tokens=%d exceed budget=%d", got, InputBudgetTokens(model))
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
	if c.LastPrepareResult().OmittedCount == 0 {
		t.Fatalf("expected omitted history")
	}
}

func TestUnknownBudgetUsesSummaryAndRecentTwenty(t *testing.T) {
	store := &budgetSummaryStoreFake{summary: "older facts", covered: 5}
	history := make([]MessageRow, 0, 30)
	messages := []*schema.Message{schema.SystemMessage("system")}
	for i := 1; i <= 30; i++ {
		content := fmt.Sprintf("m-%02d", i)
		history = append(history, MessageRow{ID: int64(i), Role: "assistant", Content: content})
		messages = append(messages, schema.AssistantMessage(content, nil))
	}
	messages = append(messages, schema.UserMessage("now"))
	c := New(context.Background(), Config{History: history, Current: "now", Store: store})
	prepared, err := c.Prepare(context.Background(), messages, "test")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	result := c.LastPrepareResult()
	if !result.SummaryApplied {
		t.Fatalf("summary not applied: %+v", result)
	}
	if len(prepared) != 23 {
		t.Fatalf("prepared count=%d", len(prepared))
	}
	if !strings.Contains(prepared[1].Content, SummaryMessagePrefix) ||
		!strings.Contains(prepared[2].Content, "m-11") ||
		!strings.Contains(prepared[len(prepared)-2].Content, "m-30") {
		t.Fatalf("unexpected layout: %+v", messageContents(prepared))
	}
}

func TestBudgetRejectsInvalidAndFixedOnlyOverflow(t *testing.T) {
	invalid := New(context.Background(), Config{Model: ModelInfo{ContextWindowTokens: 256, MaxOutputTokens: 10}, Current: "q"})
	if _, err := invalid.Prepare(context.Background(), []*schema.Message{schema.SystemMessage("s"), schema.UserMessage("q")}, "test"); ErrorCode(err) != ConfigInvalidCode {
		t.Fatalf("invalid config error=%v", err)
	}
	overflow := New(context.Background(), Config{
		Model: ModelInfo{ContextWindowTokens: 520, MaxOutputTokens: 100}, Current: "q",
	})
	_, err := overflow.Prepare(context.Background(), []*schema.Message{schema.SystemMessage(strings.Repeat("界", 200)), schema.UserMessage("q")}, "test")
	if ErrorCode(err) != BudgetExceededCode {
		t.Fatalf("overflow error=%v", err)
	}
}

func TestSyncSummaryFailureFallsBackToTrimming(t *testing.T) {
	store := &budgetSummaryStoreFake{}
	gen := &budgetSummaryGeneratorFake{err: errors.New("synthetic")}
	history := make([]MessageRow, 0, 15)
	messages := []*schema.Message{schema.SystemMessage("system")}
	for i := 1; i <= 15; i++ {
		content := fmt.Sprintf("old-%d %s", i, strings.Repeat("x", 80))
		history = append(history, MessageRow{ID: int64(i), Role: "assistant", Content: content})
		messages = append(messages, schema.AssistantMessage(content, nil))
	}
	messages = append(messages, schema.UserMessage("q"))
	c := New(context.Background(), Config{
		Model:   ModelInfo{ContextWindowTokens: 600, MaxOutputTokens: 100},
		Current: "q", History: history, Store: store, Generator: gen,
	})
	prepared, err := c.Prepare(context.Background(), messages, "test")
	if err != nil {
		t.Fatalf("prepare should trim after summary failure: %v", err)
	}
	result := c.LastPrepareResult()
	if gen.calls != 1 || result.OmittedCount == 0 || result.SummaryApplied {
		t.Fatalf("calls=%d result=%+v", gen.calls, result)
	}
	if int64(EstimateMessagesEnvelopeTokens(prepared, nil)) > int64(InputBudgetTokens(c.model)) {
		t.Fatal("trimmed request exceeds budget")
	}
}

func TestAsyncSummaryAtSixtyPercent(t *testing.T) {
	store := &budgetSummaryStoreFake{done: make(chan struct{})}
	gen := &budgetSummaryGeneratorFake{value: "generated"}
	history := make([]MessageRow, 0, 12)
	messages := []*schema.Message{schema.SystemMessage(strings.Repeat("a", 1000))}
	for i := 1; i <= 12; i++ {
		content := fmt.Sprintf("h-%d-%s", i, strings.Repeat("x", 20))
		history = append(history, MessageRow{ID: int64(i), Role: "assistant", Content: content})
		messages = append(messages, schema.AssistantMessage(content, nil))
	}
	messages = append(messages, schema.UserMessage("q"))
	c := New(context.Background(), Config{
		Model:   ModelInfo{ContextWindowTokens: 900, MaxOutputTokens: 100},
		Current: "q", History: history, Store: store, Generator: gen,
	})
	if _, err := c.Prepare(context.Background(), messages, "test"); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	select {
	case <-store.done:
	case <-time.After(2 * time.Second):
		t.Fatal("async summary was not triggered")
	}
}

func TestSelectWithinBudgetDropsMemoryWhenTooLarge(t *testing.T) {
	fixed := []*schema.Message{
		schema.SystemMessage("system governance"),
		schema.UserMessage("current question"),
	}
	history := []*schema.Message{schema.AssistantMessage("older reply", nil)}
	largeMemory := strings.Repeat("长期记忆 ", 120)
	fixedBudget := int64(EstimateMessagesEnvelopeTokens(ComposeMessages(fixed, nil, "", "", -1), nil))
	budget := fixedBudget + 2

	prepared, omitted, _, memoryApplied, fixedOverflow := SelectWithinBudget(nil, fixed, history, "", largeMemory, -1, budget)
	if fixedOverflow {
		t.Fatal("fixed-only envelope should fit")
	}
	if memoryApplied {
		t.Fatal("memory should be dropped under tight budget")
	}
	if PreparedMemoryText(prepared) != "" {
		t.Fatalf("memory section present: %#v", messageContents(prepared))
	}
	if int64(EstimateMessagesEnvelopeTokens(prepared, nil)) > budget {
		t.Fatalf("prepared tokens exceed budget=%d", budget)
	}
	if omitted != 1 {
		t.Fatalf("omitted = %d, want 1 history message trimmed", omitted)
	}
}

func TestPrepareDropsMemoryUnderTightBudget(t *testing.T) {
	largeMemory := strings.Repeat("candidate prefers remote work ", 80)
	c := New(context.Background(), Config{
		Model:         ModelInfo{ContextWindowTokens: 600, MaxOutputTokens: 100},
		Current:       "current question",
		MemorySection: largeMemory,
	})
	messages := []*schema.Message{
		schema.SystemMessage("system governance"),
		schema.UserMessage("current question"),
	}
	prepared, err := c.Prepare(context.Background(), messages, "test")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	result := c.LastPrepareResult()
	if result.MemoryApplied {
		t.Fatal("memory_applied should be false when section dropped")
	}
	if PreparedMemoryText(prepared) != "" {
		t.Fatalf("memory still injected: %#v", messageContents(prepared))
	}
	budget := int64(InputBudgetTokens(c.model))
	if int64(EstimateMessagesEnvelopeTokens(prepared, nil)) > budget {
		t.Fatalf("prepared tokens=%d exceed budget=%d", EstimateMessagesEnvelopeTokens(prepared, nil), budget)
	}
}

func TestComposeInjectsMemorySection(t *testing.T) {
	prepared := ComposeMessages(
		[]*schema.Message{schema.SystemMessage("base system")},
		nil,
		"rolling facts",
		"memory one\nmemory two",
		-1,
	)
	if len(prepared) != 3 {
		t.Fatalf("prepared count=%d", len(prepared))
	}
	if !strings.HasPrefix(prepared[1].Content, SummaryMessagePrefix) {
		t.Fatalf("summary message missing: %q", prepared[1].Content)
	}
	if !strings.HasPrefix(prepared[2].Content, MemoryMessagePrefix) {
		t.Fatalf("memory message missing: %q", prepared[2].Content)
	}
}

func messageContents(messages []*schema.Message) []string {
	out := make([]string, 0, len(messages))
	for _, message := range messages {
		if message != nil {
			out = append(out, message.Content)
		}
	}
	return out
}
