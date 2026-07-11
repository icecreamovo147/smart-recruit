package observability

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var DefaultMetrics = NewRegistry("web_gin_service")

type Registry struct {
	service string
	mu      sync.Mutex
	http    map[httpKey]*histogram
	rpc     map[rpcKey]*histogram
	panics  map[panicKey]uint64
}

type httpKey struct {
	Method string
	Route  string
	Status string
}

type rpcKey struct {
	Method string
	Code   string
}

type panicKey struct {
	Method string
	Route  string
}

type histogram struct {
	Count   uint64
	Sum     float64
	Buckets []uint64
}

var defaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

func NewRegistry(service string) *Registry {
	return &Registry{
		service: service,
		http:    make(map[httpKey]*histogram),
		rpc:     make(map[rpcKey]*histogram),
		panics:  make(map[panicKey]uint64),
	}
}

func (r *Registry) ObserveHTTPRequest(method, route string, status int, elapsed time.Duration) {
	if r == nil {
		return
	}
	route = normalizeLabel(route, "unknown")
	key := httpKey{
		Method: normalizeLabel(method, "UNKNOWN"),
		Route:  route,
		Status: strconv.Itoa(status),
	}
	seconds := elapsed.Seconds()
	r.mu.Lock()
	defer r.mu.Unlock()
	h := r.http[key]
	if h == nil {
		h = &histogram{Buckets: make([]uint64, len(defaultBuckets))}
		r.http[key] = h
	}
	h.Count++
	h.Sum += seconds
	for i, bucket := range defaultBuckets {
		if seconds <= bucket {
			h.Buckets[i]++
		}
	}
}

func (r *Registry) ObserveGRPCClient(method, code string, elapsed time.Duration) {
	if r == nil {
		return
	}
	key := rpcKey{Method: normalizeLabel(method, "unknown"), Code: normalizeLabel(code, "UNKNOWN")}
	seconds := elapsed.Seconds()
	r.mu.Lock()
	defer r.mu.Unlock()
	h := r.rpc[key]
	if h == nil {
		h = &histogram{Buckets: make([]uint64, len(defaultBuckets))}
		r.rpc[key] = h
	}
	h.Count++
	h.Sum += seconds
	for i, bucket := range defaultBuckets {
		if seconds <= bucket {
			h.Buckets[i]++
		}
	}
}

func (r *Registry) RecordHTTPPanic(method, route string) {
	if r == nil {
		return
	}
	key := panicKey{Method: normalizeLabel(method, "UNKNOWN"), Route: normalizeLabel(route, "unknown")}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.panics[key]++
}

func (r *Registry) Prometheus() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	snapshots := make([]httpSnapshot, 0, len(r.http))
	for key, h := range r.http {
		buckets := append([]uint64(nil), h.Buckets...)
		snapshots = append(snapshots, httpSnapshot{key: key, count: h.Count, sum: h.Sum, buckets: buckets})
	}
	rpcSnapshots := make([]rpcSnapshot, 0, len(r.rpc))
	for key, h := range r.rpc {
		rpcSnapshots = append(rpcSnapshots, rpcSnapshot{
			key:     key,
			count:   h.Count,
			sum:     h.Sum,
			buckets: append([]uint64(nil), h.Buckets...),
		})
	}
	panicSnapshots := make([]panicSnapshot, 0, len(r.panics))
	for key, count := range r.panics {
		panicSnapshots = append(panicSnapshots, panicSnapshot{key: key, count: count})
	}
	r.mu.Unlock()
	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].key.Method != snapshots[j].key.Method {
			return snapshots[i].key.Method < snapshots[j].key.Method
		}
		if snapshots[i].key.Route != snapshots[j].key.Route {
			return snapshots[i].key.Route < snapshots[j].key.Route
		}
		return snapshots[i].key.Status < snapshots[j].key.Status
	})
	sort.Slice(rpcSnapshots, func(i, j int) bool {
		if rpcSnapshots[i].key.Method != rpcSnapshots[j].key.Method {
			return rpcSnapshots[i].key.Method < rpcSnapshots[j].key.Method
		}
		return rpcSnapshots[i].key.Code < rpcSnapshots[j].key.Code
	})
	sort.Slice(panicSnapshots, func(i, j int) bool {
		if panicSnapshots[i].key.Method != panicSnapshots[j].key.Method {
			return panicSnapshots[i].key.Method < panicSnapshots[j].key.Method
		}
		return panicSnapshots[i].key.Route < panicSnapshots[j].key.Route
	})

	var b strings.Builder
	fmt.Fprintf(&b, "# HELP smart_recruit_http_requests_total HTTP requests handled by %s.\n", r.service)
	b.WriteString("# TYPE smart_recruit_http_requests_total counter\n")
	for _, s := range snapshots {
		fmt.Fprintf(&b, "smart_recruit_http_requests_total%s %d\n", httpLabels(s.key, ""), s.count)
	}
	fmt.Fprintf(&b, "# HELP smart_recruit_http_request_duration_seconds HTTP request latency for %s.\n", r.service)
	b.WriteString("# TYPE smart_recruit_http_request_duration_seconds histogram\n")
	for _, s := range snapshots {
		var cumulative uint64
		for i, bucket := range defaultBuckets {
			cumulative = s.buckets[i]
			fmt.Fprintf(&b, "smart_recruit_http_request_duration_seconds_bucket%s %d\n", httpLabels(s.key, fmt.Sprintf("%g", bucket)), cumulative)
		}
		fmt.Fprintf(&b, "smart_recruit_http_request_duration_seconds_bucket%s %d\n", httpLabels(s.key, "+Inf"), s.count)
		fmt.Fprintf(&b, "smart_recruit_http_request_duration_seconds_sum%s %.9f\n", httpLabels(s.key, ""), s.sum)
		fmt.Fprintf(&b, "smart_recruit_http_request_duration_seconds_count%s %d\n", httpLabels(s.key, ""), s.count)
	}
	b.WriteString("# HELP smart_recruit_http_panics_total HTTP panics recovered by the gateway.\n")
	b.WriteString("# TYPE smart_recruit_http_panics_total counter\n")
	for _, s := range panicSnapshots {
		fmt.Fprintf(&b, "smart_recruit_http_panics_total%s %d\n", panicLabels(s.key), s.count)
	}
	fmt.Fprintf(&b, "# HELP smart_recruit_grpc_client_requests_total gRPC client requests sent by %s.\n", r.service)
	b.WriteString("# TYPE smart_recruit_grpc_client_requests_total counter\n")
	for _, s := range rpcSnapshots {
		fmt.Fprintf(&b, "smart_recruit_grpc_client_requests_total%s %d\n", rpcLabels(s.key, ""), s.count)
	}
	fmt.Fprintf(&b, "# HELP smart_recruit_grpc_client_request_duration_seconds gRPC client latency for %s.\n", r.service)
	b.WriteString("# TYPE smart_recruit_grpc_client_request_duration_seconds histogram\n")
	for _, s := range rpcSnapshots {
		for i, bucket := range defaultBuckets {
			fmt.Fprintf(&b, "smart_recruit_grpc_client_request_duration_seconds_bucket%s %d\n", rpcLabels(s.key, fmt.Sprintf("%g", bucket)), s.buckets[i])
		}
		fmt.Fprintf(&b, "smart_recruit_grpc_client_request_duration_seconds_bucket%s %d\n", rpcLabels(s.key, "+Inf"), s.count)
		fmt.Fprintf(&b, "smart_recruit_grpc_client_request_duration_seconds_sum%s %.9f\n", rpcLabels(s.key, ""), s.sum)
		fmt.Fprintf(&b, "smart_recruit_grpc_client_request_duration_seconds_count%s %d\n", rpcLabels(s.key, ""), s.count)
	}
	return b.String()
}

type httpSnapshot struct {
	key     httpKey
	count   uint64
	sum     float64
	buckets []uint64
}

type rpcSnapshot struct {
	key     rpcKey
	count   uint64
	sum     float64
	buckets []uint64
}

type panicSnapshot struct {
	key   panicKey
	count uint64
}

func httpLabels(key httpKey, le string) string {
	labels := []string{
		`service="web-gin-service"`,
		fmt.Sprintf(`method="%s"`, escapeLabel(key.Method)),
		fmt.Sprintf(`route="%s"`, escapeLabel(key.Route)),
		fmt.Sprintf(`status="%s"`, escapeLabel(key.Status)),
	}
	if le != "" {
		labels = append(labels, fmt.Sprintf(`le="%s"`, escapeLabel(le)))
	}
	return "{" + strings.Join(labels, ",") + "}"
}

func rpcLabels(key rpcKey, le string) string {
	labels := []string{
		`service="web-gin-service"`,
		fmt.Sprintf(`grpc_method="%s"`, escapeLabel(key.Method)),
		fmt.Sprintf(`grpc_code="%s"`, escapeLabel(key.Code)),
	}
	if le != "" {
		labels = append(labels, fmt.Sprintf(`le="%s"`, escapeLabel(le)))
	}
	return "{" + strings.Join(labels, ",") + "}"
}

func panicLabels(key panicKey) string {
	labels := []string{
		`service="web-gin-service"`,
		fmt.Sprintf(`method="%s"`, escapeLabel(key.Method)),
		fmt.Sprintf(`route="%s"`, escapeLabel(key.Route)),
	}
	return "{" + strings.Join(labels, ",") + "}"
}

func normalizeLabel(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func escapeLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}
