package compare

import (
	"fmt"
	"path/filepath"
	"slices"

	"lab6/internal/variant"
)

var scenarioOrder = []string{"ramp", "stress", "stable"}

var scenarioTitles = map[string]string{
	"ramp":   "Ступенчатое увеличение нагрузки",
	"stress": "Перегрузка и восстановление",
	"stable": "Фиксированная нагрузка",
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
