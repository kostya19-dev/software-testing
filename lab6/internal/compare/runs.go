package compare

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"lab6/internal/jaeger"
)

const steadyPhase = "стабильная нагрузка"

type Stat struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	P95    float64 `json:"p95"`
}

type Usage map[string]map[string]Stat

type latency struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

type phase struct {
	Phase       string  `json:"phase"`
	RPSMean     float64 `json:"rps_mean"`
	P50         float64 `json:"p50_ms"`
	P95         float64 `json:"p95_ms"`
	P99         float64 `json:"p99_ms"`
	FailureRate float64 `json:"failure_rate"`
}

type summary struct {
	Overall struct {
		Requests    int     `json:"requests"`
		RPS         Stat    `json:"rps"`
		Latency     latency `json:"latency_ms"`
		FailureRate float64 `json:"failure_rate"`
	} `json:"overall"`
	Phases   []phase `json:"phases"`
	Capacity *struct {
		StableVUs  int     `json:"stable_vus"`
		StableRPS  float64 `json:"stable_rps"`
		BreakingVU int     `json:"breaking_vus"`
	} `json:"capacity"`
	Recovery *struct {
		Recovered     bool    `json:"recovered"`
		RecoveryTimeS float64 `json:"recovery_time_s"`
	} `json:"recovery"`
}

type resources struct {
	Overall Usage            `json:"overall"`
	Steady  map[string]Usage `json:"steady_phases"`
}

type manifest struct {
	ID       string  `json:"id"`
	Scenario string  `json:"scenario"`
	Scale    float64 `json:"scale"`
	Variant  string  `json:"variant"`
}

type Run struct {
	manifest
	Dir      string
	Window   string
	Requests int
	RPS      float64
	Latency  latency
	Failures float64
	Usage    Usage
	Summary  summary
	Traces   *jaeger.Summary
}

func LoadRuns(resultsDir string) ([]Run, error) {
	manifests, err := filepath.Glob(filepath.Join(resultsDir, "*", "*", "summary.json"))
	if err != nil {
		return nil, err
	}
	var runs []Run
	for _, path := range manifests {
		run, err := loadRun(filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	slices.SortFunc(runs, func(a, b Run) int { return strings.Compare(a.ID, b.ID) })
	return runs, nil
}

func loadRun(dir string) (Run, error) {
	run := Run{Dir: dir}
	if err := readJSON(filepath.Join(dir, "manifest.json"), &run.manifest); err != nil {
		return run, err
	}
	if err := readJSON(filepath.Join(dir, "summary.json"), &run.Summary); err != nil {
		return run, err
	}
	var usage resources
	if err := readJSON(filepath.Join(dir, "resources.json"), &usage); err != nil {
		return run, err
	}

	overall := run.Summary.Overall
	run.Window, run.Requests, run.RPS, run.Latency, run.Failures, run.Usage =
		"весь прогон", overall.Requests, overall.RPS.Mean, overall.Latency, overall.FailureRate, usage.Overall
	for _, p := range run.Summary.Phases {
		if p.Phase == steadyPhase && usage.Steady[steadyPhase] != nil {
			run.Window, run.RPS, run.Failures, run.Usage = steadyPhase, p.RPSMean, p.FailureRate, usage.Steady[steadyPhase]
			run.Latency = latency{P50: p.P50, P95: p.P95, P99: p.P99}
		}
	}
	if traces, ok := jaeger.LoadSummary(dir); ok {
		run.Traces = &traces
	}
	return run, nil
}

func (u Usage) Get(component, metric string) Stat {
	return u[component][metric]
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
