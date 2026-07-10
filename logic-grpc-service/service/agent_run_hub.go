package service

import (
	"sync"

	"logic-grpc-service/model"
)

// agentRunEventHub is an in-process fan-out for live durable run subscribers.
// Multi-instance deployments still recover via DB poll + ListEventsAfter.
type agentRunEventHub struct {
	mu   sync.RWMutex
	subs map[uint64]map[chan *model.AgentRunEvent]struct{}
}

func newAgentRunEventHub() *agentRunEventHub {
	return &agentRunEventHub{
		subs: make(map[uint64]map[chan *model.AgentRunEvent]struct{}),
	}
}

// Subscribe registers a buffered channel for runID. Caller must invoke cancel.
func (h *agentRunEventHub) Subscribe(runID uint64) (<-chan *model.AgentRunEvent, func()) {
	if h == nil {
		ch := make(chan *model.AgentRunEvent)
		close(ch)
		return ch, func() {}
	}
	ch := make(chan *model.AgentRunEvent, 64)
	h.mu.Lock()
	if h.subs[runID] == nil {
		h.subs[runID] = make(map[chan *model.AgentRunEvent]struct{})
	}
	h.subs[runID][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if set, ok := h.subs[runID]; ok {
			if _, exists := set[ch]; exists {
				delete(set, ch)
				close(ch)
			}
			if len(set) == 0 {
				delete(h.subs, runID)
			}
		}
		h.mu.Unlock()
	}
	return ch, cancel
}

// Publish non-blocks: full subscriber buffers drop the event (DB poll recovers).
func (h *agentRunEventHub) Publish(runID uint64, event *model.AgentRunEvent) {
	if h == nil || event == nil {
		return
	}
	h.mu.RLock()
	set := h.subs[runID]
	targets := make([]chan *model.AgentRunEvent, 0, len(set))
	for ch := range set {
		targets = append(targets, ch)
	}
	h.mu.RUnlock()
	for _, ch := range targets {
		select {
		case ch <- event:
		default:
		}
	}
}
