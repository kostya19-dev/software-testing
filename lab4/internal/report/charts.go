package report

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	"lab4/internal/analysis"
	"lab4/internal/load"
	"lab4/internal/samples"
	"lab4/internal/stand"
	"lab4/internal/stats"
)

const timeAxis = "Время от начала теста, с"

type Chart struct {
	File  string
	Title string
}

type chartBook struct {
	dir       string
	samples   *samples.Samples
	phases    []load.Phase
	resources stand.Resources
	end       float64
	slo       load.Slo
	charts    map[string]Chart
}

func renderCharts(dir string, in analysis.Input, result analysis.Result) (map[string]Chart, error) {
	book := &chartBook{dir: dir, samples: in.Samples, phases: in.Phases, resources: in.Resources, end: in.End(), slo: in.Scenario.Slo, charts: map[string]Chart{}}
	renderers := []func() error{
		book.latencyTimeline,
		book.latencyPercentiles,
		book.latencyHistogram,
		book.operations,
		book.throughputTimeline,
		book.throughputDistribution,
		book.resourceTimeline,
	}
	if result.Capacity != nil {
		renderers = append(renderers,
			func() error { return book.limit(result.Capacity) },
			func() error { return book.capacity(result.Capacity) },
		)
	}
	if result.Recovery != nil {
		renderers = append(renderers, func() error { return book.recovery(result.Recovery) })
	}
	for _, render := range renderers {
		if err := render(); err != nil {
			return nil, err
		}
	}
	return book.charts, nil
}

func (b *chartBook) latencyTimeline() error {
	width := b.window()
	p := newPlot(fmt.Sprintf("Задержка запросов во времени (окно %g с)", width), timeAxis, "Задержка, мс")
	logY(p)
	b.shade(p)
	windows := stats.Windows(b.samples.Time, 0, b.end, width)
	for _, q := range []float64{50, 90, 95, 99} {
		xs, ys := make([]float64, len(windows)), make([]float64, len(windows))
		for i, w := range windows {
			xs[i], ys[i] = w.Center(), stats.Percentile(b.samples.Latency[w.From:w.To], q)
		}
		addLine(p, points(xs, ys, 0.1), percentileColors[q], 1.2, fmt.Sprintf("p%g", q), false)
	}
	return b.save("latency_timeline", "Задержка во времени", p, b.vusPanel())
}

func (b *chartBook) latencyPercentiles() error {
	sorted := slices.Clone(b.samples.Latency)
	slices.Sort(sorted)
	p := newPlot(fmt.Sprintf("Распределение задержки по перцентилям (%d запросов)", len(sorted)), "Доля запросов с задержкой не выше указанной, %", "Задержка, мс")
	logY(p)
	curve := make(plotter.XYs, 0, 2000)
	step := max(1, len(sorted)/2000)
	for i := 0; i < len(sorted); i += step {
		curve = append(curve, plotter.XY{X: percentileAxis(float64(i+1) / float64(len(sorted)) * 100), Y: math.Max(sorted[i], 0.1)})
	}
	addLine(p, curve, blue, 1.5, "", false)
	var lower, upper plotter.XYs
	var lowerTexts, upperTexts []string
	for i, q := range stats.Percentiles {
		value := stats.SortedPercentile(sorted, q)
		mark := plotter.XY{X: percentileAxis(q), Y: math.Max(value, 0.1)}
		text := fmt.Sprintf("p%g = %.1f мс", q, value)
		addMarkers(p, plotter.XYs{mark}, red, "")
		if i%2 == 0 {
			lower, lowerTexts = append(lower, mark), append(lowerTexts, text)
		} else {
			upper, upperTexts = append(upper, mark), append(upperTexts, text)
		}
	}
	addLabels(p, lower, lowerTexts, darkGray, below())
	addLabels(p, upper, upperTexts, darkGray, aboveLeft())
	ticks := []float64{0, 50, 75, 90, 95, 99, 99.9, 99.99}
	marker := make(plot.ConstantTicks, len(ticks))
	for i, t := range ticks {
		marker[i] = plot.Tick{Value: percentileAxis(t), Label: fmt.Sprintf("%g", t)}
	}
	p.X.Tick.Marker = marker
	p.X.Min, p.X.Max = 0, percentileAxis(99.99)
	return b.save("latency_percentiles", "Задержка по перцентилям", p)
}

func (b *chartBook) latencyHistogram() error {
	latency := b.samples.Latency
	edges := logEdges(slices.Min(latency), slices.Max(latency), 80)
	var ok, failed []float64
	for i, v := range latency {
		if b.samples.OK[i] {
			ok = append(ok, v)
		} else {
			failed = append(failed, v)
		}
	}
	p := newPlot("Гистограмма распределения задержки", "Задержка, мс (логарифмическая шкала)", "Число запросов")
	logX(p)
	expected := bars{edges: edges, counts: histogram(ok, edges), color: blue}
	p.Add(expected)
	p.Legend.Add("ожидаемый ответ", expected)
	if len(failed) > 0 {
		refused := bars{edges: edges, counts: histogram(failed, edges), color: red}
		p.Add(refused)
		p.Legend.Add("отказ", refused)
	}
	for _, q := range []float64{50, 95, 99} {
		addMarker(p, stats.Percentile(latency, q), percentileColors[q], true, fmt.Sprintf("p%g", q))
	}
	return b.save("latency_histogram", "Гистограмма задержки", p)
}

func (b *chartBook) operations() error {
	valid := b.samples.Where(func(i int) bool { return !b.samples.Invalid[i] })
	names := valid.Operations()
	p := newPlot("Задержка по операциям пользовательского сценария (без ошибочных операций)", "Задержка, мс", "")
	width := vg.Points(7)
	for offset, q := range []float64{50, 95, 99} {
		values := make(plotter.Values, len(names))
		for i, name := range names {
			subset := valid.Where(func(j int) bool { return valid.Operation[j] == name })
			values[i] = math.Max(0, nanToZero(stats.Percentile(subset.Latency, q)))
		}
		chart, err := plotter.NewBarChart(values, width)
		if err != nil {
			return err
		}
		chart.Horizontal = true
		chart.Color = percentileColors[q]
		chart.LineStyle.Width = 0
		chart.Offset = width * vg.Length(offset-1)
		p.Add(chart)
		p.Legend.Add(fmt.Sprintf("p%g", q), chart)
	}
	p.NominalY(names...)
	p.Legend.Left = false
	return b.save("operations", "Задержка по операциям", p)
}

func (b *chartBook) throughputTimeline() error {
	rps := stats.PerSecond(b.samples.Time, 0, b.end)
	failed := stats.PerSecond(b.samples.TimesWhere(func(i int) bool { return !b.samples.OK[i] }), 0, b.end)
	userErrors := stats.PerSecond(b.samples.TimesWhere(func(i int) bool { return b.samples.Invalid[i] && b.samples.OK[i] }), 0, b.end)
	seconds := make([]float64, len(rps))
	for i := range seconds {
		seconds[i] = float64(i) + 0.5
	}
	p := newPlot("Пропускная способность во времени", timeAxis, "Запросов в секунду")
	b.shade(p)
	addLine(p, points(seconds, rps, 0), hex(0xa5c3f0), 0.7, "RPS (1 с)", false)
	addLine(p, points(seconds, stats.Smooth(rps, 10), 0), navy, 1.6, "RPS (скользящее 10 с)", false)
	addLine(p, points(seconds, stats.Smooth(userErrors, 10), 0), yellow, 1.4, "ожидаемые ошибки пользователя", false)
	addLine(p, points(seconds, stats.Smooth(failed, 10), 0), red, 1.4, "отказы", false)
	p.Y.Min = 0
	return b.save("throughput_timeline", "Пропускная способность во времени", p, b.vusPanel())
}

func (b *chartBook) throughputDistribution() error {
	var rps []float64
	for _, phase := range b.phases {
		if phase.Steady {
			rps = append(rps, stats.PerSecond(b.samples.Time, phase.Start, phase.End)...)
		}
	}
	edges := linearEdges(slices.Min(rps), slices.Max(rps)+1, 60)
	p := newPlot("Распределение пропускной способности по секундам (установившиеся фазы)", "RPS за секунду", "Число секунд")
	p.Add(bars{edges: edges, counts: histogram(rps, edges), color: blue})
	for _, q := range stats.Percentiles {
		value := stats.Percentile(rps, q)
		addMarker(p, value, percentileColors[q], true, fmt.Sprintf("p%g = %.0f", q, value))
	}
	return b.save("throughput_distribution", "Распределение RPS", p)
}

type resourcePanel struct {
	title string
	lines []resourceLine
}

type resourceLine struct {
	metric string
	suffix string
	dashed bool
}

var resourcePanels = []resourcePanel{
	{"CPU, ядер", []resourceLine{{"cpu_cores", "", false}}},
	{"RAM, МБ", []resourceLine{{"memory_mb", "", false}}},
	{"Диск, КБ/с", []resourceLine{{"disk_write_kbps", " запись", false}, {"disk_read_kbps", " чтение", true}}},
	{"Сеть, КБ/с", []resourceLine{{"net_tx_kbps", " передача", false}, {"net_rx_kbps", " приём", true}}},
}

func (b *chartBook) resourceTimeline() error {
	panels := make([]*plot.Plot, 0, len(resourcePanels))
	for i, panel := range resourcePanels {
		title := ""
		if i == 0 {
			title = "Использование ресурсов по компонентам"
		}
		p := newPlot(title, timeAxis, panel.title)
		b.shade(p)
		for _, line := range panel.lines {
			for _, component := range b.resources.Components() {
				timeline := b.resources.Timeline(line.metric, component, b.samples.Start)
				if len(timeline.V) == 0 || slices.Max(timeline.V) <= 0 {
					continue
				}
				addLine(p, points(timeline.T, timeline.V, 0), componentColors[component], 1.1, component+line.suffix, line.dashed)
			}
		}
		p.X.Min, p.X.Max = 0, b.end
		p.Y.Min = 0
		panels = append(panels, p)
	}
	return b.save("resources", "Ресурсы во времени", panels...)
}

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

func (b *chartBook) vusPanel() *plot.Plot {
	p := newPlot("", timeAxis, "Число виртуальных пользователей")
	b.shade(p)
	addLine(p, points(b.samples.VUsTime, b.samples.VUs, 0), darkGray, 1.3, "VU", false)
	p.Y.Min = 0
	return p
}

func (b *chartBook) shade(p *plot.Plot) {
	var spans [][2]float64
	for _, phase := range b.phases {
		if !phase.Steady {
			spans = append(spans, [2]float64{phase.Start, phase.End})
		}
	}
	p.Add(bands{spans: spans})
	p.X.Min, p.X.Max = 0, b.end
}

func (b *chartBook) window() float64 {
	return math.Max(1, math.Round(b.end/300))
}

func (b *chartBook) save(name, title string, plots ...*plot.Plot) error {
	file := name + ".png"
	if err := savePanels(filepath.Join(b.dir, file), plots...); err != nil {
		return err
	}
	b.charts[name] = Chart{File: "charts/" + file, Title: title}
	return nil
}

func percentileAxis(q float64) float64 {
	return -math.Log10(math.Max(1-q/100, 1e-6))
}

func nanToZero(value float64) float64 {
	if math.IsNaN(value) {
		return 0
	}
	return value
}
