// Package grafana turns the results of lab6 runs into data for the local Grafana dashboard:
// OpenMetrics for a backfilled Prometheus and OTLP JSON for Jaeger.
package grafana

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"lab6/internal/bench"
	"lab6/internal/compare"
	"lab6/internal/jaeger"
	"lab6/internal/variant"
)

const (
	MetricsFile = "metrics.om"
	TracesDir   = "traces"
	baseline    = "baseline"
)

// Epoch is the moment every run starts on the dashboard time axis, so variants of a scenario overlay each other.
// Keep it in sync with the time range of grafana/dashboards/lab6.json.
const Epoch = 1767225600.0 // 2026-01-01T00:00:00Z

var resourceMetrics = []struct{ source, name, help string }{
	{"cpu_cores", "lab6_cpu_cores", "CPU контейнера, ядер"},
	{"memory_mb", "lab6_memory_mb", "Память контейнера, МБ"},
	{"disk_write_kbps", "lab6_disk_write_kbps", "Запись на диск, КБ/с"},
	{"net_tx_kbps", "lab6_net_tx_kbps", "Сеть, отправлено КБ/с"},
}

type Result struct {
	Runs    int
	Samples int
	Traces  int
	Dir     string
}

type series struct {
	T []float64  `json:"t"`
	V []*float64 `json:"v"`
}

func Export(resultsDir, outDir string, variants variant.Set) (Result, error) {
	runs, err := compare.LoadRuns(resultsDir)
	if err != nil {
		return Result{}, err
	}
	latest := map[string]map[string]compare.Run{}
	for _, run := range runs {
		if _, err := variants.Get(run.Variant); err != nil {
			continue
		}
		if latest[run.Scenario] == nil {
			latest[run.Scenario] = map[string]compare.Run{}
		}
		latest[run.Scenario][run.Variant] = run
	}
	if len(latest) == 0 {
		return Result{}, fmt.Errorf("в %s нет прогонов: сначала make suite или make run", resultsDir)
	}

	if err := os.RemoveAll(outDir); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(filepath.Join(outDir, TracesDir), 0o755); err != nil {
		return Result{}, err
	}

	m := NewMetrics()
	result := Result{Dir: outDir}
	for scenario, byVariant := range latest {
		for name, run := range byVariant {
			labels := Labels{"scenario": scenario, "variant": name}
			start := loadStart(run)
			if err := exportResources(m, run.Dir, labels, start); err != nil {
				return result, fmt.Errorf("%s: %w", run.Dir, err)
			}
			if err := exportLoad(m, filepath.Join(run.Dir, "load"), labels, start); err != nil && !os.IsNotExist(err) {
				return result, fmt.Errorf("%s: %w", run.Dir, err)
			}
			exportSummary(m, run, labels)
			if run.Traces != nil {
				exportTraceSummary(m, *run.Traces, labels)
				n, err := ConvertTraces(filepath.Join(run.Dir, jaeger.TracesDir, jaeger.TracesFile), filepath.Join(outDir, TracesDir), scenario, name, start)
				if err != nil {
					return result, fmt.Errorf("%s: %w", run.Dir, err)
				}
				result.Traces += n
			}
			result.Runs++
		}
		exportDeltas(m, scenario, byVariant)
	}
	if report, ok := bench.Load(filepath.Join(resultsDir, bench.ResultFile)); ok {
		exportBench(m, report)
	}
	result.Samples = m.Samples()
	return result, m.Write(filepath.Join(outDir, MetricsFile))
}

// loadStart is the moment the load generator started, the zero of the run on the dashboard.
func loadStart(run compare.Run) float64 {
	var meta struct {
		StartUnixMs float64 `json:"start_unix_ms"`
	}
	data, err := os.ReadFile(filepath.Join(run.Dir, "load", "meta.json"))
	if err == nil && json.Unmarshal(data, &meta) == nil && meta.StartUnixMs > 0 {
		return meta.StartUnixMs / 1000
	}
	return math.NaN()
}

func exportResources(m *Metrics, dir string, labels Labels, start float64) error {
	data, err := os.ReadFile(filepath.Join(dir, "resources-timeseries.json"))
	if err != nil {
		return err
	}
	var timeseries map[string]map[string]series
	if err := json.Unmarshal(data, &timeseries); err != nil {
		return err
	}
	if math.IsNaN(start) {
		start = math.Inf(1)
		for _, s := range timeseries["cpu_cores"] {
			if len(s.T) > 0 {
				start = min(start, s.T[0])
			}
		}
	}
	for _, metric := range resourceMetrics {
		for component, s := range timeseries[metric.source] {
			l := labels.With("component", component)
			for i, t := range s.T {
				if i >= len(s.V) || s.V[i] == nil || t < start {
					continue
				}
				m.Add(metric.name, metric.help, l, Epoch+t-start, *s.V[i])
			}
		}
	}
	return nil
}

func exportSummary(m *Metrics, run compare.Run, labels Labels) {
	m.Add("lab6_run_rps", "Средний RPS в окне сравнения", labels, Epoch, run.RPS)
	m.Add("lab6_run_failure_ratio", "Доля отказов в окне сравнения", labels, Epoch, run.Failures)
	for q, v := range map[string]float64{"p50": run.Latency.P50, "p95": run.Latency.P95, "p99": run.Latency.P99} {
		m.Add("lab6_run_latency_ms", "Задержка в окне сравнения, мс", labels.With("q", q), Epoch, v)
	}
	for component := range run.Usage {
		l := labels.With("component", component)
		m.Add("lab6_run_cpu_cores", "CPU в окне сравнения, среднее, ядер", l, Epoch, run.Usage.Get(component, "cpu_cores").Mean)
		m.Add("lab6_run_cpu_max_cores", "CPU в окне сравнения, максимум, ядер", l, Epoch, run.Usage.Get(component, "cpu_cores").Max)
		m.Add("lab6_run_memory_mb", "Память в окне сравнения, среднее, МБ", l, Epoch, run.Usage.Get(component, "memory_mb").Mean)
		m.Add("lab6_run_memory_max_mb", "Память в окне сравнения, максимум, МБ", l, Epoch, run.Usage.Get(component, "memory_mb").Max)
		m.Add("lab6_run_disk_write_kbps", "Запись на диск в окне сравнения, КБ/с", l, Epoch, run.Usage.Get(component, "disk_write_kbps").Mean)
	}
	if c := run.Summary.Capacity; c != nil {
		m.Add("lab6_run_capacity_vus", "Последняя ступень ramp, где выполнено SLO, VU", labels, Epoch, float64(c.StableVUs))
		m.Add("lab6_run_capacity_rps", "RPS на последней ступени ramp с выполненным SLO", labels, Epoch, c.StableRPS)
	}
	if r := run.Summary.Recovery; r != nil && r.Recovered {
		m.Add("lab6_run_recovery_seconds", "Время восстановления после перегрузки, с", labels, Epoch, r.RecoveryTimeS)
	}
}

func exportDeltas(m *Metrics, scenario string, byVariant map[string]compare.Run) {
	base, ok := byVariant[baseline]
	if !ok {
		return
	}
	for name, run := range byVariant {
		if name == baseline {
			continue
		}
		labels := Labels{"scenario": scenario, "variant": name}
		for metric, source := range map[string]string{"cpu": "cpu_cores", "memory": "memory_mb", "disk": "disk_write_kbps"} {
			v, b := run.Usage.Get("api", source).Mean, base.Usage.Get("api", source).Mean
			if b > 0 {
				m.Add("lab6_run_delta_ratio", "Отличие ресурсов api от baseline того же сценария", labels.With("metric", metric), Epoch, v/b-1)
			}
		}
	}
}

func exportTraceSummary(m *Metrics, s jaeger.Summary, labels Labels) {
	m.Add("lab6_trace_count", "Трейсов в выгрузке из Jaeger", labels, Epoch, float64(s.Traces))
	m.Add("lab6_trace_spans_per_trace", "Спанов на трейс", labels, Epoch, s.SpansPerTrace)
	for _, op := range s.Operations {
		l := labels.With("operation", op.Name)
		m.Add("lab6_trace_spans", "Спанов операции в выгрузке", l, Epoch, float64(op.Spans))
		m.Add("lab6_trace_span_mean_ms", "Средняя длительность спана, мс", l, Epoch, op.MeanMs)
		m.Add("lab6_trace_span_p95_ms", "p95 длительности спана, мс", l, Epoch, op.P95Ms)
	}
}

func exportBench(m *Metrics, report bench.Report) {
	var base *bench.Result
	for i := range report.Results {
		if report.Results[i].Variant == baseline {
			base = &report.Results[i]
		}
	}
	for _, r := range report.Results {
		labels := Labels{"variant": r.Variant}
		m.Add("lab6_bench_us_per_op", "Бенчмарк: время на операцию, мкс", labels, Epoch, r.NsPerOp.Mean/1000)
		m.Add("lab6_bench_kb_per_op", "Бенчмарк: память на операцию, КБ", labels, Epoch, r.BytesPerOp.Mean/1024)
		m.Add("lab6_bench_allocs_per_op", "Бенчмарк: аллокаций на операцию", labels, Epoch, r.AllocsPerOp.Mean)
		if base == nil || r.Variant == baseline {
			continue
		}
		for metric, pair := range map[string][2]float64{
			"time":   {r.NsPerOp.Mean, base.NsPerOp.Mean},
			"memory": {r.BytesPerOp.Mean, base.BytesPerOp.Mean},
			"allocs": {r.AllocsPerOp.Mean, base.AllocsPerOp.Mean},
		} {
			if pair[1] > 0 {
				m.Add("lab6_bench_delta_ratio", "Бенчмарк: отличие от baseline", labels.With("metric", metric), Epoch, pair[0]/pair[1]-1)
			}
		}
	}
}
