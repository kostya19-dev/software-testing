package compare

import (
	"embed"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"time"

	"lab6/internal/bench"
	"lab6/internal/jaeger"
	"lab6/internal/variant"
)

//go:embed page.html
var pageFS embed.FS

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
