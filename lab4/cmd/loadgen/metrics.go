package main

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"

	"lab4/internal/clinic"
)

var latencyBuckets = []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 1.5, 2.5, 5, 10}

type key struct {
	operation clinic.Operation
	outcome   string
}

type liveMetrics struct {
	mu       sync.Mutex
	requests map[key]uint64
	buckets  []uint64
	sum      float64
	count    uint64
	vus      func() int
}

func newLiveMetrics(vus func() int) *liveMetrics {
	return &liveMetrics{requests: map[key]uint64{}, buckets: make([]uint64, len(latencyBuckets)), vus: vus}
}

func (t *liveMetrics) Observe(r clinic.Result) {
	seconds := r.Latency.Seconds()
	t.mu.Lock()
	defer t.mu.Unlock()
	t.requests[key{r.Operation, outcome(r)}]++
	t.sum += seconds
	t.count++
	for i, bound := range latencyBuckets {
		if seconds <= bound {
			t.buckets[i]++
		}
	}
}

func (t *liveMetrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprint(w, t.render())
}

func (t *liveMetrics) render() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	var b strings.Builder
	b.WriteString("# HELP lab4_load_vus Active virtual users\n# TYPE lab4_load_vus gauge\n")
	fmt.Fprintf(&b, "lab4_load_vus %d\n", t.vus())

	b.WriteString("# HELP lab4_load_requests_total Completed requests by operation and outcome\n# TYPE lab4_load_requests_total counter\n")
	keys := make([]key, 0, len(t.requests))
	for k := range t.requests {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b key) int {
		return cmp.Or(cmp.Compare(a.operation, b.operation), cmp.Compare(a.outcome, b.outcome))
	})
	for _, k := range keys {
		fmt.Fprintf(&b, "lab4_load_requests_total{operation=%q,outcome=%q} %d\n", k.operation, k.outcome, t.requests[k])
	}

	b.WriteString("# HELP lab4_load_request_duration_seconds Request latency\n# TYPE lab4_load_request_duration_seconds histogram\n")
	for i, bound := range latencyBuckets {
		fmt.Fprintf(&b, "lab4_load_request_duration_seconds_bucket{le=\"%g\"} %d\n", bound, t.buckets[i])
	}
	fmt.Fprintf(&b, "lab4_load_request_duration_seconds_bucket{le=\"+Inf\"} %d\n", t.count)
	fmt.Fprintf(&b, "lab4_load_request_duration_seconds_sum %g\n", t.sum)
	fmt.Fprintf(&b, "lab4_load_request_duration_seconds_count %d\n", t.count)
	return b.String()
}

func outcome(r clinic.Result) string {
	switch {
	case !r.Expected:
		return "failed"
	case r.Invalid:
		return "user_error"
	default:
		return "ok"
	}
}
