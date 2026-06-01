package ai

import (
	"context"
	"errors"
	"testing"
	"time"
)

// newTestClient creates a minimal client for retry testing.
func newTestClient(retryMaxAttempts int) *Client {
	return &Client{
		timeout:          30 * time.Second,
		retryMaxAttempts: retryMaxAttempts,
		retryBaseDelay:   1 * time.Millisecond,
		sem:             make(chan struct{}, 1),
		breaker:         NewCircuitBreaker(5, 30*time.Second, 2),
	}
}

func TestCallNoRetryNonStreaming(t *testing.T) {
	c := newTestClient(2) // max 3 attempts
	var attempts int
	err := c.call(context.Background(), func(ctx context.Context) error {
		attempts++
		return context.DeadlineExceeded // is retryable
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts < 3 {
		t.Errorf("non-streaming call should retry, got %d attempts (expected >= 3)", attempts)
	}
}

func TestCallStreamingStopsRetryAfterOutput(t *testing.T) {
	c := newTestClient(2) // max 3 attempts
	var attempts int
	outputSent := false

	// simulate: first attempt streams delta then fails with a retryable error
	fn := func(ctx context.Context) error {
		attempts++
		if attempts == 1 {
			outputSent = true // simulate onDelta call
			return context.DeadlineExceeded
		}
		return errors.New("should never reach attempt 2")
	}

	hasOutput := func() bool { return outputSent }
	err := c.callWithRetry(context.Background(), fn, hasOutput)
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("streaming call should NOT retry after output sent, got %d attempts (expected 1)", attempts)
	}
}

func TestCallStreamingAllowsRetryBeforeOutput(t *testing.T) {
	c := newTestClient(2) // max 3 attempts
	var attempts int
	outputSent := false

	// First 2 attempts fail BEFORE any output is streamed (retryable errors).
	// retryBaseDelay is tiny so 3 attempts should complete quickly.
	fn := func(ctx context.Context) error {
		attempts++
		if attempts <= 2 {
			return context.DeadlineExceeded // retryable, no output yet
		}
		outputSent = true
		return nil // third attempt succeeds
	}

	hasOutput := func() bool { return outputSent }
	err := c.callWithRetry(context.Background(), fn, hasOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("should retry when no output sent yet, got %d attempts (expected 3)", attempts)
	}
}

func TestCallNoRetryForNonRetryableError(t *testing.T) {
	c := newTestClient(2)
	var attempts int
	err := c.call(context.Background(), func(ctx context.Context) error {
		attempts++
		return errors.New("some non-retryable error")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("non-retryable error should not retry, got %d attempts (expected 1)", attempts)
	}
}
