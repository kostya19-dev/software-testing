package main

import (
	"runtime"
	"sort"
	"time"
)

type Phase struct {
	Operations int   `json:"operations"`
	ElapsedNS  int64 `json:"elapsed_ns"`
}
type Sample struct {
	AtNS      int64 `json:"at_ns"`
	LatencyNS int64 `json:"latency_ns"`
}
type Result struct {
	Codec        string   `json:"codec"`
	Scenario     string   `json:"scenario"`
	Size         int      `json:"size"`
	Bytes        int      `json:"bytes"`
	Order        int      `json:"order"`
	Warmup       Phase    `json:"warmup"`
	Batch        Phase    `json:"batch"`
	Requests     Phase    `json:"requests"`
	Teardown     Phase    `json:"teardown"`
	BatchNSPerOp float64  `json:"batch_ns_per_op"`
	Samples      []Sample `json:"samples"`
	TotalNS      int64    `json:"total_ns"`
}

//go:noinline
func empty() {}
func batch(fn func(), n int) Phase {
	start := time.Now()
	for i := 0; i < n; i++ {
		fn()
	}
	return Phase{n, time.Since(start).Nanoseconds()}
}
func measure(fn func(), n, samples int, epoch time.Time) Result {
	start := time.Now()
	// Collection before warmup. Normal GC stays enabled during measurement.
	runtime.GC()
	warmup := batch(fn, n)
	main := batch(fn, n)
	values := make([]Sample, samples)
	requestsStart := time.Now()
	for i := range values {
		t := time.Now()
		fn()
		elapsed := time.Since(t).Nanoseconds()
		values[i] = Sample{t.Sub(epoch).Nanoseconds(), elapsed}
	}
	requests := Phase{samples, time.Since(requestsStart).Nanoseconds()}
	teardown := batch(fn, n)
	return Result{Warmup: warmup, Batch: main, Requests: requests, Teardown: teardown,
		BatchNSPerOp: float64(main.ElapsedNS) / float64(n), Samples: values, TotalNS: time.Since(start).Nanoseconds()}
}

type Overhead struct {
	BatchNSPerOp       float64   `json:"batch_ns_per_op"`
	BatchRepeats       []float64 `json:"batch_repeats_ns_per_op"`
	TimerMeanNS        float64   `json:"timer_mean_ns"`
	TimerLoopNSPerOp   float64   `json:"timer_loop_ns_per_op"`
	TimerMinPositiveNS int64     `json:"timer_min_positive_ns"`
	TimerZeroSamples   int       `json:"timer_zero_samples"`
	TimerP50NS         int64     `json:"timer_p50_ns"`
	TimerP99NS         int64     `json:"timer_p99_ns"`
	Operations         int       `json:"operations"`
	ElapsedNS          int64     `json:"elapsed_ns"`
}

func overhead() Overhead {
	start := time.Now()
	const n = 1000000
	const repeats = 9
	batch(empty, n)
	values := make([]float64, repeats)
	for i := range values {
		values[i] = float64(batch(empty, n).ElapsedNS) / n
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	timer := make([]int64, 10000)
	timerStart := time.Now()
	for i := range timer {
		t := time.Now()
		empty()
		timer[i] = time.Since(t).Nanoseconds()
	}
	timerLoop := float64(time.Since(timerStart).Nanoseconds()) / float64(len(timer))
	var total, minPositive int64
	zeros := 0
	for _, v := range timer {
		total += v
		if v == 0 {
			zeros++
		}
		if v > 0 && (minPositive == 0 || v < minPositive) {
			minPositive = v
		}
	}
	sort.Slice(timer, func(i, j int) bool { return timer[i] < timer[j] })
	return Overhead{sorted[repeats/2], values, float64(total) / float64(len(timer)), timerLoop, minPositive, zeros, timer[len(timer)/2-1], timer[len(timer)*99/100-1], n*(repeats+1) + len(timer), time.Since(start).Nanoseconds()}
}
