package ai

import (
	"context"
	"time"
)

// applyTotalBudget returns a derived context that enforces an end-to-end total
// AI conversation budget. If the parent already has a tighter deadline, no new
// timeout is applied (parent wins, cancel is a no-op).
func applyTotalBudget(parent context.Context, totalBudget time.Duration) (context.Context, context.CancelFunc) {
	if totalBudget <= 0 {
		return parent, func() {}
	}
	if deadline, ok := parent.Deadline(); ok {
		// Parent already has a deadline. Only tighten when our budget is shorter.
		if time.Until(deadline) <= totalBudget {
			return parent, func() {}
		}
	}
	return context.WithTimeout(parent, totalBudget)
}

// budgetExhausted reports whether elapsed has consumed totalBudget. It is the
// loop-side companion to applyTotalBudget — applyTotalBudget enforces a hard
// deadline via context, while this helper lets callers branch into a fallback
// path before issuing another model call.
func budgetExhausted(start time.Time, totalBudget time.Duration) bool {
	if totalBudget <= 0 {
		return false
	}
	return time.Since(start) >= totalBudget
}

// toolBudgetExhausted reports whether a tool's cumulative elapsed time has
// consumed its dedicated budget. Returns false when budget <= 0 (disabled).
func toolBudgetExhausted(toolCumulative, toolBudget time.Duration) bool {
	if toolBudget <= 0 {
		return false
	}
	return toolCumulative >= toolBudget
}
