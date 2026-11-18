package compare

import (
	"embed"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"slices"
	"time"

	"lab6/internal/bench"
	"lab6/internal/jaeger"
	"lab6/internal/variant"
)

//go:embed page.html
var pageFS embed.FS

var scenarioOrder = []string{"ramp", "stress", "stable"}

var scenarioTitles = map[string]string{
	"ramp":   "Ступенчатое увеличение нагрузки",
	"stress": "Перегрузка и восстановление",
	"stable": "Фиксированная нагрузка",
}

type Row struct {
	Variant       variant.Variant
	Report        string
	RPS           float64
	Latency       latency
	Failures      float64
	APICPU        Stat
	APIRAM        Stat
	APIDisk       Stat
	MongoCPU      Stat
	MongoRAM      Stat
	Tracing       Stat
	TracingRAM    Stat
	Monitoring    float64
	MonitoringRAM float64
	CPUDelta      string
	RAMDelta      string
	DiskDelta     string
	Outcome       string
}

type Scenario struct {
	Name   string
	Title  string
	Window string
	Scale  float64
	Rows   []Row
	Chart  string
}

type BenchRow struct {
	bench.Result
	TimeDelta   string
	BytesDelta  string
	AllocsDelta string
}

type TraceRun struct {
	Scenario string
	Report   string
	Summary  jaeger.Summary
}

type Page struct {
	Generated string
	Variants  variant.Set
	Scenarios []Scenario
	Bench     *bench.Report
	BenchRows []BenchRow
	Traces    []TraceRun
}

func Build(resultsDir string, variants variant.Set) (string, error) {
	runs, err := LoadRuns(resultsDir)
	if err != nil {
		return "", err
	}
	page := Page{Generated: time.Now().Format("02.01.2006 15:04"), Variants: variants}
	latest := latestRuns(runs)
	for _, name := range scenarioNames(latest) {
		block, err := scenarioBlock(resultsDir, name, latest[name], variants)
		if err != nil {
			return "", err
		}
		page.Scenarios = append(page.Scenarios, block)
	}
	for _, name := range scenarioNames(latest) {
		for _, run := range latest[name] {
			if run.Traces != nil {
				page.Traces = append(page.Traces, TraceRun{Scenario: name, Report: relative(resultsDir, run.Dir), Summary: *run.Traces})
			}
		}
	}
	if report, ok := bench.Load(filepath.Join(resultsDir, bench.ResultFile)); ok {
		page.Bench, page.BenchRows = &report, benchRows(report)
	}
	return page.write(resultsDir)
}

func latestRuns(runs []Run) map[string]map[string]Run {
	latest := map[string]map[string]Run{}
	for _, run := range runs {
		if latest[run.Scenario] == nil {
			latest[run.Scenario] = map[string]Run{}
		}
		latest[run.Scenario][run.Variant] = run
	}
	return latest
}

func scenarioNames(latest map[string]map[string]Run) []string {
	var names []string
	for _, name := range scenarioOrder {
		if latest[name] != nil {
			names = append(names, name)
		}
	}
	for name := range latest {
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}

func scenarioBlock(resultsDir, name string, runs map[string]Run, variants variant.Set) (Scenario, error) {
	block := Scenario{Name: name, Title: scenarioTitles[name]}
	baseline, hasBaseline := runs["baseline"]
	for _, v := range variants {
		run, ok := runs[v.Name]
		if !ok {
			continue
		}
		block.Window, block.Scale = run.Window, run.Scale
		row := Row{
			Variant:       v,
			Report:        relative(resultsDir, filepath.Join(run.Dir, "report.html")),
			RPS:           run.RPS,
			Latency:       run.Latency,
			Failures:      run.Failures,
			APICPU:        run.Usage.Get("api", "cpu_cores"),
			APIRAM:        run.Usage.Get("api", "memory_mb"),
			APIDisk:       run.Usage.Get("api", "disk_write_kbps"),
			MongoCPU:      run.Usage.Get("mongo", "cpu_cores"),
			MongoRAM:      run.Usage.Get("mongo", "memory_mb"),
			Tracing:       run.Usage.Get("jaeger", "cpu_cores"),
			TracingRAM:    run.Usage.Get("jaeger", "memory_mb"),
			Monitoring:    run.Usage.Get("prometheus", "cpu_cores").Mean + run.Usage.Get("exporter", "cpu_cores").Mean,
			MonitoringRAM: run.Usage.Get("prometheus", "memory_mb").Max + run.Usage.Get("exporter", "memory_mb").Max,
			Outcome:       outcome(run),
		}
		if hasBaseline && v.Name != "baseline" {
			row.CPUDelta = delta(row.APICPU.Mean, baseline.Usage.Get("api", "cpu_cores").Mean)
			row.RAMDelta = delta(row.APIRAM.Mean, baseline.Usage.Get("api", "memory_mb").Mean)
			row.DiskDelta = delta(row.APIDisk.Mean, baseline.Usage.Get("api", "disk_write_kbps").Mean)
		}
		block.Rows = append(block.Rows, row)
	}
	chart := filepath.Join(resultsDir, "charts", name+".png")
	if err := renderUsage(chart, block); err != nil {
		return block, err
	}
	block.Chart = relative(resultsDir, chart)
	return block, nil
}

func outcome(run Run) string {
	switch {
	case run.Summary.Capacity != nil:
		c := run.Summary.Capacity
		if c.BreakingVU == 0 {
			return fmt.Sprintf("SLO выполнено на всех ступенях, до %d VU / %.0f RPS", c.StableVUs, c.StableRPS)
		}
		return fmt.Sprintf("предел %d VU / %.0f RPS, нарушение SLO с %d VU", c.StableVUs, c.StableRPS, c.BreakingVU)
	case run.Summary.Recovery != nil && run.Summary.Recovery.Recovered:
		return fmt.Sprintf("восстановление за %.0f с", run.Summary.Recovery.RecoveryTimeS)
	case run.Summary.Recovery != nil:
		return "не восстановилась"
	}
	return fmt.Sprintf("%d запросов", run.Requests)
}

func benchRows(report bench.Report) []BenchRow {
	var baseline *bench.Result
	for i := range report.Results {
		if report.Results[i].Variant == "baseline" {
			baseline = &report.Results[i]
		}
	}
	rows := make([]BenchRow, len(report.Results))
	for i, result := range report.Results {
		rows[i] = BenchRow{Result: result}
		if baseline != nil && result.Variant != "baseline" {
			rows[i].TimeDelta = delta(result.NsPerOp.Mean, baseline.NsPerOp.Mean)
			rows[i].BytesDelta = delta(result.BytesPerOp.Mean, baseline.BytesPerOp.Mean)
			rows[i].AllocsDelta = delta(result.AllocsPerOp.Mean, baseline.AllocsPerOp.Mean)
		}
	}
	return rows
}

func delta(value, baseline float64) string {
	if baseline == 0 || math.IsNaN(baseline) {
		return ""
	}
	return fmt.Sprintf("%+.0f%%", (value/baseline-1)*100)
}

func relative(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func (p Page) write(resultsDir string) (string, error) {
	tmpl, err := template.New("page.html").Funcs(template.FuncMap{
		"num": func(v float64) string { return formatNumber(v) },
		"pct": func(v float64) string { return fmt.Sprintf("%.2f%%", v*100) },
		"us":  func(ns float64) string { return formatNumber(ns / 1000) },
		"kb":  func(b float64) string { return formatNumber(b / 1024) },
	}).ParseFS(pageFS, "page.html")
	if err != nil {
		return "", err
	}
	path := filepath.Join(resultsDir, "index.html")
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return path, tmpl.Execute(file, p)
}

func formatNumber(v float64) string {
	switch {
	case math.IsNaN(v):
		return "—"
	case math.Abs(v) >= 100:
		return fmt.Sprintf("%.0f", v)
	case math.Abs(v) >= 10:
		return fmt.Sprintf("%.1f", v)
	}
	return fmt.Sprintf("%.2f", v)
}
