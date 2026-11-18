package analysis

import (
	"fmt"
	"math"

	"lab4/internal/load"
	"lab4/internal/stats"
)

const (
	recoveryWindowS       = 5.0
	recoveryConfirmations = 4
	recoveryLatencyFactor = 2.0
	recoveryLatencySlack  = 50.0
	soakSlices            = 5
)

type Capacity struct {
	Steps      []PhaseStats `json:"steps"`
	StableVUs  int          `json:"stable_vus"`
	StableRPS  stats.Num    `json:"stable_rps"`
	StableP95  stats.Num    `json:"stable_p95_ms"`
	BreakingVU int          `json:"breaking_vus"`
	BreakingAt stats.Num    `json:"breaking_at_s"`
	PeakRPS    stats.Num    `json:"peak_rps"`
	PeakRPSVUs int          `json:"peak_rps_vus"`
}

func capacity(in Input) *Capacity {
	result := &Capacity{}
	broken := false
	rampStart := 0.0
	for _, phase := range in.Phases {
		if !phase.Steady {
			rampStart = phase.Start
			continue
		}
		step := in.phaseStats(phase)
		result.Steps = append(result.Steps, step)
		if !step.WithinSlo && !broken {
			broken = true
			result.BreakingVU, result.BreakingAt = step.VUs, stats.Num(rampStart)
		}
		if step.WithinSlo && !broken {
			result.StableVUs, result.StableRPS, result.StableP95 = step.VUs, step.RPSMean, step.P95
		}
		if step.RPSMean > result.PeakRPS {
			result.PeakRPS, result.PeakRPSVUs = step.RPSMean, step.VUs
		}
	}
	return result
}

type RecoveryCriteria struct {
	P95MaxMs       stats.Num `json:"p95_ms_max"`
	FailureRateMax stats.Num `json:"failure_rate_max"`
	ConfirmS       float64   `json:"confirm_s"`
}

type RecoveryPoint struct {
	T           stats.Num `json:"t_s"`
	P95         stats.Num `json:"p95_ms"`
	FailureRate stats.Num `json:"failure_rate"`
}

type Recovery struct {
	Baseline      PhaseStats       `json:"baseline"`
	Overload      PhaseStats       `json:"overload"`
	AfterRecovery PhaseStats       `json:"after_recovery"`
	Criteria      RecoveryCriteria `json:"criteria"`
	OverloadEndS  stats.Num        `json:"overload_end_s"`
	Recovered     bool             `json:"recovered"`
	RecoveryTimeS stats.Num        `json:"recovery_time_s"`
	Timeline      []RecoveryPoint  `json:"recovery_timeline"`
}

func recovery(in Input) *Recovery {
	warmup := in.phase("базовая нагрузка")
	overload := in.phase("перегрузка")
	settle := in.phase("восстановление")

	reference := load.Phase{Name: "эталон", Start: (warmup.Start + warmup.End) / 2, End: warmup.End, VUs: warmup.VUs, Steady: true}
	tail := load.Phase{Name: "после", Start: settle.End - settle.Duration()/3, End: settle.End, VUs: settle.VUs, Steady: true}
	result := &Recovery{
		Baseline:      in.phaseStats(reference),
		Overload:      in.phaseStats(overload),
		AfterRecovery: in.phaseStats(tail),
		OverloadEndS:  stats.Num(overload.End),
		RecoveryTimeS: stats.Num(math.NaN()),
	}
	base := float64(result.Baseline.P95)
	result.Criteria = RecoveryCriteria{
		P95MaxMs:       stats.Num(math.Max(recoveryLatencyFactor*base, base+recoveryLatencySlack)),
		FailureRateMax: stats.Num(math.Max(in.Scenario.Slo.FailureRate, 2*float64(result.Baseline.FailureRate))),
		ConfirmS:       recoveryWindowS * recoveryConfirmations,
	}

	after := in.Samples.Between(overload.End, settle.End)
	healthy := make([]bool, 0)
	for _, w := range stats.Windows(after.Time, overload.End, settle.End, recoveryWindowS) {
		p95 := stats.Percentile(after.Latency[w.From:w.To], 95)
		failure := 1 - stats.Share(after.OK[w.From:w.To])
		result.Timeline = append(result.Timeline, RecoveryPoint{T: stats.Num(w.Center()), P95: stats.Num(p95), FailureRate: stats.Num(failure)})
		healthy = append(healthy, p95 <= float64(result.Criteria.P95MaxMs) && failure <= float64(result.Criteria.FailureRateMax))
	}
	for i := 0; i+recoveryConfirmations <= len(healthy); i++ {
		if allTrue(healthy[i : i+recoveryConfirmations]) {
			result.Recovered = true
			result.RecoveryTimeS = stats.Num(float64(i) * recoveryWindowS)
			break
		}
	}
	return result
}

type Stability struct {
	Steady              PhaseStats           `json:"steady"`
	Slices              []PhaseStats         `json:"slices"`
	P95Drift            stats.Num            `json:"p95_drift"`
	RPSDrift            stats.Num            `json:"rps_drift"`
	MemoryGrowthMBPerH  map[string]stats.Num `json:"memory_growth_mb_per_hour"`
	ConfiguredErrorRate float64              `json:"configured_error_rate"`
}

func stability(in Input) *Stability {
	steady := in.phase("стабильная нагрузка")
	result := &Stability{
		Steady:              in.phaseStats(steady),
		MemoryGrowthMBPerH:  map[string]stats.Num{},
		ConfiguredErrorRate: in.Scenario.ErrorRate,
	}
	width := steady.Duration() / soakSlices
	for i := range soakSlices {
		start := steady.Start + float64(i)*width
		slice := load.Phase{Name: fmt.Sprintf("%d/%d", i+1, soakSlices), Start: start, End: start + width, VUs: steady.VUs, Steady: true}
		result.Slices = append(result.Slices, in.phaseStats(slice))
	}
	first, last := result.Slices[0], result.Slices[soakSlices-1]
	result.P95Drift = stats.Num(float64(last.P95) / float64(first.P95))
	result.RPSDrift = stats.Num(float64(last.RPSMean) / float64(first.RPSMean))
	for _, component := range in.Resources.Components() {
		growth := in.Resources.MemoryGrowthPerHour(component, steady.Start, steady.End, in.Samples.Start)
		result.MemoryGrowthMBPerH[component] = stats.Num(growth)
	}
	return result
}

func allTrue(values []bool) bool {
	for _, v := range values {
		if !v {
			return false
		}
	}
	return true
}

func (in Input) phase(name string) load.Phase {
	for _, phase := range in.Phases {
		if phase.Name == name {
			return phase
		}
	}
	panic(fmt.Sprintf("scenario %s has no phase %q", in.Scenario.Name, name))
}
