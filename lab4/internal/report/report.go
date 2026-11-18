package report

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"

	"lab4/internal/analysis"
	"lab4/internal/load"
	"lab4/internal/samples"
	"lab4/internal/stand"
	"lab4/internal/stats"
)

//go:embed templates/*.html
var templateFiles embed.FS

const resourcesFile = "resources-timeseries.json"

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"num":     formatNumber,
	"pct":     func(v stats.Num) string { return formatPercent(float64(v), 2) },
	"title":   stand.MetricTitle,
	"columns": func() []float64 { return stats.Percentiles },
	"pick":    func(s stats.Summary, q float64) stats.Num { return s.Percentile(q) },
	"named":   func(name string, s stats.Summary) namedSummary { return namedSummary{name, s} },
}).ParseFS(templateFiles, "templates/*.html"))

type namedSummary struct {
	Name    string
	Summary stats.Summary
}

type Summary struct {
	Run      *stand.Run     `json:"run"`
	Scenario *load.Scenario `json:"scenario"`
	Slo      load.Slo       `json:"slo"`
	analysis.Result
}

type ResourceReport struct {
	WindowS      [2]stats.Num             `json:"window_s"`
	Overall      stand.Summary            `json:"overall"`
	SteadyPhases map[string]stand.Summary `json:"steady_phases"`
}

type page struct {
	Run       *stand.Run
	Scenario  *load.Scenario
	Summary   Summary
	Resources ResourceReport
	Charts    map[string]Chart
	Verdict   string
	Rule      string
}

func Build(catalog *load.Catalog, r *stand.Run) (string, error) {
	plan, err := catalog.Get(r.Scenario)
	if err != nil {
		return "", err
	}
	data, err := samples.Read(r.LoadDir())
	if err != nil {
		return "", err
	}
	usage, err := stand.LoadResources(r.Path(resourcesFile))
	if err != nil {
		return "", err
	}

	in := analysis.Input{Scenario: plan, Phases: plan.Phases(r.Scale), Samples: data, Resources: usage}
	result := analysis.Analyse(in)
	summary := Summary{Run: r, Scenario: plan, Slo: plan.Slo, Result: result}
	usageReport := resourceReport(in)

	chartsDir := r.Path("charts")
	if err := os.MkdirAll(chartsDir, 0o755); err != nil {
		return "", err
	}
	charts, err := renderCharts(chartsDir, in, result)
	if err != nil {
		return "", err
	}
	if err := writeJSON(r.Path("summary.json"), summary); err != nil {
		return "", err
	}
	if err := writeJSON(r.Path("resources.json"), usageReport); err != nil {
		return "", err
	}

	target := r.Path("report.html")
	file, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer file.Close()
	err = templates.ExecuteTemplate(file, "report.html", page{
		Run:       r,
		Scenario:  plan,
		Summary:   summary,
		Resources: usageReport,
		Charts:    charts,
		Verdict:   verdict(summary),
		Rule:      fmt.Sprintf("SLO: p95 ≤ %g мс и доля отказов ≤ %s.", plan.Slo.P95Ms, formatPercent(plan.Slo.FailureRate, 0)),
	})
	return target, err
}

func ResourcesPath(r *stand.Run) string {
	return r.Path(resourcesFile)
}

func resourceReport(in analysis.Input) ResourceReport {
	end := in.End()
	report := ResourceReport{
		WindowS:      [2]stats.Num{0, stats.Num(end)},
		Overall:      in.Resources.Summarize(0, end, in.Samples.Start),
		SteadyPhases: map[string]stand.Summary{},
	}
	for _, phase := range in.Phases {
		if phase.Steady {
			report.SteadyPhases[phase.Name] = in.Resources.Summarize(phase.Start, phase.End, in.Samples.Start)
		}
	}
	return report
}

func formatNumber(value any) string {
	var number float64
	switch v := value.(type) {
	case stats.Num:
		number = float64(v)
	case float64:
		number = v
	case int:
		return fmt.Sprint(v)
	default:
		return fmt.Sprint(v)
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return "—"
	}
	switch {
	case math.Abs(number) >= 100:
		return fmt.Sprintf("%.0f", number)
	case math.Abs(number) >= 10:
		return fmt.Sprintf("%.1f", number)
	default:
		return fmt.Sprintf("%.2f", number)
	}
}

func formatPercent(value float64, digits int) string {
	if math.IsNaN(value) {
		return "—"
	}
	return fmt.Sprintf("%.*f%%", digits, value*100)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
