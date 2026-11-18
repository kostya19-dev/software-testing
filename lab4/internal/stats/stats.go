package stats

import (
	"encoding/json"
	"math"
	"slices"
)

var Percentiles = []float64{50, 75, 90, 95, 99}

type Num float64

func (n Num) MarshalJSON() ([]byte, error) {
	value := float64(n)
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return []byte("null"), nil
	}
	return json.Marshal(math.Round(value*1000) / 1000)
}

func (n *Num) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*n = Num(math.NaN())
		return nil
	}
	var value float64
	err := json.Unmarshal(data, &value)
	*n = Num(value)
	return err
}

type Summary struct {
	Count int `json:"count"`
	Min   Num `json:"min"`
	Mean  Num `json:"mean"`
	Max   Num `json:"max"`
	Stdev Num `json:"stdev"`
	P50   Num `json:"p50"`
	P75   Num `json:"p75"`
	P90   Num `json:"p90"`
	P95   Num `json:"p95"`
	P99   Num `json:"p99"`
}

func (s Summary) Percentile(q float64) Num {
	return map[float64]Num{50: s.P50, 75: s.P75, 90: s.P90, 95: s.P95, 99: s.P99}[q]
}

func Describe(values []float64) Summary {
	if len(values) == 0 {
		nan := Num(math.NaN())
		return Summary{Min: nan, Mean: nan, Max: nan, Stdev: nan, P50: nan, P75: nan, P90: nan, P95: nan, P99: nan}
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	mean := Mean(sorted)
	variance := 0.0
	for _, v := range sorted {
		variance += (v - mean) * (v - mean)
	}
	return Summary{
		Count: len(sorted),
		Min:   Num(sorted[0]),
		Mean:  Num(mean),
		Max:   Num(sorted[len(sorted)-1]),
		Stdev: Num(math.Sqrt(variance / float64(len(sorted)))),
		P50:   Num(SortedPercentile(sorted, 50)),
		P75:   Num(SortedPercentile(sorted, 75)),
		P90:   Num(SortedPercentile(sorted, 90)),
		P95:   Num(SortedPercentile(sorted, 95)),
		P99:   Num(SortedPercentile(sorted, 99)),
	}
}

func Percentile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	return SortedPercentile(sorted, q)
}

func SortedPercentile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return math.NaN()
	}
	rank := q / 100 * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))
	return sorted[lower] + (sorted[upper]-sorted[lower])*(rank-float64(lower))
}

func Mean(values []float64) float64 {
	if len(values) == 0 {
		return math.NaN()
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func Share(flags []bool) float64 {
	if len(flags) == 0 {
		return 0
	}
	count := 0
	for _, flag := range flags {
		if flag {
			count++
		}
	}
	return float64(count) / float64(len(flags))
}

func PerSecond(times []float64, start, end float64) []float64 {
	first, last := math.Floor(start), math.Ceil(end)
	if last <= first {
		return nil
	}
	counts := make([]float64, int(last-first))
	for _, t := range times {
		if t >= first && t < last {
			counts[int(t-first)]++
		}
	}
	return counts
}

type Window struct {
	Start, End float64
	From, To   int
}

func (w Window) Center() float64 {
	return (w.Start + w.End) / 2
}

func Windows(sortedTimes []float64, origin, end, width float64) []Window {
	var windows []Window
	for start := origin; start < end; start += width {
		from, _ := slices.BinarySearch(sortedTimes, start)
		to, _ := slices.BinarySearch(sortedTimes, start+width)
		windows = append(windows, Window{Start: start, End: start + width, From: from, To: to})
	}
	return windows
}

func Slope(x, y []float64) float64 {
	if len(x) < 2 {
		return 0
	}
	mx, my := Mean(x), Mean(y)
	num, den := 0.0, 0.0
	for i := range x {
		num += (x[i] - mx) * (y[i] - my)
		den += (x[i] - mx) * (x[i] - mx)
	}
	if den == 0 {
		return 0
	}
	return num / den
}

func Smooth(values []float64, width int) []float64 {
	out := make([]float64, len(values))
	half := width / 2
	for i := range values {
		from, to := max(0, i-half), min(len(values), i+half+1)
		out[i] = Mean(values[from:to])
	}
	return out
}
