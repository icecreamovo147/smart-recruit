package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"smart-recruit/dev-log-viewer/internal/catalog"
	"smart-recruit/dev-log-viewer/internal/stream"
)

func TestHealthz(t *testing.T) {
	handler := newTestHandler(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("missing security header: %q", got)
	}
	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "ok" || payload["service"] != "dev-log-viewer" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestServices(t *testing.T) {
	handler := newTestHandler(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/services", nil)
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Services []catalog.ServiceStatus `json:"services"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Services) != 12 {
		t.Fatalf("service count = %d, want 12", len(payload.Services))
	}
	body := response.Body.String()
	if strings.Contains(body, ".dev/logs") || strings.Contains(body, ".dev/pids") {
		t.Fatalf("services response leaked file paths: %s", body)
	}
}

func TestServicesRejectQueryParameters(t *testing.T) {
	handler := newTestHandler(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/services?path=/tmp/secret.log", nil)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestStaticAssets(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".dev", "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".dev", "pids"), 0o755); err != nil {
		t.Fatal(err)
	}
	catalog, err := catalog.New(root)
	if err != nil {
		t.Fatal(err)
	}
	static := fstest.MapFS{
		"index.html":    {Data: []byte(`<div id="root"></div>`)},
		"assets/app.js": {Data: []byte(`console.log("dev-log-viewer")`)},
	}
	handler := NewWithStreamAndStatic(catalog, nil, nil, static).Handler()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `id="root"`) {
		t.Fatalf("index body = %s", response.Body.String())
	}
	if got := response.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Fatalf("csp = %q", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("index cache-control = %q", got)
	}

	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("asset status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("asset cache-control = %q", got)
	}
}

func TestStreamRejectsUnknownService(t *testing.T) {
	handler := newTestHandler(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/logs/stream?service=unknown", nil)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestStreamSendsSnapshotAndReplay(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".dev", "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".dev", "pids"), 0o755); err != nil {
		t.Fatal(err)
	}
	catalog, err := catalog.New(root)
	if err != nil {
		t.Fatal(err)
	}
	hub := stream.NewHub(10, 10)
	replayed := hub.Publish(stream.Envelope{Type: stream.TypeLog, Service: "identity-service", Payload: map[string]string{"message": "replay"}})
	handler := NewWithStream(catalog, hub, fakeSnapshotter{}).Handler()

	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/api/v1/logs/stream?service=identity-service&tail=1", nil).WithContext(ctx)
	response := newStreamRecorder()
	done := serveAsync(handler, response, request)

	body := waitForBody(t, response, "snapshot_end")
	if response.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	if response.Header().Get("Cache-Control") != "no-cache, no-store" {
		t.Fatalf("cache-control = %q", response.Header().Get("Cache-Control"))
	}
	cancel()
	waitDone(t, done)
	if !strings.Contains(body, "event: snapshot_start") || !strings.Contains(body, "snapshot payload") || !strings.Contains(body, "event: snapshot_end") {
		t.Fatalf("snapshot SSE body = %s", body)
	}

	ctx2, cancel2 := context.WithCancel(context.Background())
	request = httptest.NewRequest(http.MethodGet, "/api/v1/logs/stream?service=identity-service", nil).WithContext(ctx2)
	request.Header.Set("Last-Event-ID", "0")
	response = newStreamRecorder()
	done = serveAsync(handler, response, request)
	body = waitForBody(t, response, "id: "+strconvFormat(replayed.EventID))
	cancel2()
	waitDone(t, done)
	if !strings.Contains(body, "replay") {
		t.Fatalf("replay SSE body = %s", body)
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".dev", "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".dev", "pids"), 0o755); err != nil {
		t.Fatal(err)
	}
	catalog, err := catalog.New(root)
	if err != nil {
		t.Fatal(err)
	}
	return New(catalog).Handler()
}

type fakeSnapshotter struct{}

func (fakeSnapshotter) Snapshot(_ []string, _ int, _ time.Time) []stream.Envelope {
	return []stream.Envelope{{Type: stream.TypeLog, Service: "identity-service", Payload: map[string]string{"message": "snapshot payload"}}}
}

type streamRecorder struct {
	header http.Header
	mu     sync.Mutex
	body   bytes.Buffer
	status int
}

func newStreamRecorder() *streamRecorder {
	return &streamRecorder{header: http.Header{}}
}

func (r *streamRecorder) Header() http.Header {
	return r.header
}

func (r *streamRecorder) WriteHeader(status int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = status
}

func (r *streamRecorder) Write(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

func (r *streamRecorder) Flush() {}

func (r *streamRecorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body.String()
}

func serveAsync(handler http.Handler, response http.ResponseWriter, request *http.Request) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(response, request)
	}()
	return done
}

func waitForBody(t *testing.T, response *streamRecorder, marker string) string {
	t.Helper()
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %q in body %q", marker, response.String())
		case <-ticker.C:
			body := response.String()
			if strings.Contains(body, marker) {
				return body
			}
		}
	}
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stream handler did not stop after context cancellation")
	}
}

func strconvFormat(value uint64) string {
	return strconv.FormatUint(value, 10)
}
