package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"lab4/internal/stats"
)

type indexRow struct {
	ID          string
	Scenario    string
	Scale       float64
	Requests    int
	RPS         stats.Num
	P50         stats.Num
	P95         stats.Num
	P99         stats.Num
	FailureRate stats.Num
	Headline    string
}

func BuildIndex(resultsDir string) (string, error) {
	files, err := filepath.Glob(filepath.Join(resultsDir, "*", "summary.json"))
	if err != nil {
		return "", err
	}
	slices.Sort(files)
	slices.Reverse(files)

	rows := make([]indexRow, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		var s Summary
		if err := json.Unmarshal(data, &s); err != nil {
			return "", err
		}
		rows = append(rows, indexRow{
			ID:          filepath.Base(filepath.Dir(file)),
			Scenario:    s.Run.Scenario,
			Scale:       s.Run.Scale,
			Requests:    s.Overall.Requests,
			RPS:         s.Overall.RPS.Mean,
			P50:         s.Overall.Latency.P50,
			P95:         s.Overall.Latency.P95,
			P99:         s.Overall.Latency.P99,
			FailureRate: s.Overall.FailureRate,
			Headline:    headline(s),
		})
	}

	target := filepath.Join(resultsDir, "index.html")
	file, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return target, templates.ExecuteTemplate(file, "index.html", rows)
}
