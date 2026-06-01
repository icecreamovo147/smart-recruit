package ai

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestClassifyAIError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want AIErrorType
	}{
		{"nil returns nil", nil, ""},
		{"circuit open sentinel", ErrCircuitOpen, AICircuitOpen},
		{"circuit open wrapped", fmt.Errorf("wrap: %w", ErrCircuitOpen), AICircuitOpen},
		{"context canceled", context.Canceled, AIContextCanceled},
		{"context deadline", context.DeadlineExceeded, AITimeout},
		{"timeout text", errors.New("upstream timeout exceeded"), AITimeout},
		{"deadline text", errors.New("context deadline exceeded"), AITimeout},
		{"rate limit 429", errors.New("HTTP 429: too many requests"), AIRateLimited},
		{"rate limit text", errors.New("rate_limit reached"), AIRateLimited},
		{"empty reply", errors.New("ai returned empty reply"), AIEmptyReply},
		{"unavailable text", errors.New("connection refused"), AIUnavailable},
		{"http 503", errors.New("HTTP 503 Service Unavailable"), AIUnavailable},
		{"unknown", errors.New("something else broke"), AIUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyAIError(tc.err)
			if tc.err == nil {
				if got != nil {
					t.Fatalf("expected nil for nil error, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected non-nil AIError")
			}
			if got.Type != tc.want {
				t.Errorf("classify(%q): want %s, got %s", tc.err.Error(), tc.want, got.Type)
			}
			if got.UserMessage == "" {
				t.Errorf("classify(%q): empty user message", tc.err.Error())
			}
			if !errors.Is(got, tc.err) && got.Cause != tc.err {
				t.Errorf("classify(%q): cause not preserved", tc.err.Error())
			}
		})
	}
}

func TestClassifyAIErrorPreservesType(t *testing.T) {
	original := NewAIError(AIToolFailed, "工具失败", errors.New("inner"))
	got := ClassifyAIError(original)
	if got != original {
		t.Fatalf("expected typed AIError to be returned as-is")
	}
}

func TestUserMessageForCoversAllTypes(t *testing.T) {
	types := []AIErrorType{AITimeout, AIRateLimited, AIUnavailable, AIEmptyReply, AIToolFailed, AIContextCanceled, AICircuitOpen, AIPartialReply, AIUnknown}
	for _, t1 := range types {
		if UserMessageFor(t1) == "" {
			t.Errorf("missing user message for %s", t1)
		}
	}
}

func TestIsRetryable(t *testing.T) {
	retry := []AIErrorType{AITimeout, AIRateLimited, AIUnavailable}
	notRetry := []AIErrorType{AIContextCanceled, AICircuitOpen, AIToolFailed, AIEmptyReply, AIPartialReply, AIUnknown}
	for _, x := range retry {
		if !IsRetryable(x) {
			t.Errorf("%s should be retryable", x)
		}
	}
	for _, x := range notRetry {
		if IsRetryable(x) {
			t.Errorf("%s should NOT be retryable", x)
		}
	}
}
