//go:build baseline

package main

import (
	"encoding/json"
	"testing"
)

func TestBaselineUsesOriginalShape(t *testing.T) {
	records, inputs, err := loadPatients()
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		if input.Depth != 3 {
			t.Fatalf("unexpected depth %d", input.Depth)
		}
		data, err := json.Marshal(records[input.Size])
		if err != nil {
			t.Fatal(err)
		}
		original, err := inputFiles.ReadFile(input.File)
		if err != nil {
			t.Fatal(err)
		}
		var actual, expected map[string]json.RawMessage
		if err := json.Unmarshal(data, &actual); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(original, &expected); err != nil {
			t.Fatal(err)
		}
		if _, hasClinic := actual["Clinic"]; hasClinic || len(actual) != len(expected) {
			t.Fatal("baseline shape changed")
		}
		if len(records[input.Size].Visits) != input.Size {
			t.Fatal("wrong visit count")
		}
		if _, _, err := parsePatient([]byte(`{"Clinic":{"Department":{"Team":{"Doctor":{}}}}}`), input.Size); err == nil {
			t.Fatal("accepted five levels")
		}
	}
}
