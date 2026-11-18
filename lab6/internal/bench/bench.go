package bench

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	pattern    = "ObservabilityOverhead"
	pkg        = "./internal/delivery/http/"
	ResultFile = "bench.json"
	RawFile    = "bench.txt"
)

var line = regexp.MustCompile(`^Benchmark` + pattern + `/(\S+?)(-\d+)?\s+\d+\s+([\d.]+) ns/op\s+([\d.]+) B/op\s+([\d.]+) allocs/op`)

type Stat struct {
	Mean  float64 `json:"mean"`
	Stdev float64 `json:"stdev"`
}

type Result struct {
	Variant     string `json:"variant"`
	Samples     int    `json:"samples"`
	NsPerOp     Stat   `json:"ns_per_op"`
	BytesPerOp  Stat   `json:"bytes_per_op"`
	AllocsPerOp Stat   `json:"allocs_per_op"`
}

type Report struct {
	CPU       string   `json:"cpu"`
	Count     int      `json:"count"`
	Benchtime string   `json:"benchtime"`
	Results   []Result `json:"results"`
}

func Run(clinicDir, resultsDir string, count int, benchtime string) (Report, error) {
	dir, err := filepath.Abs(clinicDir)
	if err != nil {
		return Report{}, err
	}
	cmd := exec.Command("go", "test", "-run", "^$", "-bench", pattern, "-benchmem",
		"-benchtime", benchtime, "-count", strconv.Itoa(count), pkg)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, os.Stderr
	if err := cmd.Run(); err != nil {
		return Report{}, fmt.Errorf("go test -bench: %w\n%s", err, out.String())
	}
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return Report{}, err
	}
	if err := os.WriteFile(filepath.Join(resultsDir, RawFile), out.Bytes(), 0o644); err != nil {
		return Report{}, err
	}
	report := Parse(out.String())
	report.Count, report.Benchtime = count, benchtime
	return report, Save(filepath.Join(resultsDir, ResultFile), report)
}

func Parse(output string) Report {
	var report Report
	samples := map[string][][3]float64{}
	var order []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		text := scanner.Text()
		if cpu, ok := strings.CutPrefix(text, "cpu: "); ok {
			report.CPU = cpu
		}
		match := line.FindStringSubmatch(text)
		if match == nil {
			continue
		}
		name := match[1]
		if _, seen := samples[name]; !seen {
			order = append(order, name)
		}
		samples[name] = append(samples[name], [3]float64{number(match[3]), number(match[4]), number(match[5])})
	}
	for _, name := range order {
		values := samples[name]
		report.Results = append(report.Results, Result{
			Variant:     name,
			Samples:     len(values),
			NsPerOp:     stat(values, 0),
			BytesPerOp:  stat(values, 1),
			AllocsPerOp: stat(values, 2),
		})
	}
	return report
}

func Load(path string) (Report, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Report{}, false
	}
	var r Report
	return r, json.Unmarshal(data, &r) == nil
}

func Save(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func stat(values [][3]float64, column int) Stat {
	mean := 0.0
	for _, v := range values {
		mean += v[column]
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, v := range values {
		variance += (v[column] - mean) * (v[column] - mean)
	}
	if len(values) > 1 {
		variance /= float64(len(values) - 1)
	}
	return Stat{Mean: mean, Stdev: math.Sqrt(variance)}
}

func number(text string) float64 {
	value, _ := strconv.ParseFloat(text, 64)
	return value
}
