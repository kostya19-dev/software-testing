package analysis

import (
	"math"
	"sort"
	"strconv"

	"lab4/internal/load"
	"lab4/internal/samples"
	"lab4/internal/stand"
	"lab4/internal/stats"
)

type Overview struct {
	Window         [2]stats.Num  `json:"window_s"`
	Requests       int           `json:"requests"`
	RPS            stats.Summary `json:"rps"`
	Latency        stats.Summary `json:"latency_ms"`
	LatencyValid   stats.Summary `json:"latency_valid_ms"`
	LatencyInvalid stats.Summary `json:"latency_invalid_ms"`
	FailedRequests int           `json:"failed_requests"`
	FailureRate    stats.Num     `json:"failure_rate"`
	InvalidShare   stats.Num     `json:"invalid_share"`
	StatusCodes    []StatusCount `json:"status_codes"`
}

type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type OperationStats struct {
	Name         string        `json:"name"`
	Latency      stats.Summary `json:"latency_ms"`
	FailureRate  stats.Num     `json:"failure_rate"`
	InvalidShare stats.Num     `json:"invalid_share"`
}

type PhaseStats struct {
	Phase       string    `json:"phase"`
	VUs         int       `json:"vus"`
	RPSMean     stats.Num `json:"rps_mean"`
	RPSMedian   stats.Num `json:"rps_p50"`
	RPSP5       stats.Num `json:"rps_p5"`
	P50         stats.Num `json:"p50_ms"`
	P95         stats.Num `json:"p95_ms"`
	P99         stats.Num `json:"p99_ms"`
	FailureRate stats.Num `json:"failure_rate"`
	APICPU      stats.Num `json:"api_cpu"`
	MongoCPU    stats.Num `json:"mongo_cpu"`
	APIMemoryMB stats.Num `json:"api_memory_mb"`
	WithinSlo   bool      `json:"within_slo"`
}

type Result struct {
	Overall    Overview         `json:"overall"`
	Operations []OperationStats `json:"operations"`
	Phases     []PhaseStats     `json:"phases"`
	Capacity   *Capacity        `json:"capacity,omitempty"`
	Recovery   *Recovery        `json:"recovery,omitempty"`
	Stability  *Stability       `json:"stability,omitempty"`
}

type Input struct {
	Scenario  *load.Scenario
	Phases    []load.Phase
	Samples   *samples.Samples
	Resources stand.Resources
}

func (in Input) End() float64 {
	return math.Max(in.Phases[len(in.Phases)-1].End, in.Samples.Duration())
}

func Analyse(in Input) Result {
	result := Result{
		Overall:    overview(in.Samples, 0, in.End()),
		Operations: operations(in.Samples),
	}
	for _, phase := range in.Phases {
		result.Phases = append(result.Phases, in.phaseStats(phase))
	}
	switch in.Scenario.Kind {
	case load.Steps:
		result.Capacity = capacity(in)
	case load.Spike:
		result.Recovery = recovery(in)
	case load.Constant:
		result.Stability = stability(in)
	}
	return result
}

func overview(s *samples.Samples, start, end float64) Overview {
	window := s.Between(start, end)
	var valid, invalid []float64
	failed := 0
	codes := map[int]int{}
	for i, latency := range window.Latency {
		if window.Invalid[i] {
			invalid = append(invalid, latency)
		} else {
			valid = append(valid, latency)
		}
		if !window.OK[i] {
			failed++
		}
		codes[window.Status[i]]++
	}
	return Overview{
		Window:         [2]stats.Num{stats.Num(start), stats.Num(end)},
		Requests:       window.Len(),
		RPS:            stats.Describe(stats.PerSecond(window.Time, start, end)),
		Latency:        stats.Describe(window.Latency),
		LatencyValid:   stats.Describe(valid),
		LatencyInvalid: stats.Describe(invalid),
		FailedRequests: failed,
		FailureRate:    stats.Num(ratio(failed, window.Len())),
		InvalidShare:   stats.Num(stats.Share(window.Invalid)),
		StatusCodes:    statusCounts(codes),
	}
}

func operations(s *samples.Samples) []OperationStats {
	var result []OperationStats
	for _, name := range s.Operations() {
		subset := s.Where(func(i int) bool { return s.Operation[i] == name })
		failed := subset.Where(func(i int) bool { return !subset.OK[i] }).Len()
		result = append(result, OperationStats{
			Name:         name,
			Latency:      stats.Describe(subset.Latency),
			FailureRate:  stats.Num(ratio(failed, subset.Len())),
			InvalidShare: stats.Num(stats.Share(subset.Invalid)),
		})
	}
	return result
}

func (in Input) phaseStats(phase load.Phase) PhaseStats {
	window := in.Samples.Between(phase.Start, phase.End)
	rps := stats.PerSecond(window.Time, phase.Start, phase.End)
	latency := stats.Describe(window.Latency)
	failed := window.Where(func(i int) bool { return !window.OK[i] }).Len()
	failure := 1.0
	if window.Len() > 0 {
		failure = ratio(failed, window.Len())
	}
	origin := in.Samples.Start
	return PhaseStats{
		Phase:       phase.Name,
		VUs:         phase.VUs,
		RPSMean:     stats.Num(stats.Mean(rps)),
		RPSMedian:   stats.Num(stats.Percentile(rps, 50)),
		RPSP5:       stats.Num(stats.Percentile(rps, 5)),
		P50:         latency.P50,
		P95:         latency.P95,
		P99:         latency.P99,
		FailureRate: stats.Num(failure),
		APICPU:      stats.Num(in.Resources.Mean("cpu_cores", "api", phase.Start, phase.End, origin)),
		MongoCPU:    stats.Num(in.Resources.Mean("cpu_cores", "mongo", phase.Start, phase.End, origin)),
		APIMemoryMB: stats.Num(in.Resources.Mean("memory_mb", "api", phase.Start, phase.End, origin)),
		WithinSlo:   float64(latency.P95) <= in.Scenario.Slo.P95Ms && failure <= in.Scenario.Slo.FailureRate,
	}
}

func statusCounts(codes map[int]int) []StatusCount {
	keys := make([]int, 0, len(codes))
	for code := range codes {
		keys = append(keys, code)
	}
	sort.Ints(keys)
	result := make([]StatusCount, 0, len(keys))
	for _, code := range keys {
		label := strconv.Itoa(code)
		if code == 0 {
			label = "нет ответа"
		}
		result = append(result, StatusCount{Status: label, Count: codes[code]})
	}
	return result
}

func ratio(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total)
}
