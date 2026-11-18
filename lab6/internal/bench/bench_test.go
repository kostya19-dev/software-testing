package bench

import "testing"

const output = `goos: linux
cpu: Intel(R) Core(TM) i5-10300H CPU @ 2.50GHz
BenchmarkObservabilityOverhead/baseline-8         	    3000	    120000 ns/op	   34000 B/op	     200 allocs/op
BenchmarkObservabilityOverhead/baseline-8         	    3000	    110000 ns/op	   34000 B/op	     200 allocs/op
BenchmarkObservabilityOverhead/tracing+logging-8  	    3000	    290000 ns/op	   70000 B/op	     480 allocs/op
PASS`

func TestParseAggregatesSamplesPerVariant(t *testing.T) {
	report := Parse(output)

	if report.CPU != "Intel(R) Core(TM) i5-10300H CPU @ 2.50GHz" {
		t.Errorf("CPU = %q", report.CPU)
	}
	if len(report.Results) != 2 {
		t.Fatalf("results = %d, want 2", len(report.Results))
	}
	baseline := report.Results[0]
	if baseline.Variant != "baseline" || baseline.Samples != 2 || baseline.NsPerOp.Mean != 115000 {
		t.Errorf("baseline = %+v", baseline)
	}
	if report.Results[1].Variant != "tracing+logging" || report.Results[1].AllocsPerOp.Mean != 480 {
		t.Errorf("tracing+logging = %+v", report.Results[1])
	}
}
