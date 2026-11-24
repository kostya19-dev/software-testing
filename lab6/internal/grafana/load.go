package grafana

import (
	"compress/gzip"
	"encoding/csv"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
)

type second struct {
	requests  int
	failed    int
	latencies []float64
}

// exportLoad turns the raw requests of the load generator into per-second RPS, latency and failures.
func exportLoad(m *Metrics, dir string, labels Labels, start float64) error {
	if math.IsNaN(start) {
		return os.ErrNotExist
	}
	seconds := map[int]*second{}
	err := eachRow(filepath.Join(dir, "requests.csv.gz"), true, func(r []string) {
		// at_ms,latency_us,operation,status,expected,invalid
		at := parse(r[0])/1000 - start
		if at < 0 {
			return
		}
		s := seconds[int(at)]
		if s == nil {
			s = &second{}
			seconds[int(at)] = s
		}
		s.requests++
		if r[4] != "1" {
			s.failed++
		}
		s.latencies = append(s.latencies, parse(r[1])/1000)
	})
	if err != nil {
		return err
	}
	for at, s := range seconds {
		ts := Epoch + float64(at)
		slices.Sort(s.latencies)
		m.Add("lab6_rps", "Запросов в секунду", labels, ts, float64(s.requests))
		m.Add("lab6_failed_rps", "Отказов в секунду", labels, ts, float64(s.failed))
		m.Add("lab6_latency_ms", "Задержка за секунду, мс", labels.With("q", "p50"), ts, percentile(s.latencies, 0.50))
		m.Add("lab6_latency_ms", "Задержка за секунду, мс", labels.With("q", "p95"), ts, percentile(s.latencies, 0.95))
	}
	return eachRow(filepath.Join(dir, "vus.csv"), false, func(r []string) {
		if at := parse(r[0])/1000 - start; at >= 0 {
			m.Add("lab6_vus", "Виртуальных пользователей", labels, Epoch+at, parse(r[1]))
		}
	})
}

func percentile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return math.NaN()
	}
	return sorted[min(len(sorted)-1, int(float64(len(sorted))*q))]
}

func eachRow(path string, gzipped bool, handle func([]string)) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var source io.Reader = file
	if gzipped {
		decompressed, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		source = decompressed
	}
	reader := csv.NewReader(source)
	reader.ReuseRecord = true
	if _, err := reader.Read(); err != nil {
		return err
	}
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		handle(record)
	}
}

func parse(value string) float64 {
	number, _ := strconv.ParseFloat(value, 64)
	return number
}
