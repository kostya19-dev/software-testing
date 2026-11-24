package grafana

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
)

// Labels are written in key order, so equal sets give the same series.
type Labels map[string]string

func (l Labels) String() string {
	keys := make([]string, 0, len(l))
	for k := range l {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + `="` + escape(l[k]) + `"`
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func (l Labels) With(key, value string) Labels {
	out := make(Labels, len(l)+1)
	for k, v := range l {
		out[k] = v
	}
	out[key] = value
	return out
}

func escape(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(value)
}

type sample struct {
	at    float64
	value float64
}

type family struct {
	help   string
	series map[string][]sample
}

// Metrics collects gauges and writes them as OpenMetrics text for `promtool tsdb create-blocks-from openmetrics`.
type Metrics struct {
	families map[string]*family
	samples  int
}

func NewMetrics() *Metrics {
	return &Metrics{families: map[string]*family{}}
}

func (m *Metrics) Add(name, help string, labels Labels, at, value float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	f := m.families[name]
	if f == nil {
		f = &family{help: help, series: map[string][]sample{}}
		m.families[name] = f
	}
	key := labels.String()
	f.series[key] = append(f.series[key], sample{at: at, value: value})
	m.samples++
}

func (m *Metrics) Samples() int {
	return m.samples
}

// Write groups samples by family and orders them by time inside every series, as promtool requires.
func (m *Metrics) Write(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	w := bufio.NewWriterSize(file, 1<<20)

	names := make([]string, 0, len(m.families))
	for name := range m.families {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		f := m.families[name]
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s gauge\n", name, escape(f.help), name)
		keys := make([]string, 0, len(f.series))
		for key := range f.series {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			samples := f.series[key]
			slices.SortStableFunc(samples, func(a, b sample) int { return compareFloat(a.at, b.at) })
			last := math.Inf(-1)
			for _, s := range samples {
				at := math.Round(s.at*1000) / 1000
				if at <= last {
					continue
				}
				last = at
				w.WriteString(name + key + " " + strconv.FormatFloat(s.value, 'g', 8, 64) + " " + strconv.FormatFloat(at, 'f', 3, 64) + "\n")
			}
		}
	}
	w.WriteString("# EOF\n")
	if err := w.Flush(); err != nil {
		return err
	}
	return file.Close()
}

func compareFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
