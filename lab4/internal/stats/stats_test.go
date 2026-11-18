package stats

import (
	"math"
	"testing"
)

func TestDescribeMatchesLinearInterpolation(t *testing.T) {
	values := make([]float64, 101)
	for i := range values {
		values[100-i] = float64(i)
	}

	s := Describe(values)

	want := map[string]Num{"p50": 50, "p75": 75, "p90": 90, "p95": 95, "p99": 99, "min": 0, "max": 100, "mean": 50}
	got := map[string]Num{"p50": s.P50, "p75": s.P75, "p90": s.P90, "p95": s.P95, "p99": s.P99, "min": s.Min, "max": s.Max, "mean": s.Mean}
	for name, expected := range want {
		if math.Abs(float64(got[name]-expected)) > 1e-9 {
			t.Errorf("%s = %v, want %v", name, got[name], expected)
		}
	}
}

func TestPerSecondCountsEventsIntoBuckets(t *testing.T) {
	counts := PerSecond([]float64{0.1, 0.9, 1.5, 3.2}, 0, 4)

	want := []float64{2, 1, 0, 1}
	for i := range want {
		if counts[i] != want[i] {
			t.Fatalf("PerSecond = %v, want %v", counts, want)
		}
	}
}

func TestNumMarshalsNaNAsNull(t *testing.T) {
	data, err := Num(math.NaN()).MarshalJSON()
	if err != nil || string(data) != "null" {
		t.Fatalf("MarshalJSON(NaN) = %s, %v", data, err)
	}
}
