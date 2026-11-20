package report

import (
	"math"
	"path/filepath"

	"gonum.org/v1/plot"

	"lab4/internal/analysis"
	"lab4/internal/load"
	"lab4/internal/samples"
	"lab4/internal/stand"
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
