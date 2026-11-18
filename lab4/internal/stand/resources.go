package stand

import (
	"encoding/json"
	"math"
	"os"
	"slices"
	"time"

	"lab4/internal/stats"
)

type Metric struct {
	Key   string
	Title string
	Query string
}

var Metrics = []Metric{
	{"cpu_cores", "CPU, ядер", "rate(lab4_cpu_seconds_total[5s])"},
	{"cpu_throttled", "Троттлинг CPU, с/с", "rate(lab4_cpu_throttled_seconds_total[5s])"},
	{"memory_mb", "RAM (working set), МБ", "lab4_memory_working_set_bytes / 1048576"},
	{"disk_read_kbps", "Чтение с диска, КБ/с", "rate(lab4_disk_read_bytes_total[5s]) / 1024"},
	{"disk_write_kbps", "Запись на диск, КБ/с", "rate(lab4_disk_write_bytes_total[5s]) / 1024"},
	{"net_rx_kbps", "Сеть, приём КБ/с", "rate(lab4_network_receive_bytes_total[5s]) / 1024"},
	{"net_tx_kbps", "Сеть, передача КБ/с", "rate(lab4_network_transmit_bytes_total[5s]) / 1024"},
	{"threads", "Процессы и потоки", "lab4_pids"},
}

func MetricTitle(key string) string {
	for _, m := range Metrics {
		if m.Key == key {
			return m.Title
		}
	}
	return key
}

type Timeline struct {
	T []float64 `json:"t"`
	V []float64 `json:"v"`
}

type Resources map[string]map[string]Timeline

func CollectResources(client *Client, start, end float64) (Resources, error) {
	result := Resources{}
	for _, metric := range Metrics {
		series, err := client.Range(metric.Query, start, end, time.Second, "service")
		if err != nil {
			return nil, err
		}
		result[metric.Key] = map[string]Timeline{}
		for component, points := range series {
			var line Timeline
			for _, p := range points {
				line.T = append(line.T, p.Time)
				line.V = append(line.V, p.Value)
			}
			result[metric.Key][component] = line
		}
	}
	return result, nil
}

func LoadResources(path string) (Resources, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Resources
	return r, json.Unmarshal(data, &r)
}

func (r Resources) Save(path string) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (r Resources) Components() []string {
	seen := map[string]bool{}
	for _, components := range r {
		for name := range components {
			seen[name] = true
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (r Resources) Timeline(metric, component string, origin float64) Timeline {
	line := r[metric][component]
	shifted := Timeline{T: make([]float64, len(line.T)), V: line.V}
	for i, t := range line.T {
		shifted.T[i] = t - origin
	}
	return shifted
}

func (r Resources) Values(metric, component string, start, end, origin float64) []float64 {
	line := r.Timeline(metric, component, origin)
	var values []float64
	for i, t := range line.T {
		if t >= start && t <= end {
			values = append(values, line.V[i])
		}
	}
	return values
}

func (r Resources) Mean(metric, component string, start, end, origin float64) float64 {
	return stats.Mean(r.Values(metric, component, start, end, origin))
}

type Aggregate struct {
	Min    stats.Num `json:"min"`
	Max    stats.Num `json:"max"`
	Mean   stats.Num `json:"mean"`
	Median stats.Num `json:"median"`
	P95    stats.Num `json:"p95"`
}

type Summary map[string]map[string]Aggregate

func (r Resources) Summarize(start, end, origin float64) Summary {
	summary := Summary{}
	for _, component := range r.Components() {
		summary[component] = map[string]Aggregate{}
		for _, metric := range Metrics {
			values := r.Values(metric.Key, component, start, end, origin)
			if len(values) == 0 {
				continue
			}
			sorted := slices.Clone(values)
			slices.Sort(sorted)
			summary[component][metric.Key] = Aggregate{
				Min:    stats.Num(sorted[0]),
				Max:    stats.Num(sorted[len(sorted)-1]),
				Mean:   stats.Num(stats.Mean(sorted)),
				Median: stats.Num(stats.SortedPercentile(sorted, 50)),
				P95:    stats.Num(stats.SortedPercentile(sorted, 95)),
			}
		}
	}
	return summary
}

func (r Resources) MemoryGrowthPerHour(component string, start, end, origin float64) float64 {
	line := r.Timeline("memory_mb", component, origin)
	var x, y []float64
	for i, t := range line.T {
		if t >= start && t <= end && !math.IsNaN(line.V[i]) {
			x = append(x, t)
			y = append(y, line.V[i])
		}
	}
	return stats.Slope(x, y) * 3600
}
