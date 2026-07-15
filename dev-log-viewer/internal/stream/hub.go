package stream

import (
	"sync"
	"sync/atomic"
)

const (
	DefaultEventBuffer = 20000
	DefaultClientQueue = 2048

	TypeSnapshotStart = "snapshot_start"
	TypeLog           = "log"
	TypeSnapshotEnd   = "snapshot_end"
	TypeReset         = "reset"
	TypeDropped       = "dropped"
	TypeHeartbeat     = "heartbeat"
)

type Envelope struct {
	Type        string `json:"type"`
	EventID     uint64 `json:"event_id,omitempty"`
	Service     string `json:"service,omitempty"`
	Payload     any    `json:"payload,omitempty"`
	Dropped     uint64 `json:"dropped,omitempty"`
	Recoverable bool   `json:"recoverable,omitempty"`
}

type Hub struct {
	mu       sync.Mutex
	nextID   uint64
	capacity int
	queue    int
	ring     []Envelope
	subs     map[*Subscription]struct{}
}

type Subscription struct {
	hub     *Hub
	filter  map[string]bool
	ch      chan Envelope
	closed  atomic.Bool
	dropped atomic.Uint64
}

func NewHub(capacity int, queue int) *Hub {
	if capacity <= 0 {
		capacity = DefaultEventBuffer
	}
	if queue <= 0 {
		queue = DefaultClientQueue
	}
	return &Hub{
		capacity: capacity,
		queue:    queue,
		subs:     map[*Subscription]struct{}{},
	}
}

func (h *Hub) Publish(envelope Envelope) Envelope {
	h.mu.Lock()
	h.nextID++
	envelope.EventID = h.nextID
	if envelope.Type == "" {
		envelope.Type = TypeLog
	}
	h.ring = append(h.ring, envelope)
	if len(h.ring) > h.capacity {
		copy(h.ring, h.ring[len(h.ring)-h.capacity:])
		h.ring = h.ring[:h.capacity]
	}
	subs := make([]*Subscription, 0, len(h.subs))
	for sub := range h.subs {
		if sub.matches(envelope.Service) {
			subs = append(subs, sub)
		}
	}
	h.mu.Unlock()

	for _, sub := range subs {
		select {
		case sub.ch <- envelope:
		default:
			sub.dropped.Add(1)
		}
	}
	return envelope
}

func (h *Hub) Replay(lastID uint64, filter map[string]bool) ([]Envelope, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.ring) == 0 {
		return nil, lastID == 0
	}
	minID := h.ring[0].EventID
	if lastID != 0 && lastID < minID {
		return nil, false
	}
	replay := []Envelope{}
	for _, event := range h.ring {
		if event.EventID > lastID && matchesFilter(filter, event.Service) {
			replay = append(replay, event)
		}
	}
	return replay, true
}

func (h *Hub) Subscribe(filter map[string]bool) *Subscription {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub := &Subscription{
		hub:    h,
		filter: copyFilter(filter),
		ch:     make(chan Envelope, h.queue),
	}
	h.subs[sub] = struct{}{}
	return sub
}

func (s *Subscription) Events() <-chan Envelope {
	return s.ch
}

func (s *Subscription) Dropped() uint64 {
	return s.dropped.Swap(0)
}

func (s *Subscription) Close() {
	if s.closed.Swap(true) {
		return
	}
	s.hub.mu.Lock()
	delete(s.hub.subs, s)
	s.hub.mu.Unlock()
	close(s.ch)
}

func (s *Subscription) matches(service string) bool {
	return matchesFilter(s.filter, service)
}

func matchesFilter(filter map[string]bool, service string) bool {
	return len(filter) == 0 || filter[service]
}

func copyFilter(filter map[string]bool) map[string]bool {
	if len(filter) == 0 {
		return nil
	}
	copied := make(map[string]bool, len(filter))
	for key, value := range filter {
		copied[key] = value
	}
	return copied
}
