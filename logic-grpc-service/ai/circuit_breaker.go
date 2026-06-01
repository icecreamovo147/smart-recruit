package ai

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"logic-grpc-service/pkg/logger"
)

var ErrCircuitOpen = errors.New("ai circuit is open")

type breakerState int

const (
	breakerClosed breakerState = iota
	breakerOpen
	breakerHalfOpen
)

type CircuitBreaker struct {
	mu                  sync.Mutex
	state               breakerState
	failures            int
	failureThreshold    int
	openUntil           time.Time
	openTimeout         time.Duration
	halfOpenInFlight    int
	halfOpenMaxRequests int
}

func NewCircuitBreaker(failureThreshold int, openTimeout time.Duration, halfOpenMaxRequests int) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 5
	}
	if openTimeout <= 0 {
		openTimeout = 30 * time.Second
	}
	if halfOpenMaxRequests <= 0 {
		halfOpenMaxRequests = 2
	}
	return &CircuitBreaker{
		failureThreshold:    failureThreshold,
		openTimeout:         openTimeout,
		halfOpenMaxRequests: halfOpenMaxRequests,
	}
}

func (b *CircuitBreaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case breakerOpen:
		return "open"
	case breakerHalfOpen:
		return "half_open"
	default:
		return "closed"
	}
}

func (b *CircuitBreaker) BeforeCall() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if b.state == breakerOpen {
		if now.Before(b.openUntil) {
			return fmt.Errorf("%w: retry after %s", ErrCircuitOpen, time.Until(b.openUntil).Round(time.Second))
		}
		b.state = breakerHalfOpen
		b.halfOpenInFlight = 0
		logger.L().Warn("[熔断器] 进入半开状态，开始探测", zap.Duration("open_timeout", b.openTimeout))
	}
	if b.state == breakerHalfOpen {
		if b.halfOpenInFlight >= b.halfOpenMaxRequests {
			return ErrCircuitOpen
		}
		b.halfOpenInFlight++
	}
	return nil
}

func (b *CircuitBreaker) AfterCall(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == breakerHalfOpen && b.halfOpenInFlight > 0 {
		b.halfOpenInFlight--
	}
	if err == nil || errors.Is(err, context.Canceled) {
		b.state = breakerClosed
		b.failures = 0
		b.halfOpenInFlight = 0
		return
	}
	b.failures++
	if b.state == breakerHalfOpen || b.failures >= b.failureThreshold {
		prevState := b.state
		b.state = breakerOpen
		b.openUntil = time.Now().Add(b.openTimeout)
		b.halfOpenInFlight = 0
		logger.L().Warn("[熔断器] 熔断打开",
			zap.Int("consecutive_failures", b.failures),
			zap.Int("prev_state", int(prevState)),
			zap.Duration("open_for", b.openTimeout),
		)
	}
}
