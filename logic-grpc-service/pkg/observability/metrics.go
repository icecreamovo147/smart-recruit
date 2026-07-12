package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var DefaultMetrics = NewRegistry("logic-grpc-service")

type Registry struct {
	service string
	mu      sync.Mutex
	rpc     map[rpcKey]*histogram
	panics  map[panicKey]uint64
	auth    map[authKey]uint64
}

type rpcKey struct {
	Method string
	Code   string
}

type panicKey struct {
	Method string
}

type authKey struct {
	Method string
	Reason string
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
		rpc:     make(map[rpcKey]*histogram),
		panics:  make(map[panicKey]uint64),
		auth:    make(map[authKey]uint64),
	}
}

func (r *Registry) ObserveRPC(method, code string, elapsed time.Duration) {
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

func (r *Registry) RecordRPCPanic(method string) {
	if r == nil {
		return
	}
	key := panicKey{Method: normalizeLabel(method, "unknown")}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.panics[key]++
}

func (r *Registry) RecordAuthRejection(method, reason string) {
	if r == nil {
		return
	}
	key := authKey{Method: normalizeLabel(method, "unknown"), Reason: normalizeLabel(reason, "unknown")}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.auth[key]++
}

func (r *Registry) Prometheus() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	snapshots := make([]rpcSnapshot, 0, len(r.rpc))
	for key, h := range r.rpc {
		snapshots = append(snapshots, rpcSnapshot{
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
	authSnapshots := make([]authSnapshot, 0, len(r.auth))
	for key, count := range r.auth {
		authSnapshots = append(authSnapshots, authSnapshot{key: key, count: count})
	}
	r.mu.Unlock()
	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].key.Method != snapshots[j].key.Method {
			return snapshots[i].key.Method < snapshots[j].key.Method
		}
		return snapshots[i].key.Code < snapshots[j].key.Code
	})
	sort.Slice(panicSnapshots, func(i, j int) bool {
		return panicSnapshots[i].key.Method < panicSnapshots[j].key.Method
	})
	sort.Slice(authSnapshots, func(i, j int) bool {
		if authSnapshots[i].key.Method != authSnapshots[j].key.Method {
			return authSnapshots[i].key.Method < authSnapshots[j].key.Method
		}
		return authSnapshots[i].key.Reason < authSnapshots[j].key.Reason
	})

	var b strings.Builder
	fmt.Fprintf(&b, "# HELP smart_recruit_grpc_server_requests_total gRPC requests handled by %s.\n", r.service)
	b.WriteString("# TYPE smart_recruit_grpc_server_requests_total counter\n")
	for _, s := range snapshots {
		fmt.Fprintf(&b, "smart_recruit_grpc_server_requests_total%s %d\n", rpcLabels(s.key, ""), s.count)
	}
	fmt.Fprintf(&b, "# HELP smart_recruit_grpc_server_request_duration_seconds gRPC server latency for %s.\n", r.service)
	b.WriteString("# TYPE smart_recruit_grpc_server_request_duration_seconds histogram\n")
	for _, s := range snapshots {
		for i, bucket := range defaultBuckets {
			fmt.Fprintf(&b, "smart_recruit_grpc_server_request_duration_seconds_bucket%s %d\n", rpcLabels(s.key, fmt.Sprintf("%g", bucket)), s.buckets[i])
		}
		fmt.Fprintf(&b, "smart_recruit_grpc_server_request_duration_seconds_bucket%s %d\n", rpcLabels(s.key, "+Inf"), s.count)
		fmt.Fprintf(&b, "smart_recruit_grpc_server_request_duration_seconds_sum%s %.9f\n", rpcLabels(s.key, ""), s.sum)
		fmt.Fprintf(&b, "smart_recruit_grpc_server_request_duration_seconds_count%s %d\n", rpcLabels(s.key, ""), s.count)
	}
	b.WriteString("# HELP smart_recruit_grpc_server_panics_total gRPC panics recovered by the logic server.\n")
	b.WriteString("# TYPE smart_recruit_grpc_server_panics_total counter\n")
	for _, s := range panicSnapshots {
		fmt.Fprintf(&b, "smart_recruit_grpc_server_panics_total%s %d\n", panicLabels(s.key), s.count)
	}
	b.WriteString("# HELP smart_recruit_grpc_internal_auth_rejections_total Internal gRPC auth rejections.\n")
	b.WriteString("# TYPE smart_recruit_grpc_internal_auth_rejections_total counter\n")
	for _, s := range authSnapshots {
		fmt.Fprintf(&b, "smart_recruit_grpc_internal_auth_rejections_total%s %d\n", authLabels(s.key), s.count)
	}
	return b.String()
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

type authSnapshot struct {
	key   authKey
	count uint64
}

func rpcLabels(key rpcKey, le string) string {
	labels := []string{
		`service="logic-grpc-service"`,
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
		`service="logic-grpc-service"`,
		fmt.Sprintf(`grpc_method="%s"`, escapeLabel(key.Method)),
	}
	return "{" + strings.Join(labels, ",") + "}"
}

func authLabels(key authKey) string {
	labels := []string{
		`service="logic-grpc-service"`,
		fmt.Sprintf(`grpc_method="%s"`, escapeLabel(key.Method)),
		fmt.Sprintf(`reason="%s"`, escapeLabel(key.Reason)),
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
