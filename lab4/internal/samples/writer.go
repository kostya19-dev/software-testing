package samples

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"lab4/internal/clinic"
)

const (
	RequestsFile = "requests.csv.gz"
	VUsFile      = "vus.csv"
	MetaFile     = "meta.json"
)

type Meta struct {
	StartUnixMs int64   `json:"start_unix_ms"`
	Scenario    string  `json:"scenario"`
	Scale       float64 `json:"scale"`
}

type Writer struct {
	mu       sync.Mutex
	file     *os.File
	gzip     *gzip.Writer
	buffer   *bufio.Writer
	vus      *os.File
	firstErr error
}

func Create(dir string, meta Meta) (*Writer, error) {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, MetaFile), data, 0o644); err != nil {
		return nil, err
	}
	file, err := os.Create(filepath.Join(dir, RequestsFile))
	if err != nil {
		return nil, err
	}
	vus, err := os.Create(filepath.Join(dir, VUsFile))
	if err != nil {
		file.Close()
		return nil, err
	}
	compressed := gzip.NewWriter(file)
	w := &Writer{file: file, gzip: compressed, buffer: bufio.NewWriterSize(compressed, 1<<20), vus: vus}
	w.write(w.buffer, "at_ms,latency_us,operation,status,expected,invalid\n")
	w.write(w.vus, "at_ms,vus\n")
	return w, nil
}

func (w *Writer) Observe(r clinic.Result) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.write(w.buffer, fmt.Sprintf("%d,%d,%s,%d,%d,%d\n",
		r.At.UnixMilli(), r.Latency.Microseconds(), r.Operation, r.Status, flag(r.Expected), flag(r.Invalid)))
}

func (w *Writer) RecordVUs(at time.Time, vus int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.write(w.vus, fmt.Sprintf("%d,%d\n", at.UnixMilli(), vus))
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, closer := range []func() error{w.buffer.Flush, w.gzip.Close, w.file.Close, w.vus.Close} {
		if err := closer(); err != nil && w.firstErr == nil {
			w.firstErr = err
		}
	}
	return w.firstErr
}

func (w *Writer) write(target interface{ WriteString(string) (int, error) }, line string) {
	if _, err := target.WriteString(line); err != nil && w.firstErr == nil {
		w.firstErr = err
	}
}

func flag(value bool) int {
	if value {
		return 1
	}
	return 0
}
