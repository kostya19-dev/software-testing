// Package k6 turns the CSV output of a k6 run into the samples the lab4 report reads:
// load/requests.csv.gz (one line per request) and load/vus.csv (active VUs every second).
package k6

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

const (
	OutputFile   = "k6.csv.gz"
	requestsFile = "requests.csv.gz"
	vusFile      = "vus.csv"
	metaFile     = "meta.json"
)

type Result struct {
	Requests int
	Failed   int
	Invalid  int
}

// Convert reads <loadDir>/k6.csv.gz written with K6_CSV_TIME_FORMAT=unix_micro and removes it afterwards:
// the per-request samples are kept in the lab4 format only.
func Convert(loadDir string) (Result, error) {
	var result Result
	if _, err := os.Stat(filepath.Join(loadDir, metaFile)); err != nil {
		return result, fmt.Errorf("k6 не записал %s: %w", metaFile, err)
	}
	source, err := os.Open(filepath.Join(loadDir, OutputFile))
	if err != nil {
		return result, err
	}
	defer source.Close()
	decompressed, err := gzip.NewReader(source)
	if err != nil {
		return result, err
	}
	reader := csv.NewReader(bufio.NewReaderSize(decompressed, 1<<20))
	reader.ReuseRecord = true
	header, err := reader.Read()
	if err != nil {
		return result, err
	}
	column := map[string]int{}
	for i, name := range header {
		column[name] = i
	}
	for _, name := range []string{"metric_name", "timestamp", "metric_value", "status", "expected_response", "extra_tags"} {
		if _, ok := column[name]; !ok {
			return result, fmt.Errorf("в выводе k6 нет колонки %s", name)
		}
	}

	requests, err := newGzipFile(filepath.Join(loadDir, requestsFile))
	if err != nil {
		return result, err
	}
	vus, err := os.Create(filepath.Join(loadDir, vusFile))
	if err != nil {
		requests.Close()
		return result, err
	}
	vusOut := bufio.NewWriter(vus)
	requests.WriteString("at_ms,latency_us,operation,status,expected,invalid\n")
	vusOut.WriteString("at_ms,vus\n")

	lastVUs := int64(0)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			requests.Close()
			vus.Close()
			return result, err
		}
		micros, _ := strconv.ParseInt(record[column["timestamp"]], 10, 64)
		value, _ := strconv.ParseFloat(record[column["metric_value"]], 64)
		switch record[column["metric_name"]] {
		case "http_req_duration":
			tags, _ := url.ParseQuery(record[column["extra_tags"]])
			expected := record[column["expected_response"]] == "true"
			invalid := tags.Get("invalid") == "1"
			fmt.Fprintf(requests, "%d,%d,%s,%s,%d,%d\n", micros/1000, int64(value*1000+0.5),
				tags.Get("op"), record[column["status"]], flag(expected), flag(invalid))
			result.Requests++
			if !expected {
				result.Failed++
			}
			if invalid {
				result.Invalid++
			}
		case "vus":
			fmt.Fprintf(vusOut, "%d,%d\n", micros/1000, int64(value))
			lastVUs = micros / 1000
		}
	}
	if lastVUs > 0 {
		fmt.Fprintf(vusOut, "%d,0\n", lastVUs+1000)
	}
	if err := errors.Join(vusOut.Flush(), vus.Close(), requests.Close()); err != nil {
		return result, err
	}
	if result.Requests == 0 {
		return result, errors.New("в выводе k6 нет ни одного запроса")
	}
	source.Close()
	return result, os.Remove(filepath.Join(loadDir, OutputFile))
}

func flag(value bool) int {
	if value {
		return 1
	}
	return 0
}

type gzipFile struct {
	file   *os.File
	gzip   *gzip.Writer
	buffer *bufio.Writer
}

func newGzipFile(path string) (*gzipFile, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	compressed := gzip.NewWriter(file)
	return &gzipFile{file: file, gzip: compressed, buffer: bufio.NewWriterSize(compressed, 1<<20)}, nil
}

func (g *gzipFile) Write(p []byte) (int, error) {
	return g.buffer.Write(p)
}

func (g *gzipFile) WriteString(s string) (int, error) {
	return g.buffer.WriteString(s)
}

func (g *gzipFile) Close() error {
	return errors.Join(g.buffer.Flush(), g.gzip.Close(), g.file.Close())
}
