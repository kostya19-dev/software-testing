package compare

import (
	"fmt"
	"math"

	"lab6/internal/bench"
)

func benchRows(report bench.Report) []BenchRow {
	var baseline *bench.Result
	for i := range report.Results {
		if report.Results[i].Variant == "baseline" {
			baseline = &report.Results[i]
		}
	}
	rows := make([]BenchRow, len(report.Results))
	for i, result := range report.Results {
		rows[i] = BenchRow{Result: result}
		if baseline != nil && result.Variant != "baseline" {
			rows[i].TimeDelta = delta(result.NsPerOp.Mean, baseline.NsPerOp.Mean)
			rows[i].BytesDelta = delta(result.BytesPerOp.Mean, baseline.BytesPerOp.Mean)
			rows[i].AllocsDelta = delta(result.AllocsPerOp.Mean, baseline.AllocsPerOp.Mean)
		}
	}
	return rows
}

func delta(value, baseline float64) string {
	if baseline == 0 || math.IsNaN(baseline) {
		return ""
	}
	return fmt.Sprintf("%+.0f%%", (value/baseline-1)*100)
}
