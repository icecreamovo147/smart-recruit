package outbox

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDispatchOncePublishesAndMarksPublished(t *testing.T) {
	store := &fakeStore{events: []Event{{
		ID:         10,
		EventID:    "evt-10",
		RoutingKey: "notification.create",
		Payload:    `{"event_id":"evt-10"}`,
	}}}
	publisher := &fakePublisher{}
	dispatcher := NewDispatcher(store, publisher, Options{
		BatchSize:     7,
		WorkerID:      "worker-test",
		MaxRetryCount: 3,
	})

	count, err := dispatcher.DispatchOnce(context.Background())
	if err != nil {
		t.Fatalf("DispatchOnce returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("DispatchOnce count = %d, want 1", count)
	}
	if store.claimLimit != 7 || store.claimWorkerID != "worker-test" {
		t.Fatalf("claim args = limit %d worker %q", store.claimLimit, store.claimWorkerID)
	}
	if len(publisher.calls) != 1 || publisher.calls[0].routingKey != "notification.create" || publisher.calls[0].body != `{"event_id":"evt-10"}` {
		t.Fatalf("publish calls = %#v", publisher.calls)
	}
	if len(store.published) != 1 || store.published[0] != 10 {
		t.Fatalf("published IDs = %#v", store.published)
	}
	if len(store.retryable) != 0 || len(store.dead) != 0 {
		t.Fatalf("unexpected retry/dead calls: retry=%#v dead=%#v", store.retryable, store.dead)
	}
}

func TestDispatchOnceMarksRetryableFailure(t *testing.T) {
	store := &fakeStore{events: []Event{{
		ID:         11,
		EventID:    "evt-11",
		RoutingKey: "email.send",
		Payload:    `{"event_id":"evt-11"}`,
		RetryCount: 1,
	}}}
	publisher := &fakePublisher{err: errors.New("broker down")}
	dispatcher := NewDispatcher(store, publisher, Options{
		BackoffBase:   time.Second,
		MaxBackoff:    time.Second,
		MaxRetryCount: 5,
		WorkerID:      "worker-test",
	})
	before := time.Now()

	count, err := dispatcher.DispatchOnce(context.Background())
	if err != nil {
		t.Fatalf("DispatchOnce returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("DispatchOnce count = %d, want 1", count)
	}
	if len(store.retryable) != 1 {
		t.Fatalf("retryable calls = %#v", store.retryable)
	}
	call := store.retryable[0]
	if call.id != 11 || call.errMsg != "broker down" || !call.nextRetry.After(before) {
		t.Fatalf("retryable call = %#v", call)
	}
	if len(store.published) != 0 || len(store.dead) != 0 {
		t.Fatalf("unexpected published/dead calls: published=%#v dead=%#v", store.published, store.dead)
	}
}

func TestDispatchOnceMarksDeadAfterRetryBudget(t *testing.T) {
	store := &fakeStore{events: []Event{{
		ID:         12,
		EventID:    "evt-12",
		RoutingKey: "resume.parse",
		Payload:    `{"event_id":"evt-12"}`,
		RetryCount: 2,
	}}}
	publisher := &fakePublisher{err: errors.New("broker down")}
	dispatcher := NewDispatcher(store, publisher, Options{
		MaxRetryCount: 3,
		WorkerID:      "worker-test",
	})

	count, err := dispatcher.DispatchOnce(context.Background())
	if err != nil {
		t.Fatalf("DispatchOnce returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("DispatchOnce count = %d, want 1", count)
	}
	if len(store.dead) != 1 || store.dead[0].id != 12 || store.dead[0].errMsg != "broker down" {
		t.Fatalf("dead calls = %#v", store.dead)
	}
	if len(store.retryable) != 0 || len(store.published) != 0 {
		t.Fatalf("unexpected retry/published calls: retry=%#v published=%#v", store.retryable, store.published)
	}
}

func TestDispatcherRejectsMissingDependencies(t *testing.T) {
	if err := NewDispatcher(nil, &fakePublisher{}, Options{}).Start(context.Background()); !errors.Is(err, ErrStoreRequired) {
		t.Fatalf("missing store error = %v, want %v", err, ErrStoreRequired)
	}
	if err := NewDispatcher(&fakeStore{}, nil, Options{}).Start(context.Background()); !errors.Is(err, ErrPublisherRequired) {
		t.Fatalf("missing publisher error = %v, want %v", err, ErrPublisherRequired)
	}
}

type fakeStore struct {
	events        []Event
	claimLimit    int
	claimWorkerID string
	claimTimeout  time.Duration
	published     []uint64
	retryable     []retryableCall
	dead          []deadCall
}

func (s *fakeStore) ClaimPending(_ context.Context, limit int, workerID string, lockTimeout time.Duration) ([]Event, error) {
	s.claimLimit = limit
	s.claimWorkerID = workerID
	s.claimTimeout = lockTimeout
	return append([]Event(nil), s.events...), nil
}

func (s *fakeStore) MarkPublished(_ context.Context, id uint64) error {
	s.published = append(s.published, id)
	return nil
}

func (s *fakeStore) MarkRetryableFailure(_ context.Context, id uint64, errMsg string, nextRetry time.Time) error {
	s.retryable = append(s.retryable, retryableCall{id: id, errMsg: errMsg, nextRetry: nextRetry})
	return nil
}

func (s *fakeStore) MarkDead(_ context.Context, id uint64, errMsg string) error {
	s.dead = append(s.dead, deadCall{id: id, errMsg: errMsg})
	return nil
}

type retryableCall struct {
	id        uint64
	errMsg    string
	nextRetry time.Time
}

type deadCall struct {
	id     uint64
	errMsg string
}

type fakePublisher struct {
	err   error
	calls []publishCall
}

func (p *fakePublisher) Publish(_ context.Context, routingKey string, body []byte) error {
	p.calls = append(p.calls, publishCall{routingKey: routingKey, body: string(body)})
	return p.err
}

type publishCall struct {
	routingKey string
	body       string
}
