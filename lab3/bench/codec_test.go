package main

import (
	"reflect"
	"testing"
	"time"
)

func TestCodecsPreserveData(t *testing.T) {
	records, _, err := loadPatients()
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{4, 64} {
		for _, c := range codecs {
			r := records[size]
			b, err := validate(c, r)
			if err != nil {
				t.Fatal(err)
			}
			for _, scenario := range []string{"serialize", "deserialize", "roundtrip"} {
				operation(c, scenario, r, b)()
				if sinkError != nil {
					t.Fatal(sinkError)
				}
				if scenario != "serialize" && !reflect.DeepEqual(r, sinkRecord) {
					t.Fatal("decoded data mismatch")
				}
				if scenario != "deserialize" {
					got, err := c.deserialize(sinkBytes)
					if err != nil || !reflect.DeepEqual(got, r) {
						t.Fatalf("invalid output: %v", err)
					}
				}
			}
			if _, err := c.deserialize([]byte("corrupt")); err == nil {
				t.Fatal("corrupt input accepted")
			}
		}
	}
}
func TestPhaseAccounting(t *testing.T) {
	calls := 0
	got := measure(func() { calls++ }, 11, 7, time.Now())
	if calls != 40 {
		t.Fatalf("calls=%d, want 40", calls)
	}
	if got.Warmup.Operations != 11 || got.Batch.Operations != 11 || got.Requests.Operations != 7 || got.Teardown.Operations != 11 || len(got.Samples) != 7 {
		t.Fatal("phase counts")
	}
	for i, s := range got.Samples {
		if s.LatencyNS < 0 || (i > 0 && s.AtNS < got.Samples[i-1].AtNS) {
			t.Fatal("invalid timing")
		}
	}
	if got.TotalNS < got.Batch.ElapsedNS {
		t.Fatal("invalid total duration")
	}
}
