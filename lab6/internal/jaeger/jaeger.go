package jaeger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"
)

const (
	TracesDir   = "traces"
	TracesFile  = "jaeger.json"
	SummaryFile = "summary.json"
)

type Client struct {
	base string
	http *http.Client
}

func NewClient(base string) *Client {
	return &Client{base: base, http: &http.Client{Timeout: time.Minute}}
}

type response struct {
	Data []trace `json:"data"`
}

type trace struct {
	TraceID string `json:"traceID"`
	Spans   []span `json:"spans"`
}

type span struct {
	OperationName string `json:"operationName"`
	Duration      int64  `json:"duration"`
}

type Operation struct {
	Name   string  `json:"name"`
	Spans  int     `json:"spans"`
	MeanMs float64 `json:"mean_ms"`
	P95Ms  float64 `json:"p95_ms"`
}

type Summary struct {
	Service       string      `json:"service"`
	Traces        int         `json:"traces"`
	Spans         int         `json:"spans"`
	SpansPerTrace float64     `json:"spans_per_trace"`
	Operations    []Operation `json:"operations"`
}

func (c *Client) Export(service string, limit int, runDir string) (Summary, error) {
	query := url.Values{"service": {service}, "limit": {strconv.Itoa(limit)}, "lookback": {"3h"}}
	resp, err := c.http.Get(c.base + "/api/traces?" + query.Encode())
	if err != nil {
		return Summary{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Summary{}, fmt.Errorf("jaeger: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Summary{}, err
	}
	var parsed response
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Summary{}, fmt.Errorf("jaeger: %w", err)
	}

	dir := filepath.Join(runDir, TracesDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Summary{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, TracesFile), body, 0o644); err != nil {
		return Summary{}, err
	}
	summary := summarize(service, parsed.Data)
	return summary, writeJSON(filepath.Join(dir, SummaryFile), summary)
}

func LoadSummary(runDir string) (Summary, bool) {
	data, err := os.ReadFile(filepath.Join(runDir, TracesDir, SummaryFile))
	if err != nil {
		return Summary{}, false
	}
	var s Summary
	return s, json.Unmarshal(data, &s) == nil
}

func summarize(service string, traces []trace) Summary {
	durations := map[string][]float64{}
	spans := 0
	for _, t := range traces {
		spans += len(t.Spans)
		for _, s := range t.Spans {
			durations[s.OperationName] = append(durations[s.OperationName], float64(s.Duration)/1000)
		}
	}
	summary := Summary{Service: service, Traces: len(traces), Spans: spans}
	if len(traces) > 0 {
		summary.SpansPerTrace = float64(spans) / float64(len(traces))
	}
	for name, values := range durations {
		slices.Sort(values)
		total := 0.0
		for _, v := range values {
			total += v
		}
		summary.Operations = append(summary.Operations, Operation{
			Name:   name,
			Spans:  len(values),
			MeanMs: total / float64(len(values)),
			P95Ms:  values[min(len(values)-1, len(values)*95/100)],
		})
	}
	slices.SortFunc(summary.Operations, func(a, b Operation) int { return b.Spans - a.Spans })
	return summary
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
