package ai

import (
	"context"
	"testing"
	"time"
)

func TestApplyTotalBudgetNoBudget(t *testing.T) {
	ctx, cancel := applyTotalBudget(context.Background(), 0)
	defer cancel()
	if _, ok := ctx.Deadline(); ok {
		t.Fatal("expected no deadline when budget is 0")
	}
}

func TestApplyTotalBudgetAddsDeadline(t *testing.T) {
	ctx, cancel := applyTotalBudget(context.Background(), 100*time.Millisecond)
	defer cancel()
	d, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline")
	}
	if remaining := time.Until(d); remaining > 100*time.Millisecond || remaining <= 0 {
		t.Errorf("unexpected remaining: %v", remaining)
	}
}

func TestApplyTotalBudgetParentTighter(t *testing.T) {
	parent, parentCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer parentCancel()
	ctx, cancel := applyTotalBudget(parent, 1*time.Second)
	defer cancel()
	d, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected deadline")
	}
	if remaining := time.Until(d); remaining > 100*time.Millisecond {
		t.Errorf("expected to keep parent's tighter deadline, got remaining=%v", remaining)
	}
}

func TestBudgetExhausted(t *testing.T) {
	start := time.Now().Add(-200 * time.Millisecond)
	if !budgetExhausted(start, 100*time.Millisecond) {
		t.Error("expected exhausted")
	}
	if budgetExhausted(start, 1*time.Second) {
		t.Error("not yet exhausted")
	}
	if budgetExhausted(start, 0) {
		t.Error("zero budget should report not exhausted (disabled)")
	}
}

func TestToolBudgetExhausted(t *testing.T) {
	if !toolBudgetExhausted(150*time.Millisecond, 100*time.Millisecond) {
		t.Error("expected exhausted when over budget")
	}
	if toolBudgetExhausted(50*time.Millisecond, 100*time.Millisecond) {
		t.Error("expected not exhausted under budget")
	}
	if toolBudgetExhausted(time.Hour, 0) {
		t.Error("zero budget = disabled")
	}
}
