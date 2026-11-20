package report

import (
	"fmt"
	"slices"

	"lab4/internal/stats"
)

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
