package compare

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SummaryFile is the Markdown version of the comparison, appended to the CI job summary.
const SummaryFile = "summary.md"

func (p Page) writeMarkdown(resultsDir string) (string, error) {
	var b strings.Builder
	b.WriteString("## Сравнение ресурсов по вариантам\n\n")
	b.WriteString("CPU — в ядрах (1,0 = одно ядро), RAM — в МБ, запись — в КБ/с. В скобках — отличие от baseline того же сценария. ")
	b.WriteString("Нагрузку даёт k6 (`k6/clinic.js`). Мониторинг — Prometheus и экспортер метрик Docker. Полный отчёт с графиками — `results/index.html` в артефакте.\n\n")

	for _, s := range p.Scenarios {
		fmt.Fprintf(&b, "### %s (`%s`, масштаб %g, окно: %s)\n\n", s.Title, s.Name, s.Scale, s.Window)
		b.WriteString("| Вариант | RPS | p95, мс | Отказы | CPU api | RAM api | Запись api | CPU / RAM Jaeger | CPU / RAM мониторинга | Итог |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
		for _, r := range s.Rows {
			fmt.Fprintf(&b, "| %s | %s | %s | %.2f%% | %s | %s | %s | %s | %s / %s | %s |\n",
				r.Variant.Name, formatNumber(r.RPS), formatNumber(r.Latency.P95), r.Failures*100,
				withDelta(r.APICPU.Mean, r.CPUDelta), withDelta(r.APIRAM.Mean, r.RAMDelta), withDelta(r.APIDisk.Mean, r.DiskDelta),
				jaegerCell(r), formatNumber(r.Monitoring), formatNumber(r.MonitoringRAM), r.Outcome)
		}
		b.WriteString("\n")
	}

	if p.Bench != nil {
		b.WriteString("### Стоимость трассировки и логирования внутри процесса\n\n")
		fmt.Fprintf(&b, "Go-бенчмарк `BenchmarkObservabilityOverhead`: одна операция — 4 HTTP-запроса. Повторов: %d, итераций: %s, CPU: %s.\n\n",
			p.Bench.Count, p.Bench.Benchtime, p.Bench.CPU)
		b.WriteString("| Вариант | Время, мкс/оп | Память, КБ/оп | Аллокаций/оп |\n")
		b.WriteString("| --- | ---: | ---: | ---: |\n")
		for _, r := range p.BenchRows {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", r.Variant,
				withDelta(r.NsPerOp.Mean/1000, r.TimeDelta), withDelta(r.BytesPerOp.Mean/1024, r.BytesDelta), withDelta(r.AllocsPerOp.Mean, r.AllocsDelta))
		}
		b.WriteString("\n")
	}

	path := filepath.Join(resultsDir, SummaryFile)
	return path, os.WriteFile(path, []byte(b.String()), 0o644)
}

func withDelta(value float64, delta string) string {
	if delta == "" {
		return formatNumber(value)
	}
	return fmt.Sprintf("%s (%s)", formatNumber(value), delta)
}

func jaegerCell(r Row) string {
	if r.Tracing.Mean == 0 && r.TracingRAM.Max == 0 {
		return "—"
	}
	return fmt.Sprintf("%s / %s", formatNumber(r.Tracing.Mean), formatNumber(r.TracingRAM.Max))
}
