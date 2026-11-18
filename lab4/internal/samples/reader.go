package samples

import (
	"cmp"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
)

type Samples struct {
	Start     float64
	Time      []float64
	Latency   []float64
	Status    []int
	OK        []bool
	Invalid   []bool
	Operation []string
	VUsTime   []float64
	VUs       []float64
}

func (s *Samples) Len() int {
	return len(s.Time)
}

func (s *Samples) Duration() float64 {
	end := 0.0
	if n := len(s.Time); n > 0 {
		end = s.Time[n-1]
	}
	if n := len(s.VUsTime); n > 0 {
		end = max(end, s.VUsTime[n-1])
	}
	return end
}

func (s *Samples) Operations() []string {
	seen := map[string]bool{}
	for _, op := range s.Operation {
		seen[op] = true
	}
	names := make([]string, 0, len(seen))
	for op := range seen {
		names = append(names, op)
	}
	slices.Sort(names)
	return names
}

func (s *Samples) Where(keep func(i int) bool) *Samples {
	out := &Samples{Start: s.Start, VUsTime: s.VUsTime, VUs: s.VUs}
	for i := range s.Time {
		if keep(i) {
			out.Time = append(out.Time, s.Time[i])
			out.Latency = append(out.Latency, s.Latency[i])
			out.Status = append(out.Status, s.Status[i])
			out.OK = append(out.OK, s.OK[i])
			out.Invalid = append(out.Invalid, s.Invalid[i])
			out.Operation = append(out.Operation, s.Operation[i])
		}
	}
	return out
}

func (s *Samples) Between(start, end float64) *Samples {
	from, _ := slices.BinarySearch(s.Time, start)
	to, _ := slices.BinarySearch(s.Time, end)
	return &Samples{
		Start:     s.Start,
		Time:      s.Time[from:to],
		Latency:   s.Latency[from:to],
		Status:    s.Status[from:to],
		OK:        s.OK[from:to],
		Invalid:   s.Invalid[from:to],
		Operation: s.Operation[from:to],
		VUsTime:   s.VUsTime,
		VUs:       s.VUs,
	}
}

func (s *Samples) TimesWhere(keep func(i int) bool) []float64 {
	var times []float64
	for i, t := range s.Time {
		if keep(i) {
			times = append(times, t)
		}
	}
	return times
}

func Read(dir string) (*Samples, error) {
	var meta Meta
	data, err := os.ReadFile(filepath.Join(dir, MetaFile))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	s := &Samples{Start: float64(meta.StartUnixMs) / 1000}
	if err := s.readRequests(filepath.Join(dir, RequestsFile)); err != nil {
		return nil, err
	}
	if err := s.readVUs(filepath.Join(dir, VUsFile)); err != nil {
		return nil, err
	}
	if s.Len() == 0 {
		return nil, fmt.Errorf("%s: no requests recorded", dir)
	}
	return s.sorted(), nil
}

func (s *Samples) sorted() *Samples {
	order := make([]int, s.Len())
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(s.Time[a], s.Time[b]) })
	return &Samples{
		Start:     s.Start,
		Time:      permute(s.Time, order),
		Latency:   permute(s.Latency, order),
		Status:    permute(s.Status, order),
		OK:        permute(s.OK, order),
		Invalid:   permute(s.Invalid, order),
		Operation: permute(s.Operation, order),
		VUsTime:   s.VUsTime,
		VUs:       s.VUs,
	}
}

func permute[T any](values []T, order []int) []T {
	out := make([]T, len(order))
	for i, index := range order {
		out[i] = values[index]
	}
	return out
}

func (s *Samples) readRequests(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decompressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	return eachRecord(decompressed, func(r []string) {
		s.Time = append(s.Time, parseFloat(r[0])/1000-s.Start)
		s.Latency = append(s.Latency, parseFloat(r[1])/1000)
		s.Operation = append(s.Operation, r[2])
		s.Status = append(s.Status, int(parseFloat(r[3])))
		s.OK = append(s.OK, r[4] == "1")
		s.Invalid = append(s.Invalid, r[5] == "1")
	})
}

func (s *Samples) readVUs(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return eachRecord(file, func(r []string) {
		s.VUsTime = append(s.VUsTime, parseFloat(r[0])/1000-s.Start)
		s.VUs = append(s.VUs, parseFloat(r[1]))
	})
}

func eachRecord(source io.Reader, handle func([]string)) error {
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

func parseFloat(value string) float64 {
	number, _ := strconv.ParseFloat(value, 64)
	return number
}
