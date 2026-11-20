package report

import (
	"fmt"
	"math"
	"slices"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	"lab4/internal/stats"
)

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
