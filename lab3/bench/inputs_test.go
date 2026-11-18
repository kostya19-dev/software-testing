//go:build !baseline

package main

import (
	"encoding/json"
	"testing"
)

func TestMeasuredInputs(t *testing.T) {
	records, inputs, err := loadPatients()
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range inputs {
		data, err := inputFiles.ReadFile(info.File)
		if err != nil {
			t.Fatal(err)
		}
		var tree map[string]any
		if err := json.Unmarshal(data, &tree); err != nil {
			t.Fatal(err)
		}
		if len(records[info.Size].Visits) != info.Size || info.Depth != 5 {
			t.Fatal(info)
		}
		doctor := tree["Clinic"].(map[string]any)["Department"].(map[string]any)["Team"].(map[string]any)["Doctor"].(map[string]any)
		doctor["Name"] = "Изменённое имя"
		edited, err := json.Marshal(tree)
		if err != nil {
			t.Fatal(err)
		}
		record, _, err := parsePatient(edited, info.Size)
		if err != nil || record.Clinic.Department.Team.Doctor.Name != "Изменённое имя" {
			t.Fatalf("changed input field lost: %v", err)
		}
		for _, c := range codecs {
			if _, err := validate(c, record); err != nil {
				t.Fatal(err)
			}
		}
		if _, _, err := parsePatient(data, info.Size+1); err == nil {
			t.Fatal("accepted wrong size")
		}
	}
	if _, _, err := parsePatient([]byte(`{"Visits":[{}]}`), 1); err == nil {
		t.Fatal("accepted shallow input")
	}
}
