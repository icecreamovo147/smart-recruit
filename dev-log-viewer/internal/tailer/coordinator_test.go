package tailer

import (
	"path/filepath"
	"testing"
	"time"

	"smart-recruit/dev-log-viewer/internal/catalog"
	"smart-recruit/dev-log-viewer/internal/stream"
)

func TestCoordinatorPublishesParsedLogRecords(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".dev", "logs", "identity-service.log")
	write(t, path, "")
	hub := stream.NewHub(10, 10)
	coordinator := NewCoordinator(root, []catalog.ServiceDefinition{catalog.DefaultDefinitions()[0]}, hub)
	sub := hub.Subscribe(nil)
	defer sub.Close()

	appendFile(t, path, "2026-07-15T09:00:00.123Z\tINFO\tsvc/main.go:1\tstarted\n")
	coordinator.PollOnce(time.Now())

	event := <-sub.Events()
	if event.Type != stream.TypeReset {
		t.Fatalf("first event = %+v", event)
	}
	event = <-sub.Events()
	if event.Type != stream.TypeLog || event.Service != "identity-service" {
		t.Fatalf("log event = %+v", event)
	}
}
