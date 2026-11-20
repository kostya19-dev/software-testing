package report

import (
	"slices"

	"gonum.org/v1/plot"
)

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
