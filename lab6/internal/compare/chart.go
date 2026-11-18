package compare

import (
	"image/color"
	"os"
	"path/filepath"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

var variantColors = []color.Color{
	color.RGBA{R: 0x86, G: 0x8e, B: 0x96, A: 0xff},
	color.RGBA{R: 0x2a, G: 0x6f, B: 0xdb, A: 0xff},
	color.RGBA{R: 0xe8, G: 0x59, B: 0x0c, A: 0xff},
	color.RGBA{R: 0x2f, G: 0x9e, B: 0x44, A: 0xff},
}

var components = []string{"api", "mongo", "jaeger", "мониторинг"}

func renderUsage(path string, block Scenario) error {
	cpu := groupedBars("CPU, ядер (среднее)", block, func(r Row) []float64 {
		return []float64{r.APICPU.Mean, r.MongoCPU.Mean, r.Tracing.Mean, r.Monitoring}
	})
	ram := groupedBars("RAM, МБ (среднее; мониторинг — максимум)", block, func(r Row) []float64 {
		return []float64{r.APIRAM.Mean, r.MongoRAM.Mean, r.TracingRAM.Mean, r.MonitoringRAM}
	})

	img := vgimg.NewWith(vgimg.UseWH(11*vg.Inch, 3.4*vg.Inch), vgimg.UseDPI(110))
	canvas := draw.New(img)
	tiles := draw.Tiles{Rows: 1, Cols: 2, PadX: vg.Points(18), PadY: vg.Points(6), PadTop: vg.Points(6), PadBottom: vg.Points(6), PadLeft: vg.Points(6), PadRight: vg.Points(6)}
	canvases := plot.Align([][]*plot.Plot{{cpu, ram}}, tiles, canvas)
	cpu.Draw(canvases[0][0])
	ram.Draw(canvases[0][1])

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = vgimg.PngCanvas{Canvas: img}.WriteTo(file)
	return err
}

func groupedBars(title string, block Scenario, values func(Row) []float64) *plot.Plot {
	p := plot.New()
	p.Title.Text = title
	p.Legend.Top = true
	p.Add(plotter.NewGrid())
	width := vg.Points(14)
	count := len(block.Rows)
	for i, row := range block.Rows {
		bars, err := plotter.NewBarChart(plotter.Values(values(row)), width)
		if err != nil {
			panic(err)
		}
		bars.Color = variantColors[i%len(variantColors)]
		bars.LineStyle.Width = 0
		bars.Offset = width * vg.Length(float64(i)-float64(count-1)/2)
		p.Add(bars)
		p.Legend.Add(row.Variant.Name, bars)
	}
	p.NominalX(components...)
	p.Y.Min = 0
	p.Y.Max *= 1.3
	return p
}
