package ai

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerClosed(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second, 2)
	if err := cb.BeforeCall(); err != nil {
		t.Fatalf("expected no error in closed state, got %v", err)
	}
	cb.AfterCall(nil) // success resets
	if err := cb.BeforeCall(); err != nil {
		t.Fatalf("still closed after success, got %v", err)
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	cb := NewCircuitBreaker(2, 200*time.Millisecond, 1)
	cb.BeforeCall()
	cb.AfterCall(errors.New("fail 1"))
	cb.BeforeCall()
	cb.AfterCall(errors.New("fail 2"))
	// Third call should be rejected
	if err := cb.BeforeCall(); err == nil {
		t.Fatal("expected circuit open error after consecutive failures")
	} else if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond, 1)
	cb.BeforeCall()
	cb.AfterCall(errors.New("fail"))
	// Circuit is now open
	time.Sleep(20 * time.Millisecond) // wait for open timeout
	// Should transition to half-open and allow one probe
	if err := cb.BeforeCall(); err != nil {
		t.Fatalf("expected probe allowed in half-open, got %v", err)
	}
	// Second probe while half-open with in-flight should be rejected
	if err := cb.BeforeCall(); err == nil {
		t.Fatal("expected second half-open probe to be rejected")
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond, 1)
	cb.BeforeCall()
	cb.AfterCall(errors.New("fail"))
	// Circuit open
	time.Sleep(20 * time.Millisecond)
	// Half-open probe succeeds
	cb.BeforeCall()
	cb.AfterCall(nil)
	// Should be closed again
	if err := cb.BeforeCall(); err != nil {
		t.Fatalf("expected closed after successful probe, got %v", err)
	}
}

func TestCircuitBreakerContextCanceled(t *testing.T) {
	cb := NewCircuitBreaker(1, 100*time.Millisecond, 1)
	cb.BeforeCall()
	cb.AfterCall(errors.New("fail")) // circuit opens
	time.Sleep(150 * time.Millisecond) // wait for open timeout
	cb.BeforeCall() // half-open probe
	// context.Canceled should reset to closed, not count as failure
	cb.AfterCall(context.Canceled)
	if err := cb.BeforeCall(); err != nil {
		t.Fatalf("expected closed after context.Canceled, got %v", err)
	}
}

func TestCircuitBreakerDefaults(t *testing.T) {
	cb := NewCircuitBreaker(0, 0, 0)
	// Should use defaults: threshold=5, timeout=30s, halfOpenMax=2
	if err := cb.BeforeCall(); err != nil {
		t.Fatalf("expected closed with defaults, got %v", err)
	}
	cb.AfterCall(errors.New("fail"))
	cb.AfterCall(errors.New("fail"))
	cb.AfterCall(errors.New("fail"))
	cb.AfterCall(errors.New("fail"))
	cb.AfterCall(nil) // reset at 4 failures before threshold=5
	if err := cb.BeforeCall(); err != nil {
		t.Fatalf("expected closed after reset at 4 failures, got %v", err)
	}
}
