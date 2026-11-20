package report

import (
	"fmt"
	"math"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"

	"lab4/internal/analysis"
	"lab4/internal/stats"
)

func (b *chartBook) capacity(result *analysis.Capacity) error {
	throughput := newPlot("Кривая производительности по ступеням нагрузки", "Число виртуальных пользователей", "Запросов в секунду")
	latency := newPlot("", "Число виртуальных пользователей", "Задержка, мс")
	logY(latency)

	var rps, p95, p99 plotter.XYs
	var failures plotter.XYs
	var failureTexts []string
	for _, step := range result.Steps {
		x := float64(step.VUs)
		rps = append(rps, plotter.XY{X: x, Y: float64(step.RPSMean)})
		p95 = append(p95, plotter.XY{X: x, Y: math.Max(nanToZero(float64(step.P95)), 0.1)})
		p99 = append(p99, plotter.XY{X: x, Y: math.Max(nanToZero(float64(step.P99)), 0.1)})
		if step.FailureRate > 0 {
			failures = append(failures, plotter.XY{X: x, Y: float64(step.RPSMean)})
			failureTexts = append(failureTexts, fmt.Sprintf("отказы %.1f%%", float64(step.FailureRate)*100))
		}
	}
	addLine(throughput, rps, blue, 1.6, "RPS (среднее на ступени)", false)
	addMarkers(throughput, rps, blue, "")
	if result.StableVUs > 0 {
		addHorizontal(throughput, float64(result.StableRPS), orange, fmt.Sprintf("предел: %s RPS", formatNumber(result.StableRPS)))
	}
	if len(failures) > 0 {
		addLabels(throughput, failures, failureTexts, red, above())
	}
	addLine(latency, p95, orange, 1.6, "p95", false)
	addMarkers(latency, p95, orange, "")
	addLine(latency, p99, red, 1.4, "p99", true)
	addHorizontal(latency, b.slo.P95Ms, green, "SLO p95")

	for _, p := range []*plot.Plot{throughput, latency} {
		if result.StableVUs > 0 {
			addMarker(p, float64(result.StableVUs), green, true, "стабильная работа")
		}
		if result.BreakingVU > 0 {
			addMarker(p, float64(result.BreakingVU), red, true, "нарушение SLO")
		}
		p.X.Min, p.X.Max = 0, p.X.Max*1.1
	}
	return b.save("capacity", "Кривая производительности", throughput, latency)
}

func (b *chartBook) limit(result *analysis.Capacity) error {
	latency := newPlot("Предел производительности во времени", timeAxis, "Задержка, мс")
	b.shade(latency)
	windows := stats.Windows(b.samples.Time, 0, b.end, 2.5*b.window())
	for _, q := range []float64{50, 95, 99} {
		xs, ys := make([]float64, len(windows)), make([]float64, len(windows))
		for i, w := range windows {
			xs[i], ys[i] = w.Center(), stats.Percentile(b.samples.Latency[w.From:w.To], q)
		}
		addLine(latency, points(xs, ys, 0), percentileColors[q], 1.2, fmt.Sprintf("p%g", q), false)
	}
	addHorizontal(latency, b.slo.P95Ms, green, fmt.Sprintf("допустимый p95 = %g мс", b.slo.P95Ms))
	latency.Y.Min = 0

	rps := stats.PerSecond(b.samples.Time, 0, b.end)
	succeeded := stats.PerSecond(b.samples.TimesWhere(func(i int) bool { return b.samples.OK[i] }), 0, b.end)
	failed := stats.PerSecond(b.samples.TimesWhere(func(i int) bool { return !b.samples.OK[i] }), 0, b.end)
	seconds := make([]float64, len(rps))
	for i := range seconds {
		seconds[i] = float64(i) + 0.5
	}
	throughput := newPlot("", timeAxis, "Запросов в секунду")
	b.shade(throughput)
	addLine(throughput, points(seconds, stats.Smooth(rps, 10), 0), navy, 1.3, "все ответы", false)
	addLine(throughput, points(seconds, stats.Smooth(succeeded, 10), 0), green, 1.3, "успешные ответы", false)
	addLine(throughput, points(seconds, stats.Smooth(failed, 10), 0), red, 1.3, "отказы", false)
	throughput.Y.Min = 0

	users := b.vusPanel()
	var plateaus plotter.XYs
	var levels []string
	for _, phase := range b.phases {
		if phase.Steady {
			plateaus = append(plateaus, plotter.XY{X: phase.Start, Y: float64(phase.VUs)})
			levels = append(levels, fmt.Sprintf("%d", phase.VUs))
		}
	}
	addLabels(users, plateaus, levels, darkGray, above())

	if result.StableVUs > 0 {
		addHorizontal(throughput, float64(result.StableRPS), orange, fmt.Sprintf("предел: %s RPS", formatNumber(result.StableRPS)))
		addHorizontal(users, float64(result.StableVUs), orange, fmt.Sprintf("предел: %d пользователей", result.StableVUs))
	}
	if result.BreakingVU > 0 {
		label := "предел"
		for i, p := range []*plot.Plot{latency, throughput, users} {
			if i > 0 {
				label = ""
			}
			addMarker(p, float64(result.BreakingAt), red, true, label)
		}
	}
	return b.save("limit", "Предел производительности во времени", latency, throughput, users)
}

func (b *chartBook) recovery(result *analysis.Recovery) error {
	p := newPlot("Восстановление после перегрузки", timeAxis, "p95, мс")
	logY(p)
	b.shade(p)
	windows := stats.Windows(b.samples.Time, 0, b.end, b.window())
	xs, ys := make([]float64, len(windows)), make([]float64, len(windows))
	for i, w := range windows {
		xs[i], ys[i] = w.Center(), stats.Percentile(b.samples.Latency[w.From:w.To], 95)
	}
	addLine(p, points(xs, ys, 0.1), hex(0xadb5bd), 1, "p95 за весь тест", false)

	var after plotter.XYs
	for _, point := range result.Timeline {
		if !math.IsNaN(float64(point.P95)) {
			after = append(after, plotter.XY{X: float64(point.T), Y: math.Max(float64(point.P95), 0.1)})
		}
	}
	addLine(p, after, orange, 1.8, "p95 после перегрузки (окно 5 с)", false)
	addHorizontal(p, float64(result.Criteria.P95MaxMs), green, "порог восстановления")
	addMarker(p, float64(result.OverloadEndS), red, true, "конец перегрузки")
	if result.Recovered {
		moment := float64(result.OverloadEndS + result.RecoveryTimeS)
		addMarker(p, moment, green, false, fmt.Sprintf("восстановление через %.0f с", float64(result.RecoveryTimeS)))
	}
	return b.save("recovery", "Восстановление после перегрузки", p)
}
