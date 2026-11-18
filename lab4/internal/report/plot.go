package report

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"slices"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

const (
	chartWidth = 11 * vg.Inch
	panelHigh  = 3.6 * vg.Inch
	chartDPI   = 110
)

var (
	blue     = hex(0x2a6fdb)
	navy     = hex(0x1c4fa3)
	cyan     = hex(0x15aabf)
	yellow   = hex(0xfab005)
	orange   = hex(0xe8590c)
	red      = hex(0xc92a2a)
	green    = hex(0x2f9e44)
	violet   = hex(0xae3ec9)
	gray     = hex(0x868e96)
	darkGray = hex(0x495057)
	shade    = color.RGBA{R: 0xe9, G: 0xec, B: 0xef, A: 0xff}

	percentileColors = map[float64]color.Color{50: blue, 75: cyan, 90: yellow, 95: orange, 99: red}
	componentColors  = map[string]color.Color{"api": blue, "mongo": green, "loadgen": orange, "prometheus": violet, "exporter": gray}
)

func hex(value uint32) color.RGBA {
	return color.RGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 0xff}
}

func newPlot(title, xLabel, yLabel string) *plot.Plot {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel
	p.Legend.Top = true
	p.Legend.Left = true
	p.Add(plotter.NewGrid())
	return p
}

func logY(p *plot.Plot) {
	p.Y.Scale = plot.LogScale{}
	p.Y.Tick.Marker = plot.LogTicks{Prec: -1}
}

func logX(p *plot.Plot) {
	p.X.Scale = plot.LogScale{}
	p.X.Tick.Marker = plot.LogTicks{Prec: -1}
}

func points(xs, ys []float64, floor float64) plotter.XYs {
	out := make(plotter.XYs, 0, len(xs))
	for i := range xs {
		if math.IsNaN(ys[i]) || math.IsInf(ys[i], 0) {
			continue
		}
		out = append(out, plotter.XY{X: xs[i], Y: math.Max(ys[i], floor)})
	}
	return out
}

func addLine(p *plot.Plot, xys plotter.XYs, c color.Color, width float64, label string, dashed bool) {
	if len(xys) == 0 {
		return
	}
	line, err := plotter.NewLine(xys)
	if err != nil {
		panic(err)
	}
	line.Color = c
	line.Width = vg.Points(width)
	if dashed {
		line.Dashes = []vg.Length{vg.Points(5), vg.Points(3)}
	}
	p.Add(line)
	if label != "" {
		p.Legend.Add(label, line)
	}
}

func addMarkers(p *plot.Plot, xys plotter.XYs, c color.Color, label string) {
	scatter, err := plotter.NewScatter(xys)
	if err != nil {
		panic(err)
	}
	scatter.Color = c
	scatter.Radius = vg.Points(3)
	scatter.Shape = draw.CircleGlyph{}
	p.Add(scatter)
	if label != "" {
		p.Legend.Add(label, scatter)
	}
}

type placement struct {
	offset vg.Point
	align  draw.XAlignment
}

func addLabels(p *plot.Plot, xys plotter.XYs, texts []string, c color.Color, at placement) {
	if len(xys) == 0 {
		return
	}
	labels, err := plotter.NewLabels(plotter.XYLabels{XYs: xys, Labels: texts})
	if err != nil {
		panic(err)
	}
	for i := range labels.TextStyle {
		labels.TextStyle[i].Color = c
		labels.TextStyle[i].XAlign = at.align
	}
	labels.Offset = at.offset
	p.Add(labels)
}

func above() placement {
	return placement{offset: vg.Point{X: vg.Points(4), Y: vg.Points(5)}, align: draw.XLeft}
}

func below() placement {
	return placement{offset: vg.Point{X: vg.Points(6), Y: vg.Points(-12)}, align: draw.XLeft}
}

func aboveLeft() placement {
	return placement{offset: vg.Point{X: vg.Points(-6), Y: vg.Points(5)}, align: draw.XRight}
}

type bands struct {
	spans [][2]float64
}

func (b bands) Plot(c draw.Canvas, p *plot.Plot) {
	trX, _ := p.Transforms(&c)
	for _, span := range b.spans {
		x0, x1 := trX(span[0]), trX(span[1])
		c.FillPolygon(shade, c.ClipPolygonX([]vg.Point{{X: x0, Y: c.Min.Y}, {X: x1, Y: c.Min.Y}, {X: x1, Y: c.Max.Y}, {X: x0, Y: c.Max.Y}}))
	}
}

type marker struct {
	x     float64
	style draw.LineStyle
}

func newMarker(x float64, c color.Color, dashed bool) marker {
	style := draw.LineStyle{Color: c, Width: vg.Points(1.3)}
	if dashed {
		style.Dashes = []vg.Length{vg.Points(5), vg.Points(3)}
	}
	return marker{x: x, style: style}
}

func (m marker) Plot(c draw.Canvas, p *plot.Plot) {
	trX, _ := p.Transforms(&c)
	x := trX(m.x)
	c.StrokeLine2(m.style, x, c.Min.Y, x, c.Max.Y)
}

func (m marker) Thumbnail(c *draw.Canvas) {
	y := c.Center().Y
	c.StrokeLine2(m.style, c.Min.X, y, c.Max.X, y)
}

func addMarker(p *plot.Plot, x float64, c color.Color, dashed bool, label string) {
	m := newMarker(x, c, dashed)
	p.Add(m)
	if label != "" {
		p.Legend.Add(label, m)
	}
}

func addHorizontal(p *plot.Plot, y float64, c color.Color, label string) {
	f := plotter.NewFunction(func(float64) float64 { return y })
	f.Color = c
	f.Width = vg.Points(1.2)
	f.Dashes = []vg.Length{vg.Points(5), vg.Points(3)}
	p.Add(f)
	if label != "" {
		p.Legend.Add(label, f)
	}
}

type bars struct {
	edges  []float64
	counts []float64
	color  color.Color
}

func (b bars) Plot(c draw.Canvas, p *plot.Plot) {
	trX, trY := p.Transforms(&c)
	for i, count := range b.counts {
		if count <= 0 {
			continue
		}
		x0, x1 := trX(b.edges[i]), trX(b.edges[i+1])
		y0, y1 := trY(math.Max(p.Y.Min, 0)), trY(count)
		c.FillPolygon(b.color, c.ClipPolygonXY([]vg.Point{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1}}))
	}
}

func (b bars) DataRange() (xmin, xmax, ymin, ymax float64) {
	ymax = 0
	for _, count := range b.counts {
		ymax = math.Max(ymax, count)
	}
	return b.edges[0], b.edges[len(b.edges)-1], 0, ymax
}

func (b bars) Thumbnail(c *draw.Canvas) {
	c.FillPolygon(b.color, []vg.Point{{X: c.Min.X, Y: c.Min.Y}, {X: c.Max.X, Y: c.Min.Y}, {X: c.Max.X, Y: c.Max.Y}, {X: c.Min.X, Y: c.Max.Y}})
}

func histogram(values, edges []float64) []float64 {
	counts := make([]float64, len(edges)-1)
	last := len(edges) - 1
	for _, v := range values {
		if v < edges[0] || v > edges[last] {
			continue
		}
		i, exact := slices.BinarySearch(edges, v)
		if !exact || i == last {
			i--
		}
		counts[i]++
	}
	return counts
}

func logEdges(min, max float64, count int) []float64 {
	low, high := math.Log10(math.Max(min, 0.01)), math.Log10(max*1.01)
	edges := make([]float64, count+1)
	for i := range edges {
		edges[i] = math.Pow(10, low+(high-low)*float64(i)/float64(count))
	}
	return edges
}

func linearEdges(min, max float64, count int) []float64 {
	if max <= min {
		max = min + 1
	}
	edges := make([]float64, count+1)
	for i := range edges {
		edges[i] = min + (max-min)*float64(i)/float64(count)
	}
	return edges
}

func reserveLegendSpace(p *plot.Plot) {
	if _, logarithmic := p.Y.Scale.(plot.LogScale); logarithmic {
		p.Y.Max *= 6
		return
	}
	p.Y.Max = p.Y.Min + (p.Y.Max-p.Y.Min)*1.4
}

func savePanels(path string, plots ...*plot.Plot) error {
	for _, p := range plots {
		reserveLegendSpace(p)
	}
	img := vgimg.NewWith(vgimg.UseWH(chartWidth, panelHigh*vg.Length(len(plots))), vgimg.UseDPI(chartDPI))
	dc := draw.New(img)
	grid := make([][]*plot.Plot, len(plots))
	for i, p := range plots {
		grid[i] = []*plot.Plot{p}
	}
	tiles := draw.Tiles{Rows: len(plots), Cols: 1, PadX: vg.Points(6), PadY: vg.Points(10), PadTop: vg.Points(6), PadBottom: vg.Points(6), PadLeft: vg.Points(6), PadRight: vg.Points(12)}
	canvases := plot.Align(grid, tiles, dc)
	for i, p := range plots {
		p.Draw(canvases[i][0])
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := (vgimg.PngCanvas{Canvas: img}).WriteTo(file); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
