package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"smart-recruit/dev-log-viewer/internal/catalog"
	"smart-recruit/dev-log-viewer/internal/stream"
)

type Server struct {
	catalog     *catalog.Catalog
	hub         *stream.Hub
	snapshotter Snapshotter
	heartbeat   time.Duration
	static      fs.FS
	mux         *http.ServeMux
}

type Snapshotter interface {
	Snapshot(serviceIDs []string, tailLines int, observedAt time.Time) []stream.Envelope
}

func New(catalog *catalog.Catalog) *Server {
	return NewWithStream(catalog, nil, nil)
}

func NewWithStream(catalog *catalog.Catalog, hub *stream.Hub, snapshotter Snapshotter) *Server {
	return NewWithStreamAndStatic(catalog, hub, snapshotter, nil)
}

func NewWithStreamAndStatic(catalog *catalog.Catalog, hub *stream.Hub, snapshotter Snapshotter, static fs.FS) *Server {
	if hub == nil {
		hub = stream.NewHub(stream.DefaultEventBuffer, stream.DefaultClientQueue)
	}
	server := &Server{
		catalog:     catalog,
		hub:         hub,
		snapshotter: snapshotter,
		heartbeat:   15 * time.Second,
		static:      static,
		mux:         http.NewServeMux(),
	}
	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	return securityHeaders(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /api/v1/services", s.handleServices)
	s.mux.HandleFunc("GET /api/v1/logs/stream", s.handleStream)
	s.mux.HandleFunc("GET /", s.handleStatic)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) > 0 {
		http.Error(w, "query parameters are not supported", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "dev-log-viewer",
	})
}

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) > 0 {
		http.Error(w, "query parameters are not supported", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]catalog.ServiceStatus{
		"services": s.catalog.Statuses(),
	})
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	serviceIDs, filter, tailLines, ok := s.parseStreamQuery(w, r)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	lastEventHeader := r.Header.Get("Last-Event-ID")
	if strings.TrimSpace(lastEventHeader) != "" {
		lastID, err := parseLastEventID(lastEventHeader)
		if err != nil {
			writeSSE(w, stream.Envelope{Type: stream.TypeReset, Recoverable: false})
		} else if replay, recoverable := s.hub.Replay(lastID, filter); recoverable {
			for _, event := range replay {
				writeSSE(w, event)
			}
		} else {
			writeSSE(w, stream.Envelope{Type: stream.TypeReset, Recoverable: false})
		}
	}

	writeSSE(w, stream.Envelope{Type: stream.TypeSnapshotStart})
	if s.snapshotter != nil {
		for _, event := range s.snapshotter.Snapshot(serviceIDs, tailLines, time.Now().UTC()) {
			writeSSE(w, event)
		}
	}
	writeSSE(w, stream.Envelope{Type: stream.TypeSnapshotEnd})
	flusher.Flush()

	sub := s.hub.Subscribe(filter)
	defer sub.Close()
	ticker := time.NewTicker(s.heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			writeSSE(w, stream.Envelope{Type: stream.TypeHeartbeat})
			flusher.Flush()
		case event, open := <-sub.Events():
			if !open {
				return
			}
			if dropped := sub.Dropped(); dropped > 0 {
				writeSSE(w, stream.Envelope{Type: stream.TypeDropped, Dropped: dropped})
			}
			writeSSE(w, event)
			flusher.Flush()
		}
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if s.static == nil {
		http.Error(w, "static assets are unavailable; run pnpm --filter dev-log-viewer build", http.StatusServiceUnavailable)
		return
	}
	if len(r.URL.Query()) > 0 {
		http.Error(w, "query parameters are not supported", http.StatusBadRequest)
		return
	}
	assetPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if assetPath == "" {
		assetPath = "index.html"
	}
	servePath := assetPath
	data, err := fs.ReadFile(s.static, assetPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			servePath = "index.html"
			data, err = fs.ReadFile(s.static, servePath)
		}
		if err != nil {
			http.Error(w, "static asset is unavailable", http.StatusNotFound)
			return
		}
	}
	w.Header().Set("Cache-Control", staticCacheControl(servePath))
	http.ServeContent(w, r, path.Base(servePath), time.Time{}, bytes.NewReader(data))
}

func (s *Server) parseStreamQuery(w http.ResponseWriter, r *http.Request) ([]string, map[string]bool, int, bool) {
	query := r.URL.Query()
	tailLines := 300
	if values := query["tail"]; len(values) > 0 {
		if len(values) > 1 {
			http.Error(w, "tail may only be specified once", http.StatusBadRequest)
			return nil, nil, 0, false
		}
		parsed, err := strconv.Atoi(values[0])
		if err != nil || parsed < 0 || parsed > 1000 {
			http.Error(w, "tail must be in range 0..1000", http.StatusBadRequest)
			return nil, nil, 0, false
		}
		tailLines = parsed
	}

	known := map[string]bool{}
	for _, definition := range s.catalog.Definitions() {
		known[definition.ID] = true
	}
	filter := map[string]bool{}
	ids := []string{}
	for _, service := range query["service"] {
		if !known[service] {
			http.Error(w, "unknown service", http.StatusBadRequest)
			return nil, nil, 0, false
		}
		if !filter[service] {
			filter[service] = true
			ids = append(ids, service)
		}
	}
	for key := range query {
		if key != "service" && key != "tail" {
			http.Error(w, "unsupported query parameter", http.StatusBadRequest)
			return nil, nil, 0, false
		}
	}
	if len(filter) == 0 {
		filter = nil
	}
	return ids, filter, tailLines, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeSSE(w http.ResponseWriter, envelope stream.Envelope) {
	if envelope.EventID > 0 {
		_, _ = fmt.Fprintf(w, "id: %d\n", envelope.EventID)
	}
	eventName := envelope.Type
	if eventName == "" {
		eventName = stream.TypeLog
	}
	_, _ = fmt.Fprintf(w, "event: %s\n", eventName)
	data, err := json.Marshal(envelope)
	if err != nil {
		data = []byte(`{"type":"error"}`)
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
}

func parseLastEventID(value string) (uint64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return strconv.ParseUint(value, 10, 64)
}

func Shutdown(ctx context.Context, server *http.Server) error {
	return server.Shutdown(ctx)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		next.ServeHTTP(w, r)
	})
}

func staticCacheControl(assetPath string) string {
	if assetPath == "index.html" {
		return "no-cache"
	}
	return "public, max-age=31536000, immutable"
}
