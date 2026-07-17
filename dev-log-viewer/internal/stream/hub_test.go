package stream

import "testing"

func TestHubMonotonicRingReplayAndResetGap(t *testing.T) {
	hub := NewHub(3, 2)
	for _, service := range []string{"a", "b", "a", "a"} {
		hub.Publish(Envelope{Type: TypeLog, Service: service})
	}

	replay, ok := hub.Replay(2, map[string]bool{"a": true})
	if !ok {
		t.Fatal("expected replay to be recoverable")
	}
	if len(replay) != 2 || replay[0].EventID != 3 || replay[1].EventID != 4 {
		t.Fatalf("replay = %+v", replay)
	}
	if _, ok := hub.Replay(1, nil); ok {
		t.Fatal("expected old last id to be unrecoverable")
	}
}

func TestSlowSubscriberDropsWithoutBlockingOthers(t *testing.T) {
	hub := NewHub(10, 1)
	slow := hub.Subscribe(nil)
	defer slow.Close()
	fast := hub.Subscribe(nil)
	defer fast.Close()

	hub.Publish(Envelope{Type: TypeLog, Service: "svc"})
	hub.Publish(Envelope{Type: TypeLog, Service: "svc"})
	hub.Publish(Envelope{Type: TypeLog, Service: "svc"})

	if dropped := slow.Dropped(); dropped == 0 {
		t.Fatal("expected slow subscriber drops")
	}
	if event := <-fast.Events(); event.EventID != 1 {
		t.Fatalf("fast first event = %+v", event)
	}
}
